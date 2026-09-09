# Vendor Evidence Reconciliation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Reconcile a prior vendor document against a current collection item and show truthful pending vendor work, bank review and accepted evidence in one clear workspace.

**Architecture:** Extend existing assessment/request/capture and protected document occurrence services. A separate bank receipt clears collection without creating respondent answers or inheriting acceptance. React renders current server facts and reuses the existing document browser and review actions.

**Tech Stack:** Go, PostgreSQL/pgx, React/TypeScript, Vitest, shared ClearSight UI contracts.

## Task 1 — Persist and expose collection reconciliation

Owner: backend implementation agent; files in `internal/evidence`, `internal/thirdparty`, `internal/httpapi`, migrations and backend API documentation.

- [x] Write failing domain tests for receipt creation with actual source occurrence, duplicate retry, version conflict, wrong relationship/entity, revoked authority, expired/unavailable source and absence of forged vendor answers.
- [x] Extract distribution preparation without issuing access or delivery; add assessment preparation preserving READY_TO_SEND and the exact origin. Existing Send resumes the same request.
- [x] Add current collection read and append-only reconciliation receipt with transactional assessment/request versions, authority re-evaluation, event and outbox. Reject client-supplied collection resolution metadata in ordinary request/form creation.
- [x] Add HTTP routes and forged-identity/current-scope tests. Material commands derive actor from verified context.

```text
GET  /api/v1/vendor-assessments/{id}/collection
POST /api/v1/vendor-assessments/{id}/prepare-request
POST /api/v1/vendor-assessments/{id}/collection/{field_id}/reconcile
```

```ts
type PrepareRequestInput = { expected_version: number; audience: string; deadline: string };
type ReconcileInput = {
  expected_version: number; request_id: string; expected_request_version: number;
  source_submission_id: string; source_field_id: string; source_artifact_id: string;
  source_response_revision_id?: string; rationale: string;
};
// Reconcile result: { collection: AssessmentCollection; receipt: CollectionResolution }.
// A resolution contains the protected DocumentOccurrence, bank actor/time/reason,
// and its own identity/version. It is not a new vendor submission or acceptance.
```

## Task 2 — Respect receipts through collection and review

- [x] Write failing capture tests: mixed outstanding fields still validate; receipt-backed documents are not demanded again; receipts never enter vendor answer maps or inflate answered counts; callers cannot submit arbitrary reused artifacts.
- [x] Consume current server-owned resolutions under submission transaction/version checks. Present safe already-received notices to the invited vendor.
- [x] Update request progress/reminder eligibility without treating an outstanding scalar response as complete. Conditional fields retain correct applicability.
- [x] Merge reused evidence into bank review through explicit source membership. Require separate document decisions. Add a real bank transition for all-held collection without manufacturing a vendor submission.
- [x] Run `go test ./internal/evidence ./internal/thirdparty ./internal/httpapi` and affected tagged PostgreSQL tests. Record any unavailable deployment prerequisites accurately.

## Task 3 — Build the clear checklist and protected selection flow

Owner: root; files `web/src/vendorCollectionApi.ts`, `web/src/components/VendorEvidenceChecklist.tsx`, its tests/style, document browser selection extension, `VendorDueDiligence.tsx`, `VendorsWorkspace.tsx` and API/interaction tests.

- [x] Write failing tests for distinguishing reused-but-unreviewed evidence from accepted evidence, pending-first rows, protected document selection, saved reconciliation, failure/retry and stale status.
- [x] Implement typed collection API with abortable reads and exact encoded source selectors for writes.
- [x] Implement scoped summary/filter controls and requirement rows using shared UI contracts. Unknown/loading states cannot show fabricated zeros. Scalar receipt is not labelled compliance.
- [x] Add an optional real document-selection action to the existing browser, retaining read-only behavior elsewhere. Show the selected source and reason before committing reconciliation.
- [x] Wire preparation, checklist refresh and current review actions into the vendor workspace. Preserve actual backend decision/authority boundaries and submitted-history views.

```powershell
node node_modules/vitest/vitest.mjs run src/components/VendorEvidenceChecklist.test.tsx src/components/VendorDueDiligence.test.tsx src/components/VendorsWorkspace.test.tsx src/copyQuality.test.ts --maxWorkers=4 --reporter=dot
node node_modules/typescript/bin/tsc -b --pretty false
node node_modules/vite/bin/vite.js build --config vite.evidence.config.ts
```

## Task 4 — Render, review and document the complete vertical

- [x] Add production-component evidence fixtures for the decision brief's states and preserve the before-state structure/source baseline.
- [x] Inspect rendered desktop/light, desktop/dark, mobile and reflow with the browser tool. Check pending/satisfied hierarchy, source labels, primary action, focus, errors and overflow; fix and recheck observed defects.
- [x] Independently review backend/spec conformance and frontend quality, fix findings and rerun affected checks.
- [x] Synchronize `DESIGN.md`, product/architecture notes and acceptance receipt with actual behavior and its limits. Do not claim the broader spreadsheet interpretation or every onboarding correction is delivered by this evidence-reconciliation vertical.

Baseline: three focused frontend files passed, 104 tests, before implementation. Source checkout had no tracked edits; unrelated untracked files remain untouched. Work is on `codex/vendor-evidence-reconciliation`.

Verification: 198 affected frontend tests; 12 runtime/UI contract checks; 23 retained rendered states; TypeScript and both build entries; evidence/thirdparty/HTTP unit suites; PostgreSQL-tagged runtime compilation; actual PostgreSQL 18.6 reconciliation, document-review and four capture/currency scenarios. See [acceptance](../../acceptance/vendor-evidence-reconciliation.md) for proof and release limitations. No deployment or external delivery performed.
