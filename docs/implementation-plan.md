# ClearSight Implementation Plan

This plan delivers ClearSight as a source-led, AI-assisted, continuously compliant, bank-grade operating system without recreating cumbersome register workflows behind a modern interface.

It conforms to:

- [`../README.md`](../README.md)
- [`product/continuous-compliance-operating-model.md`](product/continuous-compliance-operating-model.md)
- [`product/ease-of-use-standard.md`](product/ease-of-use-standard.md)
- [`product/operating-model.md`](product/operating-model.md)
- [`product/experience-principles.md`](product/experience-principles.md)
- [`../AGENTS.md`](../AGENTS.md)

Checkboxes indicate planned work, not completed implementation.

---

# 1. Delivery thesis

The first product must prove:

```text
Approved source or institutional event
→ Program update or Matter creation
→ bank context and inventory automatically assembled
→ only missing facts requested
→ grounded recommendation or first draft
→ authorized decision, action, or response
→ verification or acknowledgement
→ Program and reporting views updated
```

The implementation must not digitize every spreadsheet as a separate module or defer usability until after domain implementation.

---

# 2. Cross-cutting delivery principles

## 2.1 Programs and Matters are the vertical slices

Each milestone must improve either:

- continuing Program maintenance; or
- handling of a bounded Matter.

Avoid isolated data models, generic forms, dashboards, and AI demos.

## 2.2 Five-minute active-effort budget

Every key workflow must define:

- routine active-effort target;
- maximum major workspace transitions;
- fields expected to be prefilled;
- AI or deterministic assistance;
- save/resume behavior;
- safe fallback when source or AI is unavailable.

Routine work targets less than five minutes. Complex work must reach a safe saved next state within five minutes.

## 2.3 Source trust before manual entry

Source Registry, approved inventories, source authority, freshness, mapping, and data quality are early product capabilities.

Every proposed field must be evaluated for possible prefill from:

- institution profile;
- CMDB or architecture inventory;
- asset systems;
- branch and organization directory;
- HR and IAM;
- procurement and vendor systems;
- core customer and account systems;
- channel and merchant systems;
- ITSM and project systems;
- ROPA and BIA;
- policy and evidence repositories.

## 2.4 AI prepares first drafts

Where approved, AI should prepare structured drafts for extraction, mapping, summarization, requests, actions, verification, policy changes, and response indexes.

AI assistance must have a measurable human-effort benefit and safe deterministic fallback.

## 2.5 Review by exception

Interfaces and workflows should focus reviewers on changes, contradictions, low-confidence values, unsupported mappings, material effects, and high-impact actions.

## 2.6 Progressive integration

Support controlled lists, spreadsheets, managed imports, APIs, and events through the same Observation contract and user semantics.

## 2.7 Correct interaction form

Use:

- cards for small attention queues;
- tables for Requirements and populations;
- step flows for imports and capture;
- comparisons for contradictions and version change;
- timelines for history;
- paths for lineage and dependencies;
- charts for defined analytical questions.

## 2.8 Coherent modular core

Begin with authoritative relational data, explicit bounded contexts, durable workflows, outbox, versioned object storage, authorization-aware projections, and replaceable adapters.

No premature microservice or graph-engine requirement.

---

# 3. Initial product wedge

One pilot bank and legal entity should prove four connected journeys.

## A. Continuous NDPA Program

- import existing checklist and ROPA material;
- reconcile source provisions;
- use application, vendor, project, and organization inventories;
- define Requirements, applicability, controls, and Evidence Contracts;
- trigger targeted ROPA updates;
- create DPIA and breach Matters;
- prepare annual filing package;
- keep routine owner updates below five minutes.

## B. Regulatory Change Matter

- ingest an official CBN publication;
- classify source status;
- preserve exact provisions;
- extract candidate Requirements;
- propose applicability and control mappings;
- obtain human approval;
- create implementation Matters;
- update continuing Program state.

## C. Protected Authority Request Matter

- ingest and verify an authority request;
- review legal instrument and disclosure scope;
- resolve subjects and periods;
- prefill known KYC/account/address data from approved sources;
- route focused legal, KYC, records, AML, fraud, or branch tasks;
- prepare response package;
- approve, transmit, and record acknowledgement.

## D. Legacy finding or exception

