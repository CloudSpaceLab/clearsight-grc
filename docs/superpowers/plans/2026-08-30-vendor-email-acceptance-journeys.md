# Vendor Email Acceptance Journeys Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver and prove real hosted email journeys for vendor registration, staff address verification, compliance closure, and ISO 27001/PCI DSS certification refresh.

**Architecture:** Reuse the existing protected invitation delivery, email OTP, Vendor Assessment, Vendor Work, governed Form, and Matter lifecycles. Add one shared email-client-safe renderer and persist a bounded Vendor Work request kind so the same governed workflow can present and email the correct actor without granting authority through an inbox. Seed two reusable governed collection forms, add deterministic UI evidence, then configure the existing hosted SMTP adapter through protected server environment values and run a controlled live acceptance sequence.

**Tech Stack:** Go 1.25, PostgreSQL migrations, React 19/TypeScript, Vitest/Testing Library, Playwright UI evidence, Docker Compose, Bash deployment scripts, SMTP with STARTTLS.

---

## File map

- `internal/evidence/email_presentation.go`: shared inline-styled HTML/plain-text presentation for invitations and OTP messages.
- `internal/evidence/email_presentation_test.go`: escaping, CTA, metadata, redaction and parity tests.
- `internal/evidence/invitation_delivery.go`: protected invitation message context and default rendering boundary.
- `internal/evidence/invitation_delivery_test.go`: invitation defaults and protected-value regression tests.
- `internal/evidence/communication_render.go`: wraps governed communication nodes in the same email-safe shell.
- `internal/evidence/communications_test.go`: governed preview and secure-link presentation assertions.
- `internal/evidence/email_otp_delivery.go`: OTP content through the shared presentation.
- `internal/evidence/email_otp_delivery_test.go`: OTP expiry, escaping and delivery tests.
- `internal/evidence/smtp_delivery.go`: MIME headers and defensive message construction.
- `internal/evidence/smtp_delivery_test.go`: multipart, STARTTLS, authentication and header-injection checks.
- `migrations/000059_vendor_work_request_kind.up.sql`: bounded request-kind persistence with a backward-compatible default.
- `migrations/000059_vendor_work_request_kind.down.sql`: reversible schema removal.
- `internal/thirdparty/vendor_work.go`: request-kind types, validation, and invitation context.
- `internal/thirdparty/vendor_work_memory.go`: memory persistence parity.
- `internal/thirdparty/vendor_work_postgres.go`: PostgreSQL reads/writes for request kind.
- `internal/thirdparty/vendor_work_test.go`: state and delivery behavior by request kind.
- `internal/thirdparty/vendor_work_schema_test.go`: migration contract.
- `internal/httpapi/vendor_work_handlers_test.go`: API validation and verified-actor behavior.
- `internal/bankverticals/vendor_acceptance_forms.go`: address-verification and certification-refresh form contracts.
- `internal/bankverticals/vendor_acceptance_forms_test.go`: exact form field, condition and lifecycle assertions.
- `internal/bankverticals/install_service.go`: install both forms through the existing maker-checker seed path.
- `internal/bankverticals/install_test.go`: installation idempotency and activation assertions.
- `web/src/vendorWorkTypes.ts`: request-kind transport types.
- `web/src/components/VendorWorkPanel.tsx`: request-kind choice, actor-specific labels and state copy.
- `web/src/components/VendorWorkPanel.test.tsx`: address, certification, review and authorization copy/actions.
- `web/src/vendor-work.css`: responsive request-kind and status presentation.
- `web/src/staticDemo.ts`: deterministic, clearly sample-labelled visual-regression states only.
- `web/scripts/forms-evidence-scenarios.mjs`: representative desktop/mobile acceptance states.
- `web/scripts/forms-evidence-scenarios.nodecheck.mjs`: coverage assertions.
- `deploy/scripts/verify-email-readiness.sh`: redacted SMTP/recipient-security readiness checks.
- `deploy/tests/deployment_config_test.py`: deployment-script safety and secret-output regression tests.
- `.env.example`: documented variable names and safe descriptions without values.
- `docs/quality/acceptance-tests.md`: automated and controlled live-run acceptance contract.
- `docs/quality/rendered-ui-evidence.md`: rendered artifact index and scope statement.
- `docs/product/governed-forms.md`: external-recipient, evidence-review and acceptance semantics.
- `docs/product/use-case-catalogue.md`: vendor registration, address-verification and certification-refresh use cases.
- `docs/implementation-plan.md`: exact completion and remaining external dependencies.

