# 71: Centralized Network Egress Boundary & SSRF Shield

**What to build:** A network egress validation guard that intercepts outgoing HTTP requests and database connection tests, blocking connections targeting loopback addresses, cloud metadata (169.254.169.254), and private internal networks unless explicitly permitted.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] Core package provides ValidateSafeEgress(host string) checking IP resolution against loopback, link-local, and private CIDRs
- [x] ConnectionHandler.TestRaw rejects connection attempts to blocked internal IP addresses with an explicit validation error
- [x] OAuth2CCManager.GetToken validates tokenURL against safe egress rules before making HTTP POST requests
- [x] Service creation and target validation reject SSRF attempts targeting metadata and loopback endpoints
- [x] Unit tests verify blocking of 127.0.0.1, localhost, 169.254.169.254, and private subnets
