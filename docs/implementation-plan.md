# ClearSight implementation ledger

**Status date:** 2026-08-07

This is the authoritative execution ledger for current repository work. Detailed product, design and enterprise-productization documents remain reference specifications, but they do not override the order below.

The current sequence is driven by two linked issues:

- **#26 — executable integrity and continuous evidence reconciliation**: closes architectural seams that can make the repository appear complete while runtime behavior is incomplete or unsafe.
- **#27 — agent-managed compliance and human decision packets**: defines the future operating model, but its remaining governed-execution work depends on the #26 seams rather than building a parallel authorization, worker or audit layer.

## 1. Canonical sequencing

Remaining #27 operator execution MUST wait for the shared #26 foundations in this order:

1. HTTP route classification, verified actor/tenant binding and write-route integrity.
2. Evidence event reconciliation through durable outbox/inbox delivery.
3. Worker work-class isolation and bounded failure domains.
4. Compound-command transaction truth and recovery.
5. API/query/OpenAPI/client-contract reconciliation.
6. Complete governance/configuration authorization matrix.
7. Governed operator execution, receipts and intervention mutations from #27.

This order prevents duplicate auth middleware, duplicate task/execution models and frontend-only autonomy controls.

## 2. Completed foundation

### Canonical domain

- [x] Programs model ongoing obligations and compliance continuity.
- [x] Matters model bounded change, exception, findings, decisions, actions, response and verification.
- [x] Evidence Sources, Requests, Submissions and artifacts have tenant-scoped repositories and explicit lifecycle semantics.
- [x] Matter Actions remain domain remediation/commitment truth.
- [x] Workflow Tasks remain actor-facing routed/manual work; they are not a second Matter Action model.
- [x] Signals remain observations; deterministic assessment decides whether they create drift or attention.
- [x] implementation and verified outcome remain separate states.

### Identity, authority and governance

- [x] verified actor context with tenant, legal entity, principal, actor kind and assurance metadata.
- [x] deterministic authority resolution, simulation, delegation and segregation foundations.
- [x] governance policy/delegation maker-checker lifecycle and append-only decisions.
- [x] material Program/Matter command authorization foundation.
- [x] restricted Matter reads fail closed.

### Workflow, delivery and projections

- [x] durable Workflow Tasks, events and optimistic transitions.
- [x] leased timers, outbox/inbox primitives and retry/recovery foundations.
- [x] independent in-process worker classes isolate source maintenance, Program projection, delegation lifecycle, timers and outbox delivery.
- [x] timer/outbox poison work terminates visibly after a bounded retry budget instead of retrying forever.
- [x] separately versioned Program-status maintenance with health, reconcile and rebuild operations.
- [x] projection-first Program/Matter summaries with bounded pagination and lazy details.

### Evidence and capture

- [x] evidence-source health and freshness model.
- [x] focused internal/external capture with bounded invitation/session capability.
- [x] immutable submission records and artifact integrity metadata.
- [x] governed document import with deterministic extraction, source anchors and explicit proposal review.

### Agentic human experience

- [x] intervention-first Today UI.
- [x] non-demo Today projects active Workflow Tasks assigned to the verified principal.
- [x] exact Program/Matter/Evidence targets remain reachable outside first-page pagination.
- [x] Programs, Matters, Evidence and Imports use progressive disclosure rather than record walls.
- [x] evidence capture uses `enter → review exact assertions → submit → receipt`.
- [x] structured intervention/recommendation/verification/read-receipt contracts exist without fabricating operator execution.
- [x] automation-policy read visibility exposes governed eligibility, blast radius and verification boundaries in Configure.

## 3. Active P0 seam-integrity sequence

### P0.1 — executable route/identity boundary — IMPLEMENTED IN PR #25

- [x] one typed HTTP route registry owns route class, access boundary and material-command policy.
- [x] only health endpoints are true public routes.
- [x] bounded invitation/session routes remain capability-scoped instead of requiring staff identity.
- [x] all other protected routes require verified actor context.
- [x] tenant query scope is rebound from verified identity and conflicting tenant scope is rejected.
- [x] authenticated JSON writes bind tenant and relevant actor fields from verified identity.
- [x] command actor binding happens even when command-authorization mode is off; mode changes authority lookup, not identity truth.
- [x] material command actor fields are descriptor-driven so strict JSON decoding receives only fields supported by each DTO.
- [x] legal-entity scope is an authorization input and is injected into a request body only when that body declares it.
- [x] public/capability classification has adversarial registry tests.
- [x] CORS allows `Authorization` for bearer capture sessions.

