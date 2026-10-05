# IT Governance Insights implementation plan

Issue: #344. Delivery: PR #345, documentation only.

Plan branch base: `a482a594e266cebf9102acb9599120a360245a80`. The earlier source/metric audit used `ca12fa5f327c445726b00fae0853e53ce022b7e2`; its delta to this base was reviewed. Reconcile current main before each implementation PR.

Companion: [Semantic domain presentation plan](2026-10-05-semantic-domain-presentation.md), issue #346. That plan owns native Indicator, RCSA Cycle and type-aware Matter presentation. This plan owns external source composition and the IT Governance use cases consuming those presentations. Do not build parallel implementations.

## 1. Outcome and product boundary

Meet the Fidelity IT Governance requirements across projects, ITSM, change/release, assets, budget, disaster recovery, channels and staffing while preserving ClearSight's existing operating model.

ClearSight owns governance interpretation, approved thresholds, Risks/Controls, service/BIA targets where implemented, evidence, decisions, interventions, metric observations and reproducible reports. The specialist systems retain their operational populations and transaction detail:

| Population | Source authority | ClearSight presentation |
| --- | --- | --- |
| Projects and delivery | Jira / Azure DevOps / PMO | Source-backed register; delivery/financial Indicators |
| Incidents, problems, changes | ManageEngine | Source-backed register; SLA/change Indicators; Matters only for intervention |
| Release/deployment outcomes | ManageEngine / Azure DevOps | Shared change/release register and Indicators |
| Architecture exports | Sparx EA / approved file service | Exact ChangeID artifact lookup |
| Assets | Axonius / HardCat / ManageEngine | Source-backed register; patch/lifecycle Indicators |
| Budget, actuals, commitments | ERP | Financial measures by existing organization/project scope |
| Backup/test observations | Veeam / ASR / Zerto | Indicators against #271 governed service/BIA targets |
| Channel performance | Core banking monitoring / NIBSS / Switch | Bounded aggregate series and Indicators |
| Employment/recruitment | HR source | Staffing aggregates; existing people/positions provide governance context |

SCIM account activation/deactivation is not automatically an employment join/exit. Position count, occupied positions and distinct people are different measures. The supplied organogram is organization input, not another budget or staffing hierarchy.

Target shell under #266: Home, Work, Portfolio, Insights and Configure. Reports retain their existing authorized destination and deep links until an equivalent route and approved consolidation exist. This plan does not authorize hiding Reports or adding eight primary destinations.

## 2. Reuse inventory and verified limitations

### Sources and checks

Reuse `internal/evidence` Source Registry and health; `internal/sourceaccess` Connection/View/Binding revisions; `internal/assurance` typed conditions and source-side evaluation; and `internal/monitoring` form/source checks, coverage, freshness, owners/reviewers and adverse-result handling.

PostgreSQL, REST/JSON, tabular-artifact and webhook/event adapter code exists. INSPECT/PAGE/LOOKUP/AGGREGATE/CHANGES are contract capabilities, not a promise that every adapter implements all operations. Verify exact adapter/version capabilities, authentication, pagination, rate limits and historical semantics for each bank source. A generic REST adapter does not prove a deployed ManageEngine/Azure/ERP integration.

Existing bindings already retain selected fields, keys, mapping, parameter/output schemas, limits, freshness and completeness. The current Configure surface does not expose the full catalog; production activation must be re-audited rather than inferred from lifecycle enum values or seed installers.

### Governance records and UI

Reuse Programs, Matters, Actions, Decisions, Evidence, verification, Risks/appetite, reusable Controls, KRI/KCI links, RCSA cycles, operational Losses, organization hierarchy and positions. Reuse the dedicated Risk/Loss/Vendor/Processing Activity views and shared response browser. Do not copy every external row into Program or Matter.

Current KRI links expose a normalized monitoring risk score, not arbitrary native quantities. Historical imported form fields may be long text even when typed static fixtures look correct. The semantic plan specifies reviewed mapping, score-only fallback, period and quality handling.

### Metrics, attention and reports

Reuse `internal/metricview` definitions, exact count memberships, observations, trends, rollups, matrices and safe Group foundations. The baseline durable metric vocabulary is count/current-posture oriented. Non-count measures require one deliberate compatible extension, not another KPI engine.

