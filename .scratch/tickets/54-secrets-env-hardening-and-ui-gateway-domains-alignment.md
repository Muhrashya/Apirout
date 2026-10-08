# Ticket 54: Secrets, Environment Hardening, and UI Gateway Domains Alignment

**Status:** completed

## Overview
Purge legacy environment variable naming, synchronize `.env.example` with runtime configuration, rename the misleading "Domains & TLS" navigation to "Gateway Domains" per ADR-0005, and implement RBAC filtering in documentation exports.

## Acceptance Criteria
- [x] Purge `KONG_ROLE` alias from `internal/config/config.go` (ADR-0007 compliance).
- [x] Add all missing operational variables (`LOG_LEVEL`, `ENABLE_CONSOLE`, `STATIC_DIR`, `DATA_DIR`, `DB_PATH`, `REDIS_URL`, `DATABASE_URL`) to `.env.example` with clear comments and secure defaults.
- [x] Rename "Domains & TLS" navigation link, page title, and breadcrumb to "Gateway Domains" in `web/src/layout/AppLayout.tsx`, `web/src/pages/settings/Domains.tsx`, and `web/src/pages/settings/DomainFormPage.tsx`.
- [x] Implement user group RBAC filtering in `internal/console/server/docs.go` for OpenAPI and Postman export endpoints.
- [x] Verify Go compilation and frontend build pass with zero errors.
