# ClearSight Visual and Documentation Conformance Review

**Review date:** 2026-08-04  
**Scope:** README, AGENTS, product documents, architecture mapping, implementation plan, and acceptance tests.

This review evaluated whether the repository consistently supports the simplified, situation-first ClearSight product model and whether the visual specification is sufficient for real bank operation across regional, national, and multinational institutions.

---

# 1. Executive finding

The original documentation had strong visual taste and unusually good controls around evidence, AI, authority, accessibility, temporal history, and verified remediation.

The principal issue was structural:

- the README had moved to a situation-first product;
- supporting documents still described a graph/module-first product;
- the visual system emphasized executive cards and graph explanation but under-specified operational populations, imports, source quality, reconciliation, photo capture, and safe bulk work;
- the implementation plan could therefore have produced an impressive executive shell before solving the integration and workflow friction that determines bank adoption.

The corrected canonical direction is now:

> Users operate through bounded bank Risk Situations, focused Capture, population reconciliation, authorized Decisions, and verified outcomes. Graphs, evidence engines, decision ledgers, workflows, and AI operators remain internal platform capabilities rather than mandatory navigation concepts.

---

# 2. Visual findings and resolutions

## Persistent scope and context anchoring — resolved

Added requirements for:

- institution, legal entity, jurisdiction, channel, service, branch, merchant group, population, and period visibility;
- deliberate context switching;
- draft and selection protection across scopes;
- scope re-confirmation before approval, export, bulk action, or submission.

## Operational population views — resolved

Added first-class requirements for:

- population worklists and tables;
- denominators and exclusions;
- resolved, unresolved, stale, contradictory, not-applicable, excluded, and unauthorized states;
- compact density;
- sticky identifiers and headers;
- keyboard navigation;
- virtualization and pagination;
- safe selection and bulk review.

## Spreadsheet mapping and reconciliation — resolved

Added:

- sheet and source-profile selection;
- column mapping and sample preview;
- type and required-field validation;
- duplicate and identifier matching;
- partial acceptance;
- unresolved-row queue;
- row-level provenance;
- import summary and rollback reference;
- explicit separation of upload, parse, mapping, observation acceptance, reconciliation, and evidence sufficiency.

## Source authority and data-quality visibility — resolved

Added the governed Source Profile with:

- owner;
- authoritative fields and explicit limitations;
- scope;
- expected and current freshness;
- health;
- mapping version;
- known limitations;
- unresolved mappings;
- affected conclusions.

## Reconciliation and entity matching — resolved

Added:

- side-by-side comparison;
- normalized identifiers;
- proposed match explanation;
- confirmed, provisional, unresolved, duplicate, contradictory, rejected, and superseded states;
- merge and unmerge history;
- downstream impact.

## Photo and scan capture boundaries — resolved

Added:

- framing and quality guidance;
- blur, glare, crop, and readability checks;
- metadata and location notice;
- data minimization and redaction;
- original and extraction-region preservation;
- machine-inferred versus user-confirmed state;
- bounded visible-attribute claims rather than broad “AI verified” claims.

## Safe bulk operations — resolved

Added:

- exact selection criteria;
- writable, excluded, failed, and unauthorized counts;
- per-object server authorization;
- side effects and reversibility;
- proportional approval;
- idempotency;
- post-action reconciliation and audit manifest.

## Absence and lifecycle state distinctions — resolved

The visual system now distinguishes:

- no material change;
- no data received;
- not assessed;
- insufficient evidence;
- stale evidence;
- source unavailable;
- unauthorized;
- not applicable;
- unknown;
- contradictory;
- implemented;
- awaiting verification;
- verified effective;
- verified ineffective;
- indeterminate.

## Density modes — resolved

Added comfortable and compact density with accessibility-preserving rules.

## Attention and notification model — resolved

Added situation-level grouping, role and authority awareness, deduplication, resolution-aware cancellation, and distinctions among information, work, decision, deadline, escalation, and protected communication.

## Localization and bank reporting formats — resolved

Added requirements for currency, number separators, dates, time zones, exact timestamps, long translated labels, and right-to-left readiness where supported.

## Privacy-aware meeting and protected modes — resolved

Added meeting-mode, point-in-time freeze, protected surfaces, presentation privacy, controlled copy/print/export, watermarking or masking where justified, and explicit separation from authorization.

## Offline and unstable-network capture — resolved

