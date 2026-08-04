# ClearSight Bank Operating Model

This document defines the canonical product semantics beneath ClearSight.

It describes the smallest coherent set of objects through which ClearSight continuously maintains compliance, handles exceptional work, governs evidence and decisions, and remains adaptable across regional, national, and multinational banks.

It must be read together with [`continuous-compliance-operating-model.md`](continuous-compliance-operating-model.md). The continuous-compliance document explains how these objects replace legacy registers; this document defines what the objects mean and how they relate.

---

# 1. Product objective

ClearSight should help each stakeholder:

1. know what requirements and risks apply to their scope;
2. understand how the institution intends to satisfy them;
3. see what evidence currently supports that position;
4. identify what changed, failed, expired, or became uncertain;
5. provide only missing information;
6. make or route the correct authorized decision or response;
7. verify the defined outcome;
8. preserve complete institutional history.

Users operate primarily through **Programs** and **Matters**. Shared primitives such as Scope, Requirement, Control, Observation, Claim, Evidence Contract, Decision, and Verification support both.

---

# 2. Canonical operating loops

## 2.1 Continuing Program loop

```text
Authority or institutional objective
→ Requirements and applicability
→ scoped controls and owners
→ Evidence Contracts and source coverage
→ continuous observations and scheduled reviews
→ current compliance and assurance state
→ targeted refresh, exception, or Matter when needed
→ filing, certification, or management reporting
→ historical reconstruction
```

## 2.2 Matter loop

```text
Trigger or source
→ classify Matter
→ resolve scope and affected Programs
→ establish required Claims or directives
→ find or request missing evidence
→ conclude, decide, or prepare response
→ execute actions
→ verify outcome or obtain acknowledgement
→ update Programs, risk state, and institutional memory
```

---

# 3. User-facing aggregates

## 3.1 Program

A Program is a stable, long-lived governance aggregate for a continuing body of requirements, controls, evidence obligations, reviews, filings, exceptions, and assurance.

Examples:

- NDPA and NDPC compliance;
- AML/CFT;
- CBN cybersecurity and technology risk;
- PCI DSS;
- ISO 27001 and ISO 22301;
- operational resilience;
- third-party assurance;
- RCSA;
- policy lifecycle;
- regulatory returns;
- annual IT risk and control reviews.

A Program includes:

- Program identity, purpose, version, and lifecycle state;
- governing Authority Sources, standards, internal policies, or objectives;
- Requirements and Applicability Conclusions;
- scope and covered populations;
- Control Objectives and Control Implementations;
- Evidence Contracts and source dependencies;
- owners, performers, reviewers, committees, and authority;
- calendar obligations and trigger subscriptions;
- current Compliance State;
- exceptions, waivers, and linked Matters;
- assurance activities and conclusions;
- filings, certifications, response packages, and history.

Suggested Program states:

```text
DRAFT
→ UNDER_REVIEW
→ ACTIVE
→ ACTIVE_WITH_GAPS
→ SUSPENDED
→ SUPERSEDED
→ RETIRED
```

A Program state is not its compliance conclusion. An active Program can contain current, at-risk, unknown, excepted, or non-compliant Requirements.

## 3.2 Matter

A Matter is a bounded aggregate for work created by change, exception, uncertainty, harm, request, or required judgment.

Matter types include:

- regulatory change;
- supervisory finding;
- authority or enforcement request;
- risk situation;
- control gap;
- audit finding;
- exception or waiver;
- incident;
- operational loss;
- data breach;
- vendor deficiency;
- customer or conduct concern;
- overdue obligation;
- failed verification;
- evidence contradiction;
- KRI threshold breach.

A Matter includes:

- type, source, scope, effective period, and materiality;
- linked Programs, Requirements, controls, policies, services, customers, accounts, assets, vendors, projects, incidents, or losses;
- what changed and why it matters;
- source Observations and required Claims or directives;
- evidence and current Conclusion;
- owner, authority, deadline, and escalation;
- Decisions, Actions, communications, and Response Packages;
- Verification Contract, acknowledgement, or closure criteria;
- history and point-in-time state.

Generic Matter states:

```text
RECEIVED_OR_DETECTED
→ TRIAGE
→ UNDER_ASSESSMENT
→ AWAITING_EVIDENCE
→ AWAITING_DECISION_OR_RESPONSE
→ AUTHORIZED
→ IN_PROGRESS
→ IMPLEMENTED_OR_TRANSMITTED
→ AWAITING_VERIFICATION_OR_ACKNOWLEDGEMENT
→ VERIFIED_OR_ACKNOWLEDGED
→ CLOSED_WITH_ACCEPTED_EVIDENCE
```

