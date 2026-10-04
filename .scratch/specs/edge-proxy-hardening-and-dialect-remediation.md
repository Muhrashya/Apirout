# Spec: Edge Reverse Proxy Hardening, Subdomain Routing, and SQL Dialect Seam Remediation

**Triage Label:** `ready-for-agent`

## Problem Statement

Following the recent architectural reviews of Apirout's deployment configuration and SQL dialect engine, several security gaps, dialect discrepancies, and code smells were identified that impact production deployments and maintainability:

1. **Unshielded Control Plane & Incomplete Edge TLS**: The edge reverse proxy configuration currently listens only on HTTP port 80 without an active HTTPS port 443 TLS termination server block. Furthermore, the administrative control plane route is exposed on the public reverse proxy without active CIDR allowlisting, violating documented network isolation standards that prohibit exposing the control plane directly to the public internet.
2. **Subdomain Routing Ambiguity**: Traffic routing through the edge proxy relies entirely on path prefix matching under a single catch-all virtual host instead of dedicated virtual hosts for consumer API traffic (`api.domain.com`) versus management console traffic (`console.domain.com`), preventing clean corporate DNS and certificate segregation.
3. **Synthetic Edge Liveness Checks**: The edge reverse proxy serves a static synthetic health check response that returns success regardless of whether the backend gateway or control plane services are actually healthy or running, masking backend outages from upstream load balancers.
4. **MSSQL Derived Table Pagination Syntax Vulnerability**: When raw SQL queries containing an inner sort order are wrapped for pagination in Microsoft SQL Server, SQL Server rejects derived tables containing `ORDER BY` clauses without a top-level limit or offset clause (error `Msg 1033`).
5. **Silent Query Failure Risk**: The gateway Instant CRUD row counter ignores errors returned during query generation, risking silent runtime execution failures against the target database.
6. **Code Duplication & Repetitive Action Mapping**: Type-coercion logic for array parameters is duplicated across all relational database dialect generators, and event action parsing is repeated across publisher components rather than being centralized within the core domain model.

## Solution

Harden the edge reverse proxy, implement dedicated virtual hosts with TLS termination, correct MSSQL pagination wrapping, ensure strict error checking across query execution, and refactor duplicated dialect and event mapping logic into deep core abstractions:

1. **Production-Grade Edge TLS & Control Plane Shielding**:
   - Introduce an active HTTPS server block on port 443 with modern TLS ciphers, HTTP/2 support, and automatic HTTP-to-HTTPS redirection on port 80.
   - Enforce default CIDR IP allowlisting on administrative control plane routes at the edge proxy, allowing only private subnet and loopback access while providing clear operator configuration for public ingress overlays.
   - Separate virtual host server blocks for consumer API endpoints (`api.*`), administrative console GUI (`console.*`), and internal monitoring metrics (`status.*`).
   - Replace synthetic edge health checks with direct proxy pass-through to authentic backend service liveness probes.
2. **MSSQL Pagination Robustness**:
   - Enhance the Microsoft SQL Server dialect generator to intelligently handle queries with existing order clauses, ensuring generated derived table subqueries remain 100% compliant with T-SQL syntax constraints.
3. **Strict Query Error Propagation**:
   - Enforce rigorous error checking in the gateway Instant CRUD handler when building count queries, returning descriptive HTTP errors immediately if query assembly fails.
4. **Core Seam Consolidation**:
   - Extract duplicated slice reflection and conversion logic into a shared core dialect utility.
   - Centralize action string parsing onto the formal EventBus domain model, removing repetitive switch cascades across publisher adapters.
   - Ensure event bus network subscriptions cleanly release underlying resources if contexts are canceled prior to explicit termination.

## User Stories

1. As a security engineer, I want the edge reverse proxy to terminate TLS on port 443 using modern cryptographic protocols, so that all consumer and management traffic is encrypted in transit.
2. As an infrastructure operator, I want plain HTTP requests on port 80 to automatically redirect to HTTPS, so that unencrypted connections are systematically prevented.
3. As a platform administrator, I want administrative control plane endpoints to be restricted to private CIDR subnets by default at the edge proxy, so that internal management APIs cannot be scanned or breached from the public internet.
4. As a DevOps engineer, I want dedicated Nginx virtual host definitions for `api.example.com` and `console.example.com`, so that consumer gateway traffic and management console traffic can be routed to independent domain names with distinct SSL certificates.
5. As a site reliability engineer, I want edge proxy health checks to forward to the authentic backend liveness endpoint, so that external load balancers detect real backend gateway outages rather than receiving false-positive healthy status responses.
6. As an API developer querying a Microsoft SQL Server database, I want paginated queries with sort criteria to execute without syntax errors, so that T-SQL restrictions on subquery order clauses do not break my application's pagination.
7. As a gateway operator, I want query generation errors during Instant CRUD count requests to be returned with appropriate HTTP status codes, so that invalid queries fail fast rather than executing empty statements.
8. As a database administrator, I want query parameter array conversions to behave identically across PostgreSQL, MySQL, and Microsoft SQL Server, so that `IN` filter operators produce deterministic parameterized SQL across all database engines.
9. As a backend developer maintaining the event bus, I want invalidation action strings (`CREATE`, `UPDATE`, `DELETE`, `PURGE`) to be parsed through a centralized domain method, so that invalidation logic is never duplicated across adapters.
10. As a software architect, I want core dialect types to remain free of unused speculative generality fields, so that the domain model strictly reflects required gateway capabilities.
11. As a cluster operator running distributed data planes, I want event bus Redis subscription routines to shut down cleanly without leaving background resource leaks if the parent context terminates, so that worker pods maintain optimal memory stability.
12. As a frontend maintainer, I want all existing REST API paths, query parameter syntaxes, and console UI capabilities to remain completely unchanged, so that hardening the edge infrastructure requires zero frontend modifications.