**Still open within the wider identity boundary:** direct OIDC/SAML integration, directory synchronization, RLS/ABAC defense in depth, step-up assurance and a complete authorization matrix for governance/configuration writes.

### P0.2 — evidence event reconciliation — IMPLEMENTED IN PR #25

The first durable reconciliation bridge now reuses the existing evidence outbox, runtime inbox, compliance Signal/Drift model and Program trigger path rather than adding another event system.

- [x] `SourceHealthChanged` is consumed internally before the outbox event can be marked published.
- [x] inbox receipts make completed internal delivery idempotent.
- [x] source degradation creates/upserts the exact `source_quality` drift.
- [x] source recovery resolves only the matching active source drift.
- [x] generic `/compliance/signals` ingestion cannot forge `SOURCE_RECOVERED`; recovery is an internal governed-source operation.
- [x] active Evidence Contract mappings and currently-effective approved Requirement `source_id` mappings resolve every affected Program without a silent fan-out cap.
- [x] Program trigger dedupe keys include both source event and Program ID.
- [x] unhealthy→unhealthy changes update drift but do not open a duplicate degradation episode/Matter.
- [x] recovery reaches a Program only when every currently-required source for that Program is healthy.
- [x] existing transactional `ApplyTriggerBundle` remains responsible for Program trigger, optional Matter, outbox and projection-job truth.
- [x] logging occurs after internal reconciliation; a failed internal delivery is retried instead of being logged-and-marked-published.
- [x] no reconciliation-specific schema migration or broker was added.

**Still required before production release:** one full PostgreSQL replay test proving the complete source-health outbox → inbox → Signal/Drift → Program trigger/Matter → projection consequence chain under duplicate delivery and lag/recovery conditions.

### P0.3 — worker work-class isolation — IMPLEMENTED IN PR #25

The deployable worker remains one process, but its independent responsibilities no longer share one serial failure domain.

- [x] evidence-source maintenance, Program projection, delegation lifecycle, workflow timers and outbox delivery run as five named work classes.
- [x] each class has an independent poll interval, execution timeout, lease, retry/backoff ceiling, batch limit and retry budget.
- [x] each class is intentionally single-flight; no generic worker-pool or agent-executor abstraction was added.
- [x] class errors and panics degrade only that class; ordinary retriable failures do not terminate the worker process or stop unrelated classes.
- [x] execution timeout is always shorter than the claim lease, preventing normal timeout handling from making live work simultaneously reclaimable.
- [x] publisher panics are contained to the affected outbox item so later items in the claimed batch continue.
- [x] in-memory outbox claiming now honours leases consistently with PostgreSQL.
- [x] workflow timers move to durable `FAILED` after their retry budget is exhausted and are no longer claimable.
- [x] outbox events receive `dead_lettered_at` after their retry budget is exhausted and are no longer claimable.
- [x] class health distinguishes `STARTING`, `CURRENT`, `DEGRADED` and `NEEDS_ATTENTION`; timer/outbox health exposes pending work, terminal work, highest attempts and oldest pending time from the authoritative queue tables.
- [x] shared-database `postgresintegration` packages run serially in CI so one package's fixture cleanup cannot race another package.
- [x] PostgreSQL integration tests prove terminal timer/dead-letter behavior and no reclaim after terminal failure.

### P0.4 — compound-command transaction truth — NEXT

Audit every compound material mutation. Authoritative rows, append-only domain event, transactional outbox and required maintenance job must commit together. Remove any path where a committed material mutation can return failure because a derived action failed afterward.

The first concrete target is `RecordVerificationResult`: it currently commits the verification result before REOPEN/ESCALATE/CREATE_MATTER follow-up handling. Cross-aggregate follow-up must become a durable instruction committed with the authoritative result, while same-aggregate required transitions must share one optimistic transaction. Repository errors used to determine follow-up scope must never be discarded.

