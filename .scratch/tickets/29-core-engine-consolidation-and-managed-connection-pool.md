# 29: Core Engine Consolidation & Managed Connection Pool with SSH Bastion Lifecycle

**What to build:** Unify the caching and database execution engines into `internal/core/` to eliminate duplicate maintenance across planes, and embed SSH bastion tunnel forwarding and socket teardown directly into `ManagedConnectionPool` to eliminate socket and file descriptor leaks.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] Create `internal/core/cache/` providing thread-safe `ManagedConnectionPool`, `CountCache`, and `EndpointCache`.
- [x] Implement `ManagedConnectionPool` with automatic SSH bastion port forwarding scoping: starting a forwarding listener when a connection with bastion settings is opened, and cleanly tearing down the listener and SSH client when the connection pool is closed or evicted.
- [x] Migrate `internal/gateway/executor/executor.go` and `internal/console/executor/executor.go` to consume the consolidated core connection pool and cache modules, removing duplicate cache implementations.
- [x] Unit tests for `ManagedConnectionPool` verify that acquiring connections manages tunnels properly and closing/evicting pools terminates forwarding listeners without leaving leaked goroutines or open sockets.
- [x] Existing database execution and CRUD tests continue to pass green with zero regressions.
