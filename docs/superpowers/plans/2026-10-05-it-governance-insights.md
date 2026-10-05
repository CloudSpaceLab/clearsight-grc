# IT Governance Insights implementation plan

**Issue:** #344  
**Plan branch base:** `main@a482a594e266cebf9102acb9599120a360245a80`  
**Source/metric architecture audit:** `ca12fa5f327c445726b00fae0853e53ce022b7e2`; the 45-commit delta to the branch base was reviewed and is confined mainly to response-browser and notification-preference work, with no conflicting IT-governance source/metric architecture change.  
**Scope:** Fidelity-style IT Governance dashboard requirements without parallel Project/ITSM/CMDB/finance/HR products.

## 1. Outcome

ClearSight should provide one IT Governance operating view over authoritative enterprise sources while retaining its current product model:

- **Home** — material posture and immediate intervention;
- **Portfolio** — persistent governed objects only;
- **Work** — existing governed intervention queue;
- **Insights** — IT Governance analysis over canonical metrics and source-backed registers;
- **Reports** — reproducible governed packs;
- **Configure** — source, mapping and authority administration.

The source systems continue to own operational detail:

- ManageEngine: incidents/problems/changes;
- Jira/Azure DevOps/PMO: projects and delivery state;
- ERP: budget/actual/commitments;
- Sparx/file export: architecture artifacts;
- Axonius/HardCat/ManageEngine: assets;
- Veeam/ASR/Zerto: backup/recovery observations;
- NIBSS/Switch/channel monitoring: transaction/service observations;
- HR/SCIM sources: people/employment lifecycle.

ClearSight owns only the governance interpretation, thresholds, approved service/risk/control state, evidence, decisions, interventions, metric observations and reporting provenance.

---

# 2. Existing implementation that must be reused

Do not start implementation until each tranche confirms that these current contracts still satisfy the requirement.

## 2.1 Sources and source execution

Existing:

- `internal/evidence` — business Source identity, ownership, health and freshness;
- `internal/sourceaccess` — revisioned Connection / View / Binding catalog;
- `POSTGRES`, `REST_JSON`, `TABULAR_ARTIFACT`, `WEBHOOK_EVENT` adapters;
- source operations `INSPECT`, `PAGE`, `LOOKUP`, `AGGREGATE`, `CHANGES`;
- exact Binding mapping, parameter/output schema, freshness and completeness;
- checkpoint/retry infrastructure;
- exact operation receipts and schema fingerprints;
- source catalog HTTP APIs already registered under `/api/v1/config/...`.

Important current limitation:

- Configure → Data & integrations does not expose the catalog;
- source catalog draft/revision state exists, but a complete user-facing activation/maker-checker lifecycle is not yet an operational product flow.

Do not create another connector registry.

## 2.2 Connected checks / continuous assurance

Existing:

- `internal/assurance` logical types, schema profiling and hints;
- typed condition compilation;
- PostgreSQL source-side predicate evaluation;
- explicit MATCH/CLEAR/UNKNOWN and schema/source failure semantics;
- lexical recognition already covers status, risk/severity, owner, RTO/RPO/MAO, patch/CVE/CVSS/EDR, backup/restore/recovery-test and expiry fields;
- `internal/monitoring` source/form checks with exact Binding ID/version, rules, thresholds, freshness, coverage, owner/reviewer and maker-checker lifecycle;
- adverse monitoring results can converge into one canonical Matter rather than one ticket per source row.

Do not create ProjectMonitor / AssetMonitor / ChangeMonitor / ChannelMonitor packages.

## 2.3 Governance records

Existing:

- Program — ongoing governed responsibility, requirements, controls, evidence and checks;
- Matter — typed intervention record, including INCIDENT, EXCEPTION, CONTROL_GAP, RISK_SITUATION, KRI_BREACH, OPERATIONAL_LOSS and AUDIT_FINDING;
- Action / Decision / verification / evidence;
- canonical Risks, appetite, reusable Controls, KRI/KCI, RCSA and operational losses;
- Group / OpCo / organization scopes;
- governed positions, reporting lines, vacancy/delegation and handoff.

