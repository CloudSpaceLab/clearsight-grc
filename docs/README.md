# ClearSight Documentation Map

This directory contains the canonical product, architecture, implementation, quality, and review specifications for ClearSight.

The documentation is deliberately layered so that internal architecture does not become user-interface architecture and legacy register concepts do not become separate truth systems.

---

# Start here

1. [`../README.md`](../README.md) — product vision, customer problem, Programs, Matters, continuous compliance, primary surfaces, and product wedge.
2. [`product/continuous-compliance-operating-model.md`](product/continuous-compliance-operating-model.md) — how continuing Programs and bounded Matters replace disconnected registers and periodic reconstruction.
3. [`product/operating-model.md`](product/operating-model.md) — canonical Program, Matter, Scope, Requirement, control, evidence, Decision, Action, Verification, Response Package, and relationship semantics.
4. [`product/regulatory-and-enforcement-intelligence.md`](product/regulatory-and-enforcement-intelligence.md) — regulatory change, supervisory work, protected authority cases, rule composition, response packages, and source lineage.
5. [`product/experience-principles.md`](product/experience-principles.md) — Today, Programs, Work, Explore, Configure, contextual Capture/Respond, visual language, accessibility, and golden screens.
6. [`../AGENTS.md`](../AGENTS.md) — mandatory implementation and non-regression rules.
7. [`architecture/continuous-compliance-architecture.md`](architecture/continuous-compliance-architecture.md) — cross-cutting architecture that composes Programs, Matters, graph, evidence, workflow, authorization, regulatory ingestion, protected cases, and AI.
8. [`implementation-plan.md`](implementation-plan.md) — phased delivery plan beginning with source trust, Program engine, Matter engine, NDPA, and external-authority workflows.
9. [`quality/acceptance-tests.md`](quality/acceptance-tests.md) — complete Program, Matter, evidence, security, visual, AI, recovery, and historical tests.
10. [`quality/regulatory-and-enforcement-acceptance-tests.md`](quality/regulatory-and-enforcement-acceptance-tests.md) — specialized external-authority tests.

---

# Product

- [`product/continuous-compliance-operating-model.md`](product/continuous-compliance-operating-model.md) — canonical product aggregate model: Programs maintain; Matters mobilize.
- [`product/operating-model.md`](product/operating-model.md) — universal shared objects and lifecycles.
- [`product/differentiation.md`](product/differentiation.md) — product moat, category boundaries, source-to-outcome differentiation, and feature tests.
- [`product/experience-principles.md`](product/experience-principles.md) — continuing Program and Matter experience, operational populations, regulatory interpretation, protected cases, imports, capture, accessibility, and visual quality.
- [`product/regulatory-and-enforcement-intelligence.md`](product/regulatory-and-enforcement-intelligence.md) — External Authority Workbench semantics for normative regulation, supervisory Matters, investigative requests, protected subject work, compliance rule packages, and responses.

---

# Architecture

- [`architecture/continuous-compliance-architecture.md`](architecture/continuous-compliance-architecture.md) — cross-cutting bounded contexts, Program computation, trigger engine, Matter orchestration, Regulatory Change Compiler, protected Authority Request Cases, legacy migration, APIs/events, authorization, AI, and deployment.
- [`architecture/product-semantics-mapping.md`](architecture/product-semantics-mapping.md) — mapping between Programs, Matters, shared primitives, and deeper component mechanisms.
- [`architecture/risk-graph-and-decision-engine.md`](architecture/risk-graph-and-decision-engine.md) — canonical entities and relationships, temporal context, signals, materiality, appetite, Decisions, Actions, and verification.
- [`architecture/living-evidence-fabric.md`](architecture/living-evidence-fabric.md) — Claims, immutable evidence, assertions, sufficiency, contradiction, evidence debt, source resolution, dynamic requests, protected evidence, and chain of custody.
- [`architecture/governed-ai-operators.md`](architecture/governed-ai-operators.md) — operator identities, action classes, model gateway, grounding, tool policy, authority thresholds, prompt-injection defence, evaluation, and audit.

The continuous-compliance architecture defines how component architectures compose. Component documents must not override Programs-and-Matters semantics or become mandatory navigation.

---

# Delivery and quality

- [`implementation-plan.md`](implementation-plan.md) — delivery sequence: semantics and pilot, trust foundation, sources and ingestion, Program engine, Matter engine, NDPA, external authority, UX, governed AI, integrations, assurance, and GA.
- [`quality/acceptance-tests.md`](quality/acceptance-tests.md) — legacy-register migration, ROPA, DPIA, breaches, filings, regulatory change, authority cases, findings, ATM/POS, KRIs, source degradation, malicious content, scope, degraded mode, and reconstruction.
- [`quality/regulatory-and-enforcement-acceptance-tests.md`](quality/regulatory-and-enforcement-acceptance-tests.md) — final-versus-draft source classification, exact provision lineage, applicability, supervisory remediation, legal-instrument uncertainty, subject resolution, KYC/address workflows, suspicious-reporting authority, response packages, amendments, and minimized systemic signals.

---

# Reviews

- [`reviews/2026-08-04-visual-and-document-conformance-review.md`](reviews/2026-08-04-visual-and-document-conformance-review.md) — visual and documentation audit.
- [`reviews/2026-08-04-prospective-bank-workflow-patterns.md`](reviews/2026-08-04-prospective-bank-workflow-patterns.md) — sanitized analysis of prospective-bank compliance, IT risk, operational risk, privacy, vendor, BIA, KRI, loss, and dashboard workflows.

Future major reviews should be stored under `docs/reviews/` with an absolute date and should not contain identifiable customer data in a public repository.

---

# Planned decision records

Architecture decisions should be added under `docs/decisions/`.

Minimum ADRs:

- modular core and split criteria;
- backend and frontend stack;
- Program and Matter aggregate boundaries;
- scope hierarchy and institution/licence model;
- temporal/versioning model;
- Observation and Evidence Contract;
- Source Registry and source authority;
- Authority Source authentication and provision segmentation;
- Requirement, applicability, and Directive Atom schema;
- protected reporting and Authority Request Case isolation;
- spreadsheet/document/media processing security;
- offline capture boundary;
- workflow runtime;
- authorization and inference resistance;
- object storage and evidence integrity;
- outbox and event architecture;
- graph/search/vector projections;
- model gateway and provider routing;
- initial deployment mode;
- audit and observability separation.

---

# Canonical precedence

When requirements conflict, apply:

1. safety, confidentiality, legal boundaries, and tenant isolation;
2. [`../README.md`](../README.md) for product intent;
3. [`product/continuous-compliance-operating-model.md`](product/continuous-compliance-operating-model.md) for Programs-and-Matters behavior;
4. [`product/operating-model.md`](product/operating-model.md) for shared semantics;
5. specialized product specifications for domain behavior;
6. [`product/experience-principles.md`](product/experience-principles.md) for user and visual behavior;
7. [`../AGENTS.md`](../AGENTS.md) for normative implementation rules;
8. [`architecture/continuous-compliance-architecture.md`](architecture/continuous-compliance-architecture.md) for cross-cutting internal composition;
9. component architecture documents for detailed mechanisms;
10. implementation plan for sequencing;
11. acceptance tests for release proof.

A material change must update:

- the relevant product document;
- affected architecture mapping or ADR;
- implementation-plan task or gate;
- acceptance tests;
- and a review document where it resolves a recorded issue.

Do not silently change product semantics in code, schema, prompts, UI, integrations, or reporting projections.