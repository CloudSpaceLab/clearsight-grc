# Semantic domain presentation — implementation plan

Issue: #346. Parent: #265. Delivery: PR #345, documentation only.

Companion: [IT Governance Insights plan](2026-10-05-it-governance-insights.md), issue #344.

## 1. Decision and scope

Reuse the governance engines; present each business object in its appropriate shape. A submitted response is evidence, a monitoring score is not necessarily a native KRI measurement, and a Matter is intervention work rather than a substitute for every register record.

Preserve the existing Risk, Loss, Vendor and Processing Activity workspaces. Improve their gaps incrementally. Prioritize native KRI/KCI presentation, then RCSA cycle presentation, then type-aware Matter context. Small Loss identity/entry-point fixes may proceed independently; they must not delay Indicators.

Presentation patterns are design contracts, not a new framework or persisted taxonomy:

| Pattern | Primary use | Existing authority |
| --- | --- | --- |
| Register | Scan Risks, Losses, Vendors, source Projects/Assets/Incidents | Respective domain or exact Source Binding |
| Indicator | Inspect value, unit, limit, movement and breach | Monitoring/metric definitions, observations and source receipts |
| Ledger | Reconcile loss, recoveries and reversals | `internal/oploss` |
| Cycle | Inspect RCSA population, submissions and challenge | `internal/rcsa`; later the owning resilience/audit/certification domain |
| Case/intervention | Resolve an incident, exception, finding or breach | Existing Matter/Decision/Action/Evidence/Outcome |
| Program | Maintain continuing requirements, controls and evidence | Existing Program aggregate |

An object can use a register for its list and a ledger or cycle for its detail. Do not build six independent screen engines. Add shared compositions only at demonstrated reuse points.

## 2. Evidence and baseline

Code baseline: `a482a594e266cebf9102acb9599120a360245a80`. Re-read current main before each implementation PR.

The prior audit visually inspected retained Program/Branch KRI captures, not a newly authenticated production session. Other findings below are code inspection. Older screenshots and fixture tests do not establish current deployed behavior or representative-user acceptance.

| Finding | Repository evidence | Consequence |
| --- | --- | --- |
| Dedicated Loss register/record already exists | `web/src/components/losses/LossRegister.tsx`, `LossRecord.tsx`, `web/src/lossApi.ts` | Refine, do not rebuild. API client exposes list/detail/intervention, not a complete Loss authoring UI. |
| Loss list omits several supported filters | `lossApi.ts` supports event type, currency, organization and linked Risk; register uses search/status/recovery | Expose existing useful filters before adding endpoints. |
| Loss/Risk owner can render only Assigned; Loss links only Linked | `LossRecord.tsx`, `web/src/components/risks/RiskRecord.tsx` | Resolve authorized business names and distinguish missing from unavailable. |
| Native indicator measurement is absent | `web/src/riskTypes.ts`, `internal/risk/model.go`, `internal/httpapi/risk_indicator_reads.go` | Current supported measurement is MONITORING_RISK_SCORE / RISK_POINTS / 100. Native measures require a real contract, not relabelling. |
| Indicator owner/reviewer labels exist in read output | `risk_indicator_reads.go`, `RiskIndicatorsSection.tsx` | Reuse available fields; do not rebuild identity resolution. |
| KRI row opens its Program rather than exact indicator | `RiskIndicatorsSection.tsx` | Add exact indicator/result navigation; retain Program as supporting context. |
| RCSA cycle backend exists without dedicated web operating view in baseline | `internal/rcsa`, `internal/httpapi/rcsa_routes.go` | Build bounded list/detail composition over existing cycle and commands. |
| Matter uses the same four tabs and generic context | `web/src/components/MatterRecordWorkspace.tsx` | Type-specific context is missing; common work panels remain useful. |
| Program response browser has real shared improvements | `ProgramResponsesPanel.tsx`, `forms/ResponseBrowser.tsx`, `forms/AnswerValueDisplay.tsx`, `forms/ResponseAssessment.tsx` | Preserve typed answers, filters, column controls, review and document drill. |
| Historical source fields may still be long text | `cmd/seed-bank-reference/source_records.go`; `docs/design/2026-09-22-typed-response-display-and-branch-kri-register-design.md` | Typed static fixtures do not prove typed imported bank data. Map and validate before displaying native metrics. |

