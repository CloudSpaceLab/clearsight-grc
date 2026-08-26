# Vendor-linked work design

**Status:** Approved design, pending implementation  
**Date:** 2026-08-26  
**Scope:** Vendor relationships, Programs, issues and changes, secure external capture, review, and recovery

## Purpose

Bank teams need one dependable way to manage new and existing vendors, complete due diligence, and request work that supports a Program or an issue or change. A vendor may need to provide information, upload a document, add a bounded signature, or complete a structured form. The vendor's response is an external submission; it does not transfer the bank's ownership, review, authorization, signatory, verification, or closure responsibilities.

This change extends the existing vendor, Program, Matter, Evidence Request, form, artifact, invitation, and workflow domains. It does not create parallel task, form, document, approval, signature, or evidence systems.

## Design principles

- Programs remain the record for ongoing obligations. Matters remain the record for change, exception, findings, requests, and bounded work.
- A vendor relationship can relate to several Programs and Matters. A Program or Matter can relate to several vendor relationships.
- A vendor request has one primary work target: a Program or a Matter. Optional context may identify an existing Matter Action or Evidence Contract without changing its authority semantics.
- Capture Request status remains the source of truth for external collection. Vendor-work state adds only the bank review and handling state that Capture does not own.
- Vendor submission, bank acceptance, internal implementation, and verified outcome remain distinct.
- All material links and decisions are versioned, scoped, auditable, and reconstructable.
- Work remains usable without AI or a live integration.
- Visible copy is short, operational, and based on stored state. It does not describe internal architecture or promise unsupported automation.

## Canonical records

### Relationship links

Two relational association types preserve database integrity without polymorphic foreign keys:

1. `third_party_relationship_program_links`
2. `third_party_relationship_matter_links`

Each link stores tenant, legal entity, relationship, target, a bounded purpose code and label, effective dates where relevant, version, creator, timestamps, and an active or ended state. The product suggests common purposes such as service support, evidence provider, delivery party, and affected party, but accepts a concise bank-defined purpose. Purpose codes are validated for format and length rather than restricted to a fixed database enum, so an edge case does not require a product migration.

The general Matter link becomes the canonical relationship-to-Matter association. Assessment review and deficiency associations reference it rather than duplicating a second relationship-to-Matter truth. Existing assessment links are migrated safely and retain their assessment-specific purpose.

Link and unlink commands:

- require verified tenant, legal-entity, and principal identity;
- resolve current authority for the target and relationship;
- reject cross-scope targets;
- use expected versions;
- write the association, domain event, outbox event, and required projection job in one transaction;
- preserve ended links for point-in-time reconstruction.

### Vendor work request

`third_party_work_requests` orchestrates vendor work without replacing Capture Requests. It stores:

- tenant and legal-entity scope;
- vendor relationship ID;
- one primary target kind (`PROGRAM` or `MATTER`) and the corresponding validated link ID;
- optional Matter Action ID or Evidence Contract ID when the request supplies work for that existing object;
- purpose and concise vendor instructions;
- bank owner and reviewer principals resolved from verified authority;
- current collection request ID;
- bank review state and disposition;
- due date, version, timestamps, and completion metadata.

The bank review state is limited to what the capture domain does not represent:

- `PREPARING`
- `AWAITING_VENDOR`
- `RESPONSE_RECEIVED`
- `UNDER_REVIEW`
- `CHANGES_REQUESTED`
- `ACCEPTED`
- `CANCELLED`

`third_party_work_request_capture_links` records the immutable sequence of initial and clarification Capture Requests. Capture owns drafts, fields, presentation, invitations, sessions, artifacts, submission, expiry, and recipient truth. The work request owns why the vendor was asked, its Program or Matter relationship, and the bank's review outcome.

An accepted vendor response may satisfy a requested contribution. It cannot automatically mark a Matter Action implemented, pass an outcome check, close a Matter, or label a Program current. Those remain separate authorized commands.

## Commands and transaction boundaries

