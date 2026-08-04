# Respond and Capture

This document defines how ClearSight collects focused data and evidence from internal users, branches, vendors, customers, other external parties, and protected reporters.

The product goal is:

> **Ask the best-placed person only for the smallest unresolved fact, through the safest practical channel, with no need to understand the wider GRC system.**

Respond and Capture is not a generic public form builder. It is a governed request and evidence-collection system derived from Claims, Evidence Contracts, case directives, scope, authority, and policy.

## 1. Interaction types

### Internal focused request

For authenticated employees, control owners, managers, reviewers, and records custodians. Prefer enterprise SSO and direct routing into the relevant step.

### External invited request

For vendors, consultants, customers, former employees, or other parties without normal ClearSight access. Use a request-scoped invitation and proportionate identity verification.

### Field or mobile capture

For branch, asset, location, incident, document, photo, scan, voice, or short structured observation. Support low bandwidth and governed offline use where permitted.

### Protected reporting

For confidential or anonymous whistleblowing, misconduct, control-bypass, retaliation, or other protected reports. Use a separate identity-isolated channel and protected-case boundary.

## 2. Request model

A request is created only when an exact unresolved need exists.

Required attributes:

- request ID and version;
- purpose and linked Program, Matter, Claim, Evidence Contract, or directive;
- tenant, legal entity, scope, population, and period;
- recipient-selection rationale;
- known facts and source versions;
- unresolved facts;
- response schema and acceptable evidence types;
- sensitivity and purpose limitation;
- channel and identity policy;
- active-effort estimate;
- deadline, reminders, escalation, and stop condition;
- retention and legal hold;
- owner, reviewer, and required authority;
- lifecycle state.

The rendered wizard is a versioned projection of this governed request.

## 3. Request lifecycle

```text
DRAFT
→ POLICY_CHECKED
→ READY
→ INVITED or ASSIGNED
→ VIEWED
→ IN_PROGRESS
→ SUBMITTED
→ VALIDATING
→ ACCEPTED_AS_OBSERVATION
→ REVIEW_REQUIRED or SUFFICIENT
→ FOLLOW_UP_REQUIRED
→ COMPLETED
```

Alternative terminal or intermediate states:

- REDIRECTED;
- DELEGATED;
- PARTIALLY_SUBMITTED;
- WRONG_RECIPIENT;
- CONCERN_RAISED;
- DECLINED where policy permits;
- EXPIRED;
- REVOKED;
- CANCELLED;
- SUPERSEDED.

Submission is not automatically evidence sufficiency or Matter closure.

## 4. Minimum-question wizard

Before rendering a field, ClearSight must:

1. resolve the exact purpose, scope, and recipient authority;
2. search current authorized evidence and prior responses;
3. identify missing, stale, contradictory, or insufficient facts;
4. select the least burdensome approved response type;
5. prefill known values with source and freshness;
6. generate only conditional follow-up questions;
7. stop when the need is satisfied or no longer relevant.

A routine wizard should normally contain:

- one primary question;
- a small group of related fields or one evidence item;
- an optional explanation;
- a final assertion review.

Long forms must be split by distinct user responsibility rather than displayed as one campaign questionnaire.

## 5. Form schema

The request schema may use:

- confirmation of a prefilled value;
- searchable source-backed selection;
- yes/no/unknown with conditional explanation;
- bounded numeric, date, amount, period, or threshold input;
- single or multiple choice;
- structured narrative;
- document upload;
- spreadsheet correction;
- photo or scan;
- voice note with transcript review;
- signature or attestation where legally appropriate.

Every field must define:

- exact assertion produced;
- data type and validation;
- source or recipient authority;
- sensitivity;
- optionality and reason;
- correction behavior;
- whether user confirmation is required;
- downstream Claims and decisions affected.

Do not permit arbitrary HTML, executable logic, uncontrolled scripts, or unrestricted custom database fields.

## 6. Prefill and review

Prefilled values must be visibly classified as:

- authoritative and read-only;
- authoritative with a correction route;
- copied from a prior approved submission;
- provisional source value;
- machine-extracted;
- inferred recommendation;
- stale or contradicted;
- user-confirmation required.

The recipient should not be forced to reconfirm every unchanged value. The final step shows the exact assertions, evidence, period, and scope being submitted.

## 7. Invitation security

### Capability principle

An invitation grants only the narrow ability to view and respond to one governed request or request bundle. It does not grant general account, Program, Matter, search, or tenant access.

### Token requirements

