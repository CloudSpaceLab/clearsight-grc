import { vendorRelationshipEvidenceScenarios, vendorRelationshipViews } from "./vendor-relationship-evidence-scenarios.mjs";

const same = (a, b) => JSON.stringify(a) === JSON.stringify(b);
const rectangle = (value) => value && ["x", "y", "width", "height", "right", "bottom"].every((key) => Number.isFinite(value[key]))
  && value.width > 0 && value.height > 0 && Math.abs(value.x + value.width - value.right) <= 1 && Math.abs(value.y + value.height - value.bottom) <= 1;
const labels = vendorRelationshipViews.map((view) => view.label);
const relationshipID = "vendor-relationship-payments";

export function reviewVendorRelationshipCaptures(captures) {
  const byName = new Map(captures.map((capture) => [capture.name, capture]));
  const failures = [];
  for (const scenario of vendorRelationshipEvidenceScenarios) {
    const capture = byName.get(scenario.name);
    if (!capture) { failures.push(`${scenario.name}: Vendor relationship evidence is missing`); continue; }
    const layout = capture.vendorRelationshipLayout;
    const width = scenario.viewport.width;
    if (["route", "fixture", "state", "theme"].some((key) => capture[key] !== scenario[key])
      || !same(capture.viewport, scenario.viewport) || !layout || layout.kind !== scenario.vendorRelationshipView
      || layout.evidenceKind !== "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION"
      || capture.navigation?.activePage?.join("|") !== "Vendors") {
      failures.push(`${scenario.name}: Vendor relationship metadata, fixture or navigation owner is inconsistent`);
      if (!layout) continue;
    }
    const expectedLabel = vendorRelationshipViews.find((view) => view.id === scenario.vendorRelationshipView).label;
    const expectedTitle = scenario.vendorRelationshipCheck === "long-labels" ? "Sample · International Settlement and Merchant Assurance Services Limited" : "Acme Processing Limited";
    if (layout.headingText !== expectedTitle || !rectangle(layout.heading) || !rectangle(layout.title) || layout.titleContained !== true
      || layout.selectedRelationshipID !== relationshipID || layout.selectedRoute !== `#vendors/register/${relationshipID}`
      || layout.selectedPanelControlsMatch !== true || !layout.selectedPanelID || !same(layout.visiblePanelIDs, [layout.selectedPanelID])
      || !same(layout.localTabs, labels) || layout.selectedTab !== expectedLabel || !rectangle(layout.tablist)
      || !["tabs", "select"].includes(layout.menuMode) || layout.menuMode === "select" && layout.compactValue !== expectedLabel
      || !rectangle(layout.selectedTabBounds) || layout.selectedTabVisible !== true
      || layout.selectedTabBounds.x < layout.tablist.x - 1 || layout.selectedTabBounds.right > layout.tablist.right + 1
      || layout.heading.bottom > layout.tablist.y + 1 || layout.parentHeadingVisible !== false || layout.parentMenuVisible !== false || layout.registerVisible !== false
      || layout.documentWidth !== width || layout.scrollWidth > width + 1
      || scenario.vendorRelationshipMatrix && (!rectangle(layout.firstUsefulContent) || layout.firstUsefulContentVisible !== true
        || layout.firstUsefulContent.y < 0 || Math.min(layout.firstUsefulContent.bottom, scenario.viewport.height) - layout.firstUsefulContent.y < 44)
      || scenario.touch && (layout.minTargetWidth < 44 || layout.minTargetHeight < 44 || layout.undersizedTargets?.length !== 0)) {
      failures.push(`${scenario.name}: Vendor heading, complete local menu, replaced portfolio chrome or first useful content is missing`);
    }
    const { main, aside } = layout;
    const placement = width > 840 ? "beside" : "stacked";
    if (!rectangle(main) || !rectangle(aside) || layout.placement !== placement
      || placement === "beside" && (Math.abs(aside.width - 316) > 2 || Math.abs(main.y - aside.y) > 2 || main.right > aside.x + 1)
      || placement === "stacked" && (main.bottom > aside.y + 1 || Math.abs(main.x - aside.x) > 2 || Math.abs(main.width - aside.width) > 2)) {
      failures.push(`${scenario.name}: Vendor detail requires the measured 316px rail and 840px stacked replacement`);
    }
    const reads = layout.apiReads ?? [];
    const boundary = layout.selectionBoundary;
    if (scenario.vendorRelationshipCheck && (!boundary || boundary.event !== "RELATIONSHIP_SELECTION_POINTERDOWN" || boundary.relationshipID !== relationshipID || !Number.isFinite(boundary.at)
      || reads.some((read) => read.phase !== "SELECTED_RELATIONSHIP" || !(read.initiatedAt >= boundary.at))
      || !Array.isArray(layout.preparationReads) || layout.preparationReads.some((read) => read.phase !== "PREPARING" || !(read.initiatedAt <= boundary.at)))) {
      failures.push(`${scenario.name}: preparation and selected relationship reads require an observed selection boundary`);
    }
    const assessmentStatus = scenario.vendorRelationshipCheck === "loading" ? "PENDING" : scenario.vendorRelationshipCheck === "assessment-unavailable" ? 503 : 404;
    const foreignRead = reads.some((read) => {
      const record = read.path === "/api/v1/vendors/form-summaries" ? null : /^\/api\/v1\/vendors\/([^/]+)(?:\/.*)?$/.exec(read.path);
      const query = new URLSearchParams(read.query);
      return record && decodeURIComponent(record[1]) !== relationshipID
        || read.path === "/api/v1/authority/resolve" && (read.method !== "POST" || read.requestKind !== "AUTHORITY_READ_QUERY" || read.queryInput?.object_type !== "THIRD_PARTY_RELATIONSHIP" || read.queryInput?.object_id !== relationshipID)
        || read.path === "/api/v1/vendor-work" && query.get("relationship_id") !== relationshipID
        || read.path === "/api/v1/forms/documents" && (query.get("relationship_id") !== relationshipID || query.get("limit") !== "25" || query.get("current_only") !== "true")
        || read.path === "/api/v1/forms/responses" && (query.get("subject_type") !== "VENDOR_RELATIONSHIP" || query.get("subject_id") !== relationshipID);
    });
    if (!Array.isArray(layout.apiReads) || !Array.isArray(layout.commands) || layout.commands.length || foreignRead
      || !reads.some((read) => read.path === `/api/v1/vendors/${relationshipID}/assessments/current` && read.status === assessmentStatus)) {
      failures.push(`${scenario.name}: relationship viewing must retain scoped reads without sending commands`);
    }
    if (scenario.vendorRelationshipMatrix && (!layout.settledContent || layout.settledContent.pendingVisible !== 0
      || scenario.vendorRelationshipView === "evidence" && !/\d+ on this page/.test(layout.settledContent.requestsSummary ?? "")
      || scenario.vendorRelationshipView === "issues" && layout.settledContent.exceptionRows !== 5
      || scenario.vendorRelationshipView === "activity" && !/^\d+ submitted revisions loaded$/.test(layout.settledContent.responseRevisionSummary ?? ""))) failures.push(`${scenario.name}: visible selected panel lacks settled scoped content`);
    if (scenario.vendorRelationshipView === "activity" && !layout.activityScope?.includes("Response history")) failures.push(`${scenario.name}: Activity must disclose its submitted response scope`);
    const interaction = layout.interaction;
    const documentReads = reads.filter((read) => read.path === "/api/v1/forms/documents");
    const documentsBound = documentReads.length > 0 && documentReads.every((read) => {
      const query = new URLSearchParams(read.query);
      return read.method === "GET" && read.status === 200 && query.get("relationship_id") === relationshipID && query.get("limit") === "25" && query.get("current_only") === "true";
    });
    const historyReads = reads.filter((read) => read.path === "/api/v1/forms/responses");
    const historyBound = historyReads.length > 0 && historyReads.every((read) => {
      const query = new URLSearchParams(read.query);
      return read.method === "GET" && read.status === 200 && query.get("subject_type") === "VENDOR_RELATIONSHIP"
        && query.get("subject_id") === relationshipID && query.get("current_only") === "false"
        && query.get("sort") === "COMPLETED_DESC" && query.get("limit") === "25";
    });
    if (scenario.vendorRelationshipView === "activity" && !historyBound) failures.push(`${scenario.name}: Activity requires successful submitted response reads for the selected relationship`);
    if (scenario.verifyVendorRelationshipNavigation && (!same(interaction?.visited, labels) || interaction.relationshipPreserved !== true
      || interaction.evidenceDocumentsReachable !== true || interaction.evidenceFormsReachable !== true || interaction.responseHistoryReachable !== true
      || !documentsBound || !historyBound
      || interaction.backRestoredFocus !== true || interaction.baselineRestored !== true || interaction.keyboard?.complete !== true
      || width > 760 && !same(interaction.keyboard.sequence, [...labels.slice(1), labels[0]])
      || width <= 760 && (!same(interaction.keyboard.options, labels) || !same(interaction.keyboard.sequence, ["Due diligence"])
        || !(interaction.nextActionNavigation?.fromScrollY > 0) || interaction.nextActionNavigation.focused !== true
        || interaction.nextActionNavigation.visible !== true || interaction.nextActionNavigation.relationshipPreserved !== true))) failures.push(`${scenario.name}: relationship tabs, evidence subviews and register return lack interaction proof`);
    const check = scenario.vendorRelationshipCheck;
    if (check === "loading" && (interaction?.loading !== true || !(interaction.pendingReads > 0))) failures.push(`${scenario.name}: pending reads lack a loading state`);
    if (check === "unavailable" && (interaction?.unavailable !== true || interaction.retryIssued !== true || interaction.failureRetained !== true)) failures.push(`${scenario.name}: unavailable review requires visible retry and retained failure`);
    if (check === "long-labels" && (interaction?.longLabels !== true || !interaction.title?.includes("International Settlement"))) failures.push(`${scenario.name}: long record identity was not rendered`);
    if (check === "documents" && (interaction?.documentsReachable !== true || interaction.protectedDocumentOwner !== true || !documentsBound
      || !["rows", "empty"].includes(layout.settledContent?.documentState))) failures.push(`${scenario.name}: Evidence did not retain the protected document browser`);
    if (check === "activation-denied" && (interaction?.denied !== true || interaction.activationCommandAbsent !== true
      || interaction.stateCapture?.visible !== true || !layout.apiReads.some((read) => read.path.endsWith("/activation") && read.status === 403))) failures.push(`${scenario.name}: denied activation lacks a visible recovery state or exposed a command`);
    if (check === "assessment-unavailable" && (interaction?.assessmentUnavailable !== true || interaction.stateCapture?.visible !== true)) failures.push(`${scenario.name}: unavailable assessment is absent from the captured viewport`);
    const formsPath = `/api/v1/vendors/${relationshipID}/forms`;
    // The existing portfolio batch summary is fetched before selecting a row.
    // Retain that receipt separately and only use summaries whose explicit
    // relationship_ids include this selected record.
    const summaryReads = [...reads, ...(layout.preparationReads ?? [])].filter((read) => read.path === "/api/v1/vendors/form-summaries"
      && new URLSearchParams(read.query).get("relationship_ids")?.split(",").includes(relationshipID));
    if (check === "empty" && (interaction?.empty !== true || interaction.requestFormReachable !== true || interaction.status !== "Not assessed" || interaction.forms !== 0
      || layout.settledContent?.formsEmptyState !== true || layout.settledContent?.complianceStatus !== "Not assessed"
      || !reads.some((read) => read.path === formsPath && read.status === 200 && read.responseItems === 0 && read.nextCursor === false)
      || !summaryReads.some((read) => read.status === 200 && read.responseSummary?.relationship_id === relationshipID && read.responseSummary?.submitted_forms === 0 && read.responseSummary?.outstanding_forms === 0))) failures.push(`${scenario.name}: empty forms need observed empty reads and a neutral request action`);
    if (check === "partial" && (interaction?.partial !== true || interaction.status !== "Unavailable" || interaction.forms !== 1 || interaction.moreFormsReachable !== true
      || interaction.unavailableCapture?.visible !== true || interaction.partialCapture?.visible !== true
      || layout.settledContent?.summaryCountsVisible !== false || layout.settledContent?.formsShownScope !== "1 forms shown"
      || !["Submitted forms", "Awaiting review"].every((label) => layout.statusFields?.some((field) => field.label === label && field.value === "Unavailable"))
      || !reads.some((read) => read.path === formsPath && read.status === 200 && read.responseItems === 1 && read.nextCursor === true)
      || !summaryReads.some((read) => read.status === 503))) failures.push(`${scenario.name}: partial forms need visible limited rows and unknown summary counts`);
  }
  return { failures, check: { name: "Vendor relationship source layout and supported workflows", status: failures.length ? "FAIL" : "PASS",
    detail: `${vendorRelationshipEvidenceScenarios.filter((scenario) => byName.has(scenario.name)).length}/${vendorRelationshipEvidenceScenarios.length} captures; five sections at five widths in two themes with loading, empty, partial, unavailable, long-label, documents and activation permission states` } };
}
