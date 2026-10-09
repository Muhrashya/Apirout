# 67: Target Pool Ketama Consistent Hash Ring

**What to build:**
Extract target selection and load balancing logic out of `handler.go` into an isolated `TargetPool` module. When an upstream is configured with `consistent-hashing`, construct an authentic Ketama consistent hash ring with 160 virtual nodes per target instance on a 32-bit ring using MD5 digests. Ensure that scaling the target pool up or down redistributes at most $1/N$ of keys, preserving upstream cache locality and sticky sessions. Implement scoped connection lease tracking with automated lifecycle decrement, eliminating unbounded memory growth.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] `TargetPool` module extracted from `handler.go` with minimal `SelectTarget` seam
- [x] Ketama consistent hash ring implemented with 160 virtual nodes per target on 32-bit ring
- [x] Consistent hash key extraction orders: `X-Consumer-ID`, `X-User-ID`, client IP fallback
- [x] Scoped connection lease tracking with release closure, replacing global unbounded map
- [x] Unit tests verify ring key distribution stability ($1/N$ remapping) and connection lease cleanup
