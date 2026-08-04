# ClearSight GRC

> **The AI-native continuous compliance and risk operating system for banks.**  
> Know what applies. Keep proof current. Route the right people. Handle what changed. Verify the outcome.

ClearSight is designed for banks whose compliance, risk, security, privacy, resilience, audit, legal, business, and executive teams need a simpler alternative to fragmented registers, recurring questionnaires, manual evidence chasing, and dashboard-driven status reporting.

The product goal is:

> **Help every stakeholder understand what the institution must do, what currently proves it, what changed or became uncertain, who is responsible, who must review or authorize, and whether the required outcome was achieved—with the minimum reasonable human effort.**

ClearSight is at the **product-definition and architecture stage**. The repository defines intended behavior, release gates, and architecture—not completed implementation.

## Product model

Users operate two primary objects:

- **Programs** maintain continuing obligations, controls, evidence, reviews, filings, exceptions, and assurance.
- **Matters** handle bounded change, failure, uncertainty, findings, incidents, decisions, external requests, and remediation.

Every material position remains traceable through:

```text
Authority Source or Standard
→ Requirement and Applicability
→ Control Objective and Implementation
→ Evidence Contract and Current Observations
→ Conclusion or Compliance State
→ Matter, Decision, Action, Response, and Verification
```

A completed task, uploaded file, submitted response, or implemented change is not automatically a verified outcome.

## The operating experience

ClearSight should do the assembly work before asking a person to act.

Routine authorized work should normally require only a few clear steps and less than five minutes of active effort. Complex work should still reach a clear, saved, correctly routed next state within five minutes.

The product achieves this through:

- source-backed prefill from approved bank systems and inventories;
- minimum-question evidence requests;
- role-specific workspaces and one dominant next action **per actor and workflow state**;
- easy responsibility, review, authorization, delegation, and escalation configuration;
- secure internal and external invitation-based capture;
- grounded AI recommendations and first drafts;
- review by exception with visible denominators and sampling controls;
- durable save, resume, handoff, and point-in-time reconstruction;
- verification before closure.

Ease of use never bypasses evidence, legal review, segregation of duties, privacy, purpose limitation, or material decision authority.

## Responsibility and authority routing

ClearSight separates:

- who performs or supplies information;
- who owns the obligation or Matter;
- who reviews evidence or recommendations;
- who independently challenges;
- who authorizes a material decision or external representation;
- who receives escalation when time, authority, conflict, or availability prevents progress.

Roles are not hard-coded user labels. They are resolved from versioned role templates, organizational positions, scoped assignments, authority grants, delegation, conflicts, thresholds, and workflow state.

Administrators should be able to define and simulate routing through a visual matrix and sequence builder without creating unsafe generic permissions or customer-specific code forks.

See [`docs/product/authority-routing-and-escalation.md`](docs/product/authority-routing-and-escalation.md).

## Respond and Capture

ClearSight supports focused evidence and data collection from:

- employees and control owners;
- branches and field staff;
- vendors and other external parties;
- customers where a governed case requires it;
- protected or anonymous reporters.

A recipient receives a purpose-bound wizard containing only the authorized request, known context, unresolved facts, acceptable response forms, deadline, sensitivity, and redirect or concern options.

Internal and external invitations use short-lived, revocable, request-scoped access. Sensitive content must not appear in notifications or URLs. Anonymous protected reporting uses a separate identity-isolated two-way channel.

See [`docs/product/respond-and-capture.md`](docs/product/respond-and-capture.md).

## Initial product wedge

The first release proves four connected journeys in one pilot bank and legal entity:

1. **Continuous NDPA Program** — targeted ROPA updates, DPIA and breach Matters, evidence refresh, and filing readiness.
2. **Regulatory Change Matter** — official source to approved obligations, affected controls, owners, implementation, and verification.
3. **Protected Authority Request Matter** — legal review, subject resolution, focused tasks, governed response package, transmission, and acknowledgement.
4. **Legacy Finding or Exception** — import, ownership, evidence review, action, and verified remediation.

The broader product catalogue—including incidents, resilience, third-party risk, RCSA, KRI, audit, policy, conduct, protected reporting, automation, and multi-entity operation—is documented with release maturity and acceptance references in [`docs/product/use-case-catalogue.md`](docs/product/use-case-catalogue.md).

## Product surfaces

- **Today** — a small role-specific brief of Programs and Matters requiring attention.
- **Programs** — continuing compliance position, obligations, evidence, schedule, gaps, and assurance.
- **Work** — Matters, requests, reviews, decisions, actions, responses, and verification.
- **Explore** — authorized inquiry across requirements, services, systems, vendors, evidence, history, and relationships.
- **Configure** — restricted, versioned administration of Programs, sources, authority, routing, evidence contracts, retention, integrations, and automation.
- **Respond and Capture** — focused web and mobile experiences for invited or protected participants.

