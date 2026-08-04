# ClearSight Documentation Map

This directory contains the canonical product, architecture, implementation, quality, and review specifications for ClearSight.

The documentation is deliberately layered so that internal architecture does not become user-interface architecture.

---

# Start here

1. [`../README.md`](../README.md) — final product vision, continuous compliance model, external-authority workflows, experience, and initial product wedge.
2. [`product/continuous-compliance-operating-model.md`](product/continuous-compliance-operating-model.md) — canonical Programs and Matters model for ongoing compliance, legacy workflow replacement, NDPA, regulatory change, and authority work.
3. [`product/operating-model.md`](product/operating-model.md) — canonical Scope, Exposure Pattern, Risk Situation, Claim, Evidence Recipe, Observation, Conclusion, Decision, and Verification semantics used inside Programs and Matters.
4. [`product/regulatory-and-enforcement-intelligence.md`](product/regulatory-and-enforcement-intelligence.md) — authoritative regulatory change, supervisory work, enforcement cases, rule composition, response packages, and protected authority workflows.
5. [`product/experience-principles.md`](product/experience-principles.md) — information architecture, interaction patterns, visual system, capture, population workflows, accessibility, and golden screens.
6. [`../AGENTS.md`](../AGENTS.md) — mandatory implementation and non-regression rules.
7. [`implementation-plan.md`](implementation-plan.md) — phased delivery plan and acceptance gates.
8. [`quality/acceptance-tests.md`](quality/acceptance-tests.md) — end-to-end proof and release requirements.
9. [`quality/regulatory-and-enforcement-acceptance-tests.md`](quality/regulatory-and-enforcement-acceptance-tests.md) — specialized source, legal-authority, obligation, case, response, and leakage tests.

---

# Product

- [`product/continuous-compliance-operating-model.md`](product/continuous-compliance-operating-model.md) — stable Programs, dynamic Matters, the requirement-to-evidence chain, continuous Compliance State, trigger-driven work, and migration from compliance registers, RCSA, KRI, BIA, vendor, loss, policy, certification, and dashboard workflows.
- [`product/operating-model.md`](product/operating-model.md) — the shared bank-domain semantics supporting Programs and Matters.
- [`product/differentiation.md`](product/differentiation.md) — product moat, category boundaries, bank-size adaptability, and differentiation tests.
- [`product/experience-principles.md`](product/experience-principles.md) — interaction and visual requirements. This document requires a later terminology pass to align its top-level surfaces with Today, Programs, Work, Explore, Configure, and contextual Respond/Capture.
- [`product/regulatory-and-enforcement-intelligence.md`](product/regulatory-and-enforcement-intelligence.md) — External Authority Workbench semantics for normative regulation, supervisory matters, investigative requests, compliance-rule packages, case directives, response packages, source trust, AI safety, and migration from legacy registers.

---

# Architecture

- [`architecture/product-semantics-mapping.md`](architecture/product-semantics-mapping.md) — mapping between product objects and the deeper graph, evidence, decision, workflow, and AI architecture. This document requires expansion for Programs, Requirements, Controls, Compliance State, and Matters.
- [`architecture/risk-graph-and-decision-engine.md`](architecture/risk-graph-and-decision-engine.md) — canonical entities and relationships, temporal graph, signals, materiality, appetite, decision records, action, and verification.
- [`architecture/living-evidence-fabric.md`](architecture/living-evidence-fabric.md) — claims, immutable evidence, assertions, sufficiency, contradiction, evidence debt, source resolution, dynamic requests, protected evidence, and chain of custody.
- [`architecture/governed-ai-operators.md`](architecture/governed-ai-operators.md) — operator identities, action classes, model gateway, grounding, tool policy, authority thresholds, prompt-injection defence, evaluation, and audit.

Architecture documents explain how ClearSight works internally. They must not override the Programs-and-Matters product semantics or become mandatory navigation concepts.

