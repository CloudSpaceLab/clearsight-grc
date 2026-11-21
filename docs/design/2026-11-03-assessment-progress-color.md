# Assessment progress color

The approved Assessment HTML uses a blue completion indicator. Chromium's
native progress rendering ignored the existing accent color and showed green
when an answer was entered. Style the existing native progress element with
the shared primary-action and surface tokens, including browser-specific fill
pseudo-elements. Preserve its answer-derived value, semantics and dimensions.
Check an answered fixture in both themes and at all approved widths.
