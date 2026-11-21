# ClearSight V2.9 — operational-loss intelligence

## What the source requires

The supplied `LOSS DATA BASE.xlsx`, Sheet1, contains separate columns for account name, loss group, transaction particulars, branch, regional bank, directorate, amount, currency, occurrence, recognition, recovery and the source reporting month/year/quarter. These are distinct reporting dimensions.

The reviewed extraction contains repeated non-period transaction fingerprints across reporting months, a mixture of Excel serial and written occurrence dates, and recovery narratives without dated receipt amounts. These patterns justify explicit review and provenance. They do not establish that every repeated row is a duplicate or that a narrative claim represents received money.

No source customer names, transactions, amounts or private narratives have been copied into the demonstration. Tests use clearly synthetic events.

## Reporting workflow

Open **Reports & insights → Operational losses**, or the `loss-insights` entry point.

Choose a reporting basis, date window and one original currency:

| Basis | Included event population | Practical use |
| --- | --- | --- |
| Date of occurrence | Reviewed events that occurred within the selected dates. | Understand operational events arising during the period. |
| Date of recognition | Reviewed events whose financial loss was recognised within the selected dates. | Understand recognised loss, including events that occurred earlier. |
| Reporting month | Reviewed events assigned to the explicit source reporting months touched by the date window. | Reproduce the reporting-period view without pretending it is the occurrence or recognition date. |

Reporting-month filtering includes whole months. Cash receipts and reversals continue to use their exact transaction dates. A partial November date window therefore includes the November reporting cohort, while cash activity remains restricted to the specified November days. Both meanings are stated on the page.

Records lacking the selected date basis are excluded from that cohort. Their count and source links remain available under **Source reconciliation** so users can complete the metadata. Missing dates are never guessed from another column.

Near misses have no financial recognition date. They are excluded from the recognition cohort and counted separately as **not applicable**, without being presented as missing-date defects. They remain available under occurrence and reporting-month views.

## What directors and managers can see

- Gross reviewed loss, verified recoveries retained, unrecovered amount and the recovery retention ratio for the same event cohort.
- Concentration by loss group, account name, branch, regional bank or directorate. Account name and loss group are not substituted for the linked risk category.
- Aging of unrecovered actual losses from recognition to the snapshot, including a separate missing-recognition bucket.
- Occurrence-to-recognition lag, with the measured event count, median and missing/invalid-date counts.
- Management's recorded recovery prospects, expected amounts and target dates. Estimates that exceed the remaining balance and targets that have passed are explicit follow-up items.
- Cash inflows, approved reversals and net recovery movement in the exact receipt window, including cash linked to older loss cohorts.
- Source-level management questions for material unrecovered balances, delayed action, incomplete cause analysis and recovery estimates.
- Review coverage, unaccepted submissions, pending corrections, missing causes, unlinked risks, reporting-period discrepancies and repeated-event candidates.

Each chart has an underlying table. Loss and receipt tables retain source IDs. The event links open the normal loss workspace and its review, recovery and action flows.

## Financial meaning

`lossFinanciallyIncluded` owns the reviewed-event boundary: accepted and closed events are included. A pending correction remains separate from the previously accepted event. An unaccepted intake or submission does not alter the financial views.

The reporting layer uses the canonical recovery helpers from `loss-register.js`:

| Helper | Meaning |
| --- | --- |
| `lossReceivedReceiptRows(loss, asOf)` | Current verified gross receipt entries, classified by their cash date, including receipts later reversed. |
| `lossRecoveryReversalRows(loss, asOf)` | Approved recovery reversals effective by the snapshot. |
| `lossVerifiedReceiptRows(loss, asOf)` | Verified receipts still retained after effective approved reversals. |
| `lossVerifiedRecoveries(loss, asOf)` | Integer minor-unit sum of retained verified receipts. |

This is a view of the currently reviewed ledger classified by effective cash dates, not a reconstruction of everything that was known at an earlier time. Frozen report packs preserve the knowledge captured when the pack was created.

Unrecovered amount equals gross loss minus retained verified recoveries. Expected recoveries, notes and claims do not reduce it. The separate cash-movement calculation is gross inflows in the cash window minus approved reversals in that same window. It can be negative when the period reverses an earlier receipt.

The supported currencies are NGN, USD, EUR, GBP and ZAR, each stored in integer minor units with two decimal places. A report selects one currency. There is no implicit exchange rate, mixed-currency sum or consolidated financial total. The pre-existing management dashboard retains its separate reviewed NGN occurrence basis.

