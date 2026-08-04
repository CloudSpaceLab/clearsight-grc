# Product Semantics to Architecture Mapping

This document maps ClearSight’s canonical user-facing semantics to its deeper graph, evidence, workflow, authorization, and AI architecture.

The architecture was originally described through the Institutional Risk Graph, Materiality Compiler, Living Evidence Fabric, Decision Ledger, and Governed AI Operators. Those remain useful internal mechanisms. They are not the primary product language or mandatory navigation.

In case of semantic conflict:

1. [`../product/continuous-compliance-operating-model.md`](../product/continuous-compliance-operating-model.md) controls Programs-and-Matters behavior;
2. [`../product/operating-model.md`](../product/operating-model.md) controls shared object meaning;
3. specialized product specifications control their domains;
4. architecture documents control internal implementation mechanisms.

---

# 1. Core rule

> **Architecture explains how ClearSight works internally. Programs and Matters define how users operate it.**

The graph, evidence services, materiality analysis, workflow runtime, decision records, projections, and AI operators may span multiple internal modules. Users should ordinarily experience them through a Program, Matter, focused Capture/Respond flow, Work queue, or authorized inquiry.

---

# 2. Canonical mapping

| Product object | Internal architectural representation | Important boundary |
|---|---|---|
| Program | Versioned aggregate/projection connecting Authority Sources, Requirements, applicability, policies, controls, Claims, Evidence Contracts, Review Activities, indicators, Matters, assurance, filings, and history | Program is not one giant table or service; it is the continuing product aggregate over shared records |
| Matter | Typed workflow aggregate connecting source/trigger, scope, evidence, Conclusion, Decision/response, Actions, Verification/acknowledgement, and history | Matter types share common mechanics but retain domain-specific authority and privacy |
| Scope | Institution, legal entity, licence, jurisdiction, business unit, Program, service, process, branch, product, project, vendor, customer/account, asset population, and typed relationships | Scope must be visible before action; ontology must not become ordinary navigation |
| Authority Source | Immutable Evidence Item/source record plus authority, authenticity, document type, version, dates, confidentiality, and supersession relationships | Working register rows are secondary observations, not Authority Sources |
| Source Provision | Addressable source fragment with coordinates and dependencies | Every material Requirement, finding, or directive must preserve exact lineage |
| Requirement | Versioned governance/obligation entity derived from Source Provisions | Source text, normalized Requirement, applicability, policy, and control remain distinct |
| Applicability Conclusion | Scoped versioned Conclusion using institution facts, source interpretation, and authority | AI may propose; authorized human approval is required where material |
| Policy | Versioned governance document aggregate with lifecycle and source relationships | Policy does not itself prove implementation or effectiveness |
| Control Objective | Governance outcome entity | Remains distinct from scoped implementations |
| Control Implementation | Scoped operational aggregate linked to systems, processes, people, vendors, evidence, exceptions, and tests | One objective may have many implementations |
| Exposure Pattern | Reusable risk-scenario/failure-pattern template | Reusable across Programs and channels; not an active incident or Matter |
| Risk Situation | Matter subtype plus exposure, appetite, affected scope, evidence, Decision, Action, and Verification state | Risk Situation is one Matter type, not the sole product aggregate |
| Claim | Living Evidence Fabric Claim | Precise statement for purpose, scope, population, and period |
| Evidence Contract | Versioned sufficiency policy over required facts, sources, coverage, freshness, independence, contradiction, authority, and triggers | Evidence Recipe is a task/template representation of the same policy |
| Observation | Depending on use: Signal, Evidence Item, Evidence Assertion, source-health record, import row, test result, human assertion, event, or measurement | Observation is not automatically a verified fact |
| Conclusion | Claim Conclusion, compliance Conclusion, risk-state Conclusion, control-effectiveness Conclusion, assurance Conclusion, reportability Decision input | Included/excluded evidence, assumptions, contradiction, authority, and valid period are required |
| Compliance State | Rebuildable governed projection over interpretation, applicability, design, implementation, evidence, effectiveness, exception, assurance, deadline, and source quality | No unexplained single score may become authoritative |
| Review Activity | Scheduled/durable workflow plus scope, Requirements, controls, Evidence Contracts, owner, reviewer, and results | Workplan and calendar are views, not duplicate task truth |
| KRI/Indicator | Definition plus source query/population, values, threshold, history, and response policy | Value should derive from canonical observations/Matters where possible |
| Decision | Decision Ledger aggregate | Human authority remains distinct from AI recommendation and workflow state |
| Action | Action/remediation aggregate and external execution reference | External task completion is implementation state, not verified outcome |
| Verification Contract | Decision/remediation verification aggregate | Evaluates defined outcome criteria without overstating causal proof |
| Response Package | Point-in-time evidence/export aggregate with directives, inclusion/exclusion, redaction, approval, transmission, acknowledgement, retention, and manifest | Package generation and transmission do not automatically close underlying Matter or Program gap |
| Source Profile | Source Registry, integration trust, mapping, freshness, limitation, purpose, health, and dependent-object metadata | Automation does not create authority by itself |

