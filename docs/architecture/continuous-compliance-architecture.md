# ClearSight Continuous Compliance Architecture

This document defines the cross-cutting architecture through which ClearSight implements continuing Programs, bounded Matters, continuous evidence, regulatory change, supervisory work, authority cases, and derived legacy views.

It composes the existing component specifications:

- [`risk-graph-and-decision-engine.md`](risk-graph-and-decision-engine.md)
- [`living-evidence-fabric.md`](living-evidence-fabric.md)
- [`governed-ai-operators.md`](governed-ai-operators.md)
- [`product-semantics-mapping.md`](product-semantics-mapping.md)

The component documents define detailed mechanisms. This document defines how they operate together as one continuous-compliance system.

---

# 1. Architectural objective

ClearSight must support this invariant:

```text
Stable obligations remain continuously governed in Programs.
Change, exception, harm, uncertainty, or external requests become Matters.
Both use the same source, evidence, authority, decision, action, and history substrate.
```

The architecture must allow:

- source-backed Requirements and Applicability Conclusions;
- scoped controls and Evidence Contracts;
- current evidence and multidimensional Compliance State;
- trigger-driven refresh;
- typed Matters with durable workflows;
- regulatory-source ingestion and amendment handling;
- protected authority cases and Response Packages;
- legacy register and dashboard projections;
- independent assurance;
- governed AI assistance;
- point-in-time reconstruction.

---

# 2. Bounded contexts

## 2.1 Institution and Scope Context

Owns:

- institution and tenant;
- legal entities and licences;
- jurisdictions and regions;
- business units, committees, and authority roles;
- products, services, channels, branches, and processes;
- projects and processing activities;
- customers, accounts, merchants, vendors, systems, assets, data, and models;
- typed, temporal relationships and aliases.

This context provides the shared scope and relationship substrate. It does not own domain conclusions.

## 2.2 Source and Regulatory Intelligence Context

Owns:

- Source Profiles;
- Authority Sources and versions;
- authenticity and source status;
- Source Provisions and document structure;
- amendment, supersession, and related-source relationships;
- candidate Directive Atoms;
- source ingestion health and provenance.

It does not publish final Requirements or applicability without the required domain authority.

## 2.3 Program Context

Owns:

- Program identity, purpose, state, and version;
- governing source references;
- Program scope and ownership;
- Requirement membership;
- trigger subscriptions;
- calendar and filing configuration;
- linked Matters, exceptions, assurance, and package references;
- Program-level projections and history.

Program is an aggregate over shared records, not a duplicate repository for controls or evidence.

## 2.4 Requirement and Applicability Context

Owns:

- normalized Requirements;
- interpretation state;
- applicability Conclusions;
- scope, conditions, thresholds, effective dates, and exceptions;
- source-provision lineage;
- reviewer and approval history;
- amendment and supersession behavior.

## 2.5 Policy and Control Context

Owns:

- policy lifecycle;
- Control Objectives;
- scoped Control Implementations;
- owners, performers, reviewers, automation, frequency, and dependencies;
- Requirement, risk, service, system, vendor, and evidence relationships;
- design and operating-effectiveness states.

## 2.6 Evidence Context

Owns:

- Claims;
- Evidence Contracts;
- Evidence Items and immutable versions;
- normalized Observations and Assertions;
- Evidence Evaluations;
- contradictions;
- sufficiency and evidence debt;
- evidence requests and capture lifecycle;
- chain of custody, retention, and legal hold.

## 2.7 Matter and Workflow Context

Owns:

- Matter identity, type, source, scope, and lifecycle;
- owner, authority, deadlines, escalation, and communications;
- links to Programs, Requirements, controls, institutional objects, and evidence;
- durable workflow and human tasks;
- merge, split, reopen, and supersession.

Matter subtype policies determine additional required states and controls.

## 2.8 Decision, Action, Verification, and Response Context

Owns:

- material Decisions and options;
- authority, segregation of duties, rationale, conditions, and expiry;
- Actions and external execution references;
- Verification Contracts and outcome evaluations;
- Response Packages, inclusion/exclusion, redaction, signatory, transmission, acknowledgement, and manifests.

## 2.9 Review, Indicator, Assurance, and Reporting Context

Owns:

- Review Activities and schedules;
- KRI and compliance-indicator definitions and values;
- sample selection and test results;
- first-, second-, and third-line Conclusions;
- findings and assurance sign-off;
- filing, certification, board, committee, audit, and examiner projections.

