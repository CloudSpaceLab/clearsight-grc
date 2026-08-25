# Vendor due-diligence and data-collection design

**Status:** Approved design
**Date:** 2026-08-26
**Issue:** #80
**Depends on:** vendor-organization and service-relationship foundation in `internal/thirdparty`; shared Matter, Monitoring Form, Evidence Request, capture, invitation, authority, runtime and outbox capabilities

## 1. Outcome

ClearSight will let a bank relationship owner start one bounded vendor due-diligence episode from the Vendors workspace, send a secure request-scoped magic link, collect a partially prefilled structured response, review submitted evidence and record an authorized assessment conclusion.

The workflow must be seamless for the vendor without weakening identity, evidence or decision semantics. A vendor does not need a permanent ClearSight account. The invitation grants access only to the named Evidence Request and remains opaque, audience-bound, short-lived, revocable and single-use.

This tranche makes due diligence operable. It does not implement relationship activation, continuation, reassessment, restriction, suspension or exit. Those remain later parts of issue #80.

## 2. Non-duplication boundary

The implementation extends shared infrastructure and adds only third-party-specific assessment state.

| Concern | Authoritative owner |
| --- | --- |
| vendor organization and supplied service | `internal/thirdparty` relationship foundation |
| assessment header, scope, lifecycle and conclusion | new third-party assessment records |
| questionnaire definition and scoring configuration | existing Monitoring Form Template |
| vendor request, fields, drafts, submission and artifacts | existing Evidence Request and capture domain |
| external access | existing invitation and capture-session domain |
| onboarding/review work | canonical Matter with a vendor-review type |
| deficiencies and remediation | linked canonical `VENDOR_DEFICIENCY` Matters, Actions and outcome checks |
| authority, delegation and conflict handling | current authority routing and command guard |
| durable setup and retry | existing runtime, maintenance-job and outbox infrastructure |
| source-backed prefill | existing Source Connection/View/Binding resolution |
| notifications | shared notification capability; no vendor email subsystem |

There will be no vendor-specific questionnaire engine, upload store, approval table, task system, scheduler, notification framework, report engine or importer.

## 3. Domain model

### 3.1 Third-party assessment

A `third_party_assessment` represents one review episode for one service relationship.

Required fields:

- tenant and bank legal-entity scope;
- relationship ID;
- review kind: `ONBOARDING` in this tranche;
- stable episode key;
- status;
- exact form-template ID and version;
- current Evidence Request ID when provisioned, plus versioned request-link history;
- linked vendor-review Matter ID when provisioned;
- collection deadline;
- started-by verified principal and start time;
- submitted, review-started and completed times;
- reviewer principal from verified command context;
- structured conclusion, uncertainty, rationale and next-review recommendation;
- optimistic version and timestamps.

Statuses are:

1. `SETUP_PENDING` — the start command committed and durable setup is pending;
2. `READY_TO_SEND` — the review Matter exists and the owner must confirm recipient and deadline; an idempotently created Evidence Request may already be linked after an interrupted send attempt;
3. `COLLECTING` — an invitation has been issued and the request accepts a response;
4. `SUBMITTED` — the vendor response exists but has not been accepted as sufficient;
5. `UNDER_REVIEW` — an authorized bank reviewer is evaluating the response;
6. `COMPLETED` — an authorized conclusion was recorded;
7. `CANCELLED` — collection ended without a conclusion and the reason is retained.

The stable episode key is tenant + legal entity + relationship + review kind + source trigger. A repeated command for the same current episode returns the existing assessment. A new episode requires an explicit new trigger or a later reassessment command.

### 3.2 Assessment-to-Matter links

The assessment stores its onboarding-review Matter ID and uses a narrow link record for deficiency Matters. The link owns association only; Matter remains authoritative for assignment, decisions, Actions, closure and verification.

The generic Matter domain gains a `VENDOR_REVIEW` type for the bounded onboarding review. `VENDOR_DEFICIENCY` continues to represent each material finding. A response or assessment cannot edit Matter status indirectly.

### 3.3 Assessment-to-request links

