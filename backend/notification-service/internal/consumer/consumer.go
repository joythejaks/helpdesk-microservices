package consumer

import (
	"encoding/json"
	"log"
	"sync/atomic"
	"time"

	"notification-service/internal/delivery/ws"
	"notification-service/internal/usecase"

	amqp091 "github.com/rabbitmq/amqp091-go"
)

const (
	startupRetryDelay = 3 * time.Second // jeda antar retry saat startup
	reconnectDelay    = 2 * time.Second // jeda sebelum reconnect setelah disconnect

	// persistQueue is durable and shared across every notification-service
	// replica (classic competing consumers) — each event is persisted
	// exactly once collectively, regardless of replica count.
	persistQueue = "ticket_created"

	// eventsExchange is a fanout exchange ticket-service publishes every
	// event to. persistQueue is bound to it (so persistence keeps working
	// unchanged), and each replica additionally binds its own exclusive
	// queue to it below, purely for local WebSocket delivery — every
	// replica gets a copy, so whichever one actually holds the target
	// user's live connection can deliver it (see BACKLOG.md's
	// "notification-service can't horizontally scale" item this closes).
	eventsExchange = "ticket_events"
)

var connected atomic.Bool

// IsConnected reports whether the consumer currently has a live RabbitMQ
// connection, so /health can reflect real dependency state instead of
// always reporting ok.
func IsConnected() bool {
	return connected.Load()
}

// StartConsumer begins consuming ticket events on two independent loops:
// one persists events with a concrete TargetUserID (role broadcasts stay
// ephemeral — see backend/BACKLOG.md for why), the other pushes every
// event to whichever WebSocket clients are connected to *this* replica.
// Splitting them avoids double-persisting the same event once more than
// one replica is running.
func StartConsumer(url string, notifier *usecase.NotificationUsecase) {
	go func() {
		for {
			if err := consumePersist(url, notifier); err != nil {
				log.Println("RabbitMQ persist consumer disconnected:", err)
			}

			log.Println("Reconnecting RabbitMQ persist consumer...")
			time.Sleep(reconnectDelay)
		}
	}()

	go func() {
		for {
			if err := consumeBroadcast(url); err != nil {
				log.Println("RabbitMQ broadcast consumer disconnected:", err)
			}

			log.Println("Reconnecting RabbitMQ broadcast consumer...")
			time.Sleep(reconnectDelay)
		}
	}()
}

// consumePersist owns the exactly-once-across-the-fleet side: the shared
// durable queue, bound to the fanout exchange so it keeps receiving every
// event exactly as it did before this queue existed alongside a fanout.
func consumePersist(url string, notifier *usecase.NotificationUsecase) error {
	conn, err := dialWithRetry(url, 15, startupRetryDelay)
	if err != nil {
		return err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if err := ch.ExchangeDeclare(eventsExchange, "fanout", true, false, false, false, nil); err != nil {
		return err
	}

	q, err := ch.QueueDeclare(persistQueue, true, false, false, false, nil)
	if err != nil {
		return err
	}

	if err := ch.QueueBind(q.Name, "", eventsExchange, false, nil); err != nil {
		return err
	}

	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		return err
	}

	connClosed := conn.NotifyClose(make(chan *amqp091.Error, 1))
	chClosed := ch.NotifyClose(make(chan *amqp091.Error, 1))

	log.Println("Connected to RabbitMQ (persist consumer)")
	connected.Store(true)
	defer connected.Store(false)

	for {
		select {
		case d, ok := <-msgs:
			if !ok {
				return nil
			}

			var evt event
			if err := json.Unmarshal(d.Body, &evt); err != nil {
				log.Println("⚠️ malformed notification event, dropping:", err)
				continue
			}

			if evt.TargetUserID == nil {
				continue // role broadcasts stay WebSocket-only, never persisted
			}

			if err := notifier.Create(*evt.TargetUserID, d.Body); err != nil {
				log.Println("⚠️ failed to persist notification:", err)
			}
		case err := <-connClosed:
			if err != nil {
				return err
			}
			return nil
		case err := <-chClosed:
			if err != nil {
				return err
			}
			return nil
		}
	}
}

// consumeBroadcast owns the at-least-once-per-replica side: an
// exclusive, auto-delete queue unique to this process, bound to the
// fanout exchange, so every replica gets a copy of every event purely to
// check its own in-memory WebSocket connections.
func consumeBroadcast(url string) error {
	conn, err := dialWithRetry(url, 15, startupRetryDelay)
	if err != nil {
		return err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if err := ch.ExchangeDeclare(eventsExchange, "fanout", true, false, false, false, nil); err != nil {
		return err
	}

	// Empty name + exclusive + auto-delete: RabbitMQ generates a unique
	// name and the queue disappears when this connection closes — exactly
	// what a per-replica fanout subscriber needs, no cross-replica config.
	q, err := ch.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		return err
	}

	if err := ch.QueueBind(q.Name, "", eventsExchange, false, nil); err != nil {
		return err
	}

	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		return err
	}

	connClosed := conn.NotifyClose(make(chan *amqp091.Error, 1))
	chClosed := ch.NotifyClose(make(chan *amqp091.Error, 1))

	log.Println("Connected to RabbitMQ (broadcast consumer)")

	for {
		select {
		case d, ok := <-msgs:
			if !ok {
				return nil
			}

			raw := string(d.Body)

			var evt event
			if err := json.Unmarshal(d.Body, &evt); err != nil {
				log.Println("⚠️ malformed notification event, dropping:", err)
				continue
			}

			switch {
			case evt.TargetUserID != nil:
				ws.SendToUser(*evt.TargetUserID, raw)
			case len(evt.TargetRoles) > 0:
				ws.SendToRoles(evt.TargetRoles, raw)
			default:
				log.Println("⚠️ notification event has no target, dropping:", raw)
			}
		case err := <-connClosed:
			if err != nil {
				return err
			}
			return nil
		case err := <-chClosed:
			if err != nil {
				return err
			}
			return nil
		}
	}
}

func dialWithRetry(url string, attempts int, delay time.Duration) (*amqp091.Connection, error) {
	var lastErr error

	for i := 0; i < attempts; i++ {
		conn, err := amqp091.Dial(url)
		if err == nil {
			return conn, nil
		}

		lastErr = err
		log.Println("Waiting for RabbitMQ...")
		time.Sleep(delay)
	}

	return nil, lastErr
}
