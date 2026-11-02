# Domain-specific GRC visuals: Loss, RCSA and native KRI/KCI

Complements the merged [GRC portfolio presentation](2026-10-10-grc-portfolio-visual-language.md) and existing semantic presentation issue #346.

## Delivery

- **Losses**: Show gross, net outstanding and recovered financial amounts as three comparable bars *within each currency*. Use already-confirmed record totals and exact minor-unit summation with BigInt. Show only validated active records from the current bounded page. Exclude voided records and unsupported or inconsistent totals with explicit counts. Do not produce enterprise loss totals, mixed-currency amounts or temporal loss movements from a paginated register.
- **RCSA**: Show a proportional stage distribution from the loaded cycle statuses. These are **cycle counts**, never proportions of completed Risk/Control assessment. Preserve existing frozen-population counts, independent challenge, cycle selection, detail and original handoff targets.
- **KRI/KCI**: On an individual indicator's revision-specific history, plot up to 12 comparable native numeric observations using the shared MetricTrend. Unconnected points avoid filling intervening periods. Require identical field, unit, precision, approved limits, currency/duration unit and check revision; exclude missing/unknown or unsafe numeric values. Continue to show the authoritative full observation table and current state separately; never chart concern points as a native indicator measurement.

## Presentation boundaries

All visuals explicitly identify **loaded scope**, do not infer missing data as zero, retain accessibility text and route back to exact source records. Existing `RankedBarList`, `StackedDistribution`, `MetricTrend` and design tokens are reused. No additional charts package or independent UI framework. No new backend data sources, calculations, persisted scores or regulatory assertions.

## Verification

Focused unit and rendered view tests cover monetary reconciliation, mixed currencies, status distinction, native measurement compatibility and exact record drill. CI and deterministic UI/UX checks must be green on the merge head.
