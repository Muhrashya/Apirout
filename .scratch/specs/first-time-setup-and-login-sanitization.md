# Spec: First-Time Super Admin Setup Wizard and Login Sanitization

**Triage Label:** `ready-for-agent`

## Problem Statement

When deploying Apirout on a fresh installation without pre-configured environment credentials, operators encounter a sub-optimal onboarding experience:

1. **Missing First-Boot Web Onboarding**: If `ADMIN_EMAIL` and `ADMIN_PASSWORD` environment variables are omitted or empty, the database contains zero Super Admin accounts. Navigating to the management console presents a standard login form where login is impossible because no credentials exist, forcing operators to execute manual database queries or CLI commands (`apirout create-admin`) out-of-band.
2. **Setup State Access Exposure**: Without an explicit onboarding state machine and access guard, there is no standardized, secure way to guide the operator through initial root credential creation and ensure the setup interface becomes permanently inaccessible once configured.
3. **Hardcoded Login Defaults & Form Ghosting**: The management console login interface currently initializes form state with hardcoded placeholder values (`admin@apirout.io` and `AdminApirout2026!`) and displays a persistent "Remember me" option. When operators inspect or interact with the login screen, credentials appear prefilled or visually trapped in placeholder text even when "Remember me" was not selected, creating confusion and security concerns around credential retention.

## Solution

Implement an automated First-Time Setup Wizard with permanent lifecycle locking and sanitize the login page form:

1. **Initial Setup State Detection**:
   - Introduce a public, unauthenticated control plane discovery endpoint (`GET /api/v1/auth/setup-status`) that queries the registry to determine whether any account with the `SUPER_ADMIN` role exists.
   - If zero Super Admin accounts exist, `setup_required` is true; otherwise, false.
2. **First-Time Super Admin Setup Wizard**:
   - Provide a dedicated, beautifully styled Setup Wizard page sharing the exact visual design pattern, card elevation, typography, and branding tokens as the login interface.
   - Collect the initial Super Admin email and password (with confirmation and validation requiring at least 8 characters).
   - Upon submission, provision the root Super Admin account, automatically issue a JWT session token, and immediately transition the authenticated session to the main overview dashboard.
3. **Permanent Lifecycle Locking (409 Conflict & Route Guard)**:
   - The setup submission endpoint (`POST /api/v1/auth/setup`) strictly verifies that zero Super Admin accounts exist prior to insertion.
   - If one or more Super Admin accounts already exist, the endpoint immediately aborts with `409 Conflict` ("Setup has already been completed").
   - The web console router inspects setup status: if setup is completed, any attempt to navigate to `/setup` is automatically redirected to `/login`, making the setup page permanently inaccessible.
4. **Login Form Sanitization**:
   - Completely eliminate the "Remember me" checkbox, component state, and styling from the login form.
   - Initialize email and password input states to empty strings (`""`), preventing hardcoded credentials from displaying as values or placeholders.
   - Use clean, non-confusing placeholder guidance (`name@company.com` and `••••••••` / `Enter your password`).

## User Stories

1. As an operator deploying Apirout for the first time without environment credentials, I want to be automatically directed to a setup wizard, so that I can configure my root Super Admin account without CLI commands or manual database updates.
2. As a platform administrator, I want the setup wizard to follow the exact same visual design system and branding as the login screen, so that the onboarding experience feels cohesive and professional.
3. As a security engineer, I want the setup creation endpoint to reject any request with HTTP 409 Conflict once a Super Admin already exists, so that unauthorized users cannot re-trigger setup or overwrite credentials.
4. As an operator navigating to `/setup` on a fully configured instance, I want the application to automatically redirect me to `/login`, so that the setup wizard is completely inaccessible post-initialization.
5. As an infrastructure administrator deploying with `ADMIN_EMAIL` and `ADMIN_PASSWORD` in Docker Compose or Kubernetes, I want automated boot seeding to continue working seamlessly, so that headless deployments bypass the setup wizard automatically.
6. As a console user on the login screen, I want the email and password fields to start empty, so that no hardcoded credentials appear prefilled.
7. As a privacy-conscious administrator, I want the "Remember me" checkbox completely removed, so that credentials and persistent tokens are never stored unintentionally on shared workstations.
8. As a new Super Admin completing the setup form, I want password confirmation validation on the client and server, so that I cannot lock myself out with an accidental typo.
9. As a new Super Admin completing setup, I want to be automatically logged in and transitioned to the dashboard, so that I do not have to re-enter my credentials immediately after creating them.
10. As an automated monitoring probe or frontend router, I want a lightweight public status endpoint returning `{ "setup_required": boolean }`, so that the application state can be determined before rendering route trees.
11. As a platform auditor, I want initial Super Admin creation through the setup wizard to produce an immutable audit log record, so that initial cluster provisioning is documented.
12. As an API developer, I want all existing administrative REST APIs, token lifetimes, and RBAC rules to remain backwards-compatible, so that client integrations are uninterrupted.

