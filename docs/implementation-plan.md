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
- [x] semantic work-item vector icons, enterprise copy standard and plain-language state labels.
- [x] first-run reviewer guide and responsive UI.
- [x] plain-language UI layer over stable internal status codes.
- [ ] admin-authored guide definitions with approval/versioning.
- [x] screen-level decision-brief and rendered-evidence contracts.
- [ ] automated state-gallery screenshots, visual regression and accessibility checks.

### Continuous autonomy

- [x] Signal, drift and readiness domain model.
- [x] deterministic drift rules for evidence, source, requirements, routing, controls and verification.
- [x] memory and pgx repositories.
- [x] signal-ingestion and readiness APIs.
- [x] Today readiness and Configure routing-integrity UI.
- [x] automation-policy schema.
- [x] Program trigger deduplication, linked Matter creation and reason-bearing status snapshots.
- [ ] dependency graph propagation across shared controls and services.
- [ ] automation simulation, canary, kill switch and verification runtime.
- [ ] precedent-memory service with scope and source-version controls.

## Phase 2 — Sources, evidence and secure capture

- [x] Source Registry memory/PostgreSQL repositories and bounded source-health worker.
- [x] source observations, deterministic freshness and transactional health events.
- [x] PostgreSQL evidence-request and immutable submission repositories.
- [x] persisted hash-only invitation exchange, bounded sessions and revocation.
- [x] request-state guard, replay protection and optimistic submission.
- [x] streamed local-development object store and SHA-256 artifact manifests.
- [x] `STORED_UNSCANNED` / available / quarantined artifact-state contract.
- [x] Work workspace for source health and evidence requests.
- [ ] production object storage, encryption-key policy and malware scanning.
- [ ] legal hold, retention and deletion workers.
- [ ] verified external identity and step-up authentication.
- [ ] resumable multipart upload and spreadsheet/document import pipeline.
- [ ] protected-report identity/content separation and anonymous mailbox.

## Phase 3 — Programs and Matters

- [x] Program aggregate, Requirement, applicability, control objectives and implementations.
- [x] Requirement-to-control mapping and source-scoped Evidence Contracts.
- [x] incremental, reason-bearing Compliance State and trigger evaluation.
- [x] idempotent trigger-to-Matter creation.
- [x] typed Matter lifecycle, decisions, actions and response packages.
- [x] outcome-check contracts/results and typed closure rules.
- [x] point-in-time Program/Matter reconstruction.
- [x] PostgreSQL projections, continuity event stream and outbox delivery.
- [x] Programs and Issues/Changes user surfaces with plain-language state labels.
- [x] gateway-signed request actor binding and automatic authority checks on material Program/Matter commands.
- [x] projection-first high-cardinality summaries, indexed search, keyset pagination and lazy detail loading.
- [x] separately versioned Program-status maintenance with health, reconciliation and governed rebuild.
- [ ] bulk Program setup and controlled configuration-change workflow.

## Phase 4 — Initial bank verticals

- [ ] continuous NDPA Program;
- [ ] regulatory-change Matter;
- [ ] protected authority-request Matter;
- [ ] legacy finding/exception to verified remediation.

## Release gates

Every completed item still requires relevant authorization, invitation, source, performance, recovery, accessibility and workload tests before production release.

- [x] Projection-first Program and Matter summaries, indexed search, keyset pagination and lazy detail loading.
- [x] Signed request actor binding and authority checks for material Program/Matter commands.
- [x] Separately versioned Program-status maintenance with health, reconciliation and governed rebuild operations.
- [ ] Direct enterprise identity-provider integration, gateway key rotation and organization synchronization.
- [ ] Representative 100,000-row p95/p99 release evidence and retained query plans.
