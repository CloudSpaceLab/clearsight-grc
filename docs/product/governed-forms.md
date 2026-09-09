# Governed Forms

Governed Forms is the reusable collection layer for vendor, internal-user and third-party work. It uses the existing Evidence Request, invitation, capture, artifact, authority, document-import and outbox foundations; it is not a parallel questionnaire or email system.

## Bank field assessment and vendor requests

Fields can carry an explicit assessment mode, required bank review, weight, named reviewer role and approved rubric. Automatic rules continue to use the existing scoring profile editor. Bank judgements are immutable corrections on an exact submitted response and field checksum; they never replace respondent answers. A bank-assessed result remains provisional until its required reviews and score coverage are complete. Automatic critical effects remain effective during combined assessment.

The vendor register shows outstanding forms, overdue forms, pending bank review and highest current assessed concern for the visible, authorized population, with its observation time. A vendor's Forms and responses section opens required-field gaps, submitted evidence, exact score contributions and bank decisions. Generic vendor form requests remain separate from due-diligence approval and relationship activation. Selecting up to 50 services reuses the distribution settings with a separate recipient per vendor; per-target creation receipts prevent duplicate requests on retry.

Result policies can be configured with policy-specific automation limits or a named eligible existing policy. They retain simulation, maker/checker, effective dates, expiry, limits, suspension, rollback and execution history. Policies explicitly select automatic submission results or completed bank assessments. Activity rows open only the issue or submitted response the current bank user may read. See the [decision brief](../design/2026-09-08-vendor-field-assessment.md) and [acceptance record](../acceptance/vendor-form-assessment.md).

Replacing only part of a response retains its outstanding concerns and labels its score as partly replaced. An old response leaves current vendor risk only when all its submitted fields have been replaced through the same authorized workflow and form revision. History remains available; partial replacement does not recompute or silently lower an earlier assessed score.

## Template lifecycle

The Forms navigation opens a searchable, filterable library. A template records its bank purpose, owner or responsible team, approved uses, tags, jurisdiction, industry, sensitivity, presentation mode, sections, typed fields and scoring policy. A field may carry a percentage weight; compliance scoring is valid only when the governed weighted population totals 100. File, date and date-time questions render their native task-appropriate controls.

Manual authoring, reusable starter templates, deterministic document proposals and AI proposals all produce an ordinary draft. DOCX and XLSX structure is retained where extraction supports it; searchable PDF text retains page anchors; XLS is converted through the bounded tabular adapter. Extraction limitations and unresolved fields remain visible. A maker must review the exact proposal, and a distinct checker must approve the revision before it becomes reusable. AI and document imports never activate a form.

Revisions are immutable. Editing creates a new draft revision. Search and saved views operate on bounded legal-entity-scoped pages, while the latest stored revision and currently reusable revision remain visibly distinct.

## Advanced scoring

A scored revision owns one normalized score profile. Risk forms use a high-is-poor direction; compliance forms use low-is-poor. The server stores the raw score in the form's stated direction and an adverse score where 100 always means greatest concern. The browser never recalculates an authoritative completed score.

The profile supports weighted, typed contributions and bounded AND, OR and NOT predicates. Advanced rules may add a contribution, set an adverse-score floor or cap, or disqualify a response. Scripts, regular expressions, SQL, network calls and AI evaluation are not accepted. Every profile defines exhaustive Low, Moderate, High and Critical adverse-score bands across 0–100. Invalid weights, question references, value types, predicate depth, floor/cap combinations or bands block approval.

Preview sends test answers to the exact stored template ID and revision. A draft must be saved before preview so the result identifies the material revision being evaluated. Completed revisions retain the profile version and checksum, raw score, adverse score, band, coverage, calculation state, contribution/rule explanation and calculation time. A calculation failure leaves the response completed but labels its score unavailable; it never substitutes a favourable value.

## Sending and access

