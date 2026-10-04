# Branching and Releasing Guide

This document defines the branching strategy, versioning rules, and release procedures for the Apirout project.

---

## 1. Branching Strategy (Trunk-Based / GitHub Flow)

Apirout standardizes on **Trunk-Based Development** with short-lived feature branches:

- **`main`**: The authoritative, deployable mainline branch. All production builds and release tags originate from `main`.
- **Feature & Fix Branches**: All development work occurs in dedicated, short-lived branches created from `main`:
  - `feat/<short-description>`: New gateway features, plugins, or console capabilities.
  - `fix/<short-description>`: Bug fixes, patch repairs, and regression resolutions.
  - `docs/<short-description>`: Documentation, ADRs, glossary updates, and agent guides.
  - `chore/<short-description>`: Build tooling, dependencies, and configuration refactoring.

### Pull Request & Merge Workflow
1. Create a branch: `git checkout -b feat/my-feature`
2. Implement with test-driven development (`go test ./...` and `cd web && npm run build`).
3. Commit with Conventional Commits (e.g., `feat(gateway): add custom plugin support`).
4. Push and open a Pull Request targeting `main`.
5. Upon review and passing CI tests, squash-and-merge or rebase-merge into `main`.
6. Delete the feature branch immediately after merging.

---

## 2. Semantic Versioning Specification

Apirout follows [Semantic Versioning 2.0.0](https://semver.org/) with a mandatory `v` prefix:

$$\text{Format: } \mathbf{vMAJOR.MINOR.PATCH}$$

- **MAJOR (`v1.0.0` $\rightarrow$ `v2.0.0`)**: Incompatible API schema changes, removal of deprecated plugins, or breaking configuration rewrites.
- **MINOR (`v1.0.0` $\rightarrow$ `v1.1.0`)**: Backwards-compatible new functionality (e.g., adding a new authentication plugin, database driver, or observability view).
- **PATCH (`v1.0.0` $\rightarrow$ `v1.0.1`)**: Backwards-compatible bug fixes, security hardening, or performance optimizations.
- **Pre-releases (`v1.1.0-rc.1`)**: Release candidates for staging validation.

---

## 3. Build-Time Version Injection

Version strings are **not** hardcoded in production binaries. Instead, Git tags are the single source of truth and are injected at compile time:

```bash
go build -ldflags="-s -w -X main.version=v1.0.0" -o bin/apirout ./cmd/apirout
```

When built locally without linker flags, the application defaults to `1.0.0-dev`.

### Verifying Build Version
```bash
./bin/apirout version
# Output: Apirout version v1.0.0 (Open Source Instant API Engine & Gateway)
```

---

## 4. How to Cut a Release

Apirout provides automated release tooling via PowerShell and Make:

### Automated Release Command
```bash
# In PowerShell:
.\scripts\release.ps1 -Version v1.0.0 -Push

# Or via Makefile:
make release VERSION=v1.0.0
```

### Pre-Flight Verification Performed Automatically
1. **Clean Tree Verification**: Ensures no untracked or uncommitted changes exist.
2. **Go Test Suite**: Executes `go test -count=1 ./...` across all packages.
3. **Web Console Build**: Executes `npm run build` in `web/` to guarantee TypeScript correctness.
4. **Manifest Synchronization**: Updates `web/package.json` with the new SemVer version.
5. **Annotated Tagging**: Creates `git tag -a vX.Y.Z -m "Release vX.Y.Z"`.
6. **Push to GitHub**: Pushes the branch and tag to `origin`.
