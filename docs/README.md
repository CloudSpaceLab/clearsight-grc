# ClearSight Documentation Map

The documentation is layered so product semantics, safety, architecture and implementation remain distinct and concise.

## Required reading

1. [`../README.md`](../README.md) — product promise and current implementation.
2. [`product/use-case-catalogue.md`](product/use-case-catalogue.md) — target customers and complete use-case contract.
3. [`product/continuous-compliance-operating-model.md`](product/continuous-compliance-operating-model.md) — Programs, Matters, evidence-backed state and closure.
4. [`product/continuous-compliance-and-autonomy.md`](product/continuous-compliance-and-autonomy.md) — Signals, drift, evidence aging, readiness, precedent and governed automation.
5. [`product/authority-routing-and-escalation.md`](product/authority-routing-and-escalation.md) — responsibility, review, authority, delegation and escalation.
6. [`architecture/governance-runtime.md`](architecture/governance-runtime.md) — maker-checker policy lifecycle, delegation, timers and durable delivery.
7. [`architecture/source-evidence-and-secure-capture.md`](architecture/source-evidence-and-secure-capture.md) — source health, persisted requests, magic links and artifact integrity.
8. [`product/respond-and-capture.md`](product/respond-and-capture.md) — request-scoped internal/external capture.
9. [`product/illustration-and-guided-experience.md`](product/illustration-and-guided-experience.md) — premium illustrations, empty states and role-specific onboarding.
10. [`product/enterprise-copy-and-content-design.md`](product/enterprise-copy-and-content-design.md) — realistic operational wording, count integrity and content acceptance.
11. [`product/ease-of-use-standard.md`](product/ease-of-use-standard.md) — active-effort and minimum-question standards.
12. [`product/operating-model.md`](product/operating-model.md) — canonical domain objects.
13. [`product/experience-principles.md`](product/experience-principles.md) and [`product/ux-and-visual-language.md`](product/ux-and-visual-language.md) — experience and visual system.
14. [`architecture/application-architecture.md`](architecture/application-architecture.md) — executable application boundaries.
15. [`architecture/system-data-and-performance.md`](architecture/system-data-and-performance.md) and [`architecture/data-model-and-storage.md`](architecture/data-model-and-storage.md) — scale, consistency and storage.
16. [`../AGENTS.md`](../AGENTS.md) — mandatory implementation rules.
17. [`implementation-plan.md`](implementation-plan.md) — delivery status and next work.
18. [`quality/release-gates-and-traceability.md`](quality/release-gates-and-traceability.md) and domain acceptance tests.

## Canonical precedence

When requirements conflict:

1. safety, confidentiality, legal boundaries and tenant isolation;
2. root README product intent;
3. continuous-compliance, authority, source, capture and guided-experience product specifications;
4. canonical operating-model semantics;
5. experience, copy and visual-language standards;
6. AGENTS implementation rules;
7. architecture and ADRs;
8. implementation sequencing;
9. acceptance detail.

Architecture never overrides the simpler user-facing Program, Matter, request, decision and outcome model.

## Current executable modules

- authority routing, simulation, integrity and policy resolution;
- maker-checker routing-policy and delegation administration;
- durable workflow tasks, timers, outbox and inbox foundations;
- Source Registry, source observations and freshness maintenance;
- persisted evidence requests, submissions, invitations and sessions;
- streamed development artifact storage and integrity manifests;
- onboarding guide and user state;
- compliance Signal ingestion, drift and readiness;
- Today, Work and Configure experiences.

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
