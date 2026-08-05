# Continuous Compliance Architecture

Implementation structure is controlled by [`application-architecture.md`](application-architecture.md); data/workload rules by [`data-model-and-storage.md`](data-model-and-storage.md) and [`system-data-and-performance.md`](system-data-and-performance.md).

## Operating loop

```text
Authoritative source, schedule, event, or report
→ normalize observations and source health
→ update a Program or create/update a typed Matter
→ assemble authorized institutional context
→ resolve performer, owner, reviewer, challenger and authorizer
→ search existing evidence
→ request only unresolved facts through a safe channel
→ prepare grounded recommendation or draft
→ decide, act or respond within authority
→ verify outcome or acknowledgement
→ update Program state, queues, reports and history
```

## Core services

- Institution and Scope
- Source Registry
- Programs
- Matters
- Authority Routing
- Evidence and Capture
- Decision and Action
- Verification and Assurance
- Regulatory and Authority Intelligence
- Governed AI

## Program computation

Program state is a versioned projection over approved Requirements/applicability, scoped controls, Evidence Contracts/current conclusions, exceptions, assurance, schedules/filings/source health, and open Matters. Changes invalidate only affected dimensions. Heavy recomputation is asynchronous; pages show current state and freshness.

## Matter composition

Each subtype defines trigger/source, states/transitions, affected scope, roles/authority, evidence/contradiction, decision/action/response, and closure/cancellation/merge/split/supersession/reopening. Shared workflow mechanics do not erase domain-specific legal, privacy, independence, or deadline rules.

## Context and request compilation

ClearSight builds a purpose-bound context package containing sourced values, unresolved facts, contradictions, history, safe actions, and its version. Request compilation evaluates current evidence, identifies exact gaps, ranks eligible sources/recipients, chooses the least burdensome approved form, resolves route/channel, creates the request/session policy, and stops when satisfied elsewhere.

## Triggers and failure

Calendar, change, external source, event, evidence, degradation, threshold, and verification triggers are idempotent, version-aware, explainable, and replayable.

When a source, model, worker, or adapter fails, deterministic state remains visible, stale age is explicit, safe fallback is offered, unsafe action is blocked, drafts/jobs remain resumable, retries are idempotent, and recovery is audited.

## Conformance

A vertical slice is conformant only when it proves the complete loop, actor routing, evidence minimization, explicit consistency, degraded operation, verification before closure, point-in-time reconstruction, and its workload/SLO profile.
