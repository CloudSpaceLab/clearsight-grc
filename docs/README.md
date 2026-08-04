# ClearSight Documentation Map

This directory contains the canonical product, architecture, implementation, and quality specifications for ClearSight.

## Start here

1. [`../README.md`](../README.md) — product vision, operating model, core capabilities, and initial product wedge.
2. [`../AGENTS.md`](../AGENTS.md) — mandatory implementation rules and visual/functional non-regression constraints.
3. [`implementation-plan.md`](implementation-plan.md) — comprehensive phased delivery plan with tasks, dependencies, deliverables, and acceptance gates.

## Product

- [`product/differentiation.md`](product/differentiation.md) — the product moat, competitor-category boundaries, bank-first mechanisms, and differentiation tests.
- [`product/experience-principles.md`](product/experience-principles.md) — information architecture, interaction grammar, visual language, evidence-capture experience, protected-reporting experience, accessibility, and golden screens.

## Architecture

- [`architecture/risk-graph-and-decision-engine.md`](architecture/risk-graph-and-decision-engine.md) — canonical entities and relationships, temporal graph, signals, Materiality Compiler, risk appetite, Decision Ledger, action model, and verification contracts.
- [`architecture/living-evidence-fabric.md`](architecture/living-evidence-fabric.md) — claims, immutable evidence, assertions, sufficiency, contradiction, evidence debt, best-source resolution, dynamic micro-requests, protected evidence, and chain of custody.
- [`architecture/governed-ai-operators.md`](architecture/governed-ai-operators.md) — operator identities, action classes, model gateway, grounding, tool policy, authority thresholds, prompt-injection defense, evaluation, and audit contract.

## Quality

- [`quality/acceptance-tests.md`](quality/acceptance-tests.md) — golden journeys and required domain, security, authorization, evidence, AI, visual, accessibility, performance, resilience, and migration tests.

## Planned decision records

Architecture decisions should be added under `docs/decisions/` as numbered ADRs. At minimum, the implementation plan requires ADRs for:

- application modularity and service split criteria;
- backend and frontend stack;
- temporal/versioning model;
- workflow runtime;
- authorization engine;
- evidence object storage and integrity;
- event and outbox architecture;
- graph projection and dedicated graph-engine decision gate;
- search/vector architecture;
- model gateway and provider routing;
- deployment modes;
- and audit/observability separation.

## Canonical rule

When implementation behavior conflicts with these documents, the conflict must be resolved explicitly. Do not silently change product semantics in code.

A material change should update:

- the relevant product or architecture specification;
- the implementation-plan task or decision;
- affected acceptance tests;
- and an ADR when the change alters a foundational technical choice.