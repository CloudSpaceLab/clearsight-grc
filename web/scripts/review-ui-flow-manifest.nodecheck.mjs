import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import test from "node:test";
import "./privacy-evidence-contract.nodecheck.mjs";
import { frameworkEvidenceScenarios } from "./framework-evidence-scenarios.mjs";
import { reportsDesignEvidenceScenarios } from "./reports-design-evidence-scenarios.mjs";
import { reviewReportsDesignCaptures } from "./reports-design-evidence-contract.mjs";
import { notificationsEvidenceScenarios } from "./notifications-evidence-scenarios.mjs";
import { reviewNotificationsCaptures } from "./notifications-evidence-contract.mjs";
import { vendorRelationshipEvidenceScenarios } from "./vendor-relationship-evidence-scenarios.mjs";
import { reviewVendorRelationshipCaptures } from "./vendor-relationship-evidence-contract.mjs";
import { vendorPortfolioEvidenceScenarios, vendorPortfolioViews } from "./vendor-portfolio-evidence-scenarios.mjs";
import { reviewVendorPortfolioCaptures } from "./vendor-portfolio-evidence-contract.mjs";
import { riskLossEvidenceScenarios, riskLossViews } from "./risk-loss-evidence-scenarios.mjs";
import { reviewRiskLossCaptures } from "./risk-loss-evidence-contract.mjs";

const webRoot = fileURLToPath(new URL("..", import.meta.url));

// These deliberately incomplete manifests test reviewer diagnostics, not rendered proof.
async function review(captures) {
  const evidenceDir = await mkdtemp(path.join(tmpdir(), "clearsight-manifest-nodecheck-"));
  try {
    await writeFile(path.join(evidenceDir, "manifest.json"), JSON.stringify({ captures }));
    assert.throws(() => execFileSync(process.execPath, ["scripts/review-ui-flow-manifest.mjs"], {
      cwd: webRoot,
      env: { ...process.env, UI_EVIDENCE_DIR: evidenceDir },
      stdio: "pipe",
    }), (error) => error.status === 1);
    return JSON.parse(await readFile(path.join(evidenceDir, "review.json"), "utf8"));
  } finally {
    await rm(evidenceDir, { recursive: true, force: true });
  }
}

test("the manifest reviewer rejects duplicate capture names", async () => {
  const capture = { name: "01-today-dark-comfortable-1440x900", state: "baseline", theme: "dark", viewport: { width: 1440, height: 900 } };
  const { failures } = await review([capture, capture]);
  assert.ok(failures.includes("flow manifest contains duplicate capture names"));
});

test("the manifest reviewer rejects unknown capture names", async () => {
  const capture = { name: "01-today-dark-comfortable-1440x900", state: "baseline", theme: "dark", viewport: { width: 1440, height: 900 } };
  const unknown = { ...capture, name: "unknown-capture-light-1440x900" };
  const { failures } = await review([capture, unknown]);
  assert.ok(failures.some((failure) => failure.startsWith("flow manifest contains unexpected captures:") && failure.includes(unknown.name)));
});

test("framework evidence names are required and may not silently disappear", async () => {
  const { failures } = await review([]);
  const missing = failures.find((failure) => failure.startsWith("flow manifest is missing:")) ?? "";
  for (const scenario of frameworkEvidenceScenarios) assert.ok(missing.includes(scenario.name), scenario.name);
  assert.equal(new Set(frameworkEvidenceScenarios.map((scenario) => scenario.name)).size, frameworkEvidenceScenarios.length);
});

test("a Reports screenshot cannot substitute for measured source-menu evidence", async () => {
  const capture = { name: "140a-report-overview-light-1440x900", route: "#reports", fixture: "report-overview-loaded",
    state: "report-overview-desktop", theme: "light", viewport: { width: 1440, height: 900 } };
  const { failures } = await review([capture]);
  assert.ok(failures.includes(`${capture.name}: primary navigation source, permission or measured geometry evidence is missing or inconsistent`));
  assert.ok(failures.includes("primary navigation 1440px is missing keyboard, history or risk-family selection evidence"));
});

test("the menu receipt rejects undersized controls and wrapped primary labels", async () => {
  const navigation = {
    labels: ["Home", "Frameworks", "Programs", "Forms", "Vendors", "Risk & Losses", "Privacy", "Work & Approvals", "Reports"],
    hrefs: ["#oversight", "#programs/library", "#programs", "#forms", "#vendors", "#risks", "#ropa", "#work", "#reports"],
    activePage: ["Reports"], railWidth: 228, railX: 0, labelMode: "expanded", maxLabelLines: 1, labelsContained: true,
    minTargetWidth: 44, minTargetHeight: 49, targetsContained: true, primaryReachable: true,
    settingsPresent: true, settingsFooter: true, footerReachable: true, compactMenuAvailable: false,
    duplicateMobileNavigation: false, legacyPortfolioNavigation: false,
    activeMarkerVisible: true, activeMarkerWidth: 4, activeMarkerHeight: 30,
  };
  const capture = { name: "140a-report-overview-light-1440x900", route: "#reports", fixture: "report-overview-loaded",
    state: "report-overview-desktop", theme: "light", viewport: { width: 1440, height: 900 }, navigation };
  const diagnostic = `${capture.name}: primary navigation source, permission or measured geometry evidence is missing or inconsistent`;
  assert.ok(!(await review([capture])).failures.includes(diagnostic));
  assert.ok((await review([{ ...capture, navigation: { ...navigation, minTargetWidth: 43 } }])).failures.includes(diagnostic));
  assert.ok((await review([{ ...capture, navigation: { ...navigation, maxLabelLines: 2 } }])).failures.includes(diagnostic));
});

test("compact Navigation cannot pass with a float-narrowed content column", async () => {
  const drawer = {
    labels: ["Home", "Frameworks", "Programs", "Forms", "Vendors", "Risk & Losses", "Privacy", "Work & Approvals", "Reports"],
    activePage: ["Reports"], settingsPresent: true,
    riskChildren: ["Risks", "RCSA", "Key indicators", "Loss events"], riskExpandedByKeyboard: true,
    labelsVisible: true, maxLabelLines: 1, targetsContained: true, bounded: true,
    currentPageVisible: true, currentPageBounds: { top: 665, bottom: 709, clipTop: 104, clipBottom: 713 },
    contentFillsDialog: true, navigationFillsContent: true,
    dialogWidth: 389, contentWidth: 389, navigationWidth: 341, expectedNavigationWidth: 341,
    minTargetWidth: 44, minTargetHeight: 49, focusContained: true, dismissalRestoredFocus: true,
  };
  const capture = { name: "221-navigation-menu-light-mobile-390x844", state: "navigation-menu-light-mobile",
    fixture: "report-overview-loaded", theme: "light", viewport: { width: 390, height: 844 }, navigation: { drawer } };
  const diagnostic = `${capture.name}: compact Navigation lacks labelled destinations, permission checks or keyboard dismissal evidence`;
  assert.ok(!(await review([capture])).failures.includes(diagnostic));
  assert.ok((await review([{ ...capture, navigation: { drawer: { ...drawer, contentWidth: 285 } } }])).failures.includes(diagnostic));
  assert.ok((await review([{ ...capture, navigation: { drawer: { ...drawer,
    currentPageBounds: { ...drawer.currentPageBounds, bottom: 735 } } } }])).failures.includes(diagnostic));
});

test("the new Reports view and workflow captures cannot silently disappear", async () => {
  const { failures } = await review([]);
  const missing = failures.find((failure) => failure.startsWith("flow manifest is missing:")) ?? "";
  for (const scenario of reportsDesignEvidenceScenarios) assert.ok(missing.includes(scenario.name), scenario.name);
  assert.equal(new Set(reportsDesignEvidenceScenarios.map((scenario) => scenario.name)).size, reportsDesignEvidenceScenarios.length);
});

const rect = (x, y, width, height) => ({ x, y, width, height, right: x + width, bottom: y + height });

test("the five-view Vendor portfolio matrix and exceptional states cannot disappear", async () => {
  const { failures } = await review([]);
  const missing = failures.find((failure) => failure.startsWith("flow manifest is missing:")) ?? "";
  for (const scenario of vendorPortfolioEvidenceScenarios) assert.ok(missing.includes(scenario.name), scenario.name);
  assert.equal(new Set(vendorPortfolioEvidenceScenarios.map((scenario) => scenario.name)).size, 63);
});

