# Enterprise Risk Command Center and Group-Scale GRC Implementation Plan

**Date:** 2026-10-01  
**Umbrella:** #265  
**Baseline:** `main@e1438b32105eee5c55ca0d2eaa5491d454b046c8`

## 1. Goal

Close the major enterprise-GRC and large-organization UX gaps without turning ClearSight into a collection of disconnected modules.

The target experience is simple:

> A user opens ClearSight and can immediately see what changed, how much material exposure is open, what is outside appetite/tolerance, what may affect the organization, what requires their action, and whether the displayed numbers are complete/current.

The implementation must preserve ClearSight's strongest foundations:

- Program = continuing governed responsibility;
- Matter = specific issue/change/finding/incident/exception;
- Evidence/Source = proof/current external truth;
- Form/Capture = structured information collection;
- Decision = governed choice;
- Action = implementation work;
- Outcome check = independent confirmation;
- authority, timers, outbox/inbox and point-in-time history remain shared foundations.

No tranche may introduce a duplicate task, approval, evidence, scheduler, event bus, alert-business-state, reporting or source platform.

---

## 2. Behavioral UX contract

The system should make risk/compliance officers feel **oriented and in control**, not merely show more data.

This is achieved through recognition, stable visual anchors, trustworthy numbers and clear closure paths rather than gamification.

### 2.1 Recognition over recall

Use the same small visual vocabulary everywhere:

- critical/high/warning/current/unknown states;
- stable risk/appetite badges;
- stable intervention icons;
- four or fewer headline metric positions per Home lens;
- consistent trend/delta placement;
- consistent owner/due/next-action placement.

A user should recognize a breached KRI or failed outcome check before reading every word.

### 2.2 Semantic colour, not decoration

Colour may indicate:

- severity;
- appetite/tolerance band;
- worsening/clearing change;
- required attention;
- stale/unknown state.

It must never:

- be the only signal;
- imply “green = safe” where population/freshness is incomplete;
- score or rank employees;
- inflate urgency by colouring ordinary work red;
- use decorative gradients that compete with state.

Critical colours must remain visually rare enough to retain salience.

### 2.3 Trustworthy counts

Every number shown as a risk/compliance fact must be bound to:

- definition/version;
- scope;
- as-of/period;
- denominator/population;
- unknown/excluded count;
- freshness/completeness;
- source/projection revision;
- exact drill-through population.

The same revision must back a card and its drill-through. A “7 outside appetite” card cannot open a list of 6 or 8 unless the UI explicitly states that the data refreshed between views.

### 2.4 Movement over KPI walls

Prefer:

- “outside appetite 7, +2 since last review”;
- “3 new critical breaches, 2 cleared”;
- “evidence coverage fell from 94% to 89%”;

over large dashboards containing dozens of static counts.

The product should answer “what changed?” before “what else can be measured?”

### 2.5 Perceived control through exact next action

Every material intervention card/row should expose:

1. what happened;
2. why it matters now;
3. likely/declared organizational impact or affected scope;
4. owner/responsibility;
5. due/escalation state;
6. one dominant valid action.

Do not manufacture recommendations when the system has only an observation.

### 2.6 Calm without false reassurance

Empty/quiet states are desirable only when truthful:

- “No current assigned work” is safe.
- “No risks” is unsafe if the population is unavailable.
- “0 breaches” must distinguish complete zero from unknown/incomplete coverage.

---

## 3. Target information architecture

Primary operating navigation remains bounded:

```text
Home       posture + movement + top interventions
Work       one cross-domain intervention queue
Portfolio  persistent governed records/lenses
Insights   trends, matrices, reports and management packs
Configure  restricted administration
```

### 3.1 Home

Home is the command surface, not another module dashboard.

Above the fold:

- persistent scope selector;
- lens/role selector where authorized;
- freshness/as-of status;
- max four headline metrics;
- one dominant “Needs attention” block;
- one analytical visual.

Typical risk lens:

```text
Outside appetite | KRI breaches | Critical issues | Overdue P1/P2
Needs attention
Principal risk × OpCo appetite matrix
```

Other lower sections may include assurance coverage, due-soon work or resilience posture, but must not compete with the first viewport.

### 3.2 Work

One queue for intervention classes already represented by Today:

- Decision;
- Approval;
- Review;
- Evidence;
- Escalation;
- Outcome check;
- External response.

Default views:

- Mine;
- My team;
- Escalated;
- Completed.

Domain/source/type/severity/due filters are projections, not separate inboxes.

### 3.3 Portfolio

Persistent objects appear as authorized lenses:

- Risks;
- Programs;
- Controls;
- Third parties;
- Privacy;
- Resilience;
- Assurance;
- Policies.

