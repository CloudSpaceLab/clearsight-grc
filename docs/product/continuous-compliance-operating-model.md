# ClearSight Continuous Compliance Operating Model

This document defines how ClearSight replaces disconnected registers, recurring questionnaires, reminder-driven work, and manually assembled reports with continuously maintained Programs and bounded Matters.

It conforms to:

- [`ease-of-use-standard.md`](ease-of-use-standard.md)
- [`operating-model.md`](operating-model.md)
- [`regulatory-and-enforcement-intelligence.md`](regulatory-and-enforcement-intelligence.md)

The core product outcome is:

> **Maintain a current, evidence-backed compliance position with the least reasonable human effort, then create a governed Matter only when change, uncertainty, failure, exception, or external demand requires attention.**

---

# 1. Why continuous compliance

Banks often maintain the same reality across separate compliance registers, control sheets, RCSA workbooks, exception trackers, vendor registers, BIA files, KRI submissions, audit findings, policy trackers, workplans, and dashboards.

This creates repeated work:

- the same application, branch, vendor, owner, control, or requirement is re-entered;
- evidence is repeatedly requested;
- status is manually copied;
- source changes are missed;
- ownership is transferred through email and comments;
- filings and examinations trigger emergency reconstruction;
- completed tasks appear closed without verified outcome.

ClearSight replaces that pattern with:

```text
Programs maintain continuing obligations
+ Sources continuously provide observations
+ Trigger rules detect meaningful change
+ Matters handle exceptions and decisions
+ Evidence and verification update Program state
```

---

# 2. Programs

A Program is a long-lived governed body of requirements, controls, evidence, reviews, exceptions, filings, and assurance.

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

A Program contains:

- authority sources and standards;
- approved Requirements and applicability;
- scoped institutional requirements;
- Control Objectives and Control Implementations;
- Evidence Contracts;
- current Observations and Conclusions;
- owners, reviewers, and authority;
- schedule, filing, certification, and review events;
- current Compliance State;
- open Matters and exceptions;
- assurance conclusions;
- history.

A Program is not a static checklist. Its state is computed from source, scope, controls, evidence, exceptions, assurance, and deadlines.

---

# 3. Matters

A Matter is bounded work created because something changed, failed, expired, conflicted, exceeded a threshold, or required a response.

Matter types include:

- regulatory change;
- supervisory finding;
- authority request;
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

A Matter contains only the sections relevant to its type:

- source and trigger;
- scope and affected objects;
- evidence and missing facts;
- assessment or legal review;
- decision and authority;
- actions and dependencies;
- response package where applicable;
- outcome or acknowledgement;
- verification and closure;
- history.

A Matter should not recreate a permanent module-specific register row.

---

# 4. The continuous compliance chain

```text
Authority Source or Standard
→ Requirement
→ Applicability
→ Institutional Requirement
→ Control Objective
→ Control Implementation
→ Evidence Contract
→ Current Observations
→ Compliance Conclusion
→ Program State
→ Matter where intervention is required
```

## 4.1 Authority Source

The original law, circular, guideline, standard, licence condition, contract, court instrument, supervisory communication, or approved interpretation.

A manually authored spreadsheet row is a secondary working record until reconciled to the source.

## 4.2 Requirement

A source-linked statement describing what an actor must, must not, may, or is expected to do.

## 4.3 Applicability

A versioned human-approved determination of where the Requirement applies:

- legal entity;
- licence;
- jurisdiction;
- product or channel;
- service or process;
- customer or account population;
- system or vendor;
- processing activity;
- threshold or condition;
- effective period.

## 4.4 Control Objective

The outcome the institution must achieve.

## 4.5 Control Implementation

The actual policy, process, system rule, monitoring mechanism, review, approval gate, training activity, or operating practice used in a defined scope.

## 4.6 Evidence Contract

Defines what proves the Requirement and control are satisfied:

- claims;
- population and period;
- acceptable sources;
- source authority and limitations;
- freshness and coverage;
- independence;
- contradiction rules;
- reviewer authority;
- refresh schedule or event triggers;
- failure handling.

## 4.7 Compliance State

Separate dimensions remain visible:

- interpretation;
- applicability;
- control design;
- implementation;
- evidence sufficiency;
- operating effectiveness;
- exception;
- assurance;
- deadline and filing;
- source and data quality.

Concise states may include current, at risk, gap identified, evidence insufficient, implementation pending, overdue, under review, not applicable, or unknown.

No unexplained percentage may replace the dimensions.

---

# 5. Trigger-driven operation

Programs respond to four trigger classes.

## 5.1 Calendar triggers

