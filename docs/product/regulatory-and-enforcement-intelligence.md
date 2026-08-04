# ClearSight Regulatory and Enforcement Intelligence

This document defines how ClearSight turns authoritative publications, supervisory communications, and enforcement or information requests into institution-specific Requirements, Programs, Matters, controls, actions, evidence, responses, and verified outcomes.

It conforms to:

- [`continuous-compliance-operating-model.md`](continuous-compliance-operating-model.md)
- [`ease-of-use-standard.md`](ease-of-use-standard.md)
- [`operating-model.md`](operating-model.md)

The product goal is:

> **When an authority communicates with the bank, ClearSight preserves the exact source, determines the type of work required, assembles the institution’s known context, prepares a grounded first draft, and routes the smallest correct governed workflow.**

Routine intake and triage should take only a few clear steps and less than five minutes of active effort where source and scope are sufficiently clear. Complex legal, investigative, or interpretive work should reach a clear saved next state within five minutes.

AI may accelerate extraction, mapping, drafting, reconciliation, and explanation. It must not silently make final legal interpretation, declare applicability, file a suspicious report, restrict an account, disclose protected customer information, or represent the institution externally without authority.

---

# 1. Work classes

## 1.1 Normative regulatory change

A publication changes or clarifies what a class of institutions must, must not, or may do.

Examples:

- law, regulation, rule, circular, guideline, framework, standard, code, directive;
- amendment, addendum, FAQ, implementation guidance, or effective-date extension;
- exposure draft or consultation;
- sanctions, licensing, reporting, prudential, conduct, cyber, privacy, or payment requirement.

Primary output:

- Authority Source and provisions;
- source-linked Requirements;
- applicability decisions;
- affected Programs;
- institutional requirements;
- control and policy changes;
- implementation Matters;
- Evidence Contracts and tests;
- readiness and filing state.

## 1.2 Supervisory or examination work

An authority identifies an institution-specific concern, finding, remediation expectation, information need, attestation, or commitment.

Primary output:

- Supervisory Matter;
- management response;
- actions and milestones;
- evidence;
- authority communication;
- effectiveness verification;
- closure or continuing obligation.

## 1.3 Investigative or enforcement casework

An authority requests or orders action concerning named subjects, accounts, transactions, periods, devices, merchants, employees, branches, vendors, or records.

Primary output:

- protected Authority Request Matter;
- legal and authority review;
- subject and period resolution;
- case directives;
- focused KYC, address, records, AML, fraud, branch, legal, or technology tasks;
- Response Package;
- approval, transmission, and acknowledgement;
- retention and legal hold.

One source may create multiple work classes.

---

# 2. Authority Source

An immutable, versioned source received from or published by an external authority.

Attributes:

- authority and issuing department;
- jurisdiction;
- document type and status;
- title, reference number, and subject;
- publication, issue, receipt, and effective dates;
- response, consultation, or implementation deadline;
- official URL, portal, email channel, or delivery metadata;
- original content or protected reference;
- hash and capture method;
- language;
- confidentiality, privilege, and dissemination restrictions;
- authenticity state;
- amendment and supersession relationships;
- retention and legal hold.

A legacy compliance-register row is a secondary observation until reconciled to an Authority Source.

---

# 3. Source Provision and Directive Atom

A Source Provision is a stable, addressable fragment with page, section, heading, paragraph, table, annex, schedule, coordinates, definitions, cross-references, and effective context.

A Directive Atom is a normalized candidate statement containing:

- issuer and source provision;
- work class;
- actor;
- modality: must, must not, may, requested, ordered, expected, prohibited;
- action and object;
- scope and population;
- condition and threshold;
- frequency;
- deadline;
- exception;
- required evidence or deliverable;
- response recipient;
- confidence and interpretation state.

Every downstream Requirement, finding, or directive must retain exact source lineage.

---

# 4. Context assembly before review

Before showing a regulatory or authority workflow, ClearSight should retrieve authorized context.

## Regulatory change context