### P0.5 — API and client contract reconciliation — PLANNED

Reconcile route registry, handler DTOs, query parameters, OpenAPI, browser API client and static stakeholder transport. Add contract tests so URL or field drift cannot survive compilation/render tests.

### P0.6 — governance/configuration authorization matrix — PLANNED

Extend current actor-bound maker-checker governance writes into a complete effective authorization matrix with configuration bootstrap semantics, simulation, impact preview, effective dating and rollback. Do not hard-code CRO/GRC-admin role names as authorization.

### P0.7 — governed operator execution and receipts — GATED BY P0.1–P0.6

Only after the shared seams above are executable:

- implement the bounded operator executor;
- evaluate active Automation Policy at execution time;
- enforce action class, eligibility, blast radius, reversibility, expiry and verification contract;
- persist execution/audit receipts from the actual executor;
- expose those receipts in Intervention Summaries;
- add governed accept/edit/reject/request-evidence/escalate command surfaces where domain commands exist;
- reopen work when independent verification fails.

Do not add a receipt table or UI merely to make the product appear agentic before there is a trustworthy writer.

## 4. Current truth for Today and automation

### Today

Production/non-demo Today is currently the actor-facing projection of **active Workflow Tasks explicitly assigned to the verified principal**. Completed and cancelled work is excluded. Team/unassigned work remains in its routing queue until resolved to a principal.

Demo/reference mode may project reference journeys for stakeholder presentation.

Today is not yet the complete event-driven intervention compiler described by #27. Source-health changes now reach canonical drift and dependent Program state through P0.2, but other signal classes and actor-facing intervention compilation remain explicit future work.

### Automation policy

The existing `automation_policies` model is readable and visible in Configure. It represents a governed boundary, not proof of execution.

A policy being listed does not mean:

- the policy is currently active/effective;
- an operator action ran;
- the action succeeded;
- the outcome was verified.

Those claims require executor and verification receipts from P0.7.

## 5. Semantic invariants

Every future implementation and plan MUST preserve these distinctions:

- **Matter Action ≠ Workflow Task.** Action is domain truth; Task is routed human work.
- **Signal ≠ incident/conclusion.** A Signal is an observation that may trigger deterministic assessment.
- **Submission ≠ sufficient evidence.** Captured data still requires the Evidence Contract assessment.
- **Implementation ≠ verified outcome.** Completion is not closure.
- **Recommendation ≠ approval.** Human authority remains explicit where policy requires it.
- **Automation Policy ≠ execution receipt.** Permission is not proof of action.
- **Intervention Summary ≠ authoritative state.** It is an actor-facing read projection over canonical records.

## 6. Enterprise work after P0 seam integrity

Detailed identity, RBAC, notification, MFA, visual-system and pilot-hardening requirements remain in `engineering/enterprise-productization-implementation-plan.md` and the product/design specifications.

That document is a detailed requirements reference. **This ledger controls current execution order** whenever its sequencing differs.

Major remaining enterprise gates include:

- OIDC/SAML/SCIM/LDAP or equivalent controlled identity synchronization;
- full responsibility/decision-authority matrix and configuration authorization;
- production notification delivery and privacy-minimized templates;
- WebAuthn/TOTP/session management and policy-bound step-up;
- production object storage, malware scanning, retention and legal hold;
- PDF/OCR extraction-provider isolation;
- representative workload evidence and retained query plans;
- backup/restore/provider-outage exercises;
- pilot-bank legal/configuration approval and governed go-live.

## 7. Release and validation rules

Checkboxes describe repository capability, not production readiness.

A tranche is not complete until its relevant tests and evidence execute successfully. Required gates include:

- `gofmt` and `go vet`;
- race-enabled Go tests;
- PostgreSQL migrations/composition/integration tests;
- TypeScript strict checking;
- Vitest and axe;
- production Vite build;
- rendered desktop/mobile evidence for material UI changes;
- adversarial identity, tenant, authority, replay and degraded-path tests;
- representative performance/recovery evidence where cardinality or durability changes.

Do not describe a PR as CI-green when GitHub has not run the checks for the exact head.
