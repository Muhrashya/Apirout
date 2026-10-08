# Specification: Real-Time Route Invalidation and Zero-Downtime Hot Reload

**Status:** `completed`

## Problem Statement

When operators manage gateway routes through the Web Console or the Control Plane REST API (such as toggling an endpoint between Public and Protected, modifying path patterns, editing allowed IP addresses, or altering rate limiting rules), changes are successfully written to the metadata database, but are **never signaled to the live Data Plane**:

1. **In-Memory Routing Desynchronization:** To achieve sub-millisecond request resolution, the Data Plane evaluates traffic against an in-memory Radix tree compiled at startup. Because the Control Plane's interactive route management handler does not broadcast change notifications, the Data Plane never reloads this tree at runtime.
2. **False Operational Confidence & Confusion:** Operators who mark a route as "Public" observe the UI badge update to "Public", yet incoming edge requests continue to fail with `HTTP 401 Unauthorized: Authentication required (provide X-API-Key or Authorization Bearer token)`.
3. **Unnecessary Service Disruptions:** To apply routing changes, operators are forced to perform manual container restarts, defeating the purpose of a dynamic API gateway control plane and causing unnecessary connection disruptions.

## Solution

Complete the Control Plane $\leftrightarrow$ Data Plane invalidation pipeline by wiring the configuration event publisher into the route management handler:

1. **Event Publication on Route Mutation:** Upon every successful route creation, modification, or deletion, the Control Plane immediately broadcasts an invalidation event across the distributed Redis Pub/Sub channel.
2. **Real-Time Shadow Tree Swap:** Upon receiving the invalidation signal, the Data Plane queries active route configurations, compiles a new in-memory routing tree in the background, and atomically swaps the router pointer in under 20 milliseconds with zero dropped requests.
3. **Automated End-to-End Dynamic Verification:** Validate live propagation within the E2E test suite by verifying that changing a route from protected to public permits anonymous traffic on the very next request without gateway restarts.

## User Stories

1. As an API platform operator, I want toggling a route from Protected to Public in the Web Console to take effect immediately, so that clients can make unauthenticated calls without waiting for container restarts.
2. As an API developer, I want newly created routes published via the Control Plane API to resolve on the Data Plane within milliseconds, so that I can immediately test new integrations.
3. As a DevOps engineer, I want route path, method, and domain modifications to propagate automatically across all Data Plane workers, so that multi-instance deployments remain synchronized without rolling restarts.
4. As a security engineer, I want disabling or deleting a route in the Control Plane to immediately cause the Data Plane to reject traffic with `404 Not Found`, ensuring immediate revocation of access.
5. As an API consumer, I want route configuration changes to be applied via atomic pointer swaps without dropped connections or intermittent `500 Internal Server Error` responses during high-throughput traffic.
6. As a developer running local tests, I want the system to handle environments where Redis is in-memory or transient gracefully, falling back to local reload mechanisms without crashing the Control Plane.
7. As a QA automation engineer, I want the end-to-end test suite to verify live dynamic propagation from the Control Plane to the Data Plane, ensuring no regressions occur in cache invalidation.

## Implementation Decisions

- **EventBus Publisher Binding:**
  - Route management handlers in the Control Plane receive the distributed configuration publisher dependency during dependency injection.
  - Successful entity mutations (creation, update, deletion) trigger synchronous event emission to the configuration channel.
  - Non-blocking error handling: If the event broadcast encounters an error (e.g. transient network issue), the incident is logged with warning telemetry, allowing the metadata persistence to complete.
- **Atomic Router Reload Execution:**
  - The Data Plane invalidation subscriber invokes the thread-safe router compilation routine upon receiving route invalidation messages.
  - Active routes are fetched from the registry database, mapped into a candidate routing tree, and atomically swapped into the execution context using atomic pointer operations.
  - In-flight requests traversing the prior routing tree complete uninterrupted.

## Testing Decisions

- **Testing Seam: High-Seam End-to-End Live Propagation Test (`tests/e2e/`):**
  - *Phase 1 (Protected State):* Create a route with `is_public = false`. Call the Data Plane without credentials $\to$ assert `401 Unauthorized`.
  - *Phase 2 (Dynamic Public Toggle):* Update the route via Control Plane API (`PUT /api/v1/routes/:id`) with `is_public = true`.
  - *Phase 3 (Immediate Verification):* Immediately call the Data Plane without credentials $\to$ assert the request succeeds without requiring container restarts.
  - *Phase 4 (Dynamic Deletion):* Delete the route via Control Plane API $\to$ immediately assert the Data Plane responds with `404 Not Found`.

## Out of Scope

- Changing the underlying Radix tree search algorithm or data structures.
- Altering external client authentication credential formats (API keys, JWT tokens, Basic auth).
- Redesigning the Web Console user interface layout.

## Further Notes

This work realizes the design goals of ADR-0001 (Hybrid Gateway Architecture) and Ticket 31 (Zero-Downtime EventBus Synchronization), closing the loop between interactive Control Plane operations and live Data Plane execution.
