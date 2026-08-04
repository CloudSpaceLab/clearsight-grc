# ClearSight Regulatory and Enforcement Intelligence

This document defines how ClearSight turns authoritative publications, supervisory communications, and enforcement requests into institution-specific obligations, controls, cases, actions, evidence, and defensible responses.

It specializes the canonical objects in [`operating-model.md`](operating-model.md). It does not create a separate product silo. Regulatory and enforcement work must use the same Scope, Observation, Claim, Evidence Recipe, Risk Situation, Decision, Action, Verification, source-lineage, and temporal-history mechanisms used elsewhere in ClearSight.

The product goal is:

> **When an authority communicates with the bank, ClearSight should determine what kind of communication it is, preserve the exact source, identify what the institution may need to do, generate the smallest correct governed workflow, and retain complete proof of interpretation, action, and response.**

AI may accelerate extraction, mapping, drafting, reconciliation, and explanation. It must not silently make final legal interpretation, declare applicability, file a suspicious-activity report, restrict an account, disclose protected customer information, or represent the institution to an authority without the required human authority.

---

# 1. Why one generic compliance workflow is insufficient

Banks receive materially different forms of external-authority communication.

A new circular may change the rules for every account, channel, vendor, or legal entity. A supervisory report may identify deficiencies unique to the institution. An enforcement letter may concern named customers, accounts, transactions, merchants, employees, or documents and require a case-specific response.

These documents share source verification, extraction, ownership, evidence, deadlines, approval, and audit requirements, but they must not be collapsed into one generic “compliance task.”

ClearSight recognizes three primary work classes.

## 1.1 Normative regulatory change

A publication changes or clarifies what a class of regulated institutions must, must not, or may do.

Examples:

- act, regulation, rule, circular, guideline, framework, standard, code, directive;
- amendment, addendum, FAQ, implementation guidance, interpretation or effective-date extension;
- exposure draft or consultation requiring impact assessment but not final implementation;
- sanctions, licensing, reporting, prudential, conduct, cyber, data-protection or payment requirements.

Primary output:

- source-linked Regulatory Obligations;
- applicability decisions;
- internal requirements;
- policy and control changes;
- implementation programme;
- Evidence Recipes and tests;
- readiness and regulatory-response status.

## 1.2 Supervisory or examination work

An authority identifies a concern, finding, information need, remediation expectation, attestation requirement, or institution-specific direction.

Examples:

- examination report;
- supervisory finding;
- deficiency notice;
- remediation directive;
- thematic-review response;
- formal information request;
- attestation request;
- regulatory meeting commitment.

Primary output:

- Supervisory Matter or Finding;
- management response;
- remediation actions and milestones;
- authority and committee oversight;
- required evidence and response package;
- effectiveness verification;
- formal closure or continuing obligation.

## 1.3 Investigative or enforcement casework

An authority requests or orders action concerning defined subjects, accounts, transactions, periods, devices, merchants, employees, branches, vendors, or records.

Examples:

- law-enforcement request;
- court order or other legal instrument;
- production or preservation request;
- account or transaction inquiry;
- customer-identification or address-verification request;
- direction requiring KYC refresh or enhanced review;
- request that may trigger an internal suspicious-activity or suspicious-transaction assessment;
- request for statements, onboarding records, device information, account mandates, correspondence, video, or other evidence.

Primary output:

- Authority Request Case;
- validated legal and authority basis;
- resolved subjects and requested periods;
- case directives and case tasks;
- KYC, address, AML, fraud, legal, branch, technology or records workflows;
- governed response package and submission proof;
- retention, legal hold and continuing-monitoring state.

A single source document may create more than one work class. For example, a supervisory report can contain a general expectation, multiple institution-specific findings, and requests for defined records.

---

# 2. Current-bank workflow patterns ClearSight must replace

The prospective-bank samples demonstrate a common pattern across compliance, IT risk, operational risk, privacy, resilience and vendor management:

- requirements are manually copied into spreadsheets;
- source documents and exact provisions are often not linked to the register row;
- different spreadsheets separately track requirements, assets, findings, exceptions, plans, KRIs, incidents, losses, dependencies and evidence;
- ownership is represented through names, comments and email follow-up rather than governed assignments and authority;
- one finding can be copied into a risk register, exception register, workplan, dashboard and committee report;
- free-text comments are used to redirect work between teams;
- uploading evidence and marking a row complete can be mistaken for verified remediation;
- recurring branch and head-office data is collected through large questionnaires even where some values can be sourced from systems;
- KRI values, thresholds, underlying populations and the cases producing them are disconnected;
- law-enforcement request volumes are aggregated as KRIs but the underlying requests, legal basis, subject cases, actions and responses are not represented in the same model;
- dashboards are requested as an additional reporting layer over the same fragmented records.

