# Ticket 56: Evented Registry Storage Seam and Sync Invalidation

**Status:** completed
**Blocked by:** 55

## Overview
Wrap control-plane persistence operations with synchronous EventBus publication at the repository seam, guaranteeing that creates, updates, and deletes automatically notify the Data Plane without relying on individual HTTP handlers to remember to publish.

## Acceptance Criteria
- [x] Provide publisher binding directly on `internal/console/registry/Registry` (e.g. `SetPublisher(pub publisher.Publisher)` or decorator wrapper).
- [x] Automatically publish `InvalidationEvent` on `CreateEndpoint`, `UpdateEndpoint`, `DeleteEndpoint`, `CreateRouteAuthPlugin`, `DeleteRouteAuthPlugin`, `CreateService`, `UpdateService`, `DeleteService`.
- [x] Update console HTTP handlers to rely on registry automated event emission, simplifying handler logic.
- [x] Verify synchronous publication guarantees instant Data Plane route and plugin propagation.
