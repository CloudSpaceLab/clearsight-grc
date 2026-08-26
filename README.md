# ClearSight GRC

> **The AI-native continuous compliance and risk operating system for banks.**  
> Know what applies. Keep proof current. Route the right people. Detect change. Confirm the outcome.

ClearSight helps bank compliance, risk, security, privacy, resilience, audit, legal, business and executive teams work from one evidence-backed institutional record instead of disconnected registers, questionnaires, email and manually assembled reports.

## Current status

The repository contains a working application foundation for ongoing Programs and specific issues or changes:

- Go API, durable worker and isolated stateless AI gateway processes;
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
- dedicated, actor-scoped Program records where authorized users can maintain scope, ownership, versioned requirements, applicability, safeguards, evidence expectations, reviewer results and operating status;
- exact linked-issue handling from each Program, including bounded issue reads, new issue creation and direct record navigation;
- non-modal Program setup for channel and obligation monitoring, reusable scored forms, public HTTPS status checks and immutable risk results;
- typed Matters for changes, gaps, findings, requests, exceptions and incidents, with non-modal creation and optional Program linking;
- decisions, actions, response packages and independently checked outcomes;
- reason-bearing Program status and point-in-time reconstruction;
- actor-scoped Today work derived from current journey state;
- four Nigerian-bank **reference journeys** across privacy, regulatory change, protected authority response and verified remediation;
- recoverable, opt-in non-production reference-data installation;
- configurable stakeholder demo mode that remains separate from normal product operation;
- role-specific onboarding, premium illustrations and semantic vector icons;
- compliance Signals, drift assessment and readiness;
- rendered-state and axe accessibility tests enforced in CI;
- mechanically verified main API and isolated AI gateway route/access contracts, Docker Compose, CI and PostgreSQL integration tests;
- OpenAI-compatible Chat/Responses text-and-function transport with OpenAI/Anthropic adapters, truthful SSE, workload authentication, budgets, routing/fallback, circuit state and content-free telemetry.

The default build uses in-memory repositories for local development. The `postgres` build tag activates PostgreSQL repositories. The local artifact-store adapter is for development and testing only; production object storage, malware scanning and OCR are not implemented. Searchable PDFs are extracted automatically by the durable worker through bounded Poppler utilities.

## Product model

- **Programs** maintain continuing obligations, controls, evidence, reviews, filings, exceptions and assurance.
- **Monitoring Checks** collect structured responses or read an exact connected-source version, then calculate risk and coverage from approved deterministic rules.
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

Monitoring results are observations, not approved compliance conclusions. Form collections are created on demand; ClearSight does not automatically create a weekly request. A submitted form response is scored automatically against the exact active form and Monitoring Check versions. Connected status checks are run on demand and preserve the exact source receipt used for the result.

## Governed document imports

The Imports workspace accepts real files and keeps source handling separate from material governance decisions.

Current deterministic extraction supports:

- UTF-8 plain text and Markdown;
- CSV;
- DOCX paragraphs and tables;
- XLSX worksheets and rows.

Every import records the original file metadata, content digest, artifact state, extraction method, analysis limitations, normalized source sections and review proposals. Each proposal contains an exact quote and section, worksheet or row anchor where available.

A reviewer may accept or reject a proposal using optimistic concurrency. Acceptance records that the statement should proceed to governed follow-up; it does **not** automatically create or approve a Requirement, Program, Matter, control, legal interpretation or compliance conclusion.

PDF originals are stored and hashed. Searchable PDFs are extracted automatically into page-numbered sections through a bounded Poppler adapter, and candidate proposals retain exact page anchors. Image-only PDFs are reported explicitly as requiring OCR and never produce fabricated text.

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

The Program record shows the calculated state separately from the authorized operating lifecycle. Current responsibility routes determine who may edit details, replace a requirement, decide applicability, assign a safeguard performer, define evidence, assess results, confirm review or change operating status. Existing records and named responsible people remain visible when the signed-in user cannot act.

Primary screens use plain labels such as **Up to date**, **Evidence incomplete**, **Gap found**, **Change in progress**, **Overdue**, **Under review** and **Setup in progress**. Stable internal codes remain available in APIs, history and specialist detail.

## Protected records and tenant scope

Runtime reads are bound to the verified actor. A client-supplied tenant, principal or legal-entity query that conflicts with verified identity is rejected without revealing whether the requested scope exists.

Program and issue/change identity includes one durable legal entity. Entity filtering occurs before bounded list limits and exact reads, material commands recheck the same verified scope, and Program links cannot cross entities. Creation resolves a verified legal-entity ID or active tenant-bound code to one canonical ID before the row, continuity event and outbox record are written.

