package http

import (
	"crypto/subtle"
	"sync"
	"time"

	"auth-service/pkg/response"

	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
)

// InternalOnlyMiddleware rejects any request that doesn't carry the shared
// secret the API gateway attaches to every proxied request. This is the
// enforcement point that stops auth-service's business routes (and the
// X-User-ID trust they rely on) from being reachable by calling the service
// directly, bypassing the gateway's JWT validation.
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

// RequireRole rejects any request whose X-User-ROLE header (set by the
// gateway after JWT validation) doesn't match. Used for admin-only routes
// like staff provisioning.
func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("X-User-ROLE") != role {
			response.Error(c, 403, "forbidden", "forbidden")
			c.Abort()
			return
		}
		c.Next()
	}
}

// TraceMiddleware memastikan setiap request memiliki Trace ID untuk logging
func TraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			traceID = uuid.New().String()
		}

		c.Set("TraceID", traceID)
		c.Header("X-Trace-ID", traceID)
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
	rate     float64
	burst    float64
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

func (rl *RateLimiter) cleanupLoop() {
	for range time.Tick(time.Minute) {
		cutoff := time.Now().Add(-3 * time.Minute)
		rl.mu.Lock()
		for k, v := range rl.visitors {
			if v.lastSeen.Before(cutoff) {
				delete(rl.visitors, k)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimitMiddleware throttles brute-force/spam attempts against sensitive
// auth routes (login, register, refresh) on a per-client-IP basis.
func RateLimitMiddleware(rl *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.allow(c.ClientIP()) {
			response.Error(c, 429, "too many requests", "RATE_LIMITED")
			c.Abort()
			return
		}
		c.Next()
	}
}
