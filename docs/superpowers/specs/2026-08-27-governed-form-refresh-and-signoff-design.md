# Governed form refresh and sign-off design

**Date:** 2026-08-27

**Status:** Proposed, implementation-ready after review

**Maturity target:** End-to-end usable collection forms for internal and external respondents, including focused updates of held vendor data and expiring documents

## Problem

ClearSight can already define governed form revisions, collect typed answers, issue purpose-bound external invitations, assign internal respondents, capture attestations and signatures, and run vendor onboarding or reassessment episodes. The remaining experience is fragmented:

- form authoring is discovered through a Program monitoring screen even though active forms are reusable across the legal entity;
- vendor due-diligence orchestration accepts only an external email recipient even though the evidence-request domain supports internal principals;
- invitation expiry is presented as fixed duration presets and the effective deadline cap is not explained before send;
- the protected invitation-delivery seam has no production email adapter or branded message renderer;
- known vendor data is displayed as context, not as version-bound values a respondent can confirm or correct;
- a submitted correction cannot be reviewed and applied through the existing authority-gated vendor identity command;
- vendor documents can be validated, rejected or marked expired, but cannot be explicitly superseded by a replacement;
- no deterministic production job turns document expiry or stale vendor facts into a focused reassessment scope.

The implementation must close those gaps without creating another form engine, capture store, invitation model, vendor workflow or generic database editor.

## Existing capabilities to retain

| Capability | Existing owner | Decision |
| --- | --- | --- |
| Form contract, sections, conditions, constraints, scoring and field types | `internal/formcontract` | Extend the existing field contract with optional collection intent and a bounded target key. Do not create a vendor questionnaire schema. |
| Versioned form lifecycle and maker-checker activation | `internal/monitoring` | Keep forms Program-owned and legal-entity reusable. Expose a clearer library surface; do not create a global form aggregate. |
| Classic/wizard rendering, file/photo/vendor-document fields, attestation and signature | shared capture React components | Reuse unchanged where possible. Add current-value comparison controls around the existing inputs. |
| Internal-principal and external-audience recipients | `internal/evidence` | Make vendor orchestration accept the existing recipient union. Do not create vendor-specific recipient records. |
| Opaque invitation, bounded session, revocation, replacement and deadline clamping | `internal/evidence` | Retain token and session security. Add exact requested expiry support and display the effective expiry before send. |
| Protected invitation-delivery interface and redacted receipt | `internal/evidence` | Add a branded renderer and configured email adapter behind this interface. Do not persist or enqueue a raw token. |
| Immutable submission answers and provenance | `internal/evidence` | Treat the submission as the change proposal. Add an application receipt that links accepted fields to the resulting authoritative version. |
| Vendor identity update command, version check, authority route, event and outbox | `internal/thirdparty` | Apply approved identity corrections through this command. Do not write `third_parties` directly. |
| Onboarding, periodic and triggered assessment episodes | `internal/thirdparty` | Use periodic/triggered episodes for refresh. Add an immutable selected-field scope; do not add a separate refresh workflow. |
| Focused clarification and vendor-work change requests | `internal/thirdparty` | Reuse their exact-field filtering rules and request sequencing. |
| Document review | `internal/thirdparty` | Extend validation with explicit, version-checked supersession. |

## Non-goals

- No arbitrary table or column names in forms, APIs or browser state.
- No automatic acceptance of respondent claims into authoritative records.
- No separate vendor form builder, invitation table, signature service or email campaign system.
- No AI dependency for expiry, staleness, field selection, validation or routing.
- No silent replacement or deletion of prior identity or document history.
- No invitation token in logs, analytics, email receipts, outbox payloads or browser previews.
- No claim that a typed or drawn signature alone supplies legal authority; authority and respondent provenance remain separately recorded.

## Core design

### 1. Keep one form library

The existing monitoring form revision remains the only template record. A new **Collection forms** surface lists the existing legal-entity reusable revisions, shows their owning Program, lifecycle state, current version, field count and last update, and opens the existing builder in the selected Program context.

Creating a form still requires a Program because that is its continuity and ownership context. The library may filter by Program, state and name, but it must use the bounded reusable-form query already present. Activation retains maker-checker review. Editing an active form creates a new draft revision; it never mutates an active revision.

The vendor empty state continues to offer the governed starter form, but it must link to the same form record and lifecycle rather than copying it into a vendor store.

