import { privacyActivityID, privacyActivityName, privacyEvidenceScenarios, privacyViews } from "./privacy-evidence-scenarios.mjs";

const same = (a, b) => JSON.stringify(a) === JSON.stringify(b);
export const privacyRect = (r) => r && ["x", "y", "width", "height", "right", "bottom"].every((key) => Number.isFinite(r[key]))
  && r.width > 0 && r.height > 0 && Math.abs(r.x + r.width - r.right) <= 1 && Math.abs(r.y + r.height - r.bottom) <= 1;
const inside = (inner, outer) => privacyRect(inner) && privacyRect(outer) && inner.x >= outer.x - 1 && inner.right <= outer.right + 1;
export const privacySourceTabs = ["Overview", "ROPA", "DPIAs", "Breaches", "Vendor reviews", "Requests"];

export function privacyCaptureFailures(capture, scenario) {
  const failures = [], layout = capture?.privacyLayout, width = scenario.viewport.width, check = scenario.privacyCheck;
  const fail = (message) => failures.push(`${scenario.name}: ${message}`);
  if (!capture || !layout || ["name", "state", "route", "fixture", "theme"].some((key) => capture[key] !== scenario[key]) || !same(capture.viewport, scenario.viewport)) {
    fail("Privacy scenario metadata or measured layout is missing"); return failures;
  }
  const observed = capture.observation;
  if (!observed || !Array.isArray(observed.reads) || !observed.reads.length || !Array.isArray(observed.commands) || observed.commands.length
    || observed.reads.some((read) => new URLSearchParams(read.query).get("tenant_id") !== "bank-demo" || ![200, 403, 404, 503].includes(read.status))) fail("viewing requires scoped observed reads without commands");
  if (layout.evidenceKind !== "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION" || layout.kind !== scenario.privacyView
    || capture.metrics.scrollWidth > capture.metrics.clientWidth + 1 || capture.errors?.length) fail("fixture provenance, browser errors or document overflow");
  if (["detail", "not-found"].includes(check)) {
    const detail = capture.detail;
    if (!detail || !privacyRect(detail.bounds) || Math.abs(detail.bounds.width - Math.min(480, width)) > 1
      || Math.abs(detail.bounds.right - width) > 1 || detail.scrollWidth > detail.clientWidth + 1
      || detail.keyboardContained !== true || !privacyRect(detail.closeBounds) || detail.closeBounds.width < 44 || detail.closeBounds.height < 44) fail("selected record requires contained 480px/full-phone inspector and keyboard/touch behavior");
    if (!privacyRect(detail?.headingBounds) || detail.headingBounds.right > detail.closeBounds?.x && detail.headingBounds.x < detail.closeBounds?.right
      && detail.headingBounds.bottom > detail.closeBounds?.y && detail.headingBounds.y < detail.closeBounds?.bottom) fail("record and unavailable headings must remain clear of the close control");
    if (check === "detail" && (detail?.heading !== privacyActivityName || detail.route !== `#ropa/activity/${privacyActivityID}` || detail.historyEntries !== 3
      || !observed?.reads.some((read) => read.activityID === privacyActivityID && read.status === 200)
      || !observed?.reads.some((read) => read.path.endsWith(`/${privacyActivityID}/history`) && read.eventIDs?.length === 3))) fail("inspector identity and history must match the exact record reads");
    if (check === "detail" && (!privacyRect(detail?.contentBounds) || !privacyRect(detail?.recordBounds) || !privacyRect(detail?.factsBounds)
      || Math.abs(detail.recordBounds.x - detail.contentBounds.x - detail.contentPadding.left) > 1
      || Math.abs(detail.contentBounds.right - detail.recordBounds.right - detail.contentPadding.right) > 1
      || Math.abs(detail.factsBounds.width - detail.recordBounds.width) > 1 || !inside(detail.factsBounds, detail.recordBounds)
      || !detail.factRows?.length || detail.factRows.some((row) => !inside(row.term, row.bounds) || !inside(row.value, row.bounds) || row.term.right > row.value.x + 1))) fail("inspector facts must use the full padded content width with contained key/value fields");
    if (check === "not-found" && (detail?.unavailable !== true || Object.keys(detail.facts ?? {}).length || !observed?.reads.some((read) => read.path.endsWith("/privacy-missing-record") && read.status === 404))) fail("missing record must not render the previously selected record");
    return failures;
  }
  if (layout.headingText !== "Privacy & data" || layout.headingCount !== 1 || !privacyRect(layout.heading) || !privacyRect(layout.menu)
    || !same(layout.localTabs, privacyViews.map((item) => item.label)) || layout.selectedTab !== privacyViews.find((item) => item.id === scenario.privacyView).label
    || !same(layout.visiblePanelIDs, [layout.selectedPanelID]) || !inside(layout.selectedBounds, layout.menu)
    || layout.selectedVisible !== true || layout.heading.bottom > layout.menu.y + 1 || layout.route !== scenario.route
    || !same(layout.primaryCurrent, ["Privacy"])) fail("heading, approved supported menu, active panel or primary destination is inconsistent");
  if (scenario.touch && (layout.minTargetWidth < 44 || layout.minTargetHeight < 44 || layout.undersizedTargets.length)) fail("primary controls need 44px touch targets");
  if (scenario.privacyMatrix && (!privacyRect(layout.firstUseful) || layout.firstUsefulVisible !== true || layout.firstUseful.y < 0 || layout.firstUseful.bottom > scenario.viewport.height)) fail("first useful data must be fully visible without scrolling");
  const list = observed?.reads.filter((read) => read.path === "/api/v1/ropa/processing-activities").at(-1);
  const summary = observed?.reads.filter((read) => read.path === "/api/v1/ropa/dashboard").at(-1);
  if (scenario.privacyView === "register" && !["denied", "empty"].includes(check)) {
    if (!privacyRect(layout.controls) || !privacyRect(layout.table) || layout.controls.bottom > layout.table.y + 1
      || !same(layout.headers, ["Processing activity", "Data subjects", "Lawful basis", "Next review", "Owner", "Status", "Row action"])
      || !same(layout.recordNames, list?.records?.map((record) => record.name)) || !layout.recordNames.length || layout.metrics.length) fail("record-first register structure and rendered population must match its page response");
    if (scenario.privacyMatrix && summary) fail("register entry must not load an unused overview summary");
    if (width === 390 && (layout.tablePresentation !== "block" || !privacyRect(layout.firstRow) || layout.firstRowCells?.length !== 6
      || layout.firstRowCells.some((cell) => !inside(cell.bounds, layout.firstRow) || cell.valuesContained !== true
        || cell.labelContent !== cell.label || cell.columns !== (cell.fullWidth ? 1 : 2)))) fail("phone register must replace the wide table with contained labelled key/value records");
  }
  if (scenario.privacyView === "overview" && check !== "denied") {
    const counts = summary?.summary?.counts;
    if (!counts || layout.metrics.length !== 4 || !same(layout.metrics.map((metric) => metric.label), ["Processing activities", "In progress", "Reviews overdue", "Missing lawful basis"])
      || !same(layout.metrics.map((metric) => metric.value), [String(counts?.total), ...[counts?.open, counts?.review_overdue, counts?.missing_lawful_basis].map((value) => counts?.total === 0 ? "—" : String(value))])
      || layout.metrics.some((metric) => !privacyRect(metric.bounds)) || layout.metricColumns !== (width > 1230 ? 4 : 2)
      || !privacyRect(layout.primary) || !privacyRect(layout.aside)
      || !same(layout.primaryHeadings, ["Activity status", "Record completeness"])) fail("overview requires four genuine measures, status/completeness panels and coverage");
    const expected = width > 1230 ? "beside" : "stacked";
    if (layout.placement !== expected || expected === "beside" && (layout.primary.right > layout.aside.x + 1 || Math.abs(layout.aside.width - 285) > 2)
      || expected === "stacked" && (layout.primary.bottom > layout.aside.y + 1 || Math.abs(layout.primary.x - layout.aside.x) > 1)) fail("overview context must use the source responsive replacement");
    const expectedQuality = check === "stale" ? "stale" : check === "partial" ? "partial" : ["empty", "unknown"].includes(check) ? "unknown" : "current";
    if (layout.metrics.some((metric) => metric.quality !== expectedQuality)) fail("unknown, stale, empty and partial measures must preserve their actual quality");
    if (check === "unknown" && (!layout.coverageText.includes("excluded: unknown") || !layout.coverageText.includes("unknown records: unknown"))) fail("missing coverage must remain unknown");
    if (check === "partial" && (!layout.coverageText.includes("1 excluded") || !layout.coverageText.includes("1 unknown"))) fail("partial coverage counts must remain visible");
  }
  if (check === "denied" && (layout.recordNames.length || layout.metrics.length || !layout.text.includes("unavailable with your current access") || !observed?.reads.some((read) => read.status === 403))) fail("denied access must hide records and counts");
  if (check === "empty" && (!layout.text.includes("No activities") || layout.recordNames.length || !observed?.reads.some((read) => read.records?.length === 0 || read.summary?.counts.total === 0))) fail("empty state must match a successfully read empty population");
  if (check === "recovery" && (capture.recovery?.retry !== true || !same(observed?.reads.filter((read) => read.path.endsWith("/processing-activities")).map((read) => read.status), [503, 200]))) fail("recovery requires a failed read followed by a successful user retry");
  if (["page-empty", "page-error"].includes(check)) {
    const recovery = capture.pageRecovery, empty = check === "page-empty";
    if (!recovery || recovery.before.recordNames.length || !recovery.before.text.includes(empty ? "No more activities" : "Couldn’t load activities.")
      || empty && (!recovery.before.text.includes("Return to the previous page.") || recovery.before.text.includes("No processing activities have been recorded"))
      || !privacyRect(recovery.previousBounds) || recovery.previousBounds.width < 44 || recovery.previousBounds.height < 44
      || recovery.beforeRead.status !== (empty ? 200 : 503) || !new URLSearchParams(recovery.beforeRead.query).has("cursor")
      || new URLSearchParams(list?.query).has("cursor") || !same(layout.recordNames, [privacyActivityName])
      || !/^[a-f0-9]{64}$/.test(recovery.imageSHA256) || recovery.imageBytes <= 0) fail("later empty/error page must retain previous-page recovery and its observed failed/empty state");
  }
  if (check === "history") failures.push(...privacyHistoryFailures(capture.history).map((message) => `${scenario.name}: ${message}`));
  if (check === "filters") failures.push(...privacyFilterFailures(capture.filters).map((message) => `${scenario.name}: ${message}`));
  return failures;
}