Retained visual references:

- [Program submitted-data desktop](../../quality/screenshots/response-browser/after/program-light-1440.png): KRI responses share form/submission/assessment columns with other responses; substantial Program context precedes the records.
- [Branch KRI answer sheet](../../evidence/2026-09-22-program-data-centered-ux/56-program-register-sheet-answers-light-1440x900.png): typed answer reading is improved, but response metadata and individual answer cards remain the dominant presentation.
- [Response-browser evidence limitations](../../quality/response-browser-2026-10-05.md).

Preserve these as historical references. Capture new exact-head before/after evidence during implementation. Do not describe this plan as a completed visual acceptance run or regulatory certification.

## 3. Ownership and dependency contract

| Work | Owner | Reuse boundary |
| --- | --- | --- |
| Semantic layouts, exact domain navigation, type-aware context | #346 | No new workflow or presentation-state store |
| External source configuration/mappings/register access | #344 / #57 | Existing Connection/View/Binding catalog and adapters |
| Shared count/money/rate/duration measures | #344 T4b, coordinated with #346 Indicator slice | One backward-compatible `metricview` extension; consume shipped #268 foundation, do not rebuild it |
| Indicator links, RCSA/Loss behavior | #270 | Existing monitoring/risk/rcsa/oploss owners and commands |
| Critical Service/BIA/tolerances | #271 | No BIA subsystem in #346 |
| Audit/certification cycles | #272 / #273 | Reuse presentation patterns, preserve each domain's lifecycle |
| Navigation and Insights | #266 / #274 | One Insights shell and one report engine |
| Breach notifications/delivery | #269 | Do not create presentation-owned episodes or notification state |

Native measurement collection and the shared measure contract must land together with a real consumer. #344 and #346 must not independently create numeric types, threshold evaluators, history tables or chart libraries. Local form-backed Indicators and RCSA do not wait for every external adapter or all eight IT pillars.

## 4. Navigation and visual rules

Target remains Home, Work, Portfolio, Insights and Configure under #266. Preserve the currently exposed Reports destination and all deep links until an equivalent authorized entry exists and navigation consolidation is approved; this plan does not authorize hiding Reports.

Portfolio provides persistent record/cycle access. Insights provides cross-scope indicators and movement. Work provides intervention. Indicator ownership/setup can be reached contextually from existing Risk/Program records; do not add a separate top-level KRI application.

For each selected record or view:

- Show one identity header, scope and relevant period; avoid repeating the Program list heading above an exact domain record.
- Lead with the business value/state and current actor action. Put revisions, hashes, parser details and technical lineage in expandable source/history detail.
- Use shared DataTable for comparison and scanning, not one card per field. Keep first-view columns limited to the user's task; optional columns use existing controls.
- Make money/counts visually comparable through shared numeric formatting, right alignment and tabular numerals. Dates and durations need explicit semantics and units.
- Use stable named metrics and order within a role/lens. Do not reshuffle cards on every refresh.
- Reserve semantic color for the relevant state; Active, Assigned, Submitted and Linked are not evidence of acceptable risk or verified outcomes. Include text/icon, not color alone.
- Preserve quality independently from business condition. A stale last-known breach remains identifiable as last-known; a source failure cannot imply current Normal.
- Use domain-specific date labels: Occurred/Discovered for losses, Reporting period/Observed for KRIs, Assessment period for cycles. Do not substitute submission date for business period.
- At narrow widths use labelled stacked records and focused sections. Dense comparisons may use a contained accessible table viewport, never whole-page horizontal overflow.
- Preserve selected scope, filters, period and return target when drilling to evidence or Work. History/back must return to the original record, not always to a generic Program list.

No new palette, glass treatment, token set, density mode or per-domain typography. Reuse existing components and design tokens.

## 5. Indicator presentation and measurement

### 5.1 Primary view

