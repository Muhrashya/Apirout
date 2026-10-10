# 72: SQL Query Safety & Injection Evasion Hardening

**What to build:** Comprehensive query validation in both console and gateway executors that prevents database file-write, file-read, and denial-of-service primitives (such as INTO OUTFILE, LOAD_FILE, PG_READ_BINARY_FILE, and SLEEP) from executing.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] ValidateSQL in internal/console/executor/validator.go and internal/gateway/executor/validator.go blocks INTO OUTFILE, INTO DUMPFILE, LOAD_FILE, PG_READ_BINARY_FILE, LO_EXPORT, LO_IMPORT, SLEEP, BENCHMARK
- [x] Query validator unit tests assert that queries attempting file export or sleep delays are rejected with HTTP 400
- [x] Standard SELECT queries and CTEs continue to pass validation cleanly
