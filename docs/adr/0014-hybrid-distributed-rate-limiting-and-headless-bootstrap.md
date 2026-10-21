# ADR-0014: Hybrid Distributed Rate Limiting, Headless Bootstrapping, and Asymmetric Resilience Topology

## Status
Accepted

## Context
Following comprehensive full-stack security auditing and architectural review, four production operational frontiers were identified:

1. **Multi-Pod Distributed Rate Limiting (Q1 C)**:
   In single-instance deployments, in-memory token bucket rate limiting offers zero-latency enforcement. However, in horizontally scaled Data Plane clusters (e.g. Kubernetes with multiple replicas behind an external ingress controller), local in-memory buckets cause limits to multiply linearly by the number of pods. A centralized rate limiter backed by Redis is needed, but it must not create a hard single-point-of-failure if Redis suffers network partition or transient latency spikes.

2. **Headless Cloud Auto-Bootstrapping (Q2 A)**:
   In automated Kubernetes and Docker deployments, human operators cannot easily interact with the `/setup` web UI wizard to configure the initial Super Admin account before automated smoke tests or CI/CD pipelines run. A secure, headless environment-variable provisioning mechanism (`BOOTSTRAP_ADMIN_EMAIL` and `BOOTSTRAP_ADMIN_PASSWORD`) is needed that auto-provisions the root Super Admin on first boot when zero admins exist, while preserving the interactive `/setup` wizard for manual UI deployments and immediately locking setup mode against unauthorized takeover.

3. **On-Demand Schema Discovery & Invalidation Sync (Q3 A)**:
   Instant CRUD inspects connected relational databases dynamically. When table columns change or database credentials rotate, Data Plane nodes holding cached connection pools must not serve stale schemas or stale connections. An EventBus-driven invalidation event must broadcast schema/connection updates so connection pools flush dynamically across all Data Plane nodes without requiring gateway restarts.

4. **Asymmetric Resilience Topology (Q4 A)**:
   Target upstream health checking can easily lead to a "thundering herd" if every stateless Data Plane pod independently sends active HTTP probes to every upstream target. A decoupled, asymmetric model is required where the Control Plane asynchronously manages active HTTP health checks, while Data Plane nodes remain completely stateless and utilize localized passive circuit breakers (3 consecutive 5xx trips for 15s) for immediate edge failover.

## Decision

1. **Hybrid Distributed Rate Limiter**:
   - Provide an atomic Redis token bucket Lua script that mirrors the refill mathematics of the local token bucket (`tokens = math.min(burst, tokens + elapsed * rate)`).
   - Gate distributed rate limiting behind an explicit configuration toggle: `DISTRIBUTED_RATE_LIMIT=true` and a valid Redis connection.
   - Enforce a 100ms execution timeout on Redis calls. If Redis is partitioned, unavailable, or times out, seamlessly and automatically fall back to the local in-memory token bucket without dropping or delaying client requests.
   - Default to pure in-memory token bucket execution when `DISTRIBUTED_RATE_LIMIT=false` (or when Redis is not configured), ensuring zero added network hops and zero external dependencies for lightweight edge runtimes.

2. **Headless Cloud Auto-Bootstrap**:
   - Introduce `BOOTSTRAP_ADMIN_EMAIL` and `BOOTSTRAP_ADMIN_PASSWORD` in the application configuration.
   - In production environments, enforce strict validation: password minimum length of 8 characters, valid email format, and rejection of known insecure placeholders (`Password123!`, `admin`, `password`).
   - During Control Plane initialization (`app.New`), immediately after database migrations, check `HasSuperAdmin()`. If zero super admins exist and bootstrap credentials are provided, provision the root Super Admin account.
   - Log completion securely without leaking password material. Once provisioned, setup mode permanently locks down (`SetupRequired = false`).

3. **On-Demand Schema & Connection Invalidation Event**:
   - Enhance `internal/console/registry/connections.go` so `UpdateConnection` and `DeleteConnection` emit `PublishConnectionChanged` across the EventBus.
   - In `internal/console/server/connections.go`, when the `/connections/{id}/introspect` endpoint executes, emit `PublishConnectionChanged("UPDATE", id)` to notify all Data Plane nodes.
   - Data Plane workers subscribing to `events.ChannelConnectionEvents` or typed `eventbus.EntityConnection` immediately invalidate their cached `ManagedConnectionPool` entries for that connection ID.

4. **Asymmetric Resilience Topology**:
   - Reaffirm the decoupled architecture: the Control Plane background worker (`StartBackgroundHealthChecker`) performs centralized active health probes and publishes target health transitions (`PublishTargetHealthChanged`) over EventBus.
   - Data Plane nodes run the localized `PassiveCircuitBreaker` (tripping on 3 consecutive 5xx errors for a 15s cooldown) to failover instantaneously during live proxy operations, while listening to Control Plane health events to proactively trip or reset target availability.

## Consequences

- **Positive**:
  - Multi-pod gateway deployments achieve cluster-wide rate limit compliance with zero single-point-of-failure risk.
  - Zero-touch CI/CD and GitOps deployments can stand up fully functioning, secured Apirout environments in seconds without human manual intervention.
  - Relational schema changes propagate dynamically across all gateway nodes without downtime or connection leaks.
  - Upstream backend services are protected from probe storms while enjoying instant edge failover resilience.
- **Negative / Operational Considerations**:
  - Operators running distributed rate limiting must ensure Redis has sufficient memory for token bucket keys (each key expires automatically after 120 seconds).
  - Bootstrap environment variables should be injected securely via secrets managers (Kubernetes Secrets, HashiCorp Vault) rather than plaintext configuration files.