function portfolioCapture(scenario = vendorPortfolioEvidenceScenarios[0]) {
  const labels = vendorPortfolioViews.map((view) => view.label);
  const selected = vendorPortfolioViews.find((view) => view.id === scenario.vendorPortfolioView).label;
  const width = scenario.viewport.width;
  const contentX = width === 390 ? 68 : 260;
  const contentWidth = width - contentX - 12;
  return { ...scenario, navigation: { activePage: ["Vendors"] }, vendorPortfolioLayout: {
    kind: scenario.vendorPortfolioView, route: scenario.route, evidenceKind: "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION",
    headingText: "Vendors", heading: rect(contentX, 96, contentWidth, 90), title: rect(contentX, 110, Math.min(220, contentWidth), 40), titleContained: true,
    localTabs: labels, selectedTab: selected, selectedPanelID: "sample-portfolio-panel", visiblePanelIDs: ["sample-portfolio-panel"], selectedPanelControlsMatch: true,
    menuMode: width === 390 ? "select" : "tabs", compactValue: selected, tablist: rect(contentX, 205, contentWidth, 44), selectedTabBounds: rect(contentX, 205, 100, 44), selectedTabVisible: true,
    documentWidth: width, scrollWidth: width, selectedRelationshipVisible: false, minTargetWidth: 44, minTargetHeight: 44, undersizedTargets: [],
    main: rect(260, 405, 815, 500), aside: rect(1092, 405, 313, 600), placement: "beside",
    metrics: ["2 vendor services", "5 open exceptions", "5 open actions", "5 overdue actions"].map((label, index) => ({ label, value: index ? "5" : "2", bounds: rect(260 + index * 285, 270, 274, 105) })),
    insights: [{ heading: "Service criticality", bounds: rect(260, 405, 399, 190) }, { heading: "Exception stages", bounds: rect(675, 405, 400, 190) }],
    railHeadings: ["Needs attention", "Service renewals"], register: rect(260, 611, 815, 294), registerRows: 2,
    recordIDs: ["vendor-relationship-payments", "sample-relationship-hosting"], firstRecord: rect(contentX, 400, contentWidth, 64), recordHeights: [64, 64], firstRecordVisible: true,
    firstUsefulContent: rect(contentX, 270, contentWidth, 105), firstUsefulContentVisible: true,
    apiReads: [{ path: "/api/v1/vendors", method: "GET", query: "?limit=50", status: 200, responseItems: 2, relationshipIDs: ["vendor-relationship-payments", "sample-relationship-hosting"] },
      { path: "/api/v1/vendors/form-summaries", method: "GET", query: "?relationship_ids=vendor-relationship-payments,sample-relationship-hosting&limit=50", status: 200, summaryIDs: ["vendor-relationship-payments", "sample-relationship-hosting"] }],
    commands: [], interaction: scenario.verifyVendorPortfolioNavigation ? { visited: labels, baselineRestored: true, keyboard: { options: labels, sequence: [...labels.slice(1), labels[0]], complete: true },
      exceptionReturn: { open: true, rowVisible: true, priorScrollY: 1000, scrollY: 1000, exactMatterID: "sample-portfolio-finding-0", baselineRestored: true } } : undefined,
  } };
}
const portfolioFailures = (capture) => reviewVendorPortfolioCaptures([capture]).failures.filter((failure) => failure.startsWith(`${capture.name}:`));

test("Vendor portfolio rejects a missing menu destination, hidden first content and false population", () => {
  const capture = portfolioCapture();
  assert.deepEqual(portfolioFailures(capture), []);
  for (const mutate of [
    (layout) => layout.localTabs.pop(),
    (layout) => { layout.firstUsefulContent = rect(260, 910, 1148, 100); },
    (layout) => { layout.visiblePanelIDs.push("leaked-register-panel"); },
  ]) {
    const changed = structuredClone(capture); mutate(changed.vendorPortfolioLayout);
    assert.ok(portfolioFailures(changed).some((failure) => failure.includes("complete portfolio menu")));
  }
  const railRows = structuredClone(capture); railRows.vendorPortfolioLayout.registerRows = 7;
  assert.ok(portfolioFailures(railRows).some((failure) => failure.includes("observed list population")));
  const foreignRecord = structuredClone(capture); foreignRecord.vendorPortfolioLayout.recordIDs[0] = "foreign-relationship";
  assert.ok(portfolioFailures(foreignRecord).some((failure) => failure.includes("observed list population")));
  const lostQueue = structuredClone(capture); lostQueue.vendorPortfolioLayout.interaction.exceptionReturn.scrollY = 350;
  assert.ok(portfolioFailures(lostQueue).some((failure) => failure.includes("settled open queue")));
});

test("Vendor Overview requires two insights, ordered inline register and the responsive rail", () => {
  const capture = portfolioCapture();
  for (const mutate of [
    (layout) => layout.insights.pop(),
    (layout) => { layout.register = rect(260, 300, 815, 294); },
    (layout) => { layout.aside = rect(260, 920, 316, 600); layout.placement = "stacked"; },
    (layout) => { layout.aside = rect(1092, 405, 250, 600); },
    (layout) => { layout.metrics.pop(); },
  ]) {
    const changed = structuredClone(capture); mutate(changed.vendorPortfolioLayout);
    assert.ok(portfolioFailures(changed).some((failure) => failure.includes("four measures, two insights")));
  }
});

test("Vendor portfolio accepts bounded reads and rejects foreign scope, broad reads and commands", () => {
  const capture = portfolioCapture();
  for (const read of [
    { path: "/api/v1/vendors", method: "GET", query: "?limit=500", status: 200 },
    { path: "/api/v1/vendors/form-summaries", method: "GET", query: "?relationship_ids=foreign-relationship&limit=50", status: 200 },
    { path: "/api/v1/vendors/foreign-relationship/links", method: "GET", query: "?limit=50", status: 200 },
    { path: "/api/v1/authority/resolve", method: "POST", requestKind: "AUTHORITY_READ_QUERY", queryInput: { object_type: "THIRD_PARTY_RELATIONSHIP", object_id: "foreign-relationship" }, status: 200 },
  ]) {
    const changed = structuredClone(capture); changed.vendorPortfolioLayout.apiReads.push(read);
    assert.ok(portfolioFailures(changed).some((failure) => failure.includes("bounded observed reads")));
  }
  const command = structuredClone(capture); command.vendorPortfolioLayout.commands.push({ path: "/api/v1/vendors/vendor-relationship-payments/activate", method: "POST" });
  assert.ok(portfolioFailures(command).some((failure) => failure.includes("bounded observed reads")));
  const wrongSummary = structuredClone(capture); wrongSummary.vendorPortfolioLayout.apiReads[1].summaryIDs.push("foreign-relationship");
  assert.ok(portfolioFailures(wrongSummary).some((failure) => failure.includes("bounded observed reads")));
});

test("Vendor search evidence must carry its query through the cursor read", () => {
  const scenario = vendorPortfolioEvidenceScenarios.find((item) => item.vendorPortfolioCheck === "search-paging");
  const capture = portfolioCapture(scenario);
  capture.vendorPortfolioLayout.interaction = { search: "Acme", paged: true };
  capture.vendorPortfolioLayout.apiReads.push({ path: "/api/v1/vendors", method: "GET", query: "?limit=50&search=Acme", status: 200 },
    { path: "/api/v1/vendors", method: "GET", query: "?limit=50&search=Acme&cursor=sample-vendor-page-2", status: 200 });
  assert.deepEqual(portfolioFailures(capture), []);
  capture.vendorPortfolioLayout.apiReads.at(-1).query = "?limit=50&cursor=sample-vendor-page-2";
  assert.ok(portfolioFailures(capture).some((failure) => failure.includes("survive cursor pagination")));
});

function collectionInteraction(scenario) {
  const view = scenario.vendorPortfolioView;
  const check = scenario.vendorPortfolioCheck;
  const titles = ["Acme Processing Limited", "Sample · Northshore Hosting Limited"];
  const routeState = (view, action, observed = true) => ({ action, route: `#vendors/${view}`, view, heading: "Vendors",
    selectedTab: vendorPortfolioViews.find((item) => item.id === view).label, portfolioVisible: true, collectionVisible: true, selectedPanelVisible: true,
    selectedRelationshipVisible: false, loadingVisible: false, complete: true, recordCount: titles.length, recordTitles: titles,
    pendingAPIRequests: observed ? 0 : null, observedReadsSettled: observed, sameDocument: observed ? true : null, ...(!observed ? { newDocument: true } : {}) });
  return { relationshipPreserved: true, returned: true,
    [check === "documents-drill" ? "protectedDocuments" : check === "onboarding-drill" ? "dueDiligence" : "forms"]: true,
    initialCollection: { recordTitles: titles }, browserBack: routeState(view, "HISTORY_BACK"),
    browserForward: { action: "HISTORY_FORWARD", route: "#vendors/register/vendor-relationship-payments", selectedRelationshipVisible: true, portfolioVisible: false,
      relationshipTitle: "Acme Processing Limited", sameDocument: true, observedReadsSettled: true },
    explicitRegisterReturn: routeState("register", "IN_APP_REGISTER_RETURN"),
    refresh: { relationshipPreserved: true, initialSection: "Overview", complete: true,
      reopenedSection: check === "onboarding-drill" ? "Due diligence" : "Evidence", evidenceView: check === "onboarding-drill" ? null : check === "documents-drill" ? "Documents" : "Forms",
      directHashRoot: routeState(view, "DIRECT_HASH_ROOT"), reloadedCollection: routeState(view, "RELOAD_COLLECTION", false) } };
}

test("Vendor document collection requires the exact protected document reader and return", () => {
  const scenario = vendorPortfolioEvidenceScenarios.find((item) => item.vendorPortfolioCheck === "documents-drill");
  const capture = portfolioCapture(scenario);
  capture.vendorPortfolioLayout.interaction = collectionInteraction(scenario);
  capture.vendorPortfolioLayout.apiReads.push({ path: "/api/v1/forms/documents", method: "GET", query: "?relationship_id=vendor-relationship-payments&current_only=true&limit=25", status: 200 });
  assert.deepEqual(portfolioFailures(capture), []);
  const missing = structuredClone(capture); missing.vendorPortfolioLayout.apiReads.pop();
  assert.ok(portfolioFailures(missing).some((failure) => failure.includes("exact existing relationship reader")));
  const unprotected = structuredClone(capture); unprotected.vendorPortfolioLayout.apiReads.at(-1).query = "?relationship_id=vendor-relationship-payments&limit=25";
  assert.ok(portfolioFailures(unprotected).some((failure) => failure.includes("bounded observed reads")));
  const lostRefresh = structuredClone(capture); lostRefresh.vendorPortfolioLayout.interaction.refresh.relationshipPreserved = false;
  assert.ok(portfolioFailures(lostRefresh).some((failure) => failure.includes("exact existing relationship reader")));
});