Added encrypted bounded local queue requirements, capture-versus-upload time, unsynchronized state, retry, conflict handling, duplicate prevention, and sensitivity restrictions.

## AI presentation without theatre — resolved

Added explicit patterns for extracted values, source regions, machine-inferred versus explicit values, correction and confirmation, structured reasoning records, and transition to normal governed review for side effects.

---

# 3. Documentation status after correction

| Document | Prior issue | Current status |
|---|---|---|
| `README.md` | Only document using the simplified operating model | Conformant; remains the product vision entry point |
| `docs/product/operating-model.md` | Missing canonical semantic layer | Added; now defines Scope, Exposure Pattern, Risk Situation, Claim, Evidence Recipe, Observation, Conclusion, Decision, and Verification |
| `docs/product/experience-principles.md` | Old Today/Explore/Act/Prove/Govern architecture; missing operational workflows | Rewritten and conformant |
| `AGENTS.md` | Obsolete mission wording and graph/module-first invariants | Rewritten and conformant |
| `docs/product/differentiation.md` | Seven-platform-capability moat and insufficient source/capture differentiation | Rewritten and conformant |
| `docs/implementation-plan.md` | Graph and broad intelligence before source trust and capture; overbroad initial wedge | Re-phased and conformant |
| `docs/quality/acceptance-tests.md` | Strong tests but missing import, source, reconciliation, photo, bulk, offline, scope, and localization journeys | Rewritten and conformant |
| `docs/architecture/product-semantics-mapping.md` | No explicit bridge between new product objects and existing architecture | Added; establishes canonical mapping |
| `docs/README.md` | Missing operating model, mapping, review, and precedence | Rewritten and conformant |
| `docs/architecture/risk-graph-and-decision-engine.md` | Uses older internal terminology such as material item | Acceptable through canonical mapping; direct terminology refresh remains optional |
| `docs/architecture/living-evidence-fabric.md` | Evidence Recipe, Observation, and Source Profile not named as product-facing abstractions | Acceptable through canonical mapping; direct terminology refresh remains optional |
| `docs/architecture/governed-ai-operators.md` | AI compiler role not emphasized as product-facing model | Largely conformant; mapping document establishes the boundary |

---

# 4. Canonical precedence

When product documents conflict:

1. safety, confidentiality, legal boundaries, and tenant isolation;
2. [`../../README.md`](../../README.md) for product intent;
3. [`../product/operating-model.md`](../product/operating-model.md) for product semantics;
4. [`../product/experience-principles.md`](../product/experience-principles.md) for interaction and visual behaviour;
5. [`../../AGENTS.md`](../../AGENTS.md) for normative implementation rules;
6. [`../architecture/product-semantics-mapping.md`](../architecture/product-semantics-mapping.md) for architecture terminology;
7. deeper architecture documents for internal mechanisms;
8. implementation plan for delivery sequence;
9. acceptance tests for release proof.

An internal architecture mechanism must not override the simpler user-facing operating model without an explicit product decision and synchronized document update.

---

# 5. Remaining non-blocking work

The repository is now coherent enough to begin architecture decisions and product design.

Remaining useful refinements:

- add direct alignment sections to the older risk-graph, evidence-fabric, and governed-operator documents when they are next edited;
- define concrete design tokens through an ADR or design-system package rather than hard-coding values in documentation;
- validate the visual model through real ATM and POS prototypes with bank stakeholders;
- test comfortable and compact density on representative enterprise laptop and remote-desktop environments;
- validate photo capture with realistic branch lighting, device labels, glare, and low bandwidth;
- confirm local jurisdiction, currency, date, retention, and protected-report requirements during pilot selection;
- define first Source Profiles and Evidence Recipes with actual bank data owners.

---

# 6. Final product-design test

For every planned screen, ask:

1. What recognizable bank situation is the user handling?
2. Is the active scope and period unmistakable?
3. What facts are already known?
4. What is missing, stale, partial, or contradictory?
5. Which source is authoritative for each fact—and only for that fact?
6. Is the user being asked for the smallest unresolved input?
7. Is a card, table, capture step, comparison, path, timeline, or chart the correct visual form?
8. Is the next accountable handling step obvious?
9. Is implementation visually separate from verified outcome?
10. Can an auditor reconstruct the result later?

A screen that cannot answer these questions is likely UI bloat, architecture leakage, or generic GRC workflow.