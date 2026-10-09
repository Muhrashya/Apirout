# 66: Instant CRUD and Query Dialect Harmonization

**What to build:**
Wire Instant REST CRUD (`/api/v1/instant/:connId/:table`) and standalone queries directly through the unified `DatabaseAdapter` seam. Eliminate hardcoded PostgreSQL driver fallbacks in `instant_crud.go` (`dbTypeDriver`), dispatching query execution, parameter binding, and row serialization through the resolved `DatabaseAdapter` instance for PostgreSQL, MySQL, and Microsoft SQL Server. Ensure unsupported driver requests return typed, canonical error envelopes.

**Blocked by:** 65 (Unified Database Adapter Seam and MSSQL Introspection)

**Status:** completed

- [x] `instant_crud.go` dispatches query execution through the resolved `DatabaseAdapter` seam
- [x] Hardcoded `dbTypeDriver` fallback to PostgreSQL removed
- [x] MSSQL instant CRUD operations (`SELECT`, `INSERT`, `UPDATE`, `DELETE`) supported via `MSSQLAdapter`
- [x] Query generator and execution errors mapped to canonical gateway error envelopes
- [x] Unit and integration tests verify instant CRUD execution across PostgreSQL, MySQL, and MSSQL

## Answer
Harmonized `instant_crud.go` with the unified `DatabaseAdapter` seam, routing connection string generation, pooling, query generation, and identifier escaping through resolved adapter instances. Completely removed hardcoded `dbTypeDriver` fallback to PostgreSQL, added full Microsoft SQL Server instant CRUD support (`SELECT`, `INSERT`, `UPDATE`, `DELETE`), and verified via comprehensive unit tests.
