# Vendor relationship v2 evidence

The selected vendor relationship page is the scope of this tranche. The source is the immutable October `vendor-workspace-preview.html`; its source hash, runtime ownership and responsive adaptations are recorded in the [decision](../../design/decisions/2026-10-22-vendor-relationship-v2.md).

`before/` preserves six viewport captures plus six full-page companions from baseline `8495506ef034718c897611991abb9e2a48f8bb87`: 1440, 768 and 390px in light/dark themes. Its manifest records hashes, dimensions and the baseline identity. These are synthetic fixture renders, not production bank records.

The review harness captures five local destinations at 1440, 1280, 1024, 768 and 390px in both themes, plus eight loading, unavailable, long-label, document, denied, empty and partial states. It compares the 50 main frames with 25 original-source frames and measures menu selection, panel identity, rail placement, first useful content, touch targets, horizontal overflow, scoped readers and focus. Non-read commands are observed throughout preparation and selection; selected-record reads are measured from the explicit selection boundary.

CI retains after-state PNGs and their manifest in `web/ui-evidence/`, source PNGs and pair measurements in `web/ui-evidence/approved-reference/`, accessibility results and the flow review report in the UI/UX workflow artifact. The [machine-readable receipt](../../design/decisions/2026-10-22-vendor-relationship-evidence.json) binds the locally inspected images and the four review receipts in `reviews/`; the PR records exact final-head CI run and artifact identities. Browser versions and evidence-build/runtime-build boundaries are explicit so hashes are not mistaken for identical pixels across environments.

Whole-screen acceptance remains **0/19**. These checks do not establish live authorization, production delivery, four-actor execution or deployment acceptance.
