# ClearSight Experience Principles

This document defines the canonical product experience, information architecture, visual language, interaction rules, and visual non-regression requirements for ClearSight.

The target is a 2030-quality bank GRC interface: not decorative science fiction, but a system that understands context, removes unnecessary work, makes institutional uncertainty legible, and remains calm during high-stakes decisions.

This document must conform to [`operating-model.md`](operating-model.md). Internal architecture must not become user-interface architecture.

---

# 1. Experience objective

A ClearSight user should feel that:

- the system understands the relevant bank, entity, channel, service, branch, population, and time period;
- the current situation is explained in familiar banking language;
- known information is already assembled;
- missing or contradictory proof is explicit;
- only the next proportionate action is emphasized;
- AI reduces effort without becoming a second product to operate;
- source quality and freshness are visible;
- and every decision will remain reconstructable.

The primary interaction grammar is:

> **Understand → Evidence → Decide → Act → Verify**

For executives this compresses to:

> **Brief → Explain → Decide → Verify**

For evidence respondents it compresses to:

> **Context → Confirm or provide → Review → Submit**

The same product logic applies across operational risk, compliance, cyber, resilience, third-party, incidents, audit, customer signals, and protected reporting.

---

# 2. Product personality

ClearSight should feel:

- **calm** — no artificial urgency, constant alarm, or blinking state;
- **direct** — the current situation and next action are obvious;
- **precise** — scope, period, source, owner, authority, and state are explicit;
- **relatable** — banking channels, services, branches, merchants, customers, assets, and vendors appear before abstract control language;
- **institutional** — appropriate for branch operations, a risk committee, boardroom, audit team, or regulator;
- **intelligent** — context is prefilled and repetitive assembly work disappears;
- **premium** — refined typography, spacing, motion, and detail;
- **defensible** — evidence and reasoning are always within reach;
- **restrained** — glass, glow, depth, and color communicate meaning rather than decoration.

ClearSight must not feel:

- like a generic admin template;
- like a security operations console filled with alerts;
- like a consumer banking application;
- like a social feed;
- like a gamified compliance product;
- like a neon cyberpunk concept;
- like a large spreadsheet with cards around it;
- or like a chat assistant placed over disconnected GRC modules.

---

# 3. Experience architecture

Most users operate through five surfaces.

## 3.1 Today

The default landing surface is a role-specific brief, not a dashboard catalogue.

It should show only:

- situations that materially changed;
- decisions requiring the user’s authority;
- evidence gaps requiring intervention;
- appetite breaches or approaching limits;
- actions likely to miss a deadline;
- failed or pending verification;
- important upcoming obligations;
- and important changes that were safely automated.

The default executive view should usually contain between three and seven situation cards.

Today must support:

- a clear review period such as “since your last review” or a selected date range;
- acknowledgement without suppressing the underlying situation;
- delegation where authority permits;
- saved role views without making dashboard configuration the product;
- an expanded monitoring mode for analysts;
- and a true “no material change” state distinct from missing data.

## 3.2 Situation

Situation is the primary work surface.

One workspace combines the context that conventional GRC products divide across risk, control, evidence, issue, action, approval, and assurance modules.

### Summary

- what is happening;
- what changed;
- why it matters now;
- active scope and period;
- affected services, customers, entities, branches, merchants, systems, assets, or vendors;
- applicable exposure patterns;
- appetite or tolerance position;
- current state;
- and required next action.

### Evidence

- required claims;
- what is known;
- what is missing;
- what is stale;
- what conflicts;
- source authority and limitations;
- coverage and population;
- original evidence and derived observations;
- assumptions;
- and conclusion state.

### Decision

- available options;
- likely effect and limitations;
- cost, dependencies, time-to-effect, reversibility, and customer impact where relevant;
- required authority;
- segregation-of-duties or conflict checks;
- selected option and rationale;
- dissent, conditions, expiry, and review triggers.

### Action

- action plan;
- responsible owner;
- dependencies;
- external execution state;
- implementation evidence;
- blockers;
- and escalation.

### Outcome

