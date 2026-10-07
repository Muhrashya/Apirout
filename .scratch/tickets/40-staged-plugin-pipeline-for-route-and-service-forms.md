# 40: Staged Plugin Pipeline for Route and Service Forms

**What to build:** Refactors plugin management inside `RouteFormPage` and `ServiceFormPage` to follow a staged draft pipeline. Attaching, modifying, or removing plugins in the form updates local draft state with visual `PENDING SAVE / DRAFT` badges. Live changes to the database and Data Plane only commit when the operator explicitly clicks the primary "Update Route" or "Create Route" button. Navigating away discards uncommitted drafts without touching live gateway traffic. Supports staging multiple plugins during route creation from scratch.

**Blocked by:** 38, 39

**Status:** completed

- [x] `AuthPluginManager` supports staged mode: accepts staged plugins array and callbacks without directly calling backend APIs
- [x] Staged plugins display a clear `PENDING SAVE / DRAFT` badge
- [x] `RouteFormPage` holds draft plugins in local state and synchronizes them (inserts/updates/deletes) upon clicking "Update Route" or "Create Route"
- [x] Route creation (`/routes/new`) allows staging and attaching multiple plugins
- [x] `ServiceFormPage` similarly adopts staged plugin draft pipeline
- [x] Closing or navigating away from form leaves live gateway configuration unchanged
