# Risk & Losses — v2 page and menu adaptation

## Decision and scope

Continue the approved whole-page adaptation after the Vendor collection history correction in #585. The before state is main `ba284eb24b01141666c93783c15221263a13d7b8`, tree `7d1c2880c7082861751999290430408caea94377`.

Risk owners, reviewers and oversight users need to scan current records, identify the next review, and open the exact governed context. Adapt **Risks / RCSA / Key indicators / Loss events** together, with one persistent local menu and the source's record-first page anatomy. Existing exact records, version checks, permissions, scope and command owners remain authoritative. This is partial v2 layout integration under #462; the formal operational-acceptance register remains **0/19**.

## Immutable source and navigation authority

The source is `docs/design/references/clearsight-2026-10/source/clearsight-operational-risk-designs/`. Keep all original bytes unchanged.

| Source | SHA-256 |
| --- | --- |
| `operational-risk-preview.html` | `cf0a101d574904fc5f496cab0625889e640e2e30734d180d86b89171419e69dd` |
| `operational-risk.html` | `befc18a83765e728979846ca655268485b8ebfd6d02cff7f26c018f6a0cd2323` |
| `operational-risk.css` | `c9320886d048c3664c64ccc2898e2d9bd73e7b746daba7865e05894b809b0a10` |
| `operational-risk.js` | `6a44de4e7cf0f850dec89555da2bc91934c211646df0b198b9a45e39685e7ef5` |

The [21 October navigation decision](2026-10-21-v2-navigation-adaptation.md) supersedes the prototype's Overview / KRIs / RCSA / Losses / Actions destinations. Reuse the existing `#risks`, `#rcsa`, `#insights` and `#losses` routes in the approved order. The global contextual menu and local menu share that mapping. Governed aggregate analysis stays in Reports through its existing period-scoped deep link; canonical actions stay in Work. Do not invent a combined overview, aggregate, global date filter or create operation.

## Page anatomy and ownership

| Region | Adaptation and existing data owner |
| --- | --- |
| Common heading and local menu | One Risk & Losses h1, actual organization/entity/supported scope context, View reports, and shared underline Tabs. Keep the menu mounted across peer-route navigation. |
| Risks | Compact search/status row, additional Filters disclosure, bounded risk register, then secondary page-local appetite detail. The source has no dedicated Risks register: this is a **table-anatomy adaptation**, not a same-screen parity claim. |
| RCSA | Source-shaped cycle-card grid, actual stage, owner, frozen risk/control counts and updated date. Put the loaded-cycle stage distribution in a secondary disclosure below the cards. Existing cycles and exact review sheet own all workflow transitions. |
| Key indicators | Dense native-value register and a real Indicator type field for the existing KRI/KCI query. Preserve value, unit, limits, condition, freshness, movement, ownership and exact observation history. |
| Loss events | Compact search/status row, additional Filters disclosure and ledger with separate Gross / Recovered / Net / Currency facts. Scope names and recovery/state context come from existing records. Page-local currency totals follow the ledger. |
| Exact records | Risk and Loss detail replace collection chrome and retain their explicit return action. RCSA and indicator details use their existing focused sheets over the selected page. |

`RisksWorkspace`, `RCSAWorkspace`, `InsightsWorkspace` and `LossesWorkspace` remain their existing read and command owners. The common frame only composes presentation and calls the existing App navigation function. The governed `InsightsWorkspace` risk/loss analysis branch remains in Reports and does not receive the collection frame.

## Data and recovery limits

Risks and Losses use bounded server pages. A page count or current-page summary is not a tenant total. Risk ratings and accessible labels must use the current risk version; an old assessment must remain visibly and audibly stale. Appetite requires the current statement and assessment. Loss currencies remain separate, void/invalid entries remain excluded from summaries, and unavailable money never becomes zero.

RCSA has no supported organization-scope list filter and no reviewed/expected assessment denominator or review due date. Show its real frozen counts and stage; do not copy the prototype's progress percentage or deadlines. Indicator list queries support kind, organization scope, exact identity and cursor paging; do not add a misleading current-page search or condition filter merely to reproduce prototype controls.

Scope changes must never send a previous scope's cursor. Exact-record changes must not expose a prior record's detail, draft or enabled command. Pending readers remain abortable and identity-qualified. Preserve error/retry, denied, empty, partial and unknown states, version conflicts and existing evidence/command boundaries. The new navigation itself performs no business mutation.

## Responsive and interaction contract

Compare the original and runtime at **1440 / 1280 / 1024 / 768 / 390** CSS pixels, in both runtime themes. The common heading follows the source's 111px desktop region and 35px heading, reducing to 31px below 1190. The 49px local menu stays a single horizontally scrolling underline row on phones; shared controls retain at least 44px targets.

RCSA uses three columns above 1190, two through 601–1190 and one at 600 or below. Source card spacing, compact subsection headings and first useful records take precedence over the previous generic summary-card composition. Dense registers replace columns with labelled two-column fact records when their available content width requires it. At 768px this can occur before the source's 600px viewport breakpoint because the shared shell leaves less usable table width; record the container-width adaptation explicitly. Filters disclose additional controls without placing a tall summary ahead of the register.

