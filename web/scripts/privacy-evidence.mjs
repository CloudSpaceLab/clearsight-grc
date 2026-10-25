import { privacyActivityID, privacyActivityName } from "./privacy-evidence-scenarios.mjs";

export async function waitForPrivacy(page, scenario) {
  await page.getByRole("heading", { name: "Privacy & data", exact: true }).waitFor();
  const workspace = page.locator(".privacy-workspace");
  if (scenario.privacyCheck === "denied") await workspace.getByText(/unavailable with your current access/).waitFor();
  else if (scenario.privacyCheck === "empty") await workspace.getByRole("heading", { name: "No activities", exact: true }).waitFor();
  else if (scenario.privacyCheck === "recovery") await workspace.getByText("Couldn’t load activities.", { exact: true }).waitFor();
  else if (scenario.privacyView === "overview") await workspace.locator(".ropa-dashboard-strip__metrics .cs-metric-card").first().waitFor();
  else await workspace.locator(".ropa-register-identity strong").first().waitFor();
  await page.evaluate(() => document.fonts?.ready);
}

export async function measurePrivacyLayout(page, scenario) {
  return page.evaluate(({ view, firstName }) => {
    const visible = (node) => Boolean(node?.getClientRects().length && getComputedStyle(node).visibility !== "hidden" && !node.closest('[hidden], [inert], [aria-hidden="true"]'));
    const box = (node) => { if (!visible(node)) return null; const r = node.getBoundingClientRect(); return { x: r.x, y: r.y, width: r.width, height: r.height, right: r.right, bottom: r.bottom }; };
    const root = document.querySelector(".privacy-workspace"), menu = root?.querySelector('[role="tablist"]');
    const tabs = [...(menu?.querySelectorAll('[role="tab"]') ?? [])], selected = tabs.find((node) => node.getAttribute("aria-selected") === "true");
    const selectedBounds = box(selected), menuBounds = box(menu), panelID = selected?.getAttribute("aria-controls");
    const panel = panelID ? document.getElementById(panelID) : null;
    const overview = panel?.querySelector(".ropa-overview"), register = panel?.querySelector(".ropa-register-list");
    const table = register?.querySelector("table"), rows = [...(table?.querySelectorAll("tbody tr") ?? [])];
    const identity = rows[0]?.querySelector(".ropa-register-identity strong");
    const metrics = [...(overview?.querySelectorAll(".ropa-dashboard-strip__metrics .cs-metric-card") ?? [])].map((node) => ({
      label: node.querySelector(".cs-metric-card__label")?.textContent.trim(), value: node.querySelector(".cs-metric-card__measure strong")?.textContent.trim(), quality: node.getAttribute("data-quality"), bounds: box(node),
    }));
    const primary = box(overview?.querySelector(".ropa-overview__primary")), aside = box(overview?.querySelector(".ropa-overview__coverage"));
    const firstUseful = view === "overview" ? metrics[0]?.bounds : box(identity);
    const targets = [...(root?.querySelectorAll('button:not(:disabled), [role="tab"], input') ?? [])].filter(visible).map((node) => ({ label: node.getAttribute("aria-label") || node.textContent.trim(), bounds: box(node) }));
    return { evidenceKind: "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION", kind: view, route: location.hash,
      heading: box(root?.querySelector(".privacy-workspace__heading")), headingText: root?.querySelector("h1")?.textContent.trim(),
      headingCount: [...document.querySelectorAll("h1")].filter(visible).length,
      localTabs: tabs.map((node) => node.textContent.trim()), selectedTab: selected?.textContent.trim(), menu: menuBounds, selectedBounds,
      selectedVisible: Boolean(selectedBounds && menuBounds && selectedBounds.x >= menuBounds.x - 1 && selectedBounds.right <= menuBounds.right + 1),
      selectedPanelID: panelID, visiblePanelIDs: [...(root?.querySelectorAll(":scope > .cs-tabs > [role=tabpanel]") ?? [])].filter(visible).map((node) => node.id),
      primaryCurrent: [...document.querySelectorAll('.sidebar a[aria-current="page"]')].map((node) => (node.getAttribute("aria-label") ?? node.textContent).trim()),
      controls: box(register?.querySelector(".ropa-register-list__filters")), table: box(table), tablePresentation: table ? getComputedStyle(table).display : null,
      headers: [...(table?.querySelectorAll("thead th") ?? [])].map((node) => node.textContent.trim()),
      recordNames: rows.map((node) => node.querySelector(".ropa-register-identity strong")?.textContent.trim()),
      recordValues: rows.map((row) => Object.fromEntries([...row.querySelectorAll("td[data-label]")].map((cell) => [cell.getAttribute("data-label"), cell.textContent.trim()]))),
      firstRow: box(rows[0]), firstRowCells: [...(rows[0]?.querySelectorAll("td[data-label]") ?? [])].map((cell) => {
        const range = document.createRange(); range.selectNodeContents(cell);
        const bounds = box(cell), values = [...range.getClientRects()].filter((r) => r.width > 0 && r.height > 0);
        return { label: cell.getAttribute("data-label"), bounds, fullWidth: cell.getAttribute("data-mobile-layout") === "full-width",
          labelContent: getComputedStyle(cell, "::before").content.replace(/^"|"$/g, ""), columns: getComputedStyle(cell).gridTemplateColumns.split(" ").length,
          valuesContained: Boolean(bounds && values.length && values.every((r) => r.left >= bounds.x - 1 && r.right <= bounds.right + 1)) };
      }),
      firstIdentity: box(identity), expectedFirstName: firstName,
      firstUseful, firstUsefulVisible: Boolean(firstUseful && firstUseful.y >= 0 && firstUseful.bottom <= innerHeight),
      metrics, metricColumns: overview ? getComputedStyle(overview.querySelector(".ropa-dashboard-strip__metrics")).gridTemplateColumns.split(" ").length : null,
      primary, aside, placement: primary && aside ? aside.x >= primary.right - 1 ? "beside" : "stacked" : null,
      primaryHeadings: [...(overview?.querySelectorAll(".ropa-overview__primary h2") ?? [])].map((node) => node.textContent.trim()),
      coverageText: overview?.querySelector(".ropa-overview__coverage")?.textContent ?? "",
      text: panel?.textContent ?? "", minTargetWidth: targets.length ? Math.min(...targets.map((item) => item.bounds.width)) : null,
      minTargetHeight: targets.length ? Math.min(...targets.map((item) => item.bounds.height)) : null,
      undersizedTargets: targets.filter(({ bounds }) => bounds.width < 44 || bounds.height < 44),
      documentWidth: document.documentElement.clientWidth, scrollWidth: document.documentElement.scrollWidth,
    };
  }, { view: scenario.privacyView, firstName: privacyActivityName });
}

