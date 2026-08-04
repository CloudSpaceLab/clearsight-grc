# Institutional Risk Graph and Decision Engine

This document defines the semantic and decision architecture that turns fragmented institutional data into explainable material risk decisions.

The architecture has three tightly connected parts:

1. **Institutional Risk Graph** — the time-aware model of the institution and its risk relationships.
2. **Materiality Compiler** — the mechanism that converts raw signals into decision-relevant risk movement.
3. **Decision Ledger and Assurance Loop** — the durable record of authorized choices, actions, and verified outcomes.

---

# 1. Architectural goals

The subsystem must enable ClearSight to:

- represent bank risk as a connected institutional state rather than isolated registers;
- preserve provenance and point-in-time history;
- distinguish source facts from derived conclusions;
- explain why a change became material;
- aggregate without hiding uncertainty or concentration;
- identify the authority required for a decision;
- compare treatment options;
- preserve decision rationale and conditions;
- and update risk only after relevant evidence or verified outcomes.

---

# 2. Canonical semantic layers

The graph must separate five semantic layers.

## 2.1 Institutional structure

Stable or slowly changing context:

- institution;
- legal entities;
- business units;
- committees;
- accountable roles;
- locations and branches;
- products;
- customer segments;
- critical operations and business services;
- processes;
- systems and infrastructure;
- data assets;
- models and AI systems;
- vendors and fourth parties;
- contracts;
- and jurisdictions.

## 2.2 Governance and obligation

What the institution is expected or required to do:

- regulatory sources;
- obligations;
- licenses;
- policies;
- standards;
- risk appetite statements;
- limits and tolerances;
- control objectives;
- control implementations;
- authority matrices;
- and committee mandates.

## 2.3 Risk and operational reality

What may happen or has happened:

- risk taxonomies;
- risk scenarios;
- causes;
- events;
- incidents;
- near misses;
- losses;
- complaints;
- vulnerabilities;
- threats;
- control failures;
- concentrations;
- and emerging risks.

## 2.4 Evidence and assurance

What supports or contradicts institutional claims:

- claims;
- evidence versions;
- assertions;
- tests;
- assessments;
- contradictions;
- assurance conclusions;
- findings;
- and evidence debt.

## 2.5 Decision and action

How the institution responds:

- decisions;
- options;
- approvals;
- dissent;
- risk acceptances;
- exceptions;
- actions;
- dependencies;
- investments;
- verification contracts;
- observed outcomes;
- and lessons learned.

---

# 3. Core entity distinctions

Several entities that are commonly collapsed must remain separate.

## 3.1 Risk taxonomy versus risk scenario

- **Taxonomy item:** a classification such as cyber risk, conduct risk, or third-party risk.
- **Risk scenario:** a specific causal statement involving an event, affected asset or service, and potential consequence.

Example scenario:

> A failure of the primary payment processor during peak volume prevents retail transfers for more than the approved impact tolerance, causing customer harm, liquidity effects, regulatory notification, and reputational loss.

## 3.2 Control objective versus control implementation

- **Control objective:** the outcome that must be achieved.
- **Control implementation:** the actual mechanism operating in a defined scope.

A global objective may have multiple implementations by legal entity, system, process, or vendor.

## 3.3 Obligation source versus normalized obligation

- **Source:** original regulation, standard, license, contract, or internal policy.
- **Obligation:** normalized requirement with applicability, effective dates, jurisdiction, and provenance.

## 3.4 Signal versus incident

- **Signal:** an observation that may indicate a relevant change.
- **Incident:** a governed determination that an event meeting defined criteria occurred.

## 3.5 Finding versus issue

- **Finding:** an observed deficiency or conclusion from a review, test, incident, or assurance activity.
- **Issue:** a governed remediation object created to address one or more findings or exposures.

## 3.6 Action completion versus outcome

- **Action completion:** the planned work was performed.
- **Outcome:** the intended effect was observed.

## 3.7 Risk state versus decision

Risk state is a conclusion derived from evidence and context. A decision is an authorized choice about how to respond.

The system must not silently change risk state solely because a decision was approved.

---

# 4. Relationship model

Representative relationship types include:

## Institutional

