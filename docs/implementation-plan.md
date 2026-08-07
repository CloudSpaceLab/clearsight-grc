# ClearSight implementation ledger

**Status date:** 2026-08-07  
**P0 closure PR:** #30  
**P1.1 implementation PR:** #34

This is the authoritative execution ledger. Product, design, architecture and enterprise-productization documents define requirements, but this file controls current implementation order and capability truth.

## 1. Current sequence

### #26 P0 executable integrity — complete in PRs #25 and #30

- [x] typed route/access registry and verified actor/tenant binding;
- [x] persisted capture consolidation and bounded invitation/session security;
- [x] durable source-health reconciliation through outbox/inbox → Signal/Drift → affected Program trigger/Matter → projection;
- [x] independent worker work classes with bounded retry/dead-letter behavior;
- [x] compound-command and post-commit response truth;
- [x] executable route/OpenAPI contract parity;
- [x] bounded effective-authority convergence across routing rules, assignments, grants, active delegations and segregation constraints;
- [x] lower-priority audit findings moved to #32 P1 and #33 P2 before #26 closed.

### #32 P1 semantic/current-state correctness

P1 is correctness-first rather than feature-first.

1. **P1.1 Program-state truth — IMPLEMENTED IN PR #34**
   - [x] currently-effective Requirement, Applicability and Control Implementation selection;
   - [x] Evidence Contracts affect current state only when their linked Requirement/Control target is currently effective;
   - [x] Evidence Assessment validity is bounded by the Evidence Contract freshness interval in both state derivation and PostgreSQL persistence;
   - [x] actual current required evidence-source health drives Source Quality rather than the latest source trigger alone;
   - [x] future Requirement sources do not enter the current source denominator;
   - [x] mandatory UNKNOWN dimensions, including Assurance and Source Quality, cannot become overall `CURRENT`;
   - [x] pause/resume preserves the configured Program `effective_until`, including reconstruction of historical resume events that cleared it;
   - [x] Program summaries expose command Program version, assessed Program version, projection version, stale state and reason totals/omissions;
   - [x] stale last-known green snapshots render as `Updating status` and do not contribute to the UI current count;
   - [x] PostgreSQL and rendered-state tests cover temporal, multi-source, freshness, stale-projection and pause/resume behavior.
2. **P1.2 Matter closure current-record truth — NEXT**
   - current Decision/Response selection and supersession;
   - expiry/conditions at closure;
   - verification independence and observation-period enforcement.
3. **P1.3 lifecycle-specific command responsibility**
   - proposer/reviewer/challenger/authorizer/signatory/transmitter/acknowledgement responsibilities vary by requested transition rather than static command name.
4. **P1.4 bounded ordinary reads and explicit work-model projection contracts.**
5. **P1.5 document-import resource, durability and paging hardening.**

Only after semantic/current-state correctness should wider #27 governed operator execution be treated as trustworthy execution rather than presentation.

## 2. Canonical domain invariants

These distinctions are mandatory:

- **Program** = ongoing obligation/compliance continuity.
- **Matter** = bounded change, exception, finding, decision, action, response or verification case.
- **Matter Action ≠ Workflow Task.** Action is accountable domain work; Task is an actor-facing routed step.
- **Signal ≠ conclusion.** A Signal is an observation that deterministic assessment may convert into drift or attention.
- **Submission ≠ sufficient evidence.** Evidence Contract assessment determines sufficiency.
- **Implementation ≠ verified outcome.** Completion alone cannot close material work.
- **Recommendation ≠ approval.** Current authority remains explicit.
- **Automation Policy ≠ execution receipt.** Permission is not evidence that an action ran.
- **Intervention Summary ≠ authoritative state.** It is a read projection over canonical records.

Do not add a parallel authorization, task, event, worker, receipt or generic workflow stack that duplicates these foundations.

## 3. P0 closure truth

### Route and identity boundary

`internal/httpapi/route_registry.go` is the canonical executable route inventory.

