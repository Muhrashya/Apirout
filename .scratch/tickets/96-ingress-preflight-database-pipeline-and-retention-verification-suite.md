# 96: Ingress Preflight, Database Pipeline and Retention Verification Suite

**What to build:**
Comprehensive automated regression test suite covering all vertical slices through public HTTP interfaces (Gateway port 8000, Admin API port 8001, and Web Console): testing CORS preflights, IPv6 ingress, unified Instant CRUD with rate limiting, preview safety, wildcard path stripping, direct circuit breaking, and telemetry retention pruning, ensuring 100% green builds across Go and React.

**Blocked by:** 91, 92, 93, 94, 95

**Status:** completed

- [x] Run unit tests for ingress CORS preflight and IPv6 host splitting.
- [x] Run unit tests for unified instant database pipeline execution and CRUD methods.
- [x] Run unit tests for query preview safety and read-only connection write rejection.
- [x] Run unit tests for wildcard path stripping and direct service circuit breaking.
- [x] Run unit tests for audit log retention pruner and tenant connection group RBAC.
- [x] Run full project test suite (`go test ./...`) with zero failures.
- [x] Run Web Console production build (`npm run build`) with zero TypeScript/Vite errors.