## Implementation Decisions

1. **Edge Reverse Proxy Virtual Host Hierarchy**:
   - Transition the edge proxy configuration from a monolithic single-server catch-all file to structured virtual host specifications:
     - An HTTP-to-HTTPS redirect server listening on port 80.
     - A consumer API gateway server block listening on port 443 with HTTP/2 and WebSocket proxying, forwarding traffic to the Data Plane proxy.
     - A management console server block listening on port 443, forwarding traffic to the Control Plane web console.
     - An administrative API block protected by active CIDR allowlisting rules (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, `127.0.0.1`), denying all unauthorized external IPs.
     - A status and metrics location restricted strictly to private monitoring agents.
   - Replace synthetic 200 health responses with proxying directly to the gateway's native healthcheck endpoint.
2. **Microsoft SQL Server Dialect Engine Pagination**:
   - Refactor the SQL Server query pagination wrapper:
     - Check if the provided query contains an existing `ORDER BY` clause.
     - When pagination is applied to an arbitrary query, use T-SQL compliant wrapping mechanisms that avoid invalid `ORDER BY` usage in derived subqueries without windowing or offset clauses.
3. **Instant CRUD Query Error Handling**:
   - Refactor the gateway Instant CRUD handler to inspect and propagate errors from `BuildSelect` during row counting. If query construction fails, halt execution and write a standardized JSON error response.
4. **Domain Model & EventBus Cleanliness**:
   - Create a centralized slice coercion utility within the core SQL dialect module, removing the three duplicated functions from the dialect generators.
   - Introduce a formal parser method on the event action type within the EventBus package, accepting uppercase or lowercase action names and returning the corresponding typed domain action.
   - Refactor the publisher to delegate action parsing to the core domain method.
   - Enhance the Redis EventBus subscription lifecycle to listen on context cancellation and promptly close the underlying pub/sub channel.

## Testing Decisions

### What Makes a Good Test
- Tests must verify external observable behavior rather than internal private fields or unexported implementation details.
- Infrastructure configurations must be validated using containerized syntax and dry-run execution tools before deployment.
- Core SQL dialect and EventBus tests must execute purely in memory with zero dependencies on live network databases or Redis instances.

### Test Seams

1. **Seam 1: Nginx Configuration & Routing Dry-Run Seam (Edge Infrastructure)**:
   - Validate the edge proxy configuration using `nginx -t` within the containerized runtime environment, verifying syntax correctness, SSL parameter definitions, virtual host server blocks, and CIDR allowlist directives.
2. **Seam 2: End-to-End SQL Dialect Generation Seam (Core Engine)**:
   - Execute table-driven unit tests across PostgreSQL, MySQL, and Microsoft SQL Server verifying that `BuildSelect`, `BuildInsert`, `BuildUpdate`, `BuildDelete`, and `WrapPagination` produce valid SQL without syntax errors, including edge cases with pre-existing `ORDER BY` clauses and `IN` filter condition slices.
3. **Seam 3: Gateway Instant CRUD & Pagination Integration Seam (Component Seam)**:
   - Test Instant CRUD table querying with pagination, sorting, and filtering, asserting that count query failures immediately return HTTP 500 errors and successful queries return full pagination metadata.
4. **Seam 4: In-Memory EventBus Cluster Invalidation Seam (Core / Gateway Seam)**:
   - Verify that domain action parsing accurately maps all action string variants, and that the in-memory event bus correctly dispatches typed invalidation events to subscribers with immediate cache eviction.

### Prior Art
- Existing unit tests in `internal/core/sqldialect/dialect_test.go` and `internal/core/eventbus/bus_test.go`.
- Existing integration test in `internal/core/eventbus/integration_test.go`.
- Nginx configuration validation script in `scripts/validate_deploy.ps1`.

## Out of Scope

- Implementing automated ACME Let's Encrypt renewal background daemons inside the container stack (operators provide certs or use certbot sidecars).
- Adding support for non-relational database query dialects (e.g. MongoDB, Cassandra).
- Re-architecting internal gateway routing middleware or altering user-facing console UI layouts.

## Further Notes

- All changes must adhere to the domain glossary in `GLOSSARY.md` and production guidelines in `docs/BEST_PRACTICES.md`.
- Target repository is `Muhrashya/Apirout`.
