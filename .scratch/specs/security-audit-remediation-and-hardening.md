# Security Audit Remediation & Defense Hardening

**Status:** completed

## Problem Statement

When deploying the API Gateway in production environments or exposing it to external traffic, operators and security engineers face significant risks across perimeter identity, service availability, and container isolation:
1. External clients can forge forwarded proxy headers (`X-Forwarded-For`, `X-Real-IP`, `CF-Connecting-IP`), successfully bypassing route-level IP allowlists, Admin API CIDR shields, and rate limiters.
2. Attackers can stream gigabyte-sized HTTP request payloads into unauthenticated or administrative endpoints, exhausting server RAM and crashing the container via Out-Of-Memory termination.
3. SSH bastion tunnels configured for remote database connections blindly accept arbitrary remote server keys without verification, leaving database credentials vulnerable to Man-in-the-Middle eavesdropping on untrusted networks.
4. Administrative endpoints return encrypted credential ciphertexts to client web browsers, violating the principle of least privilege and minimal data exposure.
5. Control plane HTTP servers omit read and write timeouts, exposing them to slow-write connection starvation attacks.
6. Production container images run with root privileges (UID 0), increasing the blast radius of any theoretical container escape.

---

## Solution

A multi-layered defense-in-depth security remediation that hardens the ingress perimeter, bounds resource consumption, and enforces strict operational least privilege:
1. **Trusted Proxy Ingress Seam**: Client IP resolution is gated behind an explicit trusted proxy policy. Forwarded proxy headers are traversed from right to left only when the direct network socket originates from an authorized proxy CIDR; untrusted connections strictly fall back to direct socket addresses.
2. **Payload Shield Ingress Middleware**: A centralized ingress middleware enforces strict byte bounds on incoming request bodies, terminating oversized streams with a standardized `413 Payload Too Large` error envelope before memory exhaustion occurs.
3. **Environment-Aware Bastion Host Key Verification**: The database tunneling adapter mandates valid host key fingerprints in production while logging explicit security warnings in development.
4. **Outbound Credential Ciphertext Redaction**: Control plane endpoints redact encrypted credential ciphertexts into boolean status markers before serializing JSON to the web console.
5. **Harmonized HTTP Server Timeouts**: Control plane administration and console listeners enforce strict read and write timeouts, eliminating Slowloris connection pool exhaustion.
6. **Non-Root Container Hardening & Repository Hygiene**: The production container executes under an unprivileged user (UID 10001), while repository ignore manifests strictly filter repository metadata and private key files.

---

## User Stories

1. As an API Gateway Operator, I want the gateway to ignore client-supplied `X-Forwarded-For` headers from untrusted connections, so that external callers cannot spoof their IP address to bypass route IP allowlists.
2. As a Security Engineer, I want route-level rate limits to be calculated using the authentic client IP rather than attacker-controlled headers, so that distributed brute-force attacks cannot evade rate limiting by rotating fake headers.
3. As a Platform Administrator, I want Admin API CIDR allowlists to strictly evaluate verified source IPs, so that internet-based adversaries cannot access console management APIs with a forged internal header.
4. As a Systems Operator, I want to configure trusted proxy CIDRs via environment variables, so that our cloud load balancers can safely forward legitimate client IPs in multi-tier topologies.
5. As a Developer, I want local development environments to default to permissive private subnet proxy evaluation, so that local Docker compose and curl workflows function without manual CIDR configuration.
6. As an API Consumer, I want requests exceeding maximum permitted payload sizes to be rejected immediately with an HTTP 413 error envelope and correlation ID, so that I receive deterministic feedback without hanging connections.
7. As a Systems Operator, I want the gateway to reject incoming request streams exceeding 10MB on data plane routes and 2MB on control plane routes, so that memory exhaustion DoS attacks cannot crash gateway pods.
8. As a Security Engineer, I want SSH bastion database connections in production to strictly verify remote host key fingerprints, so that man-in-the-middle attackers cannot intercept target database credentials.
9. As a Developer, I want SSH bastion tunnels in development mode to allow omitting host keys with an explicit audit warning, so that local prototyping with ephemeral containers remains frictionless.
10. As a Web Console User, I want the outbound authentication interface to indicate whether credentials have been saved without transmitting raw ciphertexts to my browser, so that sensitive ciphertext strings are never exposed over client network links.
11. As a Security Auditor, I want control plane administration listeners to enforce explicit read and write timeouts, so that slow-reading or slow-writing clients cannot hold connection sockets open indefinitely.
12. As a Kubernetes Cluster Administrator, I want the gateway container image to execute as an unprivileged non-root user (UID 10001), so that the application adheres to CIS Docker benchmarks and enforces least privilege.
13. As a Platform Maintainer, I want git repository ignore manifests to block `.git/` from container build contexts and prevent accidental commits of private keys or certificates, so that cryptographic secrets are never exposed in build artifacts.

---

## Implementation Decisions

### 1. Ingress IP Resolution Seam
- **Module**: Ingress networking and IP utilities module.
- **Seam**: Client IP resolution interface.
- **Decision**: Introduce a `TrustedProxyChecker` interface that accepts an incoming request and evaluates the direct socket remote address.
- **Algorithm**:
  - Check whether `RemoteAddr` matches any CIDR or IP in the configured `TRUSTED_PROXIES` list.
  - If `RemoteAddr` is NOT trusted, immediately return the socket IP, completely ignoring all forwarded headers (`X-Forwarded-For`, `X-Real-IP`, `CF-Connecting-IP`, `Fastly-Client-IP`, `True-Client-IP`).
  - If `RemoteAddr` IS trusted, traverse `X-Forwarded-For` from right to left, discarding any IPs that belong to trusted proxies until the first untrusted IP is encountered.
  - If all forwarded IPs are trusted proxies, or if `X-Forwarded-For` is empty, fall back to `X-Real-IP` (if trusted) or `RemoteAddr`.
