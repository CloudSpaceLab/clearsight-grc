# Continuous Compliance Architecture

ClearSight maintains stable obligations in Programs and creates Matters only when change, exception, harm, uncertainty or external demand requires handling.

## Runtime loop

```text
Authority/source/system/human observation
→ normalized Signal or Observation
→ evidence, source, requirement, control and routing evaluation
→ Program state remains current or drift is created
→ minimum-intervention compilation
→ focused request, workflow task, Matter or decision
→ governed action
→ verification or acknowledgement
→ refreshed Program and readiness state
```

## Core services

- Source Registry and integration adapters;
- Signal ingestion and deduplication;
- Drift Engine;
- Evidence aging and contradiction;
- Context Assembly and Prefill;
- Minimum-Question Compiler;
- Authority Routing and Integrity;
- Durable Workflow;
- Recommendation and Task Compilation;
- Program state computation;
- Matter, Decision, Action and Verification;
- Continuous Readiness;
- Governed Automation Catalogue;
- temporal audit and reconstruction.

## Separation rules

- Signal is not incident.
- Drift is not approved conclusion.
- Evidence aging is not proof of control failure.
- AI recommendation is not authority.
- external execution is implementation evidence.
- implementation is not verified outcome.
- readiness is multidimensional and not an opaque score.

## Durable internal reconciliation

Source-health writes do not synchronously mutate Program state. The evidence transaction emits `SourceHealthChanged` into the existing transactional outbox. The PostgreSQL worker delivers that event through an ordered internal consumer before the event may be marked published:

```text
Evidence Source observation/maintenance transaction
→ outbox_events
→ source-health reconciliation consumer
→ compliance Signal/Drift update or exact recovery resolution
→ active Evidence Contract source dependency lookup
→ idempotent Program trigger
→ optional focused Matter through existing trigger policy
→ Program-status projection job
→ inbox_receipt
→ operational/external publisher
→ published_at
```

Delivery rules:

- `LogPublisher` is an observability sink, not evidence of internal delivery.
- An internal reconciliation error leaves the outbox event retryable.
- Inbox receipts are recorded only after all internal effects succeed.
- Retrying after a partial effect is safe because compliance Signals and Program triggers have independent dedupe keys.
- Source recovery is accepted only from the governed source-health consumer, not generic Signal ingestion.
- Program trigger dedupe includes the Program ID because trigger uniqueness is tenant-wide.
- A source recovery may emit a Program recovery trigger only when all active Evidence Contract sources for that Program are currently healthy.
- Unhealthy-to-unhealthy source changes update source drift without opening a new degradation episode.

This is the first implemented reconciliation class. Other evidence-aging, routing, control and verification event classes must use the same durable pattern rather than adding parallel event stores.

## Incremental computation

Changes invalidate only dependent Claims, Requirements, controls, routing paths and readiness dimensions. The API serves current projections; workers recompute affected state through idempotent jobs. Material commands use authoritative state and re-evaluate authority synchronously.
