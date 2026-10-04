# 23: Server-Side Paginated API Logs & Log Explorer Telemetry

**What to build:** An end-to-end server-side pagination and filtering engine for the API Logs explorer (`/logs`), including a standardized paginated API response contract, parameterized SQL queries with total count calculation, a reusable frontend pagination component with page-size selection, and full URL query parameter synchronization.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] `GET /monitoring/requests` accepts `page`, `limit`, `search`, `method`, and `status` query parameters and returns `{ "data": [...], "pagination": { "total", "page", "limit", "total_pages" } }`.
- [x] Backend SQL queries dynamically execute `SELECT COUNT(*)` and `LIMIT ? OFFSET ?` using parameterized bind variables against `audit_logs` where `event_type = 'api_request'`.
- [x] Reusable `Pagination` component in `web/src/components/Pagination.tsx` provides page size dropdown (`[10, 25, 50, 100]`), item range text (*"Showing X to Y of Z entries"*), Next/Previous buttons, and page number buttons.
- [x] `web/src/pages/observability/ApiLogs.tsx` synchronizes `page`, `limit`, `search`, `method`, and `status` with URL query parameters via `useSearchParams`.
- [x] Search input has a 300ms debounce before dispatching requests to the backend.
- [x] Changing page size or filters resets page to 1.
- [x] SQL registry tests and HTTP handler tests verify pagination and filtering behavior.
- [x] Live browser verification on `http://localhost:8002/logs` confirms responsive navigation, dropdown changes, and URL synchronization.
