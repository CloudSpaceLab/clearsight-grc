# Plain-language content standard

ClearSight has a precise internal model and a human working language. Both are required.

The internal model preserves stable codes, legal meaning, auditability and machine-safe transitions. The working language helps a bank employee understand the same state without learning the implementation vocabulary.

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

- “Resolve evidence insufficiency.” (avoid; use “Provide the missing evidence.”)
- “Operationalize compliance state remediation.” (avoid; use “Complete the agreed fix and confirm the result.”)
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
