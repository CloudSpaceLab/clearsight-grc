# Governed Forms

Governed Forms is the reusable collection layer for vendor, internal-user and third-party work. It uses the existing Evidence Request, invitation, capture, artifact, authority, document-import and outbox foundations; it is not a parallel questionnaire or email system.

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

The Responses workspace is a bounded legal-entity portfolio read. It filters and sorts stored current or historical response revisions by form, subject, score direction, raw/adverse range, concern band, calculation state and completion time. **Needs attention first** orders by adverse score, not by an ambiguous generic number. List rows contain safe response summaries; protected addresses, route selectors and answers remain outside the portfolio projection.

A governed response policy binds one exact active form revision to a typed eligible subject population, minimum coverage, score/band conditions, Matter handling, blast-radius limits, effective window and outcome check. The maker simulates the stored response population before submitting the policy. A distinct checker approves and activates it after current automation and authority routes are revalidated. Shadow mode records decisions without creating Matters; enforced rollout requires prior shadow history.

Each scored response produces an append-only policy receipt for non-match, shadow, application, reuse, suppression or failure. The first qualifying response in one adverse subject episode creates one canonical Matter. Replays and later poor responses reuse that Matter until independently verified closure ends the episode. A later poor response can then open a new episode. Suspension stops new actions; rollback creates a new governed revision and routes inappropriate prior actions for review without deleting or silently closing material records.

## Communications

Communications use legal-entity profiles and governed message revisions. The rich-text editor supports headings, emphasis, lists, links and protected variables for recipient, form, deadline, expiry, support contact and secure route. Profiles can reference an inspected logo asset. Preview, impact, test send, maker-checker activation, effective dating, retirement and rollback are available from Forms. Delivery is outbox-backed and stores redacted delivery receipts, never link tokens or message bodies in logs.

Vendor registration, staff address verification and vendor certification refresh use the same protected presentation boundary. Each email has one secure action, an HTTPS fallback route, the task deadline and route expiry, and no remote image or tracking content. Address-verification links authorize only the assigned staff response; they do not transfer Matter ownership, review or sign-off. Certification submission proves receipt of the supplied ISO 27001 or PCI DSS documents, not bank acceptance.

The active reference contracts are `VENDOR-ADDRESS-VERIFICATION` and `VENDOR-CERTIFICATION-REFRESH`. Address verification records the result, method, check date, source, PDF evidence and staff attestation. Certification refresh records applicability separately for ISO 27001 and PCI DSS and requests the corresponding current PDF only when applicable. A bank reviewer accepts the evidence with rationale or requests specific changes. Matter outcome verification and closure remain separate commands under the current authority route.

## Boundaries and release evidence

- A vendor assessment created from the Vendors workspace remains assessment-scoped; a generic distribution cannot impersonate that origin or silently advance its review.
- Protected addresses, OTP material and route selectors are not returned in list projections or logged.
- Template and distribution lists use legal-entity-scoped keyset pagination with bounded page sizes.
- PostgreSQL integration exercises 1,000 templates and 400 distributions and verifies isolation, pagination and index selection.
- Rendered evidence covers the template library, filtered-empty search, sent-form management, mobile amendment with native calendar inputs and immutable response revisions.
- The reference vendor-certification form is installed through ordinary draft, maker submission and distinct-checker activation. It asks whether each applicable ISO 27001 or PCI DSS record is current, requests a PDF only for a current record and retains a versioned compliance score profile.
- The release journey proves score calculation, response filtering, policy execution, replay, adverse-episode Matter reuse, verified episode closure and a later new episode without a static API response or browser metric.

Production acceptance still requires the tagged PostgreSQL suite, delivery-provider configuration, object scanning/storage configuration, representative bank-user timing and the hosted smoke test for the deployed commit.
