# 02: Automated Release Workflow & Version Synchronization Target

**Parent:** [#1](https://github.com/Muhrashya/Apirout/issues/1)
**Issue:** [#3](https://github.com/Muhrashya/Apirout/issues/3)
**Blocked by:** [#2](https://github.com/Muhrashya/Apirout/issues/2)
**Status:** ready-for-agent

## What to build
A release orchestration command that performs pre-flight verification (ensures git working tree is clean and all test suites pass), synchronizes version numbers across repository manifests, cuts an annotated Git release tag, and validates tag formation.

## Acceptance criteria
- [ ] A dedicated release command or script validates that git working tree is clean and aborts if uncommitted changes exist.
- [ ] Release workflow runs all unit tests and aborts if tests fail.
- [ ] Updates version fields in package manifests to match the target release version.
- [ ] Creates an annotated Git tag formatted as `vMAJOR.MINOR.PATCH` with commit summary metadata.
