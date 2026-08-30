# Vendor Email Acceptance Journeys Design

**Date:** 2026-08-30

**Status:** Proposed for implementation

**Related tracking:** GitHub issue #80 (focused acceptance slice; not full third-party lifecycle closure)

**Maturity target:** Real-email hosted acceptance for vendor registration, address verification and certification refresh using the governed Forms and Vendor Review foundations already delivered on `main`

## 1. Decision summary

Deliver two real hosted acceptance journeys using the operator-supplied SMTP account and test inboxes:

1. invite a newly added vendor to register, create one canonical Vendor Review Matter with an address-verification requirement, collect confirmation and evidence from a purpose-bound staff email recipient, then let an authorized compliance officer review the evidence, record the outcome, sign off and close the Matter; and
2. ask the same registered vendor for new ISO 27001 and PCI DSS certification evidence, collect it through a secure vendor link, review it and resolve the related Vendor Review Matter only after an explicit passed outcome.

The implementation will configure real STARTTLS email delivery on the hosted server, improve the actual HTML and plain-text messages, verify every generated link and render the material UI states at desktop and mobile sizes. It will reuse the existing governed Forms access, email OTP, response-revision, evidence and Matter lifecycle models rather than introduce a parallel invitation or case system.

SMTP credentials, generated encryption/HMAC keys and recipient addresses are deployment inputs. They must not be committed, printed in logs, added to screenshots or copied into documentation.

## 2. Goals and non-goals

### 2.1 Goals

- Prove that a real vendor recipient receives a polished registration invitation and can complete the registration workflow through the link in that message.
- Show the resulting address-verification work as pending on the vendor and its canonical Vendor Review Matter without mislabelling an unverified address as a deficiency.
- Prove that a real staff test recipient receives a purpose-bound address-confirmation request, verifies access by email OTP and can provide confirmation plus evidence.
- Keep evidence submission, evidence review, outcome recording, sign-off and Matter closure as separate observable states.
- Prove that a registered vendor can receive and answer a certification-refresh request for ISO 27001 and PCI DSS evidence.
- Make the real messages and linked flows clear, accessible, responsive and operationally credible for a bank user.
- Produce reproducible automated and rendered evidence, followed by a controlled live hosted acceptance record with timestamps and provider results.

### 2.2 Non-goals

- A general employee-notification platform or new staff-directory/contact-channel model.
- Granting internal Matter authority to a person merely because they control an email inbox.
- Replacing the canonical Vendor Review Matter with a second address or certification case model.
- Treating message acceptance by the SMTP relay as proof of inbox placement, reading or recipient action.
- Claiming uploaded documents are malware-free or authoritative when the hosted environment has no configured production scanning/inspection service.
- Closing issue #80 or claiming the complete third-party lifecycle is finished.
- Sending customer or production-vendor email. The live acceptance run is limited to the two operator-supplied test inboxes.

## 3. Current baseline and gaps

The repository already provides:

- governed form templates, publishing and distributions;
- opaque, purpose-bound recipient links whose access value remains in the URL fragment;
- direct-link access followed by email OTP and recorded `EMAIL_VERIFIED` assurance;
- immutable response revisions, evidence references and reviewer change requests;
- invitation, reminder, due-soon, expiry, change-request, amendment and completion communication actions;
- one canonical `VENDOR_REVIEW` Matter per vendor-assessment lineage;
- Matter assignment, verification contracts, recorded outcomes, sign-off and closure guards; and
- vendor assessment fields that can confirm or correct a registered address and replace supporting documents.

The hosted runtime does not yet have recipient-encryption/HMAC keys or SMTP delivery enabled. Its secure external-distribution path therefore fails closed as designed. The SMTP host is reachable from the server and its STARTTLS certificate chain has been verified, but no live message has been sent.

The current email HTML is structurally minimal and depends on CSS classes that common inboxes will not style. OTP mail is similarly bare. Existing Forms screens can preview and test-send a communication, but the end-to-end vendor journeys, address-evidence handoff, certification-refresh context and resolved UI state have not been proven together.

