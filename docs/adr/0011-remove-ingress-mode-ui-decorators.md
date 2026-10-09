# ADR-0011: Remove Ingress Mode UI Decorators from Gateway Domains Console

## Status
Accepted

## Context
Following ADR-0005, the Gateway Domains console (Domains.tsx, DomainFormPage.tsx) was updated
to display "Edge Offloaded" as the ingress architecture label — replacing the previous deceptive
"TLS 1.3 Active" mocks.

However, a subsequent audit revealed that the resulting "Edge Offloaded" indicators were
themselves entirely decorative: no `ingress_mode` field exists on the `GatewayDomain` model
(`internal/core/models/gateway_domain.go`), no corresponding column exists in the
`gateway_domains` database table, and no runtime code in `stage_ingress.go` or `handler.go`
reads or branches on an ingress mode value.

Concretely, three UI elements were hardcoded strings with zero data backing:

1. **"INGRESS ARCHITECTURE" summary card** — always displayed "Edge Offloaded / SSL/TLS terminated at Edge Reverse Proxy / Ingress", regardless of any domain configuration.
2. **"Ingress Mode" table column** — every row always showed the same badge.
3. **"Gateway Ingress Architecture" disabled input in the registration form** — hardcoded to "Edge Offloaded Ingress (HTTP Gateway)", never submitted or stored.

## Decision
Remove all three UI elements. The ingress topology (edge-offloaded TLS) is an
**infrastructure deployment concern**, not a per-domain configuration. Apirout always operates
as a plain HTTP listener; TLS termination is always the responsibility of the edge layer
(Cloudflare, Kubernetes Ingress, AWS ALB, Caddy, Nginx). This is a single, system-wide
constraint documented in ADR-0005 — it is not a property that varies per `GatewayDomain` record
and therefore has no place as a configurable or displayable field in the Domains console.

The page subtitles ("edge ingress bindings") were also corrected to accurately describe the actual
function of the page: hostname and host-header routing rule management.

## Consequences
- **Positive**:
  - Eliminates a second layer of misleading UI. Operators no longer see a "mode" field implying that ingress topology is configurable when it is not.
  - Shrinks the `GatewayDomain` interface surface: the domain model only exposes fields with actual data semantics.
- **If ingress modes are needed in future**:
  - Add an `ingress_mode` enum column to `gateway_domains` table.
  - Surface it through the `GatewayDomain` Go model and API.
  - Only then re-introduce the UI field, wired to real data.
  - A new ADR should document the chosen modes and their runtime semantics.
