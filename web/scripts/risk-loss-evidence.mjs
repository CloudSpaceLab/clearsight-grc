import { riskLossViews } from "./risk-loss-evidence-scenarios.mjs";
import { installRiskLossObservation } from "./risk-loss-evidence-fixtures.mjs";
import { riskLossStatusBadgesFit, riskLossRCSAPopulationMatches } from "./risk-loss-evidence-contract.mjs";

export const riskLossSourceSHA256 = "cf0a101d574904fc5f496cab0625889e640e2e30734d180d86b89171419e69dd";
export const riskLossSourcePath = "clearsight-operational-risk-designs/operational-risk-preview.html";
const roots = { risks: ".risk-register-page", rcsa: ".rcsa-workspace", insights: ".insights-indicator-register, .insights-workspace", losses: ".loss-register-page" };
const backNames = { risks: "Back to risks", losses: "Back to losses" };
const actionNames = { risks: "Open risk", rcsa: "Review cycle", insights: "Review indicator", losses: "Open loss" };

export async function prepareRiskLoss(page, scenario, baseline = false) {
  await installRiskLossObservation(page, scenario);
  await page.evaluate((route) => { location.hash = route; }, scenario.route);
  await waitForRiskLoss(page, scenario, baseline);
  await page.evaluate(() => document.fonts?.ready);
  await page.evaluate(() => window.scrollTo(0, 0));
}

export async function waitForRiskLoss(page, scenario, baseline = false) {
  if (baseline) await page.getByRole("heading", { name: scenario.baselineHeading, exact: true }).first().waitFor();
  else await page.locator(".risk-loss-workspace__heading").getByRole("heading", { name: "Risk & Losses", exact: true }).waitFor();
  if (scenario.riskLossCheck === "empty") {
    await page.locator(roots[scenario.riskLossView]).getByRole("heading", { name: /No (risks|cycles|indicators|losses|matching)/ }).first().waitFor();
  } else if (scenario.riskLossCheck === "denied") {
    await page.locator(roots[scenario.riskLossView]).getByText(/(unavailable|could not be loaded)/).first().waitFor();
  } else {
    await page.getByText(scenario.firstRecord, { exact: true }).first().waitFor();
  }
  await page.waitForFunction(() => window.riskLossEvidence?.pending === 0);
}

