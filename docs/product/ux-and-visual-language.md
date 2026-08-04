# ClearSight UX and Visual Language

This document defines the implementation-ready behavioral, interaction, visual, responsive, input, motion, and reporting language for ClearSight.

It complements:

- [`continuous-compliance-operating-model.md`](continuous-compliance-operating-model.md)
- [`ease-of-use-standard.md`](ease-of-use-standard.md)
- [`operating-model.md`](operating-model.md)
- [`experience-principles.md`](experience-principles.md)

The experience principles define what ClearSight should feel like and how users move through Programs and Matters. This document defines the concrete visual and component system through which that experience is implemented.

The governing standard is:

> **ClearSight should feel prepared, calm, direct, and trustworthy. The system assembles context, evidence, and recommendations before asking a person to act, so routine work can be completed in a few clear steps and less than five minutes of active effort.**

---

# 1. Overall behavioral psychology and UX target

## 1.1 Desired user state

ClearSight is used during high-accountability work. Users may be responding to a regulator, reviewing a control gap, handling an authority request, confirming branch evidence, preparing a board report, or deciding whether a material risk can be accepted.

The product should move the user from:

| Starting state | Target state |
|---|---|
| “I need to find the right spreadsheet.” | “The relevant Program or Matter is already assembled.” |
| “I do not know what this request relates to.” | “The scope, reason, authority, and required outcome are clear.” |
| “I have to enter everything again.” | “Known information is prefilled from approved sources.” |
| “I am being examined by a system.” | “The system is helping me complete my responsibility.” |
| “I cannot tell whether this is urgent.” | “Materiality, deadline, uncertainty, and consequence are explicit.” |
| “I do not trust the AI conclusion.” | “The recommendation is grounded, bounded, editable, and source-linked.” |
| “I completed the task; I hope that is enough.” | “Implementation and verified outcome are clearly separated.” |
| “I will have to reconstruct this later.” | “The complete context and progress are safely preserved.” |

The primary emotional target is **calm agency**: the user understands what is happening, retains control, and can act without unnecessary cognitive or administrative burden.

## 1.2 Behavioral design principles

### Recognition before recall

The interface should show relevant Programs, Matters, bank objects, owners, source records, prior decisions, and evidence instead of asking the user to remember identifiers or navigate taxonomies.

### Ability before motivation

ClearSight should not rely on a user being highly motivated to complete compliance work. It should maximize ability by reducing steps, prefilling context, offering safe defaults, and requesting only unresolved information.

### One clear prompt at the right moment

Prompts should arise from a meaningful trigger—deadline, change, stale evidence, contradiction, threshold, request, or decision—not from a generic reminder schedule.

### Progressive disclosure

Show only what is needed for the current judgment. Evidence lineage, framework mappings, source limitations, technical metadata, and historical detail remain available without dominating routine work.

### Review by exception

Where approved sources already establish most of a population, the user should review only changed, missing, stale, contradictory, sampled, or high-risk items.

### Safe defaults and reversible progress

Use defaults when supported by policy and context, but make their origin visible. Allow save, resume, undo, supersession, and correction. Irreversible or external actions require proportional confirmation.

### Error prevention before error messaging

Prevent wrong-scope approval, duplicate submission, incomplete evidence, stale source use, conflicting identifiers, and unauthorized disclosure before submission whenever possible.

### Trust calibration

Do not make AI appear more certain than the evidence supports. Distinguish:

- source value;
- machine-extracted value;
- inferred recommendation;
- user-confirmed value;
- approved institutional conclusion.

### Interruption resilience

Bank users are frequently interrupted. Every multi-step workflow must preserve context, progress, unsent drafts, active scope, and the reason the user was acting.

### No urgency theatre

Do not use blinking red elements, countdown anxiety, excessive badges, or unread-count pressure. Urgency must be justified by deadline, materiality, customer harm, legal duty, or risk velocity.

### No dark patterns

Never manipulate users into approval, risk acceptance, disclosure, identity reveal, or AI adoption. Default actions must not weaken authority, privacy, evidence quality, or segregation of duties.

## 1.3 Five-minute interaction target

Routine, authorized, well-scoped tasks should normally complete in less than five minutes of active user effort.

The default interaction pattern should be:

```text
Understand context
→ inspect prefilled facts or recommendation
→ resolve exceptions
→ review result
→ submit, approve, redirect, or save
```

Complex work that cannot responsibly finish within five minutes must still reach a clear, saved, correctly routed next state within that time.

## 1.4 Role-specific mental models

### Executives and committees

Think in terms of material Programs, Matters, decisions, exposure, customer impact, deadlines, accountability, and verified outcomes.