Default columns: Indicator, Scope/owner, Current value, Limit/target, Condition, Movement, Updated. Supporting detail exposes source coverage, reporting period, check revision, linked Risks/Controls, exact response/result and intervention.

Native value and unit lead. The current 0–100 concern score remains explicitly labelled supporting context when it exists. It is neither a probability nor a substitute for the source quantity. Where a check only provides a concern score, show Concern score and Native value not recorded rather than inventing a native measure.

Not every KPI is a KRI/KCI. Preserve the configured measurement purpose and Risk/Control relationship; do not reclassify all IT operational metrics as risks merely to reuse the layout.

### 5.2 Contract to implement in existing owners

Retain stable indicator/check identity, exact definition/result/source references, scope, observation time and reporting period. Add only the missing typed measurement and limit semantics:

- COUNT, RATIO/PERCENT, DURATION and MONEY through the single shared measure extension; preserve explicit RISK_POINTS for legacy score-only checks.
- Exact numeric representation; currency and scale for money, duration unit, ratio numerator/denominator where known.
- High-is-bad, low-is-bad and outside-range conditions with inclusive/exclusive boundaries and effective threshold revision.
- Current business condition separately from freshness/completeness and last-known condition.
- Comparison period, comparable definition/unit, absolute delta and percentage-point delta where appropriate.
- Existing episode/work reference; do not infer breach duration from the latest sample or open a new episode per row.

Calculate and validate on the server. Do not infer threshold policy from field labels or derive a native value backwards from a concern score. Source/manual forms continue to use existing collection and maker-checker mechanisms.

### 5.3 Trend and aggregate rules

Use a compact labelled trend or threshold marker where it helps; no decorative gauges or smooth curves implying unobserved samples. Provide a table alternative. Gaps remain gaps; no forward-filling stale data as current. Mark threshold, population or definition changes; comparisons require compatible periods and definitions.

Compute aggregate rates from valid sums of numerators and denominators, not the mean of percentages. Do not sum currencies without an approved exchange-rate basis. Zero/unknown denominators remain Not calculable/Unknown as appropriate. Do not total the currently loaded page as the enterprise population.

### 5.4 Branch/head-office returns

One submitted return may contain many indicators and non-indicator context fields. Use reviewed mappings from exact template/field revisions to indicator, scope and period. Never assume one response equals one indicator.

Preserve original answers/source cells and submission history. Long-text source values need validated parsing, unit, period and organization resolution. Invalid/ambiguous values remain unmapped or unknown for review. Do not rewrite historical answers, infer a missing year, default currency silently or promote a sample response to current approved truth.

Duplicate or amended returns must not double count. Latest submitted, reviewed and effective measurement are distinct; the bank's approved selection rule controls which one contributes.

## 6. RCSA cycle presentation

Use `internal/rcsa` and its existing first-line/challenge commands. Add the missing bounded list API and typed web client only after confirming current routes; do not synthesize a cycle by searching all Programs or Matters.

List columns: Cycle/period, Scope, Current stage, Population, Submission progress, Challenge progress, Due/next action. Show counts only where the backend establishes the numerator and denominator. A single cycle-level response cannot be represented as 81/84 individually reviewed risks without item-level evidence.

Detail leads with period, frozen population, accountable owner and current handoff. Reuse shared tabs for Overview, Assessment, Challenge and Evidence/history where useful. Display exact Risk/Control population, submitted assessment and linked challenge/interventions without duplicating their editors.

Population checksum and internal revisions belong in expandable source detail. Collection complete, challenge complete, risk accepted and remediation verified are different states. A changed source Risk must not silently alter the frozen cycle; display the difference and route it through existing governed handling.

Do not create RCSA tasks, approvals, schedules or deficiency ledgers in the presentation layer.

## 7. Type-aware Matter context

Keep one Matter workspace and shared Actions, Evidence, Decisions, Outcome and activity components. Add a small exhaustive typed presenter map alongside existing `matterPresentation` helpers only when the first consumers land. No generic schema renderer or unused descriptor framework in T0.

Initial contexts:

| Matter type | Context shown before shared work |
| --- | --- |
| INCIDENT | Available source incident/service, severity, event time, SLA and root cause |
| EXCEPTION | Requested deviation, scope, compensating controls, expiry and conditions |
| AUDIT_FINDING | Finding, rating, management response, remediation and outcome |
| KRI_BREACH | Exact indicator/result, native value, limit and known breach episode |
| REGULATORY_CHANGE | Governing source/provision, effective date and affected Program |
| OPERATIONAL_LOSS | Named canonical Loss, gross/recovered/net and related recovery work |

Resolve context from typed authorized source/domain links, not title matching or arbitrary JSON guesses. Missing source fields remain Not recorded; unavailable links remain unavailable. Do not add empty widgets for data the current model does not own.

Section visibility must combine relevance, existing content and server-authorized operations. Never hide recorded evidence, decisions, response packages, outcome checks or required actions merely because a type descriptor omits them. Keep stable tab labels/IDs where possible; use domain context rather than making users relearn work navigation for every category. Unknown/legacy types receive the complete safe generic fallback.

Separate the record's owner, current actor action, approver and reviewer. The presenter cannot grant or revoke authority. Lazy-load external context and optional panels; unrelated vendor panels need not issue requests for an unrelated case unless a real link/content/action exists.

## 8. Loss and Risk refinements

Preserve the dedicated register/ledger and current reconciliation semantics. First expose useful existing filters and resolve authorized owner, organization, Risk and intervention names. When a name cannot be read, distinguish Name unavailable from Not assigned; do not expose private identifiers as substitute labels.

Gross, recovered and net remain exact amounts by currency. Do not label all net loss Outstanding recovery: a remaining accounting loss is not necessarily legally recoverable. Show Recoverable outstanding only when the model has an approved recovery target; otherwise keep Net loss. Recovery percentage is derived only where meaningful and never implies remediation closure.

Treat loss-record lifecycle, financial recovery state and intervention lifecycle independently. Do not label a linked Matter Open solely because `matter_id` is present.

The baseline Loss web client lacks create/recovery calls. Before removing generic Operational loss creation, implement or confirm an authorized Record loss / Record recovery path using existing backend commands. Only then redirect new loss declarations to it. Preserve existing Matter URLs and standalone legacy interventions. Never auto-create financial entries from their title/body or erase historical records.

Risk refinement uses current assessment kind/method and appetite contracts. Do not equate most-recent TARGET/STRESSED assessment with current residual exposure, average ratings, or put incomparable methods on a single risk matrix.

## 9. Shared components and data budget

Reuse MetricCard, DataTable, FilterBar, StatusBadge, Surface, Tabs, FocusedSheet, EmptyState, current handoff, named-person/link components and typed answer rendering first.

Candidate shared compositions such as native Indicator value/limit, bounded trend/table and Cycle progress are justified only by an actual second consumer. Extract incremental duplication from existing components rather than introducing a base class, plugin registry or universal record component. Each domain retains a small typed composition; no file grows into a switch-based application.

No new presentation/archetype tables, source replicas, workflow engines, schedulers, dashboards or notification stores. Typed measure/source history changes may be necessary: document owner, authoritative basis, retention, reconstruction and cardinality before migration. A zero-table target must not justify losing audit history or hiding business truth in arbitrary JSON.

Resolve labels/links in bounded batched reads before pagination where authorization requires it. No per-row API loop or full-population browser aggregation. UI summaries and exact drills must share the same revision/filter/scope; current-state source reads must be labelled as current, not immutable historical membership.

## 10. Delivery sequence and small PRs

| Slice | Deliverable | Exit gate |
| --- | --- | --- |
| S0 — audit and decision brief | Exact-head inventory, baseline screenshots, data provenance and primary-record map | Scope agreed; no production schema/UI or unused framework |
| S1 — native measure + first Indicator | One shared measure extension; source and form mapping; Indicator value/limit/detail integrated with current Risk context | Score-only fallback and two materially different native indicators pass |
| S2 — KRI operating view | Bounded scoped Indicator list/trends in existing Insights shell; exact return to evidence/work | Branch/head-office and source-backed measurement acceptance; no new KRI app |
| S3 — RCSA cycle | Bounded cycle list/detail, first-line/challenge handoffs | Frozen population and independent challenge remain authoritative |
| S4 — Matter context | Initial type presenters over unchanged work panels | Required commands/content remain reachable; safe fallback |
| S5 — Loss/Risk refinement | Named links, existing filters, usable Loss creation/recovery path before redirection | No lost entry point; monetary and lifecycle semantics unchanged |
| S6 — later adoption | #271/#272/#273 use agreed service/cycle/case compositions | Owning domain is implemented; no placeholder business facts |

