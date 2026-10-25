# Risk & Losses — retained v2 source and runtime review

PR: [#586](https://github.com/CloudSpaceLab/clearsight-grc/pull/586). Decision: [page and navigation adaptation](../../design/decisions/2026-10-22-risk-loss-pages-v2.md).

The [compact evidence receipt](../../design/decisions/2026-10-22-risk-loss-evidence.json) binds the core manifests, selected-record supplement and named review by SHA-256.

This is **partial v2 layout integration** for Risks, RCSA, Key indicators and Loss events. These isolated synthetic fixtures verify page anatomy and interactions; they do not establish live authorization, operational acceptance or production cutover. The formal whole-screen acceptance register remains **0/19**. The [named visual review](visual-review.md) describes the inspected original/runtime frames and the intentional data, navigation and responsive adaptations. It does not claim pixel equivalence.

## Exact revisions and immutable source

- Before: `ba284eb24b01141666c93783c15221263a13d7b8`, tree `7d1c2880c7082861751999290430408caea94377`. [Baseline receipt](before/baseline-receipt.json) retains all 171 before-build hashes; [baseline notes](before/README.md) distinguish the 40 matrix frames from one overflow diagnostic.
- After implementation/evidence code: `527b348631e2f92fc920125b7da28692a2743e92`, tree `c6e85fc18a9e1e767fb45dfa80f7a17647c6fb4f`. [Runtime manifest](after/manifest.json) binds each frame to all 171 frozen build assets; aggregate SHA-256 `a20ab1aadace0972b9f35bb924c1d4361f805adca7af4a0b7131c84e1badc746`. Subsequent retention/documentation commits do not change runtime code.
- Original: `docs/design/references/clearsight-2026-10/source/clearsight-operational-risk-designs/operational-risk-preview.html`, SHA-256 `cf0a101d574904fc5f496cab0625889e640e2e30734d180d86b89171419e69dd`. Original HTML, CSS, JS and manifest bytes remain unchanged.

## Verified matrix

The [dedicated review receipt](after/risk-loss-review.json) passes all 68 runtime states, 20 original collection frames, 40 exact source/runtime pairs and 16 scoped accessibility states. Its 91 input artifacts are hash-bound. Each page has 10 width/theme frames and seven representative detail, history, query/paging, empty, denied or partial states. Viewports are 1440×900, 1280×800, 1024×768, 768×900 and 390×844; both runtime themes are covered.

The [source manifest](after/approved-reference/manifest.json) binds 30 same-screen collection pairs for RCSA, indicators and losses, and 10 explicit Risk table-anatomy pairs. There is no dedicated original Risk register or Risk record. Runtime source-reference pairs compare supported page anatomy; prototype example records, percentages, deadlines and unsupported actions are never treated as runtime facts.

The separate [selected-record supplement](detail-reference/README.md) retains six original RCSA, KRI and Loss inspectors at 1440×900 and 390×844, with six exact runtime-detail pair bindings. Its [manifest](detail-reference/manifest.json) preserves the selected original identities, image hashes, source geometry and explicit workflow/route adaptations. The original is light-only; the paired phone runtime is dark. Across the collection and selected-record sets the named review directly inspected **26 original frames and all 68 runtime frames**. There is no same-record or pixel-equivalence claim.

Actual Back/Forward returns, route ownership, persistent local tabs, settled readers, rendered server-page identities, supported filter reset/paging, zero business commands during reading, 480px/full-phone inspectors and no horizontal overflow are checked. Badge checks reject individual words split across lines or text escaping its badge/cell; three intact words in a three-line label remain permitted. RCSA card, inspector summary and rendered rows reconcile to three risks and two controls. Separate loss currency amounts and native indicator units remain explicit.

## First useful phone content

Top of the first record identity at 390×844 in the light theme, measured from the retained DOM geometry:

| Page | Before | After |
| --- | ---: | ---: |
| Risks | 1286.6px | 595.1px |
| RCSA | 749.5px | 524.6px |
| Key indicators | 632.9px | 522.1px |
| Loss events | 1830.7px | 655.1px |

All after identities fit within the first viewport. Baseline RCSA overflow at 768px and indicator overflow at 1280/1024/768px are absent from the after matrix. This does not imply that every fact in a full record fits without vertical scrolling.

## Accessibility and release checks

[Accessibility](after/accessibility.json): 16/16 PASS, zero axe violations, browser errors or undersized primary controls, with 44px phone targets. Automated contrast proof remains incomplete for 24 decorative-glyph observations, eight partly offscreen inactive phone-tab observations, and 58 supplemental shared-shell filter/backdrop readings. Supplemental readings contain 736 PASS, 58 INCOMPLETE and 64 REVIEW, with no FAIL or UNSUPPORTED. The scoped page canvas removes the prior gradient uncertainty below metadata; no blanket contrast claim is made for the entire application.

The corrected runtime passed hosted [CI attempt 1](https://github.com/CloudSpaceLab/clearsight-grc/actions/runs/39975370219): 263 web test files / 2,159 tests, typecheck, runtime separation, customer build and backend checks. Local full-suite execution had timing failures; one isolated Reports timeout reproduced on untouched `ba284eb`, and the isolated Forms assertion passed. Test timeouts were not increased. The PR is the authoritative record for final documentation-head CI/UI artifacts, expected-head merge and exact-SHA demo verification.

Earlier diagnostic captures at `00d93a8` and the partial `1573772` run were not rebound or promoted. The review found and corrected indicator word wrapping, stale-risk phone badge wrapping and the inconsistent RCSA sample count before this final capture.

## Released revision and hosted follow-up — 23 October

[#586](https://github.com/CloudSpaceLab/clearsight-grc/pull/586) merged as `02dd0b4f066ba58ec17cf0e2588437efe5e856fd`, tree `797536c51feb5cffdac2063cd72e4076710367c7`, identical to the tested final PR tree. Exact-main [CI](https://github.com/CloudSpaceLab/clearsight-grc/actions/runs/39986226879), [UI/UX](https://github.com/CloudSpaceLab/clearsight-grc/actions/runs/39986227010) and [non-production demo deployment](https://github.com/CloudSpaceLab/clearsight-grc/actions/runs/39987297851) all passed on attempt 1. CI passed 263 web test files / 2,159 tests and the backend gates. UI/UX passed 515 flow records, 77 Forms capabilities, 523 runtime screenshots, 180 original-source captures and 108 accessibility states, including the Risk & Losses source/binding gates. The deployed worker and public health revision were verified at that exact merge SHA.

The read-only hosted browser check reached all four collections at desktop light and phone dark. Desktop Risk/Loss record opening and return passed; RCSA was empty. SystemAdmin indicator reads returned the expected `permission_required` boundary, while a separate CRO read succeeded with an empty population. No business commands ran. These results do not establish populated RCSA/indicator workflows or full actor acceptance.

The hosted phone check **failed** when browser Back selected Risks while its active label remained clipped. The [shared-tabs follow-up](../2026-10-23-tabs-history/README.md) retains a reproduced baseline, the corrected local runtime, negative receipt checks and 12 passing local Back/Forward transitions. That follow-up remains unpublished and undeployed, so the hosted defect is still open. The temporary browser certificate exception and other hosted evidence limits are recorded in its [decision](../../design/decisions/2026-10-23-tabs-history-visibility.md). Raw hosted record identities and screenshots are not included in public evidence.
