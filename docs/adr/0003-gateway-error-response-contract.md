# ADR-0003: Gateway Error Response Contract & Telemetry Taxonomy

## Status
Accepted

## Context
A critical architectural review of the Data Plane Gateway (`:8000`) error response model highlighted key concerns regarding payload redundancy and API contract purity:
1. **Redundant String Duplication**: Prior iterations emitted both `"message"` and `"error"` containing identical string values (e.g., `"message": "Endpoint not found"`, `"error": "Endpoint not found"`). This was originally introduced as a temporary backward-compatibility shim for legacy console fetchers.
2. **Fragile Client Handling**: Without a canonical machine-readable error classification, client SDKs and UI applications were forced to rely on string-matching error messages or generic HTTP status codes.
3. **Traceability and Observability Gap**: Client-facing errors lacked explicit correlation IDs and timing telemetry, making production incident triage and latency SLA tracking cumbersome.
4. **Information Leakage Risk**: Raw reverse-proxy upstream connection errors (e.g., `dial tcp 127.0.0.1:59199: connect: connection refused`) risked leaking private infrastructure topologies to public consumers.

## Decision
1. **Normalized Public Error Envelope**: Eliminate redundant `"error"` string duplication. The canonical public gateway error schema is defined as:
   ```json
   {
     "status": false,
     "message": "Human-readable description",
     "code": "CANONICAL_ERROR_CODE",
     "request_id": "req-uuidv4",
     "timestamp": "2026-10-07T10:29:32.042386894Z"
   }
   ```
   - `status`: Always `false` on failure; preserved for fast, defensive boolean checks in frontend clients.
   - `message`: Single, authoritative human-readable message.
   - `code`: Strongly typed, uppercase enum identifier (e.g., `ROUTE_NOT_FOUND`, `METHOD_NOT_ALLOWED`, `UNAUTHORIZED`, `INVALID_API_KEY`, `RATE_LIMIT_EXCEEDED`, `NO_HEALTHY_TARGETS`, `RECORD_NOT_FOUND`, `INTERNAL_ERROR`).
   - `request_id`: High-entropy UUIDv4 correlation identifier linking edge responses to distributed audit logs.
   - `timestamp`: RFC 3339 UTC timestamp marking exact gateway emission.

2. **Transport-Level Observability Headers**: Every gateway response (success and error) is decorated with standardized telemetry headers:
   - `X-Request-ID`: Matches the body correlation ID.
   - `X-Gateway-Latency`: Gateway processing duration before egress.
   - `Server-Timing`: W3C-compliant metric tracking gateway overhead and upstream duration (`gateway;dur=X.XX, upstream;dur=Y.YY`).

3. **Upstream Error Sanitization**: Reverse proxy errors are strictly intercepted and sanitized by the Gateway Responder. Upstream socket disconnects and unreachable targets emit structured 502/504 errors without exposing internal IP addresses, ports, or raw Go runtime stack traces.

4. **Client Adapter Interoperability**: The web console client (`apiClient.ts`) adopts a progressive fallback pattern (`json.message || json.error || fallback`), ensuring smooth interoperability across transitioning endpoints.

## Consequences
- **Positive**:
  - Eliminates payload bloat and field ambiguity across all gateway endpoints.
  - Aligns with modern API gateway best practices (Kong, AWS API Gateway, GitHub, Stripe).
  - Enables deterministic client-side routing and UI handling via typed `code` enums.
  - Provides end-to-end distributed tracing between client bug reports and backend log queries.
  - Guarantees zero sensitive internal network socket exposure.
- **Trade-off**:
  - External API consumers expecting the legacy `"error"` string at the root level must update their parsing logic to read `"message"` or `code`.
