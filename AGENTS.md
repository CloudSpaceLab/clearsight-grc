# AGENTS.md

This file defines mandatory rules for every human contributor, coding agent, design agent, reviewer, and automated change applied to ClearSight.

The words **MUST**, **MUST NOT**, **SHOULD**, and **SHOULD NOT** are normative.

## 1. Mission

Every change must advance this outcome:

> **Help each bank stakeholder understand what must be done, what proves it, what changed, who is responsible, who must review or authorize, and whether the required outcome was achieved—with the minimum reasonable human effort.**

ClearSight is not optimized for the number of forms, modules, dashboards, alerts, controls, graph nodes, AI messages, or configuration options it exposes.

## 2. Required reading

Before changing product behavior, architecture, workflow, security, or UI, read:

1. [`README.md`](README.md)
2. [`docs/product/use-case-catalogue.md`](docs/product/use-case-catalogue.md)
3. [`docs/product/continuous-compliance-operating-model.md`](docs/product/continuous-compliance-operating-model.md)
4. [`docs/product/authority-routing-and-escalation.md`](docs/product/authority-routing-and-escalation.md)
5. [`docs/product/respond-and-capture.md`](docs/product/respond-and-capture.md)
6. [`docs/product/ease-of-use-standard.md`](docs/product/ease-of-use-standard.md)
7. [`docs/product/operating-model.md`](docs/product/operating-model.md)
8. [`docs/product/experience-principles.md`](docs/product/experience-principles.md)
9. [`docs/product/ux-and-visual-language.md`](docs/product/ux-and-visual-language.md)
10. [`docs/architecture/system-data-and-performance.md`](docs/architecture/system-data-and-performance.md)
11. relevant specialized product, architecture, implementation, and quality documents.

When documents conflict, follow [`docs/README.md`](docs/README.md).

## 3. Product model

### Program

A long-lived body of continuing obligations, controls, evidence, reviews, filings, exceptions, and assurance.

A Program MUST NOT be implemented as a static control list with manually maintained status.

### Matter

A bounded occurrence requiring assessment, evidence, decision, action, response, or verification.

A Matter MUST have a typed lifecycle and closure contract. Task completion alone MUST NOT close a material Matter.

### Shared governed primitives

Programs and Matters use shared Scope, Authority Source, Requirement, Control, Claim, Evidence Contract, Observation, Conclusion, Decision, Action, Response Package, Verification, Assignment, Authority, and temporal history.

Forms, imports, photos, chat, dashboards, and external tools are interaction or execution surfaces—not the domain model.

## 4. Use-case completeness

Every advertised capability MUST have a stable use-case ID and map to:

```text
customer and persona
→ trigger and outcome
→ scope and sources
→ responsibility and authority
→ happy, exception, degraded, and prohibited paths
→ state and closure contract
→ UX flow
→ architecture or ADR
→ implementation phase
→ acceptance test
```

Do not implement a feature that exists only as a noun in the README, a navigation item, an architecture component, or a mockup.

## 5. Responsibility, review, and authority

ClearSight MUST distinguish:

- performer or evidence provider;
- accountable owner;
- reviewer;
- independent challenger;
- authorizer or signatory;
- escalation owner;
- observer where required.

Do not reduce these to one `assignee`, one static role, or one editable user field.

Effective responsibility and authority MUST be resolved from versioned policy using:

- tenant and legal entity;
- role template and organizational position;
- object relationship;
- Program or Matter type and state;
- scope, purpose, materiality, amount, duration, and reversibility;
- delegation and substitution;
- conflict and segregation-of-duties rules;
- current identity and source health.

One clear next action means **one dominant action for the current actor in the current workflow state**. It does not prohibit parallel work by other actors.

### Routing configuration

Role, authority, and escalation configuration MUST support:

- source-backed people and organization data;
- scoped role templates;
- sequence, parallel, quorum, any-of, all-of, veto, and independent-challenge steps;
- reminders distinct from escalations;
- fallback queues and substitute roles;
- time zones, working calendars, and pause rules;
- simulation with representative scenarios;
- impact preview;
- maker-checker approval;
- effective dating, versioning, rollback, and audit.

Configuration MUST prevent circular routes, self-approval, privilege amplification, unsafe broad scope, and silent transfer of accountability.

## 6. Respond and Capture

Focused requests MUST be generated from an exact purpose, scope, Claim, Evidence Contract, or case directive.

A request MUST show:

- why the recipient was selected;
- what is already known;
- the smallest unresolved question;
- acceptable response forms;
- estimated effort;
- deadline and consequence;
- sensitivity and privacy notice;
- redirect, delegate, partial, not-applicable, wrong-recipient, and concern options where permitted;
- final assertions before submission.

Do not build a generic unrestricted form builder as the primary collection model.

### Invitations and magic links

Invitation access MUST be:

- opaque, request-scoped, purpose-bound, short-lived, revocable, and audience-bound;
- exchanged for a bounded server-side session rather than used as continuing authorization;
- stored hashed or equivalently protected;
- excluded from logs, analytics, referrers, and notification previews;
- protected by step-up authentication when sensitivity or consequence requires it.

A forwarded, expired, revoked, already-used, or wrong-recipient invitation MUST fail safely without leaking request details.

Protected anonymous reporting MUST use a separate identity-isolated two-way mechanism. It MUST NOT be implemented as an ordinary external form link.

## 7. Ease-of-use invariants

