# AGENTS.md

This file defines mandatory rules for every human contributor, coding agent, design agent, reviewer, and automated change applied to ClearSight.

It exists to prevent the product from regressing into a conventional GRC portal, a generic AI chat interface, a dense enterprise dashboard, a graph demo, a collection of disconnected registers, or a functionally capable product that remains cumbersome to operate.

The words **MUST**, **MUST NOT**, **SHOULD**, and **SHOULD NOT** are normative.

---

# 1. Mission

ClearSight is a direct, AI-native continuous compliance and risk operating system built first for banks.

Every implementation decision must advance this outcome:

> **Help each stakeholder understand what the institution must do, what currently proves it, what changed or became uncertain, who must act, and whether the required outcome was achieved—with the minimum reasonable human effort.**

ClearSight remains a comprehensive GRC platform, but users MUST NOT be required to operate its internal architecture or reconstruct context across disconnected modules.

The product is optimized for:

- continuing compliance Programs;
- bounded Matters created by change or exception;
- routine work completed in a few clear steps;
- source integration and prefilled context;
- focused requests for unresolved facts;
- grounded AI recommendations and first drafts;
- review by exception;
- explicit source authority and data quality;
- accountable decisions and responses;
- verified outcomes;
- durable institutional memory.

It is not optimized for the number of forms, modules, dashboards, records, alerts, controls, graph nodes, AI messages, configuration options, or clicks it can expose.

---

# 2. Required reading and precedence

Before changing product behavior, domain semantics, architecture, interface structure, workflow, or component behavior, read:

1. [`README.md`](README.md)
2. [`docs/product/continuous-compliance-operating-model.md`](docs/product/continuous-compliance-operating-model.md)
3. [`docs/product/ease-of-use-standard.md`](docs/product/ease-of-use-standard.md)
4. [`docs/product/operating-model.md`](docs/product/operating-model.md)
5. [`docs/product/experience-principles.md`](docs/product/experience-principles.md)
6. [`docs/product/regulatory-and-enforcement-intelligence.md`](docs/product/regulatory-and-enforcement-intelligence.md)
7. [`docs/product/differentiation.md`](docs/product/differentiation.md)
8. relevant architecture documents
9. [`docs/implementation-plan.md`](docs/implementation-plan.md)
10. relevant acceptance tests

When documents conflict, apply:

1. safety, confidentiality, legal boundaries, and tenant isolation;
2. README product intent;
3. continuous-compliance and ease-of-use standards;
4. canonical operating-model semantics;
5. specialized product specifications;
6. experience principles;
7. this normative file;
8. architecture documents;
9. implementation sequencing;
10. acceptance detail.

An internal architecture mechanism MUST NOT override the simpler user-facing model or add avoidable human effort without an explicit product decision and synchronized documentation change.

---

# 3. Canonical product objects

## 3.1 Program

A long-lived body of continuing obligations, controls, evidence, reviews, exceptions, filings, and assurance.

Examples include NDPA, AML/CFT, PCI DSS, CBN cybersecurity, operational resilience, third-party assurance, RCSA, policy lifecycle, and regulatory returns.

A Program MUST NOT be implemented as a static control list with manually maintained status.

## 3.2 Matter

A bounded occurrence requiring assessment, evidence, decision, action, response, or verification.

Matter types include regulatory change, supervisory finding, authority request, risk situation, control gap, exception, incident, loss, breach, vendor deficiency, KRI breach, evidence contradiction, and failed verification.

A Risk Situation is a Matter subtype, not the only primary product object.

## 3.3 Shared primitives

Programs and Matters use the same governed primitives:

- Scope;
- Authority Source and Requirement;
- Exposure Pattern;
- Control Objective and Control Implementation;
- Claim and Evidence Contract;
- Observation and Evidence;
- Conclusion and Compliance State;
- Decision and Approval;
- Action and Response Package;
- Verification Contract;
- temporal history and audit.

Forms, imports, photos, tables, chat commands, and dashboards are interaction surfaces, not the domain model.

---

# 4. Ease-of-use invariants

## 4.1 Five-minute active-effort budget

Routine, authorized, well-scoped tasks SHOULD be completable in less than five minutes of active user effort.

Initial targets:

- routine focused request: median under three minutes and 90th percentile under five minutes;
- routine approval with complete context: median under two minutes;
- familiar recurring import using a saved mapping: under five minutes of active effort, excluding processing;
- assignment or redirection: under sixty seconds;
- executive understanding of one material item: under sixty seconds;
- return to an in-progress complex Matter: next action understood within thirty seconds.

A routine flow exceeding five minutes requires documented justification and usability review.

## 4.2 Complex-work checkpoint

When work cannot responsibly finish within five minutes, the user MUST be able to reach a clear, saved, correctly routed next state within five minutes.

The product MUST preserve context, draft state, completed steps, changes since last visit, blockers, and the recommended next action.

## 4.3 Prefill before asking

Before presenting an editable field, ClearSight MUST search approved sources for an existing value.

Approved sources may include institution profiles, inventories, directories, HR, IAM, ITSM, procurement, core systems, ROPA, BIA, policy repositories, Program evidence, prior submissions, APIs, managed imports, or approved spreadsheets.

Known values SHOULD be prefilled with source and freshness. Users MUST NOT repeatedly enter information the institution already maintains.

## 4.4 Existing evidence before requests

The system MUST search authorized existing evidence before contacting a person.

Requests MUST ask only for missing, stale, contradictory, or insufficient facts and MUST stop when the evidence need is satisfied or no longer relevant.

## 4.5 One clear next action

Every primary state MUST present one obvious next action written as a specific outcome, not a generic verb.

“View details” alone is never a valid handling path.

## 4.6 Minimize navigation

Routine work SHOULD remain in one coherent Program or Matter workspace.

Users MUST NOT navigate multiple module homepages to understand one obligation, case, finding, decision, or response.

## 4.7 Review by exception

Where policy permits, the interface SHOULD focus human reviewers on changed, low-confidence, contradictory, unsupported, material, or high-impact items rather than forcing full re-review.

## 4.8 Save and resume

Multi-step or interruptible workflows MUST support safe save and resume without requiring users to reconstruct prior context.

## 4.9 AI first drafts

Where approved and useful, AI SHOULD provide grounded first drafts for obligation extraction, mappings, evidence requests, summaries, control changes, actions, verification criteria, response indexes, and assignments.

AI MUST reduce blank-page work and MUST NOT create more review effort than it removes.

## 4.10 Stable interaction across integration maturity

The same workflow semantics MUST work with controlled lists, spreadsheets, managed imports, APIs, or event streams. Increasing automation MUST NOT force users to learn a different product.

---

# 5. Product invariants

## 5.1 Programs for continuity; Matters for change

Continuing obligations belong in Programs. Changes, gaps, findings, exceptions, incidents, cases, and required responses belong in Matters.

Do not create a separate truth system for every legacy register.

## 5.2 Banking language before GRC jargon

Primary language SHOULD begin with services, channels, branches, products, customers, accounts, merchants, assets, systems, vendors, requirements, evidence, actions, and outcomes.

Framework and control identifiers remain available to specialists but MUST NOT dominate routine tasks.

## 5.3 Scope before action

Active institution, legal entity, jurisdiction, Program or Matter, population, service, channel, and period MUST be clear before material action, approval, export, bulk change, or evidence submission.

## 5.4 Source authority before automated trust

A source is authoritative only for explicitly governed facts and scope.

Every source MUST expose owner, authoritative fields, limitations, scope, freshness, health, mapping version, and known data-quality issues.

Successful ingestion is not truth, completeness, or evidence sufficiency.

## 5.5 Evidence before confidence

AI confidence MUST NOT substitute for evidence sufficiency. Original sources, versions, contradictions, assumptions, coverage, and limitations remain visible.

## 5.6 Decisions before dashboards

A material indicator MUST lead to a specific evidence need, review, decision, action, response, or verification.

## 5.7 Verification before closure

Implementation, submission, and task completion remain distinct from verified outcome or authority acknowledgement.

Material remediation requires accepted outcome evidence. A response Matter requires reconciled directives, approval, transmission proof, and acknowledgement or documented response state.

## 5.8 Human authority for material judgment

Legal interpretation, applicability, material risk acceptance, regulatory representation, suspicious reporting, protected identity disclosure, account restriction, high-impact customer action, and other restricted decisions remain human-governed.

## 5.9 Institutional memory

Material records support point-in-time reconstruction. Corrections supersede rather than overwrite.

---

# 6. UI and component rules