ClearSight must therefore avoid merely rebuilding these spreadsheets as web forms.

The desired transformation is:

```text
Authoritative source or institutional event
→ one governed source record
→ one or more typed obligations, findings, directives or situations
→ reusable relationships to affected services, assets, vendors, customers and controls
→ assigned decisions and actions
→ evidence and response
→ verified outcome
→ portfolio and KRI views derived from the same records
```

Registers remain useful views and import formats. They are not separate truth systems.

---

# 3. Canonical external-authority objects

## 3.1 Authority Source

An immutable, versioned source received from or published by an external authority.

Required attributes:

- source ID and version;
- authority and issuing department;
- jurisdiction;
- document type;
- title, reference number and subject;
- publication, issue, receipt and effective dates;
- response, consultation or implementation deadlines;
- official source URL, portal, email channel or physical-delivery metadata;
- original bytes or protected object reference;
- content hash and capture method;
- language;
- confidentiality, privilege and dissemination restrictions;
- authenticity and verification state;
- related, amended, superseding and superseded sources;
- parser and extraction versions;
- retention and legal-hold state.

Possible source types include:

- final law or regulation;
- circular or directive;
- guideline or framework;
- standard or code;
- amendment or addendum;
- FAQ or clarification;
- exposure draft or consultation;
- supervisory report;
- supervisory finding or remediation letter;
- information or attestation request;
- investigative or law-enforcement request;
- court order or other legal instrument;
- preservation request;
- sanctions or watchlist publication;
- enforcement notice;
- internal legal or compliance interpretation.

An internal spreadsheet row is not an Authority Source. It is a secondary observation or working record that must be reconciled to an Authority Source before it becomes an approved Regulatory Obligation or case directive.

## 3.2 Source Provision

A stable, addressable fragment of an Authority Source.

A provision records:

- source version;
- section, heading, page, paragraph, table, annex, schedule or coordinate;
- exact source excerpt where legally permitted;
- definitions and cross-references it depends on;
- parser structure;
- effective and temporal context;
- extraction confidence and review state.

Every downstream obligation, finding, directive or institutional interpretation must link to one or more provisions.

## 3.3 Authority Directive Atom

A normalized candidate statement extracted from a source provision.

It is the common semantic form from which Regulatory Obligations, Supervisory Findings and Case Directives are created.

Representative fields:

```yaml
issuer: authority reference
source_provision: exact source anchor
work_class: normative | supervisory | investigative
actor: institution, licence class, legal entity, named recipient or other obligated actor
modality: must | must_not | may | requested | ordered | expected | prohibited
action: normalized action
object: affected process, record, system, customer, account, transaction or deliverable
scope: jurisdiction, licence, product, channel, population, subject or period
condition: triggering condition
threshold: amount, percentage, count, duration or other limit
frequency: event-driven, continuous, daily, monthly, annual or one-time
deadline: effective, response, implementation or review deadline
exception: explicit exception or exemption
required_evidence: stated or inferred evidence need
response_recipient: authority endpoint or office
confidentiality: restrictions and handling class
legal_basis: source-stated basis or related instrument
interpretation_state: machine_candidate | under_review | approved | rejected | superseded
```

Machine extraction creates a candidate atom, never an approved legal interpretation.

## 3.4 Regulatory Obligation

An approved, normalized institutional interpretation of a normative directive.

It records:

- source provisions and version;
- interpretation owner and approvers;
- obligated entities and activities;
- applicability rationale;
- effective date and transition period;
- institutional requirement;
- mapped policies, controls, processes, systems, products, channels, vendors and data;
- implementation and evidence requirements;
- exceptions;
- readiness and assurance state;
- supersession history.

## 3.5 Supervisory Matter

A governed institution-specific expectation, concern, finding, commitment or response requirement.

It records:

- authority source and provisions;
- affected entity, service, process, product or control;
- authority’s observation and institution’s interpretation;
- severity and materiality;
- management response;
- agreed or imposed actions;
- milestones, deadlines and conditions;
- evidence and response packages;
- authority feedback and closure state;
- verification of remediation effectiveness.

## 3.6 Authority Request Case

A protected case created from an investigative, enforcement or subject-specific request.

Required attributes:

- case ID and source;
- requesting authority and contact channel;
- receipt time and response deadline;
- request type and legal-instrument state;
- case owner, legal reviewer and compliance authority;
- confidentiality and need-to-know policy;
- named or resolved subjects;
- requested periods, accounts, transactions, records and actions;
- case directives;
- case tasks and dependencies;
- communication and disclosure constraints;
- evidence collection and response-package state;
- submission and acknowledgement state;
- legal hold and retention;
- related cases and duplicate/amendment relationships;
- closure, continuing monitoring and escalation.

The case must distinguish:

- what the authority explicitly requested;
- what the institution is legally permitted or required to do;
- what ClearSight inferred as possible follow-up;
- what internal compliance, AML, fraud, KYC or legal teams independently decided.

## 3.7 Case Directive

A source-linked unit of requested or ordered casework.

Examples:

- provide account statements for a defined period;
- provide onboarding and KYC records;
- preserve specified records;
- verify or update customer identity information;
- obtain or verify an address;
- review defined transactions;
- determine whether an internal suspicious-activity or suspicious-transaction report is required;
- apply monitoring or other action only where supported by the validated authority and approved institutional policy;
- produce a formal response by a stated deadline.

A case directive includes the source provision, legal basis, subjects, required deliverable, deadline, authority, action restrictions, evidence recipe and completion criteria.

## 3.8 Subject

An entity relevant to an Authority Request Case.

Possible types:

- person or organization;
- customer profile;
- account or wallet;
- transaction or transaction set;
- merchant;
- device or terminal;
- address or location;
- employee;
- branch;
- vendor;
- document or record population.

Subject resolution must retain source identifiers, candidate matches, confidence, conflicts and merge/unmerge history. Ambiguous identity must route to review rather than silently merging records.

## 3.9 Compliance Rule Package

The approved institution-specific implementation package produced from a Regulatory Obligation.

A package may contain:

- internal requirement statement;
- policy amendment;
- control objective;
- control implementation;
- machine-enforceable or configuration rule where appropriate;
- monitoring and exception rule;
- reporting requirement;
- Evidence Recipe;
- test procedure;
- implementation actions and owners;
- transition and interim-risk treatment;
- communication and training requirements;
- verification contract.

The product must not imply that every legal obligation can or should be represented as executable code. Some rules require human judgement, procedural controls, legal interpretation or board authority.

## 3.10 Response Package

A governed package prepared for an authority.

It records:

- originating request and directives;
- included records and evidence versions;
- excluded items and reason;
- scope and period;
- redactions and privilege decisions;
- extraction or transformation history;
- preparers, reviewers and signatories;
- package manifest and integrity hashes where required;
- transmission channel and time;
- delivery confirmation or acknowledgement;
- follow-up questions and amendments.

A Response Package is not complete merely because files were attached. Each directive must be reconciled to a deliverable or an approved explanation.

---

# 4. Source and trust hierarchy

ClearSight must distinguish source authority.

## Tier 1 — Primary official source

- regulator or authority website, portal, authenticated message, signed physical communication or verified publication channel;
- court or other official legal instrument;
- official gazette or legislation repository.

## Tier 2 — Approved authoritative provider

- licensed regulatory-content provider;
- approved legal database;
- approved industry source with contractual provenance.

## Tier 3 — Institutional interpretation

- legal memorandum;
- compliance interpretation;
- approved applicability decision;
- regulator-engagement record.

## Tier 4 — Working and secondary material

- spreadsheet register;
- checklist;
- email summary;
- consultant note;
- meeting note;
- news article;
- model general knowledge.

Tier 4 material can help discover or explain requirements but cannot establish an authoritative obligation without reconciliation to a higher-tier source.

Every AI answer about a requirement must disclose the highest source tier used.

---

# 5. Regulatory Change Compiler

The Regulatory Change Compiler converts a new or changed normative source into a reviewed compliance programme.

## Stage 1 — Source intake and authentication

- capture the original source;
- identify issuer, document type, status, dates and reference number;
- verify official origin;
- detect duplicates, amendments, consultations and supersession;
- preserve original layout, tables, annexes and signatures;
- classify sensitivity and retention.

