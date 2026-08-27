# Vendor Operations and Scalable Portfolios Design

**Status:** Approved design

**Date:** 2026-08-27
**Scope:** Vendor onboarding, due-diligence readiness, vendor branding, high-volume Programs and Issues and Changes navigation, and vendor linking

## Decision brief

ClearSight will make a newly created vendor relationship operational from the browser without requiring API or database intervention. Reference environments will install a governed, active vendor due-diligence form. Production tenants without an active form will receive a direct, governed setup path rather than a dead end. Vendor creation will collect website and registered address, and a website may start the existing secure brand-discovery process.

Programs and Issues and Changes will remain bounded, server-filtered workspaces at bank-scale volumes. Search and structured filters will update quickly, survive navigation, and explain the population currently shown. Linking a vendor will move from a dense inline form to an accessible focused sheet with searchable vendor choices and one dominant action.

The visual direction is calm institutional software: existing ClearSight tokens, compact but legible typography, small supporting icons with text labels, restrained cyan focus treatment, and blur only on the backdrop of a focused task. No workflow depends on AI, an external logo service, or a live integration.

## Problem and verified root cause

The hosted reference tenant can create a vendor relationship but cannot begin due diligence. The UI correctly looks for a current active reusable form with code `VENDOR-DUE-DILIGENCE`; the deployed PostgreSQL reference installer does not create one. A matching active form exists only in static-demo fixtures, so the hosted database returns no active form and the user reaches a non-actionable message.

The current web create-vendor command also omits the website field already accepted by the service, and registered address is not represented in the vendor identity model. The existing secure vendor-brand workflow therefore cannot start from the normal browser journey.

Programs and Issues and Changes already use keyset pagination and bounded search, but the UI exposes only basic text and status controls. At approximately 100 Programs and 400 matters, users cannot narrow work by the operational dimensions they know. Vendor linking is an always-visible inline form that competes with the linked-record list and provides insufficient context for choosing the correct relationship.

## Goals

- Allow a user to create a vendor with legal name, service provided, website, registered address, criticality and owner context.
- Make the hosted reference journey ready to start due diligence immediately while preserving maker-checker governance.
- Give a real tenant a complete browser-based recovery path when no active vendor due-diligence form exists.
- Start bounded, asynchronous logo discovery after saving an eligible website and retain a usable monogram fallback.
- Make Programs and Issues and Changes efficient at hundreds of records without broad client-side loading.
- Make vendor linking focused, searchable, accessible and unambiguous.
- Preserve identity, authority, evidence, audit, versioning and bounded-query rules.

## Non-goals

- Silently activating a form in a production tenant when the first vendor is created.
- Accepting an actor, approver, owner or tenant identity from browser-submitted authority fields.
- Fetching arbitrary website content in the browser or rendering third-party image URLs directly.
- Replacing the reusable monitoring-form lifecycle or approval model.
- Adding unbounded export, client-side full-population filtering or offset pagination.
- Treating a discovered logo, submitted questionnaire or completed task as proof of a verified outcome.

## Architecture and data contracts

### Vendor identity

The shared vendor identity gains an optional `registered_address` field. It is stored on the identity rather than an individual relationship so all tenant-authorized relationships to the same vendor use the same declared legal address. Create and identity-update commands accept the field, trim surrounding whitespace, preserve meaningful internal line breaks, reject content beyond the documented limit, and emit the changed value in the versioned identity event and outbox payload. Point-in-time reconstruction and repository reads include it.

The existing `website_domain` remains the canonical stored website identifier. The web form accepts either a hostname or an HTTPS homepage URL. The API normalizes it to a lower-case ASCII hostname, removes a trailing dot, and rejects credentials, IP literals, unsupported schemes, ports outside the existing safe policy, and malformed or public-suffix-only values. UI helper text explains that the site is used to identify the vendor and, when permitted, locate its public logo.

The schema migration adds the nullable registered-address column, its command/event serialization, and any indexes required by the final query plan. Existing rows remain valid and display “Not recorded” with an edit action where the actor has permission.

### Brand discovery

Saving a valid website creates or refreshes the existing bounded brand-discovery job in the same material transaction as the identity change and outbox work. Hosted reference deployment enables this existing capability explicitly. Discovery continues to enforce DNS and address validation, redirect limits, response-size and content-type limits, timeouts, private-network denial, and sanitization. The browser receives only a same-origin approved asset.

The UI shows a textual state next to the monogram or approved logo: queued, checking website, logo available, or logo not available. Failure never blocks vendor creation or due diligence. A permitted user can retry discovery after correcting the website.

### Due-diligence form readiness

