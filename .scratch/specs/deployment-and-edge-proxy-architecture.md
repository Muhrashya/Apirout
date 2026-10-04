# Spec: Production Deployment Architecture, Unified Deploy Hierarchy, and Nginx Edge Reverse Proxy

**Triage Label:** `ready-for-agent`

## Problem Statement

When deploying Apirout in production outside of local development:
1. The repository exposes four disparate HTTP listener ports directly (`8000` Gateway Proxy, `8001` Admin Management API, `8002` Web Console GUI, `8100` Status and Metrics). Exposing these raw ports directly on public networks creates security vulnerabilities by leaving internal administrative APIs and operational metrics unshielded and unencrypted without TLS termination on standard ports 80 and 443.
2. The `deploy/k8s` directory contains critical multi-plane Kubernetes deployment blueprints, but lacks documentation, validation tooling, and unified alignment with single-node Docker Compose deployments, leading to uncertainty over whether it should be deleted or maintained.
3. Operators deploying on standalone virtual machines or cloud instances lack a standardized, production-hardened reverse proxy configuration to handle domain routing (`api.domain.com` vs. `console.domain.com`), SSL/TLS termination, HTTP/2 buffering, and client IP forwarding required by gateway IP allowlists and audit logs.

## Solution

Consolidate all infrastructure and deployment configurations into a unified `deploy/` hierarchy, maintain and document the multi-plane Kubernetes manifests, and provide a production-ready Nginx edge reverse proxy:
1. Retain and formalize `deploy/k8s/` as the official multi-plane Kubernetes production blueprint, accompanied by clear deployment documentation.
2. Introduce a production-grade Nginx edge reverse proxy configuration in `deploy/nginx/` that serves as the single public entrypoint on ports 80 and 443, routing traffic dynamically based on domain hostnames, terminating SSL/TLS, and strictly isolating sensitive admin and telemetry ports.
3. Provide a production Docker Compose recipe that runs the edge proxy alongside the gateway, keeping internal listeners within an isolated container network.
4. Preserve and forward real client IPs (`X-Forwarded-For`, `X-Real-IP`) to ensure Apirout's IP restriction filters and audit log latency tracking function with 100% accuracy behind proxies.

## User Stories

1. As an infrastructure operator, I want a single public entrypoint listening on ports 80 and 443, so that external consumers and administrators do not need to connect to non-standard ports (8000, 8001, 8002, 8100).
2. As a DevOps engineer, I want the Nginx edge proxy to route traffic based on hostnames (`api.example.com` to the Gateway Proxy and `console.example.com` to the Web Console GUI), so that our services can be hosted cleanly under dedicated subdomains.
3. As a security administrator, I want the Admin Management API and Prometheus Metrics endpoints to be shielded from unauthenticated public internet traffic, so that internal administrative surfaces cannot be scanned or attacked.
4. As an API developer, I want client real IP addresses to be preserved via standard proxy headers, so that gateway IP restriction policies and audit logs capture authentic client origins.
5. As a platform maintainer, I want the `deploy/` directory to follow a clean, predictable structure, so that all Kubernetes, Nginx, and production compose configurations are organized in a single location.
6. As a Kubernetes cluster operator, I want documented multi-plane Kubernetes manifests with separate Control-Plane and Data-Plane deployments, so that I can scale stateless gateways independently from the management control plane.
7. As a system administrator, I want the edge proxy to support WebSocket upgrades and high-throughput streaming buffers, so that real-time gateway features operate without connection drops.
8. As an open-source contributor, I want a comprehensive deployment guide explaining how to deploy on local Docker Compose, standalone production VPS, and Kubernetes, so that onboarding to production environments is friction-free.
9. As a site reliability engineer, I want automated configuration linting and dry-run validation for Nginx and Kubernetes manifests, so that syntax mistakes can never reach production environments.
10. As a DevOps engineer, I want the production compose recipe to keep internal listener ports unexposed on host network interfaces, so that only the reverse proxy can communicate with the backend services.

## Implementation Decisions

### 1. Directory Structure Consolidation
- Consolidate all infrastructure assets under `deploy/`:
  - A root deployment guide documenting architecture, security baselines, and execution steps.
  - A dedicated subdirectory for the multi-plane Kubernetes manifests (Control-Plane, Data-Plane, ConfigMaps, Secrets, NetworkPolicies).
  - A dedicated subdirectory for the Nginx edge proxy containing base server configurations, upstream proxy rules, and security headers.
  - A dedicated production Docker Compose specification integrating Nginx with the gateway stack.

### 2. Edge Proxy Routing & Security Topology
- The edge proxy will bind exclusively to ports 80 (HTTP) and 443 (HTTPS).
- HTTP traffic will automatically redirect to HTTPS.
- Dedicated virtual hosts will route:
  - Consumer API Traffic $\rightarrow$ forwards to the Data Plane Proxy listener with HTTP/2 support, 10MB client body limit, and extended proxy timeouts.
  - Web Console GUI Traffic $\rightarrow$ forwards to the Web Console listener with static asset caching and WebSocket proxy support.
  - Admin Management API Traffic $\rightarrow$ forwards to the Control Plane Admin listener, protected by Nginx-level CIDR IP allowlisting.
  - Status & Metrics Traffic $\rightarrow$ restricted to internal monitoring networks only; external access returns 403 Forbidden.

### 3. Client IP and Header Propagation
- The edge proxy will set standard forwarding headers:
  - `X-Forwarded-For` appending client IP to existing proxy chains.
  - `X-Real-IP` capturing the immediate downstream client socket address.
  - `X-Forwarded-Proto` passing the external connection protocol (http or https).
  - `X-Forwarded-Host` passing the original Host header.
- Apirout runtime will continue extracting client IPs from these headers for domain matching and audit logs.

### 4. Kubernetes Manifest Preservation & Enhancement
- Retain the existing 6 Kubernetes manifests in `deploy/k8s/`.
- Ensure container image references default to standard versioned tags (`v1.0.0`).
- Document step-by-step application instructions using standard Kubernetes tooling.

## Testing Decisions

### Test Seams
1. **Edge Proxy Syntax Seam (Highest Seam)**: Execute Nginx configuration test (`nginx -t`) inside the official Nginx container image to verify configuration correctness, upstream definitions, and syntax integrity without errors.
2. **Container Network Isolation Seam**: In the production Docker Compose profile, verify that internal ports (8000, 8001, 8002, 8100) are not bound to host interfaces, and only ports 80 and 443 are exposed externally.
3. **HTTP Header Propagation Seam**: Send requests through the edge proxy and verify that the backend receives `X-Forwarded-For`, `X-Forwarded-Proto`, and `X-Real-IP` headers intact.
4. **Kubernetes Manifest Validation Seam**: Run client-side dry-run validation (`kubectl apply --dry-run=client`) or YAML schema validation on all manifests in `deploy/k8s/` to guarantee schema compliance.

### Prior Art
- Existing Docker multi-stage builds and Docker Compose service configurations test container readiness. The edge proxy tests will follow similar containerized validation recipes.

## Out of Scope

- Automated DNS provisioning or integration with cloud DNS APIs (Route53, Cloudflare API).
- Cloud-specific Ingress controllers (AWS ALB Controller, GKE Ingress) beyond standard Nginx and K8s Ingress.
- Automated generation of production SSL certificates (Let's Encrypt Certbot cron jobs).

## Further Notes

- Target GitHub repository is `Muhrashya/Apirout`. All tickets will be created on the repository tracker with `ready-for-agent` labels.
