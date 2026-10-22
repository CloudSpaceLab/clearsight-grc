# Vendor relationship — October v2 layout adaptation

## Decision and boundary

Replace the complete selected vendor relationship page: its heading and actions, five local destinations, primary content and relationship rail, and responsive replacement. The selected record no longer sits inside the old register inspector or below the portfolio heading/menu. The portfolio/register itself remains a separate future tranche. This is partial v2 layout integration; whole-screen acceptance remains **0/19**, with production cutover blocked.

Authority is the immutable source `docs/design/references/clearsight-2026-10/source/clearsight-vendor-management-designs/vendor-workspace-preview.html`, SHA-256 `c37f35161878d2035335241fc104c41fb4c8f6176cfe81d8a2f0d7fbdba728fd`. The shared primary menu follows the 21 October navigation decision. Source HTML/CSS/JS and its data remain unchanged.

## Anatomy and owner map

| Source region | Runtime adaptation |
| --- | --- |
| Back, record identity and action block | Record `h1`, service subtitle, recorded relationship status, existing separately versioned identity/relationship editors and a navigation-only review action. Parent portfolio heading/menu are absent in selected browse mode. |
| Overview | Four literal status facts: relationship, due diligence, submitted forms and awaiting bank review. Existing `VendorComplianceOverview` supplies Review requirements and its exact response-review actions. Failed requirements stay in the primary column. |
| Due diligence | Existing `VendorDueDiligence` plus independently loaded `VendorActivationPanel`; every command and expected-version check stays with its owner. |
| Evidence | Retained Forms/Documents subviews using `VendorFormsPanel` and protected `DocumentBrowser`. Existing linked-work disclosure remains with Forms, preserving guide focus and drafts. |
| Issues & actions | Existing relationship-scoped `VendorPortfolio detail`, explicitly presenting vendor exceptions and their canonical Matter/Action drill. It loads on first visit. |
| Activity | Existing `VendorResponseHistory`, explicitly described as submitted form revisions. No full relationship audit stream is fabricated. |
| Relationship / Next action rail | Resolved accountable owner, service criticality, privacy role, renewal, jurisdiction and real record metadata. Navigation leads to the existing review owner; activation is never inferred from assessment completion. |

The existing selected record, assessment, summaries and command callbacks remain in `VendorsWorkspace`. The new `VendorRelationshipWorkspace` is a presentation composition, not a second reader/store. Internal Forms/Documents/History section keys retain their existing callback contracts. Visited panels preserve local work within a service; changing service resets them. Back returns focus to the register.

## Responsive and interaction contract

At 1440/1280/1024 the page uses a flexible primary column and a 316px relationship rail. At widths up to 840 the rail follows primary content and header actions move below identity. At widths up to 600 the identity is 50px and status facts become two columns. At widths up to 760 the shared labelled section selector replaces the source's horizontally scrolling five-item strip; every destination and its selected state remain available. Evidence keeps its two short peer tabs. Shared control targets stay at least 44px on touch.

The two real identity edit paths need more header space than the prototype's single Edit control. They wrap without truncating the selected vendor name. The header review action remains secondary navigation so the existing active workflow owns the dominant executable action. Shared theme tokens replace prototype colors. Source four-step geometry displays literal recorded facts rather than invented progress or gate completion. Renewal is not a review deadline, criticality is not risk, and form concern is not a relationship risk rating.

This supersedes only the older selected-detail composition in `DESIGN.md` and the September compliance brief: edits move from collapsed metadata to the identity header, exceptions move to their own source destination, and the portfolio heading/menu disappear on selection. Existing authority, evidence, freshness, review and failure semantics remain in force.

## Evidence and review

Baseline is exact Notifications head `8495506ef034718c897611991abb9e2a48f8bb87`. Preserve 1440/768/390 light/dark before captures. Compare all five source sections at 1440/1280/1024/768/390 with corresponding light/dark runtime renders. Assert identity, menu order/selection, removed portfolio chrome, active panel identity, 316px rail/stacking, first useful content, no horizontal overflow and 44px touch targets. Test loading, failed form/assessment reads, denied activation, long names, protected documents, retained Forms/Documents, keyboard navigation and register return focus.

Runtime checkpoint is `44c26f6ed8289fad97ffbd6fc01afca1f30f46d8`, tree `08f9d72cd1d94a00daab4108aada415b4762a91b`. The retained [before captures](../../evidence/2026-10-22-vendor-relationship-v2/before/manifest.json) bind all three light/dark viewport pairs and their full-page companions to the baseline. The [final local evidence receipt](2026-10-22-vendor-relationship-evidence.json) binds runtime, harness, built assets, source images, local test logs and named reviews separately; evidence-only follow-up commits do not imply a new runtime revision. The PR owns subsequent exact-head CI results.

