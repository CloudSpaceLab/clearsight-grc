# #462 — Actor and report operational acceptance

## Scope
Follow the merged reference receipt gate in PR #494. This tranche tests existing Program, Form, Issue and Report interfaces with four independently authenticated principals; it must not add a workflow, scoring engine, framework catalog, component library or demo tenant.

## Preconditions
- Isolated non-production deployment, seeded with explicitly permitted test data; never copy private OpsRisk source into Git.
- Record deployed commit, tenant/legal-entity/group, Program/Form/Issue identifiers, API versions and source high-water.
- Four distinct principals: administrator, contributor, reviewer, authorizer. Record each principal's verified role and authority route; do not impersonate by sending actor IDs in request bodies.
- Confirm the existing report template and source population before asserting report completeness.

## Execution matrix

| Step | Actor | Expected evidence | Negative test |
| --- | --- | --- | --- |
| Configure Program and assign existing Form | Administrator | Persisted Program revision, assignment, audit event | Contributor cannot configure |
| Submit Form | Contributor | Submission revision, linked Program, answers/evidence lineage | Unassigned principal cannot submit |
| Review submission and evidence | Reviewer | Independent assessment, reviewed revision and event | Contributor cannot approve own work |
| Resolve or escalate Issue | Authorized workflow actor | Issue state transition, linked source and version | Unrouted actor cannot transition |
| Decide routed authorization | Authorizer | Effective route, decision event and actor identity | Reviewer without authority cannot authorize |
| Generate and download Report | Permitted report actor | Report run ID, scope, template revision, generated timestamp, source snapshot and file digest | Cross-tenant, cross-entity and unentitled actors cannot view or download |

## Report parity
For each report section, capture source endpoint/query, scope, filter, time semantics, source revision/high-water, included record IDs and expected aggregate. Compare report rows and totals against the same authorized snapshot, including missing/unknown, stale, rejected and pending states. Verify source changes after the snapshot do not silently rewrite the issued report. Download bytes must match the run's digest, and report metadata, listing and download endpoints must enforce the same permissions. Never assert a compliance percentage without #433's approved policy, denominator and coverage.

## UI acceptance
Exercise the existing routed Program, Form, Issue and Report screens for each principal at 1440, 1024, 768 and 390 px. Check disabled/hidden actions against server enforcement, successful navigation and persisted state after reload, accessible names, keyboard/focus, errors, pending/empty states and reduced motion. Fix defects in existing shared components only.

## Release gate
Attach redacted request/response receipts, screenshots, source-to-report comparison and exact-head CI results to the implementation PR. Separate passing deterministic seed tests from hosted actor and report proof. Do not mark T4/T6 complete until all negative authorization tests and report parity pass. #433 and #437 retain separate ownership.