## Stage 2 — Provision segmentation

- identify headings, provisions, definitions, tables, annexes and cross-references;
- create stable source coordinates;
- preserve exact text and document structure;
- detect dates, thresholds, actors, modalities and exceptions;
- record parser confidence and unresolved structure.

## Stage 3 — Candidate directive extraction

- extract Authority Directive Atoms;
- distinguish mandatory, prohibited, permissive, advisory and consultation language;
- distinguish final requirements from explanatory or contextual text;
- identify effective, transition, filing and review dates;
- identify explicit evidence, reporting and recordkeeping requirements;
- preserve ambiguity.

## Stage 4 — Applicability analysis

Evaluate candidate directives against the Institution Profile:

- licence and institution type;
- legal entities and branches;
- jurisdictions;
- products, channels and customer populations;
- regulated activities;
- systems, data, vendors and outsourcing arrangements;
- thresholds or exemptions;
- current and planned operations.

Possible states:

- applicable;
- partially applicable;
- not applicable;
- potentially applicable—information required;
- applies only to defined entities or activities;
- applies at a future date;
- interpretation disputed or under legal review.

Applicability requires authorized human approval.

## Stage 5 — Existing coverage reconciliation

For each approved obligation, determine:

- whether an approved policy already addresses it;
- whether a control objective exists;
- where and how the control is implemented;
- which entities and systems are covered;
- whether evidence is current and sufficient;
- whether findings, exceptions or actions already exist;
- whether the new source changes or supersedes the current interpretation.

Coverage states:

- fully covered and sufficiently evidenced;
- covered but evidence stale or incomplete;
- partially covered;
- control objective exists but implementation is missing;
- implementation exists but policy or lineage is missing;
- contradictory implementations;
- not covered;
- applicability or mapping unresolved.

## Stage 6 — Rule and programme composition

Draft the Compliance Rule Package:

- internal requirement;
- policy change;
- control objective and implementation;
- technical or operational rule where possible;
- monitoring and exception logic;
- evidence and test requirements;
- implementation owners and deadlines;
- interim risk and compensating controls;
- regulatory communication or filing requirement.

## Stage 7 — Review and approval

Different reviewers approve different aspects:

- regulatory affairs or compliance: source, interpretation and applicability;
- legal: ambiguity, legal basis, conflicts, exceptions and entity boundaries;
- business or product: customer journey and operating effect;
- technology and security: systems, feasibility and technical rules;
- risk owner: exposure, interim risk and treatment;
- assurance: independent later conclusion.

Review must show the original source beside every proposed interpretation and resulting change.

## Stage 8 — Implementation and verification

- create actions in ClearSight or an approved execution engine;
- track policy, process, system, contract, training and communication changes;
- collect implementation evidence;
- test operating effectiveness for the required population and period;
- prepare authority, committee and examiner views;
- update readiness only from accepted evidence.

## Stage 9 — Amendment, clarification and supersession

When a later source changes the requirement:

- link the new source to the prior source;
- compare provisions and directive atoms;
- identify affected obligations, rules, controls, cases, actions and reports;
- preserve prior interpretations and effective periods;
- invalidate or reopen stale conclusions;
- route new approval and implementation work.

---

# 6. Enforcement and Authority Request Compiler

The Enforcement and Authority Request Compiler converts a subject-specific communication into a protected, legally reviewed case.

## Stage 1 — Intake

Supported channels may include:

- secure authority portal;
- controlled regulatory or law-enforcement mailbox;
- document-management integration;
- authenticated API;
- scan of a physical letter or legal instrument;
- manual protected upload.

Capture:

- receipt time;
- response deadline;
- sender and contact route;
- source reference and authenticity;
- attachments;
- confidentiality and handling restrictions;
- whether the source appears to be a duplicate, amendment or follow-up.

## Stage 2 — Legal and compliance triage

Before operational execution, authorized reviewers determine:

- authority and authenticity;
- legal-instrument status;
- scope and requested action;
- what information may be disclosed;
- what action is permitted or required;
- whether a court order, consent, legal basis or additional authorization is required;
- communication or notification restrictions;
- legal hold and preservation requirements;
- response owner and signatory.

ClearSight may identify missing or inconsistent legal-basis information. It must not make the final legal determination.

## Stage 3 — Subject resolution

