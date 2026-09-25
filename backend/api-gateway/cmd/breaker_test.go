package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// roundTripperFunc lets a test stand in for the real upstream transport.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func newProbeTransport(next http.RoundTripper) (*breakerTransport, func() (state string, failures uint32)) {
	br := newUpstreamBreaker("probe")
	return &breakerTransport{breaker: br, next: next}, func() (string, uint32) {
		return br.State().String(), br.Counts().TotalFailures
	}
}

// A client hanging up says nothing about the upstream's health. Counting it
// as a failure would let anyone open the circuit for every user just by
// starting requests and dropping the connection.
func TestBreakerTransport_ClientCancelIsNotAnUpstreamFailure(t *testing.T) {
	tr, stats := newProbeTransport(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		<-r.Context().Done()
		return nil, r.Context().Err()
	}))

	for i := 0; i < 20; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // the client is already gone
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://upstream/x", nil)
		_, err := tr.RoundTrip(req)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("the caller must still see the original error, got %v", err)
		}
	}

	state, failures := stats()
	if state != "closed" || failures != 0 {
		t.Fatalf("client cancels must not count against the upstream: state=%s failures=%d", state, failures)
	}
}

func TestBreakerTransport_ClientDeadlineIsNotAnUpstreamFailure(t *testing.T) {
	tr, stats := newProbeTransport(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		<-r.Context().Done()
		return nil, r.Context().Err()
	}))

	for i := 0; i < 20; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://upstream/x", nil)
		tr.RoundTrip(req)
		cancel()
	}

	if state, failures := stats(); state != "closed" || failures != 0 {
		t.Fatalf("a client's own deadline must not count: state=%s failures=%d", state, failures)
	}
}

// The other half of the contract: the exclusion must not swallow genuine
// upstream failures, or the breaker never trips at all.
func TestBreakerTransport_UpstreamErrorsStillOpenTheCircuit(t *testing.T) {
	tr, stats := newProbeTransport(roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("connection refused")
	}))

	for i := 0; i < 10; i++ {
		req, _ := http.NewRequest(http.MethodGet, "http://upstream/x", nil) // live context
		tr.RoundTrip(req)
	}

	if state, _ := stats(); state != "open" {
		t.Fatalf("consecutive upstream failures must open the circuit, state=%s", state)
	}
}

// End to end through the real proxy: patient clients keep getting 200 from a
// healthy (but slow) upstream while other clients keep hanging up.
func TestProxy_HungUpClientsDoNotBlockOthers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()
	target, _ := url.Parse(upstream.URL)

	br := newUpstreamBreaker("e2e")
	r := gin.New()
	r.Any("/x", proxyTo(target, br))
	gw := httptest.NewServer(r)
	defer gw.Close()

	for i := 0; i < 10; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, gw.URL+"/x", nil)
		http.DefaultClient.Do(req)
		cancel()
	}
	time.Sleep(400 * time.Millisecond) // let the abandoned requests settle

	resp, err := http.Get(gw.URL + "/x")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("a healthy upstream rejected a patient client: %d (breaker %s)", resp.StatusCode, br.State())
	}
}

// And a really dead upstream must still fail fast with 503.
func TestProxy_DeadUpstreamStillReturnsCircuitOpen(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	target, _ := url.Parse(upstream.URL)
	upstream.Close()

	r := gin.New()
	r.Any("/x", proxyTo(target, newUpstreamBreaker("dead")))
	gw := httptest.NewServer(r)
	defer gw.Close()

	var last int
	var body string
	for i := 0; i < 10; i++ {
		resp, err := http.Get(gw.URL + "/x")
		if err != nil {
			t.Fatal(err)
		}
		buf := new(strings.Builder)
		b := make([]byte, 512)
		n, _ := resp.Body.Read(b)
		buf.Write(b[:n])
		resp.Body.Close()
		last, body = resp.StatusCode, buf.String()
	}
	if last != http.StatusServiceUnavailable || !strings.Contains(body, "CIRCUIT_OPEN") {
		t.Fatalf("expected 503 CIRCUIT_OPEN once the dead upstream tripped the breaker, got %d %s", last, body)
	}
}
