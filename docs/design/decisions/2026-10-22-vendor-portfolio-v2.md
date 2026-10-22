# Vendor portfolio — v2 layout and collection navigation

## Decision and scope

Continue the user's approved whole-page and menu adaptation after the selected relationship release in #582 and the Reports scope correction in #583. The baseline is main `cf2710b60a02354b9258cb73014249b696b8f94d`, tree `7e7d8cfe05677958562daaec3f80284a34324aad`.

Adapt the five portfolio views together: **Overview, Register, Onboarding, Assessments and Documents**. Replace the earlier two-button navigation, exception-first overview and tall register cards with the original page hierarchy, bounded primary/context composition and responsive record views. Keep the selected relationship workspace and its existing command owners.

This remains **partial v2 layout integration**. A named visual and interaction review is required for the new composition. The formal whole-screen register remains **0/19** until separately retained actor, authorized-result, source-to-report and operational acceptance is complete. This tranche does not close #462 or alter the independent Work search PR #574.

## Immutable source authority

The page authority is `docs/design/references/clearsight-2026-10/source/clearsight-vendor-management-designs/vendor-workspace-preview.html`, SHA-256 `c37f35161878d2035335241fc104c41fb4c8f6176cfe81d8a2f0d7fbdba728fd`. Its accompanying design notes define portfolio, relationship and add-vendor behavior. Original HTML, CSS, JavaScript, notes and manifest remain unchanged.

The shared primary menu follows the [21 October navigation decision](2026-10-21-v2-navigation-adaptation.md). Source sample counts, risk classifications, approval conclusions, deadlines, document completeness and Copilot suggestions are illustrative and cannot become runtime facts.

## Layout and owner map

| Source region | Runtime adaptation and existing owner |
| --- | --- |
| Page heading and actions | Vendors heading, current organization/legal entity, short service-work context, existing Add vendor action. No decorative period control or unsupported export. |
| Five local destinations | Shared accessible Tabs and URL-addressable portfolio views in the source order. Existing relationship links remain valid. Narrow layouts expose the same selected destination through the shared responsive navigation contract. |
| Four stable measures | Loaded vendor services, linked open exceptions, open actions and overdue actions. Existing `loadVendorRiskWork` and `summarizeVendorRiskWork` retain deduplication, bounded reads and unknown totals when incomplete. These are not assessed vendor-risk or review-due counts. |
| Two insight panels | Service criticality from loaded relationship records, and stages of their readable linked exceptions. Titles and denominators state those actual populations. Exception stages do not represent due-diligence progress. |
| Inline register and full Register | One reusable presentation of current server-search results, loaded-population filters, relationship state and form-work facts. Preserve cursor loading, selection limits, form requests and exact relationship navigation. |
| Attention rail | A bounded set of actual linked exceptions, with existing Matter navigation. No invented DPO or expiring-document total. |
| Upcoming context | Actual recorded service renewal dates, explicitly named as renewals. Renewal dates do not stand in for review deadlines. |
| Onboarding | Loaded proposed and under-review relationships with exact entry into existing due diligence. Relationship state cannot establish assessment completion or approval to activate. |
| Assessments | Loaded relationship form-summary facts and exact entry into existing form work. Submitted, awaiting review, assessed and assessed concern remain distinct. |
| Documents | Loaded relationship entries opening the existing protected document browser. No document-count, expiry or sufficiency aggregate is invented. |
| Exception queue | Existing scoped findings/actions remain reachable from the overview, with their filters, source attribution and exact record actions. It no longer displaces the source's inline relationship register. |

`VendorsWorkspace` remains the relationship list, selection, form-summary, draft and command owner. `VendorPortfolio` also serves the selected relationship's Issues & actions; its detail mode must retain that behavior. Presentation extraction must not introduce another vendor, assessment, document or task store.

## Data, permissions and recovery

The vendor API performs scoped search before cursor pagination. Client facets explicitly cover loaded records. A next cursor prevents a page count from being presented as an enterprise total. Form summaries remain batched in groups of at most 50; a missing summary is unknown, including when another batch succeeds. No new per-relationship assessment/document fan-out is required for portfolio navigation.

The existing bounded exception reader checks at most 50 relationships and 50 exact linked Matters, with limited concurrency. Shared Matters/actions remain deduplicated. Failed, truncated or mismatched reads cannot produce favorable zero totals. Available records and retry actions remain usable during partial failure.

`highest_concern` describes assessed form concern, `criticality` describes service importance, and `renewal_at` describes service renewal. None establishes relationship risk, current evidence sufficiency, next review or approval. Current request identity, scope, server capability checks, optimistic versions and activation policy remain unchanged.

