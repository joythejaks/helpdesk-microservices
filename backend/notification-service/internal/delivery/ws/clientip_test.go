package ws

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func reqFrom(peer, xff string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/ws", nil)
	r.RemoteAddr = peer
	if xff != "" {
		r.Header.Set("X-Forwarded-For", xff)
	}
	return r
}

func withTrusted(t *testing.T, cidrs ...string) {
	t.Helper()
	if err := SetTrustedProxies(cidrs); err != nil {
		t.Fatalf("SetTrustedProxies: %v", err)
	}
	t.Cleanup(func() { _ = SetTrustedProxies(nil) })
}

func TestClientIP_NoTrustedProxiesIgnoresForwardedFor(t *testing.T) {
	// Default: a direct client can send any X-Forwarded-For; it must not
	// be able to pick its own rate-limit key.
	got := clientIP(reqFrom("198.51.100.9:5555", "1.2.3.4"))
	if got != "198.51.100.9" {
		t.Fatalf("expected the TCP peer, got %s", got)
	}
}

func TestClientIP_UntrustedPeerIgnoresForwardedFor(t *testing.T) {
	withTrusted(t, "10.244.0.0/16")
	got := clientIP(reqFrom("198.51.100.9:5555", "1.2.3.4"))
	if got != "198.51.100.9" {
		t.Fatalf("expected the TCP peer for an untrusted peer, got %s", got)
	}
}

func TestClientIP_TrustedPeerUsesForwardedFor(t *testing.T) {
	withTrusted(t, "10.244.0.0/16")
	got := clientIP(reqFrom("10.244.0.7:4000", "203.0.113.7"))
	if got != "203.0.113.7" {
		t.Fatalf("expected the forwarded client, got %s", got)
	}
}

func TestClientIP_SkipsTrustedHopsFromTheRight(t *testing.T) {
	withTrusted(t, "10.0.0.0/8")
	got := clientIP(reqFrom("10.0.0.2:4000", "203.0.113.7, 10.0.0.5"))
	if got != "203.0.113.7" {
		t.Fatalf("expected the right-most untrusted entry, got %s", got)
	}
}

func TestClientIP_IgnoresEntriesTheClientPrepended(t *testing.T) {
	withTrusted(t, "10.244.0.0/16")
	// The client sent "X-Forwarded-For: 1.1.1.1"; the proxy appended the
	// address it actually saw. The fake left-most entry must not win.
	got := clientIP(reqFrom("10.244.0.7:4000", "1.1.1.1, 203.0.113.7"))
	if got != "203.0.113.7" {
		t.Fatalf("expected the address the proxy saw, got %s", got)
	}
}

func TestClientIP_AllHopsTrustedUsesOutermost(t *testing.T) {
	withTrusted(t, "10.0.0.0/8")
	// Both entries are trusted proxies: use the outermost one, not the
	// immediate peer (which is just one of possibly many proxy replicas).
	got := clientIP(reqFrom("10.0.0.2:4000", "10.0.0.9, 10.0.0.5"))
	if got != "10.0.0.9" {
		t.Fatalf("expected the outermost address, got %s", got)
	}
}

func TestClientIP_MalformedForwardedForFallsBackToPeer(t *testing.T) {
	withTrusted(t, "10.244.0.0/16")
	got := clientIP(reqFrom("10.244.0.7:4000", "not-an-ip"))
	if got != "10.244.0.7" {
		t.Fatalf("expected the peer for a malformed header, got %s", got)
	}
}

func TestSetTrustedProxies_AcceptsBareIPAndRejectsGarbage(t *testing.T) {
	if err := SetTrustedProxies([]string{"10.1.2.3", "fd00::1", " "}); err != nil {
		t.Fatalf("bare IPs should be accepted: %v", err)
	}
	if !isTrusted("10.1.2.3") || isTrusted("10.1.2.4") {
		t.Fatal("a bare IP should trust exactly that host")
	}
	if err := SetTrustedProxies([]string{"not-a-cidr"}); err == nil {
		t.Fatal("expected an error for an invalid entry")
	}
	_ = SetTrustedProxies(nil)
}

// Behind a trusted proxy, two different real clients must get their own
// buckets instead of sharing the proxy's.
func TestRateLimit_SeparatesClientsBehindTrustedProxy(t *testing.T) {
	withTrusted(t, "10.244.0.0/16")
	rl := &RateLimiter{visitors: make(map[string]*visitor), rate: 0.001, burst: 1}
	h := RateLimit(rl, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	call := func(xff string) int {
		w := httptest.NewRecorder()
		h(w, reqFrom("10.244.0.7:4000", xff))
		return w.Code
	}

	if call("203.0.113.1") != http.StatusOK {
		t.Fatal("first client's first request should pass")
	}
	if call("203.0.113.2") != http.StatusOK {
		t.Fatal("a different client must not share the first client's bucket")
	}
	if call("203.0.113.1") != http.StatusTooManyRequests {
		t.Fatal("the first client's second request should be limited")
	}
}
