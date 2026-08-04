# ClearSight Continuous Compliance Operating Model

This document defines how ClearSight replaces disconnected registers, recurring questionnaires, reminder-driven work, and manually assembled reports with a continuously maintained compliance and risk operating model.

It specializes the broader [`operating-model.md`](operating-model.md) and works with [`regulatory-and-enforcement-intelligence.md`](regulatory-and-enforcement-intelligence.md).

The core product outcome is:

> **At any point, the institution should be able to see what applies, how it is intended to be satisfied, what current evidence supports that position, what has changed or become uncertain, who must act, and whether the required outcome was achieved.**

Continuous compliance does not mean that ClearSight autonomously declares the institution legally compliant. It means the institution maintains a current, evidence-backed, reviewable compliance state instead of reconstructing it periodically from spreadsheets and email.

---

# 1. The missing distinction: Programs and Matters

The prospective-bank samples reveal two fundamentally different kinds of work.

## 1.1 Programs are stable and recurring

A **Program** is a long-lived body of requirements, controls, evidence expectations, reviews, and reporting obligations.

Examples:

- Nigeria Data Protection Act and NDPC compliance;
- CBN cybersecurity and technology-risk requirements;
- AML/CFT programme;
- PCI DSS;
- ISO 27001 or ISO 22301;
- operational resilience;
- third-party assurance;
- RCSA;
- policy lifecycle;
- annual IT risk and control review;
- regulatory returns calendar.

A Program answers:

- what requirements apply;
- which legal entities, products, systems, vendors, branches, processes, customers, or data are in scope;
- how each requirement is satisfied;
- who owns and independently reviews it;
- what evidence is required;
- how frequently or under which trigger it must be refreshed;
- what exceptions are active;
- what is due next;
- and what current compliance state is defensible.

## 1.2 Matters are dynamic and require handling

A **Matter** is a bounded occurrence requiring assessment, evidence, decision, action, response, or verification.

Matter types include:

- regulatory change;
- supervisory finding;
- enforcement or authority request;
- risk situation;
- control gap;
- exception or waiver;
- audit finding;
- incident;
- operational loss;
- data breach;
- customer or conduct concern;
- vendor deficiency;
- overdue obligation;
- failed verification;
- policy expiry;
- evidence contradiction;
- KRI threshold breach.

A Matter answers:

- what happened or changed;
- why it matters;
- which Program, requirement, control, service, population, or authority request it affects;
- what is known and uncertain;
- what decision or response is required;
- who has authority;
- what actions are underway;
- and how closure or response will be verified.

## 1.3 Why this distinction matters

Without Programs, every recurring obligation becomes a ticket or risk record.

Without Matters, compliance becomes a static checklist that cannot handle new circulars, incidents, findings, exceptions, legal requests, or failed evidence.

ClearSight therefore operates as:

```text
Stable Programs
    continuously maintain requirements, controls, evidence and readiness

Dynamic Matters
    handle changes, gaps, findings, events, requests and exceptions

Shared institutional context
    connects both to legal entities, services, channels, branches, assets,
    customers, vendors, systems, data, policies, owners and authorities
```

---

# 2. The compliance chain

Every material compliance position should be traceable through one chain:

```text
Authority Source or Standard
→ Requirement
→ Applicability
→ Institutional Requirement
→ Control Objective
→ Control Implementation
→ Evidence Contract
→ Current Observations
→ Compliance State
→ Exception, Matter or Assurance Conclusion
```

## 2.1 Authority Source

The original law, circular, regulation, guideline, standard, contract, licence condition, policy, court instrument, or approved internal interpretation.

The original source and version remain preserved.

## 2.2 Requirement

A normalized statement describing what an actor must, must not, may, or is expected to do.

Every Requirement must retain exact source lineage and effective dates.

## 2.3 Applicability

The governed conclusion about where the Requirement applies.

Applicability may depend on:

- legal entity and licence;
- jurisdiction;
- product or activity;
- customer category;
- channel;
- processing type;
- asset, system or data type;
- vendor relationship;
- threshold or classification;
- effective period.

Possible states:

- applicable;
- partially applicable;
- not applicable;
- potentially applicable—information required;
- applies from a future date;
- superseded or expired.

AI may propose applicability. An authorized compliance or legal reviewer approves material interpretations.

## 2.4 Institutional Requirement

The bank-approved, operational restatement of the external Requirement in its own scope and terminology.

Example:

> Every new high-risk personal-data processing activity must complete an approved DPIA before production go-live.

## 2.5 Control Objective

The outcome that must be achieved.

Example:

> High-risk processing is identified, assessed, approved, and remediated before commencement.

