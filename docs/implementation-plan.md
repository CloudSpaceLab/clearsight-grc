# ClearSight implementation ledger

**Status date:** 2026-08-07  
**Repository baseline:** `main@df98a7f66c28642637a45a10662abac042dcd144` (PR #25 merged)

This is the authoritative execution ledger for current repository work. Detailed product, design and enterprise-productization documents remain reference specifications, but they do not override the order below.

The current sequence is driven by two linked issues:

- **#26 — executable integrity and continuous evidence reconciliation**: closes the remaining authority, transaction and API-contract seams that could make successful operations unsafe or misleading.
- **#27 — agent-managed compliance and human decision packets**: defines the future operating model, but governed execution must reuse #26 identity, authority, worker and audit foundations.

## 1. Canonical sequencing

Completed by PR #25:

1. [x] executable HTTP route classification and verified actor/tenant binding;
2. [x] persisted capture consolidation and bounded capture security fixes;
3. [x] source-health reconciliation through the existing outbox/inbox, Signal/Drift and Program paths;
4. [x] worker work-class isolation and bounded poison handling.

Remaining #26 closure sequence:

5. [ ] **P0.4 — compound-command and command-response truth**;
6. [ ] **P0.5 — runtime route / OpenAPI / browser-client contract reconciliation**;
7. [ ] **P0.6 — effective authority convergence plus governed configuration authorization/bootstrap**;
8. [ ] disposition still-valid lower-priority #26 audit findings into linked follow-up issues;
9. [ ] exact-head CI plus PostgreSQL degraded-path/fault evidence for the completed closure work.

Only after 5–9 may #27 governed operator execution, execution receipts and intervention mutations proceed.

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
- [x] typed runtime route registry with explicit access classes.
- [x] deterministic routing-policy resolution foundation, simulation, delegation persistence and segregation foundations.
- [x] governance policy/delegation maker-checker lifecycle and append-only decisions.
- [x] material Program/Matter command authorization boundary.
- [x] restricted Matter reads fail closed.

**Not yet complete:** the effective authority decision does not yet converge routing rules, assignments, grants, active delegations/substitutions and segregation constraints. See P0.6.

### Workflow, delivery and projections

- [x] durable Workflow Tasks, events and optimistic transitions.
- [x] leased timers, outbox/inbox primitives and retry/recovery foundations.
- [x] independent in-process worker classes isolate source maintenance, Program projection, delegation lifecycle, timers and outbox delivery.
- [x] timer/outbox poison work terminates visibly after a bounded retry budget.
- [x] separately versioned Program-status maintenance with health, reconcile and rebuild operations.
- [x] projection-first Program/Matter summaries with bounded pagination and lazy details.

### Evidence and capture

- [x] evidence-source health and freshness model.
- [x] one persisted Evidence Request capture domain for demo and production paths; the parallel `internal/capture` implementation is removed.
- [x] unused foundation `evidence_requests` / `invitation_grants` tables are removed by migration `000013_capture_consolidation`.
- [x] request deadline, request expiry/open-state, artifact-open-state and invitation audience-hash checks are enforced.
- [x] bounded external invitation/session capability and bearer CORS support.
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

## 3. #26 P0 seam-integrity status

### P0.1 — executable route/identity boundary — COMPLETE IN PR #25

- [x] one typed HTTP route registry owns route class, access boundary and material-command policy.
- [x] only health endpoints are true public routes.
- [x] bounded invitation/session routes remain capability-scoped instead of requiring staff identity.
- [x] all other protected routes require verified actor context.
- [x] tenant query/body scope is rebound from verified identity and conflicting scope is rejected.
- [x] authenticated JSON writes bind tenant and relevant actor fields from verified identity.
- [x] command identity binding remains enforced when command-authorization mode is off.
- [x] material command actor fields are DTO/descriptor-driven.
- [x] legal-entity scope is an authorization input and is injected only where supported.
- [x] route classification and CORS have adversarial tests.

Direct OIDC/SAML, directory synchronization, RLS/ABAC defense in depth and step-up assurance remain enterprise-release work, not #26 closure blockers.

### P0.2 — evidence event reconciliation — COMPLETE IN PR #25

- [x] `SourceHealthChanged` is consumed internally before publication completion.
- [x] inbox receipts make completed internal delivery idempotent.
- [x] degradation and recovery update the exact source-quality drift.
- [x] generic signal ingestion cannot forge source recovery.
- [x] active Evidence Contract mappings and effective approved Requirement `source_id` mappings resolve affected Programs.
- [x] Program trigger dedupe includes source event and Program ID.
- [x] unhealthy→unhealthy does not create duplicate degradation episodes.
- [x] recovery reaches a Program only when every currently-required source is healthy.
- [x] existing transactional trigger bundle owns Program trigger, optional Matter, outbox and projection-job truth.
- [x] failed internal delivery is retried rather than logged-and-marked-published.

A representative full replay/lag/recovery PostgreSQL test remains a production-release hardening item if not already covered by the final P0.4–P0.6 integration suite.

### P0.3 — worker work-class isolation — COMPLETE IN PR #25

- [x] source maintenance, Program projection, delegation lifecycle, workflow timers and outbox delivery run as named independent work classes.
- [x] each class has its own interval, timeout, lease, retry/backoff ceiling, batch limit and retry budget.
- [x] class errors/panics do not stop unrelated classes.
- [x] timeout remains shorter than claim lease.
- [x] publisher panics are contained per outbox item.
- [x] in-memory and PostgreSQL lease semantics agree.
- [x] exhausted timers become `FAILED`; exhausted outbox events become dead-lettered and neither is reclaimable.
- [x] queue/class health exposes actionable degraded state.
- [x] shared-database PostgreSQL integration packages run serially in CI.

### P0.4 — compound-command and command-response truth — ACTIVE

PR #25 already added a narrow `VerificationResultBundle` and PostgreSQL transaction for failed verification consequences. REOPEN/ESCALATE transitions and CREATE_MATTER follow-up/link creation now commit atomically with the verification result, and linked-Program lookup errors are propagated.

The remaining problem is broader **post-commit response truth**: many continuity mutators still commit and then call `GetProgram`, `GetMatter` or a derived refresh/replay. A failure in that later read can still report an error for an already committed command.

#### Subtasks

- [x] failed verification result + required same/cross-aggregate consequence commits atomically in PostgreSQL.
- [x] CREATE_MATTER verification handling propagates `LinkedProgramIDs` failures.
- [ ] define one small `CommandReceipt`/commit-result contract carrying aggregate ID, committed version, event/command identity and any durable follow-up state needed by the caller.
- [ ] return the receipt/current authoritative mutation result from the transaction boundary; do not make command success depend on full aggregate replay.
- [ ] audit Program mutators using `refreshAndGetProgram`, including create, transition, requirements, applicability, safeguards and evidence assessments.
- [ ] audit Matter mutators returning `GetMatter`, including create-with-link, lifecycle, links, decisions, actions, verification contracts/results and response packages.
- [ ] audit `ApplyTriggerBundle` and any other multi-aggregate path for the same post-commit read failure mode.
- [ ] keep required projection jobs/outbox rows in the authoritative transaction; make optional derived refresh explicitly best-effort after commit.
- [ ] add repository/service fault injection at: pre-write lookup, first authoritative write, later bundled write, transaction commit, post-commit refresh and post-commit detail read.
- [ ] PostgreSQL tests must prove rollback before commit, atomic bundle commit, no false failure after commit, idempotent retry/dedupe and correct returned committed version.

Do **not** introduce a generic transaction coordinator or orchestration framework.

### P0.5 — API / OpenAPI / browser-client contract reconciliation — ACTIVE

The route registry is now the executable runtime route inventory, but the published/client contract still drifts:

- `api/openapi.yaml` does not describe the production authentication/security boundary;
- runtime concrete governance transition routes are represented as generic `/{id}/{action}` operations;
- several client-facing schemas still require `tenant_id`/actor fields even though verified identity owns those values server-side;
- `web/src/api.ts` still injects tenant query/body scope from `/context`;
- split OpenAPI documents and handwritten TypeScript models can diverge from runtime routes/DTOs without failing CI.

#### Subtasks

- [ ] nominate one canonical API contract source and define whether split specs are generated overlays or separately verified modules.
- [ ] add OpenAPI security schemes/requirements matching PUBLIC, authenticated and bounded-capability route classes.
- [ ] publish exact runtime paths/methods; remove pseudo-paths that do not exist at runtime.
- [ ] make server-owned tenant/principal/actor fields non-required or absent from client request schemas; retain them only where they are genuine domain data rather than authority-bearing identity.
- [ ] update the browser client to stop manufacturing authority-bearing tenant/actor scope from `/context` for protected operations.
- [ ] reconcile query parameters, status/error shapes, request/response DTOs and content types across route registry, handlers and OpenAPI.
- [ ] generate TypeScript transport types/client code from the canonical contract, or add a strict mechanical drift check if generation would add unjustified tooling/bloat.
- [ ] add CI parity tests: every registered runtime route is represented, every documented route exists, and every browser endpoint/schema used by production code matches the canonical contract.
- [ ] include security-negative contract tests for missing identity, bounded capability, cross-tenant scope and actor spoofing.

### P0.6 — effective authority convergence and governed configuration authorization — ACTIVE

The current command guard is structurally correct but the authority service behind it is incomplete. PostgreSQL authority resolution still loads active routing-policy JSON and resolves selectors separately before selecting one principal. The material command decision does not yet converge persisted responsibility assignments, authority grants, active delegations/substitutions and segregation constraints.

Governance policy/delegation writes are actor-bound and permission-gated, but they remain authenticated configuration writes rather than a complete governed configuration-command matrix with safe bootstrap semantics.

#### Subtasks

- [ ] define the effective-authority decision contract across routing rules, responsibility assignments, authority grants, active delegations/substitutions and segregation/conflict constraints.
- [ ] implement one indexed effective-authority read model/query path; do not add a parallel generic RBAC engine.
- [ ] replace per-rule selector N+1 resolution with bounded queries keyed by tenant, legal entity, object/scope, responsibility, decision class, materiality and effective time.
- [ ] represent candidate sets, TEAM/QUEUE/COMMITTEE and substitution semantics explicitly instead of reducing every route to one arbitrary occupant.
- [ ] define deterministic specificity and ambiguity rules for wildcard/specific overlap; activation/simulation must reject unresolved ambiguity.
- [ ] evaluate delegation scope/expiry, authority limits/materiality, conflicts and segregation again at material execution time.
- [ ] define configuration bootstrap/break-glass semantics that do not hard-code `CRO`, `GRC_ADMIN` or similar executive role names.
- [ ] map every governance/configuration mutation to required capability, responsibility, scope, assurance and audit event; keep maker/checker rules as additional constraints, not the complete authorization decision.
- [ ] implement draft editing/new policy version, simulation, impact preview, approval, scheduled activation, supersession and rollback.
- [ ] ensure authority resolve/simulate remain verified-tenant bound and detailed administrative views require governed configuration permission.
- [ ] add end-to-end tests proving valid delegation can execute and expired/revoked/out-of-scope delegation cannot; prove segregation and materiality/grant limits deny correctly.
- [ ] add route-coverage CI ensuring every material/configuration mutation has an explicit authorization contract.
- [ ] add query-count and representative ~100k rule/assignment benchmark evidence showing authority decision cost is bounded by the requested scope rather than tenant-wide rule count.

## 4. #26 closure rule

Issue #26 may close only when:

- [ ] P0.4, P0.5 and P0.6 subtasks above are complete;
- [ ] the issue contains links to exact tests/PRs for those closures;
- [ ] every still-valid P1/P2 observation from the original audit/addendum is either fixed, explicitly superseded by current architecture, or moved to a linked follow-up issue with preserved acceptance criteria;
- [ ] stale issue comments/docs are marked historical rather than used as current implementation instructions;
- [ ] exact-head GitHub CI is green and the relevant PostgreSQL integration/fault tests actually ran on that head.

This keeps #26 finite without silently losing lower-priority findings.

## 5. #27 gate — governed operator execution and receipts

Only after #26 closes:

- implement the bounded operator executor;
- evaluate active Automation Policy at execution time;
- enforce action class, eligibility, blast radius, reversibility, expiry and verification contract;
- persist execution/audit receipts from the actual executor;
- expose those receipts in Intervention Summaries;
- add governed accept/edit/reject/request-evidence/escalate command surfaces where domain commands exist;
- reopen work when independent verification fails.

Do not add a receipt table or UI merely to make the product appear agentic before there is a trustworthy writer.

## 6. Current truth for Today and automation

### Today

Production/non-demo Today is currently the actor-facing projection of **active Workflow Tasks explicitly assigned to the verified principal**. Completed and cancelled work is excluded. Team/unassigned work remains in its routing queue until resolved to a principal.

Demo/reference mode may project reference journeys for stakeholder presentation.

Today is not yet the complete event-driven intervention compiler described by #27. That broader compilation remains downstream work and must not create another task model.

### Automation policy

The existing `automation_policies` model is readable and visible in Configure. It represents a governed boundary, not proof of execution.

A policy being listed does not mean the policy is currently active/effective, an operator action ran, the action succeeded, or the outcome was verified. Those claims require executor and verification receipts after #26.

## 7. Semantic invariants

Every future implementation and plan MUST preserve these distinctions:

- **Matter Action ≠ Workflow Task.** Action is domain truth; Task is routed human work.
- **Signal ≠ incident/conclusion.** A Signal is an observation that may trigger deterministic assessment.
- **Submission ≠ sufficient evidence.** Captured data still requires the Evidence Contract assessment.
- **Implementation ≠ verified outcome.** Completion is not closure.
- **Recommendation ≠ approval.** Human authority remains explicit where policy requires it.
- **Automation Policy ≠ execution receipt.** Permission is not proof of action.
- **Intervention Summary ≠ authoritative state.** It is an actor-facing read projection over canonical records.

## 8. Enterprise work after #26

Detailed identity, RBAC, notification, MFA, visual-system and pilot-hardening requirements remain in `docs/engineering/enterprise-productization-implementation-plan.md` and the product/design specifications.

That document is a detailed requirements reference. **This ledger controls current execution order** whenever sequencing differs.

Major later enterprise gates include:

- OIDC/SAML/SCIM/LDAP or equivalent controlled identity synchronization;
- source-backed organization/position lifecycle;
- production notification delivery and privacy-minimized templates;
- WebAuthn/TOTP/session management and policy-bound step-up;
- production object storage, malware scanning, retention and legal hold;
- PDF/OCR extraction-provider isolation;
- representative workload evidence and retained query plans;
- backup/restore/provider-outage exercises;
- pilot-bank legal/configuration approval and governed go-live.

## 9. Release and validation rules

Checkboxes describe repository capability, not production readiness.

A tranche is not complete until its relevant tests execute successfully. Required gates include:

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