Additional states may include `BLOCKED`, `REJECTED`, `NOT_APPLICABLE`, `INDETERMINATE`, `SUPERSEDED`, and `REOPENED`.

A Matter type may impose stricter states. For example, an Authority Request Case requires legal review before subject action or disclosure.

## 3.3 Risk Situation

A Risk Situation is a Matter subtype representing a current bounded instance of one or more Exposure Patterns.

It is used when observations indicate potential or realized harm, control failure, threshold movement, concentration, or material uncertainty.

Risk Situation does not replace Program. Continuing controls and obligations remain in Programs; a current failure, breach, or decision need becomes a Risk Situation Matter.

---

# 4. Shared institutional primitives

## 4.1 Scope

Scope is the bounded part of the institution, population, activity, or external relationship being governed.

Examples:

- institution or legal entity;
- licence or regulated activity;
- country or jurisdiction;
- Program;
- product, service, or channel;
- region, branch, or operating unit;
- project or processing activity;
- customer, merchant, account, or transaction population;
- system, asset, data set, model, or vendor relationship.

Scopes may be nested:

```text
Institution
└── Legal entity and licence
    └── Jurisdiction or region
        └── Program, product, channel, or service
            └── Branch, process, project, system, vendor, or population
```

Every material conclusion, decision, filing, response, export, and evidence submission must identify its scope and period.

## 4.2 Authority Source

An immutable, versioned original source of external or internal authority.

Examples:

- law, regulation, circular, guideline, standard, licence condition;
- contract or service obligation;
- court or enforcement instrument;
- supervisory report;
- approved internal policy or interpretation;
- board-approved appetite or mandate.

An Authority Source includes authenticity, issuing body, jurisdiction, source type, publication or receipt time, effective period, response or implementation deadlines, original artifact, hash, source coordinates, confidentiality, supersession, retention, and review state.

## 4.3 Source Provision

A stable, addressable section, paragraph, table, annex, schedule, clause, or other fragment of an Authority Source.

Every material Requirement, supervisory finding, case directive, or approved interpretation must link to one or more Source Provisions.

## 4.4 Requirement

A versioned statement of what an actor must, must not, may, or is expected to do.

A Requirement includes:

- source provisions;
- obligated actor or licence class;
- modality;
- normalized action and object;
- conditions, thresholds, frequency, and exceptions;
- effective period and deadline;
- reporting or evidence expectation;
- interpretation and approval state.

Requirement states may include:

```text
CANDIDATE
→ INTERPRETED
→ APPROVED
→ EFFECTIVE
→ AMENDED
→ SUPERSEDED
→ WITHDRAWN
```

A legacy spreadsheet row is a candidate Observation until reconciled to an Authority Source and approved.

## 4.5 Applicability Conclusion

A versioned conclusion about whether a Requirement applies to a defined scope and period.

Possible states:

- applicable;
- partially applicable;
- not applicable;
- potentially applicable—information required;
- applies from a future date;
- exempt under stated condition;
- superseded.

Applicability must record rationale, source facts, assumptions, reviewer, authority, and effective period.

## 4.6 Control Objective

The outcome the institution must achieve to address one or more Requirements, risks, policies, or objectives.

## 4.7 Control Implementation

The actual policy, process, system rule, review, approval, monitoring mechanism, or operating practice used in a defined scope.

One objective may have multiple implementations by entity, channel, system, branch, vendor, or period.

A Control Implementation includes:

- implementation scope;
- owner, performer, and reviewer;
- design and operating frequency;
- automation and dependencies;
- related Requirements and Exposure Patterns;
- Evidence Contracts;
- exceptions and linked Matters;
- design and operating-effectiveness Conclusions.

## 4.8 Policy

A governed institutional statement approved through a lifecycle of drafting, review, approval, publication, effective period, review, supersession, and retirement.

Policy is distinct from Requirement and Control Implementation.

## 4.9 Exposure Pattern

A reusable description of how an activity, service, population, dependency, control, or obligation can fail or create harm.

Initial families:

1. availability and resilience;
2. asset and inventory integrity;
3. identity and access;
4. transaction integrity;
5. reconciliation and settlement;
6. fraud and abuse;
7. data and privacy;
8. customer and conduct harm;
9. third-party and concentration dependency;
10. change and configuration integrity;
11. physical and environmental integrity;
12. regulatory, contractual, or policy non-conformance;
13. model or automated-decision failure;
14. evidence and data-quality uncertainty.

## 4.10 Observation

A normalized, source-preserving record of something observed, submitted, imported, measured, extracted, or asserted.

Observation fields include:

- subject and property;
- value and units;
- source and capture method;
- scope and population;
- effective time and capture time;
- original artifact or authoritative reference;
- file, sheet, row, event, API, or media coordinates;
- transformation history;
- source authority and limitations;
- sensitivity;
- confidence, review, and confirmation state;
- version and provenance.