## 2.6 Control Implementation

The actual mechanism used in a defined scope.

Examples:

- project-intake screening;
- DPO approval gate;
- vendor-onboarding privacy review;
- automated device limit;
- access-review workflow;
- breach notification procedure;
- retention job;
- periodic control review.

A single objective may have multiple implementations across entities, products, systems, branches, or vendors.

## 2.7 Evidence Contract

The continuous definition of what proves the requirement and control are satisfied.

An Evidence Contract combines:

- exact Claims;
- required populations and periods;
- acceptable source types;
- source authority and limitations;
- freshness;
- coverage;
- independence;
- contradiction rules;
- required review;
- refresh schedule or trigger;
- failure and escalation behavior.

## 2.8 Compliance State

A versioned conclusion that keeps distinct dimensions visible.

Recommended dimensions:

- source interpretation;
- applicability;
- control design;
- implementation;
- evidence sufficiency;
- operating effectiveness;
- exception state;
- assurance state;
- deadline or reporting state;
- source and data quality.

Do not reduce this to a single unexplained percentage.

A concise overall state may be shown:

- compliant with sufficient current evidence;
- compliant with qualification;
- at risk;
- gap identified;
- evidence insufficient;
- implementation pending;
- overdue;
- under review;
- not applicable;
- unknown because required information is missing.

The underlying dimensions and lineage remain inspectable.

---

# 3. Continuous compliance is trigger-driven

A Program should not rely only on recurring reminders.

Each Requirement and Evidence Contract defines triggers that can change state or create a Matter.

## 3.1 Calendar triggers

Examples:

- annual NDPC Compliance Audit Return;
- quarterly board reporting;
- monthly regulatory return;
- certificate expiry;
- policy review;
- periodic access review;
- scheduled DR test;
- vendor reassessment.

## 3.2 Change triggers

Examples:

- new regulatory publication;
- amendment or FAQ;
- new product or project;
- process change;
- new vendor or contract renewal;
- new system or cloud deployment;
- new branch or channel;
- changed customer population;
- changed data processing activity;
- changed control configuration.

## 3.3 Event triggers

Examples:

- data breach;
- control failure;
- KRI threshold breach;
- customer complaint concentration;
- transaction-monitoring alert;
- stale evidence;
- source integration outage;
- vendor certificate expiry;
- failed control test;
- incident or loss;
- law-enforcement request;
- account or subject added to a governed case.

## 3.4 Evidence triggers

Examples:

- observation expires;
- source is revoked;
- current population changes;
- two sources contradict;
- sampling coverage becomes inadequate;
- source freshness falls below policy;
- an assurance conclusion is challenged;
- a verification period fails.

The result is exception-based operation: people are contacted when a meaningful fact, decision, or proof is missing—not merely because a calendar campaign began.

---

# 4. How legacy workflows become simpler

Legacy registers remain supported as imports, exports, worklists, and familiar views. They no longer maintain separate copies of institutional truth.

| Legacy workflow | ClearSight representation | What becomes easier |
|---|---|---|
| Compliance register | Program Requirements and Compliance State view | Source lineage, applicability, current proof, deadlines and amendments remain connected |
| IT or operational risk register | Risk Matters linked to assets, services, controls and evidence | No repeated re-entry of owners, assets, controls and actions |
| Exception register | Exception Matter with authority, conditions, expiry and verification | Redirects, approvals and closure are explicit rather than hidden in comments |
| Annual workplan | Scheduled Review Activities | Reviews reuse existing asset, control, vendor and requirement records |
| RCSA | Periodic or trigger-based control review | Known facts are prefilled and only unresolved judgments are requested |
| KRI workbook | Indicators calculated from underlying observations and Matters | Every metric has scope, period, denominator and drill-down lineage |
| BIA register | Critical service and dependency context | RTO, RPO, applications, vendors and resources support resilience, risk and regulatory workflows |
| Loss register | Loss Event linked to incident, root cause, recovery, action and control failure | Recovery and recurrence no longer require manual reconciliation |
| Vendor register | Third-party profile, service relationship, requirements, evidence and Matters | Certificates and assessments can be reused across bank services |
| Policy register | Governed policy lifecycle mapped to Requirements and Controls | Policy changes show which requirements and controls are affected |
| Certification tracker | Program milestone and Evidence Contract | Expiry, gaps, testing and recertification remain one connected workflow |
| Dashboard pack | Derived role-specific view | Reports are generated from governed records rather than manually assembled copies |
| Authority-request KRI | Aggregate over protected Authority Request Cases | Counts remain traceable without exposing case subjects |

