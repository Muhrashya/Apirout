# Domain Glossary

This glossary defines canonical terminology used across the Apirout platform.

### Route
A user-facing routing rule defining how incoming HTTP requests matching specific paths, HTTP methods, and hostnames are matched and forwarded into the gateway.

### Service
An internal gateway abstraction representing an upstream target destination or data provider. A service can represent an HTTP reverse proxy backend, an upstream load-balanced target pool, or a connected database.

### Upstream
A logical pool of backend servers or instances that handle incoming traffic distributed across multiple targets using load-balancing algorithms (e.g. Round Robin, Consistent Hashing, Least Connections).

### Target
An individual server instance identified by host IP/domain and port, belonging to an Upstream pool, subject to active health checking.

### Consumer
An authenticated client entity or external application interacting with the API gateway, identifiable via API keys, basic authentication, HMAC signatures, or JWT tokens.

### Consumer Group
A logical tier or policy group applied to multiple consumers, enforcing shared rate limits (requests per minute and burst capacity) and route access permissions.

### Auth Plugin
A modular gateway interceptor executed during the request lifecycle to authenticate and authorize inbound client requests before routing.

### Instant CRUD
An automated REST generation capability that inspects a connected relational or document database schema and exposes standardized create, read, update, and delete endpoints without requiring manual SQL writing.

### Standalone SQL Query
A parameterized database query defined in the console and executed securely by the gateway with row-level security and session parameter injection.

### Request Audit Log
An authoritative record capturing the timestamp, client identity, execution path, HTTP status code, duration, gateway overhead latency, and upstream round-trip latency for an executed API call.

### Trunk Branch
The default, continuously deliverable mainline branch (`main`) in a Trunk-Based development lifecycle where short-lived feature and bugfix branches merge.

### Release Tag
An immutable, annotated Git tag formatted as `vMAJOR.MINOR.PATCH` marking a formal production-ready milestone of the Apirout platform.

### Semantic Versioning
A standardized versioning scheme (`vMAJOR.MINOR.PATCH`) where `MAJOR` indicates breaking architectural or API changes, `MINOR` adds backwards-compatible features, and `PATCH` applies backwards-compatible bug and security fixes.

### Build-Time Version Injection
The process of setting application version metadata during binary compilation via linker flags (`-ldflags="-X main.version=..."`), ensuring Git release tags act as the authoritative source of truth.

### SQL Dialect Engine
A centralized, zero-dependency query builder that accepts structured query specifications and produces dialect-specialized, parameterized SQL strings and positional argument bindings for target database engines (PostgreSQL, MySQL, Microsoft SQL Server).

### EventBus Invalidation Sync
A decoupled synchronization port linking the Control Plane configuration publisher and Data Plane cache subscribers using strongly-typed `InvalidationEvent` domain messages, with production Redis pub/sub and in-memory test adapters.

### Super Admin
The highest-privilege administrative identity within the platform control plane possessing unrestricted authority across tenant user management, role-based access control, cryptographic secrets, database connections, and audit log inspection.

### Initial Setup State
A transient bootstrap phase active exclusively when zero accounts with the Super Admin role exist in the registry database. During this phase, access to the management console directs operators to the initial setup procedure, permanently locking upon successful creation of the root administrative identity.

### Master Admin Secret
A high-entropy shared secret header (`X-Apirout-Admin-Secret`) that bypasses interactive cookie/token authentication for headless operations, CI/CD pipelines, and automated health probes. In production environments, it is strictly validated against default placeholders and minimum length constraints.

