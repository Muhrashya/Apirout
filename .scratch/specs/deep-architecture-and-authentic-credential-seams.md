# Deep Architecture and Authentic Credential Seams Specification

**Status:** ready-for-agent

## Problem Statement

Operators and developers using Apirout currently experience architectural friction, silent runtime failures, and unbacked pseudo-features across authentication, database connectivity, and upstream routing:

1. **Fake Outbound Authentication**: When configuring a service to proxy to AWS services (API Gateway, S3, Lambda) using AWS Signature V4, or to modern microservices using OAuth2 Client Credentials, the gateway emits dummy custom headers (`X-Apirout-AWS-*`, `X-Apirout-OAuth2-Client`) instead of computing real RFC-compliant cryptographic signatures or executing client-credentials token exchanges. Upstream providers immediately reject these calls with 401 or 403 errors.
2. **Phantom Inbound OAuth2 Credentials**: The consumer management interface offers an "OAuth 2.0 Client Secret" credential type, but the gateway data plane lacks an inbound OAuth 2.0 authorization server or token evaluation plugin, creating unresolvable 401 authentication lockouts for consumer applications.
3. **Fragmented Inbound Auth Seams**: Inbound authentication logic is split across multiple shallow packages, requiring callers to orchestrate separate steps for credential evaluation and route-level authorization.
4. **Disconnected Database Engine Support**: External database connectivity leaks across four shallow modules that disagree on supported engines. The connection pool supports Microsoft SQL Server but rejects SQLite; the schema inspector supports PostgreSQL and MySQL but rejects Microsoft SQL Server and SQLite; and instant CRUD hardcodes a fallback to PostgreSQL for all non-MySQL connections. The web console advertises SQLite as an external database target despite missing target driver linkage, causing silent runtime crashes.
5. **Pseudo Consistent Hashing**: Upstream load balancing advertises "Consistent Hashing" but implements naive modulo arithmetic (`hash % N`). When the target pool autoscales, 100% of keys are reshuffled, destroying upstream cache locality and sticky session affinity, while connection tracking leaks memory through unbounded global maps.

## Solution

Consolidate and deepen the Apirout runtime architecture into four robust, cohesive modules adhering to deep module principles:

1. **Authentic Outbound Credential Negotiator**: Replace pseudo-header injection with an authentic upstream authorization module that calculates standard RFC-compliant AWS SigV4 request signatures and autonomously manages OAuth2 Client Credentials token grant lifecycles with thread-safe in-memory TTL caching, proactive expiration buffers, and singleflight request deduplication.
2. **Unified Inbound Auth Engine & Phantom Purge**: Absorb standalone client authentication services directly into the central auth engine, presenting a single ingress seam that evaluates plugins and verifies route permissions in one atomic pass. Completely purge all residual phantom OAuth2 credential options and dead code branches across the console and data models.
3. **Deep Database Adapter Seam**: Replace the four fragmented database packages with a unified Database Adapter module where each supported engine (PostgreSQL, MySQL, Microsoft SQL Server) provides a complete, tested adapter encapsulating connection pooling, SSH bastion tunneling, schema introspection, dialect query generation, and instant CRUD row operations behind a single seam. Formally restrict SQLite to internal metadata storage and remove it from external target connection forms.
4. **Target Pool with Ketama Consistent Hash Ring**: Extract target selection into an isolated module implementing a standard Ketama consistent hash ring with 160 virtual nodes per target, ensuring minimal ($1/N$) key remapping during target pool scaling, paired with bounded connection lease tracking.

---

## User Stories

