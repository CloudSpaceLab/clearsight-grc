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

## Report page volume — Windows developer-box run

`internal/reporting/load_test.go` uses the same local in-memory boundary as the register check above. It creates 100,000 processing activities through the real `ropa.Service`: 60,000 complete activities and 40,000 with at least one open exception; 75,000 are in the measured `OPEN` population. It also generates 5,000 Programs and 25,000 Matters, then measures the selected populations used by each report dataset. The in-memory run fixture is populated with the already selected dataset/filter population, so this check measures bounded page and cursor materialization; it does not measure PostgreSQL filter evaluation. The fixture uses 50-row pages, 50 measured samples, and a 20-read batch per sample divided back to a per-page duration. Quantiles use the nearest-rank method; a warm-up page is excluded.

Command:

```bash
go test ./internal/reporting/ -tags load -run TestReportPageBudget -count=1 -v
```

Seed time for the recorded run was **11.5107056s**. The following are the observed p50/p95 results; the budget is applied to each reported p95 and is not widened.

| Dataset | Generated source population | Filter and measured page population | First page p50 | First page p95 | Cursor page p50 | Cursor page p95 | Max observed | Result |
|---|---:|---|---:|---:|---:|---:|---:|---|
| `PROCESSING_ACTIVITIES` | 100,000 | `status = OPEN`; 75,000 rows | 126.635µs | 203.875µs | 150.22µs | 201.27µs | 228.065µs | **PASS** |
| `PROCESSING_ACTIVITY_EXCEPTIONS` | 100,000 | `status = OPEN`; 30,000 open-exception rows | 135.17µs | 251.53µs | 144.105µs | 207.17µs | 483.105µs | **PASS** |
| `PROGRAMS` | 5,000 | `overall_state = AT_RISK`; 1,250 rows | 191.9µs | 282.195µs | 153.955µs | 206.76µs | 588.72µs | **PASS** |
| `MATTER_EXCEPTIONS` | 25,000 | `due_condition = OVERDUE`; 10,000 rows | 177.665µs | 265.615µs | 200.865µs | 281.87µs | 593.89µs | **PASS** |

All eight p50/p95 measurements, and every recorded maximum, were inside the **750ms** budget. These values measure the in-memory report page and cursor path only; they are not PostgreSQL, HTTP, browser, production-capacity or CI evidence.
