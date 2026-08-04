# ClearSight Experience Principles

This document defines the product taste, interaction model, information hierarchy, and visual non-regression rules for ClearSight.

The target is a 2030-quality enterprise interface: not decorative science fiction, but an interface that appears to understand institutional context, minimizes work, explains complexity, and remains calm under high-stakes use.

---

# 1. Experience goal

A ClearSight user should feel that:

- the system already knows the relevant institutional context;
- only material matters are being presented;
- every conclusion can be explained;
- uncertainty is visible without being overwhelming;
- the next accountable action is clear;
- AI is reducing effort rather than creating another interface to operate;
- and the interface will preserve the decision accurately.

The primary interaction grammar is:

> **Brief → Explain → Act → Prove**

This pattern must remain recognizable across risk, compliance, resilience, security, third-party, incident, audit, and protected-reporting journeys.

---

# 2. Product personality

ClearSight should feel:

- **calm** — no artificial urgency or constant visual alarm;
- **precise** — numbers, states, owners, and time scopes are explicit;
- **institutional** — suitable for a bank boardroom, risk committee, regulator, or audit team;
- **intelligent** — the interface anticipates context and removes unnecessary input;
- **premium** — refined typography, spacing, motion, and detail;
- **defensible** — evidence and reasoning are always within reach;
- **restrained** — depth and glow communicate meaning rather than decoration.

ClearSight must not feel:

- like a generic admin template;
- like a security operations console filled with alerts;
- like a consumer finance application;
- like a social feed;
- like a gamified compliance tool;
- or like a neon cyberpunk concept.

---

# 3. Experience architecture

## 3.1 Today

The default landing surface is a role-specific brief, not a dashboard catalog.

It should show:

- material changes since the user’s last relevant review;
- decisions requiring the user’s authority;
- appetite breaches or fast-moving exposures;
- critical evidence gaps;
- actions whose expected outcome is not being achieved;
- upcoming obligations or deadlines that require intervention;
- and notable changes that were safely automated.

The default view should usually contain between three and seven material cards.

The system may provide an expanded monitoring mode, but executive calm is the default.

## 3.2 Explore

Explore provides deep, connected investigation through:

- institutional graph;
- risk portfolio;
- service and dependency maps;
- regulatory lineage;
- controls and evidence;
- incidents and losses;
- scenarios and counterfactuals;
- and time-based reconstruction.

Explore is not a collection of module homepages. It is a connected inquiry surface.

## 3.3 Act

Act contains:

- decision queue;
- approvals;
- delegated work;
- remediation;
- evidence requests;
- investigations;
- risk acceptances;
- exceptions;
- and governed automation.

The user should see why an action exists and what outcome it must achieve.

## 3.4 Prove

Prove contains:

- verification contracts;
- outcome evidence;
- control tests;
- assurance conclusions;
- evidence rooms;
- audit and examination lineage;
- point-in-time exports;
- and decision history.

## 3.5 Govern

Govern contains high-authority configuration such as:

- risk appetite;
- taxonomies and ontologies;
- legal-entity and jurisdiction boundaries;
- committee and authority matrices;
- AI operator capabilities;
- model routing;
- evidence policy;
- retention and legal hold;
- authorization policy;
- and integration trust settings.

Govern should not become a dumping ground for ordinary product settings.

---

# 4. Core interaction patterns

## 4.1 Material decision card

A material decision card is the primary unit of executive interaction.

It contains, in order:

1. **Material change** — one sentence describing what changed.
2. **Why now** — the reason it requires attention at this moment.
3. **Affected scope** — service, customers, entity, jurisdiction, system, vendor, or obligation.
4. **Risk movement** — current state, expected direction, velocity, and time-to-impact.
5. **Evidence state** — sufficient, weak, stale, contradictory, or pending.
6. **Authority** — who owns the decision and when it is due.
7. **Recommended handling** — one primary action with alternatives available.

A card must not contain every supporting metric. Deep context is shown through Explain.

### Card rules

- One dominant message.
- One primary next action.
- No more than two secondary actions in the collapsed state.
- No unexplained scores.
- No status color without a textual or icon equivalent.
- No generic “view details” as the only action.
- No green state unless the outcome is verified.

## 4.2 Explain drawer or workspace

