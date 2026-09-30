# Deep Architecture Plan: Pemisahan Multi-Plane & Multi-Port Apirout (Referensi Arsitektur Kong Gateway)

Dokumen ini merinci rencana arsitektur mendalam untuk memisahkan **Apirout** ke dalam arsitektur **Multi-Port, Multi-Plane, dan Isolated Traffic Controller** mengadopsi standar industri **Kong Gateway**.

---

## 1. Analisis & Sasaran Kebutuhan

Berdasarkan kebutuhan:
1. **Web UI tidak di-VPN**: Web UI (Admin Console) dapat diakses secara publik atau semi-publik oleh admin/operator tanpa kewajiban menggunakan VPN hardware/perusahaan, namun diamankan dengan proteksi berlapis (RBAC, Session Management, dan IP Allowlist).
2. **Pemisahan Port Mandiri (Multi-Port Listeners)**:
   - **Web UI Listener**: Berjalan di port tersendiri (port `8002`).
   - **Admin / System Management API**: Berjalan di port tersendiri (port `8001`).
   - **Consumer API Proxy Gateway**: Berjalan di port tersendiri (port `8000`).
   - **Status & Metrics Exporter**: Berjalan di port tersendiri (port `8100`).
3. **Pengaturan Traffic Mandiri (Traffic Controller)**:
   - Pengaturan routing, allowed domains, dan **allowed IPs (IP Whitelist / CIDR restrictions)** diatur langsung oleh operator melalui UI Console.
   - Traffic proxy diisolasi dari traffic admin/manajemen sistem.
4. **Pemisahan Ingress & Egress**:
   - Ingress terpisah untuk setiap plane.
   - Egress NetworkPolicy ketat yang membatasi gateway hanya dapat menghubungi Redis dan target database terdaftar.

---

## 2. Kong-Style Port Matrix & Listener Topology

Mengadopsi pola port bawaan Kong Gateway:

| Listener Name | Default Port | Protokol | Sasaran Konsumen | Deskripsi Fungsi |
|---|---|---|---|---|
| **Proxy Gateway** (`APIROUT_PROXY_LISTEN`) | `8000` (HTTP) / `8443` (TLS) | HTTP/1.1, HTTP/2 | Public Clients, Mobile Apps, Mitra B2B | Menangani trafik pemanggilan API, Instant Database CRUD, validasi domain, dan IP allowlist. |
| **Admin Management API** (`APIROUT_ADMIN_LISTEN`) | `8001` (HTTP) / `8444` (TLS) | HTTP/1.1 | CI/CD, GitOps, Web UI Internal Call | REST API internal untuk konfigurasi route, koneksi DB, token, dan user. Dilindungi Admin Secret / internal network. |
| **Console Web UI (GUI)** (`APIROUT_GUI_LISTEN`) | `8002` (HTTP) / `8445` (TLS) | HTTP/1.1 | Browser Admin, Tim DevOps, Operator | Web dashboard visual React Vite. Bisa diakses publik/kantor tanpa VPN karena terpisah port dan dilindungi RBAC. |
| **Status & Metrics** (`APIROUT_STATUS_LISTEN`) | `8100` | HTTP/1.1 | Prometheus, Kubernetes Probes | Healthcheck readiness/liveness (`/health`) dan Prometheus scrape target (`/metrics`). |

---

## 3. Fitur Traffic Control Lanjutan: Domain & IP Restriction (Kong Pattern)

Setiap endpoint atau grup API di Apirout akan memiliki dua lapisan penyaring trafik (*traffic filtering*) sebelum menyentuh database target:

### A. Host-Based Domain Routing
- **Header Matching**: Mengekstrak `Host` atau `X-Forwarded-Host`.
- **Domain Whitelist**: Mendukung multi-domain (`api.corp.com`, `mitra.corp.com`) atau wildcard (`*.corp.com`, `*`).
- **Respon Domain Tidak Sah**: Menghasilkan `404 Not Found` (mencegah attacker mengetahui keberadaan endpoint di domain yang salah).

### B. Client IP Restriction / Allowlist (Kong `ip-restriction` Plugin Parity)
- **Ekstraksi Client Real IP**:
  - Membaca `X-Forwarded-For` (mengambil client IP terluar berdasarkan daftar trusted reverse proxies).
  - Fallback ke `X-Real-IP` atau `RemoteAddr`.
- **Format IP Whitelist**:
  - Single IP: `203.0.113.195`
  - Subnet CIDR: `192.168.1.0/24`, `10.0.0.0/8`, `103.20.14.0/22`
  - Wildcard: `*` (terbuka untuk semua IP)
- **Respon IP Ditolak**: Menghasilkan `403 Forbidden` (`{"error": "IP address not allowed"}`).

