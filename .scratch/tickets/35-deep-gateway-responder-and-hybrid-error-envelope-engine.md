# 35: Deep Gateway Responder & Hybrid Error Envelope Engine

**What to build:** A deep Gateway Responder module at the Data Plane response seam that encapsulates structured error formatting, backward-compatible hybrid error envelopes (`status: false`, `message`, `error`, `code`, `request_id`, `timestamp`), and standardized telemetry headers (`Server-Timing: gateway;dur=...`, `X-Gateway-Latency`). Replaces raw error writes across router matching, method validation (405 with `Allow`), authentication (401), authorization (403), and rate limiting (429 with `Retry-After`).

**Blocked by:** 34: Ingress Context Seam & Request Correlation Lifecycle

**Status:** ready-for-agent

- [ ] Create `internal/gateway/responder/codes.go` defining canonical `type ErrorCode string` enum mapping standard gateway failures (`ROUTE_NOT_FOUND`, `METHOD_NOT_ALLOWED`, `UNAUTHORIZED`, `INVALID_API_KEY`, `FORBIDDEN`, `RATE_LIMIT_EXCEEDED`, `UPSTREAM_UNREACHABLE`, `UPSTREAM_TIMEOUT`, `CIRCUIT_BREAKER_OPEN`, `NO_HEALTHY_TARGETS`, `BAD_REQUEST`, `INTERNAL_ERROR`).
- [ ] Create `internal/gateway/responder/envelope.go` defining the hybrid JSON error envelope struct and JSON serialization helpers.
- [ ] Create `internal/gateway/responder/responder.go` providing `JSON(w, r, status, data)` and `Error(w, r, status, code, msg, extraHeaders...)` that automatically attach `X-Request-ID`, `Server-Timing`, and `X-Gateway-Latency`.
- [ ] Refactor `internal/gateway/executor/handler.go` to use `responder.Error` and `responder.JSON` for all routing, auth, and rate limiting error paths.
- [ ] Unit tests in `internal/gateway/responder/responder_test.go` verify envelope fields, timing headers, custom headers (`Allow`, `Retry-After`), and legacy key preservation.
