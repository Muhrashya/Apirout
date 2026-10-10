# Full-Stack Security Hardening & Vulnerability Remediation Spec (Phase 2)

**Status:** ready-for-agent

## Problem Statement

Following the initial security audit remediation, a comprehensive re-audit of the Apirout runtime, management plane, network boundaries, and infrastructure uncovered critical security gaps across the OWASP Top 10 and OWASP API Security Top 10 (2023):

1. **Cross-Tenant BOLA on Instant CRUD Engine**: The Instant Database REST engine (`/api/v1/instant/{connId}/*`) verifies that an incoming request carries *a* valid consumer credential or token, but fails to check whether that caller has authorization to access the target database connection (`connId`). Any authenticated external client can read, insert, update, or delete records from any registered internal database connection.
2. **SSRF Redirect Chasing Vulnerability**: The egress security shield validates target hostnames and IPs at request initiation, but outbound HTTP clients (used for OAuth2 Client Credentials token exchanges and upstream target health checking) follow HTTP 301/302 redirects automatically without re-validating the target URL. A malicious external server can redirect outbound requests to internal loopback (`127.0.0.1`) or cloud metadata services (`169.254.169.254`).
3. **Unauthenticated API Inventory & Schema Disclosure**: The OpenAPI (`/openapi.json`) and Postman (`/postman.json`) specification endpoints are mounted on an unauthenticated root router. When queried without credentials, the system fails to restrict group scopes and dumps the complete, unfiltered inventory of all private endpoints, paths, query parameters, and schema models to anonymous actors.
4. **Public Metrics Exposure & Unrestricted Port Binding**: Prometheus metrics (`/metrics`) exposing internal route paths, target servers, error codes, and request volumes are mounted on the public Data Plane ingress port (`:8000`) without authentication. Additionally, the status/metrics port (`:8100`) is bound to all external network interfaces in Docker Compose.
5. **Lack of Server-Side Token Revocation & Identity State Checks**: JWT evaluation in the Control Plane and Data Plane is strictly stateless, checking cryptographic signatures and expiration timestamps without verifying whether the underlying user has been deleted, deactivated, or had their password rotated, allowing compromised tokens to remain active until full TTL expiry.
6. **Super Admin Self-Deletion & Lockout Risk**: The administrative user management API allows Super Admins to delete their own accounts or the sole remaining Super Admin in the system, creating an unrecoverable administrative lockout.
7. **Missing Data Plane Security Headers**: Defensive HTTP security headers (`X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`) are active on the Control Plane but missing from the public Data Plane proxy pipeline.
8. **Production Secret Validator Gaps**: Production startup validation checks against specific static placeholders, but ignores the default fallback keys present in the default Docker Compose configuration, risking accidental production deployments with known keys.

## Solution

Implement defense-in-depth security remediation across the platform to resolve findings SEC-01 through SEC-11:

1. **Enforce Connection Authorization on Instant CRUD**: Introduce explicit access control on the Instant Database Engine requiring that authenticated consumers or API clients possess explicit permissions or shared group membership with the target database connection, returning HTTP 403 Forbidden on unauthorized connection access.
2. **Harden Outbound HTTP Clients with Egress Redirect Validation**: Create a centralized Safe HTTP Client factory in the egress boundary that equips all outbound HTTP clients with a redirect evaluation hook (`CheckRedirect`) enforcing SSRF validation across all hops, rejecting redirects to loopback, link-local, and unauthorized private addresses.
3. **Restrict API Documentation to Authenticated Scopes**: Secure `/openapi.json` and `/postman.json` behind the authenticated administrative router, ensuring specifications are filtered strictly by the caller's assigned group permissions and requiring explicit public route tagging for unauthenticated documentation access.
4. **Isolate Observability Ports & Remove Public Metrics**: Remove the `/metrics` endpoint from the public Data Plane port (`:8000`), restricting metrics exclusively to the dedicated internal Status Plane (`:8100`), and bind the status port to host loopback (`127.0.0.1`) in container configurations.
5. **Implement Session Verification & Revocation Checks**: Augment the authentication middleware with cached database checks verifying that users remain active and that tokens were issued subsequent to the user's latest password modification timestamp.
6. **Guard Against Administrative Lockout**: Enforce business rules preventing Super Admins from deleting their own user account and blocking the deletion or demotion of the final active Super Admin in the registry.
7. **Inject Security Headers on Gateway Ingress**: Wire the defensive security header middleware into the public Data Plane proxy handler to guarantee that all API responses emit MIME sniffing and frame protection headers.
8. **Comprehensive Production Configuration Sanitization**: Expand the production startup environment validator to reject all default fallback secret values present in Docker Compose and environment templates.

