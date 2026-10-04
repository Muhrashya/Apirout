# Spec: Server-Side Pagination and Log Explorer Telemetry

**Triage Label:** `ready-for-agent`

## Problem Statement

As traffic across the Apirout API Gateway increases, inspecting runtime telemetry and administrative events becomes slow and memory-intensive:

1. **Client-Side Slicing and DOM Bloat**: Both the API Request Explorer (`/logs`) and System Activity Logs (`/settings/activity`) fetch a fixed batch of records (up to 100) and perform filtering, search, and presentation entirely in browser memory. In production environments where thousands of requests flow through the gateway, loading large record sets into client memory causes page stutters, high memory usage, and poor responsiveness.
2. **Missing Server-Side Pagination**: Neither the monitoring endpoints nor the user interface provide server-side pagination controls. Operators cannot navigate beyond the initial fetch window to inspect older telemetry or historical administrative audit logs.
3. **Lack of Density and Page-Size Selection**: The user interface provides no mechanism to choose page size density (e.g. 10, 25, 50, or 100 entries per page), forcing operators to either scroll through unpaginated tables or accept arbitrary record limits.
4. **Inaccurate Filter Counts**: Because filtering currently happens in browser memory over an arbitrarily limited server fetch, search results and status code filters display misleading telemetry counts that only reflect the fetched subset rather than the entire persistent dataset.
5. **Loss of Navigation Context on Reload**: Active page numbers, page limits, and filter selections are not synchronized with browser URL query parameters. Reloading the page or sharing an inspection link resets the view to default state, frustrating collaborative debugging and operational audits.

## Solution

Implement an end-to-end server-side pagination and query-filtering engine across the Go backend and React management console for both **API Logs** and **System Activity Logs**:

1. **Server-Driven SQL Query & Count Engine**:
   - Extend registry data-access methods to accept parameterized pagination parameters (`page`, `limit`) and query filters (`search`, `method`, `status`, `event_type`).
   - Execute a matching `SELECT COUNT(*)` query over identical `WHERE` conditions so that pagination metadata accurately reflects the filtered persistent dataset.
   - Clamp page sizes safely (`limit` between 1 and 100, defaulting to 25).
2. **Standardized Paginated API Contract**:
   - Standardize paginated endpoint responses with a top-level `data` array and a `pagination` metadata block (`total`, `page`, `limit`, `total_pages`), preserving backward compatibility for existing consumers.
3. **Reusable Frontend Pagination Component**:
   - Build a clean, accessible pagination component featuring a page-size selector dropdown (`[10, 25, 50, 100]` entries per page, default 25), entry range summary (*"Showing 1 to 25 of 1,420 entries"*), Previous/Next navigation buttons, and direct page jump buttons.
4. **URL Query Parameter Synchronization**:
   - Synchronize current page, limit, and active filter selections with the browser URL (`?page=1&limit=25&search=...`), enabling bookmarking, browser history navigation, and direct link sharing.
5. **Debounced Search Inputs**:
   - Apply a 300ms debounce to search input fields before dispatching backend network requests, ensuring smooth typing and preventing excessive database queries.

## User Stories

1. As a platform operator, I want to paginate through API request logs on the `/logs` page, so that I can easily navigate historical traffic without crashing or freezing my browser.
2. As a site reliability engineer, I want to change the number of displayed logs per page (10, 25, 50, 100) using a dropdown selector, so that I can adjust the viewing density to my screen size.
3. As a developer debugging an incident, I want to search for a specific endpoint path or client IP and have the server search the entire database rather than just the first 100 records, so that I don't miss relevant error traces.
4. As an operator filtering requests by HTTP Method (e.g., `POST`) or Status Code (e.g., `5xx`), I want the pagination counter to display the true total of matching records, so that I know the exact volume of matching traffic.
5. As a security engineer reviewing audit trails on `/settings/activity`, I want to paginate through system activity events with identical page-size controls, so that I can audit historical administrative actions.
6. As a compliance auditor, I want to filter activity logs by event type and search keyword on the server, so that I can inspect all administrative mutations across long retention periods.
7. As an engineer sharing an investigation with a teammate, I want my active page, limit, and filter settings to be reflected in the browser URL, so that sending the link opens the exact same view for my colleague.
8. As a user navigating with the browser Back and Forward buttons, I want the table state to update accordingly, so that my navigation flow feels natural and responsive.
9. As a user typing quickly into the search box, I want the application to debounce my input before querying the server, so that my typing does not feel laggy and does not overwhelm the server with queries.
10. As an automated API consumer using the monitoring endpoints, I want the paginated responses to include a `pagination` metadata object alongside `data`, so that my client can programmatically crawl all logs.

## Implementation Decisions

