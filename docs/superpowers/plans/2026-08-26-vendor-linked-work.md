# Vendor-linked work implementation plan

> Execute in the feature worktree with test-driven development. Keep Capture Requests, Programs, Matters, Actions, evidence, authority, and third-party records canonical in their existing domains.

**Goal:** Make vendor due diligence and vendor-completed Program or Matter work secure, flexible, easy to manage, and ready to merge to `main`.

**Architecture:** The third-party domain owns relationship associations and vendor-work orchestration. It reuses immutable form revisions and Capture Requests for collection, artifacts, invitations, sessions, drafts, and submissions. Programs and Matters retain internal ownership and closure semantics. New commands use verified identity, current authority, scoped relational integrity, optimistic versions, events, outbox, and recoverable delivery state.

**Stack:** Go 1.25, PostgreSQL/pgx, React 19, TypeScript 7, Vitest, Testing Library, Vite, Playwright-based repository review scripts.

---

## Task 1: Remove invitation secrets and make replacement fail safe

**Files:**

- Modify: `web/src/main.tsx`
- Modify: `web/src/components/ExternalCaptureApp.tsx`
- Modify: `web/src/components/ExternalCaptureApp.test.tsx`
- Modify: `web/src/staticExternalCapture.ts`
- Modify: `internal/evidence/repository.go`
- Modify: `internal/evidence/memory.go`
- Modify: `internal/evidence/postgres.go`
- Modify: `internal/thirdparty/assessment_reissue.go`
- Modify: `internal/thirdparty/assessment_reissue_test.go`

1. Add failing browser-unit tests proving the full token is removed from `location.search` and history before the external workspace renders, refresh resumes through the returned session, and storage keys contain no token material.
2. Add failing service tests proving replacement revokes the old invitation and sessions before the workflow reports a prepared replacement; injected issuance failure must leave the old capability unusable and return a truthful retry state.
3. Implement one-time bootstrap URL cleanup with `history.replaceState` and session-identity storage.
4. Add a durable capability-revocation operation and use it before replacement issuance. Preserve idempotency for retries.
5. Run targeted frontend and Go tests and commit.

## Task 2: Correct vendor assessment review decisions and handoffs

**Files:**

- Modify: `web/src/components/VendorDueDiligence.tsx`
- Modify: `web/src/components/VendorDueDiligence.test.tsx`
- Modify: `web/src/components/VendorsWorkspace.tsx`
- Modify: `web/src/components/VendorsWorkspace.test.tsx`
- Modify: `web/src/App.tsx`
- Modify: `web/src/appRouting.ts`
- Modify: `web/src/appRouting.test.ts`
- Modify: `internal/httpapi/vendor_assessment_review_handlers_test.go`
- Modify or add: exact scoped artifact-open handler and tests under `internal/httpapi` and `internal/evidence`

1. Add failing tests for a blank initial conclusion, disabled completion until an explicit conclusion and rationale, and no score-driven selection.
2. Add failing tests that display source-prefilled values beside vendor corrections, distinguish missing required fields from conditional omissions, and show critical responses, validation limitations, and freshness.
3. Add an exact assessment/request-scoped artifact-open route. Require authorization and block quarantined, missing, rejected, or unavailable artifacts.
4. Add `Open document` before validation/rejection and preserve review state on return.
5. Add exact `Open finding` navigation to the canonical Matter and restore vendor selection when returning.
6. Make relationship accessible names include vendor and service.
7. Run affected tests, copy-quality tests, and commit.

## Task 3: Save Wizard drafts before section navigation

**Files:**

- Modify: `web/src/components/capture/CaptureForm.tsx`
- Modify: `web/src/components/capture/CaptureForm.test.tsx`
- Modify: `web/src/components/CapturePanel.tsx`
- Modify: `web/src/components/CapturePanel.test.tsx`
- Modify: `web/src/components/ExternalCaptureApp.test.tsx`

1. Add failing tests with delayed, conflicting, and unavailable saves.
2. Change the navigation contract so `Continue` awaits a successful save.
3. Keep the current section and entered values after failure; expose concise `Try again` recovery.
4. Preserve Classic rendering and background autosave behavior.
5. Run targeted tests and commit.

## Task 4: Harden generic Evidence Request creation and material outcomes

**Files:**

