# 60: Control Plane Hardening (Timeouts, Credential Masking & Bastion SSH Policy)

**What to build:**  
Control plane HTTP servers enforce 15s read and 30s write timeouts to prevent Slowloris connection pool exhaustion; the service outbound authentication API redacts stored AES-256 ciphertexts into boolean status markers (`{ "token_configured": true }`) before returning JSON to the browser; and SSH bastion database tunnels strictly require valid host key fingerprints in production while logging explicit security warnings in development.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] Add `ReadTimeout: 15 * time.Second` and `WriteTimeout: 30 * time.Second` to `adminSrv` and `guiSrv` in `internal/console/app/app.go`.
- [x] In `internal/console/server/auth_plugins_handler.go`, redact encrypted ciphertext fields (`token_encrypted`, `secret_key_encrypted`, `password_encrypted`, `client_secret_encrypted`, `key_encrypted`) in `GetOutboundAuth` response to status boolean indicators.
- [x] In `internal/core/tunnel/tunnel.go`, enforce environment-aware `HostKey` policy (require `HostKey != ""` in production or fail fast; log security warning in development).
- [x] Unit tests for outbound credential redaction and bastion tunnel host key verification logic.
