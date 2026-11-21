# Implementation handoff — V2.15

## Deliverable

`clearsight-v2-demo.html` is self-contained. `assets/` contains readable sources; `build.py` builds the standalone and 80 module entry points. The current route coverage map is `docs/UI_COVERAGE_V215.json`. Package integrity is defined by `MANIFEST.json` and checked by `verify_reference.py`.

## Changes to integrate

| Existing module | Responsibility in this increment |
| --- | --- |
| `documents.js` | Five canonical folder/search/filter handlers persist route preferences before rendering. |
| `document-management.js` | Default restoration, exact title/version/return links, bounded comparison excerpts and explicit text/Word extraction limits. |
| `inbox.js/.css` | Active refinements, mailbox-aware empty states, concise announcements and select-all state. |
| `governance-interactions.js` | Deadline refinement integration and bounded work queue return parameters. |
| `scoped-analytics.js/.css` | Applied versus edited filter state, Apply/Revert and reset, guarded report outputs and precise response recovery. |
| `interaction-core.js` | Stable focus identities, shared Escape lifecycle, in-flight hash protection, search keyboard correction and revisitable correction focus. |
| `premium-utilities.js` | Final-render focus coordination and IME-aware search announcement. |

Load order is unchanged. The state key advances to `clearsight-v2.15-demo-20261121`; V2.14 is the newest fallback. Canonical schema remains 3. All 43 JavaScript modules and 37 CSS modules remain existing modules.

## Behavioral contracts

Report filter inputs hold temporary edits; the applied URL remains the population authority until Apply. Revert restores the exact displayed filter values and each output control's prior disabled state. Existing command permission checks still run. Frozen reports never receive live filter controls. Loss reset preserves organization/KRI context and uses stated period defaults; response clear removes only its four response filters.

Document list navigation records preferences before rendering can restore them. Title links pin the exact version; opening a record does not substitute its newest version later. Comparison is over reader output, not a guarantee of complete original-file equivalence. Retain both extraction-limit and displayed-excerpt disclosures, checksums, authorization and asynchronous context rechecks.

Inbox selection remains within the current recipient and filtered population. Refinement reset preserves mailbox and sort. Work return data carries owner, lane, type and search only; it never restores or widens scope. The current global organizational selection governs the returned queue.

Focus references hold transient DOM identity, route, scope and actor only. They do not copy domain state. Replacement targets must be unique, visible, enabled and in the same context; an unavailable return target falls back to the page heading. Queued focus work must not steal focus from a later workflow, user action or route. Keep asynchronous-save navigation protection, including empty hash transitions. Opening a continuation dialog retains the original workspace return target.

## Verification

Use `python3 tests/run_release_checks.py`. The authoritative inventory is `tests/validation-summary.json`; the current route checker is `tests/routes_v215_checks.js`. Four new suites exercise practical continuity risks in Documents, queues, report filters and navigation. Archived originals explain the adapted state-migration fixture and work-link assertion; the new assertion checks both the original task ID and return parameters.

Actual browser appearance, native history/focus, screen readers and print remain acceptance tasks under the existing restriction. Review 320/390/768/1024/1440 px, light/dark, both densities, short viewports and text zoom. Include desktop resize while search is open, unsaved/busy dialogs, named review and exact-version file workflows.

## Production boundary

This remains a local synthetic implementation reference. Production services must bind real identity/tenant, enforce exact scopes and independence, retain immutable sources/history, handle concurrent writes and implement durable Documents, safe previews, external authentication and notification delivery. No deployment or external message delivery is represented by this release.
