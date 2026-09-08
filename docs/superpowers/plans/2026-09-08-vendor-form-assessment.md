# Vendor Form Assessment Implementation Plan

> **For agentic workers:** Use subagent-driven-development for bounded implementation and review tasks, and dispatching-parallel-agents only for independent file ownership. Execute continuously; the user approved this scope on 2026-09-08.

**Goal:** Add bank field assessment and accessible policy configuration to existing Forms, then surface requests, responses and risk in Vendors.

**Architecture:** Extend immutable form contracts and response revisions with separately versioned bank judgements. Reuse distribution, evidence access, current authority routes, formpolicy execution and vendor relationship associations. Vendor summaries are scoped server reads rather than a browser-built inventory.

**Tech Stack:** Go, PostgreSQL/pgx, React/TypeScript, existing shared React Aria controls, Go tests, Vitest, browser-rendered fixtures.

Baseline: origin/main 60a6a606; feature branch codex/vendor-field-assessment. Old tracked edits remain in the named preservation stash and are not part of this implementation.

## Shared contracts

The field contract gains optional assessment configuration. Absence preserves legacy behavior.

```ts
type FieldAssessment = {
  mode: "NONE" | "MANUAL" | "AUTOMATIC" | "AUTOMATIC_REVIEW";
  required: boolean;
  weight: number;
  reviewer_role?: string;
  rubric?: Array<{ id: string; label: string; points: number }>;
};
```

Assessment reads and commands use the exact completed response ID:

```text
GET  /api/v1/forms/responses/{id}/assessment
POST /api/v1/forms/responses/{id}/assessment
```

The command accepts an expected assessment version plus field decisions (field ID, rubric outcome, rationale). Tenant, legal entity and reviewer are server-bound. GET returns the response ID, assessment version/state, applicable field labels/answers/configuration, existing decisions, required/reviewed counts, automatic score and assessed score. Historical decisions remain immutable. Implementers must publish the concrete Go/JSON result contract before dependent UI integration.

## Task 1 — Shared form assessment and durable bank decisions

Files: internal/formcontract/model.go, validation.go, scoring.go and new assessment.go/tests; internal/evidence/response_assessment*.go; the next database migration; dedicated internal/httpapi response-assessment handlers/tests; existing route registry and schema ownership contract.

- [x] Add failing contract tests for mode/rubric/weight validation, legacy preservation, hidden fields, required unreviewed fields, manual/composite scoring and critical-rule preservation.
- [x] Run `go test ./internal/formcontract -count=1`; confirm new tests fail because the contract is absent.
- [x] Add the shared field configuration and evaluator extension. Bank judgements must never enter the respondent answer map.
- [x] Add failing service/store tests for exact response scope, current revision, verified reviewer, optimistic conflict, judgement supersession, invalid rubric choice and transactional event/outbox writes.
- [x] Implement memory/PostgreSQL persistence and command/read endpoints using the existing completed-response access boundary and current authority.
- [x] Verify `go test ./internal/formcontract ./internal/evidence ./internal/httpapi -count=1` and tagged PostgreSQL compilation/integration where configured.

## Task 2 — Builder and response assessment UI

Files: web/src/monitoringTypes.ts; existing form-authoring serialization helpers and field inspector; new FieldAssessmentEditor and ResponseAssessment components/tests; web/src/formAssessmentApi.ts; Forms Responses integration; shared capture/form serialization types only where necessary.

- [x] Add failing round-trip and builder tests for all four modes, required rubric/route fields and source-generated form preservation.
- [x] Implement labelled assessment controls in the existing inspector and a form-wide assessment summary. Reuse automatic scoring controls and server preview.
- [x] Add failing response tests proving Submitted remains Awaiting bank review, rubric/rationale submission is exact-revision scoped, errors preserve entries, historical responses cannot alter current judgements and poor results are inspectable.
- [x] Implement the response assessment panel against Task 1's published contract; export it for the vendor workflow. Display automatic result separately from assessed result.
- [x] Run targeted Vitest tests and `npm run typecheck`; preserve all legacy builder tests.

