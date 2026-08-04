# ClearSight Experience Principles

This document defines the canonical product experience, information architecture, visual language, interaction rules, and visual non-regression requirements for ClearSight.

The target is a 2030-quality bank GRC interface: not decorative science fiction, but a system that continuously assembles context, keeps compliance proof current, reduces repetitive work, and makes exceptional work easy to understand and govern.

This document conforms to:

- [`continuous-compliance-operating-model.md`](continuous-compliance-operating-model.md)
- [`operating-model.md`](operating-model.md)
- [`regulatory-and-enforcement-intelligence.md`](regulatory-and-enforcement-intelligence.md)

Internal architecture must not become user-interface architecture.

---

# 1. Experience objective

A ClearSight user should feel that:

- the system understands the relevant institution, legal entity, licence, jurisdiction, Program, channel, service, branch, population, and period;
- continuing obligations are already organized and current rather than rebuilt for each review;
- changed, overdue, contradictory, or exceptional work is separated into direct Matters;
- known information is prefilled;
- missing or weak proof is explicit;
- the next accountable action is clear;
- AI reduces assembly work without becoming another product to operate;
- source quality and freshness are visible;
- and every conclusion, response, and decision remains reconstructable.

The core interaction grammars are:

### Program

> **Understand obligation → inspect coverage → refresh proof → handle gaps → assure**

### Matter

> **Understand change → gather evidence → decide or respond → act → verify**

### Evidence respondent

> **See context → confirm or provide only missing facts → review → submit**

### Executive

> **Brief → explain → decide → verify**

---

# 2. Product personality

ClearSight should feel:

- **calm** — routine compliance does not look like a permanent emergency;
- **direct** — current state and next action are obvious;
- **precise** — scope, source, period, owner, authority, and confidence are explicit;
- **relatable** — obligations, channels, branches, customers, assets, vendors, filings, and cases appear before abstract GRC architecture;
- **institutional** — appropriate for branch operations, compliance teams, risk committees, legal review, internal audit, boards, and regulators;
- **intelligent** — known context is assembled automatically;
- **premium** — refined typography, spacing, motion, and detail;
- **defensible** — source lineage and reasoning are close at hand;
- **restrained** — color, glass, glow, and depth communicate meaning rather than decoration.

ClearSight must not feel:

- like a generic admin template;
- like a security alert console;
- like a consumer banking app;
- like a social feed;
- like a gamified compliance product;
- like a neon cyberpunk concept;
- like a spreadsheet wrapped in cards;
- like a chatbot over disconnected modules;
- or like every continuing obligation is an overdue task.

---

# 3. Experience architecture

Most authenticated users operate through five primary surfaces.

## 3.1 Today

Today is a role-specific attention brief, not a dashboard catalogue.

It should show only items where attention changes the outcome:

- Program requirements or filings approaching a decision or deadline;
- evidence that became stale, contradictory, insufficient, or unavailable;
- Matters requiring the user’s authority, evidence, review, response, or verification;
- appetite, compliance, KRI, or operational thresholds that changed materially;
- actions likely to miss a deadline;
- failed or indeterminate verification;
- external-authority communications requiring triage;
- significant changes safely automated.

The default executive view should usually contain three to seven cards.

Today must support:

- an explicit review period;
- clear Program or Matter type;
- acknowledgement without suppressing the underlying record;
- delegation within authority;
- saved role views without turning customization into the product;
- expanded analyst monitoring;
- and a true no-attention-required state distinct from no data, no access, and not assessed.

## 3.2 Programs

Programs is the continuing-compliance surface.

A Program page should make a large obligation estate feel stable and manageable rather than turn every requirement into a card.

### Overview

- Program purpose and governing authorities or standards;
- active scope and entities;
- responsible executive, Program owner, control owners, and independent reviewers;
- current compliance dimensions;
- upcoming filing, review, test, and certification milestones;
- evidence at risk;
- open Matters and exceptions;
- source or data-quality concerns;
- important changes since the last approved review.

