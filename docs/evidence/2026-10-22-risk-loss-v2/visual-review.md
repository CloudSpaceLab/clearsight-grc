# Risk & Losses v2 — named visual review

**Reviewer:** ChatGPT, delegated reviewer `/root/risk_parity_audit`  
**Review date:** 23 October 2026 (UTC)  
**Decision:** **PASS for the scoped anatomy, responsive composition and legibility review, with the adaptations and limitations below.** The three findings raised during review are closed in the final evidence.  
**Acceptance boundary:** This is a named review of source-derived UI structure and synthetic evidence. It does not establish pixel equivalence, live authorization, operational workflow acceptance or production cutover. Formal acceptance remains **0/19**.

## 1. Exact reviewed revision and evidence

| Identity | Value |
| --- | --- |
| Published runtime/evidence revision | `527b348631e2f92fc920125b7da28692a2743e92` |
| Published tree, from the publisher's release record | `c6e85fc18a9e1e767fb45dfa80f7a17647c6fb4f` |
| Runtime build digest, verified in the retained runtime manifest | `a20ab1aadace0972b9f35bb924c1d4361f805adca7af4a0b7131c84e1badc746` |
| Final runtime and collection-source evidence | `docs/evidence/2026-10-22-risk-loss-v2/after/` |
| Original selected-record supplement | `docs/evidence/2026-10-22-risk-loss-v2/detail-reference/` |
| Before evidence | `docs/evidence/2026-10-22-risk-loss-v2/before/`, captured from `ba284eb24b01141666c93783c15221263a13d7b8` |

All paths in this review are relative to the repository root. The final evidence identifies the reviewed build independently of the later commit that retains this report and the PNGs.

| Retained receipt | SHA-256 |
| --- | --- |
| `after/manifest.json` | `0710d67354f6dcdaa7140d84e2f077cc6893504be0ead99ec175aceba879b214` |
| `after/approved-reference/manifest.json` | `9b9df096694b1a6d9dd1c1397d6d812fee233188301b85a7f88be5dff5325c87` |
| `after/risk-loss-review.json` | `6d69a2dd3156d4cbeeb5bf5b3878be97f00bb59db94d5d0b69eb403f28bdc192` |
| `after/accessibility.json` | `658762ceec7d9f7295abf78921855e87757e1f2f34ed6885e202684c86895945` |
| `detail-reference/manifest.json` | `a6b13da8f5aceb137d0ba48c8be7449916c2b286c34ac9c482cb0dc3bfa06b45` |

The receipt paths above are under `docs/evidence/2026-10-22-risk-loss-v2/`. PNG identity and byte count were independently checked against each manifest: **68 final runtime images, 20 original collection images and six original selected-record images, with no mismatch**. The 40 collection source/runtime bindings and six selected-record bindings also matched the actual final runtime PNGs.

## 2. Review method and coverage

The reviewer opened every one of the **94 named PNGs** individually in an image viewer. Runtime coverage consists of four destinations at **1440×900, 1280×800, 1024×768, 768×900 and 390×844**, in both light and dark themes (40 frames); exact details at 1440px light and 390px dark (eight frames); and history return, empty, denied, partial and filter/paging states for each destination (20 frames). The original collection coverage is four comparison subjects at the same five widths (20 light-source frames). The selected-record supplement adds original RCSA, KRI and Loss details at 1440×900 and 390×844 (six light-source frames).

The source HTML, CSS, JavaScript and design notes were read directly. The collection and selected-record sources were compared with their corresponding runtime layouts, with special attention to the complete captured viewport, heading/menu/collection order, native facts, condition text, action placement, phone first-record identity and inspector dimensions. Static frames establish what is visible in those viewports; they are not full-length visual inspection of all content below the fold. Separately recorded geometry and interaction receipts support overflow, exact-record navigation, focus and scroll/action reachability. The reviewer read those receipts but did not personally execute the browser interactions.

The collection manifest labels five source frames `TABLE_ANATOMY_ADAPTATION` and 15 `SAME_SCREEN`. Here `SAME_SCREEN` identifies the corresponding collection subject; it does not mean the original and runtime use the same records, data, controls, pixel positions or typography. The named decision in this document is separate from the automated reports' `visualParity: NOT_ASSESSED` and `liveAuthority: NOT_ASSESSED` fields; those fields remain unchanged.

