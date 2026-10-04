# 13: Core SQL Dialect Engine Remediation and Parameter Normalization

**What to build:** An enhanced SQL dialect generator engine that prevents Microsoft SQL Server derived-table subquery syntax errors when paginating queries containing existing sort clauses, extracts duplicated slice coercion logic into a single shared core utility, and cleans up unused returning clauses across dialect generators.

**Blocked by:** None (can start immediately)

**Status:** closed

- [x] MSSQL query pagination wrapping intelligently inspects the raw SQL query and handles queries with pre-existing `ORDER BY` clauses without triggering SQL Server `Msg 1033` derived-table errors.
- [x] Duplicated `toAnySlice` reflection and conversion logic is eliminated from `postgres.go`, `mysql.go`, and `mssql.go` and consolidated in a shared core module.
- [x] Table-driven tests in `dialect_test.go` assert correct SQL generation and parameter array construction for Postgres, MySQL, and MSSQL with complex `IN` slices, nested order clauses, and limit/offset bounds.
- [x] Unused or speculative returning parameters are cleanly resolved without impacting caller API contracts.
