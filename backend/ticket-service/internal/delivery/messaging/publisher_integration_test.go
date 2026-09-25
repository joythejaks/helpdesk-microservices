package messaging

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	amqp091 "github.com/rabbitmq/amqp091-go"
)

// Opt-in integration test against a REAL RabbitMQ — skipped unless both env
// vars are set, so CI and `go test ./...` are unaffected. Run it under the
// race detector, from a container on the compose network:
//
//	docker run --rm --network backend_default -v <repo>/backend/ticket-service:/src -w /src \
//	  -e GOTOOLCHAIN=auto -e RABBITMQ_TEST_URL=amqp://guest:guest@rabbitmq:5672/ \
//	  -e RABBITMQ_TEST_MGMT=http://guest:guest@rabbitmq:15672 \
//	  golang:1.25-bookworm go test -race -run Integration -v ./internal/delivery/messaging/
//
// It publishes from many goroutines while the broker force-closes every
// client connection (via its management API), the way a broker restart or a
// failover would.
func TestPublisherIntegration_ConcurrentPublishAcrossClosedConnections(t *testing.T) {
	brokerURL := os.Getenv("RABBITMQ_TEST_URL")
	mgmt := os.Getenv("RABBITMQ_TEST_MGMT")
	if brokerURL == "" || mgmt == "" {
		t.Skip("set RABBITMQ_TEST_URL and RABBITMQ_TEST_MGMT to run against a real RabbitMQ")
	}

	p, err := NewPublisher(brokerURL, QueueTypeClassic)
	if err != nil {
		t.Fatal(err)
	}

	const (
		workers   = 20
		slowAfter = 2500 * time.Millisecond
	)
	var (
		wg      sync.WaitGroup
		stop    = make(chan struct{})
		total   atomic.Int64
		slow    atomic.Int64
		slowest atomic.Int64
	)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				start := time.Now()
				p.Publish("integration-test")
				d := time.Since(start)
				total.Add(1)
				if d > slowAfter {
					slow.Add(1)
				}
				if int64(d) > slowest.Load() {
					slowest.Store(int64(d))
				}
				time.Sleep(10 * time.Millisecond)
			}
		}()
	}

	// Publish() runs in an HTTP request, so it must not stall a request for
	// long just because the broker dropped the connection.
	for i := 0; i < 3; i++ {
		time.Sleep(1500 * time.Millisecond)
		closeAllBrokerConnections(t, mgmt)
	}
	time.Sleep(4 * time.Second)
	close(stop)
	wg.Wait()

	t.Logf("publishes=%d slower-than-%s=%d slowest=%s", total.Load(), slowAfter, slow.Load(), time.Duration(slowest.Load()))
	if slow.Load() > 0 {
		t.Errorf("%d Publish calls stalled longer than %s while the broker dropped connections", slow.Load(), slowAfter)
	}

	// Every reconnect replaces the previous connection; the old one must be
	// closed, not leaked. Only this test's publisher is connected.
	time.Sleep(time.Second) // let the broker's connection list settle
	if n := len(brokerConnections(t, mgmt)); n > 1 {
		t.Errorf("%d connections still open on the broker; expected 1 (the rest leaked)", n)
	}
}

// A TCP proxy in front of the broker lets the test take it away completely
// (connection refused) and bring it back, without controlling Docker.
type tcpProxy struct {
	t      *testing.T
	target string
	addr   string
	mu     sync.Mutex
	ln     net.Listener
	conns  []net.Conn
}

func newTCPProxy(t *testing.T, target string) *tcpProxy {
	p := &tcpProxy{t: t, target: target}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	p.addr = ln.Addr().String()
	p.serve(ln)
	t.Cleanup(p.stop)
	return p
}

func (p *tcpProxy) serve(ln net.Listener) {
	p.mu.Lock()
	p.ln = ln
	p.mu.Unlock()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			up, err := net.Dial("tcp", p.target)
			if err != nil {
				c.Close()
				continue
			}
			p.mu.Lock()
			p.conns = append(p.conns, c, up)
			p.mu.Unlock()
			go func() { io.Copy(up, c); up.Close() }()
			go func() { io.Copy(c, up); c.Close() }()
		}
	}()
}

