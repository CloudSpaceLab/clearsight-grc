# Vendor workflow release implementation plan

> For agentic workers: execute the bounded tasks below with the existing verification and parallel-agent skills. The user has authorized merge and deployment to main.

**Goal:** Release evidence reconciliation and the copy/contrast corrections while removing the confirmed spreadsheet and onboarding dead ends.

**Architecture:** Retain governed form revisions, source-linked proposals, protected vendor submissions, independent evidence review and existing authority policies. Spreadsheet content creates an editable draft, never a legal conclusion, vendor identity match or approved requirement set. Existing saved templates remain unchanged.

**Tech stack:** Go, PostgreSQL 18, React, TypeScript, Vitest, browser evidence and the existing GitHub deployment pipeline.

## Decision brief

The two user-supplied workbooks contain row-oriented requirements and historical findings. Column-heading questions are unsuitable. Preserve row context and anchors, with explicit review before a form can be used. Do not infer that every NDPA item applies to a vendor or that a past register status is current.

Vendor creation must require an explicit criticality and privacy-role choice rather than silently asserting Standard/None. The starter must allow the respondent to declare missing assurance with an explanation. A declaration does not satisfy a document check or approve a relationship. Organizations can edit questions, applicability, document constraints, assessment rules and routing through governed revisions; approved and sent versions remain intact.

Use the established field, status, tab and disclosure components. No new palette or density mode. Required renders: new vendor with unset classification, missing assurance, available assurance, imported row proposal and vendor evidence checklist; narrow and desktop widths, light and dark. Preserve earlier audit baseline evidence.

## Tasks

- [x] Verify prior changes, commit them and integrate the newer main navigation, response and policy changes; preserve both submission truth and collection receipt semantics.
- [x] Move the unreleased reconciliation migration to 000086 without rewriting main's migration 000085.
- [x] Add failing spreadsheet extraction/proposal tests; implement source-row drafts while retaining generic table support. Files: `internal/documentimport` and dedicated acceptance documentation.
- [x] Add failing onboarding tests; remove unsupported defaults and add an explicit missing-assurance path. Files: `web/src/components/VendorsWorkspace.tsx`, its tests, `web/src/vendorDueDiligenceForm.ts` and capture contract tests.
- [x] Verify isolated PostgreSQL migrations and serial transaction tests, full Go/frontend tests, copy regression, production/evidence builds and rendered affected workflows.
- [ ] Commit integrated fixes, push the task branch, create and merge its PR after CI, wait for main CI and the normal deployment workflow, and verify the exact deployed revision and hosted flow.

## Release constraints

Do not send real invitations or copy customer workbook contents into fixtures. Use synthetic equivalents for regression tests. Clearly distinguish received, reviewed, missing and accepted evidence; preserve mandatory blockers and governed conditional conclusions. No claim of legal compliance follows from an upload or imported checklist.
