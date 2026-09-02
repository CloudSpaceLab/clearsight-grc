# Real Vendor Email and Link Journeys Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prove the registration/address-verification and ISO 27001/PCI DSS certification-refresh journeys through canonical stored records, protected links, review, outcome, sign-off, Matter closure, and the configured hosted SMTP relay.

**Architecture:** Keep `AssessmentRequestService`, `VendorWorkService`, `InvitationDeliveryService`, `SMTPDelivery`, the response workspace, staff assignment consumer, and continuity commands as the only delivery/workflow paths. Add one PostgreSQL journey acceptance suite using a recording delivery adapter, close any orchestration or UI gaps it exposes, strengthen redacted delivery/link assertions, then deploy the exact green commit and complete both received-inbox journeys manually through the hosted UI. Provider acceptance, inbox receipt, response receipt, bank acceptance, outcome, sign-off, and closure remain separate states.

**Tech Stack:** Go 1.26, pgx/PostgreSQL 18, React 19, TypeScript 7, Vitest 4, SMTP STARTTLS, existing outbox worker and recipient-access services.

---

## File map

- Add `internal/thirdparty/vendor_email_journey_postgres_integration_test.go` for both canonical stored journeys.
- Extend `internal/evidence/invitation_delivery_test.go`, `internal/evidence/smtp_delivery_test.go`, and `internal/evidence/distribution_access_postgres_integration_test.go` only where the combined journey exposes missing negative coverage.
- Extend `internal/workflow/assignment_notification_test.go` and `internal/workflow/assignment_notification_postgres_integration_test.go` for exact reassignment delivery behavior.
- Modify `internal/thirdparty/assessment_request.go`, `internal/thirdparty/vendor_work.go`, or their PostgreSQL repositories only if the failing acceptance test proves a domain gap.
- Extend `web/src/components/VendorDueDiligence.test.tsx` and `web/src/components/VendorWorkPanel.test.tsx`; create `web/src/components/MatterDetailsPanel.test.tsx` for the exact current-state/next-action contract.
- Extend `deploy/tests/deployment_config_test.py` and `deploy/scripts/verify-email-readiness.sh` for redacted exact-head readiness.
- Update `docs/quality/acceptance-tests.md` with the controlled hosted run receipt.

### Task 1: Freeze task-specific email and link contracts

- [ ] **Step 1: Add failing message-contract assertions**

Extend `internal/evidence/invitation_delivery_test.go` with a table covering all three messages used by the two journeys:

```go
tests := []struct {
	name, wantSubject, wantAction string
	kind InvitationMessageKind
}{
	{"registration", "Complete your vendor registration for Example Bank", "Complete registration", InvitationMessageVendorRegistration},
	{"address", "Verify the vendor address for Example Bank", "Provide address verification", InvitationMessageAddressVerification},
	{"certifications", "Provide current vendor certifications for Example Bank", "Provide certification evidence", InvitationMessageCertificationRefresh},
}
```

For each message assert:

- one `data-primary-action="true"` in HTML;
- the same exact secure URL once in plain text and once in HTML;
- task title, recipient role, deadline, link expiry, and recovery text in both alternatives;
- no remote image, tracking pixel, script, credential, OTP, selector, or other tenant record;
- certification copy names `ISO 27001` and `PCI DSS` separately and says applicability must be confirmed.

- [ ] **Step 2: Run the focused renderer tests**

Run: `go test ./internal/evidence -run 'Invitation.*Message|Protected.*Message' -count=1`

Expected: any missing parity or certification wording fails before implementation.

- [ ] **Step 3: Make the smallest renderer/context correction**

Use only `InvitationMessageContext` populated by `assessmentInvitationMessage` and `vendorWorkInvitationMessage`. Do not add journey-specific mail senders. Keep the established subject selection in `internal/evidence/invitation_delivery.go` and conservative HTML renderer.

- [ ] **Step 4: Verify SMTP transport ambiguity and header safety**

Extend `internal/evidence/smtp_delivery_test.go` only if absent to assert:

```go
if receipt.FailureCode != InvitationFailureOutcomeUnknown {
	t.Fatalf("post-DATA disconnect failure = %s", receipt.FailureCode)
}
```

and reject CR/LF in sender, recipient, subject, message ID, or generated header values. Run:

`go test ./internal/evidence -run 'SMTP|Invitation' -count=1`

- [ ] **Step 5: Commit the delivery contract**

