# ClearSight implementation ledger

**Status date:** 2026-08-09  
**P0 executable integrity:** PRs #25, #30 — complete  
**P1 semantic/current-state correctness:** PRs #34–#39 — complete  
**UI/UX foundation:** PR #31 — complete  
**Simple capture/input closure:** PR #40 — complete  
**P2 schema ownership / dead compatibility:** PRs #41, #42 — complete  
**Today work-queue / Matter authority truth:** PR #43 — complete  
**Deterministic lifecycle work compiler:** PR #45 — complete  
**Governed lifecycle sequencing:** PR #46 — complete  
**Evidence recipient canonical truth (B1):** PR #49 — complete  
**Evidence recipient lifecycle + Today (B2):** PR #50 — complete  
**Operating Program/Work governed mutations:** PR #51 — complete  
**Current execution:** remaining operating-work usability  
**Current execution issue:** #27  
**Umbrella pilot/GA catalogue:** #13

This is the authoritative execution ledger. Product/design/architecture documents define target behavior; this file controls **current implementation order and capability truth**. Completed-tranche detail belongs in focused documents and executable tests rather than being duplicated under new frameworks.

## 1. Completed foundation — do not rebuild

### P0 / #26

Verified route/identity/tenant/write boundaries, truthful post-commit semantics, transactional outbox/inbox recovery and effective authority convergence.

### P1 / #32

Effective/current Program state, current-record Matter Decision/Response/verification semantics, lifecycle-specific command responsibility, bounded current reads and durable/resource-bounded document imports.

### UI foundation / #31 and simple capture / #40

Intervention-first Today, exact target navigation, status/reason-first workspaces, typed degraded/conflict states, semantic theme/density, axe/Playwright evidence, low-effort typed Capture, contextual photo/file controls, Draw/Type signatures and request-bound artifact validation.

### P2 / #33

Machine-checked durable-schema ownership, reversible migrations, dead compatibility removal, no direct Workflow Task mutation surface, and one executable route/access contract: `internal/httpapi/route_registry.go` → `api/runtime.openapi.json`.

### #27.1 / PR #43

Route-bound Matter authority, Matter-priority materiality floor, restricted-record assignment protection, actor-scoped Workflow reads, pre-limit visibility/terminal filtering and deadline-first work ordering.

### #27.2a / PR #45

- deterministic current Matter state compiles into non-authoritative `WorkRequirement` / `WorkAmbiguity`;
- safe single-path Response work and ready Verification work project through existing Workflow infrastructure;
- ambiguous Decision/Response states create **no guessed actor and no actor Task**;
- assignment uses current authority plus canonical Matter visibility;
- delayed events cannot resurrect historical assignments;
- bounded reconciliation provides restart/backfill/authority convergence;
- deterministic Workflow projection identity prevents duplicate instances;
- Today admits supported Matter work without fabricated recommendation, prepared-work, approval or verification receipts.

### #27.2b-A / PR #46

- existing maker-checker `RoutingPolicy` is reused; no second lifecycle policy/workflow stack;
- lifecycle sequence rules select only the next responsibility/gate and materialize zero actor authority routes;
- actor authority remains separately resolved from current rules/assignments/grants/delegations;
- policy never pre-selects the substantive Decision/Response outcome;
- multi-outcome packets retain `allowed_targets` with no guessed `target_status`;
- malformed/conflicting selector-free sequence rules fail closed;
- routing-policy changes converge active packets through the existing reconciler.

Focused boundary: `docs/architecture/lifecycle-work-sequencing.md`.

### #27.2b-B1 / PR #49 — canonical Evidence Request recipient truth

