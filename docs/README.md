# ClearSight Documentation Map

The documentation is layered so product semantics, safety, architecture, experience and implementation remain distinct.

## Required reading

1. [`../README.md`](../README.md) — product promise, executable scope and boundaries.
2. [`../DESIGN.md`](../DESIGN.md) — interface contract, working language, states and visual proof.
3. [`product/use-case-catalogue.md`](product/use-case-catalogue.md) — target customers and complete use-case contract.
4. [`product/continuous-compliance-operating-model.md`](product/continuous-compliance-operating-model.md) — Programs, issues/changes, evidence-backed state and closure.
5. [`product/continuous-compliance-and-autonomy.md`](product/continuous-compliance-and-autonomy.md) — Signals, drift, evidence aging, readiness, precedent and governed automation.
6. [`product/authority-routing-and-escalation.md`](product/authority-routing-and-escalation.md) — responsibility, review, authority, delegation and escalation.
7. [`architecture/governance-runtime.md`](architecture/governance-runtime.md) — maker-checker policy lifecycle, delegation, isolated worker classes, timers and durable delivery.
8. [`architecture/command-integrity-and-projection-operations.md`](architecture/command-integrity-and-projection-operations.md) — executable route classes, verified actor binding, authority checks, transaction truth and Program-status operations.
9. [`architecture/durable-schema-ownership.md`](architecture/durable-schema-ownership.md) — live durable-table ownership, maturity and retention contract.
10. [`architecture/source-evidence-and-secure-capture.md`](architecture/source-evidence-and-secure-capture.md) — source health, persisted requests, bounded capture capabilities and artifact integrity.
11. [`architecture/connected-source-access.md`](architecture/connected-source-access.md) — reusable Connection/View/Binding contracts, adapter capabilities, bounded source reads and assurance compatibility.
12. [`architecture/ai-gateway-transport.md`](architecture/ai-gateway-transport.md) — isolated OpenAI-compatible transport, provider adapters, routing, budgets, streaming truth and confidentiality boundary.
13. [`product/nigerian-bank-reference-journeys.md`](product/nigerian-bank-reference-journeys.md) — connected, actionable Nigerian-bank reference journeys.
14. [`product/respond-and-capture.md`](product/respond-and-capture.md) — request-scoped internal/external capture.
15. [`product/illustration-and-guided-experience.md`](product/illustration-and-guided-experience.md) — illustrations, empty states and role-specific onboarding.
16. [`product/enterprise-copy-and-content-design.md`](product/enterprise-copy-and-content-design.md) and [`product/plain-language-content-standard.md`](product/plain-language-content-standard.md) — human working language, count integrity and content acceptance.
17. [`design/ui-delivery-workflow.md`](design/ui-delivery-workflow.md) — decision briefs, baselines, rendered review and drift control.
18. [`product/ease-of-use-standard.md`](product/ease-of-use-standard.md) — active-effort and minimum-question standards.
19. [`product/operating-model.md`](product/operating-model.md) — canonical domain objects.
20. [`product/experience-principles.md`](product/experience-principles.md) and [`product/ux-and-visual-language.md`](product/ux-and-visual-language.md) — experience and visual system.
21. [`architecture/application-architecture.md`](architecture/application-architecture.md) — executable application boundaries.
22. [`architecture/system-data-and-performance.md`](architecture/system-data-and-performance.md) and [`architecture/data-model-and-storage.md`](architecture/data-model-and-storage.md) — scale, consistency and storage.
23. [`../AGENTS.md`](../AGENTS.md) — mandatory implementation rules.
24. [`implementation-plan.md`](implementation-plan.md) — **authoritative current execution ledger and sequencing**.
25. [`design/enterprise-productization-design-plan.md`](design/enterprise-productization-design-plan.md) — finished enterprise experience reference.
26. [`engineering/enterprise-productization-implementation-plan.md`](engineering/enterprise-productization-implementation-plan.md) — detailed enterprise work/reference phases; current execution order is controlled by `implementation-plan.md`.
27. [`engineering/enterprise-identity-access.md`](engineering/enterprise-identity-access.md) — focused OSS-first identity, department-aware capabilities and multi-level escalation implementation boundary; supersedes greenfield LDAP/SAML implementation guidance.
28. [`engineering/demo-role-login.md`](engineering/demo-role-login.md) — non-production stakeholder role catalogue, supplied demo credentials, signed demo session and production isolation boundary.
29. [`quality/release-gates-and-traceability.md`](quality/release-gates-and-traceability.md), [`quality/rendered-ui-evidence.md`](quality/rendered-ui-evidence.md) and domain acceptance tests.

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
9. focused enterprise implementation plans, then broader enterprise productization references;
10. acceptance detail.

