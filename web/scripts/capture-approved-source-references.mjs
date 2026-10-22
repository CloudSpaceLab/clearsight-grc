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
const requestedReferences = process.env.UI_EVIDENCE_SCOPE === "reports"
  ? references.filter((reference) => reference.id.startsWith("reports-")) : references;
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
        const presentation = await page.evaluate(() => ({
          title: document.title,
          heading: document.querySelector("h1")?.textContent?.trim() ?? "",
          clientWidth: document.documentElement.clientWidth,
          scrollWidth: document.documentElement.scrollWidth,
        }));
        if (!presentation.heading) throw new Error(`Source screen has no h1: ${reference.source}`);
        const image = `approved-${reference.id}-${key}.png`;
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
