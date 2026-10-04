# Ticket 3: Formal EventBus Port with In-Memory & Redis Pub/Sub Adapters

## Parent
Part of #11

## What to build
A formal domain package `internal/core/eventbus` defining strongly-typed `InvalidationEvent` domain envelopes and the `EventBus` port (`Publish`, `Subscribe`, `Close`), complete with two tested adapters: `InMemoryEventBus` (using Go channels and fanout mutexes for zero-dependency test suites) and `RedisEventBus` (for multi-replica production clusters).

## Acceptance criteria
- [ ] Define `InvalidationEvent` struct with `EntityType`, `EntityID`, `Action`, and `Timestamp`
- [ ] Define `EventBus` interface with `Publish(ctx, event)`, `Subscribe(ctx, handler)`, and `Close()`
- [ ] Implement `InMemoryEventBus` adapter with channel-based fanout for fast concurrent testing
- [ ] Implement `RedisEventBus` adapter wrapping Redis pub/sub with JSON serialization
- [ ] Unit tests verifying thread-safety, multi-subscriber fanout, and graceful shutdown

## Blocked by
- None (can start immediately)
