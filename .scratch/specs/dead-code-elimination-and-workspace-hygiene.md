# Specification: Dead Code Elimination and Workspace Hygiene

**Status:** `completed`

## Problem Statement

As the Apirout platform has grown through rapid architectural maturation, several classes of residual code, dead files, redundant shims, and outdated documentation have accumulated across the repository:

1. **Frontend Dead Files and Indirection Shims:**
   - The file `web/src/pages/Endpoints.tsx` is completely unreferenced by the application router or any other component.
   - The files `web/src/pages/Dashboard.tsx` and `web/src/pages/settings/ApiClients.tsx` are trivial 4-line re-export and navigation shims. Having router definitions point to shim files rather than the target components introduces unnecessary indirection and bundle fragmentation.
2. **Test Directory Duplication & Obsolete Testing Documentation:**
   - The repository contains an obsolete root `test/` directory whose only file is an outdated Indonesian README (`test/README.md`) pointing to non-existent directories (`./test/e2e/...`), while the canonical end-to-end test suites live in `tests/e2e/`. This duality creates developer confusion regarding where to find and run test suites.
3. **Outdated Planning Artifacts & Brand Identity Violations:**
   - The document `docs/PLAN_PEMISAHAN_MULTIPLANES.md` is an obsolete pre-implementation Indonesian draft referencing third-party Kong Gateway patterns, directly violating the repository's brand autonomy standard codified in ADR-0007.
   - A broken markdown link in `README.md` points to this obsolete planning document.
   - The root `Makefile` retains a legacy brand reference in its help banner, conflicting with ADR-0007.
4. **Untracked Workspace Build Residue:**
   - A 23.8 MB pre-compiled Windows executable (`bin/apirout.exe`) resides in the workspace root `bin/` directory. While `*.exe` is ignored by `.gitignore`, the `bin/` build directory itself is not explicitly ignored, leaving the potential for untracked multi-platform binary residue to clutter developer environments.

## Solution

Perform a comprehensive repository cleanup to eliminate dead files, flatten frontend routing shims, consolidate test documentation, adhere strictly to ADR-0007 brand standards, and enforce workspace build hygiene:

1. **Frontend Dead Code Purge & Routing Flattening:**
   - Delete `web/src/pages/Endpoints.tsx`.
   - Update `web/src/App.tsx` to lazy-load and mount `Overview` directly for `/overview`, and delete `web/src/pages/Dashboard.tsx`.
   - Update `web/src/App.tsx` to redirect `/settings/api-clients` directly to `/consumers` via router navigation, and delete `web/src/pages/settings/ApiClients.tsx`.
2. **Test Directory Layout Consolidation:**
   - Remove the obsolete root `test/` directory.
   - Create a clean, authoritative English `tests/README.md` describing the end-to-end test suite structure, prerequisite environments, and command-line execution (`go test -v ./tests/e2e/...`).
3. **Documentation Hygiene & Brand Alignment (ADR-0007):**
   - Delete `docs/PLAN_PEMISAHAN_MULTIPLANES.md`.
   - Update the "Documentation & Guides" section of `README.md` to remove the dead link.
   - Update `Makefile` to align its help description with Apirout's standalone identity.
4. **Build Artifact Elimination & Git Hygiene:**
   - Purge `bin/apirout.exe` from disk.
   - Update `.gitignore` to explicitly ignore the `bin/` directory across all platforms.

## User Stories

