# ADR-0012: Runtime Architecture Boundaries and Full-Stack Security Seams

## Status
Accepted

## Context
Following comprehensive runtime architecture analysis with Archify and the remediation of full-stack security vulnerabilities (Tickets 69–76), Apirout required an authoritative architectural specification defining the boundaries between runtime components and the security seams protecting them.

Previously, component responsibilities were informally decoupled across several packages without formal boundary encapsulation:
1. Control Plane operations and Data Plane execution shared cryptographic keys (`JWT_SECRET`), enabling theoretical cross-plane token confusion.
2. Ingress traffic and outbound database/upstream connections lacked strict egress boundary validation, permitting potential Server-Side Request Forgery (SSRF) and metadata exposure.
3. Passwords and credentials at rest were stored using unauthenticated stream ciphers (AES-CFB) without tamper detection.
4. Datastore ports (PostgreSQL 5432, Redis 6379) were published across all host interfaces by default without mandatory Redis authentication.

## Decision
1. **Adopt the Four-Boundary Runtime Architecture**:
   Formally adopt the four structural boundaries established and verified via Archify:
   - **Apirout Control Plane (`:8001` / `:8002`)**: Hosts the Management REST API, RBAC policy enforcement, and the EventBus publisher (`Topic: apirout:events`) broadcasting route and schema updates.
   - **Apirout Data Plane (`:8000` / `:8100`)**: Stateless ingress engine hosting the `PayloadShield` (10MB body cap), `TrustedProxyChecker`, Radix Trie Router, and Prometheus telemetry exporter (`:8100`).
   - **Pipeline Execution Stage**: Decoupled multi-stage pipeline executing Radix Trie path matching, Multi-Plugin Inbound Authentication (Logical OR evaluation), Passive Circuit Breaking, Upstream Reverse Proxying, and Instant CRUD SQL generation.
   - **Protected Upstream & Datastore Network**: Isolated target network containing backend microservices, Redis distributed cache, and relational databases (PostgreSQL, MySQL, SQL Server) secured via loopback binding or SSH Bastion tunnels.

2. **Centralized Network Egress Boundary (`egress.Validator`)**:
   All outbound network requests—including database connection testing, upstream target probes, and OAuth2 Client Credentials token exchanges—must pass through the centralized `internal/core/egress/` boundary. The validator strictly resolves destination addresses and denies loopback (`127.0.0.0/8`, `::1`), link-local metadata (`169.254.0.0/16`), and private CIDRs unless explicitly permitted by an authorized allowlist.

3. **Cryptographic Plane Separation**:
   Administrative tokens for the Control Plane (`:8001`) are signed exclusively with `CONSOLE_JWT_SECRET`. Consumer tokens for the Data Plane (`:8000`) are evaluated exclusively with `GATEWAY_JWT_SECRET` or consumer-specific keys. The gateway auth engine enforces runtime rejection of any token carrying administrative claim types (`typ: "admin"`).

4. **Authenticated AEAD Credential Encryption at Rest**:
   All database connection passwords, SSH private keys, and inbound authentication plugin configurations are encrypted using AES-256-GCM with 96-bit unique nonces and integrity tags (`gcm:` tag prefix). Any ciphertext tampering fails immediately with an explicit authentication error, while legacy CFB ciphertexts are transparently migrated upon load. Inbound secrets are redacted in all management API responses.

5. **Living Architecture Artifact**:
   Maintain the interactive, self-contained Archify architecture model as a permanent repository documentation asset at `docs/architecture-runtime.html`.

## Consequences
- **Positive**:
  - Eliminates cross-plane privilege escalation and token confusion attacks.
  - Precludes internal network port scanning and cloud metadata theft via SSRF.
  - Ensures at-rest credential storage is tamper-evident and compliant with modern cryptographic standards.
  - Provides a single, unambiguous architectural model across engineering, operations, and security audits.
- **Operational Considerations**:
  - Automated tests and local development workflows verifying loopback services must explicitly configure test-specific allowances (e.g. `AllowInsecureEgress = true`) rather than relying on production defaults.
