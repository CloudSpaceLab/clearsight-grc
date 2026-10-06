# Home oversight T0 audit

Issue: #368  
Base: `main@7a7befe5b26f5e67f20c04147fe5811b7ebe6fe0`  
Scope: T0 only — no production UI, schema, API or metric behavior changes.

## Outcome

Establish the exact current-state contract for the role-aware Home redesign before runtime implementation.

The redesign must reuse the existing hierarchy, metric, Work, Attention and Operational Loss foundations. T0 confirms where those foundations are already sufficient and where new governed projection work is required.

## Current Home composition

`web/src/components/oversight/OversightWorkspace.tsx` currently renders one vertically growing surface containing:

1. scope and period context;
2. data freshness / source basis;
3. optionally assigned work first;
4. four headline workflow-pressure metrics;
5. optionally assigned work after the metrics;
6. priority interventions / exact metric drill;
7. detailed analysis tabs.

`home_focus` supports `POSTURE` and `MY_WORK`, but only changes ordering. It does not separate distinct Home intents.

Existing headline metrics are:

- `critical_high_open`
- `overdue_open`
- `routing_gaps`
- `outcome_failures`

These are useful intervention-pressure signals, but they are not sufficient as CRO/GRC posture anchors.

## Existing foundations that must be reused

### Exact metric identity and drill parity

#268 already provides:

- immutable/versioned metric definitions;
- retained metric observations;
- exact count/list membership for current Home metrics;
- source/definition revision binding;
- bounded trend retention;
- comparison direction/quality;
- role/lens presentation defaults;
- cross-domain Risk/KRI/assurance/loss count metrics.

Do not rebuild this as a Home-specific metric engine.

### Cross-domain CRO-relevant metric families

`internal/metricview/domain.go` currently defines:

- `risks_outside_appetite`
- `indicator_breaches`
- `assurance_failures`
- `losses_without_issue`

These are more appropriate CRO posture families than the current four workflow-pressure cards.

### Organization scope

Stable organization-scope IDs and descendant filtering already exist across the product. Current legal-entity Home metrics can also be evaluated for a selected organization scope.

However, this support is not yet symmetrical across all metric families or history.

### Operational Loss

`internal/oploss/model.go` already owns canonical Loss truth:

- stable `organization_scope_id`;
- gross amount;
- recoveries/reversals;
- net loss;
- currency;
- occurrence/discovery timestamps;
- Risk/Matter relationships;
- lifecycle/status.

The Home redesign must consume this ledger rather than create a parallel financial-loss model.

## Verified gaps

### 1. Organization-scope metric history is not supported

`internal/httpapi/metric_trend_handlers.go` explicitly rejects a non-empty `organization_scope_id`.

Therefore a statement such as:

> Technology worsened over the last 30 days

cannot currently be produced from the retained canonical trend contract.

Frontend aggregation is not an acceptable substitute because it would not preserve immutable historical scope/revision semantics.

### 2. Cross-domain metric reads are legal-entity-only

`internal/httpapi/domain_metric_handlers.go` also rejects `organization_scope_id`.

The CRO posture families therefore cannot yet be sliced through the organization hierarchy using the same governed read contract.

### 3. Current Group Home uses workflow-pressure metrics

`GroupOversightWorkspace` compares OpCos using the current four workflow-pressure metrics. Group presentation must later align to the same CRO posture model without weakening child authorization.

### 4. Loss amount analytics are not yet a canonical metric family

The Loss register can correctly show gross/recovered/net values per record, but Home does not have a governed period-flow metric/trend contract.

A Home Loss amount must not be implemented as browser totals over paginated register data.

### 5. Mixed-currency aggregation is undefined

The canonical Loss ledger preserves per-record currency. There is no approved reporting-currency/FX basis in the audited path.

Therefore T4 must fail safe:

- same-currency sums are valid;
- mixed currencies are not summed;
- event counts/currency-separated amounts remain valid;
- no implied conversion rate may be invented.

## Product boundary established by T0

### Home / Oversight

Must answer only:

1. How are we now?
2. Where is exposure concentrated?
3. What materially changed?

### Home / Attention

Owns enterprise/scope intervention pressure:

- critical/high;
- overdue;
- routing gaps;
- failed/inconclusive outcomes;
- material breach/exception conditions.

### Home / My work

Owns bounded actor-scoped assigned work.

### Insights

Retains:

- dense matrices;
- principal-risk × organization views;
- full appetite/assurance matrices;
- management packs;
- broad analytical exploration.

This boundary prevents Home from becoming another long dashboard.

## Comparison semantics fixed for later tranches

### Posture / stock metrics

Examples: outside-appetite Risks, active indicator breaches, assurance failures.

Compare:

`current as-of posture` vs `prior comparable as-of posture`

Do not compare the number of records created during two periods.

### Flow metrics

Example: operational Loss amount.

Compare:

`current period flow` vs `immediately preceding equal-length period flow`

Occurrence time is the default Loss-period basis unless the canonical Loss contract is changed explicitly.

## Recommended historical organization model

Do not persist every metric for every ancestor organization node every five minutes.

T3 should prefer:

- direct-attribution daily buckets keyed by stable `organization_scope_id`;
- metric definition/revision;
- captured hierarchy revision;
- day/as-of;
- quality/population metadata;
- aggregate descendants to a requested ancestor on read.

This avoids ancestor multiplication while preserving the ability to reconstruct historical organization posture.

The implementation must prove that historical data is evaluated against the hierarchy revision captured at observation time, not silently reclassified through today's organization tree.

## Required T0 acceptance dataset

The companion fixture must contain:

- at least five departments/areas;
- one nested child scope;
- outside-appetite Risk concentration across several areas;
- a worsening and an improving area;
- KRI/KCI breach examples;
- assurance failure;
- Loss events and recoveries;
- mixed currency;
- one unattributed Risk/Loss case;
- enough prior/current observations to exercise 7/30/90-day comparison semantics.

The fixture is illustrative acceptance data only. It must not become production seed truth.

## T0 exit decision

Runtime implementation may proceed when:

- each Home measure has one authoritative source;
- stock vs flow semantics are explicit;
- organization-history limitations are acknowledged;
- mixed-currency behavior is fail-safe;
- representative acceptance data exists;
- no new generic dashboard, query or workflow framework is required.

T1 may now safely focus only on Home intent/tab separation. T2+ must not claim department movement until organization-scope canonical reads/history exist.