The public workflow uses material commands owned by the third-party domain:

- link or end a Program relationship;
- link or end a Matter relationship;
- prepare and send vendor work;
- request changes;
- begin review;
- accept a response;
- cancel vendor work;
- retry an incomplete delivery or cross-domain handoff.

The generic Evidence Request endpoint is hardened:

- direct creation becomes a material internal command;
- `VENDOR_RELATIONSHIP`, `PROGRAM`, and `MATTER` subjects use exact subject resolvers;
- subject existence, tenant, legal entity, visibility, and current authority are checked;
- reserved origin namespaces can be used only by their owning orchestrators;
- external invitation issuance cannot bypass the vendor-work or assessment workflow.

Cross-domain operations must not return a plain failure after committing authoritative work. Where one database transaction can cover all rows, the command commits the authoritative row, capture row, event, outbox, and job together. Where an external delivery provider is involved, the transaction records a durable delivery state first and returns a truthful partial outcome with a retry action. Recovery workers use bounded leases, stable idempotency keys, and observable retry state.

Worker-created Matters and other material records re-evaluate the current authority route using a verified service identity and the approved automation policy. Missing identity, route failure, revocation, conflict, or scope mismatch fails closed before material execution.

## Invitation and protected-data handling

- The invitation token is exchanged once and removed from the address bar and browser history immediately with replacement navigation.
- Session storage keys use the returned session identity, never token material or token fragments.
- Tokens are excluded from logs, analytics, previews, error text, and referrers.
- Replacement first revokes existing invitations and sessions durably. A failed replacement cannot leave an older capability active.
- Expired, revoked, already-used, and wrong-audience states provide a safe recovery path without confirming unrelated request details.
- Cancelling vendor work revokes active invitations and sessions and records the reason without deleting the request history.

## Bank workspace experience

### Vendors

Vendors remains a first-class sidebar destination. The workspace supports both new and existing relationships:

- searchable relationship list using vendor name, service, registration reference, and external reference;
- distinct accessible names when one vendor supplies several services;
- focused relationship details with `Overview`, `Due diligence`, and `Requests` sections;
- related Programs and issues or changes with exact deep links;
- current and previous vendor requests with due date, response state, review state, and next action;
- direct actions to start due diligence, request vendor work, update the relationship, or open a related record.

Creating a relationship searches for existing vendors and exact external identities before creating a new vendor. A user can still create a relationship when optional reference data is unavailable. Possible duplicates are shown for review; the system does not silently merge records.

### Programs and issues or changes

An open Program or Matter provides a secondary `Request vendor work` action and a `Related vendors` section. The action opens a focused workflow:

1. Select an existing relationship or add a new relationship without losing the current target.
2. Confirm how the vendor relates to the target.
3. Choose a form template or assemble the permitted collection fields.
4. Select Classic, Wizard, or Automatic presentation.
5. Review safe prefilled facts, vendor contact, purpose, deadline, effort, privacy/retention notice, and support route.
6. Send the request or save it in a recoverable preparation state.

The flow supports uploads, photos, structured inputs, attestations, and bounded signature capture through the existing form contract. A signature field records an acknowledgement image; it is not presented as a qualified electronic signature or bank authorization. Where an executed document is required, the request asks for the signed document as an upload and records the bank's review separately. Field type, length, format, choices, date bounds, number bounds, accepted file types, and conditional visibility are preserved. An unavailable source removes or labels only the affected prefill; it does not block unrelated fields.

### Review

Reviewers see:

- the submitted value and any different source-prefilled value;
- unanswered required fields and conditionally omitted fields as different states;
- validation results, relevant freshness, critical scoring responses, and explicit limitations;
- every document with an authorized open or download action before validation;
- quarantined, rejected, missing, or unavailable artifacts as non-actionable with a recovery explanation;
- exact links to related Programs, Matters, actions, contracts, and vendor relationship.

