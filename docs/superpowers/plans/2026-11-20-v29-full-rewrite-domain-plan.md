# ClearSight 2.9 full UI rewrite and backend simplification: implementation plan

> **Execution:** Implement in the ordered, independently verifiable slices below using `executing-plans`; use the TDD, systematic-debugging and verification skills where applicable. This is a program plan: exact SQL and command names for genuinely new workflows must be settled in the specified slice contracts before code is written, not invented in advance.

**Goal:** Replace every current customer-facing experience with the 2.9 foundation or a consistent completion, while simplifying workflow and data ownership and retaining safe, working bank operations.

**Architecture:** Keep the React/TypeScript application and Go modular monolith, PostgreSQL, worker and versioned artifact abstraction. A single shared shell and route registry dispatch to domain-owned views; domain services own authoritative facts, Work derives executable tasks from those facts, and reports consume scoped source-backed data. Extend existing contracts only when they cannot represent the requested workflow; do not import prototype-local state into production.

**Tech stack:** React 19, TypeScript, Vite, react-aria-components, Vitest, Go, pgx/PostgreSQL, existing outbox/worker and browser evidence tooling.

**Companion design:** [Rewrite and debloat design](../../design/2026-11-20-v29-rewrite-and-debloat-plan.md); [friction and data architecture](../../design/2026-11-20-v29-simplification-and-data-architecture.md). These are proposals; implementation must reconcile the explicit user instruction with current project authority and preserve security, historical data and customer-copy rules.

**Contract investigations:** [Runtime API inventory](2026-11-20-runtime-api-inventory.md), [loss proposal](2026-11-20-v29-operational-loss-contract.md), [RCSA investigation](2026-11-20-v29-rcsa-contract.md), [Inbox recipient-state proposal](2026-11-20-v29-notification-inbox-implementation.md), [ROPA read-only](2026-11-20-v29-ropa-read-implementation.md), [Forms and Programs](2026-11-20-v29-forms-programs-implementation.md), [Evidence and documents](2026-11-20-v29-evidence-document-implementation.md), [Reports and Home](2026-11-20-v29-reports-home-implementation.md), [Risk register](2026-11-20-v29-risk-register-implementation.md), and [Vendor relationships](2026-11-20-v29-vendor-implementation.md). These documents identify remaining prerequisites and are not evidence that all domains are code-ready.

---

## Execution contract

- Check `git status --short` at the start of every slice. The current checkout contains extensive user-owned changes, including the untracked reference and both companion documents. Preserve all of them. Before changing a shared file, record its baseline bytes and merge deliberately; never `reset`, `clean`, overwrite, commit or move unrelated work. Do not create a worktree from HEAD alone and assume it includes the dirty baseline.
- Treat all 67 paths in `clearsight-v2.9/docs/route-manifest.json` as reference experiences. Entry HTML delegates to shared assets; bind evidence to the manifest, exact entry, relevant CSS/JS and their SHA-256 hashes, not only the small launcher HTML. Keep source bytes immutable. The release integrity verifier reported 199 files; its own tests use a mocked DOM, and historical screenshots do not prove 2.9 layout.
- The route sets below are reference entry names, not assertions that current URLs or backend endpoints have the same names. Preserve existing deep links and current capabilities, including ones absent from the reference. Resolve old URLs to the same authorized record and context. Use the verified `api/runtime.openapi.json` and `internal/httpapi/route_registry.go` as current route inventory, then inspect the service, SQL and current tests; absence from an example list is not proof an endpoint is absent.
- User-selected 2.9 composition governs the replacement. Keep complete page anatomy, selected states and mobile replacement behavior. Derive a compact brief for screens missing from the foundation. Apply the user's global copy rule even when prototype text differs. Real state, permissions and actions replace sample values; an unbuilt action gets an honest unavailable state until its domain slice supplies it.
- A material command uses verified identity, tenant/entity scope, current authority and expected versions; its authoritative writes, event/outbox and required jobs share a transaction. No client actor fields confer authority. Work, notifications, lists and metrics are read models, not a second domain truth. Preserve independent decisions, evidence and outcome verification.
- Each slice produces: (1) source-to-runtime mapping and friction decision, (2) focused failing behavior/security test before implementation, (3) backend and UI changes, (4) old code removal when its last consumer is proven gone, (5) paired rendered evidence at 1440/1280/1024/768/390 in affected themes, (6) source/data/authority acceptance, and (7) an explicit release/rollback record. Use labelled sample fixtures only in the evidence build.
- Existing dirty evidence scripts and tests must not be casually replaced to obtain a green result. Migrate an October-specific guard to a 2.9 equivalent only after the new guard proves the same security, data and layout properties. Historical evidence remains historical; the old 0/19 register is not automatically a 2.9 score.

