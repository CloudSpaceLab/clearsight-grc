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
  truncated: ["actualBoundedPopulation", "loadedPopulationLimitVisible", "noHorizontalOverflow"],
  denied: ["failedReadDoesNotShowRecords", "noHorizontalOverflow"],
  failure: ["failedReadDoesNotShowRecords", "noHorizontalOverflow"],
  recovery: ["failedReadDoesNotShowRecords", "retryRestoresRecords", "noHorizontalOverflow"],
  readonly: ["mutationActionsUnavailable", "noHorizontalOverflow"],
  "hierarchy-gaps": ["orphanAndCycleRemainReachable", "noHorizontalOverflow"],
  "existing-editors-and-import-guards": ["managedEditorsOpenWithoutCommands", "importedMoveRetireGuard", "importedTypeRequiresExplicitChoice", "retainedPositionBoundaryEditorsOpen", "independentMakerGuard"],
};

export async function reviewEnterpriseSetupEvidence(outputDirectory, runtimeBuild) {
  const failures = [], directory = path.join(outputDirectory, "enterprise-setup");
  const receipt = await readFile(path.join(directory, "receipt.json"), "utf8").then(JSON.parse).catch(() => null);
  if (!receipt) return { failures: ["Enterprise Setup evidence is missing or invalid"], detail: "No receipt" };
  if (receipt.baseline || receipt.failure || receipt.kind !== "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION") failures.push("Enterprise Setup receipt is not a passing runtime review");
  if (runtimeBuild && (receipt.build?.digest !== runtimeBuild.digest || receipt.revision !== runtimeBuild.revision)) failures.push("Enterprise Setup images belong to a different application build");
  if(!receipt.build?.digest||receipt.build.afterDigest!==receipt.build.digest) failures.push("Enterprise Setup application assets changed during capture");
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
    const expectedScenario=["empty","truncated","denied","failure","recovery","readonly","hierarchy-gaps"].includes(record.name)?record.name:"current";
    const statuses=(log?.reads??[]).map(read=>read.status);
    const statusesMatch=expectedScenario==="recovery"?statuses.length>=2&&statuses[0]===503&&statuses.slice(1).every(status=>status===200)
      :statuses.length>0&&statuses.every(status=>status===(expectedScenario==="denied"?403:expectedScenario==="failure"?503:200));
    if(log?.scenario!==expectedScenario||!statusesMatch) failures.push("Enterprise Setup read states differ from the recorded scenario: " + (record.name??record.viewport?.width));
    if(log?.contextRead?.path!=="/api/v1/context"||log.contextRead.status!==200||["tenant_id","legal_entity_id","principal_id"].some(key=>!log.scope?.[key]||log.contextRead.scope?.[key]!==log.scope[key])) failures.push("Enterprise Setup scope was not observed from the current identity context");
    for (const read of log?.reads ?? []) {
      if (read.path !== "/api/v1/access/overview" || read.method !== "GET" || Number(read.query?.limit) !== 50) failures.push("Enterprise Setup issued an unexpected or unbounded identity read");
      if (read.status === 200 && (!read.scope || read.scope.tenant_id !== log.scope?.tenant_id || read.scope.legal_entity_id !== log.scope?.legal_entity_id || read.scope.principal_id !== log.scope?.principal_id)) failures.push("Enterprise Setup read receipt lost the scope observed from context");
    }
    if(expectedScenario==="truncated"){
      const read=log?.reads?.at(-1);
      if(read?.truncated!==true||read.scopeIDs?.length!==500) failures.push("Enterprise Setup truncated evidence did not load the 500-area bound");
    }
  }
  const images = receipt.images ?? [], names = new Set(images.map(item => item.file));
  if (names.size !== images.length) failures.push("Enterprise Setup image identities are duplicated");
  for(const item of layouts) if([1440,390].includes(item.viewport?.width)&&!item.fullImage) failures.push("Enterprise Setup full-page runtime image is missing at " + item.viewport?.width + "px " + item.theme);
  for (const item of [...layouts, ...sources, ...behaviors]) {
    if(!item.image) failures.push("Enterprise Setup primary image is missing for " + (item.name??item.viewport?.width));
    for (const key of ["image", "fullImage", "contextImage"]) if (item[key] && !names.has(item[key])) failures.push("Enterprise Setup image receipt is missing: " + item[key]);
  }
  for (const item of images) {
    if (path.basename(item.file) !== item.file) { failures.push("Invalid Enterprise Setup image path"); continue; }
    const bytes = await readFile(path.join(directory, item.file)).catch(() => null);
    if (!bytes || bytes.length !== item.bytes || sha(bytes) !== item.sha256) failures.push("Enterprise Setup image bytes differ: " + item.file);
  }
  return { failures, detail: layouts.length + " layouts, " + behaviors.length + " behavior scenarios, " + images.length + " retained images; whole-screen acceptance not assessed" };
}
