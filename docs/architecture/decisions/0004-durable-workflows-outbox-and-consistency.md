# ADR-0004 — Durable Workflows, Outbox, and Consistency

**Status:** Accepted  
**Date:** 2026-08-04

## Context

ClearSight workflows span human decisions, timers, source retrieval, imports, AI, notifications, external tools, observation periods, and regulator acknowledgement. These operations cannot rely on synchronous request chains or distributed transactions.

## Decision

Use durable typed workflow state with:

- local authoritative transactions;
- transactional outbox events;
- idempotent inbox/consumers;
- optimistic concurrency and state versions;
- durable timers and working calendars;
- at-least-once processing with idempotent commands;
- explicit retries, partial failure, cancellation, supersession, and compensation;
- saved drafts, current owner, blockers, and changed-since-last-view;
- strong consistency for material local commands and explicit eventual consistency for projections.

External execution is not covered by a claim of exactly-once delivery. ClearSight records attempts, external versions, reconciliation, and outcome verification.

## Consequences

Long-running work remains resumable across process restarts and provider failures. Implementations must design idempotency, versioning, and reconciliation for every side effect.

## Guardrails

- no material state change from an unvalidated free-form event;
- events contain safe references, not raw evidence or tokens;
- duplicate and reordered events are tolerated;
- stale commands fail or are reconciled explicitly;
- projection lag cannot masquerade as current authoritative state;
- an external success response does not close a Matter or prove outcome;
- timers and escalations are reconstructable.

## Validation

Test worker restart, duplicate event, event reordering, retry storm, partial external action, source/model outage, policy change, concurrent review, cancellation, and replay.

## Revisit when

Revisit the workflow implementation when benchmarked scale, timer volume, isolation, or operational ownership requires a dedicated runtime or service boundary.
