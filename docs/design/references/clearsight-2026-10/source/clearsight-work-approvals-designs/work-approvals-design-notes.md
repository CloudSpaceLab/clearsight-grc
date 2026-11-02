# ClearSight · 10 — Work & Approvals

Date: 15 October 2026  
Status: design prototype only; all records are illustrative.

## Composition

- Reuse the ClearSight shared navigation and scope hierarchy. On small screens, expose all branch, department and group selectors in one horizontally scrollable row rather than removing filtering.
- Top strip is four small actionable counts: Overdue, Due this week, Your approvals, Evidence to review. Counts are derived from the current actor and selected scope in the prototype; the production view must read authorized metric/queue results.
- One bounded work queue with tabs: My work, Approvals, Team queue, Completed. Row selection updates the contextual panel; complex review opens a focused drawer.
- Keep Copilot secondary and show only one of three evidence-linked suggestions at a time. The orb is an orientation cue, not a compliance status.
- Approvals distinguish submitted evidence, independent verification, authorization and completed outcome. No implied automatic approval.

## Prototype interactions

- Work intent tabs (keyboard arrows/Home/End), status chips, due-window, search, branch/department/group selectors, selected row, upcoming deadlines, contextual assistant suggestion selector.
- Review drawer displays source context, current task, evidence and activity. An independent-verification gate blocks sample approval; an eligible request with a review note can advance the local mock state. Request changes requires a note. No API is called.
- New request creates a local illustrative item after validating title, context and date. The prototype does not create a bank record, send notifications, or record an authoritative decision.
- Escape dismisses dialogs; focus cycles inside focused review/request surfaces. CSS disables nonessential motion for reduced-motion users.

## Implementation mapping

- Existing routes/components: Home Oversight `My work`, Work, exact Matter/Action/Evidence context, governance authority, existing UI `DataTable`, `MetricCard`, `Tabs`, `FocusedSheet`, `FormField`, and enterprise scope selector. Do not manufacture a second workflow engine.
- All counts depend on verified actor, tenant/legal entity, current program/organization scope and the same current snapshot. Empty/partial/restricted cases must fail closed.
- Each decision needs a verified actor, authority route, evidence basis and durable command receipt. A completed task is not verified remediation or closure.
- Avoid showing a single approval button when a maker-checker, mandatory DPO or independent assessor review gate is incomplete; explain the blocker at the point of work.
- Preserve on-mobile semantics and action reachability at 390px and avoid horizontal data-table scrolling.

## Screenshots

- `work-approvals-4k.png`: My work at 1880 CSS pixels, 2× scale.
- `work-approvals-approvals-4k.png`: Approvals tab.
- `work-approvals-team-4k.png`: Team queue.
- `work-approvals-review-4k.png`: Focused review drawer.
- `work-approvals-new-request-4k.png`: Prototype request creation.
- `work-approvals-mobile.png`, `work-approvals-review-mobile.png`, `work-approvals-tablet.png`: responsive states.

Validation: Browser renders checked at 390, 768, 1024, 1280 and 1880 CSS pixels. UI state, keyboard and governance-gate behavior are prototype only, not a production acceptance claim.
