# 06: Production Multi-Container Docker Compose Recipe with Network Isolation

**Parent:** [#6](https://github.com/Muhrashya/Apirout/issues/6)
**Issue:** [#8](https://github.com/Muhrashya/Apirout/issues/8)
**Blocked by:** [#7](https://github.com/Muhrashya/Apirout/issues/7)
**Status:** ready-for-agent

## What to build
Create `deploy/docker-compose.prod.yml` where Nginx serves as the sole external entrypoint on ports 80 and 443. All internal gateway listeners (`:8000`, `:8001`, `:8002`, `:8100`), PostgreSQL (`:5432`), and Redis (`:6379`) are isolated within an internal Docker bridge network without binding directly to public host interfaces.

## Acceptance criteria
- [ ] `deploy/docker-compose.prod.yml` defines the full production topology (Nginx, Apirout, PostgreSQL, Redis).
- [ ] Only ports 80 and 443 are bound to host interfaces on the Nginx service.
- [ ] Internal ports (8000, 8001, 8002, 8100, 5432, 6379) are kept strictly on the private container network.
- [ ] Healthchecks ensure dependencies start in correct order.
