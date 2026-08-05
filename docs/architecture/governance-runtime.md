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

## Timers

A timer is unique by tenant and dedupe key. Workers claim due timers with `FOR UPDATE SKIP LOCKED`, a worker identity and a bounded lease. Expired claims may be reclaimed. Completion verifies claim ownership, marks the timer fired and creates its outbox event atomically.

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
```

Consumers record `(tenant, consumer, event_id)` before applying side effects. Duplicate delivery therefore does not duplicate the consumer result.

The current log publisher is an internal development adapter. External email, messaging, ITSM or Probo delivery requires a dedicated adapter, idempotency contract, reconciliation and monitoring.

## Consistency and failure behavior

Strong consistency applies to:

- governance state transitions;
- actor and version checks;
- timer completion plus outbox creation;
- outbox claim ownership;
- inbox receipts.

A worker crash leaves a lease that another worker can reclaim. A stale worker cannot complete a reclaimed timer or publish an event claimed by another worker.

## Performance

- policy/delegation command acknowledgement: p95 ≤ 500 ms;
- due timer claim batch: p95 ≤ 250 ms for 50 items;
- outbox claim batch: p95 ≤ 250 ms for 50 items;
- worker operations are bounded and backpressured;
- claim indexes contain only active/unpublished rows.

## Production boundary

This phase does not yet provide enterprise identity synchronization, business-calendar calculation, parallel workflow joins, external-channel adapters, or a full workflow-definition language. Those remain explicit later work.
