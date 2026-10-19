# Approved HTML — visual acceptance review (19 October 2026)

Tracking: #462. **Decision: 0/19 original screens accepted; production UI cutover BLOCKED.**

## Evidence and review method

- Independently SHA-256 checked original `complete designs.zip`: `65dcc0e3f26e9d41e38efd1ae1d571f5db0f0582a2d561f8c942b25817a5f6e5`. This exactly matches `source/MANIFEST.json` and includes 18 nested archives, 96 source text files and 143 original visual assets.
- Compared screenshots rendered from original, manifest-pinned HTML against **actual** React screenshots retrieved from [baseline run 39211888415](https://github.com/CloudSpaceLab/clearsight-grc/actions/runs/39211888415/artifacts/12168078733), [Assessment #541 run](https://github.com/CloudSpaceLab/clearsight-grc/actions/runs/39285723727/artifacts/12204806157), [Shell #537 run](https://github.com/CloudSpaceLab/clearsight-grc/actions/runs/39284914440/artifacts/12203430977), [Vendor #544 run](https://github.com/CloudSpaceLab/clearsight-grc/actions/runs/39285205024/artifacts/12202469928) and [Framework #545 run](https://github.com/CloudSpaceLab/clearsight-grc/actions/runs/39287097435/artifacts/12202854559).
- The reference renderer generates **90 original-HTML screenshot captures** (18 screens × 1440/1280/1024/768/390). Only **9** have declared same-screen runtime mappings in `approved-reference/manifest.json`. Also manually inspected Work, Vendor and Reports desktop/mobile runtime screens against the original HTML. Those additional comparisons do not prove every interaction/state.
- Match information hierarchy, composition, navigation, typography, spacing, content density and selected/empty/error states. Do **not** score different fixture data as a visual failure or copy prototype values into production. The reference HTML is light; dark runtime comparisons establish layout only, not color equivalence.
- Classification: **FAIL** = inspected comparable screen with material difference; **UNPAIRED** = no equivalent runtime screen/state capture (cannot accept); **SOURCE BLOCKED** = source unavailable. Distinguish these from CI success.

## Screen acceptance register

| Approved source | Decision | Blocking visual/operational findings |
| --- | --- | --- |
| Control Tower | SOURCE BLOCKED | No standalone Control Tower HTML identified in manifest. Cannot infer acceptance from another dashboard. |
| Framework Library | FAIL | Approved `framework-library.html` has Explore/In my Programs/Edition updates, scoped search/filter row, denser edition list and right inspector. Runtime retains Portfolio lens and a different hierarchy. PR #545 is an accepted **inspector-preview slice only**. |
| Framework Adoption | UNPAIRED | No unified runtime comparison for Program & Scope → Requirement Links → Questionnaire → Adoption Review, including decisions and existing Form. |
| Program | FAIL | `program-workspace-preview.html` differs in heading/CTA density, responsive tabs, assessment cards, requirement coverage/control-area visualization and attention panels. Runtime 390px places key metrics materially lower. Missing denominator must remain unknown. |
| Assessment | FAIL | Full-page conversion in #541 is genuine but still differs from `assessment-workspace-preview.html` in header/actions, progress, section/question/evidence panels and mobile viewport. 1440/1280/1024/768/390 pairs checked. Old phone bottom nav still present on its branch; combine with #537 before testing release candidate. |
| Vendor | FAIL | `vendor-workspace-preview.html` Risk distribution, Review pipeline, Needs attention, lifecycle tab hierarchy and register are not reproduced. **Accept narrow #544 card anatomy**, not screen parity. |
| Operational Risk | UNPAIRED | Native indicator-detail screenshots do not represent the source Operational Risk overview/KRI/RCSA/Losses/Actions layout. |
| Privacy & Data | UNPAIRED | ROPA register screenshots do not represent source Privacy overview/DPIA/Breaches/Vendor reviews/Requests. |
| Reports & Insights | FAIL | Source Overview/Explore/Reports/Schedules and risk concentration, assurance, losses, insights panels absent from the rendered Reports library composition. |
| Work & Approvals | FAIL | Source tab hierarchy, selected-item inspector, list layout, first-viewport density and compact cards differ from actor-scoped Work queue; do not fake Team/Completed counts. |
| Controls & Policies | UNPAIRED | Missing matched policy/control/certification/review runtime states. |
| Enterprise Setup | UNPAIRED | Missing matched groups/jurisdictions/frameworks/imports runtime states. |
| Integrations | UNPAIRED | Missing matched connections/mappings/history runtime states. |
| Search Command Center | UNPAIRED | Requires authorized federated search contract; no matched runtime surface. |
| Notifications & Activity | UNPAIRED | Missing matched inbox/receipt/history states. |
| Audit & Assurance | UNPAIRED | Missing matched audit plan/fieldwork/findings states. |
| People & Access | UNPAIRED | Missing matched roles/delegations/routing/denial states. |
| Automation Governance | UNPAIRED | Missing matched proposal/simulation/activation/rollback states. |
| Contextual Copilot | UNPAIRED | Missing source-grounded, authorized, limited-suggestion screenshot/interaction receipts. |

Summary: **6 visually rejected; 12 unpaired; 1 missing source; 0 accepted**. All 19 remain gated.

## Global shell and selected tranches

- **#537 (shared shell)**: original vs actual 390px screen confirms the new 56px left navigation rail and removal of duplicate bottom navigation; however the top context bar remains cramped and materially different, and Program/Framework content still diverges. **Accept the mobile rail geometry only, not full shell/source-screen parity.** [CI run 39284914440](https://github.com/CloudSpaceLab/clearsight-grc/actions/runs/39284914440) collected all 228 screenshots and passed flows but failed initial CSS gzip budget: **32,774 bytes vs 32,768 allowed**. The subsequent exact-head run `39289163580` also failed a **390px Work metrics 2×2 layout assertion** (rendered as one column after the 56px rail narrowed available content), plus initial CSS gzip **32,798 bytes**. A correction now reduces redundant shell styles and 390px content padding; it requires new exact-head tests and inspection before merge.
- **#541 (Assessment)**: exact-head backend/web and UI/UX passed, but **reject full-screen visual parity**. Reconcile the approved progress/header/section/question/evidence structure; rerun combined #537/#541 renders at all target widths.
- **#544 (Vendor)**: **accepted isolated four-card label-above-value presentation**, 1440px desktop + 390px 2×2 mobile, no change to source semantics or permissions. Exact-head checks green; merged as component tranche, not Vendor cutover.
- **#545 (Framework inspector)**: **accepted isolated first-loaded-edition preview/explicit selection behavior** with exact-head tests. Original overall Framework Library hierarchy still fails. Merged as component tranche, not Framework cutover.

## Acceptance procedure going forward

1. Start with the immutable original HTML and 143 archived reference images; identify a **same-screen/same-state** runtime journey. Do not compare a detail page with an overview.
2. Produce side-by-side captures at **1440, 1280, 1024, 768 and 390**, equivalent light and dark where source allows; inspect first viewport and selected/expanded action states. Name exact PR head and CI artifact.
3. Check hierarchy, cards, list/inspector layout, selected states, responsive rail/tab geometry, 44px targets, keyboard, reduced motion, overflow and **bundle budgets**. Record concrete differences.
4. Verify tenant/LE/group and actor authority, missing/unknown/stale data, exact API denominator and report-source integrity separately from visual judgment.
5. Promote a **whole screen** to ACCEPTED only once the above passes. Component-level improvements can merge but must not increase the 0/19 counter until the screen itself passes.

Original archive/screenshots and reviewer comparisons were independently inspected in this review; no prototype runtime values were substituted.