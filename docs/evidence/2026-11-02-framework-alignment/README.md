# Framework Library source comparison

Source: `clearsight-framework-designs/framework-library.html`, SHA-256
`193d13b0cf87381f20b7d84c3d4c50c80b0e064274a623fa98257506df8284bf`.
The source is illustrative; runtime captures use the `framework-library-loaded`
evidence fixture and existing catalog, scope and rights behaviour.

The before captures showed a normal-weight page title and muted record cards.
The corrected production component uses a 700-weight title and a shared
surface-card variant. The `source-*`, `before-*` and `after-*` images cover
1440, 1280, 1024, 768 and 390px; runtime after captures include light and dark
themes. The new component variant is also shown in the shared gallery captures.
At each width the page had no document-level horizontal overflow. The first
record remains at y=424px at 1440px and y=556px at 390px in this fixture; this
change does not claim to correct all density differences.

Selecting the edition by keyboard preserved the explicit selection gate: the
desktop preview could not choose a Program before selection, and the selection
enabled that action. At 390px, closing the selected-edition sheet returned focus
to the selected row. The `selected-*` captures show both themes at desktop and
phone widths. Long-title images use injected display text solely as a layout
probe; neither viewport showed document-level horizontal overflow.

Axe colour-contrast checks on the ten populated layout states found no definite
violations and one incomplete rule result per run. A WCAG A/AA automated check
on selected desktop and phone states found no definite violations; the
`aria-prohibited-attr` and `color-contrast` rules were incomplete. Manual
accessibility review remains required.

During the comparison, Windows checkout conversion changed the bytes of all 96
manifest-pinned source files. The archived and Git-blob hashes were correct;
the working files were restored only after confirming line-ending conversion
was the sole difference. `.gitattributes` now preserves their bytes, and the UI
contract check verifies the manifest hashes. No source content was changed.

The supported filters and explicit edition-selection step intentionally differ
from the prototype's example filters and adoption data. Catalog-contract,
authority, complete visual and operational acceptance remain open; this is not
a claim that the Framework screen or all 19 source screens are accepted.

The later `ink-*` captures record the shared light primary and strong text
alignment to the source's `#132547` and `#0a1d43`; dark text is unchanged.
Framework was captured at all five widths in both themes, and Reports, Risks
and Oversight at 1440px and 390px in light mode. These evidence fixtures do not
cover every data state. No document-level horizontal overflow or definite axe
colour-contrast violation was found in those 16 captures; each axe run reported
one incomplete rule result. Manual contrast and broader workspace review remain
open. The earlier `after-*` images preserve the preceding text colours.

The `filter-*` captures show the Framework search field filling its grid column.
At 1440px the gap to the rights filter fell from 163px in `ink-light-1440.png`
to 16px. The `filter-active-*` captures show the clear action after filtering.
The action previously caused horizontal overflow near 768px; the compact row now
wraps it. Search and clear restored the two fixture records without horizontal
overflow at 1440, 1280, 1024, 901, 900, 768, 767, 701, 700 and 390px in both
themes. The evidence runner now checks compact desktop spacing and active-filter
overflow. These checks do not establish full-screen acceptance.
