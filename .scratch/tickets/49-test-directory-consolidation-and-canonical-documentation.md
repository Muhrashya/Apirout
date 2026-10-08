# 49: Test Directory Consolidation and Canonical Documentation

**What to build:** Eliminates directory ambiguity by eradicating the obsolete root `test/` directory. Authors an authoritative, comprehensive English `tests/README.md` that documents the end-to-end testing suite architecture, prerequisite services (Postgres, Redis, Apirout), and standard command execution (`go test -v ./tests/e2e/...`).

**Blocked by:** 48

**Status:** completed

- [x] Obsolete root `test/` directory and `test/README.md` deleted
- [x] Authoritative English `tests/README.md` created with accurate prerequisites and command examples
- [x] All test suite references validated