### Task 1: Shared protected email presentation

**Files:**
- Create: `internal/evidence/email_presentation.go`
- Create: `internal/evidence/email_presentation_test.go`
- Modify: `internal/evidence/invitation_delivery.go`
- Modify: `internal/evidence/communication_render.go`
- Test: `internal/evidence/invitation_delivery_test.go`
- Test: `internal/evidence/communications_test.go`

- [ ] **Step 1: Write failing renderer tests**

Add table-driven tests that require one CTA, an escaped title, inline styles, a hidden preheader, due/expiry rows, a visible fallback URL, matching plain text and no remote image or tracking markup:

```go
func TestRenderProtectedEmailPresentation(t *testing.T) {
	message, err := renderEmailPresentation(emailPresentationInput{
		Preheader: "Address evidence is due 2 September.",
		Heading: "Verify Acme <Holdings> address",
		Intro: "Confirm the registered address and provide evidence.",
		ActionLabel: "Verify address", ActionURL: "https://forms.example.test/capture#form_access=protected",
		Facts: []emailFact{{Label: "Due", Value: "2 Sep 2026"}, {Label: "Link expires", Value: "1 Sep 2026"}},
		SupportContact: "vendor-risk@example.test",
	})
	if err != nil { t.Fatal(err) }
	if strings.Count(message.HTML, `data-primary-action="true"`) != 1 { t.Fatal("expected one primary action") }
	for _, required := range []string{"&lt;Holdings&gt;", "display:none", "Verify address", "Link expires", "word-break:break-all"} {
		if !strings.Contains(message.HTML, required) { t.Fatalf("missing %q", required) }
	}
	if strings.Contains(message.HTML, "<img") || strings.Contains(message.HTML, "http://") { t.Fatal("remote content is not allowed") }
}
```

- [ ] **Step 2: Run the focused tests and confirm failure**

Run: `go test ./internal/evidence -run 'TestRenderProtectedEmailPresentation|TestInvitationDeliveryBuildsDefaultMessage'`

Expected: FAIL because the presentation renderer and protected message context do not exist.

- [ ] **Step 3: Implement the minimal renderer and invitation context**

Define bounded inputs and outputs, escaping every value and validating the HTTPS CTA before rendering:

```go
type InvitationMessageKind string
const (
	InvitationMessageGeneric InvitationMessageKind = "FORM_REQUEST"
	InvitationMessageVendorRegistration InvitationMessageKind = "VENDOR_REGISTRATION"
	InvitationMessageAddressVerification InvitationMessageKind = "ADDRESS_VERIFICATION"
	InvitationMessageCertificationRefresh InvitationMessageKind = "CERTIFICATION_REFRESH"
)
type InvitationMessageContext struct {
	Kind InvitationMessageKind
	BankName, TaskTitle, TaskSummary, RecipientRole, SupportContact string
	DueAt, ExpiresAt time.Time
}
type emailFact struct { Label, Value string }
type emailPresentationInput struct {
	Preheader, Heading, Intro, BodyPlain, BodyHTML, ActionLabel, ActionURL, SupportContact string
	Facts []emailFact
}
type renderedEmailPresentation struct { PlainText, HTML string }
```

Add this protected, non-serialized field to `InvitationDeliveryRequest`:

```go
Message InvitationMessageContext `json:"-"`
```

`InvitationDeliveryService.Deliver` must normalize the protected address/link, populate `Subject`, `PlainText`, and `HTML` from `Message` only when callers did not supply governed content, and retain the current protected `String`/`GoString` behavior. `RenderCommunication` must place its already escaped governed nodes inside `BodyHTML` and `BodyPlain` in the same shell while preserving exactly one `primary-action` node and plain-text parity. Invalid kind, control characters, an invalid HTTPS action URL or overlong copy returns `ErrInvitationDeliveryRequestInvalid` before the adapter runs.

