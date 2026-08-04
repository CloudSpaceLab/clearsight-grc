# ClearSight Implementation Plan

This plan turns the ClearSight continuous-compliance model into a bank-grade product without recreating every legacy register, requiring perfect integrations, or exposing internal architecture as the interface.

It conforms to:

- [`../README.md`](../README.md)
- [`product/continuous-compliance-operating-model.md`](product/continuous-compliance-operating-model.md)
- [`product/operating-model.md`](product/operating-model.md)
- [`product/regulatory-and-enforcement-intelligence.md`](product/regulatory-and-enforcement-intelligence.md)
- [`product/experience-principles.md`](product/experience-principles.md)
- [`../AGENTS.md`](../AGENTS.md)

Checkboxes indicate planned work, not completed implementation.

---

# 1. Delivery thesis

The first product must prove three connected paths:

## Continuing compliance

```text
Authority Sources
→ approved Requirements and applicability
→ scoped controls and Evidence Contracts
→ current Observations
→ Compliance State
→ targeted refresh, review, exception, or filing
```

## Matter handling

```text
Trigger or external communication
→ classified Matter
→ evidence and decision or response
→ Actions
→ verification or acknowledgement
→ Program and institutional state updated
```

## Legacy workflow migration

```text
Existing spreadsheet or document
→ source-preserving import
→ mapping and reconciliation
→ canonical objects
→ familiar register/workplan/KRI view
→ duplicate manual records retired
```

The first release must not attempt every regulatory jurisdiction, every GRC domain, an institution-wide ontology, all deployment modes, or autonomous legal and risk judgment.

---

# 2. Recommended first product wedge

The strongest initial pilot is compliance-led and should contain three vertical slices.

## Slice A — NDPA continuous-compliance Program

Prove:

- authoritative source and Requirement lineage;
- applicability by entity, processing activity, system, vendor, and project;
- ROPA worklist;
- DPIA screening and go-live gate;
- vendor/processor evidence;
- breach Matter and timing;
- annual filing readiness and package;
- targeted evidence refresh;
- independent review and history.

## Slice B — External Authority Workbench

Prove:

- one recent CBN publication from source to approved Requirements, control changes, implementation Matters, and Evidence Contracts;
- one protected authority-request scenario from intake and legal review to subject resolution, KYC/address/records tasks, response package, and acknowledgement;
- clear human authority boundaries.

## Slice C — Legacy register to verified Matter

Prove:

- import of a compliance register and an IT risk or vendor exception register;
- source reconciliation;
- structured ownership and action;
- evidence review;
- implementation-versus-verification separation;
- derived register, work queue, and executive view.

ATM/POS population workflows remain an important second domain and may be included where pilot data permits, but they should not delay the compliance wedge.

---

# 3. Delivery principles

## 3.1 Programs maintain; Matters mobilize

Each milestone must improve either:

- a continuing Program’s ability to remain current; or
- a Matter’s ability to reach a defensible outcome.

Avoid milestones that produce only schemas, generic forms, dashboards, or AI demonstrations.

## 3.2 Source trust before automated interpretation

Source Registry, Authority Source, exact Source Provision, Observation provenance, freshness, mapping, and data-quality state are early product capabilities.

## 3.3 Progressive integration

Support forms, controlled values, photos, documents, spreadsheets, managed imports, APIs, and events through one Observation contract.

## 3.4 Correct interaction form

Use:

- cards for small attention queues;
- tables for Requirement, ROPA, control, case, account, asset, vendor, and exception populations;
- step flows for imports, DPIA, filings, and responses;
- split views for regulatory interpretation;
- comparison views for contradiction;
- timelines for Matters and history;
- calendars for recurring Program work;
- charts only for defined questions.

## 3.5 Begin as a coherent modular core

Start with a modular monolith or similarly disciplined core using authoritative relational storage, versioned object storage, durable workflows and outbox, rebuildable projections, and replaceable adapters.

## 3.6 AI compiles; policy and humans decide

AI may extract, normalize, map, compare, summarize, and draft. Domain services and authorized humans determine source status, applicability, evidence minimums, reportability, disclosure, authority, and closure.

## 3.7 Protected work uses stronger boundaries

Protected reporting, Authority Request Cases, suspicious-reporting work, legal privilege, and protected identity must not share ordinary search, analytics, export, or AI routes without explicit minimized interfaces.

