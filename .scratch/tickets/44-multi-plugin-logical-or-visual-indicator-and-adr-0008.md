# 44: Multi-Plugin Logical OR Visual Indicator and Architectural Record (ADR-0008)

**What to build:** Renders an informational banner in `AuthPluginManager` when $\ge 2$ inbound authentication plugins are staged or attached to an entity, providing visual clarity that the gateway evaluates multi-plugin configurations with Logical OR semantics (requests pass if any attached plugin authenticates the client). Automatically hides the banner when fewer than 2 plugins exist. Documents the architectural decisions in `docs/adr/0008-form-event-isolation-and-multi-plugin-or-semantics.md` and verifies the full web bundle and test suites.

**Blocked by:** 43

**Status:** completed

- [x] Informational banner rendered above plugin list in `AuthPluginManager.tsx` whenever $\ge 2$ active plugins are attached or staged
- [x] Banner explains Logical OR runtime evaluation clearly to operators
- [x] Banner automatically hidden when $< 2$ active plugins are attached or staged
- [x] Architectural decisions codified in `docs/adr/0008-form-event-isolation-and-multi-plugin-or-semantics.md`
- [x] `web/` builds cleanly without errors (`npm run build`)
- [x] Full E2E suite passes 100% (`go test -v ./tests/e2e/...`)