Reports are projections over canonical records, not separate truth.

## 2.10 AI Operator Context

Owns:

- model gateway and provider routing;
- operator definitions and identities;
- prompt/schema/evaluation versions;
- tool registry and action classes;
- invocation audit and monitoring;
- structured proposed outputs.

AI operators call domain commands. They do not directly write authoritative stores.

---

# 3. Authoritative storage and projections

Recommended initial storage model:

```text
Relational authoritative store
├── scope and typed temporal relationships
├── Programs, Requirements, applicability, policies, controls
├── Claims, Evidence Contracts, Observations, Conclusions
├── Matters, Decisions, Actions, Verification, Responses
├── Review Activities, indicators, assurance
└── source, version, authority, and audit references

Versioned object storage
├── authority documents
├── evidence files and media
├── response and filing packages
└── integrity manifests

Durable workflow and outbox
├── human and system tasks
├── triggers and scheduled work
├── retries, replay, cancellation, compensation
└── domain events

Rebuildable projections
├── search
├── graph traversal
├── Program and Work views
├── legacy registers and dashboards
├── analytics and KRI
└── vector retrieval where authorized
```

Search, graph, vector, analytics, register, dashboard, and reporting projections are not authoritative.

---

# 4. Program computation model

A Program view is constructed from versioned references and projections.

## 4.1 Program composition

```text
Program
├── approved governing sources
├── approved Requirements
├── current Applicability Conclusions
├── linked policies and controls
├── Evidence Contracts and current evidence state
├── calendar and trigger subscriptions
├── current Compliance State dimensions
├── open Matters and exceptions
├── assurance Conclusions
├── filing/certification packages
└── historical snapshots
```

## 4.2 Compliance State computation

Compliance State is computed by dimension, never as an unexplained average.

Representative dimensions:

```yaml
interpretation: approved | pending | disputed | superseded
applicability: applicable | partial | not_applicable | unknown
control_design: adequate | partial | inadequate | unassessed
implementation: implemented | partial | planned | absent
 evidence: sufficient | partial | stale | contradictory | unavailable
operating_effectiveness: effective | ineffective | indeterminate | untested
exception: none | approved | expired | breached
assurance: assured | qualified | adverse | pending | not_reviewed
filing: current | due | at_risk | overdue | submitted | acknowledged
source_quality: current | stale | degraded | unresolved
```

A concise presentation state is a policy-controlled projection of these dimensions.

## 4.3 Program snapshot

A point-in-time Program snapshot references exact versions of:

- sources and Requirements;
- applicability;
- policies and controls;
- Evidence Contracts and evidence;
- Compliance State;
- Matters and exceptions;
- assurance;
- filings and packages.

Snapshots are reproducible and authorization-aware.

---

# 5. Trigger engine

The trigger engine evaluates durable events and schedules against Program configuration.

## 5.1 Trigger classes

### Calendar

- filing, return, certification, review, test, training, policy, vendor, BIA, RCSA, or assurance due.

### Institutional change

- new or changed product, service, project, vendor, system, model, processing activity, jurisdiction, branch, customer population, or configuration.

### Operational event

- incident, loss, breach, complaint pattern, KRI breach, control failure, audit finding, vendor deficiency, or authority request.

### Regulatory and source change

- new publication, amendment, guidance, source withdrawal, source degradation, permission revocation.

### Evidence change

- expiry, contradiction, population change, test failure, invalidation, or verification failure.

## 5.2 Trigger evaluation result

A trigger may:

- refresh a Claim;
- schedule a Review Activity;
- create or update a Matter;
- invalidate a Conclusion or Decision;
- request evidence;
- change Compliance State;
- change filing readiness;
- notify or escalate.

The engine should target affected scope rather than create broad campaigns where scope can be resolved.

## 5.3 Trigger governance

Every trigger definition includes:

- owner and purpose;
- applicable Program and scope;
- event or schedule;
- conditions and thresholds;
- deduplication window;
- materiality and authority policy;
- resulting domain command;
- suppression and escalation;
- version and audit.

---

# 6. Matter orchestration

## 6.1 Common Matter pipeline

```text
Source or trigger
→ classify and deduplicate
→ establish scope and links
→ determine required Claims/directives
→ search existing evidence
→ request missing proof
→ conclude and route authority
→ decide or prepare response
→ execute Actions
→ verify or acknowledge
→ update linked Programs and projections
→ close, continue monitoring, or reopen
```

