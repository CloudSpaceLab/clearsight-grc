# Shared tabs: browser-history correction

Date: 23 October 2026

Status: **locally implemented and reviewed; publication and deployment pending**.

Decision: [selected tabs after browser history](../../design/decisions/2026-10-23-tabs-history-visibility.md).

## Scope and revision

The deployed Risk & Losses release `02dd0b4f066ba58ec17cf0e2588437efe5e856fd` exposed a phone history defect: browser Back selected Risks and restored its collection, but left its active label clipped. The shared Tabs collection now declares `selectedKey` as a cache dependency. Existing routing, node identity, focus and horizontal-only reveal remain unchanged.

The candidate runtime is local commit `ee7d0ffaf23082cb8cbc9a4d3fd1a6ab617ee177`, tree `6bd5baf4095e7eb856fdce41945da8551a6f6bb4`. Its isolated 171-file evidence build has aggregate SHA-256 `707a0cc6b92bf33a9acf41a11d665916c76ba02dbad4f89eff1dcbf445f8f688`. The preserved baseline build remains `a20ab1aadace0972b9f35bb924c1d4361f805adca7af4a0b7131c84e1badc746`.

Original page authority remains the unchanged [operational-risk HTML](../../design/references/clearsight-2026-10/source/clearsight-operational-risk-designs/operational-risk-preview.html), SHA-256 `cf0a101d574904fc5f496cab0625889e640e2e30734d180d86b89171419e69dd`. The wider [Risk & Losses evidence](../2026-10-22-risk-loss-v2/README.md) retains the original/runtime layout comparisons and explicit adaptations. This correction adds history proof; it does not repeat or extend that whole-page visual claim.

## Before and after

| Evidence | Result |
| --- | --- |
| [Preserved baseline receipt](before/manifest.json) | New peer-history regression fails against unchanged `02dd0b4f` when browser Back reaches Risks on phone |
| [Baseline failure image](before/250-risk-loss-risks-dark-390x844-failure.png) | Selected tab x=37–81 is clipped by menu x=64–382 |
| [Candidate receipt](after/manifest.json) | Both profiles pass; 12 actual Back/Forward transitions retain selected-tab containment, document and menu identity, with zero commands |
| [Desktop candidate](after/250-risk-loss-risks-light-1440x900.png) | Light, 1440×900; final restored Risks state |
| [Phone candidate](after/250-risk-loss-risks-dark-390x844.png) | Dark, 390×844; final restored Risks state; browser-Back receipt places Risks at x=64–108 inside the menu |

Candidate PNGs show the final restored state after the complete navigation check. Six numeric geometry entries in each profile establish the intermediate three Back and three Forward results. The test does not click or focus destinations while traversing history. It checks the original tablist element handle and document token, selected label, route, menu/tab rectangles and document width.

## Named reviews and validation

`/root/risk_parity_audit` reviewed the complete six-file runtime/regression/decision commit at the exact revision above and found no material issue. The new stable-item unit regression fails before the runtime change (scrollLeft remains 0 instead of 130) and all **10 Tabs tests pass** afterward, including contextual reveal/cancellation, retained panels and keyboard selection.

`/root/progress_review` verified the receipt gate's negative regressions. All three new groups fail against the old contract; all **41 scoped contract checks pass** after the gate change. Missing, shortened, extra or reordered history, wrong routes/labels, clipped or inconsistent bounds, remounted menus, changed documents and overflow are rejected. A visibility boolean cannot override the numeric bounds.

`/root/evidence_review` directly inspected the baseline failure and both candidate PNGs, and independently checked image bytes/hashes and all 12 history entries. **PASS for the local visibility/navigation correction**, with no new visible clipping or overlap. TypeScript checking and the isolated evidence build also passed.

Verified candidate image SHA-256 values:

- Desktop: `43655531528a49a70ccba1ce17cab44c247d6553b2156010ea50045bf8234af8`.
- Phone: `ab153695f14b07e12b52bd880d90d28f91ab5b69b1088d79469d93994a4d8cdc`.
- Candidate manifest: `664014123cd7661cea55bf8020d8b66aaa48e5276e92195c756247a7ade56334`.

## Release and evidence limits

The bulk GitHub publication calls were interrupted. A later individual-file publication attempt did not return within its 55-second bound; subsequent reads before 18:54 UTC on 23 October still showed `agent/462-tabs-history-followup-20261023` at `02dd0b4f`, and the expected runtime blob was absent. The follow-up runtime therefore remains **local**. No automatic approval rejection reason was returned. The successful CI/UI/deployment runs for #586 establish the preceding release only. This follow-up still requires publication, normal CI and full UI/source gates, merge, demo deployment and repeated hosted checks.

All retained images here are synthetic UI evidence, labelled `SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION`. RCSA and indicator peer reads use fixture-unavailable responses; the history test proves navigation continuity and tab visibility, not populated workflows. Raw hosted record screenshots and identifiers are excluded from this public evidence directory.

Formal whole-screen operational acceptance remains **0/19**. This review does not establish full original-HTML parity, live authorization, source-to-report reconciliation, four-actor execution, public-client TLS trust or production acceptance.
