# ClearSight V2.15 — Reliable everyday interactions

21 November 2026 · Interactive implementation reference · Synthetic records

## Open the application

Open **clearsight-v2-demo.html**. The standalone starts at My workspace and includes every script, stylesheet and synthetic fixture. Individual module entry points use the adjacent `assets/` folder.

For a consistent browser origin, run `python3 -m http.server 8080` from this folder and open `http://localhost:8080/clearsight-v2-demo.html` (on Windows, `py -3`). Native storage and download behavior depends on the browser. The operating scenario remains **13 November 2026**, separate from release date and real external-request expiry.

## This increment

The existing premium design covers all **79 registered views**. V2.15 improves practical interaction continuity within that system, using the current modules and canonical records.

| Area | Improvement |
| --- | --- |
| Documents | Folder, archive, format, status and sort selections persist before rendering. Title, Preview and Details links keep the exact version and filtered return context. Clean list routes reset omitted filters. |
| Version comparison | Changes beyond line 500 and deep within long lines are discoverable. Displayed excerpts remain bounded; text and Office extraction limits are explicit. |
| Inbox | Visible active filters, accurate mailbox-specific empty states, concise live result counts, mixed select-all state and fewer repeated filter evaluations. Clear filters retains the current mailbox, sort, actor and location. |
| Work queue | Task details carry owner, lane, work type and search back to the queue. The current organizational scope remains authoritative. |
| Reports | Applied filters and temporary edits are clearly distinguished. Apply/Revert controls prevent exports or report preparation from being mistaken for the edited population. Loss and response reset paths preserve the selected location. |
| Shared navigation | Focus returns to the same record control after a save/rebuild, or to the page heading when the control is unavailable. Route transitions, correction links, Escape and search keyboard selection behave consistently. |
| In-flight saves | Back navigation, including a return to the initial URL without a fragment, cannot replace the source context while a modal save is pending. |

The refinements preserve existing permissions, named response history, independent review, exact document versions, loss/currency semantics, external recipient isolation and frozen report snapshots. V2.14 and earlier supported local state remain migration fallbacks. Schema version stays 3.

## Review the changes

1. In Documents, select a folder and format, open a document by its title and use its return link. Compare versions containing late changes and inspect any extraction-limit disclosure.
2. In Inbox, combine search, priority and deadline filters. Compare filtered-empty and genuinely empty mailbox states; select some rows and inspect select-all. Clear filters from Archived or Snoozed.
3. In Work, set owner, lane, type and search, open a task and return to the queue.
4. In Loss insights or Response accountability, edit a filter without applying it. The existing results stay displayed; use Apply view or Revert edits before export, Save live view or Prepare board pack.
5. Check keyboard focus after navigating, saving a dialog that rebuilds its source view, closing a continuation, and revisiting a correction link. Search Arrow Up starts at the last result and respects IME composition.
6. Review light/dark and comfortable/compact layouts with long labels and narrow screens. The existing display preferences remain available in the header and Settings.

## Source and verification

`docs/PREMIUM_DESIGN_V213.md` is the visual-system specification. `docs/UI_COVERAGE_V215.md` and JSON map route presentation ownership; this is source coverage, not browser visual acceptance. Four V2.15 interaction notes describe Documents, queues, reporting filters and shared navigation. Prior release documents and adapted test fixtures remain under history folders.

Run `python3 tests/run_release_checks.py` to rebuild and run the authoritative source suites. Python 3 and modern Node are required; Node 24 was used for delivery. `python3 build.py` only rebuilds the application. `python3 verify_reference.py` verifies an untouched package against its manifest; rebuilding or rerunning tests can change covered files.

Current totals are in `docs/TEST_REPORT.md` and `tests/validation-summary.json`. Checks run the actual JavaScript with substituted browser/DOM shells. Real-browser visual, native history/focus, screen-reader, file-operation and print acceptance remains outstanding under the established execution restriction. Historical screenshots are not evidence for this release.

This is a local implementation reference. Production authentication, durable services, notification delivery and the hosted external portal require server integration. No messages or invitations are sent.
