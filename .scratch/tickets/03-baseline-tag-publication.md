# 03: Baseline Tag Publication & GitHub Release Sync (v1.0.0)

**Parent:** [#1](https://github.com/Muhrashya/Apirout/issues/1)
**Issue:** [#4](https://github.com/Muhrashya/Apirout/issues/4)
**Blocked by:** [#3](https://github.com/Muhrashya/Apirout/issues/3)
**Status:** ready-for-agent

## What to build
Create and push the official baseline annotated release tag `v1.0.0` to GitHub (`Muhrashya/Apirout`), officially tagging the verified Kong-aligned architecture, RBAC, auth plugins, Instant CRUD, and telemetry engine. Resolves the empty Tags/Releases UI on GitHub.

## Acceptance criteria
- [ ] Annotated tag `v1.0.0` is cut on the stable milestone commit.
- [ ] Tag `v1.0.0` is pushed to GitHub remote `origin`.
- [ ] Verification confirms `git describe --tags` reports `v1.0.0`.
- [ ] GitHub repository Tags UI displays the `v1.0.0` release tag cleanly.
