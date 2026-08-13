# ClearSight GRC

> **The AI-native continuous compliance and risk operating system for banks.**
> Know what applies. Keep proof current. Route the right people. Detect change. Confirm the outcome.

ClearSight helps bank compliance, risk, security, privacy, resilience, audit, legal, business and executive teams work from one evidence-backed institutional record instead of disconnected registers, questionnaires, email and manually assembled reports.

## Current status

The repository contains a working application foundation for ongoing Programs and specific issues or changes:

- Go API and durable worker processes;
- React/Vite **Today, Programs, Work, Imports, Explore and Configure** surfaces;
- PostgreSQL 18 schema and pgx-backed repositories;
- verified actor context with tenant/query-scope conflict rejection;
- deterministic authority resolution and routing-integrity checks;
- maker-checker policies, delegations and segregation rules;
- leased timers, transactional outbox and inbox deduplication;
- Source Registry, source observations and freshness maintenance;
- persisted evidence requests, immutable submissions and one-time magic links;
- bounded capture sessions, invitation/session revocation and artifact manifests;
- governed document imports with immutable original metadata, SHA-256 lineage and actor-bound review;
- deterministic TXT, Markdown, CSV, DOCX and XLSX extraction with source-location anchors;
- source-anchored analysis proposals that require explicit human acceptance or rejection;
- ongoing Programs with requirements, applicability decisions, controls and evidence checks;
- typed Matters for changes, gaps, findings, requests, exceptions and incidents;
- decisions, actions, response packages and independently checked outcomes;
- reason-bearing Program status and point-in-time reconstruction;
- actor-scoped Today work derived from current journey state;
- four Nigerian-bank **reference journeys** across privacy, regulatory change, protected authority response and verified remediation;
- recoverable, opt-in non-production reference-data installation;
- configurable stakeholder demo mode that remains separate from normal product operation;
- role-specific onboarding, premium illustrations and semantic vector icons;
- compliance Signals, drift assessment and readiness;
- rendered-state and axe accessibility tests enforced in CI;
- mechanically verified runtime API/access contract, Docker Compose, CI and PostgreSQL integration tests.

The default build uses in-memory repositories for local development. The `postgres` build tag activates PostgreSQL repositories. The local artifact-store adapter is for development and testing only; production object storage, malware scanning, PDF extraction and OCR are not implemented.

## Product model

- **Programs** maintain continuing obligations, controls, evidence, reviews, filings, exceptions and assurance.
- **Matters** are the precise internal records for a specific change, gap, finding, request, exception or incident. General screens call them **issues and changes**.
- **Imports** preserve original source material, extracted sections and source-anchored proposals for governed human review.
- **Explore journeys** connect the records into understandable, end-to-end operating paths and launch the exact next Program, issue or evidence request when stakeholder demo mode is enabled.

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

Task completion, a submission, a stored artifact, an extracted statement, an accepted analysis proposal and external execution are not verified outcomes.

## Governed document imports

The Imports workspace accepts real files and keeps source handling separate from material governance decisions.

Current deterministic extraction supports:

- UTF-8 plain text and Markdown;
- CSV;
- DOCX paragraphs and tables;
- XLSX worksheets and rows.

Every import records the original file metadata, content digest, artifact state, extraction method, analysis limitations, normalized source sections and review proposals. Each proposal contains an exact quote and section, worksheet or row anchor where available.

A reviewer may accept or reject a proposal using optimistic concurrency. Acceptance records that the statement should proceed to governed follow-up; it does **not** automatically create or approve a Requirement, Program, Matter, control, legal interpretation or compliance conclusion.

PDF originals are stored and hashed, but this build reports PDF extraction as unsupported until an approved PDF/OCR adapter is implemented.

## Demo and operational modes

`CLEARSIGHT_DEMO_MODE` controls the stakeholder reference experience independently from real operational data and document imports.

When enabled:

- Nigerian-bank reference journeys may be exposed through Explore;
- non-production sample capture and workflow records may be available;
- the UI clearly treats those records as reference data.

When disabled:

- reference fixtures are not installed by the in-memory composition;
- the bank-journey route and Explore navigation are absent;
- sample capture actions and tasks are absent;
- Today, Programs, Work, Imports and Configure remain available;
- normal document-import APIs and persisted imports continue to work.

Production refuses to start with demo mode enabled.

`CLEARSIGHT_DOCUMENT_IMPORT_ALLOW_UNSCANNED_ANALYSIS` separately controls whether deterministic local analysis may run before an approved scanning service marks an artifact available. It defaults to `true` in development and `false` in production.

## Program status

Program status is calculated from approved requirements, applicability, control coverage, implementation, evidence, open issues, source health and deadlines. It is not a manually selected red/amber/green value.

Primary screens use plain labels such as **Up to date**, **Evidence incomplete**, **Gap found**, **Change in progress**, **Overdue**, **Under review** and **Setup in progress**. Stable internal codes remain available in APIs, history and specialist detail.

## Protected records and tenant scope

Runtime reads are bound to the verified actor. A client-supplied tenant, principal or legal-entity query that conflicts with verified identity is rejected without revealing whether the requested scope exists.

Restricted Matter access is fail-closed:

- malformed or unsupported access metadata is not readable;
- a restricted record requires an explicit principal allow-list;
- legal-entity wildcard values do not bypass the allow-list;
- restricted Matter and linked evidence filtering is applied before PostgreSQL list limits and Matter keyset pagination;
- direct unauthorized reads return not found.

Document-import tenant, legal-entity and reviewer identity are also derived from verified actor context. Cross-tenant detail and review attempts return not found.

