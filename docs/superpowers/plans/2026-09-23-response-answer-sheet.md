# Plan: Response answer sheet with sections, summary and needs-attention views

## Goal

A bank user reviewing a submitted response (register answer sheets with 56+ fields, ordinary forms with 100+) can immediately read the response grouped under its form sections, see an honest completion summary, and switch to a focus view showing only defective or omitted fields. The same sectioned treatment applies to the Assessment screen and the submitter's pre-submit review. No backend contract change.

Approved design: `docs/superpowers/specs/2026-09-22-response-answer-sheet-design.md` (do not commit).

## Context and standing constraints

- Everything stays **uncommitted**. Do not commit. Do not revert or commit the pre-session vendor or Programs WIP already in the tree.
- **No backend contract change** (Approach A). Section titles come from the existing `loadFormTemplateRevision` read API where a fetch is needed, and from the already-loaded `request.sections` on the submitter's review.
- **No new tokens, density modes, motion patterns or illustration styles.** Reuse existing components, classes and `--cs-*` tokens.
- Answer values are **data, not copy** — the sheet never re-labels stored answers; per-field attention reasons reuse existing strings only.
- Customer-facing copy changes require the copy-quality regression (`web/src/copyQuality.test.ts`) and the same review as workflow changes.
- Evidence captures MUST use the evidence build (`npm run build:evidence`) served via `npx vite preview --config vite.evidence.config.ts --port 4173 --strictPort` (server currently running, pid 45560).
- Working tree is fully uncommitted (76 entries). Behave as if `git` is read-only for this work.
- Current session: `ses_f3690a2beffe6fTV6GfXOuR9EY`.

## Design decisions (locked)

### New component `ResponseAnswerSheet` (new file, TDD)

File: `web/src/components/forms/ResponseAnswerSheet.tsx` (+ `ResponseAnswerSheet.test.tsx`, + `response-answer-sheet.css`).

```ts
export type AnswerSheetField = Pick<FormTemplateField, "id" | "label" | "type" | "required" | "section_id" | "description">;
export type AnswerSheetItem = { field: AnswerSheetField; answer?: CaptureAnswerValue; decision?: { points: number; outcome_id?: string } };

type Props = {
  fields: AnswerSheetItem[];                          // order-preserving: field, answer, optional decision
  sections?: FormTemplateSection[];                   // absent/empty → single "All answers" group fallback
  assessmentContext?: ResponseAssessmentDetail;        // when present, enables POOR attention classification
  title?: string;                                     // optional h3; CaptureReview keeps its own h2 and passes none
  note?: string;                                      // form-revision line, rendered under title
  emptyLabel: string;                                 // e.g. "No answer submitted for this field."
  evidenceEmptyLabel: string;                         // e.g. "No evidence submitted for this field."
  onOpenDocuments?: () => void;                       // forwarded to AnswerValueDisplay launcher
  excludeFromAttention?: (item: AnswerSheetItem) => boolean; // CaptureReview: documentAlreadyReceived fields
  renderValue?: (item: AnswerSheetItem) => ReactNode;         // CaptureReview: "No upload needed." + provenance
};

export type AnswerAttention = "MISSING_ANSWER" | "MISSING_EVIDENCE" | "POOR";
```

