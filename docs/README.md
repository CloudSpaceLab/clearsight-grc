# ClearSight Documentation Map

This directory contains the canonical product, architecture, delivery, quality, and review specifications for ClearSight.

The documentation is deliberately layered:

- users operate Programs, Matters, focused requests, decisions, actions, and outcomes;
- roles, review, authorization, delegation, and escalation are explicit product behavior;
- internal architecture must not become navigation or user terminology;
- legacy registers are derived views, not separate truth systems;
- usability, correctness, security, and performance are release requirements.

## Required reading order

1. [`../README.md`](../README.md) — product promise and boundaries.
2. [`product/use-case-catalogue.md`](product/use-case-catalogue.md) — target customers, personas, use cases, release maturity, and required use-case contract.
3. [`product/continuous-compliance-operating-model.md`](product/continuous-compliance-operating-model.md) — Programs, Matters, triggers, evidence-backed state, and closure.
4. [`product/authority-routing-and-escalation.md`](product/authority-routing-and-escalation.md) — responsibility, review, challenge, authorization, delegation, substitution, and escalation.
5. [`product/respond-and-capture.md`](product/respond-and-capture.md) — internal/external invitation wizards, evidence collection, field capture, and protected reporting boundary.
6. [`product/ease-of-use-standard.md`](product/ease-of-use-standard.md) — active-effort budgets, prefill, minimum-question requests, save/resume, and review by exception.
7. [`product/operating-model.md`](product/operating-model.md) — canonical domain objects.
8. [`product/experience-principles.md`](product/experience-principles.md) — information architecture and interaction grammar.
9. [`product/ux-and-visual-language.md`](product/ux-and-visual-language.md) — implementation-ready visual and responsive system.
10. [`product/regulatory-and-enforcement-intelligence.md`](product/regulatory-and-enforcement-intelligence.md) — regulatory change, supervisory work, and protected authority requests.
11. [`architecture/system-data-and-performance.md`](architecture/system-data-and-performance.md) — system, data, tenancy, workflow, invitation, performance, scale, and recovery architecture.
12. [`architecture/continuous-compliance-architecture.md`](architecture/continuous-compliance-architecture.md) — cross-cutting Program, Matter, evidence, trigger, recommendation, and workflow architecture.
13. [`../AGENTS.md`](../AGENTS.md) — mandatory implementation and non-regression rules.
14. [`implementation-plan.md`](implementation-plan.md) — phased delivery plan.
15. [`quality/release-gates-and-traceability.md`](quality/release-gates-and-traceability.md) — use-case traceability and cross-cutting release gates.
16. Domain-specific acceptance tests under [`quality/`](quality/).

## Canonical product specifications

- [`product/use-case-catalogue.md`](product/use-case-catalogue.md)
- [`product/continuous-compliance-operating-model.md`](product/continuous-compliance-operating-model.md)
- [`product/authority-routing-and-escalation.md`](product/authority-routing-and-escalation.md)
- [`product/respond-and-capture.md`](product/respond-and-capture.md)
- [`product/ease-of-use-standard.md`](product/ease-of-use-standard.md)
- [`product/operating-model.md`](product/operating-model.md)
- [`product/experience-principles.md`](product/experience-principles.md)
- [`product/ux-and-visual-language.md`](product/ux-and-visual-language.md)
- [`product/regulatory-and-enforcement-intelligence.md`](product/regulatory-and-enforcement-intelligence.md)
- [`product/differentiation.md`](product/differentiation.md)

## Architecture

- [`architecture/system-data-and-performance.md`](architecture/system-data-and-performance.md) — canonical system and non-functional architecture.
- [`architecture/continuous-compliance-architecture.md`](architecture/continuous-compliance-architecture.md) — composition of Programs, Matters, requests, triggers, and recommendations.
- [`architecture/product-semantics-mapping.md`](architecture/product-semantics-mapping.md) — product-to-internal mapping.
- [`architecture/risk-graph-and-decision-engine.md`](architecture/risk-graph-and-decision-engine.md) — materiality, appetite, decisions, actions, and verification.
- [`architecture/living-evidence-fabric.md`](architecture/living-evidence-fabric.md) — claims, evidence, contradiction, sufficiency, and chain of custody.
- [`architecture/governed-ai-operators.md`](architecture/governed-ai-operators.md) — AI identities, tools, grounding, approval, evaluation, and monitoring.

Component architecture documents explain mechanisms. They must not override the simpler product semantics, actor routing, invitation contract, usability standard, or primary navigation.

## Delivery and quality

- [`implementation-plan.md`](implementation-plan.md)
- [`quality/release-gates-and-traceability.md`](quality/release-gates-and-traceability.md)
- [`quality/acceptance-tests.md`](quality/acceptance-tests.md)
- [`quality/regulatory-and-enforcement-acceptance-tests.md`](quality/regulatory-and-enforcement-acceptance-tests.md)

Every advertised capability must map to:

```text
Use-case ID
→ product specification
→ actor and authority contract
→ UX flow
→ architecture or ADR
→ implementation phase
→ acceptance test
```

A capability without this chain is not implementation-ready.

## Required ADRs

Before implementation, ADRs must cover at minimum:

- modular core and split criteria;
- tenant and legal-entity isolation;
- identity, purpose, authorization, and inference resistance;
- role, responsibility, authority, delegation, and escalation resolution;
- workflow runtime, timers, concurrency, save/resume, and idempotency;
- invitation and magic-link exchange;
- protected reporting and authority-case isolation;
- Source Registry and Observation contract;
- temporal/versioning strategy;
- evidence storage, integrity, retention, and legal hold;
- relational, object, search, graph, vector, and audit stores;
- import, media, and untrusted-content processing;
- model gateway and external automation adapters;
- offline capture;
- performance budgets, workload profiles, partitioning, and caching;
- deployment modes, backup, recovery, and residency;
- design tokens, responsiveness, accessibility, and report rendering.

## Canonical precedence

When requirements conflict:

1. safety, confidentiality, legal boundaries, tenant isolation, and purpose limitation;
2. [`../README.md`](../README.md) for product intent;
3. [`product/use-case-catalogue.md`](product/use-case-catalogue.md) for customer scope and release maturity;
4. [`product/continuous-compliance-operating-model.md`](product/continuous-compliance-operating-model.md) for Program and Matter behavior;
5. [`product/authority-routing-and-escalation.md`](product/authority-routing-and-escalation.md) and [`product/respond-and-capture.md`](product/respond-and-capture.md) for actor routing and collection behavior;
6. [`product/ease-of-use-standard.md`](product/ease-of-use-standard.md) and [`product/operating-model.md`](product/operating-model.md);
7. specialized product specifications;
8. experience and visual-language documents;
9. [`../AGENTS.md`](../AGENTS.md);
10. architecture documents;
11. implementation sequencing;
12. acceptance tests.

A material change must update the relevant product specification, architecture or ADR, implementation phase, acceptance test, and traceability row.

Do not silently change product semantics, role routing, escalation, invitation security, evidence behavior, automation authority, performance budgets, or UI behavior in code.
