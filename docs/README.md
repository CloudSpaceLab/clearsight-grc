# ClearSight Documentation Map

This directory contains the canonical product, experience, architecture, delivery, quality, and review specifications for ClearSight.

The documentation is layered so that:

- internal architecture does not become user-interface architecture;
- legacy registers do not become separate truth systems;
- usability is treated as a measurable product requirement;
- Programs maintain continuing obligations;
- Matters handle change, exception, harm, uncertainty, and external demand.

---

# Required reading order

1. [`../README.md`](../README.md) — product promise, Programs, Matters, continuous compliance, authority work, and five-minute usability standard.
2. [`product/continuous-compliance-operating-model.md`](product/continuous-compliance-operating-model.md) — how Programs remain current and triggers create Matters.
3. [`product/ease-of-use-standard.md`](product/ease-of-use-standard.md) — mandatory active-effort budgets, prefill, source reuse, AI recommendations, few-step flows, save/resume, and usability metrics.
4. [`product/operating-model.md`](product/operating-model.md) — canonical Program, Matter, Scope, Requirement, Control, Observation, Evidence, Decision, Response, and Verification semantics.
5. [`product/experience-principles.md`](product/experience-principles.md) — Today, Programs, Work, Explore, Configure, Respond/Capture, components, interaction rules, visual system, and timed golden flows.
6. [`product/regulatory-and-enforcement-intelligence.md`](product/regulatory-and-enforcement-intelligence.md) — regulatory change, supervisory work, protected authority cases, rule composition, and response packages.
7. [`product/differentiation.md`](product/differentiation.md) — product moat and category boundaries.
8. [`../AGENTS.md`](../AGENTS.md) — mandatory contributor and non-regression rules.
9. [`architecture/continuous-compliance-architecture.md`](architecture/continuous-compliance-architecture.md) — cross-cutting Program, Matter, context assembly, prefill, recommendation, trigger, workflow-efficiency, and authorization architecture.
10. [`implementation-plan.md`](implementation-plan.md) — phased delivery plan with usability gates.
11. [`quality/acceptance-tests.md`](quality/acceptance-tests.md) — timed end-to-end, usability, security, evidence, AI, visual, accessibility, and resilience requirements.
12. [`quality/regulatory-and-enforcement-acceptance-tests.md`](quality/regulatory-and-enforcement-acceptance-tests.md) — specialized authority-source, legal-basis, case, response, and leakage tests.

---

# Product specifications

- [`product/continuous-compliance-operating-model.md`](product/continuous-compliance-operating-model.md) — Programs, Matters, compliance chain, triggers, NDPA, source-led workflows, regulatory and authority work, and legacy transition.
- [`product/ease-of-use-standard.md`](product/ease-of-use-standard.md) — less-than-five-minute routine target, safe complex-work checkpoint, prefill, minimum-question generation, integration-led usability, recommendation contract, review by exception, and quantitative targets.
- [`product/operating-model.md`](product/operating-model.md) — universal product objects and invariants used across all domains and bank sizes.
- [`product/experience-principles.md`](product/experience-principles.md) — information architecture, components, visual forms, interaction budgets, accessibility, responsive behavior, and golden flows.
- [`product/regulatory-and-enforcement-intelligence.md`](product/regulatory-and-enforcement-intelligence.md) — External Authority Workbench semantics.
- [`product/differentiation.md`](product/differentiation.md) — source-led context assembly, minimum-question evidence, governed recommendations, continuous Programs, Matters, and verified outcomes as the moat.

---

# Architecture