- `OWNS`
- `OPERATES`
- `DELIVERS`
- `SUPPORTS`
- `DEPENDS_ON`
- `HOSTED_IN`
- `PROCESSES`
- `SERVES`
- `CONTRACTED_WITH`
- `SUBCONTRACTS_TO`
- `GOVERNED_BY`

## Governance

- `APPLIES_TO`
- `REQUIRES`
- `IMPLEMENTED_BY`
- `SATISFIED_BY`
- `OVERSEEN_BY`
- `APPROVED_BY`
- `ESCALATES_TO`

## Risk

- `CAUSES`
- `CONTRIBUTES_TO`
- `EXPOSES`
- `REALIZES`
- `AMPLIFIES`
- `PROPAGATES_TO`
- `MITIGATED_BY`
- `TRANSFERRED_TO`
- `WITHIN_APPETITE_OF`

## Evidence

- `SUPPORTS_CLAIM`
- `CONTRADICTS_CLAIM`
- `DERIVED_FROM`
- `TESTS`
- `ASSURES`
- `INVALIDATES`
- `SUPERSEDES`

## Decision and action

- `DECIDES_ON`
- `SELECTS_OPTION`
- `AUTHORIZES`
- `CREATES_ACTION`
- `DEPENDS_ON_ACTION`
- `VERIFIED_BY`
- `CHANGES`
- `REOPENS`

Every material edge should support:

- source and provenance;
- valid time;
- record time;
- confidence;
- lifecycle state;
- sensitivity;
- and version.

---

# 5. Temporal model

The graph should support bitemporal reasoning.

## 5.1 Valid time

When the entity state or relationship was true in the real or governed institutional world.

Examples:

- a vendor contract was effective from January to December;
- a system supported a service beginning on a deployment date;
- a control implementation applied during a defined review period.

## 5.2 Record time

When ClearSight learned, recorded, corrected, or superseded the information.

This enables questions such as:

- What was the actual service dependency on 30 June?
- What did the risk committee know about the dependency on 30 June?
- Which evidence became available only after the incident?

## 5.3 Supersession

Material records are not overwritten.

A correction creates a new version with:

- prior version;
- correction reason;
- actor;
- record time;
- and effect on dependent conclusions.

---

# 6. Authoritative storage and graph projection

A dedicated graph database is not required for the first implementation.

Recommended starting model:

- relational authoritative store for entities, versions, and relationships;
- append-only event and audit records;
- graph projection optimized for traversal and visualization;
- search and vector projections for authorized discovery;
- object storage for evidence content.

A dedicated graph engine should be introduced only if benchmarks demonstrate a need for:

- deep traversal latency;
- large-scale path analysis;
- graph algorithms;
- independent graph scaling;
- or deployment isolation.

The authoritative model and event contracts must remain portable.

---

# 7. Signal model

A signal records an observation without prematurely asserting a final conclusion.

Required fields:

- signal ID;
- type;
- source;
- source event or object ID;
- observed time;
- effective time;
- affected entities;
- raw or normalized value;
- classification;
- provenance;
- preliminary confidence;
- and ingestion state.

Signal examples:

- service latency increased;
- regulator published a change;
- evidence expired;
- vendor rating declined;
- control test failed;
- complaint volume concentrated around one product;
- privileged account reactivated;
- model drift exceeded threshold;
- recovery test missed impact tolerance;
- or a protected report alleged control bypass.

Signals must remain distinguishable from verified events and incidents.

---

# 8. Materiality Compiler

The Materiality Compiler determines whether a signal or collection of signals should create, update, suppress, group, or escalate a material risk item.

## 8.1 Inputs

- normalized signals;
- current graph context;
- risk appetite and tolerances;
- risk and control state;
- customer and service criticality;
- financial and non-financial impact models;
- jurisdiction and regulatory deadlines;
- evidence sufficiency and contradiction;
- dependency and concentration paths;
- historical incidents and losses;
- existing decisions and actions;
- authority matrix;
- and current time context.

## 8.2 Processing stages

### Stage 1: trust and normalization

- authenticate source;
- validate schema;
- normalize units and identifiers;
- resolve entities;
- deduplicate;
- and determine source health.

### Stage 2: contextual enrichment