// stop makes the broker unreachable: the port refuses connections and every
// open one is cut.
func (p *tcpProxy) stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ln != nil {
		p.ln.Close()
		p.ln = nil
	}
	for _, c := range p.conns {
		c.Close()
	}
	p.conns = nil
}

func (p *tcpProxy) start() {
	ln, err := net.Listen("tcp", p.addr)
	if err != nil {
		p.t.Fatalf("restart proxy on %s: %v", p.addr, err)
	}
	p.serve(ln)
}

// The broker disappears for several seconds and comes back. Requests must
// stay fast the whole time (Publish is best-effort and runs inside an HTTP
// handler), and delivery must resume on its own afterwards.
func TestPublisherIntegration_BrokerUnreachableThenBack(t *testing.T) {
	brokerURL := os.Getenv("RABBITMQ_TEST_URL")
	if brokerURL == "" || os.Getenv("RABBITMQ_TEST_MGMT") == "" {
		t.Skip("set RABBITMQ_TEST_URL and RABBITMQ_TEST_MGMT to run against a real RabbitMQ")
	}
	u, err := url.Parse(brokerURL)
	if err != nil {
		t.Fatal(err)
	}
	target := u.Host

	proxy := newTCPProxy(t, target)
	viaProxy := *u
	viaProxy.Host = proxy.addr
	p, err := NewPublisher(viaProxy.String(), QueueTypeClassic)
	if err != nil {
		t.Fatal(err)
	}

	// Queue depth is read over a separate, direct connection.
	direct, err := amqp091.Dial(brokerURL)
	if err != nil {
		t.Fatal(err)
	}
	defer direct.Close()
	dch, err := direct.Channel()
	if err != nil {
		t.Fatal(err)
	}
	depth := func() int {
		q, err := dch.QueueDeclarePassive(queueName, true, false, false, false, nil)
		if err != nil {
			t.Fatalf("inspect queue: %v", err)
		}
		return q.Messages
	}

	const bound = 2500 * time.Millisecond
	publishAll := func(label string, workers, each int) (slowest time.Duration) {
		var wg sync.WaitGroup
		var mu sync.Mutex
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < each; j++ {
					start := time.Now()
					p.Publish("outage-test")
					if d := time.Since(start); d > bound {
						t.Errorf("%s: a Publish stalled %s (> %s)", label, d, bound)
					}
					mu.Lock()
					if d := time.Since(start); d > slowest {
						slowest = d
					}
					mu.Unlock()
					time.Sleep(20 * time.Millisecond)
				}
			}()
		}
		wg.Wait()
		return slowest
	}

	publishAll("healthy", 5, 5)

	proxy.stop() // broker unreachable
	t.Logf("during the outage: slowest Publish = %s", publishAll("outage", 20, 25))

	proxy.start() // broker back
	time.Sleep(reconnectCooldown + 500*time.Millisecond)

	before := depth()
	publishAll("after recovery", 5, 8)
	if got := depth() - before; got < 40 {
		t.Errorf("delivery did not resume after the broker came back: only %d of 40 messages reached the queue", got)
	}
}

type brokerConn struct {
	Name string `json:"name"`
}

func brokerConnections(t *testing.T, mgmt string) []brokerConn {
	t.Helper()
	u, err := url.Parse(mgmt)
	if err != nil {
		t.Fatal(err)
	}
	pw, _ := u.User.Password()
	req, _ := http.NewRequest(http.MethodGet, u.Scheme+"://"+u.Host+"/api/connections", nil)
	req.SetBasicAuth(u.User.Username(), pw)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("management API: %v", err)
	}
	defer resp.Body.Close()
	var conns []brokerConn
	if err := json.NewDecoder(resp.Body).Decode(&conns); err != nil {
		t.Fatalf("decode connections: %v", err)
	}
	return conns
}

func closeAllBrokerConnections(t *testing.T, mgmt string) {
	t.Helper()
	u, _ := url.Parse(mgmt)
	pw, _ := u.User.Password()
	for _, c := range brokerConnections(t, mgmt) {
		req, _ := http.NewRequest(http.MethodDelete, u.Scheme+"://"+u.Host+"/api/connections/"+url.PathEscape(c.Name), nil)
		req.SetBasicAuth(u.User.Username(), pw)
		if resp, err := http.DefaultClient.Do(req); err == nil {
			resp.Body.Close()
		}
	}
}
