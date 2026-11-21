# Governance, Work, Inbox and approval interactions — V2.11

## Findings and completed changes

The V2.10 workflows had working underlying records and decisions, but several interactions required users to reconstruct the next step themselves. Work mixed owners and deadlines in an unsorted list; Inbox prioritized recency over unresolved due work; role changes offered no effective-access comparison; escalation drafts could be tested only after saving; and file approval preparation deferred obvious routing or stale-source problems until submission.

The new `assets/governance-interactions.js` and `.css` address those specific gaps. They load after `interaction-core` and `assurance-interactions`. They do not bootstrap the application or create new domain repositories.

## Work and Inbox

**Work** defaults to the acting person’s open work. Users can explicitly select everyone in the current scope or a named owner, then choose Open, Overdue, Due in seven days, Reviews, Corrections, Completed or All. Search, type, owner and queue filters are retained in the route. The projection is evaluated once per rendered queue and reused for its counts.

Rows show their concrete next step: complete a response, correct a submission, supply a document, confirm scope, evaluate a requirement or review a submitted decision. Due dates are ordered oldest first and labeled against the existing workspace snapshot. This sorting does not change source deadlines.

The new **`work-item?id=…`** page provides an exact link to an ordinary action, its accountable owner, required outcome, source and completion result. Completion or reopening still invokes the existing task workflow, preserving source checks, completion episodes and independent verification where required. Domain reviews continue directly to their exact existing route.

**Inbox** retains its existing recipient and scope filtering, receipts, personal archive/snooze settings, conversation history and source-resolution functions. A per-recipient preference adds action-priority ordering or newest-first ordering and an overdue/seven-day filter. Unresolved overdue actions rise above informational updates. Reading or acknowledging an item does not resolve it.

Each row provides a direct source action after validating the exact source. Task notifications open the exact work item. Existing follow-ups reopen their own action instead of an undifferentiated queue; a follow-up prepared while its source becomes resolved cannot be committed. These changes create no external messages.

## Access changes and review

### Role comparison

The role editor now leads to **Review changes**. The new **`role-change-review`** page shows changed settings and permissions, before/after values, affected assignments, separation-of-duties conflicts and effective access changes by person, action and covered locations.

The comparison runs the existing `gvDecision` against a temporary cloned governance state and always restores the original state. Explicit denies therefore remain effective in the preview. Merely opening a comparison grants no access. The view states its evaluation date; scheduled assignments remain visible separately from current effective changes.

Applying a role rechecks the draft hash, the complete compared access state, the original role and the acting administrator’s current access-management authority. The existing role command then performs its own validation and audit. Protected system roles and the final active organization administrator retain their existing guards.

Opening a different role or escalation policy preserves an unfinished singleton draft. The user can resume it or explicitly discard it before opening the other record.

### Assignments and direct rules

Access review defaults to assignments due or overdue within fourteen days. Filters cover person, review state and search. Separate tabs show direct rules and change history. A specific effective-access explanation remains available on demand.

Assignment, recertification, revocation and direct-rule dialogs first show the proposed effective-access impact. **Apply reviewed change** then invokes the existing canonical callback. If the form changes, its preview must be reviewed again. If the actor, navigation scope or underlying access model changes while the dialog is open, commit is blocked and the user must reopen it.

Direct rules require the named independent access reviewer to act under their own identity. Scoped authority, expiry, explicit deny precedence and separation-of-duties checks remain in force. An expired assignment offers **Create dated renewal**: a new assignment is prepared while the expired assignment remains in history.

## Escalation draft testing

The policy editor can now test the current draft before saving. It accepts an event location, record owner, severity, trigger time, acknowledgement delay and resolution delay. Short scenario actions cover no response, missing owner, Friday afternoon, acknowledgement after one hour and resolution after one hour.

Results use the existing canonical simulator. They show actual role holders, fallback recipients, unassigned levels, stopped levels and WAT timestamps. A side-by-side comparison explains the same delays under business and calendar clocks. An inactive draft can be tested in isolation and is labeled accordingly; the test does not activate or replace its saved policy.

Changing the draft marks its prior test as stale. Saving rechecks the saved policy source and current escalation-management authority. Tests never deliver notifications.

## File approval preparation and review

The preparation screen now brings together:

- The exact source version, contributor, scope and source link.
- Eligible independent reviewer choices and ordered review/sign-off stages.
- A clear deadline and the question the reviewer should decide.
- Actionable blockers for the acting identity, scope, stale snapshot, duplicate pending/completed review, invalid routing and missing original source.

The request cannot be sent while those known blockers remain. Sending still uses the existing source and file-byte verification and repeats its identity/source checks after asynchronous verification.

A stale **draft** can be refreshed explicitly from the current source while preserving its deadline and review instructions. Refresh does not alter an earlier submitted request. Vendor document requests must first link their current source in the dossier before the related approval draft can be refreshed.

Submitted approvals retain their originating route for an exact return action. The decision screen starts with **Choose a decision**, explains the selected decision’s consequence, and retains the canonical independent actor, source-version, byte-verification and sign-off-intent checks. Stale pending requests direct users to inspect the current source and have the requester withdraw the old request. Returned or closed requests link back to the source for continuation. Withdrawal rechecks actor, scope and request state after its dialog is opened.

## Model and verification

Domain records remain in their existing stores: access control, tasks, notifications, documents, file approvals and decision receipts. `S.governanceInteractions` contains interface preferences, source fingerprints and test inputs/results only. Organizational read views retain the package’s established inspection contract; these command checks do not constitute authenticated server retrieval.

The public `CLEAR_SIGHT_GOVERNANCE_INTERACTIONS` interface exposes scoped work projections, role comparison/apply, access proposal comparison, draft policy simulation, approval preparation issues and validated notification-source navigation.

Run `node tests/governance_interactions_v211_checks.js`. The saved results in `tests/governance-interactions-v211-results.json` cover **19 passing actual-source checks**: queue ordering and scope, exact-source routing, follow-up races, deny-preserving comparison, identity/source freshness, draft retention, reviewed access changes, dated renewals, escalation clocks/fallback/stops, approval refresh, real file approval and guarded withdrawal.

Relevant integrated regressions also pass: **25/25 flow checks**, **17/17 Inbox model checks**, and **15/15 vendor onboarding checks**. The tests substitute the browser shell. Browser layout, native keyboard interaction and real file persistence have not been verified in this environment.
