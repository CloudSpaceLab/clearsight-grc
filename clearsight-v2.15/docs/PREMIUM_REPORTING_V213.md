# Premium reporting presentation — V2.13

## Purpose

Give directors and operational leaders a clear reading order without changing the existing reporting basis:

1. The selected organizational population and reporting clocks.
2. The executive interpretation and headline measures.
3. Named decisions, accountable owners and recorded commitments.
4. Comparative charts, source confidence and accessible data tables.
5. Detailed source registers and methodology.

The implementation is presentation only. It does not introduce a calculation, aggregate, chart series, trend, rate, permission, approval, source population or storage model.

## Design decisions

### Executive hierarchy

The executive reading block uses a solid analytical surface with a restrained cobalt rule. Titles have a clearer scale, body copy has a consistent reading width, and the existing appetite context remains beside the conclusion. Headline cards use tabular numerals, readable secondary labels and equal visual weight. No value is restated or reformatted by the adapter.

Organizational breadcrumbs and reporting dates form a quieter context strip. Filters retain their existing fields, names, values and controls. Report content stays on solid surfaces; it does not add translucent backgrounds behind analytical text or charts.

### Decision and commentary blocks

Existing exception titles, ownership, due dates, rationale and management questions receive distinct typographic levels. The management question uses a subtle inset block so it can be scanned independently of the supporting explanation. Loss follow-up questions and response-accountability follow-up cards use the same rhythm.

The document-style board title, author/audience context, commentary and annex headings receive a consistent hierarchy. These are existing captured fields, not newly generated interpretations.

### Comparative charts and confidence

Labels and legends are larger and have enough line height for actual enterprise names. Values use tabular numerals and keep their original units. Narrow screens move horizontal chart tracks beneath their label/value pair instead of compressing labels into unreadable columns.

The adapter never overrides data-driven inline bar widths or column heights. The existing chart values, axis/bin labels, table equivalents, accessibility text and drill routes remain unchanged. All percentages, monetary amounts, observation counts and risk scores still come from the original renderers.

Confidence counts, applicability notes and source limitations use readable typography rather than tiny footnotes. KRI groups retain their original expansion behavior while their category labels and coverage notes are easier to scan. Wider grids step down to single-column analytical panels before labels become cramped.

### Published and printed reports

Published packs read like a board document, with the captured title preceding the metadata strip visually, an understated rule, restrained annex headings and legible commentary. The frozen renderer itself is untouched.

Print styles remove reporting controls, preserve scope/date/currency context, retain chart geometry, repeat table headers, and keep decision blocks together where practical. Long registers and chart-data tables can continue over pages. The outer live page title is omitted for the published-pack route because the captured report title is already present inside the pack.

The adapter handles native `beforeprint` and `afterprint` events:

- It temporarily opens only analytical chart-data details, methodology, KRI groups and response history inside reporting wrappers.
- It excludes editable report-builder disclosure controls. A captured preview inside the builder remains eligible.
- It restores each detail element's exact prior open/closed state after printing.
- Repeated `beforeprint` events do not overwrite the original saved disclosure state.
- It never writes `S`, snapshot objects or source records.

This makes chart equivalents and source caveats available for printing without permanently changing the reader's screen arrangement.

## Files and integration

- `assets/premium-reporting.css` supplies report-scoped presentation and print rules.
- `assets/premium-reporting.js` adds a semantic wrapper around existing dashboard/report route output and installs transient print-disclosure handlers.
- `tests/premium_reporting_v213_checks.js` verifies the presentation contract against the actual source handlers.

Load the new CSS after the existing reporting styles and global V2.13 tokens. It uses semantic colors `--surface`, `--soft`, `--ink`, `--muted`, `--line`, `--blue`, `--red`, `--amber` and `--teal`, together with the shared `--cs-space-*` and `--cs-radius-*` scale. Its selectors are confined to `.pr213-report`. Global shell, table, form and button design remains owned by the root premium layer; print-specific table layout is scoped to reporting output.

Load the new JavaScript after the established V2.12 report adapters and before bootstrap. It wraps these existing entry points:

| Entry point | Presentation class |
|---|---|
| `homeView` | `pr213-report--dashboard` |
| `reportsView` | `pr213-report--insights` |
| `loss-insights` | `pr213-report--losses` |
| `assessment-insights` | `pr213-report--responses` |
| `insight-detail` | `pr213-report--sources` |
| `report-builder` | `pr213-report--builder` |
| `report-pack` | `pr213-report--pack` |

The previous handler's returned markup is preserved in full inside the wrapper. There is no top-level render call and no new application render wrapper. `reportData`, `lrpCapture`, `arpCapture` and `brFrozenBody` are not replaced. Existing access-denied and invalid-filter views retain their controls and checks.

`CLEAR_SIGHT_PREMIUM_REPORTING` exposes only version metadata and `presentationOnly: true`; it does not add a new domain command API.

## Validation

`node tests/premium_reporting_v213_checks.js` passes **8/8** exact-source presentation contract checks:

- Loading the adapter causes no render or state mutation and preserves source-function identities.
- Dashboard and all report perspectives retain their exact previous content, values, controls and routes inside the wrapper.
- Source detail routes retain original scope, date/currency filters and access behavior.
- Published packs and read-only builder output retain their original frozen renderer and snapshot.
- Denied/invalid reporting states remain intact.
- Print disclosure changes are confined to analytical reporting content, excluding editable builder and unrelated controls.
- Disclosure restoration survives repeated print events; application state and captured output remain unchanged.
- Native print listeners do not trigger print or open content during loading.

The test constructs an in-memory baseline from the common actual-source harness while omitting only the new presentation module. It does not alter the shared harness or source files. Results are written to `tests/premium-reporting-v213-results.json`.

JavaScript syntax was checked with `node --check`. Browser layout, computed-style appearance and PDF pagination were not validated in this environment. No browser retry was attempted. The checks establish source and presentation-state preservation, not a visual or accessibility certification.
