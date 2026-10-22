import { vendorPortfolioViews } from "./vendor-portfolio-evidence-scenarios.mjs";

const labels = vendorPortfolioViews.map((view) => view.label);
const relationshipID = "vendor-relationship-payments";
const root = (page) => page.locator(".vendor-portfolio-workspace");
const menu = (page) => page.getByRole("tablist", { name: "Vendor management sections", exact: true });
const labelFor = (view) => vendorPortfolioViews.find((item) => item.id === view)?.label;
const regionFor = (view) => ({ onboarding: "Vendor onboarding", assessments: "Vendor assessments", documents: "Vendor documents" })[view];
const resetScroll = (page) => page.evaluate(() => window.scrollTo(0, 0));

export async function selectVendorPortfolioView(page, view) {
  const label = labelFor(view);
  if (!label) throw new Error(`Unknown Vendor portfolio view: ${view}`);
  const tab = menu(page).getByRole("tab", { name: label, exact: true });
  if (await tab.isVisible()) await tab.click();
  else {
    await root(page).locator('.vendor-portfolio-navigation > .cs-tabs > .cs-tabs__compact .cs-select-field__trigger').click();
    await page.getByRole("option", { name: label, exact: true }).click();
  }
  await page.waitForFunction((view) => location.hash === `#vendors/${view}`, view);
}

export async function prepareVendorPortfolio(page, scenario) {
  // Install the observer before a normal route remount so that the receipt
  // contains the reads which actually populated the rendered portfolio.
  await installVendorPortfolioObservation(page, scenario.vendorPortfolioCheck);
  const primary = page.getByRole("complementary", { name: "Primary navigation", exact: true });
  await primary.getByRole("link", { name: "Home", exact: true }).click();
  await page.getByRole("heading", { name: "Oversight", exact: true }).first().waitFor();
  await primary.getByRole("link", { name: "Vendors", exact: true }).click();
  await root(page).getByRole("heading", { name: "Vendors", exact: true }).waitFor();
  if (scenario.vendorPortfolioView === "overview") await selectVendorPortfolioView(page, "register");
  await selectVendorPortfolioView(page, scenario.vendorPortfolioView);
  await waitForPortfolio(page, scenario);
  await page.evaluate(() => document.fonts?.ready);
  await resetScroll(page);
}

async function waitForPortfolio(page, scenario) {
  const check = scenario.vendorPortfolioCheck;
  if (check === "loading") { await root(page).getByText(/Loading vendor relationships/).waitFor(); return; }
  if (["unavailable", "denied"].includes(check)) { await root(page).getByRole("heading", { name: "Vendor records are unavailable", exact: true }).waitFor(); return; }
  if (check === "empty") { await root(page).getByText(/No vendor relationships found/).waitFor(); return; }
  if (["summaries-unavailable", "summary-missing"].includes(check)) {
    await root(page).getByText(/Some (vendor )?form summaries (could not be loaded|are unavailable)/).waitFor();
  } else if (check === "exceptions-unavailable") {
    await root(page).getByRole("button", { name: "Unknown open exceptions", exact: true }).waitFor();
  } else if (scenario.vendorPortfolioView === "overview") {
    await root(page).getByRole("button", { name: "5 open exceptions", exact: true }).waitFor();
  }
  const collection = regionFor(scenario.vendorPortfolioView);
  if (collection) await root(page).getByRole("region", { name: collection, exact: true }).waitFor();
  await root(page).locator('.cs-selectable-record:visible').first().waitFor();
}