1. **Paginated REST API Schema**:
   The endpoints `GET /monitoring/requests` and `GET /audit-logs` will return a standardized envelope:
   ```json
   {
     "data": [...],
     "pagination": {
       "total": 1420,
       "page": 1,
       "limit": 25,
       "total_pages": 57
     }
   }
   ```
   - If `total` is 0, `total_pages` is 0, and `data` is an empty array `[]`.
   - Default `limit` is 25; allowed values are bounded between 1 and 100.
   - Default `page` is 1; values less than 1 are normalized to 1.

2. **Query Parameters Contract**:
   - `GET /monitoring/requests`:
     - `page`: Integer (default `1`)
     - `limit`: Integer (default `25`)
     - `search`: String (case-insensitive search across `path`, `ip_address`, `details`)
     - `method`: String (case-insensitive match for `GET`, `POST`, `PUT`, `DELETE`, `PATCH`)
     - `status`: String (`2xx`, `3xx`, `4xx`, `5xx`, or specific numeric status)
   - `GET /audit-logs`:
     - `page`: Integer (default `1`)
     - `limit`: Integer (default `25`)
     - `event_type`: String (exact match on `event_type`)
     - `search`: String (case-insensitive search across `details`, `ip_address`, `error_message`)

3. **Database Query Architecture**:
   - In the database registry layer, queries will dynamically construct parameterized SQL strings with bind parameters (`$1, $2, ...` or `?` normalized via dialect rebind).
   - A single method will execute two queries:
     1. `SELECT COUNT(*) FROM audit_logs WHERE <filters>`
     2. `SELECT <columns> FROM audit_logs WHERE <filters> ORDER BY timestamp DESC LIMIT ? OFFSET ?`
   - Calculation: `offset = (page - 1) * limit`.

4. **Frontend Architecture & Component Design**:
   - Create a reusable `Pagination` component in `web/src/components/Pagination.tsx` accepting:
     - `currentPage`: number
     - `totalPages`: number
     - `totalItems`: number
     - `pageSize`: number
     - `pageSizeOptions`: number[] (default `[10, 25, 50, 100]`)
     - `onPageChange`: (page: number) => void
     - `onPageSizeChange`: (size: number) => void
   - Replace local-only filtering in `ApiLogs.tsx` and `ActivityLogs.tsx` with server-side query parameters.
   - Synchronize state using React Router `useSearchParams`.
   - Add a 300ms debounce hook / timer on text search inputs to avoid dispatching a request on every keystroke.

## Testing Decisions

1. **Registry & SQL Query Seam**:
   - Primary Seam: Unit/Integration tests in `internal/console/registry/audit_test.go`.
   - Test external behavior:
     - Verify correct total count and record slicing with various page and limit values.
     - Verify that search strings correctly match path and client IP.
     - Verify status code range filtering (`2xx`, `4xx`, `5xx`).
     - Verify that edge-case offsets (e.g. page beyond total pages) return an empty array with accurate total.
2. **HTTP Handler Seam**:
   - Primary Seam: Handler tests in `internal/console/monitoring/` using `httptest.NewRecorder()`.
   - Test external behavior:
     - Request `/monitoring/requests?page=1&limit=10` and verify HTTP 200, JSON schema with `data` length 10, and correct `pagination` values.
     - Request `/audit-logs?event_type=admin_login_success&page=1&limit=25` and verify filtered results and metadata.
3. **Frontend E2E & Browser Seam**:
   - Validate live in the browser using `browser_subagent`:
     - Load `/logs` and verify default view displays 25 rows and pagination controls.
     - Change dropdown to 10 rows per page; verify table refreshes and displays 10 rows.
     - Click Next Page; verify page 2 loads and URL reflects `?page=2&limit=10`.
     - Filter by status code; verify count updates and page resets to 1.
     - Navigate to `/settings/activity-logs` and perform the same verification.

## Out of Scope

1. **Full-Text Search Engine Integration**: Integrating external search indexes (Elasticsearch, OpenSearch, Meilisearch) is out of scope; database-native parameterized matching is used.
2. **Log Archival & Vacuuming Retention Jobs**: Background cron jobs for pruning logs older than N days are out of scope for this ticket.
3. **Server-Side HTML Rendering (SSR)**: Replacing the React SPA with server-rendered HTML is out of scope; the application remains a Vite React SPA with server-driven REST pagination.

## Further Notes

- Existing API responses that return `{ "data": [...] }` will remain backward-compatible because the array continues to be exposed under the `data` field; automated clients that do not parse the `pagination` field will continue to function without breakage.
- Both the monitoring requests explorer and system activity logs utilize the underlying `audit_logs` datastore table, allowing shared patterns for query building.
