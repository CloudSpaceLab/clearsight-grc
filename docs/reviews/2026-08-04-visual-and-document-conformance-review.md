# ClearSight Visual and Documentation Conformance Review

**Review date:** 2026-08-04  
**Scope:** README, AGENTS, product documents, architecture documents, implementation plan, and acceptance tests.

This review evaluates whether the repository consistently supports the simplified, situation-first ClearSight product model and whether the visual specification is sufficient for real bank operation across regional, national, and multinational institutions.

---

# 1. Executive finding

The original documentation had a strong visual taste and unusually good controls around evidence, AI, authority, accessibility, and verified remediation.

The principal non-conformity was structural:

- the updated README described a situation-first product;
- most supporting documents still described a graph/module-first product;
- the visual system emphasized executive decision cards and graph explanation but under-specified operational populations, imports, source quality, reconciliation, and branch capture;
- the implementation plan could therefore produce an impressive executive shell before solving the data and workflow friction that determines bank adoption.

The corrected canonical direction is:

> Users operate through bounded bank risk situations, focused capture, population reconciliation, authorized decisions, and verified outcomes. Graphs, evidence engines, decision ledgers, workflows, and AI operators remain internal platform capabilities rather than mandatory navigation concepts.

---

# 2. Important visual aspects that were missing or under-specified

## 2.1 Persistent scope and context anchoring — critical

A bank user can act in the wrong legal entity, country, channel, branch, merchant population, or period if context is subtle.

Required correction:

- visible scope header or breadcrumb;
- effective period and data age;
- role or delegated authority where relevant;
- deliberate context switching;
- warning when drafts or selections belong to another scope;
- scope re-confirmation before approval, export, bulk action, or submission.

## 2.2 Operational population views — critical

ATM, POS, account, merchant, branch, vendor, control, and exception workflows operate on populations. Cards and graph views alone are insufficient.

Required correction:

- first-class worklists and tables;
- clear numerator and denominator;
- resolved, unresolved, stale, contradictory, excluded, and unauthorized counts;
- sticky identifiers and headers;
- keyboard navigation;
- compact density;
- virtualization or pagination;
- safe selection and bulk action review.

## 2.3 Spreadsheet mapping and reconciliation — critical

Excel and CSV will remain legitimate bank integration channels, especially during early deployment and in regional institutions.

Required correction:

- sheet and purpose selection;
- column mapping;
- sample preview;
- type validation;
- scope confirmation;
- duplicate and identifier matching;
- unresolved-row queue;
- import summary and rollback reference;
- visible distinction among upload, parse, map, observation acceptance, and evidence sufficiency.

## 2.4 Source authority and data-quality visibility — critical

An automated integration is not necessarily authoritative, current, complete, or correct.

Required correction:

- Source Profile;
- authoritative fields and limitations;
- owner;
- scope;
- expected and current freshness;
- health and last successful synchronization;
- known limitations;
- unresolved mappings;
- affected conclusions.

## 2.5 Reconciliation and entity matching — high

The previous design acknowledged contradiction but not the everyday matching workflow required to resolve asset, merchant, owner, vendor, branch, and system records.

Required correction:

- side-by-side source records;
- normalized identifiers;
- proposed match explanation;
- provisional versus confirmed state;
- merge and unmerge history;
- downstream impact.

## 2.6 Photo and scan capture boundaries — high

The previous documentation supported mobile media but did not sufficiently define what AI-visible evidence can establish.

Required correction:

- capture framing and quality guidance;
- blur, glare, crop, and readability checks;
- explicit metadata/location notice;
- unnecessary background and personal-data minimization;
- extracted-region review;
- machine-inferred versus user-confirmed state;
- bounded claims such as serial-number match rather than “AI verified secure.”

## 2.7 Safe bulk operations — high

Banks need bulk ownership, reconciliation, evidence, classification, and remediation workflows. Bulk action can also bypass object-level controls or hide partial failure.

Required correction:

- exact selection criteria;
- affected and excluded counts;
- authorization-aware preview;
- side effects and reversibility;
- proportional approval;
- idempotency;
- post-action reconciliation.

## 2.8 Clear distinctions among absence states — high

The original document distinguished no data from no risk, but the complete state set was not consistently applied.

Required visual distinctions:

- no material change;
- no observation received;
- not assessed;
- insufficient evidence;
- stale evidence;
- source unavailable;
- unauthorized;
- not applicable;
- unknown;
- contradiction;
- action implemented;
- awaiting verification;
- verified effective;
- verified ineffective.

## 2.9 Density modes — medium