async function registerNames(page) {
  return page.locator('.ropa-register-list:not([hidden]) .ropa-register-identity strong').allTextContents();
}

export async function openPrivacyActivity(page, { keyboard = false, notFound = false } = {}) {
  if (notFound) await page.evaluate(() => { location.hash = "#ropa/activity/privacy-missing-record"; });
  else {
    const row = page.getByRole("row", { name: new RegExp(`^${privacyActivityName},`) });
    if (keyboard) await row.press("Enter");
    else await row.getByRole("button", { name: /^View details for / }).click();
  }
  const dialog = page.getByRole("dialog", { name: "Processing activity", exact: true });
  await dialog.getByRole("heading", { name: notFound ? "Processing activity not found" : privacyActivityName, exact: true }).waitFor();
  if (!notFound) await dialog.getByRole("heading", { name: "Changes", exact: true }).waitFor();
  await page.waitForFunction((missing) => window.privacyEvidence?.reads.some((read) => read.path.endsWith(missing ? "/privacy-missing-record" : "/ropa-activity-customer-account-opening/history")), notFound);
  if (!notFound) await dialog.locator(".ropa-history-list li").nth(2).waitFor();
  const result = await dialog.evaluate((node) => {
    const rect = (element) => { const r = element.getBoundingClientRect(); return { x: r.x, y: r.y, width: r.width, height: r.height, right: r.right, bottom: r.bottom }; };
    const sheet = node.closest(".cs-sheet");
    return { route: location.hash, bounds: rect(sheet), dialog: rect(node), scrollWidth: node.scrollWidth, clientWidth: node.clientWidth,
      heading: node.querySelector(".ropa-activity-heading h2")?.textContent.trim() ?? node.querySelector("h2")?.textContent.trim(),
      facts: Object.fromEntries([...node.querySelectorAll(".ropa-fact-grid > div")].map((item) => [item.querySelector("dt")?.textContent, item.querySelector("dd")?.textContent])),
      historyEntries: node.querySelectorAll(".ropa-history-list li").length,
      focusedInside: node.contains(document.activeElement), closeBounds: rect(node.querySelector('[aria-label="Close activity details"]')),
      unavailable: node.textContent.includes("Processing activity not found"),
    };
  });
  // Verify the modal keeps keyboard focus inside without advancing a record command.
  await dialog.getByRole("button", { name: "Close activity details", exact: true }).focus();
  await page.keyboard.press("Shift+Tab");
  result.keyboardContained = await dialog.evaluate((node) => node.contains(document.activeElement));
  await dialog.getByRole("button", { name: "Close activity details", exact: true }).focus();
  return result;
}