- expected outcome;
- baseline;
- measurement source;
- observation period;
- current observed result;
- success and failure thresholds;
- acceptance authority;
- and verified, ineffective, indeterminate, or awaiting-verification state.

### History

- prior situation versions;
- evidence changes;
- decisions and overrides;
- source degradation;
- actions;
- verification outcomes;
- and point-in-time reconstruction.

Users must not need to navigate several modules to answer one material question.

## 3.3 Capture

Capture is a lightweight web and mobile surface for:

- answering one focused question;
- confirming prefilled facts;
- selecting an existing bank record;
- photographing or scanning an item;
- uploading a spreadsheet or document;
- validating AI-extracted values;
- reporting a discrepancy;
- redirecting to a better source;
- or declaring that a request is not applicable.

Capture must show:

- why the information is needed;
- the claim or practical question in plain language;
- what ClearSight already knows;
- the active scope and period;
- acceptable evidence forms;
- estimated effort;
- deadline and consequence;
- sensitivity and sharing boundary;
- and a clear final review before submission.

## 3.4 Explore

Explore is an analyst inquiry surface for:

- situations;
- banking channels and services;
- branches, merchants, customers, assets, systems, vendors, and people;
- exposure patterns;
- populations;
- sources and data quality;
- claims, observations, evidence, obligations, controls, incidents, decisions, actions, and outcomes;
- relationship paths;
- trends;
- and point-in-time reconstruction.

Explore is not a collection of module homepages.

The default is not a large node graph. Prefer:

- scoped search;
- hierarchy;
- readable relationship paths;
- dependency lists;
- population tables;
- affected-scope summaries;
- timelines;
- and progressive expansion.

Graph visualization is used only where spatial relationships improve comprehension.

## 3.5 Configure

Configure is a restricted administrative surface for:

- institution and legal-entity structure;
- channel and jurisdiction packs;
- source registry;
- controlled vocabularies;
- exposure patterns;
- evidence recipes;
- appetite, limits, and tolerances;
- authority and segregation-of-duties rules;
- access, retention, and legal hold;
- integration mapping;
- AI and automation permissions;
- and deployment-specific policy.

Ordinary users should rarely enter Configure.

Configure must not become a generic settings landfill. Configuration should be grouped by the institutional outcome it controls.

---

# 4. Mandatory context anchoring

Wrong-scope action is a material usability and governance failure.

Every material workspace must make the following available without requiring memory:

- institution or tenant;
- legal entity;
- country or jurisdiction where relevant;
- channel, service, branch, merchant group, system, vendor, or population;
- effective period;
- record or review time;
- current user role or delegated authority where relevant;
- data freshness state.

Use a compact context header or scope breadcrumb. It must not consume excessive vertical space, but it must remain visible before approval, export, bulk action, or evidence submission.

Context switching must:

- clearly indicate the new entity or scope;
- preserve or intentionally reset filters;
- prevent accidental cross-entity actions;
- warn when a draft belongs to another scope;
- and never rely on subtle color changes alone.

---

# 5. Core interaction patterns

## 5.1 Situation card

The situation card is the primary unit of attention.

It contains, in order:

1. **Situation** — one direct sentence in banking language.
2. **Why now** — the change, deadline, threshold, or uncertainty requiring attention.
3. **Affected scope** — service, channel, entity, customer group, branch, asset population, system, vendor, or obligation.
4. **Exposure** — applicable exposure family and plausible consequence.
5. **Evidence state** — sufficient, incomplete, stale, contradictory, unavailable, or pending.
6. **Required handling** — monitor, provide evidence, investigate, decide, act, or verify.
7. **Owner or authority** — accountable person or committee and due time.

Card rules:

- one dominant message;
- one primary next action;
- no more than two secondary actions while collapsed;
- no unexplained score;
- no generic “view details” as the only action;
- no green state unless the defined outcome is accepted as verified;
- no red state solely because a source is missing;
- no hidden material context available only on hover.

A situation card is not always a decision card. It may require evidence, investigation, monitoring, or verification before a decision is appropriate.

## 5.2 Evidence request

