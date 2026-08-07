# Governance Runtime

This document defines the executable runtime for routing-policy approval, delegation, timers, escalation and durable event delivery.

## Scope

The runtime is intentionally narrower than a generic workflow platform. It provides the minimum governed mechanisms required by Programs and Matters:

- maker-checker policy publication;
- approved, time-bound delegation;
- live segregation and delegation-cycle checks;
- durable reminders and escalation timers;
- transactional outbox delivery and retry;
- consumer inbox deduplication;
- independent bounded work-class supervision;
- visible terminal failure for poison timer/outbox work;
- append-only governance decisions.

## Policy lifecycle

```text
DRAFT
→ PENDING_APPROVAL
→ ACTIVE
→ RETIRED
```

The maker submits the exact version and checksum. A different authorized checker approves it. Activation is blocked when selectors are unresolved, ambiguous, or violate static routing constraints. Every transition records actor, rationale, prior state, new state and time, and emits an outbox event in the same transaction.

## Delegation lifecycle

```text
DRAFT
→ PENDING_APPROVAL
→ APPROVED
→ ACTIVE
→ EXPIRED
```

Approved or active delegation may move to `REVOKED`.

Approval checks:

- maker, delegator, delegate and approver separation;
- time-window validity;
- recursive delegation-cycle detection;
- active segregation rules for the delegated responsibility;
- optimistic version and current-state match.

The worker activates approved delegations at `starts_at` and expires them at `ends_at`.

## Work-class supervision

ClearSight keeps one deployable worker process while isolating independent failure domains. The process currently supervises five named work classes:

- `evidence-source-maintenance`;
- `program-projection`;
- `delegation-lifecycle`;
- `workflow-timers`;
- `outbox-delivery`.

Each class has its own poll interval, execution timeout, lease, batch size, retry budget and maximum backoff. Each class is intentionally single-flight. The runtime does not create a generic agent worker pool or another orchestration framework.

A class error or panic marks only that class degraded. Other classes continue running and the process remains alive unless the process context is cancelled or initialization fails. A class that later succeeds returns to current.

The execution timeout is always shorter than the claim lease. This prevents a normal class timeout from making a still-running claim eligible for simultaneous reclaim by another worker replica.

Class health is operational rather than a business-truth store. It exposes:

- `STARTING`, `CURRENT`, `DEGRADED` or `NEEDS_ATTENTION`;
- last success/failure and duration;
- bounded error detail;
- consecutive failures;
- processed count;
- for timers/outbox, pending count, terminal count, highest attempts and oldest pending time from the authoritative queue tables.

## Timers

A timer is unique by tenant and dedupe key. Workers claim due timers with `FOR UPDATE SKIP LOCKED`, a worker identity and a bounded lease. Expired claims may be reclaimed. Completion verifies claim ownership, marks the timer fired and creates its outbox event atomically.

Timer states are:

```text
READY → CLAIMED → FIRED
  ↑        │
  └────────┘ retry within budget
           └→ FAILED after retry budget
```

`CANCELLED` remains a terminal administrative state. A `FAILED` timer retains its error and failure time and is not claimable again until an explicit future recovery command exists. This prevents poison timers from consuming the worker indefinitely.

Timer types may include:

- reminder;
- deadline warning;
- escalation;
- evidence refresh;
- verification observation;
- delegation activation or expiry trigger.

## Outbox and inbox

Authoritative state and the outbox event commit together. Publishing is at-least-once:

```text
claim with lease
→ publish through constrained adapter
→ mark published
or
→ release with bounded exponential retry
→ dead letter after retry budget
```

Dead-lettered outbox events retain the last error and `dead_lettered_at` and are excluded from claims. They remain visible through queue health rather than retrying forever.

Publisher panics are contained per outbox item. One bad event is moved through its own retry/dead-letter path while later items in the same claimed batch continue.

Consumers record `(tenant, consumer, event_id)` after their required internal effects succeed. Duplicate delivery therefore does not duplicate the consumer result. Internal consumers may additionally check an existing receipt before replaying effects.

The current log publisher is an internal development/observability adapter. External email, messaging, ITSM or Probo delivery requires a dedicated adapter, idempotency contract, reconciliation and monitoring. A future external publisher must not be represented as delivered merely because an event was logged.

## Consistency and failure behavior

Strong consistency applies to:

- governance state transitions;
- actor and version checks;
- timer completion plus outbox creation;
- outbox claim ownership;
- inbox receipts;
- terminal timer/dead-letter transitions.

A worker crash leaves a lease that another worker can reclaim. A stale worker cannot complete a reclaimed timer or publish an event claimed by another worker. The memory runtime honours the same outbox lease rule as PostgreSQL so tests do not model weaker concurrency semantics.

Ordinary retriable work-class failures do not terminate the worker process. Exhausted timer/outbox retry budgets become visible terminal state instead of perpetual churn.

## Performance

- policy/delegation command acknowledgement: p95 ≤ 500 ms;
- due timer claim batch: p95 ≤ 250 ms for 50 items;
- outbox claim batch: p95 ≤ 250 ms for 50 items;
- worker operations are bounded and backpressured;
- claim indexes contain only active/unpublished, non-terminal rows;
- work classes remain single-flight until measured throughput justifies additional class-local concurrency.

## Production boundary

This phase does not yet provide enterprise identity synchronization, business-calendar calculation, parallel workflow joins, external-channel adapters, or a full workflow-definition language. It also does not expose a generic worker administration product surface. Those remain explicit later work when a concrete use case requires them.
