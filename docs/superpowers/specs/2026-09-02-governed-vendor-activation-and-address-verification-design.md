# Governed Vendor Activation and Address Verification Design

**Date:** 2026-09-02  
**Status:** Approved design awaiting implementation planning  
**Issues:** #80, #128  
**Depends on:** vendor relationships and assessments in `internal/thirdparty`; canonical Matters, Actions, Decisions and verification in `internal/continuity`; shared Evidence Requests, capture and notification delivery; verified authority and runtime foundations

## 1. Outcome

ClearSight will carry a proposed vendor relationship from completed onboarding evidence through staff address verification, independent officer sign-off and a separate fail-closed activation decision. Only an active relationship may receive a certification-refresh request.

The workflow must prove five distinct outcomes:

1. the vendor submitted onboarding information;
2. assigned staff completed the address check and supplied evidence;
3. an independently authorized officer confirmed the outcome;
4. a currently authorized actor activated the relationship against the exact effective policy and current evidence;
5. the active vendor received and completed a governed certification request.

Submission, upload, Action implementation, outcome verification, Matter closure and relationship activation remain separate records.

## 2. Non-goals

This tranche does not implement contract/SLA administration, periodic scheduling, source-driven reassessment, restriction, suspension, termination, verified exit, cross-relationship document reuse or legacy-register import. It does not create a vendor-specific task, notification, approval, artifact or questionnaire subsystem.

No unreleased compatibility route or alternate address-verification path will be retained. No database row may be patched directly to create an active acceptance-test vendor.

## 3. Canonical ownership

| Concern | Authoritative boundary |
| --- | --- |
| vendor organization and service relationship status | `internal/thirdparty` |
| onboarding evidence and assessment conclusion | third-party assessment plus shared Evidence Request |
| address-verification issue, assignment, sign-off and closure | canonical Matter, Action and verification records |
| internal staff response | shared internal Evidence Request and capture workspace |
| relationship activation criteria | effective versioned third-party activation policy |
| activation authority | current authority route for the relationship and command |
| certification collection | external vendor work using the shared form-distribution and capture boundary |
| email delivery | shared protected notification delivery with purpose-bound access routes |

The existing externally addressed `ADDRESS_VERIFICATION` Vendor Work path is unreleased and duplicates internal assignment semantics. It will be removed after the canonical internal Action/Evidence flow is live. `CERTIFICATION_REFRESH` remains vendor-facing Vendor Work.

## 4. Effective activation policy

A legal-entity-scoped `ThirdPartyActivationPolicy` controls activation eligibility. Each immutable version contains:

- allowed assessment conclusions;
- maximum assessment age;
- required current Decision types;
- whether address verification is required;
- Matter types and states that block activation;
- whether conditional assessment conclusions require recorded conditions;
- effective-from and optional effective-until timestamps.

Policy lifecycle is `DRAFT → PENDING_APPROVAL → ACTIVE → RETIRED`. A maker proposes a version and a different current checker approves it. Simulation reports affected proposed relationships, missing gates, conflicts and the policy version that would be applied. Activation of a new policy version effective-dates the prior version out atomically. Rollback creates and independently approves a new version; history is never rewritten.

There is at most one active policy for a legal entity at an instant. Missing, overlapping, expired or unreadable policy fails closed.

The hosted non-production environment may install a clearly labelled reference policy through the same propose/approve commands. The installer must not insert an active policy or active vendor row directly.

## 5. Address-verification workflow

### 5.1 Trigger and setup

When the current onboarding Evidence Request is submitted, the assessment reaction atomically records the submitted assessment state, its event/outbox fact and one deduplicated `third-party-address-verification-setup` maintenance job. The job key is stable for tenant, legal entity and assessment.

The worker creates or reuses one `VENDOR_ADDRESS_VERIFICATION` Matter linked to the assessment and relationship. It contains:

- an Action, **Verify the vendor registered address**, initially unassigned;
- the current registered address and vendor/service identifiers as known facts;
- an active verification contract requiring a supported address result and an available evidence artifact;
- a due date bounded by the onboarding review date;
- the relationship owner as Matter owner, without assigning that owner as the verifier.

Partial setup is visible and retryable. Repeated events or jobs cannot create duplicate Matters, Actions, contracts or requests.

### 5.2 One assignee control

The relationship owner, current Action assignee, or a valid manager in the reporting chain may use one **Assign verifier** control. Candidate selection comes from the current performer route and reporting hierarchy; it does not grant review, authorization or signing authority.

The compound assignment command re-evaluates verified identity, legal-entity scope, current Matter/Action versions, current assignee, manager relationship, candidate eligibility, delegation, absence and revocation. In one transaction it:

- changes the canonical Action owner;
- creates or reassigns the internal Evidence Request for the exact active address-verification form revision;
- revokes prior access when the recipient changes;
- appends reconstructable Matter and Evidence events;
- records the protected assignee-notification outbox entry.

The email contains a normal authenticated application deep link, not a bearer invitation token. It names the vendor, due date and requested result, and provides one primary action: **Provide address verification**.

### 5.3 Staff completion and officer sign-off

Assigned staff open the request from Today or the email, record the result, method, date and source, upload the required PDF evidence and attest to the check. Submission records the response and moves the Action to `IMPLEMENTED`; it does not pass the verification contract or close the Matter.

An actor selected by the current reviewer/authorizer route reviews the exact response and artifact. The Action performer cannot sign off their own work. The officer may:

- **Confirm verified address**, recording a `PASSED` verification result with evidence references and rationale; or
- **Request another check**, recording the failed/insufficient outcome and returning the Action and Evidence Request through the governed change path.

The Matter is closable only after the current verification contract has a passing result, the Action is implemented and no required review remains. Closure remains an explicit authorized command. The UI shows **Resolved** only after the closure event commits.

## 6. Relationship activation

`thirdparty.relationship.activate` is a material command with `AUTHORIZER` responsibility and high materiality. The command accepts only relationship and expected-version identifiers plus rationale and intended effective time; actor, tenant, legal entity, policy, assessment, Decisions and evidence are resolved from current trusted state.

Before changing status, it locks and re-evaluates:

- relationship status is `PROPOSED` or `UNDER_REVIEW` and the expected version is current;
- exactly one effective activation policy is current;
- the current onboarding assessment is `COMPLETED`, within the policy age limit and has an allowed conclusion;
- every policy-required Decision exists on the canonical vendor-review Matter, is approved, current, unexpired and made by the correct authority;
- every policy-required address-verification Matter is closed with a current passing verification result;
- no policy-defined blocking deficiency or unresolved contradiction remains;
- the current activation authority route is available and allows the verified actor;
- maker/checker, delegation, conflict, absence and revocation constraints still hold.

Success atomically updates the relationship to `ACTIVE`, records `effective_from`, increments its version, appends a safe third-party event, emits an outbox fact and schedules required projection maintenance. The activation receipt records the policy version, assessment version, Decision IDs, verification result IDs and relationship version without copying protected evidence.

Any missing or stale dependency returns a structured eligibility result and commits nothing. A derived projection failure after commit cannot turn a successful command into a reported failure.

## 7. Certification refresh

Certification refresh remains external Vendor Work and uses the active `VENDOR-CERTIFICATION-REFRESH` form. Preparation and send both re-read the relationship and reject any status other than `ACTIVE`, even if an earlier screen cached a different state.

The external request keeps ISO 27001 and PCI DSS applicability separate. Each applicable document must be current, independently reviewable and tied to the exact relationship and request revision. The protected email contains one purpose-bound link and one primary action: **Provide certification evidence**.

Vendor submission moves the work to response received. Bank acceptance is distinct from upload and does not silently change the relationship, Program, Matter or assessment. Expiry, recipient change, revocation and replay use the canonical distribution route and remain fail closed. The route, protected recipient and exact request are bound before an assessment link is recorded; the browser does not translate or fall back to an older token type.

