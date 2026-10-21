# Notifications workspace v2 adaptation

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