Reuse #269 episodes/delivery and the existing runtime, timers, outbox/inbox. Reuse the governed report engine, immutable definitions/runs, CSV/NDJSON/XLSX, checksums and protected downloads. Source-backed metric/report datasets, scheduling and bank-specific acceptance remain actual work.

## 3. Implementation constraints

- No Project-management engine, ITSM/CAB engine, CMDB, finance ledger, channel transaction warehouse or HR system.
- No second source registry, scheduler, workflow, notification, metric, reporting or generic dashboard framework.
- No raw-population replication by default and no full-population browser aggregation.
- No automatic one-source-row to one-Matter or one-Program conversion.
- No new schema merely to select a presentation. Any genuine history/identity/performance gap needs documented ownership, retention and reconstruction before migration.
- Source failure, schema drift, missing periods and incomplete population cannot become green or zero.
- Presentation never grants authority, manufactures domain facts or hides required commands/evidence.
- No separate correlation service while explicit source identity and mappings suffice; do not assume identity/cardinality that production data has not established.
- Use approved deterministic data interpretation first. AI does not generate material status or arbitrary source SQL.

## 4. Shared ownership with #346

| Responsibility | Owning implementation |
| --- | --- |
| Source onboarding, catalog UI, operational source reader | #344 / #57 |
| Native values, units and compatible measure persistence | One shared `metricview` extension: #344 T4b with #346 S1 |
| Native Indicator value/limit/trend presentation | #346; #344 reuses it for IT metrics |
| RCSA list/detail and semantic cycle presentation | #346 / #270 |
| Type-aware Matter context and safe canonical-parent navigation | #346 |
| Source-backed Project/ITSM/Asset register | #344, reusing DataTable and #346 visual contracts |
| Critical Service/BIA/tolerance authority | #271; #344 adds recovery telemetry composition |
| Insights navigation/report composition | #266 / #274; one shell/engine |
| Attention/notification state | #269, not a view-owned copy |

Delivery priority for semantic work is KRI/KCI, then RCSA, then Matter context. Local form-backed Indicators and RCSA need not wait for all eight IT source integrations. Small Loss name/filter fixes may run in parallel. Use the existing canonical Loss ledger for loss metrics and drills; never recreate it as an external finance module.

## 5. Source mapping and identity contract

### 5.1 Exact Binding metadata

Use existing `BindingRevision.Mapping` for a bounded, versioned interpretation schema. Keep source data shape distinct from UI presentation pattern; neither creates a new persisted presentation object.

Example, to validate against actual source fields:

```json
{
  "schema": "clearsight.it-governance.binding.v1",
  "lens": "PROJECTS",
  "shape": "REGISTER",
  "fields": {
    "external_id": "project_id",
    "change_id": "change_id",
    "display_name": "project_name",
    "owner_ref": "project_manager",
    "organization_ref": "department_code",
    "status": "rag_status",
    "planned_start": "start_date",
    "planned_end": "end_date",
    "budget_amount": "budget",
    "actual_spend": "actual_spend"
  },
  "units": { "budget_amount": "NGN", "actual_spend": "NGN" }
}
```

Initial data shapes: REGISTER, METRIC_SERIES and ARTIFACT_LOOKUP. Initial bounded lens codes: PROJECTS, ITSM_INCIDENTS, ITSM_CHANGES, RELEASES, ASSETS, IT_FINANCIALS, DR_TELEMETRY, CHANNEL_PERFORMANCE, WORKFORCE and ARCHITECTURE_ARTIFACTS. Add problem records only as a real source consumer, not an empty module.

A small parser belongs in the existing source consumer boundary, or a focused `internal/itgovernance` package if that is the smallest viable composition. Reject unsupported schema/lens/shape; require selected-field/schema membership, stable keys where needed and explicit unit/type semantics. Do not create a generic ontology or duplicate existing mapping infrastructure.

### 5.2 Semantic roles

| Family | Initial roles |
| --- | --- |
| Common | external_id, display_name, organization_ref, owner_ref, status, severity, observed_at, updated_at |
| Project/change | change_id, project_manager, planned_start, planned_end, actual_end, rag_status, budget_amount, actual_spend, committed_spend, forecast_at_completion, sprint_velocity, backlog_size, benefit_status, change_type, change_risk, approval_state, implementation_result, release_id |
| ITSM | incident_id, incident_priority, sla_target, sla_breached, major_incident, root_cause_category, opened_at, resolved_at |
| Asset | asset_id, asset_type, location, lifecycle_status, patch_age, patch_compliant, support_end, criticality |
| Resilience | service_ref, rto_target, rpo_target, actual_recovery_time, backup_status, last_dr_test_at, dr_test_result |
| Channel | channel, reporting_period, transaction_count, failed_transactions, success_rate, downtime_minutes |
| Workforce | person_ref, position_ref, employment_state, joined_at, exited_at, vacancy_state, recruitment_state |