Resolve each named or described subject against authorized institutional sources.

The system should support:

- exact identifiers;
- aliases and normalized names;
- account and customer relationships;
- device, merchant, address and transaction relationships;
- candidate matching;
- human review;
- merge and unmerge with history.

The case must display unresolved and ambiguous subjects explicitly.

## Stage 4 — Directive decomposition

Convert the request into discrete Case Directives.

For each directive, identify:

- exact source text;
- subject and period;
- requested information or action;
- deadline;
- legal review state;
- responsible team;
- required evidence and response format;
- dependencies;
- completion and approval criteria.

## Stage 5 — Case-plan generation

ClearSight proposes only the workflows required by the directives and approved institutional policy.

Possible workflows include:

- account and transaction record collection;
- KYC completeness review or refresh;
- identity-document verification;
- address capture or verification;
- branch or customer-service follow-up;
- device, merchant or channel review;
- AML or fraud investigation;
- suspicious-activity or suspicious-transaction reporting assessment;
- records preservation and legal hold;
- approved monitoring, restriction or other action;
- response drafting and submission.

A request from an authority does not automatically prove wrongdoing, suspicion, reportability or permission for every possible action.

## Stage 6 — Evidence collection

Before asking a person, search authorized sources for:

- customer and account master data;
- KYC and onboarding evidence;
- address and contact evidence;
- account mandates and beneficial ownership;
- transaction records;
- device and channel data;
- prior cases, reports and authority communications;
- branch, vendor or employee records;
- relevant policies and approvals.

Missing human knowledge should be captured through focused, protected requests.

## Stage 7 — Decision gates

Material decisions remain human-governed, including:

- legal sufficiency and disclosure scope;
- final subject match where ambiguous;
- KYC remediation or enhanced-review determination;
- suspicious-activity or suspicious-transaction report decision;
- account restriction, watchlist or other high-impact action;
- customer communication;
- privilege, redaction and information exclusion;
- external response approval and signature.

## Stage 8 — Execution and verification

Completion requires proof appropriate to the directive, for example:

- required records collected for the complete period;
- KYC fields updated in the authoritative system;
- address evidence captured and accepted;
- AML or fraud review concluded by the required authority;
- report or response filed through the approved channel;
- submission receipt or acknowledgement captured;
- legal hold applied to the required population;
- continuing monitoring or follow-up scheduled.

## Stage 9 — Response package and closure

The response workspace should reconcile every directive to:

- included deliverable;
- approved exclusion or inability to provide;
- reviewer and signatory;
- transmission proof;
- authority acknowledgement or follow-up.

The case may close only when all required directives have an accepted outcome or approved disposition. Retention, legal hold and continuing-monitoring obligations may remain active after operational closure.

---

# 7. Relationship to KRIs, losses, incidents and systemic risk

Authority work should not remain isolated from risk management.

## 7.1 Derived KRI views

ClearSight can derive metrics such as:

- requests received by authority and request type;
- requests with incomplete legal instruments or unresolved legal review;
- overdue responses;
- subjects requiring KYC or address remediation;
- repeat requests involving the same process or data-quality problem;
- response-package rejection or follow-up rate;
- regulatory obligations approaching deadline;
- obligations with weak or contradictory evidence.

The underlying cases and obligations remain available. A monthly KRI value is a view, not the source of truth.

## 7.2 Systemic issue detection

Repeated authority requests or case outcomes may indicate:

- KYC data-quality weakness;
- address-capture failure;
- transaction-monitoring gaps;
- slow records retrieval;
- branch-process inconsistency;
- policy or control ambiguity;
- customer or merchant onboarding weakness;
- inadequate retention or evidence lineage.

ClearSight may create or update a Risk Situation from validated patterns. It must not infer customer guilt, employee misconduct or reportability merely from request frequency.

## 7.3 Incident and loss linkage

A case may link to:

- operational incident;
- fraud or security incident;
- regulatory sanction;
- legal loss;
- customer remediation;
- control finding;
- disciplinary or legal action;
- recovery record.

These are relationships, not duplicated spreadsheet rows.

---

# 8. Product experience

## 8.1 Authority Inbox

A protected, role-specific queue showing:

- new and changed regulatory sources;
- supervisory matters;
- investigative or enforcement requests;
- authenticity and document-status warnings;
- response and implementation deadlines;
- required review authority;
- duplicates, amendments and supersession.