## User Stories

1. As an API Gateway operator, I want external API consumers to be restricted only to database connections they have been granted access to, so that clients cannot query unauthorized internal databases via Instant CRUD endpoints.
2. As a database administrator, I want attempts by unauthorized consumers to access `/api/v1/instant/{connId}/*` to be rejected with HTTP 403 Forbidden and logged as security audit events.
3. As a security architect, I want outbound HTTP requests made by the gateway (such as OAuth2 token retrieval and upstream health checks) to reject redirects pointing to loopback or cloud metadata services (e.g., `169.254.169.254`), so that remote servers cannot coerce the gateway into performing internal SSRF.
4. As an infrastructure engineer, I want the outbound HTTP client to validate destination IP addresses prior to TCP socket connection, preventing DNS rebinding attacks.
5. As an enterprise compliance officer, I want administrative API documentation (`/openapi.json` and `/postman.json`) restricted to authenticated users, so that unauthenticated internet actors cannot discover private endpoints and schemas.
6. As an API developer, I want my generated OpenAPI and Postman collections to display only the routes and services assigned to my user group, preventing cross-tenant route visibility.
7. As a security engineer, I want Prometheus metrics (`/metrics`) removed from the public Data Plane ingress port (`:8000`), so that external traffic cannot scrape internal route topologies, backend server names, and error rates.
8. As a DevOps engineer, I want the status and metrics port (`:8100`) bound exclusively to `127.0.0.1` in Docker Compose, ensuring monitoring endpoints are not exposed to the public internet.
9. As a platform administrator, I want deactivating or deleting a user to immediately invalidate their administrative JWT tokens, preventing unauthorized access during offboarding.
10. As a user, I want changing my password to revoke all previously issued JWT tokens, ensuring lost or stolen tokens cannot be used after a credential reset.
11. As a super administrator, I want the system to reject any request where a Super Admin attempts to delete their own account, preventing accidental self-deletion.
12. As a system operator, I want the registry to reject deletion or role demotion of the last active Super Admin, ensuring the platform never enters an unrecoverable administrative lockout.
13. As an API consumer, I want all responses from the public API gateway to carry standard HTTP security headers (`X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`), protecting web clients against MIME sniffing and clickjacking attacks.
14. As an operator deploying to production (`APP_ENV=production`), I want the application startup to fail fast if default Docker Compose secrets (`super-secret-jwt-key-min-32-characters!` or `apirout-local-dev-aes-encryption-key-32!`) are detected, preventing inadvertent insecure production deployments.
15. As an API administrator, I want the initial Super Admin provisioning endpoint (`/auth/setup`) protected by rate limiting, preventing brute-force flooding and database contention during initial cluster bootstrap.
16. As a developer reading the repository documentation, I want production deployment references in the documentation to accurately reflect existing configuration manifests in the workspace.

## Implementation Decisions

1. **Centralized Safe Egress HTTP Client Seam**:
   - Establish an egress HTTP client factory within the egress boundary.
   - The factory will configure an `http.Client` with a strict `CheckRedirect` callback that executes URL validation against loopback, cloud metadata (`169.254.0.0/16`), and private RFC 1918 networks on every redirect step.
   - Replace raw `http.Client` instances in the outbound OAuth2 manager and upstream target health checker with the safe egress client.

2. **Instant CRUD Object-Level Authorization Boundary**:
   - Enhance the Instant CRUD handler's authentication and authorization verification.
   - For consumer API key authentication: verify that the consumer belongs to the same consumer group or tenant as the connection, or that the connection is explicitly marked as shared/global.
   - For API client JWT authentication: verify that the token claims explicitly include the target connection identifier or permission scope.
   - Return standard HTTP 403 Forbidden envelopes with audit logging upon authorization failure.

