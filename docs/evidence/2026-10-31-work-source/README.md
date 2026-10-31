# Work deadline counts and source drills — retained review

Date: 31 October 2026. Reviewer: **ChatGPT, GPT-6 Astra Pro, `/root/work_contract`**.

**PASS for the source-count drill correction and responsive measure strip.** This is partial V2 integration. Whole-screen acceptance remains **0/19** and the production cutover block remains in place.

## Reviewed identity

The [original Work HTML](../../design/references/clearsight-2026-10/source/clearsight-work-approvals-designs/work-approvals-preview.html) is unchanged, with SHA-256 `202ed9c2b22ac4e921d9ba14aa41b61644ea54cbe55abcc81a86ec10b10ee8d2`. The [decision](../../design/decisions/2026-10-31-work-source-deadline-drills.md) describes the contract and adaptation.

- Baseline: `ad81d259a79b58d7e2871587900ba1570e56ffba`, built and preserved before runtime edits.
- Final capture: `e4d9a08cb16b88fde6b3cd8657fb1a7a4a75643d`, tree `c06c30bb766d10fb8ecc5247c8c6bf627e595c5e`.
- Application build digest: `3bfc7fcf130c22bcde0ece385a909632d872cb5c47f7f74867e3c41651548c51`. Application assets are unchanged from the combined Work/People revision `9584d2891f4d2da456b5c6b5c3d7097883ee0440`; the final commit adds keyboard/touch capture assertions only.
- Final receipt SHA-256: `c8212e365cba398dec8b67db6d6535840ac0283df5d1fbad3319ed9f45cea576`.

The [verification record](verification.json) binds revisions, built assets, capture inputs and image bytes. The Work reviewer passed against the same asset-digest algorithm used by the global evidence manifest; an intentionally different build digest was rejected. Historical capture/fixture inputs are retained beside the baseline receipt.

| Evidence | Retained images | Coverage |
| --- | ---: | --- |
| [Before](before/work-source/receipt.json) | 17 | Five original renders, ten runtime layouts and two defect reproductions |
| [After](after/work-source/receipt.json) | 22 | Five original renders, ten runtime layouts, four completed-drill frames and three fallback/receipt cases |
| [Existing Work scenarios](legacy/manifest.json) | 3 | Loaded queue on desktop/phone and phone inspector, on the same combined application build |

I directly inspected all twelve baseline runtime frames, the final twenty-two original/runtime/behavior frames, and all three existing Work scenario frames. Final copies that did not change were matched by image hash; changed copies were inspected again.

## Source comparison and visual findings

All five widths retain heading → local tabs → measures → queue heading → filters → records. My work and Approvals remain the supported local destinations. The measure strip now follows the source's four-column arrangement above 760px and two-column phone arrangement. Workflow-assignment and loaded-work qualifiers remain visible in both themes, including wrapped phone labels.

The former two-row tablet strip pushed the first record below the viewport at 1024/768px. The correction restores useful records at both widths. The table measures the first record's top edge; it does not claim matching density or pixel equality. Light and dark runtime positions are equal.

| Viewport | Original | Before, light / dark | After, light / dark | First record y: original → before → after |
| --- | --- | --- | --- | --- |
| 1440×900 | [Source](after/work-source/source-1440.png) | [Light](before/work-source/layout-light-1440.png) / [Dark](before/work-source/layout-dark-1440.png) | [Light](after/work-source/layout-light-1440.png) / [Dark](after/work-source/layout-dark-1440.png) | 539 → 680.75 → 636.20px |
| 1280×800 | [Source](after/work-source/source-1280.png) | [Light](before/work-source/layout-light-1280.png) / [Dark](before/work-source/layout-dark-1280.png) | [Light](after/work-source/layout-light-1280.png) / [Dark](after/work-source/layout-dark-1280.png) | 539 → 637.08 → 592.53px |
| 1024×768 | [Source](after/work-source/source-1024.png) | [Light](before/work-source/layout-light-1024.png) / [Dark](before/work-source/layout-dark-1024.png) | [Light](after/work-source/layout-light-1024.png) / [Dark](after/work-source/layout-dark-1024.png) | 524 → 828.83 → 660.28px |
| 768×900 | [Source](after/work-source/source-768.png) | [Light](before/work-source/layout-light-768.png) / [Dark](before/work-source/layout-dark-768.png) | [Light](after/work-source/layout-light-768.png) / [Dark](after/work-source/layout-dark-768.png) | 575 → 909.61 → 666.06px |
| 390×844 | [Source](after/work-source/source-390.png) | [Light](before/work-source/layout-light-390.png) / [Dark](before/work-source/layout-dark-390.png) | [Light](after/work-source/layout-light-390.png) / [Dark](after/work-source/layout-dark-390.png) | 760 → 743.66 → 764.92px |