## 4. Alternatives considered

### 4.1 Configuration-only manual run

Configure SMTP, manually assemble existing generic forms and rely on screenshots of whatever results. This is quick but leaves the staff handoff, business copy, deterministic states and regression evidence ambiguous. It is rejected.

### 4.2 Broad notification and identity expansion

Create a generic internal notification service, staff contact preferences and new authority assignments before running the journeys. This could support future use cases, but it expands scope and risks conflating communication reachability with workflow authority. It is rejected for this acceptance slice.

### 4.3 Governed Forms acceptance layer — selected

Use the existing external-audience Forms delivery and OTP boundary for both the vendor and the staff evidence respondent. The compliance officer remains the verified Matter owner and signatory. The emailed staff recipient is labelled `Address verification staff contact` and receives authority only to answer the specific evidence request. This proves the requested experience without weakening identity or governance semantics.

## 5. Actors and authority boundaries

### 5.1 Compliance officer

The compliance officer acts through a verified application session with tenant and legal-entity scope. This actor may create and assign the governed requests, review submissions, request changes, record a verification outcome and sign off only when the current versioned authority route permits those commands.

### 5.2 Vendor contact

The vendor contact uses an opaque, audience-bound, revocable form-access value plus an email OTP. This proves control of the invited inbox for the request; it does not create bank-user privileges. The contact can view and submit only the purpose-bound vendor registration or certification-refresh form.

### 5.3 Address verification staff contact

The staff test inbox is a purpose-bound email respondent for the address-confirmation evidence request. The recipient can confirm the check, explain the method and result, and upload evidence. Email verification does not make this recipient the Matter owner, reviewer, approver or signatory. The UI and email must not say otherwise.

### 5.4 Separation of duties

Submission by the vendor or staff recipient never equals acceptance. The compliance officer must inspect the current response and evidence, resolve contradictions, record the outcome and sign off. Matter closure remains guarded by an explicit passed outcome and current authority. Conflict, delegation, absence or revoked authority must fail closed through the existing routing contract.

## 6. Hosted SMTP and secret configuration

The hosted API and worker will receive these protected environment settings through the existing deployment configuration:

- SMTP host and port;
- SMTP authentication username and password;
- sender address;
- STARTTLS mode;
- external-distribution delivery enablement;
- the public capture base URL;
- a generated AES-256 recipient keyring and active key ID; and
- a generated 32-byte distribution-access HMAC key.

New cryptographic values will be generated on the server with a cryptographically secure source. Existing values, if discovered at deployment time, will be preserved unless rotation is intentionally planned. The protected environment file will remain readable only by its intended privileged owner. Commands and evidence capture must report only whether required settings are present, never their values.

STARTTLS is required. Authentication must occur only after successful TLS negotiation and certificate verification. Delivery remains fail-closed when security configuration is absent or TLS/authentication fails. No automatic downgrade to clear-text SMTP is allowed.

Configuration rollout will be reversible: back up the protected environment file to a root-readable timestamped copy, apply the minimum variables, restart only the affected services, verify health and run one narrowly addressed live acceptance send. On failure, restore the protected configuration, restart the services and retain redacted diagnostics. Rollback must not invalidate already-issued recipient access unless recipient-key configuration itself is deliberately restored to the matching prior set.

## 7. Email presentation and content

All real communication actions will use one shared email-client-safe presentation shell:

- a text brand header that remains understandable when images are blocked;
- a concise preheader hidden from the rendered body but visible in inbox previews;
- one task-specific heading and one dominant call to action;
- the business object, requesting bank, recipient role, due time and link expiry;
- a plain URL fallback with safe wrapping;
- support and recovery instructions;
- a short security notice explaining that the recipient will verify the invited email address and must not forward the message; and
- a plain-text alternative with the same information and action order.

