# Integration & End-to-End (E2E) Testing

Direktori ini digunakan untuk pengujian integrasi dan end-to-end yang menguji interaksi nyata antar subsistem Apirout:
- **Data Plane (Port 8000)**: Verifikasi proxy forwarding, circuit breaker, instant CRUD, dan rate limiting.
- **Control Plane (Port 8001 / 8002)**: Verifikasi admin management API, audit log, dan schema introspection.
- **Redis Pub/Sub**: Verifikasi cache invalidation saat konfigurasi diubah di Control Plane.
- **PostgreSQL / SQLite**: Verifikasi persistensi metadata dan schema migration.

## Struktur Direktori
- `test/e2e/`: Pengujian skenario end-to-end HTTP menggunakan suite `testing` standar Go atau Newman/Postman.
- `test/testdata/`: Data uji, mock queries, dan fixture SQL untuk database target.

## Menjalankan Pengujian
```bash
# Menjalankan unit tests
go test -v ./internal/...

# Menjalankan E2E tests (memerlukan container docker aktif)
go test -v -tags=e2e ./test/e2e/...
```
