# ADR 0005: Initial Implementation Stack

- **Status:** Accepted
- **Date:** 2026-08-05

## Context

ClearSight needs predictable performance, low operational overhead, strong transactions, a refined browser experience, and a path from pilot to large multi-entity deployment without requiring a distributed platform before product behavior is proven.

## Decision

Use Go for API/worker processes; React, TypeScript, Vite and Tailwind for the web client; PostgreSQL for authoritative state, durable workflow, outbox and initial projections; S3-compatible versioned object storage for artifacts; OpenAPI for HTTP contracts; and containers for local/deployment foundations.

Start with a modular monolith. Add dedicated search, graph, vector, broker, cache, or workflow infrastructure only through an ADR supported by benchmarks or isolation requirements.

## Consequences

Benefits: simple deployment/failure model, fast deterministic APIs, efficient workers, transactional workflow/outbox consistency, one typed accessible UI, and low early infrastructure cost.

Costs: boundaries require discipline, PostgreSQL-backed jobs need careful retry/retention, and later splits need stable command/event contracts.

## Guardrails

Runtime/dependency versions are pinned in CI. Material rules remain independent of HTTP, React, model providers, and external tools. Stack choice never bypasses tenant, purpose, evidence, authority, or temporal requirements.