- Modify: `internal/httpapi/route_registry.go`
- Modify: `internal/httpapi/evidence_handlers.go`
- Modify: `internal/httpapi/evidence_handlers_test.go`
- Modify: `internal/httpapi/command_outcome.go`
- Modify: `internal/evidence/service.go`
- Modify: `internal/evidence/service_test.go`
- Modify: `internal/evidence/recipient_subject_postgres.go`
- Modify: `internal/evidence/recipient_postgres_integration_test.go`
- Add: typed subject/origin policy files and tests under `internal/evidence`

1. Add failing tests proving arbitrary or missing Program/vendor subjects, cross-legal-entity subjects, and reserved third-party origins are rejected.
2. Replace permissive non-Matter checks with typed exact subject resolvers for Program, Matter, and Vendor Relationship.
3. Restrict reserved origins to trusted orchestration calls and make direct request creation a material route with verified identity and current authority.
4. Extend post-commit outcome/version probing to Vendor Relationship and Third-Party Assessment commands.
5. Add failure-injection tests proving a committed command never becomes a false 5xx.
6. Run Go and PostgreSQL-tagged tests and commit.

## Task 5: Add canonical vendor relationship links

**Files:**

- Add: `migrations/000041_third_party_relationship_links.up.sql`
- Add: `migrations/000041_third_party_relationship_links.down.sql`
- Add/modify: link model, service, memory repository, PostgreSQL repository, and tests under `internal/thirdparty`
- Modify: `internal/thirdparty/repository.go`
- Modify: `internal/thirdparty/assessment_postgres.go`
- Modify: assessment provisioning/deficiency services and tests
- Modify: schema ownership and schema tests

1. Write failing schema and service tests for scoped Program/Matter links, bank-defined bounded purpose codes, duplicate idempotency, expected versions, ending rather than deleting, and point-in-time history.
2. Add relational Program and Matter link tables with composite tenant/legal-entity integrity and keyset indexes.
3. Migrate assessment review/deficiency Matter associations to reference the general relationship-Matter link without dual canonical rows.
4. Implement link/list/end services with immutable events, outbox, and projection jobs in the same transaction.
5. Add restricted-Matter and cross-scope tests.
6. Run memory, PostgreSQL-tagged, migration, and integration tests and commit.

## Task 6: Add vendor-work orchestration

**Files:**

- Add: `migrations/000042_third_party_work_requests.up.sql`
- Add: `migrations/000042_third_party_work_requests.down.sql`
- Add: work-request model, repository, service, consumer, job, and tests under `internal/thirdparty`
- Modify: `cmd/api/services.go`, `cmd/api/services_memory.go`, `cmd/api/services_postgres.go`
- Modify: `cmd/worker/services.go`, `cmd/worker/services_memory.go`, `cmd/worker/services_postgres.go`
- Modify: `internal/httpapi/server.go`

1. Add failing state-machine tests for preparation, send, submission reaction, review, changes requested, acceptance, cancellation, and retry.
2. Add work-request and immutable capture-link tables; reference the canonical relationship target link.
3. Reuse the form/capture contract and invitation delivery. Do not accept actor, reviewer, approver, or tenant identity from JSON.
4. Store authoritative rows, event, outbox, and jobs transactionally. Model external delivery as durable partial success with retry.
5. Re-evaluate current authority in workers through verified service identity and automation policy; fail closed on missing or changed routes.
6. Revoke capabilities during cancellation and keep submission history.
7. Add bounded list/read projections by relationship and exact Program/Matter target.
8. Run focused and tagged Go tests and commit.

## Task 7: Expose material vendor-link and vendor-work APIs

**Files:**

- Add: `internal/httpapi/third_party_link_handlers.go`
- Add: `internal/httpapi/third_party_link_handlers_test.go`
- Add: `internal/httpapi/vendor_work_handlers.go`
- Add: `internal/httpapi/vendor_work_handlers_test.go`
- Modify: `internal/httpapi/route_registry.go`
- Modify: `api/runtime.openapi.json`

1. Add failing handler tests for verified identity, authority, legal-entity scope, conflicts, restricted target not-found behavior, and truthful partial delivery outcomes.
2. Add material endpoints for Program/Matter link, list, end, prepare/send, request changes, review, accept, cancel, and retry.
3. Bind actors only from verified request context and overwrite prohibited body fields.
4. Add bounded cursor inputs and stable response shapes.
5. Update the runtime contract and route-registry expectations.
6. Run handler and route tests and commit.

