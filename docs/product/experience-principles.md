# ClearSight Experience Principles

This document defines the canonical product experience, information architecture, visual language, interaction rules, component requirements, and visual non-regression standard for ClearSight.

The target is a 2030-quality bank GRC interface: not decorative science fiction, but a system that understands institutional context, assembles known information, proposes useful next steps, minimizes active user effort, and remains calm during high-stakes work.

This document conforms to:

- [`continuous-compliance-operating-model.md`](continuous-compliance-operating-model.md)
- [`ease-of-use-standard.md`](ease-of-use-standard.md)
- [`operating-model.md`](operating-model.md)

Internal architecture must not become user-interface architecture.

---

# 1. Experience objective

A ClearSight user should feel that:

- the system understands the relevant bank, entity, Program, Matter, channel, service, branch, population, and period;
- familiar bank records are already available from approved inventories and integrations;
- known information is prefilled;
- AI has prepared a grounded first draft where useful;
- only missing, stale, contradictory, or material information requires attention;
- the next accountable action is obvious;
- routine work can be completed in a few steps and less than five minutes of active effort;
- complex work reaches a clear saved next state within five minutes;
- evidence, source limitations, uncertainty, authority, and history remain accessible;
- the product remains usable when AI or an integration is unavailable.

Primary interaction grammars:

### Continuing compliance

> **Understand Program state → inspect exceptions → act on what changed → verify → remain current**

### Matter handling

> **Understand → Evidence → Decide or respond → Act → Verify**

### Focused respondent work

> **Context → Confirm or provide → Review → Submit**

### Executive work

> **Brief → Explain → Decide → Verify**

---

# 2. Product personality

ClearSight should feel:

- **calm** — no artificial urgency, constant alarms, or unread-count pressure;
- **direct** — the purpose and next action are immediately visible;
- **precise** — scope, period, source, owner, authority, and state are explicit;
- **relatable** — banking channels, services, customers, accounts, branches, assets, vendors, and obligations appear before internal GRC terminology;
- **prepared** — the system has already assembled the likely context and recommendation;
- **flexible** — different bank sizes, source maturity, jurisdictions, and workflows fit one coherent product;
- **institutional** — suitable for field operations, compliance teams, risk committees, boards, auditors, regulators, and legal teams;
- **premium** — refined typography, spacing, motion, and detail;
- **defensible** — evidence and reasoning are always within reach;
- **restrained** — glass, glow, depth, and color communicate hierarchy rather than decoration.

ClearSight must not feel:

- like a generic admin template;
- like a security console filled with alerts;
- like a consumer banking app;
- like a social feed;
- like a gamified compliance product;
- like a neon cyberpunk concept;
- like a spreadsheet wrapped in cards;
- like a chatbot placed over disconnected modules;
- like a large form mirroring a database schema;
- like every action requires a committee meeting.

---

# 3. Experience architecture

Most authenticated users operate through five primary surfaces.

## 3.1 Today

A role-specific attention brief, not a dashboard catalogue.

Show only:

- Programs moving out of a current state;
- Matters requiring the user’s authority or action;
- expiring or insufficient evidence;
- upcoming filings, reviews, tests, or commitments;
- failed or pending verification;
- material source degradation;
- important changes handled automatically;
- recommendations requiring review.

The default executive brief should normally contain three to seven items.

Today must support:

- a clear review period;
- acknowledgement without hiding the underlying work;
- delegation where authority permits;
- direct action without visiting a module homepage;
- expanded analyst mode;
- true no-material-change state distinct from missing or stale data;
- grouped notifications by Program or Matter;
- continuation of recently active work.

## 3.2 Programs

Programs are the calm home of continuing compliance.

The default Program view should show:

- overall current position and important dimensions;
- applicable requirements by scope;
- material gaps, exceptions, and unknowns;
- evidence becoming stale;
- upcoming filings, reviews, tests, and certifications;
- recent regulatory, institutional, or source changes;
- open Matters;
- recommended actions;
- assurance status.