## 3. Original design authority and permitted adaptations

The implementation decision is `docs/design/decisions/2026-10-22-risk-loss-pages-v2.md`. Shared menu ownership is established by `docs/design/decisions/2026-10-21-v2-navigation-adaptation.md`. The actual artifacts are under `docs/design/references/clearsight-2026-10/source/clearsight-operational-risk-designs/`.

| Original artifact | Independently verified SHA-256 |
| --- | --- |
| `operational-risk-preview.html` | `cf0a101d574904fc5f496cab0625889e640e2e30734d180d86b89171419e69dd` |
| `operational-risk.html` | `befc18a83765e728979846ca655268485b8ebfd6d02cff7f26c018f6a0cd2323` |
| `operational-risk.css` | `c9320886d048c3664c64ccc2898e2d9bd73e7b746daba7865e05894b809b0a10` |
| `operational-risk.js` | `6a44de4e7cf0f850dec89555da2bc91934c211646df0b198b9a45e39685e7ef5` |

The collection reference manifest records source archive SHA-256 `65dcc0e3f26e9d41e38efd1ae1d571f5db0f0582a2d561f8c942b25817a5f6e5`. The selected-record supplement records an unchanged original HTML file, offline rendering and no successful network requests. Original content is illustrative design material, not tenant data.

The source supplies a 111px desktop heading region, 49px underline menu, native-value indicator table, three/two/one-column RCSA cards, separate loss amounts and a 480px/full-phone inspector. The implementation retains those relationships while using supported runtime records, shared controls and existing route/command owners. **Risks has no dedicated original register or record detail. Its comparison is an explicitly approved table-anatomy adaptation; no same-screen Risks parity is claimed.**

## 4. Findings and verified closure

The first complete 68-frame review used revision `00d93a80b089bd3ad7e23a5ea4f53a35e34779be`, build digest `238dabdb3b150d1423495ef9e75548f5d349d98efaae93412bc7c50876040daa`. It identified the following issues before final promotion. That earlier evidence is diagnostic and is superseded by the exact final set above.

| ID | Original finding | Correction and final disposition |
| --- | --- | --- |
| V1 | At 1280/1024, indicator condition badges split short words: `Breach` became `Breac / h` or `Brea / ch`; `Normal` became `Norm / al`. | The scoped indicator status-cell rule in `web/src/components/insights/insights.css` preserves intrinsic width and unbroken condition words. **Closed:** all ten final indicator matrix frames were directly inspected. Condition labels and review actions are legible in both themes, including 1024 and responsive cards. |
| V2 | The stale Risk phone state split `Reassessment` before its final letter in the two-column fact layout. | Scoped existing gap/padding tokens in `web/src/components/risks/risk.css` give the badge enough room without reducing its type. **Closed:** `2590-risk-loss-risks-partial-dark-390.png` now displays `Reassessment / needed` as complete, contained words. Standard phone badges remain contained. |
| E1 | The synthetic RCSA card listed four controls while its complete selected-cycle detail contained two. | `web/src/rcsaEvidenceData.ts` now supplies two controls in the list fixture. **Closed:** the final collection cards and both selected-cycle screenshots reconcile **3 risks / 2 controls**. The detail receipt also counts three risk rows and two control rows. This was a fixture inconsistency, not evidence of an operational backend defect. |

The initial corrected-runtime recapture at `15737720637db343dc446c3b8eb3683ca6ffba6f` stopped on an overly broad evidence assertion. Direct inspection of the Risks 1024px frame showed **“Near appetite limit” on three lines, one complete word per line, entirely contained in the badge and cell**. This is accepted as legible in the compact table. The universal two-line cap was removed; per-word line checks, single-word-label checks, exact record/column/label bindings and text→badge→cell containment checks remain. The evidence agent added a positive regression for the three-word/three-line presentation and verified that old-style overrides reproduce and reject the original broken-word defects. The final `527b3486` revision corrects the evidence script while retaining the corrected runtime build digest. It does not waive V1 or V2.

No additional blocking visual defect was found in the final 68-frame runtime inspection or the 26 original-source comparisons.

## 5. Anatomy, density and responsive decisions

