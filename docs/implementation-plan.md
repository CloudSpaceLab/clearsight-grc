# ClearSight implementation ledger

**Status date:** 2026-08-07  
**P0 closure:** PRs #25 and #30  
**P1.1:** PR #34  
**P1.2:** PR #35  
**P1.3:** PR #36  
**P1.4:** PR #38  
**P1.5:** PR #39  
**UI/UX foundation reconciliation:** PR #31  
**P2 schema ownership / dead compatibility:** PR #41

This is the authoritative execution ledger. Product, design, architecture and enterprise-productization documents define requirements and target behavior; this file controls current implementation order and capability truth. Completed-tranche detail belongs in focused architecture/review documents rather than being duplicated indefinitely here.

## 1. Current sequence

### #26 P0 executable integrity — COMPLETE

- [x] typed production route/access registry and verified actor/tenant binding;
- [x] persisted capture consolidation and bounded invitation/session security;
- [x] durable source-health reconciliation through outbox/inbox into Program/Matter/projection consequences;
- [x] independent bounded worker classes with retry/dead-letter behavior;
- [x] truthful compound-command and post-commit response semantics;
- [x] executable route/runtime-contract parity;
- [x] effective-authority convergence across routing rules, assignments, grants, delegations and segregation constraints.

Issue #26 is closed. Lower-priority semantic and cleanup work moved to #32 and #33 rather than being hidden inside P0.

### #32 P1 semantic/current-state correctness — COMPLETE IN PRs #34–#39

#### P1.1 Program-state truth — COMPLETE

- [x] effective-time Requirement, Applicability and Control Implementation selection;
- [x] Evidence Contracts participate only when their current Requirement/Control target participates;
- [x] Evidence Assessment validity is bounded by contract freshness in derivation and persistence;
- [x] all currently required Evidence Sources participate in Source Quality;
- [x] future/expired sources and records cannot pollute current state;
- [x] mandatory UNKNOWN dimensions cannot silently become overall `CURRENT`;
- [x] pause/resume preserves configured Program periods;
- [x] summary freshness exposes current Program version, assessed Program version, projection version and stale state;
- [x] stale last-known green state renders as updating rather than current;
- [x] PostgreSQL, memory and rendered-state tests cover temporal/source/freshness behavior.

#### P1.2 Matter closure current-record truth — COMPLETE

- [x] Decision currentness follows authoritative append/event order within each decision type;
- [x] later rejection/return/expiry/supersession invalidates an older favorable record;
- [x] one favorable Decision cannot mask another adverse current Decision type;
- [x] every current Authority Request response lineage must be coherently transmitted and acknowledged;
- [x] expired/unresolved exception authority cannot satisfy closure;
- [x] conditional authority requires explicit structured condition resolution;
- [x] verification is revalidated for assigned authority, independence, implementation chronology and observation period at result recording and closure;
- [x] adversarial unit/PostgreSQL tests cover negative and favorable paths.

#### P1.3 lifecycle-specific command responsibility — COMPLETE

- [x] command responsibility is derived from current state plus requested lifecycle target;
- [x] Decisions distinguish proposer, reviewer, independent challenger and authorizer;
- [x] Responses distinguish proposer/reviewer, signatory, transmitter and acknowledgement recorder;
- [x] material Matter close/cancel/decision-required/reopen paths resolve authorizer responsibility;
- [x] lifecycle validity is checked before authority execution;
- [x] stage actors are preserved in current records and reconstructed from the trusted event envelope;
- [x] migration `000016_lifecycle_command_responsibility` persists/backfills lifecycle actors and lifecycle states;
- [x] governance routing supports the lifecycle responsibility types without a second authorization/workflow engine.

See `docs/product/authority-routing-and-escalation.md`.

#### P1.4 bounded current reads and explicit work projection — COMPLETE

- [x] ordinary PostgreSQL Program/Matter detail no longer replays lifetime continuity-event history;
- [x] current Program or Matter detail is assembled from normalized tables in one SQL projection query;
- [x] supported dashboard lists remain keyset-paginated summary endpoints;
- [x] compatibility aggregate lists are normalized/projection-backed rather than event-history-backed;
- [x] replay remains available for history, point-in-time audit and reconciliation;
- [x] normalized current state is reconciled against authoritative reconstruction;
- [x] query-depth tests prove one current-detail SQL call after more than thirty aggregate events;
- [x] Decision/Response current order is materialized from owning Matter event order through migration `000017_bounded_current_reads`;
- [x] Program detail/history preserve assessed Program version, `overall` and `projection_version`;
- [x] replay advances aggregate `updated_at` from event order;
- [x] Matter Action remains accountable domain work and Workflow Task remains an idempotent actor-facing projection;
- [x] direct production HTTP Task create/transition routes are removed;
- [x] nullable workflow deadlines remain nullable;
- [x] deterministic reference journeys use one scenario clock.