## Baseline and commands

Run these from the repository root unless noted. Run only read-only checks during baseline; `make format` can modify files and `docker compose down -v` destroys volumes, so neither is a baseline command.

```powershell
git status --short
python clearsight-v2.9/verify_reference.py
node --version
go version
go test ./internal/oploss ./internal/rcsa ./internal/risk
Push-Location web
npm run typecheck
npm run check:runtime-truth
npm run check:ui-contracts
npm test -- --run src/appRouting.test.ts src/copyQuality.test.ts
Pop-Location
```

Expected: the integrity command reports 199 verified files on the untouched package; other commands exit zero, or their pre-existing failures are retained with exact output and environment details before implementation. A passing in-memory `go test` does not substitute for PostgreSQL, browser, accessibility, performance or authority validation. Use `go test -tags postgres ./...` and `go test -tags "postgres postgresintegration" ./internal/integration` only with the documented test database; record skipped tests and prerequisites rather than calling them passes. `web` requires the Node version in its package.json.

## Dependency order and progress register

| ID | Slice | Depends on | Shippable result |
| --- | --- | --- | --- |
| 0 | Baseline, route ledger and authority | None | Preserved baseline, all reference/current routes and actions inventoried, dated source hashes, baseline renders and metrics. |
| 1 | Shell, navigation, tokens, shared controls | 0 | 2.9 shell; existing deep links and capture entry still work. |
| 2 | Work, approvals and notifications foundation | 1 | One actor-scoped Work entry and source-resolved Inbox, with existing source types intact. |
| 3 | Evidence, documents and capture | 1; source adapters from 2 | Exact evidence intake, reuse and review journeys without duplicated artifacts. |
| 4 | Forms and assessments | 1, 3 | Form authoring, distribution, response and review with immutable versions. |
| 5 | Frameworks and Programs | 1, 3, 4 | Catalog, adoption and full Program operating journey. |
| 6 | Risks, controls and RCSA | 2–5 | Version-bound risk/control/RCSA assessment and review. |
| 7 | Indicators and branch returns | 2, 4, 6 | Versioned indicator cycles, captured returns and review; no new form engine. |
| 8 | Losses and import | 2, 3, 6 | Reviewed financial lifecycle and reconciled original-currency ledger. |
| 9 | Vendors | 2–5 | Relationship, assessment and reviewed lifecycle using existing evidence and forms. |
| 10 | Privacy | 2–4 | ROPA, impact and case experiences with protected reads and source-linked work. |
| 11 | Reports and Home | 5–10 as needed for each measure | Source-reconciled live perspectives and durable published packs; Home shares existing work and metrics. |
| 12 | Settings, access, integrations, automation | 1; domain prerequisites per action | Governed configuration in 2.9 layouts, no client-only permissions. |
| 13 | Remaining surfaces, decommission and cutover | 1–12 | Every existing route and reference journey accounted for; obsolete code retired and operational release proved. |

Foundation slices may be developed incrementally alongside domain slices, but a domain may not expose a material action before its dependency is available. Loss implementation starts with contract and data analysis early; the complete financial slice depends on source-aware Work/evidence. Reports/Home can migrate early using existing verified measures, with new financial and domain views gated on their actual owners. Update this register with exact revision and status; no whole-product acceptance is inferred from a partial slice.

## Shared verification and acceptance

For each slice, replace `<focused test>` in the command with the existing or newly created exact test paths listed under that slice. From `web`, use `npm test -- --run <focused test>` and expect zero failures; run `npm run typecheck`, `npm run check:runtime-truth`, `npm run check:ui-contracts`, and `npm test -- --run src/copyQuality.test.ts` for each customer-facing tranche. Run the listed `go test` package commands from root and the PostgreSQL integration suite for every changed persistent path. A command is not evidence until its exit code and output are recorded on the final working tree.

For each reference entry, capture same-screen source and runtime, heading/actions, selected navigation, filters, first useful content, cards/list/inspector and keyboard/touch behavior. Exercise loading, empty, partial, stale, denied, conflict and recovery. Validate no horizontal overflow at all five widths and both themes as affected. Source comparisons require actual rendered 2.9 pages, not V2.7 screenshots; ensure synthetic reference data is not treated as bank data. Retain filenames, hashes, exact working-tree identity, tool versions, named reviewer and limitations. Accessibility tests include keyboard/focus and axe, plus manual screen-reader review of materially changed complex flows.

For each material command, prove server identity binding, cross-tenant/entity and protected-source denial, independent authority, conflict/delegation/revocation, version conflict, transaction rollback, idempotent retry, source provenance and audit/outbox atomicity. Count/read tests prove authorized filtering before LIMIT, explicit freshness and population. Reconcile results to source and report, not merely to screen labels. Record the query plan and load test for any new high-volume list or projection.

