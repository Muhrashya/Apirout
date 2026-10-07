# Specification: Deep Core Architecture, In-Memory Radix Router & Runtime Hardening

**Status:** completed

## Problem Statement

As an API platform operator and system architect, I encounter several critical operational risks, performance bottlenecks, and architectural frictions in the runtime:
1. **Deceptive GitOps Configuration**: When submitting declarative YAML/JSON configurations to the platform, the system returns an HTTP 200 OK success message, but silently drops the configuration without persisting any entity to the database, resulting in complete configuration loss.
2. **Gateway Ingress Database Overhead**: Inbound traffic on the gateway ingress hot path executes raw database queries against PostgreSQL on every single request to resolve routes, drastically limiting throughput, causing database connection pool saturation, and violating the sub-millisecond latency requirement.
3. **Severe Resource and Port Exhaustion**: Connecting to upstream databases via SSH bastion tunnels launches background forwarding goroutines and ephemeral listening ports that are never cleaned up, creating persistent socket and file descriptor leaks.
4. **Edge Traffic Blackholes**: Active health checking runs only within the control plane and does not broadcast health status updates to stateless data planes, leaving edge nodes routing production requests to crashed target servers.
5. **Code Divergence and Duplicate Maintenance**: Parallel, copy-pasted implementations of caches and query execution engines exist across control plane and data plane boundaries, making maintenance error-prone and bug fixes inconsistent.

## Solution

The platform introduces a unified deep core runtime architecture that replaces shallow stubs with hardened, resilient modules:
1. A **Declarative State Engine** that topologically sorts all imported infrastructure entities across an explicit dependency directed acyclic graph (DAG) and executes atomic, idempotent upserts inside a single database transaction with strict rollback on validation failure.
2. An **In-Memory Radix Router** operating at the data plane ingress seam that resolves paths via a segment prefix tree in sub-millisecond $O(K)$ time, handles parameterized segments and catch-alls, returns standard HTTP 405 Method Not Allowed for unsupported methods, and reconciles state without downtime via atomic pointer swaps upon EventBus invalidation events.
3. A **Managed Connection Pool** in the shared core that encapsulates database driver pooling, SSH bastion port forwarding, and background socket teardown, guaranteeing zero file descriptor leakage by strictly scoping tunnel lifecycles to database connection lifecycles.
4. An **Edge Passive Circuit Breaker** combined with EventBus health event synchronization that isolates failing upstream targets after consecutive errors, verifies recovery via half-open canary probes, and fast-fails with HTTP 503 when all targets in a pool are exhausted.
5. A **Unified Core Engine** in `internal/core/` consolidating caching, query execution, and dialect building to eliminate redundant implementations and ensure single-point locality.

## User Stories

1. As a platform operator, I want declarative GitOps configuration imports to be transactional and atomic, so that my database never ends up in a partially applied or corrupted state if a syntax or relational error occurs.
2. As a platform operator, I want declarative configuration re-imports to reconcile existing entities idempotently, so that continuous deployment pipelines can re-run without causing duplicate entity or conflict errors.
3. As a platform operator, I want declarative imports to respect dependency order (Consumer Groups and Connections before Services and Upstreams, Targets before Queries, and Queries before Routes), so that foreign key constraints are always satisfied.
4. As a platform operator, I want immediate, detailed error feedback when a declarative configuration fails validation, so that I can fix invalid references before deployment.
5. As an API client, I want inbound requests to be resolved and dispatched by an in-memory Radix tree with sub-millisecond overhead, so that my high-throughput workloads experience minimal gateway latency.
6. As an API client, I want route matching to correctly resolve exact paths, path parameters (e.g. `/users/:id`), and wildcard prefix paths according to strict specificity priority, so that my requests route to the intended destination.
7. As an API client, I want the gateway to return an HTTP 405 Method Not Allowed with an `Allow` header when I request an existing route with an unsupported HTTP method, so that I receive standard HTTP error semantics instead of generic 404 errors.
8. As a platform operator, I want route definitions to be synchronized across data plane instances via EventBus messages without restarting the gateway or dropping in-flight traffic, so that route updates achieve zero downtime.
9. As a platform operator, I want database connections configured with SSH bastion forwarding to reuse listening ports and connections cleanly, so that the host does not run out of file descriptors or ephemeral ports.
10. As a platform operator, I want SSH bastion tunnels to automatically shut down when their corresponding database connection pool is closed or evicted, so that zombie goroutines and hanging connections are eliminated.
11. As an API client, I want requests to avoid crashing backends when an upstream target experiences repeated network timeouts or 5xx responses, so that healthy targets absorb the traffic without failing the entire route.
12. As a platform operator, I want the data plane to automatically trip failing targets into a 15-second cooldown and probe them using a half-open canary request, so that target recovery is autonomous and resilient.
13. As an API client, I want the gateway to fast-fail with HTTP 503 Service Unavailable when 100% of targets in an upstream pool are down, so that client threads do not stall waiting for gateway timeouts.
14. As a platform operator, I want control-plane active health checks to broadcast target state changes via EventBus to data plane edge nodes, so that dead backends are dropped from load balancing pools cluster-wide.
15. As a core developer, I want database execution, connection pooling, and in-memory caching logic to live in a single unified core package, so that bug fixes and performance improvements automatically benefit both control plane and data plane runtimes.

## Implementation Decisions

