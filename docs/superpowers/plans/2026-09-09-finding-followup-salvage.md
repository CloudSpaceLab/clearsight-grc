# Finding follow-up branch salvage implementation plan

> **For agentic workers:** Use subagent-driven-development for isolated implementation and review. Preserve the current main workflow and port only the named behavior.

**Goal:** Improve source fidelity and generation recovery while recording the precise remaining work for assessment-specific follow-up drafts.

**Architecture:** Start from fetched main `676995b16b2e58b0dd217ba8845e3320a42a5da6`; compare feature branch `6ebe0746ec9ba5f64b6a172d634bca70a219bc9b`. Keep V2 structured spreadsheet rows, current tenant checks, bank field assessments and response provenance. No whole-branch merge or new runtime dependency.

**Tech stack:** Go extraction/proposal services, React/TypeScript review, existing Go and Vitest suites.

## 1. Source fidelity and generation recovery

- [x] Add synthetic XLSX tests that retain vertical merged values inside their declared range in both `Sections` and `Elements.Values`, stop at the range boundary, and reject malformed/overlapping ranges and conflicting populated values in inherited vertical ranges. Horizontal values remain explicit and unpropagated. Unmerged blank cells must remain blank. Test resource bounds without expanding every coordinate.
- [x] Port and harden `internal/documentimport/xlsx_merges.go`; integrate it into current `xlsx_extractor.go` while preserving structured rows and existing limits. Increment extractor provenance version when behavior changes.
- [x] Add a review test with two source rows on the same worksheet. The second proposed question must quote the second source row. Update `sourceExcerpt` in `web/src/components/forms/FormProposalReview.tsx` to match row anchors as well as sheet/page/cell anchors.
- [x] Add generation race tests in `internal/monitoring`: a competing completion returns the stored result; a still-generating, unreadable or failed receipt must not be presented as successfully generated. Adapt only the `CompleteGeneration` conflict recovery block.
- [x] Run `go test ./internal/documentimport ./internal/monitoring -count=1`, the affected frontend and copy-quality tests, TypeScript and production build. Render the source preview at desktop/mobile sizes in both themes.

## 2. Assessment follow-up readiness review

- [x] Verify branch grouping, selection, child-draft persistence and submission-status behavior against current source and transaction contracts. Record exact reusable files, modifications and blockers in the review receipt.
- [x] Preserve ordinary V2 row proposals. A follow-up mode must consume structured `Elements.Values`, require explicit group confirmation, select one complete assessment per draft, and retain historical/source limitations. It must never infer vendor identity or activate/send a form.
- [ ] Keep grouped mode out of this release until ambiguous grouping, long source context, per-assessment limits, parent/child concurrency, replay scope and PostgreSQL acceptance are covered. Track this as a concrete follow-on issue; keep #200 closed.

## 3. Review and integration

- [x] Review changes independently for specification compliance, then code quality. Fix actionable findings and rerun relevant checks.
- [ ] Record fresh verification and the exact copied/adapted/excluded behavior, commit and open a focused PR. Use the existing CI/release pipeline; do not close broad capability issues on the strength of these importer fixes.