| Region | Direct visual observation and scoped decision | Explicit comparison boundary |
| --- | --- | --- |
| Common heading and menu | One Risk & Losses title, tenant context, subdued heading wave, report action and a single-row underline menu precede the collection. The selected phone tab remains visible and identifiable. The collection canvas provides consistent separation behind small text in both themes. | The approved runtime destinations are Risks / RCSA / Key indicators / Loss events. Original Overview, Actions and period controls are adapted to existing Reports and Work ownership. The 56px phone rail and shared control sizes are retained. |
| Risks | The register follows its bounded search/status controls; secondary appetite information follows the records. Desktop facts remain aligned. At 768/390 the records use two-column labelled facts and a full-width action. First record identity is visible in the initial phone frame. The compact 1024 badge presentation is accepted as described above. | Table-anatomy adaptation only. The original KRI collection is not a Risks register. All facts remain available even where the first phone action falls below the initial viewport. |
| RCSA | Cycle identity, stage, frozen counts, handoff, owner, updated date and review action have a clear reading order. The grid uses three columns at 1440/1280, two at 1024/768 and one at 390. Two populated synthetic cycles leave the third desktop slot empty. The complete first phone card and review action fit in the capture. | Actual frozen population and stage replace illustrative completion percentages and due dates. Cards are taller: the first 1440 runtime card measures 333.97px versus 257px in the original. This is a reviewed content/control adaptation, not equivalent density. |
| Key indicators | Native value, unit/limit, condition, scope/owner, movement, timestamp and exact review action remain readable. The corrected condition words stay intact. Phone facts retain their meaning and clear action label. | The supported Indicator type query replaces unsupported original search/condition controls. Added native metadata, shared type sizes and 44px review controls create taller desktop rows: the first 1024 runtime row measures 169px versus 64.5px in the original. This substantial density difference is accepted for the supported composition, not presented as pixel parity. |
| Loss events | Gross, Recovered and Net remain separate, currency-bearing facts. Recovery status and record status remain distinct. The desktop ledger becomes two-column fact cards at 768/390. Mixed-currency partial evidence retains each currency without creating a false aggregate. | Actual search/status controls and a Record loss action remain above the phone ledger, while the original has fewer controls and hides its rail. This moves the first runtime identity lower than the original. It is a documented workflow/shared-shell adaptation. |
| 768px table reflow | Risks, Indicators and Losses use labelled cards within the available content width. Labels and actions are readable in both themes. | The shared rail reduces available container width. The source KRI table remains narrow at 768 and has clipped right-side content within its collection; matching that clipping is not a requirement of the approved container-based reflow. |
| Empty, denied and partial | Empty populations remain distinct from unavailable populations. Retry controls are readable. Partial warnings precede available records. Stale assessment and unavailable native values remain explicit. | These are synthetic UI states. They do not prove live authorization, tenant completeness or backend recovery. |

The geometry below comes from `after/manifest.json`, using the top of `firstIdentity` in each **390px light** matrix frame. It measures the actual record identity rather than the top of its surrounding card.

| Destination | Final first-identity top | Relevant comparison |
| --- | ---: | --- |
| Risks | 595.0625px | Same-fixture baseline: 1286.609375px. No dedicated original Risks identity exists. |
| RCSA | 524.5625px | Original RCSA identity: 508.78125px, with different illustrative content. |
| Key indicators | 522.0625px | Original KRI identity: 584.78125px, with different records and controls. |
| Loss events | 655.0625px | Same-fixture baseline: 1830.703125px. Original Loss identity: 477.78125px; runtime retains real search/status controls and the approved 56px rail. |

All four final first identities are visible within the 844px phone viewport. This does not imply that every record fact or action is above the fold. Risk, Indicator and Loss cards continue below it; the full-width actions remain in the scrollable record. The automated manifest reports **no horizontal document overflow in all 68 runtime captures**. Intentional horizontal scrolling of the local menu is distinct from document overflow; a partially visible unselected neighboring tab at the phone edge is accepted.

At 768, the inherited shared actor chip is visually truncated, with its role context still identifiable. This pre-existing shared-shell constraint remains outside the Risk & Losses page correction and prevents any broader claim of complete shell parity.

## 6. Selected-record source comparison

The six original selected-record frames were opened directly and compared with the already reviewed exact-runtime detail frames. Their manifest binds them to the final runtime revision and image hashes. Every source is light-themed. The 1440 comparisons pair light source with light runtime; the 390 comparisons pair light source with dark runtime to inspect anatomy and legibility, not color equivalence. **The source and runtime records differ in every pair.**

