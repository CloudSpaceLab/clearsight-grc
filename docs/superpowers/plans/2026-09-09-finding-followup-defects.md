# Finding follow-up defect repair

> **For agentic workers:** Use subagent-driven-development for isolated parser implementation and independent specification/quality review. Continue through verification and the authorized release.

**Goal:** Repair the four reproduced historical-branch failures without regressing current source fidelity or response amendments.

**Architecture:** Build on main `2e35415b`. Keep ordinary V2 row proposals as the default. Choose one source assessment before generating an optional five-field-per-finding proposal, giving it an ordinary, source-scoped proposal receipt. Do not create nested parent/child acceptance or import the historical request-status mutation. No dependency or parallel workflow.

**Tech stack:** Go, PostgreSQL, React/TypeScript, existing test/render/deployment pipeline.

## Decision brief and constraints

Imports retains its existing source/proposal view. A secondary “Prepare finding follow-up” section lists recognized assessments, with source vendor/service/date and finding count. The author selects one, generates and previews its questions, confirms the historical assessment context, then creates a draft through ordinary acceptance. Shared SelectField, CheckboxField, Notice and Button components stack on mobile; existing bounded review scrolling and full source quotes remain available. No automatic recipient selection, vendor matching, scoring, signature, sending, approval or finding closure.

Each follow-up proposal is independent and tied directly to the exact source version, digest and assessment ID. Rejection applies to that proposal, not every other assessment in the source. This replaces the old branch's parent/child design: no nested acceptance, parent rejection race or global field limit is needed. Source changes and current authority remain checked by the existing acceptance transaction. Retries reuse the same assessment proposal and draft.

Rows must be linked by an explicit assessment identifier (including values propagated from declared vertical merges). Process numbered boundary rows even without finding text. Unmerged missing identifiers cannot silently inherit a vendor; ambiguous registers retain the ordinary row proposal and require source correction. Responsibility is per finding; absent responsibility remains unknown. Long context is shortened only in field help with an explicit warning; full source remains readable.

## 1. Structured generator

Files: `internal/documentimport/finding_followup.go`, `finding_followup_test.go`, `form_template_proposal.go`.

- [x] Reproduce delimiter-like cell text, standalone numbered headings and differing finding owners with structured XLSX fixtures before implementation.
- [x] Add `FindingAssessment` metadata and `FindingFollowUpAssessments(Document) ([]FindingAssessment, error)`. Use only `Elements.Values`, exact sheet/row anchors and explicit assessment linkage.
- [x] Add `ProposeFindingFollowUp(Document, assessmentID, ProposalPolicy)`. Generate response, action/explanation, remediation owner, proposed completion date and supporting evidence fields. Require complete extraction, stable identities, one complete selected assessment and per-assessment limits. Preserve historical limitations and row-specific responsibility.
- [x] Keep `ProposeFormTemplate` V2 output behavior; expose eligible assessment choices in retained proposal provenance. Invalid grouping must not break ordinary proposals.
- [x] Run `go test -p 1 ./internal/documentimport -count=1`.

## 2. Ordinary proposal receipt integration

Files: `internal/monitoring/form_proposal*.go`, next free migration, related tests and API contract.

- [x] Test opt-in generation, default behavior, per-assessment receipt identity, concurrent/repeated generation and acceptance, unknown/partial selection, missing confirmation, source change, rejected proposal and denied/cross-entity commands.
- [x] Persist `finding_assessment_id` (empty for historical/general proposals) and include it in exact source identity. Use the existing generation worker and atomic acceptance; prohibit appending a grouped follow-up to a base form and require all generated fields plus explicit assessment confirmation.
- [x] Give separate assessments distinct draft codes. Preserve original source SHA and scope checks, events/outbox and independent approval.
- [ ] Run focused unit and real PostgreSQL tests, migration application/rerun and tagged compile; do not count skipped tests as DB verification.

## 3. UI and amendment protection

Files: `web/src/components/forms/FormProposalReview.tsx`, related API/types/tests, vendor-release evidence fixtures/script, `internal/evidence/response_workspace_same_respondent_test.go` and PostgreSQL response tests.

- [x] Test and implement explicit assessment selection, generation progress/recovery, context confirmation and complete-group acceptance. Source quotes and default row selection remain unchanged.
- [x] Verify the same respondent can submit an amendment while open, creating a second immutable revision. Add request provenance/history and closed/expired/revoked protections where coverage is missing. Do not import the broken status mutation merely to fix it again.
- [x] Render light/dark desktop/mobile states, inspect selection/confirmation/preview/acceptance/error recovery and check axe/overflow. Retain existing baseline and capture the new optional flow.
- [x] Run copy-quality, affected workflows, TypeScript and production/evidence builds.

## 4. Review and release

- [x] Specification review followed by independent code-quality review; fix findings and rerun affected checks.
- [ ] Synchronize product, architecture, ledger and acceptance evidence. Update #80 only for proven items; keep #200 closed.
- [ ] Commit, push focused PR, pass exact-head CI/UI, merge and verify deployment of the exact main revision. Preserve the historical branch and unrelated worktrees.
