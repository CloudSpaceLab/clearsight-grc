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
