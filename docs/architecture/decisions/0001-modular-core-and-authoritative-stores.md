# ADR-0001 — Modular Core and Authoritative Stores

**Status:** Accepted  
**Date:** 2026-08-04

## Context

ClearSight spans Programs, Matters, evidence, routing, invitations, reporting, AI, and integrations. Premature service decomposition would add distributed consistency, deployment, authorization, and recovery complexity before workload boundaries are known.

## Decision

Begin with a coherent modular application using:

- a relational authoritative store for domain state and temporal versions;
- versioned object storage for original and derivative artifacts;
- durable workflow/jobs, transactional outbox, and idempotent inbox;
- rebuildable search, work-queue, graph, vector, and reporting projections;
- cache as an optimization, never authority;
- explicit bounded-context schemas, commands, events, and ownership.

A dedicated graph database, vector database, or broad microservice estate is not required for the first release.

## Consequences

Benefits include simpler material transactions, consistent authorization, easier point-in-time reconstruction, and lower operational overhead. Costs include the need for disciplined internal boundaries, early partition design, and asynchronous handling of large work.

## Guardrails

- projections cannot become truth systems;
- raw evidence remains outside logs and events;
- material commands use version checks;
- large ingestion and report generation remain resumable;
- modules expose domain commands rather than generic cross-schema CRUD;
- later service extraction must preserve temporal, authorization, and idempotency semantics.

## Validation

Benchmark the reference workload in `system-data-and-performance.md`, including tenant skew, large imports, routing, Program invalidation, protected search, and recovery.

## Revisit when

Split a module only when measured evidence shows materially different scale, latency, security or residency isolation, independent release ownership, or sustained contention that cannot be addressed within the modular core.