- import from existing IT or vendor register;
- map canonical assets, vendors, controls, owners, and evidence;
- replace comment-driven routing with explicit assignment;
- verify remediation before closure;
- derive dashboard and workplan views.

---

# 4. Program-level usability targets

Initial targets:

- focused evidence request: median under 3 minutes; p90 under 5 minutes;
- routine approval with complete context: median under 2 minutes;
- repeat spreadsheet import using saved mapping: under 5 minutes active effort;
- assignment or redirection: under 60 seconds;
- executive comprehension: under 60 seconds;
- resume complex Matter: next action understood within 30 seconds;
- no routine flow above 3 major workspace transitions without documented justification;
- no repeated entry of source-available identity or scope data;
- accessibility completion time not materially worse than pointer-based completion.

Usability targets require representative bank-user testing, not internal opinion.

---

# Phase 0 — Product semantics, pilot, workflow budgets, and design foundation

## Objective

Establish Programs, Matters, source inventory, user-effort budgets, threat model, design language, and implementation decisions before feature development.

## Product and pilot

- [ ] Select pilot bank, legal entity, Programs, and Matter journeys.
- [ ] Identify personas, roles, authority, and delegated work.
- [ ] Inventory spreadsheets, repositories, databases, APIs, and human sources.
- [ ] Identify source owners, authoritative fields, freshness, and limitations.
- [ ] Define success metrics for human effort, compliance continuity, and trust.

## Workflow decomposition

For every golden journey:

- [ ] define user outcome;
- [ ] identify what bank sources already know;
- [ ] identify fields to prefill;
- [ ] identify unresolved facts requiring humans;
- [ ] identify AI first-draft opportunities;
- [ ] define routine active-effort target;
- [ ] define complex-work checkpoint;
- [ ] define save/resume and fallback;
- [ ] define accessibility and mobile requirements.

## Canonical semantics

- [ ] Program;
- [ ] Matter and Matter types;
- [ ] Scope;
- [ ] Authority Source, Requirement, Applicability;
- [ ] Control Objective and Implementation;
- [ ] Claim and Evidence Contract;
- [ ] Observation;
- [ ] Conclusion and Compliance State;
- [ ] Decision, Action, Response Package, Verification.

## Design foundation

- [ ] semantic tokens;
- [ ] typography and numeric styles;
- [ ] comfortable and compact density;
- [ ] light and dark parity;
- [ ] Today, Programs, Work, Explore, Configure shell;
- [ ] Respond and Capture shell;
- [ ] Program overview and Requirement table;
- [ ] Matter workspace;
- [ ] focused request;
- [ ] recommendation component;
- [ ] population worklist;
- [ ] spreadsheet mapper;
- [ ] source profile;
- [ ] contradiction view;
- [ ] save/resume state;
- [ ] accessibility baseline.

## Architecture decisions

ADRs for:

- modular core;
- backend and frontend stack;
- Program and Matter aggregates;
- temporal/versioning model;
- trigger and workflow runtime;
- Observation contract;
- Source Registry;
- authorization;
- evidence storage;
- search and graph projections;
- model gateway;
- protected case isolation;
- offline capture;
- initial deployment mode.

## Acceptance gate

Do not begin implementation until:

- workflow budgets are measurable;
- source reuse and prefill plans exist;
- representative low-fidelity flows pass initial usability review;
- Program and Matter semantics are distinct;
- protected workflows and authority are approved;
- first sources and evidence contracts are known.

---

# Phase 1 — Identity, scope, authority, temporal history, audit, and storage

## Objective

Build the trust foundation while avoiding repeated context entry.

- [ ] tenant, legal entity, jurisdiction, Program, service, branch, vendor, customer, account, and population scope;
- [ ] enterprise identity and directory boundary;
- [ ] role, relationship, purpose, and sensitivity authorization;
- [ ] authority and segregation of duties;
- [ ] delegated authority;
- [ ] context propagation across workflow;
- [ ] deliberate context switching;
- [ ] valid time, record time, versioning, supersession;
- [ ] immutable audit;
- [ ] secure versioned object storage;
- [ ] transactional outbox and durable jobs;
- [ ] save/resume primitives;
- [ ] cross-store isolation tests.

## Usability gate

- user context is resolved from identity and assignment;
- routine users do not repeatedly select institution or role;
- wrong-scope drafts and selections are prevented;
- interrupted workflows resume with next action visible;
- authorization failure explains the safe route without leaking protected details.

