# Issue tracker: Local Markdown

Issues and specs for this repo live as markdown files in `.scratch/`.

## Conventions

- Specs live as markdown files under `.scratch/specs/<slug>.md`.
- Implementation issues / tickets live as individual markdown files under `.scratch/tickets/<NN>-<slug>.md`, numbered sequentially (e.g. `01-dynamic-version-injection.md`, `23-server-side-paginated-api-logs-and-log-explorer-telemetry.md`), never a single combined file.
- Triage state is recorded as a `**Status:** <role>` line near the top of each ticket or spec (see `triage-labels.md` for role strings: `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `completed`, `wontfix`).
- Acceptance criteria are tracked as markdown checkboxes (`- [ ]` / `- [x]`).
- Comments and discussion append to the bottom of the file under a `## Comments` heading.

## When a skill says "publish to the issue tracker"

Create a new file under `.scratch/tickets/` (or `.scratch/specs/` for specs), creating the directory if needed.

## When a skill says "fetch the relevant ticket"

Read the file at `.scratch/tickets/<NN>-<slug>.md` or the path referenced by the user.

## Wayfinding operations

Used by `/wayfinder`. The **map** is a file with one **child** file per ticket.

- **Map**: `.scratch/<effort>/map.md` (the Notes / Decisions-so-far / Fog body).
- **Child ticket**: `.scratch/tickets/<NN>-<slug>.md` (or `.scratch/<effort>/issues/<NN>-<slug>.md`), with the goal/question in the body. A `Type:` line records the ticket type (`research`/`prototype`/`grilling`/`task`); a `Status:` line records `claimed`/`resolved`/`completed`.
- **Blocking**: a `**Blocked by:** NN, NN` line near the top. A ticket is unblocked when every ticket it lists is marked resolved or completed.
- **Frontier**: scan tickets for files that are open, unblocked, and unclaimed; first by number wins.
- **Claim**: set `Status: claimed` and save before executing work.
- **Resolve**: append the answer or completion notes under an `## Answer` heading, mark checkboxes complete, set `Status: completed`, and append a context pointer to the map's Decisions-so-far.
