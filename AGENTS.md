# AGENTS.md

This file defines mandatory rules for every human contributor, coding agent, design agent, reviewer, and automated change applied to ClearSight.

It exists to prevent the product from regressing into a conventional GRC portal, a generic AI chat interface, a dense enterprise dashboard, a graph demo, or a collection of disconnected modules.

The words **MUST**, **MUST NOT**, **SHOULD**, and **SHOULD NOT** are normative.

---

# 1. Mission

ClearSight is a direct, AI-native risk and governance operating system built first for banks.

Every implementation decision must advance this outcome:

> **Enable each stakeholder to understand the risk situation relevant to them, provide or inspect the minimum necessary evidence, make an authorized and evidence-grounded decision, and verify whether the defined outcome criteria were achieved.**

ClearSight remains a comprehensive GRC platform, but users MUST NOT be required to operate its internal architecture.

The product is optimized for:

- recognizable banking situations;
- earlier detection of material exposure;
- less human effort;
- stronger and more transparent evidence;
- explicit source authority and data quality;
- clearer accountable decisions;
- proportionate action;
- verified outcome criteria;
- and durable institutional memory.

It is not optimized for the number of forms, modules, dashboards, records, alerts, graph nodes, AI messages, or configuration options it can display.

---

# 2. Required reading and precedence

Before changing product behavior, domain semantics, architecture, or interface structure, read:

1. [`README.md`](README.md)
2. [`docs/product/operating-model.md`](docs/product/operating-model.md)
3. [`docs/product/experience-principles.md`](docs/product/experience-principles.md)
4. [`docs/product/differentiation.md`](docs/product/differentiation.md)
5. [`docs/architecture/risk-graph-and-decision-engine.md`](docs/architecture/risk-graph-and-decision-engine.md)
6. [`docs/architecture/living-evidence-fabric.md`](docs/architecture/living-evidence-fabric.md)
7. [`docs/architecture/governed-ai-operators.md`](docs/architecture/governed-ai-operators.md)
8. [`docs/implementation-plan.md`](docs/implementation-plan.md)
9. [`docs/quality/acceptance-tests.md`](docs/quality/acceptance-tests.md)

When documents conflict, use this order:

1. safety, confidentiality, legal boundary, and tenant isolation;
2. README product intent;
3. operating-model product semantics;
4. experience principles;
5. this normative file;
6. architecture documents;
7. implementation sequencing;
8. acceptance detail.

An internal architecture mechanism MUST NOT override the simpler user-facing operating model without an explicit product decision and synchronized documentation change.

Do not silently reinterpret product language in code.

---

# 3. Priority order

When requirements conflict:

1. Safety, confidentiality, legal boundaries, and tenant isolation
2. Evidence integrity and decision auditability
3. Product semantics and invariants
4. User authority and segregation of duties
5. Functional correctness
6. Accessibility and usability
7. Reliability and recoverability
8. Performance
9. Visual polish
10. Implementation convenience

Visual polish may never conceal uncertainty, weaken accessibility, or replace missing domain correctness.

---

# 4. Canonical product objects

The product-facing operating model uses the following concepts.

## 4.1 Scope

The bounded institution, legal entity, jurisdiction, channel, service, branch, merchant group, asset population, vendor relationship, customer group, or process being governed.

The active scope and effective period MUST be explicit before material action, approval, export, bulk change, or evidence submission.

## 4.2 Exposure Pattern

A reusable description of how a banking activity, population, service, dependency, or control may fail or cause harm.

Exposure patterns SHOULD be reused across ATM, POS, mobile, branch, payments, cards, vendors, cyber, resilience, and other domains rather than copied into separate module schemas.

## 4.3 Risk Situation

A current, bounded instance requiring monitoring, evidence, assessment, decision, action, or verification.

A Risk Situation is the primary user-facing object. It connects underlying risks, controls, obligations, incidents, evidence, decisions, actions, and relationships without requiring separate module navigation.

## 4.4 Claim and Evidence Recipe

A Claim is a precise statement requiring support, contradiction, qualification, or resolution.

