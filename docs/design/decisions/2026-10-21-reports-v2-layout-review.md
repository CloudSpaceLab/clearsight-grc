# Reports V2 layout adaptation — review receipt

Date: 2026-10-21. Tracking: #462. Implementation: PR #578, route `#reports`.

**Status: partial V2 layout integration. The 0/19 whole-screen acceptance register and production cutover block remain unchanged.** This tranche adapts the complete Overview composition and Reports library. Named release-reviewer approval and the four-principal operational proof in `docs/acceptance/issue-462-actor-report.md` remain outstanding.

## Design authority and baseline

- Original HTML: `docs/design/references/clearsight-2026-10/source/clearsight-reports-insights-designs/reports-workspace-preview.html`.
- HTML SHA-256: `97f9d04092fdad07e5081820ad913295dc17d8344a693bd229c52d521c5b8895`.
- Companion notes: `reports-workspace-design-notes.md`, SHA-256 `1ac00ba1e97ef6f574ae24f4a974810fbbe94f15a48c9644b668ba18b37c47a5`.
- Before-state: PR #578 head `e1fd967993b7ce17fade9c4444c252c96a34f0c5`, UI/UX run [39571684013](https://github.com/CloudSpaceLab/clearsight-grc/actions/runs/39571684013), artifact `12332448662`. Downloaded ZIP SHA-256: `4e6ca733998ddd840331eefdb80fe8a929dfbf83a61d6c1b1791a011aa5ad137`.
- The before-state CI was green, but its rendered metric labels wrapped into one or two letters per line; mobile metric cards displaced the first analysis panel. The 1280px rail collapsed too early, the metric grid retained four columns, and shared charts had a second card frame.

## Decision and resulting layout

Use the archived specialist HTML for composition and responsive replacement behavior; keep the verified shared shell, tokens, scope and controls from `DESIGN.md`.

The page retains its heading/actions and five local destinations: Overview, Explore, Reports, Schedules and Build report. Overview now places four measures before the paired Risk concentration / Assessment coverage and Loss movement / Priority findings panels, followed by recent reports. The context and saved-setup rail occupies the source column position. Real report scope and time semantics replace illustrative Copilot and saved-analysis content.

The Reports destination replaces the old summary strip and wide table with search/status/area filters, selectable report records, and a persistent selected-report inspector. Selection, older-history loading, generation, protected download and detailed source inspection use the existing run/definition state and APIs. There is one run store, one generation flow and one source-details sheet.

| Viewport | Overview measures | Context rail | Analysis layout | Library inspector |
| --- | --- | --- | --- | --- |
| 1440×900 | Four columns | 260px beside main | 1.1 / 0.9 primary pair | 315px beside records |
| 1280×800 | Two by two | 245px beside main | Equal pairs | 315px beside records |
| 1024×768 | Two by two | Below main, two columns | 1.1 / 0.9 pairs | Beside records |
| 768×900 | Two by two | Below main, two columns | Equal pairs | Beside records |
| 390×844 | Two by two | Below main, one column | Stacked | Above records |

Shared MetricCard heading/label tokens prevent narrow labels from competing with quality text. The reusable RankedBarList inline layout retains shared paint, focus and actions. Embedded chart sections remove the duplicate frame and expose a header action. Compact monetary summaries use BigInt currency scaling; exact amounts remain visible in Loss movement and Explore. Mobile values use the existing 22px token.

The UI contract checker now permits feature-scoped geometry and registered component geometry tokens. It still rejects global copies, palette/focus/state overrides, unregistered component-token overrides, `!important`, hidden shared semantic boxes and clipped focus/content. New Reports components and their focused stylesheets are registered with this gate.

## Data and action boundaries

| Surface | Authority and behavior |
| --- | --- |
| Outside appetite | Existing Domain metric `risks_outside_appetite`; exact requested entity/organization area, source and revision. Home supplies the governed reporting period, not the risk value. |
| Risk concentration | Existing organization-breakdown endpoint; source/revision/scope/count must match the current Domain bundle. Old-scope results disappear during render and late aborted results cannot replace the current scope. Zero assessed population remains qualified. |
| Loss | Existing loss-period endpoint with the verified Home interval and selected area. Mixed currencies are not combined. Malformed presentation fields become unavailable rather than crashing the page or implying an unchanged comparison. |
| Reports ready | Loaded runs with READY status and a valid future expiry only. Cursor presence marks the loaded list as partial. The same predicate governs available counts, filters and download actions. |
| Report library | Existing legal-entity report APIs. The library and context rail explicitly identify that scope even when analytics use a selected area. |
| Refresh | Reloads report definitions/history and the metric basis; an explicit refresh token also reloads domain/loss reads when dates are unchanged. |
| Navigation | Reports participates in the shared organization-scope selector. Reports ↔ Insights preserves the verified entity, area and period. |
| Assessment coverage / findings / schedules | No fabricated totals or controls. Coverage states its missing totals; Programs and Work remain actionable. Recurring delivery remains unavailable. |