## 8.2 Source Review

A source-linked split workspace:

```text
Original source provision       Proposed interpretation or directive
─────────────────────────       ───────────────────────────────────
Document page/paragraph         Actor, action, scope and deadline
Definitions and context         Applicability or legal-review state
Related source versions         Confidence and unresolved ambiguity
```

Reviewers can accept, edit, reject, split, merge or request clarification while preserving the machine candidate.

## 8.3 Regulatory Impact Workspace

Shows:

- affected legal entities and licences;
- products, channels, services, systems, vendors and customer groups;
- existing obligations and controls;
- coverage and evidence gaps;
- proposed rule packages;
- implementation dependencies and deadlines;
- readiness by obligation rather than task count.

## 8.4 Compliance Rule Composer

A structured composer for:

- internal requirement;
- policy language;
- control objective;
- implementation rule;
- monitoring and exception logic;
- Evidence Recipe;
- test method;
- owner, approver and deadline;
- verification contract.

The user must always be able to inspect the exact source provision.

## 8.5 Authority Request Case Workspace

One workspace with:

- source and legal-basis review;
- subjects and matching state;
- directives and deadlines;
- case tasks and accountable teams;
- evidence and missing facts;
- decisions and approvals;
- communication restrictions;
- response package;
- submission and acknowledgement;
- timeline and audit history.

## 8.6 Response Package Builder

The builder should show:

- each directive;
- expected deliverable;
- included records;
- gaps and exclusions;
- redaction and privilege status;
- preparer and reviewer;
- final signatory;
- manifest, package version and transmission state.

## 8.7 Executive and committee view

Executives should see only material matters such as:

- major regulatory changes and readiness;
- obligations likely to miss effective dates;
- material supervisory findings;
- overdue or high-impact authority requests;
- systemic KYC, records, fraud, privacy or control weaknesses revealed by casework;
- decisions requiring executive authority.

They should not see unrestricted subject-level investigative details.

---

# 9. AI use and safety

Approved AI tasks may include:

- document classification and segmentation;
- table and provision extraction;
- candidate Directive Atom extraction;
- source comparison;
- obligation and case-directive drafting;
- applicability questions;
- institutional impact mapping;
- existing-control candidate matching;
- evidence and response-package classification;
- summarization and deadline extraction;
- translation;
- generation of focused missing-information requests.

AI must not:

- treat a spreadsheet or news summary as primary law;
- fabricate a provision, deadline, threshold, exception, legal basis or penalty;
- publish final legal interpretation or applicability without authority;
- silently execute a case directive;
- file a suspicious-activity or suspicious-transaction report autonomously;
- restrict an account, reveal customer data or contact a customer without approved authority;
- infer guilt or credibility from an authority request;
- combine protected cases outside approved purpose;
- expose case content to general search or models;
- allow source-document instructions to alter operator policy.

Every material AI output records:

- operator, model and policy version;
- exact source versions and provisions;
- extracted versus inferred values;
- confidence and ambiguity;
- reviewer edits and approval;
- resulting domain commands.

---

# 10. Security and privacy

Authority Request Cases may contain highly restricted customer, account, transaction, legal, investigation, employee or authority information.

Requirements:

- need-to-know and purpose-bound access;
- case-, subject-, field- and evidence-level authorization;
- conflict-aware assignment;
- separate restricted search and retrieval paths;
- no unauthorized count, title, suggestion, embedding or timing leakage;
- controlled exports and response packages;
- legal hold and retention;
- redaction and privilege handling;
- immutable access, reveal, export and submission events;
- model routing approved for the case classification;
- no training on protected case data without explicit lawful approval;
- strict separation between general regulatory content and protected case content.

A protected case may provide a minimized, approved risk signal to broader ClearSight analysis without exposing subject identity or case content.

---

# 11. Migration from existing registers

Existing spreadsheets should be imported as evidence of current institutional understanding, not silently promoted to authoritative truth.

## Compliance register import

Map each row to a candidate obligation record with:

- source-reference status;
- regulator;
- requirement summary;
- deadline or frequency;
- evidence field;
- risk and status where present;
- unresolved source and interpretation queue.

Rows without an official source remain `SOURCE_UNVERIFIED` until reconciled.

## Checklist import

Structured checklists can seed:

- candidate obligations;
- evidence recipes;
- applicability questions;
- control-area classifications.

