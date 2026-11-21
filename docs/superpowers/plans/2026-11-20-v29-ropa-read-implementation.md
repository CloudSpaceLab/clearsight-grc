# 2.9 processing-activity register and record implementation plan

**Status:** Implementable read-only ROPA slice. This does not implement the reference's processing registration, impact assessments or protected privacy cases; they require separate write/authority and protected-data designs. It does not authorize removal of existing routes.

**Goal:** Replace the current Privacy overview, processing register and exact activity/history presentation with the 2.9 composition using existing authorized data.

**Architecture:** Keep `internal/ropa` as the processing-activity owner and its current versioned read contracts. React routes and views consume the existing scoped readers; this slice adds no database table, duplicate privacy store, backend mutation or sample data.

**Tech stack:** React/TypeScript, existing shared controls and `ropaApi.ts`, Go ROPA read API, PostgreSQL and existing evidence tooling.

## Exact existing HTTP contract

The following are existing routes in `internal/httpapi/ropa_routes.go`, not proposed routes. For every read below, `tenant_id` is currently a required query parameter and **must equal the verified actor's tenant**; entity and principal derive from verified identity. A mismatched tenant returns `404 ropa_scope_not_found`. Preserve this parameter and current wire response throughout this slice; removing it requires a separate backwards-compatible API change.

| Route | Query and result |
| --- | --- |
| `GET /api/v1/ropa/dashboard?tenant_id={tenant}` | Returns the `ropa.RegisterSummary` with generation/freshness/coverage and counts; an unknown coverage component stays null. |
| `GET /api/v1/ropa/processing-activities?tenant_id={tenant}&status={NEW|OPEN|CLOSED}&search={text}&include_retired={bool}&cursor={cursor}&limit={n}` | Returns the existing `ropa.ActivityPage`; `web/src/ropaApi.ts` currently normalizes both `Rows`/`NextCursor`/`HasMore` and lower-case equivalents. Preserve both wire forms until the API and all consumers are migrated together. Omit optional filters when empty. |
| `GET /api/v1/ropa/processing-activities/{id}?tenant_id={tenant}` | Returns `{state_label,activity,closure_blockers?}`; the exact activity is scoped server-side. |
| `GET /api/v1/ropa/processing-activities/{id}/history?tenant_id={tenant}&after_version={n}&limit={n}` | Returns `{events,has_more}` after exact-record scope verification. `after_version` is a nonnegative integer. |

Existing writes are `POST /api/v1/ropa/processing-activities`, `POST /api/v1/ropa/processing-activities/{id}` and `POST /api/v1/ropa/processing-activities/{id}/transition`; they are not added or changed by this read-only slice. Do not render an enabled Register, Edit, Complete, Reopen or Review button unless it is connected to a separately verified, authorized workflow. Continue to expose already-supported existing actions if present; trace their real current callers before replacing the view.

## Data and migration

No schema change or data backfill is required for this slice. Read `ropa_processing_activities`, its child categories/recipients/systems/reviews and immutable revisions through existing `internal/ropa` readers; do not copy child values into a second store. Do not update `000092_ropa_register.up.sql`. Verify current scope, keyset pagination, query plan, projection freshness and historical event checks in `internal/ropa/current_postgres.go`, `summaries_postgres.go` and `internal/httpapi/ropa_handlers.go` before adding filters. Keep authorized count and population semantics; unknown is not zero.

Deployment is frontend-only after compatible reader tests: deploy alongside existing API, retain `#ropa` and `#ropa/activity/{id}` links and Back/Forward behavior, and roll back the frontend bundle if source-layout verification fails. No database downgrade, data conversion or destructive command is needed. If a missing record or denied action requires backend changes, split that change into its own reviewed migration/contract rather than inventing a record or silently broadening access.

## Files and test-first tasks

- [ ] In `web/src/ropaApi.test.ts`, extend coverage for a second cursor page, denied/wrong-tenant and unknown coverage. In `web/src/components/RopaRegisterPage.test.tsx` (create only if no equivalent existing suite covers the behavior), assert filter/cursor preservation, selection, deep link and return to the same filtered page. Run `npm test -- --run src/ropaApi.test.ts` from `web`; new assertions should fail before UI implementation where behavior is missing.
- [ ] Inspect 2.9 `privacy.html`, `processing-activity.html`, `processing-version.html` and their referenced `assets/` source. Record hashes and capture the actual source at 1440/1280/1024/768/390. Map heading, local navigation, filters, list, selected record/history and responsive replacement to `web/src/components/RopaRegisterPage.tsx`, `RopaActivityPage.tsx`, `components/ropa/RopaRegisterList.tsx` and the existing shared components. Preserve the current `#ropa` route and programmatic exact-record selection from `web/src/appRouting.ts`.
- [ ] Recompose the existing pages with source-matched section wrappers and feature-scoped CSS. Keep the existing `ropaApi.ts` scope binding and `AbortSignal` handling; do not add a browser-level privacy cache shared across principals or entities. A scope change cancels old requests and clears selection; a late result from the previous scope cannot render.
- [ ] Use actual status `NEW|OPEN|CLOSED`, exact owner and review information. A dashboard metric must carry the source freshness, population and unknown values and link to a query with the same scope/population. Do not invent DPIA/case counts or imply that a processing activity establishes legal compliance.
- [ ] Run from repository root `go test ./internal/ropa ./internal/httpapi`; use `go test -tags "postgres postgresintegration" ./internal/ropa ./internal/httpapi` against an isolated migrated `TEST_DATABASE_URL` for scope/history checks and record skips. From `web`, run `npm test -- --run src/ropaApi.test.ts src/copyQuality.test.ts`, the identified ROPA component tests, `npm run typecheck`, `npm run check:runtime-truth` and `npm run check:ui-contracts`.
- [ ] Render source/runtime pairs at all five widths and both affected themes. Verify keyboard, selected activity inspector, browser history, long record names, empty/partial/error, access revoked after selection and no overflow. Review source and runtime screenshots by a named reviewer and record exact content identity. A green component test alone does not establish full Privacy acceptance.

## Explicit boundary

The 2.9 privacy impact and case routes are not covered by this slice. Before implementing those screens, map the actual protected case owner, exact authorized read/command paths, classification, review and retention; define additions and migrations only after verifying whether existing Matter capabilities meet them. No legal conclusion or historical reviewer may be synthesized.
