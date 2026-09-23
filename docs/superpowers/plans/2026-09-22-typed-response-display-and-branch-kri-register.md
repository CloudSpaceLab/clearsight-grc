# Implementation Plan — Typed response display + Branch KRI register + Archer gap analysis

Date: 2026-09-22
Authoritative scope: `docs/design/2026-09-22-typed-response-display-and-branch-kri-register-design.md` (approved).

## Goal

Three deliverables in one dependency-ordered plan:

1. **Stream A** — Render every submitted response answer with typed, human-readable formatting (numbers, currency, dates, badges, chips, evidence counts) across all answer consumers (`ResponseAssessment`, `CaptureReview`, `VendorResponseReview`) via one shared component. Seed fields stay `long_text`; typed reading is proven through the staticDemo fixture.
2. **Stream B** — Fix seeded Branch KRI duplication: the Excel sheet is one register form with one response *per branch*. Merge the 31 duplicated per-branch forms into one register form with per-record responses.
3. **Stream C** — Prove the typed reading in rendered UI evidence for the Branch KRI register (answers sheet) and the responses list, and re-capture Program evidence (existing captures 48–55 change; new captures 56/57).
4. **Stream D** — Cross-check the Archer requirements workbook (31 requirements) against ClearSight features and document gaps as a durable product doc.

Head-office KRI stays as-is (one register per function sheet). No new component families, design tokens, density modes or motion. No backend changes beyond `response_per_record` handling in the seed installer.

## Working-tree guardrails (read first)