See `docs/architecture/current-read-and-work-projection-boundary.md`.

#### P1.5 document-import durability/resource boundary — COMPLETE

- [x] PostgreSQL upload streams the original artifact into the configured object store before expensive extraction/analysis;
- [x] a `PENDING` import receipt and `DocumentImportProcessingRequested` outbox event commit atomically before the request returns;
- [x] restart/retry uses the existing outbox worker rather than a second document job/queue framework;
- [x] API and worker share the configured artifact root so processing is not process-memory-dependent;
- [x] OOXML extraction preflights archive-entry, aggregate expanded-size, unsafe-path and compression-ratio limits;
- [x] CSV is row-streamed; XLSX worksheets/shared strings are token-streamed instead of full worksheet materialization;
- [x] explicit worksheet, row, column, cell, per-cell, shared-string, retained-text, section and proposal budgets bound semantic materialization;
- [x] structural/resource breaches fail extraction while retaining the original artifact;
- [x] retained-source/proposal truncation exposes total/omitted counts and `content_truncated` rather than silently implying completeness;
- [x] import collection reads use body-free summaries; section text/proposal bodies are detail-only;
- [x] proposal review atomically updates one JSONB proposal element under the expected document version rather than rewriting the full proposal array in application code;
- [x] selected pending imports poll until terminal without overlapping requests or false completion claims;
- [x] hostile-file, cancellation, restart, duplicate-delivery, stale-review and rendered-state tests cover the boundary.

See `docs/architecture/document-import-resource-and-durability-boundary.md`.

Issue #32 is closed. Wider capacity tuning, malware scanning, distributed object storage and PDF/OCR adapters remain enterprise productization rather than hidden P1 blockers.

### UI/UX foundation reconciliation — COMPLETE IN PR #31

PR #31 was not merged from its stale pre-P1 history. Its UI/read-evidence work was rebuilt on completed P1 so older backend copies could not reverse current correctness.

- [x] stale lifecycle/closure/authority persistence changes and migration reversions were discarded;
- [x] direct Workflow Task mutation routes remain absent and Today preserves Matter Action → derived Task truth;
- [x] Program exact-target/deep-link summaries preserve P1.4 `projection_version` instead of fabricating revision `0`;
- [x] browser Capture/Evidence field contracts include governed file-format metadata and file/photo inputs are independently labelled for assistive technology;
- [x] Imports preserves P1.5 pending/failure/truncation/completeness truth while adding typed conflict recovery and serialized proposal review;
- [x] Today uses exact linked record/authority context, candidate-set semantics and verified actor scope rather than a contextless sample object;
- [x] typed browser errors distinguish conflict, forbidden, not-found and unavailable states without converting degraded reads into false empty states;
- [x] bank-reference UI/integration fixtures use one deterministic continuity/evidence clock;
- [x] current action vocabulary and rendered-state tests are reconciled rather than preserving stale test copy;
- [x] full backend/web CI and deterministic rendered UI evidence are required on the exact final PR head before merge.

The UI foundation is still not the same thing as full product completion. Richer governed operator execution, enterprise Configure/identity/notifications, broader Capture workflows and representative bank-user usability evidence remain later work under their existing product/enterprise plans.

### #33 P2 schema ownership and dead compatibility cleanup — COMPLETE IN PR #41

P2 starts from completed P1 and the reconciled UI foundation and removes capability ambiguity instead of adding another metadata or workflow layer.

- [x] every live durable PostgreSQL table has exactly one machine-checked ownership/maturity classification, including owner, writers, readers, lifecycle/valid-time semantics, retention/deletion policy and executable evidence;
- [x] the ownership guard reconstructs the live table set from ordered migrations and fails CI on a missing, duplicate or non-live register entry;
- [x] every `*.up.sql` migration is required to have a matching down migration;
- [x] unused foundation-era `audit_events` and `readiness_snapshots` are removed by fail-closed reversible migration `000019_schema_ownership_cleanup` rather than being mistaken for current audit/readiness capability;
- [x] migration 000019 refuses removal when either unsupported table unexpectedly contains data and CI proves apply → rollback → reapply on an empty supported state;
- [x] historical duplicate `evidence_requests` and `invitation_grants` remain retired under migration `000013_capture_consolidation` and are not counted as live ownership entries;
- [x] `workflow_instances`, `workflow_tasks` and `workflow_events` are explicitly classified as active projections; Task mutation handlers/service/repository methods are removed so Matter Action projection remains the supported Task write path;
- [x] `workflow_timers`, `outbox_events` and `inbox_receipts` are explicitly classified as infrastructure ledgers rather than business truth;
- [x] responsibility assignments, grants, routing/governance records and `effective_authority_routes` are classified according to their current authoritative/projection roles rather than their original scaffold status;
- [x] `automation_policies` is classified as governed configuration state, never execution evidence;
- [x] current Readiness remains derived from active drift assessments with `baseline_known=false`; no removed snapshot table is used to imply a known denominator;
- [x] the stale broad `api/openapi.yaml` duplicate is removed; `internal/httpapi/route_registry.go` → `api/runtime.openapi.json` is the sole executable route/access contract;
- [x] bounded bank-journey and document-import OpenAPI files remain descriptive domain schemas and cannot create routes or grant access;
- [x] the existing PostgreSQL integration contract now verifies Workflow Tasks through Matter Action projection rather than manufacturing Tasks through removed compatibility methods.

