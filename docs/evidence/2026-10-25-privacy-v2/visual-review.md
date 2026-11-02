# Privacy v2 — independent visual and integration review

Date: 25 October 2026. Reviewer: **ChatGPT, GPT-6 Astra Pro, `/root/tabs_review`**. This reviewer inspected the source, runtime implementation and retained images independently of the implementation and capture agents.

**Result: PASS for the documented Overview / ROPA adaptation and exact activity inspector.** No remaining blocking defect was found within that scope. This is partial v2 integration; formal whole-screen acceptance remains **0/19**.

## Reviewed identity and evidence

The reviewed application and capture scripts are frozen at revision `651d0692e3d4ff221389d03cf0a795be5f565b45`, tree `69cf912b4fe37d601d585700bfab011fbfc39df3`. Later evidence/documentation commits do not extend this review to different runtime code.

| Evidence directly inspected | Images | Coverage |
| --- | ---: | --- |
| [Released baseline](before/manifest.json) | 10 | Register at 1440, 1280, 1024, 768 and 390px, light and dark |
| [Immutable original renders](after/approved-reference/manifest.json) | 15 | Overview, ROPA and selected ROPA inspector at all five widths; original light theme |
| [Final runtime](after/manifest.json) | 47 | All 30 source-paired frames, all 15 additional behavior/exception frames and both intermediate later-page empty/error frames |
| **Total direct image inspections** | **72** | All retained PNGs in this receipt |

The 15 original images inspected before the final capture were byte-identical to the final original rerenders. I independently checked all 72 retained copies against the inspected files, all image SHA-256/byte receipts, all 173 current evidence-build assets and all four capture-input files. There were no mismatches. The original HTML SHA-256 remains `4baf53940862d9a3af1fbc1be32da7b27595a22efb0646759cb91685a0cb5e8d`.

- Build digest: `c4d9f917344840eaf64d4176b77a51470b86821deded7b914e8c54f6ab757533`.
- Capture-input digest: `e62f7964e0826117058ded7b545744ce1ed892b7995a38e589d5d0f80283f3a4`.
- Runtime manifest SHA-256: `ad14abd5c1330f576623f0f82797c1c2581bbcac5d246d3193af6a291c7c931c`.

I independently replayed the final 45-state Privacy validator and all 30 original/runtime pair validators; both passed without findings. The retained [accessibility result](after/accessibility.json) reports eight passing Overview/ROPA desktop/phone states, zero axe violations and 44px primary controls. The [receipt](after/receipt.json) records the remaining hashes, browser configuration and measurements.

## Visual findings

**Header and menu.** The persistent Privacy heading, contextual identity, existing actions and source-order Overview / ROPA tabs establish a clear hierarchy in both themes. The selected tab stays visible. The register no longer sits below the entire summary.

**Register.** Search, the actual lifecycle filter, current-page population and records appear in the intended sequence. First record identities are fully visible at every required width. The phone record retains paired labels/values and a visible first detail action; values and complete words wrap without horizontal clipping. The measured comparison is:

| Width | Released first identity, y | Final first identity, y | Original first identity, y |
| --- | ---: | ---: | ---: |
| 1440 | 944.47px | 527.73px | 488.50px |
| 1280 | 944.30px | 527.73px | 488.50px |
| 1024 | 1002.55px | 556.98px | 488.50px |
| 768 | 1198.33px | 510.98px | 473.72px |
| 390 | 2008.75px | 585.52px | 554.73px |

These measurements support the record-first improvement, while preserving the visible difference from the original's tighter density. They do not establish pixel equality.

**Overview.** The four primary measures use four columns at 1440/1280px and two below the source breakpoint. Primary sections and the context rail follow the source's responsive arrangement. The corrected metric values align on desktop. Empty, stale, partial and unknown populations retain distinct visible quality labels; denied states expose no counts.

**Inspector.** The selected activity uses the 480px right sheet and full-width phone replacement. The corrected body uses its full padded width: 439px in desktop/tablet sheets and 350px on phone. Facts remain paired and contained; the close control is clear of both the record heading and missing-record content. The earlier float-induced narrowing is resolved in the final images and protected by a negative geometry check.

## Integration findings

The reviewed implementation retains exact activity/history readers, bounded collection queries and existing summary ownership. Query/status changes reset cursors; aborted responses cannot replace newer results. The mounted collection retains search/page context through record opening and return, and the workspace is keyed by verified tenant, legal entity and principal.

Final observed behavior distinguishes eight recorded activities, including one ended record, from seven current activities. Counted Overview drills use the existing ended-record query scope and a removable chip. Removing it or clearing filters restores the current population. Empty/error later pages retain a previous-page action and recover to the matching response. Recorded dates remain separate from the server's summary overdue count; empty closure blockers do not imply closure eligibility. The final fixtures record reads without application commands.

## Material source adaptations and limits

The [design decision](../../design/decisions/2026-10-25-privacy-v2.md) explicitly bounds this result:

- The source's six local destinations are reduced to supported **Overview / ROPA**, in source order. DPIAs, Breaches, Vendor reviews and Requests remain excluded.
- Overview uses native ROPA counts, lifecycle, completeness and coverage within the source composition. Department coverage, DPIA pipelines, incident series, review priorities and Copilot content are unavailable here.
- The register uses supported fields, recorded review dates and lifecycle status. There is no invented department, review-currentness filter, unavailable create/review action or inferred closure command.
- At 768px the runtime uses the shared table's labelled cards because the content container is below 700px; the original still uses a table. Shared shell/navigation, theme tokens, accessible controls and density remain in use. Shared desktop sheet padding is 20px; the original uses 25px.
- The original has a light theme only. Dark images establish readable runtime composition and interaction, rather than equivalence to an original dark design.

This review does not establish live backend authorization, four-actor execution, source-to-report reconciliation, whole-Privacy functional parity or production cutover acceptance. The synthetic fixture and scoped accessibility results retain those limits.
