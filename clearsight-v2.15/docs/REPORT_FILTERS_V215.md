# Applied reporting filters — V2.15

21 November 2026 · Source implementation and controlled-DOM verification

## Problems corrected

Loss and response-accountability filter forms were explicit Apply forms, but editing their inputs left Save live view, exports and report preparation available without distinguishing temporary input from the population already displayed. The loss concentration selector could also navigate immediately and discard those temporary edits. Response-accountability's error recovery link was ineffective: the shared link adapter reattached the invalid response filters that the link was intended to clear.

## Current interaction

- Each loss/response filter form carries the exact values used to render it. A polite status distinguishes applied filters from unapplied edits. Temporary input does not recalculate or mutate displayed results.
- While inputs differ, report exports, Save live view, Prepare board pack and the in-app print action are unavailable. Their command handlers also recheck pending state before reaching the existing authorization and source commands. The loss grouping control waits for Apply or Revert.
- **Revert edits** restores the displayed filter values in place, returns focus to the first filter and restores each output control's previous disabled state. No render, source write or alternate reporting cohort occurs.
- **Reset loss view** selects occurrence dates, NGN, the default loss-group concentration and the currently selected 30-/90-/365-day window. It retains the exact organization and KRI cycle and clears stale source drilldowns. The reset values are stored in the existing loss-report preference record.
- **Clear response filters** removes only program, respondent, reviewer and response-state restrictions. Organization, reporting period and KRI cycle remain explicit.
- The blocked response report's existing recovery link now clears invalid filters after the shared context has been applied, so it opens a usable scoped report.

The existing Apply handlers, named-response model, financial calculations, permission checks and immutable report snapshots remain authoritative. No read model, network request, new persistent store, timer or observer is added. The ordinary frozen-report renderer contains no live filter controls.

## Files and verification

Implementation extends `assets/scoped-analytics.js/.css`; the presentation-only `premium-reporting` module is unchanged. Pending-state styling uses shared semantic tokens and existing touch geometry.

`tests/report_filters_v215_checks.js` has **7 passing checks** covering unchanged cohorts, blocked output commands, exact revert behavior, permission preservation, loss and response resets, invalid-filter recovery, grouping interception and frozen-report stability. The existing scoped-analytics **24 checks** and premium-reporting **8 checks** also pass.

These checks execute the actual application handlers with a controlled form DOM. Native appearance, focus/announcement behavior, browser event ordering and print acceptance remain unverified under the established browser restriction.
