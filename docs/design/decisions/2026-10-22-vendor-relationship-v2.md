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

## Evidence plan

Baseline is exact Notifications head `8495506ef034718c897611991abb9e2a48f8bb87`. Preserve 1440/768/390 light/dark before captures. Compare all five source sections at 1440/1280/1024/768/390 with corresponding light/dark runtime renders. Assert identity, menu order/selection, removed portfolio chrome, active panel identity, 316px rail/stacking, first useful content, no horizontal overflow and 44px touch targets. Test loading, failed form/assessment reads, denied activation, long names, protected documents, retained Forms/Documents, keyboard navigation and register return focus.

Exact captured hashes, verification results and the named visual verdict will be recorded after implementation. Synthetic fixtures prove composition and frontend interactions; they do not prove live authorization, four-actor execution, delivery or whole-screen production acceptance.