---

# Phase 2 — Source Registry, inventories, Observation contract, and progressive ingestion

## Objective

Make approved bank data available for prefill and workflow generation.

## Source Registry

- [ ] source owner, authority, limitations, scope, identifiers, freshness, health, mapping, and purpose;
- [ ] dependent Programs, claims, and conclusions;
- [ ] source-degradation events;
- [ ] user-facing source summary.

## Inventory adapters

Prioritize pilot sources:

- [ ] organization, branch, and owner directory;
- [ ] application or CMDB inventory;
- [ ] vendor and contract inventory;
- [ ] ROPA or processing inventory;
- [ ] policy and evidence repository;
- [ ] customer/account/KYC source for protected case pilot;
- [ ] ITSM or project/change source where useful.

## Spreadsheet and file ingestion

- [ ] secure upload;
- [ ] file/sheet selection;
- [ ] reusable mappings;
- [ ] schema-change detection;
- [ ] preview and validation;
- [ ] matching, duplicates, and contradiction;
- [ ] partial acceptance;
- [ ] row-level provenance;
- [ ] repeat-import exception review;
- [ ] rollback and supersession.

## Structured and media capture

- [ ] forms generated from unresolved facts;
- [ ] controlled values sourced from inventories;
- [ ] prefilled known fields;
- [ ] redirect/delegate/not-applicable;
- [ ] mobile capture;
- [ ] bounded media extraction;
- [ ] user confirmation;
- [ ] low-bandwidth and offline decision gate.

## AI assistance

- [ ] mapping suggestions;
- [ ] column detection;
- [ ] identifier normalization;
- [ ] document classification;
- [ ] structured extraction;
- [ ] contradiction suggestion;
- [ ] confidence and review-by-exception.

## Acceptance gate

- first import completes with clear initial mapping;
- repeat import requires under five minutes active effort;
- inventory values prefill downstream flows;
- source age and limitations remain visible;
- AI-assisted mapping reduces effort and has safe fallback;
- unresolved records enter a focused queue.

---

# Phase 3 — Program engine and continuous compliance state

## Objective

Create continuing Programs rather than static checklist modules.

- [ ] Program aggregate and templates;
- [ ] Requirements and source provisions;
- [ ] applicability by scope;
- [ ] Control Objectives and scoped Implementations;
- [ ] Evidence Contracts;
- [ ] review, filing, certification, and testing schedule;
- [ ] trigger evaluation;
- [ ] multi-dimensional Compliance State;
- [ ] Program overview;
- [ ] Requirement table and saved exception views;
- [ ] Program change summary;
- [ ] Program-to-Matter creation;
- [ ] assurance and filing history.

## AI assistance

- [ ] propose Requirement normalization;
- [ ] suggest applicability questions;
- [ ] propose control and evidence mappings;
- [ ] summarize Program changes;
- [ ] recommend missing owners or evidence;
- [ ] draft filing or review index.

## Acceptance gate

- Program state derives from governed data, not manually edited RAG status;
- users can move directly from gap to evidence, owner, or Matter;
- common Program review uses exception-focused views;
- routine Requirement review stays within five-minute budget;
- Program remains usable when AI is unavailable.

---

# Phase 4 — Matter engine, focused requests, decision, action, and verification

## Objective

Handle changes and exceptions in one coherent workspace.

- [ ] typed Matter aggregate;
- [ ] source and trigger;
- [ ] relevant section composition by Matter type;
- [ ] scope and affected-object resolution;
- [ ] evidence needs;
- [ ] focused request orchestration;
- [ ] owner, redirect, delegate, conflict, and escalation;
- [ ] decision and authority;
- [ ] action plans and external execution;
- [ ] response package;
- [ ] verification and acknowledgement;
- [ ] closure contracts;
- [ ] history and point-in-time reconstruction.

## Ease-of-use

- [ ] AI-generated first summary;
- [ ] prefilled scope and affected objects;
- [ ] recommended next action;
- [ ] save and resume;
- [ ] changed-since-last-view summary;
- [ ] next unresolved item;
- [ ] background processing;
- [ ] grouped notifications.

## Acceptance gate

- routine Matter assignment and evidence request meet time budget;
- complex Matter reaches safe next state within five minutes;
- no module hopping is required;
- action completion cannot close Matter;
- source or AI outage has manual fallback.

---

# Phase 5 — NDPA vertical slice

