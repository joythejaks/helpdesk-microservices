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
)

var connected atomic.Bool

// IsConnected reports whether the consumer currently has a live RabbitMQ
// connection, so /health can reflect real dependency state instead of
// always reporting ok.
func IsConnected() bool {
	return connected.Load()
}

// StartConsumer begins consuming ticket events. notifier persists any
// event with a concrete TargetUserID (role broadcasts stay ephemeral —
// see backend/BACKLOG.md for why) before it's pushed over WebSocket, so a
// notification survives even if nobody's connected to receive it live.
func StartConsumer(url string, notifier *usecase.NotificationUsecase) {
	go func() {
		for {
			if err := consume(url, notifier); err != nil {
				log.Println("RabbitMQ consumer disconnected:", err)
			}

			log.Println("Reconnecting RabbitMQ consumer...")
			time.Sleep(reconnectDelay)
		}
	}()
}

func consume(url string, notifier *usecase.NotificationUsecase) error {
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

	q, err := ch.QueueDeclare(
		"ticket_created",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	msgs, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	connClosed := conn.NotifyClose(make(chan *amqp091.Error, 1))
	chClosed := ch.NotifyClose(make(chan *amqp091.Error, 1))

	log.Println("Connected to RabbitMQ consumer")
	connected.Store(true)
	defer connected.Store(false)

	for {
		select {
		case d, ok := <-msgs:
			if !ok {
				return nil
			}

			raw := string(d.Body)
			log.Println("Received:", raw)

			var evt event
			if err := json.Unmarshal(d.Body, &evt); err != nil {
				log.Println("⚠️ malformed notification event, dropping:", err)
				continue
			}

			switch {
			case evt.TargetUserID != nil:
				if err := notifier.Create(*evt.TargetUserID, d.Body); err != nil {
					log.Println("⚠️ failed to persist notification:", err)
				}
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
