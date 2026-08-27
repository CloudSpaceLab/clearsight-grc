# Governed reusable forms, distribution, refresh and sign-off design

**Date:** 2026-08-27

**Status:** Approved direction, revised after repository and reference-system audit

**Maturity target:** End-to-end usable, reusable collection forms for internal and external respondents, including advanced template authoring, secure reusable response links, compliance scoring, document-to-form conversion, governed AI assistance, branded communications and focused updates of held vendor data or expiring documents

## Problem

ClearSight can already define governed form revisions, collect typed answers, issue purpose-bound external invitations, assign internal respondents, capture attestations and signatures, and run vendor onboarding or reassessment episodes. The remaining experience is fragmented:

- form authoring is discovered through a Program monitoring screen even though forms are needed as reusable legal-entity assets across Programs, vendor relationships, issues and changes, and standalone internal or third-party requests;
- there is no direct Forms workspace for finding templates, composing distributions, monitoring responses, managing imports or reviewing communication settings;
- the current builder exposes only a subset of the form contract and does not provide reusable field groups, scored sections, strong recommendations, template search or a clear compliance-weight model;
- active forms can be reused, but template ownership and lifecycle remain coupled to Program monitoring and are not presented as a coherent template library;
- an invitation is effectively a one-time redemption followed by a short browser session, so the emailed link cannot safely reopen the same server-side response workspace until its deadline or apply an explicit access-assurance policy;
- requests model one recipient rather than a governed distribution with multiple To respondents, notification-only CC recipients, reusable direct or shared access routes, and recipient-bound access grants where email verification is required;
- long responses have server-side autosave but no immediate browser recovery for refresh, crash or temporary network loss, and the UI cannot distinguish a server-synced draft from a device-only pending change;
- generated requests cannot be safely amended, reminded, reopened or superseded through a single visible lifecycle;
- imported DOCX, XLSX and PDF artifacts already have a hardened extraction path, but their extracted structure cannot yet be reviewed as a proposed form template; legacy XLS and scanned-document OCR have no complete production path;
- the governed AI gateway cannot yet propose or revise a form contract, and there is no exact-version diff or provenance view for AI-assisted authoring;
- invitation and reminder message content, placeholders and bank branding cannot be governed through a WYSIWYG configuration surface;
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
| Versioned form lifecycle and maker-checker activation | `internal/monitoring` | Promote the existing records and commands into the canonical legal-entity form-template aggregate while preserving IDs, revisions, authority and monitoring compatibility. Do not create a second template store. |
| Classic/wizard rendering, file/photo/vendor-document fields, attestation and signature | shared capture React components | Reuse unchanged where possible. Add current-value comparison controls around the existing inputs. |
| Internal-principal and external-audience recipients | `internal/evidence` | Make vendor orchestration accept the existing recipient union. Do not create vendor-specific recipient records. |
| Opaque invitation, bounded session, revocation, replacement and deadline clamping | `internal/evidence` | Retain token and session security. Add explicit link-possession, shared-link-plus-email-OTP and direct-link-plus-email-OTP policies; repeat access until the effective deadline; and one server-side response workspace per distribution. |
| Protected invitation-delivery interface and redacted receipt | `internal/evidence` | Add a branded renderer and configured email adapter behind this interface. Do not persist or enqueue a raw token. |
| Immutable submission answers and provenance | `internal/evidence` | Treat the submission as the change proposal. Add an application receipt that links accepted fields to the resulting authoritative version. |
| Vendor identity update command, version check, authority route, event and outbox | `internal/thirdparty` | Apply approved identity corrections through this command. Do not write `third_parties` directly. |
| Onboarding, periodic and triggered assessment episodes | `internal/thirdparty` | Use periodic/triggered episodes for refresh. Add an immutable selected-field scope; do not add a separate refresh workflow. |
| Focused clarification and vendor-work change requests | `internal/thirdparty` | Reuse their exact-field filtering rules and request sequencing. |
| Document review | `internal/thirdparty` | Extend validation with explicit, version-checked supersession. |
| Bounded DOCX/XLSX/PDF extraction and artifact scanning | `internal/documentimport` | Reuse the existing artifact, scan, extraction, provenance and worker pipeline. Add a form-template proposal transformation; do not add a parallel upload service. |
| Governed OpenAI/Anthropic routing, budgets and receipts | `internal/aigateway` | Add form-template proposal and revision workloads with strict schemas and human review. Do not call a model directly from the browser. |
| Response draft autosave and optimistic versions | `internal/evidence` and shared capture | Move durable draft ownership from a short session to the distribution response workspace. Add an encrypted, expiring IndexedDB recovery cache for immediate same-browser recovery; it never becomes authoritative. |

## Non-goals

- No arbitrary table or column names in forms, APIs or browser state.
- No automatic acceptance of respondent claims into authoritative records.
- No separate vendor form builder, invitation table, signature service or email campaign system.
- No wholesale embedding of SurveyJS Creator, Form.io, Formbricks, Vensuite or another survey platform as a second source of form semantics.
- No implication that link possession proves control of an email account or the identity or authority of a signatory. Link-only access remains available when explicitly selected and is recorded as `LINK_POSSESSION` assurance.
- No browser-only draft as the authoritative record, cross-device resume mechanism or source of a submitted response.
- No arbitrary HTML, scripts, remote pixels or unvalidated placeholders in configurable email content.
- No automatic activation of a document- or AI-generated template.
- No AI dependency for expiry, staleness, field selection, validation or routing.
- No silent replacement or deletion of prior identity or document history.
- No invitation token in logs, analytics, email receipts, outbox payloads or browser previews.
- No claim that a typed or drawn signature alone supplies legal authority; authority and respondent provenance remain separately recorded.

## Repository and reference-system audit

