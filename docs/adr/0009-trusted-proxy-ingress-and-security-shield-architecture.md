# ADR-0009: Trusted Proxy Ingress, Payload Shield, and Container Hardening Architecture

## Status
Accepted

## Context
A comprehensive security audit of Apirout identified critical vectors around ingress perimeter defense, client identity verification, and container execution privileges:
1. **Client IP Spoofing (CWE-290 / CWE-345)**: `httputil.GetClientIP` and `iputil.ExtractClientIP` unconditionally read `X-Forwarded-For`, `X-Real-IP`, and `CF-Connecting-IP` without verifying whether the direct network connection (`RemoteAddr`) originates from an authorized proxy. Untrusted external clients could spoof headers to bypass route-level IP allowlists, Admin API CIDR restrictions, and rate limiters.
2. **Memory Exhaustion Denial of Service (CWE-400 / CWE-770)**: Incoming HTTP request bodies across all endpoints were parsed with `json.NewDecoder(r.Body)` without an upper size bound (`http.MaxBytesReader`), enabling memory exhaustion from unbounded payload streams.
3. **Bastion SSH Host Key Blind Trust (CWE-295)**: SSH bastion tunnels for target databases fell back to `ssh.InsecureIgnoreHostKey()` when `HostKey` was omitted, introducing Man-in-the-Middle risk on untrusted networks.
4. **Outbound Credential Ciphertext Exposure (OWASP API3:2023)**: `GET /services/{id}/outbound-auth` returned encrypted credential ciphertexts directly to client browsers, violating minimal exposure principles.
5. **Container Security & Git History Hygiene**: The production container executed as root (UID 0), and a historical 2048-bit RSA key was committed to git history in an earlier commit (`6060ede`).

## Decision

1. **Trusted Proxy Ingress Seam**:
   - Introduce a centralized `TRUSTED_PROXIES` configuration (comma-separated CIDRs/IPs).
   - Resolve client IP by evaluating `RemoteAddr`. Only if `RemoteAddr` belongs to a configured trusted proxy will proxy headers (`X-Forwarded-For`) be traversed (evaluating from right to left to locate the first untrusted client IP).
   - If `RemoteAddr` is not a trusted proxy, proxy headers are discarded, preventing spoofing.
   - In `APP_ENV=production`, `TRUSTED_PROXIES` defaults to strict empty/none (`""`) unless explicitly configured by infrastructure operators. In `APP_ENV=development`, default to local/private CIDRs (`127.0.0.1/32,::1/128,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16`).

2. **Ingress Payload Shield Middleware**:
   - Introduce a unified `PayloadShield` middleware wrapping `r.Body` with `http.MaxBytesReader`.
   - Data Plane default limit: 10MB (`MAX_REQUEST_BODY_MB`).
   - Control Plane default limit: 2MB.
   - Intercept `*http.MaxBytesError` to emit a canonical `413 Payload Too Large` error envelope with code `PAYLOAD_TOO_LARGE` via `responder.Error`.

3. **Environment-Aware Bastion Host Key Verification**:
   - In `APP_ENV=production`, require `HostKey` to be provided when initializing an SSH bastion tunnel, failing fast if omitted.
   - In `APP_ENV=development`, allow omitting `HostKey` with an explicit runtime security warning log.

4. **Outbound Credential Redaction at Control Plane Egress**:
   - Redact all encrypted secrets (`token_encrypted`, `secret_key_encrypted`, `password_encrypted`, `client_secret_encrypted`) to boolean markers (`token_configured: true`) before returning responses to the Web Console UI.

5. **Server Timeouts Harmonization & Container Hardening**:
   - Set `ReadTimeout: 15s` and `WriteTimeout: 30s` on Control Plane `adminSrv` and `guiSrv` listeners.
   - Add non-root user `apirout` (UID 10001) in `Dockerfile` and declare `USER apirout`.
   - Add `.git/`, `*.key`, `*.crt`, `*.pem` to `.dockerignore` and `.gitignore`.

## Consequences
- **Positive**:
  - Eliminates client IP spoofing, IP allowlist bypass, and rate limit evasion.
  - Immune to memory exhaustion from oversized HTTP request streams.
  - Mitigates Man-in-the-Middle credential theft in SSH bastion database tunnels.
  - Upholds CIS Docker Benchmark 4.1 (non-root execution).
  - Guarantees zero sensitive ciphertext transmission to client browsers.
- **Trade-off**:
  - In production, operators deploying behind custom load balancers or Cloudflare must configure `TRUSTED_PROXIES` with their ingress CIDRs for `X-Forwarded-For` to be recognized.