1. As a frontend developer exploring the `web/src/pages` directory, I want all page files to represent active, functional screens rather than dead unreferenced code, so that I do not spend time reading or maintaining obsolete files.
2. As a frontend developer navigating route configurations in `web/src/App.tsx`, I want the `/overview` route to resolve directly to the `Overview` page component without an intermediary `Dashboard` shim, so that component hierarchy and code splitting are clear and direct.
3. As an operator or API consumer navigating to `/settings/api-clients`, I want to be redirected cleanly to the `/consumers` route without mounting a redundant intermediate wrapper component, so that route handling is lightweight and fast.
4. As an open-source contributor inspecting the codebase, I want an unambiguous test directory structure under `tests/` rather than confusing duplicate `test/` and `tests/` folders, so that I know immediately where to locate test files.
5. As an open-source contributor running integration tests, I want a clear, comprehensive English `tests/README.md` explaining how to execute tests and verify container health, so that I can validate changes locally with confidence.
6. As a platform architect reviewing documentation, I want all legacy planning documents that mention third-party architectures to be purged, so that documentation strictly complies with ADR-0007.
7. As a new developer reading `README.md`, I want all documentation hyperlinks to resolve to valid, active markdown specifications, so that I encounter no 404 broken links.
8. As a DevOps engineer running `make help`, I want the output to accurately reflect Apirout's identity as a self-hosted instant API engine and gateway without third-party references, so that CLI messaging is consistent with project documentation.
9. As a developer cloning and checking out the repository, I want no large pre-compiled binary files occupying disk space or polluting workspace searches, so that the repository remains lean and fast to clone.
10. As a developer running local `make build` or `go build` commands targeting `bin/`, I want the `.gitignore` file to ensure the entire `bin/` directory is ignored across all operating systems and architectures, so that build artifacts are never inadvertently committed.
11. As a QA engineer running automated end-to-end suites, I want all 18 existing test suites to pass without failure after dead code removal, ensuring zero regressions.
12. As a release engineer building production container bundles, I want `npm run build` in `web/` to compile cleanly without missing module errors or broken lazy imports.

## Implementation Decisions

- **Frontend Component De-Shimization:**
  - Route `/overview` directly references `web/src/pages/Overview.tsx` via `lazy(() => import('./pages/Overview'))`.
  - Route `/settings/api-clients` uses inline router navigation `<Navigate to="/consumers" replace />` directly within the `SettingsGuard` route hierarchy in `App.tsx`.
  - The legacy path `/endpoints` retains its existing `<Navigate to="/routes" replace />` redirect in `App.tsx`, ensuring backward-compatible URL handling without needing any file in `web/src/pages/Endpoints.tsx`.
- **Test Directory Standardization:**
  - The root `test/` directory is eradicated completely.
  - The authoritative testing documentation is established at `tests/README.md` in English, detailing environment prerequisites (Docker containers for postgres, redis, and apirout) and test commands.
- **Brand Identity and Documentation Integrity:**
  - Removal of `docs/PLAN_PEMISAHAN_MULTIPLANES.md` eliminates legacy language and external brand references.
  - `README.md` documentation list is updated to reference only active architectural specifications.
  - `Makefile` help string is updated to: `Apirout - Open Source Instant API Engine & Gateway Architecture`.
- **Git and Workspace Build Hygiene:**
  - `.gitignore` is updated to include `bin/`.
  - The local binary file `bin/apirout.exe` is deleted.

## Testing Decisions

- **Testing Seams:**
  - **Frontend Compilation Seam:** Execute `npm run build` within `web/` to verify that TypeScript compilation and Vite bundling succeed with zero errors and no unresolved import paths.
  - **Web Routing Seam:** Verify that navigating to `/overview`, `/dashboard`, `/endpoints`, and `/settings/api-clients` yields the expected target screens or redirects.
  - **Integration & E2E Testing Seam:** Execute the complete end-to-end test suite via `go test -v ./tests/e2e/...` against the live running container stack to ensure all 18 feature test suites pass without regression.
  - **Repository & Git Status Seam:** Verify that `git status` shows a completely clean working tree after files are removed and build artifacts are ignored.

## Out of Scope

- Modifying runtime backend API routes, database schemas, or gateway proxying logic.
- Adding new UI features or redesigning existing web console screens.
- Modifying E2E test scenario assertions or internal Go packages.

## Further Notes

This cleanup reinforces the architectural standards established in ADR-0007 (Independent Platform Brand Identity and Clean Multi-Plane Topology) by eliminating historical residue, reducing repo bloat, and simplifying onboarding for future developers.
