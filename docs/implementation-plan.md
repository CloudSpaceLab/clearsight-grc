# ClearSight Implementation Plan

Checkboxes describe repository capability, not production readiness.

The detailed finished-product experience and delivery sequence are defined in:

- [`design/enterprise-productization-design-plan.md`](design/enterprise-productization-design-plan.md);
- [`engineering/enterprise-productization-implementation-plan.md`](engineering/enterprise-productization-implementation-plan.md).

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
- [x] verified actor context for reads and rejection of conflicting tenant/principal/legal-entity query scope.
- [x] fail-closed restricted Matter reads with explicit principal allow-lists.
- [ ] enterprise identity and organization synchronization.
- [ ] authenticated actor authority for policy/delegation administration.
- [ ] absence and working-calendar source integration.
- [ ] database row-level security and synchronized restricted groups.

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

### Guided adoption and accessible interaction

- [x] role-specific guide model and state API.
- [x] memory and pgx onboarding-state repositories.
- [x] premium SVG illustration, EmptyState and IntroGuide components.
- [x] semantic work-item vector icons, enterprise copy standard and plain-language state labels.
- [x] first-run reviewer guide and responsive UI.
- [x] plain-language UI layer over stable internal status codes.
- [x] screen-level decision-brief and rendered-evidence contracts.
- [x] journey progress semantics, specific control names, focus states, responsive reflow and reduced-motion handling.
- [x] full material facts, reasons, requirements, contradictions and closure blockers without silent truncation.
- [x] rendered import-state and demo-mode tests with axe accessibility checks enforced in CI.
- [ ] admin-authored guide definitions with approval/versioning.
- [ ] automated state-gallery screenshots and visual-regression checks.
- [ ] automated WCAG contrast, keyboard and screen-reader regression suite across all product surfaces.

### Continuous autonomy

- [x] Signal, drift and readiness domain model.
- [x] deterministic drift rules for evidence, source, requirements, routing, controls and verification.
- [x] memory and pgx repositories.
- [x] signal-ingestion and readiness APIs.
- [x] Today readiness and Configure routing-integrity UI.
- [x] automation-policy schema.
- [x] Program trigger deduplication, linked Matter creation and reason-bearing status snapshots.
- [x] actor-scoped dynamic Today projection from current journey state in memory and PostgreSQL composition.
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
- [x] Matter-derived request visibility before PostgreSQL limits, regardless of duplicated request sensitivity.
- [x] actionable-request selection before submitted/cancelled historical requests.
- [x] tenant-scoped governed document-import records with immutable original metadata, SHA-256 lineage and PostgreSQL persistence.
- [x] deterministic TXT, Markdown, CSV, DOCX and XLSX extraction with section, worksheet and row anchors.
- [x] source-anchored requirement, deadline, authority-reference, control-expectation and risk proposals that remain advisory until explicit review.
- [x] actor-bound import/list/detail/review APIs and an operational Imports workbench.
- [x] optimistic, identity-recorded accept/reject review without automatic governed-record creation.
- [x] configurable stakeholder demo mode that is independent from normal document imports and prohibited in production.
- [ ] production object storage, encryption-key policy and malware scanning.
- [ ] legal hold, retention and deletion workers.
- [ ] verified external identity and step-up authentication.
- [ ] PDF text extraction, image OCR, password-protected-document handling and approved extraction-provider isolation.
- [ ] resumable multipart upload, saved mappings, partial acceptance and repeat-import reconciliation.
- [ ] accepted-proposal conversion into versioned governed records through authorized, maker-checker workflows.
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
- [x] visibility-aware Matter pagination in memory and PostgreSQL.
- [x] separately versioned Program-status maintenance with health, reconciliation and governed rebuild.
- [ ] bulk Program setup and controlled configuration-change workflow.

## Phase 4 — Initial bank reference verticals

Repository reference capability:

- [x] continuous Nigeria data-protection reference Program;
- [x] regulatory-change issue through decision, action and pending outcome check;
- [x] protected authority-request issue through approved response and acknowledgement;
- [x] legacy finding through implemented remediation, independent verification and closure;
- [x] exact current-record journey evaluation that rejects retired, superseded, cancelled, withdrawn and non-independent records;
- [x] actionable Explore launchers to linked Programs, issues and evidence requests;
- [x] recoverable non-production installer for partial Program and Matter states;
- [x] dynamic Today work in memory and PostgreSQL compositions;
- [ ] bank-approved legal configuration and current-source review;
- [ ] production restricted-group synchronization and authority-channel integration;
- [ ] complete operational write UI for every decision, response and closure mutation;
- [ ] representative production-scale journey benchmark and retained query plans.

## Phase 5 — Enterprise productization and pilot readiness

This phase is required before ClearSight is described as a completed enterprise banking product.

### P0 — Baseline and traceability

- [ ] capability maturity register distinguishing foundation, pilot and production readiness;
- [ ] complete UI inventory and retained before-state evidence;
- [ ] productization acceptance-suite structure;
- [ ] issue/PR traceability from use case to rendered and behavioral evidence.

### P1 — UI/UX cleanup and premium design system

- [ ] complete semantic token system with light and dark themes;
- [ ] production component library and state gallery;
- [ ] screen-by-screen cleanup of Today, Programs, Issues/Changes, Work, Explore and Configure;
- [ ] compact and comfortable density where justified;
- [ ] complete operational write interfaces for decisions, actions, responses, verification and closure;
- [ ] automated contrast, keyboard, focus, zoom, reduced-motion and visual-regression gates.

