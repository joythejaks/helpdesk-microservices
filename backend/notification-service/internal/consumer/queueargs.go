package consumer

import (
	"fmt"

	amqp091 "github.com/rabbitmq/amqp091-go"
)

const (
	QueueTypeClassic = "classic"
	QueueTypeQuorum  = "quorum"
)

// ValidateQueueType rejects anything but the two supported types, so a typo
// fails at startup instead of as a PRECONDITION_FAILED on the broker.
func ValidateQueueType(t string) error {
	if t != QueueTypeClassic && t != QueueTypeQuorum {
		return fmt.Errorf("RABBITMQ_QUEUE_TYPE must be %q or %q, got %q", QueueTypeClassic, QueueTypeQuorum, t)
	}
	return nil
}

// mainQueueArgs are the declare arguments for the durable ticket_created
// queue. notification-service declares the same queue and RabbitMQ rejects a
// redeclare whose arguments differ, so this must stay identical to
// ticket-service/internal/delivery/messaging/queueargs.go. For classic the
// x-queue-type argument is deliberately omitted, which keeps the arguments
// byte-for-byte what existing (pre-quorum) queues were declared with.
func mainQueueArgs(queueType string) amqp091.Table {
	args := amqp091.Table{"x-dead-letter-exchange": dlxExchange}
	if queueType == QueueTypeQuorum {
		args["x-queue-type"] = QueueTypeQuorum
	}
	return args
}

// dlqArgs are the declare arguments for the dead-letter queue: quorum too,
// so a parked message survives the loss of a broker node.
func dlqArgs(queueType string) amqp091.Table {
	if queueType == QueueTypeQuorum {
		return amqp091.Table{"x-queue-type": QueueTypeQuorum}
	}
	return nil
}