- [x] every production route has one explicit access class;
- [x] only health routes are truly public;
- [x] bounded capture routes use capability access;
- [x] other protected routes require verified actor context;
- [x] tenant/legal-entity and actor-bearing fields are rebound server-side where authoritative;
- [x] material commands resolve current authority at execution;
- [x] registry tests fail on invalid/unclassified route definitions.

Direct OIDC/SAML, directory synchronization, RLS/ABAC defense in depth and step-up assurance remain enterprise-release work, not P0 blockers.

### Capture consolidation/security

- [x] one persisted Evidence Request/Invitation/Session/Submission/Artifact domain remains;
- [x] parallel `internal/capture` implementation is removed;
- [x] migration `000013_capture_consolidation` removes abandoned foundation request/invitation tables;
- [x] invitation audience hash is verified at redemption;
- [x] request deadlines/open-state and artifact upload state are enforced;
- [x] request expiry is maintained;
- [x] bearer capture CORS is supported.

### Durable source-health reconciliation

- [x] `SourceHealthChanged` is internally consumed before publication completion;
- [x] inbox receipts provide idempotent internal delivery;
- [x] degradation/recovery updates the exact source-quality drift;
- [x] recovery cannot be forged through generic Signal ingestion;
- [x] active Evidence Contracts and currently-effective approved Requirement source mappings resolve affected Programs;
- [x] one unhealthy source cannot be cleared by recovery of a different required source;
- [x] Program trigger/Matter/projection consequences reuse existing transactional continuity paths;
- [x] PostgreSQL replay tests exercise active Program degradation, duplicate delivery and recovery.

### Worker failure isolation

- [x] evidence-source maintenance, Program projection, delegation lifecycle, workflow timers and outbox delivery run independently in one deployable worker;
- [x] each class owns its interval, timeout, lease, batch, retry and backoff budget;
- [x] ordinary class failures/panics do not terminate unrelated work;
- [x] exhausted timers become `FAILED` and exhausted outbox events become dead-lettered;
- [x] terminal work is not reclaimable;
- [x] queue/class health exposes actionable lag/failure state.

### Transaction and command-response truth

Failed verification consequences use `VerificationResultBundle` to commit the verification result together with required REOPEN/ESCALATE or CREATE_MATTER/link effects.

PR #30 additionally guarantees:

- [x] normalized Program/Matter version probes determine authoritative commit outcome without replaying event history;
- [x] a material update that commits but later fails response reconstruction returns a small `COMMITTED` degraded-response receipt rather than a false 5xx;
- [x] genuine pre-commit failures remain failures;
- [x] API PostgreSQL composition preserves only a just-committed create result as a short-lived non-authoritative response fallback;
- [x] no second durable receipt or transaction-orchestration framework was introduced.

Canonical transaction rule:

```text
identity/current-authority check
→ authoritative row change
→ append-only domain event
→ transactional outbox / required maintenance job
→ commit
→ optional detail/projection read
```

The optional final read may degrade response detail; it may not reverse or misreport the commit.

### Executable API contract parity

`api/runtime.openapi.json` is the mechanically verified production route/access contract.

- [x] exact method/path inventory matches `route_registry.go`;
- [x] access class and administrative permission match mechanically;
- [x] signed identity and bounded capture security schemes are explicit;
- [x] capability and authenticated-or-capability routes are distinguishable;
- [x] CI fails when a production route is added/removed/reclassified without contract update;
- [x] CI uses `npm ci` against `web/package-lock.json`.

Generated browser transport types/client consolidation remains a later custom-code deletion opportunity.

### Effective authority convergence

Migration `000014_effective_authority_routes` materializes currently-effective approved routing-policy rules into an indexed execution read model.

Production authority now:

1. resolves matching compiled routes and current responsibility assignments in one bounded ranked query;
2. applies deterministic priority + specificity and fails closed on same-rank ambiguity;
3. expands active scoped delegation chains with cycle/depth protection;
4. enforces applicable authority-grant materiality limits;
5. applies active segregation constraints;
6. returns an explicit candidate set when several humans are currently eligible.

Unresolved selectors are excluded from execution but remain visible as integrity findings. Collective TEAM/QUEUE/COMMITTEE routes remain collective/candidate semantics rather than arbitrary `LIMIT 1` selection.

