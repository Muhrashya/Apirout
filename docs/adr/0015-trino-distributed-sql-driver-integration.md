# ADR-0015: Trino Distributed SQL Driver Integration

## Status
Accepted

## Context
Apirout provides automated Instant REST APIs, connection pooling, and parameterized SQL query execution across relational and document databases (PostgreSQL, MySQL, SQL Server, and MongoDB). Modern enterprise data architectures increasingly deploy **Trino** (formerly PrestoSQL) as a distributed SQL engine to query lakehouses (Apache Iceberg, Delta Lake, Apache Hive on S3/MinIO), analytical warehouses, and federated heterogeneous databases through a single endpoint.

Unlike traditional OLTP systems:
1. **Catalog & Schema Hierarchy**: Trino organizes tables in a 3-tier structure (`catalog.schema.table`) rather than a single database name.
2. **Analytical & OLAP Nature**: Trino is optimized for high-throughput distributed analytical queries. Many lakehouse connectors are read-only or append-only; Trino does not support standard row-level in-place `UPDATE` or transactional locks.
3. **Primary Key Constraints**: Tables residing in object storage or data lakes typically do not enforce primary key constraints in `information_schema`.
4. **Wire Protocol**: Trino communicates over HTTP/HTTPS REST protocol (`/v1/statement`) rather than custom binary TCP database sockets.

## Decision

1. **Catalog & Schema Mapping (Q1 A)**:
   - Represent Trino's catalog in the standard `DBName` field.
   - Support optional schema specification via slash notation (`<catalog>/<schema>`, e.g. `tpch/tiny` or `iceberg/analytics`). If omitted, default schema resolution to `default` or `public`.
   - Maintain backward compatibility with Apirout's existing `connections` table schema and management APIs.

2. **Analytical Instant CRUD & Mutation Guarding (Q2 A)**:
   - Optimize Trino Instant CRUD for analytical querying: full support for collection listing (`GET /api/v1/instant/{connId}/{table}` with query filters, limit, and offset) and Custom SQL Queries (`/api/v1/queries/{id}/run`).
   - Allow `INSERT` operations where the underlying catalog supports append operations.
   - For unsupported single-row mutation operations (such as row-level `PUT`/`PATCH`), return a clean HTTP 405 Method Not Allowed or 400 Bad Request error envelope rather than unhandled coordinator failures.

3. **Schema Introspection for Lakehouse Tables (Q3 A)**:
   - Query Trino's `information_schema` partitioned by catalog:
     - Table discovery: `SELECT table_name FROM <catalog>.information_schema.tables WHERE table_schema = ?`
     - Column discovery: `SELECT column_name, data_type, is_nullable FROM <catalog>.information_schema.columns WHERE table_schema = ? AND table_name = ?`
   - When no primary key constraint exists, classify the table as collection-queryable. Default ordering and single-entity lookup fall back to the first available column.

4. **Go Driver Integration (Q4 A)**:
   - Add the official pure-Go `github.com/trinodb/trino-go-client/trino` driver registering as `"trino"` under `database/sql`.
   - Implement `TrinoAdapter` adhering to `DatabaseAdapter`, `TestTrinoConnection` with `SELECT 1` ping verification, and ANSI SQL generator `TrinoGenerator` adhering to `sqldialect.Generator`.
   - Register Trino in the Web Console database catalog (`DatabaseEnginePickerModal.tsx`) with port 8080 and official driver branding (`logo-trino.png`).

## Consequences

- **Positive**:
  - Unlocks instant REST API generation and SQL proxying over big data lakehouses and federated data sources.
  - Zero CGO dependencies; operates purely over standard HTTP/HTTPS.
  - Safe error handling on unsupported mutations prevents data corruption or coordinator strain.
- **Operational Considerations**:
  - Analytical queries over large lakehouse datasets can take seconds to minutes; operators should set appropriate HTTP client timeouts and connection pool limits on Trino connections.
