# Home T7 acceptance — #368

Baseline: current `main` after #402, #403 and #404. Product scope: legal entity and authorized Group/OpCo Home.

## Automated evidence (not human acceptance)

- `web/scripts/capture-home-t7-evidence.mjs` runs in the existing isolated evidence-build pipeline. Receipt: `ui-evidence/home-t7.json`; eight screenshots prefixed `home-t7-`.
- Group: authorized OpCo ranking, three intent tabs, Attention isolated from CRO posture, OpCo-only drill, My work handoff, 30→90-day Loss-count change, missing Risk-source coverage.
- Entity: intent separation on the existing illustrative Home fixture (which does **not** provide governed Risk/Loss organization histories).
- Render states: desktop light/dark at 1440px, mobile 390/320px, 720px CSS-zoom 200% reflow **proxy**, and incomplete coverage.
- Browser assertions: no document horizontal overflow, no axe WCAG A/AA violation in tested fixtures, keyboard arrows between tabs, keyboard OpCo action, error-free page scripts.
- A passing artifact proves only those tested fixtures and automated checks. CSS zoom is not equivalent to actual Chrome browser zoom, axe cannot verify all WCAG criteria, and illustrative numbers are not bank-source evidence.

## Existing deterministic contracts

T1–T6 have bounded tests for exact count/drill binding, organization descendant attribution, historical hierarchy revisions, incomplete trend baselines, native currency-typed Loss/recoveries and Group legal-entity authorization. Preserve the exact source/member revision invariants; do not replace them with browser sums.

Related scale harnesses:
- `internal/runtimecontext/organization_scope_search_postgres_integration_test.go`: 20,000-node scope-selection correctness.
- `internal/httpapi/group_posture_metric_handlers_test.go`: 1,001-OpCo batched authorization and restricted-child exclusion.
- `internal/oversight/postgres_integration_test.go`: 20,000 matters and snapshots (different workload; not evidence of 20,000-node Home query latency).

These are **not** an end-to-end Home p95 benchmark. No p95 claim may be made from them.

## Pending release gates

### 1. Representative Home database performance

On real PostgreSQL with representative access policies, material Risks/Losses and a 20,000-node hierarchy:
- capture exact query plans and indexes for current scoped CRO posture, Risk/Loss concentration, retained historical Risk and period Loss flow;
- include root, middle branch and leaf scoped reads; missing/unattributed records; an organization move across captured revisions;
- measure 100+ warmed and cold reads, with p50/p95/p99, connection pool saturation and record counts;
- use actual authorized Group reads over >500 OpCos and confirm the source queries remain bounded, no N+1 and no denied-child exposure;
- compare counts/revisions with exact current drill pages and Insights.
- set an approved Home p95 budget after baseline measurement. Any query plan needing 5-minute × ancestor backfill fails by design.

### 2. Accessibility and visual acceptance

- actual browser 200% zoom, separately from CSS-zoom proxy;
- inspect screenshots and focus outlines at 320px, 390px, 720px and 1440px in both themes;
- keyboard-only tab/filter/drill, accessible list/table equivalent, 44px touch target review, high-contrast/reduced-motion and manual WCAG checks;
- verify first desktop viewport communicates current posture, exposure concentration and movement using populated, stale, partial, zero, mixed-currency, missing-history and unattributed cases.

### 3. CRO/GRC Administrator comprehension

Use representative governed (or safely synthetic but internally consistent) current+prior period data. Observe at least one CRO and one GRC Administrator performing without prompts:
1. **≤5 seconds:** identify whether enterprise Risk posture improved/worsened/is unchanged, or say when history is insufficient;
2. **≤10 seconds:** identify the area with the largest outside-appetite Risk population;
3. **≤10 seconds:** explain whether net Loss increased or decreased vs the equal-length prior period, or correctly identify mixed currencies as noncomparable;
4. one action from area bar to its exact authorized current Risk/Loss population;
5. identify missing/partial/unattributed coverage without opening data basis;
6. switch Attention/My work without mistaking their counts for Risk posture;
7. open Insights without losing authorized scope/period.
Record task completion, elapsed time, screen/recording consent, defects and findings. Do not fill outcomes from automated checks.

## Close policy

Keep #368 open until the latest-head rendered evidence passes **and** all pending release gates have dated evidence with an accountable reviewer. Do not mark the user-observation checks as complete just because the browser script passes.
