# Current-read and work-projection boundary

**Status:** P1.4 runtime contract  
**Issue:** #32  
**Implemented by:** PR #38

This note defines the narrow execution boundary introduced in P1.4. It exists to keep ordinary ClearSight use bounded as records age and to preserve one business source of truth for accountable work.

## 1. Current state is not reconstructed history

Current Program and Matter state is already normalized transactionally into PostgreSQL tables. Ordinary reads must use those current records rather than rebuilding the aggregate from every historical event.

Production current-state behavior is therefore:

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

`CurrentPostgresRepository` is the production current-state reader. `PostgresRepository` remains the event-capable repository used for writes and historical reconstruction.

### Program and Matter detail

`GetProgram` and `GetMatter` assemble their current aggregate with one PostgreSQL JSON projection query. Query count is independent of aggregate event-history depth.

The acceptance suite creates more than thirty aggregate events and asserts that each current-detail read still uses exactly one SQL call while matching reconstructed authoritative state.

### List views

Supported product list views use the bounded keyset summary endpoints:

- `/api/v1/program-summaries`
- `/api/v1/matter-summaries`

The older aggregate list endpoints remain compatibility surfaces, but their item detail is normalized/projection-backed rather than lifetime-event replay. They are not the supported high-cardinality dashboard path.

## 2. Projection identity is explicit

Program state already has three distinct version concepts:

- `program.version` — current command aggregate version;
- `current_state.program_version` — Program version assessed by the state projection;
- `current_state.projection_version` — monotonic projection revision.

P1.4 exposes the existing `projection_version` consistently in current detail and point-in-time state reads. It does not create a second freshness counter.

A state snapshot is stale when its assessed Program version is behind the current Program version. Projection version identifies the particular calculated snapshot; it is not a substitute for assessed Program version.

## 3. Current Decision and Response order is materialized

Decision and Response currentness must follow authoritative Matter event order, not timestamp/UUID heuristics.

Migration `000017_bounded_current_reads` adds the owning Matter aggregate sequence (`matter_version`) to normalized Decision and Response rows, backfills it from `continuity_events`, and keeps it synchronized from newly inserted events.

This allows normalized Matter reads to preserve the same lifecycle order as event reconstruction without querying the event log.

## 4. Matter Action is business work; Workflow Task is projection

The canonical distinction is:

```text
Matter Action = accountable domain work
Workflow Task = actor-facing routed/projection work
```

A Workflow Task must not become an independent second business record that can contradict its Matter Action.

### Projection path

Matter Action events are delivered through the existing transactional outbox:

```text
ACTION_ADDED / ACTION_STATE_CHANGED
→ outbox delivery
→ inbox receipt for workflow-matter-action-v1
→ idempotent workflow instance/task projection
→ Today / workflow read surfaces
```

The projector maps Action state to Task state:

| Matter Action | Workflow Task |
| --- | --- |
| `PLANNED` | `READY` |
| `IN_PROGRESS` | `IN_PROGRESS` |
| `BLOCKED` | `BLOCKED` |
| `IMPLEMENTED` | `COMPLETED` |
| `CANCELLED` | `CANCELLED` |

One Matter Action has at most one `MATTER_ACTION` workflow instance and one `matter-action` task. Duplicate outbox delivery is idempotent through an inbox receipt.

The projected task carries the Matter and Action identifiers needed to open the authoritative issue rather than duplicating the Action payload as a new business object.

### HTTP boundary

Production HTTP exposes Workflow Tasks as a read projection only. Direct Task create/transition routes are removed from the executable route contract.

Users progress accountable work through the Matter Action command path; the derived Task converges from the resulting domain event.

Internal workflow service methods remain temporarily for compatibility/test infrastructure and are candidates for P2 dead-surface cleanup rather than P1.4 architectural expansion.

## 5. Optional deadlines remain optional

`workflow_tasks.due_at` is nullable in PostgreSQL. The Go Task model therefore uses an optional timestamp instead of manufacturing a fake deadline.

Today may render an undated task without claiming that a deadline exists.

## 6. Historical reconstruction remains available

P1.4 does not remove event history. Event replay is retained for:

- `/history` reads;
- point-in-time audit;
- reconciliation tests;
- forensic reconstruction.

Replay also advances aggregate `updated_at` from authoritative event order so current normalized state and reconstructed state do not disagree about the last material change time.

## 7. Executable contract ownership

`api/runtime.openapi.json` is the mechanically verified executable route/access contract and reflects the read-only Workflow Task boundary.

`api/openapi.yaml` remains a broader descriptive schema document with duplicated request/client detail. It is not the source of executable authorization truth. Consolidating or removing those duplicate schema/client surfaces belongs with the schema-ownership cleanup already tracked in #33 rather than introducing another contract generator in P1.4.

## 8. P1.4 acceptance evidence

P1.4 is complete only when CI proves all of the following on the exact head:

- normalized Program detail matches reconstructed authoritative Program state;
- normalized Matter detail matches reconstructed authoritative Matter state;
- current Program and Matter detail remain fixed-query as event history grows;
- Program projection version survives current and historical state reads;
- duplicate Action events do not duplicate workflow/task projections;
- an implemented/cancelled Action converges its actor-facing Task to the matching terminal state;
- nullable deadlines remain readable;
- production route/runtime contract contains no direct Workflow Task mutation route;
- existing PostgreSQL migrations, integration suites, race tests, vet and web gates remain green.