### 2. Add bounded response intent, not database mapping

Each form field gains optional metadata:

```text
collection_intent:
  CAPTURE                     default; collect a new answer
  CONFIRM_OR_CORRECT          show a held value and ask whether it remains accurate
  REPLACE_HELD_DOCUMENT       show a held document and collect a proposed replacement

record_target:
  key                         allowlisted domain key such as VENDOR.IDENTITY.REGISTERED_ADDRESS
  required_subject_type       prevents use against the wrong workflow subject
```

The server owns the target catalog. A target resolver defines the value type, readable label, eligible subject, exact read method, current-version reference and authorized apply command. The initial vendor catalog is deliberately small:

- vendor legal name;
- trading name;
- registration reference;
- jurisdiction;
- registered address;
- website domain;
- a vendor document type, such as certificate of operation.

Unknown targets fail form validation. Browser-supplied target IDs, vendor IDs, table names and column names are never trusted.

### 3. Freeze the held value into each request

When an assessment request is composed, every targeted field is resolved with an exact subject identifier and bounded read. The resulting request field stores an immutable baseline:

```text
record_baseline:
  target_key
  subject_type / subject_id
  record_id / record_version
  display_value or held_document_summary
  source_label
  observed_or_confirmed_at
  expires_at, when applicable
```

This is request evidence, not the new authoritative value. It lets the recipient see exactly what the bank held when the request was issued and lets the reviewer detect intervening changes. Sensitive values remain subject to the existing request audience and field visibility rules.

`KnownFacts` remains supporting context for untargeted facts. It is not promoted into a second record store.

### 4. Respondent experience

`CAPTURE` fields retain the current controls.

`CONFIRM_OR_CORRECT` fields show a compact current-value card with the value, source/freshness where available, and one dominant choice:

- **Confirm this is accurate** records confirmation of the frozen baseline; or
- **Update this information** reveals the existing typed control populated with the held value.

`REPLACE_HELD_DOCUMENT` fields show the current document type, reference, issuer and expiry, then use the existing vendor-document upload control for the proposed replacement. The prior document is never overwritten during submission.

The review step clearly distinguishes confirmed values, proposed changes, new files and replacements. Required attestations and signature fields continue to be rendered by the existing controls. The final submission records the authenticated internal principal or external request/session provenance already held by the capture domain.

### 5. Sign-off without a second signature system

Form authors already have attestation and signature field types. The builder adds a **Require sign-off** convenience that inserts or configures a final required attestation and, when chosen, a signature field. These remain ordinary versioned fields in the form contract.

For an internal respondent, `SubmittedBy` identifies the verified principal. For an external respondent, the request, invitation audience binding and external session identify the response channel. A signature image remains a bounded PNG artifact linked through the answer. The submission review shows the exact attestation text from the form revision and the response provenance.

No separate mutable “signed” flag is introduced.

### 6. One recipient model for internal and external requests

Vendor assessment send, reissue and focused follow-up commands accept the existing recipient union:

- **Internal colleague:** a verified eligible principal is assigned to the evidence request; no magic link is created. The request appears in their work queue and submission provenance identifies them as an internal respondent.
- **Vendor or other third party:** an approved external audience receives a purpose-bound invitation. Vendor attestation must never be inferred when an internal user responds on the vendor's behalf.

The UI uses the existing bounded internal-recipient search. External addresses are normalized, cleared from the screen after delivery attempt, and never returned by administration endpoints.

### 7. Due-date-aware invitation expiry

The invitation service remains authoritative for the five-minute minimum, 30-day maximum and request-deadline ceiling. Command inputs gain an optional absolute `invitation_expires_at`; legacy duration minutes remain accepted during migration.

The UI uses a date-and-time control with presets such as one day, seven days and “at the response due time.” Its maximum is the request due time and it previews the effective expiry before send. If a caller requests a later time, the API returns the bounded effective expiry and a machine-readable adjustment reason; the UI states that the link ends when the response is due. Replacement invitations use the same rule.

### 8. Branded invitation email through the existing protected seam

The protected delivery request is extended with non-secret message data: bank display name, form title, task summary, due time, effective link expiry and support text. The raw recipient and one-time link remain in protected fields and are never serializable.

A responsive HTML and plain-text renderer produces:

- bank and ClearSight identity;
- the specific information requested;
- response due time and secure-link expiry;
- one primary **Open secure form** action;
- a plain URL fallback;
- a warning not to forward the link;
- recovery/support guidance.