test("all Vendor collection drills require actual history and settled rendered return state", () => {
  for (const scenario of vendorPortfolioEvidenceScenarios.filter((item) => ["onboarding-drill", "assessments-drill", "documents-drill"].includes(item.vendorPortfolioCheck))) {
    const capture = portfolioCapture(scenario);
    capture.vendorPortfolioLayout.interaction = collectionInteraction(scenario);
    capture.vendorPortfolioLayout.apiReads.push(scenario.vendorPortfolioCheck === "documents-drill"
      ? { path: "/api/v1/forms/documents", method: "GET", query: "?relationship_id=vendor-relationship-payments&current_only=true&limit=25", status: 200 }
      : { path: `/api/v1/vendors/vendor-relationship-payments/${scenario.vendorPortfolioCheck === "onboarding-drill" ? "assessments/current" : "forms"}`, method: "GET", query: "", status: 200 });
    assert.deepEqual(portfolioFailures(capture), [], scenario.name);
    for (const mutate of [
      (proof) => { delete proof.browserBack; }, // The old returned:true contract cannot pass.
      (proof) => { proof.browserBack.selectedRelationshipVisible = true; proof.browserBack.portfolioVisible = false; },
      (proof) => { proof.browserBack.pendingAPIRequests = 1; },
      (proof) => { proof.browserBack.sameDocument = false; },
      (proof) => { proof.browserBack.action = "IN_APP_REGISTER_RETURN"; },
      (proof) => { proof.browserBack.recordTitles = ["Foreign relationship"]; },
      (proof) => { proof.browserForward.route = "#vendors/register/foreign-relationship"; },
      (proof) => { proof.refresh.directHashRoot.selectedRelationshipVisible = true; },
      (proof) => { proof.refresh.reloadedCollection.newDocument = false; },
    ]) {
      const changed = structuredClone(capture); mutate(changed.vendorPortfolioLayout.interaction);
      assert.ok(portfolioFailures(changed).some((failure) => failure.includes("browser Back and direct collection routes")), scenario.name);
    }
  }
});

test("Vendor phone filters require a restored baseline and omitted summaries remain unknown", () => {
  const filters = portfolioCapture(vendorPortfolioEvidenceScenarios.find((item) => item.verifyVendorPortfolioFilters));
  filters.vendorPortfolioLayout.interaction = { filters: { initiallyCollapsed: true, expanded: true, selection: "Critical", filteredRecords: 0, appliedCountRetained: true, cleared: true, baselineRestored: true } };
  assert.deepEqual(portfolioFailures(filters), []);
  filters.vendorPortfolioLayout.interaction.filters.appliedCountRetained = false;
  assert.ok(portfolioFailures(filters).some((failure) => failure.includes("phone filters require")));
  const missing = portfolioCapture(vendorPortfolioEvidenceScenarios.find((item) => item.vendorPortfolioCheck === "summary-missing"));
  missing.vendorPortfolioLayout.interaction = { unavailable: true, unknownVisible: true };
  missing.vendorPortfolioLayout.apiReads[1].summaryIDs = ["sample-relationship-hosting"];
  assert.deepEqual(portfolioFailures(missing), []);
  missing.vendorPortfolioLayout.interaction.unknownVisible = false;
  assert.ok(portfolioFailures(missing).some((failure) => failure.includes("omitted successful summary must remain unknown")));
});

test("the five-section Vendor relationship matrix cannot silently disappear", async () => {
  const { failures } = await review([]);
  const missing = failures.find((failure) => failure.startsWith("flow manifest is missing:")) ?? "";
  for (const scenario of vendorRelationshipEvidenceScenarios) assert.ok(missing.includes(scenario.name), scenario.name);
  assert.equal(new Set(vendorRelationshipEvidenceScenarios.map((scenario) => scenario.name)).size, vendorRelationshipEvidenceScenarios.length);
});

