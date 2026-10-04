# Spec: Production Secrets Hardening, Discrete Datastore Configuration, and Environment Sanitization

**Triage Label:** `ready-for-agent`

## Problem Statement

As Apirout transitions toward production readiness and multi-tenant cloud deployments, several configuration and secret hygiene vulnerabilities exist across the platform:

1. **Unshielded Production Admin Secret**: While `JWT_SECRET` and `APP_ENCRYPTION_KEY` have length and placeholder validation when `APP_ENV=production`, `ADMIN_SECRET` has zero production validation. An operator deploying to production can accidentally leave the default placeholder (`MasterSecretKey2026!`) active, granting unrestricted access to all control plane endpoints via the `X-Apirout-Admin-Secret` header.
2. **Insecure Datastore Password Defaults in Production**: The database password currently defaults to well-known development strings without failing fast in production mode, risking deployments running against public or semi-private networks with insecure database passwords.
3. **Redundant Admin Credentials in Environment**: Despite establishing the First-Time Setup Wizard as the primary onboarding mechanism, `ADMIN_EMAIL` and `ADMIN_PASSWORD` remain embedded in configuration structs, boot scripts, and compose templates, maintaining plain-text credentials in configuration files and bypassing the intended setup lifecycle.
4. **Divergent & Incomplete `.env.example`**: The repository's `.env.example` omits several actively supported configuration options (such as rate limits, CORS origins, and connection pool parameters) while lacking a dedicated production environment template (`.env.prod.example`).
5. **Dormant Certbot Artifacts in Production Compose**: The production Docker Compose stack declares dormant Certbot ACME volumes and configuration hooks that require unnecessary maintenance and generate confusion regarding external email dependencies when TLS certificates are directly mounted.

## Solution

Harden the configuration loader with strict production secret validation, sanitize all environment templates, purge legacy plain-text admin environment variables, and eliminate unused container volume artifacts:

1. **Strict Production Secret Fail-Fast Guardrails**:
   - Enhance the configuration loader in production mode (`APP_ENV=production`) to strictly validate all master secrets:
     - `ADMIN_SECRET`: must be non-empty, at least 24 characters, and reject the default placeholder string.
     - `JWT_SECRET`: must be at least 32 characters and reject default placeholder strings.
     - `APP_ENCRYPTION_KEY`: must be at least 32 characters and reject default placeholder strings.
     - `DB_PASSWORD`: must be non-empty and reject well-known default sample strings (`Password123!`, `ProductionSecurePassword123!`, `postgres`, `admin`).
     - `REDIS_PASSWORD`: emit an operational warning if empty in production environments.
   - Halt execution immediately on startup with a descriptive error message if any production constraint is violated.
2. **Purge `ADMIN_EMAIL` & `ADMIN_PASSWORD` from Code & Environment**:
   - Formally remove `AdminEmail` and `AdminPassword` from the runtime configuration struct and startup seeding logic.
   - Remove all references from environment examples and Docker Compose definitions.
   - Fresh deployments always begin in the **Initial Setup State**, establishing the web console Setup Wizard (or the manual `apirout create-admin` CLI command) as the sole authoritative onboarding mechanisms.
3. **Discrete Datastore Configuration Exclusivity**:
   - Standardize all configuration templates on explicit, discrete datastore fields (`DB_TYPE`, `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`, `DB_SSLMODE`).
   - Deprecate single-string connection URL guidance in user-facing environment templates, maintaining discrete fields as the canonical operational standard.
4. **Clean Production Docker Compose & Certbot Decoupling**:
   - Strip dormant Certbot volume declarations (`certbot_certs`, `certbot_www`) and associated comments from the production Compose stack.
   - Standardize edge TLS termination on mounted volume certificates or upstream cloud load balancers.
5. **Comprehensive Environment Templates**:
   - Overhaul the root `.env.example` with organized, clearly commented tiers for local development.
   - Provide a dedicated, hardened `deploy/.env.prod.example` for production infrastructure operators.

## User Stories

