# 41: End-to-End Multi-Plugin Verification and Live Seam Testing

**What to build:** Automated E2E testing validating that a route configured with multiple staged plugins (e.g. `key-auth` + `basic-auth`) evaluates with Logical OR on live Data Plane traffic (`:8000`), allowing clients with valid API keys or valid basic authentication, while rejecting invalid credentials with HTTP 401. Validates that updating route metadata without saving plugins leaves active authentication undisturbed. Rebuilds and verifies Docker containers and runs the full test suite.

**Blocked by:** 40

**Status:** completed

- [x] High-seam E2E test in `tests/e2e/e2e_features_test.go` verifies multi-plugin Logical OR authentication (`key-auth` and `basic-auth`) on live gateway (`:8000`)
- [x] Validates duplicate plugin creation is rejected by API with HTTP 400
- [x] Docker container rebuilt with clean web assets (`docker compose up -d --build apirout`)
- [x] Full E2E test suite (`go test -v ./tests/e2e/...`) passes 100%
