import { createHash } from "node:crypto";
import { readFile } from "node:fs/promises";
import path from "node:path";

const sha = bytes => createHash("sha256").update(bytes).digest("hex");
const sourceSHA256 = "cc7542e2a3047338b1d2fbbbe33756adbc9ee8c2084fb5920834da8aeedc04c7";
const tabs = ["Organization", "Groups", "People & access", "Jurisdictions", "Frameworks", "Data imports"];
const requiredBehaviors = {
  "selection-search-history-light-1440": ["searchFiltersLoadedScopes", "searchRetainsAncestors", "searchNoMatch", "collapseHidesDescendants", "selectionUpdatesContextAndURL", "historyRestoresSelection", "deepLinkRestoresContext"],
  "selection-search-history-dark-390": ["searchFiltersLoadedScopes", "searchRetainsAncestors", "searchNoMatch", "collapseHidesDescendants", "selectionUpdatesContextAndURL", "historyRestoresSelection", "deepLinkRestoresContext"],
  "setup-navigation": ["keyboardTraversalPreservesSelection", "keyboardImportHandoff", "unavailableDestinationsHonest", "setupHistoryPreservesUnit", "peopleHandoff", "frameworkHandoff", "importHandoff"],
  empty: ["emptyPopulationHonest", "noHorizontalOverflow"],
  truncated: ["loadedPopulationLimitVisible", "noHorizontalOverflow"],
  denied: ["failedReadDoesNotShowRecords", "noHorizontalOverflow"],
  failure: ["failedReadDoesNotShowRecords", "noHorizontalOverflow"],
  recovery: ["failedReadDoesNotShowRecords", "retryRestoresRecords", "noHorizontalOverflow"],
  readonly: ["mutationActionsUnavailable", "noHorizontalOverflow"],
  "hierarchy-gaps": ["orphanAndCycleRemainReachable", "noHorizontalOverflow"],
  "existing-editors-and-import-guards": ["managedEditorsOpenWithoutCommands", "importedMoveRetireGuard", "retainedPositionBoundaryEditorsOpen", "independentMakerGuard"],
};

