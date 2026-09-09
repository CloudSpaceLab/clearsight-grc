# Vendor compliance overview evidence

The actual `VendorComplianceOverview` component is exercised with labelled sample API records. `vendor-compliance-app-*` fixtures enter the complete Vendors workspace, select Acme Processing Limited, and retain its normal headers, navigation and response sheet. `vendor-compliance-*` fixtures isolate the same component for the state matrix. Neither fixture path is included by the customer entry point.

## Reproduce

Build the evidence entry with `node node_modules/vite/bin/vite.js build --config vite.evidence.config.ts` from `web`, then serve it with Vite preview. `capture.mjs` defaults to `http://127.0.0.1:4188`; `PAGE_URL` overrides the address. It imports local pinned Playwright 1.55.0 from `.codex-tmp/playwright-ci` and the repository's axe-core. Run once normally and once with `INTEGRATED=1`; `STATES` can select comma-separated fixture names.

The matrix uses light and dark themes at 1440 x 900, 390 x 844, and 320 x 800. Fixtures cover empty, custom submitted gaps, incomplete, reused current evidence, expired evidence, partly replaced response, awaiting review, satisfactory, conditional, adverse, unavailable, pagination, authorized population, and unknown freshness. The last state preserves a dated satisfactory review while the current status is Unknown.

## Semantics and interaction

The custom submitted form is named Sample - Card processing service checks. It exposes ISO 27001 Missing, PCI DSS Expired, contractual audit rights Not met from an automatic submitted-answer rule, and a distinct reviewed VAPT failure. Its saved rule maps the submitted No answer to the contractual audit-rights requirement. Two independent evidence checks remain pending, and Incomplete remains zero. The corresponding existing response sheet contains that exact answer, saved rule/result and pending review coverage.

The script opens both the primary Review response and the row Review using keyboard Enter, checks the exact response read, and verifies Escape restores the originating action. Other cases check request, due-diligence and received-evidence navigation, cursor pagination, and Retry recovery. Fixture callbacks use named sample destinations; no real request or review command is sent.

The reused-evidence fixture reports three documents received. Past request deadlines deliberately do not age submitted evidence. Unknown freshness and unread pages cannot produce a favourable current conclusion. Unavailable reads show no invented zero counts. The restricted-population fixture returns only one authorized sample record; this is visual coverage, not a backend authorization test.

## Baseline and limitations

`before/user-vendor-overview.png` preserves the user-provided screenshot copied by the parent task. `before` also preserves four earlier vendor overview renders copied from `docs/evidence/2026-09-09-vendor-release/forms`, in both themes at 1440 and 390. A pre-change evidence build is also preserved locally at `.codex-tmp/vendor-overview-before-dist`; its exact source revision is not certified. The four themed files are prior repository evidence and are distinguished from the user-provided screenshot.

Receipts record source SHA-256 values, Chromium version, page/component geometry, button reachability, browser errors, status assertions, keyboard handoff and axe contrast results. Initial integrated viewport screenshots retain the actual post-selection scroll position; no screenshot-only scroll is added. Full-page images and response-sheet images support inspection of below-fold content.

Axe incomplete determinations remain in the receipts. Zero reported color-contrast violations does not certify every obscured or offscreen element. API derivation, exact authorization, database consistency and hosted behavior require their separate test and release evidence.

An initial partial matrix used the earlier Not assessed pagination expectation after the component had adopted Unknown; `initial-status-expectation-receipt.json` preserves that harness failure. The run was stopped before the final header build.

## Final verification

- Standalone matrix: 84/84 cases passed in `receipt.json`.
- Complete Vendors workspace: 84/84 cases passed in `integrated-receipt.json`.
- Final duplicate-finding change: 6/6 custom-gap workspace cases passed in `final-gap-receipt.json`, including the explicit assertion that the first complete requirement clears fixed navigation in the initial viewport.
- Project TypeScript check: `tsc -b --pretty false` exited zero. `verification.json` records zero final source digest mismatches.

The standalone matrix preceded the final mobile header replacement; its component source was unchanged. The complete workspace matrix includes that header replacement. The final gap receipt includes the later duplicate-finding logic and corrected sample aggregate concern, and matches all current recorded sources, including `vendors.css`. Other fixture states were unchanged by duplicate agreement handling. These runs report no horizontal overflow, unreachable overview button, browser error or axe color-contrast violation.

Inspected final images include `after/app-gaps-light-1440-viewport.png`, `after/app-gaps-light-320-viewport.png`, `after/app-gaps-dark-320-viewport.png`, `after/app-gaps-light-390-viewport.png`, unknown freshness, pagination after loading page two, unavailable recovery, and partly replaced response renders. Desktop exposes all four failures. The 390px initial view exposes all four; 320px exposes Missing and Expired above navigation with the remaining requirements available below. Review response is visible at every tested viewport. Full-page and sheet captures preserve the remaining content.