- institution and legal entities;
- licences and jurisdictions;
- Programs and existing Requirements;
- policies and controls;
- products, channels, services, systems, vendors, and owners;
- prior related circulars and interpretations;
- current evidence and exceptions;
- implementation and assurance history.

## Authority case context

Where purpose and authority permit:

- named customer, account, merchant, employee, vendor, device, or transaction candidates;
- KYC and address state;
- account and onboarding records;
- transaction and communication records;
- prior cases and legal holds;
- records locations and custodians;
- current source health and data limitations.

Known values are prefilled. Users review unresolved identity, legal, scope, or evidential questions rather than reconstruct the case manually.

---

# 5. Governed AI first drafts

## Regulatory change

AI may propose:

- document type and final/draft status;
- provision segmentation;
- Requirement extraction;
- version comparison;
- applicability questions;
- affected Programs, systems, controls, vendors, and owners;
- control changes;
- implementation actions;
- Evidence Contracts and tests;
- executive summary.

## Supervisory work

AI may propose:

- finding decomposition;
- management-response structure;
- existing related findings and controls;
- action and milestone plan;
- evidence index;
- verification criteria.

## Authority case

AI may propose:

- directive decomposition;
- subject candidates;
- records index;
- missing-information list;
- task routing;
- response-package manifest;
- draft factual summary based on approved sources.

Every recommendation must show source, scope, assumptions, uncertainty, required authority, editable structured output, alternatives, and expected next state.

---

# 6. Five-minute workflow checkpoints

## 6.1 New regulatory publication

Within five minutes of active effort, an authorized reviewer should be able to:

1. verify or flag source authenticity and status;
2. inspect the AI-produced summary and provision count;
3. confirm likely affected Programs and entities;
4. accept, correct, or assign detailed interpretation;
5. save and route the Regulatory Change Matter.

Detailed legal interpretation may continue asynchronously.

## 6.2 Supervisory communication

Within five minutes:

1. confirm source, deadline, and responsible function;
2. review extracted findings and commitments;
3. identify immediate evidence or response gaps;
4. assign owners;
5. save the governed next step.

## 6.3 Authority request

Within five minutes:

1. preserve and classify the source;
2. confirm protected handling;
3. identify legal-review requirement and deadline;
4. review subject-match candidates without finalizing ambiguous matches;
5. assign legal/case owner and save next step.

No restricted disclosure or account action occurs during convenience-oriented triage.

---

# 7. Regulatory change workflow

```text
Official source received
→ authenticity and status
→ provision segmentation
→ candidate Requirements
→ human interpretation and applicability
→ affected Program and control mapping
→ implementation Matters
→ evidence and testing
→ Program state update
→ amendment monitoring
```

## Usability rules

- show source text beside proposed structured Requirement;
- highlight changed, low-confidence, ambiguous, and material provisions;
- reuse prior approved interpretations and mappings;
- prefill affected applications, vendors, owners, and controls from inventories;
- allow bulk acceptance only for authorized low-risk unchanged patterns;
- preserve save/resume and changed-since-last-view;
- never require users to copy source text into a register row.

---

# 8. Compliance rule package

For each approved Requirement, ClearSight may create:

- institutional requirement;
- policy change;
- Control Objective;
- scoped Control Implementations;
- machine-enforceable rule where appropriate;
- monitoring rule;
- Evidence Contract;
- test procedure;
- implementation Matter;
- exception policy;
- reporting requirement.

This package remains draft until approved.

---

# 9. Supervisory workflow

A Supervisory Matter records:

- source and exact finding;
- affected scope;
- management interpretation;
- existing controls and evidence;
- root cause or uncertainty;
- management response;
- commitments and milestones;
- authority and committee oversight;
- response package;
- verification and closure state.

Users should review AI-proposed mappings and actions rather than enter them from scratch.

---

# 10. Authority Request workflow

## 10.1 Intake

Capture original letter, portal record, email, order, attachments, reference, receipt, deadline, signature/authentication indicators, confidentiality, and legal instrument.

## 10.2 Legal and compliance triage

Authorized reviewers determine authenticity, authority, legal basis, disclosure scope, required action, customer-communication constraints, legal hold, response owner, and signatory.