## 6.1 Primary surfaces

Primary navigation is:

- Today;
- Programs;
- Work;
- Explore;
- Configure.

Focused Respond and Capture experiences may be delivered through direct links, mobile, portal, email, or enterprise messaging where policy permits.

Do not expose graph, Evidence Fabric, Decision Ledger, AI Operators, or internal bounded contexts as mandatory top-level navigation.

## 6.2 Program page

A Program page MUST prioritize:

- current position;
- material gaps and exceptions;
- evidence becoming stale;
- upcoming filings, reviews, and tests;
- recent changes;
- Matters requiring attention.

Do not default to a wall of controls.

## 6.3 Matter workspace

A Matter workspace SHOULD combine:

- summary and scope;
- evidence and source lineage;
- decisions and approvals;
- actions and dependencies;
- response or outcome verification;
- history.

## 6.4 Evidence request

A request MUST show why the recipient was selected, what is already known, what remains unresolved, acceptable response forms, estimated effort, deadline, sensitivity, and redirect/delegate/not-applicable options.

## 6.5 Forms

Known values are prefilled. Free text is reserved for explanation, not basic identity. Controlled values are sourced, searchable, and scoped. Final review shows exactly which assertions will be submitted.

## 6.6 Imports

Repeat imports MUST reuse approved mappings and focus review on changes, errors, duplicates, unresolved identifiers, and material variance.

## 6.7 Populations and bulk action

Population views expose denominators, exclusions, saved filters, next-unresolved navigation, keyboard efficiency, source freshness, and authorization-aware bulk actions.

## 6.8 AI recommendations

Recommendations MUST show sources, scope, assumptions, uncertainty, required authority, structured editable output, and safe alternatives.

Chat is optional and must not be required for standard operations.

---

# 7. Security, privacy, and authorization

- deny by default;
- enforce authorization server-side for reads, counts, search, graph traversal, exports, AI retrieval, bulk actions, and writes;
- resist inference through labels, counts, snippets, suggestions, embeddings, timing, or cache behavior;
- isolate protected authority and reporting cases;
- keep customer, account, legal, investigation, privilege, and reporter data purpose-bound;
- re-evaluate authorization at export and response-package generation;
- prevent logs from containing secrets, protected identities, or raw restricted evidence;
- make offline capture encrypted, bounded, explicit, and policy-controlled.

Fewer clicks MUST NOT weaken these controls.

---

# 8. AI implementation rules

AI acts as a governed compiler from messy inputs into proposed structured observations, requirements, mappings, questions, summaries, actions, or domain commands.

Requirements:

- exact source references and versions;
- structured, validated output;
- explicit versus inferred values;
- confidence and abstention;
- authorization and policy after model output;
- no direct persistence or unrestricted tool access;
- prompt-injection defenses;
- model independence and degraded mode;
- evaluation before release;
- measurable reduction in human effort.

General model knowledge MUST NOT establish material institutional or regulatory facts.

---

# 9. Testing and definition of done

Every meaningful feature requires tests for:

- domain invariants;
- authorization and wrong scope;
- source trust and data quality;
- evidence and contradiction;
- temporal reconstruction;
- degraded source and AI operation;
- accessibility and localization;
- timed first-use and repeat-use journeys;
- interruption and resume;
- active-effort budget;
- mobile or low-bandwidth behavior where applicable.

A feature is not complete until:

- the user outcome is clear;
- known information is reused;
- the routine path meets the five-minute target or has documented justification;
- a complex path reaches a safe saved next state within five minutes;
- there is one obvious next action;
- accessibility users do not face materially more effort;
- AI and integrations have safe fallbacks;
- governance, evidence, and audit remain complete;
- documentation and tests are synchronized.

---

# 10. Final review questions

Before merging, ask:

1. What does the user need to accomplish?
2. What does ClearSight already know?
3. Why is every editable field necessary?
4. Can an approved inventory or integration remove it?
5. Can AI provide a grounded first draft?
6. Can routine work finish within five minutes?
7. Can complex work reach a clear saved next state within five minutes?
8. Is there one obvious next action?
9. Can the user remain in one coherent workspace?
10. Are scope, evidence, uncertainty, authority, and consequence clear?
11. Does the workflow remain usable without AI or a live source?
12. Can the institution reconstruct the result later?

If the work is functionally possible but still cumbersome, it is not finished.