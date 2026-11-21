# 2.9 Forms and Programs: verified-contract implementation plan

**Status:** Code-level plan for the current form authoring/capture and Program framework-adoption contracts. The full 2.9 Forms/Program domains additionally need the gaps and acceptance evidence listed below; this file does not claim those gaps are already implemented.

**Goal:** Render the 2.9 Forms, assessment, Frameworks and Program experiences against existing versioned domain owners, preserving scoped access and independent decisions.

**Architecture:** `internal/monitoring` owns form template versions, `internal/evidence` owns distributions/workspaces/submitted responses, and `internal/continuity` owns Programs and framework adoption. Do not copy prototype `S.*` objects or create a second form or Program store. Existing material commands bind the actor and tenant from verified identity.

**Stack:** React/TypeScript/Vite, Go HTTP route registry, PostgreSQL and existing worker/outbox. Source references: `clearsight-v2.9/{forms,form-builder,form-response,assessment-workspace,check-runs,framework-library,framework-adoption,programs,program-workspace,requirement-review}.html` and the corresponding hashed shared assets.

## Forms: current API and data contracts

All paths below are current, not proposed. Runtime route classes are in `api/runtime.openapi.json`; verify actual access in `internal/httpapi/form_distribution_routes.go`, form route registry and service before showing an action.

| Request | Body / response |
| --- | --- |
| `GET /api/v1/forms/templates` | Authorized, bounded form library. Each item wraps `template`, optional `active_version`/`active_status`, `authority_available`, and `operations`; if authority cannot be checked, do not render write actions as enabled. |
| `GET /api/v1/forms/templates/{id}` | One library item with operations. |
| `GET /api/v1/forms/templates/{id}/revisions/{version}` | Exact positive form version. |
| `POST /api/v1/forms/templates` | `createLibraryFormRequest` from `internal/httpapi/forms_handlers.go`: `code`, `name`, `purpose`, optional Program/origin/owner/team/use/tags/jurisdiction/industry/sensitivity/scoring/review date, plus `presentation`, `sections`, `fields`. Result `201` form. This struct and `internal/formcontract` are the canonical complete field schemas; do not recreate a narrower inferred DTO. |
| `POST /api/v1/forms/templates/{id}/revisions` | `{ "expected_version": <int>, "form": <createLibraryFormRequest> }`; returns `201` revision. |
| `POST /api/v1/forms/templates/{id}/transition` | `{ "expected_version": <int>, "to": <monitoring.LifecycleStatus> }`; returns `200` form. Transition and maker/checker behavior remain server-owned. |
| `POST /api/v1/forms/distributions` | `createFormDistributionRequest` in `internal/httpapi/form_distribution_handlers.go`: exact form ID/version, subject type/ID, title/purpose, access policy, deadline, route expiry, reminder policy, recipients and estimated minutes. `dispatchFormDistribution` prepares, issues external routes when necessary, then opens; `201` bundle only after those steps. External issued routes are no-store and must never be logged or previewed. |
| `GET /api/v1/forms/distributions/{id}` and `/responses` | Exact authorized bundle and bounded response reads. |
| `POST /api/v1/forms/distributions/{id}/amend`, `/supersede`, `/lock`, `/reopen`, `/revoke` | Existing handlers and DTOs in `form_distribution_handlers.go`; preserve version, scope and current authority. The response and semantics must be taken from those handlers, not fabricated from the prototype. |
| `GET /api/v1/evidence/session/workspace` | Capability-bound/no-store; returns `{session,request,workspace,recovery_context}`. The token is the session's bearer capability, not an internal actor ID. |
| `PATCH /api/v1/evidence/session/workspace` | `{ "expected_version": <int>, "presentation_mode": <mode>, "edits": [{ "field_id": <id>, "value": <formcontract.AnswerValue>, "base_sequence": <int> }] }`; `200` workspace view or `409` `{current_version,changed_fields}`. |
| `POST /api/v1/evidence/session/workspace/submissions` | `{ "expected_version": <int>, "attestation_field_ids": [<field-id>] }`; `201` `{workspace,revision,submission}`; conflict `409`. |
| `GET /api/v1/forms/responses/{revision_id}/assessment` | Authorized exact response assessment with version, fields, decisions and `may_review`. |
| `POST /api/v1/forms/responses/{revision_id}/assessment` | `{ "expected_version": <int>, "decisions": [{ "field_id": <id>, "outcome_id": <id>, "rationale": <text> }] }`; current reviewer route is evaluated by server. |

