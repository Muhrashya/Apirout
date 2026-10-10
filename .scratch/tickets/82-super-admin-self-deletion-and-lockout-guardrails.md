# 82: Super Admin Self-Deletion & Lockout Guardrails

**What to build:** Prevent administrative lockout in the user management API. Super Admins must not be permitted to delete their own user account (`DELETE /api/v1/users/{id}` where `id == caller.id`). Furthermore, an atomic check must ensure that deleting or demoting any Super Admin account requires at least one other active Super Admin to remain in the system, preventing accidental permanent platform lockout.

**Blocked by:** 81

**Status:** completed

- [x] `DELETE /api/v1/users/{id}` rejects attempts by a Super Admin to delete their own account with HTTP 400 Bad Request
- [x] User deletion and role update handlers verify that at least one other active Super Admin remains before allowing deletion or demotion
- [x] Appropriate normalized error responses and audit log entries are generated upon blocked operations
- [x] Unit tests verify self-deletion rejection and last-admin deletion blocking
