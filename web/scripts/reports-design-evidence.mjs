// Browser assertions for the source-matched Reports views. Fixture values are
// intentionally asserted independently of the React implementation.
import { reportExploreFixtureSource } from "./reports-design-evidence-contract.mjs";
const stableTabs = ["Overview", "Explore", "Reports", "Schedules"];
const escape = (value) => value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

export function reportSelect(page, root, label) {
  return root.locator(".cs-select-field").filter({
    has: page.locator(".cs-select-field__label").filter({ hasText: new RegExp(`^${escape(label)}(?: \\*)?$`) }),
  });
}

export async function chooseReportField(page, root, label, value) {
  const field = reportSelect(page, root, label);
  await field.locator(".cs-select-field__trigger").click();
  await page.getByRole("option").filter({ has: page.locator("span").filter({ hasText: new RegExp(`^${escape(value)}$`) }) }).click();
  await field.locator(".cs-select-field__value").filter({ hasText: new RegExp(`^${escape(value)}$`) }).waitFor({ state: "visible" });
}

export async function chooseReportExploreMeasure(page, label) {
  await chooseReportField(page, page.locator(".reports-explore"), "Measure", label);
  await page.locator(".reports-explore__chart").getByRole("heading", { name: label, exact: true }).waitFor({ state: "visible" });
}

export async function selectReportOrganizationScope(page) {
  await page.getByRole("button", { name: /^Organization scope,/ }).click();
  await page.getByRole("dialog", { name: "Change organization scope" })
    .getByRole("button", { name: /^Risk Operations/ }).click();
  await page.getByRole("button", { name: /^Organization scope,.*Risk Operations/ }).waitFor({ state: "visible" });
}

async function resetViewportScroll(page) {
  await page.evaluate(async () => {
    window.scrollTo({ top: 0, left: 0, behavior: "instant" });
    await new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve)));
  });
}

async function assertLocalTabs(page, selected, name) {
  const tablist = page.getByRole("tablist", { name: "Reports workspace" });
  const expected = [...stableTabs, ...(selected === "Build report" ? ["Build report"] : [])];
  const labels = (await tablist.getByRole("tab").allTextContents()).map((value) => value.trim());
  if (labels.join("|") !== expected.join("|")
      || await tablist.getByRole("tab", { selected: true }).textContent() !== selected
      || await page.locator(".reports-workspace .cs-tabs__compact").count()) {
    throw new Error(`${name}: Reports local navigation must expose ${expected.join(" / ")} with ${selected} selected`);
  }
  return { localTabs: labels, selectedTab: selected };
}

