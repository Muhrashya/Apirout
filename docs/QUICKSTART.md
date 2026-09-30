# Apirout Quickstart Guide

Get up and running with **Apirout** in less than 5 minutes.

---

## 1. Quick Launch with Docker Compose

Clone the repository and launch the full stack:

```bash
git clone https://github.com/apirout/apirout.git
cd apirout

# Launch PostgreSQL, Redis, and Apirout in Traditional (All-in-One) mode
docker compose up -d
```

Verify services are online across dedicated ports:
- **Web Console GUI**: [http://localhost:8002](http://localhost:8002) (No VPN required)
- **Admin Management API**: [http://localhost:8001/health](http://localhost:8001/health)
- **Consumer API Gateway**: [http://localhost:8000/health](http://localhost:8000/health)
- **Status & Prometheus Metrics**: [http://localhost:8100/metrics](http://localhost:8100/metrics)

---

## 2. Log in to Web Console GUI

1. Open [http://localhost:8002](http://localhost:8002) in your browser.
2. Log in with initial credentials:
   - **Email**: `admin@apirout.io`
   - **Password**: `AdminApirout2026!`

---

## 3. Instant Database API in 3 Steps (Hasura Style)

### Step 1: Add a Connection
Navigate to **Settings > Connections > New Connection**:
- Type: `PostgreSQL` or `MySQL`
- Host, Port, Database name, Username, Password.
- Click **Test Connection**, then **Save**.

### Step 2: Explore Introspected Tables
Navigate to **Instant API** in the sidebar:
- Select your database connection.
- Review all automatically introspected tables, column types, and primary keys.
- Test queries live in the browser using dynamic filters:
  ```
  limit=10&order=id.desc&status=eq.active
  ```

### Step 3: Consume Instant REST Endpoints
Your database tables are immediately exposed over HTTP:

```bash
# List rows with pagination and filters
curl "http://localhost:8000/api/v1/instant/<connection-id>/users?limit=20&status=eq.active"

# Get row by ID
curl "http://localhost:8000/api/v1/instant/<connection-id>/users/1"

# Insert new row
curl -X POST "http://localhost:8000/api/v1/instant/<connection-id>/users" \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice@example.com"}'

# Update row
curl -X PUT "http://localhost:8000/api/v1/instant/<connection-id>/users/1" \
  -H "Content-Type: application/json" \
  -d '{"status":"verified"}'

# Delete row
curl -X DELETE "http://localhost:8000/api/v1/instant/<connection-id>/users/1"
```

---

## 4. Multi-Domain & Client IP Allowlisting

When creating or modifying an endpoint:
- **Allowed Domains**: restrict the Host header (e.g. `api.apirout.io,client.apirout.io` or `*` for all).
- **Allowed Client IPs**: restrict incoming client IPs by CIDR or individual IP (e.g. `10.0.0.0/8, 192.168.1.0/24, 203.0.113.15` or `*` for all).
- If client IP is outside the allowed list, the Gateway immediately returns `HTTP 403 Forbidden`.

---

## 5. Kong-Style Role Configuration

Switch operation modes easily via environment variables:

```bash
# Run All-in-One (default)
ROLE=traditional ./bin/apirout

# Run Control Plane (Admin API on :8001, Web GUI on :8002)
ROLE=control_plane ./bin/apirout

# Run Stateless Data Plane (Gateway on :8000, Status on :8100)
ROLE=data_plane ./bin/apirout
```