Mappings interpret explicitly approved source facts; they cannot widen the activated Binding, infer authority or silently convert source declarations into ClearSight-approved targets. Currency, duration and reporting-period semantics must be validated, not guessed from labels. An unknown unit or organization match remains unresolved.

### 5.3 ChangeID and related records

Use ChangeID as the supplied cross-source change reference where validated. Retain native project/release/cost-record keys: one project may contain multiple changes, and one change may have several releases, artifacts or spend entries. Do not join many-to-many tables in a way that duplicates budgets or incidents.

Namespaced keys include source and verified tenant/legal entity. Preserve exact source case/format unless an approved mapping specifies normalization. Missing, duplicate or ambiguous keys block automatic linkage; no fuzzy name matching. Introduce alias/reconciliation state only if real source history proves it necessary.

## 6. Tranches T0–T3: source access and initial consumers

### T0 — prove the contract before extending storage

Read current catalog, assurance, monitoring, metric, report and organization APIs. Document actual capability gaps. Add focused mapping/schema tests for Project, ITSM, Asset, Channel and Workforce examples; no speculative framework or presentation types without consumers.

Candidate paths: `internal/itgovernance/mapping.go` and tests only if existing packages cannot host the consumer cleanly; a compact architecture note when implementation begins. No new production table expected. PR #345 provides planning, not those tests or runtime changes.

Exit: five representative source shapes validate through the same binding contract; missing keys/types/units/scope fail explicitly; no duplicate source engine.

### T1 — make the current Source Catalog operable

Reuse registered catalog list/create/inspect/preview/where-used routes. First verify whether a complete safe activation path exists. If missing, extend current revision lifecycle and authority machinery, not a new approval engine.

Required controls: maker/checker, exact base revision, valid parent connection/view, current schema fingerprint, verified capabilities, one effective current revision, effective dates, audited pause/retire and rollback by new revision. Inventory and pending drafts remain visible when the remote source is unavailable.

Under Configure → Data & integrations, add modular inventory/detail/preview/usage compositions. Reuse DataTable, StatusBadge, Notice, FocusedSheet and existing form controls. Do not pre-create a file/component for every catalog noun unless the actual screen requires it. Preserve Program-level `DataSourceBuilder` as the simple contextual source path.

Secrets remain opaque deployment-owned references. Do not render credentials or secret references. Preview obeys field/row/byte/time limits; current authority controls lifecycle actions.

Exit: supported source setup is usable; exact revision and where-used survive navigation/reload; unavailable/forbidden/conflict/mobile states pass.

### T2 — one operational source-backed register reader

Ordinary consumers must not call the admin preview API or gain CONFIG_READ. Resolve only recognized, current, approved bindings within verified source/legal-entity scope, then enforce explicit aggregate and row-detail permission separately.

Binding discovery uses existing Purpose/Mapping and source ownership with bounded queries; add an index only if query plans justify it. Do not add a lens registry table or allow arbitrary browser Binding IDs/SQL.

Proposed route family, subject to existing route review:

```text
GET /api/v1/insights/it-governance/lenses
GET /api/v1/insights/it-governance/{lens}/records
GET /api/v1/insights/it-governance/{lens}/records/{key}
```

Response: safe mapped columns/rows, exact Binding/source provenance, generated/observed time, capabilities, quality, explicit next cursor and permitted detail target. No connection configuration. Cursor binds scope, filter, selected revision and ordering; reject stale or cross-scope reuse.

Server-side search/filter/sort is capability driven. Unsupported operations remain unavailable; never fetch the entire REST/artifact population to imitate them. Row-level access is checked before returned counts/pages; Group aggregate access never authorizes child records.

Use one shared DataTable-based register composition with small typed domain columns. Source quality and drill identity are shared. Verify Projects, ITSM incidents and Assets as three different consumers before broader expansion.

Exit: bounded source-backed pages, safe failure/recovery, no replicated operational row tables, no per-row network loops and no client totals masquerading as full population.