| Subject | Original selected record | Runtime selected record | Reviewed adaptation |
| --- | --- | --- | --- |
| RCSA | Head Office assessment, illustrative completion/pending facts | Q3 Technology RCSA, frozen risks/controls and current challenge stage | Both use a 480px right sheet at 1440 and the full 390px viewport on phone. Runtime presents frozen population, phase and an inline challenge handoff instead of source percentage/deadline facts and a fixed “View all assessments” footer. Title, 44px close control, phase and action are readable; internal records continue in the sheet's scrollable content. |
| KRI | Failed privileged logins, illustrative count and history | Mobile transaction success rate, native percentage and Program/intervention handoffs | Both use the same 480px/full-phone inspector relationship. Runtime retains the native value and limit, condition, coverage, timestamp, owner and history. Its actions are inline with the record, rather than the source's fixed collection-return footer. Phone wrapping does not overlap the close control or fact columns. |
| Loss | Account settlement exception in a 480px/full-phone source inspector | Duplicate merchant settlement on its existing standalone detail route | An explicit inspector-to-standalone workflow adaptation. Runtime retains separate amounts and currency, event facts, responsibility/links and recovery work with a visible Back to losses action. It does not replicate the original drawer, compact title, fixed footer or all-above-fold density. |

Risk exact records also retain their existing standalone route for governed assessment/control-link workflows. The original does not provide a Risk selected-record view, so no corresponding original-detail comparison is asserted. Risk and Loss standalone pages retain their existing detail heading treatment; this collection overhaul does not establish pixel-equivalent detail pages.

## 7. Automated receipts and remaining limits

`after/risk-loss-review.json` reports **PASS: 68/68 captures, 20 original collection frames, 40 source/runtime bindings and 16 accessibility scenarios**. Its bounded interaction checks include exact readers, Back/Forward, filters/paging, empty/denied/partial populations and the recorded layout rules. The separate selected-record reference manifest reports PASS for six unchanged-source captures and their exact runtime bindings. These results corroborate the named review; automated PASS is not itself a visual-parity verdict.

The final accessibility receipt reports **16/16 scenarios PASS**, zero listed axe violations, zero undersized primary controls and zero scenario errors. It explicitly retains incomplete checks: 24 non-text-character contrast readings and eight partially obscured tab readings across the scenarios. Supplemental contrast observations total 736 PASS, 58 INCOMPLETE and 64 REVIEW. Filter-affected shell readings and low control-boundary contrast still need their stated interpretation; this review does not convert those outcomes into passes or claim complete WCAG/assistive-technology certification.

The scoped visual decision is limited in the following ways:

- The original provides only a light theme. Dark runtime evidence assesses hierarchy, separation and readability; it does not imply an original dark design exists.
- Different source and runtime data, supported metadata, shared controls, navigation ownership and retained workflows prevent meaningful pixel-diff equivalence. The visible composition differences above are part of the decision.
- Screenshot coverage is limited to the named fixture states and viewports. Behavior, focus containment/return, query semantics and actions below the captured viewport depend on separate receipts, not a static picture alone.
- Synthetic unavailable/partial states and a fixture-selected actor do not establish live authorized results or four-actor operational acceptance. Production readiness, deployment and cutover require their own records.
- No formal screen is promoted by this report. The acceptance count remains **0/19**.

## 8. Individually inspected image inventory

Every listed SHA-256 was recomputed from the PNG and matched its manifest. The appendices enumerate all 94 images personally opened by the named reviewer. Source/runtime pair relationships are retained in the two source manifests; identical source hashes in the Risks and KRI collection rows are expected because the former borrows KRI anatomy.

### 8.1 Final runtime — 68 frames

Prefix: `docs/evidence/2026-10-22-risk-loss-v2/after/`.