Explain must reveal the basis of the card without forcing navigation into many pages.

The workspace should support these layers:

### Layer 1: concise explanation

- what changed;
- why it matters;
- affected scope;
- current evidence quality;
- and recommended decision.

### Layer 2: relationship path

- relevant graph nodes and edges;
- critical dependencies;
- propagation path;
- and concentration.

### Layer 3: evidence and reasoning

- claims;
- supporting and contradicting evidence;
- source lineage;
- assumptions;
- confidence;
- rule or model basis;
- and prior conclusions.

### Layer 4: history and reconstruction

- previous state;
- decisions;
- overrides;
- changes in evidence;
- and realized outcomes.

## 4.3 Evidence request

A staff-facing evidence request should feel like a short, contextual work message—not a GRC assessment.

It includes:

- why the recipient is being asked;
- what the system already knows;
- the smallest unresolved question;
- acceptable evidence forms;
- estimated effort;
- deadline and consequence;
- confidentiality classification;
- and an easy route to redirect to a better source.

Example:

> **Confirm four privileged users**  
> We found the current Treasury Operations user list and the previous review. Four accounts have no current approval. Confirm whether each still requires access and attach an approval for any exception. Estimated time: 3 minutes.

Avoid:

- control IDs as the main language;
- broad free-text prompts;
- asking for already known data;
- requiring users to navigate the full risk object;
- and repeated requests that could have been deduplicated.

## 4.4 Evidence sufficiency view

Evidence quality must be understandable.

Use dimensions such as:

- relevance;
- authenticity;
- coverage;
- freshness;
- independence;
- completeness;
- consistency;
- reliability;
- and traceability.

Do not compress these into a single unexplained percentage.

A concise state may be shown, but the dimension breakdown and evidence basis must be available.

## 4.5 Contradiction view

Contradiction is a first-class state, not an error message.

The view should show:

- disputed claim;
- conflicting evidence side by side;
- effective periods;
- source authority;
- affected decisions and conclusions;
- unresolved questions;
- assigned resolver;
- and time sensitivity.

The interface must not silently choose one source.

## 4.6 Verification contract

A remediation view must visibly separate:

- work to perform;
- implementation evidence;
- expected outcome;
- measurement source;
- observation period;
- current observed result;
- and acceptance authority.

Completion and verification must have separate visual states.

## 4.7 Natural-language command surface

The command surface is global but not dominant.

It can support:

- inquiry;
- navigation;
- drafting;
- comparison;
- simulation;
- and proposing governed actions.

It must not become the only way to operate the system.

Responses should include:

- concise answer;
- source and time scope;
- confidence and missing information;
- affected objects;
- and safe next actions.

When a command could create a material side effect, the system must transition into a structured review and approval surface.

---

# 5. Visual language

## 5.1 Overall composition

The visual model should combine:

- near-black or deep navy institutional surfaces in dark mode;
- clean warm or cool neutral surfaces in light mode;
- carefully controlled transparency;
- thin borders and subtle elevation;
- generous spacing;
- high legibility;
- and focused semantic accents.

The supplied Archer-style visual reference is useful for:

- dark institutional framing;
- restrained cyan edge emphasis;
- clear numerical hierarchy;
- colored relationship blocks;
- and visual flow between obligations and controls.

ClearSight should not copy that screen. It should improve on it by:

- reducing default density;
- making evidence state and decision ownership more prominent;
- replacing decorative flow with interactive causal explanation;
- ensuring full light-mode parity;
- using stronger accessibility;
- and preventing color from carrying meaning alone.

## 5.2 Surface hierarchy

Use a small number of surface levels:

1. **Canvas** — application background.
2. **Primary surface** — main work area.
3. **Raised surface** — focused cards, drawers, or command results.
4. **Protected surface** — sensitive or privileged context.
5. **Transient surface** — menus, tooltips, and short-lived overlays.

Do not create a separate visual treatment for every component.

## 5.3 Glass

Glass is structural, not decorative.

Appropriate uses:

- command surface over current context;
- focused decision layer;
- relationship overlay;
- protected review context;
- and temporary simulation comparison.

Inappropriate uses:

- every card;
- long text surfaces;
- dense tables;
- evidence documents;
- and low-power mobile contexts where blur harms performance.

Every glass surface must preserve readable contrast without depending on the background image or content.

