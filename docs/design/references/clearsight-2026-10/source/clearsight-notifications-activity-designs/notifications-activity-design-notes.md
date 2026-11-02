# 15 · Notifications & Activity

## Design boundary

ClearSight's notification center is an actor-scoped inbox, not another dashboard. It complements **Work & Approvals** rather than duplicating work assignment, evidence authorization, or governing decisions.

- **Inbox**: Needs action / Unread / All with search, category, period and authorized group filters.
- **Context**: Selected record shows exact origin, owner, source, decision needed and related activity. Reading is never equivalent to approving, closing or verifying.
- **Activity**: Dated audit trail, distinct from personal notifications; present actor, event, object, version, scope and outcome separately.
- **Preferences**: Channel and digest choices, with non-optional governed escalations still delivered under policy.
- **Mobile**: List first; selecting a record scrolls its context into view with an explicit return path. Avoid a clipped desktop table.

## Interaction contract

The HTML prototype is intentionally local-only. It supports the selected tabs, filters, bulk read, individual read/unread, temporary snooze, details preview, activity period, digest choices, keyboard navigation, Control/Command+K focus, Escape dismiss and reduced motion. None of these operations modify ClearSight backend state. The notification and audit data are fictional sample content.

## Implementation boundaries

1. Compose existing `NotificationCenter`/notification APIs, existing Work/Matter/Program/Third-party/Privacy destinations and existing scope-aware DataTable/Popover/Sheet controls; no new notification truth datastore or universal dashboard engine.
2. Separate: notification delivery/read state; workflow state; assessment result; approved decision; verified outcome. Never make `Mark read` close a Matter or accept evidence.
3. Preserve exact tenant/legal entity and actor permission filtering in backend reads and deep links. Group, category and date filtering must be server-backed for large populations.
4. Include source timestamp, current/correctable/stale/unknown, assigned actor and next permitted action. Email delivery receipts should not imply notifications were read or the work completed.
5. Use versioned, bounded queries and stable cursors; new activity must not silently reorder an active selection. Audit log is append-only and access controlled; no private source content is exposed in a notification toast or digest.
6. Avoid duplicate human tasks: de-duplicate repeated events, collapse noiseless successes, retain high-impact failures and required approvals. All delivered suggestions should have one supported action.
7. Avoid permanent decorative Copilot in the notification workspace; the right context panel uses source-backed facts. AI assistance may later summarize a bounded, authorized record selection without inventing conclusions.

## Render and behavior evidence

- Desktop design: 1880 CSS px / 2× scale = 3760px PNG.
- Responsive tests: 1880, 1280, 1024, 768, and 390 CSS pixels; document width did not exceed viewport.
- Tested: notifications and activity tabs, source filters, read state, local snooze, Record preview + Escape, preference changes, Control+K, mobile filter disclosure, and mobile detail focus.
- Sample records are never evidence of bank compliance, actual alerts or production readiness.
