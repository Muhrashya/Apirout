# 65: Unified Database Adapter Seam and MSSQL Introspection

**What to build:**
Consolidate connection pooling and schema introspection across PostgreSQL, MySQL, and Microsoft SQL Server into a unified `DatabaseAdapter` module. Implement standard Microsoft SQL Server `information_schema` introspection query logic to detect tables, columns, data types, and primary keys, matching the existing MSSQL query dialect. In the web console, remove the unbacked SQLite option from external target connection forms in `Connections.tsx`, keeping `modernc.org/sqlite` strictly dedicated to internal metadata registry storage.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] Unified `DatabaseAdapter` interface defined for connection pooling and schema introspection
- [x] `PostgresAdapter`, `MySQLAdapter`, and `MSSQLAdapter` implement driver pooling and schema introspection
- [x] Microsoft SQL Server schema introspection added via standard SQL Server `information_schema` views
- [x] Console `Connections.tsx` connection creation form removes unbacked SQLite target option
- [x] Unit tests verify schema introspection and driver pool acquisition across PostgreSQL, MySQL, and MSSQL

## Answer
Consolidated database connection pooling, connection strings, schema introspection, query dialect generation, and identifier escaping into a unified `DatabaseAdapter` interface (`internal/core/adapter/adapter.go`) with `PostgresAdapter`, `MySQLAdapter`, and `MSSQLAdapter`. Implemented Microsoft SQL Server schema introspection using standard SQL Server `information_schema` tables (`TABLES`, `TABLE_CONSTRAINTS`, `KEY_COLUMN_USAGE`, `COLUMNS`). Removed unbacked SQLite target options from the web console, and verified via comprehensive unit tests.