Do not copy every project/incident/asset into a Program or Matter.

## 2.4 Metrics and reporting

Existing metric engine:

- immutable metric definitions;
- exact observation source/revision;
- exact snapshot drill membership for existing count metrics;
- trends and daily rollups;
- risk/appetite and assurance matrices;
- Group-safe aggregate foundations.

Current constraint:

- the metric contract is intentionally narrow: current values are integer/count-oriented and the durable definition vocabulary is currently `COUNT` + current-posture/zero-clear/sum semantics.

Therefore:

- **do not build a second KPI engine for budget, percentages or durations**;
- first use the current metric engine for exact count/condition metrics;
- extend `metricview` once, deliberately and backwards-compatibly, when the first non-count measure is required.

Existing report engine already supplies:

- governed definitions;
- maker/reviewer/authorizer;
- immutable revision/checksum;
- XLSX/CSV/NDJSON;
- source boundary/as-of/provenance;
- bounded async generation;
- protected download.

Do not create a second management-pack engine.

---

# 3. Architecture constraints

These are release blockers, not preferences.

1. No new top-level module for any of the eight dashboard pillars.
2. No new durable table until an existing Source/Binding/Program/Matter/Metric/Report contract is proven insufficient.
3. No external full-population copy into ClearSight by default.
4. No browser aggregation of large source populations.
5. No one-source-row → one-Matter behavior by default.
6. No separate cross-system ID service while the customer's common ChangeID is valid.
7. No duplicated organization hierarchy.
8. No duplicated report scheduler or notification engine.
9. Source failure/schema drift/incomplete reads must never become a green/zero state.
10. AI can summarize or predict only from exact governed facts and never becomes material authority.

---

# 4. Source binding profiles

Issue #344 needs a small interpretation contract, not a new source model.

Use the existing `BindingRevision.Mapping` JSON.

## 4.1 Mapping schema

