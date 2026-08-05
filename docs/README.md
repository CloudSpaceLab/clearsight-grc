# ClearSight Documentation Map

The documentation is layered so product semantics, safety, architecture and implementation remain distinct and concise.

## Required reading

1. [`../README.md`](../README.md) — product promise and current implementation.
2. [`product/use-case-catalogue.md`](product/use-case-catalogue.md) — target customers and complete use-case contract.
3. [`product/continuous-compliance-operating-model.md`](product/continuous-compliance-operating-model.md) — Programs, Matters, evidence-backed state and closure.
4. [`product/continuous-compliance-and-autonomy.md`](product/continuous-compliance-and-autonomy.md) — Signals, drift, evidence aging, readiness, precedent and governed automation.
5. [`product/authority-routing-and-escalation.md`](product/authority-routing-and-escalation.md) — responsibility, review, authority, delegation and escalation.
6. [`product/respond-and-capture.md`](product/respond-and-capture.md) — request-scoped internal/external capture.
7. [`product/illustration-and-guided-experience.md`](product/illustration-and-guided-experience.md) — premium illustrations, empty states and role-specific onboarding.
8. [`product/ease-of-use-standard.md`](product/ease-of-use-standard.md) — active-effort and minimum-question standards.
9. [`product/operating-model.md`](product/operating-model.md) — canonical domain objects.
10. [`product/experience-principles.md`](product/experience-principles.md) and [`product/ux-and-visual-language.md`](product/ux-and-visual-language.md) — experience and visual system.
11. [`architecture/application-architecture.md`](architecture/application-architecture.md) — executable application boundaries.
12. [`architecture/system-data-and-performance.md`](architecture/system-data-and-performance.md) and [`architecture/data-model-and-storage.md`](architecture/data-model-and-storage.md) — scale, consistency and storage.
13. [`../AGENTS.md`](../AGENTS.md) — mandatory implementation rules.
14. [`implementation-plan.md`](implementation-plan.md) — delivery status and next work.
15. [`quality/release-gates-and-traceability.md`](quality/release-gates-and-traceability.md) and domain acceptance tests.

## Canonical precedence

When requirements conflict:

1. safety, confidentiality, legal boundaries and tenant isolation;
2. root README product intent;
3. continuous-compliance, authority, capture and guided-experience product specifications;
4. canonical operating-model semantics;
5. experience and visual-language standards;
6. AGENTS implementation rules;
7. architecture and ADRs;
8. implementation sequencing;
9. acceptance detail.

Architecture never overrides the simpler user-facing Program, Matter, request, decision and outcome model.

## Traceability

Every advertised capability maps through:

```text
Use-case ID
→ product specification
→ actor and authority contract
→ state/closure contract
→ UX reference
→ architecture or ADR
→ implementation phase
→ acceptance evidence
```

A feature without this chain is not implementation-ready.

## Current executable modules

- authority routing, simulation, policy listing and integrity;
- workflow tasks and optimistic transitions;
- onboarding guide and user state;
- compliance signal ingestion, drift and readiness;
- focused requests and invitation exchange;
- Today brief and responsive Configure experience.

## Reviews

Major review documents under [`reviews/`](reviews/) record what was inspected, changed and remains. Reviews are historical evidence; superseded reviews must say so explicitly.