### Compliance, risk, security, privacy, resilience, and legal teams

Think in terms of requirements, applicability, controls, evidence, exceptions, findings, obligations, cases, decisions, and institutional scope.

### Business, channel, branch, system, and control owners

Think in terms of the service, customer, account, branch, asset, vendor, process, or action they own—not abstract GRC architecture.

### Evidence respondents

Need one contextual question, known facts, acceptable proof, estimated effort, and an easy redirect path.

### Auditors and assurance teams

Need independent conclusions, samples, source lineage, version history, overrides, and point-in-time reconstruction.

---

# 2. Visual personality

ClearSight should feel:

- calm;
- direct;
- precise;
- premium;
- institutional;
- intelligent;
- source-grounded;
- future-facing without appearing speculative.

“Futuristic” means that the product understands context, eliminates repetitive assembly work, and presents complicated institutional relationships clearly. It does not mean neon cyberpunk styling, decorative holograms, excessive glass, animated graph edges, or science-fiction controls.

## 2.1 Composition principles

- Strong content hierarchy with generous but efficient spacing.
- One dominant purpose per page or panel.
- One clear primary action per state.
- Calm neutral surfaces with restrained semantic accents.
- Dense information only where the task requires a population or comparison.
- Cards for attention and summaries, not as a universal replacement for tables.
- Relationship paths and dependency summaries before large node graphs.
- Original sources and evidence shown in readable document or comparison surfaces.
- Light and dark themes designed independently but semantically equivalent.

## 2.2 Density modes

ClearSight supports two governed density modes:

### Comfortable

Used for:

- Today;
- Program overview;
- Matter summary;
- executive and committee work;
- decision review;
- mobile and tablet.

### Compact

Used for:

- population worklists;
- evidence reconciliation;
- regulatory provision review;
- source mapping;
- audit sampling;
- large operational tables.

Density changes spacing and row height, not information meaning or authorization.

## 2.3 Surface hierarchy

Use a small, stable set of surface levels:

1. **Canvas** — application background.
2. **Base surface** — primary work area.
3. **Subtle surface** — grouped sections, filters, quiet context.
4. **Raised surface** — focused cards, drawers, review panels.
5. **Protected surface** — privileged, sensitive, or restricted context.
6. **Transient surface** — menus, tooltips, popovers, command results.

Do not invent a new surface treatment for every component.

## 2.4 Glass, glow, and depth

Glass is permitted only for temporary or focused intelligence layers such as:

- command surface;
- recommendation overlay;
- focused decision panel;
- relationship explanation;
- simulation comparison.

Do not use backdrop blur on long text, tables, evidence documents, or every card.

Glow may indicate current focus, selected relationship path, or active analysis. It must never communicate severity by itself.

---

# 3. Color system

Color is semantic, restrained, and accessible. It must never be the only means of communicating status.

The following values are recommended starting tokens. Production tokens must be contrast-tested in context and may be adjusted while preserving their semantic role.

## 3.1 Neutral foundation

| Token | Light | Dark | Use |
|---|---:|---:|---|
| `canvas` | `#F4F7FA` | `#07111D` | Application background |
| `surface` | `#FFFFFF` | `#0D1826` | Main working surface |
| `surface-subtle` | `#EEF3F7` | `#132234` | Grouping, filters, quiet sections |
| `surface-raised` | `#FFFFFF` | `#17293D` | Cards, drawers, focused regions |
| `surface-protected` | `#F4F0FF` | `#191A32` | Restricted or privileged context |
| `text-strong` | `#111827` | `#F5F8FB` | Primary headings and values |
| `text` | `#273142` | `#D7E0E9` | Main body text |
| `text-muted` | `#687386` | `#97A5B5` | Metadata and supporting context |
| `border` | `#D8E0E8` | `#26394D` | Standard borders and dividers |
| `border-strong` | `#B9C4D0` | `#3A4D60` | Selected or emphasized boundaries |

## 3.2 Semantic colors

| Role | Light accent | Dark accent | Light tint | Dark tint | Meaning |
|---|---:|---:|---:|---:|---|
| Intelligence / context | `#0B7F95` | `#5BC6D6` | `#E7F7FA` | `#0D2A31` | New context, discovered relationship, analysis |
| Governance / authority | `#6557C5` | `#9E92F4` | `#F0EEFF` | `#211D3E` | Policy, control, approval, authority, governed AI |
| Material exposure | `#BE4058` | `#F17A8D` | `#FFF0F3` | `#3A1922` | Breach, failure, material gap, failed verification |
| Uncertainty / attention | `#99620E` | `#E9B75A` | `#FFF7E8` | `#352813` | Stale, contradictory, pending, approaching threshold |
| Verified / acceptable | `#177A55` | `#5BC996` | `#EAF8F1` | `#123127` | Verified outcome or sufficiently evidenced acceptable state |
| Informational | `#3569A8` | `#74A7E7` | `#EDF5FF` | `#172A43` | Neutral information or workflow guidance |

