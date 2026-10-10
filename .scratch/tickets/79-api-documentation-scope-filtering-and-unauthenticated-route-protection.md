# 79: API Documentation Scope Filtering & Route Shield

**What to build:** Move OpenAPI (`/openapi.json`) and Postman (`/postman.json`) generation endpoints behind administrative authentication so that callers must present valid JWT or master admin secret credentials. Filter the generated schema dynamically to display only the routes, upstreams, and services authorized for the caller's assigned user group, preventing unauthenticated enumeration and cross-tenant API inventory leakage.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] OpenAPI and Postman endpoints on the Control Plane require authentication (JWT or admin secret)
- [x] Unauthenticated requests to documentation endpoints return HTTP 401 Unauthorized
- [x] Schema generation generator inspects caller group scopes and filters the output specification accordingly
- [x] Unit tests verify that documentation generation rejects unauthenticated requests and correctly limits endpoint visibility