The Program view should not default to hundreds of controls or a wall of percentages.

Recommended Program sections:

- Overview;
- Requirements and applicability;
- Controls and implementation;
- Evidence and assurance;
- Schedule and obligations;
- Matters and exceptions;
- Scope and ownership;
- History and filings.

A requirement row should provide direct movement to its source provision, applicability, implementation, evidence, owner, exceptions, and open Matters without module hopping.

## 3.3 Work

Work is the user’s Matter queue.

It supports:

- regulatory changes;
- authority requests;
- supervisory findings;
- risk situations;
- audit findings;
- incidents and breaches;
- exceptions and waivers;
- vendor deficiencies;
- control gaps;
- actions and evidence requests;
- approvals and reviews.

Default views should be role-aware and task-oriented:

- Assigned to me;
- Needs my decision;
- Waiting for evidence;
- At risk of deadline;
- Blocked;
- Awaiting verification;
- Delegated;
- Recently changed.

## 3.4 Explore

Explore is an analyst inquiry surface for:

- Programs, Requirements, Controls, Matters, evidence, decisions, and outcomes;
- services, channels, branches, customers, accounts, merchants, assets, systems, vendors, projects, and people;
- sources and data quality;
- populations and relationships;
- incidents, losses, complaints, and trends;
- point-in-time reconstruction.

Explore is not a collection of module homepages.

Prefer:

- scoped search;
- hierarchy;
- readable relationship paths;
- dependency lists;
- population tables;
- timelines;
- affected-scope summaries;
- saved analytical views;
- progressive expansion.

Node graphs are used only when spatial relationships improve comprehension.

## 3.5 Configure

A restricted administrative surface for:

- institution and legal-entity structure;
- Programs and templates;
- channel and jurisdiction packs;
- source registry and mappings;
- controlled vocabularies;
- evidence contracts;
- authority and segregation-of-duties policy;
- thresholds and triggers;
- access, retention, legal hold, and residency;
- integration and automation policy;
- AI capabilities and evaluation gates.

Ordinary users should rarely enter Configure.

## 3.6 Respond and Capture

Focused experiences delivered through mobile, portal, direct link, email, or enterprise messaging where policy permits.

They support:

- one evidence request;
- one branch or asset confirmation;
- vendor submission;
- customer or employee report;
- protected reporting;
- short review or approval;
- clarification or redirection.

The respondent should not need access to the full Program or Matter.

---

# 4. Five-minute flow design

## 4.1 Routine completion budget

Routine flows should normally complete in less than five minutes of active effort.

Design targets:

- focused request: median under three minutes;
- routine approval: median under two minutes;
- repeat import with saved mapping: active effort under five minutes;
- assignment or redirect: under sixty seconds;
- executive comprehension: under sixty seconds;
- next action after resume: understood within thirty seconds.

No routine flow should require more than three major workspace transitions without documented justification.

## 4.2 Complex-work checkpoint

For complex cases, the user should reach a clear saved next state within five minutes:

- confirm assignment and scope;
- accept or correct the initial summary;
- identify missing evidence;
- assign specialist review;
- create a focused request;
- save a draft decision;
- approve the next investigation step;
- create a governed implementation plan.

## 4.3 Active effort, not elapsed time

Model processing, imports, external execution, observation periods, regulator acknowledgement, and asynchronous approvals do not count toward active user effort.

The UI must allow users to leave safely and notify them only when meaningful intervention is needed.

## 4.4 Time and effort visibility

Focused requests should display an estimated active effort where useful.

Longer workflows should show:

- completed sections;
- remaining decisions;
- blockers;
- background work in progress;
- who currently owns the next step.

Do not use gamified progress bars for legally or evidentially complex work. Progress must reflect meaningful completion.

---

# 5. Data-powered interaction principles

## 5.1 Inventory-backed selection

Use approved inventories as workflow sources:

- CMDB and architecture catalogues;
- asset systems;
- branch and organization directories;
- HR and identity directories;
- procurement and vendor systems;
- acquiring and channel systems;
- core customer and account systems;
- ITSM and project platforms;
- ROPA and BIA;
- policy, document, and certificate repositories.

Selections should be searchable, scoped, human-readable, and sourced.

Do not ask users to type names or identifiers that can be selected from a governed inventory.

## 5.2 Prefill behavior

Prefilled fields should indicate:

- source;
- freshness where material;
- whether the user may correct it;
- whether correction changes the source or only the current submission.

Do not overload routine users with provenance details; make them available through an inspect affordance.

## 5.3 Missing-source behavior

When a source is unavailable or stale:

- explain the affected field or conclusion;
- show last-known value and age;
- offer an approved fallback if available;
- state what cannot safely proceed;
- avoid turning manual confirmation into an equal-strength authoritative value.

## 5.4 Progressive integration stability

A flow should look and behave consistently whether values come from a controlled list, spreadsheet, scheduled import, API, or event stream.

Automation should remove steps, not reorganize the product.

---

# 6. Governed AI experience

## 6.1 First drafts, not blank pages

Where approved, provide grounded proposals for:

- regulatory obligations;
- applicability questions;
- control mappings;
- evidence requests;
- Program and Matter summaries;
- remediation options;
- verification criteria;
- policy changes;
- review plans;
- response-package indexes;
- assignments and routing.

## 6.2 Recommendation component

A recommendation must show:

- proposed action;
- why it is recommended;
- sources and versions;
- affected scope;
- assumptions;
- uncertainty or contradiction;
- required authority;
- estimated effort or complexity where useful;
- editable structured fields;
- alternatives;
- expected next state.

Actions:

- accept;
- edit and accept;
- reject;
- request more evidence;
- compare alternatives;
- escalate.

## 6.3 Review by exception

Highlight:

- changed values;
- low-confidence fields;
- new mappings;
- contradictions;
- material effects;
- missing source anchors;
- high-impact side effects.

Allow users to inspect the complete output, but do not require line-by-line review of unchanged high-confidence content unless policy demands it.

## 6.4 No mandatory chat

Chat and command surfaces may support inquiry, navigation, explanation, comparison, and drafting.

They must not become the only way to operate the product.

A prompt should not be required to perform a known structured workflow.

---

# 7. Core interaction patterns

## 7.1 Attention card

Used only for a small queue of Program or Matter items.

Contains:

1. direct situation or obligation statement;
2. why attention is needed now;
3. affected scope;
4. evidence or compliance state;
5. required handling;
6. owner or authority;
7. one primary action.

Rules:

- one dominant message;
- one primary action;
- no generic view-details-only card;
- no unexplained score;
- no green before verified state;
- no material information hidden only on hover.

## 7.2 Program requirement table

Supports:

- requirement and source reference;
- applicability;
- scoped implementation;
- evidence state;
- owner and reviewer;
- exceptions;
- next due event;
- active Matters.

Must provide saved views and exception-focused filters.

## 7.3 Matter workspace

Sections:

- Summary;
- Scope and subjects;
- Evidence;
- Decision or response;
- Actions;
- Outcome or acknowledgement;
- History.

The initial view should show only sections relevant to the Matter type.

## 7.4 Evidence request

Must include:

- purpose;
- why the recipient was selected;
- known facts;
- smallest unresolved question;
- acceptable response types;
- estimated effort;
- deadline and consequence;
- sensitivity;
- redirect, delegate, partial, not-applicable, and concern options.

## 7.5 Forms and controlled values

- prefill known values;
- minimize editable fields;
- use searchable inventories;
- progressively validate;
- hide non-applicable questions;
- avoid free text for identity;
- show final submitted assertions;
- preserve drafts.

## 7.6 Population worklist

For ATMs, POS terminals, accounts, merchants, branches, vendors, controls, cases, or exceptions.

Must support:

- total and filtered denominators;
- resolved, unresolved, contradictory, stale, excluded, and unauthorized states;
- saved filters;
- recommended filters;
- sticky identifiers;
- keyboard navigation;
- next unresolved item;
- compact and comfortable density;
- remembered column preferences;
- authorization-aware bulk action;
- post-action reconciliation.

## 7.7 Spreadsheet import

First use:

1. select file and sheet;
2. select or create Source Profile;
3. map columns;
4. preview;
5. validate and match;
6. confirm scope;
7. import;
8. resolve exceptions.

Repeat use should reuse approved mapping and show only schema changes, errors, duplicates, unresolved identifiers, and material variance.

The UI distinguishes uploaded, parsed, mapped, accepted as Observation, reconciled, and sufficient for a Claim.

## 7.8 Photo and scan capture

Guide users toward bounded visible attributes.

Support framing, legibility, blur/glare checks, metadata notice, retake, redaction where permitted, extraction-region display, field confirmation, and explicit statement of what the media cannot prove.

## 7.9 Review and approval

Show in one view:

- proposed result;
- changed or exceptional fields;
- source evidence;
- uncertainty;
- scope;
- authority basis;
- side effects;
- next state;
- verification.

Approval cannot be context-free.

## 7.10 Save and resume

On return, show:

- what was completed;
- what changed;
- what remains;
- blockers;
- the recommended next action.

---

# 8. Context and safety

Every material workspace must expose active:

- institution or tenant;
- legal entity;
- jurisdiction where relevant;
- Program or Matter;
- service, channel, branch, vendor, customer, account, asset, or population;
- period and effective date;
- source freshness;
- user role or delegated authority where relevant.

Context switching must prevent cross-entity drafts, selections, approvals, or exports.

Ease of use must not hide:

- legal-review requirements;
- material uncertainty;
- evidence contradiction;
- irreversible effects;
- protected-data scope;
- restricted authority;
- required approval;
- population exclusions.

---

# 9. Visual language

## 9.1 Composition

- deep institutional dark mode and clear neutral light mode;
- small number of surface levels;
- generous but not wasteful spacing;
- strong information hierarchy;
- restrained transparency;
- thin borders and subtle elevation;
- tabular numerals for financial and operational values;
- stable layouts while intelligence arrives.

## 9.2 Glass and glow

Glass may be used for temporary focus, command surfaces, comparison overlays, or protected context.

Do not use it on every card, dense tables, long text, evidence documents, or low-power mobile flows.

Glow may communicate focus or selected intelligence, never severity by itself.

## 9.3 Semantic color

- Cyan: context, intelligence, discovered relationship.
- Violet: governance, authority, control, approved automation.
- Coral/red: material exposure, failure, breach.
- Amber: uncertainty, stale evidence, contradiction, approaching threshold, pending verification.
- Green: verified outcome or evidence-supported acceptable state.
- Neutral: informational, unchanged, historical, unassessed.

Color never carries status alone.

## 9.4 Correct visual form

Use:

- cards for small attention queues;
- tables for populations and requirements;
- comparisons for contradiction and version change;
- timelines for history;
- paths for dependencies and lineage;
- step flows for imports and capture;
- charts only for specific analytical questions.

Do not use cards, heat maps, or node graphs as universal components.

---

# 10. Responsive, accessibility, and performance

## Desktop

Optimized for investigation, Program management, population work, evidence review, decisions, and response packages.

## Tablet

Optimized for executive and committee review, approval, evidence inspection, and meetings.

## Mobile

Optimized for focused capture, short review, field confirmation, protected reporting, and urgent approval with context.

Do not squeeze full desktop exploration onto mobile.

## Accessibility

Meet WCAG 2.2 AA at minimum:

- keyboard operation;
- visible focus;
- correct names, roles, and states;
- screen-reader announcements;
- non-color status;
- touch target size;
- reduced motion;
- 200% zoom;
- multilingual expansion;
- accessible chart summaries;
- no materially longer workflow for assistive-technology users.

## Performance

