# 31: Zero-Downtime EventBus Synchronization & Dynamic Radix Tree Pointer Swapping

**What to build:** Real-time event-driven routing tree synchronization linking the Control Plane configuration publisher to the Data Plane `RadixRouter` via the EventBus port, swapping router pointers atomically without downtime or in-flight request disruption.

**Blocked by:** 30: In-Memory Radix Router Seam with Sub-Millisecond Specificity Matching

**Status:** completed

- [x] Subscribe `internal/gateway/executor/handler.go` to the EventBus port for `ENDPOINT_CREATED`, `ENDPOINT_UPDATED`, and `ENDPOINT_DELETED` events.
- [x] Implement atomic router pointer swapping (`atomic.Pointer[RadixRouter]` or double-buffered reader locks) to install new route trees without blocking concurrent reader requests or dropping in-flight traffic.
- [x] Provide a thread-safe hot reload mechanism that re-indexes routes into a shadow tree before performing the atomic swap.
- [x] Integration tests verify that publishing invalidation events triggers an immediate route table update and new routes resolve without gateway restarts or 500 errors.