### Outbound Authentication (AWS SigV4 & OAuth2 Client Credentials)
1. As an API operator, I want Apirout to calculate authentic AWS Signature V4 request signatures, so that the gateway can securely proxy requests to private AWS API Gateway, Lambda, and S3 endpoints without being rejected with 403 Forbidden.
2. As an API operator, I want AWS SigV4 signing to automatically calculate the SHA-256 payload digest, canonical request string, and credential scope, so that I only need to supply the Access Key, Secret Key, Region, and Service in the console.
3. As an API operator, I want Apirout to negotiate OAuth2 Client Credentials grants against third-party token endpoints, so that the gateway can authenticate with OAuth2-protected enterprise microservices on my behalf.
4. As an API operator, I want OAuth2 access tokens to be cached in memory with automatic TTL accounting, so that the gateway does not introduce latency by requesting a new token on every proxied request.
5. As an API operator, I want OAuth2 tokens to be proactively refreshed when less than 60 seconds remain before expiration, so that in-flight requests never encounter expired token rejections from upstreams.
6. As a platform engineer, I want concurrent proxy requests awaiting an OAuth2 token to share a single in-flight token exchange via singleflight deduplication, so that authorization servers (Okta, Keycloak, Auth0) are never overwhelmed by thundering herd spikes.
7. As an API operator, I want the gateway to stop injecting non-standard pseudo-headers like `X-Apirout-AWS-*` and `X-Apirout-OAuth2-Client`, so that outbound traffic adheres strictly to RFC standards.
8. As a security auditor, I want outbound client secrets and AWS secret keys to remain encrypted at rest and redacted from console API responses, so that sensitive credentials cannot be harvested from browser developer tools.

### Inbound Authentication & Phantom Purge
9. As a consumer developer, I want to create API Key, Basic Auth, JWT, and HMAC credentials for my consumer identity, so that my applications can authenticate against protected gateway routes.
10. As a console administrator, I want the consumer credential type dropdown to display only genuine, supported authentication types (`api-key`, `basic-auth`, `jwt`, `hmac-auth`), so that I am never misled into creating phantom credentials that cause runtime authentication lockouts.
11. As a security engineer, I want inbound authentication and route-level authorization to be evaluated in a single atomic pass, so that unauthorized consumers cannot bypass access checks due to race conditions or inconsistent stage ordering.
12. As a gateway operator, I want all dead code branches and UI forms for the phantom inbound `oauth2` plugin to be completely removed from the console, so that route configuration options strictly mirror genuine data plane capabilities.
13. As a developer writing tests, I want to test inbound authentication through a single engine seam, so that test setup is simple and verifies the exact behavior experienced by production callers.

### Unified Database Adapter Seam (PostgreSQL, MySQL, MSSQL)
14. As an API developer, I want to connect PostgreSQL, MySQL, and Microsoft SQL Server databases through a single unified Database Adapter seam, so that instant CRUD and schema introspection behave identically across all supported database engines.
15. As an API developer, I want Microsoft SQL Server schemas to be fully introspectable in the console, so that I can discover tables, columns, and primary keys on MSSQL databases just as I can on PostgreSQL and MySQL.
16. As an API developer, I want Instant REST CRUD endpoints (`/api/v1/instant/:connId/:table`) to execute dialect-accurate SQL queries on Microsoft SQL Server targets, so that MSSQL connections do not fail by falling back to PostgreSQL drivers.
17. As an API operator, I want the external database connection form in the console to present only genuine, supported target database types (PostgreSQL, MySQL, MSSQL), so that I am not offered an unbacked SQLite target option that crashes at runtime.
18. As a system administrator, I want SQLite (`modernc.org/sqlite`) to remain dedicated exclusively to internal metadata registry storage, so that single-binary deployments remain lightweight without creating file-locking conflicts from external database traffic.
19. As a database administrator, I want target database connection pools and SSH bastion tunnels to be encapsulated within each engine's adapter, so that idle connections and sockets are cleanly torn down without resource leaks.
20. As a developer adding support for a new database engine in the future, I want to implement a single cohesive `DatabaseAdapter` interface in one file, so that changes do not leak across four separate packages.

### Target Pool & Ketama Consistent Hash Ring
21. As a platform operator, I want upstream target pools configured with Consistent Hashing to use a real Ketama hash ring with 160 virtual nodes per target, so that requests with the same hash key consistently reach the same backend target.
22. As a backend developer, I want target autoscaling events (adding or removing an upstream instance) to redistribute at most $1/N$ of keys, so that upstream cache hit rates remain high and session state is preserved for unaffected targets.
23. As an API consumer, I want consistent hashing to evaluate `X-Consumer-ID` first, then `X-User-ID`, and fall back to the authentic client IP, so that my requests maintain sticky routing even when routing headers vary.
24. As an SRE, I want active connection tracking on upstream targets to be scoped to active request leases with automated lifecycle release, so that memory usage remains bounded and free of leaks during high-throughput operations.
25. As a gateway maintainer, I want target selection logic to live in an isolated `TargetPool` module with its own test suite, so that the main request handler does not need to manage balancing algorithms or atomic counters.

