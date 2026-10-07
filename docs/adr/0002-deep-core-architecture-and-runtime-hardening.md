# ADR-0002: Deep Core Architecture, In-Memory Radix Router & Runtime Hardening

## Status
Accepted

## Context
An architecture audit of the Apirout platform identified key architectural friction points:
1. **Deceptive GitOps Import Stub**: The declarative import handler accepted YAML/JSON specifications, returned HTTP 200 OK, but executed zero database operations, silently dropping configuration.
2. **Database Hits in Ingress Hot Path**: Inbound gateway requests at `:8000` executed PostgreSQL queries on every request for route matching, violating the advertised sub-millisecond in-memory routing latency.
3. **SSH Bastion Tunnel Resource Leakage**: Database connection setup instantiated background SSH tunnels on ephemeral ports without lifecycle tracking or teardown, causing unbounded socket and goroutine exhaustion.
4. **Plane Health Check Disconnect**: Active upstream health checks ran exclusively in the Control Plane and did not broadcast state changes to edge Data Plane nodes, leading to traffic blackholes.
5. **Shallow Code Duplication**: Six near-identical cache and query executor files were duplicated across `internal/gateway/` and `internal/console/`.

## Decision
1. **Deep Declarative State Engine**: Replace the shallow GitOps handler with a transactional engine that performs topological DAG sorting across all entities (Groups, Connections, Services, Upstreams, Targets, Consumers, Queries, Routes) and executes idempotent UPSERTs inside a single atomic database transaction.
2. **In-Memory Radix Router Seam**: Introduce a Segment Trie module (`RadixRouter`) with $O(K)$ lookup complexity (Exact > Param > Wildcard priority). Preload active routes in memory at boot, serve hot-path requests with zero database overhead, return HTTP 405 when paths match unsupported methods, and swap router pointers atomically upon receiving `InvalidationEvent` messages from EventBus.
3. **Managed Connection Pool & Tunnel Lifecycle**: Deepen `ManagedConnectionPool` in `internal/core/` to absorb SSH bastion tunnel setup, port forwarding, and teardown. Tunnels are strictly scoped 1:1 to the `sql.DB` connection pool lifecycle, ensuring complete socket teardown on eviction or shutdown.
4. **Core Engine Consolidation**: Unify query execution (`executor`), dialect builders, and connection pools into `internal/core/`, eliminating duplicate files in `internal/gateway/` and `internal/console/`.
5. **Edge Passive Circuit Breaker**: Introduce a low-overhead passive circuit breaker at Data Plane ingress. Targets trip after 3 consecutive network/5xx failures into a 15-second cooldown before probing recovery via a half-open canary request. If 100% of targets in a pool are tripped, the gateway fast-fails with HTTP 503 Service Unavailable.

## Consequences
- **Positive**: Eliminates fake feature stubs; achieves true sub-millisecond gateway routing latency; eliminates socket/goroutine leaks; provides immediate failover for dead upstream targets; concentrates maintenance and testing in deep core modules.
- **Trade-off**: Data plane nodes require sufficient memory to hold the full route tree; declarative import enforces strict dependency ordering and fails atomically if relational constraints are violated.
