# 63: Outbound Credential Negotiator and AWS SigV4 Signing

**What to build:**
Deepen `OutboundInjector` into an authentic `OutboundNegotiator` module and implement RFC-compliant AWS Signature V4 request signing. When a service is configured with `aws-sigv4` outbound auth, the negotiator computes SHA-256 payload digests, canonical request strings, string-to-sign, and HMAC-SHA256 signature derivations (`AWS4-HMAC-SHA256`), attaching standard `Authorization: AWS4-HMAC-SHA256 Credential=... SignedHeaders=... Signature=...` and `X-Amz-Date` headers. Completely purge dummy custom header stubs (`X-Apirout-AWS-*`).

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] `OutboundNegotiator` interface and core engine established at reverse proxy seam
- [x] Authentic AWS SigV4 canonical request builder and HMAC-SHA256 signer implemented
- [x] Standard `Authorization: AWS4-HMAC-SHA256 ...` and `X-Amz-Date` headers injected on proxied requests
- [x] Dummy non-standard `X-Apirout-AWS-*` header injections purged
- [x] Unit tests verify standard AWS SigV4 test vectors against a mock upstream receiver

## Answer
Deepened `OutboundInjector` to implement `OutboundNegotiator`, implemented authentic RFC-compliant AWS Signature Version 4 request signing (`SignAWSSigV4` in `sigv4.go`), verified standard canonical request hashing, HMAC-SHA256 derivations, body integrity, and standard `Authorization: AWS4-HMAC-SHA256 ...` / `X-Amz-Date` header injection against a mock upstream receiver, and completely purged dummy `X-Apirout-AWS-*` header injections. All tests pass cleanly.
