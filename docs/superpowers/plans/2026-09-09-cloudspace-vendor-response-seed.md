# Cloudspace Vendor Response Seed Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Supersede the user-confirmed legacy Cloudspace Technologies Ltd OEM request with a current submitted sample response from the supplied risk register.

**Architecture:** Extend the non-production PostgreSQL reference installer with an exact Cloudspace OEM resolver and typed operating-form answers. The installer uses governed distribution supersession only for the confirmed revision-3 request titled `Vendor security and privacy review`; the replacement's installer idempotency receipt and persisted supersession event resume a pending replacement after interruption. Repeat runs validate the immutable response rather than altering it.

**Tech Stack:** Go, PostgreSQL/pgx, evidence distribution and response-workspace services, GitHub Actions demo deployment.

---

## File structure

- Modify: `internal/bankverticals/install_vendor.go` — exact Cloudspace OEM relationship resolution.
- Modify: `internal/bankverticals/install_vendor_test.go` — direct-record reuse and ambiguity coverage.
- Modify: `cmd/seed-bank-reference/operating_form_samples.go` — typed Cloudspace response and unscored submitted state.
- Create: `cmd/seed-bank-reference/operating_form_samples_integration_test.go` — PostgreSQL integration coverage.
- Modify: `docs/acceptance/2026-09-09-risk-register-migration.md` — demo-response acceptance evidence.

### Task 1: Resolve the Cloudspace OEM relationship

**Files:**

- Modify: `internal/bankverticals/install_vendor.go`
- Modify: `internal/bankverticals/install_vendor_test.go`

- [ ] **Step 1: Write failing tests**

```go
func TestEnsureOperatingVendorReusesExactCloudspaceOEM(t *testing.T) {
	existing := createRelationship(t, "Cloudspace Technologies Ltd", "OEM")
	got, err := service.ensureOperatingVendor(ctx, seed, vendors, cloudspaceOEMSpec())
	if err != nil || got.Relationship.ID != existing.Relationship.ID {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}

func TestEnsureOperatingVendorRejectsAmbiguousCloudspaceOEM(t *testing.T) {
	createRelationship(t, "Cloudspace Technologies Ltd", "OEM")
	createRelationship(t, "Cloudspace Technologies Ltd", "OEM")
	if _, err := service.ensureOperatingVendor(ctx, seed, vendors, cloudspaceOEMSpec()); err == nil {
		t.Fatal("ambiguous relationship accepted")
	}
}
```

- [ ] **Step 2: Verify RED**

Run `go test ./internal/bankverticals -run 'TestEnsureOperatingVendor(ReusesExactCloudspaceOEM|RejectsAmbiguousCloudspaceOEM)' -count=1`.

Expected: FAIL because the current resolver only looks up the managed external reference.

- [ ] **Step 3: Implement the minimum resolver**

Add `cloudspaceOEMSpec()` with external reference `vendor:cloudspace-oem`. After managed-reference lookup, search the exact legal-name/service pair. Reuse one direct record without changing it, reject more than one match, and create the managed record only when no exact match exists.

```go
func cloudspaceOEMSpec() operatingVendorSpec {
	return operatingVendorSpec{externalRef: "vendor:cloudspace-oem", legalName: "Cloudspace Technologies Ltd", serviceName: "OEM", criticality: thirdparty.CriticalityStandard, privacyRole: thirdparty.PrivacyProcessor}
}
```

- [ ] **Step 4: Verify GREEN**

Run `go test ./internal/bankverticals -run 'TestEnsureOperatingVendor(ReusesExactCloudspaceOEM|RejectsAmbiguousCloudspaceOEM)' -count=1`.

Expected: PASS.

- [ ] **Step 5: Commit**

Run `git add internal/bankverticals/install_vendor.go internal/bankverticals/install_vendor_test.go; git commit -m "Reuse Cloudspace OEM demo relationship"`.

### Task 2: Seed the submitted response

**Files:**

- Modify: `cmd/seed-bank-reference/operating_form_samples.go`
- Create: `cmd/seed-bank-reference/operating_form_samples_test.go`

- [ ] **Step 1: Write the failing PostgreSQL integration test**

```go
func TestCloudspaceRiskRegisterSampleIsSubmittedAndRepeatSafe(t *testing.T) {
	first := installOperatingSamples(t)
	row := cloudspaceVendorForm(t, first)
	if row.ResponseState != "SUBMITTED" || row.RequiredReviews != 0 {
		t.Fatalf("row=%+v", row)
	}
	requireContains(t, responseAnswer(t, row, "assurance_gap"), "expired PCI-DSS certificate")
	requireContains(t, responseAnswer(t, row, "assurance_gap"), "31 March 2026")
	second := installOperatingSamples(t)
	if second.DistributionID != first.DistributionID || second.RevisionCount != 1 {
		t.Fatalf("repeat=%+v", second)
	}
}
```

- [ ] **Step 2: Verify RED**

Run `go test -count=1 -p 1 -tags 'postgres postgresintegration' ./cmd/seed-bank-reference -run TestCloudspaceRiskRegisterSampleIsSubmittedAndRepeatSafe`.

Expected: FAIL because no Cloudspace operating-form sample exists.

- [ ] **Step 3: Implement typed source-backed answers**

Change `operatingFormSampleSpec.answers` to `map[string]formcontract.AnswerValue`. Add state `COMPLETED_UNREVIEWED`, requiring one current response with no configured score. For the exact user-confirmed Cloudspace legacy request, use `SupersedeDistribution` to retain the earlier revision and create the current replacement before submitting the response. Include the active due-diligence answers below, keeping source gaps and the 31 March 2026 deadline in `assurance_gap`.

```go
answers: map[string]formcontract.AnswerValue{
	"contact_email": formcontract.TextAnswer("assurance@cloudspace.sample.invalid"),
	"service_description": formcontract.TextAnswer("Moneytor GetPaid application and payment terminal service provider work for POS Business."),
	"data_classes": {Values: []string{"Payment data"}},
	"subprocessors": formcontract.TextAnswer("No"),
	"security_framework": formcontract.TextAnswer("ISO 27001"),
	"assurance_available": formcontract.TextAnswer("No"),
	"assurance_gap": formcontract.TextAnswer("Sample register findings: ISO 27001/22301 assurance was not provided; VAPT was not provided; the SLA lacks a right-to-audit clause; the PCI-DSS certificate is expired. Target: 31 March 2026."),
	"authorized_attestation": formcontract.TextAnswer("true"),
}
```

Make repeat validation compare the persisted current answers with the specification and fail rather than altering a different or superseded response. Before creating anything, require both the installer receipt and recorded supersession event for the confirmed legacy request: resume its pending replacement or validate its submitted response. A user-owned replacement fails closed. With no confirmed legacy request, validate the receipt-backed standalone sample.

- [ ] **Step 4: Verify GREEN**

Run `go test -count=1 -p 1 -tags 'postgres postgresintegration' ./cmd/seed-bank-reference -run TestCloudspaceRiskRegisterSampleIsSubmittedAndRepeatSafe`.

Expected: PASS.

- [ ] **Step 5: Commit**

Run `git add cmd/seed-bank-reference/operating_form_samples.go cmd/seed-bank-reference/operating_form_samples_test.go; git commit -m "Seed Cloudspace risk register response"`.

### Task 3: Acceptance and release

**Files:**

- Modify: `docs/acceptance/2026-09-09-risk-register-migration.md`

- [ ] **Step 1: Record the acceptance outcome**

State that the Cloudspace OEM response is source-labelled, submitted and unscored, preserves the superseded legacy request, and includes no fabricated certificate or compliance conclusion.

- [ ] **Step 2: Run regression coverage**

Run `go test ./internal/bankverticals ./internal/documentimport ./cmd/api -count=1`.

Run `go test -count=1 -p 1 -tags 'postgres postgresintegration' ./cmd/seed-bank-reference`.

Run `cd web; npm run typecheck; npm test -- --run src/components/VendorsWorkspace.test.tsx`.

Expected: all commands pass.

- [ ] **Step 3: Commit and promote**

Run `git add docs/acceptance/2026-09-09-risk-register-migration.md; git commit -m "Document Cloudspace response acceptance"`.

Rebase on `origin/main`, publish a pull request, merge only after CI passes, and verify the `Deploy demo` workflow. The release script invokes `/clearsight-seed-bank-reference` for every hosted deployment.

## Plan self-review

- Spec coverage: Tasks 1–2 cover relationship preservation, source facts, submitted/unreviewed state, and idempotency; Task 3 covers acceptance and hosted release.
- Placeholder scan: no incomplete implementation markers remain.
- Type consistency: typed answers use `formcontract.AnswerValue`; submission remains at the existing workspace boundary.