- Prefill before asking.
- Search existing authorized evidence before requesting more.
- Routine work SHOULD complete in under five minutes of active effort.
- Complex work MUST reach a clear saved and correctly routed next state within five minutes.
- Routine work SHOULD remain in one coherent Program or Matter workspace.
- Known values MUST show source, freshness, authority, and correction behavior where material.
- Review by exception MUST expose the full denominator, omitted population, source health, sampling policy, and full-review triggers.
- Save/resume MUST preserve scope, drafts, changes, blockers, owner, and next action.
- Accessibility users MUST NOT face materially more work.

Speed targets never override correctness, comprehension, security, or authority.

## 8. Evidence and state

- An Observation is not automatically a verified fact.
- Upload, parse, mapping, acceptance as Observation, evidence sufficiency, implementation, and verified outcome are distinct states.
- AI confidence MUST NOT substitute for evidence sufficiency.
- Source degradation MUST propagate without falsely asserting control failure.
- Contradictions MUST remain visible and actionable.
- Green MUST mean evidence-supported acceptable or verified state—not submitted, assigned, uploaded, or implemented.
- Material records MUST be versioned; corrections supersede rather than overwrite.

## 9. AI and automation

AI acts as a governed compiler from messy inputs into proposed structured records, questions, summaries, options, or domain commands.

Production AI MUST have:

- explicit operator identity, purpose, scope, allowed data, tools, and action class;
- exact source and version lineage;
- structured validated output;
- explicit versus inferred fields;
- confidence, contradiction, and abstention;
- authorization and policy outside prompts;
- evaluation, monitoring, rollback, and degraded mode.

AI MUST NOT independently make final legal interpretation, applicability, material risk acceptance, reportability, protected-identity disclosure, external representation, account restriction, or material closure.

### Automation lifecycle

Low-impact automation MUST still have:

- eligibility policy;
- simulation or dry run;
- affected population and blast-radius preview;
- approval and effective period;
- canary or staged activation where appropriate;
- idempotency, rollback or compensation;
- monitoring, suspension, kill switch, and outcome verification;
- expiry and reauthorization.

External execution success is implementation evidence, not proof of outcome.

## 10. Security and privacy

Enforce authorization server-side for reads, counts, search, graph traversal, embeddings, caches, exports, bulk actions, AI retrieval, background jobs, invitations, and writes.

Requirements:

- deny by default;
- tenant, entity, relationship, purpose, sensitivity, and workflow-state enforcement;
- no inference through counts, labels, snippets, suggestions, timing, cache keys, or manifests;
- protected reporting and authority cases isolated from ordinary search and analytics;
- export and response-package authorization re-evaluated at generation and download;
- no raw restricted evidence, tokens, protected identities, or secrets in logs or events;
- offline capture encrypted, bounded, revocable, conflict-aware, and prohibited for unsuitable data classes;
- support and break-glass access explicit, time-limited, notified, and retrospectively reviewed.

Fewer clicks MUST NOT widen access.

## 11. System, data, and performance

The initial implementation SHOULD be a coherent modular core with explicit bounded contexts, not a premature microservice estate.

Required principles:

- relational authoritative state;
- versioned object storage for artifacts;
- durable workflow state, timers, outbox/inbox, idempotency, and optimistic concurrency;
- rebuildable search, work-queue, graph, vector, and reporting projections;
- deterministic context before AI;
- strong consistency for material commands and explicit eventual consistency for projections;
- partitioned, resumable, backpressured ingestion;
- tenant/purpose-bound caching;
- point-in-time reconstruction;
- measurable SLOs, capacity profiles, recovery, and degraded operation.

Every material feature MUST define:

- expected cardinality and growth;
- read/write/query patterns;
- latency and availability budget;
- consistency requirement;
- partition and index strategy;
- authorization cost;
- failure and retry behavior;
- observability without sensitive content.

A workflow that is functionally correct but fails its workload or latency profile is not complete.

## 12. UI and component rules

Primary navigation remains Today, Programs, Work, Explore, and Configure, with focused Respond/Capture experiences.

- Programs MUST prioritize current position, material gaps, stale evidence, upcoming obligations, recent changes, and active Matters.
- Matters SHOULD combine summary, scope, evidence, decisions, actions, response or outcome, verification, and history.
- Configure MUST use versioned drafts, impact preview, simulation, maker-checker approval, effective dates, and rollback.
- Large populations use tables/worklists with denominators and authorization-aware bulk actions.
- Material approvals show scope, evidence, uncertainty, authority, side effects, next state, and verification in one view.
- No mandatory chat, KPI walls, control walls, decorative graph canvases, hidden hover-only actions, or context-free approval.

## 13. Testing and definition of done

Every feature requires tests for:

- domain and state invariants;
- use-case actor and authority resolution;
- wrong scope, conflict, delegation, absence, escalation, and revocation;
- source trust, contradiction, and evidence sufficiency;
- invitation forwarding, expiry, replay, step-up, wrong recipient, and revocation;
- AI and source degraded modes;
- tenant and protected-data leakage;
- concurrency, idempotency, retry, partial failure, and resume;
- accessibility, mobile, localization, and low bandwidth;
- timed first-use and repeat-use journeys;
- workload, latency, recovery, and scale profile;
- point-in-time reconstruction.

A feature is complete only when:

- the use-case outcome and maturity are documented;
- responsibility and authority are correctly resolved;
- known context is reused;
- the routine or checkpoint effort target is met without quality regression;
- the result is evidence-backed and reconstructable;
- safe fallbacks exist;
- documentation, ADRs, implementation plan, and tests are synchronized.

If the work is possible but cumbersome, unsafe to route, ambiguous under delegation, dependent on hidden configuration, or unproven at expected scale, it is not finished.