A configured production email adapter implements the existing `InvitationDelivery` interface. Provider responses are normalized to the existing redacted receipt. When delivery is unavailable or fails, the current one-time manual-link recovery remains visible. The initial implementation stays synchronous because persisting a reusable raw token in the generic outbox would violate invitation protections; reliable resend creates a replacement invitation instead of replaying secret material.

### 9. Review and apply proposed record changes

Submission remains immutable and does not update held data. The reviewer sees a field-level comparison of:

- request baseline and its version;
- respondent answer and provenance;
- current authoritative value and version;
- any conflict or newer change.

The reviewer can accept or reject each proposed change, then apply the accepted set. Identity fields are assembled into one complete `UpdateVendorIdentity` command using a fresh exact read, the baseline/current version rules and the verified reviewer/owner authority route. A conflict requires re-review; it never silently overwrites a newer value.

An immutable application receipt links request, submission, accepted field IDs, target keys, prior versions, resulting versions, actor and time. The domain event and outbox remain in the same transaction as the authoritative update. Rejected fields retain their submission history but do not change the vendor.

### 10. Document replacement is explicit supersession

`third_party_documents` gains:

- `SUPERSEDED` status;
- nullable `supersedes_document_id` with tenant/relationship integrity;
- the inverse current/replacement projection required by review;
- a partial index for bounded current-document lookup by relationship and document type.

When a replacement document is validated, the reviewer must identify the held document from the frozen baseline. In one transaction the repository:

1. verifies the submitted artifact is available;
2. verifies the held document and expected version are still current;
3. validates the new document;
4. marks the held document `SUPERSEDED`;
5. records the supersession link, event and outbox payload;
6. retains both artifacts and every prior decision.

Expired and rejected documents are not relabelled as superseded unless a validated replacement explicitly names them.

### 11. Smart refresh reuses reassessment episodes

Periodic and triggered assessments gain an immutable selected-field scope. Empty scope means the full approved form for backward compatibility. A scoped assessment composes the initial evidence request using the same exact-field filtering already used by clarification and vendor-work change requests.

A deterministic third-party maintenance job performs bounded, leased and deduplicated scans:

- mark validated documents expired when `expires_on` has passed;
- emit the versioned event/outbox record and evidence-aging signal;
- identify form targets whose held documents are expired or within the configured lead time;
- identify targetable vendor identity facts whose confirmation interval has elapsed;
- create attention for the relationship owner with a preselected triggered-assessment scope.

By default the job does not choose a recipient or send an invitation. The owner opens **Request updated information**, reviews the selected fields, adds/removes confirmable facts, chooses recipient, due date and link expiry, and sends. Fully automated creation or send requires an approved Automation Policy with the required blast-radius, expiry, monitoring, kill switch and outcome contract.

This provides the requested “confirm what is still accurate and update what is stale” experience without a new refresh aggregate.

## UI surfaces

### Collection forms

- searchable, filterable list of reusable current forms;
- owning Program and lifecycle shown on every row;
- create draft, edit as new revision, preview, submit for approval, approve/reject, pause and retire through existing commands;
- clear empty state linking to a Program or installing the governed starter form;
- real date/calendar controls for effective dates where lifecycle support is exposed.

### Vendor due diligence

- start onboarding, periodic or triggered reassessment with the existing form picker;
- choose full review or focused refresh where the episode permits it;
- preselect expired/expiring documents and stale facts;
- choose internal colleague, vendor contact or other third party;
- choose response due date and exact link expiry;
- preview what the recipient will be asked and the effective link lifetime;
- show delivery result and a safe recovery action.

### Vendor review

- compare held, submitted and current values;
- review document metadata and security-scan state;
- accept/reject proposed fields;
- apply accepted identity changes through one governed command;
- validate a replacement document and see what it supersedes;
- show unresolved conflicts as the dominant next action.

### Capture form

- white document surface in light mode and the corresponding high-contrast dark surface in dark mode;
- compact icons only where they aid scanning, always paired with accessible labels;
- focused dialog/drawer overlays use subtle backdrop blur without obscuring errors or required context;
- date values use native calendar inputs and readable localized review text;
- keyboard, screen-reader, reduced-motion, 200% zoom and mobile layouts remain supported.

## Security and authority

