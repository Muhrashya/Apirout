# 59: Ingress Payload Shield Middleware & Canonical HTTP 413 Seam

**What to build:**  
A centralized ingress middleware bounds incoming request body streams (10MB default on the Data Plane, 2MB on the Control Plane). Streams exceeding limits are cleanly halted, and the gateway responder emits a standardized HTTP 413 error envelope with canonical code `PAYLOAD_TOO_LARGE` and correlation tracking (`X-Request-ID`), eliminating memory exhaustion DoS from unbounded payloads.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] Add `ErrCodePayloadTooLarge = "PAYLOAD_TOO_LARGE"` to `internal/gateway/responder/codes.go`.
- [x] Create `PayloadShield` middleware in `internal/gateway/middleware/payload_shield.go` and `internal/console/middleware/payload_shield.go` wrapping `r.Body = http.MaxBytesReader(w, r.Body, maxBytes)`.
- [x] Implement error interception so that `*http.MaxBytesError` writes canonical HTTP 413 error envelope with telemetry headers.
- [x] Wire `PayloadShield` middleware into the Data Plane (`gateway/app/app.go`) with `MAX_REQUEST_BODY_MB` (default 10MB) and Control Plane (`console/app/app.go`) with 2MB.
- [x] Unit tests verifying that payloads within limit pass through while payloads exceeding limit are rejected with HTTP 413 and `PAYLOAD_TOO_LARGE` code.