- connect affected services, customers, entities, vendors, systems, obligations, and controls;
- identify criticality;
- retrieve appetite and authority;
- and gather current evidence state.

### Stage 3: causal grouping

Group signals when they plausibly represent the same underlying exposure.

Grouping must preserve:

- source signals;
- grouping rationale;
- confidence;
- and alternative groupings when ambiguity matters.

### Stage 4: impact and velocity assessment

Estimate:

- current impact;
- plausible impact range;
- likelihood or frequency where meaningful;
- time-to-impact;
- persistence;
- reversibility;
- detectability;
- and propagation.

### Stage 5: appetite comparison

Evaluate against:

- quantitative thresholds;
- qualitative prohibited states;
- tolerance windows;
- exception conditions;
- and delegated authority.

### Stage 6: evidence adjustment

Evidence weakness does not necessarily increase the underlying risk, but it reduces confidence and may increase governance urgency.

The compiler separately represents:

- estimated exposure;
- uncertainty range;
- evidence debt;
- and need for decision or investigation.

### Stage 7: decision relevance

Determine whether the item requires:

- no action;
- continued monitoring;
- evidence collection;
- delegated operational action;
- risk-owner decision;
- executive decision;
- committee escalation;
- incident declaration;
- regulatory assessment;
- or immediate protected handling.

### Stage 8: explanation generation

The compiler emits a structured explanation containing:

- material change;
- why now;
- affected scope;
- key relationships;
- appetite position;
- evidence state;
- assumptions;
- confidence;
- and required authority.

## 8.3 Materiality output

A materiality assessment is versioned and includes:

- source signals;
- graph snapshot or references;
- rules and model versions;
- impact dimensions;
- appetite comparison;
- evidence state;
- decision relevance;
- priority;
- confidence;
- and review state.

## 8.4 No single-score dependency

ClearSight may compute scores for ordering or comparison, but no material decision may depend on an unexplained composite score.

The interface should expose the dimensions that drove materiality.

---

# 9. Risk state model

A risk scenario may have multiple state views:

- inherent risk;
- current risk;
- residual risk;
- target risk;
- stressed risk;
- and accepted risk.

Each state must identify:

- assessment method;
- dimensions;
- assumptions;
- evidence;
- control state;
- time scope;
- confidence;
- assessor or operator;
- and approval state.

## 9.1 Qualitative assessments

Qualitative scales must have defined semantics and examples.

Avoid arithmetic that assumes ordinal labels are precise quantities.

## 9.2 Quantitative assessments

Quantitative models should expose:

- distributions or ranges;
- data source;
- calibration period;
- scenario assumptions;
- correlation or dependency assumptions;
- uncertainty;
- and sensitivity.

## 9.3 Hybrid assessments

A hybrid model may combine qualitative and quantitative evidence but must preserve the meaning of each dimension.

---

# 10. Risk appetite as executable governance

Risk appetite begins as board-approved language and becomes versioned policy.

## 10.1 Appetite statement

Includes:

- statement;
- scope;
- rationale;
- owner;
- approval authority;
- effective period;
- and related objectives.

## 10.2 Metrics and thresholds

May include:

- limits;
- triggers;
- tolerances;
- prohibited conditions;
- time-bound exceptions;
- and escalation rules.

## 10.3 Authority matrix

Defines who may:

- accept risk;
- approve exceptions;
- change a control;
- extend a deadline;
- declare an incident;
- reveal protected identity;
- approve regulatory communication;
- and authorize automation.

Authority may depend on:

- risk severity;
- legal entity;
- customer impact;
- duration;
- reversibility;
- financial amount;
- evidence quality;
- and conflict of interest.

## 10.4 Appetite evaluation

An appetite result must include:

- applicable appetite version;
- evaluated scope;
- threshold or qualitative rule;
- current state;
- breach or approach state;
- duration;
- and required escalation.

---

# 11. Decision Ledger

A material decision is a first-class aggregate.

## 11.1 Decision record

Required fields:

- decision ID and version;
- decision type;
- subject risk, claim, incident, issue, or obligation;
- materiality assessment;
- graph context;
- evidence set;
- uncertainties and contradictions;
- available options;
- selected option;
- rationale;
- authority policy;
- approvers and challengers;
- dissent;
- conditions;
- effective time;
- expiry and review triggers;
- action plan;
- verification contract;
- and outcome state.

