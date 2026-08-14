# Automated UI/UX flow review

ClearSight merge readiness uses the `UI/UX flow review` GitHub Actions check rather than a screenshot-approval or reviewer-sign-off gate.

The check builds the deterministic stakeholder application and exercises the product as rendered in Chromium. It fails the pull request when a governed flow, responsive state, accessibility boundary or interaction budget regresses.

## Enforced flow matrix

The current matrix produces 41 uniquely named, SHA-256-hashed screenshots while asserting behavior rather than merely capturing pixels. Coverage includes:

- Today, Program, Matter, Evidence, Import and Configure workspaces;
- light, dark, comfortable and compact presentation modes;
- 1440 px desktop, 1024 px tablet, 390 px mobile and 320 px reflow layouts;
- loading, empty, unavailable, partial-degradation and permission-denied states;
- authority disclosure, evidence entry/review/receipt, not-found, expired and optimistic-conflict paths;
- mobile focus containment, 200% zoom proxy, field-visit upload/signature and lifecycle disclosure;
- operating mutations and Program review acknowledgement.

## Automated quality boundaries

The gate combines:

1. TypeScript compilation, rendered component tests and existing axe assertions.
2. Playwright semantic flow checks using accessible roles and exact governed states.
3. Browser error, horizontal-overflow, initial-action visibility, focus-containment and critical mobile touch-target checks.
4. A rendered WCAG A/AA sweep across core route and failure states.
5. A complete evidence manifest with no missing, duplicate, stale or unexpectedly small captures.
6. Interaction bundle budgets of 500 KiB raw / 160 KiB gzip JavaScript and 32 KiB gzip CSS.

`web/ui-evidence/review.json` is the machine-readable decision receipt. `review.md` is written to the GitHub Actions step summary, and screenshots plus logs are retained as build artifacts. Any blocking finding exits non-zero; no manual interpretation is required for the merge gate.