# AGENTS.md

This file defines mandatory rules for every human contributor, coding agent, design agent, reviewer, and automated change applied to ClearSight.

It exists to prevent the product from gradually regressing into a conventional GRC portal, a generic AI chat interface, a dense enterprise dashboard, or a collection of disconnected modules.

The words **MUST**, **MUST NOT**, **SHOULD**, and **SHOULD NOT** are normative.

---

# 1. Mission

ClearSight is an AI-native risk operating system for regulated institutions, built first for banks.

Every implementation decision must advance this product outcome:

> **Enable the institution to make the safest defensible decision with the least reasonable human effort, then prove that the decision and resulting action actually worked.**

The product is not optimized for the number of records, forms, modules, dashboards, notifications, or AI messages it can produce. It is optimized for:

- earlier detection of material risk;
- less effort required from staff;
- stronger evidence;
- clearer accountable decisions;
- faster proportionate action;
- verified risk reduction;
- and durable institutional memory.

---

# 2. Required reading order

Before changing product behavior, architecture, domain semantics, or interface structure, read:

1. [`README.md`](README.md)
2. [`docs/product/differentiation.md`](docs/product/differentiation.md)
3. [`docs/product/experience-principles.md`](docs/product/experience-principles.md)
4. [`docs/architecture/risk-graph-and-decision-engine.md`](docs/architecture/risk-graph-and-decision-engine.md)
5. [`docs/architecture/living-evidence-fabric.md`](docs/architecture/living-evidence-fabric.md)
6. [`docs/architecture/governed-ai-operators.md`](docs/architecture/governed-ai-operators.md)
7. [`docs/implementation-plan.md`](docs/implementation-plan.md)
8. [`docs/quality/acceptance-tests.md`](docs/quality/acceptance-tests.md)

A change that conflicts with these documents must include an explicit architecture or product decision record and update every affected canonical document in the same change.

Do not silently reinterpret product language in code.

---

# 3. Priority order when requirements conflict

Use this order:

1. Safety, confidentiality, legal boundaries, and tenant isolation
2. Evidence integrity and decision auditability
3. Product invariants and domain correctness
4. User authority and segregation of duties
5. Functional correctness
6. Accessibility and usability
7. Reliability and recoverability
8. Performance
9. Visual polish
10. Implementation convenience

Visual polish may never conceal uncertainty, weaken accessibility, or replace missing domain correctness.

---

# 4. Product invariants

These invariants are non-negotiable.

## 4.1 Materiality before volume

The product must reduce noise rather than expose every low-level event to executives.

- Raw signals MUST remain available for investigation.
- The default executive experience MUST show a deliberately small number of material items.
- Similar signals SHOULD be causally grouped where defensible.
- Alert count MUST NOT be used as a proxy for risk severity.
- A materiality decision MUST expose the rules, appetite statements, graph relationships, evidence, assumptions, and confidence that influenced it.

## 4.2 Evidence before confidence

No material conclusion may be represented as reliable without traceable supporting evidence.

- AI confidence MUST NOT substitute for evidence sufficiency.
- A score MUST expose its dimensions and supporting facts.
- Contradictory evidence MUST remain visible.
- Original source material MUST be preserved where policy permits.
- Derived summaries MUST link to source versions.

## 4.3 Relationships before forms

The institutional graph is authoritative for cross-domain relationships.

- Do not create duplicate module-specific versions of the same risk, control, vendor, service, obligation, or evidence object.
- Forms are capture surfaces, not the domain model.
- New features MUST identify the entities and relationships they add or modify.
- A relationship MUST include provenance, effective time, and confidence where applicable.

## 4.4 Decisions before dashboards

A material indicator must lead to an accountable next step.

- A red or amber state MUST identify the required decision, evidence gap, investigation, or action.
- “View details” alone is not a valid handling path.
- Decision cards MUST show authority, options, trade-offs, evidence strength, expected outcome, and verification method.

## 4.5 Verification before closure

Task completion is not proof of risk reduction.