test("Vendor relationship evidence rejects retained portfolio chrome, narrow rails and missing handoffs", () => {
  const scenario = vendorRelationshipEvidenceScenarios[0];
  const labels = ["Overview", "Due diligence", "Evidence", "Issues & actions", "Activity"];
  const capture = { ...scenario, navigation: { activePage: ["Vendors"] }, vendorRelationshipLayout: {
    kind: "overview", evidenceKind: "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION", headingText: "Acme Processing Limited", menuMode: "tabs",
    selectedRelationshipID: "vendor-relationship-payments", selectedRoute: "#vendors/register/vendor-relationship-payments",
    selectedPanelID: "sample-overview-panel", visiblePanelIDs: ["sample-overview-panel"], selectedPanelControlsMatch: true,
    heading: rect(260, 90, 1148, 130), title: rect(340, 130, 600, 40), titleContained: true,
    localTabs: labels, selectedTab: "Overview", tablist: rect(260, 240, 1148, 48), selectedTabBounds: rect(260, 240, 100, 48), selectedTabVisible: true,
    parentHeadingVisible: false, parentMenuVisible: false, registerVisible: false, documentWidth: 1440, scrollWidth: 1440,
    main: rect(260, 305, 815, 1000), aside: rect(1092, 305, 316, 600), placement: "beside",
    firstUsefulContent: rect(260, 305, 815, 200), firstUsefulContentVisible: true,
    apiReads: [{ path: "/api/v1/vendors/vendor-relationship-payments/assessments/current", method: "GET", status: 404 },
      { path: "/api/v1/forms/responses", query: "?subject_type=VENDOR_RELATIONSHIP&subject_id=vendor-relationship-payments&current_only=false&sort=COMPLETED_DESC&limit=25", method: "GET", status: 200 },
      { path: "/api/v1/forms/documents", query: "?relationship_id=vendor-relationship-payments&limit=25&current_only=true", method: "GET", status: 200 }], commands: [],
    settledContent: { pendingVisible: 0 },
    interaction: { visited: labels, keyboard: { complete: true, mode: "tabs", sequence: [...labels.slice(1), labels[0]] }, relationshipPreserved: true, evidenceDocumentsReachable: true, evidenceFormsReachable: true, responseHistoryReachable: true, backRestoredFocus: true, baselineRestored: true },
  } };
  const failuresFor = (value) => reviewVendorRelationshipCaptures([value]).failures.filter((failure) => failure.startsWith(`${scenario.name}:`));
  assert.deepEqual(failuresFor(capture), []);
  const ownerQuery = structuredClone(capture);
  ownerQuery.vendorRelationshipLayout.apiReads.push({ path: "/api/v1/authority/resolve", method: "POST", status: 200,
    requestKind: "AUTHORITY_READ_QUERY", queryInput: { object_type: "THIRD_PARTY_RELATIONSHIP", object_id: "vendor-relationship-payments" } });
  assert.deepEqual(failuresFor(ownerQuery), []);
  ownerQuery.vendorRelationshipLayout.apiReads.at(-1).queryInput.object_id = "foreign-relationship";
  assert.ok(failuresFor(ownerQuery).some((failure) => failure.includes("without sending commands")));
  const parentChrome = structuredClone(capture);
  parentChrome.vendorRelationshipLayout.parentMenuVisible = true;
  assert.ok(failuresFor(parentChrome).some((failure) => failure.includes("replaced portfolio chrome")));
  const narrowRail = structuredClone(capture);
  narrowRail.vendorRelationshipLayout.aside = rect(1092, 305, 270, 600);
  assert.ok(failuresFor(narrowRail).some((failure) => failure.includes("316px rail")));
  const lostForms = structuredClone(capture);
  lostForms.vendorRelationshipLayout.interaction.evidenceFormsReachable = false;
  assert.ok(failuresFor(lostForms).some((failure) => failure.includes("interaction proof")));
  const fabricatedApproval = structuredClone(capture);
  fabricatedApproval.vendorRelationshipLayout.commands = [{ path: "/api/v1/vendors/sample/activate", method: "POST" }];
  assert.ok(failuresFor(fabricatedApproval).some((failure) => failure.includes("without sending commands")));
  const foreignRelationship = structuredClone(capture);
  foreignRelationship.vendorRelationshipLayout.apiReads[0].path = "/api/v1/vendors/foreign-relationship/assessments/current";
  assert.ok(failuresFor(foreignRelationship).some((failure) => failure.includes("without sending commands")));
  const absentDocumentRead = structuredClone(capture);
  absentDocumentRead.vendorRelationshipLayout.apiReads.pop();
  assert.ok(failuresFor(absentDocumentRead).some((failure) => failure.includes("interaction proof")));
  for (const read of [
    { path: "/api/v1/vendors/foreign-relationship", method: "GET", status: 200 },
    { path: "/api/v1/vendor-work", query: "?relationship_id=foreign-relationship&limit=20", method: "GET", status: 200 },
  ]) {
    const foreignOwner = structuredClone(capture);
    foreignOwner.vendorRelationshipLayout.apiReads.push(read);
    assert.ok(failuresFor(foreignOwner).some((failure) => failure.includes("without sending commands")));
  }
  const leakedPanel = structuredClone(capture);
  leakedPanel.vendorRelationshipLayout.visiblePanelIDs.push("sample-evidence-panel");
  assert.ok(failuresFor(leakedPanel).some((failure) => failure.includes("complete local menu")));
  const pendingContent = structuredClone(capture);
  pendingContent.vendorRelationshipLayout.settledContent.pendingVisible = 1;
  assert.ok(failuresFor(pendingContent).some((failure) => failure.includes("settled scoped content")));
  const activityScenario = vendorRelationshipEvidenceScenarios.find((item) => item.vendorRelationshipView === "activity" && item.theme === "light" && item.viewport.width === 1440);
  const activity = { ...structuredClone(capture), ...activityScenario };
  Object.assign(activity.vendorRelationshipLayout, { kind: "activity", selectedTab: "Activity", activityScope: "Response history · Submitted response history for this service.",
    settledContent: { pendingVisible: 0, responseRevisionSummary: "3 submitted revisions loaded" }, interaction: undefined });
  const activityFailures = () => reviewVendorRelationshipCaptures([activity]).failures.filter((failure) => failure.startsWith(`${activity.name}:`));
  assert.deepEqual(activityFailures(), []);
  activity.vendorRelationshipLayout.apiReads = activity.vendorRelationshipLayout.apiReads.filter((read) => read.path !== "/api/v1/forms/responses");
  assert.ok(activityFailures().some((failure) => failure.includes("successful submitted response reads")));

  const supplementalScenario = vendorRelationshipEvidenceScenarios.find((item) => item.vendorRelationshipCheck === "unavailable");
  const supplemental = { ...structuredClone(capture), ...supplementalScenario };
  Object.assign(supplemental.vendorRelationshipLayout, {
    selectionBoundary: { event: "RELATIONSHIP_SELECTION_POINTERDOWN", relationshipID: "vendor-relationship-payments", at: 50, observedAt: "2026-10-22T12:00:00.000Z" },
    preparationReads: [{ path: "/api/v1/vendors/portfolio-other/links", query: "?limit=50", method: "GET", status: 200, phase: "PREPARING", initiatedAt: 10 }],
    interaction: { unavailable: true, retryIssued: true, failureRetained: true },
  });
  supplemental.vendorRelationshipLayout.apiReads = supplemental.vendorRelationshipLayout.apiReads.map((read) => ({ ...read, phase: "SELECTED_RELATIONSHIP", initiatedAt: 60 }));
  const supplementalFailures = (value) => reviewVendorRelationshipCaptures([value]).failures.filter((failure) => failure.startsWith(`${supplemental.name}:`));
  assert.deepEqual(supplementalFailures(supplemental), []);
  const noBoundary = structuredClone(supplemental);
  delete noBoundary.vendorRelationshipLayout.selectionBoundary;
  assert.ok(supplementalFailures(noBoundary).some((failure) => failure.includes("observed selection boundary")));
  const relabelledRead = structuredClone(supplemental);
  relabelledRead.vendorRelationshipLayout.apiReads[0].phase = "PREPARING";
  assert.ok(supplementalFailures(relabelledRead).some((failure) => failure.includes("observed selection boundary")));
  const movedRead = structuredClone(supplemental);
  const latePreparation = movedRead.vendorRelationshipLayout.apiReads.shift();
  movedRead.vendorRelationshipLayout.preparationReads.push({ ...latePreparation, phase: "PREPARING" });
  assert.ok(supplementalFailures(movedRead).some((failure) => failure.includes("observed selection boundary")));
  const foreignSelectedRead = structuredClone(supplemental);
  foreignSelectedRead.vendorRelationshipLayout.apiReads.push({ path: "/api/v1/vendors/foreign-relationship/forms", method: "GET", status: 200, phase: "SELECTED_RELATIONSHIP", initiatedAt: 60 });
  assert.ok(supplementalFailures(foreignSelectedRead).some((failure) => failure.includes("without sending commands")));
  for (const status of [undefined, 503]) {
    const lostAssessment = structuredClone(supplemental);
    if (status === undefined) lostAssessment.vendorRelationshipLayout.apiReads.shift();
    else lostAssessment.vendorRelationshipLayout.apiReads[0].status = status;
    assert.ok(supplementalFailures(lostAssessment).some((failure) => failure.includes("without sending commands")));
  }
  const preparationCommand = structuredClone(supplemental);
  preparationCommand.vendorRelationshipLayout.commands.push({ path: "/api/v1/forms/requests", method: "POST", status: 200, phase: "PREPARING", initiatedAt: 10 });
  assert.ok(supplementalFailures(preparationCommand).some((failure) => failure.includes("without sending commands")));
});
const stableReportTabs = ["Overview", "Explore", "Reports", "Schedules"];
function exploreCapture(state = "report-explore-light-1440") {
  const scenario = reportsDesignEvidenceScenarios.find((item) => item.state === state);
  assert.ok(scenario, state);
  return { ...scenario, reportsLayout: {
    kind: "explore", localTabs: stableReportTabs, selectedTab: "Explore", heading: rect(260, 300, 1152, 50), tabs: rect(260, 245, 1152, 44),
    selectedTabBounds: rect(360, 245, 100, 44), selectedTabVisible: true,
    filters: rect(260, 370, 240, 400), chart: rect(516, 370, 896, 310), table: rect(516, 696, 896, 200),
    firstControl: rect(272, 410, 216, 44), firstUsefulContentVisible: true, firstChartRowVisible: true,
    horizontalOverflow: 0, minTargetWidth: 44, minTargetHeight: 44, placement: "beside", filterWidth: 240,
    measure: "Risks outside appetite", chartLabel: "Risks outside appetite chart", chartState: "live", dataQuality: "complete", exactValuesContained: true,
    total: "4", chartRows: [{ label: "Risk Operations", value: "2" }, { label: "Legal entity", value: "2" }],
    tableRows: [{ label: "Risk Operations", value: "2" }, { label: "Legal entity", value: "2" }],
    organizationScopeID: null, sourceID: "8f710000-0000-4000-8000-000000000001", definitionRevision: "enterprise-domain-v1",
    sourceConsistent: true, timeBasis: "CURRENT_POSTURE", unsupportedControlsDisabled: true,
    coverageText: "12 risks in scope · 0 unknown · 0 excluded.",
    interaction: { stableTabs: stableReportTabs, contextualTabs: [...stableReportTabs, "Build report"],
      arrowSequence: ["Explore", "Reports", "Schedules", "Overview", "Explore"], measureKeyboardSelection: true,
      selectedMeasurePreserved: true, hiddenBuilderRemoved: true, baselineRestored: true },
  } };
}

test("Reports screenshots require measured anatomy and preserved contextual-tab input", () => {
  const capture = exploreCapture();
  const ownFailures = (value) => reviewReportsDesignCaptures([value]).failures.filter((failure) => failure.startsWith(`${capture.name}:`));
  assert.deepEqual(ownFailures(capture), []);
  assert.ok(ownFailures({ ...capture, reportsLayout: undefined }).some((failure) => failure.includes("measured first content")));
  const lostSelection = structuredClone(capture);
  lostSelection.reportsLayout.interaction.selectedMeasurePreserved = false;
  assert.ok(ownFailures(lostSelection).some((failure) => failure.includes("retained input evidence")));
  const firstControlBelowFold = structuredClone(capture);
  firstControlBelowFold.reportsLayout.firstControl = rect(272, 910, 216, 44);
  assert.ok(ownFailures(firstControlBelowFold).some((failure) => failure.includes("measured first content")));
  const selectedTabClipped = structuredClone(capture);
  selectedTabClipped.reportsLayout.selectedTabBounds = rect(1350, 245, 100, 44);
  assert.ok(ownFailures(selectedTabClipped).some((failure) => failure.includes("measured first content")));
});

test("Reports exact breakdown and retained source cannot be replaced by a matching total alone", () => {
  const capture = exploreCapture();
  const diagnostic = `${capture.name}: Explore chosen measure, exact values, source or unavailable semantics are missing or inconsistent`;
  const differentBreakdown = structuredClone(capture);
  differentBreakdown.reportsLayout.tableRows = [{ label: "Risk Operations", value: "4" }];
  assert.ok(reviewReportsDesignCaptures([differentBreakdown]).failures.includes(diagnostic));
  const foreignSource = structuredClone(capture);
  foreignSource.reportsLayout.sourceID = "foreign-source";
  assert.ok(reviewReportsDesignCaptures([foreignSource]).failures.includes(diagnostic));
  const unavailable = exploreCapture("report-explore-unavailable");
  unavailable.reportsLayout = { ...unavailable.reportsLayout, chartState: "unavailable", chartRows: [], tableRows: [], table: null, total: null,
    sourceConsistent: false, sourceID: null, definitionRevision: null };
  assert.ok(!reviewReportsDesignCaptures([unavailable]).failures.some((failure) => failure.startsWith(`${unavailable.name}:`)));
  unavailable.reportsLayout.total = "0";
  assert.ok(reviewReportsDesignCaptures([unavailable]).failures.some((failure) => failure.startsWith(`${unavailable.name}:`) && failure.includes("unavailable semantics")));
});

