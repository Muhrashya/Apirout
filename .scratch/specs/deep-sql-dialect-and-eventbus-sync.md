# Spec: Deep SQL Dialect Engine and Formal Control-to-Data Plane EventBus Sync

## Problem Statement

Apirout operates as both a high-throughput API Gateway (routing, rate limiting, authentication) and a Database-as-an-API platform (Instant REST CRUD, query execution, dynamic filters). Currently, two architectural areas suffer from shallow abstractions and leaky seams:

1. **Dynamic SQL Dialect Fragmentation**: Dynamic SQL query generation for Instant CRUD and console query previews is implemented via procedural string concatenation scattered across `internal/gateway/executor/instant_crud.go`, `internal/console/executor/executor.go`, and `internal/console/executor/query_utils.go`. Identifier quoting (`"` vs `` ` ``), placeholder binding (`$1` for Postgres, `?` for MySQL, `@p1` for MSSQL), and pagination wrapping are inconsistently duplicated. This creates driver disparities (e.g. MSSQL offset pagination edge cases) and makes adding database engines error-prone.
2. **Implicit Cluster Invalidation Seam**: When routes, services, consumers, or auth plugins are modified in the Console control plane, the cache in the Gateway data plane is invalidated via ad-hoc strings published to Redis pub/sub. The seam is untyped, and because only a single production Redis implementation exists, testing cache invalidation currently requires running a live Redis server and two processes. This violates the two-adapter seam principle and prevents fast, isolated unit testing of cluster synchronization.

Importantly, all existing user-facing features, REST API endpoints, query parameters (`limit`, `offset`, `sort`, `filter`), and the entire web console UI design system must remain 100% identical and backwards-compatible.

## Solution

1. **Core SQL Dialect Engine (`internal/core/sqldialect`)**:
   - Introduce a deep, pure-computation module that accepts a structured `QuerySpec` (table, columns, filter conditions, sort order, pagination) and generates dialect-specialized, parameterized SQL alongside clean argument slices (`[]any`).
   - Implement first-class dialect generators for PostgreSQL, MySQL, and Microsoft SQL Server.
   - Refactor Instant CRUD and Console query executors to delegate all SQL formatting, identifier quoting, and pagination wrapping to this single tested engine.

2. **Formal EventBus Synchronization Port & Adapters (`internal/core/eventbus`)**:
   - Define a formal `EventBus` port at the seam between the Console Registry and Gateway Reader, utilizing strongly-typed domain events (`InvalidationEvent`).
   - Implement two concrete adapters satisfying the seam:
     - **Production Adapter (`RedisEventBus`)**: Publishes and subscribes to Redis pub/sub topics with JSON-encoded event envelopes.
     - **In-Memory Adapter (`InMemoryEventBus`)**: Enables deterministic, zero-dependency unit and integration testing where mutating an entity in the control plane directly verifies gateway cache eviction in memory.

## User Stories

1. As an API consumer using Instant CRUD, I want to query, insert, update, and delete database records using standard REST endpoints, so that my queries execute with optimal parameterized performance regardless of whether the target database is PostgreSQL, MySQL, or SQL Server.
2. As an API consumer, I want complex query filters (e.g., `status=eq.active`, `age=gte.21`, `name=ilike.*john*`) to produce valid parameterized SQL for my specific database engine, so that my applications never experience SQL syntax errors or injection vulnerabilities.
3. As a database administrator, I want table names and column identifiers to be automatically and safely quoted using dialect-appropriate characters (`"` for Postgres/MSSQL, `` ` `` for MySQL), so that reserved keywords and special characters in schema objects never break queries.
4. As a web console user testing queries in the query editor, I want query pagination previews to work reliably on all supported database engines including Microsoft SQL Server (`OFFSET X ROWS FETCH NEXT Y ROWS ONLY`), so that I can inspect result sets with consistent limits.
5. As a backend engineer extending Apirout, I want all dynamic SQL formatting logic to live in a single deep module with zero HTTP or database dependencies, so that I can add new filter operators or dialect optimizations in one place without touching HTTP handlers.
6. As a platform operator running multi-replica clusters, I want route and consumer modifications in the Control Plane to immediately invalidate Data Plane memory caches across all cluster nodes, so that changes take effect instantly.
7. As a site reliability engineer, I want cache invalidation messages to be strongly-typed and structured, so that invalidation payloads never fail due to string parsing mismatches or channel naming typos.
8. As a QA engineer, I want to test that updating an auth plugin in the control plane invalidates the gateway's cached authentication credentials in in-memory unit tests without running Docker or Redis, so that test suites run in milliseconds on any developer machine.
9. As a frontend maintainer, I want all Instant CRUD and Admin API URL structures, query parameters, and response JSON formats to remain unchanged, so that existing React UI forms and dashboards require zero modifications.
10. As a security auditor, I want all dynamic queries to be strictly parameterized with positional argument bindings rather than raw string concatenation, so that SQL injection risks are architecturally eliminated.

## Implementation Decisions

### 1. SQL Dialect Engine (`internal/core/sqldialect`)
- **Location**: Pure package in `internal/core/sqldialect` with no network, database, or HTTP dependencies.
- **Abstract Model**: Defines structured types:
  - `QuerySpec`: Target dialect, table name, selected columns, filter conditions, sort fields, pagination (limit, offset).
  - `FilterCondition`: Field name, operator (`eq`, `neq`, `gt`, `gte`, `lt`, `lte`, `like`, `ilike`, `in`, `is_null`, `not_null`), value.
  - `InsertSpec` / `UpdateSpec` / `DeleteSpec`: Table name, primary key / predicate, column value mappings.
- **Engine Interface**:
  - `BuildSelect(spec QuerySpec) (sql string, args []any, err error)`
  - `BuildInsert(spec InsertSpec) (sql string, args []any, err error)`
  - `BuildUpdate(spec UpdateSpec) (sql string, args []any, err error)`
  - `BuildDelete(spec DeleteSpec) (sql string, args []any, err error)`
  - `WrapPagination(rawSQL string, dialect string, limit, offset int) (string, error)`
- **Dialect Implementations**: Specialized generator strategies for `postgres`, `mysql`, and `sqlserver` (handling placeholder syntax `$1` vs `?` vs `@p1`, quoting, and pagination idioms).
- **Caller Refactoring**:
  - `internal/gateway/executor/instant_crud.go` replaces internal `quoteIdentifier`, `buildWhereClause`, and query strings with `sqldialect.Engine`.
  - `internal/console/executor/query_utils.go` delegates `WrapWithPagination` to `sqldialect.Engine`.

### 2. EventBus Synchronization Seam (`internal/core/eventbus`)
- **Location**: Core domain package in `internal/core/eventbus`.
- **Domain Event Model**:
  - `InvalidationEvent`: Entity type (Route, Service, Upstream, Consumer, Plugin, Connection), Entity ID (UUID or `*` for bulk), Action (Created, Updated, Deleted), and Timestamp.
- **Port Interface**:
  - `Publish(ctx context.Context, event InvalidationEvent) error`
  - `Subscribe(ctx context.Context, handler func(event InvalidationEvent)) error`
  - `Close() error`
- **Adapters**:
  - `RedisEventBus`: Production adapter utilizing Redis pub/sub with JSON serialization over a unified topic `apirout:events:invalidation`.
  - `InMemoryEventBus`: Zero-dependency test and single-process adapter using Go channels and fan-out mutex dispatch.
- **Caller Refactoring**:
  - `internal/console/registry/registry.go` replaces direct `r.redis.Publish` string calls with `eventbus.EventBus.Publish`.
  - `internal/gateway/cache/redis_subscriber.go` and `internal/gateway/registry/reader.go` consume events via `eventbus.EventBus.Subscribe`.

### 3. Backwards Compatibility Guarantees
- Zero changes to `web/` console files.
- Zero changes to HTTP path structures or URL query parameter parsing.
- Zero changes to database schema or migration scripts.

## Testing Decisions

### Test Seams
1. **SQL Dialect Generation Seam (Highest Seam)**: Table-driven unit tests executing `BuildSelect`, `BuildInsert`, `BuildUpdate`, `BuildDelete`, and `WrapPagination` across Postgres, MySQL, and SQL Server. Tests assert on exact SQL syntax, quoting, and positional argument arrays with 100% in-memory execution.
2. **EventBus Adapter Seams**:
   - `InMemoryEventBus`: Test concurrent publish/subscribe, multi-subscriber fanout, and graceful shutdown.
   - `RedisEventBus`: Test JSON marshaling, network reconnection resilience, and Redis pub/sub integration.
3. **End-to-End Cache Invalidation Seam**: Test that creating or updating a Route in `registry.Registry` using an `InMemoryEventBus` immediately notifies `registry.Reader` and evicts the corresponding route from gateway memory without a running Redis server.
4. **Existing Gateway & Instant CRUD Tests**: Full test suite (`go test ./...`) execution ensuring that all existing CRUD endpoints, authentication plugins, and query execution pass without regression.

### Prior Art
- Existing unit tests in `internal/core/crypto`, `internal/core/iputil`, and `internal/gateway/executor/validator_test.go` utilize standard Go table-driven tests.
- Existing audit and registry tests in `internal/console/registry/audit_test.go`.

## Out of Scope

- Implementing arbitrary raw SQL statement AST parsing (e.g. parsing complex subqueries or CTEs from arbitrary user strings).
- Supporting non-relational database query dialect generation (MongoDB queries and Redis commands continue using their native client drivers).
- Introducing external message queue infrastructure (Kafka, RabbitMQ, NATS).

## Further Notes

- Target GitHub repository is `Muhrashya/Apirout`.
- All code must follow the project's domain vocabulary in `GLOSSARY.md`.