| Inspected PNG | SHA-256 |
| --- | --- |
| `250-risk-loss-risks-dark-1024x768.png` | `3f5f44c52b715dde7617b73721f11d49f461e933f68eccdb197456e31e19c7d6` |
| `250-risk-loss-risks-dark-1280x800.png` | `5fbee4bfbe29c682cd416083baf4f3ebcc96e1bb36e2a3c0de2d472c41beb9e5` |
| `250-risk-loss-risks-dark-1440x900.png` | `135eb22099602f35a04fde382682eefdeb7b8eff1c514572cf7ddb81f47eb678` |
| `250-risk-loss-risks-dark-390x844.png` | `ab153695f14b07e12b52bd880d90d28f91ab5b69b1088d79469d93994a4d8cdc` |
| `250-risk-loss-risks-dark-768x900.png` | `69549c6450a189e8ea143f961baa20712d10ba25227057033c25d32d39a22533` |
| `250-risk-loss-risks-light-1024x768.png` | `1d81c3503b028fd57be31ea8afd6cdce8b10481f90c73e47fa969e78482dd9c8` |
| `250-risk-loss-risks-light-1280x800.png` | `0bb2a0348b033e5ed5157de3becbfc5bf5781088affa4d8b072970d441bec3fe` |
| `250-risk-loss-risks-light-1440x900.png` | `43655531528a49a70ccba1ce17cab44c247d6553b2156010ea50045bf8234af8` |
| `250-risk-loss-risks-light-390x844.png` | `6eecb2eaa4ef9db2add7d0bc159b3f480927768825d992ad4482862cdf3489f2` |
| `250-risk-loss-risks-light-768x900.png` | `389b383190bc98f40c2e5e4acebbe46f02400dd282c767a3b5d8b2b61e196d6d` |
| `251-risk-loss-rcsa-dark-1024x768.png` | `a80a9345d3ae0ce37e37643a770623154f3955dc846fa39a022ab0338c579748` |
| `251-risk-loss-rcsa-dark-1280x800.png` | `86b22c8c70bb50ca6de3b379c02871657bd738238bfe042691dd79b6936ab14b` |
| `251-risk-loss-rcsa-dark-1440x900.png` | `b159937db706c2182f33449794c55eacf4d664911383834f46419af338b7957e` |
| `251-risk-loss-rcsa-dark-390x844.png` | `ab76a3fb2dc6adfdf7309b2a7f5351f38e1f69d618cb1629fadc4bcb9a404bb0` |
| `251-risk-loss-rcsa-dark-768x900.png` | `dbbb921697153138b0759cb240ca67639aa15e0fcb643fe7ccf458c1fb7a56a4` |
| `251-risk-loss-rcsa-light-1024x768.png` | `eb0d3f1eee0d3f9df3c96dc6c1a9aabee142bdd8ccc6f4da7c8e78d4c7a845c7` |
| `251-risk-loss-rcsa-light-1280x800.png` | `06de052f2af616a0c7b58613a66a8ee1532a9ffdb68dc38ea4ea22bdba283bdc` |
| `251-risk-loss-rcsa-light-1440x900.png` | `07cceae2bcdb9ac77f09224644787cbc3411fa26e53cf4a9ed11b062ab075cf9` |
| `251-risk-loss-rcsa-light-390x844.png` | `8f41ad9f0adecd8891f714217f10815d13bee9b928560983c00edda61e133735` |
| `251-risk-loss-rcsa-light-768x900.png` | `f7d809c6b1e45c918743e49c27acbacaad8ec1c5e34d75f78c3ba77f301631ae` |
| `252-risk-loss-insights-dark-1024x768.png` | `fc8aa79e5446862e3ba75278934385c5b44b3b185d82ba7b1dd5ed5d9b1fbb35` |
| `252-risk-loss-insights-dark-1280x800.png` | `b9f2c397554a2bb0bd66b0168a773a46141e94d6630b046662ef7c3f572b2f5d` |
| `252-risk-loss-insights-dark-1440x900.png` | `9d105061e5547f6c856cd4c4dca4b8098da15c9f9bba39fda636a36808ce8652` |
| `252-risk-loss-insights-dark-390x844.png` | `9ae33e69f9dd59052c58bf20472df05821279b13b27d484ec89713e4db4e701f` |
| `252-risk-loss-insights-dark-768x900.png` | `14519e88a618f9e0d567b426ffc6dcc0b8612ccd6be2bab7a31c1500aa7e0ab2` |
| `252-risk-loss-insights-light-1024x768.png` | `f5990e0d2396a997c3ce12f1ade1ac4b82adfecb21931b1cbab87529218b8fd3` |
| `252-risk-loss-insights-light-1280x800.png` | `e9344c6c346e6debbbb381ce71d660ab6b6c41df84c3698fda0f6a4c06aaa3d6` |
| `252-risk-loss-insights-light-1440x900.png` | `a6be45cce9f6966c2914d6c5854d850078dccaaafa12dfb4abbf921aca44abf1` |
| `252-risk-loss-insights-light-390x844.png` | `232d3a0d66e9dd1da958018dac3b1628d594a8f2b59c97aa8619520baa4c687d` |
| `252-risk-loss-insights-light-768x900.png` | `a9f3470347579db3188ebcb5da75797d05c554afa9b117c0d480606519174571` |
| `253-risk-loss-losses-dark-1024x768.png` | `e9fb09b1071e1b2ea7af87947c88ed1a5db5f00719882171ac57dcf6a29b5baa` |
| `253-risk-loss-losses-dark-1280x800.png` | `7ebbb7b33d8b57fdc6fb1961dc37ae46a3b568424988260747721d9e87a75309` |
| `253-risk-loss-losses-dark-1440x900.png` | `feccb594d900378bf87cf1f052ebd3645976ec0e2e8867dd923c29315e7d5faf` |
| `253-risk-loss-losses-dark-390x844.png` | `60400e6d8d0c72a994dba5eff79cea83a2bb8dab4eeddf5d2f6d7710a93bc2b3` |
| `253-risk-loss-losses-dark-768x900.png` | `dab5e5948933258e53cf2793d0dfadf5ddfa0172578c0b296a2b193b75bb7c65` |
| `253-risk-loss-losses-light-1024x768.png` | `6d8192eb48bbbecb31d3a563a0604217512918156b71f307b35dbc3376f689cb` |
| `253-risk-loss-losses-light-1280x800.png` | `92ca055b081ff17ef61cd32ca1967b5e5b9f3e3dddd91cd6b90b65ede3d769dd` |
| `253-risk-loss-losses-light-1440x900.png` | `355e5748e0ff9283ff60107b41d39e84bd1e48e4fd775b42310af7ddc047e15e` |
| `253-risk-loss-losses-light-390x844.png` | `436028ddd9924558c3b8eb436917fc5cb34ee6edce1d661af6e5c12245f4da17` |
| `253-risk-loss-losses-light-768x900.png` | `1246bf62faece391e7f77e9abaf427485f1393346b816fee28e38351d0d740f3` |
| `2540-risk-loss-risks-detail-light-1440.png` | `f6695ec5394f17e6162fbbd2409b83947e5bc615fdf04f1ff88dedfbd584a572` |
| `2541-risk-loss-rcsa-detail-light-1440.png` | `fd8fe70f52b2cbe8e36ec776c203d4092743a443571927a74d9191824f3c83db` |
| `2542-risk-loss-insights-detail-light-1440.png` | `412330e764be5ae917973a384b6b433fe5e9d7da4800973884ec88c22e9ec297` |
| `2543-risk-loss-losses-detail-light-1440.png` | `390fc864ae8feb2d196da867cad28d1b87721690706cf3e2604b5f556526cc0f` |
| `2550-risk-loss-risks-detail-dark-390.png` | `83037dfd4bcd9451173be8862af15526c3ac7bcd0ba039fc03ff1757f8055a44` |
| `2551-risk-loss-rcsa-detail-dark-390.png` | `1ebe58f7fa8ea857a721ef8699aabcd225dfbcdd3267c42107faa285bb36c470` |
| `2552-risk-loss-insights-detail-dark-390.png` | `34b9f4159dc17a0a2792688ca8dce38c04b9ecbd89a6821392fa0b855c7ec7a0` |
| `2553-risk-loss-losses-detail-dark-390.png` | `34285cb7119334fb905283bd8973f5dbea8b77e48a69eab66e50f15435e5b82f` |
| `2560-risk-loss-risks-history-light-1440.png` | `c4a798a10e545aa25c5bffd94fb0bf3d2f655d8a3a9e9267cfd5407c96a3aa44` |
| `2561-risk-loss-rcsa-history-dark-390.png` | `ab76a3fb2dc6adfdf7309b2a7f5351f38e1f69d618cb1629fadc4bcb9a404bb0` |
| `2562-risk-loss-insights-history-light-1440.png` | `c817fe4e916efb4f0935827ab3134735e20bc3417403e099585a8c5c94f02cb0` |
| `2563-risk-loss-losses-history-dark-390.png` | `60400e6d8d0c72a994dba5eff79cea83a2bb8dab4eeddf5d2f6d7710a93bc2b3` |
| `2570-risk-loss-risks-empty-light-1440.png` | `a2bd59c748a476da94adff59ae4c3ed634cc52796899e94385da46a93b1a1194` |
| `2571-risk-loss-rcsa-empty-light-1440.png` | `3dd2c2954048703459207a8952c5be484466fd2aa48ce66d00a160144e93e8fb` |
| `2572-risk-loss-insights-empty-light-1440.png` | `544f3e1ac4a1a4c6cae8f0cb9b737c6e6a000055033b974f0980ca3dd5833cbc` |
| `2573-risk-loss-losses-empty-light-1440.png` | `c262d836eafec684c4431fa2fe683dd25c6dcdae4e28a4cc300e410796312adf` |
| `2580-risk-loss-risks-denied-dark-390.png` | `0c94d160fa1dc7dfc2c22c21627fd990a07322f7525b8daedcb2ffe958298516` |
| `2581-risk-loss-rcsa-denied-dark-390.png` | `87919c8d7804f375ad9190e913fd5b89b325fec0f9a6c9a7d8630d10d6436ebd` |
| `2582-risk-loss-insights-denied-dark-390.png` | `5bf54d65725c4bf43dca6f16fdb1d04d8b87c3ea6e23a82cb006a828cda174e2` |
| `2583-risk-loss-losses-denied-dark-390.png` | `2b9fed736352b3e77dc4e70a82b5459c0ecad8dd2be8ac8c17245fdab7a3e278` |
| `2590-risk-loss-risks-partial-dark-390.png` | `dfa2fb93ee21aae038bafc8f6c3446030475b91e68447885f5af0a4fffbd753e` |
| `2591-risk-loss-rcsa-partial-light-1440.png` | `bb1235d143a8e820c8f0a0bc9c667c44728a982808e2d79ae382d6715e382331` |
| `2592-risk-loss-insights-partial-dark-390.png` | `af6331d65ec954771a2a94673a485644b398be44df2d7374644872855b4c5f1d` |
| `2593-risk-loss-losses-partial-light-1440.png` | `101b113a0aac453d626d831cacd45723140ec128c3aa0fe5d2d84560075333d3` |
| `2600-risk-loss-risks-filters-light-390.png` | `47724f5c5b5918007dcf07c856e0c015bd582174ee336971ec152788d63bd38a` |
| `2601-risk-loss-rcsa-filters-light-390.png` | `b1c24ec418c2767eeb4c6f559790a31dd92ca23f198940df66e71742808777b1` |
| `2602-risk-loss-insights-filters-light-390.png` | `1712a910796441b092a0cb49e23d0edb59b03b7feb77317c0188e3c02f29c9f7` |
| `2603-risk-loss-losses-filters-light-390.png` | `64b275b26caca4e71e41ce129efb0fa9e156547c0652053141456bc685aa1ece` |

