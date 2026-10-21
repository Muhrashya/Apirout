# 95: Automated Telemetry Retention Pruner and Tenant RBAC Boundary Hardening

**What to build:**
Protect the registry database against unbounded table bloat by introducing a background maintenance worker governed by `AUDIT_LOG_RETENTION_DAYS` (default 14 days) that batch-deletes stale `request_audit_logs`. Enforce group permission checks in `ConnectionHandler.Update` preventing non-SuperAdmins from nullifying or altering connection `group_id`, add Group assignment controls in `UserFormPage.tsx` so users can be assigned to tenant groups upon creation, and align the `jwt-auth` schema with symmetric HS256 capabilities.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] In `internal/config/config.go`, add `AuditLogRetentionDays` (default: 14 days).
- [x] In `internal/console/monitoring/` (or registry), implement a background pruner routine that periodically executes `DELETE FROM request_audit_logs WHERE created_at < NOW() - INTERVAL 'X days'`.
- [x] In `internal/console/server/connections.go` (`Update`), prevent non-SuperAdmins from changing `existing.GroupID` to `nil` or to an unauthorized group ID.
- [x] In `web/src/pages/settings/UserFormPage.tsx`, add a tenant Group selector allowing administrators to assign users to groups upon creation and update.
- [x] In `web/src/components/AuthPluginManager.tsx`, align the `jwt-auth` configuration description and algorithm choices to explicitly declare symmetric HS256.
- [x] Write unit tests verifying that the retention pruner deletes aged audit logs and that connection group escalation is rejected with 403 Forbidden.