- regulatory returns;
- annual compliance filings;
- quarterly board reports;
- control reviews;
- access reviews;
- policy reviews;
- certificate expiry;
- DR and resilience tests;
- vendor reassessments;
- assurance workplans.

## 5.2 Institutional change triggers

- new product, project, branch, legal entity, vendor, application, account type, customer segment, data use, processing activity, jurisdiction, owner, or system configuration;
- service or dependency relationship change;
- policy or control version change;
- acquisition, outsourcing, migration, or decommissioning.

## 5.3 External change triggers

- new regulation, circular, amendment, FAQ, standard, supervisory communication, sanction, authority request, or legal instrument.

## 5.4 Event and evidence triggers

- incident, breach, complaint, loss, control failure, KRI breach, audit finding, verification failure;
- evidence expiry;
- source degradation;
- changed population;
- contradiction;
- failed test;
- revoked certificate;
- overdue action.

A trigger may:

- update Program state automatically;
- request focused evidence;
- create a Matter;
- reopen a Matter;
- invalidate a decision;
- schedule a review;
- route a notification;
- require human approval.

---

# 6. Source-led workflow simplification

## 6.1 Source Registry

Each source defines:

- owner and custodian;
- authoritative fields;
- limitations;
- scope;
- identifiers;
- freshness target;
- current health;
- mapping version;
- known data-quality issues;
- purpose and access policy.

## 6.2 Bank inventories as workflow inputs

ClearSight should use approved bank inventories to pre-populate Programs and Matters:

- applications from CMDB or enterprise architecture;
- assets from asset systems;
- branches from institution directories;
- owners and employees from HR;
- accounts and access from IAM;
- vendors and contracts from procurement;
- customers and accounts from approved core systems;
- merchants and terminals from acquiring systems;
- projects and changes from ITSM or delivery tools;
- processing activities from ROPA;
- dependencies from BIA;
- policies, certificates, and prior evidence from repositories.

Users should correct or confirm unresolved information, not recreate the inventory.

## 6.3 Progressive integration

- Level 0: controlled lists, forms, photos, spreadsheets, documents.
- Level 1: scheduled files, SFTP, exports, recurring imports.
- Level 2: APIs.
- Level 3: events and telemetry.

All levels produce the same governed Observations. The workflow should remain recognizable as automation increases.

## 6.4 Source degradation

When a source is stale or unavailable:

- affected claims and Program states are identified;
- last-known value and age remain visible;
- approved fallback is suggested;
- manual confirmation is labelled according to its authority;
- unsafe actions are blocked;
- a source-quality Matter may be created.

---

# 7. Minimum-human-effort workflow

Every Program or Matter workflow follows:

```text
Resolve scope
→ retrieve known context
→ calculate what is missing or changed
→ prepare recommendation or focused request
→ human reviews only material uncertainty or exception
→ execute or route
→ verify and update state
```

## 7.1 Prefill

Known scope, owners, systems, controls, evidence, prior decisions, and source values are prefilled.

## 7.2 Targeted requests

Requests contain only unresolved facts and stop when sufficient evidence arrives from any approved source.

## 7.3 Review by exception

Reviewers focus on changed values, contradictions, new mappings, missing source anchors, low-confidence extraction, material impact, and high-risk actions.

## 7.4 Save and resume

Complex Matters preserve completed steps, drafts, changes, blockers, and next action.

## 7.5 Active-effort targets

Routine work targets less than five minutes of active effort. Complex work must reach a safe saved next state in that period.

---

# 8. Governed AI within continuous compliance

AI may:

- classify sources;
- segment provisions;
- extract candidate Requirements;
- propose applicability questions;
- map Requirements to Programs, controls, systems, vendors, and owners;
- summarize Program changes;
- identify missing or contradictory evidence;
- draft focused requests;
- recommend remediation and verification;
- prepare first drafts of policies, reviews, filings, and response indexes;
- propose assignments;
- compare versions and prior decisions.

AI output must include:

- source references and versions;
- affected scope;
- explicit versus inferred values;
- assumptions and uncertainty;
- editable structured fields;
- required authority;
- safe alternatives.

AI must not:

- make final legal interpretation;
- silently approve applicability;
- declare compliance;
- file a suspicious report;
- restrict an account;
- disclose protected information;
- close a material Matter;
- represent the bank externally without authority.

---

# 9. NDPA Program example

An NDPA Program may contain:

- registration and classification;
- DPO governance;
- ROPA;
- lawful basis and consent;
- privacy notices;
- DPIA;
- security and technical measures;
- breach management;
- data-subject rights;
- vendor and processor governance;
- cross-border transfers;
- retention and deletion;
- annual compliance audit and filing.

## 9.1 ROPA