- **Environment Policy**:
  - `APP_ENV=production`: `TRUSTED_PROXIES` defaults to empty string (strict socket-only evaluation unless explicitly configured).
  - `APP_ENV=development`: `TRUSTED_PROXIES` defaults to standard loopback and private subnets (`127.0.0.1/32,::1/128,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16`).

### 2. Ingress Payload Shield Middleware
- **Module**: Gateway and console middleware modules.
- **Seam**: HTTP request ingress pipeline.
- **Decision**: Wrap the incoming request body with a bounded stream reader (`http.MaxBytesReader`) before routing or decoding occurs.
- **Error Contract**:
  - Add canonical error code `ErrCodePayloadTooLarge = "PAYLOAD_TOO_LARGE"` to the normalized error envelope specification (ADR-0003).
  - When a stream exceeds the threshold, `MaxBytesError` is caught and translated to HTTP 413:
    ```json
    {
      "status": false,
      "message": "Request entity exceeds maximum permitted size",
      "code": "PAYLOAD_TOO_LARGE",
      "request_id": "req-...",
      "timestamp": "..."
    }
    ```
- **Thresholds**:
  - Data Plane listener: 10MB default (`MAX_REQUEST_BODY_MB`).
  - Control Plane listener: 2MB default.

### 3. Outbound Credential Masking at Serialization Seam
- **Module**: Control plane service management module.
- **Seam**: Outbound authentication REST serializer.
- **Decision**: When serializing service outbound authentication configs (`GET /services/{id}/outbound-auth`), parse the stored encrypted JSON configuration and replace any encrypted fields (`token_encrypted`, `secret_key_encrypted`, `password_encrypted`, `client_secret_encrypted`) with boolean markers (e.g. `{ "token_configured": true }`). The browser UI continues to display masked placeholders while never receiving actual ciphertext.

### 4. Environment-Aware Bastion Host Key Verification
- **Module**: Core connection tunneling adapter.
- **Seam**: SSH client configuration builder.
- **Decision**:
  - If a public host key is provided in the connection configuration, verify it strictly via `ssh.FixedHostKey`.
  - If a host key is omitted:
    - In `APP_ENV=production`: Terminate connection establishment immediately with a typed error requiring host key verification.
    - In `APP_ENV=development`: Permit connection via `ssh.InsecureIgnoreHostKey` while logging an explicit security warning.

### 5. HTTP Listener Timeouts Harmonization
- **Module**: Control plane application coordinator.
- **Seam**: HTTP server listener lifecycle.
- **Decision**: Configure `ReadTimeout: 15s` and `WriteTimeout: 30s` on `adminSrv` and `guiSrv` listeners, matching the resilience of the data plane proxy server.

### 6. Container Hardening & Build Manifest Hygiene
- **Module**: Container build recipe and VCS ignore manifests.
- **Decision**:
  - In `Dockerfile`, provision an unprivileged system user and group (`apirout:apirout`, UID/GID 10001), set appropriate permissions on application assets, and declare `USER apirout`.
  - In `.dockerignore` and `.gitignore`, enforce exclusion of `.git/`, `*.key`, `*.crt`, `*.pem`, and `*.cert`.

---

## Testing Decisions

### What Makes a Good Test
Tests must verify external behavioral contracts and security invariants rather than implementation internals:
- A client sending a spoofed `X-Forwarded-For` from an untrusted remote address must be classified using their actual remote address.
- A client sending a request larger than the configured byte limit must receive an HTTP 413 status code with a structured error envelope.
- An SSH bastion tunnel instantiated without a host key in production mode must fail to connect.
- An outbound auth API query must never contain encrypted ciphertext strings in its JSON output.
- The built container must execute under UID 10001.

### Tested Modules
- **Ingress IP Checker**: Unit tests in core networking testing right-to-left traversal, untrusted socket rejection, and multi-proxy chains.
- **Payload Shield Middleware**: Unit tests in gateway middleware testing boundary byte limits, stream truncation, and error envelope formatting.
- **Bastion Tunnel Adapter**: Unit tests verifying host key validation and production error tripping.
- **Outbound Auth Serializer**: API tests verifying field redaction.
- **Integration Test Suite**: All 19 existing integration test suites must pass with zero regressions.

### Prior Art
- Unit tests in `internal/core/iputil/ip_checker_test.go`.
- Middleware tests in `internal/gateway/responder/middleware_test.go`.
- Existing integration suites under `tests/e2e/`.

---

## Out of Scope
- Rewriting historical Git commit SHAs across shared remotes (addressed via key revocation and local hygiene; force-rewriting upstream git branches requires explicit coordination by the repository owner).
- Mutual TLS (mTLS) client certificate authentication for the management console (scheduled for enterprise roadmap).
- Distributed Redis-backed token bucket rate limiting (in-memory rate limiting remains the standard node-local mechanism).

---

## Further Notes
- This specification implements the decisions recorded in **ADR-0009** (`docs/adr/0009-trusted-proxy-ingress-and-security-shield-architecture.md`).
- Domain terminology conforms strictly to `GLOSSARY.md` definitions (`Trusted Proxy Ingress`, `Payload Shield`, `Bastion Host Key Policy`).