An assessment keeps ordered links to its Evidence Requests. Link purpose is `INITIAL` or `CLARIFICATION`; sequence starts at one and is unique within the assessment. The assessment identifies the current request, while prior submitted, expired, cancelled or replaced requests remain reconstructable. Each request uses the immutable origin `{THIRD_PARTY_ASSESSMENT, assessment ID, sequence}` so retries reuse the exact request and a clarification creates a new request deliberately.

### 3.4 Validated vendor documents

Uploads remain capture artifacts. A `third_party_document` record is created only when an authorized reviewer classifies and validates an uploaded artifact.

It records:

- assessment and relationship scope;
- capture artifact ID and exact artifact version;
- document type;
- issuer and document reference when applicable;
- valid-from and valid-until dates;
- scope statement;
- evidence class: vendor attestation or independent evidence;
- validation conclusion, reviewer and time;
- status: `CURRENT`, `REJECTED`, `EXPIRED` or `SUPERSEDED`;
- superseded-document reference;
- optimistic version and timestamps.

Upload, successful malware scanning, reviewer acceptance and current validity remain separate states.

### 3.5 Events and outbox

The existing `third_party_events` table will support assessment aggregate events in addition to relationship events. Material assessment commands append a versioned event and safe outbox fact in the same transaction as the authoritative change and required maintenance work.

Event payloads contain identifiers, versions and non-sensitive status only. Invitation tokens, vendor answers, recipient addresses, document contents and reviewer notes do not enter outbox payloads.

## 4. Start and provisioning workflow

### 4.1 Owner action

The vendor relationship workspace shows one dominant action, **Start due diligence**, when no current onboarding assessment exists and the actor may own the command.

The start preview displays:

- selected active form and exact version;
- known vendor and relationship facts that will be included;
- current source-prefill coverage and any degraded bindings;
- estimated completion time;
- the bank review target date;
- default Classic, Wizard or Automatic presentation mode.

The owner confirms the action. The request body cannot provide tenant, legal entity, actor, reviewer or approver identity.

### 4.2 Atomic start command

`thirdparty.assessment.start` re-evaluates verified identity, legal-entity scope, relationship version, current form version and current authority route. It writes:

- the assessment in `SETUP_PENDING`;
- assessment event;
- safe outbox event;
- one deduplicated maintenance job for setup.

These writes share one transaction. The command returns a committed receipt and must not report failure because later provisioning has not completed.

### 4.3 Idempotent setup worker

The worker uses the stable assessment origin to create or reuse one `VENDOR_REVIEW` Matter through the existing Matter service. Matter creation uses its existing stable trigger/dedupe key.

When the Matter exists, the worker links its exact ID to the assessment and moves it to `READY_TO_SEND`. Partial failure remains visible as provisioning in progress and retries safely. Operators can inspect and retry the maintenance job without manually editing assessment state. The job never contains a raw vendor contact address.

## 5. Invitation and vendor access

### 5.1 Preview before send

From `READY_TO_SEND`, the owner confirms:

- external recipient address;
- collection deadline;
- invitation expiry, which cannot exceed the request deadline or 30 days;
- purpose-bound message preview.

The canonical Evidence Request recipient stores an audience hash and safe hint, not the raw address. Existing manager-access checks determine who may issue or replace the invitation.

The verified `thirdparty.assessment.send_request` command creates or reuses the external Evidence Request and then issues the invitation. Evidence Requests gain a generic immutable origin reference consisting of origin type, origin ID and origin version; tenant-scoped uniqueness prevents an interrupted or repeated send from creating duplicate requests. The assessment links the exact request before invitation issuance. If request creation or linking committed but invitation issuance did not, the command returns a truthful partial outcome and a retry issues the invitation against the same request. Raw recipient data passes directly into the protected Evidence Request recipient boundary and is never written to third-party events, jobs or outbox payloads.

### 5.2 Secure link

The existing invitation service issues an opaque, audience-bound, revocable token with one redemption. The vendor confirms the invited address before a request-scoped capture session is created. Changing the recipient revokes active invitations and sessions atomically.

The token never appears in logs, analytics, referer-bearing links, previews or stored plaintext. The external page prevents third-party scripts and does not reveal internal tenant navigation.

### 5.3 Delivery and fallback

If a shared protected invitation-delivery adapter is configured, the verified send command passes the raw recipient and one-time link directly to that adapter after invitation creation. The adapter returns a delivery receipt that contains provider reference, status and time but not the raw address or token. Delivery is intentionally separate from the safe third-party outbox because a redacted outbox event does not contain enough information to send an email.