## 5.4 Glow

Glow may indicate:

- active intelligence;
- selected graph path;
- live but verified connection;
- or current focus.

Glow must not indicate severity by itself and must not surround every interactive element.

## 5.5 Semantic color

A canonical starting palette should be tokenized around roles, not fixed component colors.

### Intelligence / context — cyan

Used for:

- discovered relationships;
- live analysis;
- selected graph paths;
- or contextual explanations.

### Governance / control — violet

Used for:

- approvals;
- policy;
- controls;
- authority;
- and governed automation.

### Material exposure — coral or red

Used for:

- appetite breach;
- severe control gap;
- failed verification;
- and material incident state.

### Uncertainty — amber

Used for:

- weak or stale evidence;
- pending verification;
- approaching threshold;
- and unresolved contradiction.

### Verified outcome — green

Used only when:

- outcome evidence meets the verification contract;
- acceptance authority has approved the result;
- and no unresolved contradictory evidence invalidates the conclusion.

### Neutral

Used for:

- informational state;
- unassessed state;
- baseline data;
- and historical context.

## 5.6 Typography

Typography should communicate authority and clarity.

Requirements:

- modern neutral sans-serif suitable for long enterprise use;
- tabular numerals for metrics and financial values;
- clear distinction between labels, values, explanation, and metadata;
- no excessive all-caps;
- no tiny text for material context;
- no ultra-light weights that reduce readability;
- and stable line-height across languages.

Suggested roles:

- display;
- page title;
- section title;
- card title;
- body;
- compact body;
- label;
- metadata;
- code/identifier;
- numeric emphasis.

All typography must come from tokens.

## 5.7 Icons

Icons should be:

- simple;
- geometric;
- optically balanced;
- consistent in stroke and corner style;
- and recognizable without decorative complexity.

Do not use different icon families in the same product.

Risk state must not depend on icons alone.

## 5.8 Charts and relationship views

Prefer:

- causal and dependency paths;
- service maps;
- obligation-to-control-to-evidence lineage;
- risk movement over time;
- confidence bands;
- evidence coverage;
- concentration views;
- scenario deltas;
- and before/after treatment projections.

Use heat maps as one analytical view, not the primary executive truth.

Every chart must provide:

- clear title;
- units;
- time period;
- source;
- accessible textual summary;
- and explanation of uncertainty.

---

# 6. Layout and responsive behavior

## 6.1 Desktop

Desktop is optimized for investigation and decision work.

A typical layout may include:

- compact navigation rail;
- main briefing or work surface;
- contextual explain panel;
- and optional command surface.

Avoid permanently reserving wide space for secondary navigation.

## 6.2 Tablet

Tablet supports executive review, approval, evidence inspection, and meeting use.

- Cards remain readable without desktop density.
- Relationship views simplify progressively.
- Important actions remain reachable with touch.
- Side panels become modal workspaces where appropriate.

## 6.3 Mobile

Mobile is primarily for:

- evidence capture;
- short review;
- approval with context;
- protected reporting;
- incident updates;
- and urgent material notifications.

Do not squeeze full desktop graph exploration onto mobile.

Mobile evidence capture should support:

- camera;
- document scan;
- screenshot;
- voice note;
- short video;
- structured confirmation;
- and offline or unstable-network recovery where feasible.

## 6.4 Large displays

Boardroom and control-room modes may use large screens, but should remain calm.

Large displays should emphasize:

- material portfolio movement;
- critical service status;
- appetite and evidence state;
- decisions and owners;
- and scenario comparison.

They should not become walls of blinking metrics.

---

# 7. State design

Every feature must design these states deliberately:

- loading;
- streaming or partially available intelligence;
- empty;
- no material change;
- insufficient evidence;
- contradictory evidence;
- pending approval;
- delegated;
- executing;
- awaiting verification;
- verified;
- failed verification;
- stale;
- superseded;
- unauthorized;
- offline;
- integration degraded;
- and AI unavailable.

“No data” must be distinguished from “no risk,” “not assessed,” and “not authorized.”

## 7.1 AI latency

AI output should not freeze the interface.

- Show immediately available deterministic context first.
- Indicate what analysis is still in progress.
- Allow cancellation where safe.
- Provide timeout and fallback behavior.
- Preserve manual workflows when AI is unavailable.