- Material remediation MUST define a verification contract before closure.
- Closure MUST require accepted outcome evidence.
- The system MUST distinguish implementation evidence from effectiveness evidence.
- Contradictory later evidence MUST be able to reopen or supersede the conclusion without deleting history.

## 4.6 Automation before reminders

Do not add reminders where approved automation can safely complete the work.

- The system SHOULD gather known information before contacting a person.
- Requests MUST ask only for unresolved facts.
- Repeated requests for the same valid evidence MUST be deduplicated.
- Automation MUST remain bounded by authority, purpose, reversibility, and policy.

## 4.7 Human authority for material judgment

Risk appetite, material risk acceptance, protected identity disclosure, regulatory representation, and other high-impact decisions remain human-governed.

- AI may recommend but MUST NOT silently execute a material judgment.
- Authority thresholds MUST be explicit and testable.
- Overrides MUST capture actor, reason, evidence, and effect.

## 4.8 Institutional memory over periodic reporting

The system must preserve how understanding and decisions evolved.

- Material objects MUST support point-in-time reconstruction.
- Corrections MUST supersede rather than overwrite.
- Evidence, graph edges, risk states, decisions, and policy versions MUST retain temporal lineage.

---

# 5. Differentiation guards

A change MUST NOT weaken the following defining capabilities.

## 5.1 Living Evidence Fabric

Evidence is claim-centric, dynamic, multimodal, and continuously reconciled.

Never reduce evidence handling to:

- a generic file upload field;
- a checklist attachment;
- an unversioned document URL;
- a self-attestation with no supporting context;
- or a binary “present/missing” flag.

Every evidence implementation must preserve:

- the claim being supported or contradicted;
- source and provenance;
- effective and capture time;
- scope and coverage;
- authenticity indicators;
- sensitivity and access policy;
- original content and derived interpretations;
- version history;
- sufficiency dimensions;
- contradictions;
- and review history.

## 5.2 Dynamic micro-requests

The system must ask the smallest useful question to the best-placed source.

A request engine MUST:

1. identify the unresolved claim;
2. search for existing valid evidence;
3. determine the minimum missing facts;
4. identify the most authoritative accessible source;
5. prefill known context;
6. select the least burdensome approved channel;
7. provide a clear reason and deadline;
8. validate the response;
9. ask a focused follow-up only when necessary;
10. stop when sufficient evidence has been obtained.

Broad recurring questionnaires require explicit justification.

## 5.3 Materiality Compiler

No feature may bypass the materiality model and directly flood the executive surface.

Materiality is contextual. It may depend on:

- affected critical services;
- customers and vulnerable customer groups;
- legal entities and jurisdictions;
- financial and non-financial impact;
- appetite and tolerance;
- propagation paths and concentration;
- control importance;
- risk velocity and time-to-impact;
- regulatory deadlines;
- evidence weakness;
- and decision authority.

A single opaque numeric score is insufficient.

## 5.4 Decision Ledger

Material decisions are first-class, durable objects.

Do not store a material risk decision only as:

- a comment;
- an email body;
- a meeting note;
- a status transition;
- or an unstructured AI transcript.

A decision MUST include context, options, evidence, uncertainty, authority, rationale, conditions, expected outcome, review date, and verification contract.

## 5.5 Protected reporting

Whistleblower and confidential-reporting controls are stricter than ordinary case management.

- Reporter identity MUST be isolated from case content.
- Anonymous two-way communication MUST not require identity disclosure.
- Access MUST be need-to-know and conflict-aware.
- Search, analytics, exports, logs, backups, and observability MUST not leak protected identity.
- AI MUST NOT score credibility from language style, emotion, accent, demographic attributes, or unsupported behavioral inference.
- Any identity reveal MUST require explicit authority, reason, and immutable audit.

## 5.6 Outcome-verified remediation

A feature MUST NOT mark risk treatment successful based solely on task state.

A verification contract must define:

- intended risk outcome;
- observable measure;
- evidence source;
- observation period;
- success and failure thresholds;
- dependencies;
- acceptance authority;
- and failure handling.

## 5.7 Governed AI operators

Do not introduce an unconstrained “AI assistant” with broad data and tool access.