An evidence request should feel like a short, contextual work message rather than a GRC assessment.

It includes:

- why this person or team is being asked;
- what is already known;
- the smallest unresolved question;
- affected banking object or population;
- acceptable response forms;
- estimated effort;
- deadline;
- confidentiality;
- and redirect, partial-response, not-applicable, and sensitivity options.

Avoid:

- control IDs as the primary language;
- broad free-text prompts;
- requiring the full risk object to be understood;
- asking for fields already available from a source;
- and repeated requests that could have been deduplicated.

## 5.3 Population worklist

Many bank workflows concern populations rather than single objects: ATMs, POS terminals, merchants, accounts, branches, vendors, controls, incidents, or exceptions.

A population worklist must support:

- total population and current filtered population;
- resolved, unresolved, contradictory, stale, excluded, and not-applicable counts;
- explicit denominator for every percentage;
- search and saved filters;
- column visibility and density controls without ad hoc layouts;
- sticky identifiers and headers;
- row-level source and freshness state;
- inline comparison where safe;
- keyboard navigation;
- virtualized or paginated loading;
- selection summaries;
- and export with scope and manifest.

Bulk actions must show:

- exact selection criteria;
- number of affected records;
- excluded or unauthorized records;
- side effects;
- reversibility;
- required approval;
- and a post-action reconciliation result.

A bulk action must never silently apply to records outside the visible authorized scope.

## 5.4 Reconciliation and matching

Reconciliation should visually distinguish:

- matched;
- provisionally matched;
- unresolved;
- contradictory;
- duplicate;
- rejected;
- and superseded.

The interface should provide:

- source records side by side;
- normalized identifiers;
- proposed match reason;
- confidence dimensions rather than false precision;
- affected downstream claims or situations;
- merge and unmerge history;
- and a clear human decision where required.

Do not hide unresolved mappings behind an import-success banner.

## 5.5 Spreadsheet import

Spreadsheet and CSV import are first-class enterprise workflows.

The import experience should contain:

1. file and sheet selection;
2. source profile and intended purpose;
3. column detection and mapping;
4. sample preview;
5. type and required-field validation;
6. identifier matching;
7. duplicate and conflict analysis;
8. scope confirmation;
9. import summary;
10. reconciliation queue and rollback reference.

Every imported observation must retain:

- file;
- sheet;
- row;
- mapping version;
- uploader or managed source;
- import time;
- and validation state.

The UI must distinguish:

- uploaded successfully;
- parsed successfully;
- mapped successfully;
- accepted as an observation;
- and sufficient for a claim.

These are not equivalent states.

## 5.6 Photo and scan capture

Photo capture should guide the user toward verifiable visible attributes.

The interface may request:

- full asset in context;
- serial number or label;
- location signage;
- tamper seal;
- screen state or error code;
- supporting document;
- or a second angle.

During capture:

- show framing or legibility guidance;
- detect blur, glare, crop, and unreadable identifiers;
- disclose whether device metadata or location is collected;
- avoid collecting unnecessary background or people;
- allow redaction or retake where policy permits;
- preserve the original;
- show extracted fields and regions;
- require confirmation where necessary;
- and clearly state what the image can and cannot establish.

Never present “AI verified secure” from a general photograph. Present bounded observations such as a visible serial number match or apparently intact external seal.

## 5.7 Forms and controlled values

Forms should be generated around unresolved facts and user intent.

Requirements:

- known fields are prefilled and usually read-only;
- controlled values come from administrator-approved or authoritative sources;
- dropdowns are searchable and scoped;
- identifiers are shown with human-readable labels;
- dependent fields update predictably;
- free text is reserved for explanation, not basic identity;
- validation occurs progressively;
- and the final review shows exactly what assertions will be submitted.

## 5.8 Evidence sufficiency

Evidence quality should be understandable across:

- relevance;
- authenticity;
- coverage;
- freshness;
- independence;
- completeness;
- consistency;
- reliability;
- and traceability.

Do not compress these dimensions into one unexplained percentage.

Use a concise state with an expandable breakdown, source references, policy requirements, and unresolved contradiction.

