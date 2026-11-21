> Historical design note. For the current implementation and limits, read `../DATA_AND_WORKFLOW_REVIEW.md` and `../TEST_REPORT.md`.

> Retained module reference from the preceding prototype. V2.4 workflow changes and actual delivery limits are documented in `../README.md`, `UX_REVIEW.md`, `PRIVACY_MODEL.md` and `TEST_REPORT.md`.

# V2.3 interaction and scope contract

The primary form editor is the respondent-shaped document, not a permanent three-pane schema workspace. Inline fields edit definitions only; response preview and submitted snapshots use the existing form engine. Published versions remain pinned for existing assignments.

The top organizational context is an ordered Branch → Department → Unit selector. Each blank value means All; a selected parent includes descendants. Changing a parent clears lower levels; selecting a deeper instance resolves its exact ancestors. Organizational IDs, not display labels, define identity. Duplicate names across branches are qualified.

A risk has an accountable organizational scope. A loss stores a scope snapshot separately, even when it references a risk. Source records are filtered once and then aggregated. Arrays of referenced risks, controls, programs and cycles must not multiply a source loss. Shared framework/form definitions do not have to belong to a branch; their assessment executions have explicit targets and bindings.

Gross, recovered and net losses use integer minor units. Event occurrence determines period membership; recoveries received through the snapshot apply to those events. Risk inventory remains a current snapshot independent of the loss period. Near misses contribute zero monetary loss and a separate event count. No records is zero; unavailable collection evidence is unknown, a different state.

Only the rendered RCSA population is narrowed by current scope. Its original frozen cycle and independent-review records are not overwritten. Program outcomes retain their original target; filtering the portfolio does not allocate bank-wide conclusions to a branch.

Production must enforce scope and authorization server-side. The demo guards and filtered projections are not security. Neither organizational scope changes nor navigation changes should silently alter a bound assessment target or stored historical loss location.