## Rendered evidence and regression checks

The focused runner is reproducible from `web/` after building the evidence application:

```sh
npm run build:evidence
# Start preview:evidence on PAGE_URL, then:
UI_EVIDENCE_SCOPE=reports node scripts/capture-ui-evidence.mjs
UI_EVIDENCE_SCOPE=reports node scripts/capture-approved-source-references.mjs
```

The capture runner records 32 Reports cases: loaded Overview in both themes at all five widths; unavailable, partial, denied and selected Risk Operations states; library/inspector in both themes at all five widths and mobile records; report source details; failed generation; saved setups and generation setup. Fixtures remain isolated behind the evidence entry point. Their risk buckets reconcile to 4/12 at entity scope and 2/5 in Risk Operations; the exact loss is NGN 10,400,000.00. These are explicit test values, not bank-operational proof.

Browser assertions check actual metric rows and rail location, card height, label line count, computed value font size, text containment, one chart frame, report selection/inspector ordering, matching source totals and scope/period API reads. They reject page-level horizontal overflow. The mobile first viewport contains all four measures and the first risk rows.

Measured cards are 125px high at 768–1440px and 137px at 390px, with values at 28px and 22px respectively and no more than two label lines. Recent reports uses the existing container-responsive DataTable contract; mobile row actions fill their 258px cell and remain 44px high.

| Original source capture | Matching runtime capture |
| --- | --- |
| `approved-reports-insights-1440x900.png` | `140a-report-overview-light-1440x900.png` |
| `approved-reports-insights-1280x800.png` | `140h-report-overview-light-1280x800.png` |
| `approved-reports-insights-1024x768.png` | `140f-report-overview-light-1024x768.png` |
| `approved-reports-insights-768x900.png` | `140i-report-overview-light-768x900.png` |
| `approved-reports-insights-390x844.png` | `140b-report-overview-light-mobile-390x844.png` |
| `approved-reports-library-1440x900.png` | `140-report-library-light-1440x900.png` |
| `approved-reports-library-1280x800.png` | `141d-report-library-light-1280x800.png` |
| `approved-reports-library-1024x768.png` | `141f-report-library-light-1024x768.png` |
| `approved-reports-library-768x900.png` | `141h-report-library-light-768x900.png` |
| `approved-reports-library-390x844.png` | `141b-report-library-light-mobile-390x844.png` |

Original captures are hash verified and activate the same local view. Light-source/dark-runtime comparisons are used only for layout, not theme identity. The UI/UX workflow retains these pairs, manifests and logs as its review artifact.

Local verification before the final focused fixes: 255 web test files / 1,915 tests passed. After those fixes, seven affected suites passed 86 tests, including report generation/paging/download denial, expiry, refresh, scoped requests, malformed responses, isolated fixtures and copy quality. UI contracts, runtime fixture boundary, typecheck and customer/evidence builds also passed. The PR must independently pass its final-head CI and UI/UX workflows.

Browser keyboard checks cover arrow-key activation of all five local tabs, Space/Enter report selection, details-sheet focus containment, and Escape returning focus to its trigger. Library search produces an honest no-results state and clears back to the loaded records. The Overview and Library axe checks at 1440 and 390px found zero automated WCAG 2 A/AA and 2.1 AA violations; all exposed mobile action targets meet 44×44px after correcting the recent-report action layout. Automated checks are not a whole-product accessibility certification.

## Remaining acceptance work

Explore retains the existing governed risk/loss workspace, and Build report retains the existing approved setup workflow; their complete specialist source compositions are not accepted by this receipt. Source Copilot suggestions, saved analysis, requirement denominators, a dedicated findings population and recurring delivery still need their authorized contracts. Full library history remains paged rather than an invented enterprise total.

Fixture denials and component tests do not establish server authorization, four-principal execution, issued-report/source reconciliation, or a production cutover. A named reviewer must approve the complete Reports page, responsive behavior and actual backend results before its screen status can advance from the unchanged 0/19 baseline.