async function measureReportsView(page, kind) {
  return page.evaluate((kind) => {
    const root = document.querySelector(kind === "explore" ? ".reports-explore" : ".report-builder");
    const box = (node) => {
      const rect = node?.getBoundingClientRect();
      return rect ? { x: rect.x, y: rect.y, width: rect.width, height: rect.height, right: rect.right, bottom: rect.bottom } : null;
    };
    const visible = (node) => Boolean(node?.getClientRects().length && getComputedStyle(node).visibility !== "hidden"
      && !node.closest('[aria-hidden="true"], [inert]'));
    const targets = [...root.querySelectorAll("button, input:not([type=hidden]), summary"),
      ...document.querySelectorAll('.reports-workspace__actions button, .reports-workspace [role="tab"]')]
      .filter(visible).map((node) => ({ label: node.getAttribute("aria-label") || node.textContent?.trim() || node.getAttribute("placeholder"), ...box(node) }));
    const tabs = box(document.querySelector('.reports-workspace [role="tablist"]'));
    const selectedTabBounds = box(document.querySelector('.reports-workspace [role="tab"][aria-selected="true"]'));
    const common = { kind, heading: box(root.querySelector(kind === "explore" ? ".reports-explore__heading" : ".report-builder__heading")),
      tabs, selectedTabBounds, selectedTabVisible: Boolean(tabs && selectedTabBounds
        && selectedTabBounds.x >= Math.max(0, tabs.x) - 1 && selectedTabBounds.right <= Math.min(innerWidth, tabs.right) + 1),
      horizontalOverflow: document.documentElement.scrollWidth - document.documentElement.clientWidth,
      minTargetWidth: Math.min(...targets.map((target) => target.width)), minTargetHeight: Math.min(...targets.map((target) => target.height)),
      targets, reads: window.reportingEvidenceReads ?? [] };
    if (kind === "explore") {
      const chart = root.querySelector(".reports-explore__chart");
      const firstRow = box(chart.querySelector(".cs-ranked-bars__row"));
      const firstControl = box(root.querySelector(".reports-explore__filters .cs-select-field__trigger"));
      const exactValuesContained = [...root.querySelectorAll(".cs-ranked-bars__value, .reports-explore__total strong, .reports-explore__table td[data-kind=number]")].every((node) => {
        const boundary = node.closest(".reports-explore__chart, .reports-explore__table").getBoundingClientRect();
        const range = document.createRange();
        range.selectNodeContents(node);
        return [...range.getClientRects()].every((rect) => rect.x >= boundary.x - 1 && rect.right <= boundary.right + 1 && rect.y >= boundary.y - 1 && rect.bottom <= boundary.bottom + 1);
      });
      return { ...common, filters: box(root.querySelector(".reports-explore__filters")), chart: box(chart),
        table: box(root.querySelector(".reports-explore__table")), firstRow, firstControl,
        firstUsefulContentVisible: Boolean(firstControl && firstControl.y >= 0 && firstControl.bottom <= innerHeight
          && (innerWidth <= 700 || !firstRow || firstRow.y < innerHeight)),
        firstChartRowVisible: Boolean(firstRow && firstRow.y < innerHeight && firstRow.bottom > 0),
        chartState: chart.getAttribute("data-state"), dataQuality: chart.getAttribute("data-quality"), exactValuesContained,
        chartLabel: chart.getAttribute("aria-label"),
        chartRows: [...chart.querySelectorAll(".cs-ranked-bars__row")].map((row) => ({
          label: row.querySelector(".cs-ranked-bars__label")?.textContent?.trim(), value: row.querySelector(".cs-ranked-bars__value")?.textContent?.trim(),
        })),
        tableRows: [...root.querySelectorAll('.reports-explore__table tbody tr')].map((row) => {
          const cells = row.querySelectorAll("td");
          return { label: cells[0]?.textContent?.trim(), value: cells[1]?.textContent?.trim() };
        }),
        measure: root.querySelector(".reports-explore__fields .cs-select-field__value")?.textContent?.trim(),
        scopeText: root.querySelector(".reports-explore__scope")?.textContent?.trim(),
        coverageText: root.querySelector(".reports-explore__coverage")?.textContent?.trim() ?? null,
        sourceText: root.querySelector(".reports-explore__source")?.textContent?.trim() ?? null,
        total: root.querySelector(".reports-explore__total strong")?.textContent?.trim() ?? null,
        emptyHeading: root.querySelector(".reports-explore__chart .cs-empty-state__title")?.textContent?.trim() ?? null,
        currency: [...root.querySelectorAll(".cs-select-field")].find((field) => field.querySelector(".cs-select-field__label")?.textContent.trim() === "Currency")
          ?.querySelector(".cs-select-field__value")?.textContent.trim() ?? null,
        expectedLoss: new Intl.NumberFormat(undefined, { style: "currency", currency: "NGN", currencyDisplay: "narrowSymbol" }).format(10400000),
      };
    }
    const firstControl = box(root.querySelector(".report-builder__setup .cs-select-field__trigger"));
    return { ...common, setup: box(root.querySelector(".report-builder__setup")), paper: box(root.querySelector(".report-builder__paper")),
      output: box(root.querySelector(".report-builder__output")), firstControl,
      firstUsefulContentVisible: Boolean(firstControl && firstControl.y >= 0 && firstControl.bottom <= innerHeight),
      sectionCount: root.querySelectorAll(".report-builder__paper-section").length,
      contents: [...root.querySelectorAll(".report-builder__contents strong")].map((node) => node.textContent.trim()),
      scopeText: root.querySelector(".report-builder__paper-heading p")?.textContent?.trim(),
      previewTitle: root.querySelector(".report-builder__paper-heading h3")?.textContent?.trim(),
      outputText: root.querySelector(".report-builder__output")?.textContent,
      renderedDataRows: root.querySelectorAll(".report-builder__paper tbody tr, .report-builder__paper .cs-ranked-bars__row").length,
    };
  }, kind);
}

function assertViewBounds(layout, capture) {
  if (!layout.heading || !layout.tabs || layout.tabs.bottom > layout.heading.y + 2 || !layout.selectedTabVisible
      || !layout.firstUsefulContentVisible || layout.horizontalOverflow > 1) {
    throw new Error(`${capture.name}: local tabs, view heading and first useful controls must be visible and bounded: ${JSON.stringify(layout)}`);
  }
  if (capture.touch && (layout.minTargetWidth < 44 || layout.minTargetHeight < 44)) {
    throw new Error(`${capture.name}: Reports exposes an undersized touch control: ${JSON.stringify(layout.targets.filter((target) => target.width < 44 || target.height < 44))}`);
  }
}

