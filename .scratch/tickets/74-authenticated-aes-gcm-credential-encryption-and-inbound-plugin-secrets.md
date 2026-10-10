# 74: Authenticated AES-GCM Credential Encryption & Inbound Plugin Secrets Protection

**What to build:** Authenticated encryption (AES-256-GCM with AEAD tag) for stored database passwords and SSH keys, with symmetric encryption and API redaction for secrets stored within inbound authentication plugin configurations.

**Blocked by:** 73: Cryptographic Plane Separation & Cross-Context Token Guard

**Status:** completed

- [x] ConnectionHandler replaces unauthenticated AES-CFB encryption with AES-256-GCM AEAD encryption
- [x] Decryptor supports transparent migration from legacy CFB ciphertexts to GCM
- [x] Inbound auth plugin secrets (e.g. HMAC secret keys, Basic auth passwords) are encrypted before DB storage and redacted in API responses
- [x] Unit tests verify encryption, decryption, tampering detection, and secret redaction