An Evidence Recipe defines acceptable observations, source authority, scope, freshness, coverage, contradiction rules, and review requirements for that claim.

## 4.5 Observation

A normalized, source-preserving record of something observed, submitted, imported, measured, extracted, or asserted.

Forms, dropdowns, photos, spreadsheets, documents, APIs, database exports, telemetry, messages, attestations, customer reports, and protected reports MUST converge on the same governed observation contract.

## 4.6 Conclusion

A versioned determination of what the current evidence supports.

## 4.7 Decision

An authorized selection among options with evidence, uncertainty, scope, authority, rationale, conditions, expiry, actions, and verification.

## 4.8 Verification Contract

A definition of the observable outcome, source, baseline, population, threshold, period, authority, and failure response.

The product verifies whether defined outcome criteria were achieved. It MUST NOT overstate that one action conclusively caused all risk movement.

---

# 5. Product invariants

These invariants are non-negotiable.

## 5.1 Situations before modules

Users MUST be able to understand and handle one situation without navigating separate risk, control, evidence, issue, action, and assurance module homepages.

Internal bounded contexts MAY remain separate in code. Their boundaries MUST NOT dictate the primary interface.

## 5.2 Banking language before GRC jargon

Primary user language SHOULD begin with channels, services, branches, merchants, customers, assets, systems, vendors, transactions, and outcomes.

Control IDs, taxonomy codes, regulatory references, and graph terminology remain available for specialists but MUST NOT dominate ordinary tasks.

## 5.3 Scope before action

The active institution, legal entity, jurisdiction, channel, service, population, and period MUST be clear enough to prevent wrong-context action.

Context switching MUST be deliberate and tested.

## 5.4 Materiality before volume

- Raw observations and signals MUST remain available.
- Executive surfaces MUST show a deliberately small number of material situations.
- Similar observations SHOULD be grouped where defensible.
- Grouping MUST preserve source records and rationale.
- Suppression MUST NOT mean analytical disappearance.
- Alert count MUST NOT represent risk severity.

## 5.5 Existing evidence before human requests

The system MUST search authorized existing observations and evidence before contacting a person.

Requests MUST ask only for unresolved facts and MUST stop when the evidence need is satisfied or no longer relevant.

## 5.6 Source authority before automated trust

An API, database, spreadsheet, or telemetry source is authoritative only for explicitly governed facts and scope.

Every source MUST expose:

- owner;
- authoritative fields;
- limitations;
- scope;
- expected freshness;
- current health;
- mapping version;
- and known data-quality issues.

Successful ingestion MUST NOT be treated as truth, completeness, or evidence sufficiency.

## 5.7 Progressive integration

The product MUST remain useful with structured manual capture, managed imports, APIs, or events.

Regional banks MUST NOT require enterprise-grade APIs before obtaining value. Multinational banks MUST be able to deepen automation without changing product semantics.

## 5.8 Evidence before confidence

- AI confidence MUST NOT substitute for evidence sufficiency.
- Scores MUST expose dimensions and supporting facts.
- Contradictory evidence MUST remain visible.
- Original source material MUST be preserved where policy permits.
- Derived summaries MUST link to source versions.

## 5.9 Data-quality transparency

Unresolved mappings, duplicate identifiers, stale sources, conflicting owners, partial imports, and incomplete populations MUST remain visible.

Data-quality weakness MAY create or affect a Risk Situation.

## 5.10 Decisions before dashboards

A material state MUST identify the required evidence, investigation, decision, action, or verification.

“View details” alone is not a valid handling path.

## 5.11 Verification before closure

- Material remediation MUST define a verification contract.
- Implementation MUST remain visually and semantically separate from verified outcome.
- Closure MUST require accepted outcome evidence.
- Later contradiction MUST reopen or supersede the conclusion without deleting history.

## 5.12 Human authority for material judgment

Risk appetite, material acceptance, reportability, protected identity disclosure, external regulatory representation, and other restricted decisions remain human-governed.

AI MUST NOT silently execute material judgment.

## 5.13 Progressive disclosure over interface density