export async function measureRiskLossLayout(page, scenario) {
  return page.evaluate(({ view, firstRecord, labels }) => {
    const visible = (node) => Boolean(node?.getClientRects().length && getComputedStyle(node).visibility !== "hidden" && !node.closest('[hidden], [inert], [aria-hidden="true"]'));
    const box = (node) => { if (!visible(node)) return null; const r = node.getBoundingClientRect(); return { x: r.x, y: r.y, width: r.width, height: r.height, right: r.right, bottom: r.bottom }; };
    const frame = document.querySelector(".risk-loss-workspace");
    const heading = frame?.querySelector(".risk-loss-workspace__heading");
    const tablist = frame?.querySelector('[role="tablist"]');
    const tabs = [...(tablist?.querySelectorAll('[role="tab"]') ?? [])];
    const selected = tabs.find((node) => node.getAttribute("aria-selected") === "true");
    const collection = document.querySelector({ risks: ".risk-register-page", rcsa: ".rcsa-workspace", insights: ".insights-indicator-register", losses: ".loss-register-page" }[view])
      ?? document.querySelector(".insights-workspace");
    const table = collection?.querySelector(".cs-data-table table");
    const grid = collection?.querySelector(".rcsa-cycle-grid");
    const records = [...(grid?.children ?? table?.querySelectorAll("tbody tr") ?? [])].filter(visible);
    const identity = [...(collection?.querySelectorAll("strong, h3") ?? [])].find((node) => visible(node) && node.textContent.trim() === firstRecord);
    const filters = collection?.querySelector(".risk-register__toolbar, .loss-register__toolbar, .rcsa-register__heading, .insights-indicator__heading, .cs-filter-bar");
    const local = box(tablist), current = box(selected), first = box(records[0]), identityBox = box(identity);
    const columns = grid ? getComputedStyle(grid).gridTemplateColumns.split(" ").filter(Boolean).length : null;
    const read = window.riskLossEvidence;
    const rowActions = records.flatMap((record) => [...record.querySelectorAll("button[aria-label]")]).filter(visible).map((node) => {
      const range = document.createRange(); range.selectNodeContents(node);
      return { label: node.getAttribute("aria-label"), bounds: box(node), lines: new Set([...range.getClientRects()].filter((r) => r.width > 1 && r.height > 1).map((r) => Math.round(r.y))).size };
    });
    const statusBadges = ["risks", "insights"].includes(view) ? records.flatMap((record, rowIndex) => [...record.querySelectorAll('td[data-kind="status"] .cs-status-badge')].filter(visible).map((node) => {
      // Measure text nodes only: the decorative marker has its own vertical alignment.
      const walker = document.createTreeWalker(node, NodeFilter.SHOW_TEXT), textRects = [], words = [];
      while (walker.nextNode()) if (walker.currentNode.textContent.trim()) {
        const range = document.createRange(); range.selectNodeContents(walker.currentNode);
        textRects.push(...[...range.getClientRects()].filter((r) => r.width > 1 && r.height > 1));
        for (const match of walker.currentNode.textContent.matchAll(/\S+/g)) {
          range.setStart(walker.currentNode, match.index); range.setEnd(walker.currentNode, match.index + match[0].length);
          words.push({ text: match[0], lines: new Set([...range.getClientRects()].filter((r) => r.width > 1 && r.height > 1).map((r) => Math.round(r.y))).size });
        }
      }
      const x = Math.min(...textRects.map((r) => r.x)), y = Math.min(...textRects.map((r) => r.y));
      const right = Math.max(...textRects.map((r) => r.right)), bottom = Math.max(...textRects.map((r) => r.bottom));
      return { label: node.textContent.trim(), rowIndex, column: node.closest("td").getAttribute("data-label"), words, bounds: box(node), cell: box(node.closest("td")),
        textBounds: textRects.length ? { x, y, width: right - x, height: bottom - y, right, bottom } : null,
        lines: new Set(textRects.map((r) => Math.round(r.y))).size };
    })) : [];
    return { kind: view, route: location.hash, evidenceKind: "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION", heading: box(heading), headingText: heading?.querySelector("h1")?.textContent ?? null, main: box(frame ?? collection), collection: box(collection), controls: box(filters),
      localTabs: tabs.map((node) => node.textContent.trim()), selectedTab: selected?.textContent.trim() ?? null, selectedTabID: selected?.getAttribute("data-key") ?? null,
      menu: local, menuRows: new Set(tabs.map((node) => Math.round(node.getBoundingClientRect().top))).size,
      selectedTabVisible: Boolean(current && local && current.x >= local.x - 1 && current.right <= local.right + 1 && current.y >= local.y - 1 && current.bottom <= local.bottom + 1),
      menuMatches: tabs.map((node) => node.textContent.trim()).join("|") === labels.join("|"), table: box(table), grid: box(grid), columns,
      presentation: grid ? "cards" : table ? getComputedStyle(table).display === "block" ? "stacked-records" : "table" : "state",
      records: records.length, firstRecord: first, firstIdentity: identityBox,
      firstRecordColumns: table && records[0] ? getComputedStyle(records[0]).gridTemplateColumns.split(" ").filter((value) => value !== "none").length : null, rowActions, statusBadges,
      recordTitles: records.map((record) => record.querySelector(grid ? "h3" : "td:first-child strong")?.textContent.trim() ?? ""),
      recordAccessibleNames: records.map((record) => record.getAttribute("aria-label") ?? record.querySelector("h3")?.textContent.trim() ?? ""),
      rowValues: records.map((record) => grid ? { title: record.querySelector("h3")?.textContent.trim(),
        facts: Object.fromEntries([...record.querySelectorAll("dl > div")].map((fact) => [fact.querySelector("dt")?.textContent, fact.querySelector("dd")?.textContent])) }
        : Object.fromEntries([...record.querySelectorAll("td[data-label]")].map((cell) => [cell.getAttribute("data-label"), cell.textContent.trim()]))),
      cardNumbers: grid && records[0] ? [...records[0].querySelectorAll(".rcsa-cycle__population > div")].map((item) => {
        const term = item.querySelector("dt"), value = item.querySelector("dd");
        return { label: term?.textContent, value: value?.textContent, labelFontSize: parseFloat(getComputedStyle(term).fontSize), fontSize: parseFloat(getComputedStyle(value).fontSize),
          aligned: Math.abs(term.getBoundingClientRect().x - value.getBoundingClientRect().x) <= 1, textAlign: getComputedStyle(value).textAlign };
      }) : [],
      stateText: collection?.textContent ?? "", disclosures: [...(collection?.querySelectorAll("details") ?? [])].map((node) => ({ label: node.querySelector("summary")?.textContent.trim(), open: node.open })),
      firstUsefulContentVisible: Boolean(identityBox && identityBox.y >= 0 && identityBox.bottom <= innerHeight),
      tableHeaders: [...(table?.querySelectorAll("thead th") ?? [])].map((node) => node.textContent.trim()),
      primaryCurrent: [...document.querySelectorAll('.sidebar a[aria-current="page"]')].map((node) => (node.getAttribute("aria-label") ?? node.textContent).trim()),
      dialogVisible: [...document.querySelectorAll('[role="dialog"]')].some(visible),
      documentWidth: document.documentElement.clientWidth, scrollWidth: document.documentElement.scrollWidth,
      populatedIDs: read?.responsePopulation ?? [], pendingReads: read?.pending, commands: read?.commands ?? [],
    };
  }, { view: scenario.riskLossView, firstRecord: scenario.firstRecord, labels: riskLossViews.map((item) => item.label) });
}