A sender chooses the exact active form revision, subject, purpose, deadline, access expiry and one or more recipients. Each recipient is explicitly To or CC and internal or external. To creates a response task; CC receives the communication without owning completion. Supported access policies are recipient-specific magic link, shared-link email OTP and recipient-specific link plus email OTP. Every route is opaque, purpose-bound, audience-bound, expiring and revocable.

Sent forms remain manageable until their lifecycle closes. An authorized sender can add recipients, revoke a recipient, change the response deadline or access expiry, lock or reopen responses, revoke the distribution, or replace it with another approved form revision. Replacement requires an impact preview; only explicitly confirmed compatible answers may carry forward. Earlier distributions and submissions remain reconstructable.

## Recipient recovery and sign-off

Capture saves optimistic server drafts and maintains encrypted browser recovery for long forms when the network is interrupted. Recovery is scoped to the exact workspace and never restores file bytes; those fields identify what must be reselected. A stale draft produces a visible conflict instead of overwriting newer answers. Access expiry or revocation ends both the route and active sessions.

Submissions create immutable response revisions with the achieved identity assurance, sign-off summary, exact scoring policy and critical-field results. A later response supersedes rather than edits the earlier revision. Submission, evidence sufficiency, vendor approval and verified outcome remain separate states.

## Held vendor information

Vendor refresh requests may show a current bank-held value and ask the recipient to confirm it, correct it or provide a replacement. The request carries only the approved field scope and its source baseline. A bank reviewer applies or rejects each proposed change separately. Application uses optimistic concurrency against the current vendor record, reports conflicts without silent overwrite, and records a durable receipt showing what changed and what did not.

## Completed responses and response policies

The Responses workspace is a bounded legal-entity portfolio read. It filters and sorts stored current or historical response revisions by form, typed subject, score direction, raw/adverse range, concern band, calculation state and completion time. **Needs attention first** orders by adverse score, not by an ambiguous generic number. List rows contain safe response summaries; protected addresses, route selectors and answers remain outside the portfolio projection. When the exact selected response can be read, subject names are resolved through the current subject read contract; otherwise the subject name remains unavailable and the response stays reviewable.

The response detail sheet separates Answers, Documents, Review and History. Answers and submitted documents remain available when no bank review is required. A response with no scoring profile is labelled not scored and does not show empty coverage or assessment totals. A review can be saved only for the current response revision after current currency and reviewer authority are known. Historical or unknown-currency responses remain readable but block new bank judgements until the reviewer reloads current history.

Assessment and vendor-work response summaries require the exact submitted request's workflow link and current read permission before pagination. Vendor-work responses also require access to the linked Program or issue. A relationship owner cannot read a restricted work response through Forms; current workflow owners and reviewers retain their authorized reads. Revoked access and missing or mismatched submitted-request links hide both summaries and exact responses. Other Forms subjects retain their existing read rules.

For ordinary vendor form responses, a currently routed bank response reviewer can discover and inspect the submitted response through Forms, independently of the vendor relationship owner. The field's configured reviewer role determines which bank judgements that reviewer can save. This grant does not expose vendor drafts or widen access to the vendor profile, due-diligence work, Programs or restricted issues.

A governed response policy binds one exact active form revision to a typed eligible subject population, minimum coverage, score/band conditions, issue handling, new-issue limits, effective window and outcome check. The maker simulates the stored response population before submitting the policy. A distinct checker approves and activates it after current automation and authority routes are revalidated. Record-only rollout stores decisions without creating issues; enforced rollout requires prior record-only history.

Each scored response produces an append-only policy receipt for non-match, record-only match, application, reuse, suppression or failure. The first qualifying response in one adverse subject episode creates one issue. Replays and later poor responses reuse that issue until independently verified closure ends the episode. A later poor response can then open a new episode. Suspension stops new actions; rollback creates a new governed revision and routes inappropriate prior actions for review without deleting or silently closing material records.

The Vendors workspace organizes a selected relationship into Overview, Forms, Documents, Due diligence and History. Overview keeps the relationship owner, service, status and renewal facts visible without repeating every command. Forms owns generic form requests and vendor-work follow-up. Documents embeds the same submitted-document inventory used by Forms. Due diligence keeps assessment review and relationship activation together. History opens submitted response revisions and blocks bank judgements when the selected revision is historical or its current status cannot be checked.