---

# 4. Recommended repository topology

```text
.
├── README.md
├── AGENTS.md
├── docs/
│   ├── product/
│   ├── architecture/
│   ├── quality/
│   ├── reviews/
│   └── decisions/
├── apps/
│   ├── web/
│   ├── external-portal/
│   └── capture-pwa/
├── services/
│   └── core/
├── packages/
│   ├── design-system/
│   ├── domain-contracts/
│   ├── event-contracts/
│   ├── authorization/
│   ├── model-gateway/
│   └── integration-sdk/
├── workers/
│   ├── ingestion/
│   ├── document-processing/
│   ├── media-processing/
│   └── projections/
├── tests/
│   ├── e2e/
│   ├── evaluations/
│   ├── security/
│   ├── visual/
│   └── performance/
└── infrastructure/
```

The first implementation may remain one deployable unit while preserving boundaries in code.

---

# Phase 0 — Product semantics, pilot, source inventory, and design foundation

## Objective

Establish the Program and Matter model, pilot scope, source estate, user journeys, security boundaries, and initial design before implementation.

## 0.1 Pilot definition

- [ ] Select bank, legal entity, licences, and participating functions.
- [ ] Confirm NDPA Program scope.
- [ ] Select one recent CBN publication.
- [ ] Define a legally safe synthetic or sanitized authority-request scenario.
- [ ] Select legacy compliance and finding/exception workbooks.
- [ ] Identify Program owners, DPO, compliance, legal, AML, risk, audit, business, technology, and records personas.
- [ ] Define measurable baseline effort and target outcomes.

## 0.2 Canonical language

Define and approve:

- [ ] Program;
- [ ] Matter and Matter types;
- [ ] Scope;
- [ ] Authority Source and Source Provision;
- [ ] Requirement and Applicability Conclusion;
- [ ] Control Objective and Control Implementation;
- [ ] Policy;
- [ ] Observation;
- [ ] Claim and Evidence Contract;
- [ ] Compliance State;
- [ ] Review Activity and KRI;
- [ ] Decision, Action, Verification Contract, and Response Package.

## 0.3 Legacy workflow mapping

- [ ] Map compliance-register columns to canonical objects.
- [ ] Map IT risk and exception workbooks.
- [ ] Map workplans to Review Activities.
- [ ] Map KRI spreadsheets to indicator definitions and underlying populations.
- [ ] Map BIA data to services, applications, dependencies, RTO, and RPO.
- [ ] Map vendor findings and evidence.
- [ ] Define import confidence and unresolved queues.

## 0.4 Architecture decisions

Create ADRs for:

- [ ] modular core and split criteria;
- [ ] backend/frontend stack;
- [ ] Program and Matter aggregate boundaries;
- [ ] relational authoritative and temporal model;
- [ ] workflow runtime;
- [ ] outbox and durable jobs;
- [ ] object storage and evidence integrity;
- [ ] authorization and inference resistance;
- [ ] Source Registry and Authority Source authenticity;
- [ ] document segmentation and Directive Atom schema;
- [ ] search and graph projections;
- [ ] model gateway;
- [ ] protected case isolation;
- [ ] offline capture boundary;
- [ ] initial deployment mode.

## 0.5 Threat and privacy model

Cover:

- [ ] cross-tenant and cross-entity access;
- [ ] wrong-scope filing or response;
- [ ] source spoofing and document replacement;
- [ ] evidence tampering;
- [ ] malicious spreadsheets and documents;
- [ ] prompt injection;
- [ ] export and response leakage;
- [ ] subject-match error;
- [ ] protected identity and suspicious-reporting leakage;
- [ ] insider misuse;
- [ ] integration compromise;
- [ ] search, graph, count, cache, and timing inference;
- [ ] offline evidence risk.

## 0.6 Design foundation

Prototype:

- [ ] Today;
- [ ] Program overview and Requirement table;
- [ ] Work queue and generic Matter;
- [ ] NDPA ROPA and DPIA;
- [ ] regulatory split review;
- [ ] protected Authority Request Case;
- [ ] spreadsheet import;
- [ ] evidence request;
- [ ] Response Package;
- [ ] compliance-state dimensions;
- [ ] light/dark and comfortable/compact density;
- [ ] accessibility baseline.

