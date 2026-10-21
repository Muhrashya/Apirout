# ADR-0016: Upstream Target Reconciliation, Instant API Safety, and Gateway Security Hardening

## Status
Accepted

## Context
Following a full architectural and product quality audit of Apirout, several functional contradictions and safety gaps were identified across upstream target management, Instant REST API generation, rate limiting keying, and read-only database connections:
1. **Silent Upstream Target Dropping**: When updating existing upstream load-balancing pools in the Web Console (`UpstreamFormPage.tsx`), edits to the targets list were bypassed due to an explicit `!isEdit` guard. Adding, editing, or deleting target server instances was ignored on existing upstreams.
2. **Instant API SQL Keyword Syntax Failures & Public Exposure**: Generated Instant APIs constructed raw SQL without quoting table names (`SELECT * FROM %s`), causing immediate syntax errors on reserved keywords (such as `order`, `user`, `group`). Furthermore, generated routes defaulted to unauthenticated public access (`is_public: true`) without rate limiting.
3. **Rate Limiting Key Collision on Anonymous Traffic**: In the gateway execution pipeline (`stage_auth.go`), non-public routes with nil consumer IDs shared a single empty consumer string (`route:<id>:client:`), collapsing all unauthenticated callers into a single token bucket.
4. **Read-Only Database Connection Write Vulnerability**: While connections could be flagged as read-only in the UI, the gateway execution plane did not enforce this restriction on Instant CRUD operations.
5. **Hardcoded Loopback Gateway Hosts**: Web console test previewers and cURL copy actions hardcoded `http://localhost:8000`, failing when operators accessed the console across LANs, VPNs, or reverse proxies.

## Decision

1. **Declarative Upstream Target Reconciliation (Q1 A)**:
   - In `web/src/pages/upstreams/UpstreamFormPage.tsx`, preserve `id` on target instances during load.
   - On save, retrieve active targets via `gatewayApi.getTargets(upstreamId)` and execute a declarative diff:
     - New instances dispatched via `gatewayApi.createTarget`.
     - Altered instances dispatched via `gatewayApi.updateTarget`.
     - Removed instances dispatched via `gatewayApi.deleteTarget`.
   - Dispatch operations concurrently via `Promise.all` and navigate with unified status feedback.

2. **Instant API Identifier Quoting & Secure Defaults (Q2 A)**:
   - In `internal/console/server/connections.go`, look up SQL dialect generators via `sqldialect.GetGenerator(conn.DBType)` and escape table names via `gen.QuoteIdentifier(table.Name)`.
   - Auto-generated endpoints default to `is_public: false` (protected by API key) with 60 RPM / 10 burst rate limiting enabled.
   - Support optional `is_public` boolean in the generation request body.
   - In `web/src/pages/InstantAPI.tsx`, introduce a confirmation modal with an Access Policy selector (`Protected with API Key` vs `Public Read-Only`).

3. **Dual-Tier Rate Limiting Shield (Q3 A)**:
   - In `internal/gateway/executor/stage_auth.go`, verify `c.ConsumerID != nil && *c.ConsumerID != ""` before deriving consumer rate keys.
   - Anonymous or unauthenticated traffic strictly derives rate limit keys from client IP (`route:<id>:ip:<client_ip>`), isolating token buckets per caller IP.
   - Skip consumer group rate limiting when consumer ID is empty.

4. **Gateway Read-Only Enforcement (Q4 A)**:
   - Persist `is_read_only` on database connection profiles across SQLite, PostgreSQL, and MySQL registries.
   - In `internal/gateway/executor/instant_crud.go` and `stage_upstream.go`, inspect `conn.IsReadOnly`.
   - Abort any mutating HTTP verbs (`POST`, `PUT`, `PATCH`, `DELETE`) with `HTTP 405 Method Not Allowed`, populating `Allow: GET, HEAD, OPTIONS` and descriptive error envelope.

5. **Dynamic Gateway Host Resolution**:
   - In `web/src/api/gatewayApi.ts`, export `getGatewayBaseUrl()` resolving from `VITE_GATEWAY_URL` or `${window.location.protocol}//${window.location.hostname}:8000`.
   - Replace hardcoded loopback addresses across `InstantAPI.tsx`, `Routes.tsx`, and `ApiLogs.tsx`.

## Consequences

- **Positive**:
  - Upstream target editing is fully bidirectional and declarative; cluster topology changes persist reliably.
  - Reserved SQL table keywords no longer fail during instant API generation.
  - New endpoints follow security best practices by default (authenticated and rate-limited).
  - High traffic or abuse from an unauthenticated IP address cannot starve other clients sharing the gateway.
  - Read-only replicas and analytical engines are guaranteed immutable at the gateway boundary.
  - Management console works seamlessly when accessed across remote network topologies.
- **Operational Considerations**:
  - Operators migrating existing instant APIs who require unauthenticated access must explicitly configure route access or choose Public Read-Only in the modal.