export async function assertReportsExploreLayout(page, capture) {
  const root = page.locator(".reports-explore");
  await root.getByRole("heading", { name: "Explore measures", exact: true }).waitFor({ state: "visible" });
  await root.locator('.reports-explore__chart:not([data-state="loading"])').waitFor({ state: "visible" });
  const tabs = await assertLocalTabs(page, "Explore", capture.name);
  const label = capture.reportExploreMeasure ?? "Risks outside appetite";
  for (const fieldLabel of ["Breakdown", "Comparison"]) {
    if (!await reportSelect(page, root, fieldLabel).locator("button").isDisabled()) {
      throw new Error(`${capture.name}: ${fieldLabel} must not offer an unsupported analysis`);
    }
  }
  await resetViewportScroll(page);
  const layout = await measureReportsView(page, "explore");
  assertViewBounds(layout, capture);
  const beside = capture.viewport.width > 700;
  if (!layout.filters || !layout.chart || layout.heading.bottom > layout.filters.y + 1
      || (beside && (Math.abs(layout.filters.width - (capture.viewport.width <= 900 ? 200 : 240)) > 1
        || layout.chart.x < layout.filters.right || Math.abs(layout.chart.y - layout.filters.y) > 2))
      || (!beside && layout.chart.y < layout.filters.bottom)
      || (layout.table && layout.table.y < layout.chart.bottom)) {
    throw new Error(`${capture.name}: Explore must place its Data rail ${beside ? "beside" : "above"} one chart and a following breakdown: ${JSON.stringify(layout)}`);
  }
  const isRisk = label === "Risks outside appetite";
  const empty = capture.fixture === "report-explore-empty";
  const mixed = capture.fixture === "report-explore-mixed-currency";
  const live = ["report-overview-loaded", "report-explore-empty", "report-explore-mixed-currency"].includes(capture.fixture)
    || (capture.fixture === "report-overview-partial" && isRisk);
  const scopeID = capture.reportOrganizationScope ?? null;
  const expectedTotal = isRisk ? empty ? "0" : scopeID ? "2" : "4" : label === "Loss events" ? empty ? "0" : "1"
    : empty ? "—" : mixed ? capture.reportExploreCurrency === "USD" ? "-$4.50" : "₦9,007,199,254,740,993.01" : layout.expectedLoss;
  const expectedRows = empty ? [] : isRisk ? [{ label: "Risk Operations", value: "2" }, ...(!scopeID ? [{ label: "Legal entity", value: "2" }] : [])]
    : [{ label: "Risk Operations", value: expectedTotal }];
  if (layout.measure !== label || layout.chartLabel !== `${label} chart` || !layout.exactValuesContained
      || (live ? layout.chartState !== "live" || layout.total !== expectedTotal || !layout.table
        || JSON.stringify(layout.chartRows) !== JSON.stringify(expectedRows) || JSON.stringify(layout.tableRows) !== JSON.stringify(expectedRows)
        : !["unavailable", "denied"].includes(layout.chartState) || layout.total !== null || layout.table || layout.chartRows.length || layout.tableRows.length)) {
    throw new Error(`${capture.name}: selected measure, chart, exact values and unavailable state disagree with the evidence source: ${JSON.stringify(layout)}`);
  }
  if (live) {
    if (!empty) {
      await root.getByRole("list", { name: `${label} by organization area`, exact: true }).waitFor({ state: "visible" });
      await root.getByRole("table", { name: "Explore measure breakdown", exact: true }).waitFor({ state: "visible" });
    }
    if (isRisk && (layout.dataQuality !== (capture.fixture === "report-overview-partial" ? "partial" : "complete")
        || !layout.coverageText?.includes(`${scopeID ? 5 : 12} assessed`)
        || !layout.coverageText.includes(`${capture.fixture === "report-overview-partial" ? 2 : 0} unknown`)
        || !layout.coverageText.includes("0 excluded")
        || (capture.fixture === "report-overview-partial" && !layout.coverageText.includes("Risk coverage is partial."))
        || !layout.sourceText?.includes("enterprise-domain-v1"))) {
      throw new Error(`${capture.name}: risk coverage and retained source metadata are incomplete`);
    }
    if (!isRisk && !layout.sourceText?.includes("operational-loss-ledger-v1")) throw new Error(`${capture.name}: loss source revision is missing`);
  }
  const expectedEmptyHeading = isRisk ? "No risks outside appetite" : label === "Loss events" ? "No new loss events" : "No monetary loss flow";
  if (empty) {
    await root.getByRole("heading", { name: expectedEmptyHeading, exact: true }).waitFor({ state: "visible" });
    if (await root.getByRole("button", { name: "Review underlying records", exact: true }).count()) {
      throw new Error(`${capture.name}: an empty measure must not offer a nonempty record drill`);
    }
  }
  if (mixed && (layout.currency !== capture.reportExploreCurrency
      || !layout.coverageText?.includes(`Showing ${capture.reportExploreCurrency} only.`)
      || !layout.coverageText.includes("Amounts are not converted or combined."))) {
    throw new Error(`${capture.name}: mixed currency totals must remain exact and separate`);
  }
  if (isRisk ? !layout.scopeText?.includes("the reporting period does not change it")
    : live && !layout.scopeText?.includes("Loss activity from 2026-09-08 to 2026-10-07")) {
    throw new Error(`${capture.name}: current risk posture and period loss must state their different time bases`);
  }
  let sourceID = null;
  let definitionRevision = null;
  if (live) {
    const reads = layout.reads.map((value) => new URL(value, "https://evidence.invalid"));
    const latest = (pathname) => reads.filter((url) => url.pathname === pathname).at(-1);
    const data = latest(isRisk ? "/api/v1/metrics/domain" : "/api/v1/metrics/losses/period");
    const breakdown = isRisk ? latest("/api/v1/metrics/domain/risks_outside_appetite/organization-breakdown") : null;
    sourceID = reportExploreFixtureSource(capture.fixture, isRisk, Boolean(scopeID));
    definitionRevision = isRisk ? "enterprise-domain-v1" : "operational-loss-period-v1";
    if (!data || (data.searchParams.get("organization_scope_id") ?? null) !== scopeID
        || (isRisk && (!breakdown || (breakdown.searchParams.get("organization_scope_id") ?? null) !== scopeID
          || breakdown.searchParams.get("source_id") !== sourceID || breakdown.searchParams.get("definition_revision") !== definitionRevision))
        || (!isRisk && (data.searchParams.get("start_date") !== "2026-09-08" || data.searchParams.get("end_date") !== "2026-10-07"))) {
      throw new Error(`${capture.name}: selected scope, source revision and reporting period were not used in the measure reads`);
    }
  }
  const { targets, expectedLoss, reads, ...retained } = layout;
  return { ...retained, ...tabs, placement: beside ? "beside" : "above", filterWidth: layout.filters.width,
    organizationScopeID: scopeID, sourceID, definitionRevision, sourceConsistent: live,
    timeBasis: isRisk ? "CURRENT_POSTURE" : "PERIOD_FLOW", unsupportedControlsDisabled: true,
    ...(empty ? { emptyStateConfirmed: true, emptyHeading: expectedEmptyHeading } : {}),
    ...(mixed ? { currencyAmountsSeparate: true } : {}) };
}