`capture_response_workspaces` is unique per distribution, and edits retain recipient, request and field-sequence lineage (migration `000054_form_distributions.up.sql`). Legacy `capture_response_drafts` is scoped to an exact request and session (migration `000036_shared_form_capture_contract.up.sql`); do not assume a new capability session may read an old session's draft. Submitted revision, assessment decision, evidence validation and Program outcome are distinct.

## Forms: file-level tasks

1. **Test first.** Extend `web/src/components/FormsWorkspace.test.tsx`, `web/src/components/capture/CaptureForm.test.tsx`, `web/src/components/capture/CaptureReview.test.tsx` and `web/src/ropaApi.test.ts` only where applicable; do not modify unrelated ROPA tests for a Forms-only change. Assert published version remains visible while a newer draft is edited; exact response revision and source fields open; a `409` workspace conflict preserves local edits and shows server values; revoked session does not recover protected data. Run from `web`:

   ```powershell
   npm test -- --run src/components/FormsWorkspace.test.tsx src/components/capture/CaptureForm.test.tsx src/components/capture/CaptureReview.test.tsx
   ```

   Record the initial failing assertions and their expected behavior.
2. **UI.** Recompose `web/src/components/FormsWorkspace.tsx`, `components/forms/`, `components/capture/`, and related scoped CSS to match the actual reference, including library, authoring, response, assessment and check-run states. Keep `web/src/main.tsx` external capture/session boundary. Reuse `components/ui` and `formcontract` field types; do not construct a new renderer with duplicate scoring logic. Route exact form/revision/distribution IDs through the existing route owner, preserve Back/Forward, and abort stale requests when entity changes.
3. **API clients.** Use `web/src/formsApi.ts`, `formsDistributionApi.ts`, `formAssessmentApi.ts` and `captureApi.ts` with the above existing DTOs. Preserve server-returned operation availability and conflict shape. Do not accept a client-supplied reviewer or principal as authority. Search authorized current evidence before requesting it; distinguish reuse receipt from acceptance.
4. **Backend and migration.** For the verified authoring/capture behavior above, no new schema or API is required. Do not rewrite migrations `000036` or `000054`. Investigate whether 2.9 return/correction is exactly supported by existing `/supersede`, `/reopen`, and immutable response revision semantics. If it is, call those paths and retain predecessor lineage. If it is not, a separate correction-specific contract/migration and reviewer policy are required before enabling that action; do not describe a fake correction as complete.
5. **Security/integration.** Add HTTP and PostgreSQL tests for form revision staleness, cross-entity source denial, current reviewer/owner route, recipient revocation, one-session draft isolation, field-level concurrent edits, exact immutable submitted revision, and failure during prepare/issue/open. Record whether a persisted draft remains after an interrupted dispatch; do not claim an external message was delivered unless a real adapter returns its receipt. Commands and worker effects must meet the existing transaction contract.
6. **Verify.** From root run `go test ./internal/monitoring ./internal/formcontract ./internal/evidence ./internal/httpapi`. With isolated migrated `TEST_DATABASE_URL`, run `go test -tags "postgres postgresintegration" ./internal/monitoring ./internal/evidence ./internal/httpapi` and report skips. From `web`, run the focused tests above plus `npm test -- --run src/copyQuality.test.ts`, `npm run typecheck`, `npm run check:runtime-truth` and `npm run check:ui-contracts`. Perform five-width, keyboard and source-to-runtime review; inspect revoked/expired/partial/error states. Record exact revision and results.
7. **Rollout.** Deploy compatible UI against unchanged endpoints. Preserve old routes and previously issued capability links. Roll back the frontend bundle if the new composition fails; do not revoke sessions or delete response history as a rollback mechanism.

## Frameworks and Programs: current API and data contracts

These operations are existing. `api/runtime.openapi.json` and route registry govern access; DTOs below are confirmed in `internal/httpapi/program_framework_adoption_handlers.go`.

