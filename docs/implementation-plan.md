# ClearSight implementation ledger

**Status date:** 2026-08-08  
**P0 executable integrity:** PRs #25, #30 — complete  
**P1 semantic/current-state correctness:** PRs #34–#39 — complete  
**UI/UX foundation:** PR #31 — complete  
**Simple capture/input closure:** PR #40 — complete  
**P2 schema ownership / dead compatibility:** PRs #41, #42 — complete  
**Today work-queue / Matter authority truth:** PR #43 — complete  
**Lifecycle work-requirement compiler:** PR #45 — complete  
**Current execution issue:** #27  
**Umbrella pilot/GA catalogue:** #13

This is the authoritative execution ledger. Product/design/architecture documents define target behavior; this file controls **current implementation order and capability truth**. Completed-tranche detail belongs in focused documents and tests rather than being duplicated here indefinitely.

## 1. Completed foundation — do not rebuild

### P0 / #26

- verified route, identity, tenant and write boundary;
- truthful compound-command/post-commit semantics;
- transactional outbox/inbox and bounded worker recovery;
- effective authority convergence across assignments, grants, routing, delegation and segregation rules.

### P1 / #32

- effective/current Program state and freshness truth;
- current-record Matter Decision/Response/verification closure semantics;
- lifecycle-specific command responsibility;
- bounded normalized current reads with replay reserved for history/audit/reconciliation;
- durable/resource-bounded document import processing.

### UI foundation / #31 and simple capture / #40

- intervention-first **Today**;
- status/reason-first Programs and handoff-first Matters;
- exact target/deep-link behavior and typed degraded/conflict states;
- semantic light/dark themes, compact/comfortable density, axe/state tests and deterministic Playwright evidence;
- low-effort typed Capture controls and contextual file/photo dropzones;
- Draw/Type signatures stored as bounded request artifacts;
- external simple verification with known facts read-only and exact review/receipt;
- request-bound photo/file/signature validation, replacement failure safety and stale async-operation guards.

### P2 / #33

- machine-checked ownership for every live durable table;
- reversible migration discipline and dead compatibility removal;
- direct Workflow Task mutation APIs/service methods removed;
- Matter Action remains accountable work; Workflow Task remains derived actor work;
- `internal/httpapi/route_registry.go` → `api/runtime.openapi.json` is the sole executable route/access contract.

### #27.1 / PR #43

- route-bound Matter authority and Matter-priority materiality floor;
- restricted Matter Action owner visibility check;
- actor-scoped Workflow Task reads;
- pre-limit terminal/unsupported/inaccessible filtering and deadline ordering;
- canonical Action → Matter access/materiality joins for Today authority inspection.

### #27.2a / PR #45

- one shared Decision/Response lifecycle responsibility policy is used by command authorization and work compilation;
- current Matter state compiles to non-authoritative `WorkRequirement` / `WorkAmbiguity` values;
- deterministic Response transitions such as transmitted → acknowledgement and rejected → draft can become actor work;
- active Verification Contracts become outcome-check work only after the observation period and only while no current result exists;
- ambiguous Decision/Response branches remain **compiler ambiguity only** — no actor is guessed and no unusable Workflow Task is persisted;
- current authority is resolved at projection time with Matter priority as materiality floor;
- required/candidate actors must still be authority-eligible **and** able to read the Matter before a READY assignment exists;
- delayed events resolve current authority rather than historical event-time authority;
- Matter events project immediately through the existing outbox publisher;
- a slower bounded maintainer reconciles restart/backfill and authority/delegation changes without another event/workflow stack;
- reconciliation targets existing lifecycle projections, deterministic Response work and ready Verification Contracts rather than scanning every Matter;
- migration `000020_workflow_projection_identity` gives deterministic `(tenant, kind, subject_type, subject_id)` Workflow identity;
- no executable lifecycle work means no empty Workflow instance;
- actor reads admit only `MATTER_ACTION` and `MATTER_LIFECYCLE`, with canonical Matter visibility enforced before limits and rechecked in Go;
- Today can show real verification context through progressive disclosure without fabricating recommendation, prepared work or completion receipt;
- external-representation work is labelled **External response**, not approval;
- exact final head `e9af7743280a41e6d7ed41210fed334ba9793349` passed full CI run `31227792004` and 36-state UI evidence run `31227791999` before merge.

Closed foundation issues #26/#32/#33 and PRs #31/#40/#43/#45 must not be recreated under new names or parallel frameworks.

Detailed work-projection boundary: `docs/architecture/current-read-and-work-projection-boundary.md`.

