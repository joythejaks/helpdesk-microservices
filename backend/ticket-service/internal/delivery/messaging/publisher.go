package messaging

import (
	"context"
	"errors"
	"log"
	"sync"
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
	queueName      = "ticket_created"
	eventsExchange = "ticket_events"

	// Startup is patient: a broker that isn't up yet when the pod boots is
	// normal, and nothing is being served yet.
	maxStartupRetries = 15
	startupRetryDelay = 2 * time.Second

	// A reconnect during a request is not: Publish runs inside an HTTP
	// handler, so it may only spend a few seconds trying.
	requestReconnectAttempts = 2
	requestReconnectDelay    = 500 * time.Millisecond
	dialTimeout              = 2 * time.Second

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

// Variables, not constants, only so tests can shorten them.
var (
	// reconnectWait is how long a publisher whose connection just died waits
	// for the goroutine that is already reconnecting before giving up on
	// its message.
	reconnectWait = 1500 * time.Millisecond
	// reconnectCooldown makes Publish fail fast for a while after a
	// reconnect attempt failed, instead of every request dialing a broker
	// that is down.
	reconnectCooldown = 2 * time.Second
)

var (
	errReconnecting = errors.New("another goroutine is still reconnecting")
	errCooldown     = errors.New("reconnect failed recently, not retrying yet")
)

// session is one live broker connection with its confirm-mode channel.
type session interface {
	// publish hands the message to the broker and waits (bounded) for its
	// confirm. It returns an error only when the message could not be handed
	// off at all (dead connection or channel), which is the caller's cue to
	// reconnect. A confirm timeout or a nack is logged, not returned: this
	// is best-effort delivery.
	publish(message string) error
	close()
}

// dialFunc opens a new session, trying up to `attempts` times.
type dialFunc func(attempts int, delay time.Duration) (session, error)

// Publisher publishes ticket events. It is called concurrently from every
// HTTP handler goroutine, so all state is guarded by mu, and only one
// goroutine at a time reconnects (single flight) — before this, every
// failing request dialed and slept on its own, overwrote the shared
// connection without closing the previous one (leaking it), and raced on
// the connection fields.
type Publisher struct {
	dial dialFunc

	mu       sync.RWMutex
	cur      session
	gen      uint64        // incremented every time cur is replaced
	inflight chan struct{} // non-nil while a reconnect runs; closed when it ends
	lastFail time.Time
}

func newPublisher(dial dialFunc) *Publisher {
	return &Publisher{dial: dial}
}

// NewPublisher takes the queue type (classic|quorum) because this service
// declares ticket_created too, and the declaration must match
// notification-service's exactly.
func NewPublisher(url, queueType string) (*Publisher, error) {
	p := newPublisher(dialAMQP(url, queueType))

	s, err := p.dial(maxStartupRetries, startupRetryDelay)
	if err != nil {
		log.Println("⚠️ RabbitMQ not ready, publisher will retry on first publish")
		return p, nil
	}
	p.cur, p.gen = s, 1
	return p, nil
}

func (p *Publisher) current() (session, uint64) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.cur, p.gen
}

// reconnect replaces the session that failed at generation failedGen. If
// another goroutine already did, it returns nil at once; if one is in the
// middle of it, it waits a bounded time for it; if a recent attempt failed,
// it gives up immediately. The dial itself runs without any lock held.
func (p *Publisher) reconnect(failedGen uint64) error {
	p.mu.Lock()
	if p.cur != nil && p.gen != failedGen {
		p.mu.Unlock()
		return nil // already replaced while we were waiting
	}
	if wait := p.inflight; wait != nil {
		p.mu.Unlock()
		select {
		case <-wait:
		case <-time.After(reconnectWait):
			return errReconnecting
		}
		if s, gen := p.current(); s != nil && gen != failedGen {
			return nil
		}
		return errCooldown
	}
	if time.Since(p.lastFail) < reconnectCooldown {
		p.mu.Unlock()
		return errCooldown
	}
	done := make(chan struct{})
	p.inflight = done
	p.mu.Unlock()

	log.Println("🔄 Publisher reconnecting to RabbitMQ...")
	s, err := p.dial(requestReconnectAttempts, requestReconnectDelay)

	p.mu.Lock()
	var old session
	if err == nil {
		old, p.cur = p.cur, s
		p.gen++
	} else {
		p.lastFail = time.Now()
	}
	p.inflight = nil
	close(done)
	p.mu.Unlock()

	if old != nil {
		old.close() // never leak the connection we just replaced
	}
	return err
}