export async function installVendorPortfolioObservation(page, check) {
  await page.evaluate((check) => {
    const original = window.fetch;
    const log = { reads: [], commands: [], check, fixturePopulation: [], fixtureSummaryIDs: [], fixtureRecords: [], pendingAPIRequests: 0, lastAPIActivity: performance.now() };
    window.vendorPortfolioEvidence = log;
    const json = (body, status = 200) => new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
    const dispatch = async (input, init) => {
      const url = new URL(typeof input === "string" ? input : input.url, location.href);
      const method = String(init?.method ?? (typeof input === "object" ? input.method : null) ?? "GET").toUpperCase();
      const relevant = url.pathname.startsWith("/api/v1/vendors") || url.pathname.startsWith("/api/v1/vendor-")
        || url.pathname === "/api/v1/forms/documents" || url.pathname === "/api/v1/forms/responses";
      const entry = { path: url.pathname, query: url.search, method, initiatedAt: performance.now() };
      if (method === "POST" && url.pathname === "/api/v1/authority/resolve") {
        const body = JSON.parse(init?.body ?? "{}");
        entry.queryInput = { object_type: body.object_type, object_id: body.object_id };
        entry.requestKind = "AUTHORITY_READ_QUERY";
        log.reads.push(entry);
      } else if (method === "GET" && relevant) log.reads.push(entry);
      else if (!["GET", "HEAD", "OPTIONS"].includes(method) && url.pathname.startsWith("/api/v1/")) log.commands.push(entry);
      const listing = method === "GET" && url.pathname === "/api/v1/vendors";
      const summaries = method === "GET" && url.pathname === "/api/v1/vendors/form-summaries";
      if (listing && check === "loading") { entry.status = "PENDING"; return new Promise(() => {}); }
      if (listing && ["unavailable", "denied"].includes(check) || summaries && check === "summaries-unavailable"
        || method === "GET" && check === "exceptions-unavailable" && /^\/api\/v1\/vendors\/[^/]+\/links$/.test(url.pathname)) {
        entry.status = check === "denied" ? 403 : 503;
        return json({ code: check === "denied" ? "permission_denied" : "sample_unavailable", message: "Sample vendor information is unavailable." }, entry.status);
      }
      const result = await original(input, init);
      entry.status = result.status;
      if (method === "GET" && /^\/api\/v1\/matters\/sample-portfolio-finding-[0-4]$/.test(url.pathname) && result.ok) {
        // The legacy measure fixture omitted the required scope object. Give
        // this isolated navigation fixture the canonical read shape so the
        // real Matter workspace can render before testing browser Back.
        const body = await result.clone().json();
        body.matter.scope ??= {};
        return json(body);
      }
      if (listing && result.ok) {
        const body = await result.clone().json();
        if (check === "empty") body.items = [];
        if (check === "long-labels") {
          body.items[0].vendor.legal_name = "Sample · International Settlement and Merchant Assurance Services Limited";
          body.items[0].relationship.service_name = "Cross-border card transaction processing, delegated settlement and assurance reporting";
        }
        if (["partial", "search-paging"].includes(check)) {
          if (url.searchParams.get("cursor")) {
            if (url.searchParams.get("cursor") !== "sample-vendor-page-2") throw new Error("Unexpected synthetic vendor page cursor");
            const item = structuredClone(body.items[0]);
            item.vendor.id = "sample-vendor-page-2";
            item.vendor.legal_name = "Sample · Acme Settlement Services";
            item.relationship.id = "sample-vendor-relationship-page-2";
            item.relationship.vendor_id = item.vendor.id;
            item.relationship.service_name = "Acme settlement reconciliation";
            body.items = [item]; delete body.next_cursor;
          } else body.next_cursor = "sample-vendor-page-2";
        }
        entry.responseItems = body.items.length;
        entry.relationshipIDs = body.items.map((item) => item.relationship.id);
        entry.nextCursor = Boolean(body.next_cursor);
        log.fixturePopulation = [...new Set([...log.fixturePopulation, ...entry.relationshipIDs])];
        const known = new Map(log.fixtureRecords.map((record) => [record.id, record]));
        for (const item of body.items) known.set(item.relationship.id, { id: item.relationship.id, title: item.vendor.legal_name, service: item.relationship.service_name });
        log.fixtureRecords = [...known.values()];
        return json(body);
      }
      if (summaries && result.ok) {
        const body = await result.clone().json();
        if (check === "summary-missing") body.items = body.items.filter((item) => item.relationship_id !== "vendor-relationship-payments");
        const ids = url.searchParams.get("relationship_ids")?.split(",") ?? [];
        if (ids.includes("sample-vendor-relationship-page-2")) {
          const originalSummary = body.items.find((item) => item.relationship_id === "vendor-relationship-payments");
          if (!originalSummary) throw new Error("Synthetic pagination summary requires the original relationship");
          const summary = { ...originalSummary, relationship_id: "sample-vendor-relationship-page-2" };
          for (const key of ["outstanding_forms", "overdue_forms", "submitted_forms", "awaiting_review", "unassessed_forms", "assessed_forms", "partially_replaced_forms", "outdated_forms", "freshness_unknown_forms"]) summary[key] = 0;
          delete summary.highest_concern;
          const existing = body.items.findIndex((item) => item.relationship_id === summary.relationship_id);
          if (existing >= 0) body.items[existing] = summary; else body.items.push(summary);
        }
        entry.summaryIDs = body.items.map((item) => item.relationship_id);
        entry.responseItems = body.items.length;
        log.fixtureSummaryIDs = [...new Set([...log.fixtureSummaryIDs, ...entry.summaryIDs])];
        return json(body);
      }
      return result;
    };
    window.fetch = async (input, init) => {
      const url = new URL(typeof input === "string" ? input : input.url, location.href);
      if (!url.pathname.startsWith("/api/")) return dispatch(input, init);
      log.pendingAPIRequests += 1; log.lastAPIActivity = performance.now();
      try { return await dispatch(input, init); }
      finally { log.pendingAPIRequests -= 1; log.lastAPIActivity = performance.now(); }
    };
  }, check);
}

async function waitForRouteReadSettlement(page, route, requireObservedReads = true) {
  // Hash changes and fetch completion can precede React's selected-record
  // update. Wait for the route read to settle before inspecting the live DOM.
  await page.waitForFunction(({ route, requireObservedReads }) => {
    const observation = window.vendorPortfolioEvidence;
    return location.hash === route && (!requireObservedReads || observation)
      && (!observation || observation.pendingAPIRequests === 0 && performance.now() - observation.lastAPIActivity >= 100);
  }, { route, requireObservedReads }, { timeout: 10000 });
  await page.evaluate(() => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))));
}

