# 75: Management API RBAC Hardening & Safe DSN Construction

**What to build:** Role-based access control restricting user enumeration (GET /api/v1/users) exclusively to SUPER_ADMIN users, and safe URL-encoding of credentials when building database connection strings.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] adminRouter wraps GET /api/v1/users with RequireRole("SUPER_ADMIN")
- [x] Database connection string construction safely URL-encodes username and password components
- [x] Non-super-admin users receive 403 Forbidden when attempting to list users
