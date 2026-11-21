# Scoped Documents

Vendor and submitted-response records open the existing Documents workspace using the relationship or response revision filter supported by the document API. Pagination, file-type and view changes retain the scope. A return link is restricted to the corresponding record; a response distribution and vendor assessment are retained when available.

Malformed or conflicting scope parameters do not issue a broad document query. A page containing an item outside its requested scope is rejected and loaded details are cleared. Server authorization remains authoritative; a record identifier in a link does not grant access. Generic failures remain retryable; access denial clears loaded records.

Responsive synthetic evidence and limitations are in `docs/evidence/2026-11-22-scoped-documents/`. Existing UI tests cover scope changes, denied reads and the filter on subsequent pages. They do not substitute for deployed end-to-end authorization checks.
