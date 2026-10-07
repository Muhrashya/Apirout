# ADR-0006: Consumer Route Authorization and Inbound Auth Plugin Harmonization

## Status
Accepted

## Context
During a forensic review of route access controls and authentication plugins, two critical issues were identified:
1. **Protected Route SQL NULL Lockout**: When an endpoint was protected (`is_public = false`) and an authenticated consumer called it, the authorization check in `reader.IsAPIClientAllowedForEndpoint` evaluated `c.group_id = e.group_id`. Because SQL treats `NULL = NULL` as `FALSE`, any consumer without an explicit group calling a route without a group was unconditionally rejected with `HTTP 403 Forbidden: Access to this endpoint is forbidden for consumer`, even when the consumer had valid API key credentials.
2. **Phantom `oauth2` Plugin in UI**: The route creation form offered an inbound `OAuth 2.0 (oauth2)` authentication plugin option. However, the Gateway Data Plane engine (`auth.Engine`) only registers `key-auth`, `basic-auth`, `jwt-auth`, and `hmac-auth`. Selecting `oauth2` resulted in unresolvable `401 Authentication Required` errors because no inbound OAuth2 evaluator existed in the Go engine.

## Decision
1. **Default-Allow for Unrestricted Consumers on General Routes**:
   - Refactor `IsAPIClientAllowedForEndpoint(apiClientID, endpointID)` to evaluate:
     - If the consumer has specific entries in `api_client_endpoints`, enforce the whitelist (`count > 0`).
     - If the consumer has no specific route whitelist (`totalRestrictions == 0`):
       - If the route has no group restriction (`e.group_id IS NULL`), allow access (`TRUE`).
       - If the route is restricted to a group (`e.group_id IS NOT NULL`), require the consumer to belong to that exact group (`c.group_id = e.group_id`).
   - In `handler.go`, only invoke consumer authorization if a valid `consumerID` is resolved.

2. **Remove Phantom `oauth2` Option from Route Form**:
   - Remove `{ id: 'oauth2' }` from `RouteFormPage.tsx`, displaying only the authentic, fully implemented inbound plugins:
     - `key-auth`: API Key via header or query parameter
     - `jwt-auth`: Cryptographic HMAC-SHA256 JWT validation
     - `basic-auth`: Standard HTTP Basic Auth against consumer credentials
     - `hmac-auth`: HMAC request signing with clock skew replay protection
     - `none`: Legacy gateway fallback chain

## Consequences
- **Positive**:
  - Protected routes now work immediately upon consumer creation with valid credentials without requiring tedious manual route-by-route assignment.
  - Consumer groups function correctly as security tiers (routes assigned to a group only accept consumers in that group).
  - Eliminates misleading UI options that lead to runtime authentication failures.