export async function verifyPrivacyHistory(page) {
  const menu = await page.getByRole("tablist", { name: "Privacy views" }).elementHandle();
  const document = await page.locator("html").elementHandle();
  const names = await registerNames(page), records = [];
  const capture = async (action, detail = false) => {
    if (detail) await page.getByRole("dialog", { name: "Processing activity" }).getByRole("heading", { name: privacyActivityName, exact: true }).waitFor();
    else await page.locator(".ropa-register-list").getByText(privacyActivityName, { exact: true }).waitFor();
    const state = { action, route: new URL(page.url()).hash, sameMenu: await menu.evaluate((node) => node.isConnected), sameDocument: await document.evaluate((node) => node.isConnected),
      detail: await page.getByRole("dialog", { name: "Processing activity" }).isVisible(), names: detail ? [] : await registerNames(page) };
    records.push(state);
    return state;
  };
  await openPrivacyActivity(page, { keyboard: true });
  await page.goBack(); await capture("back");
  await page.goForward(); await capture("forward", true);
  await page.goBack(); await capture("back-restored");
  const search = page.getByRole("searchbox", { name: "Search processing activities" });
  await search.fill("Customer");
  await page.waitForFunction(() => document.querySelectorAll(".ropa-register-list tbody tr").length === 2);
  const filtered = await registerNames(page);
  await openPrivacyActivity(page);
  await page.goBack();
  await capture("filtered-back");
  const retainedSearch = await search.inputValue();
  await search.fill("");
  await page.waitForFunction((count) => document.querySelectorAll(".ropa-register-list tbody tr").length === count, names.length);
  await page.getByRole("tab", { name: "Overview", exact: true }).click();
  await page.locator(".ropa-overview .cs-metric-card").first().waitFor();
  await page.getByRole("tab", { name: "ROPA", exact: true }).click();
  await page.goBack();
  await page.getByRole("tab", { name: "Overview", exact: true, selected: true }).waitFor();
  const peerBack = { route: new URL(page.url()).hash, sameMenu: await menu.evaluate((node) => node.isConnected), sameDocument: await document.evaluate((node) => node.isConnected) };
  await page.goForward(); await capture("peer-forward");
  const countedDrills = await verifyPrivacyCountedDrills(page);
  await page.evaluate((id) => { location.hash = `#ropa/activity/${id}`; }, privacyActivityID);
  await page.getByRole("dialog", { name: "Processing activity" }).getByRole("heading", { name: privacyActivityName, exact: true }).waitFor();
  const firstDocument = await page.evaluate(() => window.privacyEvidence);
  await page.reload({ waitUntil: "networkidle" });
  await page.getByRole("dialog", { name: "Processing activity" }).getByRole("heading", { name: privacyActivityName, exact: true }).waitFor();
  const directReload = { route: new URL(page.url()).hash, newDocument: !await document.evaluate((node) => node.isConnected).catch(() => false) };
  await page.getByRole("button", { name: "Close activity details", exact: true }).click();
  await page.locator(".ropa-register-list").getByText(privacyActivityName, { exact: true }).waitFor();
  return { names, records, filtered, retainedSearch, peerBack, countedDrills, firstDocument, directReload, restored: new URL(page.url()).hash === "#ropa", restoredNames: await registerNames(page) };
}

