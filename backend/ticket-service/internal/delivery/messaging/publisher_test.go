package messaging

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeSession stands in for a broker connection.
type fakeSession struct {
	dead      atomic.Bool // publish fails, like a connection the broker closed
	published atomic.Int64
	closed    atomic.Int64
}

func (s *fakeSession) publish(string) error {
	if s.dead.Load() {
		return errors.New("connection closed")
	}
	s.published.Add(1)
	return nil
}

func (s *fakeSession) close() { s.closed.Add(1) }

func withFastTimings(t *testing.T, wait, cooldown time.Duration) {
	t.Helper()
	oldWait, oldCooldown := reconnectWait, reconnectCooldown
	reconnectWait, reconnectCooldown = wait, cooldown
	t.Cleanup(func() { reconnectWait, reconnectCooldown = oldWait, oldCooldown })
}

func runConcurrently(n int, f func(i int)) {
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			f(i)
		}(i)
	}
	close(start)
	wg.Wait()
}

// When the connection dies, every in-flight publisher fails at once. Exactly
// one of them may reconnect; the rest reuse the new session, and the dead
// connection is closed rather than leaked.
func TestPublisher_ManyConcurrentFailuresReconnectOnce(t *testing.T) {
	withFastTimings(t, time.Second, time.Second)

	dead := &fakeSession{}
	dead.dead.Store(true)
	fresh := &fakeSession{}
	var dials atomic.Int64
	p := newPublisher(func(int, time.Duration) (session, error) {
		dials.Add(1)
		time.Sleep(50 * time.Millisecond) // a dial takes a moment
		return fresh, nil
	})
	p.cur, p.gen = dead, 1

	const n = 50
	runConcurrently(n, func(int) { p.Publish("event") })

	if got := dials.Load(); got != 1 {
		t.Fatalf("expected a single reconnect for %d concurrent failures, got %d dials", n, got)
	}
	if got := fresh.published.Load(); got != n {
		t.Fatalf("every message must be delivered once on the new session: %d/%d", got, n)
	}
	if got := dead.closed.Load(); got != 1 {
		t.Fatalf("the replaced connection must be closed exactly once, closed %d times", got)
	}
}

// A broker that is down must not make every request dial and wait: the first
// tries, the rest fail fast.
func TestPublisher_DeadBrokerDoesNotStallRequests(t *testing.T) {
	withFastTimings(t, 100*time.Millisecond, time.Second)

	var dials atomic.Int64
	p := newPublisher(func(int, time.Duration) (session, error) {
		dials.Add(1)
		time.Sleep(300 * time.Millisecond) // a failing dial is slow
		return nil, errors.New("connection refused")
	})

	var slowest atomic.Int64
	runConcurrently(30, func(int) {
		start := time.Now()
		if err := p.Publish("event"); err != nil {
			t.Errorf("Publish is best-effort and must never return an error, got %v", err)
		}
		if d := int64(time.Since(start)); d > slowest.Load() {
			slowest.Store(d)
		}
	})

	if got := dials.Load(); got != 1 {
		t.Fatalf("30 concurrent publishes must share one failed dial, got %d", got)
	}
	// The leader waits for its own dial (~300ms); everyone else is capped by
	// reconnectWait, far below what 30 sequential dials would cost.
	if d := time.Duration(slowest.Load()); d > 800*time.Millisecond {
		t.Fatalf("a request stalled %s against a dead broker", d)
	}
}

func TestPublisher_KeepsFailingFastDuringCooldownThenRecovers(t *testing.T) {
	withFastTimings(t, 100*time.Millisecond, 150*time.Millisecond)

	var dials atomic.Int64
	fresh := &fakeSession{}
	brokerUp := atomic.Bool{}
	p := newPublisher(func(int, time.Duration) (session, error) {
		dials.Add(1)
		if !brokerUp.Load() {
			return nil, errors.New("connection refused")
		}
		return fresh, nil
	})

	p.Publish("first")  // dials, fails
	p.Publish("second") // inside the cooldown: must not dial
	if got := dials.Load(); got != 1 {
		t.Fatalf("expected no dial during the cooldown, got %d dials", got)
	}

	brokerUp.Store(true)
	time.Sleep(200 * time.Millisecond) // cooldown over
	p.Publish("third")

	if fresh.published.Load() != 1 || dials.Load() != 2 {
		t.Fatalf("expected to recover after the cooldown: published=%d dials=%d", fresh.published.Load(), dials.Load())
	}
}

func TestPublisher_HealthySessionIsUsedWithoutReconnecting(t *testing.T) {
	healthy := &fakeSession{}
	var dials atomic.Int64
	p := newPublisher(func(int, time.Duration) (session, error) {
		dials.Add(1)
		return &fakeSession{}, nil
	})
	p.cur, p.gen = healthy, 1

	runConcurrently(40, func(int) { p.Publish("event") })

	if dials.Load() != 0 || healthy.published.Load() != 40 {
		t.Fatalf("a healthy session must not reconnect: dials=%d published=%d", dials.Load(), healthy.published.Load())
	}
}