export function assertRiskLossLayout(layout, scenario) {
  const fail = (message) => { throw new Error(`${scenario.name}: ${message}`); };
  if (layout.headingText !== "Risk & Losses" || !layout.menuMatches || layout.selectedTab !== scenario.label || layout.menuRows !== 1 || !layout.selectedTabVisible) fail("page heading or four-route source-style menu is missing, wrapped or clipped");
  if (layout.scrollWidth > layout.documentWidth + 1) fail("horizontal document overflow");
  if (layout.pendingReads !== 0 || layout.commands.length) fail("capture is unsettled or performed a business command");
  if (layout.primaryCurrent.join("|") !== "Risk & Losses") fail("shared primary navigation is not current for the risk family");
  if (["empty", "denied"].includes(scenario.riskLossCheck)) {
    if (layout.records || layout.populatedIDs.length) fail("empty or denied view exposed records");
    return;
  }
  if (!layout.firstUsefulContentVisible || !layout.firstRecord || !layout.records) fail("the first useful record is absent from the first viewport");
  if (layout.rowActions.some((action) => action.lines > 2 || action.bounds.width < 44)) fail("record action is clipped or wraps beyond two lines");
  if (["risks", "insights"].includes(scenario.riskLossView) && !riskLossStatusBadgesFit(layout)) fail("status badges must keep every word intact with text contained in the badge and cell");
  if (!(layout.heading.bottom <= layout.menu.y + 1 && layout.menu.bottom <= layout.firstRecord.y + 1)) fail("heading, local menu and useful content anatomy is out of order");
  if (scenario.riskLossView === "rcsa") {
    const expectedColumns = scenario.viewport.width > 1190 ? 3 : scenario.viewport.width > 600 ? 2 : 1;
    if (layout.presentation !== "cards" || layout.columns !== expectedColumns) fail(`RCSA must replace its table with ${expectedColumns} source-aligned card columns`);
    if (layout.cardNumbers.length !== 2 || layout.cardNumbers.some((number) => number.fontSize < 30 || number.labelFontSize > 14 || !number.aligned || !["start", "left"].includes(number.textAlign))) fail("RCSA frozen population counts lost their source numeric hierarchy");
  } else {
    if (!layout.table || layout.presentation === "cards") fail("source-aligned register table is missing");
    if (scenario.viewport.width === 390 && layout.presentation !== "stacked-records") fail("phone records did not replace the desktop table");
    if (scenario.viewport.width === 390 && layout.firstRecordColumns !== 2) fail("phone records must use the source two-column fact layout with full-width identity/action");
    if (scenario.riskLossView === "losses" && !["Gross", "Recovered", "Net"].every((name) => layout.tableHeaders.includes(name))) fail("loss gross/recovery/net columns are not distinct");
  }
}

