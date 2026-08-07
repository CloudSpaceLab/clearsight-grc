# ClearSight implementation ledger

**Status date:** 2026-08-07  
**P0 closure PR:** #30  
**P1.1 implementation PR:** #34  
**P1.2 implementation PR:** #35  
**P1.3 implementation PR:** #36

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

1. **P1.1 Program-state truth — IMPLEMENTED IN PR #34; consistency fixes in PR #35**
   - [x] currently-effective Requirement, Applicability and Control Implementation selection;
   - [x] Evidence Contracts affect current state only when their linked Requirement/Control target is currently effective;
   - [x] Evidence Assessment validity is bounded by the Evidence Contract freshness interval in state derivation and both PostgreSQL/memory persistence;
   - [x] actual current required evidence-source health drives Source Quality rather than the latest source trigger alone;
   - [x] future Requirement/contract sources do not enter the current source denominator;
   - [x] mandatory UNKNOWN dimensions, including Assurance and Source Quality, cannot become overall `CURRENT`;
   - [x] pause/resume preserves the configured Program `effective_until`, including reconstruction of historical resume events that cleared it;
   - [x] Program summaries expose command Program version, assessed Program version, projection version, stale state and reason totals/omissions;
   - [x] stale last-known green snapshots render as `Updating status` and do not contribute to the UI current count;
   - [x] deterministic reference scenarios evaluate Program state and evidence-review steps at the state/simulation clock rather than process wall-clock time;
   - [x] PostgreSQL, memory and rendered-state tests cover temporal, multi-source, freshness, stale-projection and pause/resume behavior.
2. **P1.2 Matter closure current-record truth — IMPLEMENTED IN PR #35; adversarial fixes in PR #36**
   - [x] current Decision is selected from authoritative append/event order within each decision type; equal wall-clock timestamps cannot reorder lifecycle truth;
   - [x] a later rejection/return/expiry/supersession cannot be bypassed by an older approval in that lineage;
   - [x] Regulatory Change closure cannot use an unrelated favorable decision to mask another adverse current decision type;
   - [x] Authority Request closure requires every current response-package lineage to be coherently transmitted and acknowledged;
   - [x] expired exception authority cannot satisfy closure;
   - [x] conditional exception authority requires explicit structured condition resolution; free-text conditions are not treated as proof of satisfaction;
   - [x] latest verification PASS is revalidated at closure for assigned reviewer authority, action-owner independence, implementation chronology and observation-period completion;
   - [x] the same verification invariants are enforced when recording the result, so invalid future/premature/self-reviewed results cannot enter authoritative state;
   - [x] reference and integration producers use real implementation/contract chronology instead of fabricated future/past observations;
   - [x] adversarial unit and PostgreSQL reconstruction tests cover approved→rejected history, mixed decision types, multiple response lineages, expired/unresolved exception authority, withdrawn replacement response, premature/self review and valid independent PASS.
3. **P1.3 lifecycle-specific command responsibility — IMPLEMENTED IN PR #36**
   - [x] material command responsibility is derived after loading current Matter/Decision/Response state and the requested lifecycle target;
   - [x] Decision stages resolve distinct `PROPOSER`, `REVIEWER`, `INDEPENDENT_CHALLENGER` and `AUTHORIZER` responsibilities;
   - [x] Response stages resolve proposer/reviewer, `SIGNATORY`, `TRANSMITTER` and `ACKNOWLEDGEMENT_RECORDER` responsibilities;
   - [x] close/cancel/decision-required and reopen-from-closed Matter transitions require current authorizer responsibility;
   - [x] lifecycle validity is checked before authority execution, including when the authority guard is configured in audit/off mode;
   - [x] Decision and Response current records retain stage-specific actor identities while the append-only event envelope remains the trusted actor source;
   - [x] memory and replay reconstruct the same actor truth from event envelopes rather than trusting client actor fields;
   - [x] migration `000016_lifecycle_command_responsibility` persists/backfills lifecycle actors and extends Decision lifecycle states with `IN_REVIEW` and `CHALLENGED`;
   - [x] governance routing policies accept the new lifecycle responsibilities without adding a second authorization or workflow engine;
   - [x] unit and PostgreSQL integration tests cover the responsibility matrix, invalid transitions, event-order currentness and persisted actor reconstruction.
4. **P1.4 bounded ordinary reads and explicit work-model projection contracts — NEXT.**
   - includes detail-level projection-version parity, deterministic as-of closure preview where needed, and removal/reconciliation of remaining duplicated or stale client/read contracts.
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

The broad descriptive `api/openapi.yaml` is not the executable access contract. Remaining payload-schema/client duplication, including lifecycle enum reconciliation, is tracked in P1.4 rather than being treated as authorization truth.

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

The Program-state projection evaluates current truth at projection time:

- future Requirements are excluded until `effective_from`;
- expired Requirements/Applicability/Control Implementations are excluded after `effective_until`;
- the latest currently-effective Applicability record is selected deterministically;
- inactive/future implementation targets cannot satisfy current implementation state;
- active Evidence Contracts are considered only when their linked Requirement or Control Implementation is currently effective;
- future Evidence Assessments do not affect current state.

PR #35 additionally closes post-merge mode-consistency bugs found during review: effective-contract selection is now inside every derivation/source-inference path, memory persistence caps the stored assessment event itself to the governed freshness boundary, and reference journey evidence review uses the Program snapshot timestamp rather than process wall-clock time.

### Evidence freshness

Evidence Assessment validity cannot outlive the Evidence Contract freshness interval.