At 390px the visible qualifiers add about 21px relative to the old strip; the first record identity remains readable. Desktop context panels and the phone inspection sheet remain readable. No new clipped measure label or horizontal page overflow was found. The existing shared table still becomes tall labelled cards at 1280/768px, while the original retains denser rows. That density difference remains a whole-screen integration gap.

## Observed behavior

The isolated fixture contains 61 overdue and 62 upcoming assignments, plus one record just outside the upcoming window and one undated record. Its four-item Today feed also contains separate loaded approval/evidence work. Every record is sample data. The fixture is injected by the browser harness and is not imported by the customer runtime.

- **Defect reproduced:** the baseline displays 61 overdue assignments but clicking the measure issues no source page read and displays only one loaded overdue item. Both desktop and phone reproduce it.
- **Source drills:** desktop Enter activation and phone touch activation clear conflicting title/status/window filters. Overdue loads 50 then 61 records; upcoming loads 50 then 62. Desktop keyboard focus stays on the activated metric through the first result replacement.
- **Exact boundaries:** overdue uses exclusive `2026-10-31T10:00:00.123456Z`. Upcoming uses that inclusive start and exclusive `2026-11-07T10:00:00.123457Z`, admitting the exact seventh-day endpoint but excluding the following microsecond. A page timestamp five seconds later does not remove the count-time boundary record. Continuations preserve the predicate and cursor.
- **Source separation:** Clear filters restores all four mixed Today items. The approval measure remains a one-item loaded-work drill. Assignment results show their source and read time separately from the deadline-count receipt.
- **Fallback/recovery:** unavailable or wrong-scope counts retain labelled loaded-work measures and issue no source drill. A wrong-scope page hides its rows and full totals and displays “Assignments unavailable” with retry. Fixtures recorded zero business commands and zero browser errors.

## Validation

| Check | Result |
| --- | --- |
| Final source capture | 10 layouts, 5 behavior scenarios, 22 images; passed |
| Scoped accessibility | Zero axe WCAG 2 A/AA and 2.1 AA violations across all ten Work layouts; tested primary controls at least 44px; no horizontal page overflow |
| Existing Work captures 150/151/152 | Passed, including visible source labels and the mobile inspector |
| Evidence identity review | All before/after image bytes, baseline capture inputs and final build/input bindings verified; mismatched build rejected |
| Combined typecheck and full web suite | Passed; 272 files, 2,252 tests |
| Combined fixture boundary, component contracts and manifest Node checks | Passed; independently replayed by `/root/ci_review`; manifest check also replayed here |
| Evidence and customer builds | Passed |
| Customer bundle budgets | Passed: initial JS 161,928 bytes gzip; initial CSS 32,126 bytes gzip; largest JS 573,959 bytes raw |

The capture runner includes `capture-work-source-evidence.mjs`; the global manifest review checks its receipt and the CI workflow retains `work-source/**`. From `web/`, serve `dist-evidence` and run `PAGE_URL=<preview-url> UI_EVIDENCE_REVISION=<exact-revision> node scripts/capture-work-source-evidence.mjs`. The existing cases use `UI_EVIDENCE_SCOPE=work node scripts/capture-ui-evidence.mjs` against the same preview.

## Acceptance limits

This review covers the corrected metric-to-assignment behavior and responsive strip within the current Work composition. Team/Completed, source-only request/suggestion sections, organization-area source filters, complete approval/evidence totals and broader page-density adaptation remain outside this tranche. The original contains only a light theme; dark frames establish readable runtime behavior, not equivalence to an original dark design.

Counts and pages are separate authorized reads in the application contract, so the browser does not promise a frozen population while records or permissions change. These synthetic responses prove frontend behavior only. They do not establish live backend authorization, four-actor execution, source-to-report reconciliation, whole-screen acceptance or production cutover readiness.