export async function openRiskLossDetail(page, scenario) {
  const root = page.locator(roots[scenario.riskLossView]).first();
  const open = root.getByRole("button", { name: new RegExp(`^${actionNames[scenario.riskLossView]} for ${scenario.firstRecord}`) }).first();
  const cardPopulation = scenario.riskLossView === "rcsa" ? await open.evaluate((node) => Object.fromEntries(
    [...node.closest("li").querySelectorAll(".rcsa-cycle__population > div")].map((fact) => [fact.querySelector("dt")?.textContent.trim(), fact.querySelector("dd")?.textContent.trim()]),
  )) : null;
  await open.click();
  const target = scenario.riskLossView === "rcsa" || scenario.riskLossView === "insights" ? page.getByRole("dialog") : page.locator(scenario.riskLossView === "risks" ? ".risk-record-page" : ".loss-record-page");
  await target.getByRole("heading", { name: scenario.firstRecord, exact: true }).first().waitFor();
  if (scenario.riskLossView === "rcsa") await target.getByRole("table", { name: "RCSA frozen Controls", exact: true }).waitFor();
  if (scenario.riskLossView === "insights") await target.getByRole("table", { name: `${scenario.firstRecord} observation history`, exact: true }).waitFor();
  if (scenario.riskLossView === "losses") await target.getByRole("table", { name: "Loss recoveries", exact: true }).waitFor();
  await page.waitForFunction(() => window.riskLossEvidence?.pending === 0);
  return page.evaluate(({ view, id, title, cardPopulation }) => {
    const dialog = document.querySelector('[role="dialog"]');
    const standalone = document.querySelector(view === "risks" ? ".risk-record-page" : ".loss-record-page");
    const root = view === "rcsa" || view === "insights" ? dialog : standalone;
    const heading = [...(root?.querySelectorAll("h1,h2,h3") ?? [])].find((node) => node.textContent.trim() === title);
    const panel = dialog?.closest(".cs-sheet") ?? root;
    const rect = panel?.getBoundingClientRect(), titleBounds = heading?.getBoundingClientRect();
    const close = dialog?.querySelector('button[aria-label="Close"]');
    const actionLabel = { risks: "Back to risks", losses: "Back to losses", rcsa: "Open challenge work", insights: "Open Program" }[view];
    const action = [...(root?.querySelectorAll("button") ?? [])].find((node) => node.textContent.trim() === actionLabel);
    const bounds = (node) => { const r = node?.getBoundingClientRect(); return r ? { x: r.x, y: r.y, width: r.width, height: r.height, right: r.right, bottom: r.bottom } : null; };
    const visibleInViewport = (node) => { const r = bounds(node); return Boolean(r && r.width > 0 && r.height > 0 && r.x >= 0 && r.y >= 0 && r.right <= innerWidth + 1 && r.bottom <= innerHeight + 1); };
    return { route: location.hash, exactTarget: view === "insights" ? new URLSearchParams(location.hash.split("?")[1]).get("indicator") === id : location.hash.endsWith(`/${id}`),
      titleMatches: Boolean(heading), placement: dialog ? "sheet" : "standalone", width: rect?.width ?? 0,
      x: rect?.x ?? 0, height: rect?.height ?? 0, titleVisible: Boolean(titleBounds && titleBounds.y >= 0 && titleBounds.bottom <= innerHeight),
      close: bounds(close), closeVisible: visibleInViewport(close), focusContained: Boolean(dialog?.contains(document.activeElement)),
      primaryAction: { label: actionLabel, bounds: bounds(action), visible: visibleInViewport(action) },
      internalTables: [...(root?.querySelectorAll(".cs-data-table") ?? [])].map((node) => ({ clientWidth: node.clientWidth, scrollWidth: node.scrollWidth,
        presentation: getComputedStyle(node.querySelector("table")).display })),
      rcsaPopulation: view === "rcsa" ? { card: cardPopulation,
        detailRows: Object.fromEntries(["Risks", "Controls"].map((kind) => [kind, root.querySelectorAll(`table[aria-label="RCSA frozen ${kind}"] tbody tr`).length])),
        detailSummary: [...root.querySelectorAll(".rcsa-detail__facts > div")].find((fact) => fact.querySelector("span")?.textContent.trim() === "Frozen population")?.querySelector("strong")?.textContent.trim() ?? null } : null,
      pendingReads: window.riskLossEvidence.pending, commands: window.riskLossEvidence.commands, documentToken: window.riskLossEvidence.documentToken };
  }, { view: scenario.riskLossView, id: scenario.detailID, title: scenario.firstRecord, cardPopulation });
}

