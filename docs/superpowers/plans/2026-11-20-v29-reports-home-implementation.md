# 2.9 Reports and Home: existing-contract implementation plan

**Status:** Implementable plan for current authorized report definition/run and Home metric views. Published board packs and 2.9 loss reporting are explicitly outside this no-schema-change slice until their independent contract and data policy are specified.

**Goal:** Render the 2.9 Home, report library, exploration and existing report-run workflow from authorized current metrics and governed report owners.

**Architecture:** `internal/metricview` and `internal/oversight` own scoped metrics/source membership; `internal/reporting` owns definitions, immutable revisions and report runs. React composes them without a second dashboard ledger, assumed completeness or client-only approval.

**Stack:** React/TypeScript, Go, PostgreSQL, existing report worker and artifact store.

## Existing HTTP contract

All routes below already exist in `internal/httpapi/reporting_routes.go` and the runtime OpenAPI. Verified actor, tenant and single legal entity come from server context; scope and actor fields supplied for wire compatibility are overwritten.

| Operation | Existing request/result |
| --- | --- |
| `GET /api/v1/reports/filter-fields` | `200 {fields:[...]}`. |
| `GET /api/v1/reports/owners?dataset={dataset}&limit={1..200}` | `200 {items:[...]}`; default limit 200. |
| `GET /api/v1/reports/definitions?include_retired={bool}` | `200 {items:[...]}`. |
| `GET /api/v1/reports/definitions/{id}` and `/history` | Exact scoped definition, or `{items:[revision...]}`. |
| `POST /api/v1/reports/definitions` | `reporting.ProposeInput`: `{code,name,description,dataset,scope_kind,scope_ref?,format,filter?,effective_from?,effective_until?}`. Returns `201` definition. Actor, maker and scope are rebound by handler. |
| `POST /api/v1/reports/definitions/{id}/{submit|review|activate|reject|retire}` | `reporting.DefinitionTransitionInput`: `{expected_version,checksum_seen,note?,effective_from?}`. `id` is path-bound. Returns `200` current definition; the route registry distinguishes proposer, reviewer and authorizer. |
| `GET /api/v1/reports/runs?definition_id={id}&cursor={cursor}&limit={1..100}` | `200 {items,next_cursor}`; default limit 50. |
| `POST /api/v1/reports/runs` | `{definition_id,expected_definition_version,parameters?:{start_date?,end_date?,owner_principal_id?}}` → `201` run receipt. Dates are `YYYY-MM-DD`; server validates ordering and rebinds requested-by identity. |
| `GET /api/v1/reports/runs/{id}` | `200` exact scoped `ReportRun`, including status, definition/version/checksum, parameters, source boundary and artifact digests when available. |
| `GET /api/v1/reports/runs/{id}/download` | Streams authorized ready artifact with private/no-store, correct content type and ETag based on data SHA-256. |

Home metric reads include `GET /api/v1/metrics/home`, `/home/{metric_id}/members`, `/home/{metric_id}/trend`, `GET /api/v1/metrics/group/posture` and the corresponding domain, loss-period and risk/assurance matrix paths in the [runtime API inventory](2026-11-20-runtime-api-inventory.md). Their exact parameter/result shapes are owned by `internal/httpapi/*metric*handlers.go` and existing TypeScript clients, not inferred from prototype cards.

`internal/reporting/model.go` defines `ReportRun.SourceBoundary` as `captured_at`, `projection_version`, `source_high_water`, `population`, `population_complete`; run has a separate `expires_at`. A ready export run is not automatically a permanently retained, independently approved or published board pack.

## Database and rollout

No new table or migration for displaying existing Home metrics and generating the existing report runs. Continue using the current `report_definitions`, immutable revisions and `report_runs` plus later migrations; do not alter historical migration `000093_report_builder.up.sql` or copy source data into a frontend store. Verify all later reporting migrations and the authorized source-membership boundaries for each displayed metric.

Ship the compatible UI against the existing endpoints; preserve `#reports` and Home deep links and report download authorization. Frontend rollback restores prior presentation and does not delete reports or immutable history. Do not label an expiring run a published board pack. The reference's frozen board pack with narrative, decision and a loss annex needs a separate approved contract for source membership, retention, artifact digest and financial basis before creating schema or enabling publication.

## Ordered implementation tasks

1. Add failing assertions to `web/src/components/reports/ReportsWorkspace.test.tsx`, `ReportsExplore.test.tsx`, `ReportsLibrary.test.tsx` and `components/oversight/OversightWorkspace.test.tsx` for scope/period preservation, same-population metric drill, stale/unknown coverage, no mixed-currency sum, authorized report transition, and a run whose artifact has not completed. The UI must not show an absent source count as zero or compliance.
2. Capture source `index.html`, `reports.html`, `insight-detail.html`, `report-builder.html`, `report-pack.html` and shared assets with hashes at 1440/1280/1024/768/390. Adapt `components/reports/`, `components/oversight/`, `web/src/reportingApi.ts`, `metricApi.ts`, routing and feature CSS to the 2.9 anatomy. Use shared controls; preserve keyboard/touch, focus, actual permission state, cursor and history.
3. Keep the existing definition proposal and transition DTOs above; pass current version and displayed checksum, never approver identity. On a `409` stale definition, reload and require an explicit re-review. Do not allow a current posture card to imply that a historical range changed it. Current scope and date filters must reach supported authorized queries and remain attached to drill links.
4. Add `internal/reporting` and `internal/metricview` tests for exact source/membership parity under tenant/entity/scope changes, definition maker/checker conflict, stale checksum, report-run source boundary and artifact hash. Existing loss period values use the currently implemented ledger basis; until reviewed loss lifecycle migration is complete, do not relabel legacy figures as independently reviewed.
5. From root run `go test ./internal/reporting ./internal/metricview ./internal/oversight ./internal/httpapi`. With isolated migrated `TEST_DATABASE_URL`, run `go test -tags "postgres postgresintegration" ./internal/reporting ./internal/metricview ./internal/oversight ./internal/httpapi` and record skips. From `web`, run `npm test -- --run src/components/reports/ReportsWorkspace.test.tsx src/components/reports/ReportsExplore.test.tsx src/components/reports/ReportsLibrary.test.tsx src/components/oversight/OversightWorkspace.test.tsx src/copyQuality.test.ts`, `npm run typecheck`, `npm run check:runtime-truth` and `npm run check:ui-contracts`.
6. Render source/runtime pairs at five widths and affected themes; verify first useful content, populated/empty/partial/denied/stale/error states, keyboard and accessible labels. Reconcile each visible total to exact authorized source members and retained as-of/coverage; test report download bytes and permission revocation.

## Remaining full-domain requirements

2.9 board-pack creation/publication, durable narrative/management decisions, frozen charts and loss annex need exact API, storage, retention, migration and policy decisions; current run receipts alone do not establish them. New 2.9 loss cohorts require the separate loss reviewed-ledger design and a bank-approved legacy reporting basis. This document does not assert the full Reports/Home domain is code-ready or complete; no prescribed tests have been run.
