# Specification: Deep Gateway Responder & Standardized Response Architecture

**Status:** ready-for-agent

## Problem Statement

As an API consumer, platform engineer, and frontend developer integrating with the Apirout platform, I experience several critical architectural frictions and inconsistencies in API responses emitted by the Gateway Data Plane:
1. **Schema Fragmentation Across Handlers**: Gateway responses do not adhere to a single coherent schema. Errors emit flat, legacy JSON (`{"status": false, "message": "...", "error": "..."}`), while Instant CRUD mutations emit disparate shapes (`POST` returns `{"status": "created", "data": ...}` with status as a string, `PUT` returns `{"status": "updated", "id": ...}` omitting the resource, and query preview returns `{"data": [...]}`), forcing client SDKs and frontend components to write brittle, verb-specific parsing logic.
2. **Absence of Request Correlation**: Responses lack an authoritative correlation identifier (`X-Request-ID`), making it impossible for client applications experiencing production timeouts or authentication errors to link their failures to gateway server logs or the Request Audit Log database.
3. **Black-Box Gateway Latency**: Although the gateway internally tracks routing duration and upstream response time in memory, it never exposes these metrics to callers in standard HTTP headers (`Server-Timing` or `X-Gateway-Latency`), leaving API consumers unable to diagnose whether latency spikes originate from the gateway ingress layer or the upstream backend target.
4. **Information Leakage on Upstream Failures**: When an upstream backend target is unreachable or times out, the reverse proxy returns raw Go network error strings (`dial tcp 127.0.0.1:59199: connect: connection refused`), leaking internal infrastructure topology, hostnames, and socket paths to untrusted external callers.
5. **Shallow Code Duplication**: Response serialization and error formatting are duplicated across 40+ ad-hoc call sites using shallow 3-line utility functions, scattering header injection, error mapping, and status codes across the codebase.

## Solution

The platform introduces a **Gateway Responder** as a deep outbound module at the Data Plane response seam (`internal/gateway/responder/`):
1. **Unified Hybrid Error Envelope**: A backward-compatible, standard error envelope that preserves legacy root attributes (`status: false`, `message`, `error`) while adding machine-readable error codes (`code`), request correlation (`request_id`), and ISO-8601 timestamps (`timestamp`).
2. **Context-Driven Ingress State Propagation**: Ingress middleware extracts or generates an `X-Request-ID` and records the monotonic arrival timestamp in the request context, eliminating manual parameter propagation across routes.
3. **Standardized Observability Headers**: Every response passing through the gateway automatically carries `X-Request-ID`, RFC-compliant `Server-Timing: gateway;dur=X, upstream;dur=Y`, and `X-Gateway-Latency`.
4. **Sanitized Upstream Reverse Proxy Egress**: An edge egress transformer intercepts upstream 502 Bad Gateway and 504 Gateway Timeout conditions, scrubs internal operating system socket paths and backend IP addresses from client payloads, records complete diagnostics to the internal Request Audit Log, and emits standardized, sanitized error payloads.
5. **Strongly-Typed Error Code Catalog**: A formal enum of typed error codes with canonical HTTP status mappings across authentication, authorization, routing, rate limiting, upstream connectivity, and database dialect execution.

## User Stories

1. As an API client developer, I want all gateway error responses to include an unambiguous, machine-readable `code` field, so that my client code can programmatically handle errors without parsing unstructured English sentences.
2. As an API client developer, I want every gateway response to include an `X-Request-ID` header, so that I can provide support teams with an exact trace identifier when reporting production issues.
3. As a platform operator, I want the gateway to accept an incoming `X-Request-ID` header if provided by an upstream client or edge load balancer and echo it back, so that end-to-end distributed tracing is maintained.
4. As an API consumer, I want every response to carry a `Server-Timing` header detailing gateway routing overhead and upstream round-trip latency, so that I can immediately determine where latency is spent.
5. As an existing client application, I want error payloads to preserve the root `"status": false`, `"message"`, and `"error"` keys, so that my existing integration does not break when the gateway updates.
6. As a security engineer, I want upstream network errors (connection refused, DNS failure, TLS handshake timeout) to be sanitized before returning to the client, so that internal network topologies and server addresses are never exposed to attackers.
7. As a platform operator, I want full raw network errors and diagnostics to be preserved in the internal Request Audit Log even when sanitized for the client, so that operators can debug infrastructure failures.
8. As a frontend engineer, I want rate limit errors to return HTTP 429 with both a standard `Retry-After` header and an error code of `RATE_LIMIT_EXCEEDED`, so that user interfaces can render accurate retry timers.
9. As an API consumer requesting an unsupported HTTP method on an existing route, I want to receive an HTTP 405 Method Not Allowed response with a valid `Allow` header and code `METHOD_NOT_ALLOWED`, so that my client can conform to HTTP standards.
10. As an API consumer missing authentication on a protected route, I want to receive an HTTP 401 Unauthorized response with error code `UNAUTHORIZED`, so that my client can automatically trigger a token refresh flow.
11. As an API consumer providing an invalid or revoked API key, I want to receive an HTTP 401 Unauthorized response with error code `INVALID_API_KEY`, so that my client knows to prompt for new credentials rather than retry.
12. As an API consumer exceeding route or consumer group burst quotas, I want the response to explicitly identify the rate limit domain, so that I understand which quota was exhausted.
13. As a core developer, I want response construction, header injection, and latency calculation to be encapsulated in a single deep module behind a minimal interface, so that new routes automatically inherit compliant response behavior without boilerplate.
14. As an automated integration test suite, I want to assert external HTTP headers and envelope shapes across public endpoints, so that response compliance is verified without coupling tests to internal handler state.
15. As an API consumer executing Instant CRUD mutations, I want successful `PUT` and `PATCH` requests to return representations aligned with standard REST expectations, so that my client state remains synchronized without an extra query.