export async function assertVendorPortfolioRouteState(page, view, { action, priorTimeOrigin, expectedRecordTitles, requireObservedReads = true } = {}) {
  await waitForRouteReadSettlement(page, `#vendors/${view}`, requireObservedReads);
  const state = await page.evaluate(({ view, action, priorTimeOrigin }) => {
    const visible = (node) => Boolean(node?.getClientRects().length && !node.closest('[hidden],[inert],[aria-hidden="true"]') && getComputedStyle(node).visibility !== "hidden");
    const portfolio = [...document.querySelectorAll(".vendor-portfolio-workspace")].find(visible);
    const detail = [...document.querySelectorAll(".vendor-relationship-workspace")].find(visible);
    const tabs = portfolio?.querySelector('.vendor-portfolio-navigation > .cs-tabs > [role="tablist"]');
    const selected = tabs?.querySelector('[aria-selected="true"]');
    const panel = selected ? document.getElementById(selected.getAttribute("aria-controls")) : null;
    const region = portfolio?.querySelector(view === "register" ? ".vendor-register" : view === "overview" ? ".vendor-portfolio-overview" : ".vendor-collection");
    const records = [...(region?.querySelectorAll(".cs-selectable-record") ?? [])].filter(visible);
    const recordTitles = records.map(node => node.querySelector(".cs-selectable-record__title")?.textContent.trim());
    const observation = window.vendorPortfolioEvidence;
    return { action, route: location.hash, view, heading: portfolio?.querySelector("h1")?.textContent.trim() ?? null,
      portfolioVisible: Boolean(portfolio), collectionVisible: visible(region), selectedRelationshipVisible: Boolean(detail), detailHeading: detail?.querySelector("h1")?.textContent.trim() ?? null,
      selectedTab: selected?.textContent.trim() ?? null, selectedPanelVisible: visible(panel), recordTitles, recordCount: records.length,
      pendingAPIRequests: observation?.pendingAPIRequests ?? null, observedReadsSettled: Boolean(observation && observation.pendingAPIRequests === 0),
      loadingVisible: Boolean(portfolio && /Loading vendor relationships|Loading form summaries/.test(portfolio.textContent)),
      sameDocument: priorTimeOrigin === undefined ? null : performance.timeOrigin === priorTimeOrigin };
  }, { view, action, priorTimeOrigin });
  if (state.heading !== "Vendors" || !state.portfolioVisible || !state.collectionVisible || state.selectedRelationshipVisible
    || state.selectedTab !== labelFor(view) || !state.selectedPanelVisible || !state.recordCount || state.loadingVisible
    || requireObservedReads && !state.observedReadsSettled || priorTimeOrigin !== undefined && !state.sameDocument
    || expectedRecordTitles && JSON.stringify(state.recordTitles) !== JSON.stringify(expectedRecordTitles)) {
    throw new Error(`Vendor collection route did not restore its settled rendered state: ${JSON.stringify(state)}`);
  }
  return { ...state, complete: true };
}