export async function assertReportBuilderLayout(page, capture) {
  const root = page.locator(".report-builder");
  await root.getByRole("heading", { name: "Build report", exact: true }).waitFor({ state: "visible" });
  await reportSelect(page, root, "Saved setup").locator("button:not(:disabled)").waitFor({ state: "visible" });
  const tabs = await assertLocalTabs(page, "Build report", capture.name);
  await root.getByRole("button", { name: "Save setup", exact: true }).waitFor({ state: "visible" });
  await root.getByText("Contents and filters only. Generate an approved setup to view report data.", { exact: true }).waitFor({ state: "visible" });
  await resetViewportScroll(page);
  const layout = await measureReportsView(page, "builder");
  assertViewBounds(layout, capture);
  const { setup, paper, output } = layout;
  const placement = capture.viewport.width > 1380 ? "three-columns" : capture.viewport.width > 900 ? "output-below" : capture.viewport.width > 700 ? "controls-above" : "stacked";
  if (!setup || !paper || !output || layout.heading.bottom > Math.min(setup.y, output.y) + 1
      || (placement === "three-columns" && (Math.abs(setup.width - 210) > 1 || Math.abs(output.width - 210) > 1
        || paper.x < setup.right || output.x < paper.right || Math.abs(output.y - setup.y) > 2 || Math.abs(paper.y - setup.y) > 2))
      || (placement === "output-below" && (Math.abs(setup.width - 210) > 1 || paper.x < setup.right || output.y < Math.max(setup.bottom, paper.bottom)))
      || (placement === "controls-above" && (setup.x < output.right || Math.abs(setup.y - output.y) > 2 || paper.y < Math.max(setup.bottom, output.bottom)))
      || (placement === "stacked" && (paper.y < setup.bottom || output.y < paper.bottom))) {
    throw new Error(`${capture.name}: Build report must use source ${placement} composition: ${JSON.stringify({ setup, paper, output })}`);
  }
  if (layout.sectionCount !== 2 || layout.contents.join("|") !== "Summary|Data|Report information"
      || layout.renderedDataRows !== 0 || layout.previewTitle !== "New report"
      || !layout.outputText?.includes("Manual download") || !layout.outputText.includes("Recurring delivery is unavailable.")
      || await root.getByRole("button", { name: "Generate preview", exact: true }).count()
      || await root.getByRole("checkbox").count()) {
    throw new Error(`${capture.name}: unsaved setup preview must describe supported output without fabricated report rows or prototype section controls`);
  }
  const { targets, reads, outputText, ...retained } = layout;
  return { ...retained, ...tabs, placement, setupWidth: setup.width,
    controlsOrder: placement === "controls-above" ? "output-setup" : "setup-output",
    previewKind: "SETUP_CONTENTS_ONLY", unsaved: true, manualDeliveryOnly: true };
}