```bash
git add internal/evidence internal/thirdparty/assessment_request.go internal/thirdparty/vendor_work.go
git commit -m "test: freeze vendor email delivery contracts"
```

### Task 2: Prove onboarding registration reaches stored review state

- [ ] **Step 1: Write the first half of the PostgreSQL journey test**

Create `internal/thirdparty/vendor_email_journey_postgres_integration_test.go` under `//go:build postgresintegration`. Seed a tenant, legal entity, verified compliance officer, vendor relationship, current authority routes, approved onboarding form, recipient keyring, and assessment through normal repository/service helpers.

Exercise:

1. `AssessmentService.Start` for `AssessmentReviewOnboarding`.
2. assessment setup maintenance until the review Matter and request exist.
3. `AssessmentRequestService.Send` with a recording `InvitationDelivery`.
4. redeem the exact captured link through `DistributionAccessService`/recipient access.
5. satisfy email verification and submit all required form fields.
6. publish the resulting `EvidenceResponseSubmitted` outbox event through `AssessmentConsumer`.
7. open bank review, accept the identity fields, and apply the response.

Assert stored state, not only returned values:

```go
if assessment.Status != AssessmentUnderReview { t.Fatalf("assessment status = %s", assessment.Status) }
if review.Response == nil || review.Response.SubmissionID == "" { t.Fatal("submitted response is absent from bank review") }
if receipt.ResultVendorVersion <= receipt.PriorVendorVersion { t.Fatalf("vendor versions = %d -> %d", receipt.PriorVendorVersion, receipt.ResultVendorVersion) }
if matter.Matter.Status == continuity.MatterClosed { t.Fatal("registration receipt closed the review Matter") }
```

- [ ] **Step 2: Run the test and confirm the first real gap**

Run:

`go test -count=1 -p 1 -tags "postgres postgresintegration" ./internal/thirdparty -run TestVendorRegistrationAddressAndCertificationEmailJourneys`

Expected: FAIL at the first missing stored transition, link, response, or authority condition.

- [ ] **Step 3: Fix only the proven orchestration gap**

Use existing commands and current authority evaluation. If registration completion does not leave a visible review Matter, repair assessment provisioning/linking; do not create a second Matter in an HTTP handler. If response application is incomplete, repair the assessment response application transaction and its append-only event/outbox record. Never derive actor or scope from the request body.

- [ ] **Step 4: Re-run through response application**

Run the focused PostgreSQL test again. Expected: the vendor identity and registration response are stored, the exact review Matter remains open, and no address result is implied yet.

- [ ] **Step 5: Commit the registration slice**

```bash
git add internal/thirdparty internal/evidence internal/continuity migrations
git commit -m "fix: complete stored vendor registration handoff"
```

### Task 3: Complete address verification, assignment, outcome, sign-off, and closure

- [ ] **Step 1: Extend the journey test with address work**

Continue the same test using the exact relationship and review Matter:

1. create or reuse a typed Matter relationship link;
2. `VendorWorkService.Prepare` with `VendorWorkAddressVerification` and the active `VENDOR-ADDRESS-VERIFICATION` form revision;
3. assign the Matter/action to the staff principal through the canonical owner/action command;
4. publish the assignment outbox event through `AssignmentNotificationConsumer`;
5. `VendorWorkService.Send` to the staff recipient and capture the purpose-bound invitation;
6. redeem that exact invitation, submit verification result, method, date, source, PDF artifact, and attestation;
7. publish submission through `VendorWorkConsumer`;
8. start review and accept the current response;
9. implement the linked action, add a verification contract, record an independent `PASS`, create/approve the response/sign-off record if required by the route, and close the Matter.

Assert each boundary independently:

```go
if work.State != VendorWorkAccepted || work.AcceptedAt == nil { t.Fatalf("work = %#v", work) }
if notification.Status != "DELIVERED" { t.Fatalf("assignment notification = %#v", notification) }
if len(matter.VerificationResults) != 1 || matter.VerificationResults[0].Result != continuity.VerificationPassed { t.Fatalf("outcomes = %#v", matter.VerificationResults) }
if matter.Matter.Status != continuity.MatterClosed || matter.Matter.ClosedAt == nil { t.Fatalf("matter = %#v", matter.Matter) }
if matter.Matter.OwnerPrincipalID == matter.VerificationResults[0].ReviewerPrincipalID { t.Fatal("assignee performed the independent outcome review") }
```

