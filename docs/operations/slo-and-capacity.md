# SLO and Capacity Model

Initial objectives; production targets are finalized with pilot deployment, residency, and support requirements.

## User-facing objectives

| Capability | Objective |
|---|---:|
| Authenticated core API availability | 99.9% monthly |
| Focused external capture availability | 99.9% monthly |
| Common deterministic page | p95 ≤1.5 s |
| Common durable command acknowledgement | p95 ≤500 ms |
| Request step save | p95 ≤500 ms |
| Request submit acknowledgement | p95 ≤750 ms |
| Authority resolution | p95 ≤100 ms uncached; ≤25 ms cached |
| Invitation redemption | p95 ≤500 ms |
| Scoped search/work queue | p95 ≤1.5 s |
| Bounded historical reconstruction | p95 ≤3 s |

AI and extraction are measured separately and never block manual operation.

## Async objectives

| Work | Objective |
|---|---:|
| Ordinary outbox/projection event | p95 visible within 5 s |
| Reminder/escalation timer | within 60 s of due time |
| Small-document extraction | p95 within 2 min |
| 100k-row import | within 10 min under pilot profile |
| Million-row import | within batch window without interactive SLO breach |
| Report/package | progress visible immediately; resumable |

## Recovery

Initial standard-hosted targets: authoritative database RPO ≤5 minutes and RTO ≤60 minutes; versioned object storage; stateless API replacement within minutes; workers resume without duplicate material effects; projections rebuild from authoritative history; invitation/session revocation survives restart.

## Capacity and backpressure

Scale or isolate on sustained API saturation, database lock/I/O/WAL/buffer pressure, queue-age breach, decision-relevant projection lag, noisy tenants, batch workloads affecting interactive SLOs, storage/egress growth, or authority-plan degradation.

Every async producer has a queue budget. Workers reduce batch/concurrency under pressure. Noncritical enrichment pauses before command/capture availability degrades. Notifications are grouped and deduplicated.

## Error budgets and cost

Review error-budget burn by capability, tenant, and dependency. Pause rollout when core availability, authorization, capture durability, or integrity burns exceed policy; AI availability has a separate budget. Track cost per active tenant/user, million observations, artifact/GB, workflow transitions, AI recommendations, imports, and packages without removing provenance, authority, history, or recovery controls.
