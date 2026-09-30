# Apirout Production Best Practices

This guide outlines security, performance, deployment, and operational best practices for deploying **Apirout** in production.

---

## 1. Network Topology & Isolation

### Separate Control Plane and Data Plane
For enterprise production workloads, do **not** expose the Control Plane to the public internet:
- **Control Plane (`APIROUT_ROLE=control_plane`)**:
  - Deploy inside an internal VPC or behind an authenticated VPN.
  - Whitelist CIDR blocks in your Kubernetes Ingress (`nginx.ingress.kubernetes.io/whitelist-source-range: "10.0.0.0/8, 172.16.0.0/12"`).
- **Data Plane (`APIROUT_ROLE=data_plane`)**:
  - Place in DMZ / public ingress.
  - Expose only port 8080 (or 443 via Ingress/TLS termination).
  - Enable Horizontal Pod Autoscaler (HPA) targeting 70-80% CPU utilization.

---

## 2. Security Hardening

### Secrets & Encryption Keys
- **JWT_SECRET**: Use a cryptographically secure string of at least 32 characters (`openssl rand -base64 32`).
- **APP_ENCRYPTION_KEY**: Target database credentials are encrypted using AES-256 in Galois/Counter or CFB mode before storing in the registry. Keep this key secure and rotated across environments.
- **Admin Secret Header**: In headless or CI/CD pipelines, authenticate requests using `X-Apirout-Admin-Secret` instead of manual token generation.

### Database Target Permissions
- When connecting target PostgreSQL or MySQL databases to Apirout, create a dedicated database user with principle of least privilege:
  ```sql
  -- Postgres example: Read-only reporting user
  CREATE USER apirout_reader WITH PASSWORD 'secure_pass';
  GRANT CONNECT ON DATABASE production_db TO apirout_reader;
  GRANT USAGE ON SCHEMA public TO apirout_reader;
  GRANT SELECT ON ALL TABLES IN SCHEMA public TO apirout_reader;
  ```

---

## 3. High Availability & Scaling

### Database Connection Pooling
Target database connections in Apirout Data Plane are managed via an adaptive LRU connection pool:
- Set `max_open_conns` to match your target DB capacity (default 50 per target database).
- Idle connections are automatically pruned every 15 minutes to prevent connection exhaustion on target databases.

### Zero-Downtime Cache Sync
- Always connect a distributed Redis instance in multi-node Data Plane clusters.
- When an API endpoint or connection credential is changed in the Control Plane, an invalidation message is broadcast over Redis Pub/Sub. All Data Plane nodes invalidate their local memory in `< 2ms` without restart.

---

## 4. Multi-Domain Routing Best Practices

When hosting APIs across multiple domains on the same Gateway cluster:
1. Configure DNS CNAME records for each domain pointing to your Gateway load balancer (e.g. `api.company.com`, `mitra.company.com`).
2. Add the domains in **Settings > Gateway Domains** in the Apirout Console.
3. In **Endpoints**, bind specific endpoints to designated domains (or `*` for all domains).
4. If a client attempts to access an endpoint on an unauthorized domain, Apirout returns `404 Not Found`, keeping partner and public scopes strictly segregated.

---

## 5. Declarative GitOps Workflow

Treat your API configurations as code:
1. Export your API definitions via Console (**Settings > GitOps Export**) or CLI:
   ```bash
   curl -H "X-Apirout-Admin-Secret: $APIROUT_ADMIN_SECRET" \
     http://console.internal:8081/api/v1/gitops/export > apirout-metadata.yaml
   ```
2. Commit `apirout-metadata.yaml` to your Git repository.
3. In your CI/CD pipeline (GitHub Actions, GitLab CI), deploy changes automatically:
   ```bash
   curl -X POST -H "X-Apirout-Admin-Secret: $APIROUT_ADMIN_SECRET" \
     --data-binary @apirout-metadata.yaml \
     http://console.internal:8081/api/v1/gitops/import
   ```
