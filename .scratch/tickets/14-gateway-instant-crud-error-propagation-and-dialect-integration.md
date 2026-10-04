# 14: Gateway Instant CRUD Error Propagation and Dialect Integration

**What to build:** Robust query execution in the gateway Instant CRUD handler where count query construction errors are verified and returned immediately as structured HTTP JSON error responses rather than silently ignored, guaranteeing reliable paginated data retrieval across all target relational database engines.

**Blocked by:** Ticket 13: Core SQL Dialect Engine Remediation and Parameter Normalization

**Status:** ready-for-agent

- [ ] Instant CRUD row count query assembly (`BuildSelect`) checks errors explicitly and writes a standard JSON 500 response if query generation fails, preventing empty SQL query execution against target databases.
- [ ] Instant CRUD endpoints (`GET /api/v1/instant/{connId}/{table}`) successfully execute sorted and filtered queries against target databases, returning authentic pagination metadata (`total`, `limit`, `offset`, `returned`).
- [ ] Integration and unit tests verify that invalid table schemas or query parameters generate clear error messages without server panics.