export function privacyHistoryFailures(history) {
  const records = history?.records ?? [], expected = ["back", "forward", "back-restored", "filtered-back", "peer-forward"];
  if (!history || !same(records.map((item) => item.action), expected) || !history.names?.length || history.retainedSearch !== "Customer"
    || !same(history.filtered, [privacyActivityName, "Customer service channel monitoring"]) || !same(history.restoredNames, history.names) || history.restored !== true
    || history.peerBack?.route !== "#ropa/overview" || history.peerBack.sameMenu !== true || history.peerBack.sameDocument !== true
    || history.directReload?.route !== `#ropa/activity/${privacyActivityID}` || history.directReload.newDocument !== true
    || records.some((item) => item.sameMenu !== true || item.sameDocument !== true || item.detail !== (item.action === "forward")
      || item.route !== (item.action === "forward" ? `#ropa/activity/${privacyActivityID}` : "#ropa")
      || item.action !== "forward" && !same(item.names, item.action === "filtered-back" ? history.filtered : history.names))) return ["Back/Forward, filtered return, peer navigation and direct reload require matching identities and preserved mounted views"];
  const observation = history.firstDocument;
  if (!observation?.reads?.length || !Array.isArray(observation.commands) || observation.commands.length
    || observation.reads.some((read) => new URLSearchParams(read.query).get("tenant_id") !== "bank-demo" || read.status !== 200)) return ["history before direct reload must retain scoped read-only observations"];
  return privacyCountedDrillFailures(history.countedDrills);
}

