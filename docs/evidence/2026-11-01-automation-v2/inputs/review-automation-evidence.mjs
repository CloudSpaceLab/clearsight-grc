import { createHash } from "node:crypto";
import { readFile } from "node:fs/promises";
import path from "node:path";
import { pathToFileURL } from "node:url";

const sha = bytes => createHash("sha256").update(bytes).digest("hex");
const tabs = ["Automations", "Schedules", "Escalations", "Activity"];
const sourceSHA256 = "7dad1ffc651fd03821cc996ff13f1203cb1ec1344a8a5bdec7b00b49035a7b6f";
const requiredBehaviors = {
  "selection-history-light-1440": ["searchFiltersLoadedPolicies", "selectionUpdatesContextAndURL", "historyRestoresSelection", "deepLinkRestoresSelection", "noMatch", "statusFiltersLatestRevisions"],
  "selection-history-dark-390": ["searchFiltersLoadedPolicies", "selectionUpdatesContextAndURL", "historyRestoresSelection", "deepLinkRestoresSelection", "returnRestoresFocus", "noMatch", "statusFiltersLatestRevisions"],
  "tabs-and-handoff-light-1440": ["unavailableTabsAreExplicit", "historyRestoresTab", "existingEscalationsHandoff", "returnKeepsAutomationTab"],
  "tabs-and-handoff-dark-390": ["unavailableTabsAreExplicit", "historyRestoresTab", "existingEscalationsHandoff", "returnKeepsAutomationTab"],
  empty: ["emptyPopulationHonest", "noUnsupportedCommands"],
  truncated: ["actualBoundedPopulation", "truncationVisibleInViewport", "noUnsupportedCommands"],
  denied: ["failedReadHidesRecords", "noUnsupportedCommands"],
  failure: ["failedReadHidesRecords", "noUnsupportedCommands"],
  recovery: ["failedReadHidesRecords", "retryRestoresRecords", "noUnsupportedCommands"],
  readonly: ["recordsReadable", "noUnsupportedCommands"],
  long: ["recordsReadable", "longContextReadable", "noUnsupportedCommands"],
  loading: ["loadingThenRecords", "noUnsupportedCommands"],
};
export async function reviewAutomationEvidence(outputDirectory, runtimeBuild) {
  const failures = [], directory = path.join(outputDirectory, "automation");
  const receipt = await readFile(path.join(directory, "receipt.json"), "utf8").then(JSON.parse).catch(() => null);
  if (!receipt) return { failures: ["Automation receipt is missing or invalid"], detail: "No receipt" };
  if (receipt.baseline || receipt.failure || receipt.kind !== "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION") failures.push("Automation receipt is not a passing runtime review");
  if (runtimeBuild && (receipt.build?.digest !== runtimeBuild.digest || receipt.revision !== runtimeBuild.revision)) failures.push("Automation images belong to another application build");
  if (!receipt.build?.digest || receipt.build.afterDigest !== receipt.build.digest) failures.push("Automation assets changed during capture");
  for (const [key, file] of [["fixtureSHA256", "src/staticDemoAutomationRuntime.js"], ["captureSHA256", "scripts/capture-automation-evidence.mjs"], ["reviewerSHA256", "scripts/review-automation-evidence.mjs"]]) {
    if (receipt.inputs?.[key] !== sha(await readFile(file))) failures.push("Automation capture input changed: " + file);
  }
  if (receipt.source?.sha256 !== sourceSHA256) failures.push("Automation lost approved source identity");
  const layouts = receipt.layouts ?? [], sources = receipt.sources ?? [], behaviors = receipt.behaviors ?? [];
  if (layouts.length !== 10 || sources.length !== 5) failures.push("Automation viewport matrix is incomplete");
  for (const width of [1440, 1280, 1024, 768, 390]) {
    const sourceMatches = sources.filter(item => item.viewport?.width === width), source = sourceMatches[0]?.geometry;
    if (sourceMatches.length !== 1 || !source?.sourceOrder) failures.push("Automation source anatomy is missing at " + width);
    if (width <= 1100 && !sourceMatches[0]?.contextImage) failures.push("Automation lower source context is missing at " + width);
    for (const theme of ["light", "dark"]) {
      const matches = layouts.filter(item => item.viewport?.width === width && item.theme === theme), item = matches[0], g = item?.geometry;
      if (matches.length !== 1 || !g?.sourceOrder || !g.contextContentOrder || !g.firstRecordInViewport || g.scrollWidth > g.documentWidth + 1
        || g.headings?.join("|") !== "Automation & governance" || g.tabLabels?.join("|") !== tabs.join("|") || g.selectedTabs?.join("|") !== "Automations"
        || g.contextPlacement !== source?.contextPlacement || g.metricColumns !== source?.metricColumns
        || item.errors?.length || !Array.isArray(item.accessibility?.violations) || item.accessibility.violations.length
        || !g.controls?.length || g.controls.some(box => box.width < 44 || box.height < 44)) failures.push("Automation anatomy/interaction evidence failed at " + width + "px " + theme);
      if (width <= 1100 && (!item?.contextImage || !item.contextGeometry?.focused || !item.contextGeometry.titleVisible || item.contextGeometry.title !== "Evidence refresh")) failures.push("Automation lower context was not revealed at " + width + "px " + theme);
      if (width > 1100 && (!g?.context?.width || !g.layout?.width || !source?.context?.width || !source.layout?.width || Math.abs(g.context.width / g.layout.width - source.context.width / source.layout.width) > 0.04)) failures.push("Automation selected-context proportion differs from source at " + width + "px " + theme);
      if ([1440, 390].includes(width) && !item?.fullImage) failures.push("Automation full-page image missing at " + width + "px " + theme);
    }
  }
  if (behaviors.length !== Object.keys(requiredBehaviors).length) failures.push("Automation behavior set is incomplete or duplicated");
  for (const [name, checks] of Object.entries(requiredBehaviors)) {
    const matches = behaviors.filter(item => item.name === name), item = matches[0];
    if (matches.length !== 1 || !item?.passed || item.errors?.length || !item.checks?.noHorizontalOverflow || checks.some(key => item.checks?.[key] !== true)) failures.push("Automation behavior failed: " + name);
    if (name.startsWith("selection-history") && !item?.preReloadObservation) failures.push("Automation selection/history lost its pre-reload transport receipt: " + name);
    if (name.startsWith("tabs-and-handoff") && item?.tabImages?.length !== 3) failures.push("Automation unavailable/handoff tab images are missing: " + name);
    if (name === "truncated") {
      const g = item?.limitGeometry;
      if (!g || g.y < g.shellBottom || g.bottom > g.height || g.x < 0 || g.right > g.width) failures.push("Automation omitted-policy disclosure is outside the viewport");
    }
    if (name === "long" && (!item?.fullImage || item.longFields?.length !== 3 || item.longFields.some(field => !(field.width > 0 && field.height > 0) || field.scrollWidth > field.width + 1))) failures.push("Automation long keys, values and codes are clipped or unmeasured");
  }
  for (const item of [...layouts, ...behaviors].flatMap(item => item.preReloadObservation ? [{ ...item, observation: item.preReloadObservation }, item] : [item])) {
    const log = item.observation;
    if (!log || log.kind !== "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION" || log.commands?.length || log.pending !== 0 || !log.reads?.length) failures.push("Automation read receipt is incomplete or executed commands");
    if (log?.contextRead?.path !== "/api/v1/context" || log.contextRead.status !== 200 || ["tenant_id", "legal_entity_id", "principal_id"].some(key => !log.scope?.[key] || log.contextRead.scope?.[key] !== log.scope[key])) failures.push("Automation context was not observed from verified scope");
    const scenario = item.scenario ?? "current";
    const expected = scenario === "denied" ? 403 : scenario === "failure" ? 503 : 200;
    const statuses = (log?.reads ?? []).map(read => read.status);
    if (scenario === "recovery" ? statuses[0] !== 503 || !statuses.slice(1).length || statuses.slice(1).some(value => value !== 200) : statuses.some(value => value !== expected)) failures.push("Automation read statuses differ from scenario: " + scenario);
    for (const read of log?.reads ?? []) {
      if (read.path !== "/api/v1/compliance/automation-policies" || read.method !== "GET") failures.push("Automation issued an unexpected read");
      if (read.status === 200 && (read.scope?.tenant_id !== log.scope?.tenant_id || read.revisionBasis !== "LATEST_PER_CODE" || read.policyIDs?.length !== read.count || read.count > 200)) failures.push("Automation population receipt is invalid");
    }
    if (scenario === "truncated" && !log?.reads?.some(read => read.count === 200 && read.truncated === true)) failures.push("Automation truncation did not exercise the 200-row bound");
  }
  const images = receipt.images ?? [], names = new Set(images.map(item => item.file));
  if (images.length !== names.size) failures.push("Automation image names are duplicated");
  for (const item of [...layouts, ...sources, ...behaviors]) {
    if (!item.image) failures.push("Automation primary image missing");
    for (const key of ["image", "fullImage", "contextImage", "loadingImage"]) if (item[key] && !names.has(item[key])) failures.push("Automation image receipt missing: " + item[key]);
    for (const file of item.tabImages ?? []) if (!names.has(file)) failures.push("Automation tab image receipt missing: " + file);
  }
  for (const item of images) {
    if (path.basename(item.file) !== item.file) { failures.push("Invalid Automation image path"); continue; }
    const bytes = await readFile(path.join(directory, item.file)).catch(() => null);
    if (!bytes || bytes.length !== item.bytes || sha(bytes) !== item.sha256) failures.push("Automation image bytes changed: " + item.file);
  }
  return { failures, detail: layouts.length + " layouts, " + behaviors.length + " behaviors, " + images.length + " images; whole-screen acceptance not assessed" };
}
if (process.argv[1] && pathToFileURL(path.resolve(process.argv[1])).href === import.meta.url) {
  const result = await reviewAutomationEvidence(path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence"));
  process.stdout.write(JSON.stringify(result, null, 2) + "\n"); if (result.failures.length) process.exitCode = 1;
}
