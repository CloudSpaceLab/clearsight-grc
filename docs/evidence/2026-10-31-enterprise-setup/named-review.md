# Named scoped review — Enterprise Setup Organization

**Verdict: PASS for the scoped synthetic layout and interaction tranche.** No remaining concrete visual blocker was found in the retained matrix. This verdict does not establish backend authorization, material command results, full WCAG compliance, whole-screen acceptance, or production readiness.

Reviewer: **GPT-6 Astra Pro — `/root/people_integration`**. Date: **31 October 2026**. Accessibility follow-up: **`/root/ci_review`**.

## Exact reviewed artifact

- Local revision: `613f07f705c383201f576504f7debf7b84592151`.
- Published equivalent revision: `74603165763e69d00b8dc2cb8e804225053f6a68`.
- Tree: `c9adf95492efe0942d3d5803d9184ddb19ae6183`.
- Application build digest before and after capture: `28e6d798572e8d37da39a977b639694fb68842515f73b4011517ff51cfe3d4b1`.
- [Receipt](after/enterprise-setup/receipt.json) SHA-256: `3ac8caa8a93ba6eb2e5bf8e6440e6a7c75f338835be5f468b39a82ba348fafd0`.
- [Canonical image manifest](after/image-manifest.json) SHA-256: `160f61e07bd372c868a79b47a8005b366d742f2380f95882b7845afda9a65ad2`.
- Original preview: `docs/design/references/clearsight-2026-10/source/clearsight-enterprise-setup-designs/enterprise-setup-preview.html`, SHA-256 `cc7542e2a3047338b1d2fbbbe33756adbc9ee8c2084fb5920834da8aeedc04c7`.

[Final provenance](final-provenance.json) retains the complete five-file source hash set, all three capture-input identities, all image bindings, the historical baseline identity, and the contrast follow-up identity. The final archived reviewer passes from the archived input directory with no active runtime files or browser dependency.

## Direct visual coverage and source adaptation

I inspected the original, BEFORE and AFTER viewport pairs at **1440, 1280, 1024, 768 and 390px**, both runtime themes, all lower-context frames, the retained full-page frames, and all behavior frames. The original is light-only. The 30 final images that match the preceding reviewed capture retain the same direct-inspection binding by SHA-256; all 12 changed final images were directly re-inspected. Eleven changes affect only 32–78 pixels inside the existing loaded-list caption. The remaining change is the intended partial-list disclosure. [The exact comparisons](after/image-change-analysis.json) and [both image hashes](after/gate-review.json) are retained.

The runtime now follows the specialist page anatomy: Enterprise setup heading/action block; compact entity/approval banner; six ordered local tabs; searchable hierarchy with an actual legal-entity row; adjacent pending-change/loaded-structure summaries; selected-area context; lower Positions & roles and Data & identity cards; and Next actions. The shared application navigation follows the approved later navigation decision. Existing reporting, provisioning and escalation destinations remain available in the lower actions rather than adding a second navigation row above the hierarchy.

Illustrative source ownership percentages, member counts, readiness totals and invitations were not reproduced as runtime facts. The fixture's displayed structure and position counts are explicitly loaded populations. The current area's type, code, parent, path, origin, version and effective date provide the context available from the real contract. The selected Technology area correctly discloses its pending proposal and disables conflicting area changes.

| Viewport | First entity top | First area bottom | Fully visible rows, including entity | Hierarchy/summary | Context |
| --- | ---: | ---: | ---: | --- | --- |
| 1440×900 | 495.89 | 583.89 | 9 | Two columns | Beside primary |
| 1280×800 | 479 | 567 | 7 | Two columns | Below primary |
| 1024×768 | 511 | 599 | 5 | Two columns | Below primary |
| 768×900 | 475 | 563 | 9 | Stacked | Below primary |
| 390×844 | 634.11 | 733.89 | 4 | Stacked | Below primary |

These measurements hold in both themes. Initial scroll is zero in every layout. All ten layouts have zero document overflow and measured control targets of at least 44×44px, including native links, tabs and row controls. At widths ≤1300, explicit selection focuses the lower context and reveals its heading below the sticky shell; initial route loading does not jump there.

The first phone capture exposed an extra navigation row that pushed the first actual area below the viewport. Moving those secondary actions into the lower source-compatible sections corrected the problem. The desktop Departments label also now remains on one line. A later visual review found that the full truncation warning was below 500 rows; the final header now says **“500 loaded areas · More areas not shown”**, visibly at **y=544.11–562.11** in the 390×844 frame. The detailed warning remains in the DOM and explicitly limits local search. The existing scenario checks the header's actual viewport geometry, the 500 returned IDs, and both disclosures.

## Behavior and authority boundaries

All **11 scenarios** pass: keyboard/touch selection, loaded-list search and ancestor context, empty search, expansion/collapse, Back/Forward, reload/deep links, manual keyboard traversal of the six tabs, actual People/Frameworks/Imports handoffs, honest Groups/Jurisdictions unavailable states, empty/truncated/denied/failure/recovery states, read-only guards, orphan/cycle reachability, and retained editors.

The editor scenario opens Edit/Move/Retire without issuing commands. Imported area Move/Retire remain disabled; Edit starts at Choose type with disabled Propose change until an explicit managed type is selected. Finance's own proposal has no Approve/Reject controls, while the independent Technology proposal has both. Positions & roles and Data boundary open their existing focused editors. A preliminary timeout was a capture-label mismatch: the editor is correctly named Edit Imported operations. Only the selector was corrected; its type and no-command assertions remained intact.

Every recorded identity overview read is a GET with `limit=50`, and the fixture records the tenant/entity/principal obtained from `/api/v1/context`. Denied reads remain 403, failures remain 503, and recovery records 503 followed by successful reads. No commands were issued. These are synthetic browser transport checks, not proof of production authentication or authorization. The Frameworks handoff reaches the existing library, whose catalog is unavailable in this fixture; successful catalog loading is not claimed.

## Accessibility review and limits

The ten scoped axe scans report **zero violations** and retain **28 color-contrast incomplete nodes**. The extra desktop node is the **People & access header button label**. The other nodes are the `aria-hidden` **⌄ carets** on **Collapse Head Office** and **Collapse Technology**; the buttons themselves have explicit accessible names.

The [focused contrast report](after/contrast-review-ed46dd.json) is pinned to local `ed46dd743417ace1cd4af588aae2121055544ba1`, build `7d92d25660e43fdf132676b7f3099ef82430bbd763422c00a12045b0a5e5acc3`. It found solid computed CSS ratios of **16.28:1 light / 16.08:1 dark** for the header label, and **15.35:1 light / 13.38:1 dark** for the carets at desktop and phone widths. All nine sampled points across each header label hit the label in its own button. No concrete low-contrast blocker was found. All **31 CSS assets** are byte-identical in the final build; the final change concerns only the partial-list caption.

The original axe incompletes are not converted into automatic passes. Computed colors and sampled DOM hit tests do not establish pixel-level contrast or comprehensive WCAG conformance. The original uses fallback fonts and illustrative data. Real command execution, four-actor authority checks, reconciliation, hosted full CI/UI and deployment verification require their separate evidence. The **0/19 whole-screen register and production cutover block remain unchanged** by this scoped review.
