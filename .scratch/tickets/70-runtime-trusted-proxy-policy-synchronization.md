# 70: Runtime Trusted Proxy Policy Synchronization & Client IP Spoofing Shield

**What to build:** Synchronize the application's runtime trusted proxy policy with the TRUSTED_PROXIES configuration variable upon server startup so that edge proxy headers (X-Forwarded-For, X-Real-IP) are only accepted from verified network proxies and cannot be spoofed by untrusted clients.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] cmd/apirout/main.go invokes iputil.SetGlobalTrustedProxies(cfg.TrustedProxies) during server initialization
- [x] If running in production with no trusted proxies configured, all proxy headers are ignored and socket IP is strictly used
- [x] Unit tests verify that configuring custom trusted proxies overrides the default internal CIDR list
