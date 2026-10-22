import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import test from "node:test";
import { frameworkEvidenceScenarios } from "./framework-evidence-scenarios.mjs";

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
