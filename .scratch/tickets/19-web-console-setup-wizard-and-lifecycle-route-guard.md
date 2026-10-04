# 19: Web Console Setup Wizard and Lifecycle Route Guard

**What to build:** Build the Web Console first-time Setup Wizard page adopting the platform's visual design system, and implement application lifecycle routing guards that guide operators to setup on first boot and permanently lock out the setup page once configured.

**Blocked by:** 17: Login Form Sanitization and Remember Me Removal, 18: Initial Setup State Detection and Super Admin Provisioning API

**Status:** closed

- [x] New `Setup.tsx` page implementing the initial Super Admin onboarding wizard matching the login screen aesthetics (card elevation, typography, branding logo, inputs for email, password, confirm password, and responsive feedback).
- [x] Application lifecycle check queries `/api/v1/auth/setup-status` upon initial load.
- [x] If `setup_required: true`, navigating to `/` or `/login` redirects automatically to `/setup`.
- [x] If `setup_required: false`, navigating to `/setup` is permanently blocked and cleanly redirects to `/login`.
- [x] Submitting the setup wizard stores the returned JWT token in client storage, updates auth state, and transitions immediately to `/overview`.
- [x] Comprehensive verification across browser UI flow and automated E2E test suite.