- [`architecture/continuous-compliance-architecture.md`](architecture/continuous-compliance-architecture.md) — authoritative stores, Program computation, Matter composition, Source Registry, Context Assembly and Prefill, Minimum-Question Compiler, Recommendation and Task Compilation, trigger engine, workflow telemetry, save/resume, degraded mode, and derived legacy views.
- [`architecture/product-semantics-mapping.md`](architecture/product-semantics-mapping.md) — mapping from user-facing objects to graph, evidence, decision, workflow, and AI mechanisms.
- [`architecture/risk-graph-and-decision-engine.md`](architecture/risk-graph-and-decision-engine.md) — temporal relationships, materiality, appetite, decisions, actions, and verification.
- [`architecture/living-evidence-fabric.md`](architecture/living-evidence-fabric.md) — claims, evidence, assertions, sufficiency, contradiction, source resolution, requests, chain of custody, and protected evidence.
- [`architecture/governed-ai-operators.md`](architecture/governed-ai-operators.md) — model gateway, identities, tools, grounding, authority, prompt-injection defenses, evaluation, monitoring, and audit.

Component architecture documents explain internal mechanisms. They must not override Program/Matters semantics, usability standards, or primary navigation.

---

# Delivery and quality

- [`implementation-plan.md`](implementation-plan.md) — delivery beginning with workflow budgets, source inventories, identity and trust, progressive ingestion, Programs, Matters, NDPA, regulatory change, authority cases, legacy migration, AI, and enterprise scale.
- [`quality/acceptance-tests.md`](quality/acceptance-tests.md) — timed first-use and repeat-use tests, active-effort budgets, prefill, integration reuse, AI usefulness, accessibility parity, and complete golden journeys.
- [`quality/regulatory-and-enforcement-acceptance-tests.md`](quality/regulatory-and-enforcement-acceptance-tests.md) — specialized source-status, applicability, authority, subject-resolution, response, suspicious-reporting, amendment, and protected-case tests.

---

# Reviews

- [`reviews/2026-08-04-visual-and-document-conformance-review.md`](reviews/2026-08-04-visual-and-document-conformance-review.md) — prior visual and documentation audit.
- [`reviews/2026-08-04-prospective-bank-workflow-patterns.md`](reviews/2026-08-04-prospective-bank-workflow-patterns.md) — sanitized analysis of prospective-bank registers, checklists, workplans, KRIs, BIA, losses, vendor, privacy, and dashboard workflows.

Future major reviews should be dated and placed under `docs/reviews/`.

---

# Required architecture decisions

ADRs should cover at minimum:

- modular core and split criteria;
- backend and frontend stack;
- Program and Matter aggregates;
- scope hierarchy and institution profile;
- temporal and versioning strategy;
- Source Registry and Observation contract;
- inventory adapter and prefill strategy;
- Context Assembly package;
- Minimum-Question Compiler;
- Recommendation Contract and AI gateway;
- workflow runtime and save/resume;
- workflow-efficiency telemetry and privacy;
- authorization and inference resistance;
- evidence storage and integrity;
- regulatory source segmentation;
- protected authority-case isolation;
- spreadsheet and media processing security;
- offline capture;
- outbox and events;
- search, graph, and vector projections;
- initial deployment mode;
- audit and observability separation.

---

# Canonical precedence

When requirements conflict:

1. safety, confidentiality, legal boundaries, and tenant isolation;
2. [`../README.md`](../README.md) for product intent;
3. [`product/continuous-compliance-operating-model.md`](product/continuous-compliance-operating-model.md) and [`product/ease-of-use-standard.md`](product/ease-of-use-standard.md) for operating and effort semantics;
4. [`product/operating-model.md`](product/operating-model.md) for canonical objects;
5. specialized product specifications;
6. [`product/experience-principles.md`](product/experience-principles.md) for UI and interaction;
7. [`../AGENTS.md`](../AGENTS.md) for contributor rules;
8. architecture documents for internal mechanisms;
9. implementation plan for sequencing;
10. acceptance tests for release proof.

A material change must update:

- relevant product specification;
- experience and ease-of-use rules where workflow changes;
- architecture mapping or ADR;
- implementation-plan task or gate;
- acceptance tests;
- review document when resolving a listed issue.

Do not silently change product semantics, workflow-effort expectations, source behavior, prompts, UI, or integrations in code.