## 5.9 Contradiction

Contradiction is a first-class state.

The interface should show:

- disputed claim;
- conflicting observations or evidence side by side;
- source authority and limitations;
- effective periods;
- population or scope mismatch;
- affected conclusions, decisions, and reports;
- unresolved questions;
- assigned resolver;
- and time sensitivity.

The system must not silently select the source that makes the situation look better.

## 5.10 Decision review

A material decision review must show:

- exact scope and period;
- conclusion and evidence state;
- uncertainty and contradiction;
- options and important trade-offs;
- affected customers, services, entities, or obligations;
- authority and segregation-of-duties basis;
- expiry and review triggers;
- irreversible or external side effects;
- and verification method.

Approval must not be a context-free button.

High-impact or irreversible actions should require deliberate confirmation, but not repetitive ceremony. Use confirmation proportional to materiality and reversibility.

## 5.11 Verification

Completion and verification must have separate visual states.

Show:

- work performed;
- implementation evidence;
- expected outcome;
- baseline;
- measurement source;
- observation period;
- current observed result;
- threshold;
- acceptance authority;
- and failure response.

Do not use green while an observation period is incomplete.

## 5.12 Natural-language command surface

The command surface is global but not dominant.

It may support:

- inquiry;
- navigation;
- comparison;
- summarization;
- drafting;
- simulation;
- and proposing governed actions.

It must not become the only way to operate ClearSight.

Responses should include:

- concise answer;
- active scope and time period;
- source references;
- confidence and missing information;
- contradictions;
- affected objects;
- and safe structured next actions.

A command that may create side effects must transition into the normal review surface.

---

# 6. Source and data-quality experience

Every integrated, imported, or human source should have a visible Source Profile appropriate to the user’s authority.

Show:

- source owner;
- what the source is authoritative for;
- what it is not authoritative for;
- scope;
- expected freshness;
- current age;
- last successful collection;
- current health;
- mapping version;
- known limitations;
- unresolved records;
- and affected conclusions.

Source health states must distinguish:

- current;
- delayed;
- stale;
- partially available;
- mapping degraded;
- authorization revoked;
- unavailable;
- and retired.

A successful API response or uploaded file must not visually imply that the data is complete, accurate, current, or sufficient.

Data-quality debt should appear where it changes a situation, decision, or assurance conclusion—not as a separate wall of technical metrics for executives.

---

# 7. Visual language

## 7.1 Overall composition

Use:

- deep institutional surfaces in dark mode;
- clean neutral surfaces in light mode;
- controlled transparency;
- thin but visible boundaries;
- subtle elevation;
- generous spacing around decisions;
- compact density inside operational populations;
- high legibility;
- and focused semantic accents.

The product must support both calm executive review and dense operational reconciliation without turning every screen into the same card grid.

## 7.2 Surface hierarchy

Use a small number of semantic surfaces:

1. **Canvas** — application background.
2. **Primary work surface** — situation, table, document, or capture area.
3. **Raised focus surface** — decision review, command result, or temporary comparison.
4. **Protected surface** — restricted evidence, case content, or identity workflow.
5. **Transient surface** — menu, tooltip, toast, popover.
6. **Offline or degraded surface** — locally queued or stale context, clearly marked.

Do not invent a new surface style for every component.

## 7.3 Glass

Glass is structural, not decorative.

Appropriate uses:

- command surface over current context;
- focused decision layer;
- temporary relationship overlay;
- protected review boundary;
- simulation comparison.

Inappropriate uses:

- every card;
- long text;
- dense tables;
- evidence documents;
- spreadsheet mapping;
- protected-report narratives;
- low-power mobile capture.

## 7.4 Glow

Glow may indicate:

- current focus;
- active analysis;
- selected relationship path;
- or newly updated context.

Glow must not represent severity by itself and must never surround all interactive elements.

## 7.5 Semantic color

Color roles are tokenized.