### Requirements

Use a structured table or grouped outline showing:

- source-linked Requirement;
- applicability and scope;
- effective period;
- mapped control objectives and implementations;
- evidence state;
- owner and reviewer;
- exceptions or open Matters;
- next review or filing date.

Requirements should be groupable by authority, topic, legal entity, channel, business service, deadline, control, or evidence state.

### Controls

Show control objectives separately from scoped implementations.

Support:

- implementation comparison across entities or systems;
- design and operating-effectiveness state;
- owners and performers;
- automation and source coverage;
- evidence contracts;
- exceptions;
- related incidents, findings, and Matters.

### Evidence

Show:

- exact Claims being proven;
- Evidence Contracts;
- source authority and limitations;
- freshness, population, coverage, independence, and contradiction;
- original and derived evidence;
- upcoming refresh;
- evidence reused across authorized Requirements;
- gaps that require focused capture.

### Calendar

A unified operating calendar for:

- filings and returns;
- periodic reviews;
- control tests;
- policy review;
- certification expiry;
- vendor reassessment;
- RCSA and BIA refresh;
- committee reporting;
- assurance activities.

Calendar items must derive from Program records rather than exist as disconnected reminders.

### Matters and exceptions

A Program should show linked Matters by type, materiality, owner, due date, and verification state without turning the Program itself into a case queue.

### Assurance

Support first-, second-, and third-line conclusions independently over shared source evidence.

Show:

- review and test scope;
- sample population and selection provenance;
- assurance conclusion;
- challenge and disagreement;
- findings;
- included and excluded evidence;
- sign-off and point-in-time freeze.

### History

- source and Requirement versions;
- applicability changes;
- control changes;
- evidence and conclusion changes;
- filings;
- exceptions;
- approvals;
- previous assurance states.

## 3.3 Work

Work is the operating queue for Matters, evidence requests, actions, reviews, approvals, and responses.

It should support:

- My work;
- team work;
- delegated work;
- Matters by type;
- evidence requests;
- approvals and challenges;
- actions and remediation;
- authority responses;
- verification due;
- overdue or blocked work.

Worklists use tables or grouped queues, not walls of cards.

A Matter workspace should contain:

### Summary

- what happened or changed;
- why it matters;
- Matter type and current state;
- active scope and period;
- linked Program, Requirement, control, service, customer, account, asset, vendor, or event;
- owner, authority, deadline, and next handling step.

### Context

- affected institutional relationships;
- applicable Requirements, controls, policies, appetite, and thresholds;
- related Matters, incidents, losses, complaints, or previous decisions;
- source and data-quality state.

### Evidence

- required Claims;
- known, missing, stale, contradictory, and excluded evidence;
- source authority and limitations;
- population and period;
- assumptions and current Conclusion.

### Decision or response

Depending on Matter type:

- options and trade-offs;
- applicability or legal interpretation;
- approval or challenge;
- management response;
- exception conditions;
- authority response package;
- reportability or disclosure decision;
- signatory and submission channel.

### Actions

- tasks and dependencies;
- owner and performers;
- external execution state;
- blockers and escalation;
- implementation evidence.

### Outcome

- expected outcome;
- baseline and measurement;
- observation period;
- current result;
- success and failure thresholds;
- acceptance authority;
- response acknowledgement;
- verified, ineffective, indeterminate, awaiting verification, or continuing-monitoring state.

### History

- versions;
- communications;
- source changes;
- decisions, overrides, and dissent;
- action and response history;
- verification outcomes;
- point-in-time reconstruction.

## 3.4 Explore

Explore is an authorized institutional inquiry surface.

It includes:

- Programs and Requirements;
- policies and controls;
- Matters and risk situations;
- channels, services, branches, customers, accounts, merchants, assets, systems, vendors, projects, and data-processing activities;
- sources and data quality;
- Claims, Observations, Evidence, Decisions, Actions, filings, responses, and outcomes;
- relationships, trends, and historical reconstruction.