RCSA and indicator selections use the source's 480px desktop inspector with full-width phone replacement. Their existing focused sheet keeps focus, Escape and return behavior; its contents adapt to the sheet's own width. Risk and Loss exact records remain standalone because they contain governed edit, control-link and recovery workflows; their complete record hierarchy and return path are reviewed as an explicit route adaptation. Existing shared tokens and control variants remain in use.

The visual review found short indicator conditions breaking inside words at intermediate desktop widths and a stale-risk label breaking inside “Reassessment” on phones. Indicator badges reserve their intrinsic word width. Risk fact badges use the existing smaller gap and padding tokens, with wrapping allowed only between words. The browser receipt measures word lines and text/badge/cell containment, rather than relying on page overflow alone.

The review also exposed an inconsistent synthetic RCSA population: the list fixture claimed four controls while its complete detail contained two. The fixture now reports its two recorded controls; capture checks reconcile the selected card with the inspector summary and actual risk/control rows. This corrects isolated evidence data, not a production count or a backend contract. The immutable before frames preserve the earlier inconsistency.

The four collection pages use the existing semantic canvas surface across the main region. This removes the body gradient beneath small metadata without changing the shared palette or the source heading wave. The earlier gradient captures passed sampled contrast checks but could not establish a uniform bound across every gradient position; the final accessibility capture checks the opaque canvas treatment.

## Verification and status

**Implemented in [#586](https://github.com/CloudSpaceLab/clearsight-grc/pull/586), with retained source/runtime evidence and a scoped named visual review.** The [evidence index](../../evidence/2026-10-22-risk-loss-v2/README.md) preserves 40 baseline page frames and one separate overflow diagnostic from the exact before build. The final runtime matrix is bound to `527b348631e2f92fc920125b7da28692a2743e92`, tree `c6e85fc18a9e1e767fb45dfa80f7a17647c6fb4f`, and all 171 frozen build files with aggregate SHA-256 `a20ab1aadace0972b9f35bb924c1d4361f805adca7af4a0b7131c84e1badc746`. Later retention/documentation commits do not change the runtime.

The dedicated review passes 68 runtime states, 20 original collection frames, 40 source/runtime pairs and 16 scoped accessibility states, with zero browser errors or business commands during reading. It covers all five widths and both runtime themes, representative exact details, actual browser Back/Forward returns, rendered supported queries and server-page identities, empty/denied/partial states, keyboard selection, touch targets and first-record geometry. The source pairs include 30 same-screen collection comparisons for RCSA, indicators and the loss ledger and ten explicitly labelled Risk anatomy adaptations.

The [selected-record supplement](../../evidence/2026-10-22-risk-loss-v2/detail-reference/README.md) adds six original RCSA, KRI and Loss inspector frames at 1440/390px, paired to the final runtime detail hashes. Original RCSA and KRI inspectors are 480px wide on desktop and full-width on phones; their contents and action placement adapt to the existing frozen records, stage transitions, native indicator facts and Work routes. Loss remains an explicit inspector-to-standalone-route adaptation. Source and runtime example records differ, and the original has no dark theme or dedicated Risk record; these are not same-record or pixel-equivalence claims.

The [named review](../../evidence/2026-10-22-risk-loss-v2/visual-review.md) records direct inspection of all 68 runtime frames and 26 original frames, closes both status-word wrapping findings and the RCSA fixture inconsistency, and documents the deliberate differences in row height, phone controls and supported data. At 390px the first Risk identity moves from 1286.6px to 595.1px and the first Loss identity from 1830.7px to 655.1px. All four first identities fit the 844px viewport; the old RCSA/indicator horizontal overflows are absent. This does not assert that all record facts or actions fit above the fold.

All 16 scoped accessibility states pass with zero axe violations and 44px primary controls. Automated contrast proof remains incomplete for decorative glyphs, partly offscreen inactive phone tabs and supplemental shared-shell filter/backdrop readings. The evidence index records their counts and preserves this limit instead of claiming complete application-wide accessibility.

Keep the existing native-indicator, RCSA and loss workflow browser gates, adapting only obsolete presentation selectors. App regressions verify menu order, route ownership, one page heading, persistent tablist identity and collection/exact-record transitions. Focused regressions cover scope cursors, stale identity and rating semantics. TypeScript, customer and evidence builds, runtime-fixture separation, UI contracts, full CI and exact-head UI artifacts remain release gates.

Hosted [CI attempt 1 at the corrected runtime](https://github.com/CloudSpaceLab/clearsight-grc/actions/runs/39975370219) passes all 263 web test files / 2,159 tests, typecheck, runtime-fixture separation, customer build and backend checks. Local full-suite runs had timing failures; one isolated Reports timeout reproduced on untouched main, while the isolated Forms assertion passed. Those local results remain recorded as failures, and no timeout was increased. The PR is the authoritative record for final documentation-head CI/UI artifacts, expected-head merge and exact-SHA demo deployment results. This layout and isolated-interaction proof cannot establish live authorized results, four-actor acceptance or production cutover; formal whole-screen acceptance remains **0/19**.
