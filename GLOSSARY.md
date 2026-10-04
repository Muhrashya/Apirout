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