Forms/Imports become contextual tools for ordinary users while retaining focused administrator/library access.

### 3.4 Insights

Insights uses the same metric definitions as Home:

- risk movement;
- appetite matrices;
- KRI/KCI trend;
- critical/open aging;
- control/evidence coverage;
- resilience;
- combined assurance;
- third-party concentration/deficiency;
- governed reports/packs.

No independent BI/query platform is introduced.

---

## 4. Issue log

| Issue | Workstream | Priority | Primary dependency | Anti-bloat boundary |
|---|---|---:|---|---|
| #266 | Enterprise UX shell | P0 | current Today/Programs/Work/Vendors/ROPA | no new domain shells |
| #267 | Group / OpCo hierarchy | P0 | legal-entity scope/authority | aggregates do not bypass local authorization |
| #268 | Canonical metrics/widgets | P0 | #267, #57 | no arbitrary dashboard/SQL builder |
| #269 | Attention/notification delivery | P0 | outbox/inbox, #57, #142 | notification is delivery metadata, not workflow truth |
| #270 | ERM/operational risk kernel | P0 | #267/#268/#57 | reuse Matter/Evidence/Form/Action |
| #271 | Operational resilience | P1 | #57/#267/#268 | no duplicate CMDB/ITSM |
| #272 | Audit + combined assurance | P1 | #270 controls + Evidence/Matters | no separate finding/remediation stack |
| #273 | Policy/regulatory/recertification | P1 | Programs/Requirements/Evidence | no second document/workflow platform |
| #274 | Enterprise Insights/report packs | P1 | #267/#268 + current reporting | extend report engine; no second BI/report engine |

Existing linked execution:

- #57 connected sources/continuous assurance;
- #80 third-party lifecycle;
- #142 escalation;
- #172 AI gateway/control plane;
- #13 production identity/isolation/evidence/security acceptance.

---

## 5. Data/model additions

Only add durable objects when existing records cannot safely express the business truth.

### 5.1 Organizational scope

Required:

- scope node;
- scope kind;
- parent/reference;
- legal-entity binding where applicable;
- effective period;
- status;
- authority/display metadata.

Do not move legal-entity ownership of existing records into a generic hierarchy table.

### 5.2 Risk

Persistent risk identity/profile:

- taxonomy/category;
- scope;
- accountable owner;
- cause/event/impact statement;
- affected objectives/services;
- assessment method/version;
- current inherent/residual assessment reference;
- appetite/tolerance reference;
- control/indicator links.

Historical assessment values remain revisioned facts, not mutable columns pretending to be history.

### 5.3 Reusable Control

Separate:

- control definition/objective;
- scope-specific implementation;
- Program/Requirement/Risk relationships;
- performer/reviewer/frequency;
- evidence/effectiveness review.

Do not duplicate current Program safeguards where they already represent a scope-specific implementation; migration/composition must be explicit.

### 5.4 Indicator/Metric

One definition contract can cover KRI/KCI and analytical metrics while keeping business semantics explicit.

Definition includes:

- unit/aggregation;
- source/check;
- population;
- threshold bands;
- owner;
- cadence;
- completeness rules;
- scope behavior.

Observation/snapshot is rebuildable/retained according to trend requirements.

### 5.5 Critical Service

Governed resilience object only.

Dependencies are references/projections from authoritative CMDB/OSS/HR/vendor sources; do not create a copied asset universe.

### 5.6 Assurance

Minimal durable concepts:

- assurance universe/auditable entity reference;
- plan;
- engagement;
- test/work item;
- conclusion;
- coverage relationship.

Findings remain canonical Matters.

---

## 6. Metric and widget architecture

### 6.1 Metric snapshot contract

A snapshot should be able to return:

```text
metric_id / definition_revision
scope_id
as_of
period_start / period_end
value
unit
previous_value
delta/direction
threshold_revision
state
population
unknown
excluded
completeness
freshness
projection/source revisions
drill predicate/token
```

Do not persist UI-only card state as authority.

### 6.2 Widget set

Support only:

1. Metric card
2. Trend
3. Distribution bars
4. Appetite/risk matrix
5. Coverage matrix
6. Ranked list

Any new widget type requires a business question that these cannot express.

### 6.3 Drill consistency

Use one server-owned predicate/snapshot identity for:

```text
metric calculation
       ↓
metric count
       ↓
drill list
```

Tests must assert same-snapshot population consistency.

---

## 7. Notification architecture

### 7.1 Canonical event → delivery

```text
domain/source event
      ↓
current domain projection / Signal/Drift
      ↓
material notification intent
      ↓
recipient resolution
      ↓
in-app / email delivery
```

