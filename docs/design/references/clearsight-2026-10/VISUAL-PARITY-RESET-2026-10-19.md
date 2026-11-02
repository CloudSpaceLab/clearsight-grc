# Approved HTML visual cutover — failed parity audit

Date: 2026-10-19 · Issue: #462 · **Accepted visual migrations: 0 / 19**.

## Evidence

Compared the hash-verified original reference screenshots with the matching runtime captures from the **successful** UI/UX review of PR #531. Evidence artifact: [runtime + approved HTML screenshots, run 39211888415](https://github.com/CloudSpaceLab/clearsight-grc/actions/runs/39211888415/artifacts/12168078733). Source files and SHA-256 inventory: [MANIFEST.json](source/MANIFEST.json).

| Screen / matched 1440 × 900 captures | Observed structural mismatch | Cutover |
| --- | --- | --- |
| Program: `approved-reference/approved-program-workspace-1440x900.png` vs `04-program-light-1440x900.png` | Different first-viewport density, header/action composition, metric-card anatomy, missing source's controlled-area distribution and stacked requirement graphic; current response shows a data-unavailable message instead | **Not accepted** |
| Assessment: `approved-reference/approved-assessment-workspace-1440x900.png` vs `211-internal-assessment-light-1440x900.png` | Approved design is a **full-page assessment workspace** with open header, progress summary and question/evidence working area; runtime is a **modal request capture sheet**. Three inner columns do not make these screens equivalent | **Not accepted** |
| Framework Library: `approved-reference/approved-framework-library-1440x900.png` vs `200-framework-library-light-1440x900.png` | Reference is a dedicated library, with compact source-aligned header, Explore / In my Programs / Edition updates, broad search+filters and list/detail composition; runtime is nested within the Portfolio switcher with different heading, controls and list/inspector proportions | **Not accepted** |
| Program mobile: `approved-reference/approved-program-workspace-390x844.png` vs `04-program-light-mobile-390x844.png` | Approved first viewport shows compact record heading, action, short tabs, 2×2 cards and early coverage; runtime spends substantially more space on the global context card, a large record heading, stacked metadata and navigation before the metrics | **Not accepted** |

These are real, side-by-side browser captures. Different fixture records and unknown/missing backend populations are **not** reasons to copy prototype data; compare hierarchy, relative placement, density, tokens and interactions independently of example totals.

Other 15 approved source areas were **not** awarded parity. A prior migration/use of shared components, a green flow test or a screenshot saved in CI is not a visual sign-off. **0/19 means no screen has an accepted fidelity receipt, not that every unreviewed screen was individually pixel-diffed.**

## Root causes

1. **Wrong unit of progress:** PRs checked individual controls and operational behavior rather than reproducing whole approved screens. A six-tab page and four live metrics do not establish the approved page composition.
2. **Reference-to-runtime geometry never gated:** `capture-approved-source-references.mjs` stores `visual_parity: "NOT_REVIEWED"`; `review-ui-flow-manifest.mjs` validates screenshot existence and state coverage but does not compare the HTML against React. The green UI/UX status was reported as visual parity.
3. **Shared shell diverges:** the approved light experience places compact real scope filters, navigation, search, identity and actions in a narrow context bar; current runtime emphasizes a large account/role block and different navigation grouping. The unified specification governs **which routes may be present**; it does not excuse differences in proportions, typography or spacing.
4. **Wrong Assessment surface:** the approved internal author/reviewer page was replaced by a modal capture. This must become an authorized full-page work surface reusing `CapturePanel`, not a new form/approval engine.
5. **Prototype data mismatch confused with visual mismatch:** reference counts/statuses are illustrative and MUST NOT ship. Real data, unknown/partial/denied states and proper denominators belong in the same approved layout.
6. **Insufficient review authority:** CI functional receipts were accepted as if they were design approvals. They are independent gates.

## New acceptance contract

A screen is **Accepted** only when all are supplied, reviewed and recorded:

- Exact manifest-pinned HTML + design notes and one named React owner. A screenshot of current UI alone is insufficient.
- Paired original-HTML/React screenshots at 1440, 1280, 1024, 768 and 390 CSS px; light theme reference vs light runtime, plus equivalent dark runtime; no hidden selected panels or unrelated routes. Capture both first viewport and important opened states.
- Structural checklist for shell, heading, actions, page-level navigation, cards, charts, list/detail sections, control sizes, typography, colors, spacing, responsive behavior and empty/partial/denied/error states. Record every mismatch with a before/after image.
- Every action is real and authorized or explicitly unavailable; every value is source-backed and exact scope/time/denominator. No prototype data/JS/CSS imported into `web/src`.
- Existing unit/backend/role/scoping/keyboard/reduced-motion checks remain mandatory. A design reviewer must explicitly approve the comparison. **Automated CI cannot self-approve visual parity.**

## Source-first execution order

| Priority | Source screen | Required structural change | Data/functional dependency |
| --- | --- | --- | --- |
| P0 | Unified shell (all screens) | Restore source-compatible visual anatomy, compact top controls, logo/navigation proportions and responsive density by adapting existing shared shell/tokens | Only permitted scope controls; search remains unavailable until authorized backend support |
| P1 | Program | Source heading/CTA/short tabs, four compact cards, analytical coverage/area layout and actionable attention rail; use honest incomplete states | Program projection, #433 reviewed denominators |
| P1 | Assessment | **Replace authorized internal modal with full page**; open header, progress, sections/questions/evidence, review and source context; keep external capture focused | Existing CaptureForm/drafts/immutable responses and reviewer authority |
| P1 | Framework Library / Adoption | Dedicated library list/detail and full four-stage adoption experience with bounded edition/links, existing Form and independent decision | #437 publisher/search/adoption and #433 scoring contracts |
| P2 | Vendor, Operational Risk, Privacy, Reports, Work | Implement original first-viewport, native panels, relevant charts and exact drills—not only card-count patches | Native domain services + report/source trace |
| P3 | Controls, Setup, Integrations, Search, Notifications, Audit, People, Automation, Copilot | Match each remaining original HTML with approved function and scoped data | Exact existing service or tracked backend dependency |

**Freeze the visual-cutover claim for PR #533 and #534:** their changes may be operationally useful, but are not completed source-screen conversions. Keep them separate from this audit and require visual review before counting toward #462.

## Release state

- Functional/component integrations: previously merged, retain their regression protection.
- Accepted approved-HTML visual migrations: **0 / 19**.
- End-to-end independently authenticated actor/report proof: **not accepted**.
- Full design cutover / production release: **blocked**.

Do not close #462 or describe a screen as cut over until its exact source-to-runtime comparison is approved.
