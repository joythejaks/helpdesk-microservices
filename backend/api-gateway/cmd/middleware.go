package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"api-gateway/pkg/response"

	"github.com/gin-gonic/gin"
)

// runSelfHealthcheck hits the gateway's own /health endpoint over localhost
// and returns a process exit code (0 healthy, 1 unhealthy). Used by
// `./main --healthcheck` from the container's HEALTHCHECK instruction.
func runSelfHealthcheck() int {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://127.0.0.1:" + port + "/health")
	if err != nil {
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}

// =======================
// REQUEST ID
// =======================

// requestIDMiddleware ensures every request has an X-Request-ID, generating
// one when the caller didn't supply it, and echoes it back on the response
// so it can be correlated across gateway and downstream service logs.
func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader("X-Request-ID")
		if reqID == "" {
			reqID = generateRequestID()
		}

		c.Set("request_id", reqID)
		c.Request.Header.Set("X-Request-ID", reqID)
		c.Writer.Header().Set("X-Request-ID", reqID)

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

// internalSecretMiddleware stamps every request with the shared secret that
// proves it passed through the gateway, before it reaches any proxy handler.
// Downstream services (auth-service, ticket-service) reject requests missing
// this, closing off direct access that would bypass JWT validation here.
func internalSecretMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Header.Set("X-Internal-Secret", secret)
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

func rateLimitMiddleware(rl *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.allow(c.ClientIP()) {
			response.Error(c, 429, "too many requests", "RATE_LIMITED")
			c.Abort()
			return
		}
		c.Next()
	}
}
