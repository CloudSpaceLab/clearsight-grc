# Program and Matter foundation

This architecture implements the continuity layer that sits between sources/evidence and workflow execution.

## Boundaries

A **Program** is the durable record for an ongoing obligation or assurance activity. It owns requirements, applicability decisions, control objectives, control implementations, evidence checks and calculated status.

A **Matter** is a bounded record for a specific change, gap, finding, request, exception or incident. It owns decisions, actions, response packages, outcome checks and closure.

Programs do not become ticket containers. Matters do not replace the continuing compliance record.

## Write model

Material commands use optimistic versions. A successful command writes, in one PostgreSQL transaction:

1. the normalized current-state projection;
2. an append-only `continuity_events` record;
3. an outbox event for downstream processing;
4. the new aggregate version.

A competing version returns a conflict. The caller must reload rather than overwrite.

## Program state

Program status is derived deterministically from:

- approved requirements;
- current applicability decisions;
- requirement-to-control coverage;
- implementation state;
- active evidence checks;
- latest evidence assessments and expiry;
- open linked Matters;
- source-health triggers;
- program period.

The state engine records a snapshot with dimension-level reasons. It never changes a legal conclusion solely because a signal arrived. A signal may record uncertainty, refresh status and open a Matter.

## Trigger idempotency

`program_trigger_events` reserves a tenant-scoped dedupe key. The Program event stream records the trigger, and supported trigger types create at most one open Matter for that key. A retry returns the existing Matter.

The service can recover when reservation succeeded but event or Matter creation did not: it checks the event stream and the open trigger key independently.

## Typed Matter closure

Closure is a domain decision, not a generic status update.

Common rules:

- no open actions;
- each active outcome check has a latest passing result;
- a regulatory change has an approved position and, unless no change is required, an implemented change and outcome check;
- an exception has approved conditions or an expiry;
- an authority request has an acknowledged response;
- high-impact findings, gaps and incidents have an outcome check.

Failed outcome checks follow the contract response: reopen work, request an authorized decision, create a follow-up Matter, or block closure.

## Reconstruction

Programs and Matters can be reconstructed at an RFC3339 timestamp from `continuity_events`. Projections are performance aids; the event stream is the reconstruction source.

## Current limits

- actor identity is supplied by the caller; authenticated actor binding remains required before production administration;
- authority resolution is not yet automatically invoked by every Program/Matter command;
- aggregate lists use bounded event replay and will need projection-first reads before high-cardinality production use;
- bulk imports, dependency graph propagation and scheduled evidence-aging triggers remain later work.

// Program Activity reads schedule history by aggregate, not by outbox delivery state.

## Program Activity read contract

GET /api/v1/programs/{id}/activity binds the verified account and legal entity before querying. The evidence read model joins retained continuity events, immutable submissions and review decisions, linked document bindings and schedule outbox receipts. SQL evaluates current request assignment, Program owner/authorizer access and restricted issue visibility before search and paging. Raw domain payloads, answer bodies and artifact storage keys are excluded.

Queries accept search (maximum 200 characters), category, event_type, inclusive UTC from/to dates, limit 1–100 and a query/account-scoped cursor. Timestamp plus event identifier provides deterministic keyset ordering at equal timestamps. Each cursor retains an upper read timestamp; permission changes still apply on every page. This is retained-event history, not a cross-request database transaction snapshot. No new domain event is inferred for a change that was never recorded. Schedule history currently uses the existing outbox receipts and follows their retention policy.

Migration 000151 adds schedule-event and request-review read indexes. Existing per-aggregate continuity and capture request/artifact indexes support exact Program reads. No history backfill or retention-policy change is included. V2 export traverses filtered pages and fails without downloading partial history on error; it asks users to narrow queries above its 20,000-event safety bound. Validation covers a 408-event fixture, equal timestamps, all five categories, immutable document identity, exact filters, restricted issues and tenant/entity isolation. Larger production cardinalities require deployment query-plan and retention review.