## Slice 0 — baseline, catalog and decision authority

**Existing inputs:** `AGENTS.md`, `DESIGN.md`, `docs/README.md`, `docs/implementation-plan.md`, `docs/quality/acceptance-tests.md`, `clearsight-v2.9/README.md`, `MANIFEST.json`, `docs/route-manifest.json`, `web/src/appRouting.ts`, `web/src/components/configure/configurationSections.ts`, `api/runtime.openapi.json`.

- [ ] Save a content-addressed baseline of tracked and untracked affected files outside the runtime tree, including diffs and the current 2.9 manifest; verify its hashes and restoration without modifying the original workspace. Record that `910598ff1` alone is not the dirty baseline.
- [ ] Create `docs/design/v29-route-ledger.md` with one row for every manifest entry and one row for every existing route/action without a matching entry. Columns: reference entry and asset hashes; current route/component; desired task; actor; source owner; API read/command; known gap; responsive states; tests; decision; owner; status. Distinguish verified facts from assumptions.
- [ ] Generate an API inventory from the OpenAPI paths, but validate actual command policy, service and repository ownership before marking any workflow reused. Include protected external capture, Imports, Matters, Explore, Insights, People, group scope, AI governance and nested Settings even if absent from the reference manifest.
- [ ] Capture the existing app and reference at five widths. Measure initial/lazy JS and CSS, request counts, p50/p95 useful-content latency, relevant query plans and projection lag using a documented representative fixture. Record test duration and task effort for loss, evidence, RCSA, vendor and reporting. Do not invent numerical improvement targets before measuring.
- [ ] Update the project authority and delivery docs in one reviewable change to recognize 2.9 and preserve original October archive/evidence as history. Update `web/scripts/approved-source-integrity.nodecheck.mjs` and its callers only with a demonstrably equivalent 2.9 integrity guard. Keep the security/copy/permissions gates intact.
- [ ] Gate: 67 manifest entries are represented; every existing destination has a proposed retained, redirected or intentionally retired destination; reference integrity and baseline commands have recorded results. No customer feature is removed here.

## Slice 1 — shell, routing and design foundation

**Reference:** `index`, `guided-setup`, plus shell shared by all entries. **Current:** `web/src/main.tsx`, `App.tsx`, `appRouting.ts`, `appStyles.ts`, `components/SessionGate.tsx`, `components/ExternalCaptureApp.tsx`, `components/configure/ConfigureWorkspace.tsx`, `components/ui/`, `design-system/` and shell CSS.

- [ ] Add tests to `web/src/appRouting.test.ts` for exact deep-link round trips, encoded IDs, unknown routes, preserved filters, Back/Forward and entity transitions; test the separate invitation entry does not load an internal session or leak tokens. Run `npm test -- --run src/appRouting.test.ts` and confirm the new missing cases fail first.
- [ ] Inventory 2.9 shell anatomy, navigation, breakpoints, theme, focus and shared component variants from the actual reference. Implement one navigation state/typed registry and domain route adapters. Move Configure's separate history synchronization behind the same contract while preserving its nested destinations. Keep verified session and authorized organization scope in the shell; domain selection stays domain-owned.
- [ ] Map prototype styles to existing primitive/semantic/component tokens and accessible `components/ui` contracts; extend those contracts only for shared behaviors. Preserve 44px touch targets, reduced motion and accessible names. Use scoped feature layout for source geometry rather than global component overrides.
- [ ] Keep compatibility aliases for current hashes and notification targets. Prove that an inaccessible record returns a safe denied/unavailable state instead of navigating to another record. Preserve explicit return context across scope change and refresh.
- [ ] Inventory `appStyles.ts` imports and dynamic selectors. Remove a legacy shell/fix file only when import graph, browser states and capture route show no remaining consumer. Record pre/post bundle and CSS bytes.
- [ ] Gate: route tests, `npm run build`, runtime-truth and UI-contract checks pass; same-screen shell evidence covers five widths, both themes, permission-gated menu and touch/keyboard. No production route imports reference seeds or evidence fixtures.

## Slice 2 — Work, approvals and Inbox

**Reference:** `work-approvals`, `approval-inbox`, `approval-review`, `approval-request`, `inbox`, `notification-detail`, `notification-preferences`. **Current:** `web/src/components/AssignedWorkQueue.tsx`, `components/notifications/`, `notificationApi.ts`, `notificationPreferencesApi.ts`, `internal/workflow/`, `internal/authority/`, `internal/governance/`; `GET /api/v1/workflow/tasks` and notification routes.

