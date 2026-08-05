# ClearSight Documentation Map

This directory contains the canonical product, architecture, engineering, quality, operations, and review specifications.

## Precedence

When requirements conflict:

1. safety, confidentiality, legal boundaries, tenant isolation, and purpose limitation;
2. [`../README.md`](../README.md) for product intent;
3. product specifications under [`product/`](product/);
4. [`architecture/application-architecture.md`](architecture/application-architecture.md) and accepted ADRs for implementation structure;
5. deeper component architecture documents;
6. [`../AGENTS.md`](../AGENTS.md) for contribution rules;
7. implementation sequencing and acceptance detail.

A component document must not override Programs/Matters, authority routing, request-scoped invitations, evidence boundaries, or usability rules.

## Required reading

1. [`../README.md`](../README.md)
2. [`product/use-case-catalogue.md`](product/use-case-catalogue.md)
3. [`product/continuous-compliance-operating-model.md`](product/continuous-compliance-operating-model.md)
4. [`product/authority-routing-and-escalation.md`](product/authority-routing-and-escalation.md)
5. [`product/respond-and-capture.md`](product/respond-and-capture.md)
6. [`product/ease-of-use-standard.md`](product/ease-of-use-standard.md)
7. [`product/operating-model.md`](product/operating-model.md)
8. [`architecture/application-architecture.md`](architecture/application-architecture.md)
9. [`architecture/data-model-and-storage.md`](architecture/data-model-and-storage.md)
10. [`architecture/system-data-and-performance.md`](architecture/system-data-and-performance.md)
11. [`engineering/development-standards.md`](engineering/development-standards.md)
12. [`implementation-plan.md`](implementation-plan.md)
13. [`quality/release-gates-and-traceability.md`](quality/release-gates-and-traceability.md)
14. [`operations/slo-and-capacity.md`](operations/slo-and-capacity.md)
15. [`../AGENTS.md`](../AGENTS.md)

## Product

- `use-case-catalogue.md` — target customers, personas, maturity, and use-case contract.
- `continuous-compliance-operating-model.md` — Programs, Matters, triggers, state, and closure.
- `authority-routing-and-escalation.md` — responsibility, review, challenge, authorization, delegation, and escalation.
- `respond-and-capture.md` — focused request wizards, invitations, field capture, and protected reporting.
- `ease-of-use-standard.md` — effort budgets, prefill, minimum-question requests, and review by exception.
- `operating-model.md` — canonical domain concepts.
- `experience-principles.md` and `ux-and-visual-language.md` — interaction and visual behavior.
- `regulatory-and-enforcement-intelligence.md` — regulatory, supervisory, and authority work.

## Architecture

- `application-architecture.md` — executable structure, module boundaries, request paths, code layout, and evolution rules.
- `data-model-and-storage.md` — authoritative records, temporal model, indexes, partitioning, projections, and retention.
- `system-data-and-performance.md` — workload, tenancy, performance, resilience, and deployment requirements.
- `continuous-compliance-architecture.md` — how Programs, Matters, routing, capture, evidence, decisions, and verification compose.
- `product-semantics-mapping.md` — product concepts to packages and persistence.
- `living-evidence-fabric.md`, `risk-graph-and-decision-engine.md`, and `governed-ai-operators.md` — supporting component mechanisms.
- [`decisions/`](architecture/decisions/) — accepted ADRs.

## Engineering and operations

- [`engineering/development-standards.md`](engineering/development-standards.md)
- [`quality/performance-test-plan.md`](quality/performance-test-plan.md)
- [`operations/slo-and-capacity.md`](operations/slo-and-capacity.md)
- [`../api/openapi.yaml`](../api/openapi.yaml)
- [`../migrations/`](../migrations/)

## Traceability rule

Every advertised capability must map through:

```text
Use-case ID
→ product contract
→ actor and authority contract
→ state lifecycle
→ UX flow
→ architecture or ADR
→ implementation phase
→ tests and SLO evidence
```

A capability without this chain is not implementation-ready.