ClearSight preloads known applications, vendors, projects, systems, departments, data stores, and relationships.

It requests only unresolved facts such as purpose, lawful basis, data categories, recipients, retention, transfer, or accountable owner.

Changes reopen only affected processing activities.

## 9.2 DPIA

A new project, vendor, AI system, sensitive-data use, or process change triggers a prefilled screening.

AI may recommend whether a full DPIA appears necessary. The DPO approves the determination, assigns remediation, and records go-live conditions.

## 9.3 Breach Matter

Contains detection and awareness times, affected systems and data, data-subject population, risk assessment, notification decision, communications, evidence, remediation, and verification.

## 9.4 Annual filing

Evidence is assembled throughout the year. Filing becomes exception review, final approval, submission, and acknowledgement rather than annual reconstruction.

---

# 10. Regulatory and authority work

## 10.1 Regulatory change Matter

```text
Official source
→ exact provisions
→ candidate Requirements
→ interpretation and applicability review
→ affected Programs and controls
→ implementation Matters
→ evidence and tests
→ updated continuing state
```

## 10.2 Supervisory Matter

Contains the finding, management response, commitments, milestones, evidence, external response, and effectiveness verification.

## 10.3 Authority Request Matter

Contains verified source and legal instrument, subjects and periods, disclosure review, directives, KYC/address/records/AML/fraud tasks, response package, transmission, acknowledgement, retention, and legal hold.

An authority request does not automatically establish guilt or reportability.

---

# 11. Legacy register transition

Legacy spreadsheets are imported as source observations and reconciled into canonical objects.

| Legacy artefact | ClearSight target |
|---|---|
| Compliance register | Program Requirements view |
| Risk register | Risk Matters and portfolio |
| Exception register | Exception Matters |
| Workplan | Review Activities |
| RCSA workbook | Program review plus generated Matters |
| KRI workbook | Derived Indicators |
| BIA register | Shared service and dependency context |
| Vendor register | Third-party profile, evidence, and Matters |
| Loss register | Loss Events linked to incidents and recovery |
| Dashboard | Derived role view |

Migration rules:

- preserve original file, sheet, row, mapping, and import version;
- do not treat row text as authoritative regulation;
- avoid duplicate canonical entities;
- surface unresolved mappings;
- retain historical comments as communication observations;
- require source and evidence reconciliation before approved status.

---

# 12. User surfaces

## Today

Programs and Matters needing attention.

## Programs

Continuing compliance position, Requirements, controls, evidence, schedule, exceptions, assurance, and recent changes.

## Work

Matter queue, actions, requests, reviews, approvals, and responses.

## Explore

Cross-Program and cross-Matter search, relationships, populations, source lineage, and history.

## Configure

Programs, source profiles, templates, packs, evidence contracts, authority, triggers, retention, and automation policy.

## Respond and Capture

Focused external or field experiences requiring minimal context and active effort.

---

# 13. Program state and Matter closure

## Program state

A Program remains active indefinitely. It changes as requirements, scope, evidence, controls, exceptions, or assurance change.

## Matter closure

Closure requirements depend on type:

- remediation: verified outcome;
- regulatory change: approved obligations incorporated and implementation verified;
- authority request: directives reconciled, response approved, transmitted, and acknowledged or formally pending;
- finding: accepted evidence and effectiveness review;
- exception: expiry, revocation, replacement, or approved revalidation;
- incident: required response, lessons, and remediation state.

Task completion alone is insufficient.

---

# 14. Success measures

## Human effort

- routine completion time;
- manually entered versus prefilled fields;
- duplicate requests avoided;
- repeat import mapping reuse;
- time to resume;
- screens or transitions;
- AI draft acceptance and edit rate.

## Compliance continuity

- applicable Requirements with mapped controls;
- current sufficient evidence;
- evidence age and coverage;
- Matters generated from meaningful triggers;
- time to filing or examination package;
- annual questionnaire fields eliminated.

## Trust

- source lineage completeness;
- unresolved contradictions;
- source-health impact;
- human authority compliance;
- point-in-time reconstruction;
- response-package integrity.

---

# 15. Definition of success

The model succeeds when:

- a bank maintains compliance position continuously rather than before deadlines;
- source systems and inventories eliminate repeated data entry;
- routine users complete work in a few steps and under five minutes;
- complex work reaches a clear saved next state quickly;
- AI prepares useful grounded recommendations without becoming the authority;
- one change updates all relevant Program, Matter, KRI, dashboard, audit, and reporting views;
- legacy registers become projections rather than separate truth systems;
- material closure requires evidence and verification;
- the institution can explain what it knew, decided, did, and proved at any point in time.