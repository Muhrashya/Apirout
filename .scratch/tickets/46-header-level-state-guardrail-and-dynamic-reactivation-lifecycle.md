# 46: Header-Level State Guardrail and Dynamic Re-activation Lifecycle

**What to build:** Protects the entity form header. When all 4 available plugin types (`key-auth`, `jwt-auth`, `basic-auth`, `hmac-auth`) are attached or staged, the top "Attach Plugin" header button becomes disabled, styled with a muted appearance, and labeled "All Plugins Attached" with a descriptive tooltip. Discarding or removing an attached plugin immediately re-enables the button and unlocks that plugin type in the catalog without requiring a page refresh.

**Blocked by:** 45

**Status:** completed

- [x] Header "Attach Plugin" button is disabled and labeled "All Plugins Attached" when all 4 plugins are active
- [x] Hovering over the disabled header button displays a descriptive tooltip explaining all plugins are attached
- [x] Discarding a staged plugin immediately re-enables the header button and makes that plugin type selectable
- [x] Removing an active plugin immediately re-enables the header button and unblocks that plugin type
- [x] Typechecks cleanly with `npm run build` in `web/`
