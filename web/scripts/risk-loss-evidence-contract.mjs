import { riskLossEvidenceScenarios, riskLossViews } from "./risk-loss-evidence-scenarios.mjs";

const same = (left, right) => JSON.stringify(left) === JSON.stringify(right);
const rect = (value) => value && ["x", "y", "width", "height", "right", "bottom"].every((key) => Number.isFinite(value[key]))
  && value.width > 0 && value.height > 0 && Math.abs(value.x + value.width - value.right) <= 1 && Math.abs(value.y + value.height - value.bottom) <= 1;
const labels = riskLossViews.map((view) => view.label);
const endpoints = { risks: "/api/v1/risks", rcsa: "/api/v1/rcsa/cycles", insights: "/api/v1/risk-indicators", losses: "/api/v1/losses" };

export function riskLossStatusBadgesFit(layout) {
  const contains = (outer, inner) => rect(outer) && rect(inner)
    && inner.x >= outer.x - 1 && inner.y >= outer.y - 1 && inner.right <= outer.right + 1 && inner.bottom <= outer.bottom + 1;
  const columns = { risks: ["Rating", "Appetite", "Status"], insights: ["Condition"] }[layout.kind];
  return Boolean(columns && Array.isArray(layout.statusBadges) && layout.statusBadges.length === layout.records * columns.length
    && layout.statusBadges.every((badge, index) => badge.rowIndex === Math.floor(index / columns.length) && badge.column === columns[index % columns.length]
      && typeof badge.label === "string" && badge.label.trim().length > 0 && badge.label === layout.rowValues?.[badge.rowIndex]?.[badge.column]
      && Number.isInteger(badge.lines) && badge.lines > 0 && badge.lines <= 2 && (/\s/.test(badge.label) || badge.lines === 1)
      && Array.isArray(badge.words) && badge.words.map((word) => word.text).join(" ") === badge.label.replace(/\s+/g, " ")
      && badge.words.every((word) => word.lines === 1)
      && contains(badge.cell, badge.bounds) && contains(badge.bounds, badge.textBounds)));
}

export function riskLossRCSAPopulationMatches(detail) {
  const population = detail.rcsaPopulation;
  return Boolean(population && ["Risks", "Controls"].every((kind) => /^\d+$/.test(population.card?.[kind] ?? "")
    && Number.isInteger(population.detailRows?.[kind]) && population.detailRows[kind] >= 0
    && Number(population.card[kind]) === population.detailRows[kind])
    && population.detailSummary === `${population.detailRows.Risks} Risks · ${population.detailRows.Controls} controls`);
}

