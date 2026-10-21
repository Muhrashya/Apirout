# Ingress Preflight Short-Circuiting, Instant API Pipeline Unification, and Telemetry Retention

**Status:** ready-for-agent

## Problem Statement

Operators and application developers using Apirout face several critical runtime friction points and functional contradictions:
1. Browser single-page applications (SPAs) initiating cross-origin HTTP calls fail immediately on preflight `OPTIONS` requests because the in-memory Radix router enforces exact route methods before evaluating CORS headers, returning `405 Method Not Allowed`. Furthermore, clients connecting over IPv6 networks experience unexpected `403 Forbidden` errors due to naive string slicing that mangles bracketed host addresses like `[::1]:8000` into invalid identifiers (`[`).
2. Instant REST APIs operate across two disconnected planes: requests sent to `/api/v1/instant/{connId}/*` bypass the gateway execution pipeline entirely, evading rate limits, IP allowlists, host validation, and audit logging. Simultaneously, custom Instant API routes generated in the management console default to `Method: "GET"`, blocking mutating CRUD operations (`POST`, `PUT`, `PATCH`, `DELETE`) with 405 errors.
3. The Web Console's interactive SQL Query previewer bypasses SQL statement validation and connection read-only permissions, permitting destructive queries against read-only replicas. On the edge proxy, path stripping breaks on wildcard routes, and direct backend services evade passive circuit breaker protection.
4. Database telemetry logs (`request_audit_logs`) accumulate infinitely on every gateway request without automated pruning, risking table bloat and disk exhaustion. Tenant role-based access control leaks allow non-SuperAdmins to modify connection group scoping, while newly created developers cannot be assigned to tenant groups through the console.

## Solution

1. Introduce **Preflight Short-Circuiting** at the gateway ingress boundary to evaluate `OPTIONS` requests against registered route paths, returning `204 No Content` with dynamic CORS headers (`Access-Control-Allow-Methods`, `Access-Control-Allow-Headers`) before strict route method matching. Implement robust host parsing using standard host-port splitting that natively supports IPv6 brackets.
2. Establish a **Unified Database Pipeline** that routes all Instant REST API operations (`/api/v1/instant/*` and custom table endpoints) through the multi-stage gateway execution pipeline. Register Instant API routes with wildcard method matching (`ANY`), ensuring full CRUD capabilities while uniformly applying rate limits, client IP shields, read-only 405 guards, and telemetry audit logging.
3. Secure the interactive SQL Query runner by validating statements against SELECT-only rules and rejecting write attempts on read-only database connections. Fix path stripping on wildcard and parameterized routes by calculating static prefix offsets, and generate **Synthetic Target Keys** (`direct:<service_id>`) so direct backends trigger edge passive circuit breakers upon consecutive 5xx failures.
4. Deploy an automated **Telemetry Retention Pruner** background worker that enforces configurable log retention (`AUDIT_LOG_RETENTION_DAYS`, defaulting to 14 days) by batch-deleting expired records. Lock down connection group assignments against non-SuperAdmin escalation, provide user group assignment controls in the Web Console, and align auth plugin schemas with actual symmetric HS256 engine capabilities.

## User Stories

1. As a frontend web developer, I want cross-origin preflight `OPTIONS` requests to return `204 No Content` with appropriate `Access-Control-Allow-Methods` headers, so that my single-page application can call gateway endpoints without encountering `405 Method Not Allowed` errors.
2. As a network administrator, I want to route traffic to the gateway over IPv6 networks using bracketed host addresses (`[::1]:8000`), so that clients connecting over IPv6 are not rejected with `403 Forbidden: Host '[' is not registered`.
3. As an API consumer, I want to execute `POST`, `PUT`, `PATCH`, and `DELETE` requests against auto-generated Instant REST API routes, so that I can perform full CRUD data operations rather than being restricted only to `GET` queries.
4. As a security architect, I want all database REST API requests (including `/api/v1/instant/*`) to pass through the core gateway execution pipeline, so that client IP restrictions, rate limiting token buckets, domain validation, and request audit logs are consistently enforced.
5. As a database administrator, I want mutating HTTP methods against read-only database connections to be blocked with `405 Method Not Allowed` across both Instant REST routes and custom query routes, so that read replicas and analytics warehouses cannot be modified.
6. As a platform operator, I want the interactive SQL Query previewer in the console to block mutating keywords (`DROP`, `INSERT`, `UPDATE`, `DELETE`) and read-only connection writes, so that users cannot accidentally damage data during query authoring.
7. As an API designer, I want routes with wildcard paths (e.g. `/api/v1/users/*`) and `strip_path: true` to forward only the trailing sub-path to the upstream backend, so that upstream services receive expected relative URLs.
8. As a site reliability engineer, I want standalone direct backend services to trip passive circuit breakers after consecutive 5xx errors, so that gateway thread pools and client requests fail fast rather than hanging during backend outages.
9. As a systems administrator, I want request audit logs older than a configured retention threshold (e.g. 14 days) to be automatically pruned in the background, so that the database disk does not fill up over time.
10. As a tenant administrator, I want non-SuperAdmin users to be prevented from altering connection group associations or making connections global, so that multi-tenant isolation is strictly preserved.
11. As a tenant manager, I want to assign newly created developer and maintainer users to specific tenant groups directly from the User creation form, so that they immediately gain access to their team's connections and queries.
12. As a gateway administrator, I want the authentication plugin configuration to transparently describe supported algorithms (symmetric HS256), so that I am not misled into configuring unsupported asymmetric RSA/ECDSA keys without corresponding engine support.

