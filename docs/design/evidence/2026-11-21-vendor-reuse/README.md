# Vendor intake: existing vendor selection

**Historical render:** reuse was subsequently hidden while transactional authority revocation synchronization is unresolved. These screenshots do not describe the currently exposed UI.

Reviewed by Codex on 21 November 2026. The five sample renders show the existing-vendor search result at 1440, 1280, 1024, 768 and 390 CSS pixels. The sample transport is isolated and the page is labelled “Sample data.” The document width equalled the viewport width at all five widths; the search and selection buttons remained visible. A native default button was replaced with the intake styling before these captures.

This evidence covers the search-result state only. Component tests cover verified tenant/entity/actor receipts, creation-route preflight, stale results, definite rejection, uncertain-result recovery and selected-identity reuse. They do not establish live browser-to-backend authority, touch/keyboard acceptance, duplicate prevention after a lost response, or a complete vendor-onboarding journey. The current create contract does not accept an expected source version or idempotency key; if the reused identity changes between selection and creation, the UI cannot prevent the write and instead treats a mismatched receipt as uncertain.