### T3 — Projects, ITSM, change/release and architecture

Projects show supplied identity, manager, organization, RAG, dates, budget/actual/forecast, velocity/backlog and benefits where available. A project becomes a Program only when ClearSight actually governs continuing requirements/checks, not because a row exists.

ITSM retains the complete external denominator; only material governance exceptions create/link Matters under approved policy. Change and release share the source-backed reader. Preserve Standard/Normal/Emergency, implementation/backout outcome, approval state and affected application/service. Do not clone CAB or deployment workflows.

Relationships use explicit source/domain identifiers and verified scope. Add durable links only where existing metadata cannot represent required many-to-many history; do not scan all Matters or infer links from titles.

Architecture lookup uses an approved ARTIFACT_LOOKUP Binding keyed by ChangeID. Resolve a bounded artifact/version reference server-side; do not guess filenames in the browser. Missing or ambiguous artifacts remain explicit. Reuse protected document delivery where supported; verify generated HTML is safe before claiming the existing viewer covers it. Use allowlisted source roots, traversal and egress checks, authorization on fetch, and an isolated/sandboxed viewer with a restrictive content policy. Do not execute untrusted exported HTML in the application origin or expose arbitrary file-server paths.

Exit: source row → correct architecture/reference → authorized existing Risk/Matter → return to original source view without duplicated business state.

## 7. Tranche T4: metrics and shared native measures

### T4a — count and adverse-population metrics

Reuse existing count semantics where appropriate: at-risk/overdue projects, SLA-breached/major unresolved incidents, failed/emergency/high-risk changes, critical patch exceptions, unsupported assets, channels outside tolerance and proven critical vacancies. Do not treat every positive count as bad: total active projects/assets/transactions are neutral context, not zero-clear alert metrics.

Extend observation source validation deliberately for external-backed observations. Receipts must identify Evidence Source, Connection/View/Binding revisions, schema fingerprint, observation time, condition/definition revision, completeness and scope. No arbitrary source rows inside metric value columns.

Exact historical drill needs retained safe membership or a genuinely reproducible immutable source snapshot. A cursor alone is not proof. If reconstruction is unavailable, label CURRENT_STATE and never claim same-snapshot parity. Retention must cover the advertised drill/report horizon. Retain only the minimum approved member/aggregate evidence, not raw transaction populations.

### T4b — native values, coordinated with #346 S1

Implement once in `metricview` and the existing observation/check owners when the first native consumer requires it; it need not wait for all sources or T5 resilience. Keep existing COUNT API behavior compatible.

Measure families: COUNT, RATIO/PERCENT, DURATION, MONEY; preserve legacy RISK_POINTS explicitly. Persist exact numeric representation with scale/unit/currency. Ratios retain numerator/denominator where known; aggregation uses valid disjoint source populations. Unknown/zero denominator, signed variance, reversals, currencies and missing samples need defined semantics.

Threshold definitions need direction (high/low/outside-range), boundary equality, effective revision and scope. Business condition is separate from quality and last-known state. Current score-only checks must not fabricate native values or probabilities. Approved native limits and normalized concern bands are different concepts.

Use one shared native-value/limit and trend/table composition from #346 for KRI, ITSM, channel, patch, duration and financial measures. Do not create separate renderers or chart dependencies per category. Trend gaps stay visible; threshold/population/definition changes prevent invalid comparisons.

### Initial measure families

| Pillar | Measures and denominator requirements |
| --- | --- |
| Portfolio | Active/late/red-amber projects, schedule variance, budget/forecast variance, benefits state; retain source RAG and planned/current dates |
| ITSM | Opened/resolved, SLA breaches over eligible incidents, major/aged unresolved, root-cause distribution; explicit exclusions and SLA cohort |
| Change/release | Implemented/failed/backed-out/emergency/high-risk changes, approval exceptions; success rate over defined implemented population |
| Assets | Managed/in-scope asset count, patch compliance over eligible observed assets, critical exceptions, unsupported assets, control-evidence gaps |
| Finance | Budget, actual, commitments, forecast, variance amount/rate; explicit fiscal period, currency and organization/project attribution |
| Resilience | #271 critical-service population, tolerance failures, BIA/test freshness, RTO/RPO comparison, backup/evidence gaps |
| Channels | Total/failed transactions, success rate and downtime by channel/time bucket; avoid duplicate intervals and averaging rates |
| Workforce | Approved/occupied positions, distinct people, vacancies, recruitment, joins/exits and aggregate attrition with approved denominator/source |