The reference-data installer idempotently creates a reusable form with code `VENDOR-DUE-DILIGENCE` under the installed reference Program. It progresses through the existing `DRAFT` to `PENDING_APPROVAL` to `ACTIVE` lifecycle using separate configured maker and checker principals. Re-running installation reuses the current active version and does not create duplicate forms or approvals.

This automatic installation is limited to the clearly labelled reference environment. A real tenant with no current active form receives a “Set up due-diligence form” action. The action opens a focused setup sheet that:

1. Confirms that the checked population is the tenant’s current reusable forms and none is active for vendor due diligence.
2. Lets an authorized maker choose a Program and create a starter draft using the existing monitoring-form command path.
3. Lets the maker review the starter questions, save changes, and send the form for approval.
4. Shows the pending checker route and current state without exposing or trusting a browser-supplied approver identity.
5. Allows an independently authorized checker to review and activate the form through the existing authority route.
6. Returns to the vendor and enables “Start due diligence” after the active form is observed.

If the actor cannot create or approve forms, the sheet names the required role or routed reviewer and provides a refresh action. Authority-service failure, missing identity, tenant mismatch, conflict or revoked delegation fails closed and leaves an auditable, recoverable state.

Starting due diligence continues to instantiate a collection from the current active form version. Subsequent form changes do not rewrite an in-flight collection. Collection completion, assessment, decision and verified outcome remain distinct.

## High-volume workspace queries

All filters execute in repository queries within the verified tenant and legal-entity scope. The API never loads a broad population and filters authorization in application memory.

### Programs

The Programs list supports:

- debounced free-text search across the existing indexed working-language fields;
- lifecycle status;
- operating state or attention state;
- jurisdiction;
- “Assigned to me,” resolved from verified identity rather than a submitted actor ID;
- keyset cursor and bounded limit.

The repository uses stable ordering with an indexed unique tie-breaker. Filter changes reset the cursor. The response reports whether more rows are available; the UI does not present a page count as an enterprise total.

### Issues and Changes

The Matters list, presented in normal user copy as Issues and Changes, supports:

- debounced free-text search;
- workflow status;
- matter type;
- priority;
- linked Program;
- due condition: overdue, due in the next seven days, or no due date;
- “Assigned to me,” resolved from verified identity;
- keyset cursor and bounded limit.

Due conditions use the tenant’s effective business date and explicit date boundaries. A record with an unknown deadline appears only under “No due date”; it is never inferred as current or overdue.

### Filter interaction

Search waits approximately 300 milliseconds after typing; Enter applies immediately. Structured filters apply immediately. Active filters appear as removable chips with a “Clear filters” action. Search and filter state is encoded in the workspace route so returning from a record restores the same view. Invalid or unauthorized filter values are ignored with a clear recovery message rather than broadening the query silently.

The empty state names the checked filters, says that no matching records were found, and offers the next valid action: clear filters or create a record when authorized. Loading, error and end-of-results states remain distinct.

## User experience

### Add vendor

“Add vendor” remains one focused task. The first view contains legal name, service provided, website and registered address. Criticality and relationship context follow in the same form without hiding required fields. Website uses an appropriate URL-capable input with inline normalization feedback; address uses a labelled multiline field with its limit. Validation appears next to the affected field and submission preserves entered values.

After save, the UI opens the new vendor relationship. Due diligence is the dominant next action when an active form exists. Logo discovery runs in the background and does not delay navigation. The page shows who owns the relationship, its criticality, website, registered address, evidence state and latest due-diligence status.

### Link vendor

The existing linked-vendor list stays in context. “Link vendor” opens the shared accessible `FocusedSheet` pattern with a restrained blurred backdrop. The sheet contains:

- a heading that names the Program or Issue and Change receiving the link;
- server-side debounced vendor search;
- selectable rows with approved logo or monogram, legal name, service, criticality and relationship status;
- a labelled purpose field that explains why the link is needed;
- one primary “Link vendor” action and a secondary cancel action.

The sheet traps focus, closes on Escape, restores focus to its trigger, announces validation and submission status, and remains usable at 200% zoom. A disabled action explains what is missing. Duplicate or unauthorized links return a specific recoverable message.

### Visual and responsive behavior

The implementation uses existing semantic tokens and component patterns. Text and controls meet contrast requirements in light and dark themes. Supporting icons never replace labels and use the compact product icon size; interactive targets remain at least 44 CSS pixels. Blur is limited to the modal backdrop and degrades to an opaque overlay when backdrop filtering is unavailable or reduced transparency is preferred.

Desktop layouts keep search and common filters visible with advanced filters in a compact popover or row. Narrow layouts replace the row with a labelled filter button and focused filter sheet; they do not merely shrink desktop controls. Document-style questionnaires and evidence review use the existing light document surface in light mode and the corresponding high-contrast dark surface in dark mode. Dates use native or established accessible calendar inputs rather than free text.