### 8.2 Original collections — 20 frames

Prefix: `docs/evidence/2026-10-22-risk-loss-v2/after/approved-reference/`. Five Risks frames are anatomy references; the remaining 15 correspond to RCSA, KRI and Loss collections.

| Inspected PNG | SHA-256 |
| --- | --- |
| `approved-risk-loss-insights-1024x768.png` | `683feb8eb8e9474e2fd1d64c2ed3e745d0a36e35e8291a96c3fd4ab5040c955a` |
| `approved-risk-loss-insights-1280x800.png` | `88949149b6125e2944bbe23b67e609ac21ff8ee54dec5ee7297d367c4e1eb8ea` |
| `approved-risk-loss-insights-1440x900.png` | `5717f9696ba3731502320c1181d5fb395a567c451ea0f78e63bf01ddfcd5fc1f` |
| `approved-risk-loss-insights-390x844.png` | `4eb706e0eeee927ae476bd3bbe420c673e00e50f8de2a6ac1d741ff3114a6869` |
| `approved-risk-loss-insights-768x900.png` | `7606b1d3da3557ce622ec412dbacc0185d8f1cc22df68d497d3bd850658fd990` |
| `approved-risk-loss-losses-1024x768.png` | `7ea0c978ff75569e549f83d31fcd01c952abd4e3d2492d654b046f6ab6bac483` |
| `approved-risk-loss-losses-1280x800.png` | `f4bfed76e0925243da44c3bcb4d018b2d07aaa5033498483cc20165ecb69a71e` |
| `approved-risk-loss-losses-1440x900.png` | `bc4953735fb257c275e628067d52f504429aba22d4017744ded8b0e0a975fd06` |
| `approved-risk-loss-losses-390x844.png` | `479bf589cc85a9e0a7b713a6ce29ff5141c94f21efea531dda51f8ee79d9b65d` |
| `approved-risk-loss-losses-768x900.png` | `95dd0a23a1fd8943981eb7ffac8231c7b226749051861103205263533ebdbb32` |
| `approved-risk-loss-rcsa-1024x768.png` | `9b20b4957491a493cae6ddc3cc2c52273386d248e5285f8e2bfa865878ddda0c` |
| `approved-risk-loss-rcsa-1280x800.png` | `527a6bee82fe59a2d0e9c1403134cda114400ff6ff4edf3c6e2490537c3156d9` |
| `approved-risk-loss-rcsa-1440x900.png` | `349df5a9dc1fe9421b6bc42d803001821e758ac9fa98bd944442a566362c2145` |
| `approved-risk-loss-rcsa-390x844.png` | `8f2df5fbbd30e900587fa253f1346d7f4b8fde1b56a50df47a6d90b41d403e89` |
| `approved-risk-loss-rcsa-768x900.png` | `cefe4dc072d20b8d288bc77e410b5794bc561f59af84c08d6bd0e81618bf839d` |
| `approved-risk-loss-risks-1024x768.png` | `683feb8eb8e9474e2fd1d64c2ed3e745d0a36e35e8291a96c3fd4ab5040c955a` |
| `approved-risk-loss-risks-1280x800.png` | `88949149b6125e2944bbe23b67e609ac21ff8ee54dec5ee7297d367c4e1eb8ea` |
| `approved-risk-loss-risks-1440x900.png` | `5717f9696ba3731502320c1181d5fb395a567c451ea0f78e63bf01ddfcd5fc1f` |
| `approved-risk-loss-risks-390x844.png` | `4eb706e0eeee927ae476bd3bbe420c673e00e50f8de2a6ac1d741ff3114a6869` |
| `approved-risk-loss-risks-768x900.png` | `7606b1d3da3557ce622ec412dbacc0185d8f1cc22df68d497d3bd850658fd990` |