## Governed AI

AI may extract, compare, map, summarize, recommend, draft, and identify missing proof. It does not become the source of institutional truth, evidence, legal authority, risk authority, or the only method of operation.

Every material AI output requires:

- source and version lineage;
- explicit versus inferred values;
- scope, assumptions, uncertainty, and contradiction;
- structured editable output;
- required authority and expected next state;
- policy and authorization checks outside the model;
- degraded manual operation.

External automation engines such as Probo may execute approved commodity tasks, but ClearSight retains institutional context, authority, evidence, audit, and outcome verification.

## System and data architecture

ClearSight begins as a coherent modular core with:

- authoritative relational data for Programs, Matters, assignments, authority, decisions, and temporal history;
- versioned object storage for source and evidence artifacts;
- durable workflows, timers, outbox/inbox, and idempotent external execution;
- authorization-aware current-state, search, graph, vector, reporting, and work-queue projections;
- scalable observation and evidence ingestion;
- governed AI and integration gateways;
- explicit performance, recovery, tenancy, and deployment profiles.

Deterministic context must appear before AI. Material commands are strongly consistent; large ingestion, projection, package generation, and verification work are asynchronous and resumable.

See [`docs/architecture/system-data-and-performance.md`](docs/architecture/system-data-and-performance.md).

## Design standard

ClearSight should feel calm, precise, direct, premium, and institutional.

- banking language before GRC jargon;
- Programs before control walls;
- Matters before module hopping;
- cards for small attention queues;
- tables for populations and requirements;
- comparisons for contradiction and version change;
- timelines for history;
- restrained glass, glow, and motion;
- full light/dark, keyboard, screen-reader, mobile, and low-bandwidth support;
- no mandatory chatbot interaction;
- no dark patterns or urgency theatre;
- green only for an evidence-supported acceptable or verified state.

## Non-goals

ClearSight is not:

- a generic form builder or workflow platform;
- a prettier spreadsheet register;
- a permanent questionnaire engine;
- a generic document-management system;
- a SIEM, fraud, AML, or transaction-monitoring engine;
- a core banking platform or payment switch;
- an autonomous compliance, risk, legal, or audit officer;
- an opaque AI scoring product;
- a collection of disconnected GRC modules.

It provides the governed Program, Matter, evidence, authority, decision, response, verification, and assurance layer across specialist systems.

## Start here

1. [`docs/product/use-case-catalogue.md`](docs/product/use-case-catalogue.md)
2. [`docs/product/continuous-compliance-operating-model.md`](docs/product/continuous-compliance-operating-model.md)
3. [`docs/product/authority-routing-and-escalation.md`](docs/product/authority-routing-and-escalation.md)
4. [`docs/product/respond-and-capture.md`](docs/product/respond-and-capture.md)
5. [`docs/product/ease-of-use-standard.md`](docs/product/ease-of-use-standard.md)
6. [`docs/product/operating-model.md`](docs/product/operating-model.md)
7. [`docs/product/experience-principles.md`](docs/product/experience-principles.md)
8. [`docs/product/ux-and-visual-language.md`](docs/product/ux-and-visual-language.md)
9. [`docs/architecture/system-data-and-performance.md`](docs/architecture/system-data-and-performance.md)
10. [`docs/implementation-plan.md`](docs/implementation-plan.md)
11. [`docs/quality/release-gates-and-traceability.md`](docs/quality/release-gates-and-traceability.md)
12. [`AGENTS.md`](AGENTS.md)

## Product invariants

1. Programs maintain continuity; Matters handle change and exception.
2. Prefill before asking; search existing evidence before requesting more.
3. One dominant next action per actor and workflow state.
4. Responsibility, review, authorization, challenge, and escalation remain distinct.
5. Scope, purpose, authority, evidence, uncertainty, and consequence are visible before material action.
6. Review by exception never hides the denominator, source health, or full-review trigger.
7. Invitation links grant the smallest request-scoped capability and remain revocable.
8. AI proposes; governed policy and authorized humans decide material matters.
9. External execution is implementation evidence, not verified outcome.
10. Material records are versioned and reconstructable.
11. The system remains usable without AI or a live integration.
12. Correctness, comprehension, and safety gates override speed targets.

**ClearSight succeeds when high-accountability bank governance work becomes continuously prepared, correctly routed, minimally demanding, and defensible years later.**
