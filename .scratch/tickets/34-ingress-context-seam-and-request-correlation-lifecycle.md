# 34: Ingress Context Seam & Request Correlation Lifecycle

**What to build:** An edge ingress context seam that extracts incoming or generates new UUIDv4 `X-Request-ID` correlation identifiers, records high-resolution arrival timestamps in the request context, and pre-populates `X-Request-ID` on the HTTP response writer so that all downstream handlers, middlewares, and early returns carry consistent correlation tracing.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] Create `internal/gateway/responder/context.go` with context keys and helper functions (`WithRequestContext`, `GetRequestID`, `GetStartTime`, `ElapsedMillis`).
- [ ] Create `internal/gateway/responder/middleware.go` providing `responder.Middleware(next http.Handler) http.Handler` that extracts or generates `X-Request-ID` and injects request state into `r.Context()`.
- [ ] Ensure `X-Request-ID` is set on `w.Header()` at the start of request execution.
- [ ] Unit tests in `internal/gateway/responder/context_test.go` verify ID extraction, UUID generation fallback, and timestamp tracking.
- [ ] Wire `responder.Middleware` into Data Plane ingress in `internal/gateway/app/app.go`.