export function privacyCountedDrillFailures(drills) {
  const snapshots = drills?.snapshots ?? [], expected = ["total", "remove-scope", "complete", "clear"], counts = drills?.summary?.counts;
  if (!counts || counts.retired !== 1 || !same(snapshots.map((item) => item.step), expected)
    || drills.defaultNames.length !== counts.total - counts.retired) return ["counted drill exercise must distinguish all recorded and current activities"];
  for (const snapshot of snapshots) {
    const query = new URLSearchParams(snapshot.read?.query), counted = ["total", "complete"].includes(snapshot.step);
    const expectedCount = snapshot.step === "total" ? counts.total : snapshot.step === "complete" ? counts.closed : drills.defaultNames.length;
    if (snapshot.read?.status !== 200 || query.get("include_retired") !== (counted ? "true" : null) || query.has("cursor")
      || query.get("status") !== (snapshot.step === "complete" ? "CLOSED" : null) || snapshot.includesEnded !== counted
      || !same(snapshot.names, snapshot.read?.records?.map((row) => row.name)) || snapshot.names.length !== expectedCount
      || !counted && !same(snapshot.names, drills.defaultNames)
      || snapshot.step === "complete" && snapshot.read.records.some((row) => row.status !== "CLOSED")) return ["counted drills must include ended records, match the summary and remove scope on clear"];
  }
  return [];
}

