# Program and Matter acceptance tests

## Program lifecycle

- A new Program is “Setup in progress,” not silently active.
- Activation requires an accountable owner, approval authority, approved requirement, actor and rationale.
- A complete applicable requirement, implemented control and current supporting evidence can produce “Up to date.”
- Missing control coverage produces “Gap found.”
- Missing, stale or contradictory evidence produces a specific reason and “Evidence incomplete” or “Gap found.”
- A duplicate trigger does not create a second Matter.
- A linked open Matter changes the Program to “Needs attention.”
- A historical read excludes later requirements, triggers and Matters.

## Program operating record

- The exact Program route loads the aggregate, calculated-state freshness, review digest and actor-scoped operations together.
- Authorized owners can edit scope and dates, choose an eligible successor, add source-anchored requirements and supersede a current requirement without overwriting its prior version.
- Applicability decisions are available only to the current authorizer and retain scope, rationale and effective date.
- Safeguard setup separates the control objective, implementation and eligible performer, then links only Program-owned requirements and implementations.
- Evidence checks select named current sources and record freshness, population coverage, independence, contradiction and failure rules.
- Evidence assessment remains distinct from monitoring results and records conclusion, coverage, basis, references, assessor time and validity.
- Operating lifecycle status remains distinct from calculated compliance state and requires a current authorizer, valid target, optimistic version and rationale.
- Linked issue reads use an exact Program filter before pagination; creating linked work preselects the Program and opens the created issue record.
- Every Program material command maps to a tested user surface or an explicitly automation-only contract.

## Matter lifecycle

- Unsupported state transitions fail.
- Closing or reopening requires an actor and rationale.
- Open actions block closure.
- Implemented work without a passing outcome check blocks closure when the Matter type requires it.
- A failed outcome check follows the configured failure response.
- A closed Matter can be reopened with preserved closure history and incremented reopen count.
- An external response that has been transmitted prevents cancellation.

## Integrity

- Cross-tenant Program, requirement, control, Matter, decision, action and outcome-check links fail at the database boundary.
- Invalid acceptable source IDs fail transactionally.
- Aggregate version conflicts do not partially update projections, events or outbox records.
- Each material event produces one append-only event and one outbox record.

## Content

- UI states use human labels, not enum codes.
- “Matter” is not required knowledge on general business screens; the specific type or “Issues and changes” is shown.
- Closure blockers explain what is missing in ordinary language.
- Counts state their scope and do not fabricate unavailable populations.
- Permission-limited Program views name the current responsible person and do not expose an enabled control that cannot execute.
