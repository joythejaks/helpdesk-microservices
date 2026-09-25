package http

import (
	"context"
	"math"
	"sync/atomic"
	"time"

	"ticket-service/pkg/logger"

	"github.com/redis/go-redis/v9"
)

// Limiter is what RateLimitMiddleware needs. Both the in-memory *RateLimiter
// and the Redis-backed *redisLimiter satisfy it.
type Limiter interface {
	allow(key string) bool
}

const (
	redisCallTimeout    = 100 * time.Millisecond
	redisErrLogInterval = 10 * time.Second
)

// tokenBucketScript is the same token bucket as RateLimiter.allow (bucket
// starts at burst-1, lazy refill capped at burst, last-seen updated even on
// a deny), run atomically in Redis so every replica shares one bucket per
// key. It uses Redis TIME, not the caller's clock, so replica clock skew
// cannot skew the refill.
// KEYS[1]=bucket key, ARGV[1]=rate (tokens/sec), ARGV[2]=burst, ARGV[3]=ttl ms.
var tokenBucketScript = redis.NewScript(`
local t = redis.call('TIME')
local now = tonumber(t[1]) * 1000 + math.floor(tonumber(t[2]) / 1000)
local rate = tonumber(ARGV[1])
local burst = tonumber(ARGV[2])
local ttl = tonumber(ARGV[3])
local data = redis.call('HMGET', KEYS[1], 'tokens', 'ts')
local tokens = tonumber(data[1])
local ts = tonumber(data[2])
local allowed = 0
if tokens == nil then
  tokens = burst - 1
  allowed = 1
else
  tokens = math.min(burst, tokens + math.max(0, now - ts) / 1000 * rate)
  if tokens >= 1 then
    tokens = tokens - 1
    allowed = 1
  end
end
redis.call('HSET', KEYS[1], 'tokens', tostring(tokens), 'ts', now)
redis.call('PEXPIRE', KEYS[1], ttl)
return allowed
`)

type redisLimiter struct {
	client     *redis.Client
	prefix     string
	rate       float64
	burst      float64
	ttlMs      int64
	lastErrLog atomic.Int64
}

func newRedisLimiter(client *redis.Client, prefix string, rps, burst float64) *redisLimiter {
	// An idle bucket is full again after burst/rate seconds, so keeping the
	// key any longer than a couple of those is pointless.
	ttlMs := int64(math.Ceil(burst/rps*2*1000)) + 1000
	return &redisLimiter{client: client, prefix: prefix, rate: rps, burst: burst, ttlMs: ttlMs}
}

// allow fails open: a Redis outage must not turn into a login outage, so on
// any error it logs (throttled) and lets the request through.
func (rl *redisLimiter) allow(key string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), redisCallTimeout)
	defer cancel()

	res, err := tokenBucketScript.Run(ctx, rl.client, []string{rl.prefix + key}, rl.rate, rl.burst, rl.ttlMs).Int()
	if err != nil {
		rl.logErr(err)
		return true
	}
	return res == 1
}

func (rl *redisLimiter) logErr(err error) {
	now := time.Now().UnixNano()
	last := rl.lastErrLog.Load()
	if now-last < int64(redisErrLogInterval) {
		return
	}
	if rl.lastErrLog.CompareAndSwap(last, now) {
		logger.Log.WithError(err).Warn("redis rate limiter unavailable, failing open")
	}
}

// NewLimiter returns the shared Redis-backed limiter when redisURL is set,
// otherwise the per-process in-memory one (same behaviour as before Redis).
func NewLimiter(redisURL, prefix string, rps, burst float64) Limiter {
	if redisURL == "" {
		return NewRateLimiter(rps, burst)
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		logger.Log.Fatal("invalid REDIS_URL: ", err)
	}
	return newRedisLimiter(redis.NewClient(opts), prefix, rps, burst)
}