- [ ] Inventory current Work source kinds and filtering in `internal/workflow/actor_work_page.go`, projectors and SQL; verify whether newer supported sources exist beyond the historical architecture document. Map each 2.9 review to its owning source command and permission; do not assume every approval is a Matter or create a generic mutable approval store.
- [ ] Add focused tests for one source-backed review: the actor sees only an authorized current item, opens the exact source revision, completes or returns it through the source command, and the projection converges after duplicate/reordered events. Denied/revoked actors and stale source versions fail closed. Extend source kinds with a stable identity and scope-aware filtering before pagination.
- [ ] Unify queue query semantics: cursor, tenant/entity/principal, filter, source population and as-of. If a search or count covers loaded rows only, label the actual population or replace it with a supported bounded server query. Preserve nullable deadlines. Add tests for count/search scope changes and pagination.
- [ ] Implement 2.9 Work and approval preparation/review layouts, durable scoped drafts where required, exact source links and return-to-filter state. Resolve eligible routes server-side; only present human selection where policy allows. Show one next action without collapsing reviewer, authorizer and performer roles.
- [ ] Inbox stores recipient delivery/read state separately from source state. Show a notification's exact source and current action only after source authorization. Read, acknowledgement, archive/snooze if implemented, delivery and source resolution remain distinct; missing delivery capability must not imply a message was sent.
- [ ] Verify `go test ./internal/workflow ./internal/authority ./internal/governance ./internal/httpapi` and the focused `AssignedWorkQueue`/Notifications tests. Add PostgreSQL replay/permission tests for each new source kind and visual evidence for source changed, route missing and denied states.

## Slice 3 — evidence, documents and secure capture

**Reference:** `evidence`, `evidence-review`, `documents`, `document-workspace`, `document-upload`. **Current:** `components/EvidenceWorkspace.tsx`, `components/documents/`, `components/capture/`, `DocumentImportWorkspace.tsx`, `evidenceRequestAdminApi.ts`, `documentApi.ts`, `submittedDocumentApi.ts`; `internal/evidence/`, `internal/documentimport/`, `internal/sourceaccess/`.

- [ ] Map reference evidence observations, document versions, reviews and approvals to existing request, capture submission, artifact and assessment ownership. Explicitly distinguish a file upload, inspected artifact, requested evidence, accepted evidence and Program conclusion. Inspect actual artifact/open permission paths before introducing any shared document search.
- [ ] Add tests: an authorized reviewer sees the pinned source version; another tenant or an actor without source authority cannot read title, bytes or metadata; rejected/expired/unscanned sources do not silently satisfy an evidence need; a pending replacement does not supersede prior accepted evidence.
- [ ] Implement scoped evidence search before asking for new material, with source attribution, freshness and field/target eligibility. Reuse existing artifact IDs and collection receipts; no copy of artifact bytes or acceptance flag per consuming screen. Offer targeted refresh and show its exact predecessor.
- [ ] Preserve capability-bound invitation redemption, session revocation, short lifetimes, scanning and safe download. Adapt external capture to the 2.9 visual system without loading internal app data or placing tokens in URL previews, logs or analytics.
- [ ] Complete upload, version, review, rejection, correction and recovery states in the 2.9 document/evidence layouts. If storage or scanning is unavailable, show a nonblocking recovery path and never assert the file is accepted.
- [ ] Run `go test ./internal/evidence ./internal/documentimport ./internal/sourceaccess ./internal/httpapi` and `npm test -- --run src/components/EvidenceWorkspace.test.tsx src/components/DocumentImportWorkspace.test.tsx src/components/documents/DocumentBrowser.test.tsx src/components/capture/CaptureForm.test.tsx`. Exercise PostgreSQL artifact permissions, version concurrency and scanner failure; inspect file access and actual bytes through the authorized path.

## Slice 4 — Forms, assessments and response cycles

**Reference:** `forms`, `form-builder`, `form-response`, `assessment-workspace`, `check-runs`. **Current:** `components/FormsWorkspace.tsx`, `components/forms/`, `components/capture/`, `formsApi.ts`, `formsDistributionApi.ts`, `formAssessmentApi.ts`; `internal/monitoring/`, `internal/formcontract/`, `internal/evidence/`; `internal/httpapi/form_distribution_routes.go`.

- [ ] Map template identity/version, draft/active state, distribution, recipient access, submitted immutable revision, assessment and monitoring check to the existing owners. Build a field/type matrix from reference `assets/forms.js` and the current `formcontract`; retain visibility, scoring, attachments, correction and source provenance.
- [ ] Test that an edit creates a draft without mutating an active form; a submitted response pins its revision; a correction creates a successor; and an old assessment remains historical. Test wrong recipient, expired/revoked invitation, stale version and field-level authorization.
- [ ] Implement library, builder, form response, assessment and check-run page anatomy. Prefill sourced known values, offer only eligible reuse and unresolved questions, and persist scoped drafts without treating them as submissions. Keep approved content and calculated score/coverage distinct.
- [ ] Reuse the same form/response engine for vendor collection, RCSA and branch KRI where semantics match; add typed links and version guards rather than a new engine. Do not copy answers or imply that a source response has a new review decision.
- [ ] Run `go test ./internal/monitoring ./internal/formcontract ./internal/evidence ./internal/httpapi` and `npm test -- --run src/components/FormsWorkspace.test.tsx src/components/forms/FormPreview.test.tsx src/components/capture/CaptureReview.test.tsx`. Verify actual sender/recipient workflows, concurrency, disabled integration and browser interruption/recovery.

