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

### Declarative State Engine
A centralized control-plane module that ingests full declarative configuration specifications (YAML/JSON) and applies them atomically across all platform entities (groups, services, upstreams, targets, consumers, queries, routes) in a single database transaction.

### Radix Router
The in-memory, prefix-tree route matching module operating at the Data Plane ingress seam that resolves incoming HTTP paths to routes with sub-millisecond lookup, updated via EventBus invalidation sync with atomic pointer swapping.

### Managed Connection Pool
The centralized database connection lifecycle module in `internal/core/` that encapsulates database driver pooling, SSH bastion port forwarding, and socket teardown behind a single connection acquisition interface.

### Passive Circuit Breaker
An edge-local upstream resilience mechanism within the Data Plane that temporarily isolates failing targets upon reaching consecutive error thresholds and probes recovery in a half-open state, complementing active control-plane health checking.

### Gateway Responder
A deep outbound edge module operating at the Data Plane response seam that encapsulates correlation tracking (X-Request-ID), latency telemetry (Server-Timing), production error sanitization, and structured normalized error envelope formatting across all proxied and generated API routes.

### Normalized Error Envelope
The canonical, machine-readable JSON error representation emitted by the Gateway Responder containing `status`, `message`, `code`, `request_id`, and `timestamp`, eliminating redundant field duplication while providing deterministic client branching and end-to-end distributed tracing.

### Gateway Domain
A registered hostname binding in Apirout (`gateway_domains`) that enforces host-header validation across incoming edge traffic, ensuring traffic targets authorized domain names.

### Edge-Offloaded Ingress
The cloud-native architectural pattern where SSL/TLS termination and certificate life cycles are handled by an edge proxy, Ingress controller, or load balancer (Cloudflare, AWS ALB, Nginx), while internal Apirout Data Plane containers operate high-performance HTTP listeners.

### Dynamic Route CORS
Fine-grained, per-endpoint Cross-Origin Resource Sharing headers (`endpoints.cors_origins`) evaluated dynamically during request execution to allow specific origin whitelists per route.

### Staged Plugin Pipeline
A form-level lifecycle pattern in the management console where gateway plugins are drafted, configured, and staged in local state, committing to the control plane and live Data Plane only upon explicit manual form submission ("Save" / "Update"), preventing accidental runtime mutations.

### Multi-Plugin Logical OR
Gateway authentication lifecycle semantics where multiple heterogeneous authentication plugins attached to the same entity (route or service) are evaluated sequentially using Logical OR semantics; incoming client requests are authorized as soon as any active plugin successfully validates client credentials (first match wins).

### Modal Event Isolation
A UI boundary pattern ensuring interactive modal dialog actions, clicks, and form submissions do not bubble synthetic events to outer parent forms or trigger unintended validation cycles.

### Execution Pipeline
An ordered multi-stage gateway execution module (`internal/gateway/pipeline/`) operating at the Data Plane ingress seam that decomposes request lifecycle processing into discrete stages (Ingress, Routing & Auth, Upstream, Telemetry) evaluated through a strongly-typed `pipeline.Context`.

### Evented Registry
A decorated storage seam adapter (`internal/console/registry/evented_registry.go`) wrapping control-plane persistence operations with synchronous domain event publishing across the EventBus, ensuring persistence and Data Plane cache synchronization are indivisible.

### Trusted Proxy Ingress
An authoritative client IP resolution strategy that parses forwarded proxy headers (`X-Forwarded-For`, `X-Real-IP`) exclusively when the direct network connection originates from a pre-configured trusted proxy CIDR, preventing header spoofing and unauthorized IP allowlist bypass.

### Payload Shield
An ingress protection mechanism operating at the HTTP listener boundary that caps request stream size via explicit byte limits, rejecting oversized payloads with standardized HTTP 413 error envelopes before buffer memory exhaustion.

### Bastion Host Key Policy
An environment-aware verification control that validates remote SSH server public key fingerprints when tunneling database connections through an SSH jump host, preventing Man-in-the-Middle credential interception.

### Outbound Credential Negotiator
An outbound gateway authentication module operating at the reverse proxy upstream seam that autonomously manages upstream credentials, including RFC-compliant AWS SigV4 canonical request signing and asynchronous OAuth2 Client Credentials grant token retrieval, caching, and TTL refresh.

### Database Adapter Seam
A consolidated database interaction port encapsulating connection pooling, SSH bastion tunneling, schema introspection, dialect query generation, and instant CRUD execution behind a unified engine contract for PostgreSQL, MySQL, and Microsoft SQL Server.

### Consistent Hash Ring
An upstream load-balancing implementation in the Target Pool module utilizing Ketama-style virtual node distribution on a 32-bit integer ring, ensuring that adding or removing upstream targets redistributes at most 1/N of keys to preserve cache and session locality.

### Egress Shield
A centralized network outbound validation boundary (`internal/core/egress/`) that intercepts all egress requests—including database connection probes, service target relays, upstream target definitions, and OAuth2 client credential token exchanges—enforcing strict destination IP validation against loopback, link-local metadata (`169.254.0.0/16`), and unapproved private networks to prevent Server-Side Request Forgery (SSRF).

### Cryptographic Plane Separation
The architectural isolation between the Control Plane token signing key (`CONSOLE_JWT_SECRET`) and Data Plane token evaluation keys (`GATEWAY_JWT_SECRET`), ensuring that management tokens cannot be reused to authenticate against public consumer routes and vice-versa.

### AEAD Credential Storage
An authenticated at-rest symmetric encryption scheme utilizing AES-256-GCM with 96-bit unique nonces and integrity tags (`gcm:` tag prefix), ensuring any ciphertext tampering or bit-flipping on database passwords and inbound authentication plugin secrets is immediately detected and rejected.

### Cross-Plane Token Guard
A runtime authorization enforcement in the gateway Data Plane authentication engine that parses JWT claims and explicitly rejects tokens bearing administrative claims (`typ: "admin"` or administrative user identifiers), preventing privilege elevation across boundaries.

