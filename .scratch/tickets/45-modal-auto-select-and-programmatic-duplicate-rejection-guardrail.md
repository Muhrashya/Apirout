# 45: Modal Auto-Select and Programmatic Duplicate Rejection Guardrail

**What to build:** Eliminates the ability to stage duplicate plugins inside the modal dialog. When the modal opens, it automatically pre-selects the first unattached plugin type. In the selection grid, attached plugins cannot be selected. The modal's primary action button (`Stage Plugin` / `Attach Plugin`) is strictly disabled (`disabled={isAttached}`) whenever the highlighted plugin type is already active. Additionally, an authoritative programmatic check in the creation/staging handler aborts and triggers an error toast if an active plugin of the same type already exists.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] `openAddModal()` calculates the first unattached plugin type and sets it as the active selection
- [x] Clicking an attached plugin in the modal type grid is blocked / non-selectable
- [x] Modal action button (`Stage Plugin` / `Attach Plugin`) is disabled when the selected plugin type is already attached
- [x] `handleCreatePlugin` programmatically verifies no active plugin of that type exists, rejecting duplicates with an error toast
- [x] Typechecks cleanly with `npm run build` in `web/`