Introduce a versioned application-level schema, for example:

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
  "units": {
    "budget_amount": "NGN",
    "actual_spend": "NGN"
  }
}
```

Supported `shape` values in the first release:

- `REGISTER` — bounded row population;
- `METRIC_SERIES` — source already provides bounded aggregate values/periods;
- `ARTIFACT_LOOKUP` — exact-key lookup such as Sparx ChangeID → HTML artifact.

Do not add a table for these profiles. They remain part of the exact Binding revision and therefore inherit Binding provenance.

## 4.2 Lens vocabulary

Initial bounded lens codes:

- `PROJECTS`
- `ITSM_INCIDENTS`
- `ITSM_CHANGES`
- `RELEASES`
- `ASSETS`
- `IT_FINANCIALS`
- `DR_TELEMETRY`
- `CHANNEL_PERFORMANCE`
- `WORKFORCE`
- `ARCHITECTURE_ARTIFACTS`

A binding may expose only selected/safe fields already allowed by its activated contract.

## 4.3 Validation

Add a focused parser/validator, preferably under a small `internal/itgovernance` package.

It should:

- reject unknown schema versions;
- reject unknown lens/shape;
- ensure mapped source fields are present in `Binding.SelectedFields` and the exact View schema;
- enforce stable-key requirements for REGISTER/ARTIFACT_LOOKUP;
- validate unit metadata;
- reject duplicate semantic roles where singular;
- never inspect or expose credentials/connection definitions.

No generic ontology engine.

---

# 5. Tranche T0 — implementation contract and tests only

**Goal:** prove that current contracts can represent the Fidelity requirements before changing schema.

### Files

Create:

- `internal/itgovernance/mapping.go`
- `internal/itgovernance/mapping_test.go`
- `docs/architecture/it-governance-source-lenses.md`

Modify only if needed:

- `docs/architecture/durable-schema-ownership.md` — documentation only; no durable row expected in T0.
- issue #344 with exact progress.

### Tests

Create deterministic source schemas for:

1. Project:
   - ChangeID
   - project name/manager
   - RAG
   - dates
   - budget/actual
2. ITSM:
   - incident/change ID
   - priority
   - SLA
   - change type/result
3. Asset:
   - asset ID/type
   - criticality
   - lifecycle
   - patch posture
4. Channel:
   - channel/period
   - transactions/failures/success rate/downtime
5. Workforce:
   - person/position
   - active/joined/exited/recruitment state

Acceptance:

- all five validate through one mapping contract;
- no persistence;
- no UI;
- no new workflow/source/report abstraction.

**Commit:** `docs/it-governance mapping contract`

---

# 6. Tranche T1 — make the existing Source Catalog operable

**Goal:** ClearSight administrators can use the source infrastructure that already exists.

## 6.1 Backend

First re-audit current #57/source-access lifecycle before writing code.

Use current routes where possible:

- Source list/create;
- Connection revision list/create;
- View revision list/create;
- inspect;
- Binding revision list/create;
- preview;
- where-used.

If production activation is still missing, add only the minimal current-revision lifecycle necessary for Connection/View/Binding:

```text
DRAFT
→ PENDING_APPROVAL
→ ACTIVE
→ PAUSED / RETIRED
```

Requirements:

- maker/checker;
- exact base/revision fencing;
- parent must be current/active before child activation;
- schema fingerprint must still match;
- verified adapter capabilities must match activation;
- one current effective revision;
- rollback = new revision, never mutation.

Prefer extending sourceaccess lifecycle semantics; do not use a new policy engine.

## 6.2 Frontend

Modify:

- `web/src/components/configure/DataIntegrationsSection.tsx`

Create modular components under:

- `web/src/components/configure/sources/`

Suggested modules:

- `SourceInventory.tsx`
- `SourceConnectionList.tsx`
- `SourceViewList.tsx`
- `SourceBindingList.tsx`
- `SourceRevisionDetail.tsx`
- `SourcePreview.tsx`
- `SourceUsage.tsx`

Reuse shared DataTable, StatusBadge, Notice, EmptyState, FocusedSheet/Popover/Dialog primitives.

Do not replace the existing narrow Program `DataSourceBuilder`; it remains the simple contextual path for basic HTTPS status checks.

## 6.3 UX

The administrator should be able to answer:

- Which sources exist?
- Is this source current/stale/degraded/unavailable?
- Which connection/view/binding revision is active?
- What fields are exposed?
- What does one bounded preview look like?
- Which Programs/checks/forms consume this Binding?
- Is there a pending revision?

No network-transport jargon on the first level.

## 6.4 Acceptance

- exact revision survives reload/deep link;
- source outage does not make configuration inventory disappear;
- secret refs are never rendered;
- preview respects Binding field/row/byte/time limits;
- mobile/320px/200%/keyboard/axe states;
- no source lifecycle action is available without current authority.

**Commit sequence:**

1. source lifecycle/backend gaps only;
2. typed web API;
3. inventory/read UI;
4. revision/preview/write UI;
5. tests/evidence.

---

# 7. Tranche T2 — operational source-backed lens reader

**Goal:** management users can inspect bounded authoritative source populations without gaining CONFIG_READ and without using the admin preview API as a product read.

Do **not** expose arbitrary Binding IDs directly to ordinary oversight users.

## 7.1 Consumer authorization

Create an IT Governance read service that resolves only current active bindings whose mapping declares one of the recognized IT Governance lenses.

Authorization contract:

- verified tenant/legal entity;
- `OVERSIGHT_READ` for Insights management views;
- source must belong to current legal entity unless consuming an already-authorized Group projection;
- aggregate access does not grant row/detail access across entities;
- selected fields remain bounded by Binding;
- no connection/secret/config details in response.

If a Portfolio use case later requires broader specialist access, add that deliberately after representative role review. Do not broaden T2 pre-emptively.

## 7.2 Binding discovery

Add the smallest catalog query needed to locate current bindings by purpose/lens for one legal entity.

Prefer:

- existing Binding `Purpose` plus validated Mapping lens;
- join through Evidence Source to legal entity;
- bounded list.

If a query/index is required, add one focused index on existing catalog columns. Do not add a source-lens table.

## 7.3 API

Suggested bounded routes:

```text
GET /api/v1/insights/it-governance/lenses
GET /api/v1/insights/it-governance/{lens}/records
GET /api/v1/insights/it-governance/{lens}/records/{key}
```

The first route returns availability/quality only.

The record page returns:

- binding ID/version only as provenance metadata;
- generated/observed time;
- source freshness;
- completeness;
- safe mapped fields;
- opaque next cursor;
- no connection definition;
- no arbitrary source query.

For REST/JSON or tabular sources where true source-side search/sort is unavailable, do not download the entire population to simulate it. Expose only capabilities the Binding can satisfy.

## 7.4 Shared web composition

Create one shared component family, e.g.:

- `SourceBackedRegister.tsx`
- `SourceQualityLine.tsx`
- `SourceRecordDetail.tsx`

The component is reused by Project, ITSM and Asset views; domain-specific column definitions remain small configuration objects.

## 7.5 Acceptance

Prove three materially different populations before continuing:

- Projects;
- ITSM incidents;
- Assets.

Each must show:

- exact source revision;
- explicit freshness/completeness;
- bounded pages;
- safe source outage state;
- no copied durable row table in ClearSight.

**Commit sequence:**

1. consumer resolver + auth tests;
2. read API;
3. shared web register;
4. Project/ITSM/Asset fixtures and rendered review.

---

# 8. Tranche T3 — Projects, ITSM and Change/Release views

## 8.1 Projects

Do not add `projects` table.

Source-backed Project view should present only governance-relevant fields:

- ChangeID;
- name;
- manager;
- organization;
- RAG;
- start/target/end;
- budget/actual/forecast where supplied;
- velocity/backlog where supplied;
- linked governance exceptions.

A Project source row is not automatically a Program.

### ClearSight relationship

Where a project has a governed Program/Risk/Matter, link by the existing source/external key in read composition or an existing record's source metadata. Add a durable explicit link only after a real many-to-many/history requirement is demonstrated.

## 8.2 ITSM

Use source views for full denominators and records.

Use canonical Matters only for governance intervention.

Example:

```text
12,482 incidents in ManageEngine
        ↓ source-backed ITSM register/metrics