## 6.2 Matter classification

Matter type is a governed classification. AI may propose; domain policy validates.

Misclassification tests are critical because the authority, privacy, deadline, and closure rules vary materially.

## 6.3 Matter linking

A Matter may affect multiple Programs and scopes without duplicating the Matter.

Example:

```text
Data breach Matter
├── NDPA Program
├── CBN cyber Program
├── operational risk Program
├── affected digital service
├── vendor and systems
├── customer population
└── incident/loss records
```

Each Program may project a different authorized view of the same Matter.

---

# 7. Regulatory Change Compiler

## 7.1 Pipeline

```text
Authority Source
→ authenticity and legal-status classification
→ provision segmentation
→ candidate Directive Atoms
→ human source review
→ candidate Requirements
→ interpretation approval
→ Applicability Conclusions
→ Program/control/evidence reconciliation
→ Program updates and implementation Matters
→ verification and continuing evidence
```

## 7.2 Existing-control reconciliation

For each Requirement, compare:

- scoped Control Objectives and Implementations;
- policy versions;
- Evidence Contracts and current evidence;
- exceptions and previous Decisions;
- related Matters and assurance.

Results:

- fully covered;
- covered but evidence stale;
- partially covered;
- wrong scope;
- policy-only;
- implementation-only;
- contradictory;
- uncovered;
- applicability unresolved.

## 7.3 Amendment propagation

An amendment or clarification may:

- supersede Source Provisions;
- version Requirements;
- invalidate Applicability Conclusions;
- affect controls and Evidence Contracts;
- change deadlines;
- reopen implementation Matters;
- invalidate filing or assurance states.

Propagation is explicit, reviewable, and reversible.

---

# 8. External Authority Case architecture

Authority Request Cases require stronger isolation and purpose controls.

## 8.1 Protected case data plane

```text
Protected case store
├── authority source and legal instrument
├── case subjects and resolution candidates
├── directives and requested periods
├── protected evidence and communications
├── legal, AML, fraud, KYC, branch, and records tasks
├── Decisions and approvals
├── Response Package
└── acknowledgement, retention, and legal hold
```

## 8.2 Minimized ordinary-plane output

Ordinary Programs, KRIs, and executive views receive only approved minimized data such as:

- existence of an authorized Matter where permitted;
- type and deadline category;
- aggregate workload;
- systemic control or records weakness;
- anonymized trend;
- linked remediation Matter.

Subject identity, allegations, requested records, suspicious-reporting state, and protected evidence remain isolated.

## 8.3 Legal and action gates

Separate Decisions govern:

- authenticity and legal sufficiency;
- subject match;
- disclosure;
- preservation;
- KYC refresh or address verification;
- monitoring or account action;
- suspicious-report assessment or filing;
- response sign-off.

Receipt of a request is not sufficient authority for every possible action.

---

# 9. Evidence and capture architecture

## 9.1 Observation normalization

All capture paths produce the same contract:

- API/event;
- scheduled file or database export;
- spreadsheet row;
- form or controlled value;
- photo, scan, audio, video;
- message or email;
- staff/vendor/customer/protected assertion;
- test or inspection.

## 9.2 Evidence request orchestration

```text
Evidence need
→ search authorized existing evidence
→ evaluate sufficiency and contradiction
→ identify unresolved facts
→ rank best source
→ choose approved channel
→ prefill known context
→ capture and validate
→ update Conclusion
→ stop, follow up, redirect, or escalate
```

## 9.3 Evidence invalidation

Expiry, source degradation, changed scope, supersession, contradiction, model extraction error, legal hold, or access change propagates to dependent Claims, Compliance State, Matters, Decisions, filings, and packages.

---

# 10. Legacy migration architecture

## 10.1 Import classification

Each workbook, sheet, table, or row is classified as one or more of:

- candidate Authority Source reference;
- candidate Requirement;
- Control Objective or Implementation;
- Observation or Evidence reference;
- Matter/finding/exception;
- Review Activity;
- KRI definition/value;
- scope/entity/vendor/asset record;
- communication/comment;
- unresolved legacy state.

## 10.2 Migration principles

- preserve original file, sheet, row, mapping, and import version;
- do not silently convert attachments into sufficient evidence;
- do not treat status text as authoritative lifecycle state;
- distinguish comments from assignment and Decision;
- detect duplicate and conflicting canonical objects;
- maintain unresolved queues;
- support parallel run and rollback;
- generate familiar projections after migration.

