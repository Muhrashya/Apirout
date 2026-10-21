# 88: Declarative Upstream Target Reconciliation on Edit

**What to build:**
Enable full two-way target synchronization when updating an existing upstream pool in the management console. Edits to server instances in the targets table (creating new targets, modifying ports/weights, or removing targets) must be diffed against backend targets and persisted reliably upon clicking "Update Upstream Pool".

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] In `web/src/pages/upstreams/UpstreamFormPage.tsx`, remove the `!isEdit` restriction preventing target syncing on edit.
- [x] Implement declarative target reconciliation: fetch existing targets on save, compute added, modified, and deleted targets.
- [x] Dispatch parallel API calls (`createTarget`, `updateTarget`, `deleteTarget`) and report unified status feedback to the operator.
- [x] Ensure frontend tests and type checks pass cleanly (`tsc && vite build`).

