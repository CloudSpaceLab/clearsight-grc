# ClearSight implementation ledger

**Status date:** 2026-08-07  
**P0 executable integrity:** PRs #25 and #30  
**P1 semantic/current-state correctness:** PRs #34–#39  
**UI/UX foundation and simple capture inputs:** PRs #31 and #40  
**P2 schema ownership / dead compatibility:** PRs #41 and #42  
**Today work-queue / Matter authority truth:** PR #43  
**Lifecycle work-requirement compiler:** PR #45 (in progress)  
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

### Simple capture/input and dropzone closure — COMPLETE IN PR #40

- Today remains the practical work surface rather than generic `Your work` copy;
- typed Capture controls use appropriate short text, long text, choice, date, number, photo/file and signature interactions;
- photo evidence uses a prominent camera/drop/tap surface with preview, document import uses an explicit pre-commit dropzone, and incidental file evidence stays compact;
- external simple field verification keeps known facts read-only and performs exact review before receipt;
- photo/file/signature answers are request-bound artifact references rather than arbitrary strings or base64 answer payloads;
- artifact field validation fails closed for wrong request/media, empty/unknown/quarantined/deleted artifacts and oversized signatures;
- failed attachment replacement preserves the last valid artifact;
- stale async upload/submission completions cannot write into a newly opened request;
- external submission receipts record a **response**, not an independently verified outcome;
- document-import proposal serialization and optimistic-conflict regression coverage remains executable;
- deterministic UI evidence covers 32 representative states/interactions.

This closes the simple input/capture tranche. Redirect/delegate/wrong-recipient, draft/resume/amendment, invitation lifecycle, production scanning/quarantine/retry, explicitly governed multi-file collection and recurring import reconciliation remain later #27 work.

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

### #27.1 Today work-queue and Matter authority truth — COMPLETE IN PR #43

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

PR #43 merged from its exact CI/UI-evidence-green head. Do not reopen this seam under a new workflow model.

### #27.2a explicit work-requirement compiler / deterministic lifecycle projection — IN PR #45

This tranche expands actor work beyond Matter Actions only where canonical state determines one safe next step. It does **not** choose an assignee from an ambiguous lifecycle state.

Implemented in the current PR:

- [x] shared Decision/Response lifecycle responsibility policy is used by both command authorization and work compilation;
- [x] current canonical Matter state compiles into non-authoritative `WorkRequirement` / `WorkAmbiguity` values;
- [x] deterministic Response transitions such as transmitted → acknowledgement and rejected → draft can become actor work;
- [x] active Verification Contracts become outcome-check work only after their observation period and only while no current result exists;
- [x] ambiguous Decision/Response branches remain explicit blocked/unassigned projection state rather than selecting an arbitrary reviewer/challenger/authorizer;
- [x] current authority is resolved at projection time and Matter priority remains the materiality floor;
- [x] a required verification reviewer is assigned only if still eligible under current authority;
- [x] authority candidates are filtered through canonical Matter visibility before a READY assignment can exist;
- [x] delayed Matter events use **current** authority rather than the authority that existed at the historical event timestamp;
- [x] Matter events project immediately through the existing transactional outbox publisher;
- [x] a slower bounded maintainer reconciles restart/backfill and authority/delegation changes without another event or workflow framework;
- [x] deterministic `(tenant, kind, subject_type, subject_id)` Workflow-instance identity prevents duplicate projected workflows;
- [x] Matters with no lifecycle work do not create empty Workflow instances;
- [x] actor Workflow reads and Today admit only the supported `MATTER_ACTION` and `MATTER_LIFECYCLE` projections and recheck Matter visibility;
- [x] real verification context can appear in Today through progressive disclosure without fabricating recommendation, prepared work or completion receipt;
- [x] external-response work is labelled as an external response rather than an approval;
- [ ] exact final branch head must pass full CI plus the expanded 34-state rendered evidence matrix before merge.

See `docs/architecture/current-read-and-work-projection-boundary.md` and the PR #45 PostgreSQL integration tests.

### #27.2b policy-selected branches and Evidence Request recipient contract — NEXT AFTER #45

The compiler intentionally leaves two unresolved ownership problems visible rather than guessing:

- [ ] define a governed policy-selection record/contract for lifecycle states with multiple valid next transitions, especially Decisions and multi-branch Response states;
- [ ] ensure that selection itself is authority-aware, versioned, auditable and invalidated by relevant state/policy changes;
- [ ] compile the selected branch through the same `WorkRequirement` → authority → Workflow projection path rather than adding a second task engine;
- [ ] define a canonical Evidence Request recipient/routing contract; `why_you`, `created_by` and free-form request copy must never be treated as assignment truth;
- [ ] support internal/external Evidence Request work only after recipient scope, delegation, conflict, expiry/revocation and protected-record visibility are explicit;
- [ ] prove reassignment, replay/restart, ambiguous-policy and restricted-record behavior in PostgreSQL before exposing those rows in production Today.

### Later #27 work, in order

After #27.2b:

1. **Operating Program/Work mutation flows** — direct governed actions, saved role-aware views, delegation/recusal/conflict/escalation where domain commands exist, save/resume for complex work.
2. **Capture/Import lifecycle completion** — provenance classes, redirect/wrong-recipient/delegation, draft/resume/amendment, invitation expiry/revocation, production scanning/quarantine/retry, explicitly governed multi-file requests, recurring mappings and governed conversion to canonical records.
3. **Configure productization** — organization/identity sources, responsibility/authority matrices, routing/escalation builders, impact preview, maker-checker, effective dating/rollback, notification and security policy surfaces.
4. **Enterprise shell** — production Explore/reconstruction, actor notifications, identity/session/step-up context.
5. **Acceptance closure** — representative bank-user timed usability, complete state coverage, production assets and final accessibility/responsive evidence.

Do not recreate work already completed in PRs #31, #40, #43 or P0–P2 under new names.

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
- **WorkRequirement ≠ authoritative workflow state.** It is a deterministic compiler output over current canonical records and policy; Workflow Task remains a rebuildable actor projection.
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
canonical Matter state
→ deterministic work-requirement compilation
→ current authority + record-visibility resolution
→ existing transactional outbox / bounded reconciliation
→ idempotent supported Workflow Task projection
→ Today / workflow read surfaces
```

Matter Actions remain accountable business work and project through `MATTER_ACTION`. PR #45 adds `MATTER_LIFECYCLE` as a second **projection kind**, not a second workflow engine. Decision/Response/Verification records remain authoritative; lifecycle Task rows are rebuildable and obsolete rows are cancelled by reconciliation.

An ambiguous state produces no guessed actor. It remains unassigned/blocked until policy selects one valid transition. Current actor assignment is always evaluated from current authority, not from stale event-time routing.

### Capture truth

A submitted response can reference a request-bound `STORED_UNSCANNED` artifact. That proves what the respondent submitted; it does **not** prove evidence sufficiency or that production malware inspection has completed. Later production scanning/quarantine/retry must preserve this distinction and fail closed for evidence use.

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