test("the 768px builder follows the rendered source Output-left and Setup-right replacement", () => {
  const scenario = reportsDesignEvidenceScenarios.find((item) => item.state === "report-builder-dark-768");
  const capture = { ...scenario, reportsLayout: {
    kind: "builder", localTabs: [...stableReportTabs, "Build report"], selectedTab: "Build report",
    tabs: rect(92, 245, 656, 44), heading: rect(92, 300, 656, 40), firstControl: rect(440, 400, 296, 44),
    selectedTabBounds: rect(600, 245, 120, 44), selectedTabVisible: true,
    firstUsefulContentVisible: true, horizontalOverflow: 0, minTargetWidth: 44, minTargetHeight: 44,
    setup: rect(428, 355, 320, 480), output: rect(92, 355, 320, 540), paper: rect(92, 911, 656, 650),
    placement: "controls-above", controlsOrder: "output-setup", setupWidth: 320,
    previewKind: "SETUP_CONTENTS_ONLY", unsaved: true, manualDeliveryOnly: true, sectionCount: 2, renderedDataRows: 0,
    previewTitle: "New report", contents: ["Summary", "Data", "Report information"],
  } };
  const diagnostic = `${capture.name}: Builder setup, preview and output lack source replacement geometry`;
  assert.ok(!reviewReportsDesignCaptures([capture]).failures.some((failure) => failure.startsWith(`${capture.name}:`)));
  const swapped = structuredClone(capture);
  [swapped.reportsLayout.setup, swapped.reportsLayout.output] = [swapped.reportsLayout.output, swapped.reportsLayout.setup];
  assert.ok(reviewReportsDesignCaptures([swapped]).failures.includes(diagnostic));
  const fakePreview = structuredClone(capture);
  fakePreview.reportsLayout.renderedDataRows = 12;
  assert.ok(reviewReportsDesignCaptures([fakePreview]).failures.some((failure) => failure.includes("without fabricated generated content")));
});

test("builder workflow claims require retained request and response versions, including a real rejected review", () => {
  const scenario = reportsDesignEvidenceScenarios.find((item) => item.state === "report-builder-blocked");
  const checksum = "a".repeat(64);
  const definition = { id: "sample-report-builder-1", name: "Sample Program exceptions", version: 1, current_version: 1, status: "DRAFT", checksum };
  const createBody = { name: definition.name, dataset: "PROGRAMS", format: "XLSX", scope_kind: "LEGAL_ENTITY",
    filter: { kind: "group", operator: "or", children: ["AT_RISK", "GAP_IDENTIFIED", "EVIDENCE_INSUFFICIENT", "IMPLEMENTATION_PENDING", "OVERDUE", "UNDER_REVIEW", "UNKNOWN"]
      .map((value) => ({ kind: "condition", field: "overall_state", operator: "is", value })) } };
  const capture = { ...scenario, reportsLayout: { kind: "builder-flow", flow: {
    evidenceMode: "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION", definitionID: definition.id, definitionName: definition.name,
    dataset: "PROGRAMS", format: "XLSX", scopeKind: "LEGAL_ENTITY", saveCount: 1,
    actions: ["create", "submit", "review"], commandStatuses: [201, 200, 403], checksumRetained: true,
    versionChecksMatched: true, savedDefinitionRetained: true, finalStatus: "PENDING_REVIEW", blockedReviewPreservedSetup: true, runID: null,
    commands: [
      { method: "POST", path: "/api/v1/reports/definitions", body: createBody, status: 201, result: definition },
      { method: "POST", path: `/api/v1/reports/definitions/${definition.id}/submit`, body: { expected_version: 1, checksum_seen: checksum }, status: 200,
        result: { ...definition, version: 2, status: "PENDING_REVIEW" } },
      { method: "POST", path: `/api/v1/reports/definitions/${definition.id}/review`, body: { expected_version: 2, checksum_seen: checksum }, status: 403,
        result: { error: { code: "report_governance_blocked" } } },
    ],
  } } };
  const diagnostic = `${capture.name}: real setup mutation, governance rejection or generation handoff evidence is missing or inconsistent`;
  assert.ok(!reviewReportsDesignCaptures([capture]).failures.includes(diagnostic));
  const staleVersion = structuredClone(capture);
  staleVersion.reportsLayout.flow.commands[2].body.expected_version = 1;
  assert.ok(reviewReportsDesignCaptures([staleVersion]).failures.includes(diagnostic));
  const unproven = structuredClone(capture);
  delete unproven.reportsLayout.flow.commands;
  assert.ok(reviewReportsDesignCaptures([unproven]).failures.includes(diagnostic));
  const overclaim = structuredClone(capture);
  overclaim.reportsLayout.flow.evidenceMode = "BACKEND_AUTHORIZED";
  assert.ok(reviewReportsDesignCaptures([overclaim]).failures.includes(diagnostic));
});

function notificationReceipt(scenario) {
  const box = (x, y, width, height) => ({ x, y, width, height, right: x + width, bottom: y + height });
  return { ...scenario, navigation: { activePage: ["Work & Approvals"] }, notificationsLayout: {
    kind: scenario.notificationView, evidenceKind: "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION",
    headingText: "Notifications", heading: box(252, 110, 1160, 120), tablist: box(252, 250, 1160, 50),
    localTabs: ["Inbox", "Activity", "Preferences"], selectedTab: "Inbox", selectedTabBounds: box(260, 252, 90, 44), selectedTabVisible: true,
    firstUsefulContent: box(268, 480, 640, 100), firstUsefulContentVisible: true,
    documentWidth: 1440, scrollWidth: 1440, unsupportedControls: 0, minTargetWidth: 44, minTargetHeight: 44, undersizedTargets: [],
    main: box(252, 370, 700, 700), aside: box(968, 370, 440, 700), placement: "beside",
    recordCount: 5, recordHeights: [100, 100, 100, 100, 100], selectedCount: 1,
    selectedRecord: "Review the Sample annual-return evidence change", detailTitle: "Review the Sample annual-return evidence change",
    unreadHeader: "Notifications, 7 unread", readFilter: [{ label: "All", selected: true }, { label: "Unread (7)", selected: false }],
    searchVisible: true, filtersToggleVisible: false, commands: [], apiReads: [{ path: "/api/v1/notifications", query: "?limit=25&unread_only=false", status: 200 }],
  } };
}
function notificationFindings(capture) { return reviewNotificationsCaptures([capture]).failures.filter((failure) => failure.startsWith(`${capture.name}:`)); }

test("all three Notifications views require source-width/theme evidence and focused states", () => {
  assert.equal(notificationsEvidenceScenarios.length, 42);
  assert.equal(new Set(notificationsEvidenceScenarios.map((scenario) => scenario.name)).size, 42);
  assert.equal(notificationsEvidenceScenarios.filter((scenario) => scenario.notificationMatrix).length, 30);
  for (const view of ["inbox", "activity", "preferences"]) {
    const matrix = notificationsEvidenceScenarios.filter((scenario) => scenario.notificationMatrix && scenario.notificationView === view);
    assert.equal(matrix.length, 10);
    assert.deepEqual(new Set(matrix.map((scenario) => scenario.viewport.width)), new Set([1440, 1280, 1024, 768, 390]));
  }
  assert.equal(reviewNotificationsCaptures([]).failures.length, 42);
});

test("Notifications rejects clipped selected tabs, reversed context and inferred unread counts", () => {
  const scenario = notificationsEvidenceScenarios.find((item) => item.notificationView === "inbox" && item.notificationMatrix && item.theme === "dark" && item.viewport.width === 1440);
  const valid = notificationReceipt(scenario);
  assert.deepEqual(notificationFindings(valid), []);
  const clipped = structuredClone(valid);
  clipped.notificationsLayout.selectedTabBounds.x = 1400;
  clipped.notificationsLayout.selectedTabBounds.right = 1490;
  assert.ok(notificationFindings(clipped).some((failure) => failure.includes("local menu")));
  const reversed = structuredClone(valid);
  [reversed.notificationsLayout.main, reversed.notificationsLayout.aside] = [reversed.notificationsLayout.aside, reversed.notificationsLayout.main];
  assert.ok(notificationFindings(reversed).some((failure) => failure.includes("replacement geometry")));
  const inferred = structuredClone(valid);
  inferred.notificationsLayout.unreadHeader = "Notifications, 3 unread";
  assert.ok(notificationFindings(inferred).some((failure) => failure.includes("server unread total")));
  const viewportClaim = structuredClone(valid);
  viewportClaim.notificationsLayout.firstUsefulContent.y = 910;
  viewportClaim.notificationsLayout.firstUsefulContent.bottom = 1010;
  assert.ok(notificationFindings(viewportClaim).some((failure) => failure.includes("first useful content")));
});

