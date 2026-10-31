# Hosted UI diagnostic: PR #599, attempt 1

**Status: FAILED. This is a partial diagnostic record, not a release or whole-screen acceptance receipt.**

[Workflow run](https://github.com/CloudSpaceLab/clearsight-grc/actions/runs/41200449260) · [Exact diagnostic metadata](diagnostic.json) · [Failed editor image](existing-editors-and-import-guards-failed.png)

## Identity and retained scope

| Item | Recorded identity |
| --- | --- |
| PR head | `6d5d43ddec18f71c7a05d668c87901e2bcf3cd26` |
| Actual checkout | `7f14b9022c15899d2837957cd75da22bec1f88e8` — printed by the hosted checkout log |
| Checkout and head tree | `00aa9de91a46787000e9fb88ddc775f3b0e5a317` |
| Job | `124535036774`, failed at `2026-10-31T12:43:13Z` |
| Diagnostic artifact | `13043067730`, 97,228,345 ZIP bytes |
| Verified ZIP SHA-256 | `445529f3cb065ee9ef3674016b8264245b3caa81d27fc966ab8c105dd2162c73` |
| Full Enterprise receipt SHA-256 | `53fee6753a6440bdcca8210b39b4ea79b0a53bc933e6fd12fdb45fd073671ece` |
| Hosted build digest | `28e6d798572e8d37da39a977b639694fb68842515f73b4011517ff51cfe3d4b1` |
| Hosted browser | Chromium `140.0.7339.16` |
| Retained local comparison browser | Chromium `141.0.7390.37` |

The full Enterprise receipt is `enterprise-setup/receipt.json` inside the diagnostic ZIP. Its exact byte count/hash, source pin and fixture/capture/reviewer hashes are recorded in `diagnostic.json`. This directory retains **one PNG only**, plus this README and the compact diagnostic. The ZIP, full receipt and other completed screenshots are not copied here. Image/archive paths in the diagnostic describe the original artifact; they do not assert local retention.

## Observed failure

`existing-editors-and-import-guards` reached the imported-area edit form and selected Department. At capture-script line 259, the wait for **Edit Imported operations** to become hidden timed out after 30 seconds; the locator remained visible for 65 observations. The final image shows Department selected, the Type trigger focused, and Propose change enabled while the parent sheet remains open. No browser error or command was recorded.

The scenario had recorded only `managedEditorsOpenWithoutCommands` and `importedMoveRetireGuard` as complete. Later checks in that behavior are not credited. This record assigns **no root-cause or fix verdict**: the final frame cannot establish the listbox/overlay state at the instant Escape was sent.

## Independently verified partial evidence

The workflow executed 19 runners: 18 completed successfully and the Enterprise runner failed. Its global review retained **652/652 flow records**, **665/665 generic PNGs**, **77/77 governed Forms capabilities**, **138 accessibility states**, **10 Configuration cases** and **92 People cases**. The downloaded generic and Configuration images were rehashed independently (665 + 12 PNGs); the Work and People receipt reviewers replayed without failures.

Within the Enterprise receipt, all **10 viewport/theme layouts** and **10 other behavior scenarios** completed. All **42 recorded PNGs**, including the failed image, match their receipt hashes. The five-width source identity and all **96 original source files** were verified. Ten per-layout geometry records in `diagnostic.json` preserve source order, visible entity/first-area rows, tabs, context placement, overflow, accessibility and minimum 44px control measurements.

At 390px in both themes, the first actual area is at **y=684–733.890625px**. The truncated state records exactly **500 loaded areas**, with **“500 loaded areas · More areas not shown”** visible at **y=544.109375–562.109375px**. These are completed partial observations within a failed workflow.

The hosted build digest and input hashes match the [retained local receipt](../after/enterprise-setup/receipt.json), while browser versions differ. **37 of 42 hosted image identities are byte-identical to their local counterparts.** Four shared filenames differ (light 1280 overview/context; light/dark 390 context); the fifth nonmatch is the new failed-editor image. The comparison does not attribute a cause to those differences. The named reviewer directly inspected the key desktop/phone/truncation frames and the four changed nonfailing frames, with the exact inspection scope recorded in the diagnostic.

## Missing later evidence

The workflow skipped approved-design reference capture, the Risk/Losses source-runtime follow-up, proposal-source rows, and assessment-follow-up recovery after the browser-review failure. Their later artifacts, including the separate 217-view approved-reference set, were **not produced**. This is expected diagnostic incompleteness, not evidence of an additional runtime failure.

Earlier retained evidence remains unchanged. Synthetic browser checks do not establish live backend authorization, four-actor outcomes, complete WCAG conformance or whole-screen acceptance.