When delivery is not configured or fails, the workspace states **Secure link created; email not sent** and provides a controlled copy-link action to an authorized request manager. It never claims that the vendor was contacted merely because a token exists.

## 6. Shared form-template contract

### 6.1 Presentation

Form templates gain versioned presentation configuration:

- default mode: `CLASSIC`, `WIZARD` or `AUTOMATIC`;
- whether the respondent may switch modes;
- ordered sections with stable IDs, titles and concise help;
- field-to-section membership.

`AUTOMATIC` chooses Wizard for forms with more than twelve visible fields, more than two sections or any conditional section; otherwise it chooses Classic. The effective choice is captured with the request version for reconstructability.

Switching presentation changes rendering only. It does not create a new request, alter answers, change scoring or reset the server-backed draft.

### 6.2 Field types

The shared field contract supports:

- `short_text` and `long_text`;
- `email`, `telephone` and `url`;
- `integer`, `decimal`, `percentage` and `currency`;
- `date`;
- `yes_no`;
- `single_select` and `multi_select`;
- `checkbox` and `attestation`;
- `file`, `photo` and `signature`;
- `vendor_document`, which renders a typed upload plus required document metadata.

Legacy `text` and `number` fields remain readable and normalize to `short_text` and `decimal`. New templates use the explicit types.

### 6.3 Constraints

Fields use bounded declarative constraints:

- text minimum and maximum length;
- email, telephone and URL format validation;
- numeric minimum, maximum, decimal precision and step;
- currency code from an approved list;
- date minimum and maximum;
- selection minimum and maximum;
- two to fifty static choices or bounded source-backed choices;
- accepted media types;
- per-file and total-request size limits;
- maximum file count;
- attestation text and required acknowledgement.

Server validation is authoritative and applies the same contract used for rendering. HTML input attributes improve the experience but are not the enforcement boundary.

Arbitrary scripts, executable expressions and unrestricted regular expressions are prohibited. Template metadata has explicit length and cardinality limits.

### 6.4 Conditional fields

Templates support one bounded show-when condition per field or section:

- dependency on an earlier field in the same form;
- operator `EQUALS`, `NOT_EQUALS`, `IN`, `NOT_IN` or `ANSWERED`;
- one to twenty literal comparison values.

Dependency graphs must be acyclic and may be at most five levels deep. Hidden fields are not submitted and do not count as required. The server recomputes visibility before accepting a draft or final submission.

### 6.5 Scoring

Existing form scoring remains the single scoring contract. It expands only where a new typed field has an explicit bounded scoring rule. The pure evaluator is extracted or exposed for reuse by Monitoring and third-party assessment; it is not copied.

A calculated score is provisional until an authorized reviewer completes the assessment. External respondents never receive the internal score or risk band.

## 7. Prefill and provenance

Prefill resolves in this order:

1. current approved Source Binding resolution;
2. current vendor and relationship facts;
3. previously accepted reusable evidence whose relationship, document type, scope and validity period match;
4. no value.

Known vendor and relationship context appears as read-only request facts when the vendor does not need to confirm it. Editable prefills retain:

- source binding and version;
- source value and observation receipt;
- request-template and field version;
- respondent value;
- whether the respondent accepted or corrected the value.

Stale, partial, ambiguous, unavailable or schema-drifted source values appear as unresolved context and are not inserted as current answers. The vendor may provide the missing value where the request permits it. A correction never overwrites the source record.

## 8. Draft, Classic and Wizard experience

### 8.1 Server-backed draft

Evidence capture gains one request-scoped draft per active capture session. Draft writes use optimistic versioning and contain answers, visible-field state and presentation preference. Artifact IDs reference uploads already bound to the request.

The client autosaves after a short idle interval and before section navigation. It displays `Saving`, `Saved`, `Could not save` or `Access ended` with a recovery action. An autosave failure does not erase local entries. Final submission revalidates against the current request version and current field contract.

Drafts are revoked or made inaccessible when the invitation/session is revoked, recipient changes, request expires, request is cancelled or submission completes. Retention follows the Evidence Request retention policy.

