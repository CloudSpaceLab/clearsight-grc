# ClearSight Continuous Compliance Architecture

This document defines the cross-cutting architecture through which ClearSight implements continuing Programs, bounded Matters, continuous evidence, regulatory change, supervisory work, authority cases, low-effort workflows, and derived legacy views.

It conforms to:

- [`../product/continuous-compliance-operating-model.md`](../product/continuous-compliance-operating-model.md)
- [`../product/ease-of-use-standard.md`](../product/ease-of-use-standard.md)
- [`../product/operating-model.md`](../product/operating-model.md)

It composes:

- [`risk-graph-and-decision-engine.md`](risk-graph-and-decision-engine.md)
- [`living-evidence-fabric.md`](living-evidence-fabric.md)
- [`governed-ai-operators.md`](governed-ai-operators.md)
- [`product-semantics-mapping.md`](product-semantics-mapping.md)

---

# 1. Architectural objective

```text
Stable obligations remain governed in Programs.
Change, exception, harm, uncertainty, or external demand becomes a Matter.
Approved bank sources assemble context before a user is asked to act.
AI and deterministic services prepare the next governed step.
Routine user work should require only a few steps and under five minutes of active effort.
```

The architecture must support:

- source-backed Requirements and applicability;
- scoped controls and Evidence Contracts;
- current evidence and multidimensional Compliance State;
- trigger-driven refresh;
- typed Matter workflows;
- context assembly and prefill;
- minimum-question generation;
- grounded recommendations;
- save and resume;
- review by exception;
- usability telemetry and budgets;
- regulatory-source ingestion;
- protected authority cases;
- legacy register projections;
- independent assurance;
- point-in-time reconstruction.

---

# 2. Logical modules

A modular core may contain:

```text
Identity and Authorization
Institution and Scope
Source Registry and Integration
Observation and Evidence
Programs and Requirements
Controls and Compliance State
Trigger and Scheduling
Matters and Cases
Context Assembly and Prefill
Recommendation and Task Compilation
Requests and Capture
Decision and Approval
Action and External Execution
Verification and Assurance
Regulatory and Authority Intelligence
Search, Projections, Reporting and Export
Workflow-Efficiency Telemetry
Governed AI Runtime
Audit and Temporal Reconstruction
```

Modules may begin in one deployable unit. Boundaries must remain explicit.

---

# 3. Authoritative data and projections

Authoritative relational records include:

- Programs;
- Requirements and Applicability Conclusions;
- Control Objectives and Implementations;
- Evidence Contracts;
- Matters and Matter state;
- Sources and Observations;
- Conclusions and Compliance State versions;
- Decisions, Actions, Response Packages, and Verification;
- workflow state and assignments;
- temporal and audit metadata.

Rebuildable projections include:

- search;
- graph traversal;
- vector retrieval;
- dashboards and KRIs;
- register-compatible views;
- committee and examination views;
- work queues and saved views.

A projection cannot become a separate truth system.

---

# 4. Source Registry and integration fabric

Every source profile contains:

- owner and custodian;
- authoritative facts;
- limitations;
- scope and identifiers;
- expected freshness;
- current health;
- mapping version;
- purpose and access policy;
- known data-quality issues;
- dependent claims and Programs.

Integration levels:

- controlled values and manual capture;
- spreadsheet and document imports;
- scheduled files and database exports;
- APIs;
- events and telemetry.

All produce governed Observations.

Priority inventory adapters should expose normalized references for:

- legal entities and branches;
- organization and people;
- applications and systems;
- assets;
- customers and accounts where approved;
- vendors and contracts;
- projects and changes;
- policies and documents;
- ROPA and BIA;
- channels, merchants, terminals, and ATMs.

---

# 5. Context Assembly and Prefill Service

The Context Assembly Service prepares a purpose-specific context package before a workflow renders.

Inputs:

- authenticated actor and authority;
- Program or Matter;
- active scope and period;
- linked Requirements, controls, claims, and evidence;
- institution profile;
- approved inventory records;
- prior submissions and decisions;
- source health and freshness;
- workflow state;
- policy and sensitivity.

Outputs:

- prefilled structured fields;
- read-only sourced values;
- correctable values and correction route;
- unresolved facts;
- contradictions;
- missing authority or scope;
- relevant history;
- safe next actions.

Requirements:

- authorization before retrieval and after relationship expansion;
- source and freshness metadata preserved;
- no unqualified fallback value silently replaces authoritative data;
- context package version recorded for material review;
- cache entries remain tenant, purpose, and scope bound.

This service is central to the five-minute usability budget.

---

# 6. Evidence Need and Minimum-Question Compiler

The compiler converts claims and Evidence Contracts into the smallest necessary human request.

Pipeline:

```text
Claim and purpose
→ retrieve current authorized evidence
→ evaluate sufficiency and contradiction
→ identify exact unresolved facts
→ rank best sources
→ choose least burdensome approved response form
→ generate focused request
→ stop or cancel when evidence arrives elsewhere
```

It must support:

- forms;
- controlled selections;
- photo or scan requests;
- document upload;
- spreadsheet correction;
- redirect, delegate, partial, not applicable, and sensitivity concern;
- delivery through approved channels.

Request state and active-effort telemetry are audited.

---

# 7. Recommendation and Task Compilation Service

This service combines deterministic policy, institutional context, and governed AI to prepare useful first drafts.

Recommendation types:

- regulatory Requirement candidates;
- applicability questions;
- control mappings;
- evidence requests;
- Matter summaries;
- owner and routing suggestions;
- remediation options;
- verification criteria;
- policy changes;
- response-package indexes;
- review plans;
- source and entity matches.