## 11.2 Decision types

- risk treatment;
- risk acceptance;
- exception or waiver;
- incident declaration;
- regulatory reportability;
- control design change;
- remediation closure;
- protected identity reveal;
- vendor onboarding or continuation;
- model approval;
- and automation authorization.

## 11.3 Option model

Each option includes:

- description;
- projected risk movement;
- expected benefit;
- cost;
- implementation time;
- dependencies;
- operational impact;
- customer impact;
- reversibility;
- uncertainty;
- and verification method.

Projected values must be distinguishable from observed values.

## 11.4 Approval and challenge

The ledger supports:

- sequential or parallel approval;
- independent challenge;
- segregation of duties;
- conflict detection;
- conditional approval;
- rejection;
- return for evidence;
- and emergency authority with later review.

A context-free approval button is prohibited.

## 11.5 Decision expiry and invalidation

A decision can become invalid due to:

- expiry;
- appetite change;
- evidence deterioration;
- changed scope;
- incident realization;
- failed verification;
- or violated conditions.

The system must trigger reassessment without deleting the original decision.

---

# 12. Action and remediation model

## 12.1 Action plan

An action includes:

- intended outcome;
- owner;
- tasks;
- dependencies;
- due date;
- resources;
- implementation evidence;
- verification contract;
- and escalation policy.

## 12.2 External execution

Tasks may be executed in:

- ClearSight;
- ITSM;
- project management;
- IAM;
- security orchestration;
- vendor systems;
- Probo;
- or another approved engine.

ClearSight retains the risk context and expected outcome.

## 12.3 State separation

Suggested action states:

```text
PLANNED
→ AUTHORIZED
→ IN_PROGRESS
→ IMPLEMENTED
→ AWAITING_VERIFICATION
→ VERIFIED_EFFECTIVE
→ VERIFIED_INEFFECTIVE
→ BLOCKED
→ CANCELLED
→ SUPERSEDED
```

`IMPLEMENTED` must not be presented as risk reduction.

---

# 13. Verification contracts

A verification contract defines how ClearSight will determine whether an action achieved its intended outcome.

Required fields:

- outcome statement;
- measurement source;
- population or scope;
- baseline;
- success threshold;
- failure threshold;
- observation period;
- required evidence;
- acceptance authority;
- and failure response.

Example:

```yaml
outcome: "Payment failover completes within impact tolerance"
baseline: "last verified test: 67 minutes"
measure:
  source: "resilience test telemetry"
  scope: "retail instant payments"
success: "recovery <= 30 minutes with no unreconciled transactions"
observation_period: "two production-like tests"
acceptance_authority: "Operational Resilience Committee"
failure_response:
  - "reopen remediation"
  - "update current risk"
  - "escalate appetite breach"
```

---

# 14. Assurance Loop

The complete loop is:

```text
Signal
→ materiality assessment
→ evidence need
→ conclusion
→ decision
→ action
→ implementation evidence
→ outcome evidence
→ verification
→ risk update
→ learning
```

## 14.1 Learning inputs

- false-positive materiality items;
- missed signals;
- expert corrections;
- decision overrides;
- realized incidents and losses;
- verification failures;
- evidence source reliability;
- and treatment effectiveness.

## 14.2 Learning governance

Learning may improve recommendations but must not silently change:

- risk appetite;
- authority;
- evidence policy;
- protected reporting policy;
- or regulatory interpretation.

Policy changes require explicit review and versioning.

---

# 15. Graph-based authorization

Authorization may depend on graph relationships.

Examples:

- a service owner can see risks connected to owned services;
- an investigator can see assigned protected cases but not reporter identity;
- an auditor can read source evidence but cannot alter first-line conclusions;
- a legal-entity officer can approve only within the entity scope;
- a vendor can access only explicitly shared requests and evidence.

Graph traversal must apply authorization at every node and edge.

The system must prevent inference through:

- counts;
- labels;
- search suggestions;
- embeddings;
- graph layout;
- timing;
- and export manifests.

---

# 16. APIs and events

