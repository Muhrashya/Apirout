# ADR-0005: Edge-Offloaded TLS Ingress, Domain Routing & Dynamic Route CORS

## Status
Accepted

## Context
An architectural audit of the Domain & TLS subsystems revealed critical discrepancies between the user interface and runtime behavior:
1. **TLS Reality vs UI Mockup**: The web console presented "TLS 1.3 Active - Automatic SSL Termination" badges and static inputs, yet the Go Gateway runtime operates exclusively as a plain HTTP listener (`:8000`, `:8001`, `:8002`, `:8100`) with zero native TLS termination or certificate storage engine.
2. **Disconnected Gateway Domains**: The `gateway_domains` database table and admin CRUD endpoints (`/domains`) were isolated; the Gateway Data Plane router never read or enforced registered gateway domains, checking only an ad-hoc string on individual endpoints.
3. **Inactive Per-Route CORS**: While `endpoints.cors_origins` existed in the schema, the Gateway applied only global CORS middleware, ignoring route-specific origin overrides.

## Decision
1. **Edge-Offloaded TLS Architecture (De-mocking UI)**:
   - Acknowledge and document that Apirout adheres to the standard cloud-native pattern: SSL/TLS termination is offloaded to the Edge Ingress layer (e.g., Cloudflare, Kubernetes Ingress, AWS ALB, Caddy, or Nginx).
   - Eliminate all deceptive/mockup "TLS 1.3 Active" badges in the Web Console (`Domains.tsx` and `DomainFormPage.tsx`), replacing them with truthful "Edge Offloaded Ingress" descriptors.

2. **Active Gateway Domain Validation**:
   - Connect `gateway_domains` directly to the Data Plane Gateway Reader (`reader.GetGatewayDomains()`).
   - Validate incoming request `Host` and `X-Forwarded-Host` headers against registered gateway domains. If custom domains are registered, requests targeting unregistered hostnames are rejected with HTTP 403 (`ErrCodeForbidden`).
   - Allow wildcard `*`, `localhost`, and IP addresses when only default development domains exist.

3. **Dynamic Route-Level CORS Enforcement**:
   - In `executor/handler.go`, evaluate `endpoint.CorsOrigins` per matched route.
   - If a route specifies custom allowed origins (e.g., `https://app.client.com`), dynamically inject route-specific `Access-Control-Allow-Origin`, `Access-Control-Allow-Methods`, and `Vary: Origin` response headers, properly handling both preflight `OPTIONS` and standard requests.

## Consequences
- **Positive**:
  - Eliminates false security claims and confusion for operations/SRE teams.
  - Transforms `Settings > Domains` into a functional, security-enforcing host routing engine.
  - Enables multi-frontend architectures via fine-grained per-route CORS policies.
- **Trade-off**:
  - Environments requiring TLS termination directly within the Go binary must place a lightweight reverse proxy or ingress controller in front of Apirout.
