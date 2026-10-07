# 32: Edge Passive Circuit Breaker with Upstream Health Synchronization

**What to build:** An edge-local passive circuit breaker in the Data Plane ingress loop that isolates failing targets after consecutive network/5xx errors, probes recovery via half-open canary requests, fast-fails with HTTP 503 when all targets are down, and syncs with Control Plane health checks via EventBus.

**Blocked by:** 30: In-Memory Radix Router Seam with Sub-Millisecond Specificity Matching

**Status:** completed

- [x] Create `internal/core/upstream/circuit_breaker.go` tracking per-target failure counters and state timestamps (`HEALTHY`, `TRIPPED`, `HALF_OPEN`).
- [x] Configure threshold: 3 consecutive network timeouts or 5xx responses trip a target into a 15-second cooldown.
- [x] Implement half-open state allowing 1 canary request through; success resets the target to `HEALTHY`, failure re-trips the cooldown.
- [x] If all targets in an upstream pool are in `TRIPPED` state, immediately fast-fail with HTTP `503 Service Unavailable` and JSON `{"error": "no healthy upstream targets available"}`.
- [x] In `internal/console/server/upstreams_handler.go`, publish `TARGET_HEALTH_CHANGED` invalidation events via EventBus when active background health checks update target status.
- [x] Subscribe Data Plane upstream pool to `TARGET_HEALTH_CHANGED` events to immediately remove dead targets cluster-wide.
- [x] Unit and integration tests verify circuit tripping, canary recovery, fast-fail 503 responses, and EventBus target health synchronization.
