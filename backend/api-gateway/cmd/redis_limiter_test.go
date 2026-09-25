package main

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestRedisLimiter(t *testing.T, mr *miniredis.Miniredis, rps, burst float64) *redisLimiter {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { client.Close() })
	return newRedisLimiter(client, "rl:test:", rps, burst)
}

func TestRedisLimiter_AllowsUpToBurstThenBlocks(t *testing.T) {
	mr := miniredis.RunT(t)
	mr.SetTime(time.Now())
	rl := newTestRedisLimiter(t, mr, 1, 3)

	for i := 0; i < 3; i++ {
		if !rl.allow("1.2.3.4") {
			t.Fatalf("expected request %d within burst to be allowed", i+1)
		}
	}
	if rl.allow("1.2.3.4") {
		t.Fatal("expected request beyond burst to be denied")
	}
}

func TestRedisLimiter_KeysAreIndependent(t *testing.T) {
	mr := miniredis.RunT(t)
	mr.SetTime(time.Now())
	rl := newTestRedisLimiter(t, mr, 1, 1)

	if !rl.allow("a") {
		t.Fatal("expected first request for key a to be allowed")
	}
	if rl.allow("a") {
		t.Fatal("expected second request for key a to be denied")
	}
	if !rl.allow("b") {
		t.Fatal("expected key b to have its own bucket")
	}
}

// The whole point of the Redis limiter: two replicas share one bucket, so
// the effective limit stays the configured one instead of N times it.
func TestRedisLimiter_ReplicasShareOneBucket(t *testing.T) {
	mr := miniredis.RunT(t)
	mr.SetTime(time.Now())
	replicaA := newTestRedisLimiter(t, mr, 1, 2)
	replicaB := newTestRedisLimiter(t, mr, 1, 2)

	if !replicaA.allow("9.9.9.9") {
		t.Fatal("expected first request (replica A) to be allowed")
	}
	if !replicaB.allow("9.9.9.9") {
		t.Fatal("expected second request (replica B) to be allowed")
	}
	if replicaA.allow("9.9.9.9") || replicaB.allow("9.9.9.9") {
		t.Fatal("expected the shared burst of 2 to be exhausted on both replicas")
	}
}

func TestRedisLimiter_ReplenishesOverTime(t *testing.T) {
	mr := miniredis.RunT(t)
	start := time.Now()
	mr.SetTime(start)
	rl := newTestRedisLimiter(t, mr, 1, 1)

	if !rl.allow("k") {
		t.Fatal("expected first request to be allowed")
	}
	if rl.allow("k") {
		t.Fatal("expected second immediate request to be denied")
	}

	mr.SetTime(start.Add(2 * time.Second))
	if !rl.allow("k") {
		t.Fatal("expected request to be allowed again after 2s at 1 token/sec")
	}
}

func TestRedisLimiter_KeysExpire(t *testing.T) {
	mr := miniredis.RunT(t)
	mr.SetTime(time.Now())
	rl := newTestRedisLimiter(t, mr, 1, 1)

	rl.allow("k")
	if ttl := mr.TTL("rl:test:k"); ttl <= 0 {
		t.Fatalf("expected an idle-bucket TTL to be set, got %v", ttl)
	}
}

func TestRedisLimiter_FailsOpenWhenRedisIsDown(t *testing.T) {
	mr := miniredis.RunT(t)
	rl := newTestRedisLimiter(t, mr, 1, 1)
	mr.Close()

	for i := 0; i < 3; i++ {
		if !rl.allow("k") {
			t.Fatal("expected fail-open: requests must be allowed while Redis is unreachable")
		}
	}
}