- **Cyan:** context, discovered relationship, selected analysis path.
- **Violet:** governance, policy, authority, approved automation.
- **Coral or red:** material exposure, breach, severe gap, failed verification.
- **Amber:** uncertainty, stale or incomplete evidence, approaching threshold, unresolved contradiction.
- **Green:** accepted verified outcome only.
- **Blue or neutral informational:** current operation, link, selection, baseline, or ordinary progress.
- **Neutral muted:** historical, not assessed, secondary metadata.
- **Protected treatment:** distinct surface and iconography, not a severity color.

Never use green for task completion, document presence, self-attestation, or absence of alerts.

Never use red merely to increase engagement.

## 7.6 State semantics

The following must remain visually and textually distinct:

- no risk observed;
- no material change;
- no data received;
- data not yet assessed;
- insufficient evidence;
- stale evidence;
- source unavailable;
- unauthorized;
- not applicable;
- unknown;
- contradictory;
- action complete;
- awaiting verification;
- verified effective;
- verified ineffective.

## 7.7 Typography

Requirements:

- modern neutral sans-serif suitable for long enterprise use;
- tabular numerals for operational, financial, percentage, and time-series values;
- clear roles for situation statement, decision, evidence, metadata, and identifier;
- readable compact text for dense tables;
- no ultra-light weights;
- no important information in tiny uppercase labels;
- stable line heights across languages;
- and controlled line length for investigation narratives.

All typography comes from tokens.

## 7.8 Numbers, currency, dates, and time

Every material number must communicate:

- unit;
- denominator where applicable;
- time period;
- currency and basis;
- rounding or approximation;
- source;
- and comparison basis.

Support institution and jurisdiction formats for:

- currency;
- decimal and thousands separators;
- date;
- time zone;
- reporting period;
- and local terminology.

Relative dates such as “yesterday” should reveal the exact timestamp.

## 7.9 Icons

Use one coherent icon family with consistent optical weight, stroke, and corner treatment.

Icons must not carry risk state or authority alone.

Banking channels and common objects may use recognizable icons, but labels remain primary.

## 7.10 Charts and relationship views

Prefer:

- risk movement over time;
- evidence coverage by population and period;
- source freshness;
- exception concentration;
- causal and dependency paths;
- service maps;
- before-and-after treatment comparison;
- confidence or uncertainty ranges;
- reconciliation variance;
- and scenario deltas.

Use heat maps as an analytical view, not the primary executive truth.

Every chart requires:

- clear question or title;
- unit and denominator;
- period;
- source;
- accessible textual summary;
- uncertainty explanation;
- and a direct path to the underlying situation or population.

Avoid 3D charts and decorative animated data flow.

---

# 8. Layout and density

## 8.1 Desktop

Desktop supports investigation, population review, decision, and administration.

A typical layout may include:

- compact navigation rail;
- scope/context header;
- main work surface;
- contextual evidence or explain panel;
- and optional command surface.

Do not permanently reserve large width for secondary navigation.

## 8.2 Density modes

Support at least:

- **comfortable** for executives, situation review, and narrative evidence;
- **compact** for operational worklists, reconciliation, controls, and imports.

Density must affect spacing and row height without hiding labels, reducing target size below accessibility requirements, or changing semantic hierarchy.

## 8.3 Tablet

Tablet supports:

- executive and committee review;
- approval;
- evidence inspection;
- meeting mode;
- and selected capture.

Side panels may become full-screen workspaces. Tables should preserve key identifiers and allow progressive detail rather than horizontal compression of every column.

## 8.4 Mobile

Mobile is primarily for:

- focused evidence capture;
- short review;
- approval with sufficient context;
- protected reporting;
- incident updates;
- and urgent material notifications.

Do not squeeze full graph exploration or wide reconciliation tables onto mobile.

## 8.5 Large display and meeting mode

Boardroom or committee mode should emphasize:

- material portfolio movement;
- critical services;
- concentrated exposures;
- appetite and evidence state;
- decisions and owners;
- and verification outcomes.

Meeting mode should provide privacy-aware presentation, stable layouts, concise annotations, and point-in-time freeze. It must not become a wall of blinking metrics.

---

# 9. Attention and notification design

ClearSight should reduce interruption rather than create another alert channel.

Notifications must be:

- deduplicated;
- role- and authority-aware;
- materiality-sensitive;
- channel-appropriate;
- grouped by situation where possible;
- and cancelled when the underlying need is resolved elsewhere.

The interface should distinguish:

- information;
- work assigned;
- decision required;
- deadline approaching;
- material escalation;
- and protected communication.

Do not use unread counts as a proxy for urgency.

Every notification must lead to a clear bounded action or explanation.

---

# 10. State and failure design

Every feature must deliberately design:

- loading;
- partial or streaming analysis;
- empty;
- no material change;
- insufficient evidence;
- contradictory evidence;
- pending approval;
- delegated;
- executing;
- implemented;
- awaiting verification;
- verified effective;
- verified ineffective;
- indeterminate;
- stale;
- superseded;
- unauthorized;
- offline;
- locally queued;
- synchronization conflict;
- integration degraded;
- AI unavailable;
- import partially accepted;
- and export failed.

## 10.1 AI latency

- Show deterministic context first.
- Identify what analysis is still running.
- Preserve stable layout.
- Allow cancellation where safe.
- Provide timeout, retry, fallback, and manual operation.
- Never show an old AI conclusion as newly generated.

## 10.2 Integration degradation

Show:

- affected source;
- last successful synchronization;
- current age;
- affected observations, claims, and situations;
- confidence impact;
- fallback source where available;
- and recovery state.

## 10.3 Offline capture

Where branch or field use requires it:

- allow a bounded local draft or encrypted queue;
- show unsynchronized state clearly;
- preserve capture time separately from upload time;
- detect conflicts after synchronization;
- prevent duplicate submission;
- and explain when protected or highly restricted evidence cannot be stored offline.

---

# 11. AI presentation

AI should appear as capability, not theatre.

Prefer visible outcomes such as:

- values extracted and ready for confirmation;
- duplicate records identified;
- a focused question generated;
- a concise explanation assembled;
- or contradictory evidence surfaced.

Avoid unnecessary sparkle buttons and persistent “AI” labels on ordinary deterministic functions.

When AI materially influences a result, show:

- what it did;
- source inputs;
- machine-inferred versus explicit values;
- confirmation state;
- model or operator identity where relevant;
- uncertainty;
- and a correction path.

The interface must not expose hidden chain-of-thought. It should show a concise, structured reasoning record from source facts, applicable policy, assumptions, alternatives, conclusion, and approval requirement.

---

# 12. Protected and privacy-sensitive experience

Protected reporting and highly restricted evidence should feel clearly separate from ordinary operations.

Requirements:

- distinct protected surface;
- visible confidentiality boundary;
- no identity hints in ordinary search, counts, notifications, or graph context;
- conflict-aware routing;
- separate privileged identity-reveal workflow;
- controlled copy, print, export, and download;
- privacy-aware presentation mode;
- automatic timeout and reauthentication where appropriate;
- and explicit warnings before moving information to a less protected context.

Consider screen masking, redacted previews, watermarking, or clipboard restrictions where justified by policy and threat model.

Do not rely on visual masking as the authorization control.

---

# 13. Protected reporting journey

The external portal should be simple, independent, accessible, and low-bandwidth tolerant.

## Entry

Explain:

- anonymous and identified options;
- how anonymous communication works;
- what information is collected;
- confidentiality limits;
- emergency or immediate-danger boundaries;
- and how to retain the case token.

## Submission

Use a guided, non-leading sequence:

- what happened;
- affected area, service, branch, product, or people;
- when it occurred;
- whether it is ongoing;
- who or what may verify it;
- available evidence;
- immediate customer, financial, safety, or legal impact;
- and preferred communication mode.

Do not force legal or risk classification.

## Follow-up

Anonymous communication must preserve the reporter’s anonymity. Token loss guidance must not promise impossible recovery.

## Investigator

Show original report, allegation-versus-fact state, protected identity boundary, conflicts, related cases, evidence and chain of custody, messages, obligations, and permitted next actions.

---

# 14. Accessibility, inclusion, and localization

ClearSight must meet WCAG 2.2 AA at minimum.