async function verifyPrivacyCountedDrills(page) {
  const snapshots = [];
  const read = async (step, count) => {
    await page.waitForFunction((count) => document.querySelectorAll(".ropa-register-list tbody tr").length === count, count);
    snapshots.push({ step, names: await registerNames(page), includesEnded: await page.getByRole("button", { name: "Remove Activity scope filter", exact: true }).isVisible(),
      read: await page.evaluate(() => window.privacyEvidence.reads.filter((item) => item.path === "/api/v1/ropa/processing-activities").at(-1)) });
  };
  const defaultNames = await registerNames(page);
  await page.getByRole("tab", { name: "Overview", exact: true }).click();
  const summary = await page.evaluate(() => window.privacyEvidence.reads.filter((item) => item.summary).at(-1).summary);
  await page.locator(".ropa-dashboard-strip__metrics").getByRole("button", { name: /^Processing activities.*Open ROPA/ }).click();
  await read("total", summary.counts.total);
  await page.getByRole("button", { name: "Remove Activity scope filter", exact: true }).click();
  await read("remove-scope", defaultNames.length);
  await page.getByRole("tab", { name: "Overview", exact: true }).click();
  await page.getByRole("button", { name: `Complete: ${summary.counts.closed}`, exact: true }).click();
  await read("complete", summary.counts.closed);
  await page.getByRole("button", { name: "Clear filters", exact: true }).click();
  await read("clear", defaultNames.length);
  return { summary, defaultNames, snapshots };
}

export async function verifyPrivacyFilters(page) {
  const snapshots = [];
  const read = async (step, name) => {
    await page.locator(".ropa-register-list").getByText(name, { exact: true }).waitFor();
    const value = await page.evaluate(() => window.privacyEvidence.reads.filter((entry) => entry.path === "/api/v1/ropa/processing-activities").at(-1));
    snapshots.push({ step, names: await registerNames(page), read: value });
  };
  await read("initial", privacyActivityName);
  const search = page.getByRole("searchbox", { name: "Search processing activities" });
  await search.fill("Azure"); await read("search", "Azure user access management");
  await page.getByRole("button", { name: "Load next page", exact: true }).click(); await read("next", "Azure device compliance management");
  await page.getByRole("button", { name: /Activity status/ }).click();
  await page.getByRole("option", { name: "In progress", exact: true }).click();
  await read("status-reset", "Azure user access management");
  await page.getByRole("button", { name: "Load next page", exact: true }).click(); await read("status-next", "Azure device compliance management");
  await search.fill("Customer"); await read("query-reset", privacyActivityName);
  await page.getByRole("button", { name: "Clear filters", exact: true }).click();
  await page.waitForFunction(() => { const last = window.privacyEvidence.reads.filter((entry) => entry.path === "/api/v1/ropa/processing-activities").at(-1); return last && !new URLSearchParams(last.query).has("search") && !new URLSearchParams(last.query).has("status") && !new URLSearchParams(last.query).has("cursor"); });
  await read("clear", privacyActivityName);
  return { snapshots, clearedSearch: await search.inputValue() };
}

export async function measurePrivacySourceLayout(page, view) {
  return page.evaluate((kind) => {
    const box = (node) => { if (!node || !node.getClientRects().length) return null; const r = node.getBoundingClientRect(); return { x: r.x, y: r.y, width: r.width, height: r.height, right: r.right, bottom: r.bottom }; };
    const heading = document.querySelector(".privacy-heading"), menu = document.querySelector(".privacy-tabs");
    const tab = menu?.querySelector('[aria-selected="true"]'), panel = tab && document.getElementById(tab.getAttribute("aria-controls"));
    const table = panel?.querySelector("table"), first = table?.querySelector("tbody tr strong");
    const metrics = [...(panel?.querySelectorAll(".privacy-metric") ?? [])].map((node) => ({ label: node.querySelector("span")?.textContent.trim(), bounds: box(node) }));
    const inspector = document.querySelector('.privacy-drawer:not([hidden])');
    return { kind, heading: box(heading), headingText: heading?.querySelector("h1")?.textContent.trim(), menu: box(menu),
      localTabs: [...(menu?.querySelectorAll('[role="tab"]') ?? [])].map((node) => node.textContent.trim()), selectedTab: tab?.textContent.trim(),
      metrics, controls: box(panel?.querySelector(".table-toolbar")), table: box(table), firstIdentity: box(first),
      headers: [...(table?.querySelectorAll("th") ?? [])].map((node) => node.textContent.trim()),
      tablePresentation: table ? getComputedStyle(table).display : null,
      firstName: first?.textContent.trim(),
      metricColumns: panel?.querySelector(".privacy-metrics") ? getComputedStyle(panel.querySelector(".privacy-metrics")).gridTemplateColumns.split(" ").length : null,
      primary: box(panel?.querySelector(".privacy-primary")), aside: box(panel?.querySelector(".privacy-context")),
      inspector: box(inspector), inspectorHeading: inspector?.querySelector("h2")?.textContent.trim(),
      documentWidth: document.documentElement.clientWidth, scrollWidth: document.documentElement.scrollWidth };
  }, view);
}
