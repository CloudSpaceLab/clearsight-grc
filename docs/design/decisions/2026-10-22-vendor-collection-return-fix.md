# Vendor collection return — hosted review correction

## Observed release behavior

PR #584 merged as `cb07cc747c2bee7d19413ed398bb1c111f51c99f`, tree `968250b91c7d98c3a7fd0c38715f46853aea40e4`. Both PR and main CI/UI checks passed, and demo deployment `39789562460` verified that exact revision. The hosted readiness response reported PostgreSQL mode and the same ready revision.

The subsequent read-only review by `/root` confirmed all five portfolio layouts and their menu: Overview, Register, Onboarding, Assessments and Documents. Register search and the existing Due diligence, Forms and Documents entry paths worked. The document browser correctly displayed its empty result; the review issued no business commands.

That review found a release defect in all three new collection drill/return paths. Browser Back restored the collection URL but, after the relationship reads settled, the selected relationship remained on screen and the portfolio menu was absent. A full refresh at the same collection URL restored the correct collection. The defect is in the mounted workspace's selection lifecycle, not the route parser or the page geometry.

## Cause and correction contract

`App` responds to browser history by removing `target.vendorRelationshipID` while retaining the mounted `VendorsWorkspace`. The target-change effect refreshes the workspace without a requested relationship ID. The previous `exact ?? preserved` fallback then reselects the old aggregate despite the collection URL.

An explicit route transition to any portfolio root must clear relationship selection. Ordinary data refreshes and read retries may preserve the selection when the current route or an active edit requires it. The correction must retain draft identity, original expected version, navigation locks and protection against obsolete asynchronous reads. It must not convert an edit into a create or submit any workflow command.

The regression covers target removal for all five collection roots, browser Back and Forward in the same mounted application, settled collection content and direct collection refresh. Existing explicit register-return controls remain a separate interaction contract.

## Evidence gap and required browser assertions

The original collection evidence clicked **Back to vendor register** and then selected the originating collection. That explicit control clears selection, so it did not exercise browser history. The stored `returned: true` value established that narrower interaction. The separate collection-refresh check created a fresh page, and Matter-to-Vendors browser Back remounted the workspace; neither exposed the retained-instance defect.

The corrected evidence must use actual browser history and wait for the resulting data reads to settle. It must verify the collection URL, visible portfolio navigation and selected collection, expected collection heading and record population, and absence of the selected-relationship panel. Checking the URL alone is insufficient. The previous in-app return path remains independently named so it cannot substitute for browser Back.

Retain a failing reproduction against the immutable deployed runtime and passing proof against the corrected runtime. Keep the original PR #584 artifacts as historical evidence. The updated offline evidence contract must reject missing or contradictory browser-return proof.

## Design authority and acceptance boundary

The authority remains `docs/design/references/clearsight-2026-10/source/clearsight-vendor-management-designs/vendor-workspace-preview.html`, SHA-256 `c37f35161878d2035335241fc104c41fb4c8f6176cfe81d8a2f0d7fbdba728fd`. This correction changes selection synchronization and its evidence; it does not change the approved five-view layout, menu, source files or data/command owners. The [portfolio decision](2026-10-22-vendor-portfolio-v2.md) and its source/runtime comparisons remain the layout basis.

This remains partial v2 layout integration under #462. The correction does not establish four-actor approval, document sufficiency, authorized report download or whole-screen operational acceptance. The formal register remains 0/19.

## Verification record

The runtime correction limits selection preservation without an exact relationship target to an active guide. The existing edit guards retain draft identity and the original optimistic version. No route, store, schema, command or layout geometry changes.

- Focused workspace, routing and copy tests: **134/134 passing**, including **98 workspace tests**. Seven new regressions cover target removal and Forward for all five collection roots, a late exact-record response and failed-return retry. The five target-removal cases failed against the prior runtime before the correction.
- Browser negative control: **3/3 deployed-build failures reproduced**, with the correct collection hash, no pending reads and the stale detail still visible. All 171 retained build-file hashes matched the reviewed PR #584 runtime before and after capture.
- Corrected browser proof: **3/3 passing** for Onboarding, Assessments and Documents. Actual Back in the same mounted App precedes any reload or corrective navigation. The proof independently checks Forward identity, direct collection hashes, cold collection reloads and the explicit Register-return control.
- UI evidence contracts: **all three nodecheck files pass**. The contract rejects generic `returned: true` proof, retained detail at the right URL, pending reads, document replacement, wrong return actions, changed record populations, incorrect Forward identity and invalid direct-root/reload states.
- Final local release checks: `npm run typecheck`, `npm run check:runtime-truth` (1/1) and `npm run build` all pass. The customer build leaves the 171-file evidence build and its aggregate digest unchanged. `git diff --check` passes.
- Independent named AI reviewer `/root/remaining_pages_audit`: **PASS**, with no blocker after reviewing every refresh caller, App history/target wiring, guides, edit/version guards, request invalidation, the three negative controls and the three corrected captures. The reviewer independently matched six runtime files, five evidence scripts, the source and all 171 corrected build-file hashes.

The [unchanged capture receipt](2026-10-22-vendor-collection-return-evidence/receipt.json), before/after manifests, six fixture-only screenshots and UI-contract log are retained together. Receipt SHA-256: `937be4793ebbbcb742890beefe31e1a4eb4b2e28c4bb9aacd2f35eeab854edc7`; corrected manifest: `cb9972cb23f843932f009f1047c6c43eb36a9fe0f0429652373fd73aa6842ca3`; corrected evidence-build aggregate: `cdbec6625d8628821027347dadd444e3f46dfde2addc467fa007ef9aeca092d5`. The receipt defines the aggregate digest and binds the runtime workspace to `b91aa661cf67dcb36fca2d27c7a2706675cfcc810bbc4391ed258462b85bb9a5`. Regression-test SHA-256: `dbbf2144815ab2bb94753b513815ec975496f0f2b9ac3676273f4589187293be`.

These local captures exercise the actual App lifecycle with synthetic reads. They are three focused scenarios, not a local rerun of the full 63-frame portfolio matrix or proof of live backend authorization. The follow-up pull request records the final exact-head full CI/UI checks, expected-head merge, demo deployment and read-only hosted Back/Forward recheck.
