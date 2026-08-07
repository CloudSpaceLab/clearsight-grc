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
8. [`architecture/command-integrity-and-projection-operations.md`](architecture/command-integrity-and-projection-operations.md) — executable route classes, verified actor binding, authority checks, transaction truth and Program-status operations.
9. [`architecture/source-evidence-and-secure-capture.md`](architecture/source-evidence-and-secure-capture.md) — source health, persisted requests, bounded capture capabilities and artifact integrity.
10. [`product/nigerian-bank-reference-journeys.md`](product/nigerian-bank-reference-journeys.md) — connected, actionable Nigerian-bank reference journeys.
11. [`product/respond-and-capture.md`](product/respond-and-capture.md) — request-scoped internal/external capture.
12. [`product/illustration-and-guided-experience.md`](product/illustration-and-guided-experience.md) — illustrations, empty states and role-specific onboarding.
13. [`product/enterprise-copy-and-content-design.md`](product/enterprise-copy-and-content-design.md) and [`product/plain-language-content-standard.md`](product/plain-language-content-standard.md) — human working language, count integrity and content acceptance.
14. [`design/ui-delivery-workflow.md`](design/ui-delivery-workflow.md) — decision briefs, baselines, rendered review and drift control.
15. [`product/ease-of-use-standard.md`](product/ease-of-use-standard.md) — active-effort and minimum-question standards.
16. [`product/operating-model.md`](product/operating-model.md) — canonical domain objects.
17. [`product/experience-principles.md`](product/experience-principles.md) and [`product/ux-and-visual-language.md`](product/ux-and-visual-language.md) — experience and visual system.
18. [`architecture/application-architecture.md`](architecture/application-architecture.md) — executable application boundaries.
19. [`architecture/system-data-and-performance.md`](architecture/system-data-and-performance.md) and [`architecture/data-model-and-storage.md`](architecture/data-model-and-storage.md) — scale, consistency and storage.
20. [`../AGENTS.md`](../AGENTS.md) — mandatory implementation rules.
21. [`implementation-plan.md`](implementation-plan.md) — **authoritative current execution ledger and sequencing**.
22. [`design/enterprise-productization-design-plan.md`](design/enterprise-productization-design-plan.md) — finished enterprise experience reference.
23. [`engineering/enterprise-productization-implementation-plan.md`](engineering/enterprise-productization-implementation-plan.md) — detailed enterprise work/reference phases; current execution order is controlled by `implementation-plan.md`.
24. [`quality/release-gates-and-traceability.md`](quality/release-gates-and-traceability.md), [`quality/rendered-ui-evidence.md`](quality/rendered-ui-evidence.md) and domain acceptance tests.

## Canonical precedence

When requirements conflict:

1. safety, confidentiality, legal boundaries and tenant isolation;
2. root README product intent;
3. continuous-compliance, authority, source and capture product specifications;
4. canonical operating-model semantics;
5. interface, experience, copy and visual-language standards;
6. AGENTS implementation rules;
7. architecture and ADRs;
8. **the current execution order in `implementation-plan.md`;**
9. enterprise productization design/implementation references;
10. acceptance detail.

Architecture never overrides the simpler user-facing Program, issue/change, request, decision and outcome model.

## Current executable modules

- verified request identity with tenant, principal and legal-entity scope;
- one typed HTTP route registry classifying public, authenticated, material-command and bounded-capability routes;
- server-bound tenant/actor fields for protected writes and explicit material-command authority policies;
- fail-closed restricted-record policy parsing and pre-pagination Matter visibility;
- authority routing, simulation, integrity and policy resolution;
- maker-checker routing-policy and delegation administration;
- durable Workflow Tasks, timers, outbox and inbox foundations;
- Source Registry, source observations and freshness maintenance;
- durable source-health reconciliation from evidence outbox events into exact source drift and dependent Program triggers, with inbox dedupe;
- persisted evidence requests, submissions, invitations and sessions;
- linked-request visibility derived from the subject Matter before PostgreSQL limits;
- streamed development artifact storage and integrity manifests;
- ongoing Programs with requirements, controls, evidence checks and calculated status;
- Program-status update queue, lag health, reconciliation and governed rebuild;
- typed Matters for changes, findings, exceptions, requests, actions, responses and outcome checks;
- point-in-time Program and Matter reconstruction;
- role-aware onboarding and per-user guide state;
- compliance Signal ingestion, drift and readiness;
- non-demo Today projection from active Workflow Tasks assigned to the verified principal;
- intervention-first Today and progressively disclosed Programs, Work/Evidence and Imports;
- automation-policy read visibility in Configure;
- exact record launchers to linked Programs, Matters and evidence requests;
- recoverable opt-in Nigerian-bank reference installation for non-production environments.

## Current execution status

The repository is a strong working foundation and reference MVP. It is not yet a completed banking product.

Issues #26 and #27 now share one execution path rather than parallel backlogs:

1. #26 route/identity seam — implemented in PR #25;
2. #26 source-health outbox/inbox reconciliation — implemented in PR #25;
3. #26 worker work-class isolation — next;
4. #26 compound-command transaction truth;
5. #26 API/OpenAPI/client contract reconciliation;
6. #26 governance/configuration authorization matrix;
7. #27 governed operator execution, receipts and intervention mutations.

The remaining enterprise identity, RBAC, notifications, MFA, production storage, accessibility, recovery and scale work remains specified in the enterprise productization documents, but those documents do not supersede this seam-first order.

## Semantic guardrails

Do not collapse these objects while implementing the sequence above:

- Matter Action = canonical remediation/commitment state.
- Workflow Task = actor-facing routed/manual work.
- Signal = observation, not incident/conclusion.
- Evidence submission = captured input, not evidence sufficiency.
- Automation Policy = permission boundary, not execution proof.
- Intervention Summary = actor-facing read projection, not authoritative state.

## Traceability

Every advertised capability maps through:

```text
Use-case ID
→ product specification
→ actor and authority contract
→ state/closure contract
→ UX decision brief
→ architecture or ADR
→ current implementation ledger
→ rendered and behavioral acceptance evidence
```

A feature without this chain is not implementation-ready.