## Communications

Communications use legal-entity profiles and governed message revisions. The rich-text editor supports headings, emphasis, lists, links and protected variables for recipient, form, deadline, expiry, support contact and secure route. Profiles can reference an inspected logo asset. Preview, impact, test send, maker-checker activation, effective dating, retirement and rollback are available from Forms. Delivery is outbox-backed and stores redacted delivery receipts, never link tokens or message bodies in logs.

Vendor registration, staff address verification and vendor certification refresh use the same protected presentation boundary. Each email has one secure action, an HTTPS fallback route, the task deadline and route expiry, and no remote image or tracking content. Address-verification links authorize only the assigned staff response; they do not transfer Matter ownership, review or sign-off. Certification submission proves receipt of the supplied ISO 27001 or PCI DSS documents, not bank acceptance.

The active reference contracts are `VENDOR-ADDRESS-VERIFICATION` and `VENDOR-CERTIFICATION-REFRESH`. Address verification records the result, method, check date, source, PDF evidence and staff attestation. Certification refresh records applicability separately for ISO 27001 and PCI DSS and requests the corresponding current PDF only when applicable. A bank reviewer accepts the evidence with rationale or requests specific changes. Matter outcome verification and closure remain separate commands under the current authority route.

## Boundaries and release evidence

Submitted file, photo and vendor-document answers are available through one scoped document inventory for Forms and Vendors. Each row identifies its immutable submission and source question; ordinary files and typed vendor documents use the same read contract. Filename and file-type filters, exact form/vendor/response scope and current/history pagination operate over authorized submitted occurrences. Draft uploads are excluded. Uploader and submitter attribution remain distinct, and missing historical attribution remains unknown.

Completed-response detail returns its own immutable answers and document occurrences. Available PDF and raster-image bytes can be opened; unsupported preview formats can be downloaded. Complete size and SHA-256 verification precedes byte delivery from the development store. Unscanned, quarantined, missing or changed bytes remain unavailable. Expiry or business rejection alone does not hide authorized history. Existing assessment decisions retain their original submitted occurrence; later reuse does not inherit acceptance. The inventory remains read-only; the assessment workflow below owns reconciliation and review. See the [submitted-document API](../../api/submitted-documents.md) and [read-model ownership and deployment bounds](../architecture/submitted-document-reads.md).

- A vendor assessment created from the Vendors workspace remains assessment-scoped; a generic distribution cannot impersonate that origin or silently advance its review.
- Protected addresses, OTP material and route selectors are not returned in list projections or logged.
- Template and distribution lists use legal-entity-scoped keyset pagination with bounded page sizes.
- PostgreSQL integration exercises 1,000 templates and 400 distributions and verifies isolation, pagination and index selection.
- Rendered evidence covers the template library, filtered-empty search, sent-form management, mobile amendment with native calendar inputs and immutable response revisions.
- The reference vendor-certification form is installed through ordinary draft, maker submission and distinct-checker activation. It asks whether each applicable ISO 27001 or PCI DSS record is current, requests a PDF only for a current record and retains a versioned compliance score profile.
- The release journey proves score calculation, response filtering, policy execution, replay, adverse-episode Matter reuse, verified episode closure and a later new episode without a static API response or browser metric.

### Fictional sample previews

Non-production demo mode permits protected preview/download of only the exact shipped fictional samples as an explicit exception to the ordinary unscanned read gate above. Forms and Vendors show **Demo check complete** with **No antivirus scan was performed** before their content and download actions. This is a simulated demonstration treatment: the file remains `STORED_UNSCANNED`, no clean scan receipt is created, and evidence acceptance retains its existing scan and review requirements. Genuine pending uploads retain **Safety check pending**. A sample with changed stored bytes, an unknown manifest, quarantine or deletion remains unavailable; production refuses demo mode.