The notification service does not own the risk, Matter, assignment or approval.

### 7.2 Episode deduplication

```text
NORMAL
  ↓ threshold crossed
BREACHED       notify
  ↓ unchanged
BREACHED       no duplicate
  ↓ worsened/escalation timer
ESCALATED      notify
  ↓ cleared
CLEARED        optional notify
```

Bind episode identity to canonical subject + scope + condition/threshold revision.

### 7.3 Daily digest

Default digest content:

- material new/worsened/cleared conditions;
- current critical exposure;
- decisions/approvals assigned;
- overdue/due-soon material work;
- significant source/coverage degradation.

No evidence attachments and no sensitive raw content.

### 7.4 Realtime UI

Prefer a small authenticated server-sent event/invalidation channel:

```text
scope + resource/projection + revision
```

The browser re-fetches authorized state. Do not push sensitive business payloads through the realtime channel.

---

## 8. Implementation tranches

### T0 — contract and anti-duplication audit

Issues: #265, all children

- [ ] freeze canonical ownership of Program/Matter/Evidence/Form/Workflow/Source/Report records;
- [ ] document route migration and deep-link compatibility;
- [ ] approve scope hierarchy semantics;
- [ ] approve metric contract and visual state vocabulary;
- [ ] decide exact durable-table budget for first tranche;
- [ ] define representative MTN-scale acceptance populations without embedding customer/private data;
- [ ] define performance targets for Home, Work, metric drill and notification burst.

**Exit:** no competing source of truth and no unresolved data/cardinality contract.

### T1 — Group scope foundation

Issue: #267

- [ ] hierarchy records/read models;
- [ ] legal-entity mapping;
- [ ] authorized scope selector;
- [ ] ancestor/descendant resolution;
- [ ] safe aggregate projection skeleton;
- [ ] no-cross-scope-leak tests.

**Exit:** a user can change authorized Group/OpCo scope and receive truthful aggregate coverage without raw cross-entity access.

### T2 — Metric kernel and visual language

Issue: #268

- [ ] metric definition/snapshot APIs;
- [ ] threshold/appetite state;
- [ ] unknown/excluded/freshness;
- [ ] time-series rollup;
- [ ] metric card/trend/matrix primitives;
- [ ] count/drill same-revision tests;
- [ ] semantic colour/icon/badge tokens.

**Exit:** four Home metrics can be rendered from one trusted metric contract.

### T3 — Home + Work shell

Issue: #266

- [ ] Home command surface;
- [ ] consolidated Work intervention queue;
- [ ] role/lens presentation;
- [ ] Portfolio/Insights shells;
- [ ] contextual Forms/Imports;
- [ ] deep-link compatibility;
- [ ] mobile/accessible route migration.

**Exit:** core existing workflows are reachable through the five-destination shell with no capability loss.

### T4 — Attention delivery

Issue: #269, coordinate #142

- [ ] notification intents/episodes;
- [ ] in-app inbox;
- [ ] immediate critical email;
- [ ] daily digest;
- [ ] delivery receipts/recovery;
- [ ] SSE invalidation;
- [ ] notification preference/policy controls;
- [ ] burst/load/security acceptance.

**Exit:** critical change reaches the right user once; routine work is digestible and never creates duplicate business state.

### T5 — ERM / operational risk

Issue: #270

Sequence:

1. Risks/taxonomy/ownership;
2. reusable Controls composition;
3. appetite/tolerance;
4. Indicators on #57/metric kernel;
5. RCSA on Forms/Capture;
6. loss/recovery;
7. bounded scenario/stress records.

**Exit:** risk officer can see exact outside-appetite population, impact context, controls, indicators, open Matters and movement from one record/Home drill.

### T6 — Operational resilience

Issue: #271

- [ ] critical services;
- [ ] BIA/tolerances/RTO/RPO;
- [ ] dependency references;
- [ ] exercises/tests;
- [ ] failed outcome → canonical Matter;
- [ ] Home/Insights resilience views.

**Exit:** current resilience posture is explainable without copying the CMDB/network estate.

### T7 — Audit and combined assurance

Issue: #272

- [ ] universe/plan;
- [ ] engagement/test/evidence;
- [ ] independent conclusion;
- [ ] canonical findings;
- [ ] assurance coverage relationships/matrix.

**Exit:** combined assurance is visible across risks/controls/services and preserves provider independence.

### T8 — Policy / regulatory / recertification

Issue: #273

- [ ] regulatory source/obligation lifecycle;
- [ ] policy lifecycle;
- [ ] impact preview/acknowledgement;
- [ ] certificate/recertification cadence and expiry episodes.