## 4. P1.1 Program-state truth

### Current-time selection

The Program-state projection now evaluates current truth at projection time:

- future Requirements are excluded until `effective_from`;
- expired Requirements/Applicability/Control Implementations are excluded after `effective_until`;
- the latest currently-effective Applicability record is selected deterministically;
- inactive/future implementation targets cannot satisfy current implementation state;
- active Evidence Contracts are considered only when their linked Requirement or Control Implementation is currently effective;
- future Evidence Assessments do not affect current state.

### Evidence freshness

Evidence Assessment validity cannot outlive the Evidence Contract freshness interval.

- the state engine always evaluates `min(valid_until, assessed_at + freshness)`;
- migration `000015_program_state_truth` derives missing validity and safely caps an over-long supplied validity at the governed maximum;
- non-positive validity windows remain invalid;
- bank-reference seed/repair data now uses the same contract freshness rule rather than a hard-coded future date.

### Source Quality

The PostgreSQL projection reads the authoritative current source denominator rather than inferring whole-Program source health from the latest source trigger.

Current dependencies include:

- source IDs on currently-effective approved Requirements;
- sources on ACTIVE Evidence Contracts whose target Requirement/Control Implementation is currently effective.

A Program is source-current only when **every** currently-required active Evidence Source reports `CURRENT`. With no source dependency, Source Quality is explicitly `NOT_APPLICABLE`. A future Requirement source does not pollute the current denominator.

### Unknown and overall state

All compliance dimensions now participate in overall-state selection. `Assurance=UNKNOWN` or `SourceQuality=UNKNOWN` cannot silently produce `CURRENT`.

### Program period and reconstruction

Migration `000015` preserves a configured `effective_until` when a PAUSED Program resumes. Event replay applies the same semantic correction to historical resume payloads that wrote `effective_until=null`, keeping normalized current state and reconstructed history aligned.

### Projection freshness and UI

Program summaries expose:

- `program_version`;
- `assessed_program_version`;
- `projection_version`;
- `projection_stale`;
- state generation time;
- total and omitted status-reason counts.

A stale last-known `CURRENT` snapshot is rendered as **Updating status**, counted as reassessing rather than current, and shows the assessed/current versions. Complete reasons remain available in Program detail; summary truncation is never silent.

## 5. Current Today and automation truth

### Today

Non-demo Today currently projects active Workflow Tasks explicitly assigned to the verified principal. Completed/cancelled work is excluded. Team/unassigned work remains in its routing queue until resolved.

Demo/reference journeys may seed stakeholder presentation data, but are not production Today truth.

Today is not yet the complete event-driven intervention compiler envisioned by #27.

### Automation policy

`automation_policies` currently exposes governed eligibility/configuration boundaries. A visible policy does **not** prove that an automated action ran, succeeded or was independently verified.

Those claims require a real governed executor and persisted execution/verification evidence.

## 6. Enterprise work after semantic P1

Detailed enterprise requirements remain in `docs/engineering/enterprise-productization-implementation-plan.md` and product/design specifications.

Major later gates include:

- OIDC/SAML/SCIM/LDAP or equivalent controlled identity synchronization;
- source-backed organization/position lifecycle;
- complete configuration administration and rollback UX;
- production notifications and privacy-minimized templates;
- WebAuthn/TOTP/session management and policy-bound step-up;
- production object storage, malware scanning, retention and legal hold;
- PDF/OCR provider isolation;
- representative capacity evidence and retained query plans;
- backup/restore/provider-outage exercises;
- pilot-bank legal/configuration approval and governed go-live.

## 7. Release and validation rules

Checkboxes describe repository capability, not deployment readiness.

A tranche is not complete until the relevant executable gates pass on its exact head:

- `gofmt` and `go vet`;
- race-enabled Go tests;
- PostgreSQL composition, migrations and integration tests;
- TypeScript strict checking;
- Vitest/axe rendered-state tests;
- production Vite build;
- adversarial identity, tenant, authority, replay and degraded-path tests;
- representative performance/recovery evidence when cardinality or durability changes.

Never claim a branch or PR is CI-green based on an older commit.