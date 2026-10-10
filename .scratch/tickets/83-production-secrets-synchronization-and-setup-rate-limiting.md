# 83: Production Secrets Validator Hardening & Setup Rate Limiting

**What to build:** Extend the production startup fail-fast validator (`APP_ENV=production`) to detect and reject default compose fallback secrets (`super-secret-jwt-key-min-32-characters!` and `apirout-local-dev-aes-encryption-key-32!`). Attach rate-limiting middleware to the initial setup endpoint (`/api/v1/auth/setup`) to protect initial cluster bootstrap from brute-force flooding. Align deployment documentation in the repository to match active manifests.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] Production configuration validator checks reject known Docker Compose fallback keys for JWT and AES encryption
- [x] Application fails fast on startup with clear remediation instructions when default secrets are detected in production mode
- [x] `/api/v1/auth/setup` endpoint enforces rate limiting against abuse
- [x] Repository documentation references are aligned with active configuration templates
- [x] Unit tests verify fail-fast rejection of default compose secrets under `APP_ENV=production`