export async function assertVendorPortfolioLayout(page, scenario) {
  const reading = await page.evaluate((view) => {
    const root = document.querySelector(".vendor-portfolio-workspace");
    const visible = (node) => {
      if (!node?.getClientRects().length || getComputedStyle(node).display === "none" || getComputedStyle(node).visibility === "hidden" || node.closest('[hidden],[inert],[aria-hidden="true"]')) return false;
      for (let parent = node.parentElement; parent; parent = parent.parentElement) if (parent.tagName === "DETAILS" && !parent.open && !parent.querySelector(":scope > summary")?.contains(node)) return false;
      return true;
    };
    const rect = (node) => { if (!visible(node)) return null; const b = node.getBoundingClientRect(); return { x: b.x, y: b.y, width: b.width, height: b.height, right: b.right, bottom: b.bottom }; };
    const header = root.querySelector(".vendor-portfolio-heading");
    const nav = root.querySelector(".vendor-portfolio-navigation > .cs-tabs");
    const list = nav?.querySelector(':scope > [role="tablist"]');
    const tabs = [...(list?.querySelectorAll('[role="tab"]') ?? [])];
    const selected = tabs.find((node) => node.getAttribute("aria-selected") === "true");
    const selectedPanelID = selected?.getAttribute("aria-controls");
    const selectedPanel = selectedPanelID ? document.getElementById(selectedPanelID) : null;
    const compact = nav?.querySelector(":scope > .cs-tabs__compact .cs-select-field__trigger");
    const menuNode = visible(list) ? list : nav?.querySelector(":scope > .cs-tabs__compact");
    const first = (selector) => [...root.querySelectorAll(selector)].find(visible);
    const main = first(".vendor-portfolio-main"); const rail = first(".vendor-portfolio-rail");
    const mainBounds = rect(main); const railBounds = rect(rail);
    const cards = [...root.querySelectorAll(".vendor-summary-counts > .cs-button")].filter(visible);
    const insights = [...root.querySelectorAll(".vendor-overview-insights > section")].filter(visible);
    const register = first("#vendor-portfolio-register");
    const recordOwner = register ?? first(".vendor-collection");
    const records = [...(recordOwner?.querySelectorAll(".cs-selectable-record") ?? [])].filter(visible);
    const firstRecord = rect(records[0]);
    const firstUseful = rect(view === "overview" ? first(".vendor-summary-counts") : records[0]);
    const navBounds = rect(menuNode); const selectedBounds = rect(visible(selected) ? selected : compact);
    const controls = [...new Set([...root.querySelectorAll('button,input:not([type="hidden"]),select,textarea,summary,[role="tab"]')]
      .map((node) => node.closest(".cs-checkbox-field") ?? node))].filter(visible).map((node) => ({ text: node.getAttribute("aria-label") ?? node.textContent?.trim().slice(0, 80), ...rect(node) }));
    const heading = rect(header); const title = rect(header?.querySelector("h1"));
    const visiblePanels = [...(nav?.querySelectorAll(':scope > [role="tabpanel"]') ?? [])].filter(visible);
    return { kind: view, route: location.hash, heading, title, headingText: header?.querySelector("h1")?.textContent.trim(),
      localTabs: tabs.map((node) => node.textContent.trim()), selectedTab: selected?.textContent.trim(),
      selectedPanelID, visiblePanelIDs: visiblePanels.map((node) => node.id), selectedPanelControlsMatch: visiblePanels.length === 1 && visiblePanels[0] === selectedPanel,
      menuMode: visible(list) ? "tabs" : "select", compactValue: compact?.querySelector(".cs-select-field__value")?.textContent.trim(),
      tablist: navBounds, selectedTabBounds: selectedBounds,
      selectedTabVisible: Boolean(navBounds && selectedBounds && selectedBounds.x >= navBounds.x - 1 && selectedBounds.right <= navBounds.right + 1),
      titleContained: Boolean(title && heading && title.x >= heading.x - 1 && title.right <= heading.right + 1),
      main: mainBounds, aside: railBounds, placement: mainBounds && railBounds ? railBounds.x >= mainBounds.right - 1 ? "beside" : "stacked" : null,
      metrics: cards.map((node) => ({ label: node.getAttribute("aria-label"), bounds: rect(node), value: node.querySelector("strong")?.textContent.trim() })),
      insights: insights.map((node) => ({ heading: node.querySelector("h2")?.textContent.trim(), bounds: rect(node) })),
      railHeadings: rail ? [...rail.querySelectorAll("h2")].map((node) => node.textContent.trim()) : [],
      register: rect(register), registerRows: records.length, firstRecord, recordHeights: records.map((node) => rect(node).height),
      recordIDs: records.map((node) => {
        const title = node.querySelector(".cs-selectable-record__title")?.textContent.trim();
        const service = node.querySelector(".cs-selectable-record__metadata")?.textContent.trim();
        return window.vendorPortfolioEvidence?.fixtureRecords.find((record) => record.title === title && record.service === service)?.id ?? null;
      }),
      firstRecordVisible: Boolean(firstRecord && firstRecord.y >= 0 && Math.min(firstRecord.bottom, innerHeight) - firstRecord.y >= 44),
      firstUsefulContent: firstUseful, firstUsefulContentVisible: Boolean(firstUseful && firstUseful.y >= 0 && Math.min(firstUseful.bottom, innerHeight) - firstUseful.y >= 44),
      selectedRelationshipVisible: Boolean(first(".vendor-relationship-workspace")),
      documentWidth: document.documentElement.clientWidth, scrollWidth: document.documentElement.scrollWidth,
      minTargetWidth: controls.length ? Math.min(...controls.map((item) => item.width)) : 0,
      minTargetHeight: controls.length ? Math.min(...controls.map((item) => item.height)) : 0,
      undersizedTargets: controls.filter((item) => item.width < 44 || item.height < 44),
      apiReads: window.vendorPortfolioEvidence?.reads ?? [], commands: window.vendorPortfolioEvidence?.commands ?? [],
      evidenceKind: "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION" };
  }, scenario.vendorPortfolioView);
  if (reading.headingText !== "Vendors" || !reading.titleContained || reading.localTabs.join("|") !== labels.join("|")
    || reading.selectedTab !== labelFor(scenario.vendorPortfolioView) || !reading.selectedPanelControlsMatch || !reading.selectedTabVisible
    || reading.menuMode === "select" && reading.compactValue !== reading.selectedTab
    || reading.heading.bottom > reading.tablist.y + 1 || reading.route !== `#vendors/${scenario.vendorPortfolioView}`
    || reading.scrollWidth > scenario.viewport.width + 1 || reading.selectedRelationshipVisible) {
    throw new Error(`${scenario.name}: portfolio heading, full menu, selected panel or overflow failed: ${JSON.stringify(reading)}`);
  }
  if (scenario.vendorPortfolioMatrix && !reading.firstUsefulContentVisible) throw new Error(`${scenario.name}: first useful portfolio content is outside the initial viewport: ${JSON.stringify(reading.firstUsefulContent)}`);
  if (scenario.touch && reading.undersizedTargets.length) throw new Error(`${scenario.name}: portfolio controls are smaller than 44px: ${JSON.stringify(reading.undersizedTargets)}`);
  if (reading.recordIDs.some((id) => !id) || new Set(reading.recordIDs).size !== reading.registerRows) throw new Error(`${scenario.name}: visible relationship records do not bind to the observed list population`);
  if (scenario.vendorPortfolioView === "overview" && !["loading", "unavailable", "denied"].includes(scenario.vendorPortfolioCheck)) {
    const expected = scenario.viewport.width > 1120 ? "beside" : "stacked";
    if (reading.metrics.length !== 4 || reading.insights.map((item) => item.heading).join("|") !== "Service criticality|Exception stages"
      || reading.railHeadings.join("|") !== "Needs attention|Service renewals" || !reading.register || !reading.main || !reading.aside
      || reading.placement !== expected || expected === "beside" && reading.main.right > reading.aside.x + 1
      || expected === "beside" && Math.abs(reading.aside.width - (scenario.viewport.width > 1420 ? 313 : 295)) > 2
      || expected === "stacked" && (reading.main.bottom > reading.aside.y + 1 || Math.abs(reading.main.width - reading.aside.width) > 2)
      || reading.register.y < Math.max(...reading.insights.map((item) => item.bounds.bottom)) - 1) {
      throw new Error(`${scenario.name}: overview requires four measures, two insights and inline register with its bounded attention/renewals rail: ${JSON.stringify(reading)}`);
    }
  }
  return reading;
}