## 4.1 Capture once, reuse safely

A source observation, owner, asset, vendor, service, control, evidence item, action, or decision is captured once and reused through authorized relationships.

Reuse still respects:

- purpose;
- confidentiality;
- legal entity;
- time period;
- source authority;
- evidence scope;
- independence;
- current access policy.

## 4.2 One event can update many views

Example: a vendor PCI DSS certificate expires.

ClearSight can update:

- the vendor profile;
- relevant Evidence Contracts;
- affected payment services;
- third-party compliance state;
- active risk Matters;
- upcoming committee brief;
- recertification workplan;
- KRI or concentration view.

The certificate expiry is one observation, not six manually copied spreadsheet rows.

---

# 5. Continuous NDPA and NDPC compliance

A privacy Program should continuously maintain the institution’s position across requirements such as:

- registration and classification;
- DPO governance;
- annual compliance audit and filing;
- ROPA;
- lawful basis and consent;
- DPIA;
- security controls;
- breach management;
- data-subject rights;
- retention and deletion;
- processor and vendor governance;
- cross-border transfers;
- digital-channel notices and cookies;
- emerging technology and automated decisions.

## 5.1 ROPA should be assembled, not repeatedly requested

ClearSight should prefill processing activities from:

- application and asset catalogues;
- project and change records;
- vendor records;
- data inventories;
- customer journeys;
- privacy notices;
- existing ROPA entries.

Department owners receive focused requests only for unresolved fields such as purpose, lawful basis, recipient, retention, transfer, or data-subject category.

A change to an application, vendor, process, dataset, jurisdiction, or purpose can reopen only the affected ROPA section.

## 5.2 DPIA should be event-driven

A new project, product, process change, vendor, AI system, cross-border transfer, or sensitive-data use triggers a short screening.

```text
Project or change created
→ existing context prefilled
→ targeted privacy screening
→ DPO determines whether full DPIA is required
→ full DPIA and remediation where needed
→ approval condition before go-live
→ post-deployment verification
```

The project owner should not navigate a separate privacy register. The privacy decision remains attached to the project or change.

## 5.3 Breach obligations should create timed Matters

A suspected personal-data breach creates a protected Breach Matter with:

- awareness time;
- affected systems and data;
- data-subject population;
- risk assessment;
- legal and regulatory notification decision;
- applicable deadline clock;
- customer communication decision;
- evidence and response package;
- remediation and verification.

The system must distinguish detection, awareness, reportability decision, notification, acknowledgement and closure.

## 5.4 Annual filing should be continuously prepared

Instead of assembling the annual compliance return at year end, ClearSight accumulates approved evidence and unresolved gaps throughout the year.

The filing workspace shows:

- requirements included;
- evidence current and missing;
- DPIAs and incidents to disclose;
- exceptions;
- reviewer and signatory state;
- source lineage;
- filing proof and acknowledgement.

## 5.5 Vendor privacy compliance should reuse third-party context

Vendor onboarding and renewal can automatically identify:

- whether personal data is processed;
- processor or controller role;
- DPA requirement;
- cross-border transfer;
- security evidence;
- subprocessors;
- retention and deletion obligations;
- privacy review and approval conditions.

The same vendor and evidence remain connected to technology, operational, resilience and privacy Programs.

---

# 6. Regulatory change and external authority work

External authority work uses the same Programs and Matters.

## 6.1 New regulation or circular

```text
Authority Source
→ provision-level extraction
→ candidate Requirements
→ authorized interpretation and applicability
→ affected Programs, Controls and Evidence Contracts
→ implementation Matters and actions
→ readiness and response
→ continuing monitoring
```

An approved new Requirement becomes part of the relevant Program rather than remaining a one-time project row.

## 6.2 Supervisory finding

A supervisory finding becomes a Matter linked to:

- exact source provision or finding;
- affected Requirements and Controls;
- management response;
- remediation milestones;
- evidence expectations;
- authority response deadline;
- verification and formal closure.

## 6.3 EFCC or other authority request

A protected Authority Request Case records:

- authentic source and legal basis;
- subjects, accounts, transactions, documents and period;
- disclosure and action authority;
- KYC, address, records, AML, fraud, branch, legal or technology tasks;
- response package;
- review, signatory and transmission;
- acknowledgement and retention.

An authority request may trigger an internal suspicious-activity assessment. It must not automatically establish suspicion, authorize an account restriction, or file an external report without the required institutional decision.

Aggregate authority-request KRIs are derived from the underlying protected case population.

---

# 7. User experience

ClearSight should expose five primary product surfaces.

## 7.1 Today