async function tabArrow(page, from, key, expected, name) {
  const tablist = page.getByRole("tablist", { name: "Reports workspace" });
  await tablist.getByRole("tab", { name: from, exact: true }).focus();
  await page.keyboard.press(key);
  const selected = tablist.getByRole("tab", { name: expected, exact: true, selected: true });
  await selected.waitFor({ state: "visible" });
  if (!await selected.evaluate((node) => node === document.activeElement)) throw new Error(`${name}: arrow navigation did not move focus to ${expected}`);
}

export async function verifyReportExploreNavigation(page, capture) {
  const root = page.locator(".reports-explore");
  const measure = reportSelect(page, root, "Measure");
  await measure.locator("button").focus();
  await page.keyboard.press("Enter");
  await page.keyboard.press("ArrowDown");
  await page.keyboard.press("Enter");
  await measure.locator(".cs-select-field__value").filter({ hasText: /^Net operational loss$/ }).waitFor({ state: "visible" });
  const sequence = ["Explore", "Reports", "Schedules", "Overview", "Explore"];
  for (let index = 0; index < sequence.length - 1; index++) await tabArrow(page, sequence[index], "ArrowRight", sequence[index + 1], capture.name);
  await measure.locator(".cs-select-field__value").filter({ hasText: /^Net operational loss$/ }).waitFor({ state: "visible" });
  await page.locator(".reports-workspace__actions").getByRole("button", { name: "New report", exact: true }).click();
  await assertLocalTabs(page, "Build report", capture.name);
  await tabArrow(page, "Build report", "ArrowLeft", "Schedules", capture.name);
  await assertLocalTabs(page, "Schedules", capture.name);
  await page.getByRole("tab", { name: "Explore", exact: true }).click();
  await measure.locator(".cs-select-field__value").filter({ hasText: /^Net operational loss$/ }).waitFor({ state: "visible" });
  await chooseReportExploreMeasure(page, "Risks outside appetite");
  await root.locator('.reports-explore__chart[data-state="live"]').waitFor({ state: "visible" });
  await assertLocalTabs(page, "Explore", capture.name);
  await resetViewportScroll(page);
  return { stableTabs, contextualTabs: [...stableTabs, "Build report"], arrowSequence: sequence,
    measureKeyboardSelection: true, selectedMeasurePreserved: true, hiddenBuilderRemoved: true, baselineRestored: true };
}

