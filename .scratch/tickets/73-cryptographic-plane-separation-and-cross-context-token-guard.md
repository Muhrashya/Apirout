# 73: Cryptographic Plane Separation & Cross-Context Token Guard

**What to build:** Distinct cryptographic keys and claim validation boundaries separating Control Plane administrator tokens from Data Plane API consumer tokens, preventing tokens issued for one plane from authenticating in the other.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] Support CONSOLE_JWT_SECRET for Control Plane admin users and separate GATEWAY_JWT_SECRET for Data Plane API clients
- [x] JWTAuthPlugin in gateway runtime explicitly rejects tokens presenting administrative claims (typ: admin)
- [x] Unit tests verify that an administrative token signed for the console is rejected by the gateway consumer auth chain