## 3.3 Color rules

- Green means a verified or accepted outcome supported by sufficient evidence. It does not mean uploaded, assigned, submitted, or implemented.
- Red/coral means material exposure, failure, breach, or failed verification. Missing data alone is normally amber or neutral.
- Amber means uncertainty, stale evidence, pending review, contradiction, or approaching threshold.
- Violet communicates governance, authority, policy, control, or approved automation.
- Cyan communicates contextual intelligence, discovered relationships, or active analysis.
- Status always includes text and, where useful, an icon or pattern.
- Large color fills should use tints; saturated accents are for borders, icons, small highlights, and key values.
- Avoid red/green-only comparisons.
- Do not use decorative gradients behind operational content.

## 3.4 Chart colors

Charts should use a restrained categorical palette derived from semantic roles plus neutral blue, teal, slate, and muted plum.

Rules:

- Preserve a consistent category color within one report or workspace.
- Use direct labels where possible instead of legends.
- Use pattern, line style, symbol, or annotation in addition to color.
- Do not assign severity colors to categories that are not severity states.
- Use confidence bands and uncertainty annotations where applicable.
- Heat maps are analytical tools, not the default executive representation of risk.

---

# 4. Font family and typography

## 4.1 Primary typeface

Recommended primary interface typeface:

```css
font-family: Inter, "Segoe UI Variable", "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
```

Use **Inter Variable** where deployment policy allows. It provides clear numerals, broad language support, strong enterprise readability, and stable rendering across dense and spacious layouts.

Do not introduce a decorative display face into operational screens.

## 4.2 Monospace

Use monospace only for technical identifiers, hashes, source references, code, query expressions, and machine-readable values:

```css
font-family: "IBM Plex Mono", "SFMono-Regular", Consolas, "Liberation Mono", monospace;
```

Do not use monospace for general metrics or financial values.

## 4.3 Numeric treatment

Use tabular numerals for:

- amounts;
- dates and time values;
- percentages;
- counts in aligned tables;
- deadlines;
- risk and control metrics.

Always show unit, currency, denominator, period, or meaning.

## 4.4 Typography scale

| Role | Size / line height | Weight | Use |
|---|---|---:|---|
| Display | `40 / 48 px` | 650 | Board mode, major report cover, rare product moments |
| Page title | `30 / 38 px` | 650 | Program, Work, Today, Explore page title |
| Object title | `24 / 32 px` | 620 | Program or Matter title |
| Section title | `20 / 28 px` | 620 | Major page section |
| Subsection title | `17 / 24 px` | 600 | Panels and grouped content |
| Card title | `16 / 24 px` | 600 | Cards and focused summaries |
| Body | `15 / 22 px` | 400–450 | Primary reading text |
| Compact body | `14 / 20 px` | 400–450 | Tables and dense workspaces |
| Label | `13 / 18 px` | 550–600 | Field and control labels |
| Metadata | `12 / 16 px` | 450–500 | Dates, source age, IDs, secondary context |
| Table | `13 / 18 px` | 400–500 | Compact operational tables |
| Large numeric | `32 / 38 px` | 650 | One dominant metric with context |

Rules:

- Do not use body text below 14 px except bounded metadata at 12 px.
- Avoid excessive all-caps. Use uppercase only for short regulated abbreviations or tiny categorical labels where accessible.
- Keep operational body line length around 55–80 characters.
- Long-form reports may use wider text blocks only where readability remains strong.
- Use weight, spacing, and placement before relying on color for hierarchy.
- Typography must scale with `rem` and support 200% zoom without clipping.

## 4.5 Responsive typography

Use controlled fluid scaling for page and object titles, for example:

```css
font-size: clamp(1.5rem, 1.2rem + 1vw, 1.875rem);
```

Body and table text should remain stable rather than shrinking aggressively on small screens.

---

# 5. Spacing, layout, radius, border, and elevation

## 5.1 Spacing system

Use a 4 px base with 8 px as the primary rhythm:

```text
4, 8, 12, 16, 24, 32, 40, 48, 64
```

Common use:

- 4 px: icon-to-label micro spacing.
- 8 px: related controls and inline metadata.
- 12 px: compact rows and grouped fields.
- 16 px: standard component padding.
- 24 px: card and section spacing.
- 32 px: major section separation.
- 48–64 px: page-level separation or report sections.