- All changes stay **UNCOMMITTED**. The working tree holds pre-session vendor WIP (VendorsWorkspace/VendorPortfolio, DESIGN.md vendor entries, untracked pptx/evidence) and prior Programs evidence WIP that MUST NOT be reverted or committed separately.
- `source_records_ops.json` and `source_records_it_vendor.json` are gitignored; `.codex-tmp/` is untracked. Update generator + working JSON together; neither appears in git.
- Match the generator's exact serialization: `out.write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n', encoding='utf8')` via `Path.write_text`.
- Fresh demo DB is the clean path for Stream B idempotency (old per-branch keys differ from new register keys). Document as an operator step; no cleanup scaffolding.
- Known env limitation: `FormBuilder` reorder-block test (5000 ms timeout) passes isolated; other suite failures unrelated to this plan are out of scope.

## File structure

| File | Responsibility | Action |
| --- | --- | --- |
| `web/src/components/forms/AnswerValueDisplay.tsx` | Shared typed answer renderer | Create |
| `web/src/components/forms/AnswerValueDisplay.test.tsx` | Type/empty/axe coverage | Create |
| `web/src/components/forms/ResponseAssessment.tsx` | Swap `answerText` → component (answersOnly L90, review row L116) | Modify |
| `web/src/components/capture/CaptureReview.tsx` | Keep "No upload needed." semantic; route rest through component (L13) | Modify |
| `web/src/components/forms/VendorResponseReview.tsx` | Swap `answerValue` → component; drop local helper (L64/L77) | Modify |
| `cmd/seed-bank-reference/source_records.go` | `ResponsePerRecord` field + gates + register installer branch | Modify |
| `cmd/seed-bank-reference/source_records_test.go` | Register-aware contract test + synthetic register case | Modify |
| `cmd/seed-bank-reference/source_records_ops.json` | Merge 31 branch groups → one register group | Modify (gitignored) |
| `.codex-tmp/build_ops_source_records.py` | Generator emits merged register shape | Modify (untracked) |
| `.codex-tmp/merge_ops_register.py` | Surgical JSON merge | Create (untracked) |
| `web/src/staticDemo.ts` | Register responses in `programResponsesPopulation()` + `programResponseAssessment()` branch | Modify |
| `web/scripts/capture-program-review-evidence.mjs` | Assert register title in list; add register answers-sheet captures 56/57 | Modify |
| `docs/quality/rendered-ui-evidence.md` + `docs/evidence/2026-09-22-program-data-centered-ux/` | Update evidence records + manifest + PNGs | Modify |
| `docs/product/archer-requirements-gap.md` | 31-item gap analysis | Create |
| `docs/README.md` | Add gap doc line to product docs list | Modify |

---

## Stream A — Typed answer rendering (TDD)

### A1. Create `AnswerValueDisplay.tsx`

Consumes `type: string` + `answer` (structurally compatible `CaptureAnswerValue` for both governance and vendor answers) + optional `attachments`, `className`, `emptyLabel`, `evidenceEmptyLabel`, `onOpenDocuments`. Normalizes via `normalizeFieldType` from `../capture/contract` (handles lowercase form types AND uppercase vendor types like `LONG_TEXT`/`NUMBER`).

Rendering rules (per approved design):

- `short_text` / `email` / `url` / `telephone` → plain text.
- `long_text` → text with `pre-wrap` (+ className passthrough).
- `integer` / `decimal` / `percentage` → `new Intl.NumberFormat(undefined, { maximumFractionDigits: 1 })`; non-numeric falls back to raw text.
- `currency` → `new Intl.NumberFormat(undefined, { style: "currency", currency: "NGN" })`.
- `date` → `formatDate` helper (medium style, same as CaptureReview L55).
- `yes_no` → neutral `StatusBadge` "Yes"/"No" (from answer values/text).
- `checkbox` / `attestation` → "Confirmed" / "Not confirmed" (text === "true").
- `single_select` → option label (text or values[0]).
- `multi_select` → chip list of values (neutral `StatusBadge` chips — existing badge token, no new CSS).
- `file` / `photo` / `signature` / `vendor_document` → evidence family: artifact/document count + metadata; `onOpenDocuments` launcher button when provided and count > 0. Empty → `evidenceEmptyLabel ?? "No evidence submitted for this field."`.
- No answer / empty → `emptyLabel ?? "Not provided"`.
- Unknown normalized type → raw text or empty label.

Exports: `AnswerValueDisplay`, optional `formatDate` helper reuse.

### A2. Create `AnswerValueDisplay.test.tsx` first (TDD)

Vitest + axe (mirror `web/src/components/forms/ResponsesView.test.tsx` / `ProgramResponsesPanel.test.tsx` conventions). Cases:

- short_text/long_text/email/url/telephone plain render.
- long_text pre-wrap class/whitespace.
- integer `"1250000"` → `"1,250,000"`; decimal; percentage; non-numeric fallback.
- currency NGN rendering.
- date medium format (`2025-11-28` → human date).
- yes_no → neutral badge Yes/No; checkbox/attestation true/false.
- single_select label; multi_select chip list.
- file count + metadata + launcher onOpenDocuments; signature "Signed"; vendor_document metadata join (document_type · reference · expiry).
- empty states: no answer → default emptyLabel; evidence family → evidenceEmptyLabel; custom emptyLabel passed through.
- uppercase vendor type `LONG_TEXT`/`NUMBER` normalized.
- `axe.run` no violations.

### A3. `ResponseAssessment.tsx` swaps

- L90 answersOnly: replace `{answerText(item)}` inside the `<p className="response-assessment__answer">` with `<AnswerValueDisplay type={item.field.type} answer={item.answer} emptyLabel={evidenceField(item) ? "No evidence submitted for this field." : "No answer submitted for this field."} />`. Keep the `<p>` wrapper + `field-assessment.css` pre-wrap.
- L116 review row: same swap; keep the existing document `<dl>` and the sheet-level `showDocumentLauncher` flow (L130 "View submitted documents") untouched.
- If `answerText` (L148–155) becomes unreferenced, remove it; keep `evidenceField` and other helpers.

### A4. `CaptureReview.tsx` swap (L13)

- Keep `sourceLabel` small text and the "No upload needed." semantic (`documentAlreadyReceived(field) && !answerIsPresent(answer)`).
- Route the rest through `<AnswerValueDisplay type={field.type} answer={answers[field.id]} attachments={attachments[field.id]} />`.
- Remove `reviewValue` (L41–53) and `formatDate` (L55) if now unreferenced; keep `humanize` (used by known-facts rendering).

### A5. `VendorResponseReview.tsx` swap (L64/L77)

- Replace `{answerValue(answer)}` with `<AnswerValueDisplay type={answer.type} answer={answer.value} />` (vendor answer types uppercase — normalized by `normalizeFieldType`).
- Remove the local `answerValue` helper if unreferenced. Keep the `assuranceLabel` small text and existing flows.

### A6. Grep/update affected existing test expectations

Run before/after and fix raw-text/join assertions that now render formatted values:

```
grep -rn "answerText\|answerValue\|submitted document\|1250000\|attached ·" web/src --include=*.test.tsx
```

Affected candidates: ResponseAssessment/fieldAssessment tests, CaptureReview tests, VendorResponseReview/VendorDueDiligence tests, ProgramResponsesPanel.test.tsx. Update expectations to formatted output; do not weaken axe or copy assertions.

---

## Stream B — Branch KRI register seed (TDD)

### B1. Struct + gates in `source_records.go`

- Add `ResponsePerRecord bool \`json:"response_per_record"\`` to `sourceRecordGroup` (L57–67).
- Compact gate in BOTH `sourceCaptureParts` (L365) and `ensureSourceForm` (L402): `compact := len(group.Records) > 20 && !group.ResponsePerRecord`.

Register = 2 sections (source + record_0), 56 fields (source_context + 55) — within `formcontract.MaxSections=20` / `MaxFields=200`.

### B2. Contract test first: extend `source_records_test.go`

- In the manifest loop (L45–81): groups with `ResponsePerRecord` skip the `sourceCaptureParts` part loop. Instead assert: schema-once (form built from `Records[:1]`, params `{, Records[:1], 0, 1}`), fields ≤ MaxFields, sections ≤ MaxSections, per-record answers preserved keyed `r0_f%d` (skip-blank policy), and no parts split.
- Add a synthetic register case: group with >20 identical-label records + `ResponsePerRecord: true` → `sourceCaptureParts` returns one unit (no split), `ensureSourceForm` over `Records[:1]` within limits, per-record answers preserved.

### B3. Installer register branch in `installSourceRecords`

Add a branch before the parts loop (L188): when `group.ResponsePerRecord` (register):

1. Build the form once: `ensureSourceForm(ctx, ms, seed, programID, group, group.Records[:1], 0, 1)` — reuse its returned `form` and `source_context` answer.
2. For each `record` in `group.Records` (all share the record-0 schema — same 55 labels):
   - Build answers keyed `r0_f%d` (mirror the L419–431 skip-blank policy), reusing returned `source_context` value.
   - Idempotency key: `fmt.Sprintf("%s:%s:%s", sourceRecordPackage, group.Key, record.Key)`.
   - Receipt-query for existing distribution; else `distributions.Create` with `Title: form.Name + " · " + record.Title` (subject PROGRAM/programID, same access/recipients pattern as L198–212).
   - Submit as `COMPLETED_UNREVIEWED` (reuse `submitOperatingFormSample`/`validateOperatingFormSampleAnswers` pattern from L213–229).
   - `receipt.Captures++`; append `receipt.Items` with `{"group", "record", "form_id", "distribution_id", "program_id"}`.

Keep the matter/parts loop unchanged for non-register groups.

### B4. Merge the 31 branch groups in `source_records_ops.json`

`sourceCaptureParts`-based test must pass with the merged manifest. Steps:

1. `.codex-tmp/merge_ops_register.py`: assert exactly 31 groups with prefix `ops-branch-kri-r` at positions 0–30, identical `source_sha256` (`18149f11…`), identical 55-field labels; build one group `ops-branch-kri` (title "Branch KRI — November 2025", keep first group's program/file/sha/sheet/period/limitations) with `"response_per_record": true` and all 31 records concatenated; rewrite with `json.dumps(result, ensure_ascii=False, indent=2) + '\n'` via `Path.write_text(encoding='utf8')`.
2. Patch `.codex-tmp/build_ops_source_records.py` L51–57: emit one register group with `['response_per_record'] = True` and loop-appended records (record keys stay `ops-branch-kri-r{r}` so new idempotency keys are `fidelity-source-records-v1:ops-branch-kri:<record.Key>`).
3. Re-run B2 contract test against the merged manifest.

### B5. Verify

```
CLEARSIGHT_SOURCE_MANIFEST_DIR=cmd/seed-bank-reference go test -tags postgres ./cmd/seed-bank-reference/ -run TestSourceRecordCaptureContractAndITVendorCoverage -count=1
go vet ./cmd/seed-bank-reference/
```

---

## Stream C — Register typed proof in staticDemo + rendered evidence

### C1. `programResponsesPopulation()` (L1292)

Append register response(s) at the END (indexes [0]/[1] stay the annual/consent entries used elsewhere):

- id `response-program-branch-kri-cac-2025`, form `form-branch-kri-register`, title `Branch KRI — November 2025 · CAC`, subject PROGRAM/`program-ndpa`, revision 1, current, `completed_at` 2025-11-28, **no score** → ScoreCell shows honest "Score unavailable".
- Optional second branch (e.g. Marina) to prove per-branch response names in the list.

### C2. `programResponseAssessment()` (L1323)

Add a branch for the register id(s) returning NOT_REQUIRED (required_count 0, like consent's NOT_REQUIRED) with typed fields/answers to exercise every display family: short_text branch, currency `1250000`, integer, date `2025-11-28`, yes_no `Yes`, multi_select values, attestation `true`, long_text with line breaks. `automatic_score` stays undefined; verify ScoreCell handles undefined.

### C3. Response list gate + capture script

- `captureProgramEvidenceResults` (L216–221): add assertion that the register form title appears in the responses list (`Branch KRI — November 2025` with per-branch response names).
- Add a register answer-sheet capture (desktop light + mobile dark) as NEW captures 56/57, appended after 48–55 (52–54 stay on the annual response for documents/review flow).

### C4. Re-run evidence + docs

```
PAGE_URL=http://localhost:4173 UI_EVIDENCE_DIR=docs/evidence/2026-09-22-program-data-centered-ux node web/scripts/capture-program-review-evidence.mjs
```

Requires the preview server on :4173. Update `docs/quality/rendered-ui-evidence.md` (L134–143 covers 48–55) + manifest: add 56/57 rows (route `#programs/program-ndpa/evidence-results`, fixture `program-responses`, state "Program evidence sheet | Branch KRI register response with typed answers"; mobile dark entry for 57) and note the responses-list state now includes one Branch KRI register form title with per-branch response names.

---

## Stream D — Archer requirements gap analysis (docs only)

### D1. Create `docs/product/archer-requirements-gap.md`

- Header: source = `Fidelity Bank_Archer (1).xlsx` (31 requirements, dated 2026-09-22), labelled **reference data, not legal advice** (AGENTS.md bank-vertical rule).
- 31-item table: `# | Requirement (module) | ClearSight status | Use-case reference | Gap note`.
- Statuses: Covered / Partial / Blueprint / Gap, linked to use-case catalogue IDs (`UC-###`).
- Required gap notes (from design): loss register #15/#19, RCSA #16, BIA #18, incident log #20, consent #30, cadence reminders #2/#4/#9/#16, DPO approval node #23, policy lifecycle #6, tool integrations #31.
- Counts summary + method note (gap analysis uses catalogue + current reference journeys; does not promise implementation).

### D2. `docs/README.md`

Add the gap doc to the product docs list.

---

## Final verification (all streams)

1. `npm run typecheck` (web).
2. Targeted vitest: `AnswerValueDisplay`, affected consumers, `staticDemo.test.ts`, `ProgramResponsesPanel.test.tsx`, plus axe runs.
3. Copy-quality regression: `web/src/copyQuality.test.ts` (values are data, not copy — no new customer-facing sentences added by Stream A; evidence labels are existing strings).
4. Stream B Go tests (postgres tag + manifest dir) + `go vet`.
5. Render evidence at representative viewports (desktop light 1440×900, mobile dark 390×844, tablet 1024×768) and confirm new captures 56/57 + re-run 48–55 intact; inspect PNGs directly.
6. Confirm no horizontal overflow, no new design tokens/families, no backend changes outside scoped installer handling.
7. Confirm working tree remains uncommitted with pre-session WIP intact.