See `docs/architecture/durable-schema-ownership.md` and `api/README.md`.

Issue #33 should close when PR #41 merges on an exact CI-green head. Future durable schema changes must update the checked ownership register in the same change.

## 2. Canonical domain invariants

- **Program** = ongoing obligation/compliance continuity.
- **Matter** = bounded change, exception, finding, decision, action, response or verification case.
- **Matter Action ≠ Workflow Task.** Action is accountable business work; Task is actor-facing projected/routed work.
- **Signal ≠ conclusion.** A Signal is an observation that deterministic assessment may convert into drift or attention.
- **Submission ≠ sufficient evidence.** Evidence Contract assessment determines sufficiency.
- **Implementation ≠ verified outcome.** Completion alone cannot close material work that requires verification.
- **Recommendation ≠ approval.** Current authority remains explicit.
- **Automation Policy ≠ execution receipt.** Permission is not evidence that an action ran.
- **Intervention Summary ≠ authoritative state.** It is a read projection over canonical records.

Do not add parallel authorization, task, event, worker, receipt, document or generic workflow stacks that duplicate these foundations.

## 3. Current executable truth

### Route/access contract

`internal/httpapi/route_registry.go` is the canonical executable route inventory. `api/runtime.openapi.json` is its mechanically verified route/access/permission contract.

Only health routes are truly public. Bounded capture routes use capability access. Other protected routes require verified identity. Material commands resolve current authority at execution.

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

An optional response/detail read may degrade response detail; it may not reverse or falsely report a committed command.

### Program-state freshness

- `program.version` = current command aggregate version;
- `current_state.program_version` = Program version assessed by the state projection;
- `current_state.projection_version` = monotonic calculated projection revision.

A projection is stale when its assessed Program version is behind the current Program version.

### Work truth

```text
Matter Action
→ continuity event / outbox
→ idempotent Workflow Task projection
→ Today / workflow read surfaces
```

A Workflow Task does not independently redefine whether the accountable Matter Action is planned, blocked, implemented or cancelled.

### Document-import truth

```text
stored original
→ PENDING import + transactional outbox request
→ bounded worker extraction/analysis
→ terminal import detail
→ explicit human proposal review
```

A stored artifact is not an extracted artifact. Extracted text is not an exhaustive source when truncation/omission metadata says otherwise. A proposal is not an approved compliance conclusion.

### Durable-schema truth

A live table is not evidence of product capability unless it has an explicit owner and executable reader/writer semantics. `docs/architecture/durable-schema-ownership.md` is checked against the ordered migration result; domain code and migrations remain authoritative.

## 4. Current Today and automation truth

Non-demo Today projects active Workflow Tasks assigned to the verified principal. Completed/cancelled tasks are excluded. Unassigned/team work remains outside the principal-specific queue until routing/ownership resolves it.

Today does not fabricate recommendations, approvals or execution receipts from Task presence.

`automation_policies` describes governed eligibility/configuration boundaries. A policy does not prove that an automated action ran, succeeded or was independently verified.

## 5. Enterprise work after P2

Detailed enterprise requirements remain in `docs/engineering/enterprise-productization-implementation-plan.md` and product/design specifications.

Major later gates include controlled enterprise identity synchronization, configuration administration/rollback, notifications, step-up assurance, production object storage/malware scanning/retention, PDF/OCR provider isolation, representative capacity evidence, backup/restore/provider-outage exercises, and pilot-bank legal/configuration approval.

## 6. Release and validation rules

Checkboxes describe repository capability, not deployment readiness.

A tranche is not complete until relevant gates pass on its **exact head**:

- `gofmt` and `go vet`;
- race-enabled Go tests;
- PostgreSQL composition, migrations and serialized integration tests;
- TypeScript strict checking;
- Vitest/axe rendered-state tests;
- production Vite build;
- adversarial identity, tenant, authority, replay and degraded-path tests;
- representative query-count/performance/recovery evidence when cardinality or durability changes.

Never claim a branch or PR is CI-green based on an older commit.
