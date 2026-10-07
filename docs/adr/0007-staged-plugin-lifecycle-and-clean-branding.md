# ADR 0007: Staged Plugin Lifecycle, Clean Branding, and Dead Code Elimination

## Status
Accepted

## Context
1. **Third-Party Product Wording:** Several UI views in the Web Admin Console (`AuthPluginManager.tsx`, `Upstreams.tsx`, `Services.tsx`, `Routes.tsx`, `Consumers.tsx`) referenced "Kong Gateway" and "Kong-aligned" terminology, undermining Apirout's identity as an independent, enterprise-grade open-source API engine.
2. **Immediate Inline Mutations vs Form Semantics:** When configuring routes and services, adding an authentication plugin via `AuthPluginManager` immediately sent HTTP POST/DELETE mutations to the control plane, updating the live database. However, this sub-component was embedded inside `RouteFormPage` and `ServiceFormPage` which featured an overarching "Update Route" / "Create Route" submit button. This mismatch caused user confusion: operators believed their plugins were not active until clicking "Update Route", while canceling or leaving the form unexpectedly left live plugin modifications on the Data Plane.
3. **Duplicate Plugin Attachments:** Operators were permitted to attach multiple instances of the exact same plugin (e.g. multiple `basic-auth` instances) to the same route, leading to redundant evaluations and clutter.
4. **Orphan Frontend Files:** `web/src/pages/AuditLogs.tsx` and `web/src/pages/Docs.tsx` remained in the repository as unused legacy files not imported by `App.tsx`.

## Decision
1. **Purge Third-Party Brand References:** Remove all references to "Kong" in user-facing console views, establishing clean "Apirout Gateway" terminology across all descriptions and headers.
2. **Adopt Staged Plugin Pipeline:**
   - Plugin additions, removals, and toggle state changes within entity forms (`RouteFormPage`, `ServiceFormPage`) are staged in local component/form state as drafts (marked with `PENDING SAVE / DRAFT` indicators).
   - Live mutations to the control plane and Data Plane are deferred until the operator explicitly clicks the primary "Create" or "Update" button on the parent form.
   - Closing or navigating away from the form discards pending drafts without affecting live gateway traffic.
   - Both Route Creation and Route Edit modes support staging multiple heterogeneous plugins simultaneously.
3. **Enforce Unique Plugin Types per Entity:**
   - In the "Attach Plugin" modal, plugin types already attached to the current entity are marked as "Already Attached" and disabled, preventing duplicate instances while fully preserving multi-credential Logical OR combinations (e.g. `key-auth` + `jwt-auth`).
4. **Purge Dead Code:**
   - Permanently delete orphan files `web/src/pages/AuditLogs.tsx` and `web/src/pages/Docs.tsx`.

## Consequences
- **Positive:** Intuitive form mental model where all configuration changes commit together. Zero risk of accidental live gateway changes when editing routes. Clean, professional branding without external vendor references. Cleaner repository with zero orphan files.
- **Trade-off:** The parent form must manage and submit the diff of staged plugins (creating newly added, updating toggled, deleting removed plugins) upon submission.
