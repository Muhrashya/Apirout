# ADR-0013: Ungrouped Connection Isolation and Declarative Stream Shielding

## Status
Accepted

## Context
Following the Phase 2 security hardening (Tickets 77–84), a Phase 3 full-stack security audit identified residual edge cases across the Data Plane and Control Plane:

1. **Instant CRUD Ungrouped Connection BOLA (SEC-12)**: Database connections configured without a Consumer Group (`group_id IS NULL`) bypassed object-level authorization checks in `internal/gateway/executor/instant_crud.go`. Consequently, any valid consumer API key or API client token from an unrelated tenant could perform read/write queries against global or ungrouped databases. In contrast, the Control Plane query runner (`queries.go`) strictly required `SUPER_ADMIN` for ungrouped connections.
2. **Declarative Import Resource Exhaustion (SEC-13)**: The GitOps declarative import handler (`internal/console/gitops/handler.go`) decoded JSON and YAML bodies directly from `r.Body`. Without explicit reader bounding and entity validation, large payloads or YAML alias expansion bombs could induce high memory consumption.
3. **SPA Sibling Directory Traversal Defense Boundary (SEC-14)**: The SPA static file handler used prefix matching (`!strings.HasPrefix(cleanTarget, cleanStatic)`) without a directory separator boundary, creating a theoretical sibling directory escape vector.
4. **Dead-Code SQLite Egress Bypass (SEC-15)**: The database connection test handler exempted `sqlite` from `egress.ValidateSafeHost`, even though SQLite is not an external target connector.

## Decision

1. **Enforce Global Connection Isolation in Instant CRUD**:
   - Classify all database connections with `group_id IS NULL` as platform-level system assets.
   - In `AuthorizeConnectionAccess`, explicitly deny tenant-scoped consumer API keys and API client tokens with `403 Forbidden` if the target connection is ungrouped.
   - Require either the Master Admin Secret (`X-Apirout-Admin-Secret`) or a verified `SUPER_ADMIN` Console JWT token to execute Instant CRUD operations on ungrouped connections.

2. **Implement Declarative Stream Shielding**:
   - Wrap the request body stream in `gitops.Handler.Import` with `io.LimitReader` bounded to 5MB (`5 * 1024 * 1024` bytes).
   - Reject malformed or oversized JSON/YAML payloads deterministically with HTTP 400 envelopes before invoking the database transaction engine.

3. **Harden Filesystem Path Boundaries in SPA Handler**:
   - Replace string prefix matching in `internal/console/server/spa.go` with canonical `filepath.Rel(cleanStatic, cleanTarget)`.
   - Reject any target whose relative path escapes the root (`strings.HasPrefix(rel, "..") || rel == ".."`).

4. **Purge Vestigial SQLite Host Bypass**:
   - Enforce `egress.ValidateSafeHost(req.Host)` unconditionally across all database connection test requests in `internal/console/server/connections.go`.

5. **Web Console Session Security (SEC-16)**:
   - Maintain `localStorage` token transport in the current React SPA milestone, relying on the existing defenses (`Content-Security-Policy: default-src 'self'`, `X-Content-Type-Options: nosniff`, and `ADMIN_ALLOWED_IPS`).
   - Plan migration to `HttpOnly`, `SameSite=Strict` cookie-based sessions for the next major Control Plane release.

## Consequences

- **Positive**:
  - Eliminates cross-tenant data access vulnerabilities on ungrouped database connections.
  - Aligns Instant CRUD authorization semantics with Control Plane query authorization.
  - Prevents memory exhaustion attacks via oversized or crafted declarative configuration imports.
  - Hardens static file routing and egress network perimeter validation.
- **Operational Considerations**:
  - Tenant applications must ensure their database connections are assigned to their respective Consumer Group to allow API key or client token access via `/api/v1/instant/{connId}/*`.
