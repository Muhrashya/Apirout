# 81: Stateful JWT Revocation & Password Rotation Invalidation

**What to build:** Enforce user state verification during JWT authentication. If a user account is deactivated or deleted, or if a user rotates their password, previously issued JWT tokens with an `IssuedAt` timestamp older than the password modification timestamp are immediately rejected with HTTP 401 Unauthorized. A lightweight in-memory cache is used to avoid datastore bottlenecking on high-frequency authenticated operations.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] User record tracks a password updated timestamp (`password_changed_at` / updated timestamp)
- [x] Authentication middleware validates that the user exists, is active, and the token was issued after the last password modification
- [x] Revocation / identity status checks utilize a cached lookup to preserve high throughput
- [x] Updating a user's password immediately invalidates older tokens
- [x] Unit tests verify that deactivating a user or rotating a password invalidates existing JWT tokens
