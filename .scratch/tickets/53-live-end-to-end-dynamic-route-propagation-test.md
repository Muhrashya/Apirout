# 53: Live End-to-End Dynamic Route Propagation Test

**What to build:** Adds an automated integration test in `tests/e2e/` verifying live runtime route propagation across the Control Plane and Data Plane: creating a protected route, asserting 401 Unauthorized, updating `is_public = true` via the Control Plane API, and asserting the Data Plane immediately (<50ms) serves the unauthenticated request without requiring a container restart, followed by verifying 404 Not Found upon deletion.

**Blocked by:** 52

**Status:** completed

- [x] Add `TestE2E_DynamicRouteInvalidationAndHotReload` to `tests/e2e/e2e_deep_features_test.go`
- [x] Verify test passes against the running live stack
- [x] Run full test suite with `go test -v ./tests/e2e/...`
