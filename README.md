# ClearSight GRC

> **The AI-native continuous compliance and risk operating system for banks.**  
> Know what applies. Keep proof current. Route the right people. Detect drift. Verify the outcome.

ClearSight helps bank compliance, risk, security, privacy, resilience, audit, legal, business and executive teams operate from one evidence-backed institutional record instead of disconnected registers, questionnaires, email and manually assembled reports.

## Current status

The repository contains canonical product and architecture specifications plus a working application foundation:

- Go API and durable worker processes;
- React/Vite Today, Work and Configure surfaces;
- PostgreSQL 18 schema and pgx-backed repositories;
- deterministic authority resolution and routing-integrity checks;
- maker-checker policies, delegations and segregation rules;
- leased timers, transactional outbox and inbox deduplication;
- Source Registry, source observations and freshness maintenance;
- persisted evidence requests, immutable submissions and one-time magic links;
- bounded capture sessions, invitation/session revocation and artifact manifests;
- role-specific onboarding, premium illustrations and semantic vector icons;
- compliance Signals, drift assessment and readiness;
- OpenAPI, Docker Compose, CI and real PostgreSQL integration tests.

The default build uses in-memory repositories for fast local development. The `postgres` build tag activates PostgreSQL repositories. The local artifact-store adapter is for development and testing only; production object storage and malware scanning are not yet implemented.

## Product model

- **Programs** maintain continuing obligations, controls, evidence, reviews, filings, exceptions and assurance.
- **Matters** handle bounded change, failure, uncertainty, findings, incidents, decisions, external requests and remediation.

```text
Authority Source or Standard
→ Requirement and Applicability
→ Control Objective and Implementation
→ Evidence Contract and Observations
→ Conclusion or Compliance State
→ Matter, Decision, Action, Response and Verification
```

Task completion, a submission, a stored artifact and external execution are not verified outcomes.

## Continuous compliance

ClearSight continuously receives authoritative and operational Signals, detects requirement/evidence/source/control/routing drift, prepares the smallest governed next step, routes the correct actors and verifies material outcomes before closure.

Source freshness, invitation expiry, delegation activation and workflow timers are deterministic. AI may explain or propose handling but is not required for these controls.

## Experience

Working screens use operational bank language: named objects, states, sources, owners, deadlines and actions. Unknown populations remain unknown. Sample data is labelled. Illustrations and icons support orientation but never replace status, evidence or required action.

The application performs assembly before asking a person to act: source-backed prefill, minimum-question requests, secure internal/external capture, review by exception, durable handoff and explicit verification.

## Architecture

```text
React/Vite client
      │
      ▼
Go API ── short, strongly consistent commands and bounded reads
      │
      ├── PostgreSQL authoritative state and durable workflow
      ├── object-store interface and versioned artifact manifests
      ├── transactional outbox and idempotent workers
      ├── rebuildable search, graph, vector, queue and report projections
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

## Performance stance

- common deterministic page: p95 ≤ 1.5 seconds;
- durable command acknowledgement: p95 ≤ 500 ms;
- authority resolution: p95 ≤ 100 ms uncached;
- invitation redemption: p95 ≤ 500 ms;
- request submission: p95 ≤ 750 ms;
- large imports, extraction, scanning, reports and AI run asynchronously;
- material commands are strongly consistent;
- projections expose freshness and are explicitly eventually consistent.

## Start here

1. [`docs/README.md`](docs/README.md)
2. [`docs/product/use-case-catalogue.md`](docs/product/use-case-catalogue.md)
3. [`docs/product/continuous-compliance-and-autonomy.md`](docs/product/continuous-compliance-and-autonomy.md)
4. [`docs/product/authority-routing-and-escalation.md`](docs/product/authority-routing-and-escalation.md)
5. [`docs/architecture/source-evidence-and-secure-capture.md`](docs/architecture/source-evidence-and-secure-capture.md)
6. [`docs/product/respond-and-capture.md`](docs/product/respond-and-capture.md)
7. [`docs/product/enterprise-copy-and-content-design.md`](docs/product/enterprise-copy-and-content-design.md)
8. [`docs/architecture/application-architecture.md`](docs/architecture/application-architecture.md)
9. [`docs/implementation-plan.md`](docs/implementation-plan.md)
10. [`AGENTS.md`](AGENTS.md)

**ClearSight succeeds when governance work is current, correctly routed, minimally demanding, performant and reconstructable.**