export function assertRiskLossDetail(detail, scenario) {
  const sheet = ["rcsa", "insights"].includes(scenario.riskLossView);
  if (!detail.exactTarget || !detail.titleMatches || !detail.titleVisible || detail.pendingReads || detail.commands.length || !(detail.width > 0 && detail.width <= scenario.viewport.width + 1)
    || sheet && (Math.abs(detail.width - Math.min(480, scenario.viewport.width)) > 1 || Math.abs(detail.x + detail.width - scenario.viewport.width) > 1
      || Math.abs(detail.height - scenario.viewport.height) > 1 || !detail.closeVisible || !detail.focusContained || !detail.primaryAction.visible
      || detail.close.width < 44 || detail.close.height < 44 || detail.internalTables.some((table) => table.scrollWidth > table.clientWidth + 1 || table.presentation !== "block"))) {
    throw new Error(`${scenario.name}: exact detail reader must retain its title and use a bounded 480px inspector with container-aware tables`);
  }
  if (scenario.riskLossView === "rcsa" && !riskLossRCSAPopulationMatches(detail)) throw new Error(`${scenario.name}: RCSA inspector frozen population must match the selected card, rendered summary and detail rows`);
}

export async function verifyRiskLossHistory(page, scenario) {
  const before = await page.evaluate(() => ({ token: window.riskLossEvidence.documentToken, ids: window.riskLossEvidence.responsePopulation }));
  const opened = await openRiskLossDetail(page, scenario);
  if (!opened.exactTarget || !opened.titleMatches) throw new Error(`${scenario.name}: initial exact record did not open`);
  await page.goBack();
  await waitForRiskLoss(page, scenario);
  const after = await measureRiskLossLayout(page, scenario);
  assertRiskLossLayout(after, scenario);
  const back = await page.evaluate(() => ({ route: location.hash, token: window.riskLossEvidence.documentToken, pending: window.riskLossEvidence.pending }));
  if (back.route !== scenario.route || back.token !== before.token || after.dialogVisible || back.pending !== 0 || after.populatedIDs.join("|") !== before.ids.join("|")) throw new Error(`${scenario.name}: real Back did not restore the same settled populated collection`);
  await page.goForward();
  const target = scenario.riskLossView === "rcsa" || scenario.riskLossView === "insights" ? page.getByRole("dialog") : page.locator(scenario.riskLossView === "risks" ? ".risk-record-page" : ".loss-record-page");
  await target.getByRole("heading", { name: scenario.firstRecord, exact: true }).first().waitFor();
  await page.waitForFunction(() => window.riskLossEvidence?.pending === 0);
  const forward = await page.evaluate(() => ({ route: location.hash, token: window.riskLossEvidence.documentToken, pending: window.riskLossEvidence.pending }));
  if (forward.route !== opened.route || forward.token !== before.token || forward.pending !== 0) throw new Error(`${scenario.name}: Forward changed the exact record or document`);
  if (backNames[scenario.riskLossView]) await page.getByRole("button", { name: backNames[scenario.riskLossView], exact: true }).click();
  else await page.getByRole("dialog").getByRole("button", { name: "Close", exact: true }).click();
  await waitForRiskLoss(page, scenario);
  return { actualBack: true, sameDocument: true, selectedMenu: after.selectedTab, noSelectedDetail: !after.dialogVisible,
    backRoute: back.route, populationBefore: before.ids, populationAfter: after.populatedIDs, pendingReads: back.pending,
    backLayout: after, backDocumentToken: back.token, initialDocumentToken: before.token, forwardDocumentToken: forward.token,
    forwardExactTarget: true, forwardRoute: forward.route, explicitReturn: true, commands: await page.evaluate(() => window.riskLossEvidence.commands) };
}

