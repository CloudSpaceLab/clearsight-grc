# Test report — V2.15

21 November 2026 · Actual application sources with substituted browser/DOM shells

## Result

**470 model/workflow checks and 126 route/action checks passed, with zero failures.** The release run rebuilt 80 module entry points and the standalone, compiled 43 JavaScript modules plus both standalone script blocks, and verified 6,400 linked asset references.

The standalone is **2,214,700 bytes** and has no external JavaScript or stylesheet dependencies. My workspace remains the default entry. The route population is unchanged: 79 registered views, mapped in `docs/UI_COVERAGE_V215.json`. Presentation ownership is source coverage, not browser visual acceptance.

Run `python3 tests/run_release_checks.py` with Python 3 and modern Node to reproduce. This delivery used Node v24.22.3. The current route checker is `tests/routes_v215_checks.js`; `tests/validation-summary.json` is the authoritative machine-readable inventory.

## Suite inventory

| Suite | Passed | Failed |
| --- | ---: | ---: |
| Branch KRI source model | 11 | 0 |
| KRI review and action history | 7 | 0 |
| Roles, access and escalation model | 14 | 0 |
| Detailed Inbox model | 17 | 0 |
| Board report models | 11 | 0 |
| Loss capture and worksheet import | 12 | 0 |
| Independent loss edge review | 10 | 0 |
| Controls, RCSA, vendor lifecycle and risk editing | 14 | 0 |
| Forms, Documents, privacy, Work and source handoffs | 25 | 0 |
| Loss reporting and captured context | 18 | 0 |
| Named assessment identity, history and review | 22 | 0 |
| Unified Documents and contextual decisions | 12 | 0 |
| Vendor intake through reviewed activation | 15 | 0 |
| Independent V2.10 crossflow/reporting under V2.11 | 16 | 0 |
| Shared save, recovery, search and correction interactions | 15 | 0 |
| Assurance, Documents and vendor interactions | 17 | 0 |
| Governance, queues and file approval interactions | 19 | 0 |
| KRI, loss, RCSA, control and report interactions | 16 | 0 |
| Privacy and local monitoring interactions | 10 | 0 |
| Role-aware workspace and actual scoped actor | 20 | 0 |
| Complete reporting populations and portable scope | 24 | 0 |
| Document preview, lifecycle and external capture | 23 | 0 |
| Isolated external assessment protocol and reviewed handoff | 32 | 0 |
| Display preferences without draft or scope loss | 8 | 0 |
| Reporting presentation and reversible print disclosures | 8 | 0 |
| Workspace queue and governance presentation preservation | 9 | 0 |
| Form reading and isolated recipient presentation preservation | 6 | 0 |
| Framework/program/editor presentation preservation | 3 | 0 |
| Risk/control/register source presentation preservation | 8 | 0 |
| Privacy and monitoring presentation preservation | 7 | 0 |
| Search, drawer and shared utility interactions | 11 | 0 |
| Document filter continuity and bounded version comparison | 7 | 0 |
| Inbox refinements and work return continuity | 6 | 0 |
| Applied reporting filters and precise recovery | 7 | 0 |
| Navigation, modal and keyboard continuity | 10 | 0 |

The 126 route/action checks cover 79 entry routes, 10 report perspectives, 25 interaction variants, 10 role-entry views and two explicitly blocked boundaries.

## V2.15 behavioral evidence

The **30 new checks** cover concrete interaction failures:

- **Documents, 7:** folder/archive/format/status/sort persistence before render; exact-version title and return context; default restoration; late-line and long-line differences; bounded/escaped output; text and Word omissions; asynchronous source rechecks. Stored Word fixtures differ at paragraph 2,501 and verify that a matching preview is accompanied by the explicit 2,500-paragraph limit.
- **Queues, 6:** truthful filtered/mailbox empty states, escaped filter labels, mailbox/sort/recipient/scope-preserving reset, mixed/empty select-all, selection pruning, one filtered population per refresh, concise announcements and bounded encoded work return context.
- **Reporting, 7:** unchanged applied populations while editing, guarded output commands, exact in-place revert and permission preservation, precise loss/response resets, invalid response recovery, grouping protection and frozen output stability.
- **Shared navigation, 10:** logical focus after source reconstruction, unavailable/ambiguous/hidden targets, actor-bound replacement matching, continuation dialogs, Escape lifecycle, busy Back protection including no hash, explicit dirty discard, search direction/composition, route/inline focus and revisitable correction links.

All **440 inherited checks** pass. Existing KRI, loss/receipt/reversal, permissions, named responses, independent reviews, vendor activation, exact-version Documents, external exchange and frozen-report behavior remain covered.

## Reviewed fixture adjustments

The appearance-persistence fixture now checks V2.14 and V2.13 migration while retaining V2.12 coverage. The V2.13 workspace fixture previously required a work detail URL to end immediately after the task ID; it now checks that same task ID plus the newly added owner/lane/type/search return parameters. Original fixtures are retained under `tests/history/`. The final suite passes with both original behavior and the expanded navigation contract asserted.

## Independent review

Source review identified and corrected three shared interaction edge cases: a menu opener hidden by desktop resize, Back to the initial no-hash entry during an asynchronous save, and IME-triggered search announcements. Document comparison review identified the Word paragraph omission and added explicit reader metadata/copy. Queue review confirmed that return data retains queue filters while organizational scope remains the current global selection.

The implementation edits existing modules. It adds no framework, runtime module, remote asset, polling loop, observer or competing business store. New styles reuse established semantic tokens and touch control rules.

## Browser acceptance boundary

No browser retry or workaround was attempted after the previously observed process/socket and local-file access restrictions. These checks do **not** establish rendered appearance, native focus/history/cancellation, actual IME behavior, screen-reader announcements, touch ergonomics, text zoom, storage, downloads, native document viewing or print pagination.

Review supported browsers at 320/390/768/1024/1440 px, both themes/densities and short viewports. Include document filter event ordering, exact-version return links, report Apply/Revert and output states, partial inbox selection, work return context, desktop resize while search is open, source-page rebuild after save, and Back during a pending operation. Historical screenshots are not V2.15 visual evidence.

## Reproducibility and integrity

`build.py` reconstructs the deliverables. `MANIFEST.json` covers release files except itself and Python bytecode caches; `python3 verify_reference.py` validates an untouched delivery. Packaging checks archive CRCs and confirms that the standalone inside the ZIP matches the separate HTML. Rebuilding or rerunning tests can change covered files.
