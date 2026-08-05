# ClearSight GRC

> **The AI-native continuous compliance and risk operating system for banks.**  
> Know what applies. Keep proof current. Route the right people. Detect change. Confirm the outcome.

ClearSight helps bank compliance, risk, security, privacy, resilience, audit, legal, business and executive teams work from one evidence-backed institutional record instead of disconnected registers, questionnaires, email and manually assembled reports.

## Current status

The repository now contains a working application foundation for ongoing Programs and specific issues or changes:

- Go API and durable worker processes;
- React/Vite Today, Programs, Work and Configure surfaces;
- PostgreSQL 18 schema and pgx-backed repositories;
- deterministic authority resolution and routing-integrity checks;
- maker-checker policies, delegations and segregation rules;
- leased timers, transactional outbox and inbox deduplication;
- Source Registry, source observations and freshness maintenance;
- persisted evidence requests, immutable submissions and one-time magic links;
- bounded capture sessions, invitation/session revocation and artifact manifests;
- ongoing Programs with requirements, applicability decisions, controls and evidence checks;
- typed Matters for changes, gaps, findings, requests, exceptions and incidents;
- decisions, actions, response packages and outcome checks;
- reason-bearing Program status and point-in-time reconstruction;
- role-specific onboarding, premium illustrations and semantic vector icons;
- compliance Signals, drift assessment and readiness;
- OpenAPI, Docker Compose, CI and real PostgreSQL integration tests.

The default build uses in-memory repositories for fast local development. The `postgres` build tag activates PostgreSQL repositories. The local artifact-store adapter is for development and testing only; production object storage and malware scanning are not yet implemented.

## Product model

- **Programs** maintain continuing obligations, controls, evidence, reviews, filings, exceptions and assurance.
- **Matters** are the precise internal records for a specific change, gap, finding, request, exception or incident. General user screens call them **issues and changes** where that is clearer.

```text
Authority Source or Standard
→ Requirement
→ Does this apply?
→ Control objective and implementation
→ Evidence check and current result
→ Program status
→ Issue or change
→ Decision and action
→ Outcome check
→ Closure
```

Task completion, a submission, a stored artifact and external execution are not verified outcomes.

## Program status

Program status is calculated from approved requirements, applicability, control coverage, implementation, evidence, open issues, source health and deadlines. It is not a manually selected red/amber/green value.

Primary screens use plain labels such as:

- **Up to date**;
- **Evidence incomplete**;
- **Gap found**;
- **Change in progress**;
- **Overdue**;
- **Under review**;
- **Setup in progress**.

Stable internal codes remain available in APIs, history and specialist detail.

## Matter lifecycle

A Matter moves through only the stages appropriate to its type:

```text
Draft
→ Initial review
→ Reviewing impact
→ Decision needed / Work in progress / Preparing response
→ Confirming outcome
→ Closed
```

Closure is type-aware. Open actions, missing approval, unacknowledged external responses or failed outcome checks prevent closure. A failed outcome check can reopen work, request a decision, create a follow-up Matter or keep closure blocked according to its contract.

## Experience and copy

Working screens use ordinary bank-operating language before specialist terminology. They name the object, current state, reason, owner, deadline and next valid action.

- “Does this apply?” is used before “applicability determination.”
- “Evidence incomplete” is used before “evidence insufficiency.”
- “Outcome check” is used before “verification contract.”
- “Issues and changes” is used on general work screens; “Matter” remains the precise record name in APIs and audit history.

Unknown populations remain unknown. Sample data is labelled. Illustrations and icons support orientation but never replace status, evidence or required action.

## Architecture

```text
React/Vite client
      │
      ▼
Go API ── short, strongly consistent commands and bounded reads
      │
      ├── PostgreSQL authoritative state and durable workflow
      ├── append-only Program and Matter event history
      ├── object-store interface and versioned artifact manifests
      ├── transactional outbox and idempotent workers
      ├── rebuildable search, graph, queue and report projections
      └── governed AI and integration gateways
```

The application begins as a modular monolith with separate API and worker processes. Components split only after measured scale, isolation, residency or availability requirements justify the operational cost.

## Repository layout

```text
cmd/api                 API composition
cmd/worker              durable worker
internal/authority      routing, simulation and integrity
internal/governance     maker-checker policies and delegations
internal/runtime        timers, outbox and inbox
internal/evidence       sources, requests, sessions and artifacts
internal/continuity     Programs, Matters, status, closure and history
internal/autonomy       Signals, drift and readiness
internal/workflow       tasks and transitions
internal/onboarding     guided-adoption state
internal/capture        legacy focused-capture demo
internal/httpapi        HTTP contracts
migrations              PostgreSQL schema
web                     React application
api/openapi.yaml        HTTP contract
docs                    canonical specifications
```

## Run

```bash
cp .env.example .env
make check
make run-api
```

For PostgreSQL repositories and local artifact storage:

```bash
make compose-up
# or
go run -tags postgres ./cmd/api
```

The web client runs at `http://localhost:5173`; the API defaults to `http://localhost:8080`.

## Current boundaries

The Program and Matter foundation does not yet claim production completion for:

- authenticated actor binding and automatic authority checks on every material command;
- projection-first list reads for high-cardinality tenants;
- bulk Program setup and configuration approval;
- production object storage, scanning, retention and legal hold;
- dependency propagation across shared controls and services;
- complete vertical workflows for NDPA, regulatory change, authority requests and imported findings.

## Start here

1. [`docs/README.md`](docs/README.md)
2. [`docs/product/use-case-catalogue.md`](docs/product/use-case-catalogue.md)
3. [`docs/product/continuous-compliance-operating-model.md`](docs/product/continuous-compliance-operating-model.md)
4. [`docs/architecture/program-and-matter-foundation.md`](docs/architecture/program-and-matter-foundation.md)
5. [`docs/product/plain-language-content-standard.md`](docs/product/plain-language-content-standard.md)
6. [`docs/architecture/source-evidence-and-secure-capture.md`](docs/architecture/source-evidence-and-secure-capture.md)
7. [`docs/architecture/application-architecture.md`](docs/architecture/application-architecture.md)
8. [`docs/implementation-plan.md`](docs/implementation-plan.md)
9. [`AGENTS.md`](AGENTS.md)

**ClearSight succeeds when governance work is current, understandable, correctly routed, minimally demanding and reconstructable.**
