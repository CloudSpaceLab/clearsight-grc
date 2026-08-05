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

## Incremental computation

Changes invalidate only dependent Claims, Requirements, controls, routing paths and readiness dimensions. The API serves current projections; workers recompute affected state through idempotent jobs. Material commands use authoritative state and re-evaluate authority synchronously.
