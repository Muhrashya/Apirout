# 27: Standalone Platform De-Kong and De-Hasura Sanitization

**What to build:** Establish Apirout as an autonomous, standalone open-source API engine and instant data proxy by sanitizing all references to "Kong" and "Hasura" across documentation, CLI help banners, and code comments, while preserving backward-compatible environment variable and header fallbacks.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] Audit and remove "Kong" and "Hasura" references from documentation files:
  - `docs/ARCHITECTURE.md` (reframe as Apirout's multi-plane architecture and instant data proxy).
  - `docs/QUICKSTART.md` (update roles and instant database steps to native terminology).
  - `docs/PLAN_PEMISAHAN_MULTIPLANES.md` (update architecture plan titles and references).
  - Architecture Decision Records under `docs/adr/`.
- [ ] Update CLI help text in `cmd/apirout/main.go` and Makefile banner to present Apirout cleanly as *"Apirout CLI - Open Source API Engine & Instant Data Proxy"*.
- [ ] Clean up internal Go code comments across `internal/config/`, `internal/gateway/`, `internal/console/`.
- [ ] Preserve silent backward-compatibility fallbacks:
  - Keep `getEnvAny("ROLE", "APIROUT_ROLE", "KONG_ROLE", "traditional")` in `internal/config/config.go`.
  - Keep `X-Hasura-Admin-Secret` header fallback in `internal/console/auth/jwt.go`.
- [ ] Execute `go test ./...` to guarantee 100% test pass rate and verify no regression in configuration or authentication.
