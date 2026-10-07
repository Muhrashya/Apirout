# 38: Clean Brand Identity Sanitization and Dead Code Purge

**What to build:** Eliminates all 3rd-party/Kong wording from the Web Console (`AuthPluginManager.tsx`, `Upstreams.tsx`, `Services.tsx`, `Routes.tsx`, `RouteFormPage.tsx`, `Consumers.tsx`, `gatewayApi.ts`), standardizing entirely on native Apirout Gateway branding. Permanently deletes orphan dead files `web/src/pages/AuditLogs.tsx` and `web/src/pages/Docs.tsx`, and verifies `npm run build` passes with zero bundler or type errors.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] All instances of "Kong" and "Kong-aligned" replaced with "Apirout Gateway" across Web Console views and descriptions
- [x] Orphan files `web/src/pages/AuditLogs.tsx` and `web/src/pages/Docs.tsx` permanently deleted
- [x] No remaining unreferenced imports in `web/`
- [x] `npm run build` succeeds cleanly with zero TypeScript or bundling errors
