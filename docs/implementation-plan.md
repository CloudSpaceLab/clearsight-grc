# ClearSight Implementation Plan

Checkboxes describe repository status, not production readiness.

## Phase 0 — Canonical product and architecture

- [x] Programs, Matters, evidence and verified-outcome semantics.
- [x] target-customer and use-case catalogue.
- [x] authority, routing, delegation and escalation specification.
- [x] Respond/Capture and invitation specification.
- [x] system, data and performance architecture.
- [x] premium illustration, empty-state and guided-adoption standard.
- [x] continuous Signals, drift, evidence aging, routing integrity, readiness and governed-autonomy specification.

## Phase 1 — Application and governance foundation

### Scaffold

- [x] Go API and worker processes.
- [x] React/Vite application shell.
- [x] OpenAPI, CI, Compose and performance smoke.
- [x] deterministic in-memory build mode.
- [x] PostgreSQL-tagged composition.

### Identity, organization and authority

- [x] principal and assignment foundation schema.
- [x] organizational positions and role templates schema.
- [x] delegation and versioned routing-policy schema.
- [x] deterministic authority resolution and explanation.
- [x] authority simulation, policy listing and integrity API.
- [x] pgx-backed active-policy and selector resolution.
- [ ] enterprise identity and organization synchronization.
- [ ] policy draft/publish maker-checker commands.
- [ ] delegation approval, revocation and absence calendar.
- [ ] conflict and segregation-of-duties evaluation against live actors.

### Durable workflow

- [x] typed task states and optimistic transitions.
- [x] workflow task/event schema and queue indexes.
- [x] memory and pgx task repositories.
- [x] task list/create/transition APIs.
- [ ] persisted workflow-definition/state-machine registry.
- [ ] timers, working calendars, escalation, parallel joins and compensation worker.
- [ ] outbox publisher and inbox deduplication runtime.

### Guided adoption

- [x] role-specific guide model and state API.
- [x] memory and pgx onboarding-state repositories.
- [x] premium SVG illustration, EmptyState and IntroGuide components.
- [x] first-run reviewer guide and responsive UI.
- [ ] admin-authored guide definitions with approval/versioning.
- [ ] visual-regression and accessibility automation.

### Continuous autonomy

- [x] Signal, drift and readiness domain model.
- [x] deterministic drift rules for evidence, source, requirements, routing, controls and verification.
- [x] memory and pgx repositories.
- [x] signal-ingestion and readiness APIs.
- [x] Today readiness and Configure routing-integrity UI.
- [x] automation-policy schema.
- [ ] worker-driven scheduled evidence aging and routing scans.
- [ ] dependency-aware Program invalidation and readiness snapshots.
- [ ] automation simulation, canary, kill switch and verification runtime.
- [ ] precedent-memory service with scope and source-version controls.

## Phase 2 — Sources, evidence and secure capture

- [ ] Source Registry repositories and source-health worker.
- [ ] object storage, malware scanning, integrity manifest and legal hold.
- [ ] spreadsheet/document import pipeline with resumable chunks.
- [ ] PostgreSQL evidence-request and submission repositories.
- [ ] persisted invitation/session exchange and step-up authentication.
- [ ] protected-report identity/content separation and anonymous mailbox.

## Phase 3 — Programs and Matters

- [ ] Program aggregate, Requirement, applicability, controls and Evidence Contracts.
- [ ] incremental Compliance State and trigger evaluation.
- [ ] typed Matter lifecycle and closure contracts.
- [ ] Decision, Action, Response Package and Verification aggregates.
- [ ] point-in-time Program/Matter reconstruction.

## Phase 4 — Initial bank verticals

- [ ] continuous NDPA Program;
- [ ] regulatory-change Matter;
- [ ] protected authority-request Matter;
- [ ] legacy finding/exception to verified remediation.

## Release gates

Every completed item still requires relevant authorization, invitation, source, performance, recovery, accessibility and workload tests before production release.