Default views show only information necessary for the current task. Specialists retain full lineage and detail.

## 5.14 Institutional memory

Material objects MUST support point-in-time reconstruction. Corrections supersede rather than overwrite.

---

# 6. Domain modeling rules

## 6.1 Keep concepts distinct

The following MUST remain distinct:

- source record;
- observation;
- signal;
- exposure pattern;
- risk situation;
- claim;
- evidence;
- conclusion;
- decision;
- action;
- implementation evidence;
- outcome evidence;
- and verified outcome.

Do not collapse them into one generic assessment or status record.

## 6.2 Forms are not the domain model

Forms, tables, imports, photos, and chat commands are capture or interaction surfaces.

New features MUST identify the canonical objects and relationships they create or modify.

## 6.3 No duplicated truth

A vendor, branch, service, asset, merchant, control objective, obligation, or risk scenario SHOULD have one canonical identity with scoped relationships and views.

Do not create domain-specific copies merely to simplify one module.

## 6.4 Temporal semantics

Material data SHOULD support valid time and record time.

At minimum, each material entity, relationship, observation, conclusion, decision, and policy must support effective period, version, actor, reason, and supersession.

## 6.5 Provenance

Every derived object MUST identify:

- source IDs and versions;
- source scope;
- derivation or mapping method;
- rule, model, operator, or import version;
- actor;
- creation time;
- and validation state.

## 6.6 Explicit finite states

Lifecycle state MUST be explicit, finite, and validated.

Do not encode state only through nullable dates, colors, or loosely interpreted strings.

---

# 7. Evidence and capture rules

## 7.1 Claim-centric evidence

Evidence exists to support or contradict a precise claim for a defined purpose, population, scope, and period.

Never reduce evidence handling to:

- a generic file field;
- a checklist attachment;
- an unversioned URL;
- a self-attestation without context;
- or a binary present/missing flag.

## 7.2 Observation contract

Every captured item MUST preserve:

- original artifact or authoritative source reference;
- subject and normalized fact;
- scope and population;
- effective and capture time;
- capture method;
- source identity and authority limits;
- transformation history;
- sensitivity;
- version;
- and review or confirmation state.

## 7.3 Spreadsheet import

Spreadsheet and CSV ingestion MUST preserve file, sheet, row, mapping version, scope, validation state, uploader or managed source, and import time.

The UI and domain MUST distinguish:

- uploaded;
- parsed;
- mapped;
- accepted as an observation;
- reconciled;
- and sufficient for a claim.

Partial success MUST surface unresolved and rejected rows.

## 7.4 Photo and media evidence

AI interpretation of photos, scans, audio, or video MUST:

- preserve the original;
- identify extraction regions or time offsets where feasible;
- distinguish visible explicit facts from inference;
- expose model and version;
- allow correction;
- and record human confirmation.

Do not claim that a general photo proves invisible security, control effectiveness, or causality.

## 7.5 Controlled values

Dropdowns and selections MUST be scoped, searchable, human-readable, and sourced from approved values.

A controlled selection is an assertion; its evidential authority depends on the user, source, purpose, and claim.

## 7.6 Contradiction

Contradiction is a first-class record and state. It MUST propagate to affected conclusions, decisions, reports, and verification.

---

# 8. Population, reconciliation, and bulk-operation rules

## 8.1 Population integrity

Any percentage or completion state MUST expose its denominator and exclusions.

Population views SHOULD distinguish resolved, unresolved, stale, contradictory, not applicable, excluded, and unauthorized records.

## 8.2 Matching

Entity resolution MUST support:

- exact and alias matching;
- provisional matches;
- unresolved state;
- human review;
- merge and unmerge;
- provenance;
- and history.

AI MAY propose a match. Material merges require governed policy or review.

## 8.3 Bulk action

Bulk operations MUST:

- use server-side authorization per object;
- show exact selection criteria and counts;
- expose excluded or failed records;
- be idempotent where applicable;
- preserve individual audit events or a reconstructable manifest;
- support proportional approval;
- and reconcile the final outcome.

