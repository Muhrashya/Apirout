<div align="center">

<img src="docs/logo.png" alt="Apirout Gateway Logo" width="110" style="border-radius: 12px; margin-bottom: 12px;" />

# ⚡ Apirout Gateway

### Cloud-Native API Gateway, Kong-Aligned Request Router & Instant Database Engine

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg?style=flat-square)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go)](go.mod)
[![React](https://img.shields.io/badge/React-18.x-61DAFB?style=flat-square&logo=react)](web/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat-square&logo=docker)](docker-compose.yml)
[![Kong Architecture](https://img.shields.io/badge/Architecture-Kong%20Hybrid%20Aligned-003366?style=flat-square)](docs/ARCHITECTURE.md)
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen?style=flat-square)]()

<p align="center">
  <a href="#-features">Features</a> •
  <a href="#-architecture">Architecture</a> •
  <a href="#-quick-start">Quick Start</a> •
  <a href="#-kong-aligned-core-entities">Core Entities</a> •
  <a href="#-declarative-configuration">Declarative Config</a> •
  <a href="#-instant-database-apis">Instant APIs</a> •
  <a href="#-observability">Observability</a> •
  <a href="#-documentation">Docs</a>
</p>

---

</div>

## 📌 What is Apirout?

**Apirout** is a cloud-native, high-performance **API Gateway** and **Instant Data Proxy** built with **Go** and **React**. Designed around the battle-tested **Kong Gateway** architecture, Apirout enables organizations to manage, secure, route, and observe incoming microservice traffic with sub-millisecond overhead.

In addition to traditional reverse proxying, Apirout features an integrated **Instant Database Engine** (inspired by Hasura / PostgREST) that turns connected relational databases (PostgreSQL, MySQL, SQL Server) into secure, paginated REST endpoints and parameterized SQL queries without writing backend code.

---

## 🚀 Key Features

* **Kong-Aligned Entity Model**: First-class support for **Services**, **Routes**, **Upstreams**, **Targets**, and **Consumers**.
* **Zero-Downtime Dynamic Routing**: Routes, services, and upstream changes are synced across gateway nodes via Redis Pub/Sub in `< 2ms` with zero process restarts.
* **Kong Hybrid Deployment Topology**: Run as an all-in-one monolith (`traditional`), or decoupled into private `control_plane` (Admin GUI & API) and internet-facing stateless `data_plane` (Proxy).
* **Load Balancing & Active Health Checks**: Upstream target pools with weighted round-robin, consistent hashing, and background TCP/HTTP health probing with automatic circuit breaking.
* **Consumer Authentication & Security**: API Keys, Bearer JWT, Basic Auth, IP Whitelisting (CIDR), Host header validation, and Token Bucket Rate Limiting per consumer or route.
* **Instant Database to REST Engine**: Auto-introspect relational tables into instant CRUD endpoints with PostgREST-style query syntax (`?status=eq.active`, `?order=id.desc`, `?limit=25`).
* **Interactive SQL Query Workbench**: Execute and parameterize custom SQL statements (e.g. `SELECT * FROM users WHERE status = {{status}}`) and publish them directly as gateway endpoints.
* **Domains & Edge TLS Termination**: Multi-tenant hostname binding (`api.company.com`, `partner.company.com`) with edge HTTPS termination and host header matching.
* **Integrated Enterprise GUI**: Sleek, modern Kong-inspired Admin Console (React 18 + Vite) featuring dark/light themes, live traffic monitors, and latency profile visualizers.
* **Deep Observability**: Live traffic telemetry, synthetic percentile distributions (P50, P90, P99), Prometheus `/metrics` exporter, and structured JSON access logs.

---

## 🏛️ Architecture & Hybrid Topology

Apirout follows Kong's proven **Control Plane / Data Plane separation**:

```
                              [ Public Clients / Web & Mobile Traffic ]
                                                 │
                                                 ▼
                              ┌─────────────────────────────────────┐
                              │      Data Plane Gateway Proxy       │
                              │           (:8000 / HTTP)            │
                              │ ─────────────────────────────────── │
                              │  • In-Memory Radix Router (<1ms)    │
                              │  • Consumer Auth & Rate Limiter     │
                              │  • Upstream Health Check & Balancer │
                              │  • Instant Database REST Proxy      │
                              └──────────────────┬──────────────────┘
                                                 │
                   ┌─────────────────────────────┼─────────────────────────────┐
                   │                             │                             │
                   ▼                             ▼                             ▼
         ┌───────────────────┐         ┌───────────────────┐         ┌───────────────────┐
         │ Upstream Service  │         │ Upstream Service  │         │ Database Source   │
         │  (Auth Microsvc)  │         │ (Payment Service) │         │ (Postgres/MySQL)  │
         └───────────────────┘         └───────────────────┘         └───────────────────┘

 ─────────────────────────────────────────────────────────────────────────────────────────
  CONTROL PLANE (Private Network / VPC / Admin Only)
 ─────────────────────────────────────────────────────────────────────────────────────────

     [ Administrators / DevOps ]
                 │
                 ▼
    ┌─────────────────────────┐               ┌─────────────────────────────────────────┐
    │   Apirout Web Console   │ ──── API ──── │        Apirout Management API           │
    │      (:8002 / HTTP)     │               │              (:8001 / HTTP)             │
    └─────────────────────────┘               └────────────────────┬────────────────────┘
                                                                   │
                                                ┌──────────────────┴──────────────────┐
                                                ▼                                     ▼
                                    ┌───────────────────────┐             ┌───────────────────────┐
                                    │  PostgreSQL Registry  │             │   Redis Invalidation  │
                                    │  (Metadata & Config)  │             │   (Pub/Sub Event Bus) │
                                    └───────────────────────┘             └───────────────────────┘
```

---

## ⚡ Quick Start

### 1. Prerequisites
- [Docker](https://docs.docker.com/get-docker/) & [Docker Compose](https://docs.docker.com/compose/)
- (Optional for local development) [Go 1.22+](https://golang.org) & [Node.js 20+](https://nodejs.org)

### 2. Launch with Docker Compose

Clone the repository and launch the full stack in one command:

```bash
git clone https://github.com/apirout/apirout.git
cd apirout

# Launch PostgreSQL 16, Redis 7, and Apirout Gateway
docker compose up -d --build
```

### 3. Service Access Endpoints

| Port | Service | Description | Authentication |
| :--- | :--- | :--- | :--- |
| **`:8000`** | **Data Plane Proxy** | High-speed traffic ingress & API endpoints | Based on Route/Consumer |
| **`:8001`** | **Control Plane Admin API** | REST API for managing Gateway configuration | Admin Secret / JWT |
| **`:8002`** | **Admin Web Console** | Web GUI dashboard & management portal | Web User Login |
| **`:8100`** | **Status & Metrics** | Prometheus `/metrics` and health checks | Open / Internal |

Default Web Console credentials:
- **URL**: [http://localhost:8002](http://localhost:8002)
- **Email**: `admin@apirout.io`
- **Password**: `AdminApirout2026!`

---

## 🧩 Kong-Aligned Core Entities

Apirout manages traffic using four interconnected foundational entities:

```
[ Incoming Request ] ──► [ Route ] ──► [ Service ] ──► [ Upstream ] ──► [ Targets ]
                              │
                      [ Consumer Auth ]
```

### 1. Service (`/api/v1/services`)
Represents an upstream backend API or microservice:
```bash
curl -i -X POST http://localhost:8001/api/v1/services \
  -H "Authorization: Bearer <ADMIN_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "users-service",
    "url": "http://user-api.internal:8080/v1",
    "retries": 3,
    "connect_timeout": 5000,
    "read_timeout": 10000
  }'
```

### 2. Route (`/api/v1/routes`)
Defines rules to match client requests and forward them to a Service:
```bash
curl -i -X POST http://localhost:8001/api/v1/routes \
  -H "Authorization: Bearer <ADMIN_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "users-route",
    "service_id": "<SERVICE_ID>",
    "paths": ["/api/v1/users"],
    "methods": ["GET", "POST"],
    "strip_path": true,
    "auth_required": true
  }'
```

### 3. Upstream & Targets (`/api/v1/upstreams`)
Virtual host load balancer distributing load across target servers:
```bash
# 1. Create an Upstream pool
curl -i -X POST http://localhost:8001/api/v1/upstreams \
  -H "Authorization: Bearer <ADMIN_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "payment-upstream",
    "algorithm": "round-robin",
    "health_check_path": "/health"
  }'

# 2. Add Target instances
curl -i -X POST http://localhost:8001/api/v1/upstreams/<UPSTREAM_ID>/targets \
  -H "Authorization: Bearer <ADMIN_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "target": "10.0.1.15:8080",
    "weight": 100
  }'
```

### 4. Consumer (`/api/v1/consumers`)
Represents a client, partner, or developer consuming your APIs:
```bash
# 1. Register a Consumer
curl -i -X POST http://localhost:8001/api/v1/consumers \
  -H "Authorization: Bearer <ADMIN_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "mobile-app-client",
    "custom_id": "client-9901"
  }'

# 2. Provision an API Key
curl -i -X POST http://localhost:8001/api/v1/consumers/<CONSUMER_ID>/key-auth \
  -H "Authorization: Bearer <ADMIN_TOKEN>"
```

---

## ⚡ Instant Database APIs

Connect a PostgreSQL, MySQL, or SQL Server database in the Web Console to instantly expose REST endpoints:

```
GET /api/v1/instant/:connection_id/customers?status=eq.active&order=created_at.desc&limit=20
```

### Supported Query Operators
| Operator | Syntax Example | Meaning |
| :--- | :--- | :--- |
| `eq` | `?role=eq.admin` | Equals |
| `neq` | `?status=neq.deleted` | Not equals |
| `gt`, `gte` | `?price=gte.100` | Greater than / or equal |
| `lt`, `lte` | `?age=lt.30` | Less than / or equal |
| `like` | `?email=like.%40gmail.com` | Pattern matching |
| `in` | `?id=in.(1,2,5,10)` | Value in list |
| `order` | `?order=created_at.desc` | Sorting direction |
| `limit` / `offset` | `?limit=50&offset=100` | Standard pagination |

---

## ⚙️ Operation Modes (`ROLE`)

Apirout supports 3 deployment roles via the `ROLE` environment variable:

```bash
# 1. Traditional Mode (Default)
# Runs Proxy (:8000), Admin API (:8001), and Web GUI (:8002) in a single binary.
ROLE=traditional ./apirout

# 2. Control Plane Mode
# Runs Admin API (:8001) and Web GUI (:8002) only. Ideal for private management networks.
ROLE=control_plane ./apirout

# 3. Data Plane Mode
# Runs ultra-fast stateless Proxy (:8000) only. Perfect for edge Kubernetes clusters.
ROLE=data_plane ./apirout
```

---

## 📊 Observability & Metrics

Apirout natively monitors ingress health and performance:

- **Prometheus Metrics**: Available on `http://localhost:8100/metrics`.
  - `apirout_http_requests_total{status, method, route}`
  - `apirout_http_request_duration_seconds{route}`
  - `apirout_upstream_health_status{upstream, target}`
- **Synthetic Latency Percentiles**: Web Console calculates real-time **P50**, **P90**, and **P99** response times.
- **Traffic Logging**: Comprehensive request and response audit logs with client IP, response time, and consumer identification.

---

## 🛠️ Configuration Reference (`.env`)

| Variable | Default | Description |
| :--- | :--- | :--- |
| `ROLE` | `traditional` | Gateway role: `traditional`, `control_plane`, `data_plane` |
| `PROXY_PORT` | `8000` | Port for incoming public API traffic (Data Plane) |
| `ADMIN_PORT` | `8001` | Port for Control Plane Management REST API |
| `GUI_PORT` | `8002` | Port for Web Console Single Page Application |
| `STATUS_PORT` | `8100` | Port for health checks and Prometheus metrics |
| `ADMIN_SECRET` | `MasterSecretKey2026!` | Master API key for administrative operations |
| `JWT_SECRET` | `super-secret-jwt-key...` | Cryptographic secret for signing tokens |
| `DB_TYPE` | `postgres` | Registry database: `postgres`, `sqlite` |
| `DB_HOST` | `postgres` | Registry database hostname |
| `DB_PORT` | `5432` | Registry database port |
| `REDIS_HOST` | `redis` | Redis server for distributed cache invalidation |
| `REDIS_PORT` | `6379` | Redis server port |

---

## 📚 Documentation & Guides

- 📘 [Architecture Blueprint](docs/ARCHITECTURE.md) — Comprehensive technical architecture specification
- 🚀 [5-Minute Quickstart Guide](docs/QUICKSTART.md) — Step-by-step tutorial from zero to running API
- 🛡️ [Production Best Practices](docs/BEST_PRACTICES.md) — Hardening, security, and performance tuning
- 🌐 [Multiplane Deployment Plan](docs/PLAN_PEMISAHAN_MULTIPLANES.md) — Control Plane & Data Plane deployment guide

---

## 📄 License

Apirout is open-source software licensed under the [Apache License, Version 2.0](LICENSE).