## Phase 0 acceptance gate

Do not begin domain implementation until pilot journeys, sources, authority, Program/Matter semantics, protected boundaries, and designs are testable and approved.

---

# Phase 1 — Identity, scope, authorization, temporal history, audit, and storage

## Objective

Build the security and historical foundation required by every Program and Matter.

## Work

- [ ] tenant, institution, legal entity, licence, jurisdiction, Program, service, branch, project, vendor, customer/account, and population scope;
- [ ] OIDC/SAML, MFA hooks, SCIM boundary, service identity, delegation, break glass;
- [ ] deny-by-default RBAC/ABAC/ReBAC/purpose/sensitivity policy;
- [ ] authority matrix, segregation of duties, conflict checks;
- [ ] field, source, evidence, case, search, count, export, worker, AI, and bulk authorization;
- [ ] valid time, record time, versions, supersession, concurrency;
- [ ] immutable audit and protected access events;
- [ ] transactional outbox, durable jobs, retry/replay/cancellation;
- [ ] versioned object storage, hash manifest, scanning, retention, legal hold, deletion, resumable upload;
- [ ] operational observability without restricted data leakage.

## Phase 1 acceptance gate

Pass cross-tenant, wrong-entity, wrong-purpose, protected-case, bulk, search/count inference, temporal reconstruction, idempotency, and evidence-integrity tests.

---

# Phase 2 — Source Registry, Authority Sources, Observation contract, and ingestion

## Objective

Make source authority and provenance reliable before building Program conclusions.

## 2.1 Source Registry

- [ ] source owner, custodian, type, purpose, scope, identifiers;
- [ ] authoritative facts and explicit limitations;
- [ ] freshness, health, collection state, mapping version;
- [ ] known data-quality issues and unresolved mappings;
- [ ] dependent Programs, Claims, Conclusions, and Matters.

## 2.2 Authority Source

- [ ] source types and authenticity states;
- [ ] issuing authority, jurisdiction, reference, dates, deadlines;
- [ ] original artifact and hash;
- [ ] confidentiality, privilege, retention, legal hold;
- [ ] amendment, supersession, and related-source graph;
- [ ] provision segmentation and coordinates;
- [ ] human verification workflow.

## 2.3 Observation contract

- [ ] subject, property, value, units;
- [ ] source and capture method;
- [ ] scope, population, effective and capture time;
- [ ] original reference and transformation history;
- [ ] authority, limitations, sensitivity, review, confirmation, and version.

## 2.4 Spreadsheet and document ingestion

- [ ] upload, sheet selection, column mapping, preview, validation;
- [ ] malicious formula, macro, hidden content, and active-content policy;
- [ ] partial acceptance and row provenance;
- [ ] identifier normalization and reconciliation;
- [ ] document segmentation and extraction;
- [ ] rollback and supersession.

## 2.5 Structured and media capture

- [ ] focused forms and controlled values;
- [ ] redirect, partial, not-applicable, sensitivity states;
- [ ] photo/scan/audio capture, quality guidance, original preservation, extraction, correction, confirmation;
- [ ] low-bandwidth and offline decision gate.

## 2.6 Managed sources

- [ ] scheduled files, SFTP, database exports;
- [ ] generic API and event connector boundaries;
- [ ] cursor, version, idempotency, deletion, revocation, and health.

## Phase 2 acceptance gate

Demonstrate official-source intake, legacy spreadsheet partial import, source profile, document provision lineage, one managed source, one human capture, and visible source degradation.

---

# Phase 3 — Program engine

## Objective

Build stable continuing compliance before broad Matter automation.

## 3.1 Program aggregate

- [ ] identity, purpose, version, state;
- [ ] governing sources and scope;
- [ ] owners, reviewers, committees, authority;
- [ ] calendar and trigger subscriptions;
- [ ] linked Matters, exceptions, assurance, filings, and history.

## 3.2 Requirements and applicability

- [ ] candidate, interpreted, approved, effective, amended, superseded, withdrawn states;
- [ ] exact provision lineage;
- [ ] applicability by entity, licence, jurisdiction, product, channel, activity, system, vendor, population, threshold, and period;
- [ ] rationale, evidence, assumptions, reviewer, and authority;
- [ ] bulk review with object-level authorization.