Bulk UI MUST NOT become a route around object-level policy.

---

# 9. AI implementation rules

## 9.1 AI is a governed compiler

AI primarily translates messy institutional inputs into proposed structured observations, relationships, mappings, claims, questions, summaries, or options.

A model is not an operator and an operator is not an authority.

## 9.2 Grounding

Material AI output MUST include exact source references and versions, scope, time period, assumptions, confidence dimensions, and unresolved contradiction.

General model knowledge MUST NOT establish a material institutional fact.

## 9.3 Structured output

AI used in workflows MUST produce validated, versioned structured output before domain mutation.

Invalid output fails closed. Free-form text MUST NOT directly trigger privileged action.

## 9.4 Prompt injection and untrusted content

All documents, emails, media, evidence, web content, spreadsheets, and user submissions are untrusted.

Tool permissions, authorization, evidence minimums, and material action thresholds MUST be enforced outside prompts.

## 9.5 Abstention

Operators MUST abstain when evidence, authorization, entity resolution, scope, policy, or evaluation coverage is insufficient.

## 9.6 Human review

Review surfaces MUST distinguish:

- explicit source value;
- machine-extracted value;
- inferred value;
- user-confirmed value;
- and approved domain conclusion.

Approval cannot be a context-free button.

## 9.7 No hidden chain-of-thought requirement

The product stores a concise structured reasoning record from source facts, policy, assumptions, alternatives, contradiction, conclusion, and approval requirement. It MUST NOT depend on hidden model chain-of-thought as the audit record.

## 9.8 Model independence and degradation

Domain logic MUST remain usable without a model provider. AI tasks require fallback, timeout, retry, kill switch, evaluation, and manual operation.

---

# 10. Security, privacy, and authorization

## 10.1 Deny by default

All access is denied unless explicitly allowed by tenant, entity, role, attributes, relationships, purpose, object sensitivity, and current workflow state.

## 10.2 Server-side enforcement

Client hiding is not authorization.

Every read, count, search, graph traversal, export, cache, embedding retrieval, AI context, bulk operation, and write must enforce policy server-side.

## 10.3 Inference resistance

Unauthorized users MUST NOT learn material existence, identity, count, label, relationship, title, snippet, suggestion, or timing information.

## 10.4 Protected reporting

- Protected case content and reporter identity require stronger isolation than ordinary records.
- Identity MUST be separately controlled.
- Anonymous communication MUST not require identity disclosure.
- Access MUST be need-to-know and conflict-aware.
- Search, analytics, exports, logs, backups, and observability MUST not leak identity.
- AI MUST NOT infer credibility from style, emotion, accent, demographics, or unsupported behavioral proxies.
- Identity reveal requires explicit privileged authority and immutable audit.

## 10.5 Exports

Exports MUST re-evaluate authorization, record requester, purpose, scope, versions, and classification, and produce a manifest.

## 10.6 Logging

Logs MUST NOT contain secrets, raw restricted evidence, unnecessary personal data, protected identity, or unrestricted model context.

## 10.7 Offline capture

Offline storage or queues MUST be bounded, encrypted, policy-controlled, clearly visible, conflict-aware, and prohibited for data classes that cannot safely remain on the device.

---

# 11. Visual and interaction rules

## 11.1 Primary surfaces

The primary user surfaces are:

- Today;
- Situation;
- Capture;
- Explore;
- Configure.

Do not reintroduce graph, evidence, decision ledger, assurance, AI operator, or internal bounded-context names as mandatory top-level navigation.

## 11.2 Product feeling

ClearSight MUST feel calm, direct, precise, relatable, premium, institutional, and trustworthy.

It MUST NOT feel noisy, gamified, decorative, cyberpunk, consumer-social, or like a generic admin template.

## 11.3 Scope anchoring

Scope and period MUST remain visible enough to prevent wrong-context action.

## 11.4 Correct visual form

Use:

- cards for limited attention items;
- tables and worklists for populations;
- comparison views for contradiction and reconciliation;
- step flows for capture and import;
- relationship paths for dependencies;
- timelines for change and reconstruction;
- charts only where they answer a decision question.

