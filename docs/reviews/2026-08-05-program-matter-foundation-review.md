# Program and Matter foundation review — 2026-08-05

## Scope reviewed

- Program, Requirement, applicability, control and evidence-check aggregates;
- calculated Program status;
- trigger deduplication and linked Matter creation;
- typed Matter lifecycle, decisions, actions, response packages and outcome checks;
- closure and reopening;
- PostgreSQL tenant integrity, optimistic versions, event history and outbox;
- Programs and Issues/Changes user surfaces;
- plain-language copy.

## Findings resolved in this phase

1. Programs now begin in **Setup in progress** and cannot become active without an accountable owner, authority, rationale and at least one approved requirement.
2. Program status is calculated from recorded facts and includes reasons; it is not a manually selected colour.
3. Evidence checks validate approved sources, freshness, coverage, independence and contradiction handling.
4. A repeated trigger cannot create duplicate open work for the same Program and dedupe key.
5. Action implementation remains separate from a confirmed outcome.
6. Closure is typed. Findings, gaps, regulatory changes, exceptions and authority responses have different completion requirements.
7. Failed outcome checks follow an explicit response: reopen, request a decision, create follow-up work or block closure.
8. Programs and Matters can be reconstructed from append-only events at a requested timestamp.
9. Composite tenant foreign keys prevent cross-tenant Program, requirement, control, source and Matter relationships.
10. Primary UI language uses human working terms while APIs and audit history retain precise codes.
11. The Programs/Issues surfaces now have a compact design brief, responsive replacement contract and rendered-evidence acceptance gate.
12. The root `DESIGN.md` provides an agent-readable implementation contract so future screens do not reconstruct the visual system from scattered CSS and prose.

## Copy review

Primary-screen replacements include:

| Specialist/internal wording | Primary-screen wording |
|---|---|
| applicability determination | Does this apply? |
| evidence insufficiency | Evidence incomplete |
| verification contract | Outcome check |
| verification stage | Confirming outcome |
| Matter queue | Issues and changes |
| implementation pending | Change in progress |
| control implementation implemented | Completed; outcome not yet confirmed |

The word **Matter** remains available in specialist detail, API documentation and audit history because it is the stable aggregate name.

## Acceptance evidence

The phase includes:

- default in-memory domain tests;
- optimistic-version conflict tests;
- idempotent-trigger tests;
- closure-block and reopening tests;
- point-in-time reconstruction tests;
- HTTP copy and error-contract tests;
- PostgreSQL integration covering state calculation, durable events, trigger deduplication, typed closure and direct cross-tenant relationship rejection;
- TypeScript type-check and production web build through CI.

## Remaining boundaries

- caller-supplied actor identity is not yet bound to authenticated identity;
- authority routing is not invoked automatically for every material command;
- list reads replay bounded event streams and need projection-first performance work before high-cardinality production use;
- Program template publication and bulk setup are not implemented;
- shared-control dependency propagation remains future work;
- the initial vertical workflows remain the next implementation phase.

## Design-process review

Two external agent-design approaches were reviewed for process ideas. The useful additions were tool-neutral: a persistent standalone design contract, before-state baselines for redesigns, selective branching for high-impact surfaces, compact decision briefs, state galleries, rendered evidence, responsive replacement and one highest-impact repair before acceptance. ClearSight does not adopt an external runtime or generic style library; bank workflow semantics, production components, accessibility and repository tokens remain authoritative.
