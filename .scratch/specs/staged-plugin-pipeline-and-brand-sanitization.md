# Specification: Staged Plugin Pipeline, Clean Brand Identity, and Dead Code Elimination

**Status:** `ready-for-agent`

## Problem Statement

Operators using the Apirout Web Console face three friction points that reduce system reliability, create cognitive confusion, and diminish brand professionalism:

1. **Premature Live Gateway Mutation During Route Editing:** When configuring authentication plugins inside the Route or Service editing views, adding, toggling, or removing a plugin immediately fires direct HTTP mutations to the control plane, updating the live database and mutating Data Plane traffic instantaneously. However, because this component is visually embedded inside a larger form that has a primary "Update Route" button at the bottom, operators are trapped in an affordance mismatch: they mistakenly believe changes are not saved until the bottom button is clicked. Conversely, if an operator experiments with adding a plugin or cancels the page, live traffic has already been altered without their consent.
2. **Third-Party Brand Intrusion:** Multiple administrative views and descriptions across the Web Console explicitly reference "Kong Gateway" and "Kong-aligned" chains (e.g. *"Inbound client identity verification (Kong Gateway plugin chain)"*). This creates the impression of an unpolished clone rather than a standalone, enterprise-grade open-source API engine.
3. **Duplicate Plugin Clutter & Potential Misconfiguration:** The plugin selection modal permits operators to attach multiple instances of the exact same plugin type (e.g. two `basic-auth` instances) to the same route or service, leading to redundant database queries, confusing UI lists, and potential configuration conflicts.
4. **Orphan Legacy Code:** Unused legacy files (`AuditLogs.tsx` and `Docs.tsx`) linger in the frontend source tree without being referenced or routed in `App.tsx`, creating dead-code clutter and maintenance debt.

## Solution

1. **Staged Plugin Pipeline in Entity Forms:** Transform the plugin management lifecycle in `RouteFormPage` and `ServiceFormPage` from immediate inline execution to a staged draft pattern. Inbound plugins are drafted, configured, toggled, and removed in local component state with clear `PENDING SAVE / DRAFT` indicators. Only upon explicit manual click of the primary form button ("Update Route" or "Create Route") are the staged plugins synchronized to the database. If the operator navigates away or cancels, no live Data Plane changes occur.
2. **Unified Multi-Plugin Creation in Route Creation Mode:** Extend the staged plugin workflow to the route creation flow (`/routes/new`), allowing operators to stage and configure multiple heterogeneous plugins simultaneously before saving the new route.
3. **Unique Plugin Constraint per Scope:** Enforce a single instance per plugin type for a given entity scope. In the selection modal, plugin types already present on the entity are disabled with an "Already Attached" badge, while preserving the ability to attach different plugin types together for Logical OR multi-credential evaluation (`key-auth` + `jwt-auth`).
4. **Complete Brand Cleansing:** Purge all external third-party references ("Kong") from user-facing Web Console components and documentation, standardizing on pure, unified "Apirout Gateway" branding.
5. **Dead Code Elimination:** Permanently delete orphan frontend files (`AuditLogs.tsx` and `Docs.tsx`), validating that the production build remains completely green.

## User Stories

1. As an API platform operator, I want newly attached plugins in a route form to remain in a staged draft state, so that live gateway traffic is not prematurely mutated while I am still configuring the route.
2. As an API platform operator, I want to clearly see a "PENDING SAVE / DRAFT" indicator next to staged plugins in the form, so that I understand changes have not yet been committed to production.
3. As an API platform operator, I want all route configuration changes and staged plugin configurations to commit together when I click "Update Route", so that my route updates are atomic and predictable.
4. As an API platform operator, I want to discard pending plugin changes by simply canceling or navigating away from the edit page, so that accidental clicks do not disrupt production clients.
5. As an API platform operator, I want to stage multiple authentication plugins when creating a new route from scratch, so that newly published endpoints are immediately protected with complete authentication chains upon creation.
6. As an API platform operator, I want already attached plugin types to be disabled in the selection modal, so that I cannot inadvertently attach redundant duplicate instances of the same plugin to a single route.
7. As an API platform operator, I want to attach different plugin types (such as `key-auth` and `jwt-auth`) to the same route, so that the gateway can evaluate them with Logical OR and admit heterogeneous client applications.
8. As an API consumer, I want to authenticate against a multi-plugin route using either a valid API key or a valid JWT bearer token, so that different application clients can access the same endpoint seamlessly.
9. As an administrator reviewing the Web Console, I want all descriptions and headers to reference Apirout Gateway natively without mentioning third-party gateway brands, so that the administrative interface presents a cohesive, professional identity.
10. As a developer contributing to the codebase, I want all orphan and unreferenced page components removed, so that the project directory remains clean, maintainable, and free of dead code.

