# Document navigation and comparison — V2.15

21 November 2026 · Actual-source interaction fixes

## Observed defects and changes

| Observed behavior | Implemented correction |
| --- | --- |
| A folder, archive, format, state or sort change could snap back to the previous selection because rendering restored the old route before the new filter was remembered. | The canonical filter handlers persist their new route before rendering. Folder creation follows the same order. The existing history state is retained during URL replacement. |
| A destination without filter parameters inherited filters from an earlier visit. | The document route explicitly restores every library filter, including its default when omitted. The URL fully describes the displayed selection. |
| The primary document title opened a generic detail route while the secondary Preview action kept the selected version and return destination. | Both entry points pin the same current version and filtered return route. Document tabs continue carrying that return context. |
| A change after line 500 produced empty comparison columns because the diff discarded later lines before comparing. | Compare all text supplied by the bounded reader; limit only displayed changed excerpts to 500 lines and 30,000 characters per version. Long first-line prefixes are cropped around the first differing character, with the omitted amount stated. |
| A comparison could be mistaken for a complete original-file diff. | Show the reader's Office/text limits alongside the comparison. A text-file excerpt explicitly states that content beyond the first 300,000 bytes is not compared. Word previews report shown/total paragraph counts and explicitly disclose omitted content after the 2,500-paragraph limit. Selected-version identity is checked again after asynchronous text extraction. |

The comparison headings use “comparison selection” and “viewing,” avoiding an incorrect chronological claim when a newer version is selected as the comparison source. Empty changed regions have a descriptive label, and both scrollable excerpts have keyboard focus targets and accessible names.

## Integration

Changes remain in `assets/documents.js` and `assets/document-management.js`; no new module, stylesheet, dependency or business store was added. The document-management API exposes `rememberFilters`, called immediately after changing filter state and before the canonical render. The previous late event/action wrappers were removed.

The canonical changes in `documents.js` are limited to folder/archive actions, new-folder completion, the search-input handler and the format/state/sort change handler. All permission checks, source reads, contextual assessment decisions, exact file versions and stored history remain authoritative.

Assessment response/review interactions were inspected and their existing suite rerun; this increment does not introduce another assessment layer.

## Verification

`tests/document_navigation_v215_checks.js` contains seven actual-source checks covering repeated filter changes, history metadata, clean-route defaults, title/preview return parity, late and long-line changes, bounded rendering, escaped content, source immutability, disclosed reader limits and concurrent version changes. The DOM/history shell is substituted; native browser history, keyboard behavior and rendered layout remain acceptance tasks under the existing browser restriction.