export async function verifyRiskLossMenu(page, scenario) {
  const menu = page.getByRole("tablist", { name: "Risk and loss sections", exact: true });
  const visited = [], keyboard = [], peerHistory = [];
  for (const view of riskLossViews) {
    await menu.getByRole("tab", { name: view.label, exact: true }).click();
    await page.waitForFunction(({ id, label }) => location.hash.split("?")[0] === `#${id}`
      && document.querySelector('.risk-loss-workspace [role="tab"][aria-selected="true"]')?.textContent.trim() === label, view);
    await page.locator(".risk-loss-workspace").getByRole("heading", { name: view.heading, exact: true }).waitFor();
    visited.push({ label: view.label, route: await page.evaluate(() => location.hash) });
  }
  // Click and arrow-key focus can reveal a tab even when controlled selection
  // cannot. Exercise real history with the same mounted collection of tabs.
  const menuHandle = await menu.elementHandle();
  const documentToken = await page.evaluate(() => window.riskLossEvidence.documentToken);
  try {
    for (const [direction, indexes] of [["back", [2, 1, 0]], ["forward", [1, 2, 3]]]) {
      for (const index of indexes) {
        const view = riskLossViews[index];
        const from = await page.evaluate(() => location.hash);
        if (direction === "back") await page.goBack(); else await page.goForward();
        await page.waitForFunction(({ id, label }) => location.hash.split("?")[0] === `#${id}`
          && document.querySelector('.risk-loss-workspace [role="tab"][aria-selected="true"]')?.textContent.trim() === label, view);
        await page.locator(".risk-loss-workspace").getByRole("heading", { name: view.heading, exact: true }).waitFor();
        await page.evaluate(async () => { await document.fonts.ready; await new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))); });
        const state = await menuHandle.evaluate((node, token) => {
          const list = document.querySelector('.risk-loss-workspace [role="tablist"][aria-label="Risk and loss sections"]');
          const active = list?.querySelector('[role="tab"][aria-selected="true"]');
          const box = (element) => { if (!element) return null; const r = element.getBoundingClientRect(); return { x: r.x, y: r.y, width: r.width, height: r.height, right: r.right, bottom: r.bottom }; };
          const menu = box(list), selected = box(active);
          return { to: location.hash, label: active?.textContent.trim(), sameTablist: node.isConnected && node === list,
            sameDocument: window.riskLossEvidence.documentToken === token, menu, selected,
            selectedTabVisible: Boolean(menu && selected && selected.x >= menu.x - 1 && selected.right <= menu.right + 1 && selected.y >= menu.y - 1 && selected.bottom <= menu.bottom + 1),
            documentWidth: document.documentElement.clientWidth, scrollWidth: Math.max(document.documentElement.scrollWidth, document.body.scrollWidth) };
        }, documentToken);
        if (!state.sameTablist || !state.sameDocument || state.label !== view.label || !state.selectedTabVisible || state.scrollWidth > state.documentWidth + 1) {
          throw new Error(`${scenario.name}: peer browser ${direction} to ${view.id} did not retain a visible selected tab in the same document/menu: ${JSON.stringify(state)}`);
        }
        peerHistory.push({ direction, from, ...state });
      }
    }
  } finally { await menuHandle.dispose(); }
  await menu.getByRole("tab", { name: "Risks", exact: true }).click();
  await menu.getByRole("tab", { name: "Risks", exact: true }).focus();
  for (const label of [...riskLossViews.slice(1).map((view) => view.label), "Risks"]) {
    await page.keyboard.press("ArrowRight");
    await page.waitForFunction((label) => document.querySelector('.risk-loss-workspace [role="tab"][aria-selected="true"]')?.textContent.trim() === label, label);
    keyboard.push(label);
  }
  await waitForRiskLoss(page, scenario);
  const restored = await measureRiskLossLayout(page, scenario);
  return { visited, keyboard, peerHistory, restoredRoute: restored.route, restoredPopulation: restored.populatedIDs,
    selectedTabVisible: restored.selectedTabVisible, commands: restored.commands };
}

