# 18: Initial Setup State Detection and Super Admin Provisioning API

**What to build:** Implement backend control plane endpoints to detect first-boot setup state and provision the root Super Admin account with permanent lifecycle locking and audit logging.

**Blocked by:** None (can start immediately)

**Status:** closed

- [x] Control plane exposes public discovery endpoint `GET /api/v1/auth/setup-status` returning `{ "setup_required": boolean, "has_super_admin": boolean }` based on atomic check for `SUPER_ADMIN` accounts.
- [x] Control plane exposes `POST /api/v1/auth/setup` accepting `email`, `password`, and `confirm_password`.
- [x] Validates email format, password matching, and minimum 8 characters; hashes password with bcrypt and inserts user with role `SUPER_ADMIN`.
- [x] Returns `201 Created` with JWT access token and user identity, and writes `super_admin_setup_completed` audit log.
- [x] Permanently locks setup: rejects any call to `POST /api/v1/auth/setup` with `409 Conflict` once a Super Admin exists.
- [x] Unit and HTTP router tests verify setup status, creation, password validation, and 409 conflict locking.