## Objective

Prove continuous Program operation using real privacy workflows.

- [ ] import and reconcile NDPA checklist;
- [ ] source provisions and Requirements;
- [ ] institution classification and scope;
- [ ] DPO governance;
- [ ] ROPA inventory and targeted updates;
- [ ] DPIA screening and full DPIA Matter;
- [ ] vendor/processor review;
- [ ] breach Matter and notification timing;
- [ ] rights requests where in scope;
- [ ] evidence refresh and stale-state handling;
- [ ] annual compliance filing package;
- [ ] assurance review.

## Ease-of-use gates

- owner ROPA update targets under five minutes;
- known application/vendor/department data is prefilled;
- DPIA screening uses existing project, vendor, and processing context;
- annual filing package is assembled throughout the year;
- users review exceptions rather than reconstruct the Program.

---

# Phase 6 — Regulatory change and authority work

## Regulatory Change Matter

- [ ] official source intake and authenticity;
- [ ] document status classification;
- [ ] provision segmentation;
- [ ] AI candidate Requirement extraction;
- [ ] source-linked review;
- [ ] applicability approval;
- [ ] impact on Programs, controls, systems, vendors, and owners;
- [ ] implementation Matter creation;
- [ ] continuing Program update;
- [ ] amendment and supersession.

## Authority Request Matter

- [ ] protected source intake;
- [ ] legal instrument and disclosure review;
- [ ] subject and period resolution;
- [ ] approved customer/account/KYC/address prefill;
- [ ] focused case directives;
- [ ] legal, KYC, records, AML, fraud, branch, or technology tasks;
- [ ] response-package reconciliation;
- [ ] approval, transmission, acknowledgement;
- [ ] retention and legal hold;
- [ ] minimized KRI and systemic signals.

## Acceptance gate

- AI provides grounded first draft without making final legal decision;
- reviewers inspect source-linked exceptions;
- routine source review and routing meet time budgets;
- protected data does not leak;
- request does not imply guilt or automatic reportability.

---

# Phase 7 — Legacy workflow migration and derived views

- [ ] compliance-register import;
- [ ] risk and exception-register import;
- [ ] workplan migration to Review Activities;
- [ ] RCSA migration;
- [ ] KRI definitions and derived indicators;
- [ ] BIA context import;
- [ ] vendor and loss-register migration;
- [ ] historical comments and evidence preservation;
- [ ] reconciliation queues;
- [ ] register-compatible exports;
- [ ] role dashboards derived from canonical data.

## Gate

- original row provenance preserved;
- no duplicate truth system;
- users can maintain future state through Programs and Matters rather than spreadsheets;
- derived KRI/dashboard values drill into underlying records;
- repeat work is measurably reduced.

---

# Phase 8 — Governed AI, advanced materiality, and automation

- [ ] model gateway and registry;
- [ ] operator/capability definitions;
- [ ] structured output and policy pipeline;
- [ ] authorization-aware retrieval;
- [ ] tool allowlists;
- [ ] evaluation harness;
- [ ] prompt-injection defense;
- [ ] cost and latency budgets;
- [ ] recommendation-quality and effort-reduction metrics;
- [ ] policy-controlled low-impact automation;
- [ ] kill switch and degraded mode.

AI capability is accepted only when it reduces active effort without reducing correctness, evidence, authority, or safety.

---

# Phase 9 — Enterprise integrations, scale, assurance, and GA

- priority bank connectors;
- multi-entity and jurisdiction scope;
- examination and evidence rooms;
- board and regulatory packages;
- performance and workload models;
- backup, recovery, and regional resilience;
- independent security review;
- privacy, residency, and deployment modes;
- migration tooling;
- operational runbooks and SLOs;
- representative usability validation across roles and bank sizes.

General availability requires both bank-grade control and demonstrated effort reduction.

---

# Definition of done

A milestone is complete only when:

- Program or Matter behavior works end to end;
- source authority and provenance are enforced;
- approved inventories and integrations are reused;
- routine flows meet five-minute budgets or have approved justification;
- complex flows reach safe saved next state quickly;
- AI first drafts are grounded and measurably useful;
- accessibility and low-bandwidth journeys work;
- authority and privacy remain intact;
- evidence and verification are complete;
- failure and degraded modes work;
- point-in-time reconstruction is possible;
- documentation and acceptance tests are synchronized.