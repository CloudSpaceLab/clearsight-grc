# Notifications workspace v2 adaptation

Tracking: #462. Status: **locally implemented and reviewed; publication pending**. This is partial V2 layout integration; the whole-screen acceptance register remains **0/19**.

## Decision and boundary

Replace the notification bell's narrow sheet with a full Notifications workspace on the existing application router. The immutable October preview is the layout authority: `docs/design/references/clearsight-2026-10/source/clearsight-notifications-activity-designs/notifications-activity-preview.html`, SHA-256 `18a5fa9e4ef0b306da152444e3c2223fc6fe8ffe359c00e0002c5098609a03bc`.

The workspace uses the established shared menu and context bar, a heading, Inbox / Activity / Preferences local navigation, an inbox toolbar, a selectable notification list and persistent selected-record panel. At 1440 and 1280 pixels the detail panel stays beside the list; at 1024 it narrows; at 768 and 390 it follows the list with an explicit return action. Mobile filters disclose separately. Activity retains its own history-and-context composition, with context preceding history below 900 pixels. Preferences retains its in-app information and email-settings columns, stacking below 900 pixels.

This tranche is based on PR #580's head `e6449c5149c52cdafa8e4c99140c3763ef980aee`. It does not change the Work queue/search behavior owned by PR #574; `App.tsx` receives Notifications routing and a stale-context-response guard. No backend migration, alternative queue, notification datastore or workflow is introduced.

## Data and command contracts

- One scope-keyed inbox owner serves both the bell and workspace. The key includes the verified tenant, legal entity and actor. Changing that scope discards the former view and invalidates in-flight responses.
- `GET /api/v1/notifications` owns bounded cursor paging, unread filtering, timestamps and unread totals. Loaded rows are not a total notification population. Any local find operation explicitly searches loaded rows only.
- `POST /api/v1/notifications/{id}/read` is the existing scoped, idempotent read-receipt command. Selection alone does not issue it. Read receipt success never means workflow completion, approval or verification. Only safe existing application paths are navigable; an invalid path does not block recording a read receipt.
- Search across the entire inbox, category/date/group filters, needs-action classification, snooze and mark-unread are not supplied by this API. The adaptation must not fabricate these capabilities or infer action urgency from unread state.
- Selected context displays actual notification type, subject, occurrence/read times and current legal entity. Owner, due date, source history and workflow status are not inferred from a notification summary. The existing record destination remains the authority for current work and its permitted action.
- Activity uses the existing system-activity reader, gated by `platform_operations_read`, with a bounded server period and exact current legal entity. It remains an audit history rather than a personal inbox. Unauthorized users see an explicit restricted state without a history request; no export capability is introduced.
- Email preferences reuse the actor-and-tenant preference contract, including `expected_version`, timezone, daily digest, quiet hours and required critical delivery. In-app category switches and weekly digests are not fabricated. Fix the existing time-input parsing defect while preserving optimistic-concurrency failure behavior.

## Verification and acceptance

Verify scope/filter races, pagination failure and retry, duplicate read commands, unknown/unavailable unread count, exact safe deep links, permission-denied Activity, versioned preference updates, and retained local navigation. Review source/runtime anatomy at 1440, 1280, 1024, 768 and 390 pixels in both themes, mobile selection/return/filter/menu behavior and accessibility. Preserve all 96 original source files and record the exact implementation revision and local artifacts. Remote CI is a separate publication gate.

This record defines a source layout and interaction tranche, not operational bank acceptance. The October acceptance register remains 0/19 until its separate required evidence is recorded. No Notifications PR, merge or deployment is recorded.

## Implementation and scoped review — 21 October 2026

Runtime checkpoint: **`c2d3593ceaebb509e8e3d1659b80000f6a88c5c6`**, based on `e6449c5149c52cdafa8e4c99140c3763ef980aee`. The following evidence/documentation commit does not change that runtime. The [machine-readable receipt](2026-10-21-notifications-workspace-evidence.json) binds the exact runtime, build assets, original/rendered images, capture scripts, manifests, test logs and review limitations.

**Named reviewers:** ChatGPT `/root/notifications_review` for the original/runtime visual matrix, supplemental states and inbox race review; ChatGPT `/root` for implementation integration, final representative renders and verification; `/root/notifications_evidence` for executable browser evidence; `/root/notifications_provenance` for immutable-source, baseline and ownership verification. These are scoped AI reviews, not representative bank-user acceptance.

The six before-state captures use an isolated, unchanged archive of e644 at 1440, 768 and 390 pixels, in light/dark. Its former 560px drawer becomes full width on phone, places preferences before the list, and supplies no three-view menu or persistent context pane. The browser-only synthetic read fixture and all six archived UI/API source identities are retained. The final matrix contains all three new views at 1440, 1280, 1024, 768 and 390 pixels in both themes, with fifteen original source renders and thirty source/runtime pairs. All source and image hashes are retained in the receipt.