## Slice 5 — Frameworks and Programs

**Reference:** `framework-library`, `framework-adoption`, `programs`, `program-workspace`, `requirement-review`. **Current:** `FrameworkLibraryWorkspace.tsx`, `ProgramsWorkspace.tsx`, `ProgramRecordWorkspace.tsx`, `programOperationsApi.ts`, `programReviewApi.ts`, `internal/frameworkcatalog/`, `internal/continuity/`; Program routes in `api/runtime.openapi.json`.

- [ ] Inventory catalog edition/provenance/license/completeness, adoption and Program requirement/control/evidence states. Inspect `internal/frameworkcatalog/adoption_preview.go` and current Program transitions before specifying missing commands. A catalog edition is not evidence of applicability, adoption or compliance.
- [ ] Write tests for authorized edition selection → preview → versioned adoption and independent decision; omitted/stale/unlicensed framework data remains unavailable or unknown. Verify current Program material version, evidence freshness, ownership versus authorizer, linked issue navigation and retained history.
- [ ] Implement 2.9 catalog, adoption, Program collection, record and requirement review compositions. Preserve exact requirement/control navigation and context. Search existing evidence and requirements before asking for new inputs; only show calculated state as current when its assessed version matches.
- [ ] Reuse Program requirements, controls, evidence contracts and linked Matters. Do not duplicate catalog entries as tenant conclusions or make a second Program state calculator. Any backend simplification must trace all consumers of current revisions and audit events.
- [ ] Run `go test ./internal/frameworkcatalog ./internal/continuity ./internal/httpapi` and `npm test -- --run src/components/FrameworkLibraryWorkspace.test.tsx src/components/ProgramRecordWorkspace.test.tsx`. Add PostgreSQL edition/version, cross-entity and adoption-race checks; verify current/unknown/stale rendering and impact on Work.

## Slice 6 — Risks, controls and RCSA

**Reference:** `risks`, `risk-workspace`, `risk-editor`, `controls`, `control-workspace`, `control-editor`, `control-test`, `rcsa`, `rcsa-review`. **Current:** `components/risks/`, `components/rcsa/`, `riskApi.ts`, `rcsaApi.ts`, `controlCatalogApi.ts`, `internal/risk/`, `internal/rcsa/`, `internal/controlcatalog/`, `internal/continuity/`.

- [ ] Trace control definition → catalog link → Program implementation → evidence → control test → RCSA cycle snapshot. Inspect which assurance owner can retain test population, sample, exceptions, exact implementation version and independent outcome. Use an existing owner only if its real contract preserves those facts; otherwise add one focused test-result owner, not a new control catalog.
- [ ] Write tests for risk edit retaining accepted rating, appetite and unrelated links; test conclusion becoming non-current after a changed implementation; review rejecting a stale source, conflicting actor or unavailable evidence. Verify the prior conclusion remains reconstructable.
- [ ] Extend the current RCSA cycle and form-distribution boundary for draft/resume and per-risk review only where needed. Preserve frozen risk/control membership, submitted source/version and challenge Matter. Source changes must lead to challenge or resubmission, not silent recalculation of an accepted rating.
- [ ] Build all nine reference experiences with source-backed query/filter/detail and exact control/RCSA handoffs. Prefill existing risk/owner/control facts; disclose only unresolved test fields; show one actor-specific action. No new rating follows task completion alone.
- [ ] Run `go test ./internal/risk ./internal/rcsa ./internal/controlcatalog ./internal/continuity ./internal/httpapi` and `npm test -- --run src/components/risks/RisksWorkspace.test.tsx src/components/rcsa/RCSAWorkspace.test.tsx`. Add PostgreSQL concurrent review and frozen-membership tests and source-to-Work reconciliation.

## Slice 7 — Indicators and branch KRI returns

**Reference:** `key-indicators`, `indicator-workspace`, `branch-kri-returns`, `branch-kri-return`, `kri-cycle-setup`, `kri-question-library`. **Current:** `components/indicators/`, `indicatorApi.ts`, `monitoringApi.ts`, `internal/monitoring/`, `internal/risk/indicator_population*.go`, `internal/metricview/`; source mapping `clearsight-v2.9/docs/BRANCH_KRI_MAPPING.json`.

