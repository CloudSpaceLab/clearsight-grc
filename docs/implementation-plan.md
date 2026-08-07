# ClearSight implementation ledger

**Status date:** 2026-08-07  
**P0 executable integrity:** PRs #25 and #30  
**P1 semantic/current-state correctness:** PRs #34–#39  
**UI/UX foundation reconciliation:** PR #31  
**P2 schema ownership / dead compatibility:** PRs #41 and #42  
**Current product execution issue:** #27  
**Umbrella bank-first PRD:** #13

This is the authoritative execution ledger. Product, design, architecture and enterprise-productization documents define requirements and target behavior; this file controls current implementation order and capability truth.

Completed work is summarized here rather than duplicated indefinitely. Detailed behavior remains in the focused architecture/product documents and executable tests.

## 1. Completed foundation

### #26 P0 executable integrity — COMPLETE

- typed executable route/access registry with verified tenant/actor binding;
- persisted capture consolidation and bounded invitation/session security;
- source-health reconciliation through transactional outbox/inbox delivery;
- bounded worker classes, retry/dead-letter behavior and truthful compound-command outcomes;
- executable route/runtime-contract parity;
- effective authority convergence across routes, assignments, grants, delegations and segregation rules.

Issue #26 is closed.

### #32 P1 semantic/current-state correctness — COMPLETE

- effective-time Program requirements, applicability, controls, evidence and source-quality state;
- stale/unknown Program projection truth and projection-version parity;
- current-record Matter closure semantics for Decisions, Responses and verification;
- lifecycle-specific proposer/reviewer/challenger/authorizer/signatory/transmitter/acknowledgement responsibilities;
- trusted lifecycle actor reconstruction;
- bounded normalized current reads with replay retained for history/audit/reconciliation;
- Matter Action as accountable business work and Workflow Task as its actor-facing projection;
- durable/resource-bounded document imports, asynchronous processing, hostile-file limits and truthful truncation/completeness semantics.

Issue #32 is closed. See:

- `docs/product/authority-routing-and-escalation.md`
- `docs/architecture/current-read-and-work-projection-boundary.md`
- `docs/architecture/document-import-resource-and-durability-boundary.md`

### UI/UX foundation reconciliation — COMPLETE IN PR #31

- intervention-first Today hierarchy;
- status/reason-first Programs and current-handoff-first Matters;
- exact target launch/deep-link behavior;
- Capture assertion review and governed import review hierarchy;
- typed degraded/error/conflict states;
- semantic light/dark theme and compact/comfortable density foundations;
- actor/authority context without fixed institutional role fiction;
- rendered-state, axe accessibility and deterministic UI-evidence gates.

This is the UI foundation, not full product completion.

### #33 P2 schema ownership and dead compatibility — COMPLETE

- every live durable PostgreSQL table has one machine-checked owner/maturity classification;
- migrations require matching down migrations;
- unused `audit_events` and `readiness_snapshots` were removed fail-closed through migration `000019_schema_ownership_cleanup`;
- historical capture duplicates remain retired;
- direct Workflow Task mutation APIs/service methods were removed;
- the Matter Action projector is the supported Task write path;
- the broad stale `api/openapi.yaml` duplicate was removed;
- `internal/httpapi/route_registry.go` → `api/runtime.openapi.json` is the sole executable route/access contract.

Issue #33 is closed. See:

- `docs/architecture/durable-schema-ownership.md`
- `api/README.md`

## 2. Current sequence — issue #27

Issue #27 now owns product/UI execution after the P0–P2 foundation. Issue #13 remains the umbrella product PRD and does not override this execution order.

### #27.1 Today work-queue and Matter authority truth — IN PR #43

This tranche fixes correctness/security before richer decision packets:

- [x] existing-Matter lifecycle authority derives the canonical Matter ID from the route rather than trusting redundant JSON identifiers;
- [x] conflicting body/route Matter or subresource identifiers fail closed;
- [x] the current Matter priority is the minimum authority materiality for existing-Matter commands;
- [x] a restricted Matter cannot assign a new Matter Action owner who is not allowed to read the Matter;
- [x] actor-facing Workflow Task reads are bound to the verified principal;
- [x] only the supported `MATTER_ACTION` projection participates in current actor work;
- [x] terminal, unsupported and inaccessible work is filtered before limits;
- [x] active work is ordered deadline-first before bounded reads;
- [x] canonical Action → Matter ID, priority and access metadata are joined internally and never exposed as a second public Task contract;
- [x] Today rechecks Matter visibility and uses canonical Matter priority for authority inspection;
- [x] recommendation/prepared-work fields remain empty unless a separately governed record supplies them;
- [x] race-enabled, PostgreSQL and rendered/UI-evidence regression coverage protects the boundary.

PR #43 must merge only from an exact CI/UI-evidence-green head.

### #27.2 explicit work-requirement compiler and governed intervention packets — NEXT

The next tranche should expand Today beyond accountable Matter Actions without guessing lifecycle ownership.

Required design:

- derive an actor work requirement only when the next responsibility/action is unambiguous or explicitly selected by policy;
- do **not** infer one assignee merely because a Decision/Response is in a given state when multiple valid transitions exist;
- resolve the current eligible principal/candidate set through the existing authority engine;
- reuse continuity events, transactional outbox and the existing Workflow Task projection instead of creating another workflow/event stack;
- keep canonical Decision/Response/Evidence state authoritative; Task remains a projection;
- show recommendation, prepared work, side effects, verification and execution receipt only from governed records that actually exist;
- support accept/edit-and-accept/reject/request-evidence/compare/escalate only where corresponding authoritative commands exist;
- preserve protected-record visibility and fail closed before queue limits;
- add restart/idempotency/current-authority/adversarial tests before exposing the work in production Today.

A Decision such as `IN_REVIEW` may have multiple legal next transitions. That ambiguity must be represented or resolved by policy; it must not be hidden by choosing an arbitrary reviewer/challenger/authorizer.

### Later #27 work, in order

After #27.2:

1. **Operating Program/Work mutation flows** — direct governed actions, saved role-aware views, delegation/recusal/conflict/escalation where domain commands exist, save/resume for complex work.
2. **Capture/Import completion** — provenance classes, redirect/wrong-recipient flows, draft/resume/amendment, recurring mappings and governed conversion to canonical records.
3. **Configure productization** — organization/identity sources, responsibility/authority matrices, routing/escalation builders, impact preview, maker-checker, effective dating/rollback, notification and security policy surfaces.
4. **Enterprise shell** — production Explore/reconstruction, actor notifications, identity/session/step-up context.
5. **Acceptance closure** — representative bank-user timed usability, complete state coverage, production assets and final accessibility/responsive evidence.

Do not recreate work already completed in PR #31 or P0–P2 under new names.

## 3. Canonical domain invariants

- **Program** = ongoing obligation/compliance continuity.
- **Matter** = bounded change, exception, finding, decision, action, response or verification case.
- **Matter Action ≠ Workflow Task.** Action is accountable business work; Task is actor-facing projected/routed work.
- **Signal ≠ conclusion.** A Signal is an observation that deterministic assessment may convert into drift or attention.
- **Submission ≠ sufficient evidence.** Evidence Contract assessment determines sufficiency.
- **Implementation ≠ verified outcome.** Completion alone cannot close material work requiring verification.
- **Recommendation ≠ approval.** Current authority remains explicit.
- **Automation Policy ≠ execution receipt.** Permission is not evidence that an action ran.
- **Intervention Summary ≠ authoritative state.** It is a read projection over canonical records.
- **Schema/spec existence ≠ capability.** Executable ownership and tested readers/writers determine capability truth.

Do not add parallel authorization, task, event, worker, receipt, document or generic workflow stacks that duplicate these foundations.

## 4. Current executable truth

### Route/access

`internal/httpapi/route_registry.go` is the canonical executable route inventory. `api/runtime.openapi.json` is its mechanically verified route/access/permission projection.

Only health routes are public. Bounded capture uses capability access. Other protected routes require verified identity. Material commands resolve current authority at execution.

### Current versus historical continuity reads

```text
material command
→ normalized current row(s)
→ append-only domain event
→ transactional outbox / required maintenance work
→ commit

ordinary current read
→ normalized current tables

history / point-in-time audit
→ append-only history / historical projection
```

### Program-state freshness

- `program.version` = current command aggregate version;
- `current_state.program_version` = Program version assessed by the state projection;
- `current_state.projection_version` = monotonic calculated projection revision.

A projection is stale when its assessed Program version is behind the current Program version.

### Work truth

```text
accountable canonical work
→ domain event / transactional outbox
→ policy/authority-backed actor work requirement
→ idempotent Workflow Task projection
→ Today / workflow read surfaces
```

Today currently has executable production coverage for the Matter Action branch of this model. #27.2 owns broader lifecycle work requirements.

### Document-import truth

```text
stored original
→ PENDING import + transactional outbox request
→ bounded worker extraction/analysis
→ terminal import detail
→ explicit human proposal review
```

A stored artifact is not an extracted artifact. Extracted text is not exhaustive when truncation/omission metadata says otherwise. A proposal is not an approved compliance conclusion.

### Durable-schema truth

A live table is not evidence of product capability unless it has an explicit owner and executable reader/writer semantics. `docs/architecture/durable-schema-ownership.md` is checked against the ordered migration result.

## 5. Enterprise work beyond current #27 tranches

Detailed enterprise requirements remain in `docs/engineering/enterprise-productization-implementation-plan.md` and product/design specifications.

Later gates include enterprise identity synchronization, controlled configuration/rollback, notifications, step-up assurance, production object storage/malware scanning/retention, PDF/OCR provider isolation, representative capacity evidence, backup/restore/provider-outage exercises and pilot-bank legal/configuration approval.

## 6. Release and validation rules

Checkboxes describe repository capability, not deployment readiness. A tranche is not complete until relevant gates pass on its **exact head**:

- `gofmt` and `go vet`;
- race-enabled Go tests;
- PostgreSQL composition, migrations and serialized integration tests;
- TypeScript strict checking;
- Vitest/axe rendered-state tests;
- production Vite build;
- deterministic UI evidence when user-facing behavior changes;
- adversarial identity, tenant, authority, replay and degraded-path tests;
- representative query-count/performance/recovery evidence when cardinality or durability changes.

Never claim a branch or PR is green based on an older commit.