The risk workspace uses the same canonical reviewed NGN totals for its lifetime financial summary. It states that the dashboard date window does not apply. Its linked-event table includes submitted records with an explicit exclusion label, retains each event's original currency, and offers the organizational loss report for currency-specific analysis. The existing risk workflow and scope wrappers remain in place. Legacy `riskDetail` and `lossDetail` calls navigate to the current scoped workspaces instead of opening the superseded NGN-only dialogs.

The source Month/Year fields are retained as an explicit reporting period. Differences from recognition month are visible. Candidate duplicate groups are flagged; the reporting layer neither silently drops an accepted event nor invents a second event from a repeated period row.

## Board packs and exports

**Prepare board pack** captures the selected operational-loss basis, currency, scope and window in `reportData().operationalLoss`. A pack containing its loss section receives an operational-loss annex with the captured figures, charts, source rows, management questions and limitations.

The original board pack and the annex each state their own basis. Changing a live loss, approving a later receipt or reversal, navigating to another branch, or changing report filters cannot refresh a published annex. Older reports without `operationalLoss` retain their original body and receive no live-data annex.

The local Copilot preview follows the visible loss report's currency, date basis and date window. On the loss register it also follows the search and workflow-state filters. Its risk context uses the same reviewed NGN lifetime population as the monetary cards. Published-pack explanations use the captured annex rather than current source records. Repeated reads of unchanged source data keep the same context hash; a freshly generated capture timestamp is not treated as a source change. This remains the existing local rules-based preview, without a connected model.

Two CSV exports serve different purposes:

- **Loss cohort:** original account/group fields, linked risk taxonomy, hierarchy, all three event/reporting dates, original currency, integer minor-unit gross/recovery/unrecovered amounts, cause, owner, recovery estimates and pending-correction status.
- **Receipt activity:** receipt and loss IDs, cash dates, current reviewer/source references, separate receipt and reversal kinds, absolute amount and signed cash movement in minor units.

The browser's print flow includes the charts, financial basis and source tables. Browser print delivery and responsive layout have not been validated in the restricted runtime used for this change.

## Integration

Load `loss-reporting.js` after `loss-register.js` and `board-reports.js`, before the final application bootstrap. Load its stylesheet after the shared report styles. Add a `loss-insights.html` entry point for route `loss-insights` if separate module launchers are required.

The module registers `loss-insights`, adds the Operational losses report tab, composes `reportData`, appends a frozen annex through the board-pack body renderer, and adapts the existing Copilot source context. It owns no approval or recovery mutation. The read-only interface is `CLEAR_SIGHT_LOSS_REPORTS`.

The global helper `lossReportRows({basis, from, to, scope, currency})` supplies the reviewed, single-currency event cohort. Invalid dates, bases, currencies or explicit scopes fail without broadening the query.

Read cohorts follow the explicit organizational navigation scope, consistently with the existing board reports. Local role checks demonstrate workflow permissions for commands. **This prototype does not establish an authorization boundary for data retrieval**; authenticated row access and tenant boundaries belong to the production service. The UI does not silently put an organization-wide heading over a smaller actor-filtered total.

## Validation

`node --check assets/loss-reporting.js` passes.

`node tests/loss_reports_v29_checks.js` passes **18 actual-source integration checks** through the shared `runtime_harness.js`, which loads the complete V2.9 module chain, covering:

1. Separate occurrence, recognition and reporting-month cohorts.
2. Currency separation, reviewed-state inclusion and pending-correction isolation.
3. Gross cash inflows, approved reversals and negative net cash windows.
4. Aging/concentration/period-chart reconciliation, including partial reporting-month windows.
5. Account/group/risk taxonomy separation.
6. Management estimates and reporting-period discrepancies.
7. Candidate duplication without silent deduplication.
8. Precise organizational scope and invalid-filter handling.
9. All report lenses and empty-state render functions.
10. Immutable new annexes and unchanged old report bodies.
11. Traceable cohort and receipt exports.
12. Near-miss recognition applicability without false missing-date gaps.
13. Reviewed NGN reconciliation across existing management summaries, concentration, location totals and source rows.
14. Day-based trend offsets and exact partitioning of the supported 30-, 90- and 365-day windows, including adjacent-period comparisons.
15. Risk-workspace lifetime totals, currency-explicit event rows, scope and preserved workflow extension.
16. Loss-report and register Copilot contexts matching currency, date, search and workflow filters with stable hashes.
17. Captured published-pack explanations and no new live context on older packs.
18. Legacy detail redirects, registered controls and valid destinations on the new report routes.

The existing V2.8 board-report suite also passes **11/11** after these changes. Both modified JavaScript files, `loss-reporting.js` and the specifically updated risk-view section of `journeys.js`, pass `node --check`.

These tests execute the actual application sources with minimal DOM stubs. They validate reporting logic, render generation and export payloads; they do not claim browser visual, accessibility or production-security conformance.