Note: earlier discussion called this prop `items`; the approved spec names it `fields`. The plan uses **`fields`** (the spec's naming); the type is `AnswerSheetItem[]`.

Renders, top to bottom:

1. `<section className="response-assessment" aria-label={title}>` root (aria-label only when `title` is provided; without a name it is not a landmark, matching the `CaptureReview` heading structure); optional `<h3>{title}</h3>` and `<p className="response-answer-sheet__note">{note}</p>`.
2. Summary strip `AnswerSheetSummary` (exported sub-component, see below).
3. Segmented switch (only when `fields.length > 0`): `<div role="group" aria-label="Answer view">` with two `<button aria-pressed>` — "All answers" and "Needs attention" — reusing the `capture-mode-switch` pattern (role=group + aria-pressed buttons, as in `CaptureForm.tsx`; no segmented-control component exists). When zero items need attention, the "Needs attention" button is `disabled` and the explanation is shown (see Copy).
4. Sectioned groups (standard sheets only; the Assessment screen composes its own groups around existing field articles via the exported `groupAnswerFields` helper, see Task 3):
   - sections provided → groups in template order, item matched by `item.field.section_id`; each section renders `<h4>` heading with divider + higher-contrast treatment from existing tokens, optional `help` line, and a per-section mini-count "10 of 12 answered"; unmatched/unsectioned items render in a trailing group titled "Other answers".
   - no/empty sections → single group titled "All answers".
   - zero items → existing empty copy "No answer fields were recorded for this submitted response." and no summary/switch.
5. Per group: `<dl className="response-answer-sheet__list">` where each item is `<div className="response-answer-sheet__row"><dt>label</dt><dd>value + (in focus view) reason</dd></div>`. Default value rendering uses `AnswerValueDisplay` with the passed `emptyLabel` / `evidenceEmptyLabel` / `onOpenDocuments`. In the "Needs attention" view each flagged row shows why, reusing existing per-field strings (see Copy).

Internal state: `view = "ALL" | "NEEDS_ATTENTION"`; `answerAttention(item, context)` decides the reason.

### Attention classifier (shared helper, exported from the sheet)

```ts
const evidenceTypes = new Set(["file", "photo", "signature", "vendor_document"]);

export function answerAttention(item: AnswerSheetItem, context?: ResponseAssessmentDetail): AnswerAttention | undefined {
  const present = answerIsPresent(item.answer);
  if (!present) return evidenceTypes.has(normalizeFieldType(item.field.type) ?? "") ? "MISSING_EVIDENCE" : "MISSING_ANSWER";
  if (!context) return undefined;
  const threshold = assessmentConcernThreshold(context);
  const decisionPoor = item.decision != null && assessmentConcernPoints(item.decision.points, context) >= threshold;
  const automaticPoor = automaticFieldResults(item as ResponseFieldAssessment, context).some((result) => result.concern !== undefined && result.concern >= threshold);
  return decisionPoor || automaticPoor ? "POOR" : undefined;
}
```

- `answerIsPresent` / `normalizeFieldType` from `web/src/components/capture/contract.ts`.
- `assessmentConcernThreshold` / `assessmentConcernPoints` / `automaticFieldResults` from `web/src/components/forms/assessmentFieldResults.ts`.
- POOR only fires when `assessmentContext` is present. Only the Assessment screen passes it, and only with full `ResponseFieldAssessment` fields (the ones `automaticFieldResults` expects — the `as ResponseFieldAssessment` cast in the classifier is safe because POOR never runs against the partial field picks used by Answers-tab/CaptureReview data, which do not pass `assessmentContext`). The classifier never invents poor results.

### `AnswerSheetSummary` (exported) — shared strip

```ts
export function AnswerSheetSummary({ fields, assessmentContext, emptyLabel, evidenceEmptyLabel }: { ... }): JSX.Element
```

Renders: "10 of 12 fields answered" (answered = `answerIsPresent`), then — only when at least one item needs attention — "2 fields need attention" plus per-criterion chips ("1 no answer", "1 no evidence", and with `assessmentContext` "2 poor results"). Denominators come from the `fields` passed in; zero fields → renders nothing (callers own their empty state).

The sheet's own strip uses this internally via `answerAttention`; the Assessment screen reuses it directly with `assessmentContext={detail}`.

### `groupAnswerFields` (exported)

```ts
export function groupAnswerFields(fields: AnswerSheetItem[], sections?: FormTemplateSection[]): Array<{ id?: string; title: string; help?: string; fields: AnswerSheetItem[] }>
```

Order-preserving; sections kept in `sections` order; unmatched/unsectioned fields trail in "Other answers". Used by the standard sheet internally, by the Assessment screen to group its existing field articles, and by tests.

## Copy (all new strings listed for the copy gate)

- "All answers", "Needs attention" — switch labels (view names; follow the "Awaiting review" convention without duplicating it).
- "10 of 12 fields answered" — summary count with a stored denominator.
- "2 fields need attention" — attention headline (only when count > 0).
- Chips: "1 no answer", "1 no evidence", "2 poor results" (unit counts; singular/plural handled).
- Disabled "Needs attention" explanation: "No fields in this response need attention."
- Focus-view reasons reuse existing strings: "No answer submitted for this field." / "No evidence submitted for this field." / "Poor result · N concern points".
- Empty state keeps existing copy: "No answer fields were recorded for this submitted response."
- Section mini-count: "10 of 12 answered".

All counts derive from stored or explicitly labelled sample data; no invented denominator. Run `web/src/copyQuality.test.ts` after wording lands.

## Tasks

Work in order; each task is TDD (write/adjust tests, then implementation, then run the target suite).

### Task 1 — `ResponseAnswerSheet` component + CSS + component tests

New files:

- `web/src/components/forms/ResponseAnswerSheet.tsx`
- `web/src/components/forms/response-answer-sheet.css` (imported by the component itself so both the forms and capture surfaces get it; reuse `--cs-*` tokens and existing `field-assessment.css` / `capture-mode-switch` patterns; no new tokens/density/motion)
- `web/src/components/forms/ResponseAnswerSheet.test.tsx`

`ResponseAnswerSheet.test.tsx` covers:

1. Groups fields under section headings in template order; per-section mini-count "10 of 12 answered"; unmatched fields land in a trailing "Other answers" group; absent sections → single "All answers" group.
2. Summary counts: "10 of 12 fields answered" with a mixed fixture; attention headline + chips appear only when at least one item needs attention.
3. All three criteria: a field with `answer: {}` (MISSING_ANSWER), an evidence-type field with `answer: {}` (MISSING_EVIDENCE), and — only with `assessmentContext` — a decision/automatic result at or above the threshold (POOR, "Poor result · N concern points"). Assert POOR does NOT appear without `assessmentContext`.
4. Segmented switch: role=group with two `aria-pressed` buttons; clicking "Needs attention" filters rows to flagged ones and shows the reason; clicking "All answers" restores; zero attention items → "Needs attention" disabled + the explanation is visible; `fields.length === 0` → no switch, empty copy only.
5. `excludeFromAttention` and `renderValue` behavior (received-looking fields render but are never flagged; custom value node replaces the default row value).
6. Axe: grouped, single-group, and focus-view renders produce no violations via `axe-core` directly in the test (`axe.run(container, { rules: { "color-contrast": { enabled: false } } })`), following the exact pattern in `web/src/Accessibility.test.tsx` (L72–73).
7. `title`/`note` render an h3 + note line; both omitted → no h3.

Acceptance: `npx vitest run src/components/forms/ResponseAnswerSheet.test.tsx` green.

### Task 2 — Answers tab wiring in `ResponseAssessment`

File: `web/src/components/forms/ResponseAssessment.tsx`.

- Import `loadFormTemplateRevision` from `../../formsApi` and `FormTemplateSection` from `../../monitoringTypes`.
- In `AssessmentContent`, add state `templateSections` and a non-blocking effect that runs whenever `detail` is present (both assessment and answersOnly modes — the spec's "same template fetch"), calls `loadFormTemplateRevision(detail.form_template_id, detail.form_template_version)`, stores `template.sections ?? []`, and silently falls back to `undefined` on error (single-group / ungrouped fallback). Guard with a cancellation flag; clean up on unmount / `detail` change. The graceful single-group render is the initial state, not an error.
- Replace the `answersOnly` branch (currently line 91: `<h3>Submitted answers</h3>` + flat per-field `<article>` loop) with:

  ```tsx
  if (answersOnly) return <ResponseAnswerSheet
    fields={fields}
    sections={templateSections}
    title="Submitted answers"
    note={`Form revision ${detail.form_template_version}. These answers cannot be changed.`}
    emptyLabel="No answer submitted for this field."
    evidenceEmptyLabel="No evidence submitted for this field."
  />;
  ```

  (`fields` is already `detail.fields ?? []`.) Do NOT pass `assessmentContext` here — POOR is Assessment-only. Remove the now-unused `evidenceField`-based `emptyLabel` ternary from the branch.

Files: `web/src/components/forms/ResponseAssessment.test.tsx` and `web/src/components/forms/ResponsesReview.test.tsx`:

- Both need `vi.mock("../../formsApi", () => ({ loadFormTemplateRevision: vi.fn() }))` because the component now imports that module.
- Fixtures: give at least two assessment fields a `section_id` and mock `loadFormTemplateRevision` to resolve a template whose `.sections` match those ids (e.g. one section containing those fields), so section headings are provable.
- Add assertions on the answers-only render: section heading visible; summary strip shows "X of Y fields answered"; a missing-answer fixture field surfaces under "Needs attention".
- `ResponsesReview.test.tsx` keeps its existing judgement/conflict flow assertions intact.

Acceptance: `ResponseAssessment.test.tsx` and `ResponsesReview.test.tsx` green (full files, not only the new tests).

### Task 3 — Assessment screen summary strip + section grouping (helpers only, no switch)

File: `web/src/components/forms/ResponseAssessment.tsx` (non-answersOnly render).

- After the scores block and before the field filter/panels, render the exported strip:

  ```tsx
  <AnswerSheetSummary fields={detail.fields ?? []} assessmentContext={detail} emptyLabel="No answer submitted for this field." evidenceEmptyLabel="No evidence submitted for this field." />
  ```

- Group the **visible** field articles under section headings using the exported helper: `const groups = groupAnswerFields(visible, templateSections)`. Each group with `title` renders an `<h4>` heading (matching the sheet's per-section heading treatment, with divider) followed by the existing field articles of that group; groups with no heading (absent sections) render the articles flat as today. The existing judgement editing panels, the filter `SelectField`, per-field `AnswerValueDisplay` formatting and the `EmptyState` stay exactly as they are. No segmented switch on this surface (the switch stays on the answers surfaces); attention here is expressed only through the summary strip's chips.
- The fetch from Task 2 supplies `templateSections` on this surface too (spec: "same template fetch"). When sections are absent or the fetch fails, fields render flat as today — the fetch is non-blocking.

File: `web/src/components/forms/ResponseAssessment.test.tsx` — assert the Assessment render includes the summary strip with counts and, for a fixture with a poor decision/automatic concern, a "poor results" chip; assert that section headings appear around the field articles when the mocked template resolves sections, and that behaviour is unchanged when it does not; assert other Assessment behaviors are unchanged (judgement panels, filter, edit/create-review flows).

Acceptance: Assessment suite green.

### Task 4 — Swap `CaptureReview` to the sheet

File: `web/src/components/capture/CaptureReview.tsx`.

- Replace the intent-grouped `groupReviewFields` list with `<ResponseAnswerSheet>`. Keep the existing `<span className="eyebrow">Review response</span><h2>Check your response</h2><p>{request.title}</p>` header and the "Request details" `<details>` block.
- Map capture fields to `AnswerSheetItem`: `{ field: { id, label, type, required, section_id, description }, answer: answers[field.id] }`. Pass `sections={request.sections}` (already on the request; no fetch). `request.sections` is optional — when undefined the sheet falls back to a single group.
- Props: `emptyLabel="Not provided"`, `evidenceEmptyLabel="No evidence submitted for this field."`.
- `excludeFromAttention`: return true for fields where `documentAlreadyReceived` applies, so received-looking documents render but are never flagged as missing.
- `renderValue` preserves existing per-row behavior: "No upload needed." for `documentAlreadyReceived(field) && !answerIsPresent(answer)` fields, and the `source-origin-review` small provenance label (from `reviewProvenanceLabel`) under every value.
- Keep `onEdit`, `onSubmit`, `submitting`, `error`, `errorKind`, `onReload` and the existing error/summary chrome untouched. Remove `groupReviewFields` and its now-unused helpers only if nothing else references them (check imports).

File: `web/src/components/capture/CaptureReview.test.tsx`:

- Update the "groups confirmations, proposed updates and document changes" test to assert form-section headings instead of intent-group names; give the fixture request `sections` (e.g. "Identity records" containing `address`/`name`, "Documents" containing `certificate`/`policy`).
- Keep/verify: "No upload needed." still renders for the received-document field; provenance labels render; a missing answer field appears in "Needs attention" while the received-document field does not.

Acceptance: `CaptureReview.test.tsx` green; related capture-flow tests elsewhere still green.

### Task 5 — Demo fixtures for the register answer sheets

File: `web/src/staticDemo.ts`.

1. Add a branch to `formsTemplatePopulation` (near L262–324) for `fixture === "program-responses"` returning a register template (id `form-branch-kri-register`, version 1) whose `sections` and per-field `section_id`s match the register assessment fields. Two sections, e.g. `{ id: "branch-details", title: "Branch details" }` and `{ id: "november-2025", title: "November 2025 register" }`. Base it on `staticFormLibraryTemplate()` shape so the existing revision route at L1006–1011 (`formRevisionMatch`) resolves it. The library list at L994 is not affected for the fixtures that render it.
2. In `programResponseAssessment` register branch (L1344–1361): add `section_id` to the ten existing register fields, and add two new fields so the needs-attention criteria render visibly in evidence:
   - a missing-answer field, e.g. `{ field: { id: "followup_owner", label: "Follow-up owner", type: "short_text", required: true }, answer: {} }` in "Branch details";
   - a missing-evidence field, e.g. `{ field: { id: "supporting_document", label: "Supporting evidence for the register", type: "file", required: false }, answer: {} }` in "November 2025 register".
   Net: 12 fields, 10 answered → summary "10 of 12 fields answered", "2 fields need attention" (1 no answer, 1 no evidence).
3. Keep captures 56/57 passing in the default All-answers view (the typed-answer rows `/1,250,000/`, "Cash overage", "Electrical surge", "Confirmed" remain visible).
4. Vendor answers tab in the demo (`forms-response-history` fixture): its assessment returns `fields: []` (L1070–1073), so the sheet renders the empty state there, not a sectioned list — no vendor template branch needed. The fetch for `loadFormTemplateRevision("form-vendor-due-diligence", 2)` cannot resolve (template version is 3), and that is covered by the designed fallback; no demo-side change required for vendor.

File: `web/src/staticDemo.test.ts`:

- Update the register assessment test (L851–856): expected field id list becomes `["branch", "directorate", "region", "cash_overage_value", "cash_shortage_count", "reporting_date", "manual_confirmation", "event_types", "attestation", "notes", "followup_owner", "supporting_document"]`; add `section_id` assertions for `branch` and `cash_overage_value`; keep the existing answer assertions (1250000, Cash overage/Electrical surge, "true").
- Add: `GET /api/v1/forms/templates/form-branch-kri-register/revisions/1` under `program-responses` returns the template with the two sections; `GET …/revisions/999` yields the existing 404.

Acceptance: `npx vitest run src/staticDemo.test.ts` green.

### Task 6 — Extend the evidence capture script + re-run captures

File: `web/scripts/capture-program-review-evidence.mjs`, function `captureProgramRegisterResponseSheets` (L272–293).

Add assertions before the screenshot (keep the existing typed-answer assertions):

```js
await page.getByRole("heading", { name: "Branch details" }).waitFor({ state: "visible" });
await page.getByRole("heading", { name: "November 2025 register" }).waitFor({ state: "visible" });
await page.getByText("10 of 12 fields answered", { exact: true }).first().waitFor({ state: "visible" });
await page.getByText("2 fields need attention", { exact: true }).first().waitFor({ state: "visible" });
await page.getByRole("button", { name: "Needs attention" }).click();
await page.getByText("Follow-up owner", { exact: true }).waitFor({ state: "visible" });
await page.getByText("No answer submitted for this field.", { exact: true }).waitFor({ state: "visible" });
```

Then switch back to the default view before capturing (keep the existing all-answers PNG compositions for 56/57 intact):

```js
await page.getByRole("button", { name: "All answers", exact: true }).click();
await page.getByText("Cash overage", { exact: true }).first().waitFor({ state: "visible" });
```

(Exception: if one of the two captures is chosen to show the focus view deliberately, keep that one in "Needs attention" and record the view state in the capture `state`; do not change both.)

Re-run: `npm run build:evidence` (if needed), then the capture script against the preview server on port 4173. Regenerated PNGs and `manifest.json` land in `docs/evidence/2026-09-22-program-data-centered-ux/`. Update `docs/quality/rendered-ui-evidence.md` entries for the register captures to mention section headings + summary strip. Keep everything uncommitted.

Acceptance: script exits clean; both 56/57 captures show sectioned headings, the summary strip and no horizontal overflow (`assertNoHorizontalOverflow` retained).

### Task 7 — Verification

1. `web/src/copyQuality.test.ts` regression (new strings are concrete/count-based; extend the pattern list only if a new class of product narration becomes detectable).
2. Typecheck (confirm the exact command from `web/package.json` — `npm run typecheck` or repo convention) for the whole repo.
3. Full web test suite (repo's test command), including `ResponseAnswerSheet`, `ResponseAssessment`, `ResponsesReview`, `CaptureReview`, `staticDemo`, `copyQuality`.
4. Render the materially affected workspaces (Answers tab for a register response, Assessment screen, CaptureReview) at desktop 1440×900 and mobile 390×844 evidence viewports; visually inspect the new PNGs; fix the highest-impact failure and re-check.
5. Confirm guides/notices/errors do not block the primary actions on these surfaces.
6. Leave the working tree uncommitted.

## Definition of done

- `ResponseAnswerSheet` + `AnswerSheetSummary`/`groupAnswerFields`/`answerAttention` exist and are covered by component tests (grouping, all three attention criteria with and without assessment context, switch + disabled state, focus filtering, single-group fallback, empty state, exclude/renderValue hooks, axe).
- Answers tab, Assessment screen and CaptureReview all use the sectioned treatment; judgement/filter workflows unchanged on Assessment and CaptureReview.
- Static demo register fixture produces 12 fields with two sections, one missing answer, one missing evidence; the demo revision route serves the register template.
- Evidence captures 56/57 re-run and assert section headings, the summary strip, the focus view and no overflow; docs updated.
- Copy-quality regression, typecheck and full web tests pass; no new tokens/density/motion/illustration styles.
- Nothing committed; pre-session vendor + Programs WIP untouched.

## Open items carried from prior session (not part of this plan)

- Human-eye review of the 21 PNGs in `docs/evidence/2026-09-22-program-data-centered-ux/`.
- Shutting down the evidence preview server (port 4173, pid 45560) after captures.
- Deciding whether to commit the full uncommitted working tree (deferred; this plan stays uncommitted).