## 7.2 Integration degradation

When a source is delayed or unavailable, show:

- affected source;
- last successful synchronization;
- affected conclusions;
- confidence impact;
- and recovery status.

Do not silently display stale data as current.

---

# 8. Evidence capture experience

## 8.1 Recipient experience

The recipient should not need to understand the entire GRC context.

A request should explain:

- what is needed;
- why the recipient is likely the best source;
- what is already known;
- what acceptable proof looks like;
- confidentiality;
- and estimated effort.

## 8.2 Redirect and delegation

Recipients must be able to:

- answer;
- redirect to a better source;
- identify that the request is not applicable;
- request clarification;
- provide partial evidence;
- or raise a conflict or sensitivity concern.

The system must preserve the routing history.

## 8.3 Progressive evidence validation

Validation should occur during capture:

- file readability;
- date coverage;
- missing pages;
- required approvals;
- duplicate detection;
- sensitivity detection;
- and obvious mismatch to the claim.

Do not wait until final submission to reveal avoidable problems.

## 8.4 Multimodal capture

When AI extracts information from image, voice, video, or documents:

- show the extracted interpretation;
- allow correction;
- retain the original source;
- record extraction model/version;
- and distinguish user-confirmed fields from machine-inferred fields.

---

# 9. Protected reporting experience

The whistleblower and confidential reporting experience must feel safe, simple, and independent from the normal internal application.

## 9.1 Entry

The portal should clearly explain:

- anonymous and identified options;
- how anonymous communication works;
- what information is collected;
- confidentiality limits;
- emergency or immediate-danger boundaries;
- and how to retain the secure case token.

Do not use manipulative language or imply guaranteed outcomes.

## 9.2 Submission

Use a guided but non-leading sequence:

- what happened;
- where or which area is affected;
- when it occurred;
- whether it is ongoing;
- who may be able to verify it;
- available evidence;
- immediate customer, safety, financial, or legal impact;
- and preferred communication mode.

Avoid forcing the reporter to categorize complex legal or risk domains.

## 9.3 Case token

Anonymous reporters should receive a secure case token and recovery guidance.

The interface must never expose internal investigator identities or protected routing information unless policy allows.

## 9.4 Investigator experience

Investigators see:

- original report;
- protected identity state;
- classification and sensitivity;
- conflicts;
- related cases;
- evidence and chain of custody;
- two-way messages;
- urgent obligations;
- and permitted next actions.

Identity reveal must be a separate privileged workflow.

---

# 10. Board and committee experience

Board and committee surfaces should prioritize:

- material portfolio movement;
- appetite position;
- critical service resilience;
- concentrated dependencies;
- major decisions and accepted risks;
- remediation effectiveness;
- evidence quality;
- emerging risk;
- and management accountability.

Committee packs should be generated from live governed data but support:

- review and sign-off;
- commentary;
- point-in-time freeze;
- version comparison;
- and source lineage.

A board pack is not merely a PDF export of dashboards.

---

# 11. Accessibility and inclusion

ClearSight must meet WCAG 2.2 AA at minimum.

Requirements include:

- complete keyboard navigation;
- visible focus;
- meaningful heading order;
- correct labels and descriptions;
- screen-reader announcements for async state changes;
- non-color status indicators;
- accessible chart summaries;
- target sizes suitable for touch;
- reduced-motion support;
- error prevention and recovery;
- sufficient contrast in glass surfaces;
- and language that does not require specialist knowledge where avoidable.

Evidence and reporting flows must support:

- multilingual content;
- assistive technologies;
- low-bandwidth environments;
- and users with limited familiarity with risk terminology.

---

# 12. Performance experience

Perceived performance is part of product trust.

## 12.1 Required behaviors

- Application shell appears immediately from cached or server-rendered state where appropriate.
- Deterministic data is not blocked by AI processing.
- Long analysis shows progress and source acquisition state.
- Lists use pagination or virtualization.
- Graph rendering progressively reveals detail.
- Evidence upload is resumable.
- Optimistic UI is used only where rollback is safe and understandable.
- Layout remains stable while data loads.

## 12.2 Visual performance