**Exit:** current applicability, policy state and recertification work are governed through shared foundations.

### T9 — Insights and management packs

Issue: #274

- [ ] Group/OpCo appetite matrices;
- [ ] risk/KRI/control/resilience/assurance trends;
- [ ] scheduled governed report runs;
- [ ] management/committee packs;
- [ ] delivery and reconstruction.

**Exit:** Home, Insights and generated packs use the same metric revisions and explain incomplete/stale data.

### T10 — Enterprise acceptance

Coordinate #13 and all program issues.

- [ ] production IdP/directory boundary validated;
- [ ] PostgreSQL cardinality/query-plan/load proof;
- [ ] object storage/scanning/retention prerequisites for included journeys;
- [ ] backup/restore/DR;
- [ ] notification provider outage/retry;
- [ ] Group aggregation security tests;
- [ ] representative CRO/CCO/CISO/risk manager/control owner/internal-audit timed usability;
- [ ] light/dark/mobile/200%/keyboard/screen-reader evidence;
- [ ] copy review: concise, functional, no narrative filler;
- [ ] no dashboard number without drill/freshness/population truth.

**Exit:** enterprise scope is operationally credible, not merely feature-complete.

---

## 9. Home acceptance scenario

A Group risk executive opens Home.

Within the first viewport the user can determine:

- current selected scope and freshness;
- how many risks are outside appetite;
- current KRI breaches;
- critical/high open issues;
- overdue material interventions;
- whether any values worsened since the prior comparison point;
- the top actionable items;
- which OpCo/risk area is driving the largest current pressure.

Selecting a metric opens the exact authorized population behind that value.

Selecting a risk/OpCo matrix cell opens the corresponding risk population/profile.

No raw evidence is shown until the user crosses the normal record authorization boundary.

---

## 10. Risk officer acceptance scenario

A risk officer opens Work and can process all current intervention types without visiting domain-specific inboxes.

The officer opens a risk profile and sees:

- statement/cause/impact;
- scope/owner;
- inherent/residual state;
- appetite/tolerance;
- indicators and trend;
- controls/effectiveness/evidence;
- current losses/incidents if linked;
- assurance coverage;
- open Matters/actions;
- last material changes.

The system never makes the officer manually reconcile independent dashboard/register counts for the same underlying population.

---

## 11. Performance and scale

Define representative large-enterprise tests before implementation.

At minimum:

- thousands of users/responsibility assignments;
- multi-level Group/OpCo hierarchy;
- tens/hundreds of thousands of governed records;
- high-volume source populations remain outside ClearSight per #57;
- metric snapshots scale with metrics/scopes/periods, not source-row × run history;
- bounded/keyset drill lists;
- per-source and projection backpressure;
- notification burst dedupe;
- Home does not execute unbounded cross-domain joins synchronously.

Prefer precomputed/rebuildable metric/aggregate projections with explicit high-water/freshness.

---

## 12. Migration / rollout

Use staged activation.

1. Add hierarchy/metric infrastructure behind capability flags.
2. Run new Home metrics in shadow beside current Oversight and compare counts.
3. Reconcile every mismatch before switching navigation.
4. Ship new Home/Work shell while retaining old hashes/deep links.
5. Move top-level destinations only after equivalent Portfolio/context routes exist.
6. Add notification delivery in shadow/log-only mode before external email.
7. Roll out ERM/resilience/assurance domain lenses after shared metric/work behavior is stable.
8. Remove compatibility navigation only after usage/acceptance confirms no lost journey.

No big-bang rewrite.

---

## 13. Quality gates

Every tranche must include:

- domain/unit tests;
- PostgreSQL integration where durable state changes;
- exact scope/authorization negative tests;
- idempotency/replay/worker-restart tests where async;
- query-plan/cardinality tests for high-volume reads;
- rendered deterministic UI evidence;
- accessibility;
- mobile/320px/200% reflow;
- dark/light;
- stale/unknown/unavailable/forbidden/conflict states;
- copy review;
- point-in-time/reconstruction proof for material facts.

A screenshot that looks correct is not evidence that the count/authority/state is correct.

---

## 14. Explicit non-goals

Do not build:

- another workflow/task engine;
- another approval/decision engine;
- another evidence/artifact store;
- another source registry;
- a generic BI/dashboard designer;
- arbitrary SQL/expression widgets;
- a copied CMDB/network inventory;
- a SIEM/ITSM/AML platform;
- separate KRI scheduler;
- domain-specific notification queues;
- employee performance scoring;
- gamification/streaks;
- AI-generated risk/compliance conclusions without governed acceptance.

The intended advantage is **one coherent operating model with several domain lenses**, not feature-count parity through duplication.