export async function verifyReportBuilderNavigation(page, capture) {
  const root = page.locator(".report-builder");
  const draftName = "Monthly Program follow-up";
  await root.getByLabel(/^Name\b/).fill(draftName);
  await chooseReportField(page, root, "Area", "Programs");
  await chooseReportField(page, root, "Report type", "Exceptions & outstanding");
  await root.locator(".report-builder__paper-heading").getByRole("heading", { name: draftName, exact: true }).waitFor({ state: "visible" });
  const sequence = ["Build report", "Schedules", "Reports", "Explore", "Overview"];
  for (let index = 0; index < sequence.length - 1; index++) await tabArrow(page, sequence[index], "ArrowLeft", sequence[index + 1], capture.name);
  await assertLocalTabs(page, "Overview", capture.name);
  await page.locator(".reports-workspace__actions").getByRole("button", { name: "New report", exact: true }).click();
  await assertLocalTabs(page, "Build report", capture.name);
  if (await root.getByLabel(/^Name\b/).inputValue() !== draftName
      || await reportSelect(page, root, "Area").locator(".cs-select-field__value").textContent() !== "Programs"
      || await reportSelect(page, root, "Report type").locator(".cs-select-field__value").textContent() !== "Exceptions & outstanding"
      || await root.count() !== 1) {
    throw new Error(`${capture.name}: hidden contextual builder lost or duplicated the unsaved report setup`);
  }
  await chooseReportField(page, root, "Saved setup", "Program health review");
  await root.getByRole("button", { name: "Activate setup", exact: true }).waitFor({ state: "visible" });
  await chooseReportField(page, root, "Saved setup", "New report setup");
  if (await root.getByLabel(/^Name\b/).inputValue() !== draftName) {
    throw new Error(`${capture.name}: inspecting a stored setup discarded the unsaved setup`);
  }
  await root.getByRole("button", { name: "New setup", exact: true }).click();
  if (await root.getByLabel(/^Name\b/).inputValue() !== "") throw new Error(`${capture.name}: New setup did not reset authoring fields`);
  await resetViewportScroll(page);
  return { stableTabs, contextualTabs: [...stableTabs, "Build report"], arrowSequence: sequence,
    draftNamePreserved: true, areaPreserved: true, reportTypePreserved: true,
    savedSetupInspectionPreservedDraft: true, hiddenBuilderRemoved: true, singleRetainedBuilder: true, baselineRestored: true };
}

export async function verifyReportExploreCurrency(page, capture) {
  const root = page.locator(".reports-explore");
  const verified = [];
  for (const [currency, total] of [["USD", "-$4.50"], ["NGN", "₦9,007,199,254,740,993.01"]]) {
    await chooseReportField(page, root, "Currency", currency);
    await root.locator(".reports-explore__total strong").filter({ hasText: new RegExp(`^${escape(total)}$`) }).waitFor({ state: "visible" });
    await page.getByRole("tab", { name: "Reports", exact: true }).click();
    await page.getByRole("tab", { name: "Explore", exact: true }).click();
    const actual = await assertReportsExploreLayout(page, { ...capture, reportExploreCurrency: currency });
    if (actual.currency !== currency || actual.total !== total) throw new Error(`${capture.name}: currency selection did not survive the report view change`);
    verified.push({ currency, total });
  }
  return { verified, selectionPreserved: true };
}

async function verifySheetFocusLoop(page, dialog, name) {
  const close = dialog.getByRole("button", { name: "Close", exact: true });
  await close.focus();
  for (const key of ["Shift+Tab", "Tab", "Tab", "Shift+Tab"]) {
    await page.keyboard.press(key);
    if (!await dialog.evaluate((node) => node.contains(document.activeElement))) throw new Error(`${name}: ${key} let focus leave the sheet`);
  }
}

export async function verifyReportExploreMembers(page, capture, layout) {
  const root = page.locator(".reports-explore");
  const opener = root.getByRole("button", { name: "Review underlying records", exact: true });
  const dialog = page.getByRole("dialog", { name: "Risks outside appetite records", exact: true });
  await opener.click();
  const table = dialog.getByRole("table", { name: "Exact risks outside appetite snapshot members", exact: true });
  await table.waitFor({ state: "visible" });
  await dialog.getByText("4 exact records in this metric snapshot.", { exact: true }).waitFor({ state: "visible" });
  await dialog.getByText("Sample settlement processing risk", { exact: true }).waitFor({ state: "visible" });
  const rowCount = await table.locator("tbody tr").count();
  const reads = await page.evaluate(() => window.reportingEvidenceReads ?? []);
  const request = reads.map((value) => new URL(value, "https://evidence.invalid"))
    .filter((url) => url.pathname === "/api/v1/metrics/domain/risks_outside_appetite/members").at(-1);
  if (rowCount !== 4 || !request || request.searchParams.get("source_id") !== layout.sourceID
      || request.searchParams.get("definition_revision") !== layout.definitionRevision
      || request.searchParams.get("organization_scope_id") !== layout.organizationScopeID || request.searchParams.get("limit") !== "50") {
    throw new Error(`${capture.name}: underlying record count or request is not bound to the selected metric source`);
  }
  await verifySheetFocusLoop(page, dialog, capture.name);
  await page.keyboard.press("Escape");
  await dialog.waitFor({ state: "hidden" });
  if (!await opener.evaluate((node) => node === document.activeElement)) throw new Error(`${capture.name}: closing underlying records did not restore the invoking action`);
  await opener.click();
  await table.waitFor({ state: "visible" });
  return { metricID: "risks_outside_appetite", sourceID: layout.sourceID, definitionRevision: layout.definitionRevision,
    organizationScopeID: layout.organizationScopeID, count: 4, rowCount, limit: 50,
    focusContained: true, escapeRestoredFocus: true, captureOpen: true };
}