export async function verifyRiskLossFilters(page, scenario) {
  const view = scenario.riskLossView;
  const root = page.locator(roots[view]).first();
  const population = async (query) => {
    await page.waitForFunction(({ selector, query }) => {
      const observed = window.riskLossEvidence, response = observed?.responses.at(-1);
      const collection = document.querySelector(selector);
      const grid = collection?.querySelector(".rcsa-cycle-grid");
      const records = [...(grid?.children ?? collection?.querySelectorAll("tbody tr") ?? [])];
      const titles = records.map((record) => record.querySelector(grid ? "h3" : "td:first-child strong")?.textContent.trim() ?? "");
      return observed?.pending === 0 && response && Object.entries(query).every(([key, value]) => (response.query[key] ?? null) === value)
        && JSON.stringify(titles) === JSON.stringify(response.titles) && titles.length === response.ids.length;
    }, { selector: roots[view], query });
    const layout = await measureRiskLossLayout(page, scenario);
    const response = await page.evaluate(() => window.riskLossEvidence.responses.at(-1));
    return { ids: layout.populatedIDs, titles: layout.recordTitles, query: response.query };
  };
  const pages = { before: await population({ cursor: null }) };
  await root.getByRole("button", { name: /^(Load next page|Next page)$/ }).click();
  pages.next = await population({ cursor: "sample-risk-loss-next-page" });
  if (!pages.next.ids.length || pages.next.ids.some((id) => pages.before.ids.includes(id))) throw new Error(`${scenario.name}: cursor paging did not render distinct bounded records`);
  await root.getByRole("button", { name: /^(Load previous page|Previous page)$/ }).click();
  pages.previous = await population({ cursor: null });
  if (JSON.stringify(pages.previous.ids) !== JSON.stringify(pages.before.ids) || JSON.stringify(pages.previous.titles) !== JSON.stringify(pages.before.titles)) {
    throw new Error(`${scenario.name}: previous page did not restore the original rendered population`);
  }
  if (view === "risks" || view === "losses") {
    const search = root.getByRole("searchbox");
    await search.fill(view === "risks" ? "network" : "merchant");
    pages.filtered = await population({ cursor: null, search: view === "risks" ? "network" : "merchant" });
    await search.fill("");
    pages.restored = await population({ cursor: null, search: null });
    await root.getByRole("button", { name: "Filters", exact: true }).click();
    await root.getByRole("button", { name: view === "risks" ? /Appetite position/ : /Recovery/ }).waitFor({ state: "visible" });
    await root.getByRole("button", { name: "Filters", exact: true }).click();
  } else {
    const label = view === "rcsa" ? "Status" : "Indicator type";
    const select = root.getByRole("button", { name: new RegExp(label) });
    await select.click();
    await page.getByRole("option", { name: view === "rcsa" ? "Challenge" : "Control indicators", exact: true }).click();
    pages.filtered = await population({ cursor: null, [view === "rcsa" ? "status" : "kind"]: view === "rcsa" ? "AWAITING_CHALLENGE" : "KCI" });
    await select.click();
    await page.getByRole("option", { name: view === "rcsa" ? "All statuses" : "Risk indicators", exact: true }).click();
    pages.restored = await population({ cursor: null, [view === "rcsa" ? "status" : "kind"]: view === "rcsa" ? null : "KRI" });
  }
  await waitForRiskLoss(page, scenario);
  if (JSON.stringify(pages.restored.ids) !== JSON.stringify(pages.before.ids) || JSON.stringify(pages.restored.titles) !== JSON.stringify(pages.before.titles)) {
    throw new Error(`${scenario.name}: clearing the selected filter did not restore the original rendered population`);
  }
  return { nextPageDistinct: true, previousPageRestored: true, filterSentToReader: true, phoneFiltersAccessible: true,
    pages, reads: await page.evaluate(() => window.riskLossEvidence.reads), commands: await page.evaluate(() => window.riskLossEvidence.commands) };
}