export async function verifyVendorPortfolioScenario(page, scenario) {
  const check = scenario.vendorPortfolioCheck;
  const result = {};
  if (scenario.verifyVendorPortfolioNavigation) {
    result.visited = [];
    for (const view of vendorPortfolioViews) {
      await selectVendorPortfolioView(page, view.id); await waitForPortfolio(page, { ...scenario, vendorPortfolioView: view.id });
      result.visited.push(view.label);
    }
    await selectVendorPortfolioView(page, "overview");
    const keyboard = { options: labels, sequence: [], complete: false };
    if (scenario.viewport.width > 760) {
      await menu(page).getByRole("tab", { name: "Overview", exact: true }).focus();
      for (const label of [...labels.slice(1), labels[0]]) {
        await page.keyboard.press("ArrowRight");
        await menu(page).getByRole("tab", { name: label, exact: true }).waitFor();
        const selected = await menu(page).getByRole("tab", { name: label, exact: true }).getAttribute("aria-selected");
        if (selected !== "true") throw new Error(`${scenario.name}: keyboard did not activate ${label}`);
        keyboard.sequence.push(label);
      }
    } else {
      const trigger = root(page).locator('.vendor-portfolio-navigation > .cs-tabs > .cs-tabs__compact .cs-select-field__trigger');
      await trigger.focus(); await page.keyboard.press("Space");
      await page.getByRole("option", { name: "Overview", exact: true }).waitFor();
      const options = (await page.locator(".cs-select-field__listbox").getByRole("option").allTextContents()).map((label) => label.trim());
      if (options.join("|") !== labels.join("|")) throw new Error(`${scenario.name}: compact menu lost a Vendor destination: ${JSON.stringify(options)}`);
      await page.keyboard.press("ArrowDown"); await page.keyboard.press("Enter");
      await page.waitForFunction(() => location.hash === "#vendors/register"); keyboard.sequence.push("Register");
      await selectVendorPortfolioView(page, "overview");
    }
    keyboard.complete = true; result.keyboard = keyboard;
    await waitForPortfolio(page, scenario); await resetScroll(page); result.baselineRestored = true;
    if (scenario.viewport.width === 1440) result.exceptionReturn = await verifyVendorPortfolioExceptionReturn(page);
    else {
      await root(page).getByRole("button", { name: "5 open exceptions", exact: true }).click();
      const queue = root(page).locator(".vendor-portfolio-exceptions[open]");
      await queue.waitFor();
      const opened = await assertVendorPortfolioLayout(page, { ...scenario, vendorPortfolioMatrix: false });
      await queue.locator(".vendor-mobile-status-filter select").selectOption("ATTENTION");
      await queue.locator(":scope > summary").click();
      await resetScroll(page);
      result.queueTargets = { minimumWidth: opened.minTargetWidth, minimumHeight: opened.minTargetHeight, undersized: opened.undersizedTargets, baselineRestored: true };
    }
  }
  if (scenario.verifyVendorPortfolioFilters) {
    const register = root(page).locator(".vendor-register");
    const toggle = register.locator(".vendor-register-filter-toggle button");
    const filters = register.locator("#vendor-register-filters");
    const baselineRecords = await register.locator(".cs-selectable-record").count();
    if (await toggle.getAttribute("aria-expanded") !== "false" || await filters.isVisible()) throw new Error(`${scenario.name}: phone filters must begin collapsed`);
    await toggle.click();
    await filters.getByRole("button", { name: /Service criticality in loaded relationships/ }).click();
    await page.locator(".cs-select-field__listbox").getByRole("option", { name: "Critical", exact: true }).click();
    await register.getByText(/0 of 2 loaded relationships match the active filters/).waitFor();
    const filteredRecords = await register.locator(".cs-selectable-record").count();
    await toggle.click();
    const appliedCountRetained = await toggle.textContent() === "Filters · 1 applied" && !await filters.isVisible();
    await toggle.click(); await filters.getByRole("button", { name: "Clear filters", exact: true }).click();
    await toggle.click();
    const restored = await register.locator(".cs-selectable-record").count() === baselineRecords && await toggle.textContent() === "Filters" && !await filters.isVisible();
    if (!appliedCountRetained || !restored || filteredRecords !== 0) throw new Error(`${scenario.name}: filters did not retain or clear the loaded-record selection`);
    result.filters = { initiallyCollapsed: true, expanded: true, selection: "Critical", filteredRecords, appliedCountRetained, cleared: true, baselineRestored: restored };
  }
  if (check === "loading") result.pending = (await page.evaluate(() => window.vendorPortfolioEvidence.reads)).filter((item) => item.status === "PENDING").length;
  if (check === "unavailable") {
    const before = await page.evaluate(() => window.vendorPortfolioEvidence.reads.length);
    await root(page).getByRole("button", { name: "Try again", exact: true }).click();
    await root(page).getByRole("heading", { name: "Vendor records are unavailable", exact: true }).waitFor();
    result.retryIssued = await page.evaluate((before) => window.vendorPortfolioEvidence.reads.length > before, before);
  }
  if (check === "empty") result.empty = await root(page).getByText(/No vendor relationships found/).isVisible();
  if (check === "partial") result.loadMoreAvailable = await root(page).getByRole("button", { name: "Load more vendors", exact: true }).isVisible();
  if (["summaries-unavailable", "summary-missing"].includes(check)) {
    result.unavailable = await root(page).getByText(/Some (vendor )?form summaries (could not be loaded|are unavailable)/).isVisible();
    result.unknownVisible = await root(page).locator(".vendor-collection-list").getByText("Form counts unavailable", { exact: true }).first().isVisible();
  }
  if (check === "long-labels") result.longLabels = await root(page).getByText("Sample · International Settlement and Merchant Assurance Services Limited", { exact: true }).isVisible();
  if (check === "denied") result.addDisabled = await root(page).getByRole("button", { name: "Add vendor", exact: true }).isDisabled();
  if (check === "exceptions-unavailable") result.unknownTotals = await root(page).getByRole("button", { name: "Unknown open exceptions", exact: true }).isVisible();
  if (check === "search-paging") {
    await root(page).getByRole("searchbox", { name: "Search vendors and services", exact: true }).fill("Acme");
    await root(page).getByRole("button", { name: "Search vendors", exact: true }).click();
    await root(page).locator(".cs-selectable-record").first().waitFor();
    await root(page).getByRole("button", { name: "Load more vendors", exact: true }).click();
    await root(page).getByText("Sample · Acme Settlement Services", { exact: true }).waitFor();
    result.search = "Acme"; result.paged = true;
  }
  if (["documents-drill", "onboarding-drill", "assessments-drill"].includes(check)) {
    const collection = root(page).getByRole("region", { name: regionFor(scenario.vendorPortfolioView), exact: true });
    result.initialCollection = await assertVendorPortfolioRouteState(page, scenario.vendorPortfolioView, { action: "INITIAL_COLLECTION" });
    const priorTimeOrigin = await page.evaluate(() => performance.timeOrigin);
    await collection.locator(".cs-selectable-record").first().click();
    await page.locator(".vendor-relationship-workspace").waitFor();
    result.relationshipPreserved = new URL(page.url()).hash === `#vendors/register/${relationshipID}`;
    if (!result.relationshipPreserved) throw new Error(`${scenario.name}: collection opened the wrong relationship`);
    if (check === "documents-drill") {
      await page.locator(".document-browser").waitFor();
      await page.waitForFunction(() => window.vendorPortfolioEvidence.reads.some((item) => item.path === "/api/v1/forms/documents" && item.status === 200));
      result.protectedDocuments = true;
    } else if (check === "onboarding-drill") {
      await page.locator(".vdd-workspace").waitFor(); result.dueDiligence = true;
    } else { await page.locator(".vendor-forms-panel").waitFor(); result.forms = true; }
    await waitForRouteReadSettlement(page, `#vendors/register/${relationshipID}`);
    // Exercise the original mounted App before any explicit Back action,
    // root navigation or reload can clear its stale selected aggregate.
    await page.goBack();
    result.browserBack = await assertVendorPortfolioRouteState(page, scenario.vendorPortfolioView, {
      action: "HISTORY_BACK", priorTimeOrigin, expectedRecordTitles: result.initialCollection.recordTitles,
    });
    result.returned = result.browserBack.complete;
    await page.goForward();
    await page.locator(".vendor-relationship-workspace").getByRole("heading", { name: "Acme Processing Limited", exact: true }).waitFor();
    await waitForRouteReadSettlement(page, `#vendors/register/${relationshipID}`);
    result.browserForward = await page.evaluate((priorTimeOrigin) => ({ action: "HISTORY_FORWARD", route: location.hash,
      selectedRelationshipVisible: Boolean(document.querySelector(".vendor-relationship-workspace")), portfolioVisible: Boolean(document.querySelector(".vendor-portfolio-workspace")),
      relationshipTitle: document.querySelector(".vendor-relationship-workspace h1")?.textContent.trim(), sameDocument: performance.timeOrigin === priorTimeOrigin,
      observedReadsSettled: window.vendorPortfolioEvidence.pendingAPIRequests === 0 }), priorTimeOrigin);
    if (!result.browserForward.selectedRelationshipVisible || result.browserForward.portfolioVisible || !result.browserForward.sameDocument
      || result.browserForward.route !== `#vendors/register/${relationshipID}` || result.browserForward.relationshipTitle !== "Acme Processing Limited" || !result.browserForward.observedReadsSettled) throw new Error(`${scenario.name}: browser Forward did not restore the selected relationship`);
    result.refresh = await verifyCollectionRefresh(page, check, scenario, result.initialCollection.recordTitles);
    await page.getByRole("button", { name: "Back to vendor register", exact: true }).click();
    result.explicitRegisterReturn = await assertVendorPortfolioRouteState(page, "register", {
      action: "IN_APP_REGISTER_RETURN", priorTimeOrigin, expectedRecordTitles: result.initialCollection.recordTitles,
    });
    await selectVendorPortfolioView(page, scenario.vendorPortfolioView);
    await waitForPortfolio(page, scenario);
  }
  await resetScroll(page);
  return Object.keys(result).length ? result : null;
}

