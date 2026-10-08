# 58: Trusted Proxy Ingress Seam & Client IP Spoofing Shield

**What to build:**  
The gateway and console resolve client IP addresses strictly according to a configurable `TRUSTED_PROXIES` policy. If incoming requests do not arrive from an authorized proxy socket address (`r.RemoteAddr`), forwarded proxy headers (`X-Forwarded-For`, `X-Real-IP`, `CF-Connecting-IP`, `Fastly-Client-IP`, `True-Client-IP`) are discarded. When incoming connections arrive from a trusted proxy, headers are traversed from right to left to accurately identify the first untrusted client IP. Prevents IP allowlist bypass, Admin CIDR evasion, and rate limit spoofing.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] Add `TrustedProxies` configuration field to `internal/config/config.go` with env parsing (`TRUSTED_PROXIES`, `APIROUT_TRUSTED_PROXIES`) defaulting to strict empty in production and private subnets (`127.0.0.1/32,::1/128,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16`) in development.
- [x] Create `TrustedProxyChecker` and right-to-left IP extraction in `internal/core/iputil/`.
- [x] Update `httputil.GetClientIP` and `iputil.ExtractClientIP` to evaluate trusted proxies before trusting headers.
- [x] Ensure `stage_ingress.go` IP allowlist and rate limiting use the authenticated client IP.
- [x] Unit tests covering untrusted socket header rejection, trusted single proxy, trusted chained proxies right-to-left resolution, and invalid IP inputs.
