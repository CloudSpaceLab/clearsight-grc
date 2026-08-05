# ADR 0009 — Event-backed Programs and typed Matters

## Status

Accepted for the application foundation.

## Context

ClearSight needs one durable record for continuing obligations and a different record for bounded change or exception. A single risk-register row cannot preserve requirement interpretation, evidence history, decision authority, remediation and verified outcome without becoming ambiguous.

## Decision

- Programs and Matters are separate aggregates.
- PostgreSQL stores normalized projections and an append-only continuity event stream.
- Aggregate commands use optimistic versions and transactional outbox delivery.
- Program status is deterministic and reason-bearing.
- Signals are idempotent inputs; they do not silently become conclusions.
- Matter lifecycles and closure are type-aware.
- Task completion, action implementation and outcome confirmation remain separate.
- API codes remain stable while primary UI labels use plain working language.

## Consequences

Positive:

- point-in-time reconstruction;
- explicit closure evidence;
- safe concurrent updates;
- one Matter can affect several Program objects;
- human-friendly copy does not weaken machine semantics.

Costs:

- more tables and lifecycle rules;
- replay and projection consistency must be tested;
- commands require expected versions;
- high-volume list reads will later need dedicated projections.

## Rejected alternatives

- one generic risk/issue table;
- mutable rows without event history;
- closing when tasks are marked complete;
- exposing enum codes as primary user copy.