Explore is not a collection of module homepages.

Prefer:

- scoped search;
- readable hierarchy;
- relationship paths;
- dependency lists;
- population tables;
- timeline;
- affected-scope summaries;
- progressive expansion.

Use node graphs only where spatial relationships materially improve comprehension.

## 3.5 Configure

Configure is a restricted administrative surface for:

- institution, entity, licence, jurisdiction, service, channel, and branch structure;
- Program templates and institution Programs;
- Requirement, applicability, and control vocabulary;
- source registry and mapping;
- Evidence Contracts and capture templates;
- calendars, triggers, thresholds, KRIs, appetite, and escalation;
- authority, segregation of duties, and access;
- retention, legal hold, and privacy;
- AI and automation permissions;
- integrations and deployment policy.

Ordinary users should rarely enter Configure.

Configuration must be grouped by the institutional outcome it governs, not by implementation component.

## 3.6 Contextual Capture and Respond

Capture and Respond are lightweight experiences launched from Today, Programs, Work, secure links, messaging, mobile notifications, or external portals.

They are optimized for:

- one focused evidence question;
- structured confirmation;
- media or document capture;
- spreadsheet or population submission;
- vendor evidence;
- branch field work;
- customer or protected reporting;
- authority-response contribution.

They are not mandatory permanent navigation for all users.

---

# 4. Mandatory context anchoring

Wrong-scope action is a material governance failure.

Every material workspace must make available:

- institution or tenant;
- legal entity and licence where relevant;
- country or jurisdiction;
- Program or Matter;
- channel, service, branch, account, vendor, population, or processing activity;
- effective period and record time;
- user role or delegated authority;
- data freshness and source health.

Use a compact context header or breadcrumb. It must remain visible before approval, filing, export, response, bulk action, or evidence submission.

Context switching must:

- clearly identify the new scope;
- preserve or deliberately reset filters;
- prevent cross-entity action;
- warn when drafts or selections belong to another scope;
- never rely on subtle color alone.

---

# 5. Compliance-state design

Compliance must not be reduced to one unexplained score.

The UI should keep these dimensions distinguishable:

- source interpretation;
- applicability;
- control design;
- implementation;
- evidence sufficiency;
- operating effectiveness;
- exception or waiver;
- assurance;
- filing or deadline;
- source and data quality.

A concise summary may use states such as:

- current;
- current with exception;
- at risk;
- gap identified;
- evidence insufficient;
- implementation pending;
- overdue;
- under review;
- not applicable;
- unknown.

Every summary state must reveal its dimensions and rationale.

Avoid averaging unrelated dimensions into a precise percentage.

---

# 6. Core interaction patterns

## 6.1 Attention card

Attention cards are used only for a deliberately small queue.

They contain:

1. direct issue or obligation;
2. why attention is needed now;
3. affected scope;
4. evidence or compliance state;
5. required handling;
6. owner or authority;
7. due time.

Rules:

- one dominant message;
- one primary action;
- no more than two secondary actions while collapsed;
- no unexplained score;
- no generic view-details-only path;
- no green for submission or implementation alone;
- no hidden material context available only on hover.

## 6.2 Program requirement table

Large Program estates require compact, accessible tables.

Support:

- hierarchical grouping;
- sticky requirement and scope identifiers;
- source-provision link;
- applicability;
- control and evidence state;
- owner and reviewer;
- next due date;
- linked Matter count;
- filters and saved views;
- column and density controls;
- keyboard navigation;
- export with manifest.

Do not render every Requirement as a card.

## 6.3 Evidence request

An evidence request should feel like a short contextual work message.

It includes:

- why the recipient is being asked;
- what is already known;
- the smallest unresolved question;
- relevant Requirement, control, object, or population in plain language;
- acceptable response forms;
- estimated effort;
- deadline and consequence;
- confidentiality;
- redirect, partial response, not applicable, and sensitivity options.

Avoid broad free-text prompts, control IDs as primary language, and repeated requests that could be deduplicated.

## 6.4 Population worklist

For ATMs, POS terminals, accounts, merchants, branches, vendors, controls, incidents, ROPA activities, or exceptions, support:

- total and filtered population;
- explicit denominator and exclusions;
- resolved, unresolved, contradictory, stale, not applicable, excluded, and unauthorized states;
- search and saved filters;
- sticky identifiers and headers;
- source and freshness state;
- inline comparison where safe;
- virtualization or pagination;
- accessible selection summary;
- export with scope and manifest.

## 6.5 Reconciliation

Reconciliation views distinguish:

- matched;
- provisionally matched;
- unresolved;
- contradictory;
- duplicate;
- rejected;
- superseded.

Show source records side by side, normalized identifiers, proposed match reason, affected downstream records, confidence dimensions, and merge/unmerge history.

## 6.6 Spreadsheet import

Spreadsheet and CSV import are first-class workflows:

1. file and sheet selection;
2. Source Profile and purpose;
3. column detection and mapping;
4. preview;
5. type and required-field validation;
6. identifier matching;
7. duplicate and conflict analysis;
8. scope confirmation;
9. import summary;
10. reconciliation queue and rollback reference.

The UI must distinguish uploaded, parsed, mapped, accepted as Observation, reconciled, and sufficient as Evidence.

## 6.7 Photo and scan capture

Guide users toward visible, verifiable attributes. Show framing, blur, glare, crop, and readability guidance; disclose metadata collection; minimize background people or data; preserve originals; show extracted fields and regions; require confirmation where necessary; state what the media cannot establish.

Never present “AI verified secure” from a general photograph.

## 6.8 Evidence sufficiency

Evidence quality should remain inspectable across:

- relevance;
- authenticity;
- coverage;
- freshness;
- independence;
- completeness;
- consistency;
- reliability;
- traceability.

Use a concise state with expandable dimensions, source references, policy requirements, and unresolved contradiction.

## 6.9 Contradiction

Show disputed Claim, conflicting sources side by side, authority and limitations, periods, population mismatch, affected conclusions and responses, resolver, and time sensitivity.

The system must not silently choose the source that produces the most favorable status.

## 6.10 Decision and approval

A material review must show exact scope and period, evidence and contradiction, options and trade-offs, affected customers or obligations, authority and segregation of duties, expiry, irreversible effects, and verification.

Approval cannot be a context-free button.

## 6.11 Verification and closure

Visually distinguish:

- planned;
- authorized;
- in progress;
- implemented;
- response transmitted;
- awaiting acknowledgement;
- awaiting verification;
- verified effective;
- verified ineffective;
- indeterminate;
- closed with accepted evidence.

Do not use green during an incomplete observation period.

## 6.12 Natural-language command surface

The command surface is global but secondary.

It may support inquiry, navigation, comparison, drafting, summarization, simulation, and proposed actions.

Responses must include active scope, time period, sources, uncertainty, contradiction, and safe structured next actions.

Any side effect transitions into the normal review surface.

---

# 7. Regulatory and authority experiences

## 7.1 Authority Inbox

Show:

- source type;
- authority and issuing office;
- publication or receipt date;
- effective or response deadline;
- authenticity state;
- confidentiality;
- likely work class;
- assigned reviewer;
- urgency based on actual deadline, not decorative alarm.

## 7.2 Regulatory change review

Use a split workspace:

```text
Original source provision | Proposed Requirement or interpretation
```

Also show:

- source version and exact location;
- definitions and dependencies;
- applicability proposal;
- affected Programs, entities, services, systems, vendors, policies, and controls;
- existing coverage and gaps;
- proposed implementation Matters;
- reviewer changes and approval history.

