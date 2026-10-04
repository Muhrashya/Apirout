# Apirout Architecture

Apirout is an open-source, high-performance Instant API Engine and Gateway inspired by the architectural principles of **Kong Gateway** and **Hasura GraphQL Engine**.

```
                           +------------------------------------------+
                           |          Web Console UI (:8002)          |
                           |       (React + Vite Admin Portal)        |
                           |     - Standard HTTPS (No VPN Req)        |
                           +--------------------+---------------------+
                                                |
                                                | HTTP :8002 -> :8001
                                                v
             +--------------------------------------------------------------+
             |                 ADMIN MANAGEMENT API (:8001)                 |
             |  - Management REST APIs (/api/v1/endpoints, /connections)    |
             |  - Database Schema Introspection (Postgres & MySQL)         |
             |  - Declarative GitOps Metadata Engine (YAML/JSON)           |
             |  - RBAC (SUPER_ADMIN, ADMIN, MEMBER) + Admin Secret Auth     |
             |  - Optional Admin IP Restriction (APIROUT_ADMIN_ALLOWED_IPS)|
             +--------------------+---------------------+------------------+
                                  |                     |
                     Postgres DB  |                     | Redis Pub/Sub
                     (Registry)   |                     | (Invalidation)
                                  v                     v
+-----------------------------------------------------------------------------------+
|                                DATA PLANE (DP)                                    |
|                                                                                   |
|  [Port :8000] Public Consumer Gateway Proxy:                                      |
|    - High-Throughput HTTP Engine (Go goroutines, zero reflection)                 |
|    - Dynamic Host-Based Domain Routing (api.corp.com, mitra.corp.com)             |
|    - Client IP Restriction & CIDR Allowlisting (10.0.0.0/8, 192.168.1.0/24)      |
|    - Hasura-Style Instant CRUD Engine (/api/v1/instant/:conn/:table)              |
|    - In-Memory LRU Routing & Schema Cache                                         |
|                                                                                   |
|  [Port :8100] Status & Prometheus Telemetry:                                      |
|    - Prometheus Metrics Scraper (/metrics)                                        |
|    - Kubernetes Liveness & Readiness Probes (/health)                             |
+----------------------------------------+------------------------------------------+
                                         |
                       Public Internet   | HTTP :8000
                                         v
                            [ External API Consumers ]
```

---

## 1. Architectural Paradigms & 4-Port Dedicated Listeners

Inspired directly by **Kong Gateway**, Apirout decouples listening ports to isolate public consumer traffic from admin operations and telemetry:

| Port | Service | Purpose | Access Control |
|---|---|---|---|
| `:8000` | **Gateway Proxy** | Public API consumer traffic (`/api/*`, Instant CRUD) | Host domain routing + Client IP CIDR allowlist + API Client JWT |
| `:8001` | **Admin API** | REST API for system management, schemas, GitOps | Master Admin Secret or JWT (SUPER_ADMIN) + IP allowlist |
| `:8002` | **Web Console GUI** | Modern React + Vite portal for administration | Web authentication (bcrypt + JWT + sessions) — **No VPN required** |
| `:8100` | **Status & Metrics**| Health checks (`/health`) and Prometheus metrics (`/metrics`)| Prometheus scraper, Kubernetes liveness/readiness probes |

### Operating Modes

Configured via `APIROUT_ROLE`:

| Mode | `APIROUT_ROLE` | Description | Typical Environment |
|---|---|---|---|
| **Traditional (Default)** | `traditional` | Single binary running Control Plane (Admin API + Web GUI) and Data Plane (Gateway + Status) on their dedicated ports. | Local development, single-server deployment. |
| **Control Plane** | `control_plane` | Runs Admin API (`:8001`) and Web Console GUI (`:8002`). Handles DB migrations, metadata exports, and schema introspection. | Dedicated management cluster / internal node. |
| **Data Plane** | `data_plane` | Ultra-fast, stateless API gateway (`:8000`) and Status/Metrics (`:8100`). Fetches metadata at boot and listens for Redis invalidation events. | Public DMZ, Kubernetes autoscaling cluster (HPA). |

### Edge Reverse Proxy & Production Ingress
In production environments, the four listener ports are shielded by an **Nginx Edge Reverse Proxy** (standalone VPS / Docker) or a **Kubernetes Ingress Controller with NetworkPolicies**:
- Public traffic enters strictly on standard ports **80** (HTTP) and **443** (HTTPS).
- The edge proxy maps domain routes to `:8000` (API Gateway) and `:8002` (Web Console), while enforcing IP allowlists on `:8001` (Admin API) and completely isolating `:8100` (Metrics) from external callers.
- Full configuration files, compose stacks, and manifests are documented in the [Deployment Guide](../deploy/README.md).

---

## 2. Hasura Engine Paradigm (Instant Database APIs)

Like Hasura GraphQL Engine, Apirout eliminates boilerplate backend code:
1. **Colok Database**: Connect PostgreSQL or MySQL via `Settings > Connections`.
2. **Instant Introspection**: Apirout queries `information_schema` to detect tables, columns, data types, primary keys, and relationships.
3. **Instant REST APIs**:
   - `GET /api/v1/instant/:connId/:table`: Filter (`?status=eq.active`), order (`?order=id.desc`), paginate (`?limit=20&offset=0`), select columns (`?select=id,name`).
   - `GET /api/v1/instant/:connId/:table/:id`: Single row lookup by primary key.
   - `POST /api/v1/instant/:connId/:table`: Insert row(s).
   - `PUT /api/v1/instant/:connId/:table/:id`: Update row.
   - `DELETE /api/v1/instant/:connId/:table/:id`: Delete row.

---

## 3. Traffic Controller & Security Isolation

### Host-Based Multi-Domain Routing
Endpoints can be bound to specific hostnames or subdomains:
- `api.company.com` -> General public APIs.
- `mitra.company.com` -> Partner & B2B integration APIs.
- Configured per endpoint via `allowed_domains` (e.g. `api.company.com,mitra.company.com` or `*` for wildcard).

### Client IP Restriction (CIDR Allowlisting)
Endpoints can restrict incoming traffic to allowed CIDR blocks or specific IPs:
- `192.168.1.0/24` -> Local subnet only.
- `10.0.0.0/8, 203.0.113.15` -> Corporate network and fixed partner IP.
- Configured per endpoint via `allowed_ips` or per domain via `default_allowed_ips`.
- Real client IP is reliably extracted from `X-Forwarded-For` (first IP) and `X-Real-IP`.

### Kubernetes Ingress & Strict Egress NetworkPolicy
- **Ingress**: Separate Ingress manifests for Web GUI (`:8002`), Gateway (`:8000`), and Admin API (`:8001`).
- **Egress**: The Data Plane pod is strictly firewalled to egress only to target databases (`:5432`, `:3306`, `:1433`), Redis (`:6379`), and DNS (`:53`). It cannot connect to internal Control Plane stores or other unauthorized services.