## 3.3 Controls and policies

- [ ] Control Objective and scoped Control Implementation;
- [ ] policy lifecycle;
- [ ] owner, performer, reviewer, frequency, automation, dependencies;
- [ ] Requirement mapping;
- [ ] design and operating-effectiveness Conclusions;
- [ ] exceptions and Matter linkage.

## 3.4 Evidence Contracts and Compliance State

- [ ] Claim and Evidence Contract;
- [ ] evidence search and reuse;
- [ ] sufficiency dimensions;
- [ ] contradiction;
- [ ] refresh schedule and trigger;
- [ ] multidimensional Compliance State;
- [ ] concise state with explainable basis.

## 3.5 Review Activity, calendar, KRI, filing, and assurance

- [ ] recurring and event-driven Review Activities;
- [ ] Program calendar;
- [ ] KRI definitions and values derived from canonical records;
- [ ] filing/certification requirements and readiness;
- [ ] first-, second-, and third-line independent Conclusions;
- [ ] point-in-time package and sign-off.

## Phase 3 acceptance gate

A Program must show source-linked Requirements, applicability, controls, evidence state, calendar, Matters, assurance, and history without duplicating records into separate modules.

---

# Phase 4 — Matter, decision, action, response, and verification engine

## Objective

Provide one governed mechanism for changes, gaps, findings, incidents, exceptions, and authority work.

## 4.1 Matter aggregate

- [ ] typed Matter lifecycle;
- [ ] source, scope, period, materiality, affected objects;
- [ ] owner, authority, deadlines, escalation;
- [ ] evidence, communications, Decisions, Actions, Response Packages, Verification, and history;
- [ ] create/update/merge/split/reopen/supersede behavior.

## 4.2 Decision and approval

- [ ] options and trade-offs;
- [ ] authority and segregation of duties;
- [ ] challenge, dissent, conditional approval, reject, request evidence;
- [ ] emergency authority and later review;
- [ ] expiry and invalidation.

## 4.3 Action and external execution

- [ ] action plans and dependencies;
- [ ] ClearSight and external task adapters;
- [ ] idempotent writes and reconciliation;
- [ ] implementation evidence;
- [ ] partial failure and compensation.

## 4.4 Verification

- [ ] outcome, baseline, population, source, threshold, period, acceptance, failure response;
- [ ] implemented versus awaiting verification;
- [ ] successful, ineffective, and indeterminate outcomes;
- [ ] reopening and risk/Program update.

## 4.5 Response Package

- [ ] purpose, recipient, scope, directive coverage;
- [ ] included/excluded evidence and redaction;
- [ ] preparer, reviewer, approver, signatory;
- [ ] transmission, acknowledgement, retention, manifest;
- [ ] response completion distinct from accepted closure.

## Phase 4 acceptance gate

Pass a legacy finding from import through assignment, evidence, action, failed and successful verification, and historical reconstruction.

---

# Phase 5 — NDPA continuous-compliance vertical slice

## Objective

Prove that ClearSight can keep a complex regulatory Program current with less manual reconstruction.

## 5.1 Source and Requirement foundation

- [ ] approved NDPA/NDPC source versions and exact provisions;
- [ ] imported checklist reconciled as secondary working records;
- [ ] Requirements, applicability, controls, Evidence Contracts, owners, and calendar.

## 5.2 ROPA

- [ ] processing-activity population;
- [ ] applications, processes, purpose, lawful basis, data categories, subjects, recipients, systems, vendors, location, transfer, retention, owner;
- [ ] prefill from approved sources;
- [ ] focused owner confirmation;
- [ ] stale and changed-activity triggers;
- [ ] completeness and evidence state.

## 5.3 DPIA

- [ ] project, process-change, vendor, AI/model, sensitive-data, and cross-border triggers;
- [ ] screening with prefilled context;
- [ ] DPO decision and rationale;
- [ ] full DPIA, risks, remediation, approval, go-live gate;
- [ ] post-deployment verification.

## 5.4 Vendors, rights, consent, retention, and transfers

- [ ] processor inventory and DPA evidence;
- [ ] due diligence and recurring review;
- [ ] data-subject request workflows;
- [ ] consent and notice evidence;
- [ ] retention/deletion Claims;
- [ ] cross-border transfer register and legal basis.