Do not average RAG/risk ratings, sum unlike currencies, assume account deactivation means exit or score individual employee attrition. Taxonomy/rules are approved business definitions, not UI defaults.

Exit: Home, Insights and Reports agree for the same metric/scope/definition/source revision. Values, units, period, denominator, exclusions, quality, condition, trend direction and drill consistency are explicit.

## 8. Tranche T5: resilience composition

#271 owns Critical Service identity/classification, approved BIA, impact tolerance, RTO/RPO, dependencies, plans, exercises and verified recovery outcomes. #344 consumes those records and adds source-backed backup/test observations.

Do not build a second DR domain or assume that a CRITICAL_SERVICE organization label already provides a complete BIA model. Source rto/rpo declarations cannot silently replace approved targets. Compare exact target revision with current observation; failed/inconclusive results route through existing Matter/Outcome paths.

The primary view is a service record with Indicator/Cycle presentation. Raw BIA/DR submissions remain evidence, not a substitute for service/tolerance state. This integration does not block other source or semantic slices.

## 9. Tranche T6: IT Governance Insights presentation

Consume the #266/#274 shell and the detailed [#346 presentation plan](2026-10-05-semantic-domain-presentation.md). Do not build an IT-only dashboard framework or re-render KRI/RCSA/Loss as generic Program responses.

- Registers: Projects, Assets, ITSM/Changes/Releases and relevant source populations.
- Indicators: Native KRI/KCI, SLA, patch, channel, duration and budget measures; concern score secondary.
- Ledger: Existing operational Loss record and recovery history.
- Cycle: Existing RCSA and later approved service/audit/certification cycles.
- Cases: Material interventions with type-aware context over common Matter work.
- Programs: Continuing IT governance configuration/evidence context.

Overview shows at most four stable high-signal headline cards selected by role/lens, then progressively disclosed Portfolio delivery, Service & change, Assets, Financials, Resilience, Channels and Workforce. Do not render every section/chart at once. Load details lazily and keep the dominant action unobstructed.

Use one identity/scope/period header. Native values, threshold and condition precede technical revisions. Tables retain numeric alignment and business columns. Color must accompany explicit status; Active, Submitted, Linked and Recovered are not interchangeable with verified closure or acceptable risk.

Reuse existing scope and date controls. Add Month/Quarter/Year and appropriate fiscal/custom periods through that contract only where required; define reporting-period versus current-posture semantics. Do not implement another date picker. Preserve scope/filter/period/back-navigation across Risk/Indicator/Cycle/Loss, source detail, Work and evidence.

Type-aware Matter context cannot hide existing evidence, decisions, responses or a required outcome action. Legacy/unknown types retain a safe full fallback. Canonical Loss authoring must be usable before any generic loss-entry redirection; no navigation removal based on a missing replacement.

Exit: users see the business object immediately; stale/incomplete data is recognizable; exact source/evidence remains accessible without dominating every first viewport. One missing pillar does not blank unrelated views.

## 10. Tranche T7: management packs and delivery

Extend existing report datasets/sections only when current canonical datasets and metric reads cannot express the required pack. Programs/Matters reports do not by themselves provide native source/KRI/financial populations.

Candidate pack definitions: CIO IT Governance, IT Governance Committee, Portfolio delivery, ITSM/change, Asset/cyber posture, DR/resilience, Channels and IT workforce. These are definitions over shared machinery, not eight report products.

Every run retains exact definition/checksum, scope, reporting period, as-of boundary, source/metric revisions, currency/unit semantics and quality. Scheduled runs use existing runtime/timer classes. Revalidate authority on execution/download/distribution; consume #269 provider/delivery receipts where available. No new report scheduler or mail engine.

Measure the >=80% manual-reporting objective against a representative baseline: active source-export, reconciliation, preparation and rework minutes for the same pack/scope. Preserve required human review; no claim based on feature count or screenshot. Repeat after implementation with quality/completeness checks.

## 11. Tranche T8: AI summaries and predictive pilots

Start only after deterministic source/metric acceptance. The product remains usable without AI.

First deliver a concise source-grounded summary of exact metrics, material changes, top interventions and quality limitations. Separate fact, pattern, recommendation and uncertainty. Retain workload/model/provider, prompt/policy revision, inputs, scope/period and generated time through existing governance boundaries.