This HTTP/repository boundary does not replace synchronized enterprise identity groups, database row-level security or authorization on every future mutation endpoint.

## Reference journeys

The Nigerian-bank journeys are labelled **Reference data**. They prove composition and product interaction, not legal completeness or bank compliance.

Explore shows:

- evidence-backed state and reason;
- accountable function and deadline;
- complete and incomplete stages;
- authoritative and supporting sources;
- the exact next Program, issue or evidence request;
- completed records for review without adding them to Today.

The installer is explicit and recoverable. It refuses `CLEARSIGHT_ENV=production` and reconciles partial installations by stable reference identity.

```bash
go run -tags postgres ./cmd/seed-bank-reference \
  -tenant <tenant-uuid-or-slug> \
  -legal-entity <legal-entity-uuid> \
  -actor <installer-principal-uuid> \
  -owner <owner-principal-uuid> \
  -reviewer <independent-reviewer-uuid> \
  -signatory <signatory-principal-uuid>
```

## Experience and copy

Working screens use ordinary bank-operating language before specialist terminology. They identify the object, current state, reason, owner, deadline and next valid action.

- “Does this apply?” precedes “applicability determination.”
- “Evidence incomplete” precedes “evidence insufficiency.”
- “Outcome check” precedes “verification contract.”
- “Issues and changes” is used on general work screens; “Matter” remains the precise API and audit-history name.

Unknown populations remain unknown. Sample data is labelled. Illustrations and icons support orientation but never replace status, evidence or required action.

## Architecture

```text
React/Vite client
      │
      ▼
Go API ── verified actor scope, short commands and bounded reads
      │
      ├── PostgreSQL authoritative state and durable workflow
      ├── append-only Program and Matter event history
      ├── object-store interface and versioned artifact manifests
      ├── document extraction and source-anchored proposal review
      ├── transactional outbox and idempotent workers
      ├── rebuildable search, graph, queue and report projections
      └── governed AI and integration gateways
```

The application begins as a modular monolith with separate API and worker processes. Components split only after measured scale, isolation, residency or availability requirements justify the operational cost.

## Repository layout

```text
cmd/api                       API composition
cmd/worker                    durable worker
cmd/seed-bank-reference       explicit non-production reference installer
internal/authority            routing, simulation and integrity
internal/governance           maker-checker policies and delegations
internal/runtime              timers, outbox and inbox
internal/evidence             sources, requests, sessions and artifacts
internal/documentimport       imports, extraction, proposals and review
internal/continuity           Programs, Matters, access, status and history
internal/bankverticals        connected bank journey projections and installer
internal/autonomy             Signals, drift and readiness
internal/workflow             derived actor-facing Task projection and reads
internal/onboarding           guided-adoption state
internal/httpapi              actor-bound HTTP contracts
web                           React application
api/runtime.openapi.json      mechanically verified executable route/access contract
api/bank-journeys.openapi.yaml focused journey schema and examples
api/document-imports.openapi.yaml focused import and review contract
api/README.md                 API contract ownership rules
docs                          canonical specifications
```

## Run

```bash
cp .env.example .env
make check
make run-api
```

The API, worker and reference-data installer load `.env` for local development.
Variables already provided by the operating system, Docker Compose or CI take
precedence. In demo mode, `CLEARSIGHT_DEMO_TENANT_ID`,
`CLEARSIGHT_DEMO_PRINCIPAL_ID` and `CLEARSIGHT_DEMO_LEGAL_ENTITY_ID` provide
the default verified scope. A demo account with its own `TenantID` uses that
tenant instead of `CLEARSIGHT_DEMO_TENANT_ID`.

For PostgreSQL repositories and local artifact storage:

```bash
make compose-up
# or
go run -tags postgres ./cmd/api
```

The web client runs at `http://localhost:5173`; the API defaults to `http://localhost:8080`.

## Current boundaries

The repository does not yet claim production completion for:

- direct enterprise identity-provider and organization synchronization;
- synchronized restricted groups and database row-level security;
- authorization on every governance/evidence mutation;
- bulk Program setup and controlled configuration changes;
- production object storage, malware scanning, retention and legal hold;
- PDF extraction, OCR, password-protected documents and extraction-provider isolation;
- resumable multipart uploads, saved mappings and repeat-import reconciliation;
- authorized conversion of accepted proposals into versioned governed records;
- direct NDPC/CBN ingestion or external authority-channel transmission;
- bank-approved legal configuration and a complete Nigerian regulatory library;
- representative production-scale journey and import benchmarks with retained query plans;
- dependency propagation across shared controls and services.

## Start here

1. [`docs/README.md`](docs/README.md)
2. [`docs/product/use-case-catalogue.md`](docs/product/use-case-catalogue.md)
3. [`docs/product/continuous-compliance-operating-model.md`](docs/product/continuous-compliance-operating-model.md)
4. [`docs/engineering/governed-document-imports.md`](docs/engineering/governed-document-imports.md)
5. [`docs/product/nigerian-bank-reference-journeys.md`](docs/product/nigerian-bank-reference-journeys.md)
6. [`docs/architecture/program-and-matter-foundation.md`](docs/architecture/program-and-matter-foundation.md)
7. [`docs/architecture/source-evidence-and-secure-capture.md`](docs/architecture/source-evidence-and-secure-capture.md)
8. [`docs/architecture/application-architecture.md`](docs/architecture/application-architecture.md)
9. [`docs/implementation-plan.md`](docs/implementation-plan.md)
10. [`AGENTS.md`](AGENTS.md)

**ClearSight succeeds when governance work is current, understandable, correctly routed, minimally demanding and reconstructable.**