export async function verifyReportBuilderFlow(page, capture, initialLayout) {
  const root = page.locator(".report-builder");
  const name = "Sample Program exceptions";
  const blocked = capture.fixture === "report-builder-blocked";
  await root.getByLabel(/^Name\b/).fill(name);
  await chooseReportField(page, root, "Area", "Programs");
  await chooseReportField(page, root, "Report type", "Exceptions & outstanding");
  await root.getByRole("button", { name: "Save setup", exact: true }).click();
  await root.getByRole("button", { name: "Send for review", exact: true }).waitFor({ state: "visible" });
  await root.getByRole("button", { name: "Send for review", exact: true }).click();
  await root.getByRole("button", { name: "Record review", exact: true }).waitFor({ state: "visible" });
  await root.getByRole("button", { name: "Record review", exact: true }).click();
  if (blocked) {
    await root.getByText("Independent reviewer required.", { exact: true }).waitFor({ state: "visible" });
    await root.getByRole("button", { name: "Record review", exact: true }).waitFor({ state: "visible" });
    if (await root.getByRole("button", { name: "Activate setup", exact: true }).count()
        || await root.getByRole("button", { name: "Generate report", exact: true }).count()) {
      throw new Error(`${capture.name}: rejected review still exposed activation or generation`);
    }
  } else {
    await root.getByRole("button", { name: "Activate setup", exact: true }).waitFor({ state: "visible" });
    await root.getByRole("button", { name: "Activate setup", exact: true }).click();
    await root.getByRole("button", { name: "Generate report", exact: true }).waitFor({ state: "visible" });
  }
  // Refresh through the real read owner, without reloading the fixture, to
  // prove that a transition is retained by its API result and list response.
  const readCount = await page.evaluate(() => window.reportingEvidenceReads?.length ?? 0);
  await root.getByRole("button", { name: "Refresh setups", exact: true }).click();
  await page.waitForFunction((count) => (window.reportingEvidenceReads?.length ?? 0) > count, readCount);
  await root.getByRole("button", { name: blocked ? "Record review" : "Generate report", exact: true }).waitFor({ state: "visible" });
  await reportSelect(page, root, "Saved setup").locator("button:not(:disabled)").waitFor({ state: "visible" });
  const savedDefinitionRetained = await reportSelect(page, root, "Saved setup").locator(".cs-select-field__value").textContent() === name
    && await root.locator(".report-builder__paper-heading").getByRole("heading", { name, exact: true }).count() === 1;
  if (!savedDefinitionRetained) throw new Error(`${capture.name}: saved report setup changed or vanished after its list refresh`);
  let generationHandoff = false;
  let generationFocusContained = false;
  let queuedRecordVisible = false;
  let dateFiltersAreRecordCreation = false;
  if (!blocked) {
    await root.getByRole("button", { name: "Generate report", exact: true }).click();
    const dialog = page.getByRole("dialog", { name: "Generate report", exact: true });
    await dialog.waitFor({ state: "visible" });
    generationHandoff = await reportSelect(page, dialog, "Saved setup").locator(".cs-select-field__value").textContent() === name
      && await reportSelect(page, dialog, "Area").locator(".cs-select-field__value").textContent() === "Programs";
    if (!generationHandoff) throw new Error(`${capture.name}: newly activated setup was not handed to the real generation form`);
    await dialog.getByText("Choose an approved setup. Date filters apply to when records were created.", { exact: true }).waitFor({ state: "visible" });
    dateFiltersAreRecordCreation = true;
    await verifySheetFocusLoop(page, dialog, capture.name);
    generationFocusContained = true;
    await dialog.getByLabel("Start date", { exact: true }).fill("2026-09-01");
    await dialog.getByLabel("End date", { exact: true }).fill("2026-09-30");
    await reportSelect(page, dialog, "Assigned owner").locator("button:not(:disabled)").waitFor({ state: "visible" });
    await chooseReportField(page, dialog, "Assigned owner", "Amina Yusuf");
    await dialog.getByRole("button", { name: "Generate report", exact: true }).click();
    await dialog.waitFor({ state: "hidden" });
    await page.getByRole("tab", { name: "Reports", exact: true, selected: true }).waitFor({ state: "visible" });
    const record = page.getByRole("list", { name: "Generated report records" })
      .getByRole("button", { name: /^Select Sample Program exceptions, Queued,/ });
    await record.waitFor({ state: "visible" });
    await record.click();
    const inspector = page.getByRole("complementary", { name: "Selected report" });
    await inspector.getByRole("heading", { name, exact: true }).waitFor({ state: "visible" });
    if (!await inspector.getByRole("button", { name: "Download report", exact: true }).isDisabled()) throw new Error(`${capture.name}: queued run exposes a nonexistent generated file`);
    queuedRecordVisible = true;
  }
  const commands = await page.evaluate(() => window.reportingEvidenceCommands ?? []);
  const expectedActions = blocked ? ["create", "submit", "review"] : ["create", "submit", "review", "activate", "generate"];
  const actions = commands.map((item) => item.path === "/api/v1/reports/definitions" ? "create" : item.path === "/api/v1/reports/runs" ? "generate" : item.path.split("/").at(-1));
  const created = commands[0]?.result;
  const expectedFilters = ["AT_RISK", "GAP_IDENTIFIED", "EVIDENCE_INSUFFICIENT", "IMPLEMENTATION_PENDING", "OVERDUE", "UNDER_REVIEW", "UNKNOWN"];
  const createdBody = commands[0]?.body;
  if (actions.join("|") !== expectedActions.join("|") || commands.some((item) => item.method !== "POST")
      || JSON.stringify(commands.map((item) => item.status)) !== JSON.stringify(blocked ? [201, 200, 403] : [201, 200, 200, 200, 201])
      || createdBody?.name !== name || createdBody.dataset !== "PROGRAMS" || createdBody.scope_kind !== "LEGAL_ENTITY" || createdBody.format !== "XLSX"
      || createdBody.filter?.kind !== "group" || createdBody.filter?.operator !== "or"
      || JSON.stringify(createdBody.filter?.children?.map((item) => [item.kind, item.field, item.operator, item.value]))
        !== JSON.stringify(expectedFilters.map((value) => ["condition", "overall_state", "is", value]))
      || created?.id !== "sample-report-builder-1" || created.status !== "DRAFT" || created.current_version !== 1 || created.version !== 1
      || typeof created.checksum !== "string" || !/^[0-9a-f]{64}$/.test(created.checksum)) {
    throw new Error(`${capture.name}: report setup commands differ from the supported persisted definition: ${JSON.stringify(commands)}`);
  }
  const transitions = commands.filter((item) => /\/(submit|review|activate)$/.test(item.path));
  const expectedStatuses = blocked ? ["PENDING_REVIEW"] : ["PENDING_REVIEW", "REVIEWED", "ACTIVE"];
  const checksumRetained = transitions.every((item) => item.body.checksum_seen === created.checksum
    && (item.status !== 200 || item.result.checksum === created.checksum));
  const versionChecksMatched = transitions.every((item, index) => item.body.expected_version === index + 1
    && item.path.startsWith(`/api/v1/reports/definitions/${created.id}/`)
    && (item.status !== 200 || item.result.version === index + 2 && item.result.current_version === 1 && item.result.status === expectedStatuses[index]));
  const runCommand = commands.find((item) => item.path === "/api/v1/reports/runs");
  if (!checksumRetained || !versionChecksMatched || (blocked ? commands.at(-1).result?.error?.code !== "report_governance_blocked"
    : runCommand?.body?.definition_id !== created.id || runCommand?.body?.expected_definition_version !== 1
      || JSON.stringify(runCommand?.body?.parameters) !== JSON.stringify({ start_date: "2026-09-01", end_date: "2026-09-30", owner_principal_id: "owner-program" })
      || runCommand?.result?.id !== "sample-report-builder-run-1" || runCommand?.result?.status !== "QUEUED"
      || runCommand?.result?.definition_checksum !== created.checksum || runCommand?.result?.definition_version !== 1)) {
    throw new Error(`${capture.name}: version, checksum or generation parameters did not follow the retained definition: ${JSON.stringify(commands)}`);
  }
  await resetViewportScroll(page);
  return { kind: "builder-flow", initialLayout, flow: {
    evidenceMode: "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION", definitionID: created.id, definitionName: name,
    dataset: created.dataset, format: created.format, scopeKind: created.scope_kind,
    actions, commandStatuses: commands.map((item) => item.status), commands,
    saveCount: commands.filter((item) => item.path === "/api/v1/reports/definitions").length,
    checksumRetained, versionChecksMatched, savedDefinitionRetained, finalStatus: blocked ? "PENDING_REVIEW" : "ACTIVE",
    blockedReviewPreservedSetup: blocked, generationHandoff, generationFocusContained, dateFiltersAreRecordCreation,
    runID: runCommand?.result?.id ?? null, runStatus: runCommand?.result?.status ?? null, queuedRecordVisible,
  } };
}