An Observation is not automatically a verified fact.

## 4.11 Claim

A precise statement that can be supported, contradicted, qualified, or remain unresolved.

A Claim includes subject, scope, period, purpose, materiality, Evidence Contract, current Conclusion, and version.

## 4.12 Evidence Contract

A versioned policy describing what evidence is acceptable for a Claim.

It defines:

- required facts;
- population and period;
- acceptable source types;
- source authority and limitations;
- freshness;
- coverage;
- independence;
- authenticity and integrity;
- contradiction rules;
- reviewer authority;
- refresh schedule or trigger;
- escalation and failure behavior;
- whether automated evaluation is allowed.

Evidence Recipe is a task-level or template-level expression of an Evidence Contract.

## 4.13 Evidence Item and Evidence Evaluation

An Evidence Item is an immutable source artifact, governed snapshot, system result, submission, or observation used for a Claim.

Evidence Evaluation records how an item supports, partially supports, contradicts, limits, duplicates, supersedes, or fails to address a Claim.

## 4.14 Conclusion

A versioned determination of what current evidence supports.

Possible states:

- supported;
- partially supported;
- unsupported;
- contradicted;
- indeterminate;
- expired;
- not applicable.

A Conclusion identifies included and excluded evidence, contradiction, assumptions, sufficiency, evaluator, approval, valid period, and supersession.

## 4.15 Compliance State

A governed projection for a Requirement, control, scope, or Program.

Dimensions include:

- source interpretation;
- applicability;
- control design;
- implementation;
- evidence sufficiency;
- operating effectiveness;
- exception;
- assurance;
- filing or deadline;
- source and data quality.

Concise presentation states may include current, current with exception, at risk, gap identified, evidence insufficient, implementation pending, overdue, under review, not applicable, and unknown.

## 4.16 Review Activity

A scheduled or event-triggered assessment, test, filing preparation, certification activity, RCSA, BIA review, vendor review, policy review, audit procedure, or other planned governance work.

A Review Activity links to Program, scope, Requirements, controls, Evidence Contracts, owner, reviewer, frequency, due date, and resulting Observations or Matters.

Workplans and calendars are views over Review Activities.

## 4.17 KRI or Compliance Indicator

A versioned indicator definition with source, population, measure, period, threshold, owner, frequency, and response policy.

Indicator values should derive from canonical Observations, Events, Matters, losses, cases, or populations where possible.

A threshold breach may create or update a Matter.

## 4.18 Decision

An authorized selection among options in response to a Requirement, Matter, risk, incident, exception, reportability question, disclosure need, or other governed choice.

A Decision includes evidence, uncertainty, options, expected effects, authority, segregation of duties, rationale, dissent, conditions, expiry, Actions, and Verification Contract.

## 4.19 Action

A governed item of work with intended outcome, owner, performer, dependencies, due date, execution system, implementation evidence, escalation, and state.

Action completion does not prove outcome.

## 4.20 Verification Contract

A machine-readable definition of how ClearSight determines whether the selected response achieved its intended observable result.

It includes expected outcome, baseline, population, measurement source, success and failure thresholds, observation period, evidence, acceptance authority, and failure response.

## 4.21 Response Package

A point-in-time governed package prepared for a regulator, authority, auditor, certifier, board, customer, or other recipient.

It includes:

- purpose and recipient;
- scope and period;
- required directives or statements;
- included evidence and versions;
- exclusions and reasons;
- redactions;
- preparers, reviewers, approvers, and signatory;
- submission channel and transmission proof;
- acknowledgement;
- package manifest and retention.

---

# 5. Relationships

Representative relationships include:

```text
Program GOVERNS Scope
Program CONTAINS Requirement
Requirement DERIVED_FROM Source Provision
Requirement APPLIES_TO Scope
Requirement SATISFIED_BY Control Objective
Control Objective IMPLEMENTED_BY Control Implementation
Control Implementation PROVES_WITH Evidence Contract
Observation PRODUCED_BY Source Profile
Evidence Item SUPPORTS_OR_CONTRADICTS Claim
Claim CONCLUDES_AS Conclusion
Matter AFFECTS Program or Scope
Matter CREATED_BY Trigger or Authority Source
Matter REQUIRES Decision or Response Package
Decision CREATES Action
Action VERIFIED_BY Verification Contract
Review Activity TESTS Control or Requirement
KRI DERIVES_FROM Observation or Matter population
Response Package ANSWERS Directive or Requirement
```

Every material relationship should support provenance, valid time, record time, sensitivity, confidence or review state, and version.

---

# 6. Program trigger model

Programs subscribe to four trigger classes.

## Calendar

