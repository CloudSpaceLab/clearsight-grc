# Shared document management implementation plan

> **For agentic workers:** Use subagent-driven-development for bounded implementation tasks, with specification and quality review before integration. Execute in this session; user approval to continue is recorded on 7 September 2026.

**Goal:** Make uploaded form evidence discoverable, previewable, reviewable, replaceable and governably removable from both Vendors and Forms.

**Architecture:** Extend Evidence/Capture rather than introduce a document store parallel to capture artifacts. Reuse exact response/submission membership, third-party review commands, authority, distribution delivery and leased workers. Keep security state, review acceptance, document currency, form signatures and parent conclusions distinct.

**Tech stack:** Go, PostgreSQL/pgx, existing object-store interface and worker runtime, React/TypeScript with shared UI components.

Approved design and detailed gap tracker: `../../reviews/2026-09-07-form-and-vendor-document-management.md`. Starting revision: `831b009d8eea743898b7c10ea3858edd25b35e2f`. Branch: `codex/shared-document-management` in the existing `governed-workflow-audit` worktree. Never stage the unrelated presentation image.

## Mandatory behavior

- Signatures are requested only where the source form defines an applicable signature field. Reviewer acceptance adds no signature requirement.
- No unscanned/quarantined bytes may be previewed, downloaded or accepted. An unavailable scanner is an explicit recoverable condition, not a clean result.
- Read membership uses exact revision → submission → answers → artifact. Do not use the artifact's first submission ID as its entire history.
- New document versions require new review; a replacement request alone does not supersede existing evidence.
- Draft removal, withdrawal from use and permanent disposal have different outcomes. Disposal fails closed without effective retention and hold checks.
- No new DMS, AI reviewer, search service, folder tree or queue service.

## Task 1 — inspection lifecycle (DOC-01)

Files: new focused `internal/evidence/artifact_scan*.go` files and tests; additive migration numbered after current main; `cmd/worker/services_postgres.go`, worker config/composition; relevant storage docs.

- [x] Write failing tests for clean, infected, unavailable, truncated/oversize, wrong digest, expired lease and replayed completion.
- [x] Implement a bounded scanner interface over an `io.Reader`; a clean scan is a receipt tied to artifact ID, exact digest, scanner/version and time. An actual scanner adapter must distinguish malware from timeout/service failure.
- [x] Persist due scan jobs transactionally with artifact manifests. Claim with bounded batches/leases; complete manifest state, scan receipt and event atomically. Retry is bounded and terminal failure remains visible.
- [x] Wire the scanner through explicit configuration in the existing worker. Keep unavailable configuration observable. Do not add a permissive scan bypass.
- [x] Run `go test ./internal/evidence ./cmd/worker ./internal/platform/config` and `go test -tags postgres ./internal/evidence ./cmd/worker`; inspect changed migration and retention ownership. Review and commit only task files.

Contract shape:

```go
type ArtifactScanner interface {
    Scan(context.Context, io.Reader, int64) (ArtifactScanResult, error)
}
// Only an explicit clean result can make an artifact AVAILABLE.
// Transport or scanner errors leave it unavailable and schedule bounded retry.
```

## Task 2 — exact document reads and safe content delivery (DOC-02)

Files: `internal/evidence/completed_response*.go`, new `response_documents*.go`; `internal/httpapi/form_responses.go`, new document handlers, route registry and API contract; vendor document handlers/tests.

- [ ] Add failing read tests for ordinary file/photo/vendor_document fields, historical revisions, reused artifacts, multiple recipients, guessed IDs and cross-entity/restricted subject reads.
- [ ] Add a bounded document query returning safe metadata, source field/response, uploader vs submitter attribution, security state and canonical review source. Filter scope before LIMIT; cursor order must be stable.
- [ ] Extend completed-response detail with immutable answers and document occurrences after the existing exact response authorization. Historical detail resolves its own submission.
- [ ] Add protected preview/download routes using the same authorization and exact membership. Permit business-rejected/expired historical evidence when bytes are AVAILABLE; enforce `private, no-store`, media type, nosniff and bounded delivery. Never expose keys or bearer tokens.
- [ ] Test both memory and PostgreSQL stores, including current-only vs history and bounded query plans. Update executable route contract and commit after review.

UI data must identify an occurrence, not merely a filename:

```ts
type DocumentOccurrence = {
  artifact_id: string;
  request_id: string;
  submission_id: string;
  field_id: string;
  file_name: string;
  media_type: string;
  size_bytes: number;
  artifact_status: string;
};
```

## Task 3 — Finder-style shared file browser and Quick Look (DOC-03)

Files: new `web/src/components/documents/` components/types/styles/tests; `FormsWorkspace.tsx`, `forms/ResponsesView.tsx`, vendor workspace/document components, relevant API clients, fixtures, DESIGN.md.

- [ ] Add component tests for same document from both interfaces, one-click opening, historical states, loading/failure/retry, authorized downloads, no-byte preview states and keyboard focus restoration.
- [ ] Build one reusable Finder-style file browser using existing table/selection, FocusedSheet, button and notice contracts. Use recognizable file icons and easy All files / PDF / Images / Word / Spreadsheets / Other filters, a compact filename search and selected-file details. Advanced search and validation stay secondary. PDF/images preview inline in Quick Look; unsupported formats show accurate metadata/download. Preserve source question, timestamp, filter/selection and next/previous navigation.
- [ ] Test file-type normalization, keyboard single-selection, Space/Enter preview outside editable fields, Escape/restored focus, and browser-pane replacement at narrow widths. Do not claim Word/spreadsheet page previews without a real safe renderer. Do not create more mockups: render the actual production component with labelled state fixtures.
- [ ] Mount in Forms Documents, exact Forms response and Vendor relationship Documents. Use exact IDs/relationships; no email/name matching. Do not put respondent draft uploads into bank review lists.
- [ ] Render desktop, narrow viewport, both themes and 200% reflow with required fixtures; fix highest-impact defects. Run affected tests, copy-quality, typecheck, build and UI contracts. Review and commit.

## Task 4 — document acceptance and rejection (DOC-04)

Files: `internal/thirdparty/assessment_document.go`, existing review repositories; new shared document decision records/services where no domain review exists; HTTP commands, document panel and tests.

- [ ] Add failing tests for file/photo parity with vendor_document, stale version, unauthorized actor, review after replacement, historical read vs current mutation and unchanged parent conclusion.
- [ ] Record accept/reject with exact occurrence, purpose, rationale, actor/time and expiry. Delegate existing vendor assessment decisions; never maintain conflicting parallel acceptance flags.
- [ ] Resolve allowed actions from current authority and workflow state; recheck during mutation. Generic form evidence uses an explicit purpose-bound reviewer route rather than treating every viewer as a reviewer.
- [ ] Show Accept document, Request new upload and secondary Reject document alongside reviewer and history. No added signature prompt; source form signature rules continue to govern capture.
- [ ] Verify atomic record/event/outbox writes, concurrency, both surfaces and permission recovery. Review and commit.

## Task 5 — targeted replacement (DOC-05)

Files: assessment clarification/vendor-work services; shared Evidence distribution successor integration; document action sheet/API tests.

- [ ] Add failing tests for single selected file field, correct recipient route, required conditional signature field, omitted unrelated upload, duplicate click/retry, obsolete form revision and completed assessment.
- [ ] Reuse assessment clarification/vendor work change commands; add generic Forms successor handling through the distribution boundary. Preselect field/recipient/owner from stored context and require reason/deadline.
- [ ] Preserve immutable predecessor attribution, require fresh selected bytes, carry no signature as newly signed, and retain unaffected answer context. Bind idempotency to the document occurrence/replacement action.
- [ ] Expose delivery/request/submission/review states and link old/new versions. Completion stops reminders; closed assessments enter an eligible follow-up episode.
- [ ] Verify a real configured delivery journey without claiming log output is delivery; review and commit.

## Task 6 — removal and disposal (DOC-06)

Files: shared document disposition service/repository/migration; existing object-store adapters/worker/config; focused UI actions, audit and tests.

- [ ] Add failing tests for draft detachment, submitted withdrawal, referenced-use preview, legal hold, absent retention authority, deletion retry and duplicate completion.
- [ ] Detach draft answers under existing workspace/version authority and clean only unreferenced eligible objects. Preserve immutable submitted answers.
- [ ] Record reason-bearing withdrawal and affected uses. Permanent disposal requires effective retention/hold/authority decisions and a durable monitored job; never reinterpret withdrawal as byte deletion.
- [ ] Wire version-aware storage deletion with original/derived bytes addressed explicitly and a retained permitted audit tombstone. Surface recovery and limits; no enabled no-op Delete control.
- [ ] Verify production storage configuration and retention behavior with integration tests, then review and commit. Missing external configuration is a deployment blocker, not a reason to invent a provider or policy.