Architecture never overrides the simpler user-facing Program, issue/change, request, decision and outcome model.

## Current executable modules

- native OIDC enterprise sign-in with server-side sessions plus signed-gateway/development compatibility;
- SCIM Users/Groups provisioning with explicit governed directory-group → existing-role mappings;
- local legal-entity and exact-department capability resolution from current ClearSight state;
- demo-only supplied role credentials and signed role-switching sessions, absent from production route inventory;
- one typed HTTP route registry classifying public, demo-only, authenticated, material-command and bounded-capability routes;
- server-bound tenant/actor fields for protected writes and explicit material-command authority policies;
- fail-closed restricted-record policy parsing and pre-pagination Matter visibility;
- authority routing, simulation, integrity and policy resolution;
- maker-checker routing-policy, escalation-guard revision and delegation administration;
- multi-level `OVERDUE` escalation with department ancestry and optional source-role / target-role-or-group candidate guards;
- Matter Action-driven Workflow Task projection, leased timers, outbox and inbox foundations;
- independent in-process worker classes for evidence maintenance, Program projection, delegation lifecycle, timers and outbox delivery;
- bounded timer/outbox retry budgets with durable terminal failure and queue health rather than infinite poison-item retry;
- Source Registry, source observations and freshness maintenance;
- durable versioned Source Connection/View/Binding catalog with bounded PostgreSQL schema, page, lookup and aggregate capabilities;
- assurance consumption of shared source sessions without a second connector registry, copied source population or gateway-specific bundle format;
- isolated stateless AI gateway transport with strict Chat/Responses ingress, OpenAI and Anthropic adapters, truthful SSE, workload budgets, fallback/circuit controls and content-free telemetry;
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
- Identity & Access and automation-policy administration in Configure;
- exact record launchers to linked Programs, Matters and evidence requests;
- recoverable opt-in Nigerian-bank reference installation for non-production environments;
- machine-checked ownership classification for every live durable PostgreSQL table;
- one mechanically verified executable production runtime route/access contract, with bounded domain schemas kept descriptive.

## Current execution status

The repository is a strong working foundation and reference MVP. It is not yet a completed banking product.

The enterprise identity/access sequence **EIA-0 through EIA-5 is implemented on PR #59**: OIDC, server sessions, SCIM, department-aware capabilities, governed directory-group mappings, executable multi-level `OVERDUE` escalation, role/group escalation guards, maker-checker guard revisions, and the compact Configure → Identity & Access surface. Demo mode additionally exposes a supplied role catalogue on a dedicated login page; those routes and credentials do not exist when demo mode is disabled.

Current execution truth is maintained in [`implementation-plan.md`](implementation-plan.md). Do not infer capability from historical issue text, a durable table name, a descriptive API schema, or an older branch.

Remaining productization is outside generic IAM: non-`OVERDUE` escalation adapters when real domain events exist, broader enterprise operator surfaces where backend capability already exists, production storage/security/recovery/scale evidence, and representative bank-user acceptance.

## Semantic guardrails

Do not collapse these objects while implementing later work:

- Matter Action = canonical remediation/commitment state.
- Workflow Task = actor-facing routed/manual projection, not an independently mutable business record.
- Signal = observation, not incident/conclusion.
- Evidence submission = captured input, not evidence sufficiency.
- Automation Policy = permission boundary, not execution proof.
- Intervention Summary = actor-facing read projection, not authoritative state.
- Durable table = storage construct whose capability meaning depends on its registered executable owner.
- Evidence Source = business authority identity; Source Connection/View/Binding = reusable technical access contracts beneath it.
- Source Binding = purpose-bound read/mapping contract, not copied source data, evidence sufficiency or workflow authority.
- Department path = organizational scope, not authorization by itself.
- Directory group = source-backed membership, not material authority.
- Escalation sequence = ordered responsibility/scope selection, not a hard-coded assignee chain.
- Escalation role/group guard = candidate restriction, not an authority grant.
- Demo role credential = non-production fixture, not an enterprise authentication mechanism.

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

### T2 product reuse

Forms/capture now retain exact field-level Source Binding references for `PREFILL`, `OPTIONS`, `VALIDATE` and `EVIDENCE`. Connected values carry canonical operation receipts and remain visibly distinct from respondent-entered or corrected answers. Evidence requests can search configured bindings before asking a person, while workflow tasks project only the exact binding IDs/versions and continue to treat the request—not the Binding—as domain truth. See [`acceptance/t2-binding-reuse.md`](acceptance/t2-binding-reuse.md).