| Review area | Result and adaptation |
| --- | --- |
| Heading and menus | Full Notifications heading/action block and Inbox / Activity / Preferences navigation. The bell enters the existing route; Work & Approvals is the selected shared-menu parent. The canonical shared menu order, authorized Settings footer and mobile disclosure are preserved. |
| Inbox composition | Supported All / Unread controls, disclosed loaded-row search, selectable list, persistent record context and explicit read/open actions. Desktop keeps the list/context proportions; 768/390 stack context after the list with selection focus and an explicit return action. |
| Activity composition | Dated timeline with recorded actor, action, outcome, record, source and event reference; permission-gated legal-entity history and fixed server date bounds. Period and refresh controls share one row. Scope and the absent-version limitation are stated in the context rail instead of repeated on every event. |
| Preferences composition | In-app information and the existing versioned email settings occupy the source columns and stack in source order. Daily digest, timezone, quiet hours and critical-delivery limits are real capabilities. Unsaved settings survive local tab navigation. |
| First content and responsive density | All matrix frames expose the selected local tab and first useful content without horizontal page overflow. Mobile Inbox retains readable summaries and 44px controls. At 768, Activity's four actual context facts wrap to two rows, placing the first event below the initial viewport; at 390 both source and runtime put context before history. Preferences keeps information before email controls on phone. These are documented density adaptations, not pixel-identical reproduction. |
| Keyboard and touch | Local-tab navigation, retained preference drafts, mobile record selection/return, filter disclosure and menu behavior pass. Required controls meet the 44px target checks. |
| Empty, failed and denied states | Empty/unavailable inbox, failed read, failed older-page load with retained rows, unavailable/empty/denied Activity, foreign-entity rejection, preference conflict and invalid record destinations have independent cases. State screenshots show the actual notice or disabled action within the captured viewport. |
| Data and commands | The bell and workspace use one owner. Server unread totals remain distinct from loaded-row counts. Selection does not issue a read command. Successful read receipts use a fresh server total without guessing a decrement. Preferences retain expected-version concurrency and actual minute values; Activity makes no request without its capability. |

The visual review corrected undefined radius/type tokens, small headings, two 43px checkbox targets, repeated Activity metadata, separated label/value alignment and unpadded empty context. The final rerender covers those corrections.

### Behavior corrections retained in this tranche

- Discard responses from obsolete tenant/entity/actor owners, filters and presentations. A reproduced stale context response could previously restore a former account view.
- Preserve the newest server `as_of` unread-count snapshot when paging and read-receipt refreshes complete out of order. Both response orders are regression-tested; loaded ordering and cursor semantics remain intact.
- Deduplicate concurrent read commands, preserve loaded rows after an older-page failure, retain selected context after marking an item read under the Unread filter, and keep unsafe record destinations disabled.
- Correct valid `HH:MM` preference parsing and preserve recoverable drafts, version conflicts and scoped/unmounted response cleanup.

### Verification receipt

| Local gate | Final result |
| --- | --- |
| Full web unit suite, including copy-quality and affected workflows | **262 files / 2,096 tests passed** |
| Customer and evidence builds, including TypeScript | Passed |
| UI contracts and customer/evidence import boundary | Passed |
| Notifications browser layout/state/interaction contract | **42/42 passed**: thirty matrix frames and twelve supplemental cases |
| Original/source pairing | **15 source renders / 30 runtime theme pairs**, hashes verified |
| Scoped accessibility and required touch targets | **12/12 passed**, zero automated axe violations, undersized required controls or browser errors |
| Customer bundle budgets | Initial JavaScript **161,862 bytes gzip**; initial CSS **31,970 bytes gzip**; largest JavaScript chunk **564,731 bytes raw**; all existing limits pass |
| Immutable source integrity | **96/96** byte counts and SHA-256 values match; no source or manifest changes |

Local browser evidence uses Chromium153 through the retained Playwright adapter. The pinned CI Chromium140 suite and full remote workflow have not run for this new branch. Automated accessibility PASS does not establish complete conformance: gradient-related axe and supplemental contrast readings remain explicitly incomplete or for review in the machine-readable receipt. Fixtures are synthetic, isolated from the customer import graph and labelled Sample; no live backend command or email delivery is established by these captures.

### Publication and remaining acceptance

Two connected GitHub branch-creation attempts did not return a result and were interrupted. Subsequent read-only checks found no remote Notifications branch or PR. The implementation and reviewed evidence are therefore saved as local commits, with a downloadable patch; no remote CI, merge or deployment is claimed. Existing #578–#580 and #574 remain separate work.

**Scoped result: partial V2 layout adaptation verified.** Publication and remote CI remain pending. Representative authorized backend results, four-actor execution and the production cutover gates require their own evidence. Whole-screen acceptance remains **0/19**.