Requirements include:

- complete keyboard operation;
- visible focus;
- meaningful headings and landmarks;
- correct names, roles, values, and states;
- screen-reader announcements for asynchronous changes;
- non-color status;
- accessible chart summaries;
- target sizes suitable for touch;
- reduced-motion support;
- error prevention and recovery;
- sufficient contrast on glass and protected surfaces;
- logical table navigation;
- 200% zoom without loss of function;
- multilingual expansion;
- and plain language where specialist terminology is unnecessary.

The design system should anticipate:

- longer translated labels;
- right-to-left layouts if supported;
- local currencies and date formats;
- varying name structures;
- low-bandwidth branches;
- and users with limited GRC familiarity.

No evidence, approval, or protected-reporting journey may require a mouse.

---

# 15. Performance experience

Perceived performance is part of institutional trust.

Required behavior:

- application shell and current deterministic context appear quickly;
- AI never blocks ordinary navigation or manual decision work;
- long lists are paginated or virtualized;
- table filters acknowledge immediately;
- graph detail loads progressively;
- spreadsheet parsing and import show stages;
- evidence upload is resumable;
- layout remains stable;
- optimistic UI is used only where rollback is safe;
- and remote desktop and integrated-GPU performance are tested.

Avoid:

- excessive backdrop blur;
- large animated gradients;
- continuously animated graph edges;
- expensive effects in dense tables;
- and background animation that competes with risk states.

---

# 16. Design-system requirements

## Foundations

- semantic color;
- typography;
- numeric styles;
- spacing;
- grid;
- radius;
- border;
- elevation;
- blur;
- motion;
- breakpoints;
- density;
- iconography;
- focus;
- protected treatment;
- and data-visualization roles.

## Core components

- application shell;
- scope and context header;
- Today brief;
- situation card;
- situation workspace;
- evidence request;
- mobile capture step;
- photo guidance and extraction review;
- spreadsheet mapper;
- import summary;
- population worklist;
- reconciliation compare view;
- source profile and health badge;
- evidence-state summary;
- sufficiency panel;
- contradiction compare view;
- relationship path;
- risk movement indicator;
- owner and authority treatment;
- decision review;
- action plan;
- verification panel;
- timeline and point-in-time control;
- protected-content surface;
- approval control;
- data table;
- filter and saved view;
- notification and work item;
- command surface;
- chart and textual alternative;
- empty, stale, offline, degraded, unauthorized, and AI-unavailable states;
- export manifest;
- and audit event viewer.

## Required variants

- light and dark;
- comfortable and compact density;
- desktop, tablet, and mobile where applicable;
- normal, hover, focus, active, selected, disabled, loading, error, warning, protected, stale, contradictory, awaiting verification, and verified states.

No production feature may invent ad hoc visual tokens when a semantic token can be added.

---

# 17. Visual anti-patterns

Do not introduce:

- a wall of KPI cards;
- architecture names as the primary navigation;
- separate module pages for one situation’s risk, evidence, decision, action, and outcome;
- multiple competing primary colors;
- glass on every card;
- constant neon glow;
- decorative animated data flow;
- oversized empty hero areas in operational screens;
- 3D charts;
- unexplained percentages;
- percentages without denominators;
- status conveyed only through red, amber, or green;
- tiny low-contrast metadata;
- hidden actions available only on hover;
- excessive modal stacking;
- full-page chat as the primary shell;
- dense forms mirroring database schemas;
- arbitrary card layouts where a table is the correct population tool;
- imports that show “success” while hiding unresolved rows;
- AI claims that exceed visible evidence;
- or mobile screens that compress desktop complexity rather than simplifying the task.

---

# 18. Functional non-conformities visible in the UI

The interface must not normalize behavior such as:

- closing a finding without accepted effectiveness evidence;
- approving risk without authority, scope, conditions, and expiry;
- showing AI output without sources and confirmation state;
- treating missing data as low risk;
- treating automated data as inherently authoritative;
- hiding contradiction or unresolved mappings;
- showing stale source data as current;
- exposing protected identity outside privileged flow;
- asking for facts already available from authorized evidence;
- treating file upload as evidence sufficiency;
- treating action completion as verified outcome;
- or presenting a single score as objective truth.

