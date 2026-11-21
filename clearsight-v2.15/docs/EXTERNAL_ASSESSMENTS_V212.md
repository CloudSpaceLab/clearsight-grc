# External assessment exchange — V2.12

## Delivered journey

External assessments now have a separate recipient form and a complete bank-side return path. The bank prepares a named request, exports a minimal HTML form, receives the completed response file into the existing assignment, and submits that assignment through the existing independent review. No full internal application or bank state is embedded in the recipient form.

The new internal routes are `external-assessments` and `external-assessment-request?id=EXREQ…`. They are discoverable from Assessment forms, named external assignments, response pages and imported-response reviews. Existing vendor onboarding assignments can be used directly; the exchange does not create a second vendor or assessment store.

### Bank preparation

1. Assign a named external person to a published questionnaire, or open an external assignment already created by vendor onboarding. The organizational target, external person, internal coordinator and independent reviewer remain separate fields.
2. Prepare the request. Select the human questions, write recipient instructions and the agreed return channel, choose the real request expiry, and optionally select exact bank reference files. Internal reviewer comments are shown only to the bank; an explicit recipient guidance field controls what is shared for a revision.
3. Review the recipient preview and export the standalone HTML form. Export records a local preparation event. It sends no email, publishes no site and creates no external account.
4. Share the exported file through the organization’s approved channel outside this application.

The coordinator needs the exact location’s `documents.edit` capability before the form offers supplied-file upload. A request containing Document questions cannot be prepared without that authority and one accountable location. A text-only request without file authority remains usable with uploads disabled. Reference selection requires `documents.export`; files that cannot be exported are excluded from the picker.

### Recipient completion

The separate form shows the named person, assessment context, cycle, instructions, response due date, request expiry and selected questions. It supports human text and numeric answers, dates, choices, attestations, matrices and fixed-row mixed grids. Only human columns are exposed. Technical collector definitions, integration credentials, bank observations, requirement mappings and internal ownership metadata are omitted.

The recipient can save an incomplete draft file, load that draft later, attach permitted Documents, read explicitly supplied reference files and create the completed response JSON. The response includes the named person’s declaration and a checksum. It is returned through the channel written into the request; the form does not make a network request or submit automatically.

Answers are also saved locally on the device when browser storage is available. Supplied document bytes are deliberately excluded from that automatic save. The UI asks the recipient to save a portable draft file to retain attachments, and clears attachment references when resuming only the device copy. The portable draft includes the supplied bytes and restores them together with their answers.

### Bank receipt and independent review

Internal request lists and direct detail routes require the selected organizational scope plus either exact `programs.view` authority at every target location or verified named responsibility for this program/assignment. An explicit `programs.view` denial overrides named responsibility. Unrelated branch contributors and report-only board users cannot inspect recipient identities or manifests through the all-location selector. Only the assignment’s actual internal coordinator can validate and receive the returned response. Before receipt, the interface displays the declared person, confirmation time, requested answers, files and checksums. The coordinator records how the named source was received and confirmed.

Receiving performs the following work without accepting the assessment:

- Validates the exact request, named person, cycle, source definition, program basis, current draft, expiry and document bytes.
- Maps only requested human answers into the canonical assignment draft. Existing bank technical cells and observations survive populated, blank and hidden recipient grids.
- Stores supplied bytes through canonical Documents, with the coordinator as actual uploader and the external person/request as source provenance.
- Creates contextual Document review records as **Submitted**, where the canonical questionnaire requires them.
- Retains a separate source receipt containing the recipient’s original declaration, original answers and response checksum.

The coordinator then completes any remaining bank-owned work and submits through `atSubmit`. The canonical submission records the external respondent separately from the actual internal submitter, pins the answers and Documents, and creates the existing Work/Inbox review task. The eligible program owner or assigned independent risk officer must accept or return the response. Receiving a file and submitting it do not change requirement conclusions or auto-accept any Document.

If the bank edits imported human answers before submission, a substantive source adjustment reason is required and recorded in the submission capture reference. The original recipient receipt remains unchanged. Review return opens the canonical next response cycle; a reissued external request links to the earlier request and can include explicitly chosen prior non-document answers plus recipient guidance. Earlier source snapshots, declarations and decisions remain in history.

## Lifecycle and source checks

Requests have stored states **Prepared**, **Exported**, **Received**, **Revoked** and **Superseded**. The display also derives **Expired**, **Source changed**, and the linked response’s review state. Reissue creates a new request identifier and nonce and retains predecessor/replacement links. Revocation requires a reason and applies only to an outstanding request. A received source is retained instead of being deleted or retrospectively revoked.

Request expiry uses the actual browser `Date.now()` clock. It is deliberately independent of the synthetic assessment reference date `S.asOf` and the response due date. In this package, `S.asOf` is 13 November 2026 and continues to control assessment freshness rules. Expiry blocks export/receipt; the recipient can still save unfinished work. Once a response has been received in time, its source history is not invalidated merely because the later request expiry passes.

The source fingerprint includes the named assignment, coordinator, reviewer, due date, exact scope, cycle/binding, published form or program definition, program digest and relevant bank-side visibility prerequisites. Receipt checks the original human-draft baseline to avoid overwriting concurrent bank work. Technical-only grid refreshes do not falsely count as human edits. Changed source definitions, program basis, assignment or draft require a fresh request.

