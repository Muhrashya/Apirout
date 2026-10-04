# Ticket 4: Control-to-Data Plane Invalidation Wiring & In-Memory Verification

## Parent
Part of #11

## What to build
Wire the Console Registry to publish invalidations via the `EventBus` port and the Gateway Reader/Subscriber to listen for invalidations, accompanied by an end-to-end integration test verifying that creating or updating an entity in the Console immediately evicts the Gateway's in-memory cached state without requiring a running Redis instance.

## Acceptance criteria
- [ ] Refactor `internal/console/registry/registry.go` to publish typed `InvalidationEvent` to `EventBus`
- [ ] Refactor `internal/gateway/cache/redis_subscriber.go` or reader to consume from `EventBus`
- [ ] Add an in-memory integration test verifying that Registry route/consumer mutation invalidates Gateway cache
- [ ] Verify production multi-container setup continues functioning with Redis
- [ ] Full backend test suite (`go test ./...`) passes with 0 failures

## Blocked by
- Ticket 3
