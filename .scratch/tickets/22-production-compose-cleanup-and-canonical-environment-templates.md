# 22: Production Compose Cleanup and Canonical Environment Templates

**What to build:** Clean up `deploy/docker-compose.prod.yml` by stripping out dormant Certbot volume mounts and ACME declarations, standardizing on mounted SSL certificates. Overhaul the root `.env.example` with clean, discrete fields only, and establish `deploy/.env.prod.example` for secure production infrastructure deployments.

**Blocked by:** 20: Production Secrets Guardrails and Fail-Fast Validation, 21: Purge Admin Credentials from Runtime and Standardize Discrete Datastore Config

**Status:** closed

- [x] Strip dormant Certbot volume mounts (`certbot_certs`, `certbot_www`) and associated declarations from `deploy/docker-compose.prod.yml`.
- [x] Ensure edge proxy in production compose mounts SSL certificates strictly from `./nginx/ssl:/etc/nginx/ssl:ro`.
- [x] Overhaul root `.env.example` to document only discrete datastore fields, omit `ADMIN_EMAIL`/`ADMIN_PASSWORD`, and include all active runtime options (ports, master secrets, rate limits, CORS, connection pooling).
- [x] Create `deploy/.env.prod.example` with production guidance and placeholder strings (`<GENERATE_..._CHARS>`).
- [x] Validate compose configuration syntax with `docker compose config` or deployment validation scripts.