### P2 — Enterprise identity and directory compatibility

- [ ] OIDC authentication-provider integration;
- [ ] SAML authentication-provider integration;
- [ ] SCIM user/group lifecycle synchronization;
- [ ] LDAP/LDAPS and Active Directory sync agent or connector;
- [ ] controlled CSV/XLSX organization import with saved mappings and dry run;
- [ ] source-backed user, group, position, manager and legal-entity reconciliation;
- [ ] identity-change impact, orphaned responsibility and safe reassignment workflows.

### P3 — RBAC, responsibility, authority and escalation

- [ ] versioned capability bundles and role catalogue;
- [ ] scoped responsibility-assignment model and matrix;
- [ ] decision-authority matrix with thresholds, quorum, challenge and signatory;
- [ ] reusable conflict and segregation engine;
- [ ] business-calendar-aware reminder and escalation policies;
- [ ] routing/escalation sequence builder, simulation and impact preview;
- [ ] full material-command authorization coverage;
- [ ] maker-checker activation, effective dating and rollback for all configuration.

### P4 — Role-aware first-time guidance

- [ ] governed, admin-authored guide definitions and versions;
- [ ] guide resolution from current roles, responsibilities and permissions;
- [ ] executive guide;
- [ ] Program-owner guide;
- [ ] reviewer/challenger guide;
- [ ] authorizer/signatory guide;
- [ ] evidence-respondent guide;
- [ ] Configure-administrator guide;
- [ ] auditor/read-only guide;
- [ ] permission-safe anchors, meaningful first task and privacy-minimized adoption telemetry.

### P5 — Notifications and email templates

- [ ] versioned notification policy and template registry;
- [ ] safe HTML/plaintext email rendering and preview;
- [ ] production email delivery adapter with retry, bounce and dead-letter handling;
- [ ] in-product notification centre;
- [ ] policy-constrained preferences, quiet hours and digests;
- [ ] assignment, evidence, reminder, escalation, approval, verification, invitation, configuration and security template packs;
- [ ] restricted-content minimization tests for subjects, previews, logs and analytics.

### P6 — MFA, sessions and step-up assurance

- [ ] assurance-policy model for commands and record classes;
- [ ] enterprise IdP step-up integration;
- [ ] WebAuthn/passkey enrolment and challenge;
- [ ] TOTP enrolment, verification and encrypted secret storage;
- [ ] one-time recovery codes and dual-controlled administrative recovery;
- [ ] active session/device management and security-event history;
- [ ] fresh-authentication enforcement for material approvals, signatory actions, restricted access, export and break glass.

### P7 — Illustration, iconography and complete state system

- [ ] production light/dark illustration package;
- [ ] complete semantic icon family;
- [ ] onboarding, no-work, no-change, no-result, not-configured, source-unavailable, expired-invitation, submitted, routing, readiness, protected-reporting, MFA, import and notification-failure illustrations;
- [ ] explicit absence/no-result/unauthorized/unavailable/stale/completed empty-state taxonomy;
- [ ] bundle-size, accessibility and visual-regression checks.

### P8 — Pilot hardening and release evidence

- [ ] production object storage, malware scanning, retention and legal hold;
- [ ] database defense-in-depth, secrets management and key rotation;
- [ ] backup/restore, point-in-time recovery and provider-outage exercises;
- [ ] identity sync, notification, worker and projection operational dashboards;
- [ ] representative identity, authority, Today, notification and 100,000-row workload evidence;
- [ ] retained p95/p99 results and query plans;
- [ ] pilot-bank legal configuration, role mapping, template approval and security policy;
- [ ] governed go-live and rollback checklist.

## Release gates

Every completed item still requires relevant authorization, invitation, source, performance, recovery, accessibility and workload tests before production release.

### Existing foundation gates

- [x] Projection-first Program and Matter summaries, indexed search, keyset pagination and lazy detail loading.
- [x] Signed request actor binding and authority checks for material Program/Matter commands.
- [x] Separately versioned Program-status maintenance with health, reconciliation and governed rebuild operations.
- [x] Cross-tenant query-scope rejection and fail-closed restricted-read tests.
- [x] Partial reference-install recovery and current-record negative tests.
- [x] Governed document-import tenant isolation, optimistic review, demo-mode separation and rendered accessibility tests.

### Enterprise productization gates

- [ ] Direct enterprise identity-provider integration, gateway key rotation and organization synchronization.
- [ ] LDAP/AD or equivalent directory reconciliation and deactivation recovery evidence.
- [ ] Full mutation authorization matrix for governance, Programs, Matters, evidence and configuration.
- [ ] Business-calendar escalation, fallback and no-valid-route acceptance evidence.
- [ ] Multi-role first-run guides reaching meaningful real tasks.
- [ ] Notification minimization, delivery retry and dead-letter operating evidence.
- [ ] WebAuthn/TOTP, recovery, session revocation and material-command step-up evidence.
- [ ] Complete light/dark state coverage, accessibility and retained visual-regression evidence.
- [ ] Production object-store, scanning, retention and legal-hold evidence.
- [ ] PDF/OCR extraction-provider security, resource limits and source-anchor acceptance evidence.
- [ ] Representative 100,000-row p95/p99 release evidence and retained query plans.
- [ ] Backup, restore, provider outage and safe degraded-mode exercises.
- [ ] Pilot-bank configuration approval and governed go-live record.