A role-specific attention brief:

- Programs at risk or approaching a deadline;
- new or changed authority communications;
- Matters requiring the user’s decision;
- evidence gaps requiring intervention;
- overdue actions;
- failed verification;
- significant source degradation;
- safely automated changes worth noting.

## 7.2 Programs

Stable workspaces for NDPA, AML/CFT, CBN cybersecurity, PCI DSS, ISO, operational resilience, RCSA, vendor assurance and other ongoing obligations.

A Program shows:

- current requirements and applicability;
- controls and owners;
- evidence state;
- active exceptions and Matters;
- upcoming reviews and filings;
- changes since last review;
- readiness by meaningful dimension;
- complete source-to-state lineage.

## 7.3 Work

A queue and workspace for Matters, cases, findings, exceptions, incidents, actions, reviews and approvals.

The user sees familiar work types and one clear next action rather than navigating separate GRC modules.

## 7.4 Explore

Authorized investigation across requirements, policies, controls, services, channels, branches, assets, customers, vendors, evidence, Matters, decisions, incidents, losses and history.

## 7.5 Configure

Restricted configuration for institution structure, sources, Programs, jurisdiction and channel packs, controlled vocabularies, evidence contracts, thresholds, authority, retention, access and automation policy.

## 7.6 Respond and Capture

Evidence respondents, branches, vendors, customers and external reporters receive lightweight contextual journeys rather than the full application shell.

They see:

- why information is needed;
- what is already known;
- the exact unresolved question;
- acceptable proof;
- sensitivity and deadline;
- answer, upload, redirect, challenge or report-conflict options.

---

# 8. Automation and AI

AI acts as a governed compiler and assistant across the compliance chain.

It may:

- classify authority documents;
- extract candidate requirements and directives;
- compare versions;
- propose applicability and control mappings;
- extract spreadsheet and media observations;
- identify duplicate or contradictory records;
- draft evidence requests;
- summarize Program and Matter state;
- draft implementation plans and response packages;
- propose test procedures and verification criteria.

It may not silently:

- publish legal interpretation;
- declare final applicability;
- mark a material Requirement compliant;
- close a major Matter;
- accept risk;
- file a regulatory or suspicious report;
- restrict an account;
- disclose protected customer information;
- represent the institution externally.

Material AI output requires source lineage, structured validation, authorization, confidence and abstention behavior, human review where required, and immutable audit.

---

# 9. Success measures

## Continuous compliance

- Requirements with approved source lineage;
- time from source publication to applicability decision;
- Requirements with current sufficient evidence;
- evidence reused rather than recollected;
- stale or contradictory compliance positions;
- time spent preparing filings, audits and examinations;
- Programs maintained without broad recurring questionnaires.

## Legacy-workflow reduction

- spreadsheets retired or converted to governed views;
- duplicate records and manual reconciliations removed;
- email and comment-based ownership transfers eliminated;
- manual report assembly hours removed;
- work items generated automatically from triggers;
- register views generated from shared objects.

## Matter handling

- time from trigger to accountable owner;
- overdue authority responses;
- findings closed without verified outcome;
- exception expiry and revalidation;
- response packages returned for missing evidence;
- repeat incidents, losses, complaints and findings.

## Trust

- source authenticity and provision coverage;
- unsupported interpretations;
- unresolved applicability;
- unauthorized actions prevented;
- protected-case access violations;
- point-in-time reconstruction time;
- independent assurance challenges and overrides.

---

# 10. Product invariants

1. **Programs for continuing obligations; Matters for dynamic work.**
2. **One authoritative object, many role-specific views.**
3. **Original authority source before register summary.**
4. **Applicability before implementation.**
5. **Control implementation before compliance claim.**
6. **Current evidence before status.**
7. **Triggers and exceptions before blanket reminders.**
8. **Existing information before human requests.**
9. **Structured ownership before comment-based routing.**
10. **Decision authority before external action.**
11. **Verification before closure.**
12. **Aggregate metrics must drill down to governed records.**
13. **AI drafts and compiles; authorized humans judge.**
14. **History is superseded, never silently rewritten.**

---

# 11. Definition of success

The operating model succeeds when the institution can answer, without rebuilding the answer from spreadsheets:

> What requirements apply to this part of the bank, how are they satisfied, what current evidence proves it, what is changing or uncertain, what work requires attention, and can we defensibly show the complete history to management, an auditor, a regulator, or an enforcement authority?

ClearSight should make continuing compliance quieter, more current, more evidence-based, and less dependent on recurring manual campaigns—while making unusual regulatory, supervisory and enforcement work faster and more governable.
