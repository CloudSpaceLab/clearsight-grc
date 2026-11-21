# Inbox and notification flows — V2.8

## Purpose

The inbox gathers mentions, program activity, data quality defects, document/evidence expiry, approaching or overdue deadlines, approval requests and escalations. Every notification preserves a source link and location. Inbox state belongs to an individual recipient; it does not change a risk, evidence decision, assessment or approval.

## Implemented interactions

- All, unread, needs-action, mentions, snoozed and archived views; category, priority, source and text filters.
- Exact source context, accountable owner, due date, priority and a practical next step on the detail page.
- Read/unread, archive/restore, snooze/unsnooze, acknowledgement and selection-based bulk actions.
- Snoozes resume when their time is reached and the application next renders. Dates use the current clock; monitored source deadlines use the workspace snapshot. Quiet-hour and digest values are stored as Africa/Lagos time.
- A source-linked follow-up creates a normal remediation task with an owner, due date, location and required outcome. Duplicate follow-up creation returns the user to the existing work queue.
- Mentions and replies form local discussions. A new mention targets a known colleague and a scoped source record. A reply can mention an additional colleague; recipients get their own notification receipts. Resolving a discussion requires an outcome and does not close the source record.
- Personal notification preferences apply to future notifications. Mandatory approval, escalation, overdue and critical in-app notifications cannot be suppressed. Existing history remains visible.
- Sidebar Inbox entry and header unread badge use the selected recipient and organizational scope.

## Identity, source and scope

`recActor()` / `S.governance.actorId` is the shared demo actor. The inbox preview selector switches that same identity. Notifications are visible only to their explicit recipients and the active organizational scope. Invalid stored scope blocks the inbox. Document-related notification text is withheld for restricted documents when the selected actor is not the owner; this is presentation simulation, not server authorization.

Source facts are derived from the same records used in the existing work, program, evidence and approval workspaces. Work notifications open the existing action decision dialog, keeping completion in the original workflow. KRI review notifications pin their submission revision; a completed review resolves that request. A changes-requested notice stays actionable until resubmission. Accepted-return exception notifications pin the exact remediation task and stay open until that task is completed, even though the return itself is approved. Acknowledgement and archive do not resolve a defect. Completion is read from its owning domain record. Unavailable source records do not get invented replacement links.

## Integration

Load `assets/inbox.js` after records, documents and usability, before the final startup render. Routes and the wrapper self-register:

- `inbox`
- `notification?id=<notification ID>`
- `notification-preferences`

The public event entry point is:

```js
CS_V28_EVENTS.emit({
  id: 'stable-event-id',
  type: 'quality', // mention, program, quality, document, deadline, approval, escalation
  title: 'Branch return needs correction',
  body: 'Describe the specific issue and expected response.',
  sourceRoute: 'kri-return?id=...',
  sourceId: '...',
  sourceRevision: 2, // for a versioned KRI workflow
  sourceTaskId: 'ACT-...', // optional exact remediation task
  sourceState: 'Changes requested', // optional source state at emission
  orgScopeId: 'B-IK',
  severity: 'High', // Critical, High, Normal, Low
  recipient: 'USR-01', // known user ID or exact known name
  actor: 'USR-03',
  due: '2026-11-20',
  createdAt: new Date().toISOString()
});
```

Use stable event IDs for deduplication. Multiple recipients can be passed as `recipientIds`. Source typed references can be passed as `{subject:{type:'kriReturn',id:'...'}}`. Source routes must be registered application routes. The event API never sends network messages.

`inboxUnreadCount()`, `CS_V28_EVENTS.events()` and `CLEAR_SIGHT_INBOX` provide bounded inspection hooks for verification. State lives under `S.notificationsV28`. Retained data follows the existing browser-local persistence behavior.

## Production boundaries

This package supplies working local UI flows. Real identity and source authorization, durable queues, email/push, immutable audit history, recipient entitlement checks, transactionality, concurrency, retention and background scheduling require backend integration. Preferences configure intended delivery behavior; they do not claim that email or jobs are connected.

## Validation

`node tests/inbox_v28_model_checks.js` loads the full integrated module stack and checks 17 notification and cross-module invariants, including the actual KRI submission, review and correction commands. DOM shell and form objects are mocked; production domain logic is not replaced. `tests/inbox_v28_checks.py` contains the browser workflow and responsive checks. The local Chromium launch was blocked by the environment’s socket restriction during the inbox agent’s run, so those browser checks are not reported as passed by this module.