Styles will be inlined or expressed with conservative table-based markup so Gmail and other common clients do not depend on application CSS classes. The design will meet readable contrast, minimum touch-target and narrow-screen requirements. It will not contain tracking pixels, remote marketing assets, slogans or claims that a task is complete before the current state proves it.

The primary messages are:

- **Complete your vendor registration** — sent to the vendor contact with the registration deadline and secure form link.
- **Verify the vendor's registered address** — sent to the address verification staff contact with the vendor name, address to check, evidence expectations, deadline and secure response link.
- **Submit current certification evidence** — sent to the registered vendor contact with separate ISO 27001 and PCI DSS requirements, accepted evidence descriptions, deadline and secure form link.
- **Verify your email address** — OTP message naming the request being accessed, code expiry and recovery action.
- **Changes needed** and **Submission received** — sent only when the corresponding governed communication event occurs.

No secret access value or OTP may appear in server logs, analytics, message previews or captured test output.

## 8. Journey 1: registration and address verification

### 8.1 Setup

Create a clearly labelled test vendor under the acceptance tenant with realistic but non-customer data and the operator-supplied vendor test inbox. The compliance officer selects `Send registration request`, reviews the recipient, due time and link expiry, then confirms the send. The material distribution, recipient record, append-only event and outbox event share one transaction.

### 8.2 Vendor registration

The worker sends the registration email through the configured SMTP relay. Acceptance evidence records the communication event ID, redacted recipient fingerprint, provider response category and timestamps without storing message credentials or access values in logs.

The vendor follows the secure link, requests and supplies the OTP sent to the invited inbox, then completes the registration form. The form collects the existing vendor identity and due-diligence fields required by the product specification, including confirmation or correction of the registered address. Validation errors identify the field, reason and recovery action. Submission creates an immutable response revision and an auditable linkage to the vendor assessment.

### 8.3 Pending address verification

Registration submission creates or updates the one canonical `VENDOR_REVIEW` Matter for the assessment lineage. The Matter displays `Address verification pending`, the address supplied by the vendor, source and freshness, accountable compliance owner, next action and deadline. It must not create a `VENDOR_DEFICIENCY` merely because the address is awaiting verification.

The vendor workspace and Matter detail show the same current state and link to each other through exact identifiers. The dominant compliance action is `Assign address check`.

### 8.4 Staff evidence request

The compliance officer assigns the address-confirmation request to the operator-supplied staff test inbox. The UI states that this grants access to the evidence request only and leaves Matter accountability with the compliance officer. Confirmation creates the governed external-audience distribution and outbox event transactionally.

The staff recipient receives the address-verification email, verifies the invited inbox by OTP and sees:

- the test vendor and registered address;
- the verification method requested;
- confirmation choices that do not force a positive answer;
- fields for the check date, source/contact, result and explanation;
- an evidence upload control; and
- the consequence of submitting: compliance review, not automatic closure.

The response supports `verified`, `could not verify` and `different address found`. A contradiction or different address keeps the Matter open and makes `Review address evidence` the dominant compliance action.

### 8.5 Review, outcome and closure

After submission, the compliance officer can inspect the response revision, evidence metadata and inspection state. Evidence that is unscanned or not successfully inspected must be visibly labelled and cannot support a passed outcome where the verification contract requires inspected evidence.

The compliance officer may request changes, reject the evidence, or record a passed/failed/inconclusive verification outcome. A passed outcome records the current response/evidence versions used. Sign-off is a separate material command routed through current authority. Only a current passed outcome plus completed sign-off enables `Resolve address verification` and Matter closure.

After closure, the vendor workspace and Matter detail show `Resolved`, the resolution time, outcome, signatory and evidence basis. Historical states remain reconstructable. Reopening or superseding evidence must make the prior outcome visibly stale rather than silently current.

## 9. Journey 2: certification refresh

### 9.1 Setup and request

