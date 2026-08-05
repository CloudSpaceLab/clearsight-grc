# ClearSight Documentation Map

The documentation is layered so product semantics, safety, architecture and implementation remain distinct and concise.

## Required reading

1. [`../README.md`](../README.md) — product promise and current implementation.
2. [`../DESIGN.md`](../DESIGN.md) — fast interface contract, working language, states and visual proof.
3. [`product/use-case-catalogue.md`](product/use-case-catalogue.md) — target customers and complete use-case contract.
4. [`product/continuous-compliance-operating-model.md`](product/continuous-compliance-operating-model.md) — Programs, Matters, evidence-backed state and closure.
5. [`product/continuous-compliance-and-autonomy.md`](product/continuous-compliance-and-autonomy.md) — Signals, drift, evidence aging, readiness, precedent and governed automation.
6. [`product/authority-routing-and-escalation.md`](product/authority-routing-and-escalation.md) — responsibility, review, authority, delegation and escalation.
7. [`architecture/governance-runtime.md`](architecture/governance-runtime.md) — maker-checker policy lifecycle, delegation, timers and durable delivery.
8. [`architecture/command-integrity-and-projection-operations.md`](architecture/command-integrity-and-projection-operations.md) — verified actors, authority checks, transaction boundaries and Program status operations.
9. [`architecture/source-evidence-and-secure-capture.md`](architecture/source-evidence-and-secure-capture.md) — source health, persisted requests, magic links and artifact integrity.
10. [`product/respond-and-capture.md`](product/respond-and-capture.md) — request-scoped internal/external capture.
11. [`product/illustration-and-guided-experience.md`](product/illustration-and-guided-experience.md) — premium illustrations, empty states and role-specific onboarding.
12. [`product/enterprise-copy-and-content-design.md`](product/enterprise-copy-and-content-design.md) and [`product/plain-language-content-standard.md`](product/plain-language-content-standard.md) — human working language, count integrity and content acceptance.
13. [`design/ui-delivery-workflow.md`](design/ui-delivery-workflow.md) — decision briefs, baselines, state galleries, rendered review and drift control.
14. [`product/ease-of-use-standard.md`](product/ease-of-use-standard.md) — active-effort and minimum-question standards.
15. [`product/operating-model.md`](product/operating-model.md) — canonical domain objects.
16. [`product/experience-principles.md`](product/experience-principles.md) and [`product/ux-and-visual-language.md`](product/ux-and-visual-language.md) — experience and visual system.
17. [`architecture/application-architecture.md`](architecture/application-architecture.md) — executable application boundaries.
18. [`architecture/system-data-and-performance.md`](architecture/system-data-and-performance.md) and [`architecture/data-model-and-storage.md`](architecture/data-model-and-storage.md) — scale, consistency and storage.
19. [`../AGENTS.md`](../AGENTS.md) — mandatory implementation rules.
20. [`implementation-plan.md`](implementation-plan.md) — delivery status and next work.
21. [`quality/release-gates-and-traceability.md`](quality/release-gates-and-traceability.md), [`quality/rendered-ui-evidence.md`](quality/rendered-ui-evidence.md) and domain acceptance tests.

## Canonical precedence

When requirements conflict:

1. safety, confidentiality, legal boundaries and tenant isolation;
2. root README product intent;
3. continuous-compliance, authority, source, capture and guided-experience product specifications;
4. canonical operating-model semantics;
5. interface contract, experience, copy and visual-language standards;
6. AGENTS implementation rules;
7. architecture and ADRs;
8. implementation sequencing;
9. acceptance detail.

Architecture never overrides the simpler user-facing Program, issue/change, request, decision and outcome model.

## Current executable modules

- verified request identity and material-command authority checks;
- authority routing, simulation, integrity and policy resolution;
- maker-checker routing-policy and delegation administration;
- durable workflow tasks, timers, outbox and inbox foundations;
- Source Registry, source observations and freshness maintenance;
- persisted evidence requests, submissions, invitations and sessions;
- streamed development artifact storage and integrity manifests;
- ongoing Programs with requirements, controls, evidence checks and calculated status;
- Program status update queue, lag health, reconcile and governed rebuild;
- typed Matters for changes, findings, exceptions, requests, actions, responses and outcome checks;
- point-in-time Program and Matter reconstruction;
- onboarding guide and user state;
- compliance Signal ingestion, drift and readiness;
- Today, Programs, Work and Configure experiences.

## Traceability

Every advertised capability maps through:

```text
Use-case ID
→ product specification
→ actor and authority contract
→ state/closure contract
→ UX decision brief
→ architecture or ADR
→ implementation phase
→ rendered and behavioral acceptance evidence
```

A feature without this chain is not implementation-ready.
