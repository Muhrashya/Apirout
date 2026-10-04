# Spec: Independent Brand Identity, Cartoony Topology Architecture, and Interactive README Overhaul

**Triage Label:** `ready-for-agent`

## Problem Statement

Apirout has matured into a multi-plane API engine and instant data proxy, yet its public presentation and documentation undermine its credibility and independent identity:

1. **AI-Generated Stock Aesthetics**: The current logo (`docs/logo.png`, `web/public/logo.png`) is an AI-generated blue ribbon with floating digital cubes that feels synthetic and generic, lacking a memorable brand identity.
2. **Text-Heavy ASCII Architecture**: The README visualizes system topology using an ASCII text block. It fails to convey the dynamic, multi-plane architecture (stateless data plane, private control plane, Redis event bus, and instant database connectors) in an engaging, intuitive manner.
3. **Derivative Brand Nomenclature**: The codebase and documentation repeatedly describe Apirout as "Kong-Aligned", "Kong-Style", and "Hasura-Inspired". While Apirout adopts proven architectural patterns (Services, Routes, Upstreams, Consumers, and Instant CRUD), advertising external commercial products diminishes Apirout's autonomy as a standalone open-source platform.
4. **Corporate "AI Slop" Tone in README**: The current README is written in a verbose, buzzword-heavy style ("battle-tested architecture", "sub-millisecond overhead", bloated bulleted lists) rather than direct, authentic developer documentation.
5. **Lack of Interactivity and Codebase Alignment**: The README does not reflect recent architectural additions (such as the four dedicated listener ports `:8000`, `:8001`, `:8002`, `:8100`, server-side paginated request telemetry, and the SQL query workbench). It lacks interactive elements like collapsible sections (`<details>`) and actionable, copy-pasteable curl workflows with real response payloads.

## Solution

Transform Apirout's public presence, visual identity, and documentation into a standalone, developer-centric, interactive open-source project:

1. **Brand Identity & Icon Modernization**:
   - Establish the official brand name as simply **Apirout** (dropping "Gateway" from primary titles).
   - Replace the AI-generated ribbon logo with a clean, high-impact **lightning bolt** icon featuring a warm orange-to-coral gradient on a transparent background, ensuring seamless rendering on GitHub light/dark modes and within the web console.
   - Update `docs/logo.png`, `web/public/logo.png`, and `web/public/favicon.png`.

2. **2D Cartoony Architecture & Topology Illustration**:
   - Create a high-resolution 2D cartoony tech illustration (`docs/architecture-topology.png`) rendered in a clean, playful hand-drawn/isometric style featuring friendly server containers, client devices, glowing lightning routing flows, and database cylinders.
   - Embed this illustration prominently in `README.md` to replace the ASCII text tree.

3. **Standalone Identity & Complete De-Kong / De-Hasura Sanitization**:
   - Remove references to "Kong" and "Hasura" across `README.md`, `docs/ARCHITECTURE.md`, `docs/QUICKSTART.md`, `docs/PLAN_PEMISAHAN_MULTIPLANES.md`, and ADRs.
   - Present core entities (**Services**, **Routes**, **Upstreams**, **Targets**, **Consumers**) and the **Instant Database Engine** as native, first-class Apirout capabilities.
   - Update the CLI help text banner in `cmd/apirout/main.go` and internal Go code comments.
   - Silently maintain backward-compatible environment variable and header fallbacks (`KONG_ROLE`, `X-Hasura-Admin-Secret`) in the runtime code to prevent breaking existing automation.

4. **Human-First, Interactive README Overhaul**:
   - Rewrite `README.md` with an authentic, developer-first voice.
   - Include a 10-second Docker Quickstart (`docker compose up -d`) and an immediate port matrix cheat-sheet.
   - Organize in-depth guides into interactive collapsible `<details>` blocks (Core Entities with real curl snippets, Instant Database Query syntax, Observability & Telemetry, and Production Deployment options).
   - Add a 60-second end-to-end tutorial with realistic request inputs and JSON response payloads.

## User Stories

