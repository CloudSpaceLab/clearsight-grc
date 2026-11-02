# Corrected Enterprise capture: named scoped review

**Local corrected capture: PASS. Final-head hosted CI/UI: pending at this review.**

Reviewer: `/root/ci_review`, 2026-10-31. The original hosted failure is preserved in [diagnostic.json](diagnostic.json). This review does not change its verdict.

## Exact inputs

| Identity | Value |
| --- | --- |
| Local correction | `5381850f1f2ebff340e4d9fde57e197253da0348` |
| Remote equivalent commit object | `bd0f45be0ce96eb1fd2bcf029b0343468203eb19` |
| Exact shared tree | `9662a00fc7117326a983fd4ccce2c1a9d5a6a463` |
| Browser | Chromium `140.0.7339.16`, matching the CI pin |
| Build digest | `28e6d798572e8d37da39a977b639694fb68842515f73b4011517ff51cfe3d4b1` |
| Full receipt SHA-256 | `696fc531b1c7e89ed3099b82503eb2ce5ca2730e8136d2fd48d57572b8bb639d` |
| Fixture SHA-256 | `8b339301595a9525f3b21e22f88b0d5438e6679a3df63cd1d7d1a5158db444d8` |
| Capture SHA-256 | `1983cea03d3ef0640ee3e35a7dfe1f8c0e6f2ebe3ff1b6ee6e37277a6885b755` |
| Reviewer SHA-256 | `9d173dec1e07234d5318759b6d536747560acd13fafc92c5067f4c5614fbf2d2` |

[Full receipt and 42 images](corrected-capture/enterprise-setup/receipt.json) · [Provenance, hashes and comparison](corrected-review.json) · [Archived reviewer replay](corrected-gate-replay.json)

The remote commit object's tree was checked through the GitHub Git commit API. At review time the PR branch had not yet advanced; a later documentation supplement and fresh hosted run remain separate steps. The correction changes capture/reviewer instructions and diagnostic documentation only. The application build is identical to both earlier captures.

## What the corrected capture proves

The complete matrix has **10 layouts, 11 successful behavior scenarios and 42 verified PNGs**, including the five-width approved-source set. The retained [capture](corrected-inputs/scripts/capture-enterprise-setup-evidence.mjs), [reviewer](corrected-inputs/scripts/review-enterprise-setup-evidence.mjs) and [fixture](corrected-inputs/src/staticDemoEnterpriseSetupRuntime.js) are exact commit snapshots. Replaying the archived reviewer against these retained bytes produces no failures.

Before the same single sheet Escape, the imported editor now records `expanded: "false"`, `focused: true`, `listboxes: 0`, and `popovers: 0`. All seven editor checks complete, including explicit type selection, the retained Positions & roles/Data boundary editors and independent maker/checker presentation. No commands or browser errors are recorded. The change adds no manual focus, fixed delay, repeated Escape, retry or longer timeout.

The [three independent reviewer-negative checks](corrected-reviewer-negative-verification.json) reject a missing observation, false focus, or a remaining popover even when success flags are present. Their source receipt and all 42 original images remained unchanged.

## Image comparison and direct inspection

**40/ 42 images match the earlier retained local capture byte for byte.** The two differences are the 1440px dark overview and desktop selection/history state. **35/42 match the failed hosted capture**; six common filenames differ and the successful editor frame replaces the separately preserved failed frame. The earlier local browser was Chromium 141; the failed hosted and corrected local captures use Chromium 140. The comparison records exact hashes and does not assign the cause of image differences.

I directly opened the union of all seven changed/new frames:

| Image | Scoped observation |
| --- | --- |
| [layout-dark-1440.png](corrected-capture/enterprise-setup/layout-dark-1440.png) | Hierarchy/context text and controls remain readable; no new overlap or blank-text defect found in this frame. |
| [selection-search-history-light-1440.png](corrected-capture/enterprise-setup/selection-search-history-light-1440.png) | Hierarchy/context text and controls remain readable; no new overlap or blank-text defect found in this frame. |
| [existing-editors-and-import-guards.png](corrected-capture/enterprise-setup/existing-editors-and-import-guards.png) | Successful final Area changes sheet; imported editor dismissed and retained editors/maker guard completed. |
| [layout-light-1280.png](corrected-capture/enterprise-setup/layout-light-1280.png) | Hierarchy/context text and controls remain readable; no new overlap or blank-text defect found in this frame. |
| [layout-light-1280-context.png](corrected-capture/enterprise-setup/layout-light-1280-context.png) | Hierarchy/context text and controls remain readable; no new overlap or blank-text defect found in this frame. |
| [layout-light-390-context.png](corrected-capture/enterprise-setup/layout-light-390-context.png) | Hierarchy/context text and controls remain readable; no new overlap or blank-text defect found in this frame. |
| [layout-dark-390-context.png](corrected-capture/enterprise-setup/layout-dark-390-context.png) | Hierarchy/context text and controls remain readable; no new overlap or blank-text defect found in this frame. |

The remaining unchanged images retain their prior named review bindings. All ten measured layouts still satisfy first entity/area visibility, source order, tab ordering, no horizontal overflow, selected-context placement/focus and minimum 44px controls. At 390px, the first actual area remains y=684–733.890625px. The 500-area disclosure remains visible at y=544.109375–562.109375px.

## Diagnosis limits

[Four bounded local probes](corrected-probe-summary.json) all passed, including the original sequence; the exact hosted failure was **not locally reproduced**. The [filtered successful focus trace](corrected-focus-trace.json) records BODY at 1883 ms after Department selection with no listbox/popover, Type-trigger focus at 1888 ms, Escape at 1916 ms, and the dialog absent at 1926 ms. These are browser-relative timestamps. This observation supports waiting for the actual focus boundary and does not prove the state that existed when the failed hosted Escape was sent.

No additional visible layout defect was found within the seven-frame inspection scope. This is not whole-screen acceptance, complete WCAG conformance, live authorization or four-actor proof. The full generic hosted flows, later approved-reference capture and exact-SHA release must be verified on the newly published final head.
