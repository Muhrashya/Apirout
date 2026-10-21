package executor

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

type mockRateLimiter struct {
	allowed bool
	calls   int
}

func (m *mockRateLimiter) Allow(key string, rpm int, burst int) bool {
	m.calls++
	return m.allowed
}

func TestHybridRateLimiter_Disabled(t *testing.T) {
	mock := &mockRateLimiter{allowed: true}
	hl := NewHybridRateLimiter(mock, nil, false, nil)

	if !hl.Allow("test-key", 60, 5) {
		t.Fatalf("expected Allow to return true")
	}
	if mock.calls != 1 {
		t.Fatalf("expected local limiter to be called once, got %d", mock.calls)
	}
}

func TestHybridRateLimiter_NilRedis(t *testing.T) {
	mock := &mockRateLimiter{allowed: false}
	hl := NewHybridRateLimiter(mock, nil, true, nil)

	if hl.Allow("test-key", 60, 5) {
		t.Fatalf("expected Allow to return false from local mock")
	}
	if mock.calls != 1 {
		t.Fatalf("expected local limiter to be called once, got %d", mock.calls)
	}
}

func TestHybridRateLimiter_RedisFallbackOnError(t *testing.T) {
	// Point to an invalid unroutable address
	badClient := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:59999",
		DialTimeout: 20 * time.Millisecond,
		ReadTimeout: 20 * time.Millisecond,
	})
	defer badClient.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mock := &mockRateLimiter{allowed: true}
	hl := NewHybridRateLimiter(mock, badClient, true, logger)

	start := time.Now()
	allowed := hl.Allow("fallback-key", 60, 5)
	duration := time.Since(start)

	if !allowed {
		t.Fatalf("expected Allow to return true via local fallback")
	}
	if mock.calls != 1 {
		t.Fatalf("expected local limiter to be called upon Redis error, got %d", mock.calls)
	}
	if duration > 2*time.Second {
		t.Fatalf("expected rapid fallback, took %v", duration)
	}
}

func TestHybridRateLimiter_LocalTokenBucketExhaustion(t *testing.T) {
	local := newTokenBucketLimiter()
	hl := NewHybridRateLimiter(local, nil, false, nil)

	key := "exhaustion-key"
	rpm := 60
	burst := 3

	// First 3 calls should pass
	for i := 0; i < 3; i++ {
		if !hl.Allow(key, rpm, burst) {
			t.Fatalf("call %d expected to be allowed within burst capacity", i+1)
		}
	}

	// 4th call should be rejected (burst exhausted)
	if hl.Allow(key, rpm, burst) {
		t.Fatalf("call 4 should be rejected when burst is exhausted")
	}
}
