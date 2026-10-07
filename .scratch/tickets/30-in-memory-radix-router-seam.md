# 30: In-Memory Radix Router Seam with Sub-Millisecond Specificity Matching

**What to build:** An in-memory Segment Trie router (`RadixRouter`) at the Data Plane ingress seam that resolves incoming HTTP requests in sub-millisecond $O(K)$ time with priority matching (Exact > Param > Wildcard), returning RFC-compliant HTTP 405 Method Not Allowed when paths match unsupported methods, eliminating per-request database lookups.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] Create `internal/core/router/radix.go` implementing a thread-safe Segment Trie router.
- [x] Support segment parsing with priority: exact string match (`/api/v1/users`), parameterized segment (`/users/:id`), and wildcard catch-all (`/service/*`).
- [x] Store route definitions per node indexed by HTTP method; return an explicit `ErrMethodNotAllowed` alongside the allowed methods list when a path matches but the method does not.
- [x] Integrate `RadixRouter` into `internal/gateway/executor/handler.go`, loading initial active routes at gateway startup and resolving all inbound requests via the in-memory tree instead of executing PostgreSQL SQL queries on `:8000`.
- [x] Unit tests in `internal/core/router/radix_test.go` verify exact matches, dynamic parameter extraction, wildcards, and 405 Method Not Allowed error behavior.
