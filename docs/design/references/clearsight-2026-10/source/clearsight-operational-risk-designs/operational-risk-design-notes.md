# 07 — Operational Risk

Design review: 15 October 2026. Continuation of ClearSight Control Tower visual direction and screens 01–06.

## Screen coverage

- **Overview**: net operational losses, KRI breaches, RCSA review coverage and overdue actions. Financial movement and concentration are separate from current-state KRI and RCSA views. First-viewport charts display source/period semantics rather than a synthetic organization-wide risk score.
- **KRIs**: native values, approved thresholds, unit of measure, current condition, responsible function and recent observations; bounded filter and specific record detail.
- **RCSA**: current cycle by unit, reviewed/expected denominators, owner, deadline and pending work. Review counts are not an assessment of effectiveness.
- **Loss ledger**: original gross amounts, separate linked recoveries and calculated net, with exact event/time/currency semantics and record detail; never infer recoveries from missing cells.
- **Actions**: separate assigned work and status, without inferring completed remediation from a submitted assessment.
- **Copilot**: single visible suggestion, three manually selectable alternatives, action to an existing screen. It does not make regulatory or risk conclusions.

## Design and copy

- Compact scope and date controls, restrained wave treatment, shared metric cards, thin borders, modest depth and purposefully sparse visualizations.
- Copy is direct: Operational risk, Net operational loss, KRI breaches, RCSA completion, Overdue actions, Review cycles. No introductory slogans or self-explanation.
- Reporting period changes the **loss flow**, not point-in-time indicator breaches or the current RCSA cycle. Subtitle distinguishes those. The 7/30/90-day selector and scope filters update the local preview data.
- Desktop 1880 CSS px @2x rendered to a 3760px PNG. Mobile 390 CSS px @2x rendered to 780px wide screenshots. Tested 1880, 1280, 768 and 390px; no horizontal document overflow.
- Responsive loss ledger becomes stacked cards with visible labels and exact values. Mobile exposes branch, department and group filters. Keyboard, Escape close, modal focus and reduced-motion support are exercised in the self-contained HTML prototype.

## Product implementation boundary

All names, transactions, charts, deadlines and metrics are **synthetic preview values**; these files do not connect to ClearSight. The prototype's record creation only previews the form. No server mutation or seeded reference data has been performed.

Reuse existing native RCSA, Monitoring Indicator, Loss Ledger (including linked recoveries), organization scope, Work/Matters, Program and metric/report pipelines. Avoid recreating a generic Matters-only view for these registers, a second chart component family, or a private scoring engine. Display unknown/stale/partial conditions from canonical source projections, preserve real currency and source version, and maintain precise drill identities.

### Production gates

- Source freshness, approved limits and revision handling for every KRI; native value not converted to an arbitrary compliance percentage.
- RCSA status and completion based on distinct required assessments with accurate expected scope and applicable reporting period.
- Loss gross/recovery/reversal/net reconciled at exact tenant/entity/organization scope and monetary precision; never sum heterogeneous currencies without approved FX basis.
- Scope/date filtering performed through bounded authorized API queries, not client-side hiding of a broad unauthorized population.
- Copilot suggestions authorized, attributable and actionable; no invented output or unverified source statements.
- Errors/absence/zero vs unknown states and snapshots verified at 390/768/1280/1440+, light/dark and reduced motion before merging.

Next screen proposed: **08 — Privacy & Data** (ROPA, DPIA, breach records and vendor privacy gates), then reporting, Forms library and full navigation/accessibility sweep.
