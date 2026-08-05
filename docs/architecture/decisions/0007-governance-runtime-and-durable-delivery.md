# ADR-0007 — Governance Runtime and Durable Delivery

Status: Accepted  
Date: 2026-08-05

## Context

ClearSight requires maker-checker routing changes, time-bound delegation, reminders, escalation and reliable downstream updates. Hard-coded approval chains, in-memory timers and direct external calls would weaken authority, recovery and reconstruction.

## Decision

Use explicit governance state machines in PostgreSQL, leased database timers, a transactional outbox and consumer inbox receipts. Begin with bounded PostgreSQL workers rather than a separate workflow platform or message broker.

## Consequences

- governance decisions and emitted events cannot diverge;
- workers can restart and reclaim expired work;
- external delivery is at-least-once and requires idempotent adapters;
- PostgreSQL remains the initial operational queue and must be monitored for backlog and lock contention;
- a broker may be added later without changing domain events.

## Guardrails

- maker and checker remain different principals;
- unresolved policy selectors block activation;
- delegation cycles and active segregation conflicts block approval;
- timers and outbox events use worker leases and ownership checks;
- a log-only publisher is development-only;
- raw protected content does not enter timer or outbox payloads.

## Revisit when

Measured backlog, retention, throughput, regional isolation or adapter fan-out exceed the PostgreSQL workload envelope documented in SLO and capacity tests.