---

# Delivery and quality

- [`implementation-plan.md`](implementation-plan.md) — current delivery sequence beginning with source trust and bounded bank workflows. It requires a follow-on expansion for the continuous-compliance Program model and External Authority Workbench.
- [`quality/acceptance-tests.md`](quality/acceptance-tests.md) — golden journeys and domain, source, import, security, evidence, AI, visual, accessibility, localization, performance, resilience, and migration tests.
- [`quality/regulatory-and-enforcement-acceptance-tests.md`](quality/regulatory-and-enforcement-acceptance-tests.md) — final-versus-draft classification, exact provision lineage, applicability, supervisory remediation, protected authority requests, legal-basis uncertainty, subject resolution, KYC/address workflows, suspicious-reporting authority, response packages, amendment handling, and systemic-signal minimization.

Required future quality expansion:

- recurring Program state and Evidence Contract refresh;
- ROPA and DPIA triggers;
- annual filing assembled from continuous evidence;
- RCSA prefill and focused challenge;
- KRI derivation from governed records;
- legacy register views generated from shared objects;
- Program-to-Matter transitions;
- cross-program reuse with purpose and authorization controls.

---

# Reviews

- [`reviews/2026-08-04-visual-and-document-conformance-review.md`](reviews/2026-08-04-visual-and-document-conformance-review.md) — audit of missing visual aspects, prior document staleness, canonical corrections, and remaining architecture-alignment work.
- [`reviews/2026-08-04-prospective-bank-workflow-patterns.md`](reviews/2026-08-04-prospective-bank-workflow-patterns.md) — sanitized analysis of prospective-bank compliance, IT risk, operational risk, privacy, vendor, BIA, KRI, loss, dashboard, and authority-request workflows.

Future major design or documentation reviews should be added under `docs/reviews/` with an absolute date.

---

# Planned decision records

Architecture decisions should be added under `docs/decisions/` as numbered ADRs.

Minimum ADRs:

- modular core and service split criteria;
- backend and frontend stack;
- Programs, Requirements, Controls, Matters, and Compliance State boundaries;
- scope hierarchy and institution model;
- temporal/versioning model;
- Observation and Evidence Contract contracts;
- Source Registry and source authority;
- authority-source authentication and source-version lineage;
- regulatory provision segmentation and Directive Atom schema;
- protected Authority Request Case isolation;
- spreadsheet and media processing security;
- offline capture boundary;
- workflow runtime and trigger engine;
- authorization engine and inference resistance;
- evidence object storage and integrity;
- outbox and event architecture;
- graph projection and dedicated-engine decision gate;
- search and vector architecture;
- model gateway and provider routing;
- protected reporting isolation;
- initial deployment mode;
- audit and observability separation.

---

# Canonical precedence

When requirements conflict, apply:

1. safety, confidentiality, legal boundaries, and tenant isolation;
2. [`../README.md`](../README.md) for product intent;
3. [`product/continuous-compliance-operating-model.md`](product/continuous-compliance-operating-model.md) for Programs, Matters, continuous compliance, and legacy-workflow semantics;
4. [`product/operating-model.md`](product/operating-model.md) for shared domain objects;
5. specialized product specifications, including [`product/regulatory-and-enforcement-intelligence.md`](product/regulatory-and-enforcement-intelligence.md), for domain behavior;
6. [`product/experience-principles.md`](product/experience-principles.md) for user and visual behaviour;
7. [`../AGENTS.md`](../AGENTS.md) for normative implementation rules;
8. architecture documents for internal mechanisms;
9. implementation plan for sequencing;
10. acceptance tests for release proof.

A material change must update:

- the relevant product document;
- affected architecture mapping or ADR;
- implementation-plan task or gate;
- acceptance tests;
- and the conformance review where it resolves a listed issue.

Do not silently change product semantics in code, schema, prompts, UI, or integrations.