- the state engine always evaluates `min(valid_until, assessed_at + freshness)`;
- migration `000015_program_state_truth` derives missing validity and safely caps an over-long supplied validity at the governed maximum;
- memory persistence applies the same cap before storing the event used for current state and replay;
- non-positive validity windows remain invalid;
- bank-reference seed/repair data uses the same contract freshness rule rather than a hard-coded future date.

### Source Quality

The PostgreSQL projection reads the authoritative current source denominator rather than inferring whole-Program source health from the latest source trigger.

Current dependencies include:

- source IDs on currently-effective approved Requirements;
- sources on ACTIVE Evidence Contracts whose target Requirement/Control Implementation is currently effective.

A Program is source-current only when **every** currently-required active Evidence Source reports `CURRENT`. With no source dependency, Source Quality is explicitly `NOT_APPLICABLE`. A future Requirement/contract source does not pollute the current denominator.

### Unknown and overall state

All compliance dimensions participate in overall-state selection. `Assurance=UNKNOWN` or `SourceQuality=UNKNOWN` cannot silently produce `CURRENT`.

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

## 5. P1.2 Matter closure current-record truth

Matter closure evaluates current authoritative state rather than searching history for any favorable record.

### Decisions and exceptions

- Decision histories are maintained in authoritative append/event order; the last record for each normalized decision type is current even when lifecycle changes share a timestamp.
- A rejected, returned, expired or superseded current decision cannot be bypassed by an older approval in that lineage.
- Regulatory Change closure fails if any current decision type remains adverse; another favorable current decision cannot mask it.
- `NO_CHANGE_REQUIRED` remains the explicit path that does not require implementation/outcome evidence, but only when current decision state is otherwise resolved.
- Exception closure requires current, unexpired authority or an explicitly resolved structured condition set.
- Free-text condition descriptions are obligations, not evidence that the conditions were satisfied.

### Responses

Authority Request closure uses current Response Package lineages by purpose/audience. Every current lineage must be transmitted and acknowledged coherently; one favorable lineage cannot hide another draft/rejected/withdrawn response. A later replacement in the same lineage supersedes historical acknowledgement. Qualifying acknowledgement must be recorded at/after transmission.

### Verification

For each active Verification Contract, the latest result must PASS and remains valid only when:

- the reviewer identity is present;
- any contract-assigned reviewer authority matches;
- a linked action exists and is actually `IMPLEMENTED`;
- the reviewer is not the linked action owner;
- observation occurs at/after implementation/contract creation plus the configured observation period;
- the observation is not in the future.

These rules execute both when recording the verification result and again when evaluating closure. Invalid verification therefore cannot enter authoritative state and cannot become valid later merely because a Matter reaches `CLOSED`.

Reference/demo data and integration fixtures use the same chronology; they are not allowed to seed logically impossible PASS results.

## 6. P1.3 lifecycle-specific command responsibility

Material authorization is now a function of the **current record plus the requested lifecycle transition**, not only the route/command name.

### Decision lifecycle

- `PROPOSED` → proposer responsibility;
- `IN_REVIEW` or `RETURNED` → reviewer responsibility;
- `CHALLENGED` → independent challenger responsibility;
- `APPROVED`, `CONDITIONALLY_APPROVED`, `REJECTED`, `EXPIRED` or `SUPERSEDED` → authorizer responsibility.

The current Decision lifecycle is validated before authority execution. The verified command actor is stored in the stage-specific current record (`proposed_by`, `reviewed_by`, `challenged_by`, or `authority_principal_id`) and in the append-only event envelope.

### Response lifecycle

- preparation/rework and ordinary withdrawal resolve proposer responsibility where appropriate;
- review/rejection resolve reviewer responsibility;
- approval and withdrawal of an approved package resolve signatory responsibility;
- transmission resolves transmitter responsibility;
- acknowledgement recording resolves acknowledgement-recorder responsibility.

`prepared_by`, `reviewed_by`, `rejected_by`, `withdrawn_by`, `approved_by`, `transmitted_by`, and `acknowledged_by` preserve who actually performed each lifecycle step. Historical records are backfilled/reconstructed from `continuity_events.actor_id`, which is the trusted actor source rather than a client-provided identity field.

### Matter lifecycle

Transitions into `DECISION_REQUIRED`, `CLOSED` or `CANCELLED`, plus reopening a closed Matter, require current authorizer responsibility. Ordinary progression remains owned/performed according to the existing command matrix.

### Configuration and compatibility

Routing policies may now target `PROPOSER`, `REVIEWER`, `INDEPENDENT_CHALLENGER`, `AUTHORIZER`, `SIGNATORY`, `TRANSMITTER`, `ACKNOWLEDGEMENT_RECORDER`, performer/owner and escalation responsibilities. Existing direct-approved internal/reference data remains compatible; production HTTP decision commands use the lifecycle-aware path.

No parallel RBAC, workflow, receipt, event or lifecycle engine was introduced.

## 7. Current Today and automation truth

### Today

Non-demo Today currently projects active Workflow Tasks explicitly assigned to the verified principal. Completed/cancelled work is excluded. Team/unassigned work remains in its routing queue until resolved.

Demo/reference journeys may seed stakeholder presentation data, but are not production Today truth.

Today is not yet the complete event-driven intervention compiler envisioned by #27.

### Automation policy

`automation_policies` currently exposes governed eligibility/configuration boundaries. A visible policy does **not** prove that an automated action ran, succeeded or was independently verified.

Those claims require a real governed executor and persisted execution/verification evidence.

## 8. Enterprise work after semantic P1

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

## 9. Release and validation rules

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