### ClearSight foundations

The repository audit found that most difficult primitives already exist:

- `internal/formcontract` owns the typed schema, conditional behaviour, constraints and weighted 0–100 scoring calculation;
- `internal/monitoring` owns reusable versioned forms and maker-checker activation;
- `internal/evidence` owns typed requests, internal/external recipients, opaque invitations, short sessions, server-side drafts, immutable submissions and protected delivery seams;
- `internal/documentimport` already performs bounded DOCX/XLSX/PDF extraction with archive, page, sheet, row, cell, time and output limits;
- `internal/aigateway` already governs model routing, budgets, redaction, response inspection, circuit breaking and receipts;
- `internal/thirdparty` already owns versioned vendor identity, assessment episodes, document review and authority-gated update commands.

The design therefore promotes and connects these capabilities. It does not introduce another form definition, response table, import upload, model client, invitation secret or vendor update path.

### Vensuite local reference

The local Vensuite form implementation at `C:\Users\Son\cowork\vensuite` was reviewed as a product reference. Useful patterns include:

- a direct Forms route with distinct list, builder, player and response views;
- compact block insertion and searchable slash commands;
- separate normalized editor/player stores and autosave conflict feedback;
- focus and classic form layouts with real date/time/file controls;
- template previews, responsive design controls, logo placement and completion screens;
- deterministic sheet-to-form inference for field type, requiredness, placeholders, options and confidence;
- AI generation as an authoring accelerator rather than a respondent dependency;
- response detail and analytics separated from authoring.

Patterns that are not suitable as ClearSight's trust model include static prompt templates held in browser session storage, unclassified public share slugs, browser-only seven-day response state, client-shaped sharing metadata, absence of compliance weighting and no governed multi-recipient/email-template lifecycle. ClearSight retains the useful low-friction link and local-recovery interactions while making their assurance, expiry, revocation and server authority explicit.

### Open-source references

