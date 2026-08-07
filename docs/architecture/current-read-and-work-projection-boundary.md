# Current-read and work-projection boundary

**Status:** P1.4 current-read contract + #27.2a lifecycle-work extension  
**Issues:** #32, #27  
**Implemented by:** PR #38; lifecycle extension in PR #45

This note defines how ClearSight reads current state and turns canonical accountable/lifecycle state into actor-facing work without creating a second business source of truth.

## 1. Current state is not reconstructed history

Current Program and Matter state is normalized transactionally into PostgreSQL tables. Ordinary reads use those current records rather than rebuilding the aggregate from every historical event.

```text
Program/Matter command
→ normalized current row(s)
→ append-only continuity event
→ transactional outbox
→ commit

ordinary current detail
→ normalized current tables

point-in-time/history read
→ append-only event history / historical projection
```

`CurrentPostgresRepository` is the production current-state reader. Event-capable repositories remain available for writes and historical reconstruction.

`GetProgram` and `GetMatter` assemble current aggregates with bounded PostgreSQL projection queries whose query count does not grow with event-history depth. Supported list views remain the bounded summary endpoints.

## 2. Projection identity is explicit

Program state keeps distinct command and projection versions:

- `program.version` — current command aggregate version;
- `current_state.program_version` — Program version assessed by the state projection;
- `current_state.projection_version` — monotonic calculated projection revision.

A snapshot is stale when its assessed Program version is behind the current Program version. Projection version identifies the calculated snapshot; it is not a replacement for assessed Program version.

Decision and Response currentness follows authoritative Matter event order. Migration `000017_bounded_current_reads` materializes owning Matter sequence on normalized lifecycle rows so current reads preserve event order without replaying lifetime history.

## 3. Business work and actor work are different records

Canonical distinction:

```text
Matter Action / Decision / Response / Verification = authoritative domain state
WorkRequirement / WorkAmbiguity                    = deterministic compiler output
Workflow Task                                       = rebuildable actor-facing projection
Today                                               = actor read projection
```

A Workflow Task may route or summarize work; it may not redefine whether an Action is implemented, a Decision is approved, a Response is transmitted, evidence is sufficient, or an outcome is verified.

Direct production Workflow Task mutation APIs and service/repository mutation methods were removed in P2. Canonical commands are the supported write path.

## 4. Matter Action projection

Matter Action events use the existing transactional outbox:

```text
ACTION_ADDED / ACTION_STATE_CHANGED
→ outbox delivery
→ inbox receipt for workflow-matter-action-v1
→ idempotent MATTER_ACTION workflow/task projection
→ Today / workflow reads
```

Action state maps to Task state:

| Matter Action | Workflow Task |
| --- | --- |
| `PLANNED` | `READY` |
| `IN_PROGRESS` | `IN_PROGRESS` |
| `BLOCKED` | `BLOCKED` |
| `IMPLEMENTED` | `COMPLETED` |
| `CANCELLED` | `CANCELLED` |

The Task carries only the routing/display context needed to open the authoritative Matter. Canonical Matter ID, priority and access scope are joined internally for actor-read enforcement and are not a second public Task contract.

## 5. Lifecycle work compiler

PR #45 adds deterministic lifecycle work without adding another workflow engine.

```text
current Matter aggregate
→ CompileMatterWork
   ├─ executable WorkRequirement
   │  → current authority resolution
   │  → canonical Matter visibility filter
   │  → MATTER_LIFECYCLE Workflow projection
   │  → actor-scoped Today / workflow reads
   └─ WorkAmbiguity
      → no Workflow Task
      → wait for governed policy selection
```

### Compiler rule

A direct `WorkRequirement` exists only when current canonical state determines one valid next action/responsibility or an explicit policy has selected one.

If state has multiple valid next transitions, the compiler emits `WorkAmbiguity`. It must not select an arbitrary reviewer, challenger, authorizer or other actor merely because the record is in a familiar lifecycle state. Ambiguity is **not persisted as an unassigned Task**; doing so would turn Workflow into a shadow lifecycle register without giving anyone executable work.

Decision/Response lifecycle responsibility rules are shared with HTTP command authorization so projected work and executable command authority cannot drift into separate policy tables.

### Deterministic Response work

Examples that are currently safe to compile:

- transmitted Response → record acknowledgement;
- rejected Response → revise/return to draft.

Response states with several valid next transitions remain compiler ambiguity until a governed policy-selection contract exists.

### Verification work

An ACTIVE Verification Contract can compile to `matter.outcome.record` only when:

- no current result already exists;
- a linked Action, when present, is implemented;
- the observation period has elapsed.

The projected responsibility/materiality matches the executable `matter.outcome.record` command boundary: `REVIEWER`, materiality 4, with Matter priority as the floor.

