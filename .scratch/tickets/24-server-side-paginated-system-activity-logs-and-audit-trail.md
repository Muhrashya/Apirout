# 24: Server-Side Paginated System Activity Logs & Audit Trail

**What to build:** An end-to-end server-side pagination and filtering engine for System Activity Logs (`/settings/activity`), integrating with the standardized pagination contract, parameterized SQL queries with total count calculation, URL query parameter synchronization, and the reusable frontend pagination component.

**Blocked by:** 23: Server-Side Paginated API Logs & Log Explorer Telemetry

**Status:** completed

- [x] `GET /audit-logs` accepts `page`, `limit`, `event_type`, and `search` query parameters and returns `{ "data": [...], "pagination": { "total", "page", "limit", "total_pages" } }`.
- [x] Backend SQL queries dynamically execute `SELECT COUNT(*)` and `LIMIT ? OFFSET ?` using parameterized bind variables against `audit_logs` excluding `api_request` events.
- [x] `web/src/pages/settings/ActivityLogs.tsx` utilizes the reusable `Pagination` component with page size selection (`[10, 25, 50, 100]`), item range text, and page buttons.
- [x] `ActivityLogs.tsx` synchronizes `page`, `limit`, `event_type`, and `search` with URL query parameters via `useSearchParams`.
- [x] Search input has a 300ms debounce before dispatching requests to the backend.
- [x] Changing page size or filters resets page to 1.
- [x] SQL registry tests and HTTP handler tests verify pagination and filtering behavior.
- [x] Live browser verification on `http://localhost:8002/settings/activity-logs` confirms responsive navigation, dropdown changes, and URL synchronization.
