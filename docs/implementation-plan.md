# ClearSight implementation ledger

**Status date:** 2026-08-08  
**P0 executable integrity:** PRs #25, #30 — complete  
**P1 semantic/current-state correctness:** PRs #34–#39 — complete  
**UI/UX foundation:** PR #31 — complete  
**Simple capture/input closure:** PR #40 — complete  
**P2 schema ownership / dead compatibility:** PRs #41, #42 — complete  
**Today work-queue / Matter authority truth:** PR #43 — complete  
**Deterministic lifecycle work compiler:** PR #45 — complete  
**Governed lifecycle sequencing:** PR #46 — in progress  
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

Closed foundation issues #26/#32/#33 and completed PRs #31/#40/#43/#45 must not be recreated under new names or parallel frameworks.

## 2. Current work — #27.2b-A / PR #46

**Goal:** resolve genuinely multi-branch Decision/Response handoffs without pre-deciding the human outcome.

The key rule is:

> Routing policy may select the next **responsibility/gate**. It may not select the substantive lifecycle outcome on behalf of that actor.

Current PR contract:

- [x] reuse existing maker-checker `RoutingPolicy`; no new policy/workflow table or public endpoint;
- [x] lifecycle sequencing is opt-in through `lifecycle_type`, `lifecycle_state` and optional `lifecycle_subtype` rule metadata;
- [x] authority-only rules remain authority-only;
- [x] malformed lifecycle declarations fail ordinary RoutingPolicy validation before activation;
- [x] sequence resolution selects responsibility with rule/policy provenance, never a principal or outcome;
- [x] equal-ranked rules selecting different next responsibilities fail closed;
- [x] legal-entity UUID/code aliases are normalized before sequence matching;
- [x] the compiler keeps only currently legal transitions executable by the selected responsibility;
- [x] multi-outcome packets preserve `allowed_targets` while leaving `target_status` empty;
- [x] the existing authority engine still resolves the current actor/candidate set after responsibility selection;
- [x] lifecycle metadata cannot introduce state-specific actor authority that bypasses existing selector-conflict checks;
- [x] existing Matter visibility remains required before READY assignment;
- [x] routing-policy changes converge through the existing bounded lifecycle reconciler without requiring a Matter event;
- [x] PostgreSQL acceptance covers reviewer → authorizer policy convergence without a pre-decided outcome;
- [ ] exact final PR head must pass full repository CI and baseline UI evidence before merge.

Focused boundary: `docs/architecture/lifecycle-work-sequencing.md`.

## 3. Next — #27.2b-B Evidence Request recipient truth

Ordinary Evidence Requests must not enter actor Today work from descriptive copy or invitation mechanics alone.

Required:

- [ ] define canonical intended-recipient/routing scope for internal and invited external requests;
- [ ] distinguish person, position/role/group and external capability/address targeting where needed;
- [ ] define redirect/delegate/wrong-recipient/insufficient-authority/conflict behavior;
- [ ] define invitation expiry/revocation/replacement semantics without leaking unrelated Matter context;
- [ ] bind protected-record visibility before recipient projection and before queue limits;
- [ ] converge identity/directory/delegation changes without duplicate requests or stale actor work;
- [ ] preserve Capture Request/session as canonical request state and Workflow as rebuildable actor projection;
- [ ] prove replay/restart/reassignment/revocation/restricted-record behavior in PostgreSQL and rendered mobile acceptance.

`why_you`, `created_by` and invitation prose remain explicitly **not assignment truth**.

## 4. Later #27 sequence

After #27.2b-B:

1. operating Program/Work mutation UX, role-aware views and safe resume/delegate/recuse/escalate paths;
2. Capture/Import lifecycle completion: provenance, wrong-recipient, draft/resume/amendment, invitation lifecycle, production scanning/quarantine/retry, governed multi-file requests, recurring mappings and canonical conversion;
3. Configure productization: organization/identity, responsibility/authority matrices, routing/escalation, simulation, maker-checker, effective dating/rollback and security/notification policy;
4. enterprise shell: production Explore/reconstruction, notifications, identity/session/step-up context;
5. human-product acceptance: representative timed bank-user usability, real browser/assistive-technology validation and final responsive/asset closure.

## 5. Canonical invariants

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
- Lifecycle sequence policy selects responsibility, **not outcome**.
- Authority resolution selects current eligible actor after sequence selection.
- Schema/spec existence ≠ capability.

Do not add parallel authorization, task, event, worker, receipt, document or generic workflow stacks that duplicate these foundations.

## 6. Current executable work truth

```text
canonical Matter state
→ deterministic work compiler
   ├─ executable requirement
   │   → current authority + record visibility
   │   → supported Workflow Task
   │   → Today
   └─ ambiguity
       → optional governed RoutingPolicy sequence rule selects next responsibility
       → shared lifecycle policy derives legal actions for that responsibility
       → current authority + record visibility
       → one actor decision/review packet
```

No sequence rule means no actor Task. Policy conflict means fail closed. A multi-outcome packet never writes the eventual Decision/Response state until the responsible actor executes an authoritative lifecycle command.

## 7. Release gates

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
