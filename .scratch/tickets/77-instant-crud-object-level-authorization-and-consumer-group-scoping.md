# 77: Instant CRUD Object-Level Authorization & Consumer Group Scoping

**What to build:** Object-level authorization (BOLA defense) for the Instant Database REST engine (`/api/v1/instant/{connId}/*`). When an external request arrives with consumer credentials (API Key or JWT), the gateway verifies that the authenticated consumer possesses rights to access the specified connection (`connId`), based on consumer group permissions or explicit shared access. Requests targeting unauthorized connections are rejected with HTTP 403 Forbidden and emitted as security audit events.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] Instant CRUD request handler verifies that the authenticated consumer or API client has access to the target database connection identifier
- [x] Unauthorized requests across tenant or connection boundaries return HTTP 403 Forbidden with a normalized error envelope
- [x] Unauthorized access attempts log a security audit event with caller identity and target connection
- [x] Unit and component tests verify that Consumer A cannot access Consumer B's database connection
