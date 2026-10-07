# 39: Unique Plugin Scope Guardrails and Duplicate Prevention

**What to build:** Enforces that a given entity (route or service) cannot attach duplicate instances of the same plugin type. In the control plane backend API (`POST /api/v1/auth-plugins`), reject duplicate plugin creation with HTTP 400 Bad Request. In the selection modal UI, disable already-attached plugin types with an "Already Attached" badge, while preserving heterogeneous plugin combinations (`key-auth` + `jwt-auth`).

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] Control Plane handler checks existing plugins for entity and rejects duplicate plugin_name with HTTP 400
- [x] Unit test in `auth_plugins_handler_test.go` verifies duplicate plugin rejection
- [x] Modal in `AuthPluginManager.tsx` inspects active plugins and disables already-attached types with "Already Attached" badge
- [x] Multiple different plugin types remain selectable and attachable