test("Activity denial evidence cannot hide a protected query or claim backend authorization", () => {
  const scenario = notificationsEvidenceScenarios.find((item) => item.notificationCheck === "denied");
  const valid = notificationReceipt(scenario);
  Object.assign(valid.notificationsLayout, { selectedTab: "Activity", main: null, aside: null, placement: "restricted", recordCount: 0,
    interaction: { restricted: true, protectedRequests: 0, inboxAvailable: true, permission: "platform_operations_read",
      stateCapture: { visible: true, bounds: structuredClone(valid.notificationsLayout.firstUsefulContent) } } });
  assert.deepEqual(notificationFindings(valid), []);
  const request = structuredClone(valid);
  request.notificationsLayout.apiReads.push({ path: "/api/v1/system-activity", query: "?legal_entity_id=bank-ng&limit=25&from=2026-09-21T12%3A00%3A00Z&to=2026-10-21T12%3A00%3A00Z", status: 403 });
  assert.ok(notificationFindings(request).some((failure) => failure.includes("Activity permission")));
  const overclaim = structuredClone(valid);
  overclaim.notificationsLayout.evidenceKind = "BACKEND_AUTHORIZED";
  assert.ok(notificationFindings(overclaim).some((failure) => failure.includes("fixture identity")));
  const hidden = structuredClone(valid);
  hidden.notificationsLayout.interaction.stateCapture.bounds.y = 910;
  hidden.notificationsLayout.interaction.stateCapture.bounds.bottom = 1010;
  assert.ok(notificationFindings(hidden).some((failure) => failure.includes("asserted state must be visible")));
});

test("Activity evidence preserves returned facts and discloses version limits once in context", () => {
  const scenario = notificationsEvidenceScenarios.find((item) => item.notificationMatrix && item.notificationView === "activity" && item.theme === "dark" && item.viewport.width === 1440);
  const valid = notificationReceipt(scenario);
  Object.assign(valid.notificationsLayout, { selectedTab: "Activity", recordCount: 3,
    activityVersionLimitation: "Record versions are not recorded in this history. Check the related record for its current version and available actions.",
    activityEvents: [
      { id: "sample-activity-event-1", action: "Assigned Sample annual-return evidence review", actor: "Sample Amina Yusuf", outcome: "Succeeded", fields: ["Record"], text: "Record matter-gaid-change · Workflow" },
      { id: "sample-activity-event-2", action: "Recorded Sample Program review", actor: "Sample Emeka Okoro", outcome: "Succeeded", fields: ["Record"], text: "Record program-ndpa · Programs" },
      { id: "sample-activity-event-3", action: "Denied Sample approval request", actor: "Sample Ruth Mensah", outcome: "Denied", fields: ["Record"], text: "Record sample-restricted-change · Authority" },
    ] });
  valid.notificationsLayout.apiReads.push({ path: "/api/v1/system-activity", query: "?legal_entity_id=bank-ng&limit=25&from=2026-09-21T12%3A00%3A00Z&to=2026-10-21T12%3A00%3A00Z", status: 200 });
  assert.deepEqual(notificationFindings(valid), []);
  for (const field of ["Scope", "Version"]) {
    const duplicated = structuredClone(valid);
    duplicated.notificationsLayout.activityEvents[0].fields.push(field);
    assert.ok(notificationFindings(duplicated).some((failure) => failure.includes("context-only version limitation")));
  }
  const absent = structuredClone(valid);
  delete absent.notificationsLayout.activityVersionLimitation;
  assert.ok(notificationFindings(absent).some((failure) => failure.includes("context-only version limitation")));
  const inferred = structuredClone(valid);
  inferred.notificationsLayout.activityEvents[2].outcome = "Succeeded";
  assert.ok(notificationFindings(inferred).some((failure) => failure.includes("Activity returned record")));
});

test("Notifications older-page failure evidence requires the notice inside the captured viewport", () => {
  const scenario = notificationsEvidenceScenarios.find((item) => item.notificationCheck === "more-failed");
  const valid = notificationReceipt(scenario);
  valid.notificationsLayout.interaction = { retainedRows: 5, olderPageFailure: true,
    cursor: "sample-notifications/older?batch=2&recipient=sample", failureVisible: true,
    failureBounds: { x: 268, y: 600, width: 640, height: 60, right: 908, bottom: 660 } };
  assert.deepEqual(notificationFindings(valid), []);
  const hidden = structuredClone(valid);
  hidden.notificationsLayout.interaction.failureBounds.y = 910;
  hidden.notificationsLayout.interaction.failureBounds.bottom = 970;
  assert.ok(notificationFindings(hidden).some((failure) => failure.includes("show its failure notice")));
  const unmeasured = structuredClone(valid);
  delete unmeasured.notificationsLayout.interaction.failureBounds;
  assert.ok(notificationFindings(unmeasured).some((failure) => failure.includes("show its failure notice")));
});

test("Notifications preference evidence requires minutes, draft retention and expected version", () => {
  const scenario = notificationsEvidenceScenarios.find((item) => item.notificationMatrix && item.notificationView === "preferences" && item.theme === "light" && item.viewport.width === 1440);
  const valid = notificationReceipt(scenario);
  const command = { method: "PUT", path: "/api/v1/preferences/notifications", status: 200,
    body: { daily_digest_enabled: true, digest_minute: 585, time_zone: "Africa/Lagos", quiet_hours_enabled: true, quiet_start_minute: 1330, quiet_end_minute: 380, expected_version: 3 },
    result: { version: 4, critical_email_required: true } };
  Object.assign(valid.notificationsLayout, { selectedTab: "Preferences", preferences: { criticalRequired: true, inAppHeading: "In-app notifications", emailHeading: "Email delivery" },
    commands: [command], interaction: { navigation: { arrowSequence: ["Activity", "Preferences", "Inbox"], routesMatched: true, count: 3 }, baselineRestored: true,
      draftRetained: true, saved: true, conflictRetainedDraft: false, criticalRequired: true, inputMinutes: [585, 1330, 380], command } });
  assert.deepEqual(notificationFindings(valid), []);
  for (const value of [undefined, 2]) {
    const stale = structuredClone(valid);
    stale.notificationsLayout.commands[0].body.expected_version = value;
    assert.ok(notificationFindings(stale).some((failure) => failure.includes("versioned command")));
  }
  const lostDraft = structuredClone(valid);
  lostDraft.notificationsLayout.interaction.draftRetained = false;
  assert.ok(notificationFindings(lostDraft).some((failure) => failure.includes("versioned command")));
  const identity = structuredClone(valid);
  identity.notificationsLayout.commands[0].body.actor_id = "supplied-actor";
  assert.ok(notificationFindings(identity).some((failure) => failure.includes("versioned command")));
});