An executive brief and an 18,000-terminal reconciliation cannot use the same card density.

Required correction:

- comfortable mode for brief, narrative, and decision;
- compact mode for operational tables and reconciliation;
- accessibility-preserving target sizes and labels;
- semantic hierarchy unchanged by density.

## 2.10 Attention and notification model — medium

The original design focused on Today but did not sufficiently specify interruption control.

Required correction:

- situation-level grouping;
- deduplication;
- role and authority awareness;
- cancellation when evidence arrives elsewhere;
- distinction among information, work, decision, deadline, escalation, and protected communication;
- no unread-count-as-urgency pattern.

## 2.11 Localization and bank reporting formats — medium

The original visual system mentioned multilingual content but not operational formatting in enough detail.

Required correction:

- currency and basis;
- decimal and thousands separators;
- time zones;
- exact timestamp behind relative time;
- local date formats;
- long-label and right-to-left readiness where supported;
- varying name structures.

## 2.12 Privacy-aware meeting and protected modes — medium

Bank and board surfaces may be displayed in shared rooms or through remote conferencing.

Required correction:

- meeting mode with point-in-time freeze;
- stable presentation hierarchy;
- privacy-aware redaction or masking where policy requires;
- protected-surface distinction;
- controlled copy, print, export, and download;
- no reliance on visual masking for authorization.

## 2.13 Offline and unstable-network capture — medium

Branch and field evidence may be collected in poor connectivity conditions.

Required correction:

- encrypted bounded local queue where policy allows;
- capture time separate from upload time;
- unsynchronized state;
- retry and conflict handling;
- duplicate prevention;
- explicit prohibition for evidence too sensitive for offline retention.

## 2.14 AI presentation without theatre — medium

The original design correctly rejected a chat-first shell, but needed more explicit operational patterns.

Required correction:

- show extracted values and affected source regions;
- display machine-inferred versus explicit values;
- provide correction and confirmation;
- show structured reasoning record, not hidden chain-of-thought;
- transition side effects into normal governed review.

---

# 3. Documentation conformance matrix

## README.md

**Status after review:** conformant.

It now defines:

- situation-first user experience;
- universal exposure patterns;
- observations and evidence recipes;
- progressive integration;
- source profiles and data quality;
- Today, Situation, Capture, Explore, Configure;
- simplified initial product wedge.

Remaining requirement: keep linked canonical documents synchronized.

## docs/product/operating-model.md

**Status after review:** new canonical specification.

Purpose:

- define product objects and semantics;
- prevent UI architecture from following platform architecture;
- support different bank sizes through configuration rather than product forks.

## docs/product/experience-principles.md

**Prior status:** materially stale and incomplete.

Prior non-conformities:

- old Today, Explore, Act, Prove, Govern navigation;
- material decision card treated as universal attention unit;
- graph-heavy Explore language;
- insufficient population, import, reconciliation, source-health, and bulk-action design;
- insufficient context-switch and localization rules;
- incomplete golden-screen set.

**Status after review:** corrected.

## AGENTS.md

**Prior status:** high-severity staleness.

Non-conformities:

- obsolete mission wording: “safest defensible decision” and “prove ... worked”;
- no canonical Scope, Exposure Pattern, Risk Situation, Evidence Recipe, and Observation semantics;
- graph described as user-facing authority without an explicit internal-architecture boundary;
- missing new invariants: situations before modules, banking language before GRC jargon, source authority before automated trust, progressive integration;
- golden-screen list omitted import, source, population, and reconciliation experiences.

**Required action:** update as a normative priority before implementation begins.

## docs/product/differentiation.md

**Prior status:** medium-to-high staleness.

Non-conformities:

- differentiation described as the interaction of seven platform capabilities;
- graph and Materiality Compiler positioned ahead of direct risk situations;
- user-facing moat not sufficiently centered on reusable exposure patterns, minimum-question evidence, progressive integration, source quality, and one situation workspace;
- competitor discussion risks overstating AI-native connected risk as unique.

**Required action:** align the moat around direct banking situations, evidence recipes, source trust, contradiction, decision memory, and verified outcomes.

## docs/implementation-plan.md

**Prior status:** high-severity staleness and overbreadth.

Non-conformities:

- all seven platform capabilities treated as initial architectural differentiators;
- graph precedes a practical source registry, progressive integration, and population reconciliation product;
- UI phase retains Today, Explore, Act, Prove, Govern;
- first vertical release still combines executive brief, graph, evidence, decision, and protected reporting;
- insufficient explicit delivery of spreadsheet import, source health, channel packs, risk situations, and compact operational worklists.

