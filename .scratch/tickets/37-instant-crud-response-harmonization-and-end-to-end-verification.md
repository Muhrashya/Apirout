# 37: Instant CRUD Response Harmonization & End-to-End Verification

**What to build:** Harmonizes Instant REST database CRUD responses (`POST`, `PUT`, `DELETE`, `GET`) with standard REST conventions using the Gateway Responder, and upgrades the end-to-end integration test suite to assert that 100% of gateway responses across all features carry valid `X-Request-ID`, `Server-Timing`, and structured error codes.

**Blocked by:** 36: Reverse Proxy Egress Transformer & Upstream Error Sanitization

**Status:** completed

- [x] Integrate `responder.JSON` and `responder.Error` into `internal/gateway/executor/instant_crud.go`.
- [x] Ensure Instant CRUD errors emit consistent hybrid envelopes with typed error codes (`BAD_REQUEST`, `RECORD_NOT_FOUND`, `INTERNAL_ERROR`).
- [x] Update `tests/e2e/e2e_features_test.go` to assert `X-Request-ID` and `Server-Timing` headers on health endpoints, proxy routes, Instant CRUD routes, and error responses.
- [x] Verify that HTTP 401, 404, 405, 429, and 503 error responses in E2E tests contain valid machine-readable `code` fields.
- [x] Run full test suite (`go test -v ./...`) and ensure 100% PASS rate with zero regressions.