## Implementation Decisions

- **Ingress Preflight Short-Circuiting**: In the ingress stage, inspect HTTP request methods. If the method is `OPTIONS`, locate the route matching the path. If found, populate CORS headers based on the endpoint's configured CORS origins and allowed HTTP methods, emit `204 No Content`, and mark the context as handled before evaluating method mismatch errors.
- **Robust Ingress Host Parsing**: Replace naive string colon searching with standard network host-port separation that accurately parses IPv4, hostnames, and bracketed IPv6 literals (`[::1]`). Strip incoming forwarded host headers unless the request originates from a trusted upstream proxy CIDR.
- **Unified Database Execution Pipeline**: Unify the dedicated Instant REST router under the standard gateway execution pipeline. Route `/api/v1/instant/{connId}/*` through the Radix router and execution stages, ensuring uniform execution of ingress, rate limiting, IP filtering, upstream data dispatch, and telemetry audit logging.
- **Instant API Method Wildcarding**: Update endpoint generation logic and route creation forms so that database-backed Instant API routes register with method `ANY` (or the complete REST verb set). Defer HTTP method validation to the Instant CRUD handler, which validates request verbs against connection read-only settings.
- **SQL Preview Validation & Read-Only Guard**: Enhance the query preview handler in the console server to invoke SQL statement validation and verify that target connections are not read-only before executing preview queries.
- **Dynamic Path Stripping for Wildcard Routes**: When calculating upstream sub-paths for routes with `strip_path: true`, compute the static prefix of the route preceding any wildcard (`*`) or parameter (`:id`) tokens. Strip that static prefix from incoming request paths rather than attempting full literal string trimming.
- **Synthetic Target Keys for Direct Services**: For services pointing directly to a host and port without an upstream pool, generate a synthetic target key (`direct:<service_id>`) and pass it to proxy handlers and response modifiers, enabling passive circuit breaker tracking for direct services.
- **Automated Telemetry Retention Worker**: Create a background maintenance routine that runs periodically (e.g. every hour) to delete records in `request_audit_logs` older than `AUDIT_LOG_RETENTION_DAYS` (default 14 days). Execute batch deletions with limit clauses to prevent database locking in high-volume environments.
- **Tenant Group Boundary Hardening**: In the connection update handler, verify that non-SuperAdmins cannot modify `group_id` or set it to `nil`. Add group selection controls to the Web Console User form and update user creation and editing handlers to maintain user-group associations.
- **Auth Plugin Schema Alignment**: Update the JWT authentication plugin form and documentation to specify that symmetric HS256 with the platform secret is active, removing misleading references to unsupported asymmetric algorithms until full JWKS verification is introduced.

## Testing Decisions

- **Test Behavior, Not Internal Details**: Validate vertical slices through public HTTP interfaces (Gateway port 8000, Admin API port 8001, and Web Console actions).
- **Ingress CORS Preflight Seam**: Execute HTTP `OPTIONS` requests against protected `POST`/`PUT` routes and verify `204 No Content`, `Access-Control-Allow-Methods`, and `Access-Control-Allow-Headers` without 405 errors. Test IPv6 `Host: [::1]:8000` headers and confirm successful domain authorization.
- **Instant CRUD & Unified Pipeline Seam**: Send `GET`, `POST`, `PUT`, and `DELETE` requests to generated Instant API endpoints and `/api/v1/instant/*`. Verify that allowed mutations persist, read-only connections return `405 Method Not Allowed`, rate limits throttle callers, and records appear in `request_audit_logs`.
- **Query Preview & Circuit Breaker Seam**: Test that `Preview` rejects `DROP TABLE` or mutating SQL and rejects read-only connections. Test that wildcard routes with `strip_path: true` correctly trim path prefixes. Test that repeated 502/503 errors against direct backend services trip the circuit breaker.
- **Retention Worker & RBAC Seam**: Insert synthetic audit log entries with past timestamps and invoke the retention cleanup worker, verifying stale rows are pruned. Attempt connection group reassignment with non-SuperAdmin credentials and verify rejection with 403.
- **Prior Art**: Leverage existing patterns in `internal/gateway/executor/handler_test.go`, `instant_crud_test.go`, `internal/console/server/connections_test.go`, and `tests/e2e/e2e_features_test.go`.

## Out of Scope

- Implementing an external distributed JWKS key-fetching agent for asymmetric token rotation.
- Real-time streaming log shipping (e.g. syslog or Kafka export) beyond existing database audit logging.
- Multi-region database replication management.

## Further Notes

- The 14-day retention default provides a solid balance between observability and disk consumption for development and production workloads.
- The preflight fix resolves SPA browser integration issues where modern fetch/axios libraries issue automated preflight checks for JSON payloads.