Reference files are opt-in exact versions. The internal request pins each reference’s current version, checksum, filename, media type, size, storage mode, classification and accountable location. These internal identifiers and metadata fingerprints are omitted from the recipient manifest. During export, each source is read through canonical Documents; the exported bytes are checked against their checksum. After **all** asynchronous reads finish, the module checks the actual actor, location, program management entitlement, source fingerprint and every selected reference’s current `documents.export` decision again before returning any HTML or changing the request to Exported.

File receipt uses the Documents import hook’s preflight and post-persistence checks. Actor/scope changes, source drift or permission revocation while bytes are being stored reject the operation before answers or document metadata are committed; unused staged bytes are removed. A repeated receipt cannot create duplicate documents or submissions.

## Data boundary and limitations

This is a portable offline exchange, with local workflow controls in the standalone bank application. It does not claim production hosted authentication, an externally authenticated signature, server-side access enforcement or remote recall of copied files.

The request nonce and SHA-256 checksums associate a return with an exact request and detect inconsistent content. They are not secrets or proof that the named person personally operated the file. The recipient types a declaration; the bank coordinator records source confirmation, and an independent reviewer evaluates the response. A production hosted service would need its own identity, authorization, audited delivery, retention and server-side validation contract.

Revocation and expiry prevent later receipt in the bank workflow. They cannot erase or disable an offline copy that has already been shared. The exported form states this practical boundary. The HTML has no internal routes, bank state or app scripts. Its content security policy disables network connections, embedded objects and form actions; its inline script has only the minimized manifest, closed-schema validator and checksum helpers.

The returned JSON is treated as untrusted input. The validator rejects unexpected properties, prototype-related keys, excessive nesting, unrequested technical answers, mismatched recipient identity, changed row identity, invalid answer values, altered manifests, invalid checksums, expired/revoked requests and invalid Document content. Original files pass canonical byte-format validation rather than trusting the filename or MIME declaration alone. Base64 decoding uses linear validation plus native decoding/canonical re-encoding, avoiding an unbounded repeated regular expression on large files.

### Package limits

| Item | Limit |
| --- | --- |
| Human questions | 100 |
| Fixed grid rows / human columns | 50 / 20 |
| Supplied Documents | 8 files |
| Individual supplied Document | 5 MiB |
| Supplied Documents total | 15 MiB |
| Returned JSON | 23 MiB |
| Explicit bank reference files | 4 files, 6 MiB total |
| Request expiry | Within the next 90 days |

The recipient file supports the existing permitted PDF, PNG/JPEG, text/CSV and Office document formats. Canonical Documents remains authoritative about format signatures and unsafe files. Requests spanning multiple accountable locations do not offer supplied-file intake; scope them to one location before asking for Document answers.

## Integration contract

Load `external-assessments.js` after role experience, scoped analytics and document management. It registers routes, actions and thin wrappers; it does not trigger a startup render. Include `external-assessments.css` for the internal management screens. Recipient CSS is embedded in the exported HTML and requires no app assets.

`window.CLEAR_SIGHT_EXTERNAL_ASSESSMENTS` exposes:

| Method | Behavior |
| --- | --- |
| `create(config)` | Prepare a named, scoped request; accepts `assignmentId`, recipient text, `expiresOn`, selected `fieldIds`, optional `referenceVersionIds` and reissue settings. |
| `export(requestId)` | Asynchronously return `{fileName, html, payload}` after current-source and export checks. |
| `inspectResponse(requestId, text)` | Validate and return a reviewable response preview without mutating source records. |
| `receive(requestId, text, {captureReference})` | Asynchronously import the source into the canonical draft and return its receipt. |
| `revoke(requestId, reason)` | Revoke an outstanding local request while retaining history. |
| `requests()` / `request(id)` | Return scoped cloned read models. |
| `state(id)` / `responseRoute(id)` | Return derived lifecycle state or the exact canonical response route. |

New state is limited to `S.externalAssessmentsV212.requests` and `.receipts`. Canonical assignments, answers, submissions, Work/Inbox tasks and Documents stay in their existing stores. Supplied file bytes live in the canonical byte store; the exchange does not keep an additional base64 copy in application state.

## Verification

`tests/external_assessments_v212_checks.js` runs against the complete current source load and also executes the actual exported recipient script in an isolated DOM shell with no internal app state. All 32 external-exchange checks pass. The suite covers scoped internal read access, named-responsibility access, explicit view denial, report-only exclusion, minimized export, reference authority and races, portable draft/resume, attached-file resume, typed declarations, checksum/identity/schema tampering, wall-clock expiry, revoke/reissue, source drift, protected technical values, canonical receipt/review lineage, an exact 1 MiB file roundtrip, invalid files, actor/scope changes, persistence rollback, explicit file authority and registered navigation/actions.

The optional and hidden mixed-grid cases also submit through the canonical response command with protected observations retained. The permission-revocation test proves it reached asynchronous file persistence before the denial was applied. File-related fixtures receive explicit test grants; the product’s role checks and Document source functions remain active.

Results are recorded in `tests/external-assessments-v212-results.json`. These are actual-source functional checks, not browser layout, hosted identity or delivery tests. No browser retry, external email, publication or actual external invitation was performed.
