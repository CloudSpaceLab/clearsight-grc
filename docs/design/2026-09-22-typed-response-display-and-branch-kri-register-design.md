# Typed response display and Branch KRI register design brief

**Decision date:** 2026-09-22

**Status:** Approved for implementation

## Product job

A bank stakeholder reading a submitted form response must see the answer the way the
question was asked: a currency as a formatted amount, a date as a date, a selection as
clearly separated options, a yes/no as a state. The same readability must hold whether
the response is being read back (submitted answers, assessment review, vendor review)
or summarised before submission (capture review). At the same time, the seeded Branch
KRI data must reflect how the source workbook is actually structured: one Branch KRI
register that each branch answers, not one near-identical form per branch.

## Problem inventory

1. **Answers read back as raw text.** `ResponseAssessment.answerText()` renders
   `answer.text`, joins `answer.values` with `", "`, or prints a document-count
   sentence. Every field type renders identically: a currency shows as `"1250000"`,
   a date as an ISO-ish string, a multi-select as comma-joined text. The fill side
   (`CaptureFieldControl`) is type-aware; the read side is not.
2. **Source rows collapse into paragraphs.** Seed imports map each source row to a
   `long_text` answer, so a branch KRI row renders as one long unbroken paragraph
   with no visual structure.
3. **Duplicate Branch KRI forms.** `source_records_ops.json` models the workbook as 31
   separate groups (`ops-branch-kri-r2…r32`), one per branch, each with one record of
   an identical 51-question schema. The installer creates one form per group, so the
   demo shows 31 near-identical "‹branch› branch KRI" forms with one response each,
   instead of one Branch KRI register with one response per branch (Archer #17).
4. **No documented Archer gap baseline.** The Fidelity Bank_Archer (1).xlsx workbook
   (31 requirements across IT Risk, Operational Risk, Data Privacy, Cyber Security)
   has no durable mapping to ClearSight's current capabilities, so "what is left"
   is not traceable.

## Target structure

### A — Shared typed answer display

New presentational component `web/src/components/forms/AnswerValueDisplay.tsx`
rendering one `CaptureAnswerValue` for one field by normalized type, using only
existing components, classes and tokens:

| Field type | Rendering |
| --- | --- |
| short_text / long_text / email / url / telephone | text, long text keeps line structure (no collapse/overflow) |
| integer / decimal / percentage | `Intl.NumberFormat` (existing `formatNumber` style) |
| currency | locale currency (NGN) formatting |
| date | existing `formatDate` |
| yes_no | Yes / No neutral `StatusBadge` |
| checkbox / attestation | Confirmed / Not confirmed |
| single_select | option label |
| multi_select | tag/chip list (existing chip/badge tokens) |
| file / photo / signature / vendor_document | document count + metadata + existing DocumentBrowser launcher |

Consumers replaced/integrated: `ResponseAssessment` (answers-only sheet and field
review rows), `CaptureReview` (fill summary), `VendorResponseReview` value
comparison, `SentFormsView` detail where answers are shown. Internal predicates keep
using `answerText()`; visible rendering routes through `AnswerValueDisplay`.

### B — Branch KRI register reshape

- `cmd/seed-bank-reference/source_records_ops.json`: merge the 31 `ops-branch-kri-*`
  groups into one group `ops-branch-kri` (title "Branch KRI — November 2025",
  same `source_file`/`source_sheet`/`period`, all 31 branch rows under `records`)
  with `"response_per_record": true`.
- `cmd/seed-bank-reference/source_records.go`: `installSourceRecords` handles
  `response_per_record` groups — build the register form once from the shared
  question schema (record 0's fields), then create one distribution + one submission
  per branch record, values keyed to the shared field IDs, idempotency key per
  record (`sourceRecordPackage:group.key:record.key`) for restart recovery.
- Outcome: one **Branch KRI — November 2025** form per program with one response per
  branch. Head-office KRI groups stay as-is (one register per function-sheet).
- Seed fields stay `long_text`; typed, formatted reading is proven by the staticDemo
  fixture (below).

### C — StaticDemo typed fixture + rendered proof

- Extend `web/src/staticDemo.ts` `program-responses` fixture with a Branch KRI
  register response carrying typed fields (currency, date, number, yes_no,
  multi_select, attestation) so the answer sheet demonstrates formatted rendering.
- Re-run the Program evidence captures (48-55) so the rendered sheet evidence shows
  the typed format and the register response; update `docs/quality/rendered-ui-evidence.md`.

### D — Archer gap document

New `docs/product/archer-requirements-gap.md`: 31-item mapping table (status per
item: **Covered / Partial / Blueprint / Gap**), linked to use-case catalogue IDs and
modules, with gap notes (loss register #15/#19, RCSA #16, BIA #18, incident log #20,
consent management #30, cadence reminders #2/#4/#9/#16, DPO approval node #23,
policy lifecycle #6, tool integrations #31). Labelled reference data from the
provided workbook, dated, not legal advice.

## State matrix

| Surface | Required fixtures |
| --- | --- |
| Answers sheet | typed fields render per type table above; long text keeps structure; documents still show expiry and launcher |
| Capture review | summary shows typed values, not raw text |
| Program evidence sheet | Branch KRI register response with typed answers at 1440 desktop light and 390 mobile (dark where the workflow covers it) |
| Responses list | one Branch KRI register form title with per-branch response names |

## Copy decisions

- Answer values are data, not copy; empty/unknown answers keep the existing honest
  wording ("No answer submitted for this field", "Not provided").
- Response title for register submissions: `Branch KRI — November 2025 · CAC` (form
  name + branch) so responses are distinguishable in the list.
- No persuasive or product-review language; copy-quality regression must pass.

## Non-goals

- No typed-field inference or mass JSON edits in the Go seed (fields remain
  `long_text`); typed reading is proven via the staticDemo fixture.
- No head-office KRI reshape (stays one register per function-sheet).
- No new component families, design tokens, density modes or motion patterns.
- No backend API surface changes beyond the seed installer's register handling
  (scoped to `cmd/seed-bank-reference`); the Archer doc records gaps without new
  product claims.
- Archer #30 (central consent management) is recorded as a gap, not implemented.

## Implementation streams

| Stream | Scope |
| --- | --- |
| A — Answer display | new `AnswerValueDisplay.tsx` + CSS reuse, integrate into `ResponseAssessment`, `CaptureReview`, `VendorResponseReview`, `SentFormsView`, their tests |
| B — KRI register | `source_records_ops.json` merge + `"response_per_record"`, `source_records.go` installer handling, `source_records_test.go` coverage |
| C — Fixture + evidence | `staticDemo.ts` program-responses typed fixture + endpoint branch/test; re-run Program captures 48-55; update `docs/quality/rendered-ui-evidence.md` |
| D — Archer doc | new `docs/product/archer-requirements-gap.md` |

## Proof requirements

- Typecheck, targeted vitest + axe, copy-quality regression, runtime-truth and
  UI-contract checks after integration; seed tests with `response_per_record`
  coverage (run when `CLEARSIGHT_SOURCE_MANIFEST_DIR` is configured).
- Rendered evidence at representative viewports (1440×900 desktop light/dark,
  390×844 mobile) covering the Branch KRI register response with typed answers and
  the capture review summary, per `docs/design/ui-delivery-workflow.md`.