## Implementation Decisions

### Seam Placement & Architecture
- The **Gateway Responder** will live as a deep module at the Data Plane outbound seam (`internal/gateway/responder/`).
- The interface will be deliberately minimal, presenting only high-level verbs to callers:
  - Ingress Middleware: `Middleware(next http.Handler) http.Handler`
  - Success Response: `JSON(w http.ResponseWriter, r *http.Request, status int, data any)`
  - Error Response: `Error(w http.ResponseWriter, r *http.Request, status int, code ErrorCode, message string, extraHeaders ...map[string]string)`
  - Reverse Proxy Egress: `ModifyProxyResponse(r *http.Request, res *http.Response, targetID string, cb *CircuitBreaker) error`
  - Reverse Proxy Error: `HandleProxyError(w http.ResponseWriter, r *http.Request, serviceName string, proxyErr error)`

### Context-Driven Request State
- Ingress middleware wraps all requests entering `:8000`.
- It inspects `X-Request-ID`; if empty, it generates a high-entropy UUIDv4 identifier.
- It records `start := time.Now()` in `r.Context()`.
- It pre-sets `X-Request-ID` on the outbound response header so early returns or streaming writes always include the correlation ID.

### Hybrid Error Envelope Shape
```json
{
  "status": false,
  "message": "Invalid API key provided",
  "error": "Invalid API key provided",
  "code": "INVALID_API_KEY",
  "request_id": "req-9b81f18e-4a6c-4e89-8d82-c2e8c2578e91",
  "timestamp": "2026-10-07T09:45:00.123Z"
}
```

### Strongly-Typed Error Code Catalog
- Canonical error codes will be defined as a strongly-typed Go enum (`type ErrorCode string`):
  - `ROUTE_NOT_FOUND` (404)
  - `METHOD_NOT_ALLOWED` (405)
  - `UNAUTHORIZED` (401)
  - `INVALID_API_KEY` (401)
  - `FORBIDDEN` (403)
  - `RATE_LIMIT_EXCEEDED` (429)
  - `UPSTREAM_UNREACHABLE` (502)
  - `UPSTREAM_TIMEOUT` (504)
  - `CIRCUIT_BREAKER_OPEN` (503)
  - `NO_HEALTHY_TARGETS` (503)
  - `BAD_REQUEST` (400)
  - `INTERNAL_ERROR` (500)

### Telemetry Header Formats
- `X-Request-ID`: Correlation ID string.
- `Server-Timing`: W3C RFC compliant header formatted as `gateway;dur=<ms>, upstream;dur=<ms>`.
- `X-Gateway-Latency`: Gateway processing duration in milliseconds.

### Upstream Reverse Proxy Sanitization
- In `reverseProxyRequest`, `proxy.ErrorHandler` and `proxy.ModifyResponse` will delegate to `responder.HandleProxyError` and `responder.ModifyProxyResponse`.
- When an upstream connection fails (connection refused, reset by peer, timeout), the error handler will sanitize the message to `"Backend service <name> unreachable"` with error code `UPSTREAM_UNREACHABLE`, omitting socket file paths or backend IP ports.
- Sensitive hop-by-hop headers and internal backend server identification headers (`Server`, `X-Powered-By`) will be stripped on egress.

## Testing Decisions

- **Good Test Philosophy**: Tests must exercise external behavior across the public HTTP seam (`http.Handler` / port `:8000`) by sending real requests and asserting response headers, status codes, and JSON envelopes. Tests must not inspect internal private fields or mock standard library structures.
- **Seams Tested**:
  1. *Gateway Responder Unit Seam* (`internal/gateway/responder/responder_test.go`): Test middleware context extraction, header generation, latency calculation, and error envelope formatting.
  2. *Data Plane Ingress HTTP Seam* (`tests/e2e/e2e_features_test.go`): Fire requests across existing E2E tests and assert that every response (success, 401, 404, 405, 429, 502, 503) contains `X-Request-ID`, `Server-Timing`, and valid hybrid error envelopes with typed codes.
- **Prior Art**: Existing end-to-end suite in `tests/e2e/e2e_features_test.go`.

## Out of Scope

- Modifying the Admin Control Plane (`:8001`) internal REST CRUD endpoints.
- Introducing external distributed tracing collectors (e.g. OpenTelemetry collector, Jaeger).
- Altering the database schema or audit log storage layout.

## Further Notes

- All terminology in this specification strictly aligns with `GLOSSARY.md`, specifically the newly established `Gateway Responder` entry.
- Complements ADR-0002 by completing the Data Plane egress response lifecycle.
