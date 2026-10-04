# Apirout Deployment Guide

This guide covers deployment topologies for the Apirout platform across local, single-server production, and Kubernetes cluster environments.

---

## Deployment Topologies at a Glance

| Deployment Target | Orchestrator | Edge Ingress | Use Case |
|---|---|---|---|
| **Local Development** | Docker Compose | Direct Multi-Port Listeners | Feature development, manual API testing, local debugging |
| **Production VPS / VM** | Docker Compose | Nginx Edge Reverse Proxy (Ports 80/443) | Single-instance production, digital ocean / AWS EC2 droplets |
| **Enterprise Cloud Cluster** | Kubernetes (EKS/GKE/K3s) | Ingress Controller + NetworkPolicy | High-availability multi-plane scaling with automated autoscaling |

---

## 1. Local Development (All-in-One Compose)

Runs PostgreSQL, Redis, and Apirout in traditional all-in-one mode with direct multi-port exposure:

```bash
docker compose up -d --build
```

### Endpoints
- **Web Console GUI**: [http://localhost:8002](http://localhost:8002)
- **Admin Management API**: [http://localhost:8001/api/v1](http://localhost:8001/api/v1)
- **Data Plane Gateway Proxy**: [http://localhost:8000/api/*](http://localhost:8000/api/*)
- **Status & Prometheus Metrics**: [http://localhost:8100/metrics](http://localhost:8100/metrics)

---

## 2. Production Single-Node VPS (Docker Compose + Nginx Edge Proxy)

Located at [deploy/docker-compose.prod.yml](docker-compose.prod.yml).

### Architecture
- **Single Public Port Entrypoint**: Only ports **80** (HTTP) and **443** (HTTPS) are exposed on the host.
- **Internal Network Isolation**: Ports `8000`, `8001`, `8002`, `8100`, `5432` (Postgres), and `6379` (Redis) are completely unexposed to the public internet and operate inside an isolated Docker bridge network (`apirout-network`).
- **Nginx Reverse Proxy**:
  - Routes `/api/` traffic to the Gateway Proxy (`:8000`) with high-throughput buffers.
  - Routes `/` traffic to the React Web Console (`:8002`) with WebSocket upgrade support.
  - Restricts `/admin-api/` (`:8001`) with CIDR allowlists.
  - Restricts `/status/` (`:8100`) strictly to internal monitoring.
  - Preserves real client IPs via `X-Forwarded-For` and `X-Real-IP`.

### Launching Production Compose
```bash
docker compose -f deploy/docker-compose.prod.yml up -d --build
```

### SSL / TLS Certificate Setup (Let's Encrypt / Certbot)
To obtain and attach SSL certificates:
```bash
docker run -it --rm \
  -v certbot_certs:/etc/letsencrypt \
  -v certbot_www:/var/www/certbot \
  certbot/certbot certonly --webroot -w /var/www/certbot \
  -d api.yourdomain.com -d console.yourdomain.com
```

---

## 3. Production Kubernetes Multi-Plane Cluster

Located at [deploy/k8s/](k8s/).

The Kubernetes manifests separate Apirout into autonomous **Control-Plane** and **Data-Plane** deployments following Kong Gateway's enterprise architectural model:

| Manifest | Purpose |
|---|---|
| [`00-namespace.yaml`](k8s/00-namespace.yaml) | Isolated `apirout` namespace |
| [`01-configmaps-secrets.yaml`](k8s/01-configmaps-secrets.yaml) | Shared configuration, database connection, and encryption keys |
| [`02-redis.yaml`](k8s/02-redis.yaml) | Distributed cache invalidation and pub/sub broker |
| [`03-control-plane.yaml`](k8s/03-control-plane.yaml) | Web Console and Admin API (2 Replicas, Ingress on :8002 and :8001) |
| [`04-data-plane-gateway.yaml`](k8s/04-data-plane-gateway.yaml) | Stateless Gateway Proxy (3 Replicas, Horizontal Pod Autoscaler up to 20 Pods, Ingress on :8000) |
| [`05-network-policy.yaml`](k8s/05-network-policy.yaml) | Zero-trust ingress and egress network isolation |

### Deploying to Kubernetes
```bash
# 1. Create namespace, configs, and secrets
kubectl apply -f deploy/k8s/00-namespace.yaml
kubectl apply -f deploy/k8s/01-configmaps-secrets.yaml

# 2. Deploy Redis broker
kubectl apply -f deploy/k8s/02-redis.yaml

# 3. Deploy Control-Plane and Data-Plane
kubectl apply -f deploy/k8s/03-control-plane.yaml
kubectl apply -f deploy/k8s/04-data-plane-gateway.yaml

# 4. Apply NetworkPolicies
kubectl apply -f deploy/k8s/05-network-policy.yaml
```

---

## 4. Validating Deployment Assets

Run the automated deployment validation suite locally:

```bash
# In PowerShell:
powershell -ExecutionPolicy Bypass -File scripts/validate_deploy.ps1
```
This tests:
1. Docker Compose production recipe syntax.
2. Nginx edge reverse proxy configuration (`nginx -t`).
3. Kubernetes manifest integrity.
