# ClearSight implementation ledger

**Status date:** 2026-08-07  
**P0 closure:** PRs #25 and #30  
**P1.1:** PR #34  
**P1.2:** PR #35  
**P1.3:** PR #36  
**P1.4:** PR #38

This is the authoritative execution ledger. Product, design, architecture and enterprise-productization documents define requirements and target behavior; this file controls current implementation order and capability truth. Detailed completed-tranche rationale belongs in the linked architecture/review documents rather than being duplicated indefinitely here.

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

### #32 P1 semantic/current-state correctness

P1 is correctness-first rather than feature-first.

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

#### P1.4 bounded current reads and explicit work projection — IMPLEMENTED IN PR #38

- [x] ordinary PostgreSQL Program/Matter detail no longer replays lifetime continuity-event history;
- [x] `CurrentPostgresRepository` assembles current Program or Matter detail from normalized tables in one SQL projection query;
- [x] supported dashboard lists remain the keyset-paginated Program/Matter summary endpoints;
- [x] compatibility aggregate list paths are normalized/projection-backed rather than event-history-backed;
- [x] event replay remains available for history, point-in-time audit and reconciliation;
- [x] reconciliation tests compare normalized current state with reconstructed authoritative state;
- [x] query-depth tests prove one current-detail SQL call after more than thirty aggregate events;
- [x] normalized Decision/Response rows materialize owning Matter event order through migration `000017_bounded_current_reads` rather than timestamp/UUID heuristics;
- [x] Program detail and historical Program state expose the existing `projection_version` consistently;
- [x] normalized Program-state JSON preserves `overall` rather than leaking the database-only `overall_state` name;
- [x] replay advances aggregate `updated_at` from authoritative event order so current and reconstructed state agree;
- [x] Matter Action remains accountable domain work; Workflow Task is an idempotent actor-facing projection produced from Matter Action outbox events;
- [x] direct production HTTP Task create/transition routes are removed; Workflow Tasks are read projections at that boundary;
- [x] nullable workflow deadlines remain nullable instead of receiving fabricated dates;
- [x] deterministic reference journeys use one scenario clock for continuity and evidence validation.

See `docs/architecture/current-read-and-work-projection-boundary.md`.

#### P1.5 document-import resource, durability and paging hardening — NEXT

Required focus:

- bounded upload/request size and parser/resource budgets;
- durable import state and retry semantics rather than process-local assumptions;
- deterministic proposal/review identity and idempotency;
- bounded list/detail/proposal paging for large documents;
- failure/restart recovery and malformed/adversarial document tests;
- no second document/evidence/task pipeline where existing foundations are sufficient.

P1.5 should begin only from the merged P1.4 main baseline.

### #33 P2 schema ownership and dead compatibility cleanup — AFTER P1

P2 owns cleanup that does not need to remain mixed into semantic P1 delivery, including:

- classification/removal of dead compatibility handlers and service methods;
- consolidation or removal of duplicated descriptive/client schema surfaces;
- broad `api/openapi.yaml` ownership versus the mechanically verified executable `api/runtime.openapi.json` route/access contract;
- generated/manual client duplication where deletion is safer than adding another framework.

Do not represent the descriptive OpenAPI document as executable authorization truth.

## 2. Canonical domain invariants

These distinctions are mandatory:

- **Program** = ongoing obligation/compliance continuity.
- **Matter** = bounded change, exception, finding, decision, action, response or verification case.
- **Matter Action ≠ Workflow Task.** Action is accountable business work; Task is actor-facing projected/routed work.
- **Signal ≠ conclusion.** A Signal is an observation that deterministic assessment may convert into drift or attention.
- **Submission ≠ sufficient evidence.** Evidence Contract assessment determines sufficiency.
- **Implementation ≠ verified outcome.** Completion alone cannot close material work that requires verification.
- **Recommendation ≠ approval.** Current authority remains explicit.
- **Automation Policy ≠ execution receipt.** Permission is not evidence that an action ran.
- **Intervention Summary ≠ authoritative state.** It is a read projection over canonical records.

Do not add parallel authorization, task, event, worker, receipt, document, or generic workflow stacks that duplicate these foundations.

## 3. Current executable truth

### Route/access contract

`internal/httpapi/route_registry.go` is the canonical executable route inventory. `api/runtime.openapi.json` is its mechanically verified route/access/permission contract.

Only the health routes are truly public. Bounded capture routes use capability access. Other protected routes require verified identity. Material commands resolve current authority at execution.

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

The distinct version meanings remain:

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

## 4. Current Today and automation truth

Non-demo Today projects active Workflow Tasks assigned to the verified principal. Completed/cancelled tasks are excluded. Unassigned/team work remains outside the principal-specific queue until routing/ownership resolves it.

Today does not fabricate recommendations, approvals or execution receipts from Task presence.

`automation_policies` describes governed eligibility/configuration boundaries. A policy does not prove that an automated action ran, succeeded or was independently verified.

## 5. Enterprise work after semantic P1

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