export async function verifyVendorPortfolioExceptionReturn(page) {
  await root(page).getByRole("button", { name: "5 open exceptions", exact: true }).click();
  const queue = root(page).locator(".vendor-portfolio-exceptions[open]");
  await queue.waitFor();
  const actionName = "Review exception: Recovery exercise evidence missing";
  const action = queue.getByRole("button", { name: actionName, exact: true });
  await action.scrollIntoViewIfNeeded();
  const before = await page.evaluate(() => window.scrollY);
  await action.click();
  await page.waitForFunction(() => location.hash === "#work/matters/sample-portfolio-finding-0");
  await page.getByRole("heading", { name: "Recovery exercise evidence missing", exact: true, level: 1 }).waitFor();
  await page.goBack();
  await root(page).getByRole("button", { name: "5 open exceptions", exact: true }).waitFor();
  await queue.getByRole("button", { name: actionName, exact: true }).waitFor();
  await page.waitForFunction(({ before, actionName }) => {
    const queue = document.querySelector(".vendor-portfolio-exceptions[open]");
    const action = [...(queue?.querySelectorAll("button") ?? [])].find((button) => button.getAttribute("aria-label") === actionName);
    const box = action?.getBoundingClientRect();
    return queue && box && box.top >= 0 && box.bottom <= innerHeight && Math.abs(scrollY - before) <= 4;
  }, { before, actionName }, { timeout: 3000 });
  const state = await queue.evaluate((node, actionName) => {
    const action = [...node.querySelectorAll("button")].find((button) => button.getAttribute("aria-label") === actionName);
    const box = action.getBoundingClientRect();
    return { open: node.open, scrollY: window.scrollY, rowVisible: box.top >= 0 && box.bottom <= innerHeight, rowTop: box.top, rowBottom: box.bottom };
  }, actionName);
  if (!state.open || !state.rowVisible || Math.abs(state.scrollY - before) > 4) throw new Error(`Vendor exception return lost the settled queue scroll position: ${JSON.stringify({ before, ...state })}`);
  await queue.getByRole("button", { name: "Needs attention", exact: true }).click();
  await queue.locator(":scope > summary").click();
  await resetScroll(page);
  return { ...state, priorScrollY: before, exactMatterID: "sample-portfolio-finding-0", baselineRestored: true };
}