## 5.2 Radius

| Token | Value | Use |
|---|---:|---|
| `radius-small` | 8 px | Inputs, chips, compact controls |
| `radius-default` | 12 px | Cards, panels, menus |
| `radius-large` | 16 px | Drawers, focused review surfaces |
| `radius-xlarge` | 20 px | Rare prominent summary regions |
| `radius-pill` | 9999 px | Status chips and segmented controls |

Avoid excessive rounding that makes the product feel consumer-oriented.

## 5.3 Border and elevation

- Standard boundaries use 1 px semantic borders.
- Selection should combine border, subtle tint, and focus treatment.
- Shadows are quiet and short; they communicate layering, not decoration.
- Dense tables rely primarily on spacing and dividers rather than shadows.
- Protected or high-authority surfaces may use a distinct border and icon, not a dramatic glow.

---

# 6. Visual hierarchy

Every primary page should follow a predictable hierarchy:

1. **Global context** — tenant, legal entity, jurisdiction, role.
2. **Local scope** — Program, Matter, channel, service, population, period.
3. **Purpose and state** — page title and concise current state.
4. **Why attention is needed** — change, deadline, contradiction, gap, or decision.
5. **Primary action** — one obvious next step.
6. **Core content** — evidence, population, requirement, decision, or result.
7. **Supporting context** — source, owner, history, mappings, technical details.

## 6.1 Page header

A page header should normally contain:

- title;
- plain-language summary;
- scope and period;
- current state;
- owner or authority;
- one primary action;
- no more than two secondary actions before overflow.

## 6.2 State hierarchy

State should be shown in this order:

1. plain-language state;
2. consequence or meaning;
3. evidence condition;
4. next action;
5. source or timestamp.

Avoid presenting a colored score before explaining what it means.

## 6.3 Metric hierarchy

A dominant metric must always include:

- label;
- value;
- unit;
- period;
- denominator where relevant;
- comparison basis;
- source or freshness;
- uncertainty where material.

Do not build walls of equal-weight KPI cards.

---

# 7. Recommended UI/UX component types

## 7.1 Application shell

- Compact primary navigation.
- Visible tenant/legal-entity context.
- Global scoped search and command surface.
- Persistent user role or delegated authority where material.
- Global source or system degradation indicator only when it affects current work.
- Contextual breadcrumbs that reflect institutional scope, not URL structure.

## 7.2 Today components

### Attention item

Used for one Program or Matter requiring intervention.

Contains:

- direct title;
- why now;
- affected scope;
- evidence or data-quality state;
- owner and deadline;
- primary action.

### Quiet summary strip

Used for upcoming obligations, automated changes, or stable Programs. It should not compete visually with material attention items.

### No-material-change state

Must distinguish:

- no material change;
- no data;
- sources unavailable;
- not authorized;
- no assigned responsibilities.

## 7.3 Program components

### Program header

Shows scope, owner, current multi-dimensional state, next due event, evidence freshness, and active Matters.

### Compliance state matrix

Shows dimensions such as interpretation, applicability, control design, implementation, evidence, effectiveness, exceptions, assurance, and filing status.

Do not compress all dimensions into one unexplained percentage.

### Requirement table

Supports source provision, applicability, institutional requirement, owner, controls, evidence state, next review, and active Matters.

### Evidence coverage panel

Shows population, period, source freshness, sufficiency dimensions, contradictions, and missing proof.

### Review calendar or schedule

Used for recurring filings, tests, policy reviews, certifications, vendor reviews, and assurance activities.

## 7.4 Matter components

### Matter card

Used in queues and summaries. Contains type, what changed, affected scope, state, owner, deadline, evidence state, and next action.

### Matter workspace

Uses stable sections:

```text
Summary
Evidence
Decision or response
Actions
Outcome or submission
History
```

Sections may be tabs, anchored regions, or a split workspace depending on task complexity.

### Decision review panel

Shows options, evidence, uncertainty, affected scope, authority, conditions, side effects, expiry, and verification.

### Action plan

Shows owner, dependencies, blockers, due date, external execution state, implementation evidence, and verification state.

### Timeline

Used for history, evidence changes, decisions, communications, submissions, and point-in-time reconstruction.

## 7.5 Population and worklist components

Use tables or structured lists for large populations such as:

- accounts;
- customers;
- branches;
- ATMs;
- POS terminals;
- vendors;
- requirements;
- controls;
- evidence items;
- findings;
- incidents.

Required capabilities:

- sticky identifier columns;
- explicit total and filtered population;
- clear denominator;
- saved filters;
- density controls;
- row-level source/freshness state;
- keyboard navigation;
- safe selection summary;
- per-object authorization;
- bulk action manifest;
- export with scope and lineage.

