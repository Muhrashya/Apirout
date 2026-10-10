# 78: Safe Outbound Egress HTTP Client & SSRF Redirect Chasing Shield

**What to build:** A centralized Safe Outbound HTTP Client factory in the core egress boundary. Outbound HTTP requests (OAuth2 Client Credentials token exchanges, upstream target health probes) must not only validate the initial target URL, but also bind a strict `CheckRedirect` callback that re-validates the target URL on every HTTP redirect hop (301/302/307/308). Redirects pointing to internal loopback (`127.0.0.1`), cloud metadata (`169.254.169.254`), or private RFC 1918 subnets are immediately blocked with an SSRF error.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] Core egress module provides a centralized Safe HTTP Client constructor with an enforced `CheckRedirect` hook and IP-validating dialer
- [x] OAuth2 Client Credentials manager utilizes the Safe HTTP Client for token fetching
- [x] Upstream target active health checker utilizes the Safe HTTP Client for health probing
- [x] Unit tests verify that HTTP redirects to loopback or cloud metadata IPs are rejected and never initiate a secondary outbound connection