Every operator MUST be:

- identity-bound;
- tenant- and entity-scoped;
- purpose-bound;
- capability-constrained;
- policy-checked;
- model-versioned;
- source-grounded;
- confidence-aware;
- approval-aware;
- and audit-emitting.

---

# 6. Domain modeling rules

## 6.1 Separate facts, claims, conclusions, and decisions

These concepts MUST remain distinct.

- **Fact:** a directly observed or authoritative source assertion.
- **Signal:** an observation that may indicate a relevant change.
- **Claim:** a statement that requires support or contradiction.
- **Evidence:** an artifact or observation evaluated against a claim.
- **Conclusion:** a reasoned assessment based on evidence.
- **Decision:** an authorized selection among options.
- **Action:** work initiated because of a decision.
- **Outcome:** an observed result after action.

Do not collapse these into one generic “assessment” record.

## 6.2 Temporal semantics

Material domain data SHOULD support bitemporal semantics:

- **valid time:** when the fact or relationship was true in the world;
- **record time:** when ClearSight learned or recorded it.

At minimum, every material entity or relationship must support effective dates, versioning, and supersession.

## 6.3 Immutability

The following are append-only or immutable by default:

- evidence versions;
- original submissions;
- material decisions;
- approval events;
- AI action records;
- audit events;
- protected identity access events;
- and exported assurance packages.

Corrections create a new version and retain the prior record.

## 6.4 Provenance

Every derived object MUST identify:

- source object IDs and versions;
- derivation method;
- rule, model, or operator version;
- actor;
- creation time;
- and confidence or validation state where relevant.

## 6.5 No duplicated truth

Avoid module-specific copies of shared entities.

- A vendor is one entity with domain-specific relationships and views.
- A control objective is distinct from each implementation of that control.
- An obligation is distinct from the source regulatory text and from the policy or control used to satisfy it.
- A risk scenario is distinct from an incident that realizes it.

## 6.6 Explicit states

Workflow states MUST be explicit, finite, and validated.

Do not encode lifecycle state only through nullable dates or loosely interpreted strings.

State transitions MUST validate authority, required evidence, and side effects.

---

# 7. AI implementation rules

## 7.1 Grounding

Material AI output MUST be grounded in approved institutional or authoritative sources.

The output record must include:

- exact source references;
- source versions;
- relevant excerpts or structured facts;
- retrieval time;
- model and prompt-policy version;
- assumptions;
- confidence;
- and unresolved contradictions.

## 7.2 Structured output

AI used in workflows MUST produce validated structured output before changing domain state.

- Schemas MUST be versioned.
- Invalid output MUST fail closed.
- Free-form text MUST NOT directly trigger privileged actions.
- Domain validation and authorization run after model output and before execution.

## 7.3 Prompt injection and untrusted content

All ingested content is untrusted.

- Documents, emails, evidence, web content, and user submissions MUST NOT be treated as operator instructions.
- Tool permissions MUST be external to prompts.
- Retrieved text must be clearly separated from system policy.
- High-impact actions require independent policy validation.
- Operator tests MUST include prompt-injection and data-exfiltration cases.

## 7.4 Confidence and abstention

Operators MUST be able to abstain.

- Confidence thresholds depend on action class and materiality.
- Low-confidence output routes to a reviewer.
- Confidence MUST NOT be represented more precisely than calibration supports.
- A confident answer with weak evidence is still weak.

## 7.5 Human review

Human review surfaces MUST show:

- proposed output;
- affected objects;
- source evidence;
- uncertainty and contradiction;
- proposed side effects;
- required authority;
- and an editable rationale.

Approval cannot be a context-free “Approve” button.

## 7.6 Model independence

Domain logic MUST NOT depend on one provider’s proprietary response shape.

Use a model gateway with:

- provider adapters;
- capability metadata;
- data-residency policy;
- classification-aware routing;
- cost and latency controls;
- fallback behavior;
- evaluation results;
- and kill switches.

## 7.7 Evaluation before release

Every operator capability needs an evaluation suite covering:

- accuracy;
- citation and lineage correctness;
- false-positive and false-negative behavior;
- abstention;
- contradiction handling;
- tool selection;
- authorization boundaries;
- prompt injection;
- sensitive-data leakage;
- latency;
- and cost.

A model or prompt change cannot be released on subjective demo quality alone.

---

# 8. Security, privacy, and authorization rules

## 8.1 Deny by default

All access is denied unless explicitly allowed by tenant, legal entity, role, attributes, relationships, purpose, and object sensitivity.

## 8.2 Authorization is server-side

Client-side hiding is not authorization.

Every read, search, graph traversal, export, AI retrieval, and action must enforce authorization server-side.

## 8.3 Relationship-aware access

Access may depend on relationships such as:

- case assignment;
- business ownership;
- committee membership;
- investigation conflict;
- legal entity;
- geography;
- control ownership;
- and audit independence.

Graph traversal must not reveal inaccessible neighboring nodes through counts, labels, embeddings, or timing.

## 8.4 Sensitive data isolation

Protected identity, legal privilege, investigation data, customer personal data, secrets, and highly restricted evidence require separate encryption and policy boundaries where appropriate.

## 8.5 Exports

Exports MUST:

- re-evaluate authorization at generation time;
- record requester, purpose, scope, and included versions;
- support watermarking where required;
- avoid hidden protected data;
- and produce a manifest suitable for later verification.

## 8.6 Logging

Logs MUST NOT contain secrets, tokens, protected reporter identities, unnecessary personal data, raw confidential evidence, or model prompts containing restricted content.

Use structured identifiers and protected diagnostic access.

## 8.7 Multi-tenancy

Every persistence, cache, search, queue, vector, object-storage, analytics, and AI-retrieval path must be tested for tenant isolation.

Tenant context MUST NOT be inferred from user-controlled payload fields when it can be derived from authenticated context.

---

# 9. Visual and interaction non-regression rules

## 9.1 Product feeling

ClearSight must feel:

- calm;
- precise;
- premium;
- institutional;
- intelligent;
- and trustworthy.

It must not feel:

- noisy;
- gamified;
- decorative;
- cyberpunk;
- consumer-social;
- or like a generic admin template.

## 9.2 Progressive disclosure

Default views show only information required for the current decision.

- Secondary metrics belong behind explain or inspect states.
- Advanced filters must not dominate first use.
- Long forms should be decomposed around user intent, not database tables.

## 9.3 Executive density

The default Today view SHOULD contain no more than seven material cards without an explicit expanded mode.

Each card must have:

- one dominant message;
- one clear owner or authority;
- one primary next action;
- concise evidence state;
- and a visible reason for materiality.

## 9.4 Color semantics

Use design tokens only.

- Cyan: intelligence or discovered relationship
- Violet: governance, control, or approved automation
- Coral/red: material exposure, gap, or breach
- Amber: uncertainty, pending verification, or approaching threshold
- Green: verified outcome
- Neutral: informational or unassessed state

Do not use green for mere completion or self-attestation.

Do not use red for decorative urgency.

## 9.5 Glass, glow, and depth

Glassy surfaces and glow are allowed only when they communicate hierarchy, focus, data flow, active intelligence, or relationship depth.

- Do not apply blur to every surface.
- Do not reduce text contrast.
- Do not create excessive GPU cost.
- Do not use glow as a substitute for information hierarchy.
- Light mode must remain equally intentional.

## 9.6 Typography and layout

- Use a tokenized type scale.
- Prefer readable line lengths.
- Maintain stable alignment across loading, empty, error, and populated states.
- Avoid tiny uppercase labels for important information.
- Numbers must include units, time scope, and meaning.
- Avoid metric walls.

## 9.7 Motion

Motion must explain change, propagation, relationship, or state transition.

- Respect reduced-motion settings.
- Avoid continuous decorative animation.
- Avoid animation that delays action.
- Data updates must not cause disorienting layout movement.

## 9.8 Accessibility

All production interfaces MUST meet WCAG 2.2 AA at minimum.

This includes:

- keyboard operation;
- visible focus;
- screen-reader naming and state;
- contrast;
- target size;
- error association;
- reduced motion;
- non-color status indicators;
- and accessible charts or textual equivalents.

## 9.9 Visual regression

Changes to shared UI components or key journeys require visual regression coverage in both light and dark modes at supported breakpoints.

Golden screens include:

- Today executive brief;
- risk explain view;
- decision card;
- evidence request;
- evidence sufficiency view;
- institutional graph;
- whistleblower submission;
- protected case review;
- and verification outcome.

---

# 10. Performance and reliability budgets

Budgets should be enforced in CI once the application scaffold exists.

Initial targets:

- No blocking network waterfall for first meaningful screen.
- Core shell remains usable while AI-derived content is pending.
- Common interactions respond within 100 ms locally and acknowledge remote work immediately.
- Search and common filtered lists target sub-second perceived response under normal enterprise load.
- Long lists and graph views use pagination, virtualization, or progressive loading.
- AI tasks expose progress, timeout, cancellation, retry, and human fallback.
- Evidence uploads are resumable and integrity-checked.
- Long-running workflows are durable across process restarts.
- Every external integration is idempotent and replay-safe.

Do not trade correctness or security for superficial speed.

---

# 11. Architecture rules

## 11.1 Begin as a modular core

Start with a coherent modular monolith or similarly disciplined core unless an ADR proves the need for independent services.

Modules must own their domain rules and expose explicit interfaces.

Candidate bounded contexts include:

- identity and authorization;
- institutional graph;
- risk and appetite;
- signals and materiality;
- evidence and claims;
- decisions and approvals;
- workflow and remediation;
- assurance;
- protected reporting;
- AI operators;
- integrations;
- and reporting/export.

## 11.2 No shared mutable domain tables

A module must not bypass another module’s invariants by directly mutating its tables.

Use application services, commands, or documented domain events.

## 11.3 Transactional outbox

State changes that publish events must use a transactional outbox or equivalent guarantee.

Consumers must be idempotent.

## 11.4 Graph technology follows evidence

Do not adopt a dedicated graph database merely because the product contains a graph.

Begin with an authoritative relational model and graph projection unless benchmarks prove a dedicated engine is necessary for required traversal, scale, or isolation.

Record the decision in an ADR.

## 11.5 Search and vector retrieval are projections

Search indexes and embeddings are rebuildable projections, not authoritative stores.

- Preserve source IDs and versions.
- Enforce authorization during indexing and retrieval.
- Remove or reindex superseded and deleted content according to policy.

## 11.6 Workflow engine selection

A workflow engine may be introduced only after evaluating:

- durability;
- human task support;
- versioning;
- cancellation and compensation;
- tenancy;
- auditability;
- deployment constraints;
- and operational complexity.

Do not scatter workflow state across queues, cron jobs, and UI flags.

---

# 12. API and event rules

## 12.1 APIs

- Use versioned contracts.
- Validate all input.
- Return stable error codes and correlation IDs.
- Support idempotency keys for repeatable writes.
- Use optimistic concurrency for versioned material objects.
- Do not expose internal persistence schemas as public contracts.
- Include authorization-relevant context in server-side evaluation, not client claims.

## 12.2 Events

Events represent completed facts, not commands disguised as facts.

Good:

- `EvidenceVersionCaptured`
- `MaterialityAssessmentCompleted`
- `DecisionApproved`
- `VerificationFailed`

Avoid vague events such as `ItemUpdated`.

Events must include:

- event ID;
- schema version;
- tenant and legal-entity context;
- aggregate ID and version;
- actor;
- occurred time and recorded time;
- correlation and causation IDs;
- classification;
- and minimal safe payload.

## 12.3 Integration adapters

Adapters must:

- map external identity and objects explicitly;
- store sync cursors and versions;
- be idempotent;
- preserve source provenance;
- handle deletion and revocation;
- surface partial failure;
- support replay;
- and avoid granting external systems broader authority than required.

---

# 13. Testing requirements

Every change needs tests at the lowest useful level and at the product-contract level.

## 13.1 Required categories

Depending on the change, include:

- domain unit tests;
- property-based tests for scoring, state transitions, and temporal rules;
- contract tests;
- authorization tests;
- integration tests;
- migration tests;
- event replay and idempotency tests;
- AI evaluation cases;
- accessibility tests;
- visual regression tests;
- end-to-end golden journeys;
- load and resilience tests;
- and security tests.

## 13.2 Tests must prove invariants

Do not only test happy-path CRUD.

Tests must prove cases such as:

- an issue cannot close without accepted outcome evidence;
- a protected identity cannot leak through search;
- an AI operator cannot act outside scope;
- contradictory evidence remains visible;
- a superseded decision can be reconstructed;
- an unauthorized graph traversal reveals nothing;
- a duplicate integration event has no duplicate side effect;
- and an executive brief does not surface non-material noise.

## 13.3 Fixtures

Fixtures must represent real domain complexity.

Avoid fixtures that bypass the exact logic under test by injecting precomputed conclusions, fake owners, or privileged service objects.

Golden journeys are defined in [`docs/quality/acceptance-tests.md`](docs/quality/acceptance-tests.md).

---

# 14. Change protocol

For each meaningful change:

1. Identify the user and institutional outcome.
2. Identify affected product invariants.
3. Identify the authoritative domain objects and relationships.
4. Define authorization and sensitivity boundaries.
5. Define source facts, derived conclusions, and side effects.
6. Define failure, retry, cancellation, and recovery behavior.
7. Define evidence and audit requirements.
8. Add or update tests before considering the work complete.
9. Review visual and interaction regressions.
10. Update canonical documentation and ADRs.

A pull request description should include:

- problem;
- intended outcome;
- affected invariants;
- architecture impact;
- security and privacy impact;
- data migration impact;
- AI/model impact;
- test evidence;
- screenshots for UI changes;
- and rollback plan.

---

# 15. Prohibited shortcuts

The following are prohibited unless explicitly approved through an ADR and product review:

- generic CRUD generated directly from database tables;
- one universal `status` field for unrelated lifecycles;
- deleting or overwriting material history;
- storing evidence only as a URL;
- claiming control effectiveness from self-attestation alone;
- closing remediation on task completion;
- hiding uncertainty behind a single score;
- unrestricted AI access to all tenant data;
- direct model tool execution without domain validation;
- putting authorization only in the frontend;
- using embeddings as an authorization boundary;
- exposing protected identities to analytics or logs;
- copying competitor screens or design systems;
- excessive glass, blur, glow, or animation;
- heat maps as the sole executive risk representation;
- broad questionnaires when a targeted request is possible;
- notifications without deduplication or materiality;
- premature microservices;
- silent integration failure;
- tests that inject the final result instead of exercising the real path;
- or documentation that describes behavior not present in the implementation without labeling it as planned.

---

# 16. Definition of done

Work is not complete until:

- domain behavior is correct;
- product invariants are preserved;
- authorization is enforced server-side;
- evidence and audit lineage are complete;
- failure and recovery paths are implemented;
- tests cover meaningful negative cases;
- accessibility is verified;
- visual regression is reviewed where applicable;
- performance is within budget;
- observability is present;
- migrations and rollback are safe;
- documentation is synchronized;
- and no planned capability is misrepresented as implemented.

For a material workflow, completion also requires a passing end-to-end golden journey from signal or request through evidence, decision, action, and verification.

---

# 17. Final review questions

Before merging, ask:

1. Does this reduce or increase human effort?
2. Does it improve evidence or merely create another record?
3. Can the institution explain the resulting conclusion years later?
4. Does the executive surface become clearer or noisier?
5. Does a material state lead to an accountable action?
6. Can the action be verified rather than merely marked complete?
7. Can an unauthorized person, tenant, AI operator, search index, or export infer restricted information?
8. Have facts, claims, conclusions, decisions, and outcomes remained distinct?
9. Has the change preserved light mode, dark mode, accessibility, and performance?
10. Is this unmistakably ClearSight, or could it belong to any generic GRC product?

If the answer to the last question is “generic,” the work is not finished.