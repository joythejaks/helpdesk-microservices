package messaging

import (
	"context"
	"log"
	"time"

	amqp091 "github.com/rabbitmq/amqp091-go"
)

const (
	// queueName is the durable, competing-consumers queue used for
	// notification-service's DB persistence (exactly-once across
	// replicas). eventsExchange is a fanout exchange every
	// notification-service replica also binds its own exclusive queue to,
	// for local WebSocket delivery (at-least-once-per-replica) — see
	// notification-service/internal/consumer/consumer.go.
	queueName         = "ticket_created"
	eventsExchange    = "ticket_events"
	maxStartupRetries = 15
	startupRetryDelay = 2 * time.Second
	reconnectDelay    = 3 * time.Second

	// dlxExchange/dlqQueue must match notification-service's consumer.go
	// declarations byte-for-byte (both sides declare queueName, and
	// RabbitMQ rejects a redeclare with mismatched args).
	dlxExchange = "ticket_events.dlx"
	dlqQueue    = "ticket_events.dlq"

	// publishConfirmTimeout bounds how long Publish waits for RabbitMQ to
	// ack/nack — Publish runs synchronously in the HTTP request path, so
	// an unbounded wait would turn a broker hiccup into a slow API.
	publishConfirmTimeout = 3 * time.Second
)

type Publisher struct {
	url  string
	conn *amqp091.Connection
	ch   *amqp091.Channel
}

func NewPublisher(url string) (*Publisher, error) {
	p := &Publisher{url: url}

	if err := p.connect(); err != nil {
		log.Println("⚠️ RabbitMQ not ready, publisher will retry on first publish")
		return p, nil
	}

	return p, nil
}

// connect dials RabbitMQ with retries and sets up the channel + queue.
func (p *Publisher) connect() error {
	var conn *amqp091.Connection
	var err error

	for i := 0; i < maxStartupRetries; i++ {
		conn, err = amqp091.Dial(p.url)
		if err == nil {
			log.Println("✅ Connected to RabbitMQ (publisher)")
			break
		}
		log.Printf("⏳ Waiting for RabbitMQ... (%d/%d)", i+1, maxStartupRetries)
		time.Sleep(startupRetryDelay)
	}

	if err != nil {
		return err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return err
	}

	if err := ch.ExchangeDeclare(eventsExchange, "fanout", true, false, false, false, nil); err != nil {
		ch.Close()
		conn.Close()
		return err
	}

	if err := ch.ExchangeDeclare(dlxExchange, "fanout", true, false, false, false, nil); err != nil {
		ch.Close()
		conn.Close()
		return err
	}

	dlq, err := ch.QueueDeclare(dlqQueue, true, false, false, false, nil)
	if err != nil {
		ch.Close()
		conn.Close()
		return err
	}

	if err := ch.QueueBind(dlq.Name, "", dlxExchange, false, nil); err != nil {
		ch.Close()
		conn.Close()
		return err
	}

	q, err := ch.QueueDeclare(queueName, true, false, false, false, amqp091.Table{
		"x-dead-letter-exchange": dlxExchange,
	})
	if err != nil {
		ch.Close()
		conn.Close()
		return err
	}

	// The persistence queue is bound to the fanout exchange too, so it
	// keeps receiving every event unchanged — only the publish target
	// below actually moves from "direct to queue" to "via the exchange".
	if err := ch.QueueBind(q.Name, "", eventsExchange, false, nil); err != nil {
		ch.Close()
		conn.Close()
		return err
	}

	// Publisher confirms: Publish() below waits (bounded) for RabbitMQ to
	// actually ack/nack receipt, instead of "sent" meaning only "written
	// to the local TCP buffer."
	if err := ch.Confirm(false); err != nil {
		ch.Close()
		conn.Close()
		return err
	}

	p.conn = conn
	p.ch = ch
	return nil
}

// reconnect closes stale connection and re-dials once.
func (p *Publisher) reconnect() error {
	log.Println("🔄 Publisher reconnecting to RabbitMQ...")
	if p.ch != nil {
		p.ch.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
	time.Sleep(reconnectDelay)
	return p.connect()
}

// Publish sends a message, reconnecting once on failure. Best-effort: a
// notification failure should never fail the underlying HTTP request, so
// this always returns nil once the message has been handed to RabbitMQ —
// it only waits (bounded) to confirm receipt, logging the outcome.
func (p *Publisher) Publish(message string) error {
	if p.ch == nil {
		if err := p.reconnect(); err != nil {
			log.Println("⚠️ Publisher not ready, skipping:", message)
			return nil
		}
	}

	confirmation, err := p.ch.PublishWithDeferredConfirm(
		eventsExchange,
		"",
		false,
		false,
		amqp091.Publishing{
			ContentType: "text/plain",
			Body:        []byte(message),
		},
	)

	if err != nil {
		log.Println("❌ Publish failed, attempting reconnect:", err)
		if reconnErr := p.reconnect(); reconnErr != nil {
			log.Println("⚠️ Reconnect failed, skipping:", message)
			return nil
		}
		// retry once after reconnect
		confirmation, err = p.ch.PublishWithDeferredConfirm(eventsExchange, "", false, false, amqp091.Publishing{
			ContentType: "text/plain",
			Body:        []byte(message),
		})
		if err != nil {
			log.Println("⚠️ Publish retry failed, skipping:", message)
			return nil
		}
	}

	p.waitForConfirm(confirmation, message)
	return nil
}

func (p *Publisher) waitForConfirm(confirmation *amqp091.DeferredConfirmation, message string) {
	ctx, cancel := context.WithTimeout(context.Background(), publishConfirmTimeout)
	defer cancel()

	acked, err := confirmation.WaitContext(ctx)
	if err != nil {
		log.Println("⚠️ publish confirm wait failed:", err, "for:", message)
		return
	}
	if !acked {
		log.Println("⚠️ RabbitMQ nacked publish:", message)
		return
	}

	log.Println("📨 Message sent and confirmed:", message)
}