Deterministic grouping precedes AI labels for recurring delays, root causes, failed changes and degradation. Prediction pilots (delivery delay, change failure, channel/service degradation, budget anomaly) require sufficient history, holdout/back-testing, calibration and measured quality against a baseline. No individual employee attrition predictions.

Predictions do not silently modify RAG, appetite, approval or other material status. No arbitrary model-generated queries or unreviewed execution.

## 12. Execution, test and release plan

### Dependency sequence

1. T0 contract and #346 S0 evidence/primary-record decisions.
2. T1 source administration and T2 bounded operational reader.
3. T3 first Project/ITSM/Asset consumers and safe architecture lookup.
4. T4a count metrics and the shared T4b/#346 S1 native measure slice as soon as its first consumer needs it.
5. #346 S2 KRI view, S3 RCSA and S4 Matter context; local semantic delivery does not wait for every source.
6. #271/T5 resilience in parallel when its domain is ready.
7. T6 progressively enabled IT Governance sections using shipped shared presentations.
8. T7 reproducible packs/scheduling and measured preparation-time reduction.
9. T8 summaries and separately gated prediction pilots.

Small Loss/Risk read refinements may run alongside this sequence. Do not create fixture-only dashboard cards before their real source contracts; honest unavailable states remain acceptable during staged rollout.

### Focused PR boundaries

Contract tests/architecture; source lifecycle gap if still present; Configure inventory; approved revision/preview commands; operational reader; shared register and three consumers; ChangeID/artifact lookup; count metrics; one shared native measure implementation; semantic Indicator/RCSA/Matter slices; Insights composition; reports/scheduling; AI.

Within each PR separate small commits for tests/contracts, backend/client, shared presentation, consumer and rendered evidence. Read main and reconcile merged overlaps before every slice. No force-push or unrelated edits. Documentation approval is not merge/deploy authorization.

### Correctness, security and performance

- Verified tenant/entity scope and authorization before population counts/pages. Aggregate permission does not grant detail access.
- Exact revisions, schema drift, stale cursors, changed limits, unavailable sources, partial data, duplicate events, source key collisions and many-to-many joins.
- Count/rate/duration/money validation, currency scale, signed variance, zeros/unknowns, disjoint aggregation and historical reconstruction.
- Source/form measurement mapping, duplicate/amended returns, unknown periods and long-text imports; no hand-typed fixture substitute for persisted-source proof.
- RCSA frozen population and real progress; Matter required-step visibility; no loss entry-point regression.
- Protected artifact resolution, traversal/egress restrictions and isolated HTML behavior; no source payload/secret leakage in logs or notifications.
- Dashboard reads use maintained observations, not synchronous broad source scans. Source predicates/aggregates push down only where capabilities exist.
- Bound source concurrency, timeouts, retries, row/byte limits and cancellation. Report ClearSight overhead separately from external-system latency; p95 <=1s bounded reads is a target to validate, not a blanket source guarantee.
- Production-shaped fixtures: 10k project/change records, million-scale asset source, high-volume channel aggregates without raw-copy, existing large-hierarchy scope proofs and independent degradation of unrelated views.

### UI and human acceptance

Reuse shared components in the app shell; capture exact-head before/after at desktop light/dark, 390/320px, keyboard and actual browser 200% zoom. Label reflow proxies separately. Include long names, large amounts, empty/stale/partial/unknown/forbidden/conflict/unavailable states, chart table alternatives, focus and primary-action obstruction.

Do not substitute accessibility automation for visual inspection or bank-user task testing. Keep confidential source-bearing captures out of public CI. Measure whether users can identify native value/limit/quality, distinguish financial recovery from closed remediation, identify cycle handoff and return from evidence without losing context.

### Data-model budget and completion

No presentation/archetype tables or Project/Incident/Change/Asset/Budget/Channel/HR replicas by default. Genuine source activation, numeric-history, reference or performance extensions need explicit owner, retention and reconstruction evidence; do not hide authoritative data in JSON merely to claim zero new tables. #271 owns necessary resilience domain storage.

Release requires reproducible metric/report truth, exact or honestly labelled current-state drill, valid source/organization mappings, safe architecture access, usable semantic views, retained legacy links, tested authority/degraded behavior, representative-user evidence and measured reporting-effort improvement. Record merged commit, executed tests and deployed acceptance separately. #344/#346 remain open until their implementation gates pass.
