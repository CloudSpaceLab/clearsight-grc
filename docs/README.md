# ClearSight Documentation Map

The documentation is layered so product semantics, safety, architecture, experience and implementation remain distinct.

## Required reading

1. [`../README.md`](../README.md) — product promise, executable scope and boundaries.
2. [`../DESIGN.md`](../DESIGN.md) — interface contract, working language, states and visual proof.
3. [`product/use-case-catalogue.md`](product/use-case-catalogue.md) — target customers and complete use-case contract.
4. [`product/continuous-compliance-operating-model.md`](product/continuous-compliance-operating-model.md) — Programs, issues/changes, evidence-backed state and closure.
5. [`product/continuous-compliance-and-autonomy.md`](product/continuous-compliance-and-autonomy.md) — Signals, drift, evidence aging, readiness, precedent and governed automation.
6. [`product/authority-routing-and-escalation.md`](product/authority-routing-and-escalation.md) — responsibility, review, authority, delegation and escalation.
7. [`architecture/governance-runtime.md`](architecture/governance-runtime.md) — maker-checker policy lifecycle, delegation, timers and durable delivery.
8. [`architecture/command-integrity-and-projection-operations.md`](architecture/command-integrity-and-projection-operations.md) — verified actors, authority checks, transaction boundaries and Program status operations.
9. [`architecture/source-evidence-and-secure-capture.md`](architecture/source-evidence-and-secure-capture.md) — source health, persisted requests, magic links and artifact integrity.
10. [`product/nigerian-bank-reference-journeys.md`](product/nigerian-bank-reference-journeys.md) — connected, actionable Nigerian-bank reference journeys.
11. [`product/respond-and-capture.md`](product/respond-and-capture.md) — request-scoped internal/external capture.
12. [`product/illustration-and-guided-experience.md`](product/illustration-and-guided-experience.md) — premium illustrations, empty states and role-specific onboarding.
13. [`product/enterprise-copy-and-content-design.md`](product/enterprise-copy-and-content-design.md) and [`product/plain-language-content-standard.md`](product/plain-language-content-standard.md) — human working language, count integrity and content acceptance.
14. [`design/ui-delivery-workflow.md`](design/ui-delivery-workflow.md) — decision briefs, baselines, state galleries, rendered review and drift control.
15. [`design/enterprise-productization-design-plan.md`](design/enterprise-productization-design-plan.md) — finished enterprise experience across UI cleanup, role onboarding, authority matrices, directory import, notifications, MFA and premium visual systems.
16. [`engineering/enterprise-productization-implementation-plan.md`](engineering/enterprise-productization-implementation-plan.md) — sequenced implementation phases, data/API work, dependencies, migrations, tests, rollout and release gates.
17. [`product/ease-of-use-standard.md`](product/ease-of-use-standard.md) — active-effort and minimum-question standards.
18. [`product/operating-model.md`](product/operating-model.md) — canonical domain objects.
19. [`product/experience-principles.md`](product/experience-principles.md) and [`product/ux-and-visual-language.md`](product/ux-and-visual-language.md) — experience and visual system.
20. [`architecture/application-architecture.md`](architecture/application-architecture.md) — executable application boundaries.
21. [`architecture/system-data-and-performance.md`](architecture/system-data-and-performance.md) and [`architecture/data-model-and-storage.md`](architecture/data-model-and-storage.md) — scale, consistency and storage.
22. [`../AGENTS.md`](../AGENTS.md) — mandatory implementation rules.
23. [`implementation-plan.md`](implementation-plan.md) — repository capability status and the next productization phase.
24. [`quality/release-gates-and-traceability.md`](quality/release-gates-and-traceability.md), [`quality/rendered-ui-evidence.md`](quality/rendered-ui-evidence.md) and domain acceptance tests.

## Canonical precedence

When requirements conflict:

1. safety, confidentiality, legal boundaries and tenant isolation;
2. root README product intent;
3. continuous-compliance, authority, source, capture and guided-experience product specifications;
4. canonical operating-model semantics;
5. interface contract, experience, copy and visual-language standards;
6. enterprise productization design plan;
7. AGENTS implementation rules;
8. architecture and ADRs;
9. enterprise productization implementation sequencing;
10. acceptance detail.

Architecture never overrides the simpler user-facing Program, issue/change, request, decision and outcome model.

## Current executable modules

- verified request identity with tenant, principal and legal-entity scope conflict rejection;
- fail-closed restricted-record policy parsing and pre-pagination Matter visibility;
- authority routing, simulation, integrity and policy resolution;
- maker-checker routing-policy and delegation administration;
- durable workflow tasks, timers, outbox and inbox foundations;
- Source Registry, source observations and freshness maintenance;
- persisted evidence requests, submissions, invitations and sessions;
- linked-request visibility derived from the subject Matter before PostgreSQL limits;
- streamed development artifact storage and integrity manifests;
- ongoing Programs with requirements, controls, evidence checks and calculated status;
- Program status update queue, lag health, reconciliation and governed rebuild;
- typed Matters for changes, findings, exceptions, requests, actions, responses and outcome checks;
- current-record journey evaluation that rejects retired, withdrawn, cancelled, superseded and non-independent evidence;
- point-in-time Program and Matter reconstruction;
- one role-labelled onboarding guide and per-user guide state;
- compliance Signal ingestion, drift and readiness;
- actor-scoped dynamic Today work;
- Today, Programs, Work, Explore and Configure experiences;
- exact journey launchers to linked Programs, issues and evidence requests;
- recoverable opt-in Nigerian-bank reference installation for non-production environments;
- initial premium illustration, empty-state and semantic icon components.

## Enterprise productization status

The current repository is a strong working foundation and reference MVP. It is not yet a completed banking product.

The canonical next phase is defined by:

- [`design/enterprise-productization-design-plan.md`](design/enterprise-productization-design-plan.md);
- [`engineering/enterprise-productization-implementation-plan.md`](engineering/enterprise-productization-implementation-plan.md).

That phase includes:

- complete UI/UX cleanup, light/dark themes and reusable components;
- complete operational write interfaces;
- enterprise OIDC/SAML, SCIM and LDAP/Active Directory compatibility;
- full responsibility, RBAC, authority, segregation and escalation administration;
- multi-role first-time guidance;
- notification centre, email templates and production publishers;
- WebAuthn/passkey, TOTP, recovery and step-up assurance;
- complete illustration, icon and empty-state families;
- security, recovery, accessibility and production-scale release evidence.

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
