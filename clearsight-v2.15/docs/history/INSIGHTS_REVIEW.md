> Historical design note. For the current implementation and limits, read `../DATA_AND_WORKFLOW_REVIEW.md` and `../TEST_REPORT.md`.

# ClearSight V2.5 — Reporting and branch KRI review

13 November 2026 · Interactive design reference · Synthetic data only

## Decision

Make Reports & Insights a decision workspace, not a wall of unrelated charts. Keep one authoritative organization scope in the global header, immediately beside the ClearSight logo. Give users a small set of purposeful perspectives, source drill-downs and a guided report-preparation workflow. Keep the direct form editor, privacy lifecycle, framework adoption and full record workspaces.

The earlier V2.4 supported scoped continuous indicators, threshold observations and histories, but did **not** implement periodic branch returns. V2.5 adds a local demonstration of monthly collection, submission, independent review and consolidation. This is not a complete bank KRI catalogue or a production reporting deployment.

## One navigation scope, three measurement clocks

Branch → Department → Unit appears in the same fixed header as the logo, not in a second navigation row. All means all descendants; clearing a parent clears dependent children. A unit selection resolves its parent branch and department. Scope persists through navigation. At narrow widths the logo becomes its mark while all three selectors remain in the header.

The scope filters lists, source records, loss totals, live reports and branch returns. It does not reassign an existing draft, submission, assessment target or published report. Shared templates remain reusable. Program assurance explicitly retains the original assessment target instead of manufacturing a branch-level score.

Reports distinguish:

- **Current posture:** accepted risk ratings, appetite, current controls and open work at the fixed demo snapshot (13 November 2026).
- **Loss occurrence window:** 30, 90 or 365 days, including the snapshot date. Recoveries are received-to-snapshot amounts for those same events, not period cash recoveries.
- **Monthly KRI period:** the selected frozen reporting cycle. Changing the loss window does not change its values.

## Reporting perspectives

| Perspective | Decision supported | Implemented content |
|---|---|---|
| Executive | What needs management attention? | Appetite exceptions, recorded net loss, approved return coverage, overdue actions, attention list, loss drivers, loss occurrence bars, quality summary and source risks. |
| Locations | Where does the issue concentrate? | Branch → department → unit comparison; risk and appetite counts; gross/recovered/net losses; KRI coverage and unit breaches; overdue work. Drill-down changes shared scope. |
| Loss drivers | What is causing recorded loss? | Category, accountable risk owner or individual risk; exact monetary reconciliation; adjacent-period comparison; source event rows. |
| Branch KRIs | Which units are breaching, and which have not reported? | Branch × metric matrix, native values and units, approved/missing coverage, worst-unit breach labels, valid consolidation and period history. |
| Assurance & actions | Are controls and corrective work keeping up? | Control effectiveness, scoped RCSA progress, requirement outcomes by actual target, action ownership/dates, evidence freshness/review. |
| Third party & privacy | Where do dependencies and review obligations accumulate? | Supplier service counts, critical engagements, accountability, renewal, processing/DPIA/case states. No invented supplier financial exposure or privacy compliance score. |
| Data quality | What prevents a reliable conclusion? | Unapproved/unknown KRI values, stale continuous readings, evidence freshness, unassigned risk records, unavailable loss-collection completeness, missing historical/exposure feeds. |
| Saved reports | What can be reused or presented? | Saved live perspectives and immutable-by-demo-workflow management snapshots. |

Cards, chart rows and comparison rows lead to records or the relevant workflow. Full-page source views preserve context. Simple save-view and recovery choices remain centered dialogs; multi-step setup and report preparation use wizards. No additional right-sidebar flyout system is introduced.

## Branch KRI workflow and numerical safeguards

A cycle freezes its reporting month, metric definition/version, expected unit population, owner(s), reviewer and optional program context. Each return is keyed by cycle and reporting unit. October, September and August are seeded; current-month cycles can hold drafts but cannot submit before month end.

Respondents enter native counts or numerator/denominator pairs, source references and explanations. A recorded zero is valid. Missing data must be explicitly marked unavailable with a reason. Breaches require explanatory context. Submitted snapshots are read-only and retain their definition and source identity.

Reviewers can request changes or approve independently of the submitter. Resubmission appends another snapshot. Only accepted snapshots feed branch comparisons and management reports. Approval may retain an explicit unknown; it does not turn unknown into zero or make the entire branch compliant. Pending reviews and subsequent breach remediation compose the existing Work queue. One accepted return episode produces at most one remediation task, rather than one task for every metric refresh.

