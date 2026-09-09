# Plain-language content standard

ClearSight has a precise internal model and a human working language. Both are required.

The internal model preserves stable codes, legal meaning, auditability and machine-safe transitions. The working language helps a person using the product understand the same state without learning the implementation vocabulary.

## Two-layer rule

| Internal/API term | Primary UI wording | Specialist detail |
|---|---|---|
| `PROGRAM` | Program | Ongoing governed obligation or assurance activity |
| `MATTER` | Issue or change | Matter identifier and typed lifecycle |
| `APPLICABILITY_DETERMINED` | Does this requirement apply? | Applicability decision, scope and approver |
| `EVIDENCE_INSUFFICIENT` | Evidence incomplete | Coverage, freshness, contradiction and independence details |
| `IMPLEMENTATION_PENDING` | Change in progress | Planned or in-progress control implementation |
| `DECISION_REQUIRED` | Decision needed | Decision type and required authority |
| `VERIFICATION` | Confirming outcome | Verification contract, baseline, threshold and observation period |
| `CLOSURE_BLOCKED` | Cannot close yet | Specific unmet closure conditions |

Visible wording MUST NOT alter the underlying semantics. “Outcome check” remains separate from task completion, upload and implementation.

## Sentence shape

Prefer:

```text
[Concrete object] + [current condition] + [why now / next action]
```

Examples:

- “Four privileged accounts still need current owner approval.”
- “The vendor certificate expires in 12 days. Request the replacement certificate.”
- “This requirement applies to mobile banking and two payment vendors.”
- “The change was implemented. Confirm that unresolved accounts are now zero.”

Avoid:

- “Resolve evidence insufficiency.” Use “Provide the missing evidence.”
- “Operationalize compliance state remediation.” Use “Complete the agreed fix and confirm the result.”
- “Review materiality-driven exception handling.”
- “Leverage continuous assurance intelligence.”

## Labels

Labels are short and familiar:

- Setup in progress
- Up to date
- Needs attention
- Evidence incomplete
- Gap found
- Change in progress
- Decision needed
- Work in progress
- Preparing response
- Confirming outcome
- Cannot close yet

A technical code may appear in a tooltip, audit panel, export or API response, but not as the only visible label.

## Content acceptance

A screen passes when an intended user can answer, without product training:

1. What am I looking at?
2. What is happening now?
3. Why does it matter?
4. What do I need to do?
5. Who owns or approves it?
6. What evidence supports it?
7. What still prevents completion?

## Shared workflow copy (9 September 2026)

Shared forms, vendor workflows, errors, notifications and guides are industry-neutral. Use **Awaiting review**, not an institutional review label. Where the exact current scoped route supplies a person's name, **Awaiting review from {full name}** is appropriate. Historical attribution uses the recorded reviewer and never today's assignee; unavailable names remain **Reviewer name unavailable**, with the recorded identifier in review details.

Prefer short state labels: **Missing**, **Incomplete**, **Received**, **Awaiting review**, **Accepted**, **Rejected**, **Expired**. These describe distinct conditions. Receipt or linking does not mean acceptance, and acceptance of evidence does not mean vendor approval. Governed conditional approval retains its required conditions, owners and deadlines.

Unknown information is not zero or success. Missing calculation/count/score data displays **Unknown** or **Unavailable**; old calculations display **Out of date** with previous results identified. A missing schedule is **Not scheduled**. Refresh-success messages require a completed successful read. An uncertain network result asks the user to check the record before repeating a command.

Headings name the task or object. Supporting text adds an owner, source, date, condition, consequence or recovery step; remove it when it repeats the heading. Avoid product commentary and descriptions of internal architecture. Preserve actual policy text, source quotations, organization names, specialist API identifiers in details, and established acronyms. New requirement authoring collects the actual obligated party, action, object and obligation strength; it does not invent these from free text.

Copy review covers the complete workflow, including accessible names, server errors, saved decisions, external receipts, empty states and sample fixtures. The recursive copy regression is a guard, not a substitute for contextual review.
