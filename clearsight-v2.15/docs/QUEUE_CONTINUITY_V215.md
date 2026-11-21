# Inbox and work queue continuity — V2.15

21 November 2026 · Source implementation notes

## Changes

The inbox now separates its selected mailbox (All notifications, Unread, Needs action, Mentions, Snoozed or Archived) from category, priority, source, search and deadline refinements. Active refinements are visible above the results. **Clear filters** removes those refinements while retaining the selected mailbox, current sorting preference, acting person and organizational scope. The older complete-reset command remains available to existing integrations.

An empty filtered result now says **No notifications match these filters** and offers the bounded reset. Unfiltered empty states describe the actual mailbox. In particular, having no unread notifications no longer implies that source work is complete; the view links to Needs action.

A concise result-count status replaces the live region on the full notification list. Search refresh leaves the search input in place, treats surrounding whitespace consistently, and evaluates the full recipient filter once for the list/bulk/count update. Existing unread chrome retains its own source calculation.

Select-all exposes the selected, partially selected and empty states. Its native `indeterminate` property is synchronized after initial render and checkbox updates; empty-list selection is disabled. Selection is still pruned against the current visible, recipient-scoped result before bulk actions.

Task links opened from Work now carry a bounded queue context. **Back to work** restores accountability, lane, work type and search. Context consists only of four encoded fields; it never accepts a return URL, changes scope, grants access or bypasses source availability. Unknown owner/lane values fall back to the normal queue defaults. Work type is capped at 120 characters and search at 200 characters. Direct task links without queue context retain the ordinary Work destination.

## Files and boundaries

Edited existing `assets/inbox.js`, `assets/inbox.css` and `assets/governance-interactions.js`; no runtime module or alternate data store was added. Notification receipts, read/acknowledgement/archive commands, source resolution, independent reviews and scoped work authorization remain authoritative.

`tests/queue_continuity_v215_checks.js` covers six behavioral cases: filter/empty-state accuracy and escaping; bounded reset with receipt preservation; mixed/empty/stale selection; single-pass result refresh and announcement markup; queue-context round trip; and malformed return-context isolation. All six pass. The inherited 19 governance-interaction checks and 17 inbox-model checks also pass.

These are actual-source checks with a controlled DOM shell. Native checkbox rendering, keyboard/screen-reader behavior and visual acceptance remain unverified under the existing browser execution restriction.
