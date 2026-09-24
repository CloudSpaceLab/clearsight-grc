# Performance Test Plan

Performance acceptance proves user outcomes under representative bank workloads, not only endpoint microbenchmarks.

## Profiles

### Developer smoke

- 25 virtual users for 30 seconds;
- Today and authority resolution;
- <1% failure;
- Today p95 <1.5 seconds;
- authority p95 <150 ms in the scaffold.

```bash
k6 run tests/performance/smoke.js
```

### Pilot bank

Model at least 5,000 named users, 500 concurrent sessions, 1 million institutional/workflow records, 10 million annual observations, 100,000 annual artifacts, 100,000-row recurring imports, and deadline/notification bursts.

### Large-bank reference

Model 25,000 named users, 2,500 concurrent sessions, 10 million institutional/workflow objects, 100 million annual observations/assertions, 1 million annual artifacts, 1-million-row imports, 5,000 workflow/request events per hour bursts, and multi-year temporal history.

## Required scenarios

- context and Today;
- Program overview and Requirement exceptions;
- Work queue with keyset pagination;
- authority resolution with 100k+ active assignments/grants;
- request load/save/submit/duplicate submit;
- invitation redemption contention and replay;
- concurrent Matter version conflicts;
- million-row import with bounded interactive impact;
- outbox backlog and worker recovery;
- projection rebuild while serving current work;
- point-in-time reconstruction;
- package generation;
- source/model outage;
- noisy-neighbor isolation.

## Measurements and pass rules

Measure p50/p95/p99, throughput, saturation, database CPU/I/O/locks/buffers/WAL/lag, Go allocation/heap/goroutines/GC, queue depth/age/retries, projection freshness, browser bundle/render/interaction/layout, authorization cost, import throughput, recovery time, and cost.

Correctness and authorization assertions must pass under load. Reject unbounded queue/goroutine/memory/retry/database growth, leakage, duplicate material effects, or interactive SLO breach without activated isolation. Use structurally realistic generated data; do not simplify routing or matching into trivial fixtures.

## ROPA register volume — Windows developer-box run

The ROPA volume check uses the real in-memory `Service` to create 100,000 processing activities in one synthetic legal-entity scope (`tenant-load` / `entity-load`). The fixture deliberately creates activities without child collections so the bounded register-list cost and memory profile are measured without retaining 100,000 fully hydrated child collections. It is a local development measurement, not a PostgreSQL or CI result and not a p95 sample.

Command:

```bash
go test -tags load ./internal/ropa/ -run . -v -timeout 30m
```

| Read / operation | Budget | Actual result |
|---|---|---|
| Seed 100,000 activities through `Service.CreateActivity` | Must complete within the 30-minute test timeout; no seed latency SLO | **2.1667088s — PASS** |
| First bounded register page, 50 rows | **750ms** | **246.056ms — PASS** |
| Second register page using the first opaque cursor, 50 rows | **750ms** | **341.8448ms — PASS** |
| Dashboard projection maintainer over 100,000 activities | Must complete within the 30-minute test timeout; no separate full-rebuild latency SLO | **3m39.0121235s — PASS**; counts reconciled as total/new **100,000/100,000**, open/closed **0/0**, excluded/unknown **0/0** |
| Scoped dashboard projection read | **500ms** | **0s as displayed by the Go timer — PASS**; single stored-snapshot read, not a live aggregate |

Machine context for the recorded run: Microsoft Windows 11 Home build 26200, AMD Ryzen 7 7435HS (8 cores / 16 logical processors), 39.69 GB RAM, Go 1.25.13 `windows/amd64`, local in-memory repositories. The page measurements are single observations on this Windows developer box, not CI measurements or production capacity evidence.

## Report page volume — real PostgreSQL at target population

`internal/reporting/load_test.go` measures a report page through
`PostgresRepository.ListReportRows`, which is the path the 750ms budget covers.
Populations: 100,000 processing activities (75,000 in the measured `OPEN`
population, 30,000 carrying an open exception), 5,000 Programs (1,250 `AT_RISK`),
and 25,000 Matters (10,000 overdue). 50-row pages, 2 warm-up reads excluded,
50 measured samples per path, nearest-rank quantiles.

Commands:

```bash
go test ./internal/reporting/ -tags load -run TestReportPageBudget -count=1 -v
# with TEST_DATABASE_URL pointing at a database carrying all migrations
```

Observed on the recorded run (Windows 11 build 26200, AMD Ryzen 7 7435HS,
Go 1.25.13 `windows/amd64`, **PostgreSQL 18.6**):

| Dataset | Source population | Measured population | First p50 | First p95 | Cursor p50 | Cursor p95 | Max | Result |
|---|---:|---|---:|---:|---:|---:|---:|---|
| `PROCESSING_ACTIVITIES` | 100,000 | `status = OPEN`; 75,000 | 6.95ms | 9.00ms | 1.57ms | 2.51ms | 9.80ms | **PASS** |
| `PROCESSING_ACTIVITY_EXCEPTIONS` | 100,000 | `status = OPEN`; 30,000 with an open exception | 120.52ms | 136.09ms | 114.99ms | 130.99ms | 140.69ms | **PASS** |
| `PROGRAMS` | 5,000 | `overall_state = AT_RISK`; 1,250 | 180.86ms | 215.92ms | 177.59ms | 214.49ms | 249.51ms | **PASS** |
| `MATTER_EXCEPTIONS` | 25,000 | `due_condition = OVERDUE`; 10,000 | 507.10ms | 659.41ms | 506.27ms | 625.51ms | 667.61ms | **PASS** |

Every p50, p95 and recorded maximum is inside the 750ms budget, which is applied
to each reported p95 and is not widened.

**`MATTER_EXCEPTIONS` has the least headroom**: 659ms p95 and 668ms maximum
against 750ms, roughly 11%. The Matter exception dataset evaluates the
visibility predicate and the due-condition filter per candidate row, so its cost
scales with the Matter population rather than with the page size. At a larger
Matter population this path is the one that will reach the budget first. It is
recorded here as a known limit, not as a passing result with no caveat.

`PROCESSING_ACTIVITIES` and its cursor pages are index-backed and two orders of
magnitude under budget. `PROCESSING_ACTIVITY_EXCEPTIONS` costs more than the
unfiltered dataset because the exception predicate is evaluated per candidate
row.

### In-process fallback is a bound, not a budget result

When `TEST_DATABASE_URL` is unset the same test falls back to an in-process
repository holding the full unsorted population, so the measured call performs
the filter and sort rather than slicing a pre-selected slice. The recorded
fallback run on the same machine was:

| Dataset | First p50 | First p95 | Cursor p50 | Cursor p95 |
|---|---:|---:|---:|---:|
| `PROCESSING_ACTIVITIES` | 566.33ms | 607.12ms | 568.89ms | 622.99ms |
| `PROCESSING_ACTIVITY_EXCEPTIONS` | 293.04ms | 350.19ms | 293.35ms | 323.76ms |
| `PROGRAMS` | 4.54ms | 6.12ms | 4.07ms | 5.63ms |
| `MATTER_EXCEPTIONS` | 53.88ms | 61.28ms | 52.84ms | 56.17ms |

These figures are slower than the PostgreSQL numbers because the in-process path
sorts the whole population in Go while PostgreSQL uses `ropa_register_keyset_idx`.
They bound the worst case for a caller with no database; they are **not** the
query budget and must not be quoted as one. A figure without its layer is not a
result.
