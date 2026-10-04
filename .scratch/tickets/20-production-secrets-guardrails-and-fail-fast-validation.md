# 20: Production Secrets Guardrails and Fail-Fast Validation

**What to build:** Implement strict fail-fast validation in `internal/config/config.go` when `APP_ENV=production`. Enforce that `ADMIN_SECRET` is non-empty, at least 24 characters, and not the default placeholder (`MasterSecretKey2026!`). Enforce that `DB_PASSWORD` is non-empty and does not match known default insecure passwords (`Password123!`, `ProductionSecurePassword123!`, `postgres`, `admin`). Verify `JWT_SECRET` and `APP_ENCRYPTION_KEY` criteria. Add comprehensive unit tests in `internal/config/config_test.go`.

**Blocked by:** None (can start immediately)

**Status:** closed

- [x] In `internal/config/config.go`, `APP_ENV=production` validates `ADMIN_SECRET` is non-empty, >= 24 characters, and rejects default placeholder `MasterSecretKey2026!`.
- [x] In `internal/config/config.go`, `APP_ENV=production` validates `DB_PASSWORD` is non-empty for relational databases and rejects default passwords (`Password123!`, `ProductionSecurePassword123!`, `postgres`, `admin`).
- [x] Unit tests in `internal/config/config_test.go` verify development mode permits defaults, while production mode fails fast with descriptive error messages when weak or default secrets are supplied.
- [x] Unit tests verify that production mode succeeds when valid high-entropy secrets and custom database passwords are provided.