- deterministic context appears before AI;
- common interactions acknowledge immediately;
- long processing is resumable;
- lists are virtualized or paginated;
- uploads are resumable;
- layouts do not shift unexpectedly;
- AI and source outages have manual fallbacks;
- effects are tested on enterprise laptops, remote desktops, and mobile devices.

---

# 11. Design-system requirements

Foundations:

- semantic color;
- typography;
- spacing;
- radius;
- border;
- elevation;
- blur;
- motion;
- breakpoints;
- density;
- iconography;
- active-effort and transition budgets.

Core components:

- application shell;
- scope header;
- Today attention card;
- Program summary and requirement table;
- Matter workspace;
- focused evidence request;
- recommendation panel;
- source profile;
- evidence-state badge;
- sufficiency panel;
- contradiction comparison;
- population worklist;
- spreadsheet mapper;
- media capture and extraction review;
- owner and authority selector;
- decision review;
- action plan;
- verification panel;
- response package;
- timeline;
- saved view and filter;
- empty, degraded, offline, and resumed states.

Required variants:

- light and dark;
- comfortable and compact density;
- desktop, tablet, and mobile where relevant;
- hover, focus, active, selected, disabled, loading, error, warning, protected, stale, contradictory, pending, and verified.

---

# 12. Golden screens and flows

Maintain visual and timed interaction references for:

1. Today brief
2. Program overview
3. Program requirement and evidence state
4. Program gap and exception view
5. Work queue
6. Matter summary
7. Regulatory change source review
8. AI obligation recommendation
9. Authority request case
10. Response package review
11. Focused evidence request
12. Mobile field capture
13. Population worklist
14. Spreadsheet mapping first use
15. Repeat import with saved mapping
16. Reconciliation and contradiction
17. Routine approval
18. Material decision review
19. Action and verification
20. ROPA update
21. DPIA screening
22. Breach Matter
23. Vendor evidence review
24. Source degradation
25. AI unavailable fallback
26. Offline capture and resume
27. Point-in-time reconstruction
28. No material change
29. No data or unknown state
30. Protected reporting and investigator view

Each relevant flow requires first-use and repeat-use timing.

---

# 13. Visual and functional anti-patterns

Do not introduce:

- KPI walls;
- control walls as the default Program view;
- forms showing every possible field;
- repeated entry of inventory data;
- generic “view details” cards;
- workflows requiring chat prompts;
- permanent questionnaires where targeted requests are possible;
- mandatory module hopping;
- remapping stable spreadsheets every cycle;
- hidden save behavior;
- approval without context;
- AI output without sources;
- AI recommendations that add review burden;
- dashboard status detached from evidence;
- green for uploaded, submitted, assigned, or implemented;
- dense graph canvases as default navigation;
- decorative glass and glow;
- hidden actions available only on hover;
- inaccessible custom controls;
- complex flows that cannot safely resume.

---

# 14. Design review checklist

Before approving a screen, component, or flow:

## Outcome

- Is the user’s intended outcome obvious?
- Is there one clear next action?
- Does the surface use Program or Matter language appropriately?

## Effort

- What does ClearSight already know?
- Are known values prefilled?
- Can a source integration remove fields?
- Can AI provide a grounded first draft?
- Can routine work finish within five minutes?
- Can complex work reach a clear saved next state within five minutes?
- Is the number of major transitions minimized?

## Trust

- Are scope, source, freshness, evidence, uncertainty, authority, and consequence available?
- Are inferred and approved values distinct?
- Are contradictions visible?

## Flexibility

- Does the flow work with controlled lists, spreadsheets, APIs, or events?
- Can different bank sizes and jurisdictions use the same semantics?
- Does configuration preserve upgradeability?

## Accessibility and performance

- Can the complete journey be performed by keyboard and assistive technology?
- Does it remain usable on enterprise hardware and poor networks?
- Is deterministic context usable while AI is pending or unavailable?

A screen has succeeded when it makes the governed outcome easier—not merely when it looks modern.