// Fabricated receipts exercise rejection rules only; browser captures supply proof.
function riskLossReceipt(scenario) {
  const bounds = (x, y, width, height) => ({ x, y, width, height, right: x + width, bottom: y + height });
  const titles = [scenario.firstRecord], ids = [scenario.detailID];
  const table = bounds(80, 400, scenario.viewport.width - 100, 200);
  const row = scenario.riskLossView === "rcsa" ? { title: scenario.firstRecord, facts: { Risks: "3", Controls: "2" } }
    : scenario.riskLossView === "insights" ? { "Current value": "98.70% 100 / 100 concern", "Limit / target": "Limit ≥ 99.50%", Movement: "−0.40 pp", Condition: "Breach" }
    : scenario.riskLossView === "losses" ? { Gross: "₦18,400,000.00", Recovered: "₦8,000,000.00 Partly recovered", Net: "₦10,400,000.00 Active" }
    : { Risk: scenario.firstRecord, Appetite: "Outside appetite", Rating: "High", Status: "Active" };
  const riskLossLayout = { kind: scenario.riskLossView, route: scenario.route, evidenceKind: "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION",
    headingText: "Risk & Losses", heading: bounds(80, 180, scenario.viewport.width - 100, 100), menu: bounds(80, 300, scenario.viewport.width - 100, 49),
    localTabs: riskLossViews.map((view) => view.label), selectedTab: scenario.label, selectedTabVisible: true, menuRows: 1,
    primaryCurrent: ["Risk & Losses"], documentWidth: scenario.viewport.width, scrollWidth: scenario.viewport.width,
    records: 1, recordTitles: titles, populatedIDs: ids, firstRecord: table, firstIdentity: bounds(90, 410, 180, 40),
    firstUsefulContentVisible: true, table, grid: scenario.riskLossView === "rcsa" ? table : null,
    firstRecordColumns: scenario.viewport.width === 390 ? 2 : 0, rowActions: [{ label: "Open record", bounds: bounds(100, 460, 120, 44), lines: 1 }],
    statusBadges: ({ insights: ["Condition"], risks: ["Rating", "Appetite", "Status"] }[scenario.riskLossView] ?? []).map((column) => ({
      label: row[column], rowIndex: 0, column, lines: 1, words: row[column].split(/\s+/).map((text) => ({ text, lines: 1 })),
      bounds: bounds(100, 410, 68, 26), cell: bounds(90, 400, 90, 50), textBounds: bounds(110, 415, 45, 16) })),
    cardNumbers: [{ label: "Risks", value: "3", fontSize: 34, labelFontSize: 12, aligned: true, textAlign: "start" }, { label: "Controls", value: "2", fontSize: 34, labelFontSize: 12, aligned: true, textAlign: "start" }],
    presentation: scenario.riskLossView === "rcsa" ? "cards" : scenario.viewport.width === 390 ? "stacked-records" : "table",
    columns: scenario.viewport.width > 1190 ? 3 : scenario.viewport.width > 600 ? 2 : 1,
    tableHeaders: scenario.riskLossView === "losses" ? ["Loss", "Gross", "Recovered", "Net"] : [], rowValues: [row],
    stateText: "", disclosures: [], dialogVisible: false, pendingReads: 0, commands: [] };
  const observation = { pending: 0, commands: [], reads: [{ path: { risks: "/api/v1/risks", rcsa: "/api/v1/rcsa/cycles", insights: "/api/v1/risk-indicators", losses: "/api/v1/losses" }[scenario.riskLossView],
    method: "GET", status: 200, query: { limit: scenario.riskLossView === "insights" ? "50" : "25" } }], responses: [{ ids, titles }] };
  const capture = { ...scenario, baseline: false, imageSHA256: "a".repeat(64), imageBytes: 50000, riskLossLayout, observation };
  if (scenario.verifyRiskLossNavigation) capture.navigation = { visited: riskLossViews.map((view) => ({ label: view.label, route: view.route })),
    peerHistory: [
      ["back", "#losses", "#insights?kind=KRI", "Key indicators"],
      ["back", "#insights?kind=KRI", "#rcsa", "RCSA"],
      ["back", "#rcsa", "#risks", "Risks"],
      ["forward", "#risks", "#rcsa", "RCSA"],
      ["forward", "#rcsa", "#insights?kind=KRI", "Key indicators"],
      ["forward", "#insights?kind=KRI", "#losses", "Loss events"],
    ].map(([direction, from, to, label]) => ({ direction, from, to, label, sameTablist: true, sameDocument: true, selectedTabVisible: true,
      menu: structuredClone(riskLossLayout.menu), selected: bounds(90, 302, 120, 44), documentWidth: scenario.viewport.width, scrollWidth: scenario.viewport.width })),
    keyboard: ["RCSA", "Key indicators", "Loss events", "Risks"], restoredRoute: scenario.route, restoredPopulation: ids, selectedTabVisible: true, commands: [] };
  if (scenario.riskLossCheck === "history") capture.history = { actualBack: true, sameDocument: true, noSelectedDetail: true, backRoute: scenario.route,
    selectedMenu: scenario.label, pendingReads: 0, initialDocumentToken: "same-document", backDocumentToken: "same-document", forwardDocumentToken: "same-document",
    populationBefore: ids, populationAfter: ids, backLayout: structuredClone(riskLossLayout), forwardExactTarget: true,
    forwardRoute: `#${scenario.riskLossView}/${scenario.detailID}`, explicitReturn: true, commands: [] };
  if (scenario.riskLossCheck === "detail") capture.detail = { route: scenario.riskLossView === "insights" ? `#insights?kind=KRI&indicator=${scenario.detailID}` : `#${scenario.riskLossView}/${scenario.detailID}`,
    exactTarget: true, titleMatches: true, titleVisible: true, pendingReads: 0, commands: [], placement: "sheet", width: Math.min(480, scenario.viewport.width),
    x: Math.max(0, scenario.viewport.width - 480), height: scenario.viewport.height, internalTables: [{ clientWidth: 438, scrollWidth: 438, presentation: "block" }] };
  if (capture.detail) Object.assign(capture.detail, { closeVisible: true, focusContained: true, close: bounds(scenario.viewport.width - 64, 20, 44, 44), primaryAction: { visible: true } });
  if (capture.detail && scenario.riskLossView === "rcsa") capture.detail.rcsaPopulation = { card: { Risks: "3", Controls: "2" }, detailRows: { Risks: 3, Controls: 2 }, detailSummary: "3 Risks · 2 controls" };
  if (scenario.riskLossCheck === "filters") {
    const query = { ...observation.reads[0].query, ...(scenario.riskLossView === "insights" ? { kind: "KRI" } : {}) };
    const filter = { risks: { search: "network" }, losses: { search: "merchant" }, rcsa: { status: "AWAITING_CHALLENGE" }, insights: { kind: "KCI" } }[scenario.riskLossView];
    const before = { ids, titles, query };
    capture.filters = { nextPageDistinct: true, previousPageRestored: true, filterSentToReader: true, phoneFiltersAccessible: true, commands: [],
      pages: { before, next: { ids: ["next-record"], titles: ["Next record"], query: { ...query, cursor: "sample-risk-loss-next-page" } },
        previous: structuredClone(before), filtered: { ...structuredClone(before), query: { ...query, ...filter } }, restored: structuredClone(before) } };
    observation.reads.push({ ...observation.reads[0], query: capture.filters.pages.next.query }, { ...observation.reads[0], query: capture.filters.pages.filtered.query });
  }
  return capture;
}
const riskLossFindings = (capture) => reviewRiskLossCaptures([capture]).failures.filter((message) => message.startsWith(`${capture.name}:`));

test("Risk & Losses evidence requires all route states and cannot accept a baseline receipt", () => {
  assert.equal(reviewRiskLossCaptures([]).failures.length, riskLossEvidenceScenarios.length);
  const capture = riskLossReceipt(riskLossEvidenceScenarios[0]);
  assert.deepEqual(riskLossFindings(capture), []);
  capture.baseline = true;
  assert.ok(riskLossFindings(capture).some((message) => message.includes("synthetic evidence boundary")));
});

test("Risk & Losses rejects clipped menus, off-screen first identity and disconnected response populations", () => {
  const valid = riskLossReceipt(riskLossEvidenceScenarios[0]);
  for (const change of [
    (layout) => { layout.menuRows = 2; },
    (layout) => { layout.selectedTabVisible = false; },
    (layout) => { layout.firstIdentity.y = 950; layout.firstIdentity.bottom = 990; },
    (layout) => { layout.populatedIDs = ["foreign-risk"]; },
    (layout) => { layout.recordTitles = ["Invented title"]; },
    (layout) => { layout.rowActions[0].lines = 14; },
  ]) {
    const invalid = structuredClone(valid); change(invalid.riskLossLayout);
    assert.ok(riskLossFindings(invalid).length > 0);
  }
});

test("Risk & Losses peer history requires all six ordered Back/Forward destinations", () => {
  for (const scenario of riskLossEvidenceScenarios.filter((item) => item.verifyRiskLossNavigation)) {
    const valid = riskLossReceipt(scenario);
    assert.deepEqual(riskLossFindings(valid), [], scenario.name);
    for (const [reason, change] of [
      ["missing history", (navigation) => { delete navigation.peerHistory; }],
      ["short history", (navigation) => { navigation.peerHistory.pop(); }],
      ["extra history", (navigation) => { navigation.peerHistory.push(structuredClone(navigation.peerHistory[0])); }],
      ["wrong order", (navigation) => { [navigation.peerHistory[0], navigation.peerHistory[1]] = [navigation.peerHistory[1], navigation.peerHistory[0]]; }],
      ["wrong direction", (navigation) => { navigation.peerHistory[0].direction = "forward"; }],
      ["wrong origin", (navigation) => { navigation.peerHistory[0].from = "#losses/another-record"; }],
      ["wrong destination", (navigation) => { navigation.peerHistory[0].to = "#reports"; }],
      ["wrong label", (navigation) => { navigation.peerHistory[0].label = "Loss events"; }],
    ]) {
      const invalid = structuredClone(valid); change(invalid.navigation);
      assert.ok(riskLossFindings(invalid).some((message) => message.includes("peer Back/Forward")), `${scenario.name}: ${reason}`);
    }
  }
});

test("Risk & Losses peer history rejects clipped measured tabs despite visible=true", () => {
  for (const scenario of riskLossEvidenceScenarios.filter((item) => item.verifyRiskLossNavigation)) {
    const valid = riskLossReceipt(scenario);
    assert.deepEqual(riskLossFindings(valid), [], scenario.name);
    for (const [reason, change] of [
      ["missing tab bounds", (step) => { delete step.selected; }],
      ["left clipping", (step) => { step.selected = rect(step.menu.x - 2, step.selected.y, step.selected.width, step.selected.height); }],
      ["right clipping", (step) => { step.selected = rect(step.menu.right - 10, step.selected.y, step.selected.width, step.selected.height); }],
      ["vertical clipping", (step) => { step.selected = rect(step.selected.x, step.menu.bottom - 10, step.selected.width, step.selected.height); }],
      ["off-screen menu", (step) => { step.menu = rect(-2, step.menu.y, step.menu.width, step.menu.height); }],
      ["off-screen vertical bounds", (step) => { step.menu = rect(step.menu.x, scenario.viewport.height - 10, step.menu.width, step.menu.height); step.selected = rect(step.selected.x, step.menu.y + 2, step.selected.width, step.selected.height); }],
      ["inconsistent rectangle", (step) => { step.selected.right += 5; }],
      ["missing width measurement", (step) => { delete step.selected.width; }],
      ["hidden tab", (step) => { step.selectedTabVisible = false; }],
    ]) {
      const invalid = structuredClone(valid); change(invalid.navigation.peerHistory[2]);
      assert.ok(riskLossFindings(invalid).some((message) => message.includes("peer Back/Forward")), `${scenario.name}: ${reason}`);
    }
  }
});

test("Risk & Losses peer history requires the same menu and document without overflow", () => {
  for (const scenario of riskLossEvidenceScenarios.filter((item) => item.verifyRiskLossNavigation)) {
    const valid = riskLossReceipt(scenario);
    assert.deepEqual(riskLossFindings(valid), [], scenario.name);
    for (const [reason, change] of [
      ["disconnected menu", (step) => { step.sameTablist = false; }],
      ["new document", (step) => { step.sameDocument = false; }],
      ["missing identity proof", (step) => { delete step.sameTablist; }],
      ["wrong viewport", (step) => { step.documentWidth += 20; }],
      ["document overflow", (step) => { step.scrollWidth += 2; }],
      ["missing overflow measurement", (step) => { delete step.scrollWidth; }],
    ]) {
      const invalid = structuredClone(valid); change(invalid.navigation.peerHistory[5]);
      assert.ok(riskLossFindings(invalid).some((message) => message.includes("peer Back/Forward")), `${scenario.name}: ${reason}`);
    }
  }
});

