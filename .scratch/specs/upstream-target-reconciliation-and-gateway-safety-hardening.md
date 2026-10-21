# Specification: Upstream Target Reconciliation, Instant API Security, and Gateway Safety Hardening

**Status:** ready-for-agent

## Problem Statement

Platform operators and security teams managing database endpoints, upstream pools, and edge routing in Apirout face four operational friction points and security hazards:

1. **Silent Dropping of Target Edits on Upstream Pools:** When editing an existing upstream pool in the management console, changes made to the targets table (such as provisioning new target instances, adjusting traffic weights, or retiring decommissioned servers) are silently dropped on submission. Operators receive a success message, but backend upstream targets remain unchanged, risking traffic delivery to failed or retired backends.
2. **Accidental Public Exposure and Syntax Failures on Auto-Generated APIs:** Generating Instant REST APIs for a connected database automatically exposes every table publicly without authentication (`is_public = true`) or rate limiting. Furthermore, table names are injected unquoted into raw SQL templates, causing queries for tables named with reserved SQL keywords (`order`, `user`, `group`) to crash with syntax errors upon generation.
3. **Rate Limiting Bucket Collisions for Anonymous Traffic:** When rate limiting is enabled on a non-public route where requests are unauthenticated or authentication is optional, an uninitialized consumer ID pointer causes all anonymous callers to share a single collision key (`route:<id>:client:`). A burst from one client locks out all other clients across the network.
4. **Mutating Operations Permitted on Read-Only Connections:** While connection profiles provide an explicit read-only toggle for read replicas and analytical engines, the Instant REST API engine continues to expose mutating endpoints (`POST`, `PUT`, `PATCH`, `DELETE`). Incoming write requests are routed to read-only replicas, generating cryptic database driver errors instead of clean, edge-enforced rejections.
5. **Hardcoded Gateway Loopback Addresses in the Web Console:** The web console uses hardcoded loopback addresses (`http://localhost:8000`) for query previews and copyable cURL commands, failing when accessed over remote networks, LAN IPs, or behind reverse proxies.

## Solution

1. **Declarative Upstream Target Reconciliation:** Implement comprehensive declarative target diffing within the upstream management workflow. Upon saving an edited upstream pool, the system calculates the delta between the desired targets and current server targets, executing creation, modification, and deletion calls in a synchronized batch.
2. **Secure-by-Default Instant API Generation with ANSI Identifier Quoting:** Ensure all auto-generated endpoints default to protected status (`is_public = false`, requiring API key authentication) and have default burst rate limits applied. Wrap all database table names in the target engine's SQL dialect quoting mechanism, and provide an interactive access policy selector in the generation dialog.
3. **Dual-Tier Rate Limiting Keying:** Enforce strict validation on consumer identity before key derivation. If a valid, non-empty consumer ID is present, isolate the bucket by consumer; otherwise, isolate the bucket by client IP address.
4. **Gateway-Level Read-Only Connection Enforcement:** Enforce connection read-only policy at the gateway routing seam. When a connection is designated as read-only, mutating HTTP verbs (`POST`, `PUT`, `PATCH`, `DELETE`) are immediately rejected at the ingress edge with `HTTP 405 Method Not Allowed`.
5. **Dynamic Gateway Host Resolution:** Dynamically construct gateway URLs using the current browser origin (`window.location`) with fallback to configurable environment overrides.

## User Stories

1. As an API platform operator, I want target modifications made while editing an upstream pool to be saved reliably, so that my traffic routing matches my configured backends.
2. As a platform operator, I want adding a new server instance to an existing upstream pool to persist immediately upon saving the pool, so that new capacity receives traffic without manual re-creation.
3. As a platform operator, I want removing a failed server target from an upstream pool to immediately delete the backend target, so that traffic is not routed to dead hosts.
4. As a platform operator, I want updating a target server's weight or listening port to persist on save, so that traffic distribution aligns with instance sizing.
5. As a database administrator, I want auto-generated Instant APIs to require API key authentication by default, so that sensitive internal database tables are not exposed to the public internet by accident.
6. As a database administrator, I want tables named with SQL reserved keywords (like `order` or `user`) to generate valid queries without syntax errors, so that I do not have to write custom wrapper queries manually.
7. As an API operator, I want to choose whether generated endpoints are protected by API keys or made public read-only during the generation workflow, so that I have explicit control over accessibility.
8. As a consumer making requests to public or unauthenticated endpoints, I want my rate limit allowance to be tracked by my own IP address, so that another caller's burst traffic does not exhaust my rate limit.
9. As an authenticated API consumer, I want my rate limit to be tracked against my consumer identity across multiple IP addresses, so that my quota follows my application credentials.
10. As a security architect, I want connections marked as read-only to reject write attempts at the gateway boundary, so that read replicas are protected against unintended mutations.
11. As an API consumer, I want mutating requests sent to read-only databases to receive a clear `HTTP 405 Method Not Allowed` with a descriptive error message, so that I immediately understand why my write was rejected.
12. As a remote developer accessing the Web Console over a LAN or VPN, I want the live query runner to connect to the actual gateway host, so that previews do not fail with loopback connection errors.
13. As an operator copying cURL commands from the console, I want the snippet to contain the current reachable gateway hostname, so that the command executes cleanly from my local terminal.
14. As an operator editing an existing route, I want toggling between public and protected status to provide clear visual feedback regarding authentication plugins, so that no hidden or orphaned plugins remain active unexpectedly.
15. As an operator configuring database services, I want inapplicable HTTP outbound authentication fields to be hidden, so that I am not confused by irrelevant configuration options.

