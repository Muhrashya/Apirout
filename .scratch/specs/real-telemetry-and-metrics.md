# Spec: Genuine Gateway Telemetry, Status Breakdown, and Statistical Percentiles

**Triage Label:** `ready-for-agent`

## Problem Statement

Users and platform administrators viewing the API Gateway dashboard, traffic monitors, and analytics pages are presented with inaccurate or zeroed-out metrics:
1. Aggregate request counts, error rates, and top route rankings show `0` or empty states even when active traffic is passing through the gateway, because the dashboard queries an unused metrics storage rather than the real request logs.
2. Latency percentiles (P50, P90, P99) displayed in the analytics dashboard are calculated using synthetic frontend multipliers (multiplying average latency by 0.8, 1.5, and 2.8) rather than actual measured request latency distributions.
3. Traffic health charts and success percentage indicators fail or display `0.0%` because the monitoring API does not provide standard HTTP status code bucket aggregations (`2xx`, `3xx`, `4xx`, `5xx`).

As a result, operators cannot reliably observe gateway health, identify slow backend endpoints, or verify SLA compliance.

## Solution

Unify the gateway's telemetry pipeline onto the primary request log datastore and provide genuine statistical metrics:
1. Re-route monitoring aggregations (`GetMetricsSummary`, `GetTopEndpoints`) to compute directly from the active request logs, ensuring real-time consistency with recent request inspector logs.
2. Compute true statistical percentiles (P50, P90, P99) in the metrics service across all recorded request durations over the selected time window.
3. Provide granular HTTP status code aggregations (`status_2xx`, `status_3xx`, `status_4xx`, `status_5xx`) from the backend API.
4. Update the frontend observability dashboards to bind directly to these real values, eliminating all client-side multiplier approximations and fallback calculations.

## User Stories

1. As a platform operator, I want all gateway requests to be recorded in a single authoritative datastore, so that aggregate metrics and detailed request inspectors never diverge.
2. As a platform operator, I want the Overview dashboard to display the genuine total request count across the last 24 hours, so that I can accurately assess current gateway ingress volume.
3. As a DevOps engineer, I want the Top Routes table to accurately rank endpoints by actual traffic volume, so that I know which services handle the most load.
4. As a site reliability engineer, I want the analytics interface to report true P50, P90, and P99 latency percentiles calculated from actual request measurements, so that I can evaluate true user-facing latency distribution without synthetic estimates.
5. As an infrastructure engineer, I want the Live Traffic dashboard to display accurate counts for 2xx, 3xx, 4xx, and 5xx responses, so that I can monitor client versus upstream error ratios in real time.
6. As a security administrator, I want gateway-rejected requests (such as 401 Unauthorized or 429 Rate Limited) to be included in the total request and error counts, so that brute-force attempts and rate limit hits are visible in traffic vitals.
7. As an API developer, I want the mean and percentile latencies for each route in the Top Routes table to reflect real measured durations, so that I can identify slow upstream backends.
8. As a frontend user, I want the analytics cards to load real values from the API without local estimation math, so that I can trust the precision of the observability data.
9. As a compliance officer, I want audit logs and telemetry to represent 100% measured system behavior, so that external audits can verify system performance claims.
10. As a platform administrator, I want empty states (0 requests) to cleanly display 0ms latency rather than falling back to default placeholder numbers, so that zero-traffic environments are clear and unambiguous.

## Implementation Decisions

### 1. Telemetry Datastore Consolidation
- The telemetry service will query the primary request log storage (`audit_logs`) where the gateway runtime writes every executed request.
- The legacy separate `metrics` table will no longer be used for dashboard queries, ensuring zero latency lag or synchronization mismatch between request lists and aggregate summaries.

### 2. Statistical Percentile Engine
- Latency percentiles for P50, P90, and P99 will be calculated on the backend across all valid `duration_ms` values within the requested time window (`timestamp > now - 24h`).
- Calculations will handle zero-data scenarios gracefully by returning `0` rather than null or error.
- The returned metrics payload will expose explicit integer fields: `latency_p50_ms`, `latency_p90_ms`, and `latency_p99_ms` alongside `avg_duration_ms`.

### 3. HTTP Status Code Bucketing
- The metrics aggregation query will categorize status codes into four canonical HTTP families:
  - `status_2xx`: Status codes >= 200 and < 300
  - `status_3xx`: Status codes >= 300 and < 400
  - `status_4xx`: Status codes >= 400 and < 500
  - `status_5xx`: Status codes >= 500
- These will be returned in the dashboard stats API response alongside `total_requests`, `success_count`, and `error_count`.

### 4. Client-Side Approximation Removal
- All synthetic multiplier formulas (`Math.round(avg * 0.8)`, `* 1.5`, `* 2.8`) in the analytics UI components will be removed.
- Components will consume `stats.latency_p50_ms`, `stats.latency_p90_ms`, and `stats.latency_p99_ms` directly from the API response with nullish coalescing to 0.
- Traffic health cards and percentage badges will compute success rate using `(status_2xx / total_requests) * 100` when total requests > 0, defaulting to 100% when 0 requests exist.

## Testing Decisions

### Test Characteristics
A good test must verify external system behavior from the outside in:
- Execute requests through the gateway runtime.
- Query the monitoring API to verify that the aggregated metrics, status code buckets, and percentiles match the exact expected distribution.
- Do not assert on private database query builders or internal struct representations.

### Modules Tested
- **Gateway Runtime Execution**: Inbound request processing and telemetry emission.
- **Monitoring & Metrics Query Service**: Aggregate computation of total requests, status buckets, and distribution percentiles.
- **Console Server HTTP Handlers**: JSON serialization and response contract of the dashboard metrics endpoint.

### Proposed Test Seam (Single Highest Seam)
- **HTTP / Service Integration Seam**:
  Send a sequence of synthetic test requests with predetermined latency durations and status codes through the gateway handler (e.g. 5x 200 OK with 10ms, 3x 404 Not Found with 2ms, 2x 500 Error with 50ms). Then call `GetDashboardStats` via the monitoring service and assert:
  - `total_requests` == 10
  - `status_2xx` == 5, `status_4xx` == 3, `status_5xx` == 2
  - `latency_p50_ms` matches the median of the distribution
  - `latency_p90_ms` matches the 90th percentile of the distribution

### Prior Art
- Existing gateway executor tests in `internal/gateway/executor/handler_test.go` and `internal/gateway/auth/jwt_auth_test.go` that construct `httptest.NewRecorder` requests and assert on emitted status and recorder headers.

## Out of Scope
- Real-time WebSocket streaming of live requests (the existing polling/refresh mechanism is retained).
- Time-series rollup caching into secondary analytical databases (ClickHouse / Prometheus). SQLite and PostgreSQL in-database aggregation over the 24-hour window satisfies current scalability requirements.
- Custom percentile configuration in the UI (P50, P90, P99 are standardized across all views).

## Further Notes
- This spec directly resolves the findings documented during the system flow and telemetry review, eliminating all remaining simulated metrics in Apirout.
