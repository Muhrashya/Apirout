# 94: Dynamic Wildcard Path Stripping and Synthetic Direct Circuit Breaking

**What to build:**
Fix path forwarding for routes with path wildcards or URL parameters (e.g. `/api/v1/users/*`), calculating the static route prefix so that upstream backends receive clean relative sub-paths. In addition, generate deterministic synthetic target IDs (`direct:<service_id>`) for direct backend services, enabling edge passive circuit breakers to isolate failing standalone backends on consecutive 5xx errors.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] In `internal/gateway/executor/handler.go` (`reverseProxyRequest`), calculate sub-paths by stripping the static route prefix up to the first wildcard (`*`) or parameter (`:id`) segment.
- [x] In `internal/gateway/executor/stage_upstream.go` (Case B: Direct Backend), generate a synthetic target ID `direct:<service_id>` and pass it to `reverseProxyRequest`.
- [x] Ensure `HandleProxyError` and `ModifyProxyResponse` update circuit breaker failure counters when using synthetic direct target IDs, and verify that `FilterAvailable` or direct target routing fast-fails when the direct circuit breaker trips.
- [x] Write unit tests verifying that wildcard route path stripping correctly extracts sub-paths (e.g. `/api/v1/users/*` with request `/api/v1/users/profile` sends `/profile`), and that consecutive 502/503 errors against direct backends trip the circuit breaker.
