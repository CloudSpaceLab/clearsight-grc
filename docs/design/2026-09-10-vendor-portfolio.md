# Vendor portfolio and curated demo

## Decision

Bank owners and risk reviewers need portfolio workload, service criticality and an immediate route to the affected vendor. Replace the split register/detail rail with a full-width dashboard and full-width relationship drilldown. Preserve approved-form requests, due diligence, protected documents, response history and ownership boundaries.

Baseline: the 9 September workspace has a sticky, independently scrolling vendor rail, per-row form counts, an empty selection panel and five detail tabs. The user supplied desktop screenshots of that composition. A larger split rail would preserve the same limitation; an all-card catalogue would make service comparisons harder. Select a metrics-led portfolio with compact, full-width service rows and dedicated detail.

## Metrics and interaction

Four metrics: loaded services, outstanding forms, awaiting review and overdue forms. Form metrics aggregate only summaries for the loaded relationship IDs; missing or refreshing summaries yield Unknown, never zero. Show the checked population and oldest summary timestamp. Search changes that population; pagination does not imply enterprise completeness. Metric actions filter form work. Criticality composition counts stored service classifications. Response review counts remain separate from vendor approval and compliance.

Desktop uses four metric cards, a two-column analytical band and a full-width register. Tablet uses two metric columns. Mobile stacks analytical sections and service rows; detail replaces the portfolio at every width, with an explicit Back to vendor register action. No decorative hero, new palette, chart dependency or animation. Use current typography, semantic tokens, shared controls and accessible text alternatives for bars.

## Demo scope

Inventory the supplied Fidelity workbooks and Ops Risk archive, preserve source coordinates and distinguish named employees from functions and generic Vendor assignments. Cloudspace/OEM retains the source-backed submitted response and unreviewed evidence gaps. Templates do not establish completed outcomes. Per the user's follow-up, all 17 people have demo logins at firstname@demo.com with password `password`; Victor Abejegah uses victor@demo.com. Each has the existing evidence-respondent/performer role only. No administrator, reviewer, signatory or approval role is granted. These accounts are restricted to the canonical demo tenant/entity and demo authenticator.

Before hosted cleanup, take a recoverable database backup and inventory exact sample scope. Preserve operational identities, authority configuration and non-demo records. Ensure subsequent releases cannot recreate excluded generic fixtures. Verify the resulting population, response state and assignments against the source manifest.

## Acceptance and implementation

1. Add failing metric and navigation tests; implement the portfolio composition.
2. Verify zero/partial/unavailable/loading/search/pagination cases, metric filters and drilldown/back navigation.
3. Repair the independent Cloudspace repeat-seed regression against an isolated PostgreSQL database.
4. Prepare exact source/person and hosted-data inventories, backup, reconcile curated fixtures, then deploy.
5. Run affected workflow and copy tests, typecheck/build, render production components at 1440/390 in light/dark and inspect keyboard/reflow. Verify hosted commit and curated data independently.

Do not claim a compliance percentage, approved vendor, reviewed evidence or verified risk closure from a submitted sample response. No new command authority is introduced by dashboard filters.
