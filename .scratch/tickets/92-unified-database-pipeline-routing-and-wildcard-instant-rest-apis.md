# 92: Unified Database Pipeline Routing and Wildcard Instant REST APIs

**What to build:**
Unify the dedicated Hasura-style `/api/v1/instant/{connId}/*` endpoints under the main gateway execution pipeline so rate limiting, IP allowlists, host validation, and request audit telemetry apply uniformly across all database traffic. Update Instant API route generation and route forms to register with method `ANY` (or full CRUD verbs), allowing clients to perform full CRUD mutations (`POST`, `PUT`, `PATCH`, `DELETE`) subject to connection `is_read_only` guards.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] In `internal/gateway/app/app.go`, remove the standalone `instantRouter` subrouter or route it directly through the gateway execution pipeline (`executorHandler.Execute`).
- [x] In `internal/gateway/executor/stage_upstream.go`, support `/api/v1/instant/{connId}/*` table CRUD resolution within the pipeline so rate limiting, IP allowlists, and telemetry audit logging are executed.
- [x] In `internal/console/server/connections.go` (`GenerateEndpoints`), register auto-generated routes with `Method: "ANY"` (or create endpoints that accept `GET`, `POST`, `PUT`, `DELETE`).
- [x] In `web/src/pages/routes/RouteFormPage.tsx`, when selecting Target: Instant API, support method `ANY` so that instant database endpoints permit CRUD operations.
- [x] Write unit tests verifying that `/api/v1/instant/{connId}/{table}` and custom Instant API routes execute through the pipeline, increment rate limits, emit audit logs, and enforce read-only 405 guards on write attempts.