### 8.3 Original selected records — six frames

Prefix: `docs/evidence/2026-10-22-risk-loss-v2/detail-reference/`. Runtime counterpart filenames refer to the `after/` inventory above. All pairs are content/workflow adaptations, with different selected records.

| Inspected original PNG | SHA-256 | Runtime counterpart |
| --- | --- | --- |
| `source-kri-selected-1440x900.png` | `e30b46211df3b6b56c77c904b1ac772176f81c565568bde607406c2764857d84` | `2542-risk-loss-insights-detail-light-1440.png` |
| `source-kri-selected-390x844.png` | `7c24131e19a2045b59b715d66cafeb6ffa16648f43b37f6a6d4b54f08bb71710` | `2552-risk-loss-insights-detail-dark-390.png` |
| `source-losses-selected-1440x900.png` | `d1d7998e78045c787b27e000c5a382956acf8de4e7d0551fbb3ba261559e72c7` | `2543-risk-loss-losses-detail-light-1440.png` |
| `source-losses-selected-390x844.png` | `e6fbcd0def3ceccc22c85e25e7ab7d8831a2d59706ae9a1be6052a667328a7b7` | `2553-risk-loss-losses-detail-dark-390.png` |
| `source-rcsa-selected-1440x900.png` | `5b35b72100b57e2c2c8cf48dd79d4a0557dbff5a6ad34fee3e57faea79036286` | `2541-risk-loss-rcsa-detail-light-1440.png` |
| `source-rcsa-selected-390x844.png` | `07386c7fc54aab501f9a9fe00a1f8bbfbb76b34e31481e5545ca0cc4c39b45ff` | `2551-risk-loss-rcsa-detail-dark-390.png` |
