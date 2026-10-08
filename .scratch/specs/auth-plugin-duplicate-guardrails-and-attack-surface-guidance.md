# Specification: Auth Plugin Duplicate Guardrails and Attack Surface Guidance

**Status:** `completed`

## Problem Statement

When operators use the Apirout Web Console to configure authentication plugins on routes and services, they can inadvertently create redundant, corrupted, or insecure configurations:

1. **Unbounded Duplicate Plugin Staging:** In the plugin configuration dialog, when all 4 available plugin types (`key-auth`, `jwt-auth`, `basic-auth`, `hmac-auth`) are attached or staged, the modal selection defaults to `key-auth`. The modal action button ("Stage Plugin" / "Attach Plugin") remains clickable, and the internal submission handler lacks duplicate checking. Consequently, operators can repeatedly click "Attach Plugin" followed by "Stage Plugin" to spawn multiple duplicate instances of the same plugin type (e.g. four identical `key-auth` entries in draft mode).
2. **Missing Header-Level State Constraints:** The entity form's "Attach Plugin" header button remains permanently clickable, even when 100% of available authentication plugin types are already attached to the entity.
3. **Lack of Attack Surface Security Feedback:** While combining 2 distinct authentication plugins (e.g. `key-auth` and `jwt-auth`) is a standard gateway pattern for multi-persona access (public partners vs. internal users) evaluated via Logical OR, attaching 3 or 4 authentication mechanisms simultaneously to a single route can unnecessarily broaden the credential attack surface (e.g., exposing Basic Auth to brute-force vectors on high-security HMAC endpoints). The console currently provides no guidance to help operators assess this trade-off.

## Solution

Implement an authoritative Three-Layer Duplicate Plugin Defense Guardrail in the Web Console, paired with Contextual Attack Surface Guidance:

1. **Header-Level Guard:** When all available authentication plugin types are active or staged on an entity, disable the header's "Attach Plugin" button, change its label to "All Plugins Attached", and provide a tooltip explaining that all supported authentication mechanisms are already in use.
2. **Modal-Level Guard & Auto-Selection:** When opening the attachment modal, automatically pre-select the first unattached plugin type. In the selection grid, render attached plugins as disabled with an "Already Attached" badge. Strictly disable the modal action button whenever an already-attached plugin type is selected.
3. **Handler-Level Programmatic Guard:** Authoritatively reject duplicate plugin types in the staging and creation handlers with an error toast notification before any state mutation can occur.
4. **Contextual Attack Surface Guidance:** In the Multi-Plugin Logical OR informational banner, dynamically display an operational security posture tip whenever 3 or more authentication plugins are active on a route, advising operators to ensure all permitted authentication schemes satisfy their organizational security policies.

## User Stories

1. As an API platform operator, I want the "Attach Plugin" header button to be disabled with the label "All Plugins Attached" when all 4 authentication plugin types are already configured on a route, so that I am clearly informed that no further plugin types can be added.
2. As an API platform operator, I want hovering over the disabled "All Plugins Attached" button to show a descriptive tooltip, so that I understand why the button is inactive.
3. As an API platform operator, I want opening the attach modal to automatically pre-select the first unattached plugin type, so that I do not accidentally start on an already-attached plugin.
4. As an API platform operator, I want plugin cards in the modal selection grid that are already attached to be clearly badged as "Already Attached" and non-clickable, so that I cannot re-select an active plugin type.
5. As an API platform operator, I want the "Stage Plugin" and "Attach Plugin" action buttons in the modal dialog to be visually and functionally disabled when an attached plugin is highlighted, so that I cannot submit redundant configurations.
6. As an API platform operator, I want the client state handler to programmatically reject any attempt to stage a duplicate plugin type, so that even scripted or rapid-fire actions cannot corrupt entity draft state.
7. As an API platform operator, I want an error toast to inform me if a duplicate plugin attachment was attempted, so that I receive immediate feedback on why the action was rejected.
8. As a security engineer reviewing route configurations, I want the console to display a security tip in the multi-plugin banner when 3 or more authentication plugins are attached, so that operators are warned about credential attack surface widening.
9. As an API platform operator, I want the gateway to continue supporting Logical OR evaluation for up to 2 heterogeneous plugins without security warnings, so that common multi-persona patterns (such as public API Key alongside internal JWT) feel natural and unencumbered.
10. As an API platform operator, I want deleting or removing a staged plugin to immediately re-enable that plugin type in the attach modal, so that I can reconfigure or replace plugins flexibly.
11. As an API platform operator, I want removing a plugin so that fewer than 4 are attached to immediately re-enable the header "Attach Plugin" button, so that I can add new plugins without refreshing the page.
12. As a platform administrator, I want these duplicate guardrails to apply consistently across both Route creation/editing and Service creation/editing forms, so that consistent validation semantics govern the entire platform.
13. As an API consumer developer, I want protected routes to have deterministic, non-duplicated plugin configurations, so that authentication evaluation overhead remains predictable and minimal.
14. As an auditor reviewing system architecture records, I want ADR-0008 to reflect the duplicate guardrail and attack surface policies, so that future engineers understand the rationale behind the UI constraints.

## Implementation Decisions

- **Three-Tier Guardrail Architecture:**
  - *Header Boundary:* Inspect the active (non-deleted) plugins against the available plugin catalog. If all available types are attached, disable the button, apply a muted style, and display "All Plugins Attached".
  - *Modal Boundary:* The selection algorithm computes the set difference between the available catalog and currently attached plugins, defaulting selection to the first unattached type. Modal action button binds `disabled` to whether the currently selected type is already attached.
  - *Submission Boundary:* In the staging and direct attachment handlers, execute an explicit duplicate check against non-deleted plugins. If matched, immediately abort and dispatch a descriptive error toast.
- **Contextual Attack Surface Guidance in Banner:**
  - When active plugins $\ge 2$, render the Multi-Plugin Logical OR explanation.
  - When active plugins $\ge 3$, render an additional contextual security guidance note advising operators on attack surface minimization.
  - When active plugins $< 2$, hide the banner entirely.
- **Dynamic Re-activation Lifecycle:**
  - Marking a plugin for removal or discarding a staged plugin immediately updates the active plugins list, re-enabling the header button and unblocking that plugin type in the selection modal without requiring page reload.

## Testing Decisions

- **Testing Seam:** High-seam Web Console user interface interactions and Control Plane state management.
  - *Test External Behavior:* Test that the header button disables when all 4 plugins are attached; test that the modal submit button is disabled when selecting an attached plugin; test that programmatic submission is rejected if a duplicate exists.
  - *Test Banner Thresholds:* Test that 0-1 plugins show no banner, 2 plugins show the Logical OR banner, and 3-4 plugins show both the Logical OR banner and the attack surface security warning.
  - *Test Deletion Re-enabling:* Test that discarding or deleting a plugin immediately makes that plugin type attachable again.
  - *Data Plane Integration:* Ensure existing end-to-end tests continue to pass 100%.

## Out of Scope

- Modifying the Go Data Plane authentication engine or changing the Logical OR evaluation logic.
- Adding new authentication plugin types beyond `key-auth`, `jwt-auth`, `basic-auth`, and `hmac-auth`.
- Database schema changes or backend migration modifications.

## Further Notes

- This specification directly resolves the issue demonstrated in the user's screenshot where four `key-auth` plugins were staged simultaneously.