Evidence-source choices follow the same boundary: lists are filtered to one exact current legal entity before pagination, and Program evidence checks or Matter outcome checks accept only active sources from that entity. Manual checks remain available when no registered source is selected.

Restricted Matter access is fail-closed:

- malformed or unsupported access metadata is not readable;
- a restricted record requires an explicit principal allow-list;
- legal-entity wildcard values do not bypass the allow-list;
- restricted Matter and linked evidence filtering is applied before PostgreSQL list limits and Matter keyset pagination;
- direct unauthorized reads return not found.

Document-import tenant, legal-entity and reviewer identity are also derived from verified actor context. Cross-tenant detail and review attempts return not found.

Evidence response controls are shown only to the exact current internal recipient. Request creators and other viewers can see the assignment state and valid recovery path, but cannot open a response form that the submission boundary would reject. Terminal assigned requests remain readable without exposing submission actions.

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
cmd/ai-gateway                isolated stateless model transport
cmd/seed-bank-reference       explicit non-production reference installer
internal/authority            routing, simulation and integrity
internal/governance           maker-checker policies and delegations
internal/runtime              timers, outbox and inbox
internal/evidence             sources, requests, sessions and artifacts
internal/documentimport       imports, extraction, proposals and review
internal/continuity           Programs, Matters, access, status and history
internal/bankverticals        connected bank journey projections and installer
internal/autonomy             Signals, drift and readiness
internal/aigateway             canonical model transport, adapters, routing and budgets
internal/workflow             derived actor-facing Task projection and reads
internal/onboarding           guided-adoption state
internal/httpapi              actor-bound HTTP contracts
web                           React application
api/runtime.openapi.json      mechanically verified executable route/access contract
api/ai-gateway.openapi.json   mechanically verified isolated gateway contract
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

For PostgreSQL repositories and local artifact storage:

```bash
make compose-up
# or
go run -tags postgres ./cmd/api
```

For the isolated stateless AI gateway, copy and replace the fail-closed example values, export the referenced provider secrets, then start the separate process:

```bash
cp deploy/ai-gateway.config.example.json ./var/ai-gateway.json
export CLEARSIGHT_AI_GATEWAY_CONFIG_FILE=./var/ai-gateway.json
export OPENAI_API_KEY=...
export ANTHROPIC_API_KEY=...
make run-ai-gateway
```

Generate workload and metrics digests with `printf %s 'a-random-secret' | sha256sum`; never place the plaintext credentials in the JSON file.

The web client runs at `http://localhost:5173`; the API defaults to `http://localhost:8080`.

## Current boundaries

The repository does not yet claim production completion for:

- direct enterprise identity-provider and organization synchronization;
- synchronized restricted groups and database row-level security;
- authorization on every governance/evidence mutation;
- bulk Program setup and controlled configuration changes;
- production object storage, malware scanning, retention and legal hold;
- OCR, password-protected document support and stronger extraction-provider isolation;
- resumable multipart uploads, saved mappings and repeat-import reconciliation;
- authorized conversion of accepted proposals into versioned governed records;
- direct NDPC/CBN ingestion or external authority-channel transmission;
- bank-approved legal configuration and a complete Nigerian regulatory library;
- representative production-scale journey and import benchmarks with retained query plans;
- dependency propagation across shared controls and services;
- governed AI workload/policy lifecycle, source-aware enforcement, durable decision receipts and execution grants beyond the stateless T3 gateway transport.

## Start here

1. [`docs/README.md`](docs/README.md)
2. [`docs/product/use-case-catalogue.md`](docs/product/use-case-catalogue.md)
3. [`docs/product/continuous-compliance-operating-model.md`](docs/product/continuous-compliance-operating-model.md)
4. [`docs/engineering/governed-document-imports.md`](docs/engineering/governed-document-imports.md)
5. [`docs/engineering/demo-deployment.md`](docs/engineering/demo-deployment.md)
6. [`docs/product/nigerian-bank-reference-journeys.md`](docs/product/nigerian-bank-reference-journeys.md)
7. [`docs/architecture/program-and-matter-foundation.md`](docs/architecture/program-and-matter-foundation.md)
8. [`docs/architecture/source-evidence-and-secure-capture.md`](docs/architecture/source-evidence-and-secure-capture.md)
9. [`docs/architecture/application-architecture.md`](docs/architecture/application-architecture.md)
10. [`docs/implementation-plan.md`](docs/implementation-plan.md)
11. [`AGENTS.md`](AGENTS.md)

**ClearSight succeeds when governance work is current, understandable, correctly routed, minimally demanding and reconstructable.**