## Implementation Decisions

- **Domain Concept Registration:** The domain model incorporates the **Staged Plugin Pipeline**, formally defined in `GLOSSARY.md` and codified in ADR-0007 (`docs/adr/0007-staged-plugin-lifecycle-and-clean-branding.md`).
- **Form State Architecture:** 
  - The plugin manager component will operate in a controlled staged mode when embedded within entity forms (`RouteFormPage`, `ServiceFormPage`).
  - Instead of firing asynchronous API mutations directly on user action, it will emit staged changes to the parent form's state.
  - The parent form maintains a baseline list of existing plugins and a staged working list. Upon form submission, the parent calculates the synchronization diff (inserts, updates, deletes) and executes them as part of the save sequence.
- **Visual Staging Indicators:**
  - Plugins newly added during the session display a prominent `PENDING SAVE` badge.
  - Plugins marked for deletion display a strikethrough or `PENDING REMOVAL` state until the form is submitted or undone.
  - Disabled/enabled toggles update local state immediately with visual feedback indicating unsaved changes.
- **Selection Modal Guardrails:**
  - The plugin selection modal inspects the currently staged plugin list for the target entity.
  - If a plugin type (e.g. `basic-auth`) is already present in the active staged list, its card is visually disabled, marked with an "Already Attached" badge, and blocked from selection.
- **Console Brand Standardization:**
  - All occurrences of third-party gateway brand strings in headers, breadcrumbs, descriptions, and comments across console pages are rewritten to use canonical "Apirout Gateway" terminology.
- **Orphan File Removal:**
  - Delete `web/src/pages/AuditLogs.tsx` and `web/src/pages/Docs.tsx`.
  - Verify that no remaining imports in `App.tsx` or router trees reference these files.

## Testing Decisions

- **Highest-Seam HTTP Testing (Control Plane & Data Plane):**
  - **Live Gateway Authentication Suite (`:8000` & `:8001`):** Verify that routes configured with multiple staged plugins (e.g. `key-auth` + `basic-auth`) allow access when either valid API keys or valid basic credentials are provided (Logical OR verification), while rejecting requests with invalid or missing credentials with HTTP 401.
  - **Single Plugin Type Invariant:** Verify that attaching duplicate plugins of the same type to the same entity is blocked and rejected by the API.
  - **Atomic Save Verification:** Verify that updating a route without saving plugins leaves existing gateway behavior unchanged.
- **Frontend Build & Bundle Integrity Testing:**
  - Run `npm run build` inside `web/` to guarantee zero compilation, typechecking, or bundler errors following the deletion of orphan files and refactoring of the plugin staging pipeline.
- **Prior Art:**
  - Builds upon existing end-to-end integration patterns established in `tests/e2e/e2e_features_test.go` and `internal/gateway/auth/engine_test.go`.

## Out of Scope

- Implementing Logical AND chaining (requiring both an API key AND a JWT token on every request). Authentication continues to operate under the established and industry-standard Logical OR semantics.
- Introducing a background cron daemon for active upstream health checks. Upstream resilience remains governed by the high-performance Passive Circuit Breaker at the Data Plane response seam.
- Modifications to the core Data Plane proxy forwarding mechanics.

## Further Notes

- All changes adhere to Trunk-Based Development with zero breaking changes to existing gateway error contracts or backward-compatible hybrid envelopes.