If the contract names a required reviewer, the projector assigns that principal only when the principal is still eligible under **current** authority and can read the Matter. The Task may expose the real expected outcome/method/check time through Today progressive disclosure, but it does not claim the outcome is verified before a Verification Result exists.

### Evidence Requests are intentionally not inferred

Current Evidence Request records do not yet have a canonical recipient/routing contract sufficient to derive actor work. `why_you`, `created_by`, invitation copy or other descriptive fields are not assignment truth.

Ordinary Evidence Request projection remains out of this compiler until #27.2b defines a governed recipient/delegation/conflict/expiry contract.

## 6. Current authority, visibility and convergence

Lifecycle work uses **current projection time** for authority resolution. A delayed outbox event must not resurrect an actor whose assignment/delegation was valid only at the historical event time.

Assignment is fail-closed:

1. resolve current authority through the existing authority engine;
2. reduce candidate principals to those who can read the owning Matter;
3. assign READY only when exactly one eligible visible actor exists, or when an explicitly required principal remains eligible and visible;
4. otherwise keep the **executable requirement** unassigned/BLOCKED with a truthful routing reason.

Unexpected authority-service failure is operational failure and returns an error for retry; it is not persisted as a fake business `BLOCKED` state.

Matter events trigger immediate recompilation through the existing outbox publisher. A slower bounded maintainer exists for restart/backfill and authority/delegation convergence when no Matter event was emitted. It does **not** scan every historical Matter: it targets Matters with an existing lifecycle projection, deterministic Response work, or a Verification Contract whose observation window is actually ready. This is reconciliation, not another scheduler or business workflow source.

## 7. Projection identity and idempotency

Migration `000020_workflow_projection_identity` adds fail-closed uniqueness for:

```text
(tenant_id, kind, subject_type, subject_id)
```

This gives one deterministic Workflow instance per supported projection identity and makes replay/concurrent delivery converge instead of creating duplicate instances.

`MATTER_LIFECYCLE` uses one workflow instance per Matter and deterministic `step_key` values per executable requirement. Material task changes emit `WORK_REQUIREMENTS_RECONCILED`; duplicate reconciliation that changes nothing emits no duplicate work event.

A Matter with no current executable lifecycle requirement does not create an empty workflow. Existing lifecycle tasks that cease to be current are cancelled by reconciliation.

## 8. Actor-read boundary

Production actor work admits only the supported Matter-backed projection kinds:

- `MATTER_ACTION`;
- `MATTER_LIFECYCLE`.

The PostgreSQL read filters terminal/unsupported/inaccessible work **before LIMIT**, orders active work deadline-first, joins canonical Matter metadata, and is followed by a Go `MatterWorkVisibleTo` recheck as defense in depth.

`GET /workflow/tasks` is bound to the verified tenant/principal. A query cannot request another tenant or principal and obtain their work.

Unsupported legacy/generic Workflow kinds do not enter current actor work merely because rows exist.

## 9. Optional deadlines remain optional

`workflow_tasks.due_at` remains nullable. Actor work may be undated without manufacturing a deadline.

For verification work, the ready/check time is derived from the actual contract observation window and may be surfaced as verification context.

## 10. Historical reconstruction remains available

None of these projections remove event history. Replay remains for:

- `/history` reads;
- point-in-time audit;
- reconciliation tests;
- forensic reconstruction.

Current actor work is derived from current canonical state; historical reconstruction remains separate from the live queue.

## 11. Executable contract ownership

`internal/httpapi/route_registry.go` is the executable route/access inventory and `api/runtime.openapi.json` is its mechanically verified projection.

The former broad duplicate `api/openapi.yaml` was removed in P2. Descriptive bank-journey/document schemas do not create routes or authorization truth.

Workflow Task remains read-only at the HTTP product boundary. Users progress work through canonical Matter/lifecycle commands, after which projection reconciliation updates actor work.

## 12. Acceptance evidence

P1.4 acceptance remains protected by normalized-current-read and Matter Action projection tests.

The #27.2a lifecycle extension additionally requires exact-head CI to prove:

- Decision/Response compiler output never invents an actor or Workflow Task for a multi-branch lifecycle state;
- Verification work does not appear before its observation period and does not repeat after a current result exists;
- delayed delivery resolves current authority rather than historical event-time authority;
- required/candidate actors must retain canonical Matter visibility;
- duplicate projection delivery is idempotent;
- a Matter with no executable lifecycle work creates no empty Workflow instance;
- authority/visibility changes converge existing Task assignment safely;
- bounded reconciliation targets candidate/previously-projected Matters rather than the full Matter population;
- `MATTER_ACTION` behavior remains intact;
- migration `000020` applies, rolls back and reapplies;
- actor reads remain tenant/principal scoped and pre-limit visibility safe;
- Today shows real verification context without fabricating recommendation, prepared work or completion receipt;
- rendered browser evidence covers collapsed and expanded lifecycle-work context on desktop and mobile in addition to the existing UI matrix.
