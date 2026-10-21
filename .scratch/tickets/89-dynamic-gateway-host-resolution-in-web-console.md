# 89: Dynamic Gateway Host Resolution in Web Console

**What to build:**
Replace hardcoded loopback addresses (`http://localhost:8000`) with dynamic origin resolution across query preview testers and cURL snippet copy actions. When accessed remotely, via LAN IPs, or through reverse proxies, test executions and copied commands must connect to the reachable gateway host.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] Create `getGatewayBaseUrl()` helper in `web/src/api/gatewayApi.ts` resolving from `VITE_GATEWAY_URL` or `${window.location.protocol}//${window.location.hostname}:8000`.
- [x] Replace hardcoded `http://localhost:8000` in `web/src/pages/InstantAPI.tsx`.
- [x] Replace hardcoded `http://localhost:8000` in `web/src/pages/Routes.tsx` and `web/src/pages/observability/ApiLogs.tsx`.
- [x] Verify frontend build passes cleanly.

