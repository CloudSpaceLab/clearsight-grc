import { createHash } from "node:crypto";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { createRequire } from "node:module";
import { fileURLToPath, pathToFileURL } from "node:url";

const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const sourceRoot = path.join(repositoryRoot, "docs/design/references/clearsight-2026-10/source");
const outputDir = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence", "approved-reference");

const references = [
  {
    id: "framework-library",
    source: "clearsight-framework-designs/framework-library.html",
    runtime: {
      "1440x900": "200-framework-library-light-1440x900.png",
      "390x844": "202-framework-library-light-mobile-390x844.png",
    },
  },
  {
    id: "framework-adoption",
    source: "clearsight-framework-designs/framework-adoption.html",
    runtime: {},
  },
  {
    id: "program-workspace",
    source: "clearsight-program-designs/program-workspace-preview.html",
    runtime: { "1440x900": "04-program-light-1440x900.png", "390x844": "04-program-light-mobile-390x844.png" },
  },
  {
    id: "assessment-workspace",
    source: "clearsight-designs-01-05/assessment-workspace-preview.html",
    runtime: { "1440x900": "211-internal-assessment-light-1440x900.png", "1280x800": "212-internal-assessment-dark-1280x800.png", "1024x768": "215-internal-assessment-light-1024x768.png", "768x900": "216-internal-assessment-dark-768x900.png", "390x844": "213-internal-assessment-light-mobile-390x844.png" },
  },
  { id: "vendor-management", source: "clearsight-vendor-management-designs/vendor-workspace-preview.html", runtime: {} },
  { id: "operational-risk", source: "clearsight-operational-risk-designs/operational-risk-preview.html", runtime: {} },
  { id: "privacy-data", source: "clearsight-privacy-data-designs/privacy-workspace-preview.html", runtime: {} },
  {
    id: "reports-insights",
    source: "clearsight-reports-insights-designs/reports-workspace-preview.html",
    sourceTab: "overview",
    runtime: {
      "1440x900": "140a-report-overview-light-1440x900.png",
      "1280x800": "140h-report-overview-light-1280x800.png",
      "1024x768": "140f-report-overview-light-1024x768.png",
      "768x900": "140i-report-overview-light-768x900.png",
      "390x844": "140b-report-overview-light-mobile-390x844.png",
    },
  },
  {
    id: "reports-library",
    source: "clearsight-reports-insights-designs/reports-workspace-preview.html",
    sourceTab: "reports",
    runtime: {
      "1440x900": "140-report-library-light-1440x900.png",
      "1280x800": "141d-report-library-light-1280x800.png",
      "1024x768": "141f-report-library-light-1024x768.png",
      "768x900": "141h-report-library-light-768x900.png",
      "390x844": "141b-report-library-light-mobile-390x844.png",
    },
  },
  { id: "work-approvals", source: "clearsight-work-approvals-designs/work-approvals-preview.html", runtime: {} },
  { id: "controls-policies", source: "clearsight-controls-policies-designs/controls-policies-preview.html", runtime: {} },
  { id: "enterprise-setup", source: "clearsight-enterprise-setup-designs/enterprise-setup-preview.html", runtime: {} },
  { id: "integrations", source: "clearsight-integrations-designs/integrations-workspace-preview.html", runtime: {} },
  { id: "search-command-center", source: "clearsight-search-command-center-designs/search-command-center-preview.html", runtime: {} },
  { id: "notifications-activity", source: "clearsight-notifications-activity-designs/notifications-activity-preview.html", runtime: {} },
  { id: "audit-assurance", source: "clearsight-audit-assurance-designs/audit-assurance-preview.html", runtime: {} },
  { id: "people-access", source: "clearsight-people-access-designs/people-access-preview.html", runtime: {} },
  { id: "automation-governance", source: "clearsight-automation-governance-designs/automation-governance-preview.html", runtime: {} },
  { id: "contextual-copilot", source: "clearsight-contextual-copilot-designs/ai-copilot-preview.html", runtime: {} },
];

const viewports = [
  { width: 1440, height: 900 },
  { width: 1280, height: 800 },
  { width: 1024, height: 768 },
  { width: 768, height: 900 },
  { width: 390, height: 844 },
];

function digest(value) {
  return createHash("sha256").update(value).digest("hex");
}

const manifest = JSON.parse(await readFile(path.join(sourceRoot, "MANIFEST.json"), "utf8"));
const byPath = new Map(manifest.files.map((entry) => [entry.path, entry]));
const runtimeManifest = await readFile(path.join(outputDir, "..", "manifest.json"), "utf8")
  .then(JSON.parse).catch((error) => { if (error.code === "ENOENT") return null; throw error; });