Invitation tokens must be:

- cryptographically random and opaque;
- short-lived and revocable;
- audience- and request-bound;
- single-use for session exchange or explicitly bounded for safe resume;
- stored hashed or using equivalent one-way protection;
- rotated when reissued;
- invalidated on request cancellation, supersession, recipient change, or security event.

Tokens and sensitive request content must not appear in:

- server or client logs;
- analytics payloads;
- referrer headers;
- notification previews;
- page titles;
- third-party scripts;
- browser history beyond the minimum exchange route.

The initial token should be exchanged for a short-lived server-side session and removed from the address bar.

### Identity assurance

Identity policy depends on sensitivity and consequence:

- no additional identity for low-sensitivity public-purpose capture where explicitly approved;
- email ownership verification;
- SMS or authenticator OTP;
- enterprise or vendor federation;
- passkey;
- known-customer verification through an approved channel;
- manual verification by an authorized case owner.

Step-up authentication is required before sensitive evidence, personal data, external representation, signature, or irreversible submission where policy demands it.

### Unsafe invitation states

A forwarded, wrong-recipient, expired, revoked, replayed, or already-consumed invitation must fail without revealing subject, customer, Program, Matter, or request details.

The failure screen should offer a safe contact or “report wrong recipient” route using the invitation identifier only.

## 8. Internal request experience

For an internal recipient:

- use SSO and current employee status;
- resolve role, scope, delegation, and conflict at open and submit time;
- route directly to the request step;
- show why the recipient was selected;
- allow redirect, delegation, conflict, insufficient authority, or wrong scope;
- preserve the accountable owner and deadline during handoff;
- cancel duplicate reminders when evidence arrives elsewhere.

A notification must not contain protected customer, reporter, investigation, or authority-case details.

## 9. External party experience

The external wizard should show only:

- requesting institution identity;
- safe request purpose;
- recipient or organization identity as appropriate;
- required information and acceptable formats;
- estimated effort and deadline;
- privacy, retention, and support notice;
- progress and safe resume;
- final assertions and receipt.

It must not expose internal control names, other vendors, customers, case subjects, reviewers, internal comments, risk ratings, or unrelated evidence.

### External organization requests

For vendors or organizations, support:

- nominated responders;
- request transfer to an authorized colleague;
- organization verification;
- multiple contributors with one accountable submitter;
- reusable evidence packages where purpose and period allow;
- response negotiation or clarification;
- certificate and evidence expiry;
- organization-level audit trail.

## 10. Customer capture

Customer requests require:

- a validated case purpose;
- minimum necessary customer and account context;
- identity assurance appropriate to the data and action;
- accessible, non-accusatory language;
- no disclosure of protected investigations or internal reporting;
- clear distinction between allegation, customer statement, observed record, and verified fact;
- safe communication and correction.

A customer invitation must not imply guilt, suspicion, or a predetermined outcome.

## 11. File, media, and narrative capture

### Files and documents

- allowlisted types and size limits;
- malware and content scanning;
- original version, hash, metadata, and chain of custody;
- resumable upload;
- extraction separated from source fact;
- redaction or minimization where permitted;
- no active content execution;
- validation and partial failure state.

### Photo and scan

- framing, blur, glare, crop, and readability guidance;
- metadata and location notice;
- visible extraction region;
- user confirmation of material fields;
- explicit statement of what the image can and cannot prove.

### Voice and narrative

Use for explanation and observations, not basic identity that can be selected. Transcription and extracted facts require review before submission.

## 12. Draft, resume, correction, and amendment

- autosave where safe;
- explicit saved state;
- request and schema version bound to the draft;
- safe resume without reusing the original token as continuing authorization;
- expiry and revocation behavior;
- changed-since-last-view summary;
- submitter amendment or correction route;
- supersession rather than overwrite;
- reviewer request for focused follow-up rather than full resubmission.

If the request changes materially while in progress, the user must see the changed fields and re-confirm affected assertions.

## 13. Reminder and escalation

Reminder policy should consider deadline, materiality, burden, recipient activity, duplicate requests, source progress, delegation, and whether the need remains relevant.

Rules:

- reminders do not change ownership;
- an unanswered request may escalate to the accountable owner or alternate source;
- expiration does not silently mark non-compliance;
- reminders stop when sufficient evidence arrives elsewhere;
- protected and sensitive notifications remain content-minimized;
- recipient-reported difficulty or concern creates governed work.

## 14. Protected reporting

Protected reporting is not an external evidence request with an anonymous checkbox.