- every new request has one canonical recipient: exact active internal `PERSON` principal or hashed external audience;
- descriptive fields, creator identity, invitation copy and submitter identity are not assignment truth;
- legacy requests are deliberately not backfilled from descriptive fields;
- internal recipient eligibility is tenant/current-person/subject-visibility bound;
- recipient and subject visibility filter actor queues before `LIMIT`;
- exact internal reads/submissions/uploads require the canonical recipient;
- external rows retain a fixed audience hash plus masked hint;
- invitation issuance is bound to canonical requester, audience and current subject visibility;
- invitation/session remains capability security state rather than assignment state.

Focused boundary: `docs/architecture/evidence-recipient-boundary.md`.

### #27.2b-B2 / PR #50 — recipient lifecycle + Today

- assigned internal recipient can explicitly declare wrong-recipient with required reason and optimistic versioning;
- wrong-recipient immediately removes that actor's request/submission rights;
- only the trusted original requester can correct/reassign the canonical recipient;
- reassignment revalidates current recipient eligibility and supported subject visibility;
- external recipient replacement revokes active invitation/session capability in the same PostgreSQL transaction;
- recipient history records ordered `WRONG_RECIPIENT` / `REASSIGNED` events without becoming a generic assignment ledger;
- eligible internal Evidence Requests project through existing `workflow_instances` / `workflow_tasks` as `EVIDENCE_REQUEST` actor work;
- recipient/principal/visibility changes converge the same Workflow/Task identity instead of duplicating work;
- Today admits Evidence Request work through the same pre-limit actor-visibility boundary;
- terminal/ineligible requests that never had actor work do not manufacture empty Workflow history;
- requester/respondent UI exposes progressive wrong-recipient/reassignment states without inventing team/role/queue targeting;
- exact final head `34439ff986636d523e294c2e2987aebbeedec718` passed CI #631 and UI evidence #245 before squash merge.

B2 does **not** imply generic redirect/delegation/team/role/queue recipient semantics. Those remain deferred until executable current membership/authority contracts exist.

### Operating Program/Work governed mutations / PR #51

- thin typed clients reuse existing governed Program/Matter commands and never send client-supplied actor identity;
- Work consumes the existing actor-scoped Workflow queue; there is no second frontend task model;
- Decision/Response/Verification controls use current `command_name`, `allowed_targets`, subresource and rationale packet fields;
- Matter Action projection carries executable `matter.action.transition` context with targets derived from the canonical Go Action lifecycle;
- Action owners can progress READY/IN_PROGRESS/BLOCKED work without React inventing legal target states;
- Program status controls remain hidden unless current `AUTHORIZER` resolution includes the actor;
- Program affordances mirror only the existing DRAFT→ACTIVE/RETIRED, ACTIVE→PAUSED/RETIRED and PAUSED→ACTIVE/RETIRED lifecycle; server authority/version/lifecycle/activation checks remain final;
- successful mutations replace displayed Program/Matter detail with the authoritative server aggregate;
- stale-version/forbidden/not-found/degraded failures remain visible and fail closed;
- deterministic operating UI evidence covers 1440×900 and 390×844 rendering, exact Action/Program target lists and horizontal-overflow safety;
- B2's PostgreSQL regression fixture was made database-clock-relative after its fixed same-day deadline expired; production expiry semantics were unchanged;
- exact final head `59f4d37e3cc11805242b46e7637103b8ad59a36b` passed CI #664 and UI evidence #277;
- squash-merged as `3da504cef68651ab34c9ccb1f3c70e7053c1dfc2`.

Closed foundation issues #26/#32/#33 and completed PRs #31/#40/#43/#45/#46/#49/#50/#51 must not be recreated under new names or parallel frameworks.

## 2. Current work — remaining operating-work usability

The core operating commands are now usable. The next work must reduce daily operator effort **without inventing domain commands, task models, or authorization truth**.

Execute in this order unless code-level evidence changes the dependency:

1. [ ] change-since-last-accepted-review and material-exception summaries derived from canonical state/history;
2. [ ] role-aware saved Work views over the existing actor-visible queries rather than a second task model;
3. [ ] protected-record focused mode that avoids unrelated existence leakage;
4. [ ] save/resume for genuinely complex Decisions/Responses only where a durable draft contract exists;
5. [ ] redirect/delegate/recuse/conflict/escalate only where executable authority/membership/domain commands exist.

