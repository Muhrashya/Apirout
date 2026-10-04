# 05: Production Nginx Edge Reverse Proxy Configuration & Header Propagation

**Parent:** [#6](https://github.com/Muhrashya/Apirout/issues/6)
**Issue:** [#7](https://github.com/Muhrashya/Apirout/issues/7)
**Blocked by:** None (can start immediately)
**Status:** ready-for-agent

## What to build
Create `deploy/nginx/` containing optimized, security-hardened Nginx configurations (`nginx.conf` and `apirout.conf`). Provide upstream routing for Gateway Proxy (`:8000`) and Web Console GUI (`:8002`), strictly restrict Admin API (`:8001`), block external access to Status/Metrics (`:8100`), enable WebSocket upgrades, and propagate client real IPs (`X-Forwarded-For`, `X-Real-IP`, `X-Forwarded-Proto`). Add automated Nginx configuration syntax verification (`nginx -t`).

## Acceptance criteria
- [ ] `deploy/nginx/nginx.conf` provides production settings (worker processes, gzip, buffer sizes, HTTP/2, connection timeouts).
- [ ] `deploy/nginx/conf.d/apirout.conf` routes traffic to Gateway Proxy (`:8000`) and Console GUI (`:8002`).
- [ ] Admin API (`:8001`) is restricted with IP allowlist directives.
- [ ] Status/Metrics (`:8100`) returns 403 Forbidden to external callers.
- [ ] Client real IP and protocol headers are propagated properly.
- [ ] Automated validation confirms configuration syntax passes `nginx -t`.
