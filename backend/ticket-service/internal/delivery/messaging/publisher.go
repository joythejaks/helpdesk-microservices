package messaging

import (
	"log"
	"time"

	amqp091 "github.com/rabbitmq/amqp091-go"
)

const (
	queueName         = "ticket_created"
	maxStartupRetries = 15
	startupRetryDelay = 2 * time.Second
	reconnectDelay    = 3 * time.Second
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

	_, err = ch.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
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

// Publish sends a message, reconnecting once on failure.
func (p *Publisher) Publish(message string) error {
	if p.ch == nil {
		if err := p.reconnect(); err != nil {
			log.Println("⚠️ Publisher not ready, skipping:", message)
			return nil
		}
	}

	err := p.ch.Publish(
		"",
		queueName,
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
		return p.ch.Publish("", queueName, false, false, amqp091.Publishing{
			ContentType: "text/plain",
			Body:        []byte(message),
		})
	}

	log.Println("📨 Message sent:", message)
	return nil
}