Do not force every object into a card or every relationship into a graph.

## 11.5 Density

Support comfortable and compact density without reducing accessibility or changing meaning.

## 11.6 Color

Use semantic tokens:

- cyan for context or selected intelligence;
- violet for governance and authority;
- coral/red for material exposure, breach, or failed verification;
- amber for uncertainty, staleness, contradiction, or approaching threshold;
- green only for accepted verified outcome;
- neutral or ordinary blue for baseline, selection, and non-risk information.

## 11.7 State distinctions

The UI MUST distinguish no material change, no data, not assessed, insufficient evidence, stale, unavailable, unauthorized, not applicable, unknown, contradictory, implemented, awaiting verification, verified effective, and verified ineffective.

## 11.8 Glass, glow, motion

Glass, glow, depth, and motion may communicate focus, hierarchy, relationship, active analysis, or state transition.

They MUST NOT reduce contrast, increase urgency decoratively, delay action, or create excessive GPU cost.

## 11.9 Accessibility and localization

All production interfaces MUST meet WCAG 2.2 AA at minimum and support keyboard operation, visible focus, screen-reader state, non-color meaning, reduced motion, 200% zoom, touch targets, error recovery, local number/date/currency formats, and multilingual expansion.

## 11.10 Golden visual coverage

Visual regression MUST include, where relevant:

- Today brief;
- situation states;
- Situation workspace;
- ATM and POS situations;
- population worklist;
- spreadsheet mapping and import reconciliation;
- Source Profile and degraded source;
- photo capture and extraction review;
- evidence sufficiency and contradiction;
- decision review;
- implementation versus verification;
- protected reporting;
- offline and AI-unavailable states;
- bulk action review;
- and export manifest.

See [`docs/product/experience-principles.md`](docs/product/experience-principles.md).

---

# 12. Architecture rules

## 12.1 Begin as a coherent modular core

Start with a modular monolith or similarly disciplined core unless an ADR proves the need for independent services.

## 12.2 Relational authority first

The authoritative model SHOULD begin in relational storage with typed, versioned relationships. Search, graph, vector, and analytical views are rebuildable projections.

Do not adopt a dedicated graph engine merely because the product has connected context.

## 12.3 Domain boundaries

Candidate bounded contexts include:

- identity and authorization;
- institution and scope;
- source registry and integration;
- exposure patterns and situations;
- claims, observations, and evidence;
- decisions and authority;
- actions and verification;
- protected reporting;
- AI gateway and operators;
- assurance and export.

No module may bypass another module’s invariants through direct table mutation.

## 12.4 Events and outbox

Material state changes use a transactional outbox or equivalent guarantee. Consumers are idempotent. Events represent completed facts, not vague updates.

## 12.5 Durable workflow

Do not scatter material workflow state across UI flags, cron jobs, and ad hoc queues.

## 12.6 Integration adapters

Adapters MUST preserve identity mapping, source object IDs, versions, cursor, health, authority limits, partial failure, replay, deletion, revocation, and provenance.

---

# 13. Testing requirements

Every meaningful change needs tests at the lowest useful level and at the product-contract level.

Depending on scope, include:

- domain unit tests;
- property-based tests;
- contract tests;
- authorization and inference tests;
- integration and replay tests;
- migration tests;
- evidence and import tests;
- AI evaluations;
- accessibility tests;
- visual regression;
- end-to-end golden journeys;
- performance and resilience;
- and security tests.

Tests MUST prove realistic negative, ambiguous, stale, partial, unauthorized, offline, contradictory, and degraded cases.

Fixtures MUST NOT inject the desired conclusion, owner, authority, mapping, or verification result.

Required product tests include:

- a situation can be understood without module hopping;
- an import exposes unresolved rows;
- a photo extraction remains bounded and correctable;
- existing evidence prevents an unnecessary request;
- source degradation affects dependent conclusions;
- bulk action cannot bypass object authorization;
- action completion does not produce verified green;
- protected identity cannot leak through search or count;
- AI cannot act outside scope;
- and a historical situation can be reconstructed without future knowledge.