## Security, authority and audit

- Request identity determines actor, tenant, legal entity and “mine” filters.
- Form creation, submission, approval and activation use the current versioned authority route and conflict rules.
- The reference maker and checker are different configured principals; production routes are never hard-coded.
- Vendor identity changes, brand jobs, append-only events, outbox entries and required maintenance jobs share the material transaction.
- Brand discovery cannot access loopback, link-local, private, metadata or otherwise prohibited destinations, including after DNS changes or redirects.
- Returned logo assets are tenant- and purpose-bound and same-origin.
- Restricted Programs, Matters and vendor relationships are excluded by repository scope, not hidden after delivery to the browser.
- Material records remain versioned and reconstructable.

## Failure and recovery states

- No active form: offer governed setup; never suggest direct database or API work.
- Draft or approval pending: show current state, routed next actor and refresh action.
- Form changed while starting: reload the current version and ask the user to confirm the new form before creating the collection.
- Brand discovery failed: retain monogram, explain the checked website and permit retry after correction.
- Search failed: keep filter state and allow retry; do not replace results with an unfiltered population.
- Cursor expired or filters changed: restart from the first bounded page with an explanatory notice.
- Link conflict: show the existing link and keep the user in context.
- Configuration or authority failure: fail closed, preserve submitted draft data where safe, and identify the recovery owner or action.

## Migrations, compatibility and rollout

The database migration is additive and nullable. API consumers that omit registered address or website continue to work. New response fields are backward compatible. Repository query additions use optional parameters and retain existing keyset behavior.

Deployment order is database migration, service/API, reference-data installer, then web application. The installer is safe to rerun. The hosted reference deployment explicitly enables secure brand discovery and runs the reference installer before smoke tests. Production tenants do not receive an active form unless their approved configuration or installation path creates one.

Rollout telemetry records bounded query latency, result-page size, form-readiness state, brand-job outcomes and workflow command failures without logging invitation tokens, protected content or fetched third-party bodies. A kill switch disables brand discovery without disabling vendor creation or due diligence.

## Required fixtures and design proof

State fixtures cover:

- new vendor with active due-diligence form;
- new vendor with no form and authorized setup actor;
- no form and unauthorized actor;
- draft awaiting an independent checker;
- logo queued, available and unavailable;
- 100+ Programs with multiple active filters;
- 400+ Matters including overdue, next-seven-days and unknown deadlines;
- empty, loading, failed and final-page lists;
- vendor-link search, duplicate link, permission failure and successful link;
- light, dark, desktop, narrow, 200% zoom, reduced motion and no-backdrop-filter modes.

The change preserves a before-state capture for the current vendor creation, no-form dead end, Program filters, Matter filters and inline vendor-link form. After-state renders are inspected at relevant desktop and mobile viewports. The highest-impact visual or interaction defect found in review is corrected and the affected render is repeated before completion.

## Test and acceptance strategy

Implementation follows red-green-refactor. Each behavior begins with a failing focused test.

Backend acceptance includes:

- idempotent reference-form installation and distinct maker/checker lifecycle;
- tenant, authority, delegation, conflict and revocation failures;
- registered-address create, update, history, outbox and reconstruction;
- website normalization and rejection cases;
- atomic brand-job creation and safe retry;
- Program and Matter filter combinations, verified “mine” resolution, stable keyset pagination and restricted-record exclusion;
- query-plan/index checks and high-cardinality load targets;
- form-version race and command idempotency.

Web acceptance includes:

- add-vendor website and address fields, inline validation and submission recovery;
- immediate due-diligence start in the installed reference journey;
- complete no-form setup and independent approval journey;
- debounced search, immediate structured filters, removable chips, restored route state and truthful result wording;
- focused vendor-link selection, duplicate handling, keyboard navigation, focus restoration and accessible announcements;
- same-origin logo rendering and textual fallback states;
- native/established date controls, copy-quality regression and axe checks.

The non-Docker local verification suite runs affected unit, component, API contract, copy-quality, accessibility and build checks. PostgreSQL integration and end-to-end browser tests run in CI. After merge to `main`, the deployed reference tenant is verified through the browser: create a vendor with website and address, observe brand status, start and complete the first due-diligence step, find records through combined filters, and link the vendor from both a Program and an Issue or Change.

## Completion criteria

The feature is complete when a bank user can perform every described workflow from the UI; the hosted reference tenant contains the active governed form; real tenants have an authorized recovery path; search remains bounded and usable at target cardinality; accessibility, responsive and degraded paths pass; audit reconstruction includes new identity data; documentation and acceptance tests are synchronized; CI is green; and the deployed `clearsight.cloudspacetechs.com` journey has been verified end to end.