---

## Implementation Decisions

### Phase 1: Authentication Integrity & Authentic Credential Negotiator

1. **Outbound Credential Negotiator Module**:
   - The outbound authentication mechanism will be refactored into a deep `OutboundNegotiator` module operating at the reverse proxy upstream seam.
   - It will encapsulate:
     - **AWS SigV4 Signing**: A dedicated signer that computes canonical request URIs, query strings, headers, payload SHA-256 hashes, credential scope (`date/region/service/aws4_request`), and the HMAC-SHA256 signature, attaching standard `Authorization: AWS4-HMAC-SHA256 ...` and `X-Amz-Date` headers.
     - **OAuth2 Client Credentials Grant Manager**: A dedicated grant engine that accepts `token_url`, `client_id`, `client_secret_encrypted`, and optional `scopes`. It issues HTTP POST requests (`grant_type=client_credentials`), parses the JSON response (`access_token`, `expires_in`, `token_type`), and caches tokens in memory.
     - **Singleflight Deduplication**: All concurrent proxy requests requiring a token for the same service use a Go `singleflight.Group` to execute at most one HTTP grant request simultaneously.
     - **Proactive Expiration Buffer**: Cached tokens are considered expired if fewer than 60 seconds remain before `expires_in`, ensuring in-flight upstream requests do not fail due to boundary clock skew.
   - The interface presented to the reverse proxy dispatcher will be:
     ```go
     type OutboundNegotiator interface {
         AuthorizeRequest(req *http.Request, outboundAuth *models.ServiceOutboundAuth) error
     }
     ```
   - Dummy pseudo-header injections (`X-Apirout-AWS-*`, `X-Apirout-OAuth2-Client`) will be completely eliminated.

2. **Inbound Auth Engine Consolidation**:
   - `apiclientauth.Service` will be absorbed directly into `auth.Engine`.
   - The unified engine will expose a single seam:
     ```go
     func (e *Engine) Authenticate(r *http.Request, endpoint *models.Endpoint) (*ConsumerContext, error)
     ```
   - Credential validation and consumer route permission verification (`reader.IsAPIClientAllowedForEndpoint`) will be executed in a single call, returning a verified context containing the consumer identity, group ID, and claims, or a typed canonical error.
   - `Consumers.tsx` will be cleaned up to remove the phantom `<option value="oauth2">` entry.
   - `AuthPluginManager.tsx` and data models will have all dead `oauth2` plugin branches and enums purged, aligning the console strictly with the four authentic plugins (`key-auth`, `basic-auth`, `jwt-auth`, `hmac-auth`).

### Phase 2: Consolidated Database Adapter Seam

3. **Unified Database Adapter Architecture**:
   - The four disparate packages (`internal/core/cache/connection_pool.go`, `internal/core/introspection/inspector.go`, `internal/core/sqldialect/`, and `internal/gateway/executor/instant_crud.go`) will be consolidated into a unified `DatabaseAdapter` module.
   - The module contract:
     ```go
     type DatabaseAdapter interface {
         Connect(ctx context.Context) (*sql.DB, error)
         Introspect(ctx context.Context) (*SchemaInfo, error)
         GenerateQuery(spec QuerySpec) (string, []interface{}, error)
         ExecuteCRUD(ctx context.Context, op CRUDOperation) (interface{}, error)
         Close() error
     }
     ```
   - A central `DatabaseAdapterRegistry` will manage target database lifecycle, pooled connections, and SSH bastion tunnels, yielding the appropriate typed adapter:
     - `PostgresAdapter`: Manages `lib/pq` connections, `public` schema introspection, PostgreSQL query dialect (`$1, $2` parameters, `RETURNING` clauses), and JSON row scanning.
     - `MySQLAdapter`: Manages `go-sql-driver/mysql` connections, database schema introspection, MySQL dialect (`` `table` ``, `?` parameters), and `LAST_INSERT_ID()` handling.
     - `MSSQLAdapter`: Manages `microsoft/go-mssqldb` connections, schema introspection using standard SQL Server `information_schema` views, MSSQL dialect (`@p1, @p2`, `OUTPUT INSERTED.*`, `TOP / OFFSET FETCH`), and row scanning.
   - In `instant_crud.go`, the hardcoded `dbTypeDriver` fallback to PostgreSQL will be removed; execution will dispatch through the resolved `DatabaseAdapter`.
   - In `Connections.tsx`, the unbacked `sqlite` target option will be removed from the dropdown, clearly indicating that external target connections support PostgreSQL, MySQL, and Microsoft SQL Server. `modernc.org/sqlite` remains active strictly for the internal metadata registry.