Owner/executive views must remain projections of the same canonical state rather than separate dashboard state.

Explicitly out of this immediate continuation:

- broad Program authoring/configuration;
- generic notification-centre or Explore expansion;
- invented delegation/team/group behavior without executable membership/authority contracts;
- presentation-only audit or verification receipts.

## 3. Later #27 sequence

After the operating-work usability continuation:

1. Capture/Import lifecycle completion: provenance, draft/resume/amendment, production scanning/quarantine/retry, governed multi-file requests, recurring mappings and canonical conversion;
2. Configure productization: organization/identity, responsibility/authority matrices, routing/escalation, simulation, maker-checker, effective dating/rollback and security/notification policy;
3. enterprise shell: production Explore/reconstruction, notifications, identity/session/step-up context;
4. human-product acceptance: representative timed bank-user usability, real browser/assistive-technology validation and final responsive/asset closure.

## 4. Canonical invariants

- Program = ongoing obligation/compliance continuity.
- Matter = bounded change, exception, finding, decision, action, response or verification case.
- Matter Action ≠ Workflow Task.
- Signal ≠ conclusion.
- Submission ≠ sufficient evidence.
- Implementation ≠ verified outcome.
- Recommendation ≠ approval.
- Automation Policy ≠ execution receipt.
- Intervention Summary ≠ authoritative state.
- WorkRequirement ≠ authoritative state.
- WorkAmbiguity ≠ actor assignment.
- Lifecycle sequence policy selects responsibility, **not outcome or actor**.
- Lifecycle sequence rule ≠ authority route.
- Authority resolution selects the current eligible actor only after sequence selection.
- Evidence Request description/invitation ≠ recipient assignment.
- Evidence Request recipient is canonical request state; Workflow work is a rebuildable actor projection.
- Workflow command packet is an executable projection, not authoritative domain state; every command is revalidated by the domain service.
- Program UI transition choices are affordances only; Program command service remains authoritative.
- Saved Work view ≠ new task ownership or authorization state.
- Schema/spec existence ≠ capability.

Do not add parallel authorization, task, event, worker, receipt, document or generic workflow stacks that duplicate these foundations.

## 5. Current executable work truth

```text
canonical Matter state
→ deterministic work compiler / canonical Action event
→ current authority or accountable owner + record visibility
→ existing Workflow Task with executable command packet
→ Today / Work
→ governed domain command with verified actor + expected version
→ authoritative Matter aggregate
→ projection converges

canonical Program
→ authority resolution for AUTHORIZER
→ lifecycle-valid UI affordance
→ governed program.transition command
→ server lifecycle + authority + version + activation-prerequisite checks
→ authoritative Program aggregate

canonical Evidence Request
→ canonical recipient + subject visibility
→ recipient lifecycle / current-recipient convergence
→ rebuildable Workflow projection
→ Today / Capture
```

No actor Task is created from ambiguity without a governed responsibility selection and separate actor authority. A multi-outcome packet never writes the eventual Decision/Response state until the separately authorized actor executes the authoritative lifecycle command. Presentation/projection/capability state never substitutes for canonical domain truth.

## 6. Release gates

A tranche is not complete until relevant gates pass on its **exact final head**:

- `gofmt` and `go vet`;
- race-enabled Go tests;
- PostgreSQL composition, migrations, latest rollback/reapply and serialized integration tests;
- TypeScript strict checking;
- Vitest/axe/state tests;
- production Vite build;
- deterministic rendered UI evidence for affected user-facing/read-contract changes;
- adversarial identity/tenant/authority/replay/degraded-path tests;
- representative query-count/performance/recovery evidence when cardinality or durability changes.

Never claim a branch or PR is green from an older commit.