The conclusion starts blank. Scores and findings may inform the reviewer but never select a conclusion. Accepting a response requires an explicit disposition and rationale. Findings open the exact canonical issue or change and preserve the vendor workspace return state.

## External vendor experience

The magic-link page identifies the requesting institution and shows the safe request purpose, deadline, estimated effort, privacy/retention notice, and support route before collection begins. It exposes only the invitation's request.

Classic and Wizard layouts use the same fields and validation. Wizard navigation saves before changing sections. When saving fails, the vendor stays on the current section with their entries intact and sees a concise retry option. Draft resume, upload recovery, expired invitation, revoked invitation, changed request, and submission receipt states remain usable at narrow widths and with keyboard or assistive technology.

The completion receipt states what was submitted, when it was received, and what happens next. It does not imply bank acceptance or compliance.

## Flexibility and edge cases

- Multiple relationships may exist for the same vendor and legal entity when services or scopes differ.
- A request may use any active form revision that satisfies the shared contract; vendor work is not tied to one questionnaire.
- A relationship can be linked without immediately sending a request.
- A request can be prepared when delivery is unavailable and sent later without recreating the record.
- A vendor contact can be replaced through the recipient lifecycle while preserving history and revoking obsolete capabilities.
- Clarifications create a new immutable Capture Request sequence rather than changing a submitted response.
- Target closure, relationship suspension, form retirement, authority changes, and invitation expiry each have explicit handling. None silently deletes history.
- Ended links remain visible in history but are excluded from current-work defaults.
- Large relationship populations use bounded search and keyset pagination; selectors do not load all vendors in browser memory.
- Restricted Matters are filtered by the API and repository. Vendor lists and request projections do not reveal restricted targets through counts or labels.
- Unknown or stale derived state is labelled with its version or freshness; it does not become a persuasive default.

## Copy rules

Primary labels use business language: `Vendors`, `Due diligence`, `Vendor requests`, `Related vendors`, `Request vendor work`, `Review response`, `Request changes`, and `Accept response`.

Supporting text states the object, current condition, source, owner, deadline, consequence, or next action. Copy avoids promotional language, implementation terminology, broad assurances, and repeated explanations. Buttons describe the immediate result. Empty states identify the scope checked and the next valid action. Sample data remains explicitly labelled.

## Acceptance and visual proof

Implementation is complete only when tests prove:

- exact tenant/legal-entity and authority enforcement for every link and command;
- reserved origin protection and non-Matter subject validation;
- transaction or truthful partial-outcome behavior under every injected failure boundary;
- token removal, session resume, replacement revocation, cancellation, expiry, and wrong-audience recovery;
- existing and new vendor selection without silent duplication;
- Program and Matter linking, unlinking, restricted-target filtering, and history reconstruction;
- form, upload, signature, clarification, review, acceptance, and cancellation journeys;
- explicit conclusion selection and authorized document inspection;
- Wizard save-before-navigation and conflict recovery;
- exact deep links and back-navigation restoration;
- 320 px, desktop, 200% reflow, keyboard, focus, screen-reader naming, and automated accessibility checks;
- copy-quality regression across bank and external vendor surfaces.

Rendered evidence covers the Vendors workspace, Program and Matter entry points, Classic and Wizard vendor capture, autosave failure and resume, invitation terminal states, typed validation, document review, findings handoff, completed work, and narrow layouts. The highest-impact defect found during inspection is fixed and the affected state is rendered again before completion.

## Documentation and rollout

The implementation updates the product use-case catalogue, Respond and Capture specification, architecture and schema ownership documents, implementation ledger, acceptance tests, `DESIGN.md`, rendered evidence, and issue #80 without creating a duplicate issue.

Migrations are additive and reversible until existing assessment Matter links are migrated and verified. Read paths tolerate records created before the new associations. New UI actions are capability-gated; unavailable commands explain the missing authority or service state. Merge to `main` occurs only from a clean exact-HEAD checkout after backend, PostgreSQL-tagged, frontend, copy, accessibility, rendered-flow, build, and code-review gates pass.
