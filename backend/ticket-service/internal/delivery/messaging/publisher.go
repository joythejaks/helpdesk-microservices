package messaging

import (
	"log"
	"time"

	amqp091 "github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	ch *amqp091.Channel
}

func NewPublisher(url string) (*Publisher, error) {
	var conn *amqp091.Connection
	var err error

	for i := 0; i < 15; i++ {
		conn, err = amqp091.Dial(url)
		if err == nil {
			log.Println("✅ Connected to RabbitMQ")
			break
		}

		log.Println("⏳ Waiting for RabbitMQ...")
		time.Sleep(2 * time.Second)
	}

	// ❗ JANGAN FATAL → biar service tetap hidup
	if err != nil {
		log.Println("⚠️ RabbitMQ not ready, skip publisher")
		return nil, nil
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Println("❌ Failed create channel:", err)
		return nil, nil
	}

	_, err = ch.QueueDeclare(
		"ticket_created",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Println("❌ Queue declare failed:", err)
		return nil, nil
	}

	return &Publisher{ch: ch}, nil
}

func (p *Publisher) Publish(message string) error {
	// ❗ kalau publisher belum ready
	if p == nil {
		log.Println("⚠️ Publisher not ready, skip publish")
		return nil
	}

	err := p.ch.Publish(
		"",
		"ticket_created",
		false,
		false,
		amqp091.Publishing{
			ContentType: "text/plain",
			Body:        []byte(message),
		},
	)

	if err != nil {
		log.Println("❌ Failed to publish:", err)
		return err
	}

	log.Println("📨 Message sent:", message)
	return nil
}
