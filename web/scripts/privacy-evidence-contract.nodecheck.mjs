import assert from "node:assert/strict";
import test from "node:test";
import { privacyActivityID, privacyActivityName, privacyEvidenceScenarios } from "./privacy-evidence-scenarios.mjs";
import { privacyCaptureFailures, privacyCountedDrillFailures, privacyFilterFailures, privacySourcePairFailures, privacySourceTabs, reviewPrivacyCaptures } from "./privacy-evidence-contract.mjs";

// Small measured-shape examples exercise rejection paths. They are not browser
// evidence and must never be saved into a rendered evidence manifest.
const rect = (x, y, width, height) => ({ x, y, width, height, right: x + width, bottom: y + height });
const rows = [{ id: privacyActivityID, name: privacyActivityName, status: "OPEN" }];
const listRead = (records = rows, query = "?tenant_id=bank-demo") => ({ path: "/api/v1/ropa/processing-activities", query, status: 200, records });
const counts = { total: 8, open: 7, closed: 1, retired: 1, review_overdue: 3, missing_lawful_basis: 2 };
function fixture(view = "register", width = 1440, check = "layout") {
  const scenario = privacyEvidenceScenarios.find((item) => item.privacyView === view && item.viewport.width === width && item.theme === "light" && item.privacyCheck === check);
  const heading = rect(80, 100, 300, 100), menu = rect(80, 220, 300, 44), firstUseful = rect(90, 410, 180, 24);
  const layout = { evidenceKind: "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION", kind: view, route: scenario.route,
    heading, headingText: "Privacy & data", headingCount: 1, menu, localTabs: ["Overview", "ROPA"], selectedTab: view === "overview" ? "Overview" : "ROPA",
    selectedBounds: rect(80, 220, 100, 44), selectedVisible: true, selectedPanelID: view, visiblePanelIDs: [view], primaryCurrent: ["Privacy"],
    controls: rect(80, 300, 300, 50), table: rect(80, 370, 300, 600), tablePresentation: width === 390 ? "block" : "table",
    headers: ["Processing activity", "Data subjects", "Lawful basis", "Next review", "Owner", "Status", "Row action"], recordNames: [privacyActivityName],
    firstUseful, firstUsefulVisible: true, firstRow: rect(80, 390, 300, 400),
    firstRowCells: ["Processing activity", "Data subjects", "Lawful basis", "Next review", "Owner", "Status"].map((label, i) => ({ label, bounds: rect(96, 410 + i * 40, 268, 40), fullWidth: i === 0, columns: i === 0 ? 1 : 2, labelContent: label, valuesContained: true })),
    metrics: [], minTargetWidth: 44, minTargetHeight: 44, undersizedTargets: [], text: "Processing activities",
  };
  let reads = [listRead()];
  if (view === "overview") {
    const metrics = ["Processing activities", "In progress", "Reviews overdue", "Missing lawful basis"].map((label, i) => ({ label, value: String([8, 7, 3, 2][i]), quality: "current", bounds: rect(80 + i * 100, 300, 90, 132) }));
    Object.assign(layout, { metrics, metricColumns: 4, primary: rect(80, 450, 750, 300), aside: rect(850, 450, 285, 300), placement: "beside", primaryHeadings: ["Activity status", "Record completeness"], coverageText: "8 checked · 0 excluded · 0 unknown" });
    reads = [{ path: "/api/v1/ropa/dashboard", query: "?tenant_id=bank-demo", status: 200, summary: { counts } }];
  }
  return { scenario, capture: { ...scenario, privacyLayout: layout, observation: { reads, commands: [] }, metrics: { clientWidth: width, scrollWidth: width }, errors: [] } };
}

test("Privacy requires its entire bounded matrix and a hash-bound rendered receipt", () => {
  assert.equal(new Set(privacyEvidenceScenarios.map((item) => item.name)).size, 45);
  assert.equal(reviewPrivacyCaptures([]).failures.length, 45);
  const { scenario, capture } = fixture();
  assert.deepEqual(privacyCaptureFailures(capture, scenario), []);
  assert.ok(reviewPrivacyCaptures([capture]).failures.some((failure) => failure.includes("hash-bound rendered capture receipt")));
});

test("a route or screenshot cannot substitute for heading, menu and first useful geometry", () => {
  const { scenario, capture } = fixture();
  for (const mutation of [
    (item) => { item.privacyLayout.localTabs.push("DPIAs"); },
    (item) => { item.privacyLayout.firstUseful = rect(90, 950, 180, 24); },
    (item) => { item.privacyLayout.firstUseful.bottom += 200; },
    (item) => { item.privacyLayout.selectedBounds.x = 0; },
    (item) => { item.privacyLayout.primaryCurrent = ["Risk & Losses"]; },
  ]) { const changed = structuredClone(capture); mutation(changed); assert.ok(privacyCaptureFailures(changed, scenario).length); }
});

