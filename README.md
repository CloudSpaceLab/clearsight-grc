# ClearSight GRC

> **The AI-native continuous compliance and risk operating system for banks.**  
> Know what applies. Keep proof current. Route the right people. Handle what changed. Verify the outcome.

ClearSight helps bank compliance, risk, security, privacy, resilience, audit, legal, business, and executive teams operate from one evidence-backed institutional record instead of disconnected registers, questionnaires, email, and manually assembled reports.

## Current status

The repository now contains:

- the canonical product, UX, authority, invitation, evidence, AI, data, and performance specifications;
- a runnable **Go API and worker foundation**;
- a **React/Vite/Tailwind** application shell demonstrating Today, authority explanation, and focused capture;
- a PostgreSQL 18 foundation schema;
- OpenAPI, Docker Compose, CI, and performance-smoke scaffolding.

The executable slice is a foundation, not a production claim. It intentionally uses in-memory demo services while PostgreSQL repositories, enterprise identity, durable workflows, and real bank sources are implemented through the phased plan.

## Product model

Users operate two primary objects:

- **Programs** maintain continuing obligations, controls, evidence, reviews, filings, exceptions, and assurance.
- **Matters** handle bounded change, failure, uncertainty, findings, incidents, decisions, external requests, and remediation.

Material state remains traceable through:

```text
Authority Source or Standard
→ Requirement and Applicability
→ Control Objective and Implementation
→ Evidence Contract and Observations
→ Conclusion or Compliance State
→ Matter, Decision, Action, Response, and Verification
```

A completed task, uploaded file, submitted response, or implemented change is not automatically a verified outcome.

## Core experience

ClearSight performs assembly before asking a person to act:

- prefill from approved institutional sources;
- ask only unresolved facts;
- show one dominant next action per actor and workflow state;
- resolve performer, owner, reviewer, challenger, authorizer, signatory, and escalation separately;
- provide secure request-scoped internal and external capture;
- prepare grounded AI recommendations without making material decisions;
- review by exception with visible denominators and full-review triggers;
- preserve save, resume, handoff, and point-in-time history;
- require verification before closure.

Routine authorized work should normally take less than five minutes of active effort. Complex work should reach a safe, saved, correctly routed next state within five minutes.

## Initial executable slice

The scaffold demonstrates three cross-cutting capabilities:

1. **Today** — a small role-specific attention brief.
2. **Authority resolution** — deterministic, explainable routing from scope, responsibility, materiality, and policy.
3. **Respond and Capture** — a focused request, optimistic version check, validation, submission receipt, and one-time invitation exchange.

These are deliberate foundations for all later Program and Matter journeys.

## Architecture

```text
React/Vite web client
        │
        ▼
Go HTTP API ── authoritative commands and deterministic reads
        │
        ├── PostgreSQL 18 relational state and durable workflow
        ├── versioned object storage for source/evidence artifacts
        ├── transactional outbox and idempotent workers
        ├── rebuildable search, graph, vector, queue and reporting projections
        └── governed AI and integration gateways
```

The system begins as a **coherent modular monolith**. It does not require a message broker, graph database, vector database, or microservice estate to deliver the first vertical slices. Components split only after measured isolation or scale requirements justify the operational cost.

See:

- [`docs/architecture/application-architecture.md`](docs/architecture/application-architecture.md)
- [`docs/architecture/system-data-and-performance.md`](docs/architecture/system-data-and-performance.md)
- [`docs/architecture/data-model-and-storage.md`](docs/architecture/data-model-and-storage.md)

## Repository layout

```text
cmd/api                 Go HTTP service
cmd/worker              durable-worker process scaffold
internal/authority      responsibility and authority resolution
internal/capture        focused requests and invitation exchange
internal/today          role-specific attention brief
internal/httpapi        transport and middleware
migrations              PostgreSQL authoritative schema
api/openapi.yaml        executable API contract
web                     React/Vite application shell
tests/performance       k6 smoke and SLO checks
docs                    canonical product and engineering specifications
```

## Run locally

Requirements:

- Go 1.23+ for the dependency-free local backend scaffold; CI and deployment pin a supported Go 1.26 patch.
- Node.js 24 LTS.
- Docker for PostgreSQL and the complete local stack.

```bash
cp .env.example .env
make check
make run-api
```

In another terminal:

```bash
make web-install
make run-web
```

Or:

```bash
make compose-up
```

Open the web client at `http://localhost:5173`. The API listens on `http://localhost:8080`.

## Performance stance

- Common deterministic pages: **p95 ≤ 1.5 s**.
- Common durable commands: **p95 ≤ 500 ms** acknowledgement.
- Authority resolution: **p95 ≤ 100 ms** uncached and **≤ 25 ms** cached.
- Invitation redemption: **p95 ≤ 500 ms**.
- Large imports, AI, extraction, package generation, and verification run asynchronously and resumably.
- Material commands are strongly consistent; projections are explicitly eventually consistent.
- Every high-volume table and queue requires a cardinality, partition, index, retention, and load-test plan.

## Product boundaries

ClearSight is not a generic form builder, generic workflow platform, SIEM, transaction-monitoring system, document repository, core banking platform, or autonomous compliance officer. It is the governed Program, Matter, evidence, authority, decision, response, verification, and assurance layer across specialist systems.

## Start here

1. [`docs/README.md`](docs/README.md)
2. [`docs/product/use-case-catalogue.md`](docs/product/use-case-catalogue.md)
3. [`docs/product/authority-routing-and-escalation.md`](docs/product/authority-routing-and-escalation.md)
4. [`docs/product/respond-and-capture.md`](docs/product/respond-and-capture.md)
5. [`docs/architecture/application-architecture.md`](docs/architecture/application-architecture.md)
6. [`docs/implementation-plan.md`](docs/implementation-plan.md)
7. [`AGENTS.md`](AGENTS.md)

**ClearSight succeeds when high-accountability bank governance work becomes continuously prepared, correctly routed, minimally demanding, performant, and defensible years later.**