## 5.5 Breach Matter

- [ ] detection and awareness times;
- [ ] affected data, systems, subjects, and impact;
- [ ] reportability and notification Decisions;
- [ ] deadlines, communications, evidence, remediation, verification.

## 5.6 Annual filing

- [ ] filing Requirement and deadline;
- [ ] evidence readiness by dimension;
- [ ] unresolved exceptions and included/excluded records;
- [ ] reviewers, signatory, submission, acknowledgement;
- [ ] point-in-time package.

## Phase 5 acceptance gate

Demonstrate that ROPA, DPIA, vendor, breach, and filing views derive from one Program and targeted triggers rather than separate workbooks and annual blanket campaigns.

---

# Phase 6 — Regulatory change and External Authority Workbench

## Objective

Turn authority communications into correct, source-linked Programs and Matters.

## 6.1 Authority Inbox

- [ ] monitored or uploaded sources;
- [ ] authenticity and work-class triage;
- [ ] final/draft/amendment/guidance/supervisory/enforcement classification;
- [ ] deadline and confidentiality;
- [ ] assignment and duplicate detection.

## 6.2 Regulatory Change Compiler

- [ ] provision segmentation;
- [ ] Directive Atom extraction;
- [ ] modality, actor, action, object, condition, threshold, frequency, deadline, exception;
- [ ] source-linked review;
- [ ] applicability proposal and approval;
- [ ] existing-control reconciliation;
- [ ] Program update and implementation Matters;
- [ ] Evidence Contracts, tests, and readiness;
- [ ] amendment and supersession propagation.

## 6.3 Supervisory Matters

- [ ] finding and institution-specific expectation;
- [ ] management response;
- [ ] commitments and milestones;
- [ ] evidence and committee oversight;
- [ ] response package and effectiveness verification.

## 6.4 Authority Request Cases

- [ ] protected intake and legal-instrument review;
- [ ] subject/account/transaction resolution with ambiguity handling;
- [ ] directives and requested periods;
- [ ] legal hold and disclosure authority;
- [ ] KYC/address/records/AML/fraud/branch/technology/legal tasks;
- [ ] reportability and high-impact Decisions;
- [ ] Response Package, transmission, acknowledgement, retention;
- [ ] minimized KRI and systemic-signal output.

## Phase 6 acceptance gate

Pass one CBN publication and one protected authority request through complete source, decision, evidence, action/response, acknowledgement, and history paths without unauthorized automation.

---

# Phase 7 — Product experience implementation

## Objective

Deliver the calm task-oriented interface defined in the experience principles.

## Work

- [ ] design system, semantic tokens, density, accessibility, light/dark;
- [ ] Today;
- [ ] Programs overview, Requirement/control/evidence tables, calendar, assurance, history;
- [ ] Work queue and Matter workspace;
- [ ] ROPA, DPIA, filing, regulatory split review, protected authority case;
- [ ] capture/respond surfaces;
- [ ] source profile, spreadsheet mapper, reconciliation, contradiction, population worklists;
- [ ] decision, response, and verification;
- [ ] command surface as secondary interaction;
- [ ] localization, low bandwidth, degraded state, meeting mode.

## Phase 7 acceptance gate

Representative compliance, business, DPO, legal, executive, audit, branch, and evidence users must complete core journeys without navigating architecture-oriented modules.

---

# Phase 8 — Governed AI, materiality, and intelligent assistance

## Objective

Add evaluated intelligence only after source, Program, Matter, evidence, and authority foundations exist.

## Work

- [ ] model gateway, routing, residency, cost/latency, fallback, kill switch;
- [ ] operator registry and versioned structured outputs;
- [ ] authorization-aware retrieval;
- [ ] source and provision extraction;
- [ ] Requirement and applicability proposal;
- [ ] mapping and reconciliation;
- [ ] focused evidence-question generation;
- [ ] Matter grouping and materiality explanation;
- [ ] executive and filing summaries;
- [ ] action and response drafting;
- [ ] evaluation datasets from sanitized bank workflows;
- [ ] prompt injection, leakage, abstention, malformed output, provider failure, regression, and outcome monitoring.

## Phase 8 acceptance gate

No capability reaches production without exact source lineage, structured validation, zero critical authorization leakage in tests, appropriate abstention, human authority gates, monitoring, and rollback.

