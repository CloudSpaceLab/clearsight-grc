import { vendorPortfolioEvidenceScenarios, vendorPortfolioViews } from "./vendor-portfolio-evidence-scenarios.mjs";

const same = (a, b) => JSON.stringify(a) === JSON.stringify(b);
const rect = (value) => value && ["x", "y", "width", "height", "right", "bottom"].every((key) => Number.isFinite(value[key]))
  && value.width > 0 && value.height > 0 && Math.abs(value.x + value.width - value.right) <= 1 && Math.abs(value.y + value.height - value.bottom) <= 1;
const labels = vendorPortfolioViews.map((view) => view.label);
const relationshipID = "vendor-relationship-payments";
const collectionRouteMatches = (state, view, titles, action, observed = true) => Boolean(state
  && state.action === action && state.route === `#vendors/${view}` && state.view === view
  && state.heading === "Vendors" && state.selectedTab === vendorPortfolioViews.find((item) => item.id === view)?.label
  && state.portfolioVisible === true && state.collectionVisible === true && state.selectedPanelVisible === true
  && state.selectedRelationshipVisible === false && state.loadingVisible === false && state.complete === true
  && Array.isArray(titles) && titles.length > 0 && state.recordCount === titles.length && same(state.recordTitles, titles)
  && (observed ? state.pendingAPIRequests === 0 && state.observedReadsSettled === true && state.sameDocument === true : state.newDocument === true));

export function reviewVendorPortfolioCaptures(captures) {
  const byName = new Map(captures.map((capture) => [capture.name, capture]));
  const failures = [];
  for (const scenario of vendorPortfolioEvidenceScenarios) {
    const capture = byName.get(scenario.name);
    if (!capture) { failures.push(`${scenario.name}: Vendor portfolio evidence is missing`); continue; }
    const layout = capture.vendorPortfolioLayout;
    const check = scenario.vendorPortfolioCheck;
    const width = scenario.viewport.width;
    const label = vendorPortfolioViews.find((view) => view.id === scenario.vendorPortfolioView).label;
    if (!layout || ["route", "fixture", "state", "theme"].some((key) => capture[key] !== scenario[key])
      || !same(capture.viewport, scenario.viewport) || layout.kind !== scenario.vendorPortfolioView
      || layout.evidenceKind !== "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION" || capture.navigation?.activePage?.join("|") !== "Vendors") {
      failures.push(`${scenario.name}: Vendor portfolio metadata, fixture or navigation owner is inconsistent`);
      if (!layout) continue;
    }
    if (layout.headingText !== "Vendors" || !rect(layout.heading) || !rect(layout.title) || layout.titleContained !== true
      || !same(layout.localTabs, labels) || layout.selectedTab !== label || !rect(layout.tablist)
      || !layout.selectedPanelID || !same(layout.visiblePanelIDs, [layout.selectedPanelID]) || layout.selectedPanelControlsMatch !== true
      || !["tabs", "select"].includes(layout.menuMode) || layout.menuMode === "select" && layout.compactValue !== label
      || !rect(layout.selectedTabBounds) || layout.selectedTabVisible !== true
      || layout.selectedTabBounds.x < layout.tablist.x - 1 || layout.selectedTabBounds.right > layout.tablist.right + 1
      || layout.heading.bottom > layout.tablist.y + 1 || layout.route !== scenario.route || layout.selectedRelationshipVisible !== false
      || layout.documentWidth !== width || layout.scrollWidth > width + 1
      || scenario.vendorPortfolioMatrix && (!rect(layout.firstUsefulContent) || layout.firstUsefulContentVisible !== true
        || layout.firstUsefulContent.y < 0 || Math.min(layout.firstUsefulContent.bottom, scenario.viewport.height) - layout.firstUsefulContent.y < 44)
      || scenario.touch && (layout.minTargetWidth < 44 || layout.minTargetHeight < 44 || layout.undersizedTargets?.length !== 0)) {
      failures.push(`${scenario.name}: complete portfolio menu, heading, first content or accessible replacement is missing`);
    }
    if (scenario.vendorPortfolioView === "overview" && !["loading", "unavailable", "denied"].includes(check)) {
      const expected = width > 1120 ? "beside" : "stacked";
      if (layout.metrics?.length !== 4 || layout.metrics.some((item) => !rect(item.bounds) || !item.label || item.value == null)
        || layout.insights?.length !== 2 || !same(layout.insights.map((item) => item.heading), ["Service criticality", "Exception stages"])
        || layout.insights.some((item) => !rect(item.bounds)) || !same(layout.railHeadings, ["Needs attention", "Service renewals"])
        || !rect(layout.main) || !rect(layout.aside) || !rect(layout.register) || layout.placement !== expected
        || expected === "beside" && (layout.main.right > layout.aside.x + 1 || Math.abs(layout.main.y - layout.aside.y) > 2 || Math.abs(layout.aside.width - (width > 1420 ? 313 : 295)) > 2)
        || expected === "stacked" && (layout.main.bottom > layout.aside.y + 1 || Math.abs(layout.main.x - layout.aside.x) > 2 || Math.abs(layout.main.width - layout.aside.width) > 2)
        || layout.register.y < Math.max(...layout.insights.map((item) => item.bounds.bottom)) - 1) {
        failures.push(`${scenario.name}: Overview needs four measures, two insights, inline register and responsive attention/renewals rail`);
      }
    }
    const reads = layout.apiReads ?? [];
    const listReads = reads.filter((read) => read.path === "/api/v1/vendors");
    const summaryReads = reads.filter((read) => read.path === "/api/v1/vendors/form-summaries");
    const population = new Set(listReads.flatMap((read) => read.relationshipIDs ?? []));
    if (!Array.isArray(layout.recordIDs) || layout.recordIDs.length !== layout.registerRows
      || new Set(layout.recordIDs).size !== layout.registerRows || layout.recordIDs.some((id) => !population.has(id))
      || scenario.vendorPortfolioMatrix && layout.registerRows < 1) {
      failures.push(`${scenario.name}: visible relationship records must bind to the observed list population`);
    }
    const invalid = reads.some((read) => {
      const query = new URLSearchParams(read.query);
      if (read.path === "/api/v1/vendors") return read.method !== "GET" || query.get("limit") !== "50";
      if (read.path === "/api/v1/vendors/form-summaries") {
        const ids = query.get("relationship_ids")?.split(",") ?? [];
        return read.method !== "GET" || query.get("limit") !== "50" || !ids.length || ids.length > 50
          || ids.some((id) => !population.has(id)) || read.summaryIDs?.some((id) => !ids.includes(id));
      }
      if (read.path === "/api/v1/authority/resolve") return read.method !== "POST" || read.requestKind !== "AUTHORITY_READ_QUERY"
        || read.queryInput?.object_type !== "THIRD_PARTY_RELATIONSHIP" || read.queryInput?.object_id !== relationshipID;
      const match = /^\/api\/v1\/vendors\/([^/]+)(?:\/.*)?$/.exec(read.path);
      if (match) return !population.has(decodeURIComponent(match[1]));
      if (read.path === "/api/v1/vendor-work") return read.method !== "GET" || query.get("relationship_id") !== relationshipID || query.get("limit") !== "20";
      if (read.path === "/api/v1/forms/documents") return read.method !== "GET" || query.get("relationship_id") !== relationshipID || query.get("current_only") !== "true" || query.get("limit") !== "25";
      if (read.path === "/api/v1/forms/responses") return read.method !== "GET" || query.get("subject_type") !== "VENDOR_RELATIONSHIP" || query.get("subject_id") !== relationshipID || query.get("limit") !== "25";
      return false;
    });
    const expectedStatus = check === "loading" ? "PENDING" : check === "unavailable" ? 503 : check === "denied" ? 403 : 200;
    if (!Array.isArray(layout.apiReads) || !Array.isArray(layout.commands) || layout.commands.length || invalid
      || !listReads.some((read) => read.status === expectedStatus)
      || scenario.vendorPortfolioMatrix && !summaryReads.some((read) => read.status === 200)) {
      failures.push(`${scenario.name}: portfolio viewing requires bounded observed reads without foreign scope or commands`);
    }
    const interaction = layout.interaction;
    if (scenario.verifyVendorPortfolioNavigation && (!same(interaction?.visited, labels) || interaction.baselineRestored !== true
      || interaction.keyboard?.complete !== true || !same(interaction.keyboard.options, labels)
      || !same(interaction.keyboard.sequence, width > 760 ? [...labels.slice(1), labels[0]] : ["Register"]))) {
      failures.push(`${scenario.name}: five-menu keyboard navigation and return lack interaction proof`);
    }
    if (scenario.verifyVendorPortfolioNavigation && width === 1440 && (interaction?.exceptionReturn?.open !== true || interaction.exceptionReturn.rowVisible !== true
      || !(interaction.exceptionReturn.priorScrollY > 0) || Math.abs(interaction.exceptionReturn.scrollY - interaction.exceptionReturn.priorScrollY) > 4
      || interaction.exceptionReturn.exactMatterID !== "sample-portfolio-finding-0" || interaction.exceptionReturn.baselineRestored !== true)) failures.push(`${scenario.name}: exception return requires the settled open queue and its previous scroll position`);
    if (scenario.verifyVendorPortfolioNavigation && width === 390 && (!(interaction?.queueTargets?.minimumWidth >= 44) || !(interaction.queueTargets.minimumHeight >= 44)
      || interaction.queueTargets.undersized?.length !== 0 || interaction.queueTargets.baselineRestored !== true)) failures.push(`${scenario.name}: opened phone queue requires 44px targets and a restored baseline`);
    if (scenario.verifyVendorPortfolioFilters && (interaction?.filters?.initiallyCollapsed !== true || interaction.filters.expanded !== true
      || interaction.filters.selection !== "Critical" || interaction.filters.filteredRecords !== 0 || interaction.filters.appliedCountRetained !== true
      || interaction.filters.cleared !== true || interaction.filters.baselineRestored !== true)) failures.push(`${scenario.name}: phone filters require expand, apply, collapse and clear proof`);
    if (check === "loading" && !(interaction?.pending > 0)) failures.push(`${scenario.name}: pending reads require visible loading proof`);
    if (check === "unavailable" && (interaction?.retryIssued !== true || listReads.filter((read) => read.status === 503).length < 2)) failures.push(`${scenario.name}: failed reads require an observed retry`);
    if (check === "empty" && (interaction?.empty !== true || !listReads.some((read) => read.status === 200 && read.responseItems === 0))) failures.push(`${scenario.name}: empty records require observed zero results`);
    if (check === "partial" && (interaction?.loadMoreAvailable !== true || !listReads.some((read) => read.nextCursor === true))) failures.push(`${scenario.name}: partial records require a visible cursor action`);
    if (check === "summaries-unavailable" && (interaction?.unavailable !== true || interaction.unknownVisible !== true || !summaryReads.some((read) => read.status === 503))) failures.push(`${scenario.name}: failed form counts must remain unknown`);
    if (check === "summary-missing" && (interaction?.unavailable !== true || interaction.unknownVisible !== true
      || !summaryReads.some((read) => read.status === 200 && !read.summaryIDs?.includes(relationshipID)))) failures.push(`${scenario.name}: an omitted successful summary must remain unknown`);
    if (check === "long-labels" && interaction?.longLabels !== true) failures.push(`${scenario.name}: long relationship identity was not rendered`);
    if (check === "denied" && (interaction?.addDisabled !== true || layout.registerRows !== 0)) failures.push(`${scenario.name}: denied scope must hide records and disable creation`);
    if (check === "exceptions-unavailable" && (interaction?.unknownTotals !== true || !reads.some((read) => read.path.endsWith("/links") && read.status === 503))) failures.push(`${scenario.name}: failed linked issues must keep totals unknown`);
    if (check === "search-paging" && (interaction?.search !== "Acme" || interaction.paged !== true
      || !listReads.some((read) => new URLSearchParams(read.query).get("search") === "Acme" && !new URLSearchParams(read.query).has("cursor"))
      || !listReads.some((read) => new URLSearchParams(read.query).get("search") === "Acme" && new URLSearchParams(read.query).get("cursor") === "sample-vendor-page-2" && read.status === 200))) {
      failures.push(`${scenario.name}: search must reach the reader and survive cursor pagination`);
    }
    if (["documents-drill", "onboarding-drill", "assessments-drill"].includes(check)) {
      const feature = check === "documents-drill" ? "protectedDocuments" : check === "onboarding-drill" ? "dueDiligence" : "forms";
      const endpoint = check === "documents-drill" ? "/api/v1/forms/documents"
        : `/api/v1/vendors/${relationshipID}/${check === "onboarding-drill" ? "assessments/current" : "forms"}`;
      if (interaction?.relationshipPreserved !== true || interaction.returned !== true || interaction[feature] !== true
        || interaction.refresh?.relationshipPreserved !== true || interaction.refresh.initialSection !== "Overview" || interaction.refresh.complete !== true
        || interaction.refresh.reopenedSection !== (check === "onboarding-drill" ? "Due diligence" : "Evidence")
        || check !== "onboarding-drill" && interaction.refresh.evidenceView !== (check === "documents-drill" ? "Documents" : "Forms")
        || !reads.some((read) => read.path === endpoint && (read.status === 200 || check === "onboarding-drill" && read.status === 404))) {
        failures.push(`${scenario.name}: collection drill must reach its exact existing relationship reader and return`);
      }
      const titles = interaction?.initialCollection?.recordTitles;
      if (!collectionRouteMatches(interaction?.browserBack, scenario.vendorPortfolioView, titles, "HISTORY_BACK")
        || interaction?.browserForward?.action !== "HISTORY_FORWARD" || interaction.browserForward.route !== `#vendors/register/${relationshipID}`
        || interaction.browserForward.selectedRelationshipVisible !== true || interaction.browserForward.portfolioVisible !== false
        || interaction.browserForward.relationshipTitle !== "Acme Processing Limited" || interaction.browserForward.sameDocument !== true || interaction.browserForward.observedReadsSettled !== true
        || !collectionRouteMatches(interaction?.refresh?.directHashRoot, scenario.vendorPortfolioView, titles, "DIRECT_HASH_ROOT")
        || !collectionRouteMatches(interaction?.refresh?.reloadedCollection, scenario.vendorPortfolioView, titles, "RELOAD_COLLECTION", false)
        || !collectionRouteMatches(interaction?.explicitRegisterReturn, "register", titles, "IN_APP_REGISTER_RETURN")) {
        failures.push(`${scenario.name}: browser Back and direct collection routes must restore settled collection DOM without a selected relationship`);
      }
    }
  }
  return { failures, check: { name: "Vendor portfolio source layout and bounded collection workflows", status: failures.length ? "FAIL" : "PASS",
    detail: `${vendorPortfolioEvidenceScenarios.filter((scenario) => byName.has(scenario.name)).length}/${vendorPortfolioEvidenceScenarios.length} captures; five views at five widths in two themes plus loading, empty, partial, denied, failed reads, search/paging and exact collection drills` } };
}