## 10.3 Directive decomposition

Examples:

- provide statements for a defined period;
- provide onboarding and KYC records;
- confirm address information;
- preserve records;
- provide device or transaction data;
- respond through approved channel by deadline.

## 10.4 Subject resolution

Possible states:

- exact match;
- provisional match;
- multiple candidates;
- no match;
- contradictory identifiers.

Ambiguous subjects are never silently merged.

## 10.5 Focused casework

Depending on validated scope and policy:

- records collection;
- KYC completeness review;
- KYC refresh;
- address capture or verification;
- transaction review;
- branch follow-up;
- device or merchant review;
- legal-hold action;
- AML or fraud investigation;
- suspicious-reporting assessment;
- approved monitoring or other action;
- external response preparation.

An authority request does not automatically prove wrongdoing, create suspicion, authorize account restriction, or require suspicious reporting.

## 10.6 Response Package

Every directive is reconciled to included evidence, approved exclusion, inability to provide, redaction, preparer, reviewer, signatory, transmission proof, and acknowledgement.

Files sent is not equivalent to completed response.

---

# 11. Protected-case usability

Protected workflows must remain easy without leaking data.

Requirements:

- direct access only for assigned authorized users;
- role-specific queue without subject details in notifications;
- context assembled only after purpose authorization;
- no ordinary search, autocomplete, analytics, or dashboard exposure;
- saved protected drafts and changed-since-last-view;
- response-package generation inside protected boundary;
- minimized systemic signals exported only after approval;
- no sensitive content in usability telemetry.

---

# 12. Change, amendment, and supersession

When a regulator publishes an amendment, FAQ, extension, replacement, or withdrawal:

1. detect relationship to prior source;
2. compare provisions;
3. identify affected Requirements, controls, Programs, Matters, filings, and decisions;
4. prepare source-linked recommendations;
5. preserve prior interpretation;
6. route only changed or uncertain items for review;
7. invalidate stale conclusions where necessary.

Users should not reread and remap unchanged provisions.

---

# 13. User experience

## Authority Inbox

New publications, supervisory communications, requests, deadlines, authenticity warnings, and assigned reviewers.

## Source Review

Split view of source provision and proposed structured output, with exception highlighting.

## Regulatory Impact

Applicability, affected Programs and scope, current coverage, control gaps, actions, evidence, and readiness.

## Authority Case

Legal basis, subjects, directives, tasks, evidence, decisions, Response Package, transmission, acknowledgement, and history.

## Focused Respond

Direct task experience for records custodians, KYC teams, branches, vendors, and reviewers, showing only the authorized unresolved request.

---

# 14. Safety invariants

- every Requirement or directive retains exact source anchor;
- final interpretation and applicability require authority;
- exposure drafts cannot become final obligations;
- general model knowledge cannot establish regulatory fact;
- AI cannot fabricate deadline, penalty, exception, or legal basis;
- authority request does not imply guilt or reportability;
- protected subjects remain isolated;
- no external representation without approval;
- no high-impact account or customer action through convenience defaults;
- response and closure remain reconstructable.

---

# 15. Success measures

- active time from receipt to governed triage;
- time to approved applicability;
- percentage of context prefilled from inventories;
- manual fields eliminated;
- AI extraction edit and rejection rate;
- source-linked Requirement completeness;
- duplicate controls avoided;
- authority cases meeting deadline;
- response directives reconciled;
- time to resume complex case;
- protected-data leakage tests;
- time to produce audit or regulator lineage.

---

# 16. Definition of success

This capability succeeds when:

- a new regulatory publication becomes a source-linked, institution-specific implementation plan without manual register reconstruction;
- existing Programs, controls, systems, vendors, owners, and evidence are reused;
- users review material exceptions rather than blank pages;
- initial triage takes only a few minutes;
- complex legal and investigative work remains safe, resumable, and correctly authorized;
- authority responses are complete and traceable;
- amendments update only affected work;
- the bank can explain exactly what source was received, how it was interpreted, what was done, what was submitted, and what remains.