- [ ] Map the reference 51-question source structure, units, conditional applicability and polarity to versioned form/check contracts. The source explicitly lacks thresholds; do not infer thresholds, comparable trends or completeness. Treat the source pack as provenance-bearing reference data and check licensing and edition before installation.
- [ ] Test a scoped cycle population, draft/resume, submitted immutable revision, independent review, return/correction, approved N/A versus missing, and an exception task that links the exact return/version. Prove evidence/source permissions and changed definition guard.
- [ ] Extend existing forms and monitoring owners for return-specific typed metadata and review only where the existing contract is insufficient. Retain native count/percent/duration/money units; do not add a second question engine or convert observations to compliance findings automatically.
- [ ] Implement register, cycle setup, question library, exact return, review and indicator detail in 2.9 anatomy. Prepopulate eligible branch and previously approved applicable answers with attribution; ask only unresolved facts. Dates and overdue counts come from the real cycle and scoped population.
- [ ] Run `go test ./internal/monitoring ./internal/formcontract ./internal/risk ./internal/metricview ./internal/workflow` and `npm test -- --run src/components/indicators/IndicatorDetail.test.tsx src/components/indicators/IndicatorValue.test.tsx`. Add source-to-aggregate and definition-comparability tests plus large-cycle query plans.

## Slice 8 — Operational losses, recovery and worksheet import

**Reference:** `losses`, `loss-workspace`, `loss-intake`, `loss-import`, `loss-insights`. **Current:** `components/losses/`, `lossApi.ts`, `internal/oploss/`, `internal/httpapi/operational_loss_routes.go`, `internal/metricview/loss_period*.go`, migration `000110_operational_losses.up.sql`. Detailed target and safe migration are in companion design §5.

- [ ] Inspect every current `oploss` reader, report/metric consumer and recovery aggregate. Record actual `ACTIVE`/`VOIDED` and immediate recovery behavior and the current version/transaction semantics; create a migration reconciliation query grouped by tenant, entity, currency and state without exporting private descriptions.
- [ ] Agree the legacy reporting policy with the accountable bank owner before promoting a new reviewed-only basis. Preserve old financial facts and unknown approval provenance; never fabricate independent reviewer/decision or recognition date. Until resolved, label legacy cohort separately and do not call it independently reviewed.
- [ ] Add failing tests for incomplete saved draft, submitted snapshot excluded from totals, independent accept/return, stale correction preserving accepted fact, verified recovery/reversal against the original receipt, simultaneous over-recovery and closure/reopening. Assert loss, receipt, decision, event/outbox and Work changes are atomic. Test int64 overflow and separate currencies.
- [ ] Design additive migration and `oploss` service/repository commands with one current accepted owner, versioned drafts/proposals and exact decision receipts. Lock the relevant loss/ledger while checking pending capacity and accepted total. Enforce verified identity/current route at execution and store explicit reporting/recognition dates without deriving one from another.
- [ ] Implement 2.9 register, four-stage intake, exact record and review, received-recovery/reversal, closure and import. Import preserves original source columns/row; map organization explicitly, flag ambiguous duplicates and period conflicts, stage only drafts, and never turn a recovery narrative into a receipt. Reuse existing ingestion/scanning where supported.
- [ ] Upgrade or block old direct writers before enabling new invariants; keep compatible reads during deployment. Reconcile pre/post totals and source-to-report cohorts, rehearse backup and rollback; after new irreversible history, roll forward or use a proven compatibility reader rather than deleting decisions.
- [ ] Run `go test ./internal/oploss ./internal/metricview ./internal/workflow ./internal/httpapi` and `npm test -- --run src/components/losses/LossAuthoringDialog.test.tsx src/components/losses/LossRecord.test.tsx src/components/losses/LossRegister.test.tsx`. Add PostgreSQL transaction/race tests and receipt/cash/reporting cohort reconciliation before enabling real writes.

## Slice 9 — Vendor lifecycle and assurance

**Reference:** `vendors`, `vendor-engagement`, `vendor-review`. **Current:** `VendorsWorkspace.tsx`, `VendorRelationshipWorkspace.tsx`, `VendorDueDiligence.tsx`, `VendorWorkPanel.tsx`, `vendorApi.ts`, `vendorAssessmentApi.ts`, `vendorCollectionApi.ts`, `internal/thirdparty/`; vendor and assessment routes in OpenAPI.

