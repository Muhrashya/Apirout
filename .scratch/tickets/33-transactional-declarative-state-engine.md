# 33: Transactional Declarative State Engine for Atomic GitOps Ingestion

**What to build:** A transactional declarative state engine replacing the empty GitOps import stub, executing topological DAG sorting across all entities (Groups, Connections, Services, Upstreams, Targets, Consumers, Queries, Endpoints) and applying idempotent UPSERTs inside a single atomic PostgreSQL transaction with strict rollback on failure.

**Blocked by:** 29: Core Engine Consolidation & Managed Connection Pool with SSH Bastion Lifecycle

**Status:** completed

- [x] Create `internal/console/gitops/engine.go` providing a deep `DeclarativeEngine` module that accepts a parsed declarative snapshot.
- [x] Implement topological DAG dependency ordering:
  1. Consumer Groups & Database Connections
  2. Services & Upstreams
  3. Targets & Consumers
  4. Standalone Queries
  5. Endpoints & Routes
- [x] Wrap all database operations inside a single SQL transaction (`tx, err := db.BeginTx(ctx, nil)`).
- [x] Perform idempotent UPSERT for each entity based on unique identifier/name within workspace.
- [x] On any error (foreign key violation, missing dependency, syntax error), immediately roll back (`tx.Rollback()`) and return a structured validation error identifying the failing entity.
- [x] On successful transaction commit, emit a `BULK_CONFIG_UPDATED` invalidation event to the EventBus port.
- [x] Connect `Handler.Import` in `internal/console/gitops/handler.go` to the deep `DeclarativeEngine`.
- [x] Integration tests verify that valid YAML/JSON imports persist all entities to PostgreSQL and invalid payloads roll back completely with zero orphaned records.
