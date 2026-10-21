import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import test from "node:test";
import { frameworkEvidenceScenarios } from "./framework-evidence-scenarios.mjs";
import { reportsDesignEvidenceScenarios } from "./reports-design-evidence-scenarios.mjs";
import { reviewReportsDesignCaptures } from "./reports-design-evidence-contract.mjs";

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