---

## 4. Arsitektur Ingress & Egress Kubernetes

### Ingress Policies (Inbound Traffic)
1. **Console Web UI Ingress (`apirout-console-ingress`)**:
   - Host: `console.apirout.io`
   - Port Target: `8002`
   - Akses: Publik / Internet (dilindungi autentikasi login JWT, rate limiting, dan proteksi brute force). Tidak perlu VPN.
2. **Consumer Proxy Gateway Ingress (`apirout-gateway-ingress`)**:
   - Hosts: `api.apirout.io`, `mitra.apirout.io`, `*.apirout.io`
   - Port Target: `8000`
   - Akses: Publik dengan optimasi HTTP/2, buffer 8MB, dan timeout tinggi untuk query streaming.
3. **Admin Management API Ingress (`apirout-admin-api-ingress`)**:
   - Host: `admin-api.internal.apirout.io` (atau ClusterIP internal saja)
   - Port Target: `8001`
   - Akses: Hanya internal cluster atau IP whitelist DevOps CIDR.

### Egress NetworkPolicy (Outbound Traffic)
Data Plane Gateway hanya diizinkan membuka koneksi keluar (*egress*) ke:
- Port `5432` (PostgreSQL registry)
- Port `6379` (Redis Pub/Sub)
- Port target database yang didefinisikan (misal `5432`, `3306`, `1433`)
- Port `53` (DNS UDP/TCP)

---

## 5. Blueprint Refaktorisasi Kode

### A. Struktur Folder Modular
```
internal/
├── config/
│   └── config.go             # APIROUT_PROXY_LISTEN, ADMIN_LISTEN, GUI_LISTEN, STATUS_LISTEN
├── core/
│   ├── models/
│   │   ├── endpoint.go       # Ditambahkan field: allowed_ips (TEXT/VARCHAR)
│   │   └── gateway_domain.go # Ditambahkan field: default_allowed_ips
│   ├── iputil/               # Parser CIDR & Real-IP extractor
│   │   └── ip_checker.go
├── server/
│   ├── proxy/                # High-speed API Gateway Proxy Listener (Port 8000)
│   │   ├── server.go
│   │   ├── router.go
│   │   └── middleware/
│   │       ├── ip_restriction.go
│   │       └── domain_matcher.go
│   ├── admin/                # Management & Configuration REST API (Port 8001)
│   │   └── server.go
│   ├── gui/                  # Web Console Static & Reverse Proxy Server (Port 8002)
│   │   └── server.go
│   └── status/               # Healthcheck & Prometheus Metrics Listener (Port 8100)
│       └── server.go
```

---

## 6. Docker Compose Multi-Port Topology

```yaml
services:
  apirout:
    build: .
    container_name: apirout
    environment:
      APIROUT_ROLE: traditional
      APIROUT_PROXY_LISTEN: "0.0.0.0:8000"
      APIROUT_ADMIN_LISTEN: "0.0.0.0:8001"
      APIROUT_GUI_LISTEN: "0.0.0.0:8002"
      APIROUT_STATUS_LISTEN: "0.0.0.0:8100"
      APIROUT_DATABASE_URL: "postgres://postgres:Password123!@postgres:5432/apirout?sslmode=disable"
      APIROUT_REDIS_URL: "redis://redis:6379"
      APIROUT_ADMIN_SECRET: "ApiroutMasterAdminSecret2026!"
    ports:
      - "8000:8000" # Proxy Gateway (Consumer traffic)
      - "8001:8001" # Admin Management API (Internal/GitOps)
      - "8002:8002" # Web Console UI (No VPN required)
      - "8100:8100" # Healthcheck & Prometheus Metrics
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
```

---

## 7. Roadmap Implementasi

1. **Tahap 1: Ekstensi Model & Database Migration**
   - Menambahkan kolom `allowed_ips` ke tabel `endpoints` dan `gateway_domains`.
2. **Tahap 2: Engine IP Restriction & Multi-Port Listeners**
   - Membuat package `internal/core/iputil/ip_checker.go`.
   - Membuat middleware `IPRestrictionMiddleware` di Data Plane Proxy.
   - Refaktor listener ke 4 port independen: `8000` (Proxy), `8001` (Admin API), `8002` (Web UI), `8100` (Status/Metrics).
3. **Tahap 3: Update UI Console**
   - Menambahkan input field `allowed_ips` di editor Endpoint dan Gateway Domains.
4. **Tahap 4: Kubernetes Ingress & Egress Manifests**
   - Ingress terpisah untuk GUI, Gateway, dan Admin API + Egress NetworkPolicy.
5. **Tahap 5: Build, Verifikasi, & Pengujian UAT**
