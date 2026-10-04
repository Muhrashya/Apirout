# 15: EventBus Action Parsing and Subscription Lifecycle Hardening

**What to build:** Formal domain action parsing on `InvalidationEvent` so all action strings (`CREATE`, `UPDATE`, `DELETE`, `PURGE`) are parsed consistently through a single core domain method, alongside a leak-free Redis EventBus subscription lifecycle that terminates background listener routines whenever the parent context is cancelled.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] A domain method `ParseActionType(str string) ActionType` is introduced in `internal/core/eventbus/event.go` that safely maps uppercase, lowercase, and alias mutation actions to canonical domain action types.
- [ ] Console publisher and registry components delegate string action parsing to `ParseActionType`, eliminating duplicated ad-hoc switch cascades.
- [ ] `RedisEventBus.Subscribe` correctly handles context cancellation and cleans up the active Pub/Sub subscription and channels without leaving orphaned goroutines or connections.
- [ ] Unit tests in `bus_test.go` and `integration_test.go` verify action parsing accuracy and leak-free cancellation.