Use the registered test vendor from Journey 1. The compliance officer creates a certification-refresh request linked to the same vendor and canonical Vendor Review context. The requested items are:

- an ISO 27001 certificate, including issuing body, certificate number, scope, issue date and expiry date; and
- PCI DSS evidence, specifically the applicable current Attestation of Compliance, including assessment type, assessor, effective date and expiry date.

The UI does not imply that either framework applies universally. The requester confirms applicability and records why each item is requested.

### 9.2 Vendor response

The vendor receives `Submit current certification evidence`, follows the opaque link, completes email OTP and sees separate cards for ISO 27001 and PCI DSS. Each card explains the expected document and metadata, accepted file constraints, deadline, current saved state and next action. The form supports save/resume, validation, upload replacement and explicit submission.

Sample documents used for acceptance are clearly marked as test evidence and contain no real customer data. Upload, storage, inspection, submission and reviewer acceptance remain distinct states. A successful form submit must not label a certificate accepted or the Matter resolved.

### 9.3 Review and resolution

The compliance officer reviews each item independently against the requested scope, dates and inspection status. The officer can request a replacement for one item without discarding the other accepted item. Every resubmission creates a new immutable response/evidence revision.

The Matter remains open until all applicable certification requirements have a current accepted review, the verification contract has a passed outcome and authorized sign-off is recorded. The resolved UI identifies which certifications were accepted, their expiry dates, evidence versions and next review date. Missing, expired, not-applicable or rejected evidence remains explicit and cannot be converted into a persuasive completeness claim.

## 10. Data, consistency and recovery

- Vendor, assessment, Matter, form distribution, recipient, response, evidence and outcome links use exact identifiers and tenant/legal-entity-scoped repository queries.
- Each material command writes authoritative state, its append-only audit event, required outbox event and maintenance work in one transaction.
- Worker delivery is idempotent. Retrying a claimed message must not create a second distribution or silently reset recipient state.
- Provider acceptance, recipient OTP verification, response submission, evidence review, outcome and sign-off have separate timestamps and states.
- Generated links remain opaque, audience-bound, short-lived and revocable. The access value stays in the URL fragment and is removed from visible browser state after exchange.
- A derived-projection refresh failure after commit is reported as stale UI data with recovery, not as a failed material command.
- Point-in-time reconstruction can identify the vendor data, request, response revision, evidence versions, outcome and signatory that supported closure.

## 11. UI states and copy

The implementation will add deterministic fixtures for the material states rather than relying on mutable live data for visual regression:

- registration invitation ready, sending, sent and delivery failed;
- vendor registration access, OTP, form error, saved draft and submitted;
- address verification pending, staff request ready/sent, staff OTP, evidence draft/submitted;
- compliance review, change requested, passed/failed/inconclusive outcome, sign-off unavailable/available and resolved;
- certification request ready/sent, vendor draft, per-item validation/replacement, submitted, partially accepted and resolved;
- expired/revoked link, offline/degraded state, permission denial and stale projection; and
- desktop, mobile, keyboard focus, 200% reflow approximation, light and dark presentation for representative states.

Every state has one dominant action for the current actor. Visible copy names the vendor, request or evidence state; identifies the owner/deadline/source where relevant; and explains the next result. It does not expose internal status codes or call a generic form submission a verified outcome.

## 12. Verification and live acceptance protocol

### 12.1 Automated verification

Fresh verification must include:

- backend unit and PostgreSQL integration tests for distribution/outbox atomicity, delivery idempotency, opaque access, OTP expiry/replay, response revisions, evidence-state guards, authority routing and Matter closure;
- SMTP adapter tests for STARTTLS requirement, authentication, MIME structure, HTML/plain parity, header-injection rejection and redaction;
- web workflow tests for both journeys, copy-quality regression, accessibility checks and responsive state behavior;
- exact link tests proving the fragment-carried access value reaches the exchange flow without appearing in server request logs;
- negative paths for wrong inbox/OTP, expired and revoked access, provider failure, unscanned evidence, contradiction, stale outcome, unauthorized sign-off and duplicate worker delivery; and
- Compose/runtime checks with a local capture server or deterministic SMTP test double. Automated tests must not send to the live inboxes.

