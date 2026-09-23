# Third-party Semantic Capture Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the flattened Cloudspace register capture with a vendor-only response, governed internal field assessments, explicit assignments and a premium grouped review while preserving source history.

**Architecture:** Add a narrowly scoped semantic adapter for the third-party workbook group while retaining the generic source importer for IT and Ops tables. The adapter creates five vendor-response fields grouped by service and carries internal facts separately into existing Matters, Actions and response-assessment commands. The web review composes the current response, assessment state and existing linked vendor findings into a dense summary and progressive-disclosure rows.

**Tech Stack:** Go 1.26, PostgreSQL 18, existing Monitoring/Evidence/Continuity services, React 19, TypeScript, Vitest/Testing Library, existing ClearSight design tokens and release scripts.

---

### Task 1: Define and validate the third-party semantic mapping

**Files:**
- Create: `cmd/seed-bank-reference/source_third_party_semantics.go`
- Create: `cmd/seed-bank-reference/source_third_party_semantics_test.go`
- Modify: `cmd/seed-bank-reference/source_records.go`

- [ ] **Step 1: Write a failing semantic-classification test**

Build a synthetic two-service group containing the 16 workbook labels and assert that `thirdPartySemanticCapture` returns five vendor requirements, excludes `S/N`, `ASSESSOR`, `BUSINESS OWNER`, `SERVICE PROVIDER`, `RESPONSIBILITY`, `TIMELINE`, `STATUS` and ratings from form fields, and retains assessor/action/source references in internal metadata.

```go
capture, ok, err := thirdPartySemanticCapture(group)
if err != nil || !ok || len(capture.Requirements) != 5 { t.Fatalf("capture=%+v ok=%v err=%v", capture, ok, err) }
for _, forbidden := range []string{"ASSESSOR", "BUSINESS OWNER", "RESPONSIBILITY", "STATUS"} {
    if slices.Contains(capture.FormLabels(), forbidden) { t.Fatalf("internal column became a vendor field: %s", forbidden) }
}
```

- [ ] **Step 2: Run the focused test and verify the missing adapter fails**

Run: `go test -tags postgres ./cmd/seed-bank-reference -run TestThirdPartySemanticCapture -count=1`

Expected: FAIL because `thirdPartySemanticCapture` is undefined.

- [ ] **Step 3: Implement the semantic adapter**

Define focused types `thirdPartySemanticGroup`, `thirdPartyRequirement` and `thirdPartyInternalAssessment`. Match normalized workbook labels, carry forward merged service/assessor cells within each source serial-number group, require finding/recommendation/comment mappings, and reject unknown nonblank labels instead of flattening them.

```go
type thirdPartyRequirement struct {
    Key, Service, Finding, VendorResponse, EvidenceLabel, SourceRange string
    Assessor, BusinessOwner, Performer, SourceSeverity, SourceRating string
    AssessmentDate, Deadline, SourceStatus, Recommendation, Implication string
}
```

- [ ] **Step 4: Run the semantic test**

Run: `go test -tags postgres ./cmd/seed-bank-reference -run TestThirdPartySemanticCapture -count=1`

Expected: PASS with two service groups, five requirements and zero forbidden vendor fields.

- [ ] **Step 5: Commit the semantic boundary**

```powershell
git add cmd/seed-bank-reference/source_third_party_semantics.go cmd/seed-bank-reference/source_third_party_semantics_test.go cmd/seed-bank-reference/source_records.go
git commit -m "Model third-party register semantics"
```

### Task 2: Seed a corrected immutable response and internal reviews

**Files:**
- Modify: `cmd/seed-bank-reference/source_records.go`
- Create: `cmd/seed-bank-reference/source_third_party_install.go`
- Test: `cmd/seed-bank-reference/source_records_test.go`
- Test: `cmd/seed-bank-reference/source_third_party_semantics_test.go`

- [ ] **Step 1: Write a failing form-contract test**

Assert the corrected contract has service sections, five response fields, five evidence fields, manual review configuration and a new stable code/package version. Assert internal names never appear in field labels or submitted answers.

```go
form, answers, err := buildThirdPartySemanticForm(group)
if err != nil { t.Fatal(err) }
if got := len(form.Fields); got != 10 { t.Fatalf("fields=%d", got) }
if strings.Contains(marshal(form.Fields), "Blessing") || strings.Contains(marshal(answers), "Hakeem") { t.Fatal("assignment leaked into vendor response") }
```

- [ ] **Step 2: Run the focused contract test and verify it fails**

Run: `go test -tags postgres ./cmd/seed-bank-reference -run 'TestThirdPartySemantic(Form|Install)' -count=1`

Expected: FAIL because the current form still contains every spreadsheet column.

- [ ] **Step 3: Build the replacement form contract**

Use code prefix `SOURCE-TPR-V3-`, no automatic vendor score, Wizard presentation and one section per source service. Each requirement gets a long-text vendor response plus an optional `vendor_document` evidence field. Configure manual bank assessment on the response requirement with reviewer responsibility `REVIEWER` and outcomes `SATISFACTORY` (0), `FOLLOW_UP` (50) and `MATERIAL_CONCERN` (100).

- [ ] **Step 4: Install the replacement through normal services**

Create an idempotent distribution key under `fidelity-source-records-v3`, submit only the workbook's vendor comments, and use ordinary response-assessment commands to record source-labelled historical bank decisions where supported. Link the response context to the existing five Matters/Actions. Mark the v1 flattened and failed v2 distributions historical/revoked only after all replacement records exist; never rewrite their response revisions.

- [ ] **Step 5: Add retry and identity assertions**

Assert a second run reuses the same form/distribution/response, rejects a changed digest or contract, preserves five existing Matter IDs and five Action IDs, and resolves Blessing/Joel/Hakeem via existing principals without granting roles.

- [ ] **Step 6: Run the focused importer checks**

Run: `go test -tags postgres ./cmd/seed-bank-reference -run 'TestThirdPartySemantic|TestSourceRecordCaptureContract' -count=1`

Expected: PASS; private-manifest coverage runs only when `CLEARSIGHT_SOURCE_MANIFEST_DIR` is set.

- [ ] **Step 7: Commit the corrected installer**

```powershell
git add cmd/seed-bank-reference/source_records.go cmd/seed-bank-reference/source_records_test.go cmd/seed-bank-reference/source_third_party_install.go cmd/seed-bank-reference/source_third_party_semantics_test.go
git commit -m "Seed semantic vendor responses and reviews"
```

### Task 3: Compose a premium response-review model

**Files:**
- Create: `web/src/components/forms/semanticResponseReview.ts`
- Create: `web/src/components/forms/semanticResponseReview.test.ts`
- Modify: `web/src/components/forms/ResponseAssessment.tsx`
- Modify: `web/src/components/forms/ResponsesView.tsx`

- [ ] **Step 1: Write failing presentation-model tests**

Create five field fixtures spanning two services and assert the model returns service groups, four exact metrics, reviewer state, evidence state and no `Existing form rules` label for unconfigured fields.

```ts
expect(buildSemanticResponseReview(fields)).toMatchObject({
  metrics: { requirements: 5, answered: 5, evidenceReceived: 0, awaitingReview: 5 },
  services: [{ name: "Moneytor GetPaid application" }, { name: "Payment Terminal Service Provider (PTSP)" }],
});
expect(fieldAssessmentLabel(unconfiguredField)).toBe("");
```

- [ ] **Step 2: Run the tests and verify the old flat model fails**

Run: `npm test --workdir web -- --run src/components/forms/semanticResponseReview.test.ts src/components/forms/ResponsesView.test.tsx`

Expected: FAIL because the semantic model and grouped review do not exist.

- [ ] **Step 3: Implement the pure presentation model**

Parse stable field IDs and section metadata into `ResponseServiceGroup`, `ResponseRequirement` and `ResponseReviewMetrics`. Counts come only from returned response/assessment data; absent evidence and unavailable review state remain explicit.

- [ ] **Step 4: Replace repeated cards with grouped disclosures**

In answer-only mode, render a compact metric strip followed by service sections. Each requirement summary contains requirement text, vendor response excerpt, evidence badge and bank-review badge. Use native `details/summary` for progressive disclosure and preserve full answer text inside.

- [ ] **Step 5: Separate internal assessment and provenance**

Keep assessment controls inside the Review tab, headed **Internal assessment**. Show assessor attribution from recorded decisions, linked findings/actions through the vendor context, and move source identifiers into a collapsed **Source details** region. Do not display implementation labels for fields with no assessment policy.

- [ ] **Step 6: Run focused component tests**

Run: `npm test --workdir web -- --run src/components/forms/semanticResponseReview.test.ts src/components/forms/ResponsesView.test.tsx src/components/forms/ResponsesView.test.tsx`

Expected: PASS for semantic grouping, truthful counts, keyboard-accessible disclosures and unavailable states.

- [ ] **Step 7: Commit the response model**

```powershell
git add web/src/components/forms/semanticResponseReview.ts web/src/components/forms/semanticResponseReview.test.ts web/src/components/forms/ResponseAssessment.tsx web/src/components/forms/ResponsesView.tsx
git commit -m "Group vendor responses by service and review state"
```

### Task 4: Apply the premium responsive treatment

**Files:**
- Modify: `web/src/components/forms/field-assessment.css`
- Modify: `web/src/components/forms/responses-view.css`
- Modify: `DESIGN.md`
- Test: `web/src/components/forms/ResponsesView.test.tsx`

- [ ] **Step 1: Add failing structure and accessibility assertions**

Assert the review exposes named metric groups, service navigation, native disclosure controls, distinct `Vendor response` and `Internal assessment` regions, and direct missing-evidence text.

- [ ] **Step 2: Run the affected UI test**

Run: `npm test --workdir web -- --run src/components/forms/ResponsesView.test.tsx`

Expected: FAIL until the premium structure exists.

- [ ] **Step 3: Implement token-based layout and states**

Add a sticky summary, four-column metric strip, compact service navigation and bordered disclosure rows using existing `--cs-*` tokens. Use 150–200ms colour/border transitions only, visible focus rings and no layout-shifting transforms.

- [ ] **Step 4: Define narrow-screen replacement**

At 700px and below, use two metric columns, horizontal service navigation and stacked requirement summaries; at 420px, use one metric column only when labels would truncate. Ensure the sheet has no horizontal overflow.

- [ ] **Step 5: Record the component pattern**

Update `DESIGN.md` with the semantic response-review hierarchy, progressive-disclosure rule, truthful metric sources and responsive replacement behaviour.

- [ ] **Step 6: Run focused UI, copy and build checks**

Run:

```powershell
npm test --workdir web -- --run src/components/forms/semanticResponseReview.test.ts src/components/forms/ResponsesView.test.tsx src/copyQuality.test.ts
npm run typecheck --workdir web
npm run build --workdir web
```

Expected: all selected tests pass, TypeScript exits 0 and the production build completes.

- [ ] **Step 7: Commit the responsive review**

```powershell
git add web/src/components/forms/field-assessment.css web/src/components/forms/responses-view.css web/src/components/forms/ResponsesView.test.tsx DESIGN.md
git commit -m "Polish semantic vendor response review"
```

### Task 5: Reconcile and deploy the demo replacement

**Files:**
- Modify: `docs/engineering/source-record-demo-installation.md`
- Modify: `docs/acceptance/2026-09-10-demo-vendor-curation.md`
- Runtime-only: private source manifests and operator receipt under `/opt/clearsight-grc/state/source-records-20260910/`

- [ ] **Step 1: Document the new semantic contract and rollback**

Record the v2 mapping, immutable v1 history, idempotency identities, named-person resolution, source-digest guard and backup path without committing private workbook contents.

- [ ] **Step 2: Build only affected deployable artifacts**

Run:

```powershell
$env:GOOS='linux'; $env:GOARCH='amd64'; $env:CGO_ENABLED='0'
go build -tags postgres -trimpath -ldflags='-s -w' -o .codex-tmp/source-records-seed ./cmd/seed-bank-reference
npm run build --workdir web
```

Expected: seed binary and web distribution build successfully.

- [ ] **Step 3: Back up and run the scoped installer**

Confirm the existing source backup digest, upload the new seed binary, and run `-source-records-only -source-manifest-dir <private-directory>` with the canonical non-production tenant/entity and verified actors. The receipt must retain 86 groups, 3,427 source rows and 40 Matters while adding one v2 current third-party response and no duplicate Matter/Action IDs.

- [ ] **Step 4: Deploy the exact merged revision**

Build the API/worker/web release with the merged Git SHA, run the normal release script and require API, worker and web revision/readiness checks to pass.

- [ ] **Step 5: Perform the focused hosted acceptance**

Verify through API/UI:

- one current Cloudspace semantic response;
- five vendor requirements in two service groups;
- no internal-role columns among answer fields;
- five existing findings and Actions;
- Blessing/Joel internal reviewer attribution and Hakeem Action assignment;
- correct stored metrics and zero routing gaps;
- representative 1440px and 390px light/dark renders.

- [ ] **Step 6: Commit public documentation and evidence only**

Do not commit private manifests, source-bearing screenshots or workbook data. Commit the acceptance record and synthetic UI fixture renders only.

```powershell
git add docs/engineering/source-record-demo-installation.md docs/acceptance/2026-09-10-demo-vendor-curation.md docs/evidence/2026-09-10-semantic-response-review
git commit -m "Document semantic vendor response deployment"
```