552 SLA breaches
17 major incidents
6 material governance interventions
        ↓
6 Matters / Work items
```

Do not import 12,482 Matters.

## 8.3 Change and release

Change and release are presentation lenses over the same source substrate.

Required fields:

- ChangeID;
- type: Standard / Normal / Emergency;
- risk;
- approval state;
- planned/actual implementation;
- success/failure/backout;
- affected service/application;
- release ID where present.

A failed/high-risk change can link to a Matter/Risk when intervention is required.

No CAB workflow is added.

## 8.4 Sparx architecture lookup

Use an `ARTIFACT_LOOKUP` Binding keyed by ChangeID.

Initial implementation should prefer governed tabular/file metadata or a fixed safe REST/file service.

The returned record contains only a safe artifact reference.

Opening the artifact must:

- use the current artifact/document authorization path where possible;
- reject path traversal;
- not infer filenames in the browser;
- verify exact source/binding provenance.

## 8.5 Acceptance

From one Project/Change row a user can:

- understand delivery/change posture;
- open the exact architecture artifact;
- open an existing Risk/Matter when linked;
- never see a fabricated “ClearSight project status” different from source truth.

---

# 9. Tranche T4 — IT governance metrics

Do this in two bounded steps.

## T4a — count/condition metrics using current metric semantics

Start only with metrics naturally represented by current count semantics:

- projects at material delivery risk;
- overdue projects;
- major unresolved incidents;
- SLA-breached incidents;
- emergency changes;
- failed/backed-out changes;
- high-risk changes;
- critical patch exceptions;
- unsupported critical assets;
- critical services outside tolerance once #271 exists;
- channels outside tolerance;
- critical vacancies where source-backed.

Reuse:

- immutable metric definition;
- observation/trend retention;
- exact source revision;
- exact drill identity.

For source-backed observations, extend the observation source contract deliberately. Do not store arbitrary external rows in `metric_observations`.

A source-backed metric source receipt must identify:

- Evidence Source;
- exact Connection/View/Binding revisions;
- source schema fingerprint;
- observation time;
- completeness;
- exact condition definition;
- retained exact member identity only where safe/necessary.

If exact member retention would duplicate a huge population, store only the adverse member set or use source-snapshot lookup semantics with an immutable source cursor/revision that can actually reconstruct the population. If neither is possible, mark drill consistency as CURRENT_STATE and do not claim exact historical parity.

## T4b — typed measures only when first required

Budget, rates and durations cannot be forced into `COUNT`.

Extend `metricview` rather than introduce a separate KPI engine.

Required measure families only:

- COUNT;
- RATIO/PERCENT;
- DURATION;
- MONEY.

Design requirements:

- exact decimal/integer semantics; no float persistence;
- explicit currency for MONEY;
- explicit numerator/denominator for a ratio where available;
- aggregation rule per measure;
- no summing percentages;
- no averaging RAG/risk ratings;
- backwards-compatible existing count JSON/API behavior.

Prefer source-provided bounded aggregate values for:

- budget/actual;
- transaction totals;
- success rate;
- downtime.

Do not scan millions of transactions in ClearSight to calculate a dashboard value when the source system can provide a governed aggregate view.

## Metrics acceptance

For every metric:

- value;
- unit;
- scope;
- period/current-posture basis;
- freshness;
- completeness;
- population/denominator where meaningful;
- source revision;
- aggregation rule;
- drill contract;
- trend direction.

Home, Insights and Reports must consume the same definition/revision.

---

# 10. Tranche T5 — resilience integration

Do not implement DR as a #344 subsystem.

Consume #271 once it provides:

- Critical Service;
- BIA;
- approved impact tolerance;
- RTO/RPO;
- dependencies;
- exercises/tests;
- verified recovery outcome.

#344 adds only source composition:

- Veeam/ASR/Zerto Source/Binding;
- backup status;
- last test;
- actual recovery time;
- current recovery observation;
- resilience metric projection.

Governance truth:

```text
Approved RTO/RPO/BIA      → #271 / ClearSight authoritative
Observed backup/test      → external recovery source
Tolerance comparison      → governed metric/check
Material failure          → existing Matter/Work/Outcome
```

No second recovery-plan workflow.

---

# 10.5 Presentation semantics dependency

IT Governance must consume the semantic-presentation architecture tracked by #346 rather than rendering every domain through generic Program/response/Matter chrome.

Required mapping for this program:

- Projects / Assets / ITSM populations → **REGISTER**;
- service/channel/patch/SLA/budget threshold measures → **INDICATOR**;
- resilience review/test periods → **CYCLE** once #271 owns the governed service/BIA state;
- material source exceptions → **CASE / INTERVENTION** through existing Matters;
- continuing IT governance Programs remain **PROGRAM** context, not the primary visual representation of source populations;
- operational losses remain the existing **LEDGER** presentation and link into IT Governance Insights by metric/drill rather than being re-rendered as generic Matters.

The Program response viewer remains the exact evidence/provenance surface. It must not become the primary KRI, RCSA, BIA, project, asset or channel-management surface when a semantic presentation exists.

For KRI/KCI and IT operational measures, show the source/native value, unit and approved threshold as the primary value. Any normalized 0–100 concern/risk score remains secondary cross-domain context.

# 11. Tranche T6 — IT Governance Insights

This should compose with #266/#274 rather than create a new dashboard shell.

## 11.1 Navigation

Add **Insights** to the bounded operating navigation when the #266 route is ready.

IT Governance is one Insights lens, not a primary nav destination.

## 11.2 First viewport

Maximum four headline cards.

Candidate role-aware selection:

- projects at risk;
- critical service/channel breaches;
- failed/high-risk changes;
- material budget variance or critical control exposure.

Cards require:

- exact value;
- semantic state;
- freshness/completeness;
- direct drill.

## 11.3 Sections

Progressive disclosure:

1. Portfolio delivery
2. Service & change
3. Asset / cyber posture
4. Financials
5. Resilience
6. Channels
7. Workforce

Each section starts with 2–4 meaningful measures and one compact source-backed table/trend where useful.

Do not show every source field or every KPI.

## 11.4 Scope / period

Reuse existing:

- Group/OpCo/entity/organization scope;
- reporting period;
- metric trend semantics.

Organization hierarchy is the common dimension for budget, people, risk and project views.

Do not build a finance/org tree.

## 11.5 RAG

RAG is source/governance state, not decorative color.

Every red/amber state must have text/icon and drill explanation.

---

# 12. Tranche T7 — report packs and measured manual-effort reduction

## 12.1 Reuse report engine

Do not create “dashboard export”.

Add pack/report definitions over the same IT Governance metrics and source-backed datasets.

Candidate packs:

- CIO IT Governance;
- IT Governance Committee;
- Portfolio delivery;
- ITSM & change;
- Asset/cyber posture;
- DR/resilience;
- Channels;
- IT workforce/capacity.

## 12.2 Scheduling

Use existing runtime/timer infrastructure.

A scheduled run is still an ordinary governed report run bound to:

- definition checksum/version;
- exact period;
- scope;
- metric/source revisions;
- completeness.

Delivery should consume #269's external delivery/provider receipt capability when available.

## 12.3 80% objective

Do not claim success from feature count.

Baseline one representative monthly/quarterly IT Governance pack:

- current manual preparation active minutes;
- number of manual source exports;
- number of reconciliation steps;
- review/rework time.

Repeat with ClearSight.

Acceptance target:

- ≥80% reduction in active preparation effort;
- source review/approval time may remain human;
- no quality/completeness regression.

---

# 13. Tranche T8 — AI summaries and predictive pilots

Start only after deterministic metric/source acceptance.

## 13.1 First AI capability

Generate a bounded executive summary from:

- exact metric observations;
- material movement;
- top interventions;
- source freshness/unknowns;
- linked approved facts.

Output must distinguish:

- fact;
- detected pattern;
- recommendation;
- uncertainty.

Store/reconstruct:

- workload;
- model/provider;
- prompt/policy revision;
- exact source/metric revisions;
- generated time.

## 13.2 Pattern detection

Allowed before predictive scoring:

- recurring delay categories;
- recurring incident root causes;
- repeated failed/emergency change patterns;
- repeated service degradation.

Use deterministic grouping first. AI may label/summarize clusters.

## 13.3 Predictive pilots

Only after history and back-testing are adequate:

- project delivery risk;
- change failure probability;
- channel/service degradation;
- budget forecast anomaly.

No employee attrition probability.

A prediction is advisory and cannot silently modify Project RAG, Risk appetite, change approval or any material status.

---

# 14. Testing matrix

Every tranche adds focused tests; do not wait for end-to-end release.

## Backend

- legal-entity isolation;
- unauthorized Binding/lens access;
- stale Binding revision;
- schema drift;
- source timeout;
- partial/incomplete result;
- large result boundedness;
- duplicate webhook/source event;
- current vs historical source revision;
- Group aggregate without sibling-detail leakage;
- exact metric drill parity where claimed;
- ratio/money aggregation correctness when T4b lands.

## Frontend

For each major surface:

- loading;
- live/current;
- stale;
- partial;
- unknown;
- empty;
- unavailable;
- forbidden;
- conflict/revision changed;
- narrow mobile;
- 320px reflow;
- 200%;
- keyboard;
- forced colors where current release gate requires;
- no horizontal overflow.

## Performance

Production-shaped fixtures:

- 10k projects/change rows;
- 1m+ asset population with source-side filtering;
- high-volume channel metric source without raw transaction copy;
- >500 organization areas / existing 20k hierarchy proof reused;
- source outage while unrelated lens remains responsive;
- metric observation/trend reads remain bounded.

---

# 15. Data-model budget

Expected new durable tables before T5: **zero**, except where a proven source-catalog lifecycle gap cannot be expressed by current revision tables.

Possible later schema changes:

1. source catalog index/transition metadata — existing tables only;
2. metricview typed-measure extension — existing metric definitions/observations plus minimum necessary columns/constraints;
3. #271 owns its own first-class resilience records.

Do not create:

- projects;
- incidents;
- changes;
- releases;
- assets;
- budgets;
- channel_transactions;
- staffing_events;

inside ClearSight merely to satisfy dashboard presentation.

If a later requirement proposes one of these, the PR must first document why Source Access + existing governed records cannot satisfy reconstruction, performance or authority.

---

# 16. Recommended execution order

Dependency order:

1. **T0 mapping contract**
2. **T1 Source Catalog administration / minimum activation gap**
3. **T2 operational source-backed reader**
4. **T3 Projects + ITSM/change + architecture lookup**
5. **T4a count metrics**
6. **#271 / T5 resilience composition in parallel where possible**
7. **T4b typed metric measures**
8. **T6 Insights**
9. **T7 management packs/scheduling**
10. **T8 AI**

Do not start T6 with fixture-only dashboard cards while T2/T4 source truth is absent.

---

# 17. Small-commit / PR strategy

Keep changes independently reviewable.

Recommended PRs:

- **PR A:** T0 mapping contract + architecture note.
- **PR B:** Source catalog lifecycle gap, if still required after re-audit.
- **PR C:** Configure source inventory/read-only.
- **PR D:** Configure source revision/preview/write operations.
- **PR E:** IT Governance operational source reader.
- **PR F:** Shared source-backed register + Project/ITSM/Asset proof.
- **PR G:** Projects/change/release + ChangeID/Sparx lookup.
- **PR H:** count metric projection/drills.
- **PR I:** typed measure extension.
- **PR J:** IT Governance Insights composition.
- **PR K:** reports/scheduling.
- **PR L:** AI summary/pilot.

Each PR must reconcile current `main` first and avoid reimplementing work that merged while the program was in progress.

---

# 18. Release definition

The program is ready for stakeholder acceptance when:

1. an administrator can configure approved source bindings without deployment-file edits for supported adapters;
2. a CIO/risk user can open IT Governance Insights and see truthful posture across the requested pillars;
3. source-backed records remain in their authoritative systems;
4. material adverse conditions converge into existing Risk/Matter/Work flows;
5. exact counts drill correctly and non-exact historical drills are labelled honestly;
6. organization filters use the existing hierarchy;
7. ChangeID opens the correct architecture artifact;
8. #271 provides authoritative resilience targets;
9. report packs reproduce the same metrics/source revisions;
10. manual report preparation reduction is measured at ≥80%;
11. source outage/schema drift produces stale/unknown/degraded state, never false green;
12. no parallel Project, ITSM, CMDB, finance, HR, workflow, source, metric, notification or report platform was introduced.