## 8. UI and copy

The selected vendor relationship remains the single workspace; no second vendor dashboard is introduced.

The dominant action changes with current state:

| State | Dominant action |
| --- | --- |
| onboarding request outstanding | Review request status |
| address-verification setup incomplete | Retry address-verification setup |
| address Action unassigned | Assign verifier |
| assigned staff | Provide address verification |
| evidence submitted | Review address evidence |
| address outcome passed, Matter open | Close address-verification issue |
| activation gates satisfied | Activate vendor relationship |
| active relationship | Request certifications |

The activation panel shows the exact policy version and each satisfied or missing gate. A disabled activation control explains the missing record and links to the next valid action. Internal codes remain in API/history detail; primary UI uses **Vendor registration**, **Address verification**, **Activate vendor relationship** and **Request certifications**.

Desktop may show relationship context beside the focused action. Tablet collapses secondary context below the action. Mobile uses a full-screen focused flow. Light/dark, compact/comfortable, 200% reflow, keyboard, focus, reduced-motion and axe checks are required. No select, modal or side panel may shift the main layout when opened.

## 9. Failure and recovery

- Setup jobs are leased, bounded, observable and idempotent.
- Missing identity, policy, authority, relationship, assessment, Decision, Matter, response or evidence fails closed.
- Stale relationship, Matter, Action, request or policy versions return conflict and require reload.
- Assignment failure commits no owner change, request reassignment or notification claim.
- Notification failure preserves the committed assignment and reports **Assigned; email not sent** with a retry action.
- Successful SMTP acceptance is not presented as confirmed inbox reading.
- Invitation tokens, raw recipients and message bodies never enter logs, analytics, events, jobs, previews or acceptance documentation.
- Reassignment revokes prior request access before the replacement recipient can act.
- The system remains usable through Today and the vendor workspace when email delivery is unavailable.

## 10. Data, performance and reconstruction

New policy, activation receipt and setup-job tables use tenant/legal-entity composite ownership, immutable versions, bounded status/effective-time indexes and explicit retention. Exact relationship, assessment, Matter, Action and Evidence identifiers are used; no broad population replay locates known journey records.

Material writes share transactions with their authoritative event, outbox fact and required maintenance job. Reads use keyset pagination where populations can grow. Point-in-time reconstruction must show the effective policy, assessment, Decisions, address result and actor route that supported activation.

## 11. Acceptance

The tranche is complete only when automated and hosted acceptance prove:

1. a compliance officer creates a proposed vendor and sends a real registration email;
2. the vendor opens the purpose-bound link and submits onboarding information;
3. one address-verification Matter is created from stored submission state;
4. assignment/reassignment sends the correct staff notification and invalidates prior access;
5. staff submit the exact address result and evidence through the internal request;
6. the staff member cannot approve their own evidence;
7. an authorized officer records a passing outcome and closes the Matter;
8. activation fails before every required gate and succeeds once against the exact active policy;
9. an active vendor receives the real ISO/PCI-DSS request and reaches bank review;
10. expiry, wrong audience, replay, revoked access and delivery failure are safe and truthful;
11. PostgreSQL integration proves transactionality, idempotency, scope isolation, effective-time policy selection and reconstruction;
12. current-contract backend, web, copy-quality, route inventory, clean-schema, rendered-state and accessibility gates pass.

Hosted evidence records only safe IDs, versions, states and timestamps. It never includes a recipient, bearer link, OTP, provider payload or message body.

## 12. Documentation and maturity

Implementation updates `README.md`, `DESIGN.md` when component contracts change, the documentation map, use-case catalogue, implementation ledger, durable-schema ownership, runtime API inventory, issue #80, issue #128 and the relevant acceptance fixtures.

`UC-TPRM-01` may advance only after the activation path above is executable and proven. Broader continuation, reassessment, restriction and exit remain open under issue #80.