The sample metric pack has **three authored demonstration metrics**: unresolved reconciliation items, aged complaints and failed payment rate. The thresholds are not regulatory limits. The new KRI return grid is not yet connected to arbitrary forms made in the general form builder, nor to automated technical source observations. A production bank pack needs its own governed definitions, applicability and ownership.

For the October fixture, two of four units are approved. Accepted failed payments are 170 and accepted attempts are 60,000, giving 0.283333…%. Averaging the two reported unit percentages would give a different, incorrect consolidation. The matrix still shows the underlying unit breach. Unit thresholds are not silently applied to bank-wide sums of unit counts.

For count measures, the demo sums approved values. For the ratio measure it uses sum(numerator)/sum(denominator). The same cycle, definition and cohort are retained; completeness accompanies each comparison. No favourable trend is inferred from fewer reporting units. A location outside the frozen population is **Not in cycle**, not a missing return.

## Live views versus published management packs

Save view stores the organizational scope, report perspective, loss period, KRI period and loss grouping. It is a live query configuration, not a frozen record.

Build report uses three steps: audience and sections; captured scope/periods and management commentary; preview and publish. The source snapshot is captured when preparation starts and remains explicit. Later changes to navigation or accepted KRI returns do not rewrite it. The audience label does not silently alter calculations. The pack includes source identifiers, accepted observation versions, provenance, denominator details and limitations. CSV/JSON payload export and print layout are available; automated distribution and native PDF generation are not implemented.

## Information not fabricated

There is no combined bank risk score, no invented historical risk-migration chart, no claimed full bank credit/market/liquidity view, no loss-rate league table without matching business denominators, and no proof of nil loss merely because no events were recorded. Forward-looking scenarios, operational resilience dependency modelling, root-cause taxonomy and full assurance planning require additional governed data. Those are useful next reporting capabilities, not data that this demo pretends to possess.

## Production architecture

Use existing authoritative risk, loss, control, program, evidence, monitoring and Work records. Do not transplant the browser-local storage model as a backend or create a second independent monitoring engine.

A production metric definition should carry versioned unit, direction, thresholds, aggregation, period/calendar, applicable populations, denominator semantics and permitted input/source binding. A reporting obligation identifies the exact subject, definition and period; human forms and connected checks supply observations. Preserve raw observations, accepted evaluations and report snapshots separately. Never reuse one monitoring-check identity for all branches or overwrite another branch's latest value.

Resolve authorized descendant IDs server-side and apply the same resolved scope/snapshot to metrics, lists, charts and exports. Count shared risks/controls once by canonical identity. Use an explicit primary loss attribution or governed allocation, never repeat a full financial amount for every related control. Version threshold changes; do not reinterpret old submissions using today's threshold without an explicit restatement.

Use incremental reporting projections and batch source collection rather than rescanning all observations or scheduling a separate job per form cell. Add concurrency/version checks, audit retention, data-quality controls, authenticated role separation, tenant isolation, evidence permissions, governed templates, import reconciliation and delivery monitoring before production use.

## Research grounding

The design is informed by BCBS operational-risk monitoring/reporting guidance: reporting should support board, senior-management and business-unit decisions, expose appetite exceptions and losses, and remain manageable in volume. It also follows the SRP36 emphasis on accurate, comprehensive, understandable reporting with appropriate interpretation and audience relevance. This is a design rationale, **not a claim of BCBS239 compliance or universal legal applicability**.

Primary sources reviewed 13 November 2026:

- BCBS, Operational risk, Principle 8, paragraphs 10.53–10.56: https://www.bis.org/committees/bcbs/basel-consolidated-guidelines/module/orr/10
- BCBS, Risk data aggregation and risk reporting, particularly 36.43–36.60: https://www.bis.org/committees/bcbs/basel-framework/standard/srp/36/inforce/2019-12-15/published/2019-12-15

## Boundaries

All records are synthetic. Role selection and local snapshots illustrate workflow, not security or server-enforced immutability. No live database/API access, scheduled collection, notifications, regulator submission, evidence bytes or production authorization are implemented. Bulk branch spreadsheet import, arbitrary KRI catalogue authoring, per-metric applicability, automated-source-to-branch-return binding and concurrent-user reconciliation remain production work. The existing general form snapshots also remain separate from the earlier requirement-evaluation acceptance pipeline.