---

# 19. Golden screens

Maintain design references and automated visual regression for:

1. Today executive brief.
2. Situation card — evidence-needed, decision-needed, and verification-failed states.
3. Situation workspace — Summary, Evidence, Decision, Action, Outcome, and History.
4. ATM inventory exposure.
5. POS settlement or terminal-identity exposure.
6. Population worklist with unresolved and contradictory rows.
7. Spreadsheet mapping and validation.
8. Import summary and reconciliation queue.
9. Source Profile and degraded-source state.
10. Evidence micro-request on desktop and mobile.
11. Branch asset photo capture with AI extraction review.
12. Form and controlled-value confirmation.
13. Evidence sufficiency view.
14. Contradiction comparison.
15. Decision approval with scope, authority, and irreversible side effects.
16. Action implemented but awaiting verification.
17. Verification failed and situation reopened.
18. Relationship path and dependency summary.
19. Point-in-time reconstruction.
20. No material change, no data, not assessed, and unauthorized states.
21. Whistleblower intake.
22. Anonymous reporter follow-up.
23. Protected investigator case.
24. Regulatory source-to-evidence lineage.
25. Board or committee meeting mode.
26. AI extraction review and correction.
27. AI unavailable with manual fallback.
28. Offline capture queued and synchronization conflict.
29. Bulk action review and post-action reconciliation.
30. Export review and manifest.

Each golden screen must cover:

- light and dark mode;
- relevant comfortable and compact density;
- relevant desktop, tablet, and mobile breakpoints;
- normal, loading, empty, stale, contradictory, unauthorized, error, and degraded variants where applicable;
- 125%, 150%, and 200% zoom checks;
- and representative long-label localization.

---

# 20. Design review checklist

## Product clarity

- Is the active bank, entity, channel, population, and period clear?
- Is the situation stated in familiar language?
- Is the reason it matters now visible?
- Is the evidence state understandable?
- Is the required handling clear?
- Can the user remain in one situation workspace?

## Effort

- Is known information prefilled?
- Are only unresolved facts editable?
- Can evidence be collected automatically?
- Is a spreadsheet, photo, or form workflow proportionate?
- Can the user redirect or report a mismatch?
- Are bulk operations safe and comprehensible?

## Data trust

- Is source authority visible?
- Are freshness and source health clear?
- Are unresolved mappings and contradictions explicit?
- Is upload success distinguished from evidence sufficiency?
- Are machine-inferred fields distinguishable from confirmed fields?

## Decision safety

- Are scope, authority, side effects, reversibility, conditions, and expiry visible?
- Is the action separate from the expected outcome?
- Is verification defined before closure?
- Does the UI avoid overstating causality?

## Visual quality

- Is there one dominant hierarchy?
- Is the screen calm but not empty?
- Is a table used where a population requires one?
- Are glass and glow semantic and bounded?
- Does light mode feel equally designed?
- Are protected surfaces clearly distinct?

## Accessibility and localization

- Is the journey keyboard-operable?
- Are focus, errors, and async states announced?
- Is status available without color?
- Does the layout work at 200% zoom?
- Do long translated labels and local number formats remain usable?

## Performance

- Is deterministic information available before AI completes?
- Are long lists virtualized or paginated?
- Does the layout remain stable?
- Are effects inexpensive on enterprise hardware?
- Can capture recover from low bandwidth or interruption?

---

# 21. Final standard

A ClearSight screen should not merely look modern.

It should make a bank risk situation feel:

- bounded;
- recognizable;
- smaller;
- clearer;
- better evidenced;
- safer to decide;
- easier to act upon;
- and possible to verify.

The design has succeeded when:

- a senior executive understands the material situation in seconds;
- a risk specialist can inspect every source, assumption, contradiction, and relationship;
- a branch or channel employee can provide the required evidence with minimal effort;
- a large population can be reconciled without spreadsheet chaos;
- and an auditor can reconstruct the full outcome without needing an undocumented explanation.