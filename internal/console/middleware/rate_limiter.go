package middleware

import (
	"github.com/apirout/apirout/internal/core/httputil"
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements a simple token bucket rate limiter per IP
type RateLimiter struct {
	mu         sync.Mutex
	requests   map[string]*bucket
	rate       int           // requests per window
	window     time.Duration // time window
	enabled    bool
	cleanupInt time.Duration

	done     chan struct{}
	stopOnce sync.Once
}

type bucket struct {
	tokens    int
	lastReset time.Time
}

// NewRateLimiter creates a new rate limiter
// rate: max requests per window, window: time window duration
func NewRateLimiter(rate int, window time.Duration, enabled bool) *RateLimiter {
	rl := &RateLimiter{
		requests:   make(map[string]*bucket),
		rate:       rate,
		window:     window,
		enabled:    enabled,
		cleanupInt: 5 * time.Minute,
		done:       make(chan struct{}),
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Stop terminates the background cleanup goroutine. It is safe to call
// multiple times.
func (rl *RateLimiter) Stop() {
	rl.stopOnce.Do(func() {
		close(rl.done)
	})
}

// Handler returns the rate limiting middleware
func (rl *RateLimiter) Handler(next http.Handler) http.Handler {
	if !rl.enabled {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := httputil.GetClientIP(r)

		if !rl.allow(ip) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"Rate limit exceeded. Please try again later."}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// allow checks if request from IP is allowed
func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, exists := rl.requests[ip]

	if !exists {
		rl.requests[ip] = &bucket{
			tokens:    rl.rate - 1,
			lastReset: now,
		}
		return true
	}

	// Reset bucket if window has passed
	if now.Sub(b.lastReset) > rl.window {
		b.tokens = rl.rate - 1
		b.lastReset = now
		return true
	}

	// Check if tokens available
	if b.tokens > 0 {
		b.tokens--
		return true
	}

	return false
}

// cleanup removes expired entries periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupInt)
	defer ticker.Stop()

	for {
		select {
		case <-rl.done:
			return
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for ip, b := range rl.requests {
				if now.Sub(b.lastReset) > rl.window*2 {
					delete(rl.requests, ip)
				}
			}
			rl.mu.Unlock()
		}
	}
}

// SetEnabled enables or disables rate limiting
func (rl *RateLimiter) SetEnabled(enabled bool) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.enabled = enabled
}

// SetRate updates the rate limit
func (rl *RateLimiter) SetRate(rate int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.rate = rate
}

// Stats returns current stats
func (rl *RateLimiter) Stats() map[string]interface{} {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return map[string]interface{}{
		"enabled":     rl.enabled,
		"rate":        rl.rate,
		"window":      rl.window.String(),
		"active_keys": len(rl.requests),
	}
}
