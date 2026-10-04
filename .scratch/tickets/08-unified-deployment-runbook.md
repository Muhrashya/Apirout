# 08: Unified Deployment Runbook & DevOps Onboarding Documentation

**Parent:** [#6](https://github.com/Muhrashya/Apirout/issues/6)
**Issue:** [#10](https://github.com/Muhrashya/Apirout/issues/10)
**Blocked by:** [#8](https://github.com/Muhrashya/Apirout/issues/8), [#9](https://github.com/Muhrashya/Apirout/issues/9)
**Status:** ready-for-agent

## What to build
Publish `deploy/README.md` and update root `README.md` and `docs/ARCHITECTURE.md`. Deliver step-by-step operational runbooks for:
1. Local Development (`docker compose up`)
2. Production Single-Node VPS with Nginx SSL (`deploy/docker-compose.prod.yml`)
3. Production Kubernetes Multi-Plane Cluster (`deploy/k8s/`)

## Acceptance criteria
- [ ] `deploy/README.md` documents all three deployment paths with prerequisites, env configuration, and execution commands.
- [ ] Root `README.md` and `docs/ARCHITECTURE.md` link to the new deployment runbooks.
- [ ] Security recommendations (firewalling, SSL certificate generation, IP allowlists) are documented clearly.