## Implementation Decisions

1. **Initial Setup State Contract**:
   - The Control Plane exposes a public endpoint `GET /api/v1/auth/setup-status`.
   - Response payload:
     ```json
     {
       "setup_required": false,
       "has_super_admin": true
     }
     ```
   - Evaluated atomically via registry database query checking `COUNT(*) FROM users WHERE role = 'SUPER_ADMIN'`.
2. **First-Time Super Admin Registration Contract**:
   - The Control Plane exposes `POST /api/v1/auth/setup`.
   - Request payload:
     ```json
     {
       "email": "admin@example.com",
       "password": "StrongPassword123!",
       "confirm_password": "StrongPassword123!"
     }
     ```
   - Validations:
     - Check `COUNT(*) FROM users WHERE role = 'SUPER_ADMIN'`. If count > 0, return `409 Conflict` with `{"error": "Setup has already been completed"}`.
     - Validate email format with standard email regex.
     - Validate password length (minimum 8 characters).
     - Validate password matching `confirm_password`.
   - Result:
     - Hashes password using bcrypt.
     - Generates UUID user record with `role = 'SUPER_ADMIN'`.
     - Inserts record and logs `super_admin_setup_completed` audit event.
     - Generates JWT access token and returns standard authentication payload:
       ```json
       {
         "token": "<jwt-token>",
         "expires_at": "<timestamp>",
         "user": {
           "id": "<uuid>",
           "email": "admin@example.com",
           "role": "SUPER_ADMIN"
         }
       }
       ```
3. **Frontend Lifecycle Gating & Routing**:
   - An initial setup check hook queries `GET /api/v1/auth/setup-status` upon application load.
   - If `setup_required` is true:
     - Navigating to `/` or `/login` redirects to `/setup`.
   - If `setup_required` is false:
     - Navigating to `/setup` redirects immediately to `/login`.
4. **Login Page Sanitization**:
   - Remove `rememberMe` boolean state and associated HTML elements (`<label className="remember-me-label">...`).
   - Remove initial state strings `'admin@apirout.io'` and `'AdminApirout2026!'`; state variables initialize as empty strings `""`.
   - Set informative, non-credential placeholders (`admin@example.com` and `••••••••`).

## Testing Decisions

1. **High-Level Seam Selection**:
   - Primary Backend Testing Seam: The HTTP Router Seam (`internal/console/auth/handlers.go` and `internal/console/app/routes.go`). We will test the external HTTP behavior directly by sending requests to `/api/v1/auth/setup-status` and `/api/v1/auth/setup` against a test database registry.
   - Verification covers:
     - Querying setup status when zero Super Admins exist returns `setup_required: true`.
     - Submitting setup with valid credentials provisions the user, returns 201 Created with valid JWT, and logs the audit event.
     - Subsequent calls to `/api/v1/auth/setup` return `409 Conflict`.
     - Querying setup status after creation returns `setup_required: false`.
     - Submitting mismatched passwords or weak passwords returns `400 Bad Request`.
2. **End-to-End Verification Seam**:
   - Automated E2E test in `tests/e2e/e2e_features_test.go` and live browser subagent verification:
     - Testing the login screen UI to ensure the form starts empty without placeholder ghosting and without "Remember me".
     - Testing setup status gating and redirection.
3. **Prior Art**:
   - Follows existing auth test suites in `internal/console/auth/` and E2E integration test suites in `tests/e2e/`.

## Out of Scope

- Multi-tenant organization hierarchies or multi-step wizard onboarding (e.g. database connection wizards during first boot).
- Two-Factor Authentication (2FA) / WebAuthn setup during the initial setup wizard (can be managed in post-login profile settings).
- OAuth / SSO federated login integration for initial Super Admin setup.

## Further Notes

- Existing automated environments with `ADMIN_EMAIL` and `ADMIN_PASSWORD` environment variables set will continue to seed the default Super Admin on boot, automatically setting `setup_required` to `false` and requiring zero operational changes.
