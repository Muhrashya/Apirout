# 62: Inbound Auth Engine Consolidation and Complete Phantom OAuth2 Purge

**What to build:**
Absorb `apiclientauth.Service` directly into `auth.Engine`, presenting a single unified ingress authentication seam. `auth.Engine.Authenticate` evaluates active plugins (`key-auth`, `basic-auth`, `jwt-auth`, `hmac-auth`) and verifies consumer route-level permissions in a single atomic pass, returning a verified `ConsumerContext` or a canonical error envelope. Completely purge all residual phantom `oauth2` dropdown options and dead code branches across `Consumers.tsx`, `AuthPluginManager.tsx`, and models (`models.PluginOAuth2`), strictly aligning the console with genuine gateway capabilities and eliminating unresolvable 401 lockouts.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] Inbound auth engine absorbs client auth token validation and route authorization checks in one atomic pass
- [x] `stage_auth.go` simplified to call only the consolidated `auth.Engine.Authenticate` seam
- [x] `Consumers.tsx` credential type dropdown purged of phantom `oauth2` option
- [x] `AuthPluginManager.tsx` dead `oauth2` plugin state and branches purged
- [x] `internal/core/models/auth_plugin.go` removes `PluginOAuth2` from valid plugin names list
- [x] Unit and regression tests verify multi-plugin authentication and route access permissions through the unified seam

## Answer
Absorbed `apiclientauth.Service` token methods into `auth.Engine`, executed consumer route authorization checks directly within `auth.Engine.Authenticate`, simplified `stage_auth.go` to delegate all authentication and route permission checks to `authEngine.Authenticate`, and purged all dead inbound `oauth2` branches and dropdown options from `Consumers.tsx`, `AuthPluginManager.tsx`, and `models.PluginOAuth2`. Verified via Go unit tests and Vite production build.
