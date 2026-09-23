# Response answer sheet: sections, summary and needs-attention views

## Outcome

A bank user reviewing a submitted response (including register answer sheets with 56+ fields and ordinary forms with 100+) can immediately read the response grouped under its form sections, see an honest completion summary, and switch to a focus view showing only defective or omitted fields. The same sectioned treatment applies to the Assessment screen and the submitter's pre-submit review.

## Problem

The Answers tab of a submitted response currently renders a flat list: every field as its own article under the sheet title, with no section grouping, no heading hierarchy or dividers, no summary, and no way to isolate what is missing or defective. In a 100+ field form this forces the reviewer to scan every answered row to find the few that matter, and it does not reflect the structure of the form the subject actually answered. Google Forms, Typeform and SurveyMonkey all present individual responses under the form's sections with clear label/value contrast and an overview layer that front-loads completion state.

## Decision

Add one shared component, `ResponseAnswerSheet`, and use it on all three surfaces. No backend contract change; section titles come from the existing form-template read API where a fetch is needed, and from the already-loaded request on the submitter's review.

Surfaces and their section sources:

- **Answers tab** (`ResponsesView` and `ProgramResponsesPanel` review sheets): `ResponseAssessment` in `answersOnly` mode fetches the form template through the existing `loadFormTemplateRevision(form_template_id, form_template_version)` and passes `sections` to the sheet. The fetch is non-blocking: fields render immediately in a single "All answers" group, upgraded to sectioned headings when the template arrives.
- **Assessment screen**: same template fetch; additionally passes an `assessmentContext` so fields with poor automatic or reviewed results count as needs-attention. The sheet provides the summary strip and the per-field answer formatting; the existing judgement editing panels remain unchanged.
- **Submitter's pre-submit review** (`CaptureReview`): the request already carries `sections`; no fetch. Only missing-answer and missing-evidence criteria apply because no assessment results exist yet.

Component props: `fields` (order-preserving, with `field`, `answer`, optional `decision`), optional `sections`, optional `assessmentContext`, `title`, `emptyLabel`, `evidenceEmptyLabel`, optional `onOpenDocuments` (when omitted, evidence rows show counts and metadata without a launcher, matching today's answers-only render). Internal state is `view = "ALL" | "NEEDS_ATTENTION"`.

### Views and layout

- **Summary strip** (persistent above both views): honest counts computed from the stored fields — "54 of 56 fields answered", "2 fields need attention" — plus per-criterion chips ("1 no answer", "1 no evidence", and on the Assessment screen "2 poor results"). Per-section mini-counts ("12 of 14 answered") render beside each section heading. Denomination comes from the actual fields; zero fields render unknown phrasing, never a persuasive number.
- **Segmented switch**: "All answers" / "Needs attention", reusing the product's segmented-control pattern. With zero needs-attention items the focus option renders disabled with an explanation.
- **Sectioned list** (both views): each section renders an `h4` heading with divider and higher-contrast treatment from existing tokens, optional section help line, then answer rows reusing `AnswerValueDisplay` and the existing `response-assessment__answer` styling. In the focus view each flagged row shows why, using existing per-field copy: "No answer submitted for this field." / "No evidence submitted for this field." / "Poor result · N concern points".

### Attention criteria

- **Missing answer**: the field has no present answer value (`answerIsPresent` semantics used by the capture flow).
- **Missing evidence**: the field is an evidence type (`file`, `photo`, `signature`, `vendor_document`) and has no artifact.
- **Poor result** (Assessment screen only): the saved decision reaches the existing `assessmentConcernThreshold`, or an automatic field result carries a concern at or above the threshold, using the existing `assessmentConcernPoints` / `automaticFieldResults` helpers.

### Fallbacks

- Template fetch fails or sections are absent: render a single "All answers" group; summary and focus filtering still work.
- Zero fields: keep the existing copy "No answer fields were recorded for this submitted response."
- Answers tab fetch is non-blocking; a graceful single-group render is the initial state, not an error.

### Copy

- Segmented labels "All answers" and "Needs attention" follow the existing "Awaiting review" convention without duplicating it.
- All per-field attention reasons reuse existing strings; no new product narration.
- Buttons name the direct result; supporting copy explains why the row is shown.

## Alternatives considered

1. **Backend returns sections in the assessment response.** Cleaner single source of truth, but changes the Go response struct, API contract, all fixtures and tests for the same visible result, and contradicts the scoped-change direction for the surrounding work.
2. **Group by `section_id` with generic headings.** No template fetch, but headings would read "Section 1 / Section 2", failing the heading-contrast requirement.
3. **Jump list instead of a focus view.** An upfront "needs attention" jump list scrolls to fields in the full list; rejected in favour of a genuine focus mode so reviewers are not forced to scroll past answered rows.

## Validation

- `ResponseAnswerSheet.test.tsx`: section grouping, summary counts for all three criteria, segmented switch and disabled state, focus-view filtering, single-group fallback, empty state, axe accessibility.
- `ResponseAssessment.test.tsx` and `ResponsesReview.test.tsx` fixtures updated to assert section headings and the summary strip on the answers-only render.
- Register answer-sheet captures 56/57 in `capture-program-review-evidence.mjs` assert section headings and the summary strip; no horizontal overflow at desktop and mobile widths.
- Copy-quality regression, typecheck, full web test suite, and rendered-evidence re-capture pass before completion.
- New tokens, density modes, motion patterns and illustration styles are not introduced; existing tokens and classes are reused.

### Demo and rendered evidence

- The static demo gains a `loadFormTemplateRevision` route for `form-branch-kri-register` (and the existing vendor template where needed) so the answers-only render can group by section in the evidence build.
- The two register demo responses (CAC and Marina) extend their fields with matching `section_id` values so the sectioned headings, per-section counts, summary strip and needs-attention filtering are provable in captures 56/57.
- The register fixture keeps at least one missing-answer field and one evidence field without an artifact so the needs-attention criteria render visibly in evidence.