- [ ] Map canonical vendor identity versus scoped relationship, service criticality, assessment, collection receipt, evidence review and activation. Trace current `internal/thirdparty/activation_policy.go` and assessment lifecycle; compare 2.9 periodic review, suspension/resumption and verified exit to actual commands and events.
- [ ] Test exact relationship visibility, same-entity evidence reuse without permission expansion, pending/expired/returned assessment, independent activation and lifecycle decision, stale assurance at execution, and data-disposition evidence for exit. A form submission alone does not activate a vendor.
- [ ] Build reference collection, engagement and review with a single current source of vendor/relationship truth. Add missing versioned lifecycle decisions in `thirdparty` only when current state cannot represent them. Reviewers see source snapshot and change; submitter sees the exact correction path.
- [ ] Reuse forms, capture, documents, Work and current authority; prefill existing linked evidence and contracts with freshness. Avoid an additional vendor score or cloned risk engine. Preserve existing identity edit and vendor communication boundaries.
- [ ] Run `go test ./internal/thirdparty ./internal/monitoring ./internal/evidence ./internal/workflow ./internal/httpapi` and `npm test -- --run src/components/VendorsWorkspace.test.tsx src/components/VendorDueDiligence.test.tsx`. Prove exact return/deep links, bulk request duplicate handling, cross-entity denial and retained lifecycle history.

## Slice 10 — Privacy, processing and protected cases

**Reference:** `privacy`, `processing-activity`, `processing-registration`, `processing-version`, `privacy-impact-assessment`, `privacy-impact-wizard`, `privacy-case`, `privacy-case-intake`. **Current:** `components/ropa/`, `ropaApi.ts`, `internal/ropa/`, `internal/httpapi/ropa_routes.go`, existing Matter and protected-access contracts.

- [ ] Map ROPA registration/update/history/transition to the existing service; trace actual case and impact owners separately. The visible ROPA routes do not by themselves establish DPIA or case functionality. For each, document data classification, permitted roles, legal entity, retention, decision authority and exact source link.
- [ ] Test processing version history, authorized registration and transition; for cases test intake, independent review, return, closure and reasoned reopening with previous episode intact. Verify restricted rows, counts, notifications, document titles and source links do not leak through a general Work or report query.
- [ ] Implement all eight 2.9 privacy experiences. Reuse ROPA and the appropriate protected case/Matter ownership if semantics and source-level permissions fit. Otherwise introduce a narrow protected aggregate with explicit purpose and retention, rather than storing sensitive details in generic task context.
- [ ] Reuse approved organization, vendor, system and evidence records; surface source and freshness, ask only unresolved impact facts. Do not infer legal effect, breach status or review conclusion from an uploaded document or a Signal.
- [ ] Run `go test ./internal/ropa ./internal/continuity ./internal/evidence ./internal/workflow ./internal/httpapi` and the affected ROPA/Privacy browser and component suites identified from `web/src/components/ropa/`. Add PostgreSQL scope tests and role-separated access/reopening evidence.

## Slice 11 — Reports, board packs and Home

**Reference:** `reports`, `insight-detail`, `report-builder`, `report-pack`, `loss-insights`, `index`. **Current:** `components/reports/`, `components/oversight/`, `reportingApi.ts`, `metricApi.ts`, `internal/reporting/`, `internal/metricview/`, `internal/oversight/`; `reporting_routes.go` and relevant later migrations.

- [ ] Trace live metrics, authorization, source-membership drills, report definition, worker run, artifacts, expiry and board-brief paths through current code and all later migrations. Classify live perspective, export run and retained published pack separately; do not assume `000093` alone describes the current schema.
- [ ] Write tests for scope/freshness/denominator, metric-to-source member parity, missing versus zero, current-posture versus period selection, and loss cohort/cash window by original currency. Verify a captured pack remains identical after source edits and permission changes do not broaden access to retained source detail.
- [ ] Reuse existing bounded metrics and report-run infrastructure where it meets the contract. If a published pack needs different immutable membership/retention, add one narrow reporting-owned record with source versions, filters, as-of, narrative and artifact digests. Do not create a second operational ledger or a dashboard-specific work queue.
- [ ] Implement 2.9 Home, perspectives, insight detail, report builder, pack and loss insights. Keep one source-backed Work destination, show no unsupported KPI or sample score, and make filters issue authorized queries. Avoid requiring a new governed report definition merely to view an existing authorized live perspective.
- [ ] Run `go test ./internal/reporting ./internal/metricview ./internal/oversight ./internal/httpapi` and `npm test -- --run src/components/reports/ReportsWorkspace.test.tsx src/components/reports/ReportsExplore.test.tsx src/components/oversight/OversightWorkspace.test.tsx`. Add PostgreSQL cohort, snapshot/replay, download and source-to-report tests; inspect printed/exported artifact at each relevant page size.

## Slice 12 — Settings, identity, authority, integrations and automation

