# 80: Data Plane Metrics Isolation & Proxy Security Headers

**What to build:** Remove the `/metrics` endpoint from the public Data Plane port (`:8000`), restricting metrics scraping exclusively to the internal Status Plane (`:8100`). Bind port `8100` strictly to `127.0.0.1` in Docker Compose configurations. Wire the defensive HTTP security header middleware into the Data Plane proxy pipeline so that all gateway responses emit `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, and `Referrer-Policy: no-referrer`.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] `/metrics` route is removed from the public Data Plane router (`:8000`)
- [x] Internal status and metrics server (`:8100`) continues to serve `/metrics` for internal scrapers
- [x] Docker Compose binds port `8100` to `127.0.0.1:8100` rather than `0.0.0.0:8100`
- [x] Public Data Plane proxy responses include defensive security headers (`X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`)
- [x] Unit and component tests verify that `GET /metrics` on port `:8000` returns HTTP 404 and proxy responses carry required security headers