## Implementation Decisions

### 1. Upstream Target Reconciliation Engine
- The upstream management form will track baseline targets loaded from the control plane alongside in-memory staged target modifications.
- Upon saving an existing upstream pool, the form will execute a declarative differential reconciliation:
  - Targets present in the form but absent on the server are dispatched to target creation.
  - Targets present on the server but deleted from the form are dispatched to target deletion.
  - Targets existing on both with modified ports, weights, or enabled states are dispatched to target update.
- The UI will display a unified saving state and notify the operator of consolidated success.

### 2. Instant API Security and Identifier Escaping
- The control plane's endpoint generation handler will accept an optional access policy in the request payload, defaulting to private (`is_public = false`).
- Default route rate limits of 60 RPM with a burst of 10 will be automatically applied to generated endpoints.
- Table names will be quoted using the SQL dialect engine associated with the target connection (e.g. double quotes for PostgreSQL and Trino, backticks for MySQL, brackets for SQL Server).
- The web console generation modal will present a radio selection: `Protected with API Key (Recommended)` vs `Public Read-Only`.

### 3. Rate Limiter Key Derivation Guard
- The execution pipeline's authentication stage will verify that consumer identifiers are non-nil AND contain non-empty string content before assigning a consumer-specific rate key.
- If consumer authentication has not resolved a non-empty identity (public routes or unauthenticated requests), the rate limiter will key strictly on client IP address.
- Consumer group rate limits will similarly be skipped if no valid non-empty consumer ID is attached.

### 4. Read-Only Target Ingress Guard
- The Instant CRUD handler and gateway execution router will evaluate connection metadata prior to query execution.
- If the target connection has `is_read_only = true` and the incoming HTTP method is not idempotent (`POST`, `PUT`, `PATCH`, `DELETE`), request processing will halt immediately with `HTTP 405 Method Not Allowed` and error code `ERR_READ_ONLY_TARGET`.
- The response will include an `Allow: GET, HEAD, OPTIONS` header.

### 5. Dynamic Gateway URL Factory
- A shared utility function in the web application will resolve the gateway base URL dynamically:
  - If a build-time or runtime environment variable `VITE_GATEWAY_URL` is defined, use it.
  - Otherwise, resolve dynamically via `${window.location.protocol}//${window.location.hostname}:8000`.
- All query runners, live preview inspectors, and cURL snippet generators will reference this utility.

## Testing Decisions

A high-quality test validates observable external behavior and contracts rather than internal state:
- **Upstream Target Reconciliation:** Test via the control plane management API that updating an upstream with an edited target list correctly creates new targets, modifies changed targets, and deletes omitted targets.
- **Instant API Generation Safety:** Test via the console HTTP server that `POST /api/v1/connections/{id}/generate-endpoints` generates endpoints with `is_public == false`, rate limiting enabled, and SQL containing correctly quoted table names.
- **Rate Limiting Key Isolation:** Test via the gateway execution pipeline that two distinct anonymous IP addresses sending requests to a non-public rate-limited route receive independent rate-limiting token buckets.
- **Read-Only Connection Enforcement:** Test via the Instant CRUD test suite that querying a read-only connection with `GET` succeeds, while `POST`, `PUT`, and `DELETE` return `HTTP 405 Method Not Allowed`.

Prior art for these tests exists in:
- `internal/gateway/executor/instant_crud_test.go`
- `internal/gateway/executor/handler_test.go`
- `internal/console/server/upstreams_handler_test.go`

## Out of Scope

- Introducing new third-party rate-limiting dependencies or external proxy libraries.
- Changing the underlying distributed Redis rate limiting sliding-window algorithm.
- Modifying physical database user privileges on target database servers (read-only enforcement is handled at the gateway layer).

## Further Notes

- Architectural decisions will be recorded in `docs/adr/0016-target-reconciliation-and-instant-api-safety.md`.
- Canonical domain terminology is maintained in `GLOSSARY.md`.
