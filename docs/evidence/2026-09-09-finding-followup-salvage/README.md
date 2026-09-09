# Selective follow-up port verification

The branch review at `docs/reviews/2026-09-09-finding-followup-branch-salvage.md` records exact source revisions and excluded behavior. Implementation began at main `676995b1`; main `59041af2` (PR #212) was merged cleanly before publication.

- The complete Go suite passed with `go test -p 1 ./...` before integration of #212. Earlier broad attempts hit Windows paging/disk exhaustion, rather than a changed-code assertion.
- Focused documentimport and monitoring tests passed before/after implementation. The new tests first reproduced lost vertical merged context, repeated first-row source quotes and unhandled competing completion.
- After integrating main `59041af2`, all three affected Go packages passed uncached; 29 proposal, preview, document-preview and copy tests passed, as did 52 runtime/UI contract checks. TypeScript and the evidence build passed again, and the four rendered states were recaptured from the integrated source.
- Receipt mismatch probes cover tenant, entity, proposal ID, source revision and digest. The same-respondent amendment regression passes on main and fails on the reviewed feature branch.
- TypeScript and the production/evidence builds passed locally. Focused proposal/copy tests passed (11 tests). A broad frontend attempt exhausted host resources; a subsequent run was stopped when main integration changed its source snapshot. Neither is claimed as a successful full-suite receipt. CI owns full frontend verification for the published revision.
- `after/manifest.json` records four rendered states at 1440px/390px in light/dark: distinct row quotes, no page errors or horizontal overflow, selection removes/restores the expected preview question, and the draft action remains enabled/reachable. Axe found no violations in the proposal component after correcting review-note contrast and heading order. This is component evidence, not whole-product accessibility conformance.
- The attempted before-state build was interrupted by disk exhaustion. The pre-fix unit failure records repeated first-row quotes. No successful before-render is claimed.

The screenshots use synthetic fixtures rendered through the real components; they are verification evidence, not a new mockup or a live bank submission. The evidence application remains outside the customer runtime import graph. CI repeats the four source-row checks and retains screenshots as workflow artifacts.

To reproduce branch defects, create a disposable checkout of feature revision `6ebe0746ec9ba5f64b6a172d634bca70a219bc9b`. Copy `documentimport-probes.go.txt` into its `internal/documentimport/salvage_audit_test.go` and `evidence-probe.go.txt` into `internal/evidence/salvage_audit_test.go`, then run the `TestSalvageAudit` commands in the review. These probes intentionally fail on that historical branch and are not compiled into the release suite.
