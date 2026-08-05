# ClearSight GRC

> **The AI-native continuous compliance and risk operating system for banks.**  
> Know what applies. Keep proof current. Route the right people. Detect drift. Verify the outcome.

ClearSight helps bank compliance, risk, security, privacy, resilience, audit, legal, business and executive teams operate from one evidence-backed institutional record instead of disconnected registers, questionnaires, email and manually assembled reports.

## Current status

The repository contains canonical product and architecture specifications plus a working application foundation:

- Go API and worker processes;
- React/Vite application shell;
- PostgreSQL schema and tagged repositories;
- deterministic authority resolution, simulation and routing-integrity checks;
- focused evidence capture and invitation exchange;
- durable workflow task contracts and optimistic transitions;
- role-specific onboarding state and premium illustration primitives;
- compliance signals, drift assessment and continuous readiness;
- OpenAPI, Docker Compose, CI and performance scaffolding.

The default build uses in-memory repositories for fast local development. A `postgres` build tag activates pgx-backed authority, workflow, onboarding and autonomy repositories.

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

Task completion, evidence upload and external execution are not verified outcomes.

## Continuous compliance

ClearSight continuously:

1. receives authoritative and operational Signals;
2. detects requirement, evidence, source, control, routing and verification drift;
3. identifies affected Programs, Matters and Claims;
4. reuses current evidence and approved precedent;
5. prepares the smallest governed next step;
6. resolves performer, owner, reviewer, challenger, authorizer and escalation;
7. executes only low-impact actions permitted by policy;
8. verifies the outcome and refreshes readiness.

See [`docs/product/continuous-compliance-and-autonomy.md`](docs/product/continuous-compliance-and-autonomy.md).

## Experience

ClearSight performs assembly before asking a person to act:

- source-backed prefill;
- minimum-question requests;
- one dominant next action per actor and workflow state;
- versioned responsibility and authority routing;
- secure internal and external capture;
- grounded AI recommendations with deterministic fallback;
- review by exception with visible denominators;
- durable save, resume and handoff;
- verification before closure.

Premium illustrations support empty states, first-run guidance and education without replacing status, evidence or action. Role-specific guides are skippable, resumable and tied to a first meaningful task.

See [`docs/product/illustration-and-guided-experience.md`](docs/product/illustration-and-guided-experience.md).

## Architecture

```text
React/Vite client
      │
      ▼
Go API ── short, strongly consistent commands and bounded reads
      │
      ├── PostgreSQL authoritative state and durable workflow
      ├── versioned object storage for evidence/source artifacts
      ├── transactional outbox and idempotent workers
      ├── rebuildable search, graph, vector, queue and report projections
      └── governed AI and integration gateways
```

The application begins as a coherent modular monolith with separate API and worker processes. Components split only after measured scale, isolation, residency or availability requirements justify the operational cost.

## Repository layout

```text
cmd/api                 API composition
cmd/worker              durable-worker process
internal/authority      routing, simulation and integrity
internal/autonomy       signals, drift and readiness
internal/workflow       tasks and state transitions
internal/onboarding     guided-adoption state
internal/capture        focused requests and invitations
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

For PostgreSQL repositories:

```bash
make compose-up
# or
go run -tags postgres ./cmd/api
```

The web client runs at `http://localhost:5173`; the API defaults to `http://localhost:8080`.

## Performance stance

- common deterministic page: p95 ≤ 1.5 seconds;
- durable command acknowledgement: p95 ≤ 500 ms;
- authority resolution: p95 ≤ 100 ms uncached and ≤ 25 ms cached;
- invitation redemption: p95 ≤ 500 ms;
- large imports, AI, extraction, reports and verification run asynchronously;
- material commands are strongly consistent;
- projections expose freshness and are explicitly eventually consistent.

## Start here

1. [`docs/README.md`](docs/README.md)
2. [`docs/product/use-case-catalogue.md`](docs/product/use-case-catalogue.md)
3. [`docs/product/continuous-compliance-and-autonomy.md`](docs/product/continuous-compliance-and-autonomy.md)
4. [`docs/product/authority-routing-and-escalation.md`](docs/product/authority-routing-and-escalation.md)
5. [`docs/product/respond-and-capture.md`](docs/product/respond-and-capture.md)
6. [`docs/product/illustration-and-guided-experience.md`](docs/product/illustration-and-guided-experience.md)
7. [`docs/architecture/application-architecture.md`](docs/architecture/application-architecture.md)
8. [`docs/implementation-plan.md`](docs/implementation-plan.md)
9. [`AGENTS.md`](AGENTS.md)

**ClearSight succeeds when governance work becomes continuously prepared, correctly routed, minimally demanding, performant and defensible years later.**