- [ ] **Step 2: Add reassignment and delivery idempotency cases**

Extend `internal/workflow/assignment_notification_test.go` to prove:

- the new current assignee receives exactly one notification;
- a superseded assignment is recorded without sending;
- replay of the same outbox event does not send twice;
- temporary SMTP failure retries delivery without rolling back assignment;
- ambiguous post-acceptance outcome is recorded and not blindly re-sent;
- possession of either email link grants no review, signatory, ownership, or closure authority.

- [ ] **Step 3: Add link lifecycle negative cases**

In the PostgreSQL journey fixture, issue replacement invitations and assert wrong audience/OTP, expiry, explicit revocation, replay, and old-session invalidation return the generic protected-access failure without task metadata. Query audit/receipt rows only by redacted IDs.

- [ ] **Step 4: Run the complete address slice**

Run:

```bash
go test ./internal/workflow ./internal/evidence -run 'AssignmentNotification|RecipientAccess|Invitation' -count=1
go test -count=1 -p 1 -tags "postgres postgresintegration" ./internal/thirdparty ./internal/workflow ./internal/evidence -run 'VendorRegistrationAddressAndCertificationEmailJourneys|AssignmentNotification|RecipientAccess'
```

Expected: PASS.

- [ ] **Step 5: Commit the address journey**

```bash
git add internal/thirdparty internal/workflow internal/evidence internal/continuity migrations
git commit -m "fix: complete address verification email journey"
```

### Task 4: Complete certification refresh with per-item review

- [ ] **Step 1: Extend the PostgreSQL journey test with certification work**

Use the already registered relationship and exact active `VENDOR-CERTIFICATION-REFRESH` form. Prepare and send `VendorWorkCertificationRefresh`, redeem the captured link, and submit:

- ISO 27001 applicability, current state, certificate metadata, labelled test PDF, and attestation;
- PCI DSS applicability, current state, evidence metadata, labelled test PDF, and attestation.

Assert the stored request kind, exact form version, link purpose, response revision, and artifacts.

- [ ] **Step 2: Prove targeted change handling**

Start review, accept the ISO item, request changes only for PCI DSS with an explicit field ID and rationale, send the replacement request, submit the corrected PCI artifact, and accept the current revision. Assert the accepted ISO evidence remains referenced and the superseded PCI response is reconstructable but not current.

```go
if !containsField(review.AcceptedFieldIDs, "iso_27001_evidence") { t.Fatal("accepted ISO evidence was discarded") }
if containsField(review.AcceptedFieldIDs, "pci_dss_evidence") { t.Fatal("rejected PCI evidence was accepted") }
if history.Items[0].Current == history.Items[1].Current { t.Fatal("response history has no single current revision") }
```

- [ ] **Step 3: Keep Matter outcome separate from work acceptance**

If the certification request targets a Matter, prove `VendorWorkAccepted` does not close it. Record current outcome, authorized sign-off, and closure separately. If policy says one certification is not applicable, preserve the recorded applicability basis rather than fabricating a certificate.

- [ ] **Step 4: Run both journeys twice**

Run the combined PostgreSQL journey test twice against a clean database, then repeat the idempotent outbox deliveries without cleaning. Expected: one current invitation per active request, one final delivery receipt per event, one current response revision, and no duplicate Matter/work reaction.

- [ ] **Step 5: Commit the certification journey**

```bash
git add internal/thirdparty internal/evidence internal/continuity migrations
git commit -m "fix: complete certification refresh email journey"
```

### Task 5: Make the bank and recipient UI state boundaries explicit

- [ ] **Step 1: Add failing bank-workspace tests**

Extend `VendorDueDiligence.test.tsx`, `VendorWorkPanel.test.tsx`, and `MatterDetailsPanel.test.tsx` to render these states:

- registration sent / response received / under review;
- `Address verification pending` before any finding conclusion;
- assigned staff and notification outcome shown separately;
- address response received, artifact unavailable/unscanned, accepted, outcome passed, signed off, resolved;
- certification ISO accepted while PCI DSS needs replacement;
- stale response/outcome after a newer response revision.

For each state assert one dominant enabled action and that enabled controls invoke a real API method.

- [ ] **Step 2: Run and confirm failures**

Run:

`cd web && npm test -- VendorDueDiligence.test.tsx VendorWorkPanel.test.tsx MatterDetailsPanel.test.tsx`

- [ ] **Step 3: Implement the minimal presentation changes**