test("Risk & Losses paging requires rendered populations rather than success flags alone", () => {
  for (const scenario of riskLossEvidenceScenarios.filter((item) => item.riskLossCheck === "filters")) {
    const valid = riskLossReceipt(scenario);
    assert.deepEqual(riskLossFindings(valid), []);
    for (const change of [
      (filters) => { delete filters.pages; },
      (filters) => { filters.pages.next.ids = [...filters.pages.before.ids]; },
      (filters) => { filters.pages.previous.ids = ["stale-page-record"]; },
      (filters) => { filters.pages.previous.titles = ["Stale page title"]; },
      (filters) => { filters.pages.restored.ids = ["filtered-page-record"]; },
      (filters) => { filters.pages.filtered.query.cursor = "sample-risk-loss-next-page"; },
      (filters) => { filters.pages.filtered.query = {}; },
    ]) {
      const invalid = structuredClone(valid); change(invalid.filters);
      assert.ok(riskLossFindings(invalid).some((message) => message.includes("real cursor paging")), scenario.name);
    }
  }
});

test("Risk & Losses history requires same-document settled Back and exact Forward rather than returned=true", () => {
  const scenario = riskLossEvidenceScenarios.find((item) => item.riskLossView === "rcsa" && item.riskLossCheck === "history");
  const valid = riskLossReceipt(scenario);
  assert.deepEqual(riskLossFindings(valid), []);
  for (const change of [
    (history) => { delete history.backLayout; history.returned = true; },
    (history) => { history.backDocumentToken = "new-document"; },
    (history) => { history.backLayout.dialogVisible = true; },
    (history) => { history.backLayout.pendingReads = 1; },
    (history) => { history.backLayout.populatedIDs = []; },
    (history) => { history.forwardRoute = "#rcsa/another-cycle"; },
  ]) {
    const invalid = structuredClone(valid); change(invalid.history);
    assert.ok(riskLossFindings(invalid).some((message) => message.includes("real Back/Forward")));
  }
});

test("RCSA cards retain frozen counts and the source responsive column structure", () => {
  const scenario = riskLossEvidenceScenarios.find((item) => item.riskLossView === "rcsa" && item.riskLossMatrix && item.viewport.width === 1440);
  const valid = riskLossReceipt(scenario);
  assert.deepEqual(riskLossFindings(valid), []);
  for (const change of [(layout) => { layout.presentation = "table"; }, (layout) => { layout.columns = 2; }, (layout) => { layout.rowValues[0].facts.Risks = "83% reviewed"; }, (layout) => { layout.cardNumbers[0].fontSize = 14; }]) {
    const invalid = structuredClone(valid); change(invalid.riskLossLayout);
    assert.ok(riskLossFindings(invalid).some((message) => message.includes("RCSA source card columns")));
  }
});

test("Risk & Losses sheets require the source 480px inspector and bounded internal table replacement", () => {
  const scenario = riskLossEvidenceScenarios.find((item) => item.riskLossView === "rcsa" && item.riskLossCheck === "detail" && item.viewport.width === 1440);
  const valid = riskLossReceipt(scenario);
  assert.deepEqual(riskLossFindings(valid), []);
  for (const change of [(detail) => { detail.width = 880; detail.x = 560; }, (detail) => { detail.internalTables[0].presentation = "table"; }, (detail) => { detail.internalTables[0].scrollWidth = 650; }]) {
    const invalid = structuredClone(valid); change(invalid.detail);
    assert.ok(riskLossFindings(invalid).some((message) => message.includes("source 480px inspector")));
  }
});

test("indicator receipts cannot replace native percentages or percentage points with concern scores", () => {
  const scenario = riskLossEvidenceScenarios.find((item) => item.riskLossView === "insights" && item.riskLossMatrix);
  const valid = riskLossReceipt(scenario);
  assert.deepEqual(riskLossFindings(valid), []);
  for (const [field, value] of [["Current value", "100%"], ["Limit / target", "100"], ["Movement", "−0.40%"]]) {
    const invalid = structuredClone(valid); invalid.riskLossLayout.rowValues[0][field] = value;
    assert.ok(riskLossFindings(invalid).some((message) => message.includes("native percentage")));
  }
});

test("RCSA inspector counts must reconcile with the selected card and rendered detail rows", () => {
  for (const scenario of riskLossEvidenceScenarios.filter((item) => item.riskLossView === "rcsa" && item.riskLossCheck === "detail")) {
    const valid = riskLossReceipt(scenario);
    assert.deepEqual(riskLossFindings(valid), []);
    for (const change of [
      (detail) => { delete detail.rcsaPopulation; },
      (detail) => { detail.rcsaPopulation.card.Controls = "4"; },
      (detail) => { detail.rcsaPopulation.detailRows.Controls = 1; },
      (detail) => { detail.rcsaPopulation.detailSummary = "3 Risks · 4 controls"; },
    ]) {
      const invalid = structuredClone(valid); change(invalid.detail);
      assert.ok(riskLossFindings(invalid).some((message) => message.includes("RCSA inspector frozen population")), scenario.name);
    }
  }
});

test("Risk & Losses rejects wrapped or clipped one-word indicator condition badges", () => {
  for (const scenario of riskLossEvidenceScenarios.filter((item) => item.riskLossView === "insights" && item.riskLossMatrix && [1280, 1024].includes(item.viewport.width))) {
    const valid = riskLossReceipt(scenario);
    assert.deepEqual(riskLossFindings(valid), []);
    for (const change of [
      (layout) => { delete layout.statusBadges; },
      (layout) => { layout.statusBadges[0].lines = 2; },
      (layout) => { layout.statusBadges[0].lines = 0; },
      (layout) => { layout.statusBadges[0].label = "Different condition"; },
      (layout) => { const badge = layout.statusBadges[0]; badge.textBounds.right = badge.bounds.right + 2; badge.textBounds.width = badge.textBounds.right - badge.textBounds.x; },
      (layout) => { const badge = layout.statusBadges[0]; badge.cell.right = badge.bounds.right - 2; badge.cell.width = badge.cell.right - badge.cell.x; },
    ]) {
      const invalid = structuredClone(valid); change(invalid.riskLossLayout);
      assert.ok(riskLossFindings(invalid).some((message) => message.includes("status badges")), scenario.name);
    }
  }
});

test("Risk & Losses allows whole-word Risk label wrapping but rejects a split word in a status badge", () => {
  const scenario = riskLossEvidenceScenarios.find((item) => item.riskLossView === "risks" && item.riskLossCheck === "partial");
  const valid = riskLossReceipt(scenario);
  valid.observation.responses[0].nextCursor = "sample-risk-loss-next-page";
  for (const column of ["Rating", "Appetite"]) {
    valid.riskLossLayout.rowValues[0][column] = "Reassessment needed";
    const badge = valid.riskLossLayout.statusBadges.find((item) => item.column === column);
    Object.assign(badge, { label: "Reassessment needed", lines: 2, words: [{ text: "Reassessment", lines: 1 }, { text: "needed", lines: 1 }] });
  }
  assert.deepEqual(riskLossFindings(valid), []);
  // The named visual review accepts this contained three-word label at 1024px.
  const threeWordScenario = riskLossEvidenceScenarios.find((item) => item.riskLossView === "risks" && item.riskLossMatrix && item.viewport.width === 1024);
  const threeWords = riskLossReceipt(threeWordScenario);
  threeWords.riskLossLayout.rowValues[0].Appetite = "Near appetite limit";
  Object.assign(threeWords.riskLossLayout.statusBadges[1], { label: "Near appetite limit", lines: 3,
    words: [{ text: "Near", lines: 1 }, { text: "appetite", lines: 1 }, { text: "limit", lines: 1 }] });
  assert.deepEqual(riskLossFindings(threeWords), []);
  for (const change of [
    (badge) => { badge.words[0].lines = 2; },
    (badge) => { badge.words.pop(); },
    (badge) => { badge.rowIndex = 1; },
  ]) {
    const invalid = structuredClone(valid); change(invalid.riskLossLayout.statusBadges[0]);
    assert.ok(riskLossFindings(invalid).some((message) => message.includes("status badges")));
  }
});

test("loss receipt rejects combined amounts and commands performed while viewing records", () => {
  const scenario = riskLossEvidenceScenarios.find((item) => item.riskLossView === "losses" && item.riskLossMatrix);
  const valid = riskLossReceipt(scenario);
  assert.deepEqual(riskLossFindings(valid), []);
  const combined = structuredClone(valid); combined.riskLossLayout.tableHeaders = ["Loss", "Amounts"];
  assert.ok(riskLossFindings(combined).some((message) => message.includes("separate exact currency")));
  const command = structuredClone(valid); command.observation.commands.push({ method: "POST", path: "/api/v1/losses" });
  assert.ok(riskLossFindings(command).some((message) => message.includes("zero business commands")));
});
