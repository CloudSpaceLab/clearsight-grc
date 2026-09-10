# Third-party semantic capture and response review

## Outcome

The Cloudspace sample must read as a vendor submission reviewed by bank staff, not as a spreadsheet rendered one cell per card. The current immutable source submission remains in history. A replacement form revision and sample response become current only after the corrected records are complete.

## Source interpretation

The workbook is a combined register. Its columns have different business meanings and must not share one form-field treatment.

| Workbook content | Canonical destination | Current presentation |
| --- | --- | --- |
| Service provider and services offered | Existing vendor relationship and service scope | Response summary |
| Vendor comment | Vendor response to the relevant requirement | Vendor responses |
| Supporting certificate/report/addendum | Evidence request; absent unless actually supplied | Evidence coverage |
| Assessor and assessment date | Internal assessment attribution | Internal assessment |
| Finding, risk/implication, severity and overall rating | Bank field assessment and linked deficiency Matter | Internal assessment and findings |
| Recommendation, responsibility, timeline and status | Assigned Action and its source status/deadline | Findings and actions |
| Business owner | Relationship or service accountability | Assignment summary |
| Serial number, file, sheet, cells and digest | Source provenance | Collapsed source details |

Blessing and Joel are internal assessors. Hakeem is the performer for remediation Actions. POS Business is the accountable business function. Cloudspace is the vendor relationship. None of these names or roles is a vendor-answer field.

## Replacement capture

Create one replacement revision titled **Third-party security and continuity review**, grouped by the two source services. Each finding becomes a requirement row with:

- the vendor's recorded response as the submitted answer;
- a separate supporting-evidence request where the workbook identifies a certificate, report or contractual document;
- manual bank assessment configuration with a concise rubric such as `Satisfactory`, `Needs follow-up` and `Material concern`;
- the appropriate internal reviewer attribution outside the answer contract.

The source does not prove that any supporting file was submitted, so evidence fields remain unanswered and the UI must say that no document was received. The source rating remains a recorded historical bank judgement and does not become an automatic current score. Existing five deficiency Matters and five Actions remain canonical; the replacement submission links to them rather than creating duplicates.

The earlier flattened form distribution is superseded only after the replacement submission, reviewer assignments and links are present. Its submission and audit history remain immutable and readable as historical data.

## Premium response review

The wide review sheet uses a dense, calm hierarchy built from existing ClearSight tokens, typography and light/dark themes. External design recommendations support data density and progressive disclosure; they do not replace the established colour or font system.

1. A sticky summary header identifies vendor, service scope, submitted date, response revision and assurance.
2. Four compact metrics show requirements answered, evidence received, fields awaiting bank review and linked open findings. Unknown values remain unknown.
3. A service switch or anchored section navigation groups Moneytor GetPaid and PTSP without repeating vendor metadata.
4. Each requirement uses a compact comparison row: requirement, vendor response, evidence state and bank assessment state. Expanding a row reveals the full answer, provenance and assessment controls.
5. A separate **Internal assessment** section shows assessor, assessment date, source severity/rating and recorded field decisions. It must never imply the vendor supplied these values.
6. **Findings and actions** shows the five linked deficiencies, Hakeem's assignments, source deadline and current action state.
7. **Source details** is collapsed by default and contains the workbook identity, digest, sheet and cell range.

The phrase **Existing form rules** is removed when no assessment policy exists. Answer cards do not repeat **Submitted answer and evidence** for every row. Labels identify `Vendor response`, `No document received`, `Bank review needed` and `Reviewed` directly.

At narrow widths, the metrics become a two-column grid and requirement comparison rows become stacked disclosure cards. The service selector remains visible, controls retain keyboard order, status is never colour-only, and focus returns to the triggering control when nested evidence views close.

## Data flow and authority

The private source manifest gains an explicit semantic mapping per group/field rather than relying on column position. The importer validates every mapped source column and fails closed on an unknown semantic category. It creates or reuses the replacement form, saves the vendor response through the normal protected capture path, records bank field assessments through authorized reviewer commands, and reuses existing Matter/Action identifiers.

Actors supplied by a manifest are suggestions only. The installer resolves each named person against the existing demo principals and current responsibility routes. Missing or ambiguous mappings stop that assignment and report the source name; they do not create an arbitrary form answer or silently grant authority.

## Failure and recovery

- A changed source digest requires a new source version; no immutable answer is overwritten.
- A partial replacement leaves the existing current submission untouched and is safe to retry by idempotency key.
- A missing reviewer route leaves the bank assessment visibly unassigned while preserving the vendor response.
- Missing evidence stays missing and cannot be converted into a positive assessment.
- Projection refresh errors do not roll back committed captures or assessments; freshness remains explicit.

## Focused acceptance

- Cloudspace has one current semantically modelled submission and the flattened submission is historical.
- No submitted-answer field is named assessor, business owner, service provider, responsibility, timeline, status, risk-owner comment or serial number.
- Two service groups and five vendor requirements are visible.
- Blessing/Joel appear only as internal assessors, Hakeem only as Action performer, and POS Business only as accountable function.
- Five existing deficiency Matters and five Actions remain, with no duplicates.
- The response review metrics equal stored answers, evidence, required reviews and links.
- Representative 1440px and 390px light/dark renders show the summary, one expanded requirement, internal assessment and findings without horizontal overflow or trapped focus.
- Verification is limited to the importer contract, affected response-review tests, production build, exact hosted counts and representative renders.

## Non-goals

This change does not establish vendor approval, evidence sufficiency, remediation completion or verified outcomes. It does not introduce a second vendor-register database, infer documents not present in the source, or redesign unrelated Forms screens.