## Task 8: Build the reusable vendor-work UI

**Files:**

- Add: `web/src/vendorWorkTypes.ts`
- Add: `web/src/vendorWorkApi.ts`
- Add: `web/src/vendorWorkApi.test.ts`
- Add: `web/src/components/VendorWorkPanel.tsx`
- Add: `web/src/components/VendorWorkPanel.test.tsx`
- Add/modify: focused styles under `web/src`
- Modify: `web/src/staticDemo.ts`
- Modify: `web/src/staticDemo.test.ts`

1. Add failing component tests for existing relationship search, inline relationship creation, target linking, template selection, Automatic/Classic/Wizard rendering, typed fields, prefill degradation, preview, send, retry, review, clarification, acceptance, and cancellation.
2. Implement a focused reusable panel whose target is supplied by Program, Matter, or Vendor context.
3. Use bounded server search; do not preload all vendor relationships.
4. Preserve user inputs across duplicate warning, service failure, conflict, or form-source degradation.
5. Keep one dominant action per state and concise copy grounded in stored state.
6. Add explicitly labelled static fixtures for rendered acceptance states.
7. Run component, API, static-demo, and copy-quality tests and commit.

## Task 9: Integrate Vendors, Programs, and issues or changes

**Files:**

- Modify: `web/src/components/VendorsWorkspace.tsx`
- Modify: `web/src/components/VendorsWorkspace.test.tsx`
- Modify: `web/src/components/ProgramsWorkspace.tsx`
- Add or modify: `web/src/components/ProgramsWorkspace.test.tsx`
- Modify: `web/src/components/MattersWorkspace.tsx`
- Add or modify: `web/src/components/MattersWorkspace.test.tsx`
- Modify: `web/src/App.tsx`
- Modify: `web/src/appRouting.ts`
- Modify: workspace CSS and `DESIGN.md`

1. Add failing tests for Vendors `Overview`, `Due diligence`, and `Requests`; related Program/Matter deep links; and request state/deadline/next action.
2. Add failing Program/Matter tests for `Related vendors`, `Link vendor`, and `Request vendor work` with capability-gated unavailable states.
3. Integrate the shared panel without obscuring each workspace's dominant current action.
4. Restore exact target, tab, and scroll state across cross-workspace navigation and browser Back.
5. Verify duplicate vendor candidates do not silently merge or block a justified new relationship.
6. Run affected tests, typecheck, and copy quality; commit.

## Task 10: Complete rendered evidence, documentation, issue, and merge

**Files:**

- Modify: `web/scripts/capture-ui-evidence.mjs`
- Modify: `web/scripts/review-ui-accessibility.mjs`
- Modify: `web/scripts/review-ui-flow-manifest.mjs`
- Modify: `docs/quality/rendered-ui-evidence.md`
- Modify: `docs/quality/program-matter-acceptance-tests.md`
- Modify: `docs/product/respond-and-capture.md`
- Modify: `docs/product/use-case-catalogue.md`
- Modify: `docs/architecture/application-architecture.md`
- Modify: `docs/architecture/durable-schema-ownership.md`
- Modify: `docs/implementation-plan.md`
- Modify: `README.md` and `DESIGN.md` where required

1. Add executable desktop, 390 px, 320 px, and 200% reflow scenarios for bank and external vendor workflows, including terminal and recovery states.
2. Add axe and keyboard/focus checks for Vendors, Program/Matter request entry, Classic/Wizard capture, document review, and completed work.
3. Render all materially affected states, inspect them, fix the highest-impact defect, and render again.
4. Run full verification from a clean exact-HEAD checkout:
   - `go test ./... -count=1`
   - `go test -tags postgres ./... -count=1`
   - `go test -tags "postgres postgresintegration" ./... -count=1`
   - `go vet ./...`
   - `npm test -- --run` in `web`
   - `npm run typecheck` in `web`
   - `npm run build` in `web`
   - `npm run review:ui` in `web`
5. Request an independent code/security review and resolve every blocking finding.
6. Update issue #80 with delivered scope, remaining lifecycle work, and verification evidence; do not create a duplicate issue.
7. Confirm `main` and the feature branch state, merge locally to `main`, rerun smoke tests on the merge commit, and report the merge hash.
