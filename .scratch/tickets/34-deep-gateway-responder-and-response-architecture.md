# 34: Deep Gateway Responder & Standardized Response Architecture

**What to build:** A deep outbound Gateway Responder module (`internal/gateway/responder`) operating at the Data Plane response seam that encapsulates request correlation (`X-Request-ID`), W3C RFC `Server-Timing` and `X-Gateway-Latency` headers, production error sanitization, and structured hybrid error envelope formatting (`code`, `request_id`, `timestamp`) across all proxied, instant CRUD, and route error execution paths.

**Blocked by:** 30: In-Memory Radix Router Seam for Hot-Path Ingress Routing, 32: Edge Passive Circuit Breaker & Health Synchronization

**Status:** ready-for-agent

- [ ] Create `internal/gateway/responder/` with `codes.go`, `envelope.go`, and `responder.go`.
- [ ] Define canonical `type ErrorCode string` enum mapping standard gateway failures (`ROUTE_NOT_FOUND`, `METHOD_NOT_ALLOWED`, `UNAUTHORIZED`, `INVALID_API_KEY`, `FORBIDDEN`, `RATE_LIMIT_EXCEEDED`, `UPSTREAM_UNREACHABLE`, `UPSTREAM_TIMEOUT`, `CIRCUIT_BREAKER_OPEN`, `NO_HEALTHY_TARGETS`, `BAD_REQUEST`, `INTERNAL_ERROR`).
- [ ] Implement `responder.Middleware`: extract incoming or generate new UUIDv4 `X-Request-ID`, record start time, and inject into `r.Context()`.
- [ ] Implement `responder.Error`: write hybrid JSON error envelope preserving legacy fields (`status: false`, `message`, `error`) while adding `code`, `request_id`, `timestamp`, `Server-Timing`, and `X-Gateway-Latency`.
- [ ] Implement `responder.JSON`: write successful responses with automatic `X-Request-ID`, `Server-Timing`, and `X-Gateway-Latency` headers.
- [ ] Implement `responder.ModifyProxyResponse` and `responder.HandleProxyError`: scrub internal hop-by-hop headers and backend server banners, sanitize upstream 502/504 errors without exposing raw OS socket paths or internal IPs to clients.
- [ ] Wire `responder.Middleware` into Data Plane ingress in `internal/gateway/app/app.go`.
- [ ] Replace raw `httputil.WriteJSONError` calls across `internal/gateway/executor/handler.go` and `internal/gateway/executor/instant_crud.go` with the deep responder.
- [ ] Add comprehensive unit tests in `internal/gateway/responder/responder_test.go`.
- [ ] Update end-to-end assertions in `tests/e2e/e2e_features_test.go` to verify `X-Request-ID`, `Server-Timing`, and error codes on all responses.