**Required action:** re-phase around source trust, bounded channel situations, capture/reconciliation, decision/verification, and only then broader materiality and AI.

## docs/quality/acceptance-tests.md

**Prior status:** strong but partially stale.

Strengths:

- realistic negative and degraded conditions;
- evidence and contradiction;
- tenant isolation;
- point-in-time reconstruction;
- AI security;
- implementation-versus-effectiveness distinction.

Non-conformities:

- golden screens and journeys use old interface names;
- no spreadsheet import, source profile, reconciliation, photo validation, scope switching, bulk action, localization, or offline-capture acceptance journeys;
- privileged-access and payment scenarios do not fully test the new Situation and Observation model.

**Required action:** retain existing depth while adding situation-first visual and data-quality journeys.

## docs/architecture/risk-graph-and-decision-engine.md

**Status:** conceptually strong, terminology partially stale.

Strengths:

- source facts separated from conclusions;
- bitemporal history;
- no single-score dependency;
- evidence uncertainty separated from exposure;
- decision and verification rigor.

Non-conformities:

- “material item” should map explicitly to Risk Situation;
- Exposure Pattern should be added as a reusable product-semantic object;
- the graph must be described as internal shared context, not default navigation;
- source authority and data-quality debt should be stronger inputs;
- projected treatment impact must remain clearly estimated and not causal proof.

## docs/architecture/living-evidence-fabric.md

**Status:** conceptually strong, product-facing terminology partially stale.

Strengths:

- claim-centric evidence;
- immutable sources;
- assertion extraction;
- sufficiency dimensions;
- contradiction;
- best-source resolution;
- protected evidence.

Non-conformities:

- Evidence Recipe should be explicit;
- Observation should be the common normalized product object across forms, files, media, imports, APIs, and telemetry;
- Source Profile and source-authority limits should be first-class;
- spreadsheet row provenance and photo-verification boundaries need explicit requirements;
- progressive integration levels should be acknowledged.

## docs/architecture/governed-ai-operators.md

**Status:** largely conformant.

Strengths:

- AI identity and purpose;
- model separate from operator;
- tool allowlists;
- structured output;
- authorization outside prompts;
- abstention;
- degraded mode;
- protected-data controls.

Alignment additions recommended:

- describe AI as a compiler from messy inputs into proposed structured observations and relationships;
- ensure every user-facing inference distinguishes extracted, inferred, and confirmed values;
- avoid presenting operator architecture as primary UI navigation.

## docs/README.md

**Prior status:** structurally stale.

Required correction:

- include operating-model document as canonical reading;
- distinguish product-semantic documents from deeper architecture;
- reference visual conformance review and dedicated visual acceptance coverage;
- establish conflict priority.

---

# 4. Canonical precedence after alignment

When product documents conflict, use:

1. safety, confidentiality, legal boundary, tenant isolation;
2. [`../../README.md`](../../README.md) for product intent;
3. [`../product/operating-model.md`](../product/operating-model.md) for product semantics;
4. [`../product/experience-principles.md`](../product/experience-principles.md) for interaction and visual behavior;
5. `AGENTS.md` for normative implementation invariants;
6. architecture documents for internal mechanisms;
7. implementation plan for sequence;
8. acceptance documents for release proof.

An architecture mechanism must not override the simpler user-facing operating model without an explicit product decision.

---

# 5. Recommended immediate document updates

Priority 0:

- align `AGENTS.md` mission and invariants;
- align documentation map and precedence;
- ensure operating-model document is mandatory reading.

Priority 1:

- rewrite differentiation around direct banking situations and evidence operations;
- re-phase implementation plan;
- expand acceptance tests with operational visual journeys.

Priority 2:

- add terminology alignment sections to risk graph and evidence architecture;
- add AI compiler semantics to governed operators;
- introduce ADRs for scope model, source registry, observation contract, authorization-aware projections, offline capture, and protected-report isolation.

---

# 6. Final product-design test

For every planned screen, ask:

1. What recognizable bank situation is the user handling?
2. Is the active scope and period unmistakable?
3. What facts are already known?
4. What is missing, stale, or contradictory?
5. Which source is authoritative for each fact—and only for that fact?
6. Is the user being asked for the smallest unresolved input?
7. Is a table, capture step, comparison, or relationship path the correct visual form?
8. Is the next accountable action obvious?
9. Is completion visually separate from verified outcome?
10. Can an auditor reconstruct the result later?

A screen that cannot answer these questions is likely UI bloat, architecture leakage, or generic GRC workflow.