# ClearSight V2.8 — reports for management and directors

## Review the flows

Open **Reports & insights**. The default view starts with a short executive reading, four decision-relevant measures, an exception agenda, return coverage, loss concentration, a residual-risk distribution and a loss-occurrence chart. The agenda identifies the source record, accountable owner, recorded date, existing response and a question management should resolve.

Use the other perspectives for a more specific decision:

| Perspective | Practical interpretation |
| --- | --- |
| Locations | Concentration of recorded loss, appetite exceptions, KRI coverage and overdue work, without presenting branch size as management performance. |
| Loss drivers | Current and immediately previous occurrence windows, financial reconciliation, category concentration and source-event drilldown. |
| Branch KRIs | Approved observations, condition breaches, explicit N/A, context questions, missing responses and the cause/impact/action captured in the source return. |
| Assurance & actions | Requirement conclusions by their actual assessment target, evidence freshness and the age of open commitments. |
| Third party & privacy | Critical service engagements and unresolved impact/case reviews. |
| Data confidence | Separate coverage denominators and the limits on interpreting the available data. |
| Report library | Saved live perspectives and fixed published management packs. |

**Assign follow-up** creates a normal, scoped Work item. It keeps the source ID, accountable location, owner, due date and expected outcome. An existing open follow-up is reused instead of creating another reporting-only task. The normal Work action opens existing commitments.

## Preparing a board pack

1. Select the intended scope and loss/KRI periods in the live report, then choose **Prepare board pack**.
2. Set the title, audience, meeting date, preparer and included sections.
3. Record management's assessment and add explicit decisions or requests. A recorded exception can be used as an editable starting point. Each decision requires a substantive rationale, accountable owner and target date. A source ID must belong to the captured report or be left blank for a general management item.
4. Acknowledge the reporting coverage, preview the complete report and publish the snapshot.
5. Use **Print / save PDF** for the browser's print dialog or **Export captured data** for the fixed JSON snapshot.

Publishing a pack does not approve a risk acceptance, grant resources, change a rating, satisfy a requirement or send a message. Author statements and source-derived management questions remain distinct.

## Calculation and snapshot contracts

- The reporting layer composes the existing `reportData`, `scopeTotals`, `ixLossPeriod`, `ixKriStats` and `ixAggregate` functions. It does not establish a second ledger or a second KRI truth.
- Loss charts reconcile to the recorded event cohort exactly. Their financial values remain integer kobo and are displayed in NGN. The previous window has the same duration and ends immediately before the current window. Recoveries share the same snapshot basis.
- Risk posture, loss occurrence windows and periodic KRI months retain separate clocks. Changing the loss window does not recalculate the current risk population or KRI definition.
- A location total is not labelled with a single-branch threshold. Boolean aggregates display a number of exception locations. Context-only questions are displayed as context. Currency values, decimal hours and weighted ratios retain their units.
- KRI histories require matching complete canonical question definitions and the same reporting cohort. Boolean adverse polarity, context classification, conditional applicability and policy basis are part of that comparison. Each period shows its own approval and known-value coverage. A newly introduced 51-question branch definition therefore has no invented trend from the older three-metric unit definition.
- Missing and unapproved observations are not converted to zero. Approved N/A is visible separately and excluded from both numeric aggregates and the applicable-observation coverage denominator. The full expected population and unresolved gaps remain visible.
- The 51-question result register is grouped into the eight source categories. Categories with breaches open first in the live view; published packs include every category. Each table has four columns rather than a column for every question.
- No historical risk-rating movement, loss-collection completeness, transaction-normalised branch performance or complete bank-wide risk score is inferred from the available records.
- A board snapshot captures its chart series, location totals, source rows, metric definitions, owner/due context, quality counts and management statements. Published HTML reads the snapshot only. Changing live records or the navigation scope cannot silently refresh the pack.
- Old snapshots without the additional chart context retain their captured data; unavailable comparisons are labelled as not captured.

## Source integration

Load `board-reports.css` after the existing styles and `board-reports.js` after `branch-kri.js`, `documents.js` and `usability.js`, before the final application bootstrap.

The module replaces `reportsView`, the `insight-detail`, `report-builder` and `report-pack` route handlers, and the frozen report body. It wraps report snapshot capture and report step/publication validation. It retains the existing saved-view, period, source export and published-pack export commands.

The public read-only test interface is `CLEAR_SIGHT_BOARD`. Follow-up notifications use `CS_V28_EVENTS.emit` when that optional shared inbox interface is available. All state remains local demonstration state; authenticated delivery, approval authority and server-side enforcement are production integration concerns.

## Validation

`node tests/board_reports_v28_model_checks.js` passes **11 actual-source integration checks**. These cover source reconciliation, all report renderers, currency/Boolean meaning, comparable KRI cohorts, scope and period separation, decision validation, frozen snapshot/HTML immutability after live mutations, source-linked Work creation and duplicate prevention, and briefing CSV fields.

`node --check assets/board-reports.js` also passes.

The browser integration suite is `tests/board_reports_v28_checks.py`. It includes responsive overflow checks, UI authoring, source-linked Work, published snapshot immutability and print-media visibility. In this execution environment Chromium could not start because its profile socket was disallowed. Those browser assertions are **not claimed as passed** by this module's test report; the Node harness does not establish visual or browser accessibility conformance.