// Publish sends a message, reconnecting once on failure. Best-effort: a
// notification failure should never fail the underlying HTTP request, so
// this always returns nil — it logs what happened instead.
func (p *Publisher) Publish(message string) error {
	s, gen := p.current()
	if s == nil {
		if err := p.reconnect(gen); err != nil {
			log.Println("⚠️ Publisher not ready, skipping:", message, "-", err)
			return nil
		}
		s, gen = p.current()
	}

	err := s.publish(message)
	if err == nil {
		return nil
	}

	log.Println("❌ Publish failed, attempting reconnect:", err)
	if rerr := p.reconnect(gen); rerr != nil {
		log.Println("⚠️ Reconnect failed, skipping:", message, "-", rerr)
		return nil
	}

	// retry once on the new session
	s, _ = p.current()
	if err := s.publish(message); err != nil {
		log.Println("⚠️ Publish retry failed, skipping:", message)
	}
	return nil
}

// amqpSession is the real session, over an amqp091 connection.
type amqpSession struct {
	conn *amqp091.Connection
	ch   *amqp091.Channel
}

func (s *amqpSession) publish(message string) error {
	confirmation, err := s.ch.PublishWithDeferredConfirm(
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
		return err
	}
	waitForConfirm(confirmation, message)
	return nil
}

func (s *amqpSession) close() {
	s.ch.Close()
	s.conn.Close()
}

func waitForConfirm(confirmation *amqp091.DeferredConfirmation, message string) {
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

// dialAMQP returns a dialFunc that connects to RabbitMQ, declares the
// topology and turns on publisher confirms.
func dialAMQP(url, queueType string) dialFunc {
	return func(attempts int, delay time.Duration) (session, error) {
		var conn *amqp091.Connection
		var err error

		for i := 0; i < attempts; i++ {
			// A bounded dial: amqp091.Dial's default can block for 30s on an
			// unreachable broker.
			conn, err = amqp091.DialConfig(url, amqp091.Config{
				Dial: amqp091.DefaultDial(dialTimeout),
			})
			if err == nil {
				log.Println("✅ Connected to RabbitMQ (publisher)")
				break
			}
			log.Printf("⏳ Waiting for RabbitMQ... (%d/%d)", i+1, attempts)
			if i < attempts-1 {
				time.Sleep(delay)
			}
		}
		if err != nil {
			return nil, err
		}

		ch, err := conn.Channel()
		if err != nil {
			conn.Close()
			return nil, err
		}
		fail := func(err error) (session, error) {
			ch.Close()
			conn.Close()
			return nil, err
		}

		if err := ch.ExchangeDeclare(eventsExchange, "fanout", true, false, false, false, nil); err != nil {
			return fail(err)
		}
		if err := ch.ExchangeDeclare(dlxExchange, "fanout", true, false, false, false, nil); err != nil {
			return fail(err)
		}

		dlq, err := ch.QueueDeclare(dlqQueue, true, false, false, false, dlqArgs(queueType))
		if err != nil {
			return fail(err)
		}
		if err := ch.QueueBind(dlq.Name, "", dlxExchange, false, nil); err != nil {
			return fail(err)
		}

		q, err := ch.QueueDeclare(queueName, true, false, false, false, mainQueueArgs(queueType))
		if err != nil {
			return fail(err)
		}

		// The persistence queue is bound to the fanout exchange too, so it
		// keeps receiving every event unchanged — only the publish target
		// actually moves from "direct to queue" to "via the exchange".
		if err := ch.QueueBind(q.Name, "", eventsExchange, false, nil); err != nil {
			return fail(err)
		}

		// Publisher confirms: publish() waits (bounded) for RabbitMQ to
		// actually ack/nack receipt, instead of "sent" meaning only "written
		// to the local TCP buffer."
		if err := ch.Confirm(false); err != nil {
			return fail(err)
		}

		return &amqpSession{conn: conn, ch: ch}, nil
	}
}
