# ADR 0008: Form Event Isolation, Toast Temporal Deduplication, and Multi-Plugin Logical OR Semantics

## Status
Accepted

## Context
1. **Modal Event Bubbling into Parent Forms:** In the Apirout Web Console, `AuthPluginManager` provides an embedded modal dialog allowing operators to attach or stage authentication plugins within entity configuration pages (`RouteFormPage` and `ServiceFormPage`). Because React synthetic events bubble through the DOM hierarchy to parent `<form>` elements, clicking the modal's primary action button triggered the outer route form's submit listener prematurely. This resulted in unexpected validation errors (e.g., *"Match Path must start with /api/ (e.g. /api/v1/resource)"*) being surfaced to the operator before they had finished authoring the route.
2. **Duplicate Toast Notification Spam:** Rapid successive button clicks or dual-triggered event dispatches produced identical stacked toast notifications (such as two identical *"✓ Staged hmac-auth (pending save)"* pills) due to the lack of temporal deduplication in `ToastProvider`.
3. **Ambiguity Around Multi-Plugin Gateway Semantics:** Operators configuring multiple heterogeneous plugins on a single route (e.g., `key-auth` alongside `hmac-auth`) had no visual feedback in the Web Console explaining how the gateway evaluates multiple plugins at runtime—specifically, whether all credentials were required simultaneously (Logical AND) or if providing any valid credential granted access (Logical OR).

## Decision
1. **Modal Form Event Isolation:**
   - Decouple the plugin creation dialog in `AuthPluginManager` from native `<form>` submission semantics.
   - Convert the modal action button to `type="button"` with an explicit click handler that invokes `e.preventDefault()` and `e.stopPropagation()`.
   - Prevent any inner form submission event from propagating to outer `<form onSubmit={...}>` handlers in `RouteFormPage` and `ServiceFormPage`.
2. **Global Toast Temporal Window Deduplication:**
   - Implement an in-memory 1,000ms window debounce in `ToastProvider` (`web/src/components/Toast.tsx`).
   - If a toast notification with the exact same `(type, message)` key is dispatched within 1 second of an existing toast, the subsequent notification is suppressed.
   - Differing message contents or toast types continue to dispatch immediately without delay.
3. **Visual Multi-Plugin Logical OR Indicator:**
   - When $\ge 2$ active (non-deleted) authentication plugins are attached or staged on an entity, `AuthPluginManager` renders an informational banner directly above the plugin list:
     > **Multi-Plugin Logical OR:** Requests are granted access if any attached plugin successfully authenticates the client (first match wins).
   - When fewer than 2 active plugins exist, the banner is automatically hidden to keep the UI clean and minimalist.
   - Reaffirm Data Plane engine architecture (`internal/core/engine.go`): multi-plugin evaluation follows Logical OR semantics, granting access on the first matching valid credential, while preserving duplicate-type prevention (a route cannot attach two of the same plugin type).

## Consequences
- **Positive:**
  - Operators can freely stage and configure authentication plugins without triggering premature parent form validations or displaying confusing route path errors.
  - User feedback is clean and professional; rapid double-clicks or multiple event triggers do not generate stacked duplicate toast notifications.
  - Security engineers and operators gain immediate clarity on gateway authorization behavior when combining multiple authentication methods (e.g., public API key vs. internal HMAC signature).
- **Trade-off:**
  - Inner dialogs within form trees must be explicitly guarded with `e.stopPropagation()` and `type="button"` to avoid accidental bubbling in React's synthetic event model.