The first requirement begins at 579.6px on desktop, 576.2px at 390, and 667.8px at 320. The mobile navigation begins at 772px and 728px respectively. Both themes use the same geometry. Requirement status badges retain their full labels, including Not met, without horizontal scrolling.

## Broader Forms suite

`run-forms-suite.mjs` runs the existing 75-scenario Forms suite into this directory's separate `full-forms` folder. The single requested full run stopped after 30 completed captures at `116-forms-builder-large-performance-light-1440x900`: question update took 766ms against the existing 500ms budget (`web/scripts/forms-evidence-scenarios.mjs:507`). The failure is preserved in `full-forms/manifest.json`; later vendor document-launcher scenarios were not reached. `full-forms-source.json` records matching source hashes for the frozen vendor UI, scenario definitions, canonical runner and built entry. No product code was changed and the suite was not retried for this run.

### Timing investigation and continuation

The original Forms checkpoint recorded 287ms for the same question update and 612ms to render the 120-question builder, using the same Chromium 140.0.7339.16 and viewport. No builder source or performance assertion changed between that checkpoint and this verification. An isolated run during the coordinated idle-host window passed at 325ms for the update and 697ms for rendering. The 500ms update and 3000ms render budgets were unchanged. The original 766ms failure did not reproduce in that run. No host-load telemetry was recorded at the moment of failure, so environmental timing variation is plausible but not established as the cause; this is not a claim of a fixed performance regression.

`run-forms-continuation.mjs` selects the original scenario 116 or the remaining scenarios beginning 117 through a local module-load selection hook. It leaves their actions and assertions unchanged. Use `ONLY_SCENARIO=116-` for `full-forms-targeted` or `FROM_SCENARIO=117-` for `full-forms-continuation`. The targeted receipt has 1 passing scenario; the continuation has 44, including the vendor Documents section, keyboard return and preview cases. Together with the original 30 captures there are 75 distinct passing scenarios, without claiming one uninterrupted full-suite pass. The original failed manifest remains intact. Inspection included `full-forms-continuation/128-forms-documents-vendors-dark-320.png`.

### Sample aggregate correction

Both repository summary paths only populate highest assessed concern from a FINAL effective score and increment assessed forms alongside it (`internal/evidence/vendor_forms.go` and `vendor_forms_postgres.go`). The custom-gap sample previously combined HIGH with zero assessed forms. Its aggregate now omits that band; automatic rule failures, submitted answers, and pending independent review remain present. The desktop register correctly says No assessed concern recorded while the Compliance overview still says Action required and names all four failures.

The six final integrated gap captures were rebuilt after that fixture correction and passed again, including the explicit initial-viewport requirement check. Other fixture states were unchanged. The Forms suite ran before this evidence-only fixture rebuild; its recorded product and scenario source hashes still match, while its built entry hash differs because of the fixture correction. `verification.json` retains the distinct run outcomes and confirms that the final gap receipt has zero current source mismatches.

## Vendor identity CI navigation repair

The flow-review failure waited for Website icon available while its fact row was inside the collapsed Vendor details and record history section. `capture-premium-first-run-evidence.mjs` now opens that section in the shared `openVendor` path and waits for Edit vendor details before checking the brand gallery or entering the editor. Product controls and all existing scenario assertions are unchanged.

`run-vendor-identity.mjs` selects the canonical brand gallery and identity workflows (68 through 78) and uses pinned local Playwright. It omits unrelated introductory scenes and the presentation cover without changing the selected workflows. All 11 focused scenarios passed after rebuilding the current rebased source. `vendor-identity/manifest.json` records the 11 captures and source hashes; they match the current script, vendor component, vendor CSS and built entry. Coverage includes website and approved images, pending/unavailable/broken icons, validation, staged upload retention, optimistic conflict, permission failure, both logo-removal fallbacks, and 390px dark identity editing.

Inspected screenshots: `vendor-identity/68-vendor-brand-website-light-1440x900.png` shows the expanded facts and website-icon status; `vendor-identity/78-vendor-identity-mobile-dark-390x844.png` shows the reachable editor and focused Legal name field. Existing horizontal-overflow, first-field focus and preserved-entry assertions passed. No helper-mirroring unit test was added because all 11 real browser paths directly verify the shared navigation.

The parent task also updated the optional unscanned-document harness to open the existing Due diligence section. Its four allowed-review cases passed across desktop/mobile and light/dark; the separate `unscanned-review` receipts retain that verification. This did not change product UI.
