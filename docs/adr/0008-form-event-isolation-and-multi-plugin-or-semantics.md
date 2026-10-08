# ADR 0008: Form Event Isolation, Toast Temporal Deduplication, and Multi-Plugin Logical OR Semantics

## Status
Accepted

## Context
1. **Modal Event Bubbling into Parent Forms:** In the Apirout Web Console, `AuthPluginManager` provides an embedded modal dialog allowing operators to attach or stage authentication plugins within entity configuration pages (`RouteFormPage` and `ServiceFormPage`). Because React synthetic events bubble through the DOM hierarchy to parent `<form>` elements, clicking the modal's primary action button triggered the outer route form's submit listener prematurely. This resulted in unexpected validation errors (e.g., *"Match Path must start with /api/ (e.g. /api/v1/resource)"*) being surfaced to the operator before they had finished authoring the route.
2. **Duplicate Toast Notification Spam:** Rapid successive button clicks or dual-triggered event dispatches produced identical stacked toast notifications (such as two identical *"✓ Staged hmac-auth (pending save)"* pills) due to the lack of temporal deduplication in `ToastProvider`.
3. **Ambiguity Around Multi-Plugin Gateway Semantics:** Operators configuring multiple heterogeneous plugins on a single route (e.g., `key-auth` alongside `hmac-auth`) had no visual feedback in the Web Console explaining how the gateway evaluates multiple plugins at runtime—specifically, whether all credentials were required simultaneously (Logical AND) or if providing any valid credential granted access (Logical OR).
4. **Repeated Staging of Duplicate Plugin Types:** When all available plugins were attached or when an operator repeatedly clicked "Stage Plugin", the modal allowed duplicate instances of the same plugin type (e.g. 4 `key-auth` plugins) to be staged on a single route due to missing disabled states on modal action buttons and lack of programmatic duplicate rejection in the staging handler.

## Decision
1. **Modal Form Event Isolation:**
   - Decouple the plugin creation dialog in `AuthPluginManager` from native `<form>` submission semantics.
   - Convert the modal action button to `type="button"` with an explicit click handler that invokes `e.preventDefault()` and `e.stopPropagation()`.
   - Prevent any inner form submission event from propagating to outer `<form onSubmit={...}>` handlers in `RouteFormPage` and `ServiceFormPage`.
2. **Global Toast Temporal Window Deduplication:**
   - Implement an in-memory 1,000ms window debounce in `ToastProvider` (`web/src/components/Toast.tsx`).
   - If a toast notification with the exact same `(type, message)` key is dispatched within 1 second of an existing toast, the subsequent notification is suppressed.
   - Differing message contents or toast types continue to dispatch immediately without delay.
3. **Visual Multi-Plugin Logical OR Indicator & Security Guidance:**
   - When $\ge 2$ active (non-deleted) authentication plugins are attached or staged on an entity, `AuthPluginManager` renders an informational banner directly above the plugin list:
     > **Multi-Plugin Logical OR:** Requests are granted access if any attached plugin successfully authenticates the client (first match wins).
   - When $\ge 3$ distinct plugins are active, the banner appends an operational security tip:
     > **Security Tip:** Attaching 3 or more authentication mechanisms widens credential attack surface. Ensure all allowed methods meet your security policy.
   - When fewer than 2 active plugins exist, the banner is automatically hidden to keep the UI clean and minimalist.
   - Reaffirm Data Plane engine architecture (`internal/core/engine.go`): multi-plugin evaluation follows Logical OR semantics, granting access on the first matching valid credential, while preserving duplicate-type prevention (a route cannot attach two of the same plugin type).
4. **Three-Layer Duplicate Plugin Guardrails:**
   - **Header Level:** When all available plugin types (`key-auth`, `basic-auth`, `jwt-auth`, `hmac-auth`) are already attached or staged, disable the "Attach Plugin" header button, update its label to "All Plugins Attached", and apply a disabled cursor with tooltip.
   - **Modal Level:** Auto-select the first unattached plugin when opening the modal. In the plugin type selection grid, mark attached plugins as "Already Attached" and prevent selection. The primary action button ("Stage Plugin" / "Attach Plugin") is strictly disabled if the selected plugin is already attached.
   - **Handler Level:** Programmatically guard `handleCreatePlugin`: if an active plugin with the same type already exists, abort execution immediately with an error toast, guaranteeing state integrity even against unexpected DOM manipulation.

## Consequences
- **Positive:**
  - Operators can freely stage and configure authentication plugins without triggering premature parent form validations or displaying confusing route path errors.
  - User feedback is clean and professional; rapid double-clicks or multiple event triggers do not generate stacked duplicate toast notifications.
  - Impossible to spawn duplicate instances of the same plugin type in either the UI or internal state.
  - Clear visual distinction between valid multi-persona Logical OR combinations and unnecessary security attack surface widening.
- **Trade-off:**
  - Inner dialogs within form trees must be explicitly guarded with `e.stopPropagation()` and `type="button"` to avoid accidental bubbling in React's synthetic event model.