### 8.2 Classic rendering

Classic mode renders all visible sections on one page with:

- a section index for longer forms;
- semantic headings and fieldsets;
- inline validation and a summary linked to invalid fields;
- sticky or repeated save/progress feedback that does not cover content;
- review-before-submit.

### 8.3 Wizard rendering

Wizard mode renders one section at a time with:

- section title and progress;
- Back and Continue actions;
- validation of the current visible section before Continue;
- server-backed save before moving forward;
- a complete review page before submission;
- preserved progress when switching mode or reopening the link.

Back never discards answers. Browser refresh or temporary network failure does not silently restart the form.

### 8.4 Accessibility and responsive behavior

Both modes use native input semantics, programmatic labels, error summaries, keyboard navigation, visible focus, status announcements and WCAG AA contrast. Controls remain usable at 320 CSS pixels and 200% zoom without horizontal scrolling. File and signature controls have non-drag and non-pointer alternatives.

## 9. Submission and review

Final submission creates the existing immutable capture submission and moves the assessment to `SUBMITTED` through a deduplicated domain reaction. Submission proves only that a response was recorded.

The internal review workspace shows:

- request and exact template version;
- source-prefilled value beside any vendor correction;
- unanswered or conditionally omitted fields;
- artifact scan state;
- document metadata supplied by the vendor;
- provisional score, coverage and critical responses;
- source freshness and limitations;
- current linked deficiencies.

An authorized reviewer may:

- accept or reject a document and record validation metadata;
- request clarification through the next ordered Evidence Request link with a new immutable origin sequence;
- create a canonical `VENDOR_DEFICIENCY` Matter from a material gap;
- record a conclusion with rationale and uncertainty;
- complete the assessment when required review conditions are satisfied.

The assessment conclusion options are `SATISFACTORY`, `SATISFACTORY_WITH_CONDITIONS`, `UNSATISFACTORY` and `INCONCLUSIVE`. These are assessment conclusions, not relationship activation decisions. Later activation policy decides which conclusions and Decisions permit activation.

## 10. Vendors workspace

The relationship detail gains a Due diligence section rather than a second dashboard.

For the current actor and state it shows one dominant next action:

| State | Dominant action |
| --- | --- |
| no assessment | Start due diligence |
| setup pending | View setup status |
| ready to send | Confirm and send request |
| collecting | Review request status |
| submitted | Review vendor response |
| under review | Record assessment conclusion |
| completed | View completed assessment |

Supporting details include recipient hint, deadline, response status, template version, source freshness, validated documents, deficiencies and current assessment version. Internal score and reviewer notes never appear in the external capture experience.

The form builder is upgraded from its current Yes/No-only editor to a shared section-and-field builder. Authors select a field type, configure only relevant constraints, preview Classic and Wizard modes and submit the exact version through the existing maker-checker lifecycle.

### 10.1 Enterprise copy gate

Every internal and external string must read like finished bank operating software. Headings name the record, task or decision; buttons use a direct verb and state the result; supporting text adds status, consequence or recovery information. Copy must not mention demos, prototypes, AI, internal architecture, implementation guarantees or product-review commentary. It must not use slogans, vague reassurance, urgency theatre, repetitive explanations or long instructional prose where a label and one sentence are sufficient.

The complete affected workflow is reviewed as one content system: Vendors workspace, due-diligence setup, form builder, invitation delivery, external collection, autosave, validation, receipt, internal review, empty/error/conflict states, notifications and API errors. Sample fixtures are explicitly labelled as sample data and remain operationally plausible. Copy-quality tests must cover reliable semantic patterns, while rendered review confirms that necessary text remains concise and usable at mobile widths.

## 11. Failure and recovery states

- **No active template:** start fails closed and tells an administrator to activate a due-diligence form.
- **Missing authority route:** the material command fails closed; no assessment is created.
- **Provisioning retry:** the committed assessment remains `SETUP_PENDING`; the maintenance job retries without duplicates.
- **Source degradation:** unresolved fields remain blank or explicitly stale; the request can proceed only where policy permits manual response.
- **Invitation expired or revoked:** access fails without revealing request existence; the request manager can issue a replacement.
- **Recipient correction:** current invitation and session are revoked before the replacement recipient becomes active.
- **Draft conflict:** the newest server draft is identified; local entries remain visible until the respondent chooses how to recover.
- **Submission conflict:** final submission is rejected against a changed request version and preserves the draft.
- **Upload unavailable or unscanned:** submission or reviewer acceptance is blocked according to the field and evidence policy.
- **Delivery failure:** the UI distinguishes link creation from email delivery and offers retry or controlled copy-link fallback.
- **Review conflict:** optimistic version conflict preserves reviewer entries and requires reload before resubmission.