---

# 14. Performance and reliability

- Deterministic context MUST remain usable while AI is pending.
- Common local interaction SHOULD respond within 100 ms.
- Remote work MUST acknowledge immediately and expose durable progress.
- Long lists MUST be paginated or virtualized.
- Spreadsheet processing MUST expose stages and partial failure.
- Evidence uploads MUST be resumable.
- External integrations MUST be idempotent and replay-safe.
- Layout MUST remain stable.
- Dense and visual surfaces MUST be tested on enterprise laptops, integrated GPUs, remote desktops, and relevant mobile devices.

Correctness and security take precedence over superficial speed.

---

# 15. Change protocol

For each meaningful change:

1. Identify the bank stakeholder and recognizable situation.
2. Identify active scope, period, authority, and sensitivity.
3. Identify canonical objects and relationships.
4. Identify source authority, limitations, and freshness.
5. Separate observations, claims, conclusions, decisions, actions, and outcomes.
6. Define failure, retry, cancellation, partial success, and recovery.
7. Define evidence, audit, and point-in-time requirements.
8. Choose the correct interaction form: card, table, capture step, comparison, path, timeline, or chart.
9. Add tests before considering the work complete.
10. Update all affected canonical documents and ADRs.

A pull request should include:

- problem and stakeholder;
- intended outcome;
- affected invariants;
- domain and architecture impact;
- security and privacy impact;
- source and data-quality impact;
- migration impact;
- AI/model impact;
- test evidence;
- screenshots for UI changes;
- accessibility review;
- and rollback plan.

---

# 16. Prohibited shortcuts

Do not:

- expose internal architecture as primary navigation;
- generate CRUD directly from database tables;
- use one status field for unrelated lifecycles;
- duplicate shared institutional truth by module;
- overwrite material history;
- store evidence only as a URL;
- treat upload or self-attestation as sufficient evidence;
- treat integration success as data truth;
- hide unresolved mappings or partial import failure;
- use AI confidence as evidence sufficiency;
- infer invisible control effectiveness from a photograph;
- close remediation on task completion;
- use a single unexplained score;
- allow unrestricted AI access to tenant data or tools;
- put authorization only in the frontend;
- use embeddings as an authorization boundary;
- expose protected identity through analytics, logs, counts, search, or exports;
- use cards where a population table is required;
- allow bulk operations to bypass per-object authorization;
- use glass, glow, heat maps, or chat as the primary product identity;
- ask broad questionnaires when a focused request is possible;
- require perfect APIs before the product is useful;
- silently display stale integration data;
- create tests that inject the final result;
- or describe planned behavior as implemented.

---

# 17. Definition of done

Work is complete only when:

- the situation is understandable in familiar bank language;
- scope and authority are explicit;
- domain behavior is correct;
- source authority and data quality are visible;
- evidence and audit lineage are complete;
- authorization is enforced server-side;
- failure, partial, stale, offline, and recovery paths work;
- meaningful negative tests pass;
- accessibility and localization are verified;
- visual regression is reviewed;
- performance is within budget;
- migrations and rollback are safe;
- documentation is synchronized;
- and planned capability is not represented as implemented.

For a material workflow, completion requires a real end-to-end path from observation through situation, claim, evidence, conclusion, decision, action, and verification.

---

# 18. Final review questions

Before merging, ask:

1. What recognizable bank situation does this solve?
2. Does the user see the correct scope and period?
3. Does this reduce human effort?
4. Does it improve evidence or merely create another record?
5. Is source authority explicit?
6. Are unresolved, stale, partial, and contradictory states visible?
7. Is the correct interaction form being used?
8. Does a material state lead to accountable handling?
9. Is implementation separate from verified outcome?
10. Can the institution reconstruct the result later?
11. Can an unauthorized user, operator, search index, export, or bulk action infer restricted information?
12. Is the experience unmistakably ClearSight rather than generic GRC?

If the last answer is “generic,” the work is not finished.