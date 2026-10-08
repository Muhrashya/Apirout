# Ticket 55: Inbound Auth Engine Consolidation and Typed Error Seam

**Status:** completed
**Blocked by:** 54

## Overview
Deepen `internal/gateway/auth/engine.go` to absorb fallback API key and JWT authentication behind `Authenticate(r, endpoint, serviceID)`, delete the duplicate 40-line procedural fallback loop in `internal/gateway/executor/handler.go`, and introduce typed `*auth.Error` with canonical `responder.ErrorCode`.

## Acceptance Criteria
- [x] Define `type Error struct { StatusCode int; Code responder.ErrorCode; Message string }` in `internal/gateway/auth/engine.go` implementing the `error` interface.
- [x] Implement fallback API key and JWT token verification inside `auth.Engine.Authenticate()` when an endpoint requires authentication and has no attached auth plugins or no plugin matched.
- [x] Refactor `internal/gateway/executor/handler.go` to call `authResult, err := h.authEngine.Authenticate(r, endpoint, serviceID)` directly, deleting lines 567-608.
- [x] If `err` is returned, write standard error envelope via `responder.Error` using the typed `auth.Error`.
- [x] Unit tests for `auth.Engine` testing both plugin OR resolution and fallback verification.
- [x] Verify existing gateway auth tests and E2E suites pass.
