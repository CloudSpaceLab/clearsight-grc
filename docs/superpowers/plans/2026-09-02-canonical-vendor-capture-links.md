# Canonical Vendor Capture Links Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ensure vendor registration and vendor-work emails contain canonical, purpose-bound form-distribution links that open until their recorded expiry, and remove these journeys from the unreleased invitation-token path.

**Architecture:** Vendor assessment and vendor-work collection will create exact form distributions, issue access routes, and open those distributions before sending email. Their workflow links will reference the issued access route and request together. No browser fallback or dual-token bridge will be added. Existing expired/unusable links will be revoked and replaced after deployment.

**Tech Stack:** Go 1.26, PostgreSQL 18/pgx, React/Vite capture client, SMTP STARTTLS, existing form-distribution/access-route services.

---

### Task 1: Freeze the production defect

- [x] Add an integration test that sends a vendor registration request and redeems the exact selector embedded in the delivered email through `DistributionAccessService`.
- [x] Assert the request belongs to an open distribution, the route is unrevoked and unexpired, and the legacy invitation table is not used.
- [x] Run the focused test and retain the expected failure before implementation.

### Task 2: Make exact distribution requests carry workflow context

- [x] Extend `CreateDistributionInput` with the governed request context needed by assessment/vendor-work flows: origin, why-selected text, known facts, and an exact scoped form contract.
- [x] Validate scoped sections/fields against the exact active form revision and persist them in both memory and PostgreSQL stores.
- [x] Preserve exact origin lookup and request reconstruction without creating a standalone request.
- [x] Add memory and PostgreSQL tests for origin idempotency, scoped fields, protected recipient state, and transaction rollback.

### Task 3: Replace assessment invitation issuance

- [x] Introduce a canonical external-dispatch service that prepares a distribution, issues/rotates its direct access route, opens the distribution, and returns only the one-time selector needed for delivery.
- [x] Update initial assessment send, clarification, and reissue to use the canonical dispatch result.
- [x] Change assessment request-link proof from `capture_invitations` to the route/recipient/request chain and update the schema foreign key.
- [x] Revoke the canonical route/distribution when finalization fails; do not add a compatibility endpoint.

### Task 4: Replace vendor-work invitation issuance

- [ ] Create vendor-work capture requests as canonical distributions for initial collection and requested changes.
- [ ] Reserve/finalize canonical route IDs, update route proof constraints, and rotate/revoke routes on retry, cancellation, or recipient change.
- [ ] Prove address-verification and certification-refresh email selectors redeem through the same public capture endpoints.

### Task 5: Remove the obsolete journey path

- [ ] Delete assessment/vendor-work tests and interfaces that require `IssueInvitation`/`RedeemInvitation` for these journeys. Assessment complete; vendor work remains.
- [ ] Add a regression scan that prevents these packages from calling the retired invitation issuer.
- [ ] Keep unrelated internal capture migration work out of this change unless required by a failing canonical test.

### Task 6: Verify, merge, deploy, and reissue

- [ ] Run focused Go tests, PostgreSQL integration tests, the full backend suite, web capture tests/typecheck/build, schema tests, copy-quality checks, and `git diff --check`.
- [ ] Review the exact diff for selectors, recipients, credentials, or full secure URLs; none may be committed or logged.
- [ ] Open and merge the PR, deploy the exact merge SHA, and confirm public readiness reports the same revision.
- [ ] Revoke the unusable registration invitation, issue a fresh canonical route, deliver a replacement email, and verify stored route/request/distribution state without exposing the selector.
- [ ] Record the redacted provider receipt and remaining manual inbox-click confirmation in issue #128.
