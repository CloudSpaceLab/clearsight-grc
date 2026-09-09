# Hosted vendor workflow corrections

## Decision brief

The hosted verification of release `fa2f9ea2` uploaded a three-row synthetic XLSX successfully, then exposed a blank workspace when opening its form proposal. The API retains empty Go slices as JSON null. The proposal view called `filter` on null sections before rendering its state. Treat those empty lists as empty and use the existing respondent contract normalization for previews, so unsectioned questions remain visible. Preserve source fields, selection, proposal versions and approval boundaries.

The same check found the original approved eight-question vendor starter still at version 3. Its recorded maker and checker match the shipped template, but its historical section help and omitted field defaults differ from the current normalized comparison. Reconcile only recognized shipped contracts through the guarded owner/reviewer revision path. Customized wording, collection behavior and cache restrictions must remain untouched.

## Required states

- Unsectioned XLSX proposal: three source rows, three visible answer controls, selected fields can create a draft.
- Generating and failed proposals with null fields, changes, sections and unresolved lists: loading/recovery remains visible; no draft action is offered.
- Existing 47-row proposal: bounded review and preview preserve every source field.
- Legacy starter: original wording and absent default metadata qualify only with the known maker/checker history; customized contracts do not.

## Verification

The null-list regression reproduced three failures before the fix. All 22 focused proposal, capture, API and copy tests passed after normalization, and TypeScript passed. [Twenty rendered states](../evidence/2026-09-09-proposal-null-recovery/manifest.json) passed with no page errors or horizontal overflow, including four unsectioned previews across desktop/mobile and light/dark themes. The unresolved count is hidden when empty.

The guarded matcher passed against the read-only hosted version 3 response and the preserved August 27 historical fixture. Unit regressions reject changes to wording, cache policy, collection intent, options and requiredness. Seven PostgreSQL upgrade scenarios passed, including historical serialization, preserved issued requests, customized forms, restart/idempotency and authority denial. The actual source was not changed through the read-only diagnosis; deployment runs the ordinary guarded revision commands.

The uploaded workbook is synthetic release-verification data. No private workbook or vendor invitation was sent during this check.