- filing and return dates;
- periodic control review;
- policy or certificate expiry;
- training, DR, vendor, BIA, RCSA, and assurance schedules.

## Institutional change

- new product, system, vendor, project, processing activity, jurisdiction, branch, customer group, model, or material configuration change.

## Operational event

- incident, breach, complaint pattern, loss, control failure, KRI breach, authority request, audit finding, or failed test.

## Regulatory or evidence change

- new source or amendment;
- applicability change;
- source degradation;
- evidence expiry;
- changed population;
- contradiction;
- failed verification.

A trigger may:

- refresh affected Claims;
- schedule a Review Activity;
- create or update a Matter;
- invalidate a Conclusion or Decision;
- request focused evidence;
- update reporting or filing readiness.

---

# 7. Configuration and reuse

## Base banking model

Shared primitives and exposure families.

## Program packs

Reusable source, Requirement, control-objective, Evidence Contract, calendar, trigger, and reporting templates for NDPA, AML/CFT, cybersecurity, PCI DSS, operational resilience, third-party assurance, and other Programs.

## Channel and domain packs

ATM, POS, mobile, internet, branch, cards, payments, lending, treasury, technology, privacy, vendors, and other domains.

## Jurisdiction packs

Authorities, source types, obligations, local terminology, thresholds, filing and response rules, retention, and legal conditions.

## Institution profile

Legal entities, licences, hierarchy, Programs, services, channels, source authority, control implementations, owners, authority, thresholds, terminology, and approved extensions.

Configuration must not create arbitrary per-bank schemas that prevent upgrades or cross-bank reuse.

---

# 8. Progressive integration and Source Registry

ClearSight must support:

- structured manual capture;
- photos, scans, documents, and spreadsheets;
- managed scheduled imports;
- APIs;
- events and telemetry.

All methods produce governed Observations.

Every Source Profile defines:

- source name, owner, and custodian;
- collection method;
- authoritative facts and explicit limitations;
- scope and identifiers;
- freshness target and current age;
- health and last successful collection;
- mapping version and known limitations;
- unresolved mappings;
- access and purpose policy;
- dependent Requirements, Claims, Conclusions, Programs, and Matters.

Data-quality weakness remains visible and may create a Matter or alter Compliance State without falsely asserting control failure.

---

# 9. AI role

AI acts as a governed compiler between messy sources and proposed structured objects.

AI may:

- segment source documents;
- extract candidate Requirements or directives;
- extract Observations from files, media, messages, and systems;
- normalize and propose matches;
- map Requirements, controls, services, and evidence;
- identify contradiction;
- draft focused requests, interpretations, actions, responses, and summaries;
- propose Program or Matter updates.

AI must not silently approve applicability, legal interpretation, material risk, reportability, suspicious reporting, disclosure, account restriction, regulatory response, major finding closure, or protected identity access.

---

# 10. User experience boundary

Primary surfaces are Today, Programs, Work, Explore, and Configure.

Programs present continuing obligations and assurance.

Work presents Matters, actions, evidence requests, reviews, approvals, and responses.

Capture and Respond are contextual lightweight experiences.

Graph, Evidence Fabric, Decision Ledger, workflow engine, and AI operators are internal capabilities and must not become mandatory navigation.

---

# 11. Product invariants

1. Programs maintain; Matters mobilize.
2. One truth produces many register, workplan, KRI, dashboard, and report views.
3. Authority Source before approved Requirement.
4. Applicability remains explicit and reviewable.
5. Control Objective remains distinct from scoped implementation.
6. Existing evidence before human request.
7. Source authority before automated trust.
8. Triggered refresh before blanket recurring questionnaire.
9. Evidence before confidence.
10. Contradiction before false certainty.
11. Decision or response before dashboard status.
12. Verification before closure.
13. Human authority for material judgment.
14. Progressive integration over perfect-integration dependency.
15. Institutional memory over periodic snapshots.
16. Internal architecture must not become user-interface architecture.

---

# 12. Definition of success

The operating model succeeds when:

- continuing obligations remain current without repeated spreadsheet reconstruction;
- a new regulation updates affected Programs and creates only necessary implementation Matters;
- an authority request becomes a protected case with source, subjects, directives, evidence, decisions, response, and acknowledgement;
- NDPA ROPA, DPIA, breach, vendor, rights, transfer, and filing work derive from one Program;
- legacy compliance, risk, exception, workplan, KRI, BIA, vendor, and loss registers become views over shared objects;
- regional banks can begin with spreadsheets and focused capture;
- larger banks can add APIs, events, entities, licences, and jurisdictions without changing semantics;
- staff are asked only for missing facts;
- task completion never masquerades as verified effectiveness;
- and every material compliance or risk position can be reconstructed from original source to accepted outcome.