## Task 7 — integration and release evidence (DOC-07)

- [ ] Synchronize approved design, this ledger, schema ownership, acceptance fixtures and product docs with actual maturity.
- [ ] Run full Go, postgres-tagged, vet, web, copy, accessibility, runtime/API contract and build checks. PostgreSQL integration must execute against a real database; a skip is not proof.
- [ ] Verify the 11 acceptance scenarios in the approved evaluation, including both interfaces, multiple contributors, preview, signature rules, replacement and hold-protected disposal.
- [ ] Perform specification review, then code-quality/security review, repair findings and repeat affected checks.
- [ ] Integrate without overwriting unrelated changes. Verify CI and deployment when releasing; do not claim production readiness without storage/scanner/retention/provider evidence.

## Execution log

- 7 September: proposal approved, signature requirement confirmed, current main fetched; no remote divergence. Focused baseline Go suites and 52 web tests passed during evaluation. Feature branch created in the existing worktree.
- 7 September: Task 1 implemented in `d916a018`; independent specification review passed, code-quality/security review remains pending. Portable PostgreSQL 18.6 was initialized on loopback with sample-only data. All migrations applied in ordinal filename order; focused scan transaction tests and the complete Evidence `postgres,postgresintegration` suite executed successfully against this database. No production scanner or storage configuration has been changed. Failed-job operator requeue remains a release recovery gap, alongside provider setup and immutable/digest-verified content delivery.
- 7 September: user extended the scope to focused premium viewing, advanced filters and AI-assisted validation rulesets, and requested a browser mockup before implementation. See the proposed [extension](../specs/2026-09-07-document-review-and-validation-design.md). New validation behavior awaits design review; no automatic acceptance authority is assumed. The sample concept passed browser interaction checks at 1440, 720 and 390px, with light/dark visual inspection and corrections. This is not application UI completion.
- 7 September: user rejected that mockup and directed implementation of a familiar macOS Finder-style browser with easy file-type filters/previews and no further mockups. Task 3 and the extension now record that superseding instruction. Original scanner specification and independent quality review passed; a nonblocking adapter/worker integrity-error classification finding is being corrected before the next tranche.
- 7 September: scanner integrity-error fix `03b2be03` passed independent re-review and real PostgreSQL checks. Task 1 implementation is complete; production scanner setup and exhausted-job requeue remain explicit release gaps, not clean-scan claims.
- 7 September: actual shared file browser implemented in Forms Documents and vendor/response document launchers. Specification review passed after preserving full upload/submission/review timestamps and distinct actors. Quality review found no critical/important issues; its minor duplicate-selection emission was reproduced and fixed. Both builds, runtime isolation and UI-contract checks pass. Ten Forms/Vendor browser combinations include light/dark desktop, 720px reflow, 390px and 320px, with no scoped axe violations or page overflow and confirmed Escape/focus restoration. A real PNG decoded in the browser using labelled sample transport. Full-suite rerun and final backend reviews are pending; no new production deployment has occurred.
- 7 September: frontend specification and quality reviews now pass with no remaining findings. Stable full web run passed all 153 files / 976 tests using two workers; the earlier stale six-tab assertion was updated for Documents, and the realistic focus-then-click regression passed. Canonical vendor review status labels now match VALIDATED/REJECTED/EXPIRED/SUBMITTED/SUPERSEDED. Backend Task 2 committed as `01f44002`, but spec review found legacy/work document currency and exact work-form binding gaps plus missing real-PostgreSQL contributor/reuse/query-plan proof. Those corrections and the schema ownership documentation regression are in progress; Task 2 and end-to-end release remain open.
- 7 September: native Chromium PDF rendering verified with a generated one-page sample file through the actual protected-preview component. Visual inspection confirmed readable page content. Disabled browser PDF support exposed a blank-viewer case; a regression test and explicit download fallback now cover it. Focused preview/copy tests (7) and evidence build passed, and the disabled-browser fallback was verified in the browser. No production evidence was used.