S5's small read/label fixes may run in parallel. S1/S2 do not wait for all #344 connectors. S3 does not wait for #271. The #344 frontend consumes these components instead of forking them.

Within each slice commit separately: contract/regression tests, bounded backend/client, shared presentation, domain composition, rendered evidence. Reconcile current main before every PR and record overlaps/merged work. Do not merge or deploy merely because this plan is approved.

Suggested touched paths, not mandatory scaffolding:

- `internal/metricview`, `internal/monitoring`, `internal/risk`, `internal/httpapi/risk_indicator_reads.go` for the shared native measure and Indicator reads.
- `web/src/riskTypes.ts`, `components/risks/RiskIndicatorsSection.tsx` and existing Insights routing for presentation.
- `internal/rcsa`, `internal/httpapi/rcsa_routes.go`, a bounded RCSA client and small cycle components for S3.
- `web/src/matterPresentation.ts`, `components/MatterRecordWorkspace.tsx` and existing handoff/work panels for S4.
- `internal/oploss`, `web/src/lossApi.ts`, `components/losses/*`, `components/MatterSetupWorkspace.tsx` for S5.
- Existing import/capture/response owners for reviewed historic mappings; do not place business conversion in React.

## 11. Tests and acceptance

### Correctness and authority

- Native quantity differs from concern score; score-only records remain honest.
- High/low/two-sided limits, boundary equality, signed movement and percentage-point deltas.
- Currency scale, reversals, zero loss/zero denominator, missing/invalid amounts and mixed-currency totals.
- Reporting period differs from submission time; amendments, duplicate rows, missing periods and version changes.
- Current, stale, partial, unknown, unavailable and restricted states remain distinct.
- One indicator linked to multiple Risks counts once in an indicator population; a Risk count remains a separate metric.
- Exact historical drill remains fixed, current-state drill is labelled, permissions are rechecked on every detail read.
- Frozen RCSA population, first-line submission, independent challenge and governed outcome are not conflated.
- Matter type changes cannot hide mandatory steps or disclose restricted parent records.
- Loss declaration remains possible before any generic-entry redirection; existing links/bookmarks still resolve.

### Rendered proof

Capture real shared components in the app shell at desktop light/dark, mobile 390/320, keyboard and actual browser 200% zoom. Record reflow proxies separately; they are not the same test. Include loading, empty, unavailable, permission loss, long names, large values and partial-source states.

Review information hierarchy, numeric alignment, reachable filters, visible focus, contained table scrolling, chart alternatives and primary-action obstruction. Do not substitute axe results for visual inspection or bank-user comprehension. Include persisted representative imports; do not prove typed data only with hand-typed static fixtures. Keep confidential source-bearing artifacts out of public evidence.

### User tasks

Use representative bank users; proposed acceptance targets, not completed measurements:

1. Within 10 seconds identify a KRI's native value, limit, period and current quality.
2. Within 10 seconds distinguish loss gross/recovered/net from whether its remediation is closed.
3. Within 30 seconds identify the current RCSA handoff and which population the cycle assessed.
4. From an Indicator/Loss/Cycle open the exact related work or evidence and return without losing scope/filters.
5. Recognize when data is stale/unavailable without reading implementation metadata.

Record task times, errors and required assistance against before-state. No capability is marked complete solely because a backend exists, an issue checkbox was checked, or a static screenshot looks correct.

## 12. Completion evidence

S0 documentation can be reviewed independently. Implementation completion requires merged commit, executed tests, exact-head renders, deployed journey evidence where applicable, and representative-user acceptance. #346 remains open until these gates pass. #344 remains the IT-source/Insights delivery tracker, not a duplicate semantic-presentation backlog.
