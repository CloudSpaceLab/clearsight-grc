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
**Current execution:** #27.2b-B2 Evidence Request recipient lifecycle + Today  
**Current execution issue:** #27  
**Umbrella pilot/GA catalogue:** #13

This is the authoritative execution ledger. Product/design/architecture documents define target behavior; this file controls **current implementation order and capability truth**. Completed-tranche detail belongs in focused documents and executable tests rather than being duplicated here indefinitely.

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
- Today admits supported `MATTER_ACTION` / `MATTER_LIFECYCLE` work without fabricating recommendation, prepared work, approval or verification receipts;
- exact-head CI and 36-state rendered evidence passed before merge.

### #27.2b-A / PR #46

- reuses existing maker-checker `RoutingPolicy`; no lifecycle-sequence table, event stack or public command surface;
- lifecycle sequencing is opt-in through `lifecycle_type`, `lifecycle_state` and optional `lifecycle_subtype` metadata;
- lifecycle sequence rules are **selector-free** and select only the next responsibility/gate;
- selector-free sequence rules materialize **zero** effective authority routes and cannot grant state-independent actor authority;
- actor authority remains in separate ordinary authority rules/assignments/grants/delegations and is resolved only after sequence selection;
- authority-only rules remain authority-only and still require supported actor selectors;
- maker-checker selector-cardinality checks receive an authority-only projection of policy definitions, so sequence rules do not weaken or poison actor conflict checks;
- malformed lifecycle declarations and lifecycle rules containing selectors fail policy validation;
- sequence policy never selects the substantive Decision/Response outcome;
- equal-ranked sequence rules selecting different next responsibilities fail closed;
- legal-entity UUID/code aliases normalize before sequence matching;
- the shared lifecycle policy derives only currently legal transitions executable by the selected responsibility;
- multi-outcome packets retain `allowed_targets` while leaving `target_status` empty;
- current authority resolves the actor/candidate set after sequence selection;
- canonical Matter visibility remains required before READY assignment;
- routing-policy changes converge through the existing bounded lifecycle reconciler without requiring a Matter event;
- PostgreSQL acceptance proves reviewer → authorizer packet convergence, zero authority routes from sequence rules, and separate current actor authority without pre-deciding an outcome;
- baseline CI and rendered UI evidence remain green.

Focused boundary: `docs/architecture/lifecycle-work-sequencing.md`.

### #27.2b-B1 / PR #49 — canonical Evidence Request recipient truth

- every new request has one canonical recipient: exact active internal `PERSON` principal or hashed external audience;
- `why_you`, `created_by`, invitation prose, subject readability and prior submitter identity are not assignment truth;
- legacy requests are deliberately not backfilled from descriptive fields and remain outside recipient actor queues;
- internal recipient eligibility is tenant/current-person/subject-visibility bound;
- recipient and subject visibility filter actor queues before `LIMIT`;
- exact internal request reads, submissions and authenticated artifact uploads require the canonical recipient;
- external request rows retain a fixed hash plus masked hint rather than the raw audience;
- invitation issuance must match canonical requester, audience and current subject visibility;
- invitation/session remains capability security state rather than assignment state;
- PostgreSQL adversarial coverage proves tenant recipient integrity, restricted-record behavior, pre-limit filtering, legacy exclusion and external audience binding;
- migration `000021_capture_recipient_truth` is reversible and covered by the dynamic latest-migration gate.

Focused boundary: `docs/architecture/evidence-recipient-boundary.md`.

Closed foundation issues #26/#32/#33 and completed PRs #31/#40/#43/#45/#46/#49 must not be recreated under new names or parallel frameworks.

## 2. Current work — #27.2b-B2 recipient lifecycle + Today

B1 establishes assignment truth. B2 must make recipient changes and actor-facing Evidence Request work operational without introducing a second assignment/workflow model.

Required:

- [ ] add internal wrong-recipient declaration with explicit, auditable semantics;
- [ ] add requester correction/reassignment while preserving one canonical current recipient;
- [ ] support redirect/delegation only where current executable authority/directory semantics can resolve it safely;
- [ ] define insufficient-authority/conflict behavior without silently selecting another actor;
- [ ] add explicit external invitation replacement/revocation and invalidate old invitations/sessions when recipient truth changes;
- [ ] converge recipient, principal-status and supported delegation changes without duplicate requests or stale actor work;
- [ ] project eligible internal Evidence Requests into existing Workflow/Today infrastructure as rebuildable actor work;
- [ ] preserve Capture Request/session as canonical request state; Workflow remains a projection only;
- [ ] prove replay/restart/reassignment/revocation/restricted-record and stale-capability behavior in PostgreSQL;
- [ ] render recipient/wrong-recipient/reassignment/expiry/revocation states on desktop and mobile before production Today exposure.

Do not add team/role/queue recipient claims until one executable current membership-resolution contract exists. Do not infer recipient changes from display labels, `why_you`, invitation copy or broad subject visibility.

## 3. Later #27 sequence

After #27.2b-B2:

1. operating Program/Work mutation UX, role-aware views and safe resume/delegate/recuse/escalate paths;
2. Capture/Import lifecycle completion: provenance, draft/resume/amendment, production scanning/quarantine/retry, governed multi-file requests, recurring mappings and canonical conversion;
3. Configure productization: organization/identity, responsibility/authority matrices, routing/escalation, simulation, maker-checker, effective dating/rollback and security/notification policy;
4. enterprise shell: production Explore/reconstruction, notifications, identity/session/step-up context;
5. human-product acceptance: representative timed bank-user usability, real browser/assistive-technology validation and final responsive/asset closure.

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
- Schema/spec existence ≠ capability.

Do not add parallel authorization, task, event, worker, receipt, document or generic workflow stacks that duplicate these foundations.

## 5. Current executable work truth

```text
canonical Matter state
→ deterministic work compiler
   ├─ executable requirement
   │   → current authority + record visibility
   │   → supported Workflow Task
   │   → Today
   └─ ambiguity
       → selector-free governed RoutingPolicy sequence rule selects next responsibility
       → shared lifecycle policy derives legal actions for that responsibility
       → separate current authority + record visibility resolves actor
       → one actor decision/review packet

canonical Evidence Request
→ canonical recipient + subject visibility
→ B2 recipient lifecycle / current-recipient convergence
→ rebuildable Workflow projection
→ Today
```

No sequence rule means no actor Task. Sequence-policy conflict means fail closed. A sequence rule grants no actor authority. A multi-outcome packet never writes the eventual Decision/Response state until the separately authorized actor executes an authoritative lifecycle command. Evidence Request presentation/capability state never substitutes for canonical recipient truth.

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
