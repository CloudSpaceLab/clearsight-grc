# Forms status and tag filters — 22 November 2026

Reference: `clearsight-v2.15/forms.html`, SHA-256 `dc38d0f563e6d45be2ae9dc56496e4b177d990751e6721578e476d82747dfc19`. The existing Forms register retains its navigation and card/table composition; a compact name/tag search and status scope row use the existing library API.

The API evaluates name, tag and status in the verified tenant/entity, and returns status facets over the same name/tag population with the selected status removed. Changing status preserves the other filters and restarts pagination. Counts are shown only from a valid facet map; missing, malformed or unrecognized facets leave counts unknown. Pending queries discard earlier counts and a repeated continuation cursor is rejected before results are appended. A filtered empty result offers Clear filters.

Synthetic selection, empty and missing-count states were captured at 1440, 768 and 390 pixels. The layout uses the shared field, button and navigation tokens. Captures demonstrate responsive interaction and no page overflow, not production authorization, database reconciliation or full-screen acceptance. The migrated status-scope suite retains authoritative counts and preservation of the tag when changing status; the old nested filter representation is replaced by the existing direct tag/status query contract.
