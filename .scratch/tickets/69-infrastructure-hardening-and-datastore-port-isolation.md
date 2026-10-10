# 69: Infrastructure Hardening & Datastore Port Isolation

**What to build:** Secure default container infrastructure that binds PostgreSQL and Redis ports strictly to localhost (127.0.0.1) to prevent external network exposure, and enforces password authentication on Redis via --requirepass.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] docker-compose.yml binds PostgreSQL port 5432 to 127.0.0.1::5432 instead of all interfaces
- [x] docker-compose.yml binds Redis port 6379 to 127.0.0.1::6379 instead of all interfaces
- [x] Redis container command enforces --requirepass using a secure default or configured ${REDIS_PASSWORD}
- [x] pirout service passes the matching REDIS_PASSWORD to ensure continuous operational health