export async function reviewEnterpriseSetupEvidence(outputDirectory, runtimeBuild) {
  const failures = [], directory = path.join(outputDirectory, "enterprise-setup");
  const receipt = await readFile(path.join(directory, "receipt.json"), "utf8").then(JSON.parse).catch(() => null);
  if (!receipt) return { failures: ["Enterprise Setup evidence is missing or invalid"], detail: "No receipt" };
  if (receipt.baseline || receipt.failure || receipt.kind !== "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION") failures.push("Enterprise Setup receipt is not a passing runtime review");
  if (runtimeBuild && (receipt.build?.digest !== runtimeBuild.digest || receipt.revision !== runtimeBuild.revision)) failures.push("Enterprise Setup images belong to a different application build");
  for (const [key, filename] of [["fixtureSHA256", "src/staticDemoEnterpriseSetupRuntime.js"], ["captureSHA256", "scripts/capture-enterprise-setup-evidence.mjs"], ["reviewerSHA256", "scripts/review-enterprise-setup-evidence.mjs"]]) {
    if (receipt.inputs?.[key] !== sha(await readFile(filename))) failures.push("Enterprise Setup capture input changed: " + filename);
  }
  if (receipt.source?.sha256 !== sourceSHA256) failures.push("Enterprise Setup review lost its approved HTML identity");
  const layouts = receipt.layouts ?? [], sources = receipt.sources ?? [], behaviors = receipt.behaviors ?? [];
  if (layouts.length !== 10 || sources.length !== 5) failures.push("Enterprise Setup viewport/theme matrix is incomplete");
  for (const width of [1440, 1280, 1024, 768, 390]) {
    const sourceMatches = sources.filter(item => item.viewport?.width === width), source = sourceMatches[0]?.geometry;
    if (sourceMatches.length !== 1 || !source?.sourceOrder || !source.firstRecordInViewport) failures.push("Enterprise Setup source render is missing or invalid at " + width + "px");
    if(width<=1300&&!sourceMatches[0]?.contextImage) failures.push("Enterprise Setup lower source context image is missing at " + width + "px");
    for (const theme of ["light", "dark"]) {
      const matches = layouts.filter(item => item.viewport?.width === width && item.theme === theme), item = matches[0], g = item?.geometry;
      if (matches.length !== 1 || !g?.sourceOrder || !g.firstRecordInViewport || !g.firstAreaInViewport || g.scrollWidth > g.documentWidth + 1
        || g.headings?.join("|") !== "Enterprise setup" || g.tabLabels?.join("|") !== tabs.join("|") || g.selectedTabs?.join("|") !== "Organization"
        || g.contextPlacement !== source?.contextPlacement || g.innerColumns !== source?.innerColumns
        || item.errors?.length || !Array.isArray(item.accessibility?.violations) || item.accessibility.violations.length
        || !g.controls?.length || g.controls.some(box => box.width < 44 || box.height < 44)) failures.push("Enterprise Setup anatomy/interaction evidence failed at " + width + "px " + theme);
      if (width===1440 && (!g?.layout?.width || !g.context?.width || !source?.layout?.width || !source.context?.width || Math.abs(g.context.width/g.layout.width-source.context.width/source.layout.width)>0.04)) failures.push("Enterprise Setup selected-context proportion differs from source");
      if(width<=1300&&(!item?.contextImage||item.contextGeometry?.title!=="Technology"||!item.contextGeometry.focused||!item.contextGeometry.titleVisible||!item.contextGeometry.hash?.includes("unit=sample-org-technology"))) failures.push("Enterprise Setup selection does not reveal the lower context at " + width + "px " + theme);
    }
  }
  if (behaviors.length !== Object.keys(requiredBehaviors).length) failures.push("Enterprise Setup behavior scenario set is incomplete or duplicated");
  for (const [name, checks] of Object.entries(requiredBehaviors)) {
    const matches = behaviors.filter(item => item.name === name), record = matches[0];
    if (matches.length !== 1 || !record?.passed || record.errors?.length || checks.some(key => record.checks?.[key] !== true)) failures.push("Enterprise Setup behavior evidence failed: " + name);
  }
  for (const record of [...layouts, ...behaviors]) {
    const log = record.observation;
    if (!log || log.kind !== "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION" || log.commands?.length || log.pending !== 0 || !log.reads?.length) failures.push("Enterprise Setup transport evidence is incomplete or wrote a command");
    for (const read of log?.reads ?? []) {
      if (read.path !== "/api/v1/access/overview" || read.method !== "GET" || Number(read.query?.limit) !== 50) failures.push("Enterprise Setup issued an unexpected or unbounded identity read");
      if (read.status === 200 && (!read.scope || read.scope.tenant_id !== log.scope.tenant_id || read.scope.legal_entity_id !== log.scope.legal_entity_id || read.scope.principal_id !== log.scope.principal_id)) failures.push("Enterprise Setup read receipt lost verified scope identity");
    }
  }
  const images = receipt.images ?? [], names = new Set(images.map(item => item.file));
  if (names.size !== images.length) failures.push("Enterprise Setup image identities are duplicated");
  for (const item of [...layouts, ...sources, ...behaviors]) {
    for (const key of ["image", "fullImage", "contextImage"]) if (item[key] && !names.has(item[key])) failures.push("Enterprise Setup image receipt is missing: " + item[key]);
  }
  for (const item of images) {
    if (path.basename(item.file) !== item.file) { failures.push("Invalid Enterprise Setup image path"); continue; }
    const bytes = await readFile(path.join(directory, item.file)).catch(() => null);
    if (!bytes || bytes.length !== item.bytes || sha(bytes) !== item.sha256) failures.push("Enterprise Setup image bytes differ: " + item.file);
  }
  return { failures, detail: layouts.length + " layouts, " + behaviors.length + " behavior scenarios, " + images.length + " retained images; whole-screen acceptance not assessed" };
}