3. **Documentation Router Seam & Scope Filtering**:
   - Dismount `/openapi.json` and `/postman.json` from the unauthenticated root router on the Control Plane.
   - Serve documentation exclusively through the authenticated admin subrouter, ensuring user claims are available.
   - If an unauthenticated request arrives at documentation endpoints, require explicit public flag evaluation or return HTTP 401 Unauthorized.

4. **Edge Ingress Cleanliness & Port Separation**:
   - Remove the Prometheus metrics handler registration from the public Data Plane router (`:8000`).
   - Retain `/metrics` exclusively on the dedicated Status & Metrics router (`:8100`).
   - Update container compose definitions to bind port `:8100` to `127.0.0.1`.
   - Incorporate the security headers middleware into the Data Plane proxy execution chain.

5. **Stateful JWT Revocation & Identity Integrity**:
   - Update the administrative authentication service to verify user record status during token validation.
   - Check that the user account exists, is active, and that the token's issued-at timestamp is greater than or equal to the user's password updated timestamp.
   - Maintain a lightweight in-memory cache with short TTL (e.g. 1 minute) to avoid database bottlenecking on high-frequency authenticated operations.

6. **Administrative Lockout Guardrails**:
   - In user management deletion and role update handlers, check the authenticated caller's identity against the target user identifier and reject self-deletion.
   - Perform an atomic query checking the count of active Super Admins before allowing a Super Admin deletion or role change, rejecting operations that would reduce the count below 1.

7. **Production Configuration Whitelist & Blacklist Synchronization**:
   - Expand the startup validator to reject all static placeholder values originating from Compose files, documentation examples, and test fixtures when `APP_ENV=production`.
   - Ensure clear, actionable error messages instruct operators to configure high-entropy environment variables.

8. **Setup Rate Limiting**:
   - Attach rate limiting middleware to the initial Super Admin creation endpoint (`/api/v1/auth/setup`), capping attempts to 5 requests per minute per client IP.

## Testing Decisions

1. **High-Seam Behavioral Verification**:
   - Tests will evaluate external HTTP responses and status codes via end-to-end and component integration tests rather than testing private struct fields.
2. **Instant CRUD BOLA Tests**:
   - Authenticate with Consumer A's API key and attempt to access `/api/v1/instant/{connId_B}/table`; assert that HTTP 403 Forbidden is returned.
   - Verify that accessing authorized connections succeeds with HTTP 200 OK.
3. **SSRF Redirect Chasing Tests**:
   - Configure a mock HTTP test server that responds with an HTTP 302 redirect pointing to `http://169.254.169.254` or `http://127.0.0.1:8001`.
   - Execute an OAuth2 token fetch or health check against the mock server; assert that the redirect is rejected with an explicit security error and zero network packets are sent to the redirect destination.
4. **Documentation Privacy Tests**:
   - Send unauthenticated `GET /openapi.json` and `GET /postman.json`; assert that unauthenticated requests receive HTTP 401 Unauthorized or are rejected, rather than receiving private endpoints.
5. **Data Plane Metrics & Headers Tests**:
   - Request `GET /metrics` on port `:8000`; assert that the route returns HTTP 404 Not Found.
   - Send requests to public Data Plane proxy routes; assert that responses contain `X-Content-Type-Options: nosniff` and `X-Frame-Options: DENY`.
6. **Administrative Guardrail Tests**:
   - Attempt to call `DELETE /api/v1/users/{current_admin_id}` with Super Admin credentials; assert that HTTP 400 Bad Request is returned.
   - In a test database with exactly one Super Admin, attempt to delete that user; assert that the operation is blocked.
7. **Prior Art**:
   - Builds directly upon `tests/e2e/e2e_security_hardening_test.go`, `internal/core/egress/validator_test.go`, and `internal/console/server/users_test.go`.

## Out of Scope

1. Redesigning relational database schemas or modifying third-party database engines.
2. External OpenID Connect Identity Provider federation (Apirout remains an API Gateway).
3. Modifying Web Console user interface layout or styling.
4. Upgrading external Go compiler binaries on the host system (handled via Dockerfile and compiler toolchain instructions).

## Further Notes

- All changes maintain strict backward compatibility for valid configurations, existing consumer groups, and registered services.
- Remediation tasks will be decomposed into sequential vertical tracer-bullet tickets following the repository issue tracker workflow.
