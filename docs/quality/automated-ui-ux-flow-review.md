# Automated UI/UX and functional defect review

ClearSight merge readiness uses the `UI/UX flow review` GitHub Actions check rather than screenshot approval or reviewer sign-off.

The check builds the deterministic stakeholder application and exercises it as rendered in Chromium. Screenshots are retained to diagnose failures, but screenshot hashes do not determine whether a change passes. PASS is produced only by executable assertions over usability, state transitions, responsive behavior, accessibility and functional outcomes.

## Defects the gate must catch

The review fails when it detects, among other things:

- sticky or fixed chrome covering the work, selected target or current action;
- horizontal overflow, broken mobile reflow or an invalid 200% zoom layout;
- undersized exposed controls on field-device flows;
- browser and console errors;
- unavailable, forbidden, stale, empty or conflict states rendered with misleading semantics;
- enabled actions with missing or whitespace-only required input;
- duplicate commands while a mutation or submission is in flight;
- a completed flow without the expected receipt or updated state;
- grammatically incorrect operational counts or unnecessarily noisy timestamps;
- accessibility A/AA violations across core routes and failure states;
- interaction bundles above the agreed JavaScript chunk, total compressed JavaScript and compressed CSS budgets.

## Executable flow coverage

The current review exercises Today, Program, Vendor, Matter, Evidence, Import and Configure workspaces across light/dark modes and desktop, tablet, mobile and 320-pixel reflow layouts. It includes loading, empty, unavailable, partial-degradation and permission-denied states; authority disclosure; evidence entry/review/receipt; not-found, expiry and optimistic conflict; external field-visit upload/signature; vendor due-diligence start, request, review and delivery recovery; vendor work linked to exact Programs and issues/changes; Automatic, Classic and Wizard collection choices; typed request inputs; partial delivery and retry; submitted-answer and document review; change preparation; accepted history; lifecycle disclosure; operating mutations; and Program review acknowledgement.

The vendor-work reflow checks fail when the panel, form, card, response or typed control is clipped by its containing Program or issue/change record. They also require answer labels, values and provenance to stack into one readable column at 390px and 320px. Document actions remain available only for artifacts reported as available by the response contract.

A dedicated behavioral defect runner additionally checks deep-link visibility beneath sticky headers, fixed-navigation overlap, actual 200% CSS-pixel reflow, required-rationale validation, in-flight double-submit prevention, concise operational copy and mobile target sizing.

## Review receipts

`web/ui-evidence/defects.json` records the behavioral scenarios and any blocking defect. `review.json` combines those results with rendered-state coverage, accessibility and bundle budgets. `review.md` is written to the GitHub Actions step summary. Screenshots, logs and digests are retained as diagnostic artifacts only.

Any blocking finding exits non-zero. No manual interpretation is required for the merge gate.
