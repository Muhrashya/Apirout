# 84: Full-Stack Security Hardening Phase 2 Verification Suite

**What to build:** A comprehensive end-to-end integration test suite (`tests/e2e/e2e_security_phase2_test.go`) validating all Phase 2 security defenses across live HTTP endpoints: Instant CRUD cross-connection BOLA 403 enforcement, SSRF redirect chasing rejection, documentation authentication and scoping, metrics port isolation and data plane defensive security headers, stateful JWT revocation upon password update, and Super Admin self-deletion guardrails.

**Blocked by:** 77, 78, 79, 80, 81, 82, 83

**Status:** completed

- [x] E2E test asserts unauthorized consumer access to `/api/v1/instant/{connId}/*` returns HTTP 403 Forbidden
- [x] E2E test asserts outbound HTTP requests reject 301/302 redirects to internal loopback and cloud metadata
- [x] E2E test asserts documentation endpoints require authentication and restrict schema by group
- [x] E2E test asserts `/metrics` on port `:8000` returns HTTP 404, while port `:8100` serves metrics
- [x] E2E test asserts public Data Plane responses emit defensive security headers
- [x] E2E test asserts JWT tokens are revoked upon password update
- [x] E2E test asserts Super Admin self-deletion and last-admin deletion are blocked
- [x] All tests pass cleanly across `go test ./...` and `web/` build verification
