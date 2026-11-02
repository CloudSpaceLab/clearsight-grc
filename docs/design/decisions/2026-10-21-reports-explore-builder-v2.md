# Reports Explore and Build report — V2 layout adaptation

Tracking: #462. Status: implemented and scoped review passed; **partial V2 layout integration, 0/19 whole screens accepted**.

## Decision and source

Continue from the reviewed shared-menu branch at `8ee6b43d1ef03d2072dee6fbab02387942643814` (#579), which includes the Reports Overview/library composition in #578. Replace the remaining Explore and Builder page anatomy, including their local menu and entry actions. Do not embed the former multi-panel analysis or saved-setup table inside a new decorative frame.

The immutable layout authority is `docs/design/references/clearsight-2026-10/source/clearsight-reports-insights-designs/reports-workspace-preview.html`, SHA-256 `97f9d04092fdad07e5081820ad913295dc17d8344a693bd229c52d521c5b8895`. The companion CSS hash is `15bad94b38866fc32cf2d464360ebf769c575c3407f058c0859745fb6a470de7`; JavaScript hash is `10e7d118dfac2fcd9f00daf05fe36751e6a960b00a8e263b53ed7726e7f95cea`. Source files, notes and manifest remain immutable.

## Whole-page mapping

| Area | Source anatomy | Production adaptation |
| --- | --- | --- |
| Local menu | Overview / Explore / Reports / Schedules; Build report appears only while selected | A contextual shared Tabs item, retained after first visit so navigation does not discard drafts or pending work |
| Entry actions | New report opens Builder | New report opens Builder; Generate report retains the existing approved-setup generation operation |
| Explore | Data rail, selected chart, exact breakdown table | Existing governed risk/loss reads, one measure at a time, exact values and source/period/scope information; unsupported comparisons or saved analyses cannot imply success |
| Builder setup | Title, template and included sections | Existing name, area, report-type presets and exact issue selection for board briefs |
| Builder paper | Substantive document-shaped preview | Clearly labelled setup preview showing the actual definition/preset, scope, filters and supported output contents; no fabricated rows, figures or generated-file preview |
| Builder output | Format, delivery, progress and preview | Stored format and governance state; existing save/review/activate/retire commands and generation for active/effective setups; manual download, no simulated recipients or recurring delivery |

## Responsive contract

Explore uses a 240px Data rail, 200px at 900px and below, a full-width three-control row at 700px and below, then stacked controls at 480px and below. The selected chart and exact-value breakdown follow the controls. Existing shared chart/table semantics and minimum 44px targets apply.

Builder uses 210px setup / flexible paper / 210px output above 1380px. At 901–1380px output spans the row below setup and paper. At 701–900px output sits left of setup above the full-width paper, matching the original CSS auto-placement verified in the rendered source. At 700px and below the order is setup, paper, output. The shared document tokens provide light/dark reading surfaces. Local navigation remains an underline row with horizontal scrolling. Selecting a contextual tab reveals its current marker horizontally without moving the page vertically, an accessibility adaptation of the source phone state that clipped the selected Builder tab.

## Behavior and data boundaries

Reuse the current metric APIs and report definition/run APIs; introduce no domain store. Risk posture is current; loss flow uses the verified reporting interval and requested organization scope. Report run date filters select record creation dates and must not inherit the loss accounting interval. Board-brief generation uses its exact issue and accepts no date/owner overrides.

The workspace owns one definition snapshot shared by Builder and generation. Definition creation and transitions return the resulting stored definition to that snapshot so a newly activated setup can be generated immediately. Refreshes reconcile returned versions; a retired or removed selection cannot silently substitute another setup. A delayed run-list read preserves an accepted queued receipt and its status polling. Version/checksum checks, distinct server-verified actors and current authority remain server enforced. Prevent duplicate in-flight submissions and retain recoverable drafts/errors. Unsupported source operations—arbitrary sections, saved analyses and recurring recipient delivery—remain unavailable.

## Evidence and acceptance

Retain the #579 baseline artifact `12358294584` from run `39628265479` (ZIP SHA-256 `92549447578e3900c23ba9ff425f0f05a16e9ec51d8410dc462e20c023a54043`). Add original/runtime same-screen Explore and Builder pairs at 1440, 1280, 1024, 768 and 390px, light/dark runtime states, geometry/first-content assertions, keyboard/touch checks and meaningful partial/empty/denied/error/command cases. Source Builder capture must use its real New report action rather than force a hidden tab.

The scoped implementation review below records the exact runtime revision, source hashes, inspected images, workflow results and limitations. The final-head CI and retained artifact are separate PR gates. These checks do not establish four-actor execution, backend authorization, source-to-report reconciliation, production deployment or whole-screen acceptance. The production cutover block and 0/19 register remain unchanged.


## Implementation review — 21 October 2026

**Reviewer: ChatGPT `/root`, AI visual and interaction-evidence review.** Independent source/geometry review: `/root/layout_audit`. Lifecycle and race audit: `/root/ci_review`. Runtime implementation checkpoint: `0ea642b01afd96213827605433d6a00727c8bfab`, based on #579 at `8ee6b43d1ef03d2072dee6fbab02387942643814`. The following evidence commit contains the capture contracts and this review; it does not change that runtime. The [machine-readable scoped receipt](2026-10-21-reports-explore-builder-evidence.json) records the image inventory, hashes and limitations.

The reviewer inspected all 20 final matrix images, `222-report-explore-{light,dark}-{1440x900,1280x800,1024x768,768x900,390x844}.png` and the equivalent `223-report-builder` set, against ten original `approved-reports-{explore,builder}` renders. Additional full-page inspection covered the setup paper/output at 1280/768/390 and the large exact NGN amount at 390. Original content remains illustrative and was never used as runtime figures.

The retained #579 baseline includes `140c-report-explore-light-1440x900.png`, `140d-report-explore-light-mobile-390x844.png` and `143-report-setups-dark-mobile-compact-390x844.png`. Explore previously embedded the older risk/loss surface and collapsed to a notice when reads were unavailable. Builder previously led with saved-setup counts/table. The replacement preserves the Data rail and chart region in unavailable states and provides the complete setup/paper/output composition for new and stored definitions. Loaded synthetic cases demonstrate layout and request contracts, not an improvement in live bank data availability.

| Review area | Result |
| --- | --- |
| Heading/actions and menus | Real New report enters Builder; Generate report keeps the existing approved-setup command. Four persistent local tabs plus selected-only Build report; current destination is visible on phone entry and return. The shared primary menu from #579 is preserved. |
| Explore composition and density | Source-width Data rail, one selected measure/chart and an exact breakdown table. Current risk and period loss bases remain distinct. Explanations for unavailable controls stay readable outside disabled opacity. |
| Builder composition | Setup/paper/output at 1440; output below setup/paper at 1280/1024; output left/setup right above paper at 768; setup → paper → output at 390. Supported contents and stored filters are explicit; generated data is not simulated. |
| First viewport and overflow | All 20 matrix frames show the selected tab and first useful controls; maximum page horizontal overflow is 0px. As in the original, the phone chart and phone/tablet paper can begin below the first viewport. |
| Keyboard and touch | Tab arrows and contextual return, retained draft/measure/currency, sheet focus containment, Escape restoration and 44px required targets pass. The member table uses its container-responsive contract. |
| Data and recovery | Empty/partial/unavailable/denied, selected organization scope, exact source/revision/count and separate large/signed currency values pass. Retired definitions cannot substitute another setup; delayed refreshes retain accepted queued runs and polling. |
| Commands | Isolated fixtures record create → submit → review → activate → queued run with returned version/checksum, plus a 403 review rejection that preserves pending review and creates no run. Board briefs retain their exact saved issue reference and ignore date/owner overrides. |

### Verification receipt

- Full web suite before the final focused accessibility additions: **258 files / 2,021 tests passed**. Subsequent Explore/copy tests passed **26/26**, shared Tabs/gallery **9/9**, and member-sheet/DataTable **13/13**. The exact final-head full suite remains a PR gate.
- Customer/evidence builds, TypeScript, runtime-fixture boundary, copy-quality and UI-contract checks pass. Initial customer CSS is **31,882 bytes gzip**, below the unchanged **32,768-byte** budget.
- Final Reports capture: **38/38** cases, including **36/36** new source-layout/state/command cases and two retained generation/setup cases. Ten immutable original renders pair with twenty runtime theme frames; all image hashes verify.
- Scoped accessibility: **12/12** Explore, Builder and nonempty member-sheet cases across light/dark and desktop/phone, with **zero automated violations and zero undersized required controls**. This receipt follows contrast and member-sheet corrections; the final tab-reveal-only change is covered separately by final geometry and Tabs tests.
- All **96** original archive file byte lengths and SHA-256 values verify; no original source or source manifest changed.

| Retained local receipt | SHA-256 |
| --- | --- |
| Runtime capture manifest | `834b50f9969376e6874cd07bb982c69f71dc0b0ed0024b8838b20e0dac18bd20` |
| Source pair manifest | `84ef6183dff0368040a43ccd9763084c365cb20d502f10f13f6e4ff97f15bdaf` |
| Accessibility receipt | `6b7e5cb67b5e2163ac6a80506371093f4eb1b6376a4f9e375f87c60d4751fec3` |

**Result: partial V2 layout adaptation verified in this scope.** The local Chromium153 review is supplemented by the pinned CI Chromium140 artifact gate. Gradient-backed header/tab text and decorative select chevrons retain manual accessibility readings; automated PASS is not full accessibility conformance. Four-actor execution, backend authorization, hosted generated files and source-to-report reconciliation require their own retained evidence. Whole-screen acceptance remains **0/19**, and the production cutover block remains.


## Post-review corrections — 21 October 2026

Runtime follow-up: `c73cece84a90b0fd54fe1cb1e4885be3ca08ad2d`. Reviewer: **ChatGPT `/root`**, with `/root/ci_review` for lifecycle regression, `/root/reports_data_contract` for data and selection semantics, and `/root/layout_audit` for targeted browser evidence. The earlier scoped matrix remains a historical receipt at its recorded revision; its hashes have not been reassigned to this follow-up.

The first PR head `ef697f1b3e8f5ae8ebcaea8f2d81a9345bbad243` failed one existing unmount-cleanup assertion in [CI39645557310](https://github.com/CloudSpaceLab/clearsight-grc/actions/runs/39645557310) and the same component-validation step in [UI39645557122](https://github.com/CloudSpaceLab/clearsight-grc/actions/runs/39645557122). Both checked synthetic merge `b96b6a98d98b4e4ba8f964b85b159fb74490a2e8`; the UI workflow stopped before browser capture. Tabs now cancels its owned animation frame on replacement and unmount, while a previous trigger's ref detach cannot cancel the newly selected trigger's reveal. The original failing assertion is unchanged and passes.

The same review removed implementation commentary from the complete shared record-drill workflow. Explore now labels the backend's entire active-risk population as **risks in scope**, retaining separate unknown/excluded counts; it does not mislabel unknown assessments as assessed. Board brief search changes and failures clear the previous issue selection, and saving requires a live visible issue. A regression proves pending and failed searches cannot save the old hidden target, followed by successful explicit reselection and recovery.

Focused results are **17/17** lifecycle/Tabs/gallery, **29/29** Builder/Reports workspace/copy, **26/26** Explore/copy, and **28/28** shared member/oversight/copy tests. The evidence build, correlated browser/manifest expectations and whitespace checks pass.

All **six** member-sheet cases at 1440/768/390 in both themes and **two** phone Builder entry/return cases pass. `/root` inspected all eight retained PNGs. Corrected count and table labels are visible; source-bound counts, focus containment, Escape restoration, 44px targets, draft retention and contextual selected-tab visibility pass. Maximum horizontal overflow is 0px; phone Builder return leaves vertical page scroll at 0.

Two tablet axe contrast readings remain incomplete because the count paragraph's full-width box overlaps the close-button box by 1px at the far right. The actual count text at x21–224.97/y65–79 intersects neither the heading (bottom53), close button (x704–748/y20–64), nor focus ring. Computed opaque-color contrast is **5.4467:1 light** and **6.6673:1 dark**. Manual screenshot and text-range review pass; these automated incompletes remain explicit and do not establish complete accessibility conformance. There are zero automated violations and zero undersized required controls in the six targeted member cases.

| Follow-up receipt | SHA-256 |
| --- | --- |
| Member and phone Builder checks | `9e77901ccd9185d93b865276eaf10c91a5e3a34f53ea8005f9c0f732834cadf1` |
| Tablet count geometry and contrast | `eebaa5bda128581b33ffa58f2839e7d69d1f5dc24f40db0a1db75d3b976c600e` |

The inaccessible-record notice and Board brief failed-search recovery are unit-tested; this receipt does not claim browser actor/backend acceptance for them. Final-head full CI and its retained artifact remain PR gates. **Partial V2 integration, 0/19 whole screens accepted, and the production cutover block remain unchanged.**

## Full CI and capture correction — 21 October 2026

Head `ec313b08522b30592bd478526d7b2420649a0edd` passed [CI39648103177](https://github.com/CloudSpaceLab/clearsight-grc/actions/runs/39648103177), including **258 web files / 2,024 tests**, TypeScript, the customer-fixture boundary, customer build and backend gates. Both workflows checked merge `e781db64e3fd1029f1a5cc2bd0cb1549f41443fb` onto the unchanged #579 base; its tree matches the head tree `16441c1182d5fedbbb461f7d3e67c7497c8aa001`.

[UI39648103187](https://github.com/CloudSpaceLab/clearsight-grc/actions/runs/39648103187) passed the same component suite, both builds, all **36 Reports layout/state/command cases**, **10 menu viewport plus four disclosure/permission cases**, **40 accessibility/touch states**, nine behavioral scenarios, eight Home T7 scenarios and the unchanged bundle budgets. The main capture then stopped because a generic text locator chose a retained inactive panel. Its artifact is incomplete: **242/284 flow records and 250/292 screenshots**; the later approved-source and follow-up verification steps did not run. Artifact `12368722100` is 37,806,180 bytes, SHA-256 `728a47e87521adf2c516394b37aad9e57212959c599e6f8e40aab27dbd82ac57`. This is a scoped Reports pass and a failed full UI gate.

The evidence-only correction scopes report records and inspector assertions to the active Reports tabpanel and Generated reports region. Expected content identifies the selected record, setup-preview heading, Report details dialog or Generate report dialog, as appropriate. Geometry, counts, inspector matching and failure checks remain intact. The focused `reports-expectations` capture passes **18/18** cases: all 16 library/detail/setup/generation cases, including every library variant, plus both navigation sheets over the Reports library. No runtime code or source-layout authority changes in this correction.

Reviewers: **ChatGPT `/root`** for the patch and retained visuals, **`/root/layout_audit`** for the 18-case capture, **`/root/reports_data_contract`** for the 36 Reports data/command receipts, and **`/root/ci_review`** for workflow provenance. `/root` inspected the 20 current CI layout frames and eight representative state/command frames; all 250 retained runtime image hashes verify. Final complete-artifact review and both workflows on the corrected head remain required before this PR leaves draft. The final receipt belongs in the PR body so recording it does not create an untested documentation head.
