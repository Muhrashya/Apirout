# ADR-0017: Gateway Ingress Preflight, Instant API Pipeline Unification, and Audit Log Retention

## Status
Accepted

## Context
Following a secondary product quality and architecture audit, critical friction points and functional contradictions were identified across CORS preflight handling, Instant REST API pipeline routing, SQL query preview safety, path stripping on dynamic routes, direct service resilience, and audit log storage lifecycle:

1. **CORS Preflight 405 Rejections**: In `stage_ingress.go`, browser preflight `OPTIONS` requests against non-OPTIONS routes (such as `POST /api/v1/orders`) were matched by the Radix router and immediately rejected with `HTTP 405 Method Not Allowed` before reaching the route CORS block. Furthermore, naive string slicing on host headers (`strings.Index(host, ":")`) truncated bracketed IPv6 addresses (`[::1]:8000`) into `[`, resulting in `403 Forbidden: Host '[' is not registered`.
2. **Instant API Execution Path Duality & Method Lockout**: Dedicated Hasura-style endpoints (`/api/v1/instant/{connId}/*`) were mounted directly on Gorilla Mux, bypassing the execution pipeline and dodging rate limiting, IP whitelists, domain validation, and telemetry audit logging. Simultaneously, custom Instant API routes generated in the console were locked to `Method: "GET"`, blocking mutating CRUD operations (`POST`, `PUT`, `DELETE`) with 405 errors.
3. **Interactive SQL Preview Security & Direct Service Resilience**: In `QueryHandler.Preview`, user-supplied queries bypassed `executor.ValidateSQL` and connection `is_read_only` guards, allowing arbitrary mutating statements (`DROP`, `UPDATE`, `DELETE`) against read-only replicas. On the proxy edge, path stripping failed on wildcard/parameter routes (`strings.TrimPrefix`), and direct backends passed empty target IDs, evading passive circuit breaker tracking.
4. **Unbounded Telemetry Growth & Tenant RBAC Leaks**: `request_audit_logs` grew infinitely without a TTL retention worker, risking disk exhaustion. Meanwhile, `ConnectionHandler.Update` allowed non-SuperAdmins to nullify or modify connection `group_id`, while `UserFormPage.tsx` lacked group assignment controls, stranding newly created non-superadmin users with empty group associations.

## Decision

1. **Ingress Preflight Short-Circuiting & Standard Host Parsing (Q1 A)**:
   - In `internal/gateway/executor/stage_ingress.go`, intercept incoming `r.Method == http.MethodOptions`. If a route exists for the path, short-circuit preflight handling and emit `204 No Content` with `Access-Control-Allow-Methods` matching the endpoint's allowed verbs and `Access-Control-Allow-Headers`.
   - Parse `r.Host` and proxy headers using standard `net.SplitHostPort`, correctly accommodating bracketed IPv6 addresses (`[::1]`). Strip untrusted `X-Forwarded-Host` headers unless incoming requests originate from a verified trusted proxy CIDR.

2. **Unified Database Execution Pipeline & Wildcard CRUD Routes (Q2 A)**:
   - Route `/api/v1/instant/*` through the unified gateway execution pipeline so that rate limiting, client IP allowlists, domain validation, and request audit telemetry apply uniformly across all database traffic.
   - Register auto-generated Instant API routes with method `ANY` (or full CRUD verbs), allowing `POST`, `PUT`, `PATCH`, and `DELETE` requests to reach the Instant CRUD handler, which validates permissions and enforces connection `is_read_only` guards.

3. **Strict Query Preview Guard, Static Prefix Stripping & Synthetic Direct Circuit Keys (Q3 A)**:
   - Enforce `executor.ValidateSQL(req.SQL)` and inspect `conn.IsReadOnly` inside `QueryHandler.Preview`, rejecting mutations on read-only databases with `405 Method Not Allowed`.
   - In `internal/gateway/executor/handler.go`, compute path stripping up to the static prefix preceding wildcard or parameterized segments (e.g. trimming `/api/v1/users/` from `/api/v1/users/*`).
   - Generate deterministic synthetic target IDs (e.g. `direct:<service_id>`) for direct backend services, allowing edge passive circuit breakers to trip on consecutive 5xx failures.

4. **Telemetry Retention Pruner, Tenant Isolation & Auth Alignment (Q4 A)**:
   - Implement an automated background retention worker in `internal/console/monitoring/` (or registry) governed by `AUDIT_LOG_RETENTION_DAYS` (defaulting to 14 days) that periodically purges expired `request_audit_logs` in batches.
   - Enforce group permission checks in `ConnectionHandler.Update` preventing non-SuperAdmins from nullifying `group_id` or transferring connections across groups.
   - Add a Group selection field in `UserFormPage.tsx` so administrators can assign users to tenant groups upon creation.
   - Update `AuthPluginManager.tsx` and documentation to accurately reflect that `jwt-auth` currently uses symmetric HS256 with the platform secret.

## Consequences

- **Positive**:
  - Browser SPAs and web clients can seamlessly invoke cross-origin APIs without preflight 405 rejections.
  - IPv6 network ingress operates flawlessly without host header corruption.
  - All database traffic receives uniform rate limiting, security inspection, and audit logging.
  - Instant REST APIs support full CRUD operations as intended.
  - Interactive SQL previews can never mutate or damage read-only databases.
  - Direct backend failures trigger passive circuit breaking, preventing thread pool exhaustion.
  - Audit log tables are bounded and self-cleaning.
  - Tenant group boundaries are strictly maintained during connection edits and user onboarding.
- **Operational Considerations**:
  - Operators wishing to retain audit logs longer than 14 days must configure `AUDIT_LOG_RETENTION_DAYS` in their environment.