## 7.6 Evidence and source components

### Source chip

Shows source identity, age, authority level, and health in a compact form.

### Source Profile

Shows owner, authoritative fields, limitations, scope, identifiers, freshness, mapping version, health, and affected conclusions.

### Evidence item

Shows original artifact, extracted observations, confirmation state, scope, period, sensitivity, and chain of custody.

### Evidence sufficiency panel

Shows relevance, authenticity, coverage, freshness, independence, completeness, consistency, reliability, and traceability.

### Contradiction compare

Shows conflicting records side by side with source authority, effective periods, affected claims, and resolution action.

### Lineage path

Shows source → requirement → control → evidence → conclusion → decision or report.

## 7.7 Input and capture components

- Searchable scoped combobox.
- Prefilled review form.
- Exception-only review list.
- Spreadsheet mapper.
- File and document intake.
- Photo/scan capture.
- Voice note with transcript review.
- Date/period selector.
- Population selector.
- Structured justification input.
- Redirect/delegate control.
- Save-and-resume checkpoint.
- Final assertion review.

## 7.8 AI recommendation components

### Recommendation panel

Must show:

- proposed recommendation;
- purpose;
- source basis;
- important assumptions;
- uncertainty and contradiction;
- affected objects;
- editable fields;
- primary action;
- reject or correct route.

### AI extraction review

Displays original source beside extracted fields. Machine values remain visually distinct from user-confirmed or approved values.

### Command surface

Global but not dominant. It can navigate, summarize, compare, draft, or propose actions. Side effects always transition into structured review.

## 7.9 Component-selection guide

| User need | Preferred component |
|---|---|
| Small set of items requiring attention | Cards or concise queue |
| Large population | Table or virtualized worklist |
| Repeated structured comparison | Matrix or table |
| Sequence with clear stages | Stepper or staged workspace |
| One bounded high-stakes review | Focused panel or full workspace |
| Source-versus-extraction review | Split view |
| Contradictory records | Side-by-side compare |
| Dependency explanation | Readable path, tree, or dependency list |
| Historical reconstruction | Timeline |
| Trend over time | Line or area chart with uncertainty |
| Composition | Stacked bar or segmented summary |
| Geographic distribution | Map only where location changes the decision |
| Regulatory or evidence lineage | Directed path, not free-form graph |

---

# 8. Input collection UX

## 8.1 Input strategy

The default order is:

```text
Retrieve from approved sources
→ infer or recommend from grounded context
→ prefill known values
→ ask only unresolved facts
→ validate during entry
→ show final assertions
→ submit with provenance
```

A blank form is the last resort, not the default.

## 8.2 Prefill behavior

Prefilled values must show:

- source;
- freshness;
- whether editable;
- whether machine-inferred;
- whether confirmation is required.

Do not silently overwrite a user correction when a source refreshes. Reconcile the new value and preserve history.

## 8.3 Question design

Questions should be:

- singular;
- concrete;
- scoped;
- answerable by the recipient;
- expressed in familiar banking language;
- tied to a reason and outcome.

Avoid:

- compound questions;
- abstract risk jargon;
- control IDs as the main prompt;
- requests for facts already available;
- broad “provide all evidence” prompts;
- large free-text boxes where a structured choice is possible.

## 8.4 Form length

Routine requests should normally contain no more than:

- one primary question;
- a small group of related fields;
- one optional explanation;
- one review step.

Longer forms must be divided by user intent and reveal sections conditionally.

## 8.5 Controlled inputs

Use searchable, scoped comboboxes for branches, services, systems, assets, vendors, owners, customers, accounts, controls, and requirements.

Requirements:

- human-readable labels plus identifiers;
- recent and likely choices;
- source and scope;
- keyboard access;
- no silent creation of new canonical entities;
- explicit “not found” or “request addition” path.

## 8.6 Spreadsheet input

For first use:

1. identify purpose and Source Profile;
2. choose sheet;
3. detect headers;
4. propose mappings;
5. preview sample rows;
6. resolve required errors;
7. confirm scope;
8. import valid observations;
9. route unresolved rows.

For repeat use, saved mappings and source settings should reduce the workflow to file selection, changed-field review, and confirmation.

## 8.7 Document input

- Preserve the original file.
- Detect type, language, version, and likely document class.
- Show extraction progress without blocking navigation.
- Present source text and extracted structures side by side.
- Require confirmation for material dates, thresholds, actors, obligations, and legal interpretations.

## 8.8 Photo and scan input

Guide the user toward visible facts:

- full object in context;
- serial or label region;
- location sign;
- screen state;
- seal or external condition;
- supporting document.

Show blur, glare, crop, and readability feedback during capture. Explain what metadata is collected. Allow retake and appropriate redaction. State clearly what the image can and cannot prove.

## 8.9 Voice and narrative input

Use voice or narrative for events, explanation, and field observations—not for basic structured identity that can be selected.

AI may transcribe and structure the input. The user must review material extracted facts before submission.

## 8.10 Validation

Validation should be progressive and specific:

- identify the exact problem;
- explain why it matters;
- preserve valid input;
- suggest a correction where safe;
- distinguish warning from blocking error;
- never reveal restricted data through validation messages.

## 8.11 Save, resume, and handoff

Every multi-step workflow should support:

- autosave;
- explicit saved state;
- safe draft ownership;
- resume at the same step;
- summary of what changed since last visit;
- assignment or handoff with preserved context;
- draft expiry or retention policy.

## 8.12 High-impact inputs

Risk acceptance, regulatory response, protected identity reveal, suspicious-reporting decisions, external submissions, account restrictions, and other high-impact actions require:

- exact scope;
- authority basis;
- evidence and uncertainty;
- side effects;
- approver identity;
- deliberate confirmation;
- immutable audit.

Confirmation must be proportional, not repetitive ceremony.

---

# 9. Interactivity and animation

Motion should explain hierarchy, relationship, progress, or state change. It must never delay action or create decorative noise.

## 9.1 Timing tokens

| Token | Duration | Use |
|---|---:|---|
| `motion-instant` | 80–100 ms | Press, hover, selection feedback |
| `motion-fast` | 140–180 ms | Menus, tooltips, small state changes |
| `motion-standard` | 200–240 ms | Drawers, accordions, panel transitions |
| `motion-emphasis` | 280–360 ms | Major context change or relationship reveal |

Avoid routine animation beyond 400 ms.

## 9.2 Easing

Recommended easing:

```css
--ease-standard: cubic-bezier(0.2, 0, 0, 1);
--ease-enter: cubic-bezier(0, 0, 0.2, 1);
--ease-exit: cubic-bezier(0.4, 0, 1, 1);
```

## 9.3 Appropriate motion

- Highlight the row or field updated by a source refresh.
- Animate a panel expansion to preserve spatial orientation.
- Trace a selected lineage or dependency path once.
- Show an item moving from pending to verified only after the state is accepted.
- Indicate background processing with bounded progress.
- Use subtle motion when switching scope to make the context change obvious.

## 9.4 Prohibited motion

- continuously animated graph edges;
- pulsing severity states;
- ambient floating particles;
- parallax in operational screens;
- long page transitions;
- animation that conceals loading;
- motion required to understand status;
- celebratory effects for compliance completion.

## 9.5 Loading and AI progress

Show deterministic data immediately. AI and source processing should appear as progressive layers:

- what is already available;
- what is being retrieved;
- what analysis remains;
- estimated stage, not false precision;
- cancel or continue-in-background where safe;
- manual fallback.

Avoid full-page spinners when partial context is available.

## 9.6 Reduced motion

Respect operating-system reduced-motion settings. Replace movement with immediate state change, opacity, border, or static path emphasis.

---

# 10. Responsiveness

## 10.1 Recommended breakpoints

| Mode | Width | Primary use |
|---|---:|---|
| Mobile | `< 768 px` | Capture, focused response, urgent review |
| Tablet | `768–1023 px` | Executive review, approval, meetings |
| Desktop | `1024–1439 px` | Full Program and Matter work |
| Wide desktop | `≥ 1440 px` | Split evidence, populations, deep analysis |
| Presentation / board | task-specific | Committee and board display |

Breakpoints are starting implementation values, not a substitute for testing actual content.

## 10.2 Desktop

Typical structure:

- compact navigation;
- main workspace;
- optional contextual panel;
- optional command surface.

Use split views for source review, evidence comparison, and reconciliation. Avoid permanently reserving large width for secondary navigation.

## 10.3 Tablet

- Preserve full decision context.
- Convert side panels to modal or full-height sheets.
- Keep touch targets at least 44 × 44 px.
- Reduce simultaneous columns but do not hide material scope or authority.
- Support committee annotation and approval.

## 10.4 Mobile

Mobile prioritizes:

- Respond/Capture;
- evidence requests;
- photos and scans;
- short approvals with sufficient context;
- incident updates;
- protected reporting;
- urgent Matter triage.

Do not compress full graph exploration, large report builders, or 30-column tables onto mobile.

## 10.5 Responsive tables

On smaller screens:

- preserve the primary identifier and state;
- prioritize columns by task;
- allow explicit horizontal scroll where comparison is essential;
- provide a focused row-detail sheet;
- retain selection and bulk-operation clarity;
- do not convert every row into a visually heavy card.

## 10.6 Low bandwidth and offline use

- Load text and structured context before media previews.
- Support resumable uploads.
- Cache only policy-approved data.
- Make queued, uploaded, submitted, and synchronized states distinct.
- Preserve capture time separately from upload time.
- Route synchronization conflicts instead of overwriting newer data.

## 10.7 Large displays and meeting mode

Board and control-room views should emphasize:

- material Program movement;
- significant Matters;
- evidence limitations;
- decisions and owners;
- deadlines;
- verified outcomes.

They must not become walls of blinking metrics. Meeting mode should support point-in-time freeze, privacy-aware presentation, and drill-down without exposing restricted details accidentally.

---

# 11. Reports

Reports are governed, point-in-time representations of Programs, Matters, decisions, evidence, and outcomes. They are not screenshots of dashboards.

## 11.1 Report types

### Executive and board report

Focuses on:

- material change;
- appetite and exposure;
- important Programs and Matters;
- decisions required;
- evidence limitations;
- remediation effectiveness;
- accountability.

### Regulatory or authority response

Focuses on:

- exact request or requirement;
- scope and period;
- approved interpretation;
- included records and evidence;
- exclusions and rationale;
- approvals and signatory;
- transmission and acknowledgement.

### Program report

Focuses on:

- applicable requirements;
- control coverage;
- evidence freshness and sufficiency;
- exceptions;
- review and filing status;
- active Matters;
- assurance conclusions.

### Matter report

Focuses on:

- what happened or changed;
- affected scope;
- evidence and contradiction;
- decisions;
- actions;
- response or outcome;
- history.

### Audit and assurance report

Focuses on:

- objective and scope;
- population and sample;
- evidence sources;
- findings;
- management responses;
- independent conclusion;
- limitations;
- lineage.

### Operational work report

Focuses on:

- due and overdue work;
- owners;
- blockers;
- source health;
- evidence requests;
- verification queue.

## 11.2 Report grammar

A formal report should normally follow:

1. title, scope, period, version, and confidentiality;
2. executive summary;
3. what changed since the previous version;
4. current state and important limitations;
5. material Programs, Matters, or findings;
6. decisions and actions;
7. outcome or readiness;
8. source and evidence notes;
9. appendix and manifest.

## 11.3 Report visual hierarchy

- Use a clear title and scope block.
- Keep executive summaries concise and left aligned.
- Use section dividers and generous report spacing.
- Limit large metrics to those that support a decision.
- Use tables for traceable detail.
- Use charts only when they reveal trend, comparison, concentration, or composition.
- Every chart includes title, unit, period, source, and accessible summary.
- Use footnotes for source age, exclusions, methodology, and uncertainty.

## 11.4 Report state and versioning

Reports support:

- live draft;
- review;
- approved;
- frozen point-in-time version;
- submitted;
- acknowledged;
- superseded.

A later source change must not silently alter an approved or submitted report. It creates a new version or amendment.

## 11.5 AI-assisted report drafting

AI may draft:

- executive summaries;
- change explanations;
- management-response language;
- source-linked narratives;
- appendix descriptions.

Requirements:

- every material statement links to approved facts or evidence;
- assumptions and missing information remain visible;
- generated narrative is editable;
- reviewer and approval are explicit;
- AI must not invent legal conclusions, penalties, deadlines, or outcomes.

## 11.6 Report formats

Support as appropriate:

- interactive HTML;
- accessible PDF;
- structured XLSX or CSV for populations;
- machine-readable JSON or API packages;
- controlled evidence bundles.

PDF and printed reports should preserve hierarchy in grayscale and at common page sizes. Spreadsheet exports must include scope, period, source, version, and manifest information rather than raw unlabeled rows.

## 11.7 Report security

Every export or report generation must:

- re-evaluate authorization;
- record requester and purpose;
- apply field and relationship restrictions;
- preserve point-in-time versions;
- include classification and watermark where required;
- exclude hidden protected content;
- generate a manifest;
- log download or submission.

## 11.8 Report review experience

The review interface should show:

- changed sections since prior version;
- unresolved source or evidence warnings;
- narrative statements without supporting lineage;
- required approvers;
- comments and decisions;
- final export preview.

Review by exception is preferred over requiring reviewers to reread unchanged content.

---

# 12. Accessibility, inclusion, and localization

ClearSight must meet WCAG 2.2 AA at minimum.

Required:

- complete keyboard operation;
- visible focus;
- semantic headings and landmarks;
- correctly labelled controls;
- non-color status indicators;
- target sizes suitable for touch;
- reduced motion;
- screen-reader announcements for asynchronous changes;
- accessible chart summaries;
- error prevention and recovery;
- sufficient glass and overlay contrast;
- support for 200% zoom;
- no essential hover-only interactions.

Localization must support:

- local date, number, currency, and time formats;
- multiple time zones;
- long translations;
- multilingual source documents;
- right-to-left readiness where required;
- consistent legal and regulatory terminology per jurisdiction.

Avoid idioms, culturally narrow metaphors, and unexplained abbreviations.

---

# 13. Design-system implementation requirements

The design system must expose semantic tokens for:

- color;
- typography;
- spacing;
- radius;
- border;
- elevation;
- blur;
- motion;
- breakpoints;
- density;
- focus;
- iconography;
- charts;
- reports.

Core production components should include:

- application shell;
- context header;
- Program header and state matrix;
- Matter card and workspace;
- attention queue;
- requirement table;
- population worklist;
- source chip and Source Profile;
- evidence item and sufficiency panel;
- contradiction compare;
- lineage path;
- recommendation panel;
- decision review;
- action plan;
- verification panel;
- timeline;
- spreadsheet mapper;
- photo/scan capture;
- prefilled form;
- scoped combobox;
- save/resume state;
- report viewer and report builder;
- export manifest;
- empty, unauthorized, degraded, and offline states.

Required variants:

- light and dark;
- comfortable and compact density;
- desktop, tablet, and mobile where relevant;
- default, hover, focus, active, selected, disabled, loading, warning, error, protected, stale, contradictory, implemented, awaiting verification, verified, and failed verification.

No production feature should invent ad hoc colors, typography, shadows, or states when a semantic token or component variant can be extended.

---

# 14. Visual and UX anti-patterns

Do not introduce:

- a wall of KPI cards;
- cards for every table row;
- multiple competing primary actions;
- decorative glass on every surface;
- neon glow or cyberpunk styling;
- huge empty hero areas in operational screens;
- full-page chat as the application shell;
- generic “AI confidence” without source and evidence state;
- hidden actions available only on hover;
- tiny low-contrast metadata;
- dense forms mirroring database columns;
- broad recurring questionnaires where source integration or targeted questions are possible;
- irreversible bulk actions without clear scope and manifest;
- red/amber/green as the only state language;
- report screenshots presented as governed reporting;
- animation that delays work or creates anxiety;
- a “successful” state for upload, integration, or task completion that implies compliance or verified outcome.

---

# 15. Design review checklist

Before approving a screen, component, or workflow, verify:

## Behavioral clarity

- Can the user explain why they are here?
- Is the active scope and period clear?
- Is the next action obvious?
- Does the experience reduce uncertainty rather than merely display data?

## Effort

- Has the system assembled all available context first?
- Are known values prefilled?
- Is the user reviewing exceptions rather than reconstructing the whole record?
- Can routine work complete within five minutes of active effort?
- Can complex work reach a saved next state within five minutes?

## Trust

- Are sources, freshness, authority, and limitations visible?
- Are machine-extracted, inferred, user-confirmed, and approved values distinct?
- Is contradictory evidence visible?
- Can the user correct or reject AI output?

## Visual hierarchy

- Is there one dominant purpose and primary action?
- Are headings, body, labels, and metadata clearly differentiated?
- Are semantic colors used correctly?
- Does the layout remain calm at high information density?

## Inputs

- Are questions singular, concrete, and necessary?
- Is validation progressive?
- Can the user redirect, save, resume, and review final assertions?
- Are bulk and high-impact actions proportional and explicit?

## Responsiveness and accessibility

- Does the full journey work with keyboard and screen reader?
- Does it remain usable at 200% zoom?
- Is mobile focused on appropriate tasks rather than compressed desktop complexity?
- Are status and charts understandable without color?

## Reports

- Is the report a governed point-in-time object rather than a screenshot?
- Are scope, version, source, period, and limitations explicit?
- Can changed content be reviewed by exception?
- Are exports authorized, classified, and manifested?

---

# 16. Final standard

A ClearSight interface has succeeded when:

- the system appears to understand the bank before asking questions;
- a routine user can complete their work in a few clear steps;
- a specialist can inspect every source, assumption, and contradiction;
- a decision-maker can understand material meaning in seconds;
- a regulator or auditor can trace a conclusion to original evidence;
- the experience remains calm, accessible, and useful under uncertainty;
- visual polish makes the work clearer without hiding its seriousness.

> **ClearSight should make high-accountability governance work feel smaller, more prepared, and easier to complete—without making it less rigorous.**