async function verifyCollectionRefresh(page, check, scenario, expectedRecordTitles) {
  // The existing selected-record URL owns identity, while its local section
  // starts at Overview on refresh. Verify that exact contract independently
  // of the collection page and reopen the intended destination with its menu.
  const refreshed = await page.context().newPage();
  const errors = [];
  refreshed.on("pageerror", (error) => errors.push(error.message));
  refreshed.on("console", (message) => { if (message.type() === "error") errors.push(message.text()); });
  try {
    await refreshed.goto(page.url(), { waitUntil: "networkidle" });
    await refreshed.locator(".vendor-relationship-workspace").getByRole("heading", { name: "Acme Processing Limited", exact: true }).waitFor();
    await refreshed.reload({ waitUntil: "networkidle" });
    const workspace = refreshed.locator(".vendor-relationship-workspace");
    await workspace.getByRole("heading", { name: "Acme Processing Limited", exact: true }).waitFor();
    const navigation = workspace.locator(".vendor-relationship-navigation > .cs-tabs");
    const initialSection = await navigation.locator(':scope > [role="tablist"] [aria-selected="true"]').textContent();
    if (new URL(refreshed.url()).hash !== `#vendors/register/${relationshipID}` || initialSection !== "Overview") throw new Error("Refreshed collection drill lost its relationship identity or Overview route contract");
    const choose = async (tabs, label) => {
      const tab = tabs.locator(':scope > [role="tablist"]').getByRole("tab", { name: label, exact: true });
      if (await tab.isVisible()) await tab.click();
      else {
        await tabs.locator(":scope > .cs-tabs__compact .cs-select-field__trigger").click();
        await refreshed.locator(".cs-select-field__listbox").getByRole("option", { name: label, exact: true }).click();
      }
    };
    const section = check === "onboarding-drill" ? "Due diligence" : "Evidence";
    await choose(navigation, section);
    let evidenceView = null;
    if (check === "onboarding-drill") await workspace.locator(".vdd-workspace:visible").waitFor();
    else {
      evidenceView = check === "documents-drill" ? "Documents" : "Forms";
      await choose(workspace.locator(".vendor-relationship-evidence:visible > .cs-tabs"), evidenceView);
      await workspace.locator(evidenceView === "Documents" ? ".document-browser:visible" : ".vendor-forms-panel:visible").waitFor();
    }
    const relationshipPreserved = new URL(refreshed.url()).hash === `#vendors/register/${relationshipID}`;
    await installVendorPortfolioObservation(refreshed);
    const priorTimeOrigin = await refreshed.evaluate(() => performance.timeOrigin);
    await refreshed.evaluate((view) => { location.hash = `#vendors/${view}`; }, scenario.vendorPortfolioView);
    const directHashRoot = await assertVendorPortfolioRouteState(refreshed, scenario.vendorPortfolioView, {
      action: "DIRECT_HASH_ROOT", priorTimeOrigin, expectedRecordTitles,
    });
    await refreshed.reload({ waitUntil: "networkidle" });
    await waitForPortfolio(refreshed, scenario);
    const reloadedCollection = await assertVendorPortfolioRouteState(refreshed, scenario.vendorPortfolioView, {
      action: "RELOAD_COLLECTION", expectedRecordTitles, requireObservedReads: false,
    });
    reloadedCollection.newDocument = await refreshed.evaluate((priorTimeOrigin) => performance.timeOrigin !== priorTimeOrigin, priorTimeOrigin);
    if (!reloadedCollection.newDocument) throw new Error("Collection reload did not create a new document");
    if (errors.length) throw new Error(`Collection route refresh emitted browser errors: ${errors.join("\n")}`);
    return { relationshipPreserved, initialSection, reopenedSection: section, evidenceView, directHashRoot, reloadedCollection, complete: true };
  } finally { await refreshed.close(); }
}