The primary action is **Create or update draft compliance programme**, never **Mark compliant**.

## 7.3 Supervisory Matter

Show finding, authority expectation, management response, commitments, milestones, evidence, committee oversight, response package, and effectiveness verification.

## 7.4 Authority Request Case

Use a protected workspace showing only authorized case information:

- verified source and legal instrument;
- subjects and match state;
- requested periods and records;
- directives;
- disclosure and action authority;
- legal hold;
- KYC, address, records, AML, fraud, branch, technology, or legal tasks;
- response package;
- signatory and transmission;
- acknowledgement and continuing monitoring.

Do not expose protected subjects in ordinary Today counts, search suggestions, analytics, or Program summaries.

---

# 8. NDPA Program experience

The NDPA Program should avoid presenting privacy as one checklist.

Use domain views for:

- registration and classification;
- DPO governance;
- ROPA and lawful basis;
- DPIA and project/vendor screening;
- consent and notices;
- retention and deletion;
- data-subject rights;
- vendor and processor management;
- cross-border transfers;
- breach Matters;
- annual filing and assurance.

### ROPA

Use a processing-activity worklist with application, process, purpose, data categories, lawful basis, subjects, recipients, vendor, location, retention, transfer, owner, last confirmation, and evidence state.

Prefill known application, vendor, project, and data-flow information. Ask departments only for unresolved business meaning.

### DPIA

Present a change-triggered workflow with prefilled project, system, vendor, data, automation, and jurisdiction context; screening decision; full DPIA if required; remediation; DPO approval; go-live gate; and post-deployment verification.

### Annual filing

Show filing requirements, current evidence readiness, unresolved exceptions, included and excluded records, approvers, signatory, submission, acknowledgement, and point-in-time package freeze.

---

# 9. Visual language

## 9.1 Surface hierarchy

Use a small surface system:

1. canvas;
2. primary workspace;
3. raised focus surface;
4. protected surface;
5. transient overlay.

Do not create a unique card style for every record type.

## 9.2 Glass and glow

Glass is appropriate for bounded focus layers, command overlays, relationship explanations, and protected-review context. It is inappropriate for every card, dense tables, long text, evidence documents, or low-power mobile use.

Glow may indicate selected analysis or current focus, not severity by itself.

## 9.3 Semantic color

- **Cyan:** new intelligence, context, or relationship.
- **Violet:** governance, control, decision, or approved automation.
- **Coral/red:** material exposure, failed outcome, breach, or serious gap.
- **Amber:** uncertainty, stale evidence, contradiction, pending review, or approaching threshold.
- **Green:** sufficiently evidenced and accepted current state or verified outcome.
- **Neutral:** informational, unchanged, historical, or unassessed.

Status must never depend on color alone.

## 9.4 Typography and numbers

Use a modern neutral sans-serif, clear hierarchy, stable multilingual line height, tabular numerals, readable line lengths, and no tiny low-contrast metadata.

Every metric must include units, period, scope, source, denominator where relevant, and explanation of uncertainty.

## 9.5 Density

Support:

- comfortable density for executives and focused reviews;
- compact density for compliance populations, control estates, imports, and reconciliation.

Density changes spacing and row height, not information semantics or accessibility.

## 9.6 Charts

Prefer charts only for specific questions:

- compliance movement over time;
- evidence coverage;
- filing readiness dimensions;
- KRI movement and thresholds;
- control and exception trends;
- source freshness;
- concentration and dependency;
- projected versus observed outcomes.

Every chart requires title, units, period, source, denominator, accessible summary, and uncertainty explanation.

---

# 10. Responsive behavior

## Desktop

Optimized for Program management, population reconciliation, regulatory interpretation, Matter review, decision work, and evidence inspection.

## Tablet

Optimized for executive review, committee meetings, approvals, Program status, and selected evidence inspection.

## Mobile

