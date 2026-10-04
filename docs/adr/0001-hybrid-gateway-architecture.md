# ADR-0001: Hybrid Gateway Architecture & Unified Request Telemetry

## Status
Accepted

## Context
Apirout serves two complementary operational patterns:
1. **Kong-Aligned API Gateway**: Inbound requests route through Consumer Authentication -> Rate Limiting -> Route Matching -> Service Forwarding -> Upstream Target Pools.
2. **Database-as-an-API**: Routes target connected databases directly to perform Instant REST CRUD operations or execute parameterized SQL queries with Row-Level Security.

Previously, request metrics were split: the runtime wrote detailed logs into `audit_logs`, while dashboard analytics queried a secondary `metrics` table that was unpopulated, causing total volume and top route charts to report 0 while request logs showed live traffic. Furthermore, frontend percentiles were calculated via synthetic multipliers rather than measured latency distributions.

## Decision
1. **Dual-Route Dispatching**: The gateway executor dispatches routes targeting HTTP services or upstream target pools to the reverse proxy engine, and routes targeting database services to the Instant CRUD or SQL execution engine.
2. **Unified Telemetry Datastore**: All gateway execution events are recorded directly into `audit_logs` with exact round-trip upstream latency, gateway overhead duration, status code, and routing identifiers.
3. **True Statistical Metrics**: Dashboard statistics (`GetMetricsSummary`, `GetTopEndpoints`) aggregate directly from `audit_logs`, calculating exact status code breakdowns (`2xx`, `3xx`, `4xx`, `5xx`) and true statistical percentiles (P50, P90, P99) across recorded request durations, eliminating all client-side multiplier math.

## Consequences
- **Positive**: Single source of truth for telemetry; dashboard metrics, traffic monitors, and request logs are 100% consistent; real percentiles reflect genuine user latency distributions.
- **Trade-off**: Querying `audit_logs` over large timeframes requires indexing on `(event_type, timestamp)`. An index on `audit_logs(timestamp DESC)` is already maintained.
