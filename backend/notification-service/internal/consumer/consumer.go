package consumer

import (
	"log"
	"time"

	"notification-service/internal/delivery/ws"

	amqp091 "github.com/rabbitmq/amqp091-go"
)

func StartConsumer(url string) {
	go func() {
		var conn *amqp091.Connection
		var err error

		// retry connection
		for i := 0; i < 15; i++ {
			conn, err = amqp091.Dial(url)
			if err == nil {
				log.Println("✅ Connected to RabbitMQ (consumer)")
				break
			}

			log.Println("⏳ Waiting for RabbitMQ...")
			time.Sleep(2 * time.Second)
		}

		// ❗ jangan crash
		if err != nil {
			log.Println("⚠️ RabbitMQ not ready, consumer skipped")
			return
		}

		ch, err := conn.Channel()
		if err != nil {
			log.Println("❌ Channel error:", err)
			return
		}

		q, err := ch.QueueDeclare(
			"ticket_created",
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			log.Println("❌ Queue declare error:", err)
			return
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
			log.Println("❌ Consume error:", err)
			return
		}

		for d := range msgs {
			msg := string(d.Body)
			log.Println("📥 Received:", msg)

			ws.Send(msg)
		}
	}()
}
