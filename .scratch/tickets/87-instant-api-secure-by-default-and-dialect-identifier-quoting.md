# 87: Instant API Secure-by-Default and Dialect Identifier Quoting

**What to build:**
Secure auto-generated Instant APIs and prevent syntax errors on reserved keywords. Auto-generated endpoints must default to protected status (`is_public: false` requiring API key authentication) with default 60 RPM rate limits. Table names must be escaped with dialect-specific quotes (e.g. `"order"` or `` `user` ``), and an interactive access policy selector (`Protected with API Key` vs `Public Read-Only`) must be available in the generation modal.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] In `internal/console/server/connections.go`, look up SQL dialect generator for the connection's `db_type` and escape table names via `gen.QuoteIdentifier(table.Name)`.
- [x] In `internal/console/server/connections.go`, set `IsPublic: false` by default, accept optional `is_public` boolean in request body, and enable rate limiting (60 RPM / 10 burst).
- [x] In `web/src/pages/InstantAPI.tsx`, add an access policy radio selector in the generation confirmation dialog.
- [x] Add unit test verifying that tables named with reserved keywords (`order`, `select`) generate valid quoted SQL text and protected endpoints.

