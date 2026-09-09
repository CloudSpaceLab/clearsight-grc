# Document browser in narrow panes

The desktop vendor workspace can give Documents only about 630px even when the viewport is 1280px wide. Viewport-only rules retained the 208px file-type sidebar and a six-column table. The rendered baseline left filenames 33.8px wide; selecting a row opened another 224px inspector and reduced them to 10.6px, producing nearly vertical text. This correction responds to available space without changing document filtering, availability, selection or preview behavior.

## Layout decision

DocumentBrowser declares the named `document-browser` inline-size container. Its ordinary navigation switches to the existing labelled File type SelectField at 960px of available width. The sidebar remains for wider standalone views and is 14rem wide so all labels fit one line with their normal padding and weight. At 1100px and below, the selected-file inspector is hidden so it cannot consume the table's remaining space; Preview continues to expose file details. The picker retains its horizontal, independently scrolling type navigation above a 760px container width, with the existing selector below that size. The existing 760px viewport fallback remains.

DataTable adds the opt-in `responsiveTo="container"` contract. It declares the named `cs-data-table` inline-size container and reuses the shared stacked-row anatomy at 700px of table space, including the full-width filename column. Its compact labels align left and badges/actions retain intrinsic width. At 360px of table space, every cell stacks its label above its value so short status labels and Preview fit normally. Existing tables default to viewport responsiveness. The existing 700px viewport fallback remains for both modes. No new tokens, custom card implementation, resize observer, remount or breakpoint-triggered data load is introduced. Container selectors target browser descendants and table anatomy; preview dialogs keep their existing contract.

The table's existing component-owned compact rules are mirrored in a distinct named-container context, rather than restyled by document feature CSS. They must remain synchronized when the compact shared contract changes.

The document stylesheet also declares the already-approved layer order before its feature layer. A fixture build loaded its lazy CSS before the global stylesheet, otherwise establishing `features` before the reset's `base` layer and losing browser border/padding. [Computed before](../evidence/2026-09-09-document-explorer-pane/cascade-before.json) and [after](../evidence/2026-09-09-document-explorer-pane/cascade-after.json) receipts show the same 16px spacing token with heading padding restored from 0px to 16px. This is a bounded cascade correction, not a new layer or palette.

## Rendered proof

[Before evidence](../evidence/2026-09-09-document-explorer-pane/before/receipt.json) and [after evidence](../evidence/2026-09-09-document-explorer-pane/after/receipt.json) each contain ten cases with light/dark pairs: nested vendor Documents at 1280px, standalone Forms Documents at 1440/390/320px, and the horizontal picker at 1440px. Each case records initial and selected states, with matching PNGs. The before capture set the theme but did not explicitly persist density, and exhibited the recorded cascade problem. Final after evidence explicitly sets comfortable density and records computed spacing/border styles. The root-owned canonical nested cases also reproduced the original navigation failure with explicit comfortable density. The [render harness](../evidence/2026-09-09-document-explorer-pane/render.mjs) checks navigation replacement, table/card mode, usable full filenames, contained layout, inspector visibility, compact label/action alignment and selection continuity without document reads on resize. It waits two animation frames after viewport changes so nested container layout has settled before geometry measurements.

| State | Result |
| --- | --- |
| Nested vendor pane at 1280px viewport | Approximately 627px of table space; one File type selector; stacked rows; filenames remain on a single 21px line; inspector stays hidden after selection. |
| Standalone at 1440px | Approximately 1173px browser; sidebar and table retained; inspector remains available when selected. |
| Standalone at 390/320px | Selector and stacked rows retained; full filename values wrap normally, with 138px of filename space in the 320px case; field labels stack above values below 360px of table space. |
| Desktop picker | 839px browser; horizontal navigation and table retained. |

All ten after cases passed with zero document/browser horizontal overflow and no recorded page errors. Every visible file-type navigation label was measured as a single line after the final 14rem sidebar adjustment. Across each resize interaction, the selected row was preserved and zero document-list reads were made. Representative nested light/dark, wide standalone selected, picker and mobile renders were inspected. The original nested baseline demonstrates that zero horizontal overflow alone was insufficient: filenames were severely compressed inside the available layout.

The new shared responsive-mode test first failed against the old contract, then passed while preserving the selected row's DOM identity, focus and Enter action. DataTable and DocumentBrowser tests passed **15/15** after the change. The root-owned canonical Forms matrix includes additional nested-pane assertions and records its own final result separately.
