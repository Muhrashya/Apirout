# 16: Edge Reverse Proxy HTTPS/TLS Termination and Subdomain Virtual Hosts

**What to build:** A hardened production Nginx edge reverse proxy configuration featuring an active HTTPS port 443 listener with modern TLS ciphers, automatic HTTP-to-HTTPS redirection on port 80, dedicated virtual host server blocks for consumer API traffic (`api.*`) and console traffic (`console.*`), default CIDR IP allowlists protecting administrative management endpoints (`/admin-api/`), and genuine backend healthcheck proxying.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] Edge proxy binds to port 443 with TLS termination and redirects all port 80 HTTP traffic to HTTPS.
- [ ] Dedicated virtual host configurations route consumer API requests to Gateway upstream (port 8000) and web console traffic to GUI upstream (port 8002).
- [ ] Administrative management endpoints (`/admin-api/`) are protected by active CIDR allowlisting rules (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, `127.0.0.1`), denying unauthorized public traffic by default per `docs/BEST_PRACTICES.md`.
- [ ] Static synthetic `/healthz` is replaced with authentic proxy pass-through to the backend gateway liveness check.
- [ ] Production compose stack mounts SSL certificates and verifies syntax with `nginx -t` cleanly in the automated deployment validation suite.
