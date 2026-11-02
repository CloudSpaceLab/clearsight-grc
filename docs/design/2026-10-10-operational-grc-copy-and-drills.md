# GRC operational copy and drill acceptance

Scope: Program portfolio, Risk and Loss registers, RCSA cycles, and indicator history.

## Copy

- Headings identify a business object or action: Program status, Most open issues, Risk appetite, Financial impact, Cycle status, Value history.
- Scope is explicit. Counts refer to displayed records; no sample is presented as the institution-wide total.
- Unknown and stale assessments stay separate from confirmed current conditions.
- Remove implementation narration, duplicate counts, generic "loaded" metadata and chart explanations that do not affect a decision.
- Exact specialist details (check version, revision, source and period) remain accessible in the record.

## Actions

| Starting point | Action | Expected result |
| --- | --- | --- |
| Program status | Select specific condition | Existing Program filter selects that operating state |
| Program issue ranking | Select Program | Its Issues and Actions section opens with existing filters preserved |
| Risk appetite | Review outside/near appetite | Server-filtered Risk register; selected organization scope retained |
| Loss financial impact | Show currency losses | Currency-filtered Loss register; other filters and organization scope retained |
| RCSA cycle stages | Review challenge/first line | Existing status-filtered cycle register |
| Indicator observations | Review observation | Exact dated record, native value, source result, and condition |

No new API, state engine, design-system family or inferred cross-currency/portfolio totals.

## Release checks

- Copy acceptance in visible and accessible labels; records remain named and contextual.
- Empty, unknown, stale, incomplete, paginated and filter transitions.
- Light/dark, 390 px and 1440 px render checks, keyboard/contrast, no page overflow.
- Domain-specific regression assertions and required backend/web/UI review gates.