## 16.1 Representative APIs

- create or version entity;
- create or version relationship;
- query authorized graph;
- reconstruct point in time;
- ingest signal;
- assess materiality;
- create risk scenario;
- assess risk state;
- evaluate appetite;
- create decision;
- add option;
- challenge or approve decision;
- authorize action;
- define verification contract;
- record outcome;
- and supersede conclusion.

## 16.2 Representative events

- `InstitutionalRelationshipChanged`
- `SignalIngested`
- `SignalGrouped`
- `MaterialityAssessmentCompleted`
- `RiskStateChanged`
- `AppetiteApproachDetected`
- `AppetiteBreached`
- `DecisionRequested`
- `DecisionChallenged`
- `DecisionApproved`
- `DecisionExpired`
- `ActionImplemented`
- `VerificationStarted`
- `VerificationSucceeded`
- `VerificationFailed`
- `RiskConclusionSuperseded`

---

# 17. Explainability contract

Every material item must be able to answer:

1. What changed?
2. Which signals caused the change?
3. Which institutional relationships made it relevant?
4. Which appetite statements or thresholds applied?
5. What evidence supports or contradicts the conclusion?
6. Which assumptions and models were used?
7. What alternative interpretations were considered?
8. Which authority is required?
9. Which options are available?
10. How will the selected action be verified?

This explanation must be generated from structured domain state, not only from a model-written narrative.

---

# 18. Metrics

## Materiality quality

- executive items later judged non-material;
- material events detected late;
- grouping precision;
- time from signal to materiality decision;
- evidence debt on material items;
- and executive dismissal or escalation reasons.

## Decision quality

- time to accountable decision;
- decisions returned for missing evidence;
- override rate;
- expired decisions not reviewed;
- decisions invalidated by later evidence;
- and projected versus observed treatment effect.

## Remediation quality

- implemented actions awaiting verification;
- verification failure rate;
- reopened issues;
- repeat incidents;
- overdue exposure;
- and median time from implementation to verified outcome.

## Graph quality

- unresolved entity matches;
- stale relationships;
- orphaned critical services;
- source provenance coverage;
- temporal consistency errors;
- and unauthorized traversal attempts.

---

# 19. Acceptance scenarios

## Scenario A: one exposure, multiple signals

- service latency rises;
- vendor status degrades;
- a change ticket was recently completed;
- failover evidence is stale;
- and a related audit finding remains open.

The compiler creates one material item with a causal path and avoids five executive alerts.

## Scenario B: evidence uncertainty without exaggerated risk

- the underlying control may still operate;
- evidence has expired;
- no incident is observed.

The system increases evidence debt and governance urgency without falsely asserting that the control failed.

## Scenario C: decision expires after context change

- a risk was accepted for six months;
- a new critical customer segment begins using the service;
- the acceptance conditions no longer hold.

The system invalidates the active acceptance, preserves the original decision, and requests a new authorized decision.

## Scenario D: action completed but outcome failed

- remediation ticket closes;
- implementation evidence is valid;
- outcome telemetry fails the verification threshold.

The issue remains open, risk state updates, and the failed verification is visible to the appropriate authority.

## Scenario E: point-in-time board statement

The system reconstructs the risk state, evidence, appetite version, decision, and known uncertainty used for a board statement on a selected historical date.

---

# 20. Prohibited shortcuts

Do not:

- model the graph as untyped arbitrary links;
- overwrite relationship history;
- use one generic assessment table for facts, claims, risk states, and decisions;
- treat all signals as incidents;
- derive materiality from severity alone;
- hide evidence debt inside risk severity;
- use one composite score without dimensional explanation;
- update residual risk merely because an action was approved;
- close a material issue at task completion;
- allow AI narrative to become the only decision record;
- or bypass authority policy through direct integration writes.

---

# 21. Definition of success

This architecture succeeds when:

- cross-domain risk can be understood as a connected institutional state;
- executives see fewer but more relevant items;
- every material conclusion is explainable from signals, relationships, appetite, and evidence;
- decisions preserve authority and rationale;
- actions remain linked to intended outcomes;
- risk moves only when the evidence supports movement;
- and the institution can reconstruct what it knew and why it acted at any material point in time.