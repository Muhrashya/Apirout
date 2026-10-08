# 43: Modal Action Event Isolation and Parent Route Validation Shield

**What to build:** Decouples the plugin configuration modal inside `AuthPluginManager` from native form submission semantics. Converts the modal action button to `type="button"` with an explicit click handler, and invokes `e.stopPropagation()` and `e.preventDefault()`. Ensures staging or adding an authentication plugin within route and service forms never triggers premature parent route form submission or validation error toasts (such as "Match Path must start with /api/").

**Blocked by:** 42

**Status:** completed

- [x] Modal action button in `AuthPluginManager.tsx` converted to `type="button"` with dedicated handler stopping event propagation (`stopPropagation()`)
- [x] Modal dialog form prevents bubbling to parent `<form>` elements
- [x] Staging an authentication plugin in `RouteFormPage` does not trigger route path validation error toasts
- [x] Staging an authentication plugin in `ServiceFormPage` does not trigger service form validation
- [x] Typechecks cleanly with `npm run build` in `web/`
