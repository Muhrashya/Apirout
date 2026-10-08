# 48: Frontend Dead Code Purge and Routing Flattening

**What to build:** Eliminates unreferenced dead page components and redundant re-export shims in the web console. Purges `web/src/pages/Endpoints.tsx`. Updates `web/src/App.tsx` to lazy-load and mount `Overview` directly for `/overview`, deleting `web/src/pages/Dashboard.tsx`. Updates `web/src/App.tsx` to redirect `/settings/api-clients` directly to `/consumers` via router navigation, deleting `web/src/pages/settings/ApiClients.tsx`. Verifies clean TypeScript compilation and bundle generation with `npm run build`.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] `web/src/pages/Endpoints.tsx` deleted
- [x] `web/src/pages/Dashboard.tsx` deleted and `web/src/App.tsx` lazy-loads `Overview` directly
- [x] `web/src/pages/settings/ApiClients.tsx` deleted and `web/src/App.tsx` redirects `/settings/api-clients` to `/consumers`
- [x] `npm run build` in `web/` executes with 0 errors
