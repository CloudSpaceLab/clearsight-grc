# Vendor evidence reconciliation decision brief

The user approved a seamless vendor review with clear satisfied/pending states and explicit reuse of already submitted documents. This implementation makes collection and evidence review understandable in the existing due-diligence workspace. The broader journey review remains in `docs/reviews/2026-09-08-vendor-journeys-requirements-and-onboarding.md`.

## Job and outcome

The bank owner prepares a request, checks evidence already held, and sends only outstanding collection work. A routed reviewer can link an existing submitted document to the exact requested item. The vendor no longer owes that upload; the bank still owes any required acceptance decision. An existing file never becomes a fabricated vendor response.

## Interface

A prominent request checklist shows the scoped number of requested items and labelled summaries for vendor action, bank review and accepted evidence. Pending work appears first. Every row pairs the requested item, receipt/review state, available evidence and actor-specific action. Information received, evidence received, evidence accepted, conditional omission and unresolved applicability retain distinct labels. Accepted evidence is not presented as overall legal compliance or approval.

Final copy uses **Checklist**, **All items**, **Missing**, **Pending review**, **Accepted** and **Received**. Missing excludes unresolved applicability; the latter remains visible with **Applicability pending**. Supporting sentences do not repeat statuses. Vendor reference facts are expandable and form history follows due diligence. The document decision opens in a focused sheet with Accept/Reject so clicking Review does not leave the action below the viewport.

Use the existing protected document browser to select and preview an existing occurrence. Reconciliation requires a short reason confirming its relevance to this item. The saved result says that the document is already held and no vendor upload is needed. Current bank decisions and source metadata remain visible.

Preparation uses the real recipient/deadline and creates no invitation or email. Existing Send continues from the prepared request. Unknown applicability does not silently remove an obligation. When all collection work is fulfilled, the journey must offer the actual bank review transition rather than demand an empty vendor submission.

## Baseline and structural choice

Before-state structure: due diligence renders a scope header, lifecycle action, all submitted answers, then a separate document list, findings and conclusion. No saved reconciliation action or pre-send checklist exists. Source baseline is commit `11687656`; earlier audit/browser evidence is retained in the review document. A unified checklist was selected over another tab or standalone document dashboard because reconciliation needs the requirement context.

Desktop rows keep the request, state and evidence adjacent. Mobile replaces rows with stacked cards in the same reading order; filters wrap and actions remain reachable. Use existing Button, StatusBadge, Notice, FocusedSheet, field and document-preview contracts, semantic tokens and spacing. No new palette, density, illustration or motion is introduced. Scope/read failures display unknown/unavailable and recovery, never zero outstanding items.

## Integrity and recovery

The server owns collection receipts and current status. Reconciliation binds verified authority, current assessment/request versions, the exact source submission/field/artifact, scope and reason. Material records, audit and outbox share a transaction. Receipt is separate from supplier answers and independent bank document acceptance. Source access and safety/currency checks remain required. Prepared-request retries, reconciliation retries and stale-version recovery preserve work and prevent duplicates.

## Required evidence

Fixtures: not prepared, mixed missing/received/reused/validated items, unresolved condition, no outstanding collection, source unavailable/expired/wrong scope, permission denied, load error/retry and version conflict. Test the real callbacks and source links; render light/dark desktop, 390px and 320px, plus reflow. Verify keyboard selection, focus restoration, error placement, no overflow and no misleading compliance counts. Run affected frontend tests, copy regression, typecheck/build, backend domain/HTTP/capture tests and PostgreSQL checks for changed durable paths.