export function reviewRiskLossCaptures(captures) {
  const failures = [], byName = new Map(captures.map((capture) => [capture.name, capture]));
  for (const scenario of riskLossEvidenceScenarios) {
    const capture = byName.get(scenario.name);
    if (!capture) { failures.push(`${scenario.name}: Risk & Losses evidence is missing`); continue; }
    const layout = capture.riskLossLayout, observed = capture.observation, check = scenario.riskLossCheck;
    const fail = (message) => failures.push(`${scenario.name}: ${message}`);
    if (capture.baseline !== false || ["route", "fixture", "state", "theme", "riskLossView", "riskLossCheck", "sourceComparison"].some((key) => capture[key] !== scenario[key])
      || !same(capture.viewport, scenario.viewport) || !layout || layout.kind !== scenario.riskLossView
      || layout.evidenceKind !== "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION"
      || !/^[a-f0-9]{64}$/.test(capture.imageSHA256 ?? "") || !(capture.imageBytes > 0)) {
      fail("metadata, screenshot binding, source comparison or synthetic evidence boundary is inconsistent");
      if (!layout) continue;
    }
    if (!observed || !Array.isArray(observed.reads) || !Array.isArray(observed.commands) || observed.commands.length || observed.pending !== 0
      || layout.pendingReads !== 0 || !Array.isArray(layout.commands) || layout.commands.length
      || layout.primaryCurrent?.join("|") !== "Risk & Losses" || layout.documentWidth !== scenario.viewport.width || layout.scrollWidth > layout.documentWidth + 1) {
      fail("settled reads, zero business commands, primary owner or document geometry is unproven");
    }
    const listReads = (observed?.reads ?? []).filter((read) => read.path === endpoints[scenario.riskLossView] && !read.query.check_id);
    const expectedStatus = check === "denied" ? 403 : 200;
    if (!listReads.some((read) => read.method === "GET" && read.status === expectedStatus && read.query.limit === (scenario.riskLossView === "insights" ? "50" : "25"))) {
      fail("bounded collection read and its actual response status are missing");
    }
    if (check === "detail") {
      const detail = capture.detail;
      const exactRoute = scenario.riskLossView === "insights" ? detail?.route?.split("?")[0] === "#insights" && new URLSearchParams(detail.route.split("?")[1]).get("indicator") === scenario.detailID
        : detail?.route === `#${scenario.riskLossView}/${scenario.detailID}`;
      if (!exactRoute || detail?.exactTarget !== true || detail.titleMatches !== true || detail.titleVisible !== true || detail.pendingReads !== 0 || detail.commands?.length !== 0
        || detail.placement !== (["rcsa", "insights"].includes(scenario.riskLossView) ? "sheet" : "standalone")
        || !(detail.width > 0 && detail.width <= scenario.viewport.width + 1)
        || ["rcsa", "insights"].includes(scenario.riskLossView) && (Math.abs(detail.width - Math.min(480, scenario.viewport.width)) > 1
          || Math.abs(detail.x + detail.width - scenario.viewport.width) > 1 || Math.abs(detail.height - scenario.viewport.height) > 1
          || detail.closeVisible !== true || detail.focusContained !== true || detail.primaryAction?.visible !== true || !rect(detail.close) || detail.close.width < 44 || detail.close.height < 44
          || !Array.isArray(detail.internalTables) || detail.internalTables.some((table) => table.scrollWidth > table.clientWidth + 1 || table.presentation !== "block"))) fail("exact record reader, source 480px inspector, container-aware tables or settled read boundary is missing");
      if (scenario.riskLossView === "rcsa" && !riskLossRCSAPopulationMatches(detail ?? {})) fail("RCSA inspector frozen population must match the selected card, rendered summary and detail rows");
      continue;
    }
    if (layout.route !== scenario.route || layout.headingText !== "Risk & Losses" || !rect(layout.heading) || !rect(layout.menu)
      || !same(layout.localTabs, labels) || layout.selectedTab !== scenario.label || layout.menuRows !== 1 || layout.selectedTabVisible !== true
      || layout.heading.bottom > layout.menu.y + 1 || layout.dialogVisible !== false) {
      fail("four-route scrolling menu, current section or collection heading is missing, wrapped or clipped");
    }
    if (["empty", "denied"].includes(check)) {
      if (layout.records !== 0 || layout.populatedIDs?.length !== 0 || layout.recordTitles?.length !== 0
        || check === "empty" && !(observed?.responses ?? []).some((response) => response.ids.length === 0)) fail("empty or denied collection leaked records or lacks observed zero results");
      continue;
    }
    const latestResponse = observed?.responses?.at(-1);
    if (!rect(layout.firstRecord) || !rect(layout.firstIdentity) || layout.firstUsefulContentVisible !== true
      || layout.firstIdentity.y < 0 || layout.firstIdentity.bottom > scenario.viewport.height + 1 || layout.menu.bottom > layout.firstRecord.y + 1
      || !(layout.records > 0) || layout.populatedIDs?.length !== layout.records || new Set(layout.populatedIDs).size !== layout.records
      || !same(layout.populatedIDs, latestResponse?.ids) || !same(layout.recordTitles, latestResponse?.titles)) {
      fail("first useful record is below the viewport or visible records do not bind to the settled response population");
    }
    if (scenario.riskLossView === "rcsa") {
      const columns = scenario.viewport.width > 1190 ? 3 : scenario.viewport.width > 600 ? 2 : 1;
      if (layout.presentation !== "cards" || layout.columns !== columns || !rect(layout.grid)
        || layout.rowValues?.[0]?.facts?.Risks !== "3" || layout.rowValues?.[0]?.facts?.Controls !== "2"
        || layout.cardNumbers?.length !== 2 || layout.cardNumbers.some((number) => number.fontSize < 30 || number.labelFontSize > 14 || number.aligned !== true || !["start", "left"].includes(number.textAlign))) fail("RCSA source card columns or frozen population counts are missing");
    } else if (!rect(layout.table) || !["table", "stacked-records"].includes(layout.presentation)
      || scenario.viewport.width === 390 && layout.presentation !== "stacked-records") fail("register table did not adapt to readable phone records");
    const row = layout.rowValues?.[0] ?? {};
    if (!layout.rowActions?.length || layout.rowActions.some((action) => !rect(action.bounds) || action.bounds.width < 44 || action.lines > 2)
      || scenario.riskLossView !== "rcsa" && scenario.viewport.width === 390 && layout.firstRecordColumns !== 2) fail("record actions must stay readable and phone facts must use two source-aligned columns");
    if (["risks", "insights"].includes(scenario.riskLossView) && !riskLossStatusBadgesFit(layout)) fail("status badges must keep whole words on at most two lines with text contained in the badge and cell");
    if (scenario.riskLossView === "insights") {
      if (check === "partial" ? !row["Current value"]?.includes("Native value unavailable") || row.Condition !== "Unknown"
        : !row["Current value"]?.includes("98.70%") || !row["Limit / target"]?.includes("99.50%") || !/0\.40\s*pp/.test(row.Movement ?? "")) {
        fail("native percentage, approved limit, percentage-point movement or unknown measurement semantics were lost");
      }
    }
    if (scenario.riskLossView === "losses") {
      if (!["Gross", "Recovered", "Net"].every((name) => layout.tableHeaders?.includes(name))
        || !row.Gross?.includes("18,400,000.00") || !row.Recovered?.includes("8,000,000.00") || !row.Net?.includes("10,400,000.00")) fail("gross, recovered and net values must remain separate exact currency amounts");
      if (check === "partial" && (!layout.rowValues?.[1]?.Gross?.includes("1,200.00") || !layout.rowValues[1].Gross.includes("$")
        || !layout.rowValues[1].Net?.includes("1,000.00") || !row.Gross.includes("₦"))) fail("mixed currencies were combined or their recorded amounts changed");
    }
    if (check === "partial") {
      if (!latestResponse?.nextCursor) fail("partial population needs an observed remaining cursor");
      if (scenario.riskLossView === "risks" && !Object.values(row).includes("Reassessment needed")) fail("stale risk assessment must be unknown until reassessed");
      if (["rcsa", "insights"].includes(scenario.riskLossView) && (latestResponse?.complete !== false || !layout.stateText.includes("unavailable"))) fail("partial reader completeness must remain disclosed");
    }
    if (layout.disclosures?.some((disclosure) => disclosure.open)) fail("secondary summaries must not displace first useful records");
    if (check === "history") {
      const history = capture.history, restored = history?.backLayout;
      if (history?.actualBack !== true || history.sameDocument !== true || history.noSelectedDetail !== true
        || history.backRoute !== scenario.route || history.selectedMenu !== scenario.label || history.pendingReads !== 0
        || !history.initialDocumentToken || history.backDocumentToken !== history.initialDocumentToken || history.forwardDocumentToken !== history.initialDocumentToken
        || !history.populationBefore?.length || !same(history.populationBefore, history.populationAfter)
        || restored?.route !== scenario.route || restored.dialogVisible !== false || restored.pendingReads !== 0 || restored.selectedTab !== scenario.label
        || !same(restored.populatedIDs, history.populationBefore) || restored.records !== history.populationBefore.length || restored.firstUsefulContentVisible !== true
        || history.forwardExactTarget !== true || !history.forwardRoute?.includes(scenario.detailID) || history.explicitReturn !== true || history.commands?.length !== 0) {
        fail("real Back/Forward must restore the same settled populated collection and exact record in the same document");
      }
    }
    if (check === "filters") {
      const filters = capture.filters;
      const pages = filters?.pages;
      const pageComplete = (page) => Array.isArray(page?.ids) && Array.isArray(page.titles) && page.ids.length === page.titles.length
        && new Set(page.ids).size === page.ids.length && page.query && typeof page.query === "object";
      const filterField = scenario.riskLossView === "rcsa" ? "status" : scenario.riskLossView === "insights" ? "kind" : "search";
      const filterValue = { risks: "network", losses: "merchant", rcsa: "AWAITING_CHALLENGE", insights: "KCI" }[scenario.riskLossView];
      if (filters?.nextPageDistinct !== true || filters.previousPageRestored !== true || filters.filterSentToReader !== true || filters.phoneFiltersAccessible !== true || filters.commands?.length !== 0
        || !["before", "next", "previous", "filtered", "restored"].every((name) => pageComplete(pages?.[name]))
        || !pages?.before?.ids?.length || !pages?.next?.ids?.length || pages.next.ids.some((id) => pages.before.ids.includes(id))
        || !same(pages?.previous?.ids, pages?.before?.ids) || !same(pages?.previous?.titles, pages?.before?.titles)
        || !same(pages?.restored?.ids, layout.populatedIDs) || !same(pages?.restored?.titles, layout.recordTitles)
        || !same(pages?.restored?.ids, pages?.before?.ids) || !same(pages?.restored?.titles, pages?.before?.titles)
        || pages?.next?.query?.cursor !== "sample-risk-loss-next-page" || pages?.previous?.query?.cursor || pages?.filtered?.query?.cursor || pages?.restored?.query?.cursor
        || pages?.filtered?.query?.[filterField] !== filterValue
        || (pages?.restored?.query?.[filterField] ?? null) !== (scenario.riskLossView === "insights" ? "KRI" : null)
        || !listReads.some((read) => read.query.cursor === "sample-risk-loss-next-page" && read.status === 200)
        || !listReads.some((read) => scenario.riskLossView === "risks" ? read.query.search === "network" : scenario.riskLossView === "losses" ? read.query.search === "merchant"
          : scenario.riskLossView === "rcsa" ? read.query.status === "AWAITING_CHALLENGE" : read.query.kind === "KCI")) fail("real cursor paging and selected filters must reach the bounded reader");
    }
    if (scenario.verifyRiskLossNavigation && (!same(capture.navigation?.visited?.map((item) => item.label), labels)
      || !same(capture.navigation?.keyboard, [...labels.slice(1), labels[0]]) || capture.navigation.restoredRoute !== scenario.route
      || capture.navigation.selectedTabVisible !== true || !same(capture.navigation.restoredPopulation, layout.populatedIDs) || capture.navigation.commands?.length !== 0)) {
      fail("all four local destinations need click, arrow-key and populated return evidence");
    }
  }
  return { failures, check: { name: "Risk & Losses source layout and bounded record workflows", status: failures.length ? "FAIL" : "PASS",
    detail: `${riskLossEvidenceScenarios.filter((scenario) => byName.has(scenario.name)).length}/${riskLossEvidenceScenarios.length} captures; four routes at five widths in two themes, exact readers, real Back/Forward, filters/paging, empty, denied and partial populations; Risks is an explicit table-anatomy adaptation` } };
}