---

# Phase 9 — Enterprise integrations and legacy migration

## Objective

Scale from pilot sources while preserving one canonical model.

## Priority connectors

- [ ] document and regulatory-source repositories;
- [ ] IAM and HR;
- [ ] ITSM/change/project systems;
- [ ] CMDB, asset, and enterprise architecture;
- [ ] vendor/procurement;
- [ ] complaints/CRM;
- [ ] incident and loss systems;
- [ ] core customer/KYC and records platforms for protected authority workflows;
- [ ] security and service telemetry;
- [ ] data warehouse and reporting.

## Legacy migration

- [ ] source inventory and workbook classification;
- [ ] mapping templates;
- [ ] duplicate Requirement, control, vendor, asset, Matter, and evidence detection;
- [ ] unresolved mapping queues;
- [ ] provenance and import version;
- [ ] parallel run and reconciliation;
- [ ] familiar register/workplan/KRI exports;
- [ ] rollback and archive strategy.

## Phase 9 acceptance gate

A legacy register can be retired without loss of source lineage, familiar reporting, history, ownership, or control evidence.

---

# Phase 10 — Assurance, scale, security review, and general availability

## Objective

Validate enterprise reliability, performance, privacy, assurance, and operations.

## Work

- [ ] control testing, samples, independence, findings, audit planning, examiner evidence rooms;
- [ ] board and regulatory packages;
- [ ] load tests for Requirements, Programs, populations, imports, search, Matters, evidence, and exports;
- [ ] backup, restore, workflow recovery, replay, source/model outage, projection rebuild, regional failure;
- [ ] penetration, tenant isolation, protected-case, supply-chain, key rotation, incident-response review;
- [ ] retention, deletion, legal hold, data residency, provider governance;
- [ ] dedicated-tenant production pattern, then other deployment modes based on demand;
- [ ] onboarding, support, runbooks, release, rollback, status, and product analytics.

## General-availability gate

Require:

- all selected Program and Matter golden journeys passing;
- no unresolved critical isolation or protected-data defect;
- tested backup and recovery;
- approved SLOs and runbooks;
- independent security review;
- pilot evidence of reduced effort and stronger evidence;
- and no material product-invariant regression.

---

# 5. Program-level success measures

## Continuous compliance

- Requirements with approved source and applicability;
- evidence current and sufficient by materiality;
- time spent preparing filings, certifications, and audits;
- targeted refresh versus blanket questionnaire volume;
- exceptions and overdue obligations;
- assurance conclusion reversals.

## Regulatory change

- publication to triage, approved interpretation, applicability, implementation plan, and evidenced control;
- missed or duplicate Requirements;
- reviewer edit and rejection rate;
- source-lineage completeness.

## Authority cases

- receipt to legal triage and response;
- directives fully reconciled to evidence;
- ambiguous subject matches;
- unauthorized action or disclosure attempts;
- response acknowledgement and retention completeness.

## Legacy workflow reduction

- spreadsheets retired;
- duplicate records and requests avoided;
- comment/email rerouting replaced by structured assignment;
- manual dashboard and committee preparation eliminated;
- canonical records reused across Program, Matter, KRI, assurance, and reporting views.

## Decision and verification

- time to accountable decision;
- actions awaiting verification;
- failed or indeterminate verification;
- reopened Matters;
- projected versus observed outcomes.

## Trust

- source freshness and health;
- unresolved mappings;
- evidence-chain integrity;
- point-in-time reconstruction time;
- unauthorized or out-of-scope AI and user actions;
- protected-data leakage tests.

---

# 6. Completion standard

A milestone is complete only when:

- real Program or Matter behavior works through application boundaries;
- source and evidence lineage are complete;
- authority and privacy are enforced;
- data-quality limitations remain visible;
- human effort is measurably reduced;
- failure and degraded modes work;
- visual and accessibility standards pass;
- and the resulting decision, response, filing, or remediation can be reconstructed and verified.

The product’s defining test is:

> Can ClearSight maintain a continuing compliance position, detect what changed, ask only for missing proof, route the correct authorized work, produce a defensible response or implementation, and preserve the outcome without forcing users to rebuild or reconcile multiple registers?

Until the answer is yes, the core product is not complete.