| Request | Body / result |
| --- | --- |
| `GET /api/v1/framework-catalog/editions` | Catalog edition read; edition provenance/license and release digest are catalog facts, not applicability or compliance. |
| `GET /api/v1/programs`, `GET /api/v1/programs/{id}`, `/operations`, `/review-digest`, `/history` | Scoped existing Program reads. Preserve the existing Program/current-calculation version and freshness in display. |
| `POST /api/v1/programs/{id}/standards/propose` | `{ "expected_version": <Program version>, "edition_id": <uuid>, "rationale": <text> }`; returns `200` Program aggregate. Optional `tenant_id` is accepted only if it matches verified identity. |
| `POST /api/v1/programs/{id}/standards/{adoption_id}/approve` or `/reject` | `{ "expected_version": <Program version>, "rationale": <text> }`; returns `200` Program aggregate. Actor is verified identity, and material route requires current Owner/Authorizer policy. |
| `GET /api/v1/programs/{id}/standards/{adoption_id}/candidates`, `/mappings` | Existing source mapping readers. |
| `POST /api/v1/programs/{id}/standards/{adoption_id}/mappings/propose` | `{ "expected_version": <Program version>, "clause_id": <uuid>, "requirement_id": <uuid>, "mapping_kind": "EXACT"\|"PARTIAL"\|"SUPPORTING", "rationale": <text> }`. Optional `tenant_id` must match verified identity. |
| `POST /api/v1/programs/{id}/standards/{adoption_id}/mappings/{mapping_id}/approve` or `/reject` | `{ "expected_version": <Program version>, "rationale": <text> }`. Actor and reviewer come from verified context. Handler and response construction are in `internal/httpapi/program_framework_mapping_handlers.go`. |

Migration `000135_framework_catalog_editions.up.sql` owns catalog editions/clauses; `000137_program_framework_adoptions.up.sql` owns Program adoption and links with scoped foreign keys, one pending/active standard per Program and independent maker/checker constraint; `000138_framework_mapping_review.up.sql` extends mapping review. An edition is a source reference, not a Program requirement or assessment conclusion. Verify every later migration before changing these tables.

## Frameworks and Programs: file-level tasks

1. **Test first.** Extend `web/src/components/FrameworkLibraryWorkspace.test.tsx` and `web/src/components/ProgramRecordWorkspace.test.tsx` for edition selection → Program → adoption propose → independent reviewer; cancelled/delegated/revoked or same-actor review must not produce approval. Add stale Program-version and source-edition-unavailable behavior. Backend tests belong in `internal/httpapi/program_framework_adoption_handlers_test.go` and relevant `internal/frameworkcatalog`/`internal/continuity` tests. Run the focused tests and document failing outcomes before changing behavior.
2. **UI.** Adapt `FrameworkLibraryWorkspace.tsx`, `ProgramsWorkspace.tsx`, `ProgramRecordWorkspace.tsx`, `program-record.css`, `framework-library.css` and `appRouting.ts` to the reference's full composition and responsive states. Preserve exact Program section/item links, `#programs/library`, catalog edition selection, filters and entity context through history. Keep existing data/authority operations and `components/ui` contracts.
3. **Client.** Use `web/src/programFrameworkCandidatesApi.ts`, `programOperationsApi.ts` and `programReviewApi.ts`; send exact `expected_version` and `edition_id`/`rationale` from the current Program snapshot, never an actor/approver ID. On version conflict, reload and show current source and decisions; do not auto-resubmit.
4. **Data.** The existing adoption and mapping tables and Program revision/event history are the target owner; this presentation and existing adoption workflow require **no new table or data migration**. Do not change existing adoption states or create tenant applicability/coverage from catalog membership. For any reference action not representable by the existing route, document the required decision and add a separately reviewed exact contract rather than fabricating an endpoint.
5. **Verify.** Run `go test ./internal/frameworkcatalog ./internal/continuity ./internal/httpapi`; run tagged PostgreSQL tests with isolated migrated `TEST_DATABASE_URL`, inspecting skips. In `web`, run `npm test -- --run src/components/FrameworkLibraryWorkspace.test.tsx src/components/ProgramRecordWorkspace.test.tsx src/copyQuality.test.ts`, `npm run typecheck`, `npm run check:runtime-truth`, `npm run check:ui-contracts`. Verify edition source hash, permission isolation, material version freshness, adoption/mapping history, partial catalog and five-width visual evidence.
6. **Rollout.** Ship frontend/route compatibility first with existing Program and catalog schema unchanged. Keep historical source data and revision links intact. Frontend rollback is safe if no new API semantics are introduced; it must not delete accepted adoptions or mapping history.

## Remaining domain gaps

The two plans above make the **existing** operations and no-schema-change slices concrete. They do not establish that all 2.9 check-run, return/correction, framework mapping or Program review journeys are fully supported. Complete route/action comparison and document any additional API/schema and data migration before claiming full domain acceptance. The prototype does not establish bank policy, live delivery or production permissions.
