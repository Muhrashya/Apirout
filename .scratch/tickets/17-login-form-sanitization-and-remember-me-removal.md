# 17: Login Form Sanitization and Remember Me Removal

**What to build:** Sanitize the management console login form by eliminating hardcoded credentials in component state, completely removing the "Remember me" checkbox, state, and styling, and establishing clean, non-confusing placeholder guidance so fields initialize blank without credential ghosting.

**Blocked by:** None (can start immediately)

**Status:** closed

- [x] Component state in `Login.tsx` initializes `email` and `password` to empty strings `""` rather than prefilled default values.
- [x] The "Remember me" checkbox, toggle state, associated label, and CSS styling are completely eliminated from `Login.tsx` and `Login.css`.
- [x] Form input placeholders provide clean semantic guidance (`name@company.com` and `••••••••` / `Enter your password`) without holding actual credentials.
- [x] Submitting the form continues to authenticate successfully against `/api/v1/auth/login` and transitions to `/dashboard`.