### 12.2 Rendered evidence

Capture representative states at desktop and mobile widths, inspect every image and correct the highest-impact defect before rechecking. Evidence must include the rendered email HTML, vendor registration, pending address Matter, staff evidence form, compliance review/sign-off, resolved Matter, certification request, vendor certification form and certification review states. OTP values and opaque links must be redacted or replaced in artifacts.

### 12.3 Controlled live run

After automated gates pass and the exact commit is deployed:

1. verify health, release revision and presence—not values—of required protected configuration;
2. send one registration invitation to the vendor test inbox and confirm provider acceptance plus inbox receipt;
3. traverse Journey 1 using the real messages and links, including the staff inbox and compliance closure;
4. send one certification-refresh request to the registered test vendor and traverse Journey 2;
5. confirm the final hosted UI state and point-in-time audit history; and
6. record redacted timestamps, message/event identifiers, screenshots and any external limitation.

The run stops if a recipient, tenant, legal entity or sender differs from the approved test setup. It does not broaden to any other address.

## 13. Deployment and rollback

Deploy only after repository tests and rendered review pass. The release procedure will:

- preserve a protected backup of the prior server configuration;
- apply SMTP and recipient-security settings without printing them;
- deploy the exact tested revision;
- restart API and worker services and verify readiness;
- run a redacted configuration check and one controlled delivery;
- retain the prior application image and configuration for rollback; and
- avoid rotating or deleting keys needed to decrypt existing recipient records during rollback.

If delivery or link exchange fails, disable new external delivery, preserve the outbox and recipient audit state, diagnose without copying access values, then either resume idempotently or restore the prior configuration and image. Already committed requests remain visible with an accurate delivery state and recovery action.

## 14. Acceptance criteria

This slice is complete only when fresh evidence shows:

- both operator-supplied test inboxes receive their intended real messages over STARTTLS;
- each message has the approved sender, clear preheader/body, one dominant action, correct deadline/expiry and a usable plain-text alternative;
- every generated link exchanges successfully, requires the invited inbox OTP and resists expiry, revocation and replay paths;
- vendor registration produces the canonical Vendor Review Matter and an explicit pending address-verification state;
- the staff recipient can submit confirmation and evidence but cannot review, sign off or close the Matter;
- the compliance officer can separately review, record an outcome, sign off and close only after a current passed outcome;
- the final hosted UI shows the address Matter resolved with its evidence basis and audit history;
- the registered vendor can submit separately identified ISO 27001 and PCI DSS evidence, receive a targeted change request and resubmit without losing accepted work;
- certification submission, evidence inspection, acceptance and Matter resolution remain distinct;
- desktop/mobile rendered evidence is visually reviewed and the copy-quality, accessibility and affected workflow suites pass; and
- the deployed release, protected configuration presence and live-run evidence are recorded without exposing credentials, OTPs or opaque access values.

## 15. Explicit remainder after acceptance

Completing these journeys will not by itself finish the following work:

- a production staff notification/contact-channel model tied to authenticated internal principals;
- production-grade versioned object storage, malware scanning and document-inspection evidence if those services remain unconfigured;
- bounce, complaint, deliverability monitoring, sender-domain SPF/DKIM/DMARC governance and provider operational dashboards;
- automated reminders/escalations beyond the communication actions already governed by the Forms lifecycle;
- broad third-party onboarding, renewal, termination, fourth-party, concentration and portfolio operating coverage tracked by issue #80;
- representative-user usability testing, assistive-technology testing and sustained production load/recovery exercises; and
- use of the acceptance SMTP account or test inboxes for customer communication.

These remain named follow-up items. The acceptance result must state which were configured and proven, which were observed only at provider acceptance, and which remain external dependencies.
