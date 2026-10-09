# ADR-0010: Deep Architecture and Authentic Credential Seams

## Status
Accepted

## Context
An architectural review conducted under the `improve-codebase-architecture` skill revealed critical module shallowness, broken seams, and unbacked/pseudo features across the Apirout runtime:

1. **Pseudo Outbound Authentication Implementations**:
   - `aws-sigv4` in `internal/gateway/auth/outbound.go` sets dummy non-standard headers (`X-Apirout-AWS-Access-Key`, `X-Apirout-AWS-Region`, `X-Apirout-AWS-Service`) rather than calculating standard canonical request hashes and HMAC-SHA256 signatures (`Authorization: AWS4-HMAC-SHA256 ...`). AWS endpoints immediately reject requests with `403 Forbidden`.
   - `oauth2-cc` sets a static client ID header without issuing grant requests to token endpoints or maintaining token lifecycles with TTL caching, failing upstream authentication.
2. **Inbound Auth Seam Fragmentation & Phantom Credentials**:
   - Inbound client authentication is split between `auth.Engine` and `apiclientauth.Service`.
   - `Consumers.tsx` retains a legacy `<option value="oauth2">OAuth 2.0 Client Secret</option>` and `AuthPluginManager.tsx` has dead code branches for `oauth2`, even though ADR-0006 established that Apirout possesses no inbound OAuth2 token endpoint or evaluator. Creating such credentials produces unresolvable 401 lockouts.
3. **Database Engine Seam Divergence**:
   - External database handling leaks across 4 shallow modules with conflicting capabilities:
     - `internal/core/cache/connection_pool.go` supports MSSQL but rejects SQLite (`unsupported database driver: sqlite`).
     - `internal/core/introspection/inspector.go` supports PostgreSQL and MySQL, but rejects MSSQL and SQLite (`unsupported database type for introspection`).
     - `internal/core/sqldialect/` implements PostgreSQL, MySQL, and MSSQL, but lacks SQLite.
     - `internal/gateway/executor/instant_crud.go` hardcodes fallback to PostgreSQL for all non-MySQL databases.
     - The web console advertises SQLite as an external database target despite missing driver linkage.
4. **Target Selection Modulo Hashing & Memory Growth**:
   - Upstream "consistent-hashing" is implemented in `handler.go` as `(31*h + char) % len(targets)`. This is naive modulo arithmetic rather than consistent ring hashing; changing target pool size reshuffles 100% of keys.
   - Global `activeConns` tracking uses an unbounded `sync.Map` susceptible to memory growth for transient targets.

## Decision

We establish four deep module consolidations structured in three execution phases:

### Phase 1: Authentication Integrity & Authentic Credential Negotiator
1. **Outbound Credential Negotiator**:
   - Deepen `OutboundInjector` into a self-contained `OutboundNegotiator` module at the upstream reverse proxy seam.
   - Implement RFC-compliant AWS SigV4 request signing: calculate SHA-256 payload digests, canonical request strings, string-to-sign, and HMAC-SHA256 signature derivations (`AWS4-HMAC-SHA256`).
   - Implement authentic OAuth2 Client Credentials grant manager: execute HTTP POST requests to third-party token endpoints (`grant_type=client_credentials`), cache access tokens with in-memory TTL accounting, apply a 60-second expiration buffer, and enforce singleflight request deduplication to prevent thundering herd spikes.
2. **Inbound Auth Engine Consolidation & Phantom Purge**:
   - Absorb `apiclientauth.Service` directly into `auth.Engine`.
   - Expose a single unified ingress seam: `Authenticate(r *http.Request, endpoint *models.Endpoint) (*ConsumerContext, error)`, which evaluates active plugins (`key-auth`, `basic-auth`, `jwt-auth`, `hmac-auth`) and verifies consumer route-level permissions in a single atomic pass.
   - Completely purge residual `oauth2` dropdown options and model enums from `Consumers.tsx`, `AuthPluginManager.tsx`, and `models.PluginOAuth2`, strictly aligning the console with genuine gateway capabilities.

### Phase 2: Consolidated Database Adapter Seam
1. **Unified Database Adapter Architecture**:
   - Replace the 4 fragmented packages (`cache`, `introspection`, `sqldialect`, `instant_crud`) with a deep `DatabaseAdapter` module.
   - Define a single cohesive interface:
     ```go
     type DatabaseAdapter interface {
         Connect(ctx context.Context) (*sql.DB, error)
         Introspect(ctx context.Context) (*SchemaInfo, error)
         GenerateQuery(spec QuerySpec) (string, []interface{}, error)
         ExecuteCRUD(ctx context.Context, op CRUDOperation) (interface{}, error)
         Close() error
     }
     ```
   - Provide complete, verified implementations for `PostgresAdapter`, `MySQLAdapter`, and `MSSQLAdapter` (adding MSSQL `information_schema` introspection to match its dialect generator).
   - Explicitly remove the unbacked SQLite option from external database connection forms in `Connections.tsx`, preserving `modernc.org/sqlite` strictly for the internal metadata registry.

### Phase 3: Dedicated Target Pool with Ketama Consistent Hash Ring
1. **Target Pool Deepening**:
   - Extract target selection from `handler.go` into a dedicated `TargetPool` module.
   - Implement a standard Ketama consistent hash ring with 160 virtual nodes per target instance on a 32-bit ring.
   - Ensure scaling up or down redistributes at most $1/N$ of keys, preserving upstream cache locality and sticky sessions.
   - Implement scoped connection lease tracking with automated decrement and cleanup upon request termination.

## Consequences

- **Positive**:
  - **Elimination of Pseudo Features**: AWS SigV4 and OAuth2 Client Credentials become real, RFC-compliant authentication mechanisms capable of connecting to enterprise cloud services.
  - **Zero Phantom 401s**: Purging residual `oauth2` options prevents operators from configuring invalid credentials.
  - **High Locality & Leverage**: All database operations for an engine live in a single adapter file. Callers hit one seam instead of coordinating across 4 packages.
  - **Session & Cache Affinity**: Ketama consistent hashing prevents 100% cache invalidation across upstream nodes during deployment autoscaling events.
- **Trade-offs**:
  - Outbound OAuth2 requires external network connectivity from the gateway Data Plane to the configured Identity Provider token endpoint.
  - SQLite is restricted to internal metadata registry storage and cannot be used as an external Instant CRUD target database.