export function privacyFilterFailures(filters) {
  const snapshots = filters?.snapshots ?? [], expected = ["initial", "search", "next", "status-reset", "status-next", "query-reset", "clear"];
  if (!same(snapshots.map((item) => item.step), expected) || filters.clearedSearch !== "") return ["filter/page exercise is incomplete"];
  for (const [index, snapshot] of snapshots.entries()) {
    const query = new URLSearchParams(snapshot.read?.query), search = index >= 1 && index <= 4 ? "Azure" : index === 5 ? "Customer" : null;
    if (snapshot.read?.status !== 200 || query.get("search") !== search || query.get("status") !== (index >= 3 && index <= 5 ? "OPEN" : null)
      || query.has("cursor") !== [2, 4].includes(index) || !same(snapshot.names, snapshot.read?.records?.map((item) => item.name)) || snapshot.names.length !== 1) return ["filters must reset stale cursors and bind every displayed page to its actual query response"];
  }
  return [];
}

export function reviewPrivacyCaptures(captures) {
  const byName = new Map(captures.map((item) => [item.name, item]));
  const failures = privacyEvidenceScenarios.flatMap((scenario) => byName.has(scenario.name) ? privacyCaptureFailures(byName.get(scenario.name), scenario) : [`${scenario.name}: Privacy evidence missing`]);
  for (const scenario of privacyEvidenceScenarios) {
    const capture = byName.get(scenario.name);
    if (capture && (!/^[a-f0-9]{64}$/.test(capture.imageSHA256) || !/^[a-f0-9]{64}$/.test(capture.buildDigest) || !(capture.imageBytes > 1024))) failures.push(`${scenario.name}: hash-bound rendered capture receipt missing`);
  }
  return { failures, check: { name: "Privacy v2 layouts, exact records and history", status: failures.length ? "FAIL" : "PASS", detail: `${privacyEvidenceScenarios.length} scoped states` } };
}

// Compare the retained original screen anatomy without pretending its sample
// departments, DPIAs, incidents or review decisions are backed runtime data.
export function privacySourcePairFailures(source, capture, scenario) {
  const failures = privacyCaptureFailures(capture, scenario), width = scenario.viewport.width;
  const fail = (message) => failures.push(`${scenario.name}: ${message}`);
  if (!source || source.kind !== scenario.privacyView || source.headingText !== "Privacy & data"
    || !same(source.localTabs, privacySourceTabs) || source.selectedTab !== privacyViews.find((view) => view.id === scenario.privacyView)?.label
    || !privacyRect(source.heading) || !privacyRect(source.menu) || source.scrollWidth > source.documentWidth + 1) {
    fail("original Privacy heading, six-item menu or same-screen selection is missing"); return failures;
  }
  if (scenario.privacyView === "overview") {
    const placement = source.primary && source.aside ? source.aside.x >= source.primary.right - 1 ? "beside" : "stacked" : null;
    if (source.metrics.length !== 4 || source.metrics.some((metric) => !privacyRect(metric.bounds))
      || source.metricColumns !== (width > 1230 ? 4 : 2) || !privacyRect(source.primary) || !privacyRect(source.aside)
      || placement !== capture.privacyLayout.placement) fail("original four-metric Overview and responsive primary/context anatomy is missing");
  } else if (!privacyRect(source.controls) || !privacyRect(source.table) || !privacyRect(source.firstIdentity)
    || source.firstName !== "Customer onboarding" || source.headers.length !== 7
    || width === 390 && source.tablePresentation !== "block") fail("original ROPA filter/table/labelled-phone-record anatomy is missing");
  if (scenario.privacyCheck === "detail" && (!privacyRect(source.inspector) || source.inspectorHeading !== "Customer onboarding"
    || Math.abs(source.inspector.width - Math.min(480, width)) > 1 || Math.abs(source.inspector.right - width) > 1)) fail("original selected ROPA record must be paired with the runtime inspector");
  return failures;
}