### Declarative State Engine
- The declarative engine will be implemented as a deep module that accepts a complete declarative snapshot.
- Entity ingestion will be strictly ordered via topological sorting:
  1. Consumer Groups & Database Connections
  2. Services & Upstreams
  3. Targets & Consumers
  4. Standalone Queries
  5. Endpoints & Routes
- All entity operations will be executed within a single database transaction (`BEGIN ... COMMIT`).
- On any constraint violation, validation error, or missing dependency, the transaction will immediately roll back (`ROLLBACK`), returning a structured error indicating the offending entity and constraint.
- Existing entities matched by natural unique keys (e.g., name or slug within a workspace) or UUID will be idempotently updated (UPSERT) with declared attributes.
- Upon successful transaction commit, the engine will publish a `BULK_CONFIG_UPDATED` event to the EventBus port.

### In-Memory Radix Router Seam
- The router will be implemented as a Segment Trie (`RadixRouter`) where paths are tokenized by `/`.
- Trie node lookup will follow strict specificity priority: Exact Segment Match &gt; Parameterized Segment (`:param`) &gt; Catch-all Wildcard (`*`).
- Each leaf/matching node will maintain an HTTP method map (`map[string]*RouteDefinition`).
- If a path matches but the requested method does not exist in the node's method map, the router will produce an RFC-compliant HTTP 405 Method Not Allowed error alongside the allowed methods list.
- During gateway initialization, all active routes will be queried from the database once and compiled into the Radix tree.
- When an `InvalidationEvent` is received by the EventBus subscriber, a fresh Radix tree will be constructed and swapped into active duty using an atomic pointer swap, guaranteeing zero-lock contention on the hot path.

### Managed Connection Pool & Bastion Lifecycle
- A unified `ManagedConnectionPool` module in the shared core will absorb driver management, connection pooling, and SSH bastion port forwarding.
- SSH tunnels will be strictly scoped 1:1 to the connection identifier's `sql.DB` instance.
- The tunnel lifecycle will manage an authenticated SSH client, a local TCP listener on loopback, and bi-directional stream copying.
- When a connection pool is closed or evicted from cache, the tunnel's listener and SSH client will be cleanly closed, terminating background forwarding goroutines.
- Callers will interact solely through `pool.Acquire(connID)` without managing ports, addresses, or SSH configurations.

### Edge Passive Circuit Breaker & Target Health
- Each Upstream pool in the data plane will maintain atomic consecutive failure counters and state timestamps for its constituent targets.
- A target will transition from `HEALTHY` to `TRIPPED` after 3 consecutive connection timeouts or 5xx upstream responses.
- While `TRIPPED`, the target will be excluded from the load balancer for a 15-second cooldown period.
- After 15 seconds, the target will enter `HALF_OPEN` state, allowing exactly one canary request. A successful response resets the target to `HEALTHY`; a failed response resets the cooldown.
- If all targets in an upstream pool are in `TRIPPED` state, the gateway will fast-fail with HTTP 503 and a structured JSON payload (`{"error": "no healthy upstream targets available"}`).
- The control plane's active health checker will emit `TARGET_HEALTH_CHANGED` invalidation messages to the EventBus port upon target state transitions to notify all data plane nodes.

### Core Engine Consolidation
- The shared caching logic (`connection_pool`, `count_cache`, `endpoint_cache`) and query execution logic will be consolidated into `internal/core/`.
- Duplicate implementations currently existing in `internal/gateway/cache` and `internal/console/cache` will be removed.
- Control plane and data plane handlers will consume the unified core modules directly.

## Testing Decisions

- **Good Test Philosophy**: Tests must exercise external behavior across public interfaces rather than asserting private implementation variables or goroutine counts.
- **Seams Tested**:
  1. *Declarative Ingestion Seam*: Tested by executing declarative imports with valid and invalid payloads. Assertions verify that valid payloads result in all entities present in the datastore, and invalid payloads result in complete rollback with zero orphaned records.
  2. *Gateway Ingress Seam*: Tested by firing HTTP requests against the gateway ingress port. Assertions verify sub-millisecond route resolution, accurate parameter extraction, HTTP 405 Method Not Allowed responses, and atomic zero-downtime routing swaps upon simulated EventBus events.
  3. *Connection Pool Lifecycle Seam*: Tested by acquiring database connections with bastion configurations, executing queries, and verifying that pool eviction releases listening ports and terminates forwarding streams.
  4. *Circuit Breaker Seam*: Tested by simulating upstream target failures. Assertions verify traffic rerouting away from failing targets after 3 failures, fast-fail HTTP 503 when all targets are down, and canary recovery after cooldown.
- **Prior Art**: Existing integration tests in `internal/core/sqldialect/dialect_test.go` and `internal/core/eventbus/eventbus_test.go`.

## Out of Scope

- Introducing external service mesh sidecars (e.g. Istio or Envoy).
- Dynamic runtime compilation of custom Lua or WebAssembly plugins.
- Distributed consensus engines (e.g. Raft or etcd) for data plane state; Redis pub/sub and in-memory sync remain canonical.
- Modifying UI layouts or CSS styles in the web console frontend.

## Further Notes

- All terminology in this specification strictly aligns with `GLOSSARY.md`.
- Architectural decisions in this specification are codified in `docs/adr/0002-deep-core-architecture-and-runtime-hardening.md`.
