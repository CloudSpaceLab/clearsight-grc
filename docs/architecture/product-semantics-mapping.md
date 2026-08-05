# Product Semantics to Implementation Mapping

Architecture explains implementation; Programs and Matters define how users operate ClearSight.

| Product concept | Application ownership | Persistence/projection boundary |
|---|---|---|
| Program | Programs | authoritative aggregate plus current-state projections |
| Matter | Matters/Workflow | typed aggregate, workflow, assignments and timeline |
| Scope | Institution and Scope | versioned relational entities/relationships |
| Responsibility/Authority | Authority Routing | assignments, grants, policies and candidate index |
| Authority Source/Requirement | Regulatory and Programs | immutable source metadata, provisions and versioned Requirements |
| Control Objective/Implementation | Controls/Programs | objective and separately scoped implementations |
| Claim/Evidence Contract | Evidence | relational policy and conclusion |
| Observation | Source/Evidence | normalized metadata; artifact may live in object storage |
| Evidence Item | Evidence | immutable object version plus relational lineage/classification |
| Conclusion/Compliance State | Evidence/Programs | versioned conclusion and rebuildable current projection |
| Request/Invitation | Capture | request aggregate, hashed grant and bounded session |
| Decision | Decisions | authoritative versioned options/authority/conditions |
| Action | Actions | execution reference; not verified outcome |
| Verification Contract | Verification | outcome, source, threshold, period and authority |
| Response Package | Authority/Reporting | manifest, artifacts, approval and acknowledgement |
| Today/Work | Projection | actor-authorized read views with freshness |

## Current scaffold

- `internal/authority` — deterministic policy-resolution prototype.
- `internal/capture` — focused request, validation, conflict and invitation exchange.
- `internal/today` — attention projection prototype.
- `internal/httpapi` — transport only.

These are executable foundations, not generic final domain models. PostgreSQL repositories and remaining modules arrive through vertical slices.

## Boundary rules

- A source row/artifact does not become an Authority Source, fact, Requirement, or sufficient evidence without its governed transition.
- AI proposals remain distinct from approved records.
- Projections mutate state only through domain commands.
- External success becomes execution evidence and may require verification.
- User navigation does not expose internal graph, evidence-engine, ledger, operator, or storage terminology as mandatory concepts.