export async function measureVendorPortfolioSourceLayout(page, view) {
  return page.evaluate((view) => {
    const visible = (node) => Boolean(node?.getClientRects().length && getComputedStyle(node).display !== "none" && !node.closest("[hidden]"));
    const rect = (node) => { if (!visible(node)) return null; const b = node.getBoundingClientRect(); return { x: b.x, y: b.y, width: b.width, height: b.height, right: b.right, bottom: b.bottom }; };
    const panel = document.querySelector(`#panel-${view}`);
    const tabs = [...document.querySelectorAll('.vendor-tabs [role="tab"]')];
    const selected = tabs.find((node) => node.getAttribute("aria-selected") === "true");
    const main = rect(panel?.querySelector(".vendor-primary")); const aside = rect(panel?.querySelector(".vendor-context"));
    const metric = panel?.querySelector(".vendor-metrics");
    const register = panel?.querySelector(".vendor-register-card");
    const firstRecord = panel?.querySelector(".vendor-table tbody tr,.alt-row");
    const first = rect(metric ?? firstRecord);
    return { kind: view, heading: rect(document.querySelector(".vendor-heading")), tablist: rect(document.querySelector(".vendor-tabs")),
      localTabs: tabs.map((node) => node.childNodes[0].textContent.trim()), selectedTab: selected?.childNodes[0].textContent.trim(),
      main, aside, placement: main && aside ? aside.x >= main.right - 1 ? "beside" : "stacked" : null,
      selectedPanel: rect(panel), metrics: metric ? [...metric.children].map(rect) : [], register: rect(register), firstRecord: rect(firstRecord),
      firstUsefulContent: first, firstUsefulContentVisible: Boolean(first && first.y >= 0 && Math.min(first.bottom, innerHeight) - first.y >= 44),
      documentWidth: document.documentElement.clientWidth, scrollWidth: document.documentElement.scrollWidth };
  }, view);
}