## Task 3 — Policy configuration and assessed-result eligibility

Files: web/src/components/forms/FormPolicyEditor.tsx, FormPoliciesView.tsx and tests; web/src/formPoliciesApi.ts; internal/formpolicy model/service/executor and tests; dedicated policy-choice handlers if canonical choice reads are absent.

- [x] Add failing UI tests that eligible named automation policies and subjects can be selected without copied IDs.
- [x] Replace raw identifier controls with bounded scoped selectors and contextual canonical configuration links/flows. Preserve all current policy lifecycle actions.
- [x] Add explicit automatic-versus-bank-assessed eligibility, preserving legacy automatic policies. The assessed basis must require final required bank assessment.
- [x] Test simulation/execution basis, no duplicate adverse episode across both result types, revoked authority and replay safety. Reuse current event/outbox/inbox and result access.
- [x] Run targeted Go/Vitest tests and relevant API-contract checks.

## Task 4 — Vendor request/response summaries and bulk request reuse

Files: internal/evidence vendor-form query implementation/tests and HTTP handlers; web/src/vendorFormsApi.ts; new VendorFormsPanel/VendorFormRequest components/tests; VendorsWorkspace.tsx/tests and vendors.css; existing DistributionComposer via backward-compatible initial context.

- [x] Add failing scoped query tests for current vendor/service/workflow links, restricted records, multiple forms, partial capture, missing assessment and historical response handling.
- [x] Implement bounded overview and exact vendor-detail query contracts. Count/filter before pagination; no client fan-out or authorization after loading broad data.
- [x] Add a request action that reuses approved forms and existing distribution/request origins. Preselect vendor context, preserve recipient/delivery/access and existing due-diligence/work handling.
- [x] Add explicit multi-vendor selection and bounded idempotent batch handling over canonical distribution commands where absent. Persist per-target receipts and retry only pending/failed targets.
- [x] Add UI tests for incomplete response, submitted poor result, pending bank review, open findings and exact drill-down. Preserve relationship detail, identity and existing lifecycle controls.
- [x] Run scoped Go/Vitest tests, API contracts and typecheck.

## Task 5 — Integration, workbook journey and rendered proof

Files: docs/product/governed-forms.md, DESIGN.md, docs/implementation-plan.md, docs/acceptance/vendor-form-assessment.md, rendered fixtures/evidence script, README as applicable.

- [x] Use the existing document-to-form path and the supplied XLSX to verify source mappings survive into ordinary assessment configuration. Do not create a special importer or infer approved numeric scoring from source Medium ratings.
- [x] Exercise one mixed form, two vendor targets, partial save, submission, automatic poor result, manual bank judgement, policy simulation and duplicate-safe handling.
- [x] Perform spec compliance review, fix gaps, then independent code-quality/security review and fixes.
- [x] Run affected tests, full frontend tests, copy-quality, typecheck, UI contracts, build, Go tests and tagged compilation; run configured database tests and explicitly record skips.
- [x] Capture before/after desktop, tablet, 390/320px, light/dark and zoom states; inspect and repair the highest-impact defect. Verify focus/keyboard, axe, failed-save recovery, no overflow and unobstructed controls.
- [x] Record measured bounded-query evidence against existing performance targets. Synchronize documentation and API/schema contracts, then review `git diff --check` and final status.

## Progress

Implementation and local verification are complete. The [acceptance record](../../acceptance/vendor-form-assessment.md) contains exact test, migration, workbook, responsive-render and interaction receipts. Remote synchronization preceded implementation; the earlier tracked changes remain separately preserved. Existing importer anchor limitations, production-volume acceptance, bank-user timing, deployed migration checks and real recipient delivery remain explicitly recorded release boundaries. No deployment or live vendor communication was performed.
