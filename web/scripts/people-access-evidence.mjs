import { peopleAccessViews } from "./people-access-evidence-scenarios.mjs";

const selectors = {
  people: { records: ".people-directory__records", detail: ".people-directory__detail" },
  roles: { records: ".people-role-catalog__records", detail: ".people-role-catalog__detail" },
  routes: { records: ".people-governance-layout__records", detail: ".people-governance-layout__detail" },
  delegations: { records: ".people-governance-layout__records", detail: ".people-governance-layout__detail" },
};
export async function waitForPeopleAccess(page, scenario) {
  const root = page.locator(".people-access-workspace");
  await root.getByRole("heading", { name: "People & access", exact: true }).waitFor();
  const content = root.locator('[role="tabpanel"]:visible');
  const check = scenario.peopleAccessCheck;
  if (check === "denied") await root.getByText(/restricted|unavailable with your current access|could not be loaded/i).first().waitFor();
  else if (check === "recovery") await root.getByText("People and access settings could not be loaded.", { exact: true }).waitFor();
  else if (check === "empty") await root.getByText(/No people returned|No role definitions returned|No routing policies|No approval routes|No delegations/i).first().waitFor();
  else if (check === "partial" && scenario.peopleAccessView === "routes") await root.getByText(/unavailable|could not be loaded/i).first().waitFor();
  else await content.locator(`${selectors[scenario.peopleAccessView].records} .cs-selectable-record`).first().waitFor();
}
export async function measurePeopleAccessLayout(page, scenario) {
  return page.evaluate(({ view, selectors }) => {
    const root = document.querySelector(".people-access-workspace"), menu = root?.querySelector('[role="tablist"]');
    const visible = (node) => Boolean(node?.getClientRects().length && getComputedStyle(node).visibility !== "hidden");
    const box = (node) => { if (!visible(node)) return null; const r = node.getBoundingClientRect(); return { x: r.x, y: r.y, width: r.width, height: r.height, right: r.right, bottom: r.bottom }; };
    const tabs = [...(menu?.querySelectorAll('[role="tab"]') ?? [])], selected = tabs.find((node) => node.getAttribute("aria-selected") === "true");
    const content = selected?.getAttribute("aria-controls") ? document.getElementById(selected.getAttribute("aria-controls")) : root;
    const recordList = content?.querySelector(selectors.records), records = [...(recordList?.querySelectorAll(".cs-selectable-record") ?? [])];
    const first = records[0], identity = first?.querySelector(".cs-selectable-record__title"), primary = content?.querySelector(".people-access-list-panel"), detail = content?.querySelector(selectors.detail);
    const bounds = box(identity), panel = box(primary), detailBounds = box(detail), menuBounds = box(menu), selectedBounds = box(selected);
    const targets = [...(root?.querySelectorAll("button,[role=tab],input,[role=combobox],a[href]") ?? [])].filter(visible).map((node) => ({ label: node.getAttribute("aria-label") || node.textContent.trim(), ...box(node) }));
    const summary = content?.querySelector(".people-access-summary,.people-governance-summary");
    return {
      evidenceKind: "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION", kind: view, route: location.hash,
      heading: box(root?.querySelector(".people-access-workspace__heading") ?? root?.querySelector("header")), headingText: root?.querySelector("h1")?.textContent.trim(), headingCount: root?.querySelectorAll("h1").length,
      menu: menuBounds, localTabs: tabs.map((node) => node.textContent.trim()), selectedTab: selected?.textContent.trim(), selectedBounds,
      selectedVisible: Boolean(selectedBounds && menuBounds && selectedBounds.x >= menuBounds.x - 1 && selectedBounds.right <= menuBounds.right + 1),
      visiblePanelIDs: [...(root?.querySelectorAll('[role="tabpanel"]') ?? [])].filter(visible).map((node) => node.id),
      selectedPanelID: selected?.getAttribute("aria-controls"), settingsCurrent: document.querySelector('.sidebar-secondary a[href="#configure"]')?.getAttribute("aria-current"),
      summary: box(summary), summaryText: summary?.textContent.trim(),
      metrics: [...(summary?.querySelectorAll(".cs-metric-card") ?? [])].map((node) => ({ text: node.textContent.trim(), label: node.querySelector(".cs-metric-card__label")?.textContent.trim(), value: node.querySelector(".cs-metric-card__measure strong")?.textContent.trim(), quality: node.getAttribute("data-quality"), bounds: box(node) })),
      controls: box(content?.querySelector(".people-directory__toolbar")), primary: panel, context: detailBounds,
      placement: panel && detailBounds ? (detailBounds.x >= panel.right - 1 ? "beside" : "stacked") : null,
      firstRow: box(first), firstRowVisible: Boolean(first && box(first).y >= 0 && box(first).bottom <= innerHeight), firstIdentity: bounds, firstUsefulVisible: Boolean(bounds && bounds.y >= 0 && bounds.bottom <= innerHeight && bounds.x >= 0 && bounds.right <= innerWidth),
      recordNames: records.map((node) => node.querySelector(".cs-selectable-record__title")?.textContent.trim()), recordCount: records.length,
      selectedRecord: records.find((node) => node.getAttribute("aria-pressed") === "true" || node.getAttribute("data-selected") === "true")?.querySelector(".cs-selectable-record__title")?.textContent.trim(),
      detailHeading: detail?.querySelector("h2")?.textContent.trim(), detailText: detail?.textContent.trim(), contextHeading: box(detail?.querySelector("h2")), stickyContextBar: box(document.querySelector(".context-bar")),
      contextClientWidth: detail?.clientWidth, contextScrollWidth: detail?.scrollWidth, detailFocused: document.activeElement === detail,
      footerText: content?.querySelector(".people-access-list-footer")?.textContent.trim(),
      targets, minTargetWidth: targets.length ? Math.min(...targets.map((node) => node.width)) : 0, minTargetHeight: targets.length ? Math.min(...targets.map((node) => node.height)) : 0,
      undersizedTargets: targets.filter((node) => node.width < 44 - 1 || node.height < 44 - 1),
      documentWidth: document.documentElement.clientWidth, scrollWidth: document.documentElement.scrollWidth, text: root?.textContent.trim(),
      legacyRail: Boolean(root?.querySelector(".configure-navigation")),
    };
  }, { view: scenario.peopleAccessView, selectors: selectors[scenario.peopleAccessView] });
}
export async function openPeopleAccessContext(page, scenario) {
  const content = page.locator('.people-access-workspace [role="tabpanel"]:visible');
  const record = content.locator(`${selectors[scenario.peopleAccessView].records} .cs-selectable-record`).first();
  await record.press("Enter");
  const detail = content.locator(selectors[scenario.peopleAccessView].detail);
  await detail.locator("h2").waitFor();
  await detail.scrollIntoViewIfNeeded();
  return { selectionViaKeyboard: true, route: await page.evaluate(() => location.hash), name: await detail.locator("h2").textContent(), focused: await detail.evaluate((node) => node === document.activeElement) };
}
export async function verifyPeopleAccessFilters(page) {
  const list = page.locator(".people-directory__records"), search = page.getByRole("searchbox", { name: "Search people" });
  const snapshots = [];
  const record = async (step) => snapshots.push({ step, names: await list.locator(".cs-selectable-record__title").allTextContents(), reads: await page.evaluate(() => window.peopleAccessEvidence.reads.length) });
  await record("loaded");
  await search.fill("Zara"); await page.getByRole("button", { name: "Inspect Zara Nwosu", exact: true }).waitFor(); await record("search-twelfth");
  await search.fill("zzzz-none"); await page.getByText("No people match these filters", { exact: true }).waitFor(); await record("empty-search");
  await page.getByRole("button", { name: "Clear filters", exact: true }).click(); await record("cleared");
  const choose = async (label, option) => { await page.getByRole("button", { name: new RegExp(label) }).click(); await page.getByRole("option", { name: option, exact: true }).click(); };
  await choose("Profile status", "Inactive"); await record("inactive");
  await choose("Profile status", "Any status");
  await choose("Provisioning source", "Source not returned"); await record("source-not-returned");
  await choose("Provisioning source", "All sources"); await record("restored");
  return { snapshots, search: await search.inputValue() };
}
export async function verifyPeopleAccessHistory(page) {
  await page.evaluate(() => { window.__peopleDocument = document; window.__peopleMenu = document.querySelector('.people-access-workspace [role="tablist"]'); });
  const queryless = [];
  const tabs = page.getByRole("tablist", { name: /People.*access/i });
  for (const view of peopleAccessViews) {
    if (view.id !== "people") await tabs.getByRole("tab", { name: view.label, exact: true }).click();
    await waitForPeopleAccess(page, { peopleAccessView: view.id, peopleAccessCheck: "layout" });
    const content = page.locator('.people-access-workspace [role="tabpanel"]:visible');
    const list = content.locator(`${selectors[view.id].records} .cs-selectable-record`);
    const detail = content.locator(`${selectors[view.id].detail} h2`);
    const first = await detail.textContent(), querylessRoute = await page.evaluate(() => location.hash);
    await list.nth(1).press("Enter");
    const second = await detail.textContent(), selectedRoute = await page.evaluate(() => location.hash);
    await page.goBack();
    await detail.filter({ hasText: first }).waitFor();
    const restored = await detail.textContent(), restoredRoute = await page.evaluate(() => location.hash);
    queryless.push({ view: view.id, first, second, restored, querylessRoute, selectedRoute, restoredRoute });
  }
  await tabs.getByRole("tab", { name: "People", exact: true }).click();
  await waitForPeopleAccess(page, { peopleAccessView: "people", peopleAccessCheck: "layout" });
  const snapshots = [];
  const snapshot = async (step) => snapshots.push(await page.evaluate((step) => { const menu = document.querySelector('.people-access-workspace [role="tablist"]'), selected = menu?.querySelector('[aria-selected="true"]'), m = menu?.getBoundingClientRect(), s = selected?.getBoundingClientRect(); return { step, route: location.hash, sameDocument: document === window.__peopleDocument, sameMenu: menu === window.__peopleMenu, selected: selected?.textContent.trim(), selectedVisible: Boolean(s && m && s.x >= m.x - 1 && s.right <= m.right + 1), search: document.querySelector('.people-directory__toolbar input')?.value, names: [...document.querySelectorAll('.people-directory__records .cs-selectable-record__title')].map((node) => node.textContent.trim()), detail: document.querySelector('.people-directory__detail h2')?.textContent.trim() }; }, step));
  for (const view of peopleAccessViews.slice(1)) { await tabs.getByRole("tab", { name: view.label, exact: true }).click(); await waitForPeopleAccess(page, { peopleAccessView: view.id, peopleAccessCheck: "layout" }); }
  await snapshot("delegations");
  for (const view of [...peopleAccessViews.slice(0, 3)].reverse()) { await page.goBack(); await waitForPeopleAccess(page, { peopleAccessView: view.id, peopleAccessCheck: "layout" }); await snapshot(`back-${view.id}`); }
  await page.goForward(); await waitForPeopleAccess(page, { peopleAccessView: "roles", peopleAccessCheck: "layout" }); await snapshot("forward-roles");
  await page.goBack(); await waitForPeopleAccess(page, { peopleAccessView: "people", peopleAccessCheck: "layout" });
  await page.getByRole("searchbox", { name: "Search people" }).fill("Zara");
  await page.getByRole("button", { name: "Inspect Zara Nwosu", exact: true }).press("Enter"); await snapshot("selected-person");
  await page.goBack(); await waitForPeopleAccess(page, { peopleAccessView: "people", peopleAccessCheck: "layout" }); await snapshot("back-person");
  await page.goForward(); await page.getByRole("heading", { name: "Zara Nwosu", exact: true }).waitFor(); await snapshot("forward-person");
  await page.goBack(); await page.getByRole("searchbox", { name: "Search people" }).fill("");
  const organization = [];
  for (const name of ["Add position", "Add area"]) {
    const peopleRoute = await page.evaluate(() => location.hash);
    await page.getByRole("button", { name: "Organization settings", exact: true }).click();
    const entry = page.getByRole("button", { name, exact: true });
    await entry.click();
    const dialog = page.getByRole("dialog", { name, exact: true });
    await dialog.waitFor();
    const route = await page.evaluate(() => location.hash), focused = await dialog.evaluate((node) => node.contains(document.activeElement));
    await page.goBack(); await dialog.waitFor({ state: "hidden" });
    await waitForPeopleAccess(page, { peopleAccessView: "people", peopleAccessCheck: "layout" });
    const back = { route: await page.evaluate(() => location.hash), selected: await tabs.getByRole("tab", { selected: true }).textContent(), dialogCount: await page.getByRole("dialog").count() };
    await page.goForward(); await entry.waitFor();
    const forward = { route: await page.evaluate(() => location.hash), selected: await tabs.getByRole("tab", { selected: true }).textContent(), dialogCount: await page.getByRole("dialog").count() };
    organization.push({ name, peopleRoute, route, focused, back, forward });
    await page.goBack(); await waitForPeopleAccess(page, { peopleAccessView: "people", peopleAccessCheck: "layout" });
  }
  return { snapshots, queryless, organization };
}
export async function verifyPeopleAccessStickyContext(page) {
  await page.evaluate(() => scrollTo({ top: 500, behavior: "instant" }));
  await page.evaluate(() => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))));
  const result = await page.evaluate(() => {
    const box = (node) => { const r = node?.getBoundingClientRect(); return r ? { x:r.x,y:r.y,width:r.width,height:r.height,right:r.right,bottom:r.bottom } : null; };
    return { scrollY, context: box(document.querySelector(".people-directory__detail")), heading: box(document.querySelector(".people-directory__detail h2")), bar: box(document.querySelector(".context-bar")) };
  });
  await page.evaluate(() => scrollTo({ top: 0, behavior: "instant" }));
  return result;
}
export async function verifyPeopleAccessComposer(page, scenario) {
  const view = peopleAccessViews.find((item) => item.id === scenario.peopleAccessView);
  const tabs = page.getByRole("tablist", { name: "People and access views" });
  const content = page.locator('.people-access-workspace [role="tabpanel"]:visible');
  await content.locator(`${selectors[view.id].records} .cs-selectable-record`).first().press("Enter");
  await tabs.getByRole("tab", { name: "People", exact: true }).click();
  await waitForPeopleAccess(page, { peopleAccessView: "people", peopleAccessCheck: "layout" });
  await tabs.getByRole("tab", { name: view.label, exact: true }).click();
  await waitForPeopleAccess(page, { peopleAccessView: view.id, peopleAccessCheck: "layout" });
  const entryName = view.id === "routes" ? "New routing policy" : "Propose delegation";
  const dialogName = view.id === "routes" ? "New routing policy" : "New delegation";
  const entry = page.getByRole("button", { name: entryName, exact: true });
  const dialog = page.getByRole("dialog", { name: dialogName, exact: true });
  const route = await page.evaluate(() => location.hash);
  await entry.click(); await dialog.waitFor();
  const initialFocusInside = await dialog.evaluate((node) => node.contains(document.activeElement));
  await page.keyboard.press("Shift+Tab");
  const focusTrapped = await dialog.evaluate((node) => node.contains(document.activeElement));
  await page.keyboard.press("Escape"); await dialog.waitFor({ state: "hidden" });
  const focusRestored = await entry.evaluate((node) => node === document.activeElement);
  await entry.click(); await dialog.waitFor();
  await page.goBack();
  await waitForPeopleAccess(page, { peopleAccessView: "people", peopleAccessCheck: "layout" });
  const back = { route: await page.evaluate(() => location.hash), selected: await tabs.getByRole("tab", { selected: true }).textContent(), dialogCount: await page.getByRole("dialog").count() };
  await page.goForward();
  await waitForPeopleAccess(page, { peopleAccessView: view.id, peopleAccessCheck: "layout" });
  const forward = { route: await page.evaluate(() => location.hash), dialogCount: await page.getByRole("dialog").count() };
  await entry.click(); await dialog.waitFor();
  let candidateRecovery;
  if (view.id === "delegations") {
    await dialog.getByRole("button", { name: /Responsibility/ }).click();
    await page.getByRole("option", { name: "Reviewer", exact: true }).click();
    const recovery = dialog.getByText("No eligible people were found for this responsibility in the current legal entity. Ask the access administrator to check current assignments.", { exact: true });
    await recovery.waitFor();
    candidateRecovery = { text: await recovery.textContent(), disabledSubmit: await dialog.getByRole("button", { name: "Create delegation draft", exact: true }).isDisabled() };
  }
  await dialog.evaluate((node) => { for (let parent = node; parent; parent = parent.parentElement) parent.scrollTop = 0; });
  const geometry = await dialog.evaluate((node) => {
    const box = (element) => { const r = element.getBoundingClientRect(); return { x:r.x,y:r.y,width:r.width,height:r.height,right:r.right,bottom:r.bottom }; };
    const sheet = node.closest(".cs-sheet");
    return { dialog: box(node), sheet: box(sheet), heading: box(node.querySelector("h3")), headingText: node.querySelector("h3")?.textContent.trim(), clientWidth: sheet.clientWidth, scrollWidth: sheet.scrollWidth,
      targets: [...node.querySelectorAll("button,input,select,textarea")].filter((element) => !element.closest('[aria-hidden="true"],[hidden]') && element.getClientRects().length && getComputedStyle(element).visibility !== "hidden").map((element) => ({ label: element.getAttribute("aria-label") || element.labels?.[0]?.textContent.trim() || element.textContent, disabled: element.disabled, ...box(element) })) };
  });
  return { route, initialFocusInside, focusTrapped, escapeClosed: true, focusRestored, back, forward, candidateRecovery, ...geometry };
}
export async function verifyPeopleAccessDefinition(page) {
  const content = page.locator('.people-access-workspace [role="tabpanel"]:visible');
  await content.getByRole("button", { name: "Inspect policy Independent evidence review", exact: true }).press("Enter");
  const detail = content.locator(".people-governance-layout__detail");
  await detail.getByRole("heading", { name: "Independent evidence review", exact: true }).waitFor();
  await detail.getByRole("button", { name: "View full policy definition", exact: true }).click();
  const region = detail.getByRole("region", { name: "Full policy definition", exact: true });
  await region.waitFor();
  const fullDefinition = JSON.parse(await region.textContent());
  const checksum = await detail.locator(".people-governance-definition > dl dd").textContent();
  const facts = await detail.locator(".people-governance-rules li").evaluateAll((rows) => rows.map((row) => ({ responsibility:row.querySelector("h4")?.textContent.trim(), fields:Object.fromEntries([...row.querySelectorAll("dt")].map((term)=>[term.textContent.trim(),term.nextElementSibling?.textContent.trim()])) })));
  const action = detail.getByRole("button", { name: "Approve Independent evidence review", exact: true });
  const returnAction = detail.getByRole("button", { name: "Return Independent evidence review for changes", exact: true });
  await returnAction.scrollIntoViewIfNeeded();
  const actions = await detail.evaluate((node) => {
    const bar = document.querySelector(".context-bar")?.getBoundingClientRect();
    return [...node.querySelectorAll("button")].filter((button) => /^(Approve|Return) Independent/.test(button.textContent.trim())).map((button) => { const r = button.getBoundingClientRect(); return { label: button.textContent.trim(), disabled: button.disabled, top:r.top,bottom:r.bottom,height:r.height,width:r.width,visible:r.top>=(bar?.bottom??0)&&r.bottom<=innerHeight }; });
  });
  await detail.getByRole("button", { name: "Hide policy definition", exact: true }).click(); await region.waitFor({ state: "hidden" });
  await detail.getByRole("button", { name: "View full policy definition", exact: true }).click(); await region.waitFor();
  await region.scrollIntoViewIfNeeded();
  const regionGeometry = await region.evaluate((node) => { const r=node.getBoundingClientRect(); return { x:r.x,y:r.y,width:r.width,height:r.height,right:r.right,bottom:r.bottom,clientWidth:node.clientWidth,scrollWidth:node.scrollWidth }; });
  await detail.locator(".people-governance-rules li").first().scrollIntoViewIfNeeded();
  const visibleRule = await detail.locator(".people-governance-rules li").first().evaluate((node) => { const r=node.getBoundingClientRect();return { x:r.x,y:r.y,width:r.width,height:r.height,right:r.right,bottom:r.bottom }; });
  return { route: await page.evaluate(() => location.hash), fullDefinition, checksum, facts, visibleRule, actions, approvalEnabled: await action.isEnabled(), returnDisabledWithoutReason: await returnAction.isDisabled(), expandedAfterToggle: true, region: regionGeometry };
}
export async function showPeopleAccessContinuation(page, scenario) {
  const target = scenario.peopleAccessCheck === "composer"
    ? page.getByRole("dialog").getByRole("button", { name: scenario.peopleAccessView === "routes" ? "Create policy draft" : "Create delegation draft", exact: true })
    : page.getByRole("button", { name: "Return Independent evidence review for changes", exact: true });
  await target.scrollIntoViewIfNeeded();
  return target.evaluate((node) => { const r=node.getBoundingClientRect();return { label:node.textContent.trim(),disabled:node.disabled,x:r.x,y:r.y,width:r.width,height:r.height,right:r.right,bottom:r.bottom }; });
}
export async function measurePeopleAccessSourceLayout(page, view) {
  return page.evaluate((view) => {
    const visible = (node) => Boolean(node?.getClientRects().length && getComputedStyle(node).visibility !== "hidden");
    const box = (node) => { if (!visible(node)) return null; const r = node.getBoundingClientRect(); return { x:r.x,y:r.y,width:r.width,height:r.height,right:r.right,bottom:r.bottom }; };
    const panel = document.querySelector(`#${view}`), list = panel?.querySelector(".people-list,.role-list,.route-list,.delegation-list"), primary = panel?.querySelector(".access-surface"), detail = panel?.querySelector("aside"), first = list?.querySelector("button strong");
    const p = box(primary), d = box(detail), f = box(first);
    return { kind: view, headingText: document.querySelector("h1")?.textContent.trim(), heading:box(document.querySelector(".access-heading")), menu:box(document.querySelector(".access-tabs")), localTabs:[...document.querySelectorAll('.access-tabs [role="tab"]')].map((node)=>node.childNodes[0].textContent.trim()), selectedTab: document.querySelector('.access-tabs [aria-selected="true"]')?.childNodes[0].textContent.trim(), summary:box(document.querySelector(".access-summary")), controls:box(panel?.querySelector(".people-toolbar")), primary:p, context:d, firstIdentity:f, firstName:first?.textContent.trim(), detailHeading:detail?.querySelector("h2")?.textContent.trim(), contextHeading:box(detail?.querySelector("h2")), placement:p&&d ? d.x >= p.right -1 ? "beside":"stacked":null, firstUsefulVisible:Boolean(f&&f.y>=0&&f.bottom<=innerHeight), recordCount:list?.querySelectorAll("button").length, documentWidth:document.documentElement.clientWidth,scrollWidth:document.documentElement.scrollWidth, declaredFontFamily:getComputedStyle(document.body).fontFamily };
  }, view);
}