Every recommendation contains:

- recommendation type and version;
- affected Program, Matter, and scope;
- source references and versions;
- explicit facts;
- inferred values;
- assumptions;
- uncertainty and contradictions;
- required authority;
- expected next state;
- editable structured output;
- alternatives;
- model/rule/operator lineage.

Recommendations are proposals. Domain commands apply only after validation, authorization, policy, and approval.

---

# 8. Trigger engine

Trigger types:

- calendar;
- institutional change;
- external regulatory or authority change;
- operational event;
- evidence expiry or contradiction;
- source degradation;
- threshold breach;
- verification failure.

A trigger may:

- recompute Program state;
- create or update a Matter;
- generate a request;
- route a recommendation;
- invalidate a decision;
- schedule review;
- update a filing package;
- notify a user only when intervention is required.

Trigger processing must be idempotent, version-aware, explainable, and replayable.

---

# 9. Program computation

Program state is a derived, versioned conclusion over:

- current Requirement set;
- approved applicability;
- scoped controls;
- Evidence Contracts;
- Observations and source health;
- exceptions and waivers;
- assurance conclusions;
- schedule and filing state;
- open Matters.

The Program service emits dimensions rather than one opaque score.

Program pages use projections optimized for exception-focused review.

---

# 10. Matter composition

Matter type controls which sections and state transitions apply.

Common components:

- trigger and source;
- scope and affected objects;
- evidence and missing facts;
- assessment;
- authority and decision;
- action or response;
- outcome or acknowledgement;
- verification;
- history.

Matter workflows must support:

- typed finite states;
- save and resume;
- changed-since-last-view summary;
- assignment, redirect, delegate, conflict, and escalation;
- durable background work;
- direct navigation to current step;
- closure contract by Matter type.

---

# 11. Workflow-efficiency telemetry

Every key flow emits privacy-minimized usability events:

- flow and version;
- role and coarse scope type;
- start, pause, resume, completion, and abandonment;
- active interaction time;
- workspace transitions;
- fields entered manually;
- fields prefilled;
- corrections;
- redirects and delegations;
- AI recommendation accepted, edited, rejected, or unavailable;
- source or integration fallback;
- accessibility mode where safely measurable;
- outcome and error class.

Do not record sensitive field content in usability telemetry.

The telemetry service calculates:

- median and p90 active effort;
- transition count;
- manual-to-prefilled ratio;
- duplicate request rate;
- time to resume;
- recommendation edit/rejection rate;
- abandonment and correction rate;
- accessibility parity.

Release gates may consume these metrics.

---

# 12. Save, resume, and asynchronous work

Durable workflow state records:

- completed steps;
- current step and owner;
- draft structured values;
- source/context package version;
- pending background work;
- blockers;
- due date;
- last viewed state;
- changes since last view;
- recommended next action.

Background operations include imports, source retrieval, AI processing, external execution, package generation, and verification observation.

The UI must remain usable while these operations run.

---

# 13. Authorization and privacy

Effective access is the intersection of:

- tenant;
- legal entity and scope;
- role and relationship;
- purpose;
- Program or Matter assignment;
- data classification;
- legal privilege;
- authority;
- current workflow state.

Authorization applies to:

- context assembly;
- prefill;
- search and graph expansion;
- recommendation inputs;
- counts and suggestions;
- exports and response packages;
- usability telemetry;
- background jobs;
- AI tools.

Protected reporting and authority cases require isolated content and restricted indexing.

Ease of use may not widen access.

---

# 14. Governed AI runtime

AI capabilities use:

- approved model gateway;
- source-grounded context;
- structured output schemas;
- allowlisted tools;
- confidence and abstention;
- domain validation;
- policy and authority checks;
- human review where required;
- immutable invocation audit;
- degraded manual mode.

AI capability release requires both correctness/safety thresholds and measurable effort reduction.

---

# 15. Import and mapping acceleration

Import architecture stores:

- source profile;
- file and sheet;
- mapping template and version;
- schema fingerprint;
- row provenance;
- validation results;
- entity matches;
- accepted and rejected observations;
- reconciliation report.

Repeat imports compare the new schema fingerprint with the approved mapping and route only changes or exceptions for review.

AI may suggest mappings and matches but cannot silently merge material entities.

---

# 16. Derived legacy views

Legacy views are projections:

- compliance register;
- risk register;
- exception register;
- workplan;
- RCSA;
- KRI;
- BIA;
- vendor register;
- loss register;
- dashboard;
- filing and examination package.

Each view must drill into canonical Programs, Matters, sources, evidence, and decisions.

No edit in a projection may bypass domain services.

---

# 17. Failure and degraded mode

The system must handle:

- source stale or unavailable;
- model unavailable;
- import failure;
- workflow worker restart;
- partial external action;
- authorization revocation;
- projection corruption;
- offline capture conflict.

Required behavior:

- deterministic context remains available;
- exact stale age is shown;
- safe fallback is offered;
- unsafe action is blocked;
- drafts remain resumable;
- retries are idempotent;
- recovery is audited.

---

# 18. Architecture acceptance

Architecture is conformant only when:

- Program and Matter workflows use the same shared primitives;
- approved inventories prefill routine flows;
- focused requests contain only unresolved facts;
- AI provides grounded first drafts without becoming authority;
- routine flows can meet five-minute budgets;
- complex flows preserve a safe checkpoint;
- replacing a spreadsheet source with an API removes effort without changing semantics;
- source limitations and contradictions remain visible;
- accessibility users can complete equivalent workflows;
- external completion cannot set verified state;
- point-in-time reconstruction includes context, recommendation, review, action, and outcome;
- usability telemetry does not leak sensitive content.