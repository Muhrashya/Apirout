# 64: Outbound OAuth2 Client Credentials Grant Manager

**What to build:**
Implement authentic outbound OAuth2 Client Credentials grant management within `OutboundNegotiator`. When a service is configured with `oauth2-cc` outbound auth, the negotiator issues HTTP POST requests (`grant_type=client_credentials`) to the configured token URL, parses access tokens, caches tokens in-memory with TTL accounting and a proactive 60-second expiration buffer, and injects standard `Authorization: Bearer <token>` headers. Concurrency is handled using singleflight deduplication to prevent thundering herd spikes against token endpoints. Completely purge static `X-Apirout-OAuth2-Client` pseudo-headers.

**Blocked by:** 63 (Outbound Credential Negotiator and AWS SigV4 Signing)

**Status:** completed

- [x] Outbound OAuth2 Client Credentials HTTP POST token exchange engine implemented
- [x] Thread-safe in-memory access token cache with TTL and 60-second proactive refresh buffer
- [x] Go singleflight request deduplication preventing concurrent thundering herd token requests
- [x] Proxied upstream requests carry authentic `Authorization: Bearer <token>` headers
- [x] Static `X-Apirout-OAuth2-Client` pseudo-header purged
- [x] Unit tests verify token grant exchange, proactive refresh, and concurrent singleflight deduplication

## Answer
Implemented authentic outbound OAuth2 Client Credentials grant management in `OAuth2CCManager` (`oauth2_cc.go`) integrated with `OutboundNegotiator`. Features HTTP POST form exchanges (`grant_type=client_credentials`), thread-safe in-memory caching with a 60-second proactive refresh buffer, and Go `singleflight.Group` deduplication preventing thundering herd concurrency spikes. Proxied upstream requests inject authentic `Authorization: Bearer <token>` headers and static `X-Apirout-OAuth2-Client` headers have been completely purged. All tests pass cleanly.