| Local verification | Result |
| --- | --- |
| Main matrix and original-source pairing | 50 runtime frames, 25 original-source frames and all 50 same-screen pairs pass |
| Supplemental states | All eight pass: loading, unavailable forms, long labels, protected Documents, denied activation, unavailable assessment, empty and partial forms |
| Accessibility | 20/20 PASS; zero reported violations, horizontal overflows, browser errors or undersized controls |
| Web suite | 262 files / 2,100 tests passed before the last two return-focus tests and bounded markup corrections; the final affected workflow/copy suites pass all 99 tests |
| Typecheck, customer build, runtime fixture boundary | PASS |
| Expanded UI contracts | 51/51 PASS, including seven preparation/selection rejection mutations |
| Existing browser workflows | 18 activation, six due-diligence and 21 first-run/brand captures pass |
| Customer bundle budgets | PASS: initial JavaScript 161,867 bytes gzip, initial CSS 31,970 bytes gzip, largest JavaScript chunk 564,731 bytes raw |

The 50-frame main capture completed before a focused loading case exposed a preparation-phase observer error: a legitimate portfolio read was being checked as a selected-record read. The eight supplemental cases were rerun against the same frozen build with an explicit selection event/timestamp. The receipt retains both actual runs and the earlier error; it does not claim a single uninterrupted local run. Selected reads still require the exact relationship and response owner, and every non-read API command remains observed across both phases. The historical partial-delivery fixture alone pins its clock to 8 September so its September invitation deadline can be exercised; production date validation is unchanged.

Three named AI reviewers inspected all 50 main runtime images and 25 original-source images directly: ChatGPT `/root` reviewed Overview/Due diligence; `/root/current_pr_review` reviewed Evidence/Activity; `/root/vendor_contract_audit` reviewed Issues & actions. Their scoped verdict is **PASS with documented adaptations**, with no P1/P2 rendering blocker. This is an anatomy and interaction review, not pixel identity with the prototype. The Issues destination retains real counts and exception filters: the first exception is only partly within the initial 1440px viewport and requires scrolling at the narrower widths, while the summary and filters remain visible. The existing 768px global actor chip truncation is recorded separately from this relationship composition.

Review also corrected four actual regressions: navigation to Due diligence now scrolls and focuses the real workflow; Back restores focus after the route-driven register refresh; the compliance count description uses valid definition-list markup; and Activity avoids repeating its section introduction so the first submitted revision is visible at 390px. Cancelled assessment guidance now names starting a new review.

Synthetic fixtures prove composition and frontend interactions; they do not prove live authorization, four-actor execution, delivery or whole-screen production acceptance. Local Chromium 153 is explicitly distinguished from CI's pinned Chromium 140. Automated accessibility PASS means no blocking result in the checks run: 26 incomplete rule instances cover 308 nodes (20 color-contrast rules and six aria-prohibited-attr rules). Supplemental contrast readings are 1,090 PASS, 404 INCOMPLETE and 68 REVIEW; these remain disclosed rather than converted into universal conformance. Evidence-build bundle metrics are informational and do not use the customer entry's budget.

## Initial CI and Forms launcher compatibility correction

Published head `db6cd0620706a76137d9567fe4d40fc8084254aa` passed the full [CI run 39745478826](https://github.com/CloudSpaceLab/clearsight-grc/actions/runs/39745478826), including all backend gates and 262 web test files / 2,102 tests. Its [UI review run 39745479008](https://github.com/CloudSpaceLab/clearsight-grc/actions/runs/39745479008) passed all 58 new relationship cases, then failed at `127-forms-documents-vendors-light-1440`: the older Forms capture helper still searched for the former Vendor section menu and a direct Documents destination. Approved-source capture did not run, so this attempt is not a complete UI pass.

The bounded correction reuses the relationship navigation helper and selects **Evidence → Documents** through the scoped Evidence tablist. Due diligence remains its top-level destination, and the helper asserts that navigation preserves the selected relationship route. Production runtime files and the frozen evidence build are unchanged. The [affected-launcher recheck](../../evidence/2026-10-22-vendor-relationship-v2/reviews/forms-vendor-launcher-recheck.json) retains all 18 passing cases (16 document launchers and both Due diligence callers), 52 successful relationship-scoped document reads, and the original preview, focus and blocked-download assertions. All 15 Forms evidence nodechecks pass. The recheck receipt SHA-256 is `b7ed2aa2bafa33f1eb26595da49c4c6640f5168364102f92f0491b2e0be36411`; the corrected helper SHA-256 is `0cadf978af0b5cba731dec67ed563e4b9a7e930833f73a6b5d25e8a4c77ee1f7`. PR #582 records the subsequent exact-head CI and visual closeout separately.
