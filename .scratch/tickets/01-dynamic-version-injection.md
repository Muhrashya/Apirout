# 01: Dynamic Build-Time Version Injection & CLI Version Flag Verification

**Parent:** [#1](https://github.com/Muhrashya/Apirout/issues/1)
**Issue:** [#2](https://github.com/Muhrashya/Apirout/issues/2)
**Blocked by:** None (can start immediately)
**Status:** ready-for-agent

## What to build
Decouple the application runtime version from hardcoded source strings. Compile the Go binary and Docker image with linker-injected versions, ensuring `apirout version`, `apirout -v`, and `apirout --version` report the authoritative injected SemVer. Add automated CLI version test coverage.

## Acceptance criteria
- [ ] The CLI entrypoint accepts dynamic version injection via linker flags during build and falls back to a development default when not injected.
- [ ] Running `apirout version`, `apirout -v`, and `apirout --version` prints the active version string cleanly.
- [ ] Build definitions propagate the version parameter into the binary.
- [ ] Automated tests verify the version command output format.