Production acceptance still requires the tagged PostgreSQL suite, delivery-provider configuration, object scanning/storage configuration, representative bank-user timing and the hosted smoke test for the deployed commit.

## Existing vendor evidence in due diligence

An assessment owner can prepare its request with a contact and deadline before sending. Preparation issues no invitation or email. A currently authorized reviewer can link an exact, current vendor-submitted document from the same service to a document requirement, with a reason. Staff-supplied documents cannot become vendor-supplied evidence through this action.

The bank checklist distinguishes Missing, Pending review, Accepted, Received and Applicability pending. Linking clears the vendor upload when its source remains usable; it starts a separate bank decision. The invited vendor receives only a safe receipt flag, without the earlier submission, reviewer identity or protected source details. Actual answers, held documents, bank acceptance and overall approval retain separate counts and records. One source may support multiple requirements, but each review targets its own field.

For an unconditional collection containing only required document items, completing the collection with held evidence moves the assessment to bank review without creating a vendor submission. An existing respondent draft is retained. An all-held vendor session shows Received instead of demanding an empty submission, and no vendor reminder is due for those fulfilled requirements. Conditional applicability still requires its controlling answers; it cannot silently disappear.

This implementation covers evidence reconciliation within existing assessment forms. The row-based draft support below extends spreadsheet interpretation; cross-policy requirement deduplication, full onboarding orchestration and expanded conditional-approval routing remain separately scoped in the [vendor journey review](../reviews/2026-09-08-vendor-journeys-requirements-and-onboarding.md). Existing policy-governed approval gates remain in force.

## Spreadsheet requirement and finding drafts — September 2026

Recognized complete finding registers also offer **Prepare finding follow-up**. The author selects one assessment identified by explicit S/N values, checks the source vendor/service/date and previews response, action/explanation, remediation owner, proposed date and supporting-evidence questions for every finding. Numbered header rows contribute assessment context even without a finding. Unmerged blank identifiers cannot inherit the previous vendor; each finding retains its own recorded responsibility. Dates, ratings and status remain historical. Missing or ambiguous source rows explain the correction needed while ordinary row proposals remain available.

Each assessment has an independent, retry-safe proposal and ordinary draft receipt. Rejecting one does not reject other source assessments; there is no parent/child acceptance. All its proposed fields and explicit assessment confirmation are required at draft creation; subsequent editing and independent approval use the normal form lifecycle. Limits apply to that assessment (200 fields, 20 sections), and at most 200 source assessments are offered. Long help is visibly shortened without deleting source quotes. No vendor identity or recipient is matched, and no request is sent or finding closed. The existing response-revision model continues to allow same-respondent amendments while the distribution remains open. See [defect-repair acceptance](../acceptance/2026-09-09-finding-followup-defects.md).

XLSX extraction now retains explicit vertical merged-cell values within their declared ranges in both source text and structured rows. Unmerged blanks stay blank; missing rows are not invented, horizontal merges do not propagate values, and malformed/overlapping ranges or contradictory populated values within inherited vertical ranges fail extraction. The extraction receipt identifies `XLSX_XML_STREAM_V4`. Older retained imports keep their original parser provenance; upload again to use the updated extraction. Field review quotes match retained row/location anchors. A competing generation worker returns the same-source stored result, including a failed outcome, rather than a fabricated success. These changes do not enable automatic assessment grouping; see the [branch review](../reviews/2026-09-09-finding-followup-branch-salvage.md).

Recognized XLSX checklists and finding registers now propose one editable field per source row. The proposal retains row anchors, applicability/timing context and review warnings; generic tables retain column-based proposals. Accepting selected fields creates an optional, unscored draft. Before approval, the author selects recipient scope, removes internal-only content, adjusts document fields and applicability controls, and sets the organization's reviewer and scoring rules. Historical dates and findings remain historical until reviewed. No vendor matching or compliance decision is inferred. Older imports require uploading again to retain structured rows. See the [acceptance record](../acceptance/2026-09-09-spreadsheet-vendor-proposal.md).
