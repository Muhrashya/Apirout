# Spec: Standardize Git Branching Strategy, Semantic Versioning, and Release Tagging Baseline

**Triage Label:** `ready-for-agent`

## Problem Statement

The repository currently operates without formal branching guidelines or Git release tags on GitHub:
1. The GitHub repository (`Muhrashya/Apirout`) has zero release tags (`git tag -l` is empty, and the GitHub Tags/Releases tab displays "Nothing to show"), preventing operators and package consumers from referencing reproducible, immutable releases.
2. Versioning declarations are manually hardcoded and duplicated across disparate files (`cmd/apirout/main.go`, `web/package.json`, and `Makefile`), creating risks of version drift where binaries report obsolete versions.
3. There is no formalized branching model or release automation target to validate that working trees are clean, test suites pass, and tags are pushed consistently to origin.

## Solution

Establish a clear Trunk-Based (GitHub Flow) branching lifecycle, create the initial `v1.0.0` baseline release tag, and standardize Semantic Versioning across all components:
1. Formalize Trunk-Based development where `main` is the primary production trunk and all changes land via short-lived, validated feature/bugfix branches.
2. Baseline the repository by cutting and pushing annotated Git tag `v1.0.0` on the current tested commit milestone.
3. Standardize Semantic Versioning (`vMAJOR.MINOR.PATCH`) with build-time version injection via Go `ldflags` and Docker build arguments.
4. Implement a standardized release workflow script and make targets (`make release`) to automate pre-flight validation (test suite execution, clean git working tree), version synchronization, annotated tagging, and remote pushing.

## User Stories

1. As a platform operator, I want immutable Git release tags (e.g., `v1.0.0`) in GitHub, so that I can trace exactly which source commit produced the binary running in my environment.
2. As a DevOps engineer, I want Docker images to be built and tagged with matching SemVer tags (e.g., `apirout:v1.0.0` and `apirout:latest`), so that Kubernetes and Compose deployments can pin to deterministic images.
3. As a developer executing `apirout version` or `apirout -v`, I want the CLI to report the exact version injected at build time, so that I can verify the active build during deployment and debugging.
4. As a repository maintainer, I want all day-to-day development to merge into `main` via short-lived branches, so that we eliminate long-lived branch drift and complex merge conflicts.
5. As an open-source contributor, I want documented branch naming and contribution conventions, so that I know how to structure my branches and target pull requests to `main`.
6. As a release engineer, I want a single command (`make release VERSION=v1.0.0`) that validates all unit tests and working tree status before tagging, so that releases cannot be accidentally cut from dirty or broken code.
7. As a security auditor, I want release tags to be annotated with release notes and author metadata, so that the provenance of release artifacts can be verified.
8. As a Go developer, I want module consumers to be able to import or install `github.com/apirout/apirout` using standard SemVer tags, so that dependency resolution adheres to Go module conventions.
9. As a frontend maintainer, I want the web console `package.json` version to be automatically kept in sync with the repository version, so that npm package manifests never display obsolete version metadata.
10. As a site reliability engineer, I want the GitHub Releases page to display clear release notes, changelog highlights, and tagged source trees, so that platform upgrades are transparent and predictable.

## Implementation Decisions

### 1. Branching Model (Trunk-Based / GitHub Flow)
- The default branch `main` is the single production and continuous-integration trunk.
- Feature and bugfix work will occur on short-lived branches using descriptive prefixes:
  - `feat/<topic>` for new gateway or console capabilities.
  - `fix/<topic>` for bug fixes and security patches.
  - `docs/<topic>` for documentation and agent guides.
  - `chore/<topic>` for dependency updates and build maintenance.
- Feature branches are merged into `main` via pull requests and deleted upon merge.

### 2. Semantic Versioning Specification
- All release tags adhere strictly to Semantic Versioning 2.0 with a `v` prefix: `vMAJOR.MINOR.PATCH` (e.g. `v1.0.0`, `v1.0.1`, `v1.1.0`).
- Pre-release candidates use hyphenated suffixes (e.g. `v1.1.0-rc.1`).
- The initial baseline tag for the repository is established as `v1.0.0` on the current verified HEAD commit containing the complete Kong-aligned architecture, RBAC, auth plugins, Instant CRUD, and telemetry unification.

### 3. Build-Time Version Injection
- The application runtime version will be injected at compile time via Go linker flags (`-ldflags="-X main.version=${VERSION}"`), ensuring that git tags are the authoritative source of truth.
- The default fallback version in source code will serve as a fallback for local `go run` invocations without flags (e.g. `"1.0.0-dev"`).
- Docker multi-stage builds will accept an optional `ARG VERSION=1.0.0` and propagate it through the Go builder stage.
- The web console build will accept a `VITE_APP_VERSION` environment variable during Vite production builds to surface the release version in UI branding and console footers.

### 4. Release Automation Tooling
- A dedicated make target (`make release VERSION=vX.Y.Z`) and helper script will be added to orchestrate releases:
  - Pre-flight verification: ensuring working tree is clean (`git diff --quiet`) and tests pass (`go test ./...` and `npm run build`).
  - Synchronization: updating version fields in `web/package.json` and local manifests to match the release version.
  - Annotated Tagging: creating an annotated Git tag (`git tag -a vX.Y.Z -m "Release vX.Y.Z"`) with commit summary highlights.
  - Remote Synchronization: pushing the release tag and updated branch commits to `origin`.

### 5. Documentation
- Document the branching strategy, release process, and versioning rules in a new guide (`docs/BRANCHING_AND_RELEASING.md`).
- Update domain terminology in `GLOSSARY.md` to define canonical terms: `Release Tag`, `Trunk Branch`, `Semantic Versioning`, and `Build-Time Version Injection`.

## Testing Decisions

### Test Seams
1. **Application Version Flag Seam (Highest Runtime Seam)**: Execute the compiled binary with `version`, `-v`, and `--version` flags and verify that the output string matches the injected SemVer value.
2. **Build Automation Seam**: Execute `make build VERSION=v1.0.0` and `make docker-build VERSION=v1.0.0`, verifying that binaries and Docker images are created with expected version tags without errors.
3. **Release Pre-Flight Seam**: Run release verification logic under two conditions:
   - Dirty working tree (must fail and prevent tag creation).
   - Failing tests (must fail and abort release).
   - Clean tree and passing tests (must successfully create annotated tag).
4. **Git Remote Seam**: Verify `git describe --tags` reports the newly created tag and that `git ls-remote --tags origin` confirms synchronization on GitHub.

### Prior Art
- Unit and integration tests in the existing codebase test CLI flags and server initialization. The new test cases will leverage standard Go tests and PowerShell/bash verification scripts.

## Out of Scope

- Multi-year Long Term Support (LTS) maintenance branches (e.g. `release/v0.x`).
- Automated publishing to external OS package managers (Homebrew, Chocolatey, APT, AUR).
- Cloud deployment and continuous delivery orchestration (ArgoCD, Kubernetes Helm releases).

## Further Notes

- Target GitHub repository is `Muhrashya/Apirout`. Authentication credentials and permissions are verified and active.
