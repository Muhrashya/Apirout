# 36: Reverse Proxy Egress Transformer & Upstream Error Sanitization

**What to build:** A deep reverse proxy egress transformer that attaches W3C RFC `Server-Timing: gateway;dur=X, upstream;dur=Y` headers to proxied responses, strips sensitive hop-by-hop headers and backend server banners, and sanitizes upstream 502 Bad Gateway / 504 Gateway Timeout network failures into clean client-facing envelopes without leaking internal IP addresses or OS socket paths, while preserving complete error traces in internal audit logs.

**Blocked by:** 35: Deep Gateway Responder & Hybrid Error Envelope Engine

**Status:** completed

- [x] Implement `responder.ModifyProxyResponse(r *http.Request, res *http.Response, targetID string, cb *circuitbreaker.CircuitBreaker)` in `internal/gateway/responder/proxy.go`.
- [x] Implement `responder.HandleProxyError(w http.ResponseWriter, r *http.Request, serviceName string, proxyErr error, targetID string, cb *circuitbreaker.CircuitBreaker)`.
- [x] Strip hop-by-hop and internal backend server identification headers (`Server`, `X-Powered-By`) in `ModifyProxyResponse`.
- [x] Inject `Server-Timing: gateway;dur=..., upstream;dur=...` into successful and proxied responses.
- [x] Refactor `reverseProxyRequest` in `internal/gateway/executor/handler.go` to delegate `proxy.ErrorHandler` and `proxy.ModifyResponse` to the deep responder.
- [x] Unit tests verify header stripping, dual-timing calculation, and error sanitization.