- Avoid excessive backdrop blur.
- Avoid large animated gradients.
- Avoid continuously animated graph edges.
- Use GPU-intensive effects only on focused, bounded surfaces.
- Test on enterprise laptops, integrated GPUs, remote desktops, and mobile devices.

---

# 13. Design-system requirements

The design system must include:

## Foundations

- color roles;
- typography;
- spacing;
- radius;
- border;
- elevation;
- blur;
- motion;
- breakpoints;
- density;
- and iconography.

## Core components

- application shell;
- command surface;
- material decision card;
- evidence state badge;
- evidence sufficiency panel;
- relationship path;
- risk movement indicator;
- authority and owner chip;
- verification contract panel;
- source lineage list;
- contradiction compare view;
- protected-content surface;
- approval review;
- timeline;
- data table;
- filters;
- forms;
- empty and degraded states;
- graph controls;
- and export manifest.

## Required variants

- light and dark;
- default and compact density;
- desktop, tablet, and mobile;
- normal, hover, focus, active, selected, disabled, loading, error, warning, protected, and verified states.

No production feature may invent ad hoc visual tokens when a semantic token can be added to the system.

---

# 14. Visual anti-patterns

Do not introduce:

- a wall of KPI cards;
- multiple competing primary colors;
- decorative glass on every component;
- constant neon glow;
- oversized empty hero areas inside operational screens;
- 3D charts without decision value;
- unexplained percentages;
- status conveyed only by red/amber/green;
- tiny low-contrast metadata;
- long centered body text;
- hidden actions appearing only on hover;
- inaccessible custom controls;
- excessive modal stacking;
- full-page chat as the primary product shell;
- or dense forms mirroring database schemas.

---

# 15. Functional anti-patterns visible in the interface

The interface must not normalize poor domain behavior such as:

- closing a finding with no effectiveness evidence;
- approving risk without showing authority and expiry;
- showing AI output without sources;
- treating missing data as low risk;
- hiding contradictory evidence;
- displaying stale integration data as current;
- showing a protected reporter’s identity outside a privileged flow;
- asking users for facts already available from integrated systems;
- or presenting a single score as objective truth.

---

# 16. Golden screens

The following screens require maintained design references and automated visual regression tests once implemented:

1. Executive Today brief
2. Material decision card — collapsed and expanded
3. Risk Explain workspace
4. Institutional dependency graph
5. Evidence micro-request on desktop and mobile
6. Evidence sufficiency and contradiction view
7. Decision approval with options and authority
8. Verification contract and outcome view
9. Whistleblower intake
10. Anonymous reporter follow-up
11. Protected investigator case
12. Regulatory source-to-evidence lineage
13. Board or committee pack review
14. AI operator review and approval
15. Integration degraded state
16. No material change state

Each golden screen must be maintained in:

- light mode;
- dark mode;
- desktop;
- and the relevant mobile or tablet breakpoint.

---

# 17. Design review checklist

Before approving a UI change, verify:

## Product clarity

- Is the user’s current decision obvious?
- Is the reason for materiality visible?
- Is evidence state understandable?
- Is the accountable owner or authority clear?
- Does the workflow end in proof rather than completion?

## Effort

- Is known information prefilled?
- Are unnecessary fields removed?
- Can the system collect the evidence automatically?
- Can the request be answered in fewer steps?

## Trust

- Are sources and time scope available?
- Is uncertainty visible?
- Are stale and contradictory states explicit?
- Can the user understand what AI did?

## Visual quality

- Is the screen calm?
- Is there one dominant hierarchy?
- Are glass and glow semantic?
- Does light mode feel equally designed?
- Does the screen remain clear at 125–200% zoom?

## Accessibility

- Is the full journey keyboard-operable?
- Are focus and errors visible?
- Is status available without color?
- Are charts explained textually?
- Is reduced motion respected?

## Performance

- Is critical information available before AI completes?
- Are long lists virtualized or paginated?
- Are effects inexpensive?
- Does the layout remain stable?

---

# 18. Final standard

A ClearSight screen should not merely look modern.

It should make an institutional risk problem feel:

- smaller;
- clearer;
- better evidenced;
- more governable;
- and easier to resolve.

The design has succeeded when a senior executive can understand the material issue in seconds, an expert can inspect every assumption, a staff member can provide evidence with minimal effort, and an auditor can reconstruct the outcome without needing a separate explanation.