Reuse shared notices, status chips, buttons, form fields, date inputs, and review rows. Required labels remain exactly distinct: `Submission received`, `Evidence accepted`, `Outcome passed`, `Signed off`, and `Resolved`. Do not call pending verification a deficiency.

- [ ] **Step 4: Verify accessibility and copy quality**

Run:

```bash
cd web
npm test -- VendorDueDiligence.test.tsx VendorWorkPanel.test.tsx MatterDetailsPanel.test.tsx copyQuality.test.ts
npm run typecheck
npm run check:ui-contracts
```

- [ ] **Step 5: Render all material states**

Capture desktop/mobile, light/dark, keyboard focus, and 200% reflow. Inspect email HTML/plain fixtures and recipient workspace states. Fix the highest-impact defect and re-render before committing.

- [ ] **Step 6: Commit the UI slice**

```bash
git add web DESIGN.md
git commit -m "fix: clarify vendor email journey states"
```

### Task 6: Strengthen redacted hosted readiness

- [ ] **Step 1: Add failing deployment safety tests**

Extend `deploy/tests/deployment_config_test.py` to prove the readiness script checks, without printing values:

- SMTP host/port/user/secret reference/from address/TLS mode;
- external delivery enabled;
- recipient keyring, active key, access HMAC key;
- secure capture public URL and allowed origin;
- API and worker exact release SHA;
- worker publisher/maintainer health;
- no recipient, credential, OTP, opaque selector, or secure URL in output.

- [ ] **Step 2: Run and confirm failure**

Run: `python -m unittest discover -s deploy/tests -p 'test*.py'`

- [ ] **Step 3: Update `verify-email-readiness.sh` with status-only output**

The script may output keys such as:

```text
smtp_configured=true
starttls_required=true
recipient_protection_configured=true
capture_origin_secure=true
api_revision_matches=true
worker_revision_matches=true
```

It must never echo the configured values.

- [ ] **Step 4: Run deployment checks and commit**

```bash
python -m unittest discover -s deploy/tests -p 'test*.py'
bash -n deploy/scripts/verify-email-readiness.sh
git add deploy
git commit -m "test: harden redacted email readiness"
```

### Task 7: Run exact-head automated verification

- [ ] **Step 1: Backend and database gates**

```bash
gofmt -w internal/thirdparty internal/evidence internal/workflow cmd/api cmd/worker
go test -race -coverprofile=coverage.out ./...
go test -tags postgres ./...
go test -count=1 -p 1 -tags "postgres postgresintegration" ./internal/...
go vet ./...
```

- [ ] **Step 2: Web and evidence gates**

```bash
cd web
npm run typecheck
npm test
npm run check:ui-contracts
npm run build
npm run review:ui
```

- [ ] **Step 3: Review the exact diff and secret exposure**

Run `git diff --check`, search changed files and generated evidence for recipient addresses, SMTP values, OTPs, full secure URLs, or opaque selectors, and verify `git status --short` contains only intended work.

### Task 8: Deploy and traverse both received-inbox journeys

- [ ] **Step 1: Merge only the exact green commit**

Push the branch, open the PR, wait for required backend/web/UI checks, and merge without including PR #129. Record the merge SHA.

- [ ] **Step 2: Deploy and verify exact revision**

Preserve protected configuration and active recipient keys. Deploy the merge SHA, confirm `/health/ready` and worker revision match it, then run the redacted readiness script.

- [ ] **Step 3: Execute registration/address journey in the hosted UI**

Create a fresh `Reference data` vendor, send registration to the approved vendor inbox, observe inbox receipt, traverse the exact received link, complete registration, create/assign address verification to the approved staff recipient, traverse the received purpose-bound link, submit test-labelled PDF evidence, review, pass the outcome, sign off, and close the Matter.

- [ ] **Step 4: Execute certification journey in the hosted UI**

Send certification refresh for the registered reference vendor, traverse the exact received link, submit ISO 27001 and PCI DSS answers/evidence, perform per-item review including one targeted replacement when safe, and verify the intended final Matter state.

- [ ] **Step 5: Record a redacted receipt**

In `docs/quality/acceptance-tests.md`, record only merge SHA, deployment timestamp, redacted message IDs/fingerprints, provider-accepted timestamps, human-observed receipt timestamps, stored aggregate IDs, final states, and remaining limitations. Do not record addresses, OTPs, invitation tokens, full URLs, or credentials.
