# Response-browser verification

Decision: [response-browser brief](../design/2026-10-05-response-browser.md). The supplied Archer image informed comparable results and a desktop filter rail, without treating scoring coverage as review progress or document validity. Shared controls keep theme, keyboard and mobile behavior consistent.

Baseline: the supplied Program screenshot and [existing desktop render](screenshots/program-follow-up/desktop-responses.png); the Forms structure inventory is recorded in the brief.

## Rendered evidence

The isolated `program-responses` fixture supplies four sample submissions, stored scoring facts and response-scoped documents. It is excluded from the production import graph. Captures use `web/scripts/capture-response-browser.mjs` against the built evidence preview, not a live bank database.

- [Program desktop](screenshots/response-browser/after/program-light-1440.png) and [Forms desktop](screenshots/response-browser/after/forms-light-1440.png).
- [Program mobile](screenshots/response-browser/after/program-light-390.png) and [Forms 320px dark](screenshots/response-browser/after/forms-dark-320.png).
- [Column changes](screenshots/response-browser/after/forms-columns.png), [empty search](screenshots/response-browser/after/forms-filtered-empty.png), [invalid dates](screenshots/response-browser/after/program-invalid-date.png), [load failure](screenshots/response-browser/after/forms-error.png) and [response review](screenshots/response-browser/after/program-review.png).
- [Viewport receipt](screenshots/response-browser/after/receipt.json): Program and Forms at 1440px, 390px and 320px in light/dark themes; all twelve checks report no document-level horizontal overflow.

Desktop refinements remain visible beside results. Mobile replaces the rail with a disclosure and results with cards; active chips and reset remain outside the disclosure. Optional columns preserve form identity and review. Keyboard Space toggles columns. Missing scores stay unavailable, and loading/error/invalid-query counts do not claim zero. Inspection identified duplicate date-error announcements; the result-region announcement now appears only when the field error is hidden by the collapsed rail.

Independent review reproduced a stuck loading state after clicking an already-active Forms sort header. The refresh effect now tracks each query revision, including a repeated sort command. Its regression test failed before the fix and passes afterward; desktop captures repeat the active sort before recording results.

## Verification and limitations

- 120 web tests passed across thirteen suites: DataTable, component gallery, ProgramResponsesPanel, ProgramRecordWorkspace, ResponsesView, ResponseBrowser, API serialization, copy quality and the existing answer/assessment/score/review suites. Tests cover stable Program scope, stale-page rejection, column changes, reset, date recovery, repeated active sorting and semantic accessibility.
- Typecheck, customer production build and isolated evidence build passed. The production build retains the existing large-chunk warning.
- All 29 runtime-boundary, UI-contract and evidence-manifest checks passed.
- `go test -tags postgres ./internal/evidence ./internal/httpapi` passed. The focused untagged response suites also passed.
- The PostgreSQL integration title-search test compiles, but its live database test was skipped because `TEST_DATABASE_URL` is not configured. Live query-plan/performance and database collation behavior were not revalidated. Literal escaping, entity isolation and pre-pagination search are covered by the added integration assertions for the next configured run.

No scoring, permission, workflow-command, delivery, saved-view or export behavior was added. Column preferences are session-local. No push, deployment or external email was performed.
