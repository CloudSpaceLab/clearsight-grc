# ClearSight — Automation & Governance Settings

Screen 18, approved ClearSight light visual language. Design-only prototype (October 2026).

## Operating model

- Automations: searchable scoped rules with a selected-rule detail, current revision, status and last execution. A rule is not a permission grant.
- Schedules: due runs and reminders in 7/30/90-day windows, independent of governance outcomes.
- Escalations: thresholds, recipient roles and scope; notifying someone is not authorizing a decision.
- Activity: policy revisions, execution receipts, delivery/failure states, and separate decision history.
- Editor: create an immutable draft, simulate impact and recipients, request independent authorization, then activate under current authority and version. No unreviewed production edits.
- Simulation: review recipient resolution, unsupported routes and outcome counts before applying. Current sample shows one intentionally unresolved route to illustrate fail-closed behavior.

## Interaction and presentation

- Existing shell and shared ClearSight prototype CSS. Selected row drives the contextual detail panel, avoiding a second dashboard.
- Stronger visual feedback only on selection, changes, dialogs and status. No decorative auto-running animation or KPI counters.
- Controls are keyboard navigable; Ctrl/Cmd+K focuses search, Escape dismisses dialogs; mobile layout stacks the details below the list.
- Filters and local draft changes are functional within the preview, but produce no production commands. All example records, timestamps and simulation counts are invented solely for demonstrating interface states.

## Implementation contract

- Reuse existing authority, workflow automation policy, versioned configuration, notification, outbox, source binding, and execution-receipt contracts. No additional generic rules framework or independent store.
- Production activation must require maker-checker, scoped role resolution, simulation, effective dates, blast radius, rollback/kill switch, idempotency and explicit monitoring/outcome checks.
- Differentiate trigger detected, notification queued, notification delivered, action executed and final verified outcome. Never silently close risk, incident or compliance state.
- Show current scope, freshness, authority and failure at the point of work. No seeded demo data in customer tenants.

## Render evidence

Desktop 1880 CSS px / 2× render, tablet 768 CSS px, mobile 390 CSS px. Key interactions checked: tabs, search, category/status/group filters, simulation, editor, local draft creation, keyboard focus and mobile layout. No horizontal overflow at tested widths.