1. As a security engineer, I want the server to refuse to boot in production if `ADMIN_SECRET` is set to the default placeholder or is shorter than 24 characters, so that management APIs are never exposed with default keys.
2. As a platform administrator, I want the server to fail fast in production if the database password is set to a default sample password, so that insecure default passwords cannot accidentally reach production.
3. As an operator deploying a new instance, I want the system to prompt me through the Setup Wizard on first access without having plain-text admin credentials sitting in `.env` files, so that root credential provisioning is secure and intentional.
4. As an infrastructure engineer, I want database configuration in `.env.example` to use discrete, granular fields (`DB_HOST`, `DB_PORT`, `DB_USER`, etc.) rather than single connection strings, so that parameters can be managed cleanly in orchestrator secret stores.
5. As a DevOps engineer, I want `deploy/docker-compose.prod.yml` to be free of unused Certbot volumes, so that my production deployment does not prompt for ACME emails or mount empty directories.
6. As a site reliability engineer, I want a dedicated `deploy/.env.prod.example` template with production-grade guidance, so that I have a turnkey blueprint for deploying to secure environments.
7. As a developer running locally in development mode (`APP_ENV=development`), I want permissive secret validation, so that local development workflows and automated tests remain fast and frictionless.
8. As an auditor, I want all production configuration rejections to output clear, human-readable error messages detailing which environment key failed and how to remediate it.

## Implementation Decisions

1. **Configuration Loader Validation Contract**:
   - When `APP_ENV == "production"`:
     - Check `ADMIN_SECRET`:
       - Must not be empty.
       - Length must be >= 24 characters.
       - Value must not equal `"MasterSecretKey2026!"` or `"APIROUT_ADMIN_SECRET"`.
     - Check `JWT_SECRET`:
       - Length must be >= 32 characters.
       - Value must not equal default placeholder strings.
     - Check `APP_ENCRYPTION_KEY`:
       - Length must be >= 32 characters.
       - Value must not equal default placeholder strings.
     - Check `DB_PASSWORD`:
       - Must not be empty when `DB_TYPE` is `postgres` or `mysql`.
       - Must not equal `"Password123!"`, `"ProductionSecurePassword123!"`, `"postgres"`, or `"admin"`.
   - Any failure returns an explicit `fmt.Errorf` detailing the requirement, causing the process to exit with code 1.
2. **Purging Plain-Text Admin Credentials**:
   - In `internal/config/config.go`:
     - Delete `AdminEmail` and `AdminPassword` fields from `Config`.
     - Remove `getEnvAny("ADMIN_EMAIL", ...)` and `getEnvAny("ADMIN_PASSWORD", ...)`.
   - In `internal/console/app/app.go`:
     - Remove the `EnsureSuperAdmin(cfg.AdminEmail, cfg.AdminPassword)` boot block.
     - Registry startup log reports DB connection status without referencing admin email.
   - In `cmd/apirout/main.go`:
     - The explicit CLI command `apirout create-admin` remains available for manual console administration when needed.
3. **Production Compose & Reverse Proxy Cleanup**:
   - In `deploy/docker-compose.prod.yml`:
     - Remove `certbot_certs` and `certbot_www` from volumes.
     - Remove volume mounts referencing certbot from `edge-proxy`.
     - Standardize SSL volume mount to `./nginx/ssl:/etc/nginx/ssl:ro`.
4. **Environment Template Synchronization**:
   - Update `.env.example` with discrete fields only, organized by section: Core Settings, Network Ports, Master Secrets, Registry Database, Connection Pool Tuning, Redis Cache, Security & CORS.
   - Create `deploy/.env.prod.example` with placeholder strings formatted as `<GENERATE_SECURE_..._MIN_..._CHARS>` to prevent copy-paste deployment of default keys.

## Testing Decisions

1. **Configuration Validation Seam**:
   - Primary Seam: Unit tests in `internal/config/config_test.go`.
   - Tests verify:
     - Development mode (`APP_ENV=development`) allows defaults without error.
     - Production mode (`APP_ENV=production`) with default `ADMIN_SECRET` fails with expected error message.
     - Production mode with short `ADMIN_SECRET` (< 24 chars) fails.
     - Production mode with default `DB_PASSWORD` fails.
     - Production mode with strong, valid secrets (`ADMIN_SECRET`, `JWT_SECRET`, `APP_ENCRYPTION_KEY`, `DB_PASSWORD`) passes cleanly.
2. **E2E & Integration Seam**:
   - `tests/e2e/e2e_features_test.go` and full test suite (`go test ./...`):
     - Ensure existing tests and Docker compose environments pass without regressions.
     - Verify clean container startup with hardened configuration.
3. **Prior Art**:
   - Builds directly on existing configuration loading tests in `internal/config/` and auth validation in `internal/console/auth/setup_test.go`.

## Out of Scope

- Integrating external cloud KMS SDKs (AWS KMS, Azure Key Vault, HashiCorp Vault) into the binary.
- Automating ACME/Let's Encrypt HTTP challenge renewals directly inside the Apirout runtime.
- Multi-region database connection string rotation.

## Further Notes

- By eliminating `ADMIN_EMAIL` and `ADMIN_PASSWORD` from environment files, any fresh installation will always prompt operators through the Setup Wizard on first access, unifying the onboarding user journey across all deployment targets.