## 10.3 Legacy projection writes

If users edit through a familiar register view, writes must invoke governed domain commands. Direct projection-table mutation is prohibited.

---

# 11. APIs and domain commands

Representative commands:

- `CreateProgram`
- `AddRequirementToProgram`
- `ApproveRequirementInterpretation`
- `RecordApplicabilityConclusion`
- `LinkControlImplementation`
- `DefineEvidenceContract`
- `RecordObservation`
- `EvaluateClaim`
- `ScheduleReviewActivity`
- `CreateMatter`
- `ClassifyMatter`
- `RequestEvidence`
- `SubmitDecisionForApproval`
- `AuthorizeAction`
- `CreateResponsePackage`
- `ApproveAndTransmitResponse`
- `RecordAcknowledgement`
- `EvaluateVerification`
- `IssueAssuranceConclusion`
- `FreezeProgramSnapshot`

Representative events:

- `AuthoritySourceCaptured`
- `SourceProvisionReviewed`
- `RequirementApproved`
- `ApplicabilityChanged`
- `ControlImplementationChanged`
- `EvidenceContractChanged`
- `ObservationCaptured`
- `EvidenceSufficiencyChanged`
- `ContradictionDetected`
- `ComplianceStateChanged`
- `ProgramTriggerMatched`
- `MatterCreated`
- `MatterReopened`
- `DecisionApproved`
- `ActionImplemented`
- `VerificationFailed`
- `ResponsePackageTransmitted`
- `AuthorityAcknowledgementReceived`
- `FilingPackageFrozen`
- `AssuranceConclusionIssued`

---

# 12. Authorization architecture

Effective access is the intersection of:

- tenant;
- legal entity and licence;
- role and delegated authority;
- relationship to Program, Matter, control, or case;
- purpose;
- source/evidence classification;
- legal privilege and conflict;
- workflow state;
- time-limited authorization.

Authorization is enforced before and after retrieval, traversal, expansion, aggregation, export, and AI use.

Counts, titles, snippets, relationship paths, suggestions, timing, caches, embeddings, and package manifests must not leak protected records.

---

# 13. AI architecture

## 13.1 Operator capabilities

Initial capabilities:

- source classification and segmentation;
- Directive Atom and Requirement proposal;
- applicability support;
- control/evidence mapping;
- legacy register classification;
- entity reconciliation;
- evidence extraction and contradiction;
- focused request generation;
- Program and Matter summarization;
- action, test, and response drafting.

## 13.2 Controlled pipeline

```text
Trigger
→ authenticate actor/operator
→ resolve scope and purpose
→ authorize retrieval
→ retrieve source versions
→ model or deterministic analysis
→ structured validation
→ domain validation
→ authority and policy gates
→ human review where required
→ domain command
→ side-effect verification
→ immutable audit and monitoring
```

No free-form model text writes authoritative state.

## 13.3 Evaluation

Datasets must include:

- final, draft, amended, and ambiguous regulation;
- incomplete and contradictory sources;
- legacy spreadsheets;
- NDPA ROPA and DPIA cases;
- supervisory findings;
- protected authority requests;
- subject-match ambiguity;
- prompt injection and malicious documents;
- insufficient evidence and out-of-scope requests.

---

# 14. Scaling and deployment

Initial target:

- dedicated single-tenant cloud deployment;
- portable containerized modular core;
- relational database;
- object storage;
- durable queue/workflow;
- optional managed search;
- controlled model routes.

Scale independently only when measured need justifies splitting ingestion, document processing, media processing, search, projections, AI, or protected-case services.

Support additional deployment modes after the first production pattern is stable.

---

# 15. Architecture acceptance standard

The architecture is conformant only when:

- Programs remain current without duplicate registers;
- Matters represent change and exception without turning every Requirement into a case;
- one canonical record supports multiple authorized views;
- source and provision lineage remain exact;
- applicability and control implementation remain scoped and versioned;
- evidence is reused and contradictions propagate;
- NDPA Program journeys share one model;
- a regulatory amendment updates affected Programs and Matters safely;
- authority cases remain protected and purpose-bound;
- legacy views cannot bypass domain invariants;
- AI cannot bypass authority or structured commands;
- external task or response transmission cannot falsely close work;
- and point-in-time reconstruction spans source, Program, Matter, evidence, Decision, response, and outcome.