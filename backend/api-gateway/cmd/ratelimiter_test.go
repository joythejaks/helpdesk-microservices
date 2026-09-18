package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimiter_AllowsUpToBurstThenBlocks(t *testing.T) {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     1,
		burst:    3,
	}

	for i := 0; i < 3; i++ {
		if !rl.allow("1.2.3.4") {
			t.Fatalf("expected request %d within burst to be allowed", i+1)
		}
	}
	if rl.allow("1.2.3.4") {
		t.Fatal("expected request beyond burst to be denied")
	}
}

func TestRateLimiter_ReplenishesOverTime(t *testing.T) {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     10, // 10 tokens/sec
		burst:    1,
	}

	if !rl.allow("5.6.7.8") {
		t.Fatal("expected first request to be allowed")
	}
	if rl.allow("5.6.7.8") {
		t.Fatal("expected second immediate request to be denied (burst exhausted)")
	}

	// Simulate 200ms elapsed (well over 1 token at 10/sec) without a real sleep.
	rl.mu.Lock()
	rl.visitors["5.6.7.8"].lastSeen = time.Now().Add(-200 * time.Millisecond)
	rl.mu.Unlock()

	if !rl.allow("5.6.7.8") {
		t.Fatal("expected request to be allowed again after simulated replenishment")
	}
}

func TestRateLimiter_EvictsStaleVisitors(t *testing.T) {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     1,
		burst:    1,
	}

	rl.allow("stale-ip")
	rl.allow("fresh-ip")

	rl.mu.Lock()
	rl.visitors["stale-ip"].lastSeen = time.Now().Add(-10 * time.Minute)
	rl.mu.Unlock()

	rl.evict(time.Now().Add(-3 * time.Minute))

	rl.mu.Lock()
	_, staleExists := rl.visitors["stale-ip"]
	_, freshExists := rl.visitors["fresh-ip"]
	rl.mu.Unlock()

	if staleExists {
		t.Fatal("expected stale visitor to be evicted")
	}
	if !freshExists {
		t.Fatal("expected fresh visitor to survive eviction")
	}
}

func TestRateLimitMiddleware_RejectsWith429(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     1,
		burst:    1,
	}

	r := gin.New()
	r.Use(rateLimitMiddleware(rl))
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })

	req1 := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req1.RemoteAddr = "9.9.9.9:1234"
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("expected first request to succeed, got %d", w1.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req2.RemoteAddr = "9.9.9.9:1234"
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second request from same IP to be rate limited (429), got %d", w2.Code)
	}
}
