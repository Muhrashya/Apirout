# 21: Purge Admin Credentials from Runtime and Standardize Discrete Datastore Config

**What to build:** Formally remove `AdminEmail` and `AdminPassword` from the runtime `Config` struct and delete the startup seeding logic in `internal/console/app/app.go`, establishing the web console Setup Wizard as the sole onboarding mechanism for fresh instances. Standardize database configuration exclusively on discrete parameters (`DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`, `DB_SSLMODE`).

**Blocked by:** None (can start immediately)

**Status:** closed

- [x] Remove `AdminEmail` and `AdminPassword` fields from `Config` in `internal/config/config.go` and remove associated environment variable parsing.
- [x] Remove `EnsureSuperAdmin` boot call in `internal/console/app/app.go`, ensuring fresh instances always start in the Initial Setup State.
- [x] Update console registry startup logging in `app.go` to report database connection status without referencing admin email.
- [x] Ensure manual CLI command `apirout create-admin` in `cmd/apirout/main.go` remains intact for out-of-band administration.
- [x] Verify test suite and compile step pass cleanly with fields removed.
