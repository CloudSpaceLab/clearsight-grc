# Enterprise copy and content design

ClearSight copy is operational interface content. It affects decisions, authority, evidence quality and user action, so it is reviewed as product behavior rather than marketing polish.

## Standard

Every page should answer, in plain language:

1. What object or population is shown?
2. What state is it in?
3. Why is it here now?
4. Who owns the next action?
5. What is the next valid action?
6. What source and timestamp support the statement?

Use bank-operating terms that match the domain model: Program, Matter, review, approval, evidence request, control, obligation, exception, delegation, escalation and verification.

## Avoid

Do not place product claims inside working screens. Avoid copy such as:

- “Only the work that needs your judgment.”
- “Everything material is handled.”
- “Continuously prepared.”
- “Automatically maintained.”
- “Watch drift, not dashboards.”

These phrases are vague, promotional and can overstate population completeness.

## Prefer

| Avoid | Prefer |
|---|---|
| Needs attention | Open items |
| Your attention | Assigned to you |
| Automatically maintained | Current items |
| No intervention required | No action due |
| Inspect authority | View approval route |
| Open capture wizard | Request evidence |
| Everything material is handled | No assigned items |
| Autonomous configuration checks | Configuration checks |

A label may be more specific when the page has a narrower population, for example “4 approvals awaiting CRO review” or “2 vendor certificates expire within 30 days.”

## Counts and status

A count must have a defined denominator, scope and freshness source. When a governed baseline is unavailable, display an em dash or “Population not connected.” Never substitute a demo number.

Empty states are scoped statements. “No open approvals assigned to you” is valid; “No compliance issues” is not unless the entire governed population and freshness basis are visible.

## Guidance, illustrations and icons

First-run guides are brief, role-specific and tied to a real task. Titles describe the action: “Review assigned work,” “Check the approval route,” “Request additional evidence.”

Use premium vector illustrations for orientation, empty states and education. Use semantic SVG icons for recurring work types. Neither may carry status or replace text.

## Acceptance checks

Before release:

- a domain owner confirms terminology;
- an intended user can state the object, state and next action without explanation;
- every count has scope and freshness;
- unknown data is explicit;
- sample data is labelled;
- no copy suggests autonomous approval, verified outcome or enterprise completeness without evidence;
- screen-reader labels carry the same meaning as visible content.
