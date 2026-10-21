# 85: Rate Limiter Anonymous Key Guard and Client IP Shield

**What to build:**
Eliminate rate-limiter bucket collisions for anonymous and public traffic in the execution pipeline. Requests without an authenticated, non-empty consumer ID must be strictly isolated by client IP address (`route:<id>:ip:<client_ip>`), preventing one caller's traffic from exhausting quotas for all other unauthenticated callers.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] In `internal/gateway/executor/stage_auth.go`, verify `c.ConsumerID != nil && *c.ConsumerID != ""` before deriving a client-specific rate limit key.
- [x] Fall back cleanly to `route:<id>:ip:<client_ip>` when consumer ID is nil or empty string.
- [x] Skip consumer group rate limiting if `c.ConsumerID` is empty.
- [x] Add unit test verifying that two anonymous requests with different IP addresses receive distinct rate limit keys and independent token buckets.