### Phase 3: Dedicated Target Pool with Ketama Consistent Hash Ring

4. **Target Pool with Ketama Hash Ring**:
   - Target balancing and connection tracking logic will be extracted from `handler.go` into an isolated `TargetPool` module.
   - The module contract:
     ```go
     type TargetPool interface {
         SelectTarget(req *http.Request, upstream *models.Upstream, availableTargets []models.Target) (*models.Target, func(), error)
     }
     ```
   - When the upstream algorithm is `consistent-hashing`, the pool constructs a Ketama hash ring with 160 virtual nodes per target instance using MD5 digests mapped across a uint32 circular ring.
   - Binary search (`sort.Search`) resolves the nearest target on the ring for the extracted hash key (`X-Consumer-ID`, `X-User-ID`, or client IP fallback).
   - The returned release function decrements active connection counts safely, preventing memory leaks in connection accounting.

---

## Testing Decisions

### What Makes a Good Test
- Tests must verify external behavior through public module interfaces, never private internal fields or implementation trivia.
- Tests must assert standard protocol compliance (RFC 7515 / AWS SigV4 / RFC 6749) against mock HTTP receivers.
- Tests must verify deterministic error envelopes and canonical status codes when invalid credentials or unsupported configurations are provided.

### Modules to Test
1. **Outbound Credential Negotiator Test Suite**:
   - Verify AWS SigV4 signing against an HTTP test server that parses and verifies the canonical request hash, signed headers, and HMAC signature matching the AWS test suite.
   - Verify OAuth2 Client Credentials grant flow: test server receives standard `POST` form data (`grant_type=client_credentials`), returns token and expiry; negotiator injects `Authorization: Bearer <token>`.
   - Verify singleflight deduplication: launch 50 concurrent requests simultaneously; assert the mock token endpoint receives exactly 1 HTTP request.
   - Verify token refresh: fast-forward clock past TTL; assert negotiator requests a fresh token.
2. **Inbound Auth Engine Test Suite**:
   - Verify `auth.Engine.Authenticate` against routes with `key-auth`, `basic-auth`, `jwt-auth`, and `hmac-auth`.
   - Verify that consumer route permissions are enforced atomically, returning typed 401/403 errors when access is denied.
   - Verify that all phantom `oauth2` options are rejected and cannot be registered.
3. **Database Adapter Test Suite**:
   - Verify `PostgresAdapter`, `MySQLAdapter`, and `MSSQLAdapter` schema introspection, query generation, and CRUD execution using standard database interface mocks.
   - Verify that requesting an unsupported driver cleanly returns a typed error rather than falling back to PostgreSQL.
4. **Target Pool Test Suite**:
   - Verify Ketama consistent hash distribution: assert that hash keys deterministically hit the same target.
   - Verify ring stability: add a 4th target to a 3-target pool; assert that approximately $75\%$ of keys remain on their original targets and at most $25\%$ are remapped.
   - Verify connection tracking: assert active connection count increases during request handling and drops to zero when the release closure executes.

### Prior Art
- Unit and integration tests in `internal/gateway/auth/engine_test.go`, `internal/gateway/auth/outbound_test.go`, and `internal/core/sqldialect/dialect_test.go`.
- End-to-end test suite in `tests/e2e/e2e_features_test.go`.

---

## Out of Scope
- Building a full OAuth 2.0 Authorization Server for external user login inside Apirout.
- Dynamic runtime installation of CGo-based database drivers.
- Distributed shared memory for OAuth2 tokens across separate physical cluster instances without Redis.
- GraphQL query generation (instant APIs remain RESTful).

---

## Further Notes
- This specification formalizes the architectural decisions captured in `ADR-0010: Deep Architecture and Authentic Credential Seams`.
- All vocabulary conforms to `GLOSSARY.md` and the deep-module design principles of `codebase-design`.
