# 93: Interactive SQL Preview Security and Read-Only Guard

**What to build:**
Secure the console's interactive SQL runner against data corruption and destructive execution. In `QueryHandler.Preview`, validate statements against `executor.ValidateSQL` to reject mutating keywords (`DROP`, `INSERT`, `UPDATE`, `DELETE`), and check whether target connections are flagged as `is_read_only`, rejecting write attempts with `405 Method Not Allowed`.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] In `internal/console/server/queries.go` (`Preview` handler), call `executor.ValidateSQL(req.SQL)` before dispatching preview queries.
- [x] In `Preview` handler, check `conn.IsReadOnly`; if the connection is read-only, ensure mutating statements are prohibited and reject with `405 Method Not Allowed`.
- [x] In `web/src/pages/Queries.tsx`, display clear visual indicators when executing against a Read-Only connection and handle validation/permission errors gracefully.
- [x] Write unit tests verifying that `Preview` rejects `DROP TABLE`, `UPDATE`, and `DELETE` queries, and returns 405 on read-only connections.