- [SurveyJS Form Library](https://github.com/surveyjs/survey-library) demonstrates a JSON-defined core separated from framework renderers, shared validation/logic, autosave, partial response and dynamic-section patterns.
- [Form.io JavaScript](https://github.com/formio/formio.js) demonstrates builder/renderer schema parity, nested components and a broad input catalog.
- [LimeSurvey](https://github.com/LimeSurvey/LimeSurvey) and its [email-template documentation](https://help.limesurvey.org/portal/en/kb/articles/email-templates) provide mature participant-token, invitation/reminder/completion and validated-placeholder patterns.
- [Documenso](https://github.com/Documenso/documenso) provides useful distinctions among signer, approver, viewer and CC recipient roles and per-recipient access.
- [Docling supported formats](https://github.com/docling-project/docling/blob/main/docs/usage/supported_formats.md) show a possible optional normalization/OCR path for difficult PDF and legacy Office documents.
- [MultiXtract](https://github.com/srivnamrata/multixtract) provides useful extractor-registry, per-page structure, degradation-reporting, image-deduplication and optional-adapter patterns. Its alpha maturity, incomplete hostile-file limits, broad catch-and-return failure behavior and lack of an accuracy corpus make it a reference rather than a production dependency.
- [PyMuPDF](https://github.com/pymupdf/PyMuPDF) provides strong PDF layout, table, image, OCR, bounding-box and AcroForm capabilities. Its AGPL-3.0-or-commercial licensing requires explicit approval before it can be embedded in proprietary ClearSight workers.
- [Microsoft Simplify-Docx](https://github.com/microsoft/Simplify-Docx) demonstrates semantic DOCX-to-JSON and Word checkbox, dropdown, text-field, list and table mappings. ClearSight should reproduce the bounded OOXML mappings it needs rather than depend on its low-activity package and custom `python-docx` fork.
- [Lexical](https://github.com/facebook/lexical) is a suitable accessible React editor framework candidate for the constrained communication WYSIWYG; the persisted contract remains ClearSight's smaller server-validated document model.

Licensing, bundle size, backend assumptions and GRC authority semantics prevent blindly embedding a reference product. Any dependency choice requires a separate license, security, accessibility, bundle and maintenance review.

## Approaches considered

### A. Extend Program monitoring forms in place

This is the smallest migration, but it keeps generic internal, vendor and third-party collection awkwardly coupled to a Program and would make direct template ownership and authority harder to explain. It is acceptable only as a compatibility phase, not the target domain.

### B. Promote the existing form records into a canonical Forms domain — selected

This preserves current IDs, revisions, commands and integrations while making the template a legal-entity asset with optional Program use. Monitoring, vendor workflows and standalone requests all reference the same exact revision. It requires a careful compatibility migration but creates no duplicate store and best matches the requested first-class form system.

### C. Embed a general survey/form platform

This could accelerate field breadth and drag/drop authoring, but it would create competing validation, scoring, identity, invitation, response and lifecycle semantics. Licensing and styling would also vary by product. Reference libraries may be used selectively after review, but an embedded platform is not the system of record.

## Core design

### 1. Promote the existing records into one form-template library

The existing monitoring form revision remains the only persisted template record during migration. Its canonical domain meaning is promoted from “Program monitoring form” to **legal-entity form template**. Existing IDs, versions, events, references, maker-checker decisions and scheduled monitoring uses remain valid. Compatibility APIs continue to resolve Program form routes to the same records while callers move to the Forms domain.

A template has legal-entity scope and optional ownership/classification metadata:

- responsible team and template owner;
- optional originating or primary Program;
- approved uses such as vendor due diligence, control testing, issue remediation, regulatory evidence, internal attestation or general collection;
- tags, jurisdiction, industry and sensitivity;
- lifecycle state, exact current revision and next review date.

A Program is no longer mandatory merely to create a reusable form. Program monitoring references an exact active form revision just as vendor assessment or another workflow does. This removes awkward Program coupling without introducing a second schema or store.

Activation retains maker-checker review. Editing an active template always creates a new draft revision; the active revision remains immutable. A distribution pins the exact template revision used at send time. Retiring a template prevents new distributions but does not invalidate existing requests, drafts or submissions.

Curated starter templates are installed as governed template records or explicitly labelled product reference fixtures. They are not hard-coded prompts or browser-session objects. The vendor empty state links to this same library and lifecycle.

### 2. Separate template, distribution, access route, grant and response workspace

The form system uses five related but distinct records:

1. **Form template and revision** — the reusable question, validation, logic, scoring, presentation and sign-off contract.
2. **Form distribution** — a generated assessment/questionnaire/request pinned to one template revision, subject, purpose, due time, reminder policy, recipient set and access policy.
3. **External access route** — the reusable direct or shared opaque URL that starts the configured external-access ceremony and expires no later than the distribution deadline.
4. **Recipient access grant** — an internal assignment, a direct link-possession grant or an email-verified external grant. The grant records the actual assurance achieved and is independently revocable.
5. **Response workspace and revisions** — the shared server-side draft for the distribution and every immutable submitted or amended revision.

Multiple To recipients collaborate on one shared response workspace. CC recipients receive notifications and status messages but have no edit authority by default. Granting response rights requires an explicit To role. A direct link may intentionally authorize its possessor; shared-link access requires selection and verification of an eligible To recipient. The chosen access policy and achieved assurance are visible in distribution review and submission provenance.

Internal recipients authenticate normally and receive work-queue assignments. External respondents may reopen the configured link repeatedly until the earlier of request deadline, configured link expiry, revocation or relationship withdrawal. Successful access produces a new short-lived request-scoped session; it does not create a new draft.

Draft ownership therefore moves from `session_id` to the distribution response workspace. Optimistic workspace versions and field-level edit provenance prevent silent overwrites. The initial release does not require CRDT or real-time co-editing: a stale save returns the latest changed fields and a focused merge/retry experience. Every save records the verified internal principal or external access grant responsible for changed fields.

The response remains editable until the effective deadline. Submission creates an immutable response revision while leaving the workspace reopenable until the deadline unless the sender explicitly locks it. A later edit and sign-off creates an amended revision that supersedes, but never deletes, the prior submission. Reviewers always see which revision is current and whether their decision was based on an older revision.

Distribution changes are versioned:

- recipients, reminder timing and an earlier/later due time within policy can be amended with impact preview and notification;
- a revoked recipient loses future redemption immediately without deleting their contribution history;
- changing form fields after send requires a superseding distribution pinned to a new template revision;
- compatible answers may be proposed for carry-forward by stable field keys, but the sender must preview and confirm the mapping; incompatible or removed fields are retained only in prior history.

### 2a. Keep long-form browser recovery without making the browser authoritative

Server autosave remains the primary durable draft and runs after a short bounded debounce and on explicit page transitions. The capture application also writes an immediate recovery envelope to IndexedDB so a refresh, browser crash or temporary network loss does not erase painful long-form work.

The recovery envelope contains only the distribution/workspace identifier, schema revision, last known server workspace version, current page, permitted scalar answers, local edit sequence and timestamps. It is encrypted with Web Crypto using a non-exportable device key scoped to the ClearSight origin and distribution. This reduces exposure to casual disk inspection but is not represented as protection from same-origin script compromise.

The browser must not cache:

- invitation secrets, OTPs, session tokens or complete email addresses;
- signature strokes/images, uploaded file bytes or protected document previews;
- fields marked `NO_BROWSER_CACHE` by the versioned template sensitivity policy.

For an unsynced file answer the envelope may retain only the field key, filename, size, media type and the fact that reselection is required. The UI states **Reselect file to upload** and never implies that the file is safe on the server.

The recovery lifetime is the earliest of the distribution deadline, access-route expiry and the legal-entity device-cache maximum, initially capped at seven days. It is purged after successful final submission and synchronization, distribution completion, revocation, expiry or **Clear saved response on this device**. A sensitive template can disable browser caching while retaining server autosave.

The capture header exposes two truthful states: **Saved to ClearSight** and **Saved on this device — waiting to sync**. On reconnect, the client supplies the cached base workspace version and changed field set. The server either applies it through optimistic concurrency or returns current field changes for an explicit field-level merge. It never silently applies last-write-wins. Cross-device resume always uses the server workspace.

### 3. Add explicit compliance scoring without weakening existing risk scoring

The existing scoring contract remains backward compatible. A form revision declares `scoring_mode` as `NONE`, `RISK` or `COMPLIANCE`. Existing forms default to their current risk semantics. The UI never labels a risk score as compliance or silently calculates one from the other.

In compliance mode:

- each scored answer maps to an achievement from 0 to 100;
- scored section weights total exactly 100%;
- scored fields within each section total exactly 100%, or a form without sections has field weights totalling exactly 100%;
- the builder shows a live remaining-weight indicator and prevents approval while totals are invalid;
- an explicitly not-applicable conditional field is removed from the section denominator and the remaining applicable field weights are normalized within that section;
- unanswered applicable fields contribute zero to a provisional score and reduce displayed coverage;
- a final compliance score is labelled final only after all required scored fields and any reviewer-assessed evidence are complete;
- critical-answer rules may set a failure band or escalation while preserving the numeric score and explanation.

File and narrative fields are not automatically treated as compliant because a file exists or text was entered. They require an explicit deterministic criterion or reviewer score with provenance. Every score stores the form revision, answer revision, scoring policy version, denominator, coverage, critical results and calculation time.

### 4. Add document-to-template proposals through the existing import pipeline

DOCX, XLSX, XLS and PDF uploads enter the existing bounded artifact, malware-scan and extraction workflow. A new `FORM_TEMPLATE_PROPOSAL` transformation consumes the extracted representation and produces a draft proposal with source anchors; it does not create or activate a template directly.

Deterministic transformation runs first:

- workbook headers, data types, repeated values and sample distributions suggest labels, field types, placeholders, requiredness, validation and dropdown options;
- document headings, numbered questions, tables, checkboxes, blank answer regions and instructions suggest sections, questions, choices and help text;
- every proposed field carries extraction confidence and its source page, sheet, cell range, paragraph or table anchor;
- ambiguous content remains visibly unresolved instead of being guessed.

The existing bounded Go DOCX/XLSX/PDF extractors stay authoritative. MultiXtract is not added as a second upload or extraction service. ClearSight selectively ports or reimplements only useful patterns: a common element model, explicit per-page/per-sheet provenance, text/table/image/link separation, image filtering and duplicate-reference handling, and a structured degradation list. Every adapter must return `EXTRACTED`, `PARTIAL`, `UNSUPPORTED`, `TRUNCATED` or `FAILED`; caught errors may not become an empty successful document.

DOCX parsing gains bounded OOXML mappings inspired by Simplify-Docx for content controls, legacy form fields, checkboxes, dropdowns, text inputs, numbering, indentation and tables. It retains ClearSight's archive-entry, expanded-byte, compression-ratio, output and time limits and does not enable an XML “huge tree” mode for untrusted files.

Legacy XLS support is added behind the same bounded extractor interface using a maintained parser or isolated LibreOffice conversion worker. Scanned PDFs use a configured OCR adapter only when ordinary extraction reports insufficient text. OCR and legacy conversion run in CI/production workers with explicit size, time, page, memory and output limits; they do not require installing a heavy Docker/model stack on the user's development machine.

PyMuPDF may be evaluated behind an optional PDF adapter for layout blocks, bounding boxes, tables, page images, OCR fallback and AcroForm widgets. It is not added to the default build unless commercial licensing is approved or counsel confirms a compatible distribution model. Docling may likewise be evaluated for difficult layout cases. Neither replaces the current path until a fixed representative corpus proves a material accuracy gain without weakening resource, failure, provenance or recovery semantics.

The evaluation corpus includes native and scanned PDFs, AcroForms, DOCX content controls and legacy fields, nested/repeated tables, XLS/XLSX questionnaires, malformed archives, decompression bombs, encrypted files and deliberately ambiguous layouts. Golden expectations cover detected fields/options/types, reading order, table shape, source anchors, confidence, degradation and explicit failure—not merely that extraction returned a document.

### 5. Add governed AI authoring as proposals and exact-version diffs

The existing AI gateway gains two workloads:

- **Generate form-template proposal** from a natural-language objective and optional extracted source anchors;
- **Revise form-template proposal** against an exact draft revision, optionally limited to selected sections or fields.

The model returns a strict, bounded schema that is normalized and validated through `internal/formcontract`. It cannot supply actor identity, authority, tenant scope, activation state or arbitrary record targets. The proposal stores model alias, provider route, prompt/template version, source artifact hashes and anchors, input/output policy decisions, validation results and gateway receipt.

AI changes open in a reviewable diff showing added, removed and changed fields, logic, options, scoring and copy. The maker may accept all, accept selected changes or reject the proposal. Acceptance writes a normal draft revision; activation still requires the normal checker. Manual and deterministic creation remain fully usable when AI is unavailable.

### 6. Provide a robust authoring experience

The builder and capture renderer consume the same normalized server contract and shared conformance fixtures. Client-only condition, validation or scoring semantics are prohibited because they can make the builder preview disagree with production capture.

Authoring supports:

- drag insertion and a searchable slash/add menu with icon-plus-label field choices;
- keyboard reordering and section navigation;
- reusable field groups and section duplication using stable field keys;
- inline labels with a focused property panel for placeholder, help text, validation, options, conditional logic, scoring and record-update intent;
- recommended placeholders, common dropdown values, validation ranges and evidence types from a versioned server catalog;
- paste/import of option lists with duplicate detection and preview;
- desktop, tablet and mobile preview using the actual capture renderer;
- autosave with visible saved, saving, conflict and retry states;
- undo/redo within the draft plus immutable server revision history;
- a concise pre-approval quality panel for missing labels, invalid logic, inaccessible copy, scoring totals, unbounded files and unreachable required fields.

Vensuite's compact block insertion, separate builder/player modules, design preview, local recovery and deterministic spreadsheet suggestions are useful interaction references. ClearSight keeps recovery as an encrypted convenience cache and classifies link assurance explicitly; it does not adopt static prompt templates or client-defined form authority.

### 7. Add bounded response intent, not database mapping

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

### 8. Freeze the held value into each request

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

### 9. Respondent experience

`CAPTURE` fields retain the current controls.

`CONFIRM_OR_CORRECT` fields show a compact current-value card with the value, source/freshness where available, and one dominant choice:

- **Confirm this is accurate** records confirmation of the frozen baseline; or
- **Update this information** reveals the existing typed control populated with the held value.

`REPLACE_HELD_DOCUMENT` fields show the current document type, reference, issuer and expiry, then use the existing vendor-document upload control for the proposed replacement. The prior document is never overwritten during submission.

The review step clearly distinguishes confirmed values, proposed changes, new files and replacements. Required attestations and signature fields continue to be rendered by the existing controls. The final submission records the authenticated internal principal or external request/session provenance already held by the capture domain.

### 10. Sign-off without a second signature system

Form authors already have attestation and signature field types. The builder adds a **Require sign-off** convenience that inserts or configures a final required attestation and, when chosen, a signature field. These remain ordinary versioned fields in the form contract.

For an internal respondent, `SubmittedBy` identifies the verified principal. For an external respondent, the request, invitation audience binding and external session identify the response channel. A signature image remains a bounded PNG artifact linked through the answer. The submission review shows the exact attestation text from the form revision and the response provenance.

No separate mutable “signed” flag is introduced.

### 11. Use one recipient-role model across form distributions

Vendor assessment send, reissue and focused follow-up commands compose the same distribution and recipient-role model:

- **To — internal colleague:** a verified eligible principal is assigned to the evidence request; no magic link is created. The request appears in their work queue and submission provenance identifies them as an internal respondent.
- **To — vendor or other third party:** an approved external audience receives its own purpose-bound invitation. Vendor attestation must never be inferred when an internal user responds on the vendor's behalf.
- **CC — notification only:** receives the configured invitation/status message without a response link. CC does not imply visibility or contribution authority.

The UI uses the existing bounded internal-recipient search and repeatable recipient rows with explicit To/CC labels. External addresses are normalized and stored in the protected recipient boundary. Broad list projections show a masked address and recipient label; an authorized recipient-management read may return the exact address for that one distribution. Addresses never enter logs, analytics or general administration responses. Each external To recipient has its own eligibility, delivery, verification, revocation and redacted receipt state even when respondents enter through one shared URL.

### 12. Provide reusable links with selectable assurance

Each external distribution chooses one access policy. The UI explains its assurance and friction before send:

1. **Direct magic link (`DIRECT_MAGIC_LINK`)** — each external To recipient receives an opaque reusable URL. Possession of the URL is sufficient to open the form. This is the lowest-friction option for lower-sensitivity collection, but forwarding can transfer access. Sessions and response revisions record `LINK_POSSESSION`; the UI must not describe the respondent as email verified or identity verified.
2. **Shared link plus email OTP (`SHARED_LINK_EMAIL_OTP`)** — the recommended vendor due-diligence default. One reusable distribution URL shows only masked eligible To addresses. The respondent selects their address, receives an OTP at the exact bound address and enters it before a recipient-specific session is issued. CC addresses are never offered. Where masks collide, a configured non-sensitive contact label or domain suffix disambiguates them without revealing the full address.
3. **Direct link plus email OTP (`DIRECT_LINK_EMAIL_OTP`)** — each external To recipient receives an opaque URL and must also prove access to the bound email address. This is the default for higher-sensitivity, signatory or explicitly identity-reliant collection and may require another OTP on a new device or risk event.

Internal users continue through normal ClearSight authentication. A legal-entity policy may constrain which access modes are allowed by template sensitivity, record-update intent, signature requirement or subject type. The distribution stores the selected policy, and each session/submission stores the achieved assurance. Reviewers see that assurance next to sign-off provenance.

OTP values are generated cryptographically, stored only as a keyed hash, single-use, and expire after ten minutes by default. Verification, selection, resend and delivery are rate-limited by distribution, recipient, network risk and bounded time window. Attempts and resend counts are capped; error text does not reveal whether an unmasked address exists. OTPs and full addresses never enter logs, analytics, previews, generic outbox payloads or receipts. Revocation and recipient removal invalidate outstanding OTP challenges and sessions. A verified device may be remembered only when policy permits and never beyond the earliest of distribution deadline, access-route expiry, revocation or the configured device maximum.

The invitation service remains authoritative for the five-minute minimum, configured maximum and request-deadline ceiling. Command inputs gain an optional absolute `invitation_expires_at`; legacy duration minutes remain accepted during migration. The maximum remains 30 days unless an approved legal-entity policy explicitly permits a different bounded maximum.

The UI uses a date-and-time control with presets such as one day, seven days and “at the response due time.” Its maximum is the request due time and it previews the effective expiry before send. If a caller requests a later time, the API returns the bounded effective expiry and a machine-readable adjustment reason; the UI states that the link ends when the response is due. Replacement invitations use the same rule.

Successful access no longer consumes the route permanently. It records an access event and issues a short session bound to the route, achieved assurance, eligible recipient where verified, and response workspace. Rate limits, policy checks, revocation and maximum failed-attempt controls apply on every access. Replacing a direct route revokes its prior secret; rotating a shared route revokes the old route and all sessions derived from it. Tokens never appear in URLs subsequently loaded by analytics, referrers or preview services; the first request exchanges the secret and immediately removes the secret-bearing URL from browser history.

### 13. Govern branded form communications through the existing protected seam

The protected delivery request is extended with non-secret message data: bank display name, form title, task summary, due time, effective link expiry, selected access policy and support text. The raw recipient, direct/shared route and OTP material remain in protected fields and are never serializable.

A legal-entity **Form communications** configuration owns versioned templates for invitation, reminder, due soon, overdue or expired, changes requested, response amended, completion and internal notification. Each action supports subject, preheader, structured rich-text body, button label, locale and fallback plain text. Global defaults may be overridden by an approved form-template communication profile; distributions reference the exact effective configuration version.

The WYSIWYG editor uses a restricted structured document model rather than saving arbitrary HTML. The initial node allowlist includes paragraphs, headings, bold, italic, links, bulleted/numbered lists, dividers, callout text and one primary action button. The server validates the structure and link protocols, renders responsive HTML and plain text, and rejects scripts, remote tracking pixels, inline event handlers, unknown nodes and unsafe CSS.

An allowlisted placeholder picker exposes values such as recipient name, bank name, form title, task summary, due time, secure-link expiry, access instructions, support contact and secure form link. Required placeholders are validated before approval. Preview uses labelled sample values and never materializes a real route secret or OTP. Send-time expansion fails closed if a required protected placeholder cannot be resolved.

Branding configuration supports a versioned logo upload with file-type, dimension, size, malware-scan and alt-text checks, plus bank display name, approved colours and support details. Configuration changes require simulation, impact preview, maker-checker approval, effective dating, rollback and audit.

A responsive HTML and plain-text renderer produces:

- bank and ClearSight identity;
- the specific information requested;
- response due time and secure-link expiry;
- one primary **Open secure form** action;
- a plain URL fallback;
- access guidance appropriate to the selected policy, including a forwarding warning for direct magic links and email-verification instructions for OTP modes;
- recovery/support guidance.

A configured production email adapter implements the existing `InvitationDelivery` interface. Provider responses are normalized to the existing redacted receipt. Each distribution recipient has scheduled reminder state and delivery attempts, but the generic outbox never contains raw token, OTP or full recipient data. A protected delivery job receives only a secret reference and resolves the address/link inside the invitation boundary just before send. Retries are idempotent and stop after completion, revocation or deadline. When invitation delivery is unavailable, an authorized owner can copy the active route after a fresh authority check: a shared OTP route may be shared through another channel, while a direct magic link is shown with its link-possession warning. Reliable resend reuses the still-valid route; suspected exposure uses explicit rotation and revocation rather than silently multiplying valid secrets.

### 14. Review and apply proposed record changes

Submission remains immutable and does not update held data. The reviewer sees a field-level comparison of:

- request baseline and its version;
- respondent answer and provenance;
- current authoritative value and version;
- any conflict or newer change.

The reviewer can accept or reject each proposed change, then apply the accepted set. Identity fields are assembled into one complete `UpdateVendorIdentity` command using a fresh exact read, the baseline/current version rules and the verified reviewer/owner authority route. A conflict requires re-review; it never silently overwrites a newer value.

An immutable application receipt links request, submission, accepted field IDs, target keys, prior versions, resulting versions, actor and time. The domain event and outbox remain in the same transaction as the authoritative update. Rejected fields retain their submission history but do not change the vendor.

### 15. Document replacement is explicit supersession

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

### 16. Smart refresh reuses reassessment episodes

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

### Direct Forms workspace

**Forms** is a labelled primary navigation destination. Its tabs preserve one dominant action per state:

- **Templates:** server-side search and keyset pagination across name, owner, Program, use, tag, state and last update; create from blank, governed starter, duplicate, document import or AI proposal; preview and manage exact revisions.
- **Sent forms:** distributions grouped by due state, subject and owner; compose recipients, send, remind, amend due date, revoke/replace access, lock/reopen response, supersede or close.
- **Responses:** response workspaces and immutable submissions with completion, coverage, score, sign-off and review state; bounded filters for vendor, Program, form, owner, deadline and status.
- **Imports:** uploaded sources, scan/extraction progress, unresolved pages/sheets, proposal confidence, source-anchored comparison and retry/recovery.
- **Communications:** visible shortcut to the same governed email-template and branding configuration also available under Configure.

The template library supports grid and dense table views, saved filters, recent items and accessible bulk selection where actions share the same authority route. Counts always state their bounded population and freshness. At hundreds or thousands of records, filtering remains server-side and URLs retain search/filter state.

Template actions include create draft, edit as new revision, preview, compare, submit for approval, approve/reject, pause and retire through existing commands. Rows show owner, optional Program, lifecycle, active version, scored/not scored, field count, current distributions and last update. Empty states state the searched population and offer the next valid action without requiring a Program.

Authoring uses real date/calendar controls, compact labelled icons, clear typography, a white document canvas in light mode, corresponding dark-mode surface, and subtle blur only behind focused dialogs/drawers where required context remains visible.

### Distribution composer

- choose a reusable template and preview the exact revision;
- bind the business subject and purpose;
- add one or more To respondents and CC notification recipients;
- choose direct magic link, shared link plus email OTP or direct link plus email OTP, with the actual assurance and forwarding implications stated before send;
- set due date/time, secure-link expiry and reminder schedule with calendar controls;
- preview each communication and the exact population that will receive it;
- review field scope, held-value baselines, scoring and sign-off before send;
- show per-recipient delivery/revocation state and safe recovery actions after send.

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
- a persistent save indicator distinguishes server-synced work from encrypted device-only recovery and offers retry or **Clear saved response on this device**;
- reopening after a crash or refresh explains which answers were recovered, which are still syncing and which files must be reselected;
- shared-link access shows only masked eligible To addresses, then a focused accessible OTP step with resend timing and recovery guidance;
- keyboard, screen-reader, reduced-motion, 200% zoom and mobile layouts remain supported.

### Form communications configuration

- action tabs for invitation, reminder, due soon, overdue/expired, changes requested, amendment confirmation and completion;
- subject and preheader fields with searchable placeholder insertion;
- accessible WYSIWYG body editor with constrained formatting and keyboard toolbar;
- desktop/mobile email preview and plain-text preview;
- logo upload, alt text, approved colour controls and support details;
- sample-recipient preview, placeholder validation and provider-independent send test;
- draft, impact preview, maker-checker approval, effective date, rollback and audit history.

## Security and authority

- Every material command uses verified request identity; actor-like body fields remain ignored.
- Template, communication, branding and scoring-policy transitions retain maker-checker separation, effective dating, simulation and rollback.
- Internal recipients must have exact subject access.
- External routes remain distribution-, request- and purpose-bound, revocable and bounded by the effective deadline. Recipient/audience binding is enforced for email-OTP modes; direct magic-link mode deliberately treats link possession as its lower assurance.
- CC recipients never receive edit authority. A shared route lists only masked To recipients and requires OTP verification before binding a session to one of them.
- Short sessions remain request scoped. Repeated access resumes the same response workspace only after access-policy, purpose, tenant, legal-entity, revocation and deadline checks succeed.
- OTP challenges are hashed, single-use, short-lived and rate-limited. Full addresses, OTPs, route secrets and session tokens never enter client analytics, logs or generic delivery infrastructure.
- Draft writes require the current workspace version and retain field-level actor provenance. Conflicts never resolve by last-write-wins without user visibility.
- IndexedDB recovery is an encrypted convenience copy, not an authorization source or authoritative draft. It excludes secrets, signature material and file bytes, obeys per-field cache policy, expires automatically and cannot bypass server validation or current access checks.
- Record target resolution is server-side and allowlisted.
- Applying a response re-evaluates the current authority route and fails closed on missing identity, route failure, tenant/entity mismatch, conflict, delegation expiry or revoked responsibility.
- Invitation secret material never enters events, outbox, logs, analytics, previews or saved delivery receipts.
- Rich email content and placeholders are server validated and rendered; raw customer HTML is never sent directly.
- Imported files are scanned and bounded before extraction. Extracted or model-produced text is data, never executable instructions.
- AI proposals cannot activate templates, change authority, select hidden targets or send a distribution.
- Protected records are filtered by repository/API scope, not hidden only in the browser.

## Performance and retention

- Reuse the existing bounded reusable-form index and keyset portfolio patterns, adding legal-entity, lifecycle, owner, use/tag and updated-at indexes needed by the direct Forms workspace.
- Distribution lists use keyset pagination over legal entity, status, deadline and stable ID. Recipient and reminder work is claimed in bounded leased batches with dedupe keys.
- Response workspaces are addressed by exact distribution ID. Draft saves read/write one workspace version and changed field set; they never replay broad response populations.
- Template previews and normalized schemas are cached by tenant, legal entity and exact immutable revision. Draft content is not shared through cross-purpose caches. Device recovery envelopes are keyed by exact origin, legal entity, distribution and schema revision; their expiry is bounded and their contents are never restored before current route/session authorization succeeds.
- Import workers record parser/adapter version, source size, format, pages/sheets/cells, extraction duration, output size, proposal size, status, truncation and degradations; limits fail explicitly and recoverably.
- AI inputs are chunked from exact source anchors with per-workload budget and output cardinality limits.
- Request composition reads only selected target keys for one exact relationship.
- Current document lookup uses tenant, legal entity, relationship, document type and current status.
- Expiry maintenance uses a partial due-date index, bounded claims, leases, dedupe keys and retry/dead-letter visibility.
- Form revisions, source artifacts, proposals, distributions, recipient grants, draft edit provenance, submission revisions, communication versions, delivery receipts, request baselines, application receipts, document versions and events follow explicit retention and point-in-time reconstruction rules.

## Acceptance criteria

1. **Direct library:** an authorized user can open Forms directly, find templates through server-side search/filtering, and create one without first navigating through a Program. Existing Program monitoring forms appear with their original IDs and history.
2. **Governed lifecycle:** a maker can create and preview typed fields, sections, logic, file limits, record intents, scoring, attestation and signature; a distinct checker can activate the exact revision. Editing it creates a draft without changing active distributions.
3. **Builder parity:** the builder preview and production capture renderer pass the same condition, validation, normalization and scoring fixtures. Required unreachable fields and invalid logic block approval with actionable messages.
4. **Compliance weights:** a compliance-scored form cannot be approved unless section and field weights total 100 as defined. A completed fixture produces a reconstructable score, coverage and critical-result explanation; existing risk-scored forms retain their prior result.
5. **Document proposal:** DOCX, XLSX, XLS and text-based PDF fixtures produce source-anchored form proposals with confidence and unresolved items. DOCX form controls and PDF AcroForm fixtures retain useful field structure. A scanned PDF either uses the configured bounded OCR adapter or states the recovery action. Malformed, oversized and ambiguous files return explicit status/degradation rather than empty success. No import activates a form automatically, and an optional parser cannot become default without licensing approval and golden-corpus evidence.
6. **AI proposal:** an authorized maker can generate or revise a selected draft through the existing AI gateway, review a field-level diff and accept selected changes. Provider failure leaves manual/deterministic authoring usable and changes no active form.
7. **Multi-recipient distribution:** a due-diligence owner can send one distribution to eligible internal and external To respondents and notification-only CC recipients. Internal assignments create no token; external access follows the selected policy; CC receives no edit authority.
8. **Selectable external assurance:** direct magic-link access records `LINK_POSSESSION`. Shared-link and direct-link OTP modes verify an eligible To address and record email-verified assurance. Masked selection never exposes complete addresses or CC recipients; OTP expiry, retry, resend, rate-limit, revocation and generic-error fixtures pass.
9. **Reusable secure access:** an eligible external respondent can reopen the configured route in a new browser session before the deadline and resume the same server-side draft. Revocation, rotation, expiry or failed policy checks prevent access. Access does not expose route or OTP secrets to logs, analytics or subsequent browser history.
10. **Browser recovery:** after refresh, crash and offline edits, permitted scalar answers recover from encrypted IndexedDB with a truthful device-only status and synchronize through optimistic versions. Secrets, signatures and file bytes are absent; files require reselection; expiry/revocation/submission purges the envelope; a concurrent server edit produces a visible field-level merge.
11. **Shared response:** two To recipients can save to one response workspace. A stale conflicting save produces a visible field-level conflict rather than silently overwriting work, and edit provenance identifies each contributor and achieved assurance.
12. **Amendment until deadline:** submission creates an immutable revision. Before the deadline, an authorized respondent can reopen, edit and sign off again; a new revision supersedes the prior one and reviewers see which decisions reference stale submissions.
13. **Safe distribution amendments:** an authorized sender can amend due date, reminder policy or recipients with impact preview and notification. Changing questions requires a superseding distribution and explicit compatible-answer carry-forward preview.
14. **Due-date-aware access:** the sender can choose exact link expiry no later than the request due time and configured maximum. The UI displays the effective expiry and any adjustment before and after send.
15. **Governed communications:** a maker can edit invitation/reminder/completion messages in the constrained WYSIWYG editor, insert allowlisted placeholders, upload a validated logo, preview HTML/plain text and submit the version for checker approval. Unsafe content and missing required placeholders fail validation.
16. **Protected delivery:** with email configured, each recipient receives the correct approved action template and route for the selected policy. Delivery retries are idempotent and redacted. With email unavailable, authorized route-copy recovery preserves the same assurance warning and does not expose OTP or recipient data.
17. **Held-data refresh:** a form can request confirmation or correction of vendor legal name, registered address or website domain using the frozen request baseline. A respondent can confirm or propose a correction; submission changes no vendor record.
18. **Governed application:** an authorized reviewer can accept selected proposed identity fields. The existing vendor identity command creates the new version, event and outbox, and an application receipt links it to the exact submission revision. A newer vendor version causes a visible conflict.
19. **Document supersession:** a replacement certificate submission retains the old document. Reviewer validation supersedes the exact prior document in one transaction and preserves both versions.
20. **Deterministic refresh:** expired documents are marked by deterministic maintenance and produce owner attention. Starting the suggested refresh opens a triggered assessment with the relevant document and stale-fact fields selected.
21. **Sign-off provenance:** required attestations and signatures are visible in template preview, capture and final review; every response revision identifies internal principals or external route/grant/session provenance and its achieved assurance.
22. **Experience and compatibility:** every affected screen has loading, empty, degraded, unauthorized, conflict, success and recovery fixtures; desktop/mobile/200%-zoom renders pass accessibility, visual and copy-quality review; existing onboarding, clarification, vendor work, invitation administration and static-demo workflows remain compatible.

## Verification strategy

- `internal/formcontract`: normalization, builder/renderer conformance, logic reachability, target-catalog rejection, risk compatibility and compliance-weight/coverage/critical-rule tests.
- promoted Forms/`internal/monitoring`: migration, revision, maker-checker, legal-entity reusable lookup, optional Program reference, retirement and compatibility-route tests.
- `internal/documentimport`: DOCX content controls/legacy fields, XLSX/XLS, PDF/AcroForm/OCR fixtures, golden accuracy expectations, explicit partial/truncated/failed states, resource limits, scan gate, source anchors, confidence, retry and malicious-document tests.
- `internal/aigateway`: strict form workload schemas, budgets, redaction, source provenance, invalid output, diff acceptance and provider-degraded tests.
- `internal/evidence`: distribution recipients/roles, all three external-access policies, masked-email selection, OTP lifecycle/rate limits, achieved assurance, shared workspace versions, field conflicts, repeated access, exact expiry bounds, revocation/rotation, amendment revisions, baseline serialization, sign-off artifacts and protected-delivery redaction tests.
- communications: structured-document validation, unsafe markup/link rejection, placeholder requirements, logo validation, locale fallback, maker-checker lifecycle, protected expansion and idempotent reminder tests.
- `internal/thirdparty`: selected-scope assessment, identity apply/conflict, document supersession, expiry maintenance, dedupe, transaction/outbox and reconstruction tests.
- `internal/httpapi`: verified identity, route authority, field-target tampering and error mapping tests.
- React: direct Forms workspace, keyset search/filter state, authoring recommendations, weight editor, import/AI diff, distribution composer, access-policy guidance, masked-recipient/OTP flow, exact expiry, server/device save states, crash/offline recovery, file reselection, amend/conflict, WYSIWYG/branding, confirmation/correction, replacement, sign-off, review/apply, responsive and accessibility tests.
- Browser security: IndexedDB envelope schema/version migration, Web Crypto key lifecycle, excluded-field assertions, TTL/purge, failed decryption, revoked access, XSS controls and optimistic merge tests.
- Static demo: equivalent happy, empty, conflict, email-fallback, access-assurance, device-recovery and expiry states without weakening production authority semantics.
- Run Go unit/integration suites, migration tests, web unit tests, copy-quality regression, production build, affected UI state renderer, responsive screenshots and deployed end-to-end smoke checks before completion.

## Rollout sequence

1. Add characterization/conformance tests around the existing form contract, monitoring lifecycle, scoring, drafts, invitations, document extraction and capture renderer.
2. Promote the existing records and compatibility routes into the legal-entity Forms domain; add the direct Forms workspace and bounded template search without duplicating data.
3. Complete builder parity, recommendations, reusable groups, scoring modes/weights and pre-approval quality checks.
4. Add form-template import proposals on the existing document pipeline, bounded DOCX form-control mappings, explicit extraction states and the golden corpus; follow with bounded XLS/OCR adapters and source-anchored review. Gate any PyMuPDF or Docling adapter on licensing, security and demonstrated corpus gain.
5. Add governed AI proposal/revision workloads and exact-version diff acceptance.
6. Add distributions, To/CC roles, direct/shared routes, all three external-assurance policies, OTP challenges, grants, shared response workspaces, repeated access, encrypted device recovery and response amendments while retaining legacy one-recipient requests during migration.
7. Add governed communication/branding configuration, protected asynchronous delivery, reminders and safe fallback.
8. Add request baselines, confirmation/correction, reviewer comparison and vendor identity application receipts.
9. Add document supersession, scoped reassessments and deterministic expiry/staleness attention.
10. Complete state fixtures, load tests, accessibility/visual QA, compatibility tests, migration rollback checks and deployed end-to-end verification.

Each step must remain deployable with old forms and requests. New behavior activates only when a form revision, distribution or field uses the corresponding explicit versioned capability. Compatibility projections are removed only after all callers and stored references have migrated and rollback evidence has passed.