Optimized for focused capture, short review, approval with sufficient context, branch work, incident updates, vendor response, and protected reporting.

Do not squeeze full Program tables or graph exploration into mobile.

## Large display

Boardroom modes emphasize Program movement, significant Matters, major decisions, evidence quality, assurance, and management accountability. They must not become walls of blinking metrics.

---

# 11. State design

Every applicable feature must deliberately design:

- loading;
- partially available;
- empty;
- current;
- no attention required;
- no data;
- not assessed;
- not applicable;
- unknown applicability;
- insufficient evidence;
- contradictory evidence;
- pending review or approval;
- delegated;
- blocked;
- executing;
- implemented;
- response transmitted;
- awaiting acknowledgement;
- awaiting verification;
- verified effective;
- verified ineffective;
- stale;
- superseded;
- unauthorized;
- offline;
- integration degraded;
- AI unavailable.

“No data,” “no issue,” “not applicable,” and “not authorized” must be distinct.

---

# 12. Accessibility, localization, and performance

ClearSight must meet WCAG 2.2 AA at minimum.

Requirements include keyboard operation, visible focus, screen-reader state, meaningful headings, non-color status, chart alternatives, target size, reduced motion, 200% zoom, error prevention, and accessible protected-reporting and evidence journeys.

Support local currencies, number formats, time zones, date formats, long translated labels, and multilingual evidence. Avoid hard-coded text widths and ambiguous date presentation.

Deterministic content must render before AI. Layout must remain stable. Long lists use pagination or virtualization. Uploads are resumable. Avoid excessive blur, animated gradients, continuously animated graph edges, and high GPU cost.

---

# 13. Golden screens

Maintain design references and visual regression for at least:

1. Today executive brief
2. Today compliance-owner brief
3. Program overview
4. Program requirement table
5. Control implementations across scopes
6. Evidence Contract and sufficiency
7. Program calendar
8. Program exceptions and Matters
9. Assurance review and sign-off
10. Work queue
11. Generic Matter workspace
12. Risk Situation Matter
13. Regulatory change split review
14. Supervisory Matter
15. Protected Authority Request Case
16. Response Package review
17. NDPA ROPA worklist
18. DPIA screening and approval
19. Breach Matter timeline
20. Annual filing readiness and package
21. ATM population worklist
22. POS reconciliation
23. Spreadsheet import mapping
24. Source Profile and degradation
25. Evidence micro-request desktop and mobile
26. Photo capture and extraction confirmation
27. Contradiction comparison
28. Decision approval
29. Verification outcome
30. Point-in-time reconstruction
31. Protected report intake and follow-up
32. AI and integration degraded mode
33. No attention required

Each relevant screen requires light and dark modes, supported density, breakpoints, loading, empty, error, stale, unauthorized, and degraded states.

---

# 14. Visual anti-patterns

Do not introduce:

- one card per Requirement in a large Program;
- a wall of KPIs;
- red/amber/green as the only compliance explanation;
- decorative glass on every component;
- constant neon glow;
- oversized empty operational heroes;
- 3D charts without decision value;
- hidden hover-only actions;
- tiny metadata;
- excessive modal stacking;
- full-page chat as the shell;
- dense forms mirroring database schemas;
- separate visual mini-products for each GRC domain;
- or authority cases appearing in ordinary analytics without minimization.

---

# 15. Final standard

A ClearSight experience succeeds when:

- a compliance owner can maintain a Program without reconstructing registers;
- a business owner sees only the facts and action relevant to them;
- a DPO can understand NDPA coverage, ROPA, DPIA, breaches, vendors, and filing readiness in one coherent Program;
- a new regulation becomes a reviewable implementation plan with source lineage;
- a protected authority request becomes a controlled response case;
- an executive understands material change in seconds;
- an expert can inspect every assumption;
- an auditor can reconstruct the approved state;
- and the interface makes institutional complexity feel smaller rather than merely better decorated.