Required capabilities:

- identity and content separation;
- anonymous submission without requiring traceable account creation;
- recovery code or protected mailbox for two-way communication;
- optional confidential identity disclosure to a restricted identity custodian;
- conflict-aware investigator routing;
- retaliation concern handling;
- safe evidence upload and metadata minimization;
- legal privilege and retention markers;
- restricted search, analytics, notifications, export, and AI routes;
- allegation versus verified-fact language;
- separate privileged identity-reveal workflow;
- no credibility scoring from demographics, emotion, style, grammar, or channel.

The ordinary domain receives only approved minimized signals or aggregate findings.

## 15. Offline and low-bandwidth behavior

Offline capture is policy-controlled and normally limited to authenticated internal field workflows.

Requirements:

- encrypted local storage using platform-protected keys;
- request, user, device, tenant, scope, and expiry binding;
- minimum cached context;
- explicit captured, queued, uploaded, submitted, and synchronized states;
- capture time separate from upload time;
- remote revocation and local expiry;
- conflict detection and no silent overwrite;
- duplicate prevention through idempotency keys;
- safe handling after logout, device loss, or employment change;
- protected cases, highly restricted customer data, and reporter identity offline only under explicit approved design.

External magic-link wizards should normally require connectivity rather than persist sensitive anonymous drafts broadly on a device.

## 16. Data architecture

Authoritative records include:

- request and schema versions;
- recipient candidates and selected recipient;
- invitation records and token hashes;
- sessions and identity assurance events;
- draft responses;
- submissions and assertions;
- uploaded evidence references;
- validation and review state;
- reminders, redirects, delegation, and escalation;
- access, export, retention, and legal-hold events.

Invitation and session data must be stored separately from raw evidence. Raw evidence remains in versioned object storage; events and logs carry safe identifiers only.

## 17. Performance and reliability targets

Initial targets:

- invitation redemption and session exchange: p95 under 500 ms;
- deterministic wizard shell and known context: p95 under 1.5 seconds on representative enterprise/mobile networks;
- ordinary step save: p95 under 500 ms;
- ordinary final submission acknowledgement: p95 under 750 ms excluding upload processing;
- uploads resumable and acknowledged immediately after durable chunk receipt;
- validation and AI extraction asynchronous when they exceed the interaction budget;
- no dependency on AI to open, save, or submit a request;
- invitation validation available at the core service SLO;
- retries idempotent and safe after network loss.

## 18. Acceptance scenarios

### A. Internal focused request

A manager receives four unresolved access-review accounts, not the full population. Known employee and role data is prefilled. The manager confirms two, rejects one, and redirects one. Median active effort remains under three minutes.

### B. Vendor evidence invitation

A vendor contact redeems an invitation, verifies email ownership, transfers one section to an authorized colleague, uploads a certificate, and submits. The bank sees source, submitters, period, and remaining evidence limitations.

### C. Forwarded link

An invitation is forwarded. The new browser cannot satisfy audience verification. No request metadata is shown; the recipient can report wrong delivery.

### D. Revoked request

The request is cancelled after invitation delivery. The link fails safely and old draft content is retained or deleted according to policy without permitting submission.

### E. Customer evidence

A customer provides an address document for a scoped case. Extraction is confirmed, but upload alone does not update the core system or establish verified address.

### F. Offline branch capture

A branch user captures a device label offline. Authorization expires before synchronization. The draft remains encrypted, submission is blocked, and an authorized revalidation route is offered.

### G. Protected report

An anonymous reporter submits narrative and audio, receives a recovery code, and responds to investigator questions without identity exposure. A conflicted investigator is excluded.

## 19. Prohibited shortcuts

Do not:

- expose a broad Matter through a request link;
- reuse one permanent magic link;
- put tokens or sensitive context in email subject lines or analytics;
- treat possession of a link as sufficient identity for high-impact actions;
- allow external users to browse the tenant;
- use a generic form builder disconnected from Claims and Evidence Contracts;
- require broad questionnaires when exact unresolved facts are known;
- treat any response as sufficient evidence;
- silently replace authoritative source values with respondent input;
- implement anonymous reporting inside ordinary request storage or search;
- allow offline capture without revocation, expiry, and conflict handling.

## 20. Definition of success

Respond and Capture succeeds when the best-placed internal or external person can safely provide the exact missing information in minutes, without receiving unnecessary access or context, while ClearSight preserves identity assurance, purpose, provenance, review, escalation, and evidence sufficiency.