- Every material command uses verified request identity; actor-like body fields remain ignored.
- Form transitions retain maker-checker separation.
- Internal recipients must have exact subject access.
- External invitations remain request-, audience- and purpose-bound, revocable and short-lived.
- Record target resolution is server-side and allowlisted.
- Applying a response re-evaluates the current authority route and fails closed on missing identity, route failure, tenant/entity mismatch, conflict, delegation expiry or revoked responsibility.
- Invitation secret material never enters events, outbox, logs, analytics, previews or saved delivery receipts.
- Protected records are filtered by repository/API scope, not hidden only in the browser.

## Performance and retention

- Reuse the existing bounded reusable-form index and keyset portfolio patterns.
- Request composition reads only selected target keys for one exact relationship.
- Current document lookup uses tenant, legal entity, relationship, document type and current status.
- Expiry maintenance uses a partial due-date index, bounded claims, leases, dedupe keys and retry/dead-letter visibility.
- Form revisions, request baselines, submissions, application receipts, document versions and events follow existing retention and point-in-time reconstruction rules.

## Acceptance criteria

1. An authorized maker can create and preview a form with typed fields, conditions, file limits, attestation and signature; a distinct checker can activate it; the active revision appears in the reusable form list.
2. A due-diligence owner can send the same form to an eligible internal principal or an external recipient. Internal assignment creates no invitation; external send creates one opaque invitation.
3. The sender can choose an exact invitation expiry no later than the request due time and 30 days. The UI displays the effective expiry before and after send.
4. With email configured, the external recipient receives the branded HTML/plain-text message and opens only the bound request. With email unavailable, the sender receives the existing one-time manual-link recovery without secret leakage.
5. A form can request confirmation or correction of vendor legal name, registered address or website domain using the frozen request baseline.
6. A respondent can confirm an unchanged value or propose a corrected value. Submission changes no vendor record.
7. An authorized reviewer can accept selected proposed identity fields. The existing vendor identity command creates the new version, event and outbox, and an application receipt links it to the submission.
8. A newer vendor identity version causes a visible conflict and prevents silent application.
9. A replacement certificate submission retains the old document. Reviewer validation supersedes the exact prior document in one transaction and preserves both versions.
10. Expired documents are marked by deterministic maintenance and produce owner attention. Starting the suggested refresh opens a triggered assessment with the relevant document field selected.
11. A refresh can also ask the recipient to confirm current address and other selected facts, while allowing corrections only to those fields.
12. Required attestations and signatures are visible in form preview, capture and final review; submission provenance identifies the internal principal or external session channel.
13. Every affected screen has loading, empty, degraded, unauthorized, conflict, success and recovery fixtures; desktop and mobile renders pass visual and copy-quality review.
14. Existing full-form onboarding, clarification, vendor work, invitation administration and static-demo workflows remain compatible.

## Verification strategy

- `internal/formcontract`: normalization and target-catalog rejection tests.
- `internal/monitoring`: revision, maker-checker, reusable lookup and compatibility tests.
- `internal/evidence`: baseline serialization, internal/external recipient, absolute expiry bounds, revocation, session, sign-off artifact and protected-delivery redaction tests.
- `internal/thirdparty`: selected-scope assessment, identity apply/conflict, document supersession, expiry maintenance, dedupe, transaction/outbox and reconstruction tests.
- `internal/httpapi`: verified identity, route authority, field-target tampering and error mapping tests.
- React: form library, authoring, recipient chooser, exact expiry, confirmation/correction, replacement, sign-off, review/apply, responsive and accessibility tests.
- Static demo: equivalent happy, empty, conflict, email-fallback and expiry states without weakening production authority semantics.
- Run Go unit/integration suites, web unit tests, copy-quality regression, production build, affected UI state renderer, responsive screenshots and deployed smoke checks before completion.

## Rollout sequence

1. Extend the shared contract and request baseline with backward-compatible optional fields.
2. Expose and harden the existing reusable form library and builder.
3. Generalize recipient selection and exact invitation expiry while retaining legacy duration inputs.
4. Add the branded delivery adapter and safe fallback.
5. Add reviewer comparison and vendor identity application receipts.
6. Add document supersession and its current projection.
7. Add scoped reassessments and deterministic expiry/staleness attention.
8. Complete state fixtures, visual QA, compatibility tests, migration checks and deployment verification.

Each step must remain deployable with old forms and requests. New behavior activates only when a field has a recognized collection intent and target.