## 12. Security and privacy

- Verified identity supplies every internal actor and scope.
- Vendor access is request-scoped and never grants tenant membership.
- Recipient raw addresses are limited to the protected delivery boundary; request state stores a hash and safe hint.
- Tokens and session credentials are redacted from logs, analytics and error telemetry.
- External responses cannot enumerate tenants, relationships, assessments, reviewers, scores, deficiencies or other vendors.
- Protected draft and response data use the existing capture authorization and storage controls.
- Repository queries enforce tenant and legal-entity scope before limits, counts or pagination.
- Required authority routes are re-evaluated for start, send, review and completion commands.
- Reviewer and vendor roles cannot be selected through request-body identity fields.

## 13. Performance and operational limits

Initial limits are:

- at most 20 sections and 200 fields per form version;
- at most 50 static choices per field;
- conditional dependency depth at most five;
- at most 20 comparison values per condition;
- request and draft reads by exact indexed identifiers;
- assessment lists by legal entity, status, updated time and keyset cursor;
- artifact size and count enforced by the existing storage policy plus field-specific lower limits;
- one active draft per request-scoped session;
- one current onboarding episode per stable episode key.

High-cardinality answers and artifacts remain outside assessment rows. Assessment projections use bounded summaries rather than loading every answer or artifact.

## 14. Testing and proof

### Domain and repository

- stable episode replay returns one assessment;
- tenant and legal-entity isolation precede limit and cursor handling;
- start atomically writes assessment, event, outbox and maintenance job;
- provisioning retries create one Matter and one Evidence Request;
- submission reaction advances one assessment once;
- completion requires current authority and optimistic version;
- point-in-time reconstruction returns the exact template, request, conclusion and linked Matter versions.

### Shared form and capture

- every supported field renders with its correct native control and server validation;
- legacy `text` and `number` remain readable;
- malformed constraints, cycles, excessive depth and unsupported types fail closed;
- hidden required fields do not block submission and hidden answers are rejected;
- source prefill and respondent correction retain separate provenance;
- Classic and Wizard use the same draft and final payload;
- autosave, resume, mode switch and conflict recovery preserve answers;
- request expiry, cancellation, revocation and recipient change revoke draft access;
- document artifacts cannot become validated documents without reviewer action.

### HTTP and security

- forged tenant, legal entity, owner, sender and reviewer fields are ignored or rejected;
- invitation redemption is audience-bound, one-use and non-enumerating;
- tokens and raw addresses are absent from logs, outbox payloads and API projections;
- external sessions cannot access internal scores, notes, Matters or other requests;
- manager access is required to issue, replace or copy an invitation.

### UI and accessibility

- Start due diligence through ready-to-send, collection, submission and review states;
- Classic and Wizard desktop/mobile flows;
- typed input limits and accessible validation summaries;
- autosave states and offline/transient-failure recovery;
- prefill, correction and stale-source explanations;
- expired, revoked, unavailable, conflict, delivery-failure and unsupported-field states;
- no duplicate primary actions or horizontal overflow at required viewports;
- copy-quality and WCAG regression gates.

### PostgreSQL and recovery

- real migration rollback/reapply;
- concurrent start and provisioning dedupe;
- transaction rollback leaves no partial assessment event/outbox/job state;
- worker crash between shared commands recovers by origin-key lookup;
- keyset plans use scoped indexes at representative assessment, field, draft and artifact cardinalities.

## 15. Documentation and maturity

The implementation must update the durable schema ownership map, runtime OpenAPI inventory, implementation ledger, rendered UI evidence and issue #80. `UC-TPRM-01` remains Expansion until the full onboarding decision and fail-closed activation acceptance criteria are executable. This tranche may truthfully claim operable due-diligence collection and review, not approved or active vendor relationships.
