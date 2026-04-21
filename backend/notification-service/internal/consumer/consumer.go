package consumer

import (
	"log"
	"time"

	"notification-service/internal/delivery/ws"

	amqp091 "github.com/rabbitmq/amqp091-go"
)

const reconnectDelay = 2 * time.Second

func StartConsumer(url string) {
	go func() {
		for {
			if err := consume(url); err != nil {
				log.Println("RabbitMQ consumer disconnected:", err)
			}

			log.Println("Reconnecting RabbitMQ consumer...")
			time.Sleep(reconnectDelay)
		}
	}()
}

func consume(url string) error {
	conn, err := dialWithRetry(url, 15, reconnectDelay)
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

	for {
		select {
		case d, ok := <-msgs:
			if !ok {
				return nil
			}

			msg := string(d.Body)
			log.Println("Received:", msg)
			ws.Send(msg)
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