**Reference:** `integrations`, `settings`, `data-checks`, `roles-permissions`, `role-detail`, `access-review`, `escalation-matrix`, `escalation-policy`. **Current:** `components/configure/`, `components/access/`, `identityAccessApi.ts`, `governanceAdminApi.ts`, `sourceConfigApi.ts`, `internal/access/`, `internal/authority/`, `internal/governance/`, `internal/sourceaccess/`, `internal/autonomy/`.

- [ ] Map every 2.9 matrix cell, scope and escalation policy to actual server permissions, role bindings, organizational positions, effective authority and current configuration lifecycle. Identify access-review or direct-grant gaps from real services; the prototype's local role model is not a server authorization model.
- [ ] Write tests for simulation and impact preview, maker-checker, conflict/delegation/revocation, future effective date, missing route, rollback, audit and last-admin protection where applicable. A forged actor or scope cannot grant authority; execution re-resolves the current route.
- [ ] Build reference layouts around shared governance contracts, showing purpose/scope and eligible presets first; expose full detailed authority and policy configuration when needed. Retain distinct review, authorize and signatory roles. No fixed approvers or silent permission change.
- [ ] Map integrations to existing Connection/View/Binding and source-health contracts; show actual configuration and errors, not a simulated connected status. Preserve organization, automation, AI governance and operational settings even where the 2.9 reference is incomplete; design missing pages in its visual language.
- [ ] Run `go test ./internal/access ./internal/authority ./internal/governance ./internal/sourceaccess ./internal/autonomy ./internal/httpapi` and `npm test -- --run src/components/configure/ConfigureWorkspace.test.tsx src/components/configure/PeopleAccessSection.test.tsx src/components/configure/DataIntegrationsSection.test.tsx`. Include PostgreSQL concurrent activation, restore and fail-closed tests.

## Slice 13 — Matters, imports, AI, cleanup and production acceptance

**Current-only surfaces:** `components/MattersWorkspace.tsx`, `MatterRecordWorkspace.tsx`, `DocumentImportWorkspace.tsx`, `BankJourneysWorkspace.tsx`, `EmployeeProfileWorkspace.tsx`, `components/insights/`, `components/configure/AIGovernanceSection.tsx`, `internal/continuity/`, `internal/documentimport/`, `internal/aigovernance/`, `internal/aigateway/`.

- [ ] For each retained current-only task, add a 2.9-style page brief and destination to the route ledger. Preserve Program/Matter semantics, protected response/signatory flows, import provenance and source review, onboarding, group scope and existing authorized functionality. An absent prototype entry does not authorize silent deletion.
- [ ] For Copilot, use the current governed gateway only when available and authorized; limit visible suggestions to relevant actions and disclose source/freshness. Source context is scoped and minimized; no synthetic regulatory conclusion, hidden action or dependency on AI for deterministic controls. Test unavailable model, prompt-injection content and stale source. Do not send messages externally without explicit authorization.
- [ ] Perform the removal backlog in the companion design: trace shell/CSS imports, dynamic classes, `staticDemo.ts`, evidence and runtime builds, API clients, stale route wrappers and duplicated tests. Remove only a verified last consumer and its obsolete rules. Keep database history, migrations and evidence unless a separate approved retention/data migration exists.
- [ ] Run full `go test ./...`, `go vet ./...`, supported PostgreSQL and integration tests, `npm test`, `npm run build`, `npm run build:evidence`, runtime-truth and UI-contract checks; record environment, failures, skips and final exact tree. Use targeted browser evidence plus complete route-ledger verification and accessible interaction tests.
- [ ] Rehearse deployment with sanitized representative data: source-to-record/report totals, permissions, migration, jobs, retries, retention, backup, rollback/roll-forward, capacity and recovery. Keep old URL aliases until actual retained links and consumers are migrated. Monitor by domain and prohibit automatic production cutover on visual or CI results alone.
- [ ] Obtain named acceptance for each route and four-actor end-to-end outcomes, authorized backend results and source-to-report reconciliation. Declare completion only when the route ledger has no unresolved capability, every enabled action works, all relevant acceptance gates have evidence, and outstanding exceptions are explicitly recorded and accepted.

## Sequencing checkpoints and handoff

At the end of each slice, record: exact source hashes; dirty baseline plus final content identity; route/action rows closed; tests and results; paired renders and named reviewer; measurable friction/bundle impact; migrations and reconciliation; remaining unsupported states; and the rollback point. Implement only the next independently testable slice after checking these artifacts, and preserve current service compatibility until its dependent domains have moved.

**Not yet established:** The current repository inspection has not executed real-browser, database, user-effort or migration tests. The legacy-loss reporting policy, actual production data cardinalities and some reference-to-service mappings require evidence or bank policy decisions. They are implementation gates, not permission to invent values or fabricate receipts. This plan is detailed at slice/contract level; a slice owner must write exact test code, SQL and endpoint payloads after the specified baseline inspection and before implementing that slice.
