# GRC portfolio presentation — reference patterns and data contracts

**Scope:** Program portfolio and Risk register visual orientation. Follow-on guidance for existing Loss, KRI/KCI and RCSA workspaces. Complements [semantic domain presentation](../superpowers/plans/2026-10-05-semantic-domain-presentation.md) (#346); it does not create a replacement visual system.

## Reference patterns reviewed

- [ServiceNow Risk Workspace](https://www.servicenow.com/docs/r/governance-risk-compliance/grc-risk-management-workspace/risk-workspace.html): role-based summary, KRI breaches, risk concentration, heatmaps, linked records and tasks.
- [ServiceNow risk heatmap workbench](https://www.servicenow.com/docs/r/governance-risk-compliance/grc-risk-management-workspace/risk-heatmap-workbench.html): impact/likelihood coordinates, prior/current movement, context on selecting a cell.
- [AuditBoard IT Risk Management](https://auditboard.com/itrm/): risk posture overview, quantified impact, mitigation status and drill into action plans.
- [MetricStream ERM](https://www.metricstream.com/products/enterprise-risk-management.htm): hierarchy/category slices, inherent-versus-residual risk movement and role-based reporting.
- [LogicGate visual reports](https://help.logicgate.com/hc/en-us/articles/4402674160788-Create-Visual-Reports): explicit chart types, field-bound reports and contextual dashboards.

The common useful principle is **visual comparison supported by authoritative measures**, not decoration. Copy the conceptual grammar, not a vendor's layout or palette.

## Delivery in #421

### Programs

1. Preserve the exact Program list, filters, routes, statuses and paginated query.
2. Show a distribution of the **loaded** Program operating states: follow-up, current, assessment required, not applicable.
3. Rank open issue counts by Program only where the Program assessment and count can be compared. Provide a direct Program drill and visible omitted-population information.
4. Treat stale/unassessed Program states separately; do not infer risk severity from requirement/evidence/issue counts.
5. Make list record state scannable with a semantic left rail, with the descriptive state and numeric facts retained.

### Risks

1. Show the loaded Risk appetite position distribution (breached, approaching, within, unknown/outdated).
2. Reuse the same current-version assessment and active appetite validity contract already used by RiskRegister's status badges.
3. Keep the Risk register and row drill as the operational surface. The visual must not be presented as a complete tenant population when a cursor remains.
4. Do not convert arbitrary assessment dimension objects into heatmap coordinates or standardized risk percentages.

## Existing domain-specific presentation boundaries

| Domain | Meaningful visual | Required data contract | Avoid |
| --- | --- | --- | --- |
| Risk | Inherent/residual matrix, appetite exceptions, movement | Frozen comparable scored assessments, declared axes and appetite revision | Heatmap inferred from issue priority |
| Loss | Occurred-period loss trend, gross/recovered/net decomposition, currency grouping | Exact monetary units, recovery/reversal receipts, occurred dates and bounded population | Summing different currencies; negative recovery hidden |
| KRI/KCI | Threshold band, observation history and trend gaps | Typed native measure, unit, threshold revision, observed/reporting period and freshness | Smooth invented curves, conflating form score with native measurement |
| RCSA | Cycle completion and challenge position by area | Frozen assessment population, submitted vs challenged states and exact group scope | Percent-complete from noncomparable cycle populations |
| Controls | Assurance supported/failed/unknown by framework | Exact control/assurance links and valid evidence revision | Converting missing checks into passed controls |
| Forms | Draft → requested → submitted → assessed handoff | Exact linked distributions, immutable submissions, authorized assessment receipts | Treating submitted as approved |
| Portfolio | Operating position, issue concentration, authorized scope | Current state projection, issue counts and revision consistency | Treating the first 20 records as the entire portfolio |

## Visual rules

- One top-level visual orientation region, followed by the operational register; never stack an independent dashboard on every tab.
- Use existing shared chart components (RankedBarList, MetricTrend and StackedDistribution) only with their appropriate measure. Use the established theme and semantic color tokens.
- Every chart names its scope, including when it is limited to loaded results, and distinguishes missing/stale from genuinely zero.
- Keep descriptive status, date/period, owner and evidence accessible without interpreting color.
- Visual selection must route to the exact underlying record and preserve workspace filters/context.
- Do not add an unapproved data model, metric calculation, trend, or risk heatmap to satisfy a visual-design request.
- Verify light/dark, wide/narrow, keyboard and forced-colors behavior. Fixture render tests do not establish live bank readiness.

## Follow-on acceptance

1. Confirm rendered Program portfolio and Risk register at desktop and 390 px, including long descriptions and mixed assessment states.
2. For source-backed KRIs, verify correct native measures and threshold history before adding trend plots.
3. For Losses, verify monetary basis, scope and full-population availability before adding an aggregate chart.
4. Maintain snapshot-to-drill parity and role-based authorization in Insights rather than developing a second GRC reporting engine.
