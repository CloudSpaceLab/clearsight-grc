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
- [x] PostgreSQL-tagged composition and runtime integration tests.

### Identity, organization and authority

- [x] principal, organizational position and role-template schema.
- [x] deterministic authority resolution, simulation and explanation.
- [x] legal-entity-safe pgx selector resolution and integrity findings.
- [x] routing-policy draft, submit, approve, reject and retire commands.
- [x] policy checksum, maker-checker and selector-cardinality validation.
- [x] delegation draft, approval, activation, expiry and revocation lifecycle.
- [x] recursive delegation-cycle and active segregation-rule checks.
- [x] append-only governance decisions and state-change outbox events.
- [ ] enterprise identity and organization synchronization.
- [ ] authenticated actor authority for policy/delegation administration.
- [ ] absence and working-calendar source integration.

### Durable workflow and delivery

- [x] typed task states and optimistic transitions.
- [x] workflow task/event schema and queue indexes.
- [x] durable leased timers with stale-claim recovery.
- [x] transactional timer completion and outbox creation.
- [x] outbox lease, bounded retry and claim ownership.
- [x] consumer inbox deduplication.
- [x] memory and PostgreSQL worker composition.
- [ ] persisted workflow-definition/state-machine registry.
- [ ] business calendars, parallel joins and compensation definitions.
- [ ] approved email, messaging, ITSM and automation publishers.
- [ ] backlog SLO dashboards and dead-letter operating workflow.

### Guided adoption

- [x] role-specific guide model and state API.
- [x] memory and pgx onboarding-state repositories.
- [x] premium SVG illustration, EmptyState and IntroGuide components.
- [x] semantic work-item vector icons and enterprise copy standard.
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
