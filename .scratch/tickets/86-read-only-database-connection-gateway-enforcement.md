# 86: Read-Only Database Connection Gateway Enforcement

**What to build:**
Enforce read-only connection guarantees at the gateway routing boundary. When a connection is designated as read-only (`is_read_only: true`), any mutating Instant CRUD HTTP requests (`POST`, `PUT`, `PATCH`, `DELETE`) are aborted immediately with `HTTP 405 Method Not Allowed`, protecting read replicas and analytical engines from unintended writes.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] In `internal/gateway/executor/instant_crud.go`, check `conn.IsReadOnly` in `ExecuteTableCRUD` and CRUD operation handlers.
- [x] For mutating operations (`POST`, `PUT`, `PATCH`, `DELETE`) against read-only connections, return `HTTP 405 Method Not Allowed` with `Allow: GET, HEAD, OPTIONS` and descriptive error message.
- [x] In `internal/gateway/executor/stage_upstream.go`, enforce read-only checks before executing table CRUD mutations.
- [x] Add unit test asserting that `GET` succeeds while `POST`/`PUT`/`DELETE` return 405 on read-only connection profiles.

