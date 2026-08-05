# Enterprise copy and content design

ClearSight copy is operational interface content. It affects decisions, authority, evidence quality and user action, so it is reviewed as product behavior rather than marketing polish.

The detailed translation rules live in [`plain-language-content-standard.md`](plain-language-content-standard.md).

## Standard

Every page should answer, in plain language:

1. What object or population is shown?
2. What state is it in?
3. Why is it here now?
4. Who owns the next action?
5. What is the next valid action?
6. What source and timestamp support the statement?

Use familiar bank-operating terms. Program is a primary product noun. On general business screens, prefer the specific issue type or “Issues and changes” to the abstract noun Matter.

Technical state codes belong in APIs, audit history and specialist detail. Primary UI wording translates them without changing meaning.

## Avoid

Do not place product claims or implementation vocabulary inside working screens. Avoid copy such as:

- “Only the work that needs your judgment.”
- “Everything material is handled.”
- “Continuously prepared.”
- “Automatically maintained.”
- “Watch drift, not dashboards.”
- “Applicability determination required.”
- “Evidence sufficiency failure.”
- “Execute verification contract.”

## Prefer

| Avoid | Prefer |
|---|---|
| Needs attention | Open items, or the specific condition |
| Your attention | Assigned to you |
| Automatically maintained | Current items |
| No intervention required | No action due |
| Inspect authority | View approval route |
| Open capture wizard | Request evidence |
| Applicability determination | Does this apply? |
| Evidence insufficiency | Evidence incomplete |
| Verification contract | Outcome check |
| Decision required | Decision needed |
| Action in progress | Work in progress |
| Everything material is handled | No assigned items in this scope |

A label should be more specific when the page has a narrower population, for example “4 approvals awaiting CRO review” or “2 vendor certificates expire within 30 days.”

## Counts and status

A count must have a defined denominator, scope and freshness source. When a governed baseline is unavailable, display an em dash or “Coverage not available.” Never substitute a demo number.

Empty states are scoped statements. “No open approvals assigned to you” is valid; “No compliance issues” is not unless the entire governed population and freshness basis are visible.

## Guidance, illustrations and icons

First-run guides are brief, role-specific and tied to a real task. Titles describe the action: “Review assigned work,” “Check the approval route,” “Request additional evidence.”

Use premium vector illustrations for orientation, empty states and education. Use semantic SVG icons for recurring work types. Neither may carry status or replace text.

## Acceptance checks

Before release:

- a domain owner confirms terminology;
- an intended user can state the object, state and next action without explanation;
- enum codes are not the only visible labels;
- every count has scope and freshness;
- unknown data is explicit;
- sample data is labelled;
- no copy suggests autonomous approval, verified outcome or enterprise completeness without evidence;
- screen-reader labels carry the same meaning as visible content.