test("Privacy rows require response identity, scoped reads and no commands", () => {
  const { scenario, capture } = fixture();
  for (const mutation of [
    (item) => { item.observation.reads[0].query = "?tenant_id=other-bank"; },
    (item) => { item.observation.reads[0].records[0].name = "Another activity"; },
    (item) => { item.observation.commands.push({ method: "POST", path: "/api/v1/ropa/processing-activities" }); },
  ]) { const changed = structuredClone(capture); mutation(changed); assert.ok(privacyCaptureFailures(changed, scenario).length); }
});

test("phone record proof rejects unlabeled, clipped or single-column metadata", () => {
  const { scenario, capture } = fixture("register", 390);
  assert.deepEqual(privacyCaptureFailures(capture, scenario), []);
  for (const mutation of [
    (item) => { item.privacyLayout.firstRowCells[1].labelContent = ""; },
    (item) => { item.privacyLayout.firstRowCells[1].valuesContained = false; },
    (item) => { item.privacyLayout.firstRowCells[1].columns = 1; },
  ]) { const changed = structuredClone(capture); mutation(changed); assert.ok(privacyCaptureFailures(changed, scenario).length); }
});

test("Overview count, quality and context geometry must match supported responses", () => {
  const { scenario, capture } = fixture("overview");
  assert.deepEqual(privacyCaptureFailures(capture, scenario), []);
  for (const mutation of [
    (item) => { item.privacyLayout.metrics[0].value = "128"; },
    (item) => { item.privacyLayout.metrics[1].label = "DPIAs awaiting decision"; },
    (item) => { item.privacyLayout.metrics[0].quality = "unknown"; },
    (item) => { item.privacyLayout.metricColumns = 2; },
    (item) => { item.privacyLayout.aside = rect(80, 800, 300, 300); },
  ]) { const changed = structuredClone(capture); mutation(changed); assert.ok(privacyCaptureFailures(changed, scenario).length); }
});

test("immutable original Overview comparison records six tabs and supported source anatomy", () => {
  const { scenario, capture } = fixture("overview"), runtime = capture.privacyLayout;
  const source = { kind: "overview", headingText: "Privacy & data", heading: runtime.heading, menu: runtime.menu,
    localTabs: privacySourceTabs, selectedTab: "Overview", documentWidth: 1440, scrollWidth: 1440,
    metrics: runtime.metrics, metricColumns: 4, primary: runtime.primary, aside: runtime.aside };
  assert.deepEqual(privacySourcePairFailures(source, capture, scenario), []);
  assert.ok(privacySourcePairFailures({ ...source, localTabs: ["Overview", "ROPA"] }, capture, scenario).length);
  assert.ok(privacySourcePairFailures({ ...source, primary: null }, capture, scenario).length);
});

test("summary drills include ended records and reset the scope when removed or cleared", () => {
  const active = Array.from({ length: 7 }, (_, i) => ({ id: `a${i}`, name: `Activity ${i}`, status: "OPEN" }));
  const ended = { id: "ended", name: "Ended activity", status: "CLOSED" }, all = [...active, ended];
  const snapshot = (step, records, query, includesEnded) => ({ step, names: records.map((row) => row.name), read: listRead(records, `?tenant_id=bank-demo${query}`), includesEnded });
  const drills = { summary: { counts }, defaultNames: active.map((row) => row.name), snapshots: [
    snapshot("total", all, "&include_retired=true", true), snapshot("remove-scope", active, "", false),
    snapshot("complete", [ended], "&include_retired=true&status=CLOSED", true), snapshot("clear", active, "", false),
  ] };
  assert.deepEqual(privacyCountedDrillFailures(drills), []);
  for (const mutation of [
    (item) => { item.snapshots[0].read.query = "?tenant_id=bank-demo"; },
    (item) => { item.snapshots[0].names.pop(); },
    (item) => { item.snapshots[3].includesEnded = true; },
    (item) => { item.snapshots[1].read.query += "&cursor=old-page"; },
  ]) { const changed = structuredClone(drills); mutation(changed); assert.ok(privacyCountedDrillFailures(changed).length); }
});

test("search and status changes cannot reuse an earlier page cursor", () => {
  const steps = ["initial", "search", "next", "status-reset", "status-next", "query-reset", "clear"];
  const filters = { clearedSearch: "", snapshots: steps.map((step, i) => ({ step, names: [privacyActivityName], read: listRead(rows,
    `?tenant_id=bank-demo${i >= 1 && i <= 4 ? "&search=Azure" : i === 5 ? "&search=Customer" : ""}${i >= 3 && i <= 5 ? "&status=OPEN" : ""}${[2, 4].includes(i) ? "&cursor=next" : ""}`) })) };
  assert.deepEqual(privacyFilterFailures(filters), []);
  filters.snapshots[3].read.query += "&cursor=stale";
  assert.ok(privacyFilterFailures(filters).length);
});