export async function measureRiskLossSourceLayout(page, view) {
  return page.evaluate((view) => {
    const visible = (node) => Boolean(node?.getClientRects().length && getComputedStyle(node).visibility !== "hidden");
    const box = (node) => { if (!visible(node)) return null; const r = node.getBoundingClientRect(); return { x: r.x, y: r.y, width: r.width, height: r.height, right: r.right, bottom: r.bottom }; };
    const sourceView = view === "insights" || view === "risks" ? "kri" : view;
    const panel = document.querySelector(`#pane-${sourceView}`);
    const table = panel?.querySelector("table");
    const grid = panel?.querySelector(".rcsa-cycle-grid");
    const records = [...(grid?.children ?? table?.querySelectorAll("tbody tr") ?? [])].filter(visible);
    const first = box(records[0]);
    const identity = records[0]?.querySelector(grid ? ".cycle-head strong" : "td:first-child strong");
    const inspector = document.querySelector(".risk-drawer");
    return { kind: view, sourceView, comparison: view === "risks" ? "TABLE_ANATOMY_ADAPTATION" : "SAME_SCREEN", heading: box(document.querySelector(".risk-heading")),
      main: box(document.querySelector(".risk-page")), menu: box(document.querySelector(".risk-tabs")), localTabs: [...document.querySelectorAll('.risk-tabs [role="tab"]')].map((node) => node.textContent.trim()),
      selectedTab: document.querySelector('.risk-tabs [aria-selected="true"]')?.textContent.trim(), controls: box(panel?.querySelector(".alt-filters")),
      table: box(table), grid: box(grid), columns: grid ? getComputedStyle(grid).gridTemplateColumns.split(" ").length : null,
      presentation: grid ? "cards" : table && getComputedStyle(table).display === "block" ? "stacked-records" : "table", firstRecord: first,
      firstIdentity: box(identity), firstRecordColumns: table && records[0] ? getComputedStyle(records[0]).gridTemplateColumns.split(" ").filter((value) => value !== "none").length : null,
      inspectorWidth: inspector ? parseFloat(getComputedStyle(inspector).width) : null,
      firstUsefulContentVisible: Boolean(first && first.y >= 0 && first.y + 30 <= innerHeight),
      documentWidth: document.documentElement.clientWidth, scrollWidth: document.documentElement.scrollWidth };
  }, view);
}
