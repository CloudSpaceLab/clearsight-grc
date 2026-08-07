# ClearSight implementation ledger

**Status date:** 2026-08-07  
**P0 executable integrity:** PRs #25, #30 — complete  
**P1 semantic/current-state correctness:** PRs #34–#39 — complete  
**UI/UX foundation:** PR #31 — complete  
**Simple capture/input closure:** PR #40 — complete  
**P2 schema ownership / dead compatibility:** PRs #41, #42 — complete  
**Today work-queue / Matter authority truth:** PR #43 — complete  
**Lifecycle work-requirement compiler:** PR #45 — in progress  
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

Closed foundation issues #26/#32/#33 and PRs #31/#40/#43 must not be recreated under new names or parallel frameworks.

## 2. Current work — #27.2a / PR #45

**Goal:** extend actor work beyond Matter Actions only where current canonical state determines one safe executable next step.

Current PR contract:

- [x] one shared Decision/Response lifecycle responsibility policy is used by command authorization and work compilation;
- [x] current Matter state compiles to non-authoritative `WorkRequirement` / `WorkAmbiguity` values;
- [x] deterministic Response transitions such as transmitted → acknowledgement and rejected → draft can become actor work;
- [x] active Verification Contracts become outcome-check work only after the observation period and only while no current result exists;
- [x] ambiguous Decision/Response branches remain **compiler ambiguity only** — no actor is guessed and no unusable Workflow Task is persisted;
- [x] current authority is resolved at projection time with Matter priority as materiality floor;
- [x] required/candidate actors must still be authority-eligible **and** able to read the Matter before a READY assignment exists;
- [x] delayed events resolve current authority rather than historical event-time authority;
- [x] Matter events project immediately through the existing outbox publisher;
- [x] a slower bounded maintainer reconciles restart/backfill and authority/delegation changes without another event/workflow stack;
- [x] reconciliation targets existing lifecycle projections, deterministic Response work and ready Verification Contracts rather than scanning every Matter;
- [x] migration `000020_workflow_projection_identity` gives deterministic `(tenant, kind, subject_type, subject_id)` Workflow identity;
- [x] no executable lifecycle work means no empty Workflow instance;
- [x] actor reads admit only `MATTER_ACTION` and `MATTER_LIFECYCLE`, with canonical Matter visibility enforced before limits and rechecked in Go;
- [x] Today can show real verification context through progressive disclosure without fabricating recommendation, prepared work or completion receipt;
- [x] external-representation work is labelled **External response**, not approval;
- [ ] final clean PR head must pass full CI plus the expanded **36-state** rendered evidence matrix before merge.

Detailed boundary: `docs/architecture/current-read-and-work-projection-boundary.md`.

## 3. Next — #27.2b

Two ownership gaps remain intentionally unresolved rather than guessed:

1. **Policy-selected lifecycle branches**
   - define a governed, versioned selection contract for Decision/Response states with several valid next transitions;
   - make selection authority-aware/auditable and invalidated by relevant state/policy changes;
   - compile the selected branch through the existing `WorkRequirement → authority → Workflow` path.

2. **Evidence Request recipient contract**
   - define canonical recipient/routing scope; `why_you`, `created_by` and invitation prose are not assignment truth;
   - include delegation, conflict, expiry/revocation and protected-record visibility;
   - only then project ordinary internal/external Evidence Requests into actor work.

Both require PostgreSQL replay/restart/reassignment/restricted-record evidence before production Today exposure.

## 4. Later #27 sequence

After #27.2b:

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
- WorkRequirement ≠ authoritative state; it is deterministic compiler output over canonical records/policy.
- WorkAmbiguity ≠ work assignment; ambiguity creates no actor Task until policy selects a valid transition.
- Schema/spec existence ≠ capability.

Do not add parallel authorization, task, event, worker, receipt, document or generic workflow stacks that duplicate these foundations.

## 6. Current executable truth

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

## 7. Release gates

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

Never claim a branch or PR is green from an older commit.