# Ticket 2: Instant CRUD & Console Query Integration with SQL Dialect Engine

## Parent
Part of #11

## What to build
Refactor Instant CRUD (`internal/gateway/executor/instant_crud.go`) and Console Query Preview (`internal/console/executor/query_utils.go`) to delegate all dynamic query assembly and pagination wrapping to the core dialect engine, eliminating duplicate string formatting routines while preserving 100% backwards compatibility for URL query filters and JSON payloads.

## Acceptance criteria
- [ ] Refactor `instant_crud.go` to use `sqldialect.Engine` for SELECT, INSERT, UPDATE, and DELETE
- [ ] Replace custom filter parsing in `instant_crud.go` with `sqldialect.FilterCondition`
- [ ] Refactor `query_utils.go` to delegate `WrapWithPagination` to `sqldialect.Engine`
- [ ] Ensure all existing Instant CRUD endpoints and URL query parameters remain 100% identical
- [ ] All unit and integration tests in `internal/gateway/executor` pass without regression

## Blocked by
- Ticket 1
