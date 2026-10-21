package executor

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter evaluates whether a request identified by key is allowed within rate limits
type RateLimiter interface {
	Allow(key string, rpm int, burst int) bool
}

// tokenBucketLuaScript runs an atomic token-bucket rate limit calculation inside Redis.
// KEYS[1] = rate limit key (e.g. apirout:ratelimit:route:<id>:ip:<ip>)
// ARGV[1] = rpm (requests per minute)
// ARGV[2] = burst capacity
// ARGV[3] = current timestamp in milliseconds
var tokenBucketLuaScript = redis.NewScript(`
local key = KEYS[1]
local rpm = tonumber(ARGV[1])
local burst = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

local rate = rpm / 60.0
local data = redis.call('HMGET', key, 'tokens', 'last_update')
local tokens = tonumber(data[1])
local last_update = tonumber(data[2])

if not tokens then
    tokens = burst - 1
    last_update = now
    redis.call('HMSET', key, 'tokens', tokens, 'last_update', last_update)
    redis.call('EXPIRE', key, 120)
    return 1
end

local elapsed = math.max(0, (now - last_update) / 1000.0)
tokens = math.min(burst, tokens + elapsed * rate)

if tokens >= 1.0 then
    tokens = tokens - 1.0
    redis.call('HMSET', key, 'tokens', tokens, 'last_update', now)
    redis.call('EXPIRE', key, 120)
    return 1
else
    redis.call('HMSET', key, 'tokens', tokens, 'last_update', now)
    redis.call('EXPIRE', key, 120)
    return 0
end
`)

// HybridRateLimiter coordinates rate limiting between a centralized Redis token bucket
// and an in-memory fallback token bucket.
type HybridRateLimiter struct {
	local   RateLimiter
	rdb     *redis.Client
	enabled bool
	logger  *slog.Logger
}

// NewHybridRateLimiter creates a hybrid rate limiter wrapping a local limiter and Redis client.
func NewHybridRateLimiter(local RateLimiter, rdb *redis.Client, enabled bool, logger *slog.Logger) *HybridRateLimiter {
	if local == nil {
		local = newTokenBucketLimiter()
	}
	return &HybridRateLimiter{
		local:   local,
		rdb:     rdb,
		enabled: enabled,
		logger:  logger,
	}
}

// Allow evaluates rate limit permissions. If Redis is enabled and accessible, it uses Redis.
// On any Redis timeout or error, it safely falls back to the in-memory token bucket.
func (h *HybridRateLimiter) Allow(key string, rpm int, burst int) bool {
	if rpm <= 0 {
		return true
	}
	if burst <= 0 {
		burst = rpm / 6
		if burst < 1 {
			burst = 1
		}
	}

	if !h.enabled || h.rdb == nil {
		return h.local.Allow(key, rpm, burst)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	redisKey := "apirout:ratelimit:" + key
	nowMs := time.Now().UnixMilli()

	res, err := tokenBucketLuaScript.Run(ctx, h.rdb, []string{redisKey}, rpm, burst, nowMs).Int()
	if err != nil {
		if h.logger != nil {
			h.logger.Debug("Distributed Redis rate limiter failed or timed out, falling back to local bucket", "key", key, "error", err)
		}
		return h.local.Allow(key, rpm, burst)
	}

	return res == 1
}

// Local returns the underlying in-memory rate limiter
func (h *HybridRateLimiter) Local() RateLimiter {
	return h.local
}

// RedisClient returns the underlying Redis client
func (h *HybridRateLimiter) RedisClient() *redis.Client {
	return h.rdb
}