Create/edit flows retain their navigation lock, unsaved inputs, exact selected record and failure recovery. Changing portfolio tabs is navigation only. Record drills must preserve the intended Forms, Documents or Due diligence destination, and returning to the portfolio must restore a useful focus target. Existing bulk form requests retain explicit selection and confirmation; opening a view never sends a request.

Review must explicitly cover a newer server version arriving while an older edit draft is preserved. A read retry cannot silently replace that draft's expected version or turn an edit into a create. Likewise, an older cursor response cannot append to a newer search, and returning from an exception must preserve the queue's expansion and filters as well as its scroll position. A sample-exception marker must remain visible in Overview when its detail queue is collapsed.

## Responsive and interaction requirements

Compare the original and runtime at 1440, 1280, 1024, 768 and 390 CSS pixels, in both runtime themes. The source uses four measures and a primary/context arrangement on wide screens, two-by-two measures and a following context area at 1120px or below, then stacked insights and record cards on phones. The implementation may normalize source spacing to existing tokens and enforce 44px targets, but cannot retain the previous generic layout for component convenience.

Verify heading/action proportions, local menu order and selection, two insight panels, inline first useful relationship content, bounded context, register density, narrow replacement and absence of horizontal overflow. Inspect the selected relationship after every new collection entry path. Reuse shared controls, focus, overlays and tokens; do not add a second mobile menu or unbacked global filter.

The source's horizontally scrolling local tab strip is adapted to the existing labelled **Vendor view** selector at 760px or below. It retains all five destinations, their order and current selection, following the already reviewed selected-relationship navigation contract. This is an explicit accessibility adaptation, not a claim that the two narrow-screen menus are visually identical. The wide shared underline tabs retain the source menu anatomy.

## Verification plan and status

Preserve before-state Overview/Register captures at 1440/768/390 in both themes from the exact baseline. The old workspace has no Onboarding, Assessments or Documents portfolio pages; record those as absent rather than manufacturing before pairs. Capture all five implemented views against the same source view at all five approved widths.

Required checks cover route round trips, local tab keyboard/selected state, search and cursor continuity, loaded filters, bulk selection, exact collection drills, edit locking, empty/search-empty states, delayed/failed/partial reads, unavailable summaries, long labels, record return focus, touch targets and the customer-build fixture boundary. Existing relationship, activation, first-run and Forms-document launchers remain regression gates.

**Runtime implemented; local layout and interaction review complete.** Corrected local checkpoint `85817bf9f00d99cd168ab2b3eb312f85b07f7ba1` and published runtime checkpoint `766d2bb3890c85906efc9661bab98824503faf09` have the identical tree `4b722a081326b5cbb2504a9564fab3956621c4c6`. Publication through the connected GitHub API changes commit metadata, not the reviewed files.

The earlier full web run passed 2,111 tests. The corrected checkpoint passes 137 focused workflow/routing/copy tests, TypeScript, the customer build, runtime-fixture separation and UI-contract checks. A focused browser return now restores the expanded exception queue at scroll Y 884, with its selected action visible; opened phone queue controls meet 44×44px. Restoration waits for both linked issues and register summaries to settle, including partial reads, and consumes the saved position only after its animation frame runs. The earlier `99c962c` checkpoint is rejected for premature restoration and is not final visual evidence.

The before-state contains 24 PNGs and its exact baseline manifest under `docs/evidence/2026-10-22-vendor-portfolio-v2/before/`. The [retained evidence receipt](2026-10-22-vendor-portfolio-evidence.json) records 63 passing browser scenarios, 25 original-source frames, 50 source/runtime pairs, 20 accessibility scenarios with zero axe violations, two full-page Overview renders, and the named AI reviews. Compatibility checks pass eight selected-relationship states, two existing measure states, and 12 legacy portfolio captures with four detail/return checks. The associated pull request records final-head CI, artifact verification, merge and deployment results. These local checks establish layout and isolated interaction behavior; operational screen acceptance remains separate.

The review retained one nonblocking recovery-copy limitation: a denied list response currently shares the unavailable/Try again message. Its state still contains no records, disables Add vendor and issues no command. The receipt does not claim that generic copy proves authorization behavior. Source prototypes, test fixtures and browser geometry likewise do not establish live bank approval, document sufficiency or full four-actor acceptance.

## Hosted review follow-up

PR #584 passed PR/main CI and UI checks and deployed as `cb07cc747c2bee7d19413ed398bb1c111f51c99f`. The hosted read-only review confirmed all five layouts and initial record drills, then found that actual browser Back could retain selected relationship detail at a collection URL. The earlier collection evidence used the explicit in-app register-return control, which clears selection through a different path. The [collection return correction](2026-10-22-vendor-collection-return-fix.md) records that evidence gap, the route/selection contract and the required failing/passing regression. The original receipts remain historical and must not be read as proof of the previously untested browser-history path.