1. As a developer visiting the repository for the first time, I want to see a crisp, recognizable lightning bolt logo and a clean title ("Apirout"), so that the project feels polished, authentic, and memorable.
2. As a software architect evaluating Apirout, I want to view a 2D cartoony topology illustration in the README, so that I can immediately understand the relationship between clients, the data plane proxy, the control plane, Redis, and database backends.
3. As an open-source contributor, I want the documentation to present Apirout as a standalone platform rather than an imitation of Kong or Hasura, so that I understand its unique value proposition and unified architecture.
4. As a devops engineer, I want a copy-pasteable 1-line Docker Compose command in the README, so that I can spin up the full stack in under 10 seconds without hunting through documentation.
5. As an operator configuring firewalls and ingress, I want a clear port matrix cheat-sheet (:8000, :8001, :8002, :8100), so that I know exactly which ports are public, private, or internal.
6. As a backend developer, I want collapsible `<details>` sections for Core Entities (Services, Routes, Upstreams, Consumers), so that I can inspect full curl examples without being overwhelmed by a wall of text.
7. As a data engineer, I want an interactive guide to the Instant Database REST Proxy, so that I can learn how to query relational tables with filtering (`eq`, `gte`, `like`), sorting, and server-side pagination.
8. As a developer running `apirout -h` from the terminal, I want the CLI banner and help text to cleanly identify Apirout without legacy vendor strings, so that the command-line interface feels cohesive.
9. As an automated CI/CD pipeline using legacy environment variables (`KONG_ROLE` or `X-Hasura-Admin-Secret`), I want my configurations to continue working without breaking changes, so that existing deployments run smoothly.
10. As a user browsing the Web Console in light or dark mode, I want the new transparent lightning bolt logo to render cleanly in the top navigation bar, so that the UI feels unified with the repository branding.
11. As a site reliability engineer, I want the README to document real telemetry capabilities (synthetic P50/P90/P99 latency percentiles, Prometheus metrics, and server-side paginated request logs), so that I know what observability tools are built in.
12. As a platform engineer deploying to production, I want clear references to the VPS Nginx edge reverse proxy and Kubernetes manifests in `<details>` blocks, so that I can proceed directly to production deployment.
13. As a GitHub reader on a mobile phone or smaller screen, I want interactive `<details>` blocks to keep the page scannable and easy to navigate.
14. As a developer sharing Apirout with colleagues, I want a 60-second end-to-end tutorial in the README showing a complete flow from registering a service to querying it through the gateway.
15. As a maintainer reviewing docs, I want all markdown links and documentation references to resolve to valid, active files in the repository.

## Implementation Decisions

1. **Logo & Asset Specifications**:
   - Design: Minimalist sharp lightning bolt with warm orange-to-coral linear gradient (`#FF8A00` to `#FF3E3E`), matching the reference icon.
   - Transparency: Alpha channel transparent background (PNG format), cropped tightly with subtle balance margin.
   - Target files:
     - `docs/logo.png`: 512x512 high-res version for GitHub README and documentation.
     - `web/public/logo.png`: 128x128 version for Web Console topbar.
     - `web/public/favicon.png`: 64x64 version for browser tab favicon.

2. **Cartoony Topology Illustration**:
   - Concept: 2D cartoony tech illustration featuring:
     - Public client icons (mobile phone, laptop, API client) sending traffic.
     - Data Plane Gateway Proxy (:8000) with internal routing, auth check, and instant database proxy.
     - Control Plane (:8001 Admin API + :8002 Web Console) with admin users.
     - Redis Event Bus for live config invalidation.
     - Connected Databases (PostgreSQL, MySQL) and Upstream Services.
   - Target file: `docs/architecture-topology.png`.
   - Visual Style: Clean hand-drawn / isometric cartoon style with soft rounded outlines, playful server boxes, glowing energy lines, and high readability.

3. **De-Kong & De-Hasura Sanitization Scope**:
   - `README.md`:
     - Change title from `# ⚡ Apirout Gateway` to `# ⚡ Apirout`.
     - Remove "Kong-Aligned", "Kong Hybrid", "Hasura-style", etc.
     - Rebrand badges: Remove Kong badge, maintain License, Go, React, Docker, and Build badges.
     - Update core entity and instant database sections to native Apirout terminology.
   - Documentation in `docs/`:
     - `docs/ARCHITECTURE.md`: Revise overview and section headers to present Apirout's multi-plane architecture and instant data engine autonomously.
     - `docs/QUICKSTART.md`: Update role definitions and instant database steps.
     - `docs/PLAN_PEMISAHAN_MULTIPLANES.md`: Update historical references to reflect native multi-plane design.
     - `docs/adr/0001-hybrid-gateway-architecture.md`: Retain architectural rationale while framing decisions as Apirout's native multi-plane design.
   - CLI Banner & Code Comments:
     - In `cmd/apirout/main.go`: Replace help text `Apirout CLI - Open Source Instant API Engine & Gateway (Kong & Hasura Architecture)` with `Apirout CLI - Open Source API Engine & Instant Data Proxy`.
     - In `Makefile`: Update help text.
     - In Go comments: Clean up references across `internal/config`, `internal/gateway`, `internal/console`.
   - Backward Compatibility:
     - Keep `getEnvAny("ROLE", "APIROUT_ROLE", "KONG_ROLE", "traditional")` in `internal/config/config.go`.
     - Keep `r.Header.Get("X-Hasura-Admin-Secret")` fallback in `internal/console/auth/jwt.go`.

4. **README Structure & Interactive Components**:
   - **Header**: Centered new logo, title `# ⚡ Apirout`, punchy tagline: *"High-performance API Engine, Multi-Plane Request Router & Instant Database Proxy"*, clean badges, navigation anchor bar.
   - **Why Apirout?**: 3 concise bullet points focusing on real developer pains (instant APIs without backend boilerplate, sub-millisecond dynamic routing, all-in-one or decoupled multi-plane deployment).
   - **10-Second Quickstart**: Single `docker compose up -d` snippet and default admin credentials.
   - **Dedicated Listener Ports**: Clean markdown table detailing ports `:8000`, `:8001`, `:8002`, `:8100`.
   - **Architecture & Topology**: The 2D cartoony topology illustration embedded with clear descriptions of Data Plane vs Control Plane.
   - **Interactive Deep Dives (`<details>` tags)**:
     - `<details><summary><b>🧩 Core Entities (Services, Routes, Upstreams, Consumers)</b></summary>`: Concrete curl examples for each entity.
     - `<details><summary><b>⚡ Instant Database REST APIs & SQL Workbench</b></summary>`: Supported operators table (`eq`, `neq`, `gt`, `like`, `order`, `limit`) and custom parameterized queries.
     - `<details><summary><b>📊 Observability, Real Latency Percentiles & Logs</b></summary>`: Prometheus metrics, P50/P90/P99 latency calculations, and paginated request explorer.
     - `<details><summary><b>🚀 Production VPS & Kubernetes Deployment</b></summary>`: Nginx reverse proxy architecture (`deploy/docker-compose.prod.yml`) and Kubernetes manifests.
   - **60-Second End-to-End Walkthrough**: Direct example showing service creation, route configuration, and request execution through port 8000.
   - **License**: Standard Apache 2.0 license notice.

## Testing Decisions

The changes will be tested across three primary seams:

1. **Asset & Visual Rendering Seam**:
   - Verify `docs/logo.png`, `web/public/logo.png`, and `web/public/favicon.png` are valid PNG images with transparent backgrounds and sharp rendering.
   - Verify `docs/architecture-topology.png` is generated, high-resolution, visually legible, and renders properly in markdown viewers and browsers.
   - Check Web Console (`http://localhost:8002`) in browser to confirm the new logo and branding appear seamlessly in the topbar and login views.

2. **Text Sanitization & Link Integrity Seam**:
   - Run grep searches across `README.md` and `docs/` to verify zero remaining occurrences of "Kong" or "Hasura" in user-facing documentation text.
   - Validate that all internal markdown links (e.g. `[ARCHITECTURE.md](docs/ARCHITECTURE.md)`, `[QUICKSTART.md](docs/QUICKSTART.md)`) resolve to valid existing files.
   - Verify CLI help output (`go run ./cmd/apirout -h`) displays the clean Apirout banner.

3. **Build & Test Suite Regression Seam**:
   - Run `npm run build` in `web/` to confirm all frontend assets compile cleanly.
   - Run `go test ./...` across the entire backend to guarantee that zero existing tests or backward-compatible fallbacks are broken.

## Out of Scope

- Changing underlying Go package paths, module names (`github.com/apirout/apirout`), or Docker container names (`apirout`, `apirout-redis`, `apirout-registry-db`).
- Breaking changes to REST API endpoint paths (such as `/api/v1/services`, `/api/v1/routes`, `/api/v1/instant`).
- Removing legacy environment variable fallbacks (`KONG_ROLE`, `X-Hasura-Admin-Secret`) that guarantee non-breaking execution for existing users.

## Further Notes

- The cartoony topology illustration should use playful, colorful aesthetics (inspired by Notion/Stripe/Excalidraw tech art) while maintaining technical precision regarding port numbers and traffic flows.