## 2. Current work — #27.2b

Two ownership gaps remain intentionally unresolved rather than guessed.

### A. Policy-selected lifecycle branches

Decision and some Response states legitimately have several valid next transitions. State alone is not enough to choose one next actor/action.

Required:

- define a canonical, versioned selection contract for an allowed lifecycle branch;
- bind a selection to exact Matter/subresource/current version and transition;
- record the selecting actor/policy and rationale;
- require current authority for material selection/change;
- invalidate or require re-evaluation when canonical lifecycle state, relevant authority/policy or scope materially changes;
- keep unresolved branching visible as `WorkAmbiguity`, not a manufactured Task;
- compile an accepted current selection through the existing `WorkRequirement → current authority → Workflow Task` path;
- prove stale selection, replay, restart, reassignment, conflict, restricted-record and ambiguous-policy behavior in PostgreSQL.

Do not add a second lifecycle/state-machine/workflow framework merely to store branch choices.

### B. Evidence Request recipient/routing truth

The current Evidence Request model has secure invitations/capture, but descriptive values such as `why_you`, `created_by` and invitation prose are not assignment truth.

Required before ordinary Evidence Requests enter production Today:

- define canonical intended recipient/routing scope for internal and invited external requests;
- distinguish person, position/role/group and external-address/capability targeting where applicable;
- define delegation, redirect, wrong-recipient, insufficient-authority and conflict behavior;
- define invitation expiry/revocation/replacement without leaking unrelated Matter context;
- bind protected-record visibility before recipient projection and before queue limits;
- converge identity/directory/delegation changes without duplicate requests or stale actor work;
- keep Capture request/session as canonical request state and Workflow as rebuildable actor projection;
- prove replay/restart/reassignment/revocation/restricted-record behavior in PostgreSQL and rendered mobile acceptance.

## 3. Later #27 sequence

After #27.2b:

1. operating Program/Work mutation UX, role-aware views and safe resume/delegate/recuse/escalate paths;
2. Capture/Import lifecycle completion: provenance, wrong-recipient, draft/resume/amendment, invitation lifecycle, production scanning/quarantine/retry, governed multi-file requests, recurring mappings and canonical conversion;
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
- WorkRequirement ≠ authoritative state; it is deterministic compiler output over canonical records/policy.
- WorkAmbiguity ≠ work assignment; ambiguity creates no actor Task until policy selects a valid transition.
- Schema/spec existence ≠ capability.

Do not add parallel authorization, task, event, worker, receipt, document or generic workflow stacks that duplicate these foundations.

## 5. Current executable truth

### Route/access

`internal/httpapi/route_registry.go` is canonical executable route/access inventory. `api/runtime.openapi.json` is its mechanically verified projection. Material commands resolve current authority at execution.

### Current vs historical reads

```text
material command
→ normalized current rows
→ append-only domain event
→ transactional outbox / required maintenance work
→ commit

ordinary current read
→ normalized current tables

history / point-in-time audit
→ append-only history / historical projection
```

### Work

```text
canonical Matter state
→ deterministic work-requirement compiler
   ├─ executable requirement → current authority + record visibility → supported Workflow Task → Today
   └─ ambiguity → no Task; wait for governed policy selection
```

`MATTER_ACTION` and `MATTER_LIFECYCLE` are projection kinds, not separate business workflow engines. Obsolete lifecycle Tasks are cancelled by reconciliation.

### Capture

A submitted request-bound `STORED_UNSCANNED` artifact proves what the respondent submitted; it does not prove evidence sufficiency or completed production malware inspection.

### Document import

```text
stored original
→ PENDING import + transactional outbox
→ bounded extraction/analysis
→ terminal import detail
→ explicit human proposal review
```

A stored artifact is not extracted content; extracted content is not exhaustive when truncation says otherwise; a proposal is not an approved conclusion.

## 6. Release gates

A tranche is not complete until relevant gates pass on its **exact final head**:

- `gofmt` and `go vet`;
- race-enabled Go tests;
- PostgreSQL composition, migrations, latest rollback/reapply and serialized integration tests;
- TypeScript strict checking;
- Vitest/axe/state tests;
- production Vite build;
- deterministic rendered UI evidence for user-facing changes;
- adversarial identity/tenant/authority/replay/degraded-path tests;
- representative query-count/performance/recovery evidence when cardinality or durability changes.

The CI latest-migration rollback gate must discover the latest ordered `*.up.sql` migration dynamically; it must never be left hard-coded to an older migration number.

Never claim a branch or PR is green from an older commit.