- [ ] **Step 4: Run focused tests**

Run: `go test ./internal/evidence -run 'TestRenderProtectedEmailPresentation|TestInvitationDelivery'`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/evidence/email_presentation.go internal/evidence/email_presentation_test.go internal/evidence/invitation_delivery.go internal/evidence/invitation_delivery_test.go internal/evidence/communication_render.go internal/evidence/communications_test.go
git commit -m "feat: add protected invitation email presentation"
```

### Task 2: Premium OTP and defensive MIME delivery

**Files:**
- Modify: `internal/evidence/email_otp_delivery.go`
- Modify: `internal/evidence/email_otp_delivery_test.go`
- Modify: `internal/evidence/smtp_delivery.go`
- Modify: `internal/evidence/smtp_delivery_test.go`

- [ ] **Step 1: Write failing OTP and MIME tests**

Require the OTP email to use the shared shell with one visually prominent code, an expiry fact, recovery copy and no action link. Extend SMTP tests to assert multipart/alternative ordering, CRLF headers, quoted-printable bodies, a safe `Auto-Submitted: auto-generated` header, and STARTTLS before authentication.

```go
func TestEmailOTPUsesProtectedPresentation(t *testing.T) {
	var delivered InvitationDeliveryRequest
	service := NewInvitationDeliveryService(invitationDeliveryFunc(func(_ context.Context, request InvitationDeliveryRequest) (InvitationDeliveryReceipt, error) {
		delivered = request
		return InvitationDeliveryReceipt{Status: InvitationDelivered}, nil
	}))
	adapter := NewEmailOTPDelivery(service)
	err := adapter.DeliverDistributionOTP(context.Background(), DistributionOTPDelivery{Address: "person@example.test", Code: "194207", ExpiresAt: time.Date(2026, 8, 30, 12, 5, 0, 0, time.UTC)})
	if err != nil { t.Fatal(err) }
	if strings.Count(delivered.HTML, "194207") != 1 || !strings.Contains(delivered.HTML, "12:05 UTC") { t.Fatal("OTP presentation is incomplete") }
}
```

- [ ] **Step 2: Run focused tests and confirm failure**

Run: `go test ./internal/evidence -run 'TestEmailOTP|TestSMTP'`

Expected: FAIL on the new presentation and MIME header assertions.

- [ ] **Step 3: Implement OTP rendering and MIME safeguards**

Render OTP through a dedicated `renderOTPEmail(code, expiresAt)` wrapper over the shared shell. Add the auto-generated header inside `buildSMTPMessage`; keep `From`, `To`, subject and message ID validation, and never add raw recipient or body content to receipts/errors. Retain TLS 1.2 minimum and the existing production restriction on plaintext SMTP.

- [ ] **Step 4: Run focused tests**

Run: `go test ./internal/evidence -run 'TestEmailOTP|TestSMTP'`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/evidence/email_otp_delivery.go internal/evidence/email_otp_delivery_test.go internal/evidence/smtp_delivery.go internal/evidence/smtp_delivery_test.go
git commit -m "feat: render secure OTP and SMTP messages"
```

### Task 3: Send task-specific assessment registration messages

**Files:**
- Modify: `internal/thirdparty/assessment_request.go`
- Modify: `internal/thirdparty/assessment_request_test.go`
- Modify: `internal/thirdparty/assessment_reissue.go`
- Modify: `internal/thirdparty/assessment_reissue_test.go`
- Modify: `internal/thirdparty/assessment_clarification.go`
- Modify: `internal/thirdparty/assessment_clarification_test.go`

- [ ] **Step 1: Write failing protected-message tests**

Capture the `InvitationDeliveryRequest` at initial send, replacement send and clarification. For an onboarding assessment, require `VENDOR_REGISTRATION`, the request title/purpose, vendor-contact role, response deadline and route expiry. For periodic/triggered assessments and clarifications, require `FORM_REQUEST` with the current task summary.

