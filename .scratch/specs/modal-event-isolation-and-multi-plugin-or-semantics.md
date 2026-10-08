# Specification: Modal Event Isolation, Toast Deduplication, and Multi-Plugin Logical OR Semantics

**Status:** `completed`

## Problem Statement

When operators use the Apirout Web Console to configure authentication plugins within route and service forms (e.g. staging `hmac-auth` alongside an existing `key-auth` plugin), they encounter three user-facing defects that degrade usability and create cognitive friction:

1. **Premature Parent Route Validation via Event Bubbling:** When an operator clicks "Stage Plugin" inside the authentication plugin modal, the submit action bubbles through the React component tree into the parent entity form's submit listener (`RouteFormPage` / `ServiceFormPage`). This prematurely triggers overarching route validation, throwing an unexpected red error toast (such as *"Match Path must start with /api/ (e.g. /api/v1/resource)"*), despite the operator having only intended to stage an authentication plugin in the draft modal.
2. **Duplicate Toast Notification Spam:** The notification system lacks temporal deduplication. Fast clicks, re-renders, or bubbling events dispatch identical notifications in rapid succession, displaying duplicate green pills stacked on top of each other (e.g. two identical *"✓ Staged hmac-auth (pending save)"* toasts).
3. **Ambiguous Multi-Plugin Evaluation Semantics:** When multiple heterogeneous plugins are staged or attached to a single route (such as `key-auth` and `hmac-auth`), operators have no visual indication of whether the gateway enforces both credentials simultaneously (chained AND) or accepts either credential (Logical OR). This causes uncertainty during production deployment and security reviews.

## Solution

1. **Modal Form Event Isolation:** Decouple the plugin configuration modal from native form submission. Convert the modal action button to `type="button"` with an explicit click handler, and invoke event propagation stops (`stopPropagation()`), guaranteeing that modal interactions never bubble into the parent route or service form handlers.
2. **Toast Notification Deduplication:** Implement a 1,000ms window debounce deduplication layer within the global toast provider. If a notification with the identical type and message is triggered within 1 second of an existing instance, suppress the duplicate to ensure a clean, single-toast user experience.
3. **Multi-Plugin Logical OR Visual Indicator:** When two or more inbound authentication plugins are staged or attached to a route or service, render an informative visual banner directly above the plugin cards clarifying the runtime behavior: *"Multi-Plugin Logical OR: Requests are granted access if any attached plugin successfully authenticates the client."*

## User Stories

1. As an API platform operator, I want clicking "Stage Plugin" in the modal to only stage the plugin in local state, so that the parent route form is not prematurely submitted or validated.
2. As an API platform operator, I want to add an authentication plugin to a draft route without being interrupted by a premature "Match Path must start with /api/" error toast, so that I can configure my route and plugins in whatever order I prefer.
3. As an API platform operator, I want to see a single confirmation toast when staging or modifying a plugin, so that my screen is not cluttered by duplicate stacked notifications.
4. As an API platform operator, I want rapidly clicking action buttons in the console to never spawn multiple identical toasts, so that notification feedback remains clean, professional, and readable.
5. As an API platform operator configuring multiple plugins on a route, I want to see a clear explanation that the gateway uses Logical OR evaluation, so that I know clients with either an API key or an HMAC signature will be granted access.
6. As a security engineer reviewing route configurations in the Web Console, I want visual clarity on how multi-plugin authentication behaves at runtime, so that I can verify access control policies without needing to inspect backend source code.
7. As an API platform operator, I want the Logical OR indicator banner to automatically disappear when only a single plugin is attached, so that the interface remains minimalist and avoids redundant information.
8. As a developer navigating the codebase, I want an architectural record (ADR-0008) explaining the modal event isolation and multi-plugin UI contract, so that future UI modifications do not reintroduce event bubbling leaks.

## Implementation Decisions

- **Event Propagation Boundary:**
  - The modal dialog within the authentication plugin manager will no longer rely on a native `<form onSubmit={...}>` wrapper that bubbles synthetic React events to outer form elements.
  - The primary modal action button will use `type="button"` with a dedicated click handler that explicitly prevents default behavior and stops event propagation.
- **Toast Provider Deduplication Engine:**
  - The toast context provider will maintain an in-memory timestamp map tracking the most recent dispatch time for each unique `(type, message)` tuple.
  - Any call to dispatch a toast with an identical type and message within a 1,000ms threshold will be ignored.
  - Toasts with different messages or different types will continue to render immediately without delay.
- **Multi-Plugin Semantics Banner:**
  - The authentication plugin manager will inspect the active non-deleted plugin count (`currentPlugins.filter(p => p._stagedState !== 'deleted').length`).
  - When the active count is $\ge 2$, a styled informational banner will render above the plugin cards indicating Logical OR evaluation.
  - When the active count is $< 2$, the banner will remain hidden.
- **Architectural Documentation:**
  - Codify the decisions in `docs/adr/0008-form-event-isolation-and-multi-plugin-or-semantics.md`.

## Testing Decisions

- **Testing Seam:** High-seam frontend UI verification and Data Plane runtime alignment.
  - **Modal Isolation Verification:** Verify that submitting a new plugin within the modal does not trigger the route form's validation error toast.
  - **Toast Deduplication Verification:** Verify that invoking identical toast dispatches within 1 second produces exactly one visible notification pill in the toast container.
  - **Logical OR Banner Verification:** Verify that attaching two plugins renders the informational banner, and removing down to one plugin hides the banner.
  - **Runtime Consistency:** Ensure existing end-to-end tests (`tests/e2e/e2e_features_test.go` and `tests/e2e/e2e_deep_features_test.go`) validating multi-plugin Logical OR behavior on port `:8000` continue to pass 100%.

## Out of Scope

- Modifying the Go Data Plane authentication engine or changing the Logical OR evaluation logic (already established and verified).
- Altering the route data schema or database migrations for auth plugins.
- Replacing the custom React toast notification system with an external third-party library.

## Further Notes

- This specification addresses the direct observations from production usage where modal submission triggered parent form validation while simultaneously generating stacked duplicate toast notifications.
