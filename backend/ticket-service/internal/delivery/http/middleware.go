package http

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"ticket-service/pkg/response"

	"github.com/gin-gonic/gin"
)

// RequestIDMiddleware reads the X-Request-ID header api-gateway already
// generates/forwards (or generates one, for requests that reach this
// service directly), so every request can be correlated across services —
// this service had no such middleware before.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader("X-Request-ID")
		if reqID == "" {
			reqID = generateRequestID()
		}

		c.Set("request_id", reqID)
		c.Header("X-Request-ID", reqID)
		c.Next()
	}
}

func generateRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// InternalOnlyMiddleware rejects any request that doesn't carry the shared
// secret the API gateway attaches to every proxied request — closes off
// calling ticket-service directly and spoofing X-User-ID/X-User-ROLE,
// bypassing the gateway's JWT validation.
func InternalOnlyMiddleware(secret string) gin.HandlerFunc {
	secretBytes := []byte(secret)

	return func(c *gin.Context) {
		got := c.GetHeader("X-Internal-Secret")
		if got == "" || subtle.ConstantTimeCompare([]byte(got), secretBytes) != 1 {
			response.Error(c, 403, "forbidden", "forbidden")
			c.Abort()
			return
		}
		c.Next()
	}
}

// =======================
// RATE LIMITER (token bucket per client IP)
// =======================

type visitor struct {
	tokens   float64
	lastSeen time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     float64 // tokens replenished per second
	burst    float64 // max tokens (also the initial bucket size)
}

func NewRateLimiter(rps, burst float64) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rps,
		burst:    burst,
	}
	go rl.cleanupLoop()
	return rl
}

func (rl *RateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	v, exists := rl.visitors[key]
	if !exists {
		rl.visitors[key] = &visitor{tokens: rl.burst - 1, lastSeen: now}
		return true
	}

	elapsed := now.Sub(v.lastSeen).Seconds()
	v.tokens += elapsed * rl.rate
	if v.tokens > rl.burst {
		v.tokens = rl.burst
	}
	v.lastSeen = now

	if v.tokens < 1 {
		return false
	}
	v.tokens--
	return true
}

// cleanupLoop evicts visitors that haven't been seen in a while so the map
// doesn't grow unbounded under a long-running process.
func (rl *RateLimiter) cleanupLoop() {
	for range time.Tick(time.Minute) {
		rl.evict(time.Now().Add(-3 * time.Minute))
	}
}

func (rl *RateLimiter) evict(cutoff time.Time) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	for k, v := range rl.visitors {
		if v.lastSeen.Before(cutoff) {
			delete(rl.visitors, k)
		}
	}
}

// RateLimitMiddleware throttles high-frequency ticket creation/attachment
// uploads on a per-client-IP basis (attachment size is already capped
// elsewhere; this caps request rate).
func RateLimitMiddleware(rl Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.allow(c.ClientIP()) {
			response.Error(c, 429, "too many requests", "RATE_LIMITED")
			c.Abort()
			return
		}
		c.Next()
	}
}