---

# 3. Program architecture

A Program is best implemented as a governed aggregate and set of projections over canonical records rather than a self-contained schema copied for every compliance domain.

A Program composes:

```text
Program identity and scope
├── Authority Sources and Source Provisions
├── Requirements and Applicability Conclusions
├── Policies
├── Control Objectives and scoped Control Implementations
├── Claims and Evidence Contracts
├── Source Profiles and current Observations
├── Review Activities, calendars, and indicators
├── Compliance State projections
├── linked Matters and exceptions
├── independent assurance Conclusions
├── filings, certifications, and Response Packages
└── temporal history
```

Internal bounded contexts may own these records. The Program service/projection must provide one coherent authorized experience.

Program changes must emit durable events such as:

- `ProgramActivated`
- `RequirementApproved`
- `ApplicabilityChanged`
- `ControlImplementationChanged`
- `EvidenceContractChanged`
- `ComplianceStateChanged`
- `ReviewActivityDue`
- `ProgramMatterCreated`
- `FilingReadinessChanged`
- `AssuranceConclusionIssued`

Events carry safe references, not raw restricted evidence.

---

# 4. Matter architecture

Matter is a typed durable workflow aggregate.

Common Matter mechanics:

- source or trigger;
- classification and type;
- scope and affected objects;
- owner, authority, deadline, escalation;
- Claims, evidence, Conclusion, and contradiction;
- Decision or response;
- Actions and external execution;
- Verification or acknowledgement;
- communications and history.

Matter subtypes may add rules:

## Regulatory Change Matter

- Authority Source and provisions;
- candidate Requirements;
- interpretation and applicability review;
- impacted Programs and controls;
- implementation Actions and Evidence Contracts;
- amendment propagation.

## Supervisory Matter

- authority finding or expectation;
- management response;
- commitments and milestones;
- response package;
- effectiveness verification.

## Authority Request Case

- protected source and legal-instrument review;
- subjects and match state;
- directives and requested periods;
- disclosure/action authority;
- legal hold;
- protected tasks and response package;
- acknowledgement and minimized outputs.

## Risk Situation

- Exposure Patterns;
- materiality and appetite;
- affected services/customers/assets/vendors;
- risk and evidence state;
- treatment Decision and Verification.

## Finding/Exception/Incident/Breach/Vendor Matter

Each retains its domain-specific authority, timelines, privacy, and closure requirements while using common evidence and workflow mechanisms.

---

# 5. Materiality and Matter creation

The Materiality Compiler should not create an executive alert stream detached from Programs and Matters.

It may:

- create a new Matter;
- update, group, split, merge, reopen, or supersede Matters;
- change required handling or authority;
- link a Matter to affected Programs, Requirements, controls, and scopes;
- suppress executive visibility while preserving analyst access;
- trigger evidence refresh or Review Activity.

It must separately represent:

- estimated exposure or compliance impact;
- evidence uncertainty;
- source and data-quality debt;
- deadline and velocity;
- decision relevance;
- confidence and alternative interpretations.

A stable continuing Requirement does not become a Matter merely because it exists. A trigger creates a Matter when action or judgment is required.

---

# 6. Regulatory-source architecture

Authority Source processing should use:

```text
Original artifact
→ integrity/authenticity validation
→ document classification
→ provision segmentation
→ candidate Directive Atoms or findings
→ structured review
→ approved Requirements, directives, or supervisory Matters
→ applicability and institution mapping
```

Required internal separation:

- original source bytes;
- extracted text and structure;
- candidate AI output;
- reviewer edits;
- approved source interpretation;
- institution-specific Requirement;
- applicability Conclusion;
- control and evidence mapping.

Amendment and supersession propagate through versions and dependent objects without overwriting historical state.

---

# 7. Evidence architecture

Observation is the common capture contract.

An Observation may be represented internally as:

- Signal;
- Evidence Item;
- Evidence Assertion;
- source-health record;
- test result;
- import row;
- system event;
- human assertion;
- communication;
- reconciliation result.

One artifact may produce multiple Observations.

The product must preserve the difference between:

- explicit source value;
- machine-extracted value;
- inferred candidate;
- human-confirmed value;
- approved Conclusion.

Evidence Contracts bind multidimensional sufficiency to a Claim and Program/Matter purpose.

Source degradation must propagate to dependent Claims, Compliance State, filings, Decisions, and Matters without falsely asserting control failure.

---

# 8. Review Activity, KRI, and legacy-view architecture

## Review Activity

Use durable workflow/scheduling with versioned scope, owner, reviewer, due date, frequency, Evidence Contract, and outcome.

Annual workplans and Program calendars are projections over Review Activities.

## KRI and compliance indicator

Indicator definition includes population query, source/version, measure, period, threshold, exclusions, owner, and response policy.

Values derive from authorized canonical records where possible. Overrides are versioned Decisions with rationale and expiry.

## Legacy register and dashboard

Compliance registers, risk registers, exception trackers, RCSA workbooks, BIA views, vendor registers, loss registers, and dashboards are read/write projections only where writes are mapped back through governed domain commands.

A projection must not become a second authoritative store.

---

# 9. Decision, workflow, and response architecture

The Decision Ledger remains authoritative for material judgment.

Users ordinarily review Decisions inside Program or Matter context rather than a separate Decision Ledger module.

Actions use domain commands and replaceable adapters. External system success becomes implementation evidence and execution state.

Response Package generation uses current authorized source/evidence versions, directives, inclusion/exclusion, redaction, approval, signatory, transmission, acknowledgement, and manifest.

A successful email/API/file transfer does not itself prove complete or accepted response.

---

# 10. Governed AI architecture

AI operators remain internal constrained actors.

Product-facing principle:

> AI compiles messy authoritative and institutional inputs into proposed structured Program and Matter objects and explanations.

AI may propose:

- source segmentation;
- Directive Atoms and candidate Requirements;
- applicability candidates;
- control and evidence mappings;
- Observations and entity matches;
- contradictions;
- focused requests;
- Program summaries and Compliance State explanations;
- Matter summaries, options, Actions, tests, and response drafts.

AI must not become:

- the source of institutional truth;
- the legal or risk authority;
- evidence itself;
- the only operating method;
- or a route around domain commands and policy.

All production AI use requires service identity, purpose, scope, approved sources, model/prompt/schema versions, tool allowlists, validation, authorization, approval, audit, evaluation, monitoring, and degraded operation.

---

# 11. Authorization and protected domains

Authorization applies to:

- Programs and Requirements;
- source artifacts and provisions;
- applicability and legal interpretations;
- controls and evidence;
- Matters and case subjects;
- search, counts, graph paths, caches, embeddings, analytics, exports, notifications, and AI retrieval;
- bulk actions and projections.

Protected reporting and Authority Request Cases should use isolated content and identity boundaries.

Recommended relationship:

```text
Protected domain
├── protected source/case content
├── subject or reporter identity controls
├── investigator/legal workspace
├── restricted evidence and response
└── approved protected AI route

Ordinary ClearSight domain
└── receives only approved minimized Program/Matter signals or aggregate indicators
```

---

# 12. Initial technical shape

Recommended starting architecture:

- relational authoritative store for canonical entities, Programs, Matters, versions, and relationships;
- immutable/versioned object storage for source and evidence artifacts;
- transactional outbox and durable workflow runtime;
- authorization policy service or library;
- rebuildable search and optional graph projections;
- model gateway and operator runtime;
- integration SDK;
- append-only audit store.

A dedicated graph engine, vector database, large microservice estate, or autonomous-agent platform is not required for the first release.

---

# 13. Required architecture-alignment tests

Architecture is conformant only when:

- a Program can present Requirements, controls, evidence, calendar, Matters, assurance, filing, and history without duplicate module truth;
- a Matter can be handled without module hopping;
- the same Requirement/control/evidence objects support register, Program, Matter, KRI, assurance, and export views;
- legacy spreadsheet import remains secondary until reconciled;
- NDPA ROPA, DPIA, breach, vendor, and filing workflows share one Program;
- a regulatory amendment updates dependent Programs and Matters without overwriting history;
- a protected authority request cannot leak subjects or perform unauthorized actions;
- forms, photos, spreadsheets, APIs, and telemetry produce traceable Observations;
- source authority and limitations are enforced;
- Compliance State dimensions remain explainable;
- materiality creates or updates Matters rather than flooding alerts;
- AI cannot bypass domain services;
- external task completion cannot close a Matter or Program gap;
- and point-in-time reconstruction includes source, Requirements, applicability, controls, evidence, conclusions, Matters, Decisions, Actions, responses, and verification.