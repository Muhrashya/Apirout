# Ticket 1: Core SQL Dialect Engine with PostgreSQL, MySQL & MSSQL Generators

## Parent
Part of #11

## What to build
A pure, zero-dependency `internal/core/sqldialect` package that accepts structured query specifications and generates dialect-specialized, parameterized SQL strings and positional argument bindings for PostgreSQL ($1), MySQL (?), and Microsoft SQL Server (@p1), along with dialect-aware identifier quoting and pagination wrapping.

## Acceptance criteria
- [ ] Define `QuerySpec`, `FilterCondition`, `InsertSpec`, `UpdateSpec`, and `DeleteSpec` domain models
- [ ] Implement Postgres generator with `"ident"`, `$1` positional bindings, and `LIMIT/OFFSET`
- [ ] Implement MySQL generator with `` `ident` ``, `?` positional bindings, and `LIMIT/OFFSET`
- [ ] Implement SQL Server generator with `"ident"` (or `[ident]`), `@p1` bindings, and `OFFSET X ROWS FETCH NEXT Y ROWS ONLY`
- [ ] Implement `WrapPagination(rawSQL, dialect, limit, offset)` supporting all 3 dialects
- [ ] 100% in-memory table-driven unit tests verifying SQL syntax and argument values

## Blocked by
- None (can start immediately)