const runtimeByName = new Map((runtimeManifest?.captures ?? []).map((capture) => [`${capture.name}.png`, capture]));
const requestedReferences = process.env.UI_EVIDENCE_SCOPE === "reports"
  ? references.filter((reference) => reference.id.startsWith("reports-"))
  : process.env.UI_EVIDENCE_SCOPE === "navigation"
    ? references.filter((reference) => reference.id === "reports-insights") : references;
const verified = await Promise.all(requestedReferences.map(async (reference) => {
  const expected = byPath.get(reference.source);
  if (!expected) throw new Error(`Approved source not indexed: ${reference.source}`);
  const absolutePath = path.join(sourceRoot, reference.source);
  const bytes = await readFile(absolutePath);
  if (bytes.length !== expected.bytes || digest(bytes) !== expected.sha256) {
    throw new Error(`Approved source hash mismatch: ${reference.source}`);
  }
  return { ...reference, absolutePath, bytes: expected.bytes, sha256: expected.sha256 };
}));

await mkdir(outputDir, { recursive: true });
const browser = await chromium.launch({ headless: true });
const captures = [];

try {
  for (const reference of verified) {
    for (const viewport of viewports) {
      const key = `${viewport.width}x${viewport.height}`;
      const context = await browser.newContext({
        viewport,
        colorScheme: "light",
        hasTouch: viewport.width <= 390,
        reducedMotion: "reduce",
        deviceScaleFactor: 1,
      });
      try {
        // Archived prototypes are local design sources, never remote app fixtures.
        await context.route(/^https?:\/\//, (route) => route.abort());
        const page = await context.newPage();
        const errors = [];
        page.on("pageerror", (error) => errors.push(error.message));
        await page.goto(pathToFileURL(reference.absolutePath).href, { waitUntil: "load" });
        await page.evaluate(() => document.fonts?.ready);
        if (reference.sourceTab) {
          const tab = page.locator(`[role="tab"][data-tab="${reference.sourceTab}"]`);
          await tab.click();
          const panelID = await tab.getAttribute("aria-controls");
          await page.locator(`[id="${panelID}"]`).waitFor({ state: "visible" });
          if (await tab.getAttribute("aria-selected") !== "true") throw new Error(`Source tab did not activate: ${reference.id}/${reference.sourceTab}`);
        }
        const presentation = await page.evaluate(() => {
          const sidebar = document.querySelector(".sidebar");
          const links = [...(sidebar?.querySelectorAll("a.nav-item") ?? [])];
          const visible = (node) => Boolean(node?.getClientRects().length && getComputedStyle(node).visibility !== "hidden");
          const name = (node) => (node.getAttribute("aria-label") ?? node.textContent).trim().replace(/\s+/g, " ");
          const labelVisibility = links.map((node) => visible(node.querySelector("span")));
          const labelLineCounts = links.map((node, index) => {
            if (!labelVisibility[index]) return 0;
            const range = document.createRange();
            range.selectNodeContents(node.querySelector("span"));
            return new Set([...range.getClientRects()].filter((box) => box.width > 3 && box.height > 3).map((box) => Math.round(box.top))).size;
          });
          const active = links.filter((node) => node.classList.contains("active") || node.getAttribute("aria-current") === "page");
          const marker = active[0] ? getComputedStyle(active[0], "::before") : null;
          return {
            title: document.title,
            heading: document.querySelector("h1")?.textContent?.trim() ?? "",
            clientWidth: document.documentElement.clientWidth,
            scrollWidth: document.documentElement.scrollWidth,
            navigation: sidebar && links.length ? {
              labels: links.map(name), hrefs: links.map((node) => node.getAttribute("href")),
              activePage: active.map(name), railWidth: sidebar.getBoundingClientRect().width,
              labelMode: labelVisibility.every(Boolean) ? "expanded" : labelVisibility.every((value) => !value) ? "icons" : "mixed",
              maxLabelLines: Math.max(...labelLineCounts),
              footerLabels: [...sidebar.querySelectorAll(".side-bottom button, .side-bottom a")].map(name),
              minTargetWidth: Math.min(...links.map((node) => node.getBoundingClientRect().width)),
              minTargetHeight: Math.min(...links.map((node) => node.getBoundingClientRect().height)),
              activeMarkerWidth: marker ? parseFloat(marker.width) || 0 : 0,
              activeMarkerHeight: marker ? parseFloat(marker.height) || 0 : 0,
            } : null,
          };
        });
        if (!presentation.heading) throw new Error(`Source screen has no h1: ${reference.source}`);
        const runtimeCapture = runtimeByName.get(reference.runtime[key]);
        const runtimeNavigation = runtimeCapture?.navigation;
        let navigationComparison = null;
        if (reference.id.startsWith("reports-")) {
          const sourceNavigation = presentation.navigation;
          const sourceLabels = ["Home", "Frameworks", "Programs", "Forms", "Vendors", "Risk & Losses", "Privacy", "Reports"];
          const sourceRail = viewport.width <= 480 ? 56 : viewport.width <= 1024 ? 72 : 228;
          if (!sourceNavigation || sourceNavigation.labels.join("|") !== sourceLabels.join("|")
              || sourceNavigation.activePage.join("|") !== "Reports" || Math.abs(sourceNavigation.railWidth - sourceRail) > 1
              || sourceNavigation.labelMode !== (viewport.width <= 1024 ? "icons" : "expanded")
              || sourceNavigation.maxLabelLines !== (viewport.width <= 1024 ? 0 : 1)
              || !sourceNavigation.footerLabels.includes("Settings")) {
            throw new Error(`Hash-verified Reports source menu did not render its expected anatomy at ${key}`);
          }
          if (runtimeManifest) {
            if (!runtimeNavigation || runtimeCapture.viewport?.width !== viewport.width || runtimeCapture.viewport?.height !== viewport.height
                || runtimeNavigation.labels.filter((label) => label !== "Work & Approvals").join("|") !== sourceNavigation.labels.join("|")
                || runtimeNavigation.activePage.join("|") !== sourceNavigation.activePage.join("|")
                || runtimeNavigation.labelMode !== sourceNavigation.labelMode || runtimeNavigation.maxLabelLines !== sourceNavigation.maxLabelLines
                || Math.abs(runtimeNavigation.railWidth - sourceNavigation.railWidth) > 1) {
              throw new Error(`${reference.id}/${key}: matching runtime menu evidence is missing or differs from the approved source structure`);
            }
            navigationComparison = {
              runtime_capture: reference.runtime[key], runtime_theme: runtimeCapture.theme,
              commonDestinationsInSourceOrder: true, activeDestinationMatches: true, labelModeMatches: true,
              primaryLabelLinesMatch: true,
              sourceRailWidth: sourceNavigation.railWidth, runtimeRailWidth: runtimeNavigation.railWidth,
              retainedRuntimeDestination: "Work & Approvals", settingsPermission: "config_read",
              conclusion: "STRUCTURE_CHECKED_VISUAL_REVIEW_REQUIRED",
            };
          }
        }
        const image = `approved-${reference.id}-${key}.png`;
        await page.mouse.move(0, 0);
        await page.screenshot({ path: path.join(outputDir, image), fullPage: false, animations: "disabled" });
        captures.push({
          id: reference.id,
          viewport,
          source: reference.source,
          source_tab: reference.sourceTab ?? null,
          source_sha256: reference.sha256,
          image,
          runtime_capture: reference.runtime[key] ?? null,
          ...presentation,
          navigation_comparison: navigationComparison,
          prototype_errors: errors,
          visual_parity: "NOT_REVIEWED",
        });
      } finally {
        await context.close();
      }
    }
  }
} finally {
  await browser.close();
}

await writeFile(path.join(outputDir, "manifest.json"), JSON.stringify({
  archive_sha256: manifest.source_archive_sha256,
  captures,
}, null, 2) + "\n");

const lines = [
  "# Approved design reference captures",
  "",
  "These screenshots render the hash-verified original HTML, not the ClearSight application.",
  "Runtime captures have separate files in the parent UI evidence artifact.",
  "Different source/demo data prevents meaningful pixel-diff scoring. Visual parity remains unreviewed.",
  "Original HTML references use their approved light theme; paired dark runtime images support layout comparisons only.",
  "Menu metadata records each original rail and item order without changing the source. Some originals use a 55px phone rail; the shared runtime uses 56px to preserve 44px targets. Reports already uses 56px.",
  "Reports menu comparisons retain Work & Approvals as a direct runtime destination and permission-gate Settings. Matching structure does not establish whole-screen visual acceptance.",
  "",
  "| Screen | Viewport | Approved screenshot | Runtime screenshot |",
  "| --- | --- | --- | --- |",
  ...captures.map((capture) =>
    `| ${capture.id} | ${capture.viewport.width}×${capture.viewport.height} | ${capture.image} | ${capture.runtime_capture ?? "No matching journey captured"} |`),
  "",
  "No standalone Control Tower HTML is indexed in the source archive; verify its missing reference before visual sign-off.",
  "Review navigation, hierarchy, spacing, typography, panels, content density, interaction states, and responsive behavior manually.",
  "Treat prototype values as illustrative; validate live data, authority, and scope separately under #462.",
  "",
];
await writeFile(path.join(outputDir, "README.md"), lines.join("\n"));
process.stdout.write(`Captured ${captures.length} approved source viewports with verified checksums. No parity conclusion.\n`);
