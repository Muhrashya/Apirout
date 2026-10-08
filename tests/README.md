# Apirout End-to-End (E2E) & Integration Testing

This directory contains integration and end-to-end tests for the Apirout platform. The test suites verify live HTTP interactions across all platform subsystems against running Docker containers:

- **Data Plane (Port 8000)**: Verifies reverse proxy forwarding, passive circuit breakers, multi-auth plugins (Key Auth, JWT, Basic, HMAC), Logical OR authentication evaluation, Instant CRUD database operations, and normalized error envelopes.
- **Control Plane Management API (Port 8001)**: Verifies administrative CRUD for routes, services, upstreams, targets, consumers, consumer groups, SQL queries, database connections, and audit log telemetry.
- **Console Web UI (Port 8002)**: Verifies web application delivery, setup lifecycle guards, and static asset serving.
- **Telemetry & Health (Port 8100)**: Verifies Prometheus metrics export (`/metrics`) and Kubernetes readiness/liveness probes (`/health`).
- **Distributed Cache & State Invalidation (Redis & PostgreSQL)**: Verifies EventBus invalidation sync when configuration changes are published from the Control Plane to the Data Plane.

---

## Directory Structure

```
tests/
├── README.md                      # This documentation
└── e2e/
    ├── e2e_features_test.go       # Core gateway, authentication plugins, CRUD, and rate limiting
    └── e2e_deep_features_test.go  # Deep features: multi-plugin OR, circuit breakers, declarative state, and auditing
```

---

## Prerequisites

Before running the E2E test suite, ensure the local Docker environment is running:

```bash
docker compose up -d
```

Verify that the containers are healthy:
- `apirout` listening on ports `8000`, `8001`, `8002`, `8100`
- `apirout-registry-db` listening on port `5432`
- `apirout-redis` listening on port `6379`

---

## Running Tests

### 1. Execute All E2E Suites

```bash
go test -v ./tests/e2e/...
```

### 2. Execute Specific Test Suite

```bash
# Run core features suite
go test -v -run TestE2E_CompleteGatewaySuite ./tests/e2e/e2e_features_test.go

# Run deep architectural features suite
go test -v -run TestE2E_DeepFeaturesSuite ./tests/e2e/e2e_deep_features_test.go
```

### 3. Run Internal Unit Tests

```bash
go test -v ./internal/...
```