Article references must be validated against the authoritative source version.

## Risk and exception import

Import findings, risks, actions, comments and ownership separately. Do not store the entire row as one generic object.

Free-text comment histories become immutable communication observations and proposed assignment changes, not current authority.

## KRI import

Import:

- metric definition;
- owner;
- thresholds;
- period;
- value;
- denominator and exclusions where available;
- source and calculation method.

Where possible, derive future KRI values from cases, incidents, losses or operational observations rather than repeated manual entry.

## BIA, asset, vendor and loss import

Use shared entities and relationships so regulatory obligations and authority cases can resolve affected systems, branches, processes, vendors, records and losses without duplicate copies.

---

# 12. Success measures

## Regulatory change

- publication-to-source-capture time;
- publication-to-candidate-obligation time;
- publication-to-approved applicability decision;
- publication-to-approved implementation programme;
- missed-obligation and false-obligation rate;
- reviewer edit, rejection and abstention rate;
- existing-control reuse and duplicate-control avoidance;
- obligations with complete source-to-control-to-evidence lineage;
- implementation and operating-effectiveness readiness by effective date.

## Supervisory work

- receipt-to-owned-response time;
- overdue supervisory commitments;
- evidence completeness and rejection rate;
- remediation verified effective;
- repeat findings;
- authority closure or continuing-obligation state.

## Authority request cases

- receipt-to-authenticity and legal triage;
- subjects resolved versus ambiguous;
- directives with complete dispositions;
- deadline compliance;
- evidence-request burden;
- response-package completeness;
- submission acknowledgement time;
- KYC or address remediation completion;
- duplicate and amended request reconciliation;
- unauthorized access, disclosure or action attempts.

## Trust

- source-tier disclosure;
- outputs with exact provision lineage;
- unsupported assertion rate;
- protected-case leakage tests;
- time to reconstruct an interpretation, action or response;
- human override and correction quality.

---

# 13. Initial product wedge

The first production capability should support two connected vertical slices.

## Slice A — CBN-style regulatory circular

- ingest an official circular and attachments;
- classify final versus draft or clarification;
- segment and extract candidate directives;
- obtain compliance/legal approval;
- assess applicability to one legal entity and selected banking channels;
- reconcile with existing policies and controls;
- compose a small Compliance Rule Package;
- create implementation actions and Evidence Recipes;
- verify one technical and one procedural outcome;
- preserve source-to-evidence lineage.

## Slice B — Authority request concerning customer accounts

- ingest a protected request and legal instrument;
- authenticate and triage;
- resolve a small subject population;
- decompose requested records and actions;
- create KYC, address, records and AML-review tasks as applicable;
- collect only missing evidence;
- route material decisions to humans;
- build and approve a response package;
- record submission and acknowledgement;
- derive a minimized KRI view without exposing subject content.

This pair proves that ClearSight can govern both broad regulatory change and specific external-authority casework without reverting to generic forms or disconnected registers.

---

# 14. Product invariants

1. **Primary source before register row**
2. **Exact provision before obligation**
3. **Document class before workflow**
4. **Applicability before implementation**
5. **Legal authority before external action**
6. **Case directive before task**
7. **Existing institutional evidence before human request**
8. **Human authority for legal interpretation, reportability and high-impact action**
9. **Response completeness before submission**
10. **Acknowledgement and retained proof before closure**
11. **Protected case content never becomes general AI context**
12. **KRI and dashboard views derive from governed source records**
13. **Amendments supersede; they do not overwrite history**
14. **Authority request does not imply guilt or reportability**
15. **Task completion is not regulatory or remediation effectiveness**

---

# 15. Definition of success

This capability succeeds when ClearSight can receive a new authority document and answer, with complete lineage:

- What kind of document is this?
- Is it authentic and final?
- What exact provisions matter?
- Is this a general rule, a supervisory matter, a case-specific directive, or a combination?
- Which bank entities, products, channels, systems, vendors, customers or accounts are affected?
- What is already covered and evidenced?
- What decisions require legal, compliance, risk, business or executive authority?
- What rules, controls, case tasks, evidence and responses are required?
- What remains ambiguous or prohibited from automation?
- Has the institution implemented, verified, responded and retained proof?

The user should not need to manually recreate the same authority communication across a compliance register, risk register, exception tracker, workplan, case spreadsheet, KRI workbook, email chain and management dashboard.