```go
if delivered.Message.Kind != evidence.InvitationMessageVendorRegistration {
	t.Fatalf("message kind = %q", delivered.Message.Kind)
}
if delivered.Message.RecipientRole != "Vendor contact" || !delivered.Message.DueAt.Equal(request.Deadline) {
	t.Fatalf("message context = %#v", delivered.Message)
}
```

- [ ] **Step 2: Run focused tests and confirm failure**

Run: `go test ./internal/thirdparty -run 'Assessment.*(Request|Reissue|Clarification).*Message'`

Expected: FAIL because assessment delivery currently supplies only recipient and link.

- [ ] **Step 3: Pass complete protected message context at every assessment send boundary**

Build the context from the stored assessment, evidence request and issued invitation route. Never derive actor or scope from the audience string. Initial onboarding uses heading `Complete your vendor registration`; periodic/triggered requests use the stored request title. Reissues preserve the original business task and use the new route expiry. Clarifications use the clarification message and selected-field request title.

- [ ] **Step 4: Run focused tests**

Run: `go test ./internal/thirdparty -run 'Assessment.*(Request|Reissue|Clarification)'`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/thirdparty/assessment_request.go internal/thirdparty/assessment_request_test.go internal/thirdparty/assessment_reissue.go internal/thirdparty/assessment_reissue_test.go internal/thirdparty/assessment_clarification.go internal/thirdparty/assessment_clarification_test.go
git commit -m "feat: send vendor registration email context"
```

### Task 4: Persist purpose-bound Vendor Work request kinds

**Files:**
- Create: `migrations/000059_vendor_work_request_kind.up.sql`
- Create: `migrations/000059_vendor_work_request_kind.down.sql`
- Modify: `internal/thirdparty/vendor_work.go`
- Modify: `internal/thirdparty/vendor_work_memory.go`
- Modify: `internal/thirdparty/vendor_work_postgres.go`
- Modify: `internal/thirdparty/vendor_work_test.go`
- Modify: `internal/thirdparty/vendor_work_schema_test.go`
- Test: `internal/httpapi/vendor_work_handlers_test.go`

- [ ] **Step 1: Write failing domain, API and schema tests**

Add exact acceptance for three values and rejection for unknown values:

```go
type VendorWorkRequestKind string
const (
	VendorWorkGeneral VendorWorkRequestKind = "GENERAL"
	VendorWorkAddressVerification VendorWorkRequestKind = "ADDRESS_VERIFICATION"
	VendorWorkCertificationRefresh VendorWorkRequestKind = "CERTIFICATION_REFRESH"
)

func TestPrepareVendorWorkRequiresKnownRequestKind(t *testing.T) {
	service, actor, input := vendorWorkFixture(t)
	input.RequestKind = "UNBOUNDED_ACTION"
	if _, err := service.Prepare(vendorWorkContext(actor), Actor{}, input); !errors.Is(err, ErrInvalid) { t.Fatalf("error = %v", err) }
}
```

Schema tests must assert `request_kind text NOT NULL DEFAULT 'GENERAL'` plus a check constraint containing all three values.

- [ ] **Step 2: Run focused tests and confirm failure**

Run: `go test ./internal/thirdparty ./internal/httpapi -run 'VendorWork.*Kind|VendorWork.*Invitation|VendorWorkSchema'`

Expected: FAIL because request kind is not part of the model or schema.

- [ ] **Step 3: Add the reversible migration and persistence**

Use a backward-compatible default and no data rewrite outside the new bounded column:

```sql
BEGIN;
ALTER TABLE third_party_work_requests
  ADD COLUMN request_kind text NOT NULL DEFAULT 'GENERAL'
  CHECK (request_kind IN ('GENERAL','ADDRESS_VERIFICATION','CERTIFICATION_REFRESH'));
COMMIT;
```

The down migration drops only `request_kind`. Add the field to prepare input, record, memory clone, PostgreSQL insert/select/scan, event payload and JSON response. Normalize an omitted value to `GENERAL`; reject any other value before authority or persistence calls.

- [ ] **Step 4: Build task-specific invitation context**

At initial send, retry and change-request delivery, pass an `evidence.InvitationMessageContext` derived from the persisted kind:

```go
func vendorWorkInvitationContext(work VendorWorkRequest, expiresAt time.Time) evidence.InvitationMessageContext {
	return evidence.InvitationMessageContext{
		Kind: invitationKindForVendorWork(work.RequestKind),
		TaskTitle: work.Purpose, TaskSummary: work.Instructions,
		RecipientRole: vendorWorkRecipientRole(work.RequestKind),
		DueAt: work.DueAt, ExpiresAt: expiresAt,
	}
}
```

`ADDRESS_VERIFICATION` maps to `Address verification staff contact`; it never alters `OwnerPrincipalID`, `ReviewerPrincipalID`, the verified request actor or Matter authority. `CERTIFICATION_REFRESH` maps to `Vendor contact`.

- [ ] **Step 5: Run migration and focused suites**

Run: `go test ./internal/thirdparty ./internal/httpapi`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add migrations/000059_vendor_work_request_kind.* internal/thirdparty/vendor_work*.go internal/httpapi/vendor_work_handlers_test.go
git commit -m "feat: distinguish vendor evidence request kinds"
```

### Task 5: Seed governed address and certification forms

**Files:**
- Create: `internal/bankverticals/vendor_acceptance_forms.go`
- Create: `internal/bankverticals/vendor_acceptance_forms_test.go`
- Modify: `internal/bankverticals/install_service.go`
- Modify: `internal/bankverticals/install_test.go`

- [ ] **Step 1: Write failing form-contract tests**

Require exact codes, customer-facing labels, document limits, conditions and maker-checker activation:

```go
func TestVendorAcceptanceFormContracts(t *testing.T) {
	address := vendorAddressVerificationFormInput("program-1", "entity-1")
	assertFieldIDs(t, address.Fields, "verification_result", "verification_method", "checked_on", "source_contact", "address_evidence", "staff_attestation")
	certs := vendorCertificationRefreshFormInput("program-1", "entity-1")
	assertFieldIDs(t, certs.Fields, "iso_applicable", "iso_certificate", "pci_applicable", "pci_attestation", "vendor_attestation")
}
```

The address result options are `Verified`, `Could not verify`, and `Different address found`. Both forms accept bounded PDF documents and include explicit attestation; PCI and ISO uploads are conditional on applicability being `yes`.

- [ ] **Step 2: Run tests and confirm failure**

Run: `go test ./internal/bankverticals -run 'VendorAcceptance|Install'`

Expected: FAIL because the contracts are absent.

- [ ] **Step 3: Implement idempotent maker-checker installation**

Create form codes `VENDOR-ADDRESS-VERIFICATION` and `VENDOR-CERTIFICATION-REFRESH`. Reuse a single helper that lists the exact code, creates only when absent, resumes interrupted draft/pending states, requires independent maker/checker activation and rejects an active non-current mismatch. Call it from `Install` after the existing due-diligence form.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/bankverticals`

Expected: PASS, including two consecutive installs producing no duplicate active forms.

- [ ] **Step 5: Commit**

```bash
git add internal/bankverticals/vendor_acceptance_forms.go internal/bankverticals/vendor_acceptance_forms_test.go internal/bankverticals/install_service.go internal/bankverticals/install_test.go
git commit -m "feat: seed vendor verification collection forms"
```

### Task 6: Present address and certification workflows in the bank UI

**Files:**
- Modify: `web/src/vendorWorkTypes.ts`
- Modify: `web/src/components/VendorWorkPanel.tsx`
- Modify: `web/src/components/VendorWorkPanel.test.tsx`
- Modify: `web/src/vendor-work.css`

- [ ] **Step 1: Write failing workflow tests**

Add tests that select each request kind and assert its actor-specific UI:

```tsx
it("prepares an address check for a staff evidence contact without changing Matter authority", async () => {
  render(<VendorWorkPanel targetType="MATTER" targetID="matter-1" />);
  await user.click(await screen.findByRole("button", { name: "Request evidence" }));
  await user.selectOptions(screen.getByLabelText("Request type"), "ADDRESS_VERIFICATION");
  expect(screen.getByLabelText("Address verification staff email")).toBeTruthy();
  expect(screen.getByText(/access to this evidence request only/i)).toBeTruthy();
  expect(screen.getByRole("button", { name: "Send address check" })).toBeTruthy();
});
```

Also assert certification copy names ISO 27001 and PCI DSS, accepted history says `Evidence accepted` rather than `Matter resolved`, unscanned documents disable acceptance, and only one dominant action is enabled in every active state.

- [ ] **Step 2: Run focused tests and confirm failure**

Run with Node 24: `node node_modules/vitest/vitest.mjs run src/components/VendorWorkPanel.test.tsx`

Expected: FAIL on the missing request-kind controls and copy.

- [ ] **Step 3: Implement the request-kind experience**

Add `request_kind` to transport types and a required select with these visible labels:

```tsx
<select aria-label="Request type" value={requestKind} onChange={...}>
  <option value="GENERAL">Vendor information or evidence</option>
  <option value="ADDRESS_VERIFICATION">Registered address verification</option>
  <option value="CERTIFICATION_REFRESH">ISO 27001 and PCI DSS evidence</option>
</select>
```

For address verification, label the recipient `Address verification staff email` and show: `This link permits a response to this evidence request only. Matter ownership, review and sign-off remain with the assigned bank roles.` For certification refresh, label the recipient `Vendor contact email` and explain that submission does not equal acceptance. Cards use `Address verification pending`, `Staff confirmation received`, `Certification evidence received`, `Evidence accepted`, and `Changes requested` based on kind and state.

Keep `Confirm acceptance` separate from Matter outcome/sign-off controls already rendered by the Matter workspace. Do not add a closure action to Vendor Work.

- [ ] **Step 4: Run focused tests, copy quality and typecheck**

Run:

```bash
node node_modules/vitest/vitest.mjs run src/components/VendorWorkPanel.test.tsx src/copyQuality.test.ts
node node_modules/typescript/bin/tsc -b --pretty false
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/vendorWorkTypes.ts web/src/components/VendorWorkPanel.tsx web/src/components/VendorWorkPanel.test.tsx web/src/vendor-work.css
git commit -m "feat: guide vendor verification requests"
```

### Task 7: Add deterministic rendered acceptance evidence

**Files:**
- Modify: `web/src/staticDemo.ts`
- Modify: `web/scripts/forms-evidence-scenarios.mjs`
- Modify: `web/scripts/forms-evidence-scenarios.nodecheck.mjs`
- Modify: `docs/quality/rendered-ui-evidence.md`

- [ ] **Step 1: Add failing scenario coverage assertions**

Require capability tags for `vendor-registration`, `address-pending`, `address-staff-response`, `address-review`, `matter-resolved`, `certification-request`, `certification-partial-review`, `email-desktop`, and `email-mobile`. Require both desktop and mobile coverage for the two recipient forms.

- [ ] **Step 2: Run the registry check and confirm failure**

Run: `node web/scripts/forms-evidence-scenarios.nodecheck.mjs`

Expected: FAIL listing the missing acceptance capabilities.

- [ ] **Step 3: Add clearly labelled visual-regression states**

Create deterministic sample-only states for layout review; they must not send email or stand in for the controlled hosted run. Use realistic owner/deadline/source labels and explicit `Sample acceptance record` text. Add representative desktop/light and mobile/dark scenarios to the registry with stable expected headings and dominant actions.

- [ ] **Step 4: Capture and inspect**

Run:

```bash
node web/scripts/forms-evidence-scenarios.nodecheck.mjs
node web/scripts/capture-forms-evidence.mjs
```

Expected: scenario validation and capture succeed with no horizontal overflow, clipped CTA, obscured focus or console error. Inspect every new PNG, correct the highest-impact visual defect, then rerun its capture.

- [ ] **Step 5: Commit**

```bash
git add web/src/staticDemo.ts web/scripts/forms-evidence-scenarios.mjs web/scripts/forms-evidence-scenarios.nodecheck.mjs docs/quality/rendered-ui-evidence.md
git commit -m "test: add vendor email journey visual evidence"
```

### Task 8: Add redacted hosted email readiness verification

**Files:**
- Create: `deploy/scripts/verify-email-readiness.sh`
- Modify: `deploy/tests/deployment_config_test.py`
- Modify: `.env.example`
- Modify: `deploy/scripts/verify-hosted-release.sh`

- [ ] **Step 1: Write failing deployment safety tests**

Assert the script checks only presence/shape of required configuration, requires STARTTLS, tests the configured TCP endpoint and certificate with `openssl s_client -starttls smtp`, and never prints environment values:

```python
def test_email_readiness_never_dumps_protected_values():
    script = Path("deploy/scripts/verify-email-readiness.sh").read_text()
    assert "env |" not in script
    assert "printenv" not in script
    assert "CLEARSIGHT_SMTP_PASSWORD" not in script
    assert "-starttls smtp" in script
```

- [ ] **Step 2: Run and confirm failure**

Run: `python -m unittest deploy.tests.deployment_config_test`

Expected: FAIL because the readiness script does not exist.

- [ ] **Step 3: Implement the redacted verifier**

The script receives the protected environment-file path, sources it without tracing, checks required variable names through indirect expansion, validates key material by base64-decoded length without output, requires `CLEARSIGHT_SMTP_TLS_MODE=STARTTLS`, and prints only named PASS/FAIL checks. It must never send a message. `verify-hosted-release.sh` invokes it only when `VERIFY_EMAIL_READINESS=true` is explicitly set during the acceptance release.

- [ ] **Step 4: Run deployment tests and shell syntax checks**

Run:

```bash
python -m unittest deploy.tests.deployment_config_test
bash -n deploy/scripts/verify-email-readiness.sh deploy/scripts/verify-hosted-release.sh
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add deploy/scripts/verify-email-readiness.sh deploy/tests/deployment_config_test.py deploy/scripts/verify-hosted-release.sh .env.example
git commit -m "ops: verify hosted email readiness safely"
```

### Task 9: Synchronize acceptance and product documentation

**Files:**
- Modify: `docs/quality/acceptance-tests.md`
- Modify: `docs/product/governed-forms.md`
- Modify: `docs/product/use-case-catalogue.md`
- Modify: `docs/implementation-plan.md`

- [ ] **Step 1: Add exact traceability**

Document the two journeys as ordered state tables. Each row names actor, verified identity boundary, material command, stored state, email action, dominant UI action and evidence needed to advance. State explicitly that staff email verification grants no Matter authority and that Vendor Work acceptance does not resolve a Matter.

- [ ] **Step 2: Record the remaining external dependencies**

Keep production object storage/malware inspection, bounce/complaint monitoring, sender-domain governance, authenticated staff notification channels, representative-user accessibility testing and the broader issue #80 lifecycle open. Describe hosted SMTP acceptance as relay acceptance plus observed inbox receipt, not universal deliverability.

- [ ] **Step 3: Run documentation and copy scans**

Run:

```bash
rg -n "bug[- ]free|fully compliant|automatically resolved|email means approved" docs web/src internal
git diff --check
```

Expected: no new prohibited completion claim and no whitespace errors.

- [ ] **Step 4: Commit**

```bash
git add docs/quality/acceptance-tests.md docs/product/governed-forms.md docs/product/use-case-catalogue.md docs/implementation-plan.md
git commit -m "docs: define vendor email journey acceptance"
```

### Task 10: Full local verification and branch review

**Files:**
- Modify only files required to correct failures found by the following gates.

- [ ] **Step 1: Run backend verification**

Run: `go test -p 1 ./...`

Expected: PASS.

- [ ] **Step 2: Run web verification with Node 24**

Run:

```bash
node node_modules/vitest/vitest.mjs run
node node_modules/typescript/bin/tsc -b --pretty false
node node_modules/vite/bin/vite.js build
```

Expected: all test files pass, typecheck passes and the production bundle builds.

- [ ] **Step 3: Run deployment and evidence checks**

Run:

```bash
python -m unittest deploy.tests.deployment_config_test
node web/scripts/forms-evidence-scenarios.nodecheck.mjs
git diff --check origin/main...HEAD
```

Expected: PASS.

- [ ] **Step 4: Review the branch diff**

Check that no credential, server address, PEM path, live recipient, OTP or opaque access value appears in tracked content:

```bash
git grep -n -E 'CLEARSIGHT_SMTP_PASSWORD=[^[:space:]]|BEGIN [A-Z ]*PRIVATE KEY|form_access=[A-Za-z0-9_-]{20,}' -- ':!docs/superpowers/plans/2026-08-30-vendor-email-acceptance-journeys.md'
git status --short
```

Expected: the secret scan returns no result and only intentional tracked changes are present.

- [ ] **Step 5: Commit verification corrections**

Commit only if a gate required code or documentation corrections, with a message naming the corrected contract.

### Task 11: Merge, configure and run controlled hosted acceptance

**Files:**
- No secret-bearing repository files.
- Server-only protected configuration under the existing deployment directory.
- Redacted acceptance evidence stored in the repository’s approved evidence location only after review.

- [ ] **Step 1: Push and open the pull request**

Push `codex/vendor-email-acceptance-20260830`, open a PR that links the approved design and implementation plan, and include exact local verification totals plus the explicit external remainder. Wait for required CI and review checks.

- [ ] **Step 2: Merge only after required checks pass**

Use the repository’s normal merge method. Confirm `origin/main` contains the merge and record its exact SHA. Do not deploy the feature branch directly.

- [ ] **Step 3: Apply protected server configuration**

Connect with the approved SSH identity, verify the resolved target host, make a root-readable timestamped backup of the existing protected environment file, and add/update the SMTP, capture base URL, delivery enablement, recipient keyring, active key ID and distribution HMAC settings. Generate missing AES/HMAC values on the server with `openssl rand -base64 32`. Disable shell tracing and command history expansion for the operation; never echo or inspect secret values.

- [ ] **Step 4: Deploy and verify the merged SHA**

Deploy through the existing immutable release workflow. Confirm API, worker and web readiness, exact revision, migrations, email readiness and worker health. On failure, restore the prior protected environment and release, preserving keys needed by existing recipient records.

- [ ] **Step 5: Run Journey 1 against real hosted records**

Use the supplied vendor test inbox only. Create a clearly labelled acceptance vendor, start onboarding, send the registration request, and record redacted provider/inbox timestamps. Complete the vendor response through the received link and OTP. In the resulting Vendor Review Matter, create `Registered address verification` with request kind `ADDRESS_VERIFICATION`, select the seeded address form, use the supplied staff test inbox, and send. Submit `Verified` plus the clearly labelled sample evidence through the staff email link and OTP. As the verified compliance officer, start review, inspect the evidence state, accept it, record the Matter verification outcome, sign off and resolve. Confirm the final Matter UI shows `Resolved`, outcome, signatory and evidence basis.

- [ ] **Step 6: Run Journey 2 against real hosted records**

Start a triggered reassessment for the same registered acceptance vendor so it receives its own canonical Vendor Review Matter. Create `Submit current certification evidence` with request kind `CERTIFICATION_REFRESH`, choose the seeded certification form and send to the supplied vendor test inbox. Through the real email link and OTP, submit clearly labelled sample ISO 27001 and PCI DSS documents and metadata. Prove a one-field change request/resubmission, inspect both current document states, accept the Vendor Work response, then record the Matter outcome/sign-off and resolve only when the verification contract permits it.

- [ ] **Step 7: Record redacted results and exact remainder**

Capture desktop/mobile hosted states without tokens, OTPs or full recipient addresses. Record message IDs only if the provider identifiers contain no protected value. State whether relay acceptance, inbox receipt, link exchange, OTP, response, inspection, acceptance, sign-off and closure each passed. List any unconfigured external dependency instead of weakening or bypassing its guard.
