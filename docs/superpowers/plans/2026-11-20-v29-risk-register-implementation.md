# 2.9 Risk register and record: existing-contract implementation plan

**Status:** Plan for the existing register, exact record and risk context edit. Full control testing, RCSA and branch KRI journeys are separate and not claimed complete.

**Goal:** Replace the current risk register and record with the 2.9 layout, preserving the canonical risk owner and existing scoped queries.

**Architecture:** `internal/risk` owns versioned risk identity, current context, assessments, appetite and links; catalog/Program owners retain control implementation identity. The UI sends existing typed commands and never calculates an approved rating from a task or evidence upload.

**Stack:** React/TypeScript, Go, PostgreSQL, existing route/authority registry and worker.

## Existing exact API

These operations are registered by `internal/httpapi/risk_routes.go`. Tenant, legal entity and actor are obtained from verified identity by `risk_handlers.go`; list scope is checked by `resolveOrganizationScopeSelection`.

| Method/path | Contract |
| --- | --- |
| `GET /api/v1/risks?status=&category=&owner_principal_id=&search=&appetite_position=&organization_scope_id=&cursor=&limit=` | Returns `risk.Page` `{items,next_cursor?,organization_scope_id?}`. `limit` is validated by `riskLimit`; query scope is authorized before the service query. |
| `GET /api/v1/risks/{id}` | Returns the scoped aggregate plus authorized detail from `riskAggregateWithDetails`; not-found is not a cross-entity discovery channel. |
| `POST /api/v1/risks` | `risk.CreateInput`: `{organization_scope_id?,code,name,category?,statement,cause?,event?,impact,scope?}`. Handler overwrites owner and actor with verified principal. Returns `201` risk. |
| `POST /api/v1/risks/{id}` | `risk.UpdateInput`: `{expected_version,name,category?,statement,cause?,event?,impact,scope?,status}`. Returns `200` risk. Preserve the existing server behavior for owner and links; do not assume a UI actor field can reassign ownership. |
| `POST /api/v1/risks/{id}/assessments` | `risk.AssessmentInput` with `expected_risk_version`, `kind`, `method_code`, `method_version`, `dimensions`, `assumptions?`, `evidence_references?`, `confidence?`, `appetite_statement_id?`, `appetite_position`, `appetite_rationale?`. This is reviewer-authorized; a context edit must not invoke it. |
| `POST /api/v1/risks/{id}/controls` | `{expected_risk_version,catalog_link_id}`; links a catalog implementation reference, not a control test result. |
| `POST /api/v1/risks/{id}/indicators` | `risk.LinkIndicatorInput` in `internal/risk/model.go`; exact monitoring check/version is verified by service. |

`Risk` statuses are `DRAFT`, `ACTIVE`, `RETIRED`. An assessment includes risk version, method, evidence references and appetite position; historical records are retained. `risk.UpdateInput` does not itself specify a user-visible change-reason field, and current verified handler does not establish the 2.9 requested reason/owner-edit behavior. Do not add an enabled reason/owner control until an exact authority/data extension is designed and tested.

## Schema and migration

For list/detail and existing context update, use `risks`, `risk_revisions`, `risk_assessments`, `risk_appetite_statements`, `risk_events`, `risk_control_links` and `risk_indicator_links` from migrations `000099`–`000101` and later changes. No new table, migration or data conversion is required for this slice. Do not mutate immutable prior assessments or fabricate a changed rating. Keep prior URLs and the existing response format.

If 2.9 requires a different owner reassignment or a mandatory recorded change reason, inspect `internal/httpapi/risk_command_lifecycle.go`, `internal/risk/service.go` and `postgres.go` and propose a versioned server command and additive event schema with current authority. That is a separate contract/migration, not an optional browser-only field. Roll back this slice by restoring the prior frontend bundle; never drop risk history.

## Tasks

1. Extend `web/src/components/risks/RisksWorkspace.test.tsx` and add focused `RiskRecord`/editor tests for authorized list filters, exact detail link and return, stale version, wrong scope, two risk versions and preserved active rating/appetite/control links. Run `npm test -- --run src/components/risks/RisksWorkspace.test.tsx` from `web` and record newly failing assertions before implementation.
2. Capture 2.9 `risks.html`, `risk-workspace.html`, `risk-editor.html` with hashes for the referenced CSS/JS at five approved widths. Recompose `components/risks/RiskRegister.tsx`, `RiskRecord.tsx`, `RisksWorkspace.tsx`, `risk.css` and routing using shared controls. Maintain server-backed filtering/paging, first useful record, exact ID and current entity; cancel stale in-flight loads on scope change.
3. Add typed existing update client to `web/src/riskApi.ts` using the exact `risk.UpdateInput` fields above and current `expected_version`. On `409`, show current source and require explicit retry; never auto-overwrite. Retain `actor_id`, owner and authority as server-bound facts. Unsupported 2.9 owner/reason changes must not be represented as an enabled success action.
4. Extend `internal/risk/service_test.go` and `internal/httpapi/risk_handlers_test.go` with same-tenant cross-entity denial, forged owner, stale write, current-authority denial, unchanged assessment/appetite after context edit and preservation of unrelated control links. Add PostgreSQL concurrent update tests in `internal/risk/postgres_integration_test.go`.
5. Run `go test ./internal/risk ./internal/httpapi`; with isolated migrated `TEST_DATABASE_URL`, run `go test -tags "postgres postgresintegration" ./internal/risk ./internal/httpapi` and inspect skips. From `web`, run `npm test -- --run src/components/risks/RisksWorkspace.test.tsx src/copyQuality.test.ts`, the added risk editor tests, `npm run typecheck`, `npm run check:runtime-truth` and `npm run check:ui-contracts`. Retain reference/runtime images, accessibility checks and source/authorized-result evidence at 1440/1280/1024/768/390.

**Unresolved full-domain work:** Version-bound control tests, per-risk RCSA decision, owner-reassignment policy, reason persistence and KRI return lifecycle need their own exact APIs, migrations and tests. This slice does not establish those requirements or whole-domain acceptance. No tests above have been executed.
