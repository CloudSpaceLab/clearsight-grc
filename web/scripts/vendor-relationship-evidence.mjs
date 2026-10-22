import { vendorRelationshipViews } from "./vendor-relationship-evidence-scenarios.mjs";

const labels = vendorRelationshipViews.map((view) => view.label);
const fixtureRelationshipID = "vendor-relationship-payments";
const viewLabel = (view) => vendorRelationshipViews.find((item) => item.id === view)?.label;
const workspace = (page) => page.locator(".vendor-relationship-workspace");
const navigation = (page) => page.locator(".vendor-relationship-navigation > .cs-tabs");
const resetScroll = (page) => page.evaluate(() => { window.scrollTo(0, 0); });

export async function selectVendorRelationshipView(page, view) {
  const label = viewLabel(view);
  const tab = navigation(page).getByRole("tablist", { name: "Vendor sections", exact: true }).getByRole("tab", { name: label, exact: true });
  if (await tab.isVisible()) await tab.click();
  else {
    const trigger = navigation(page).locator(":scope > .cs-tabs__compact .cs-select-field__trigger");
    await trigger.click();
    await page.getByRole("option", { name: label, exact: true }).click();
  }
}

export async function prepareVendorRelationship(page, scenario) {
  await page.locator(".vendor-row").first().waitFor({ state: "visible" });
  await installFixtureObservation(page, scenario.vendorRelationshipCheck);
  if (["loading", "unavailable", "long-labels", "assessment-unavailable", "empty", "partial"].includes(scenario.vendorRelationshipCheck)) {
    // Remount the same real workspace so its existing readers observe the
    // intentionally changed fixture response, rather than editing React state.
    await page.getByRole("complementary", { name: "Primary navigation", exact: true }).getByRole("link", { name: "Home", exact: true }).click();
    await page.getByRole("heading", { name: "Oversight", exact: true }).first().waitFor();
    await page.getByRole("complementary", { name: "Primary navigation", exact: true }).getByRole("link", { name: "Vendors", exact: true }).click();
    const register = page.getByRole("navigation", { name: "Vendor sections", exact: true }).getByRole("button", { name: "Register", exact: true });
    if (await register.isVisible()) await register.click();
    await page.locator(".vendor-row").first().waitFor({ state: "visible" });
  }
  await page.locator(".vendor-row").first().evaluate((node, relationshipID) => {
    node.addEventListener("pointerdown", () => {
      const log = window.vendorRelationshipEvidence;
      log.selectionBoundary = { event: "RELATIONSHIP_SELECTION_POINTERDOWN", relationshipID, at: performance.now(), observedAt: new Date().toISOString() };
      log.phase = "SELECTED_RELATIONSHIP";
    }, { once: true, capture: true });
  }, fixtureRelationshipID);
  await page.locator(".vendor-row").first().click();
  await workspace(page).getByRole("heading", { level: 1 }).waitFor({ state: "visible" });
  if (new URL(page.url()).hash !== `#vendors/register/${fixtureRelationshipID}`) throw new Error(`${scenario.name}: selected relationship identity differs from the requested fixture`);
  await selectVendorRelationshipView(page, scenario.vendorRelationshipView);
  await waitForVendorView(page, scenario);
  await page.evaluate(() => document.fonts?.ready);
  await resetScroll(page);
}

async function waitForVendorView(page, scenario) {
  const primary = workspace(page).locator(".vendor-relationship-primary:visible");
  const check = scenario.vendorRelationshipCheck;
  if (check === "loading") await primary.getByText(/Loading/).first().waitFor();
  else if (check === "unavailable") await primary.getByText("Compliance status unavailable", { exact: true }).waitFor();
  else if (check === "empty") await primary.getByText("No forms requested", { exact: true }).waitFor();
  else if (check === "partial") {
    await primary.getByText("Compliance status unavailable", { exact: true }).waitFor();
    await primary.getByText("1 forms shown", { exact: true }).waitFor();
  }
  else if (scenario.vendorRelationshipView === "overview") await primary.getByText("Superseded", { exact: true }).waitFor();
  else if (scenario.vendorRelationshipView === "due-diligence") {
    await primary.locator(".vdd-workspace").waitFor();
    if (check === "assessment-unavailable") await primary.getByText("Due diligence is unavailable", { exact: true }).waitFor();
    else {
      await primary.locator(".vendor-activation-panel").waitFor();
      await primary.locator('.vdd-workspace [aria-busy="true"]').waitFor({ state: "hidden" });
      await primary.getByText("Checking activation requirements…", { exact: true }).waitFor({ state: "hidden" });
      await primary.getByText("Checking expired items…", { exact: true }).waitFor({ state: "hidden" });
    }
  } else if (scenario.vendorRelationshipView === "evidence") {
    await primary.locator(".vendor-forms-panel").waitFor();
    await primary.locator(".vendor-request-details > summary").filter({ hasText: /\d+ on this page/ }).waitFor();
    await primary.getByText("Loading forms and responses for this vendor service…", { exact: true }).waitFor({ state: "hidden" });
  } else if (scenario.vendorRelationshipView === "issues") {
    await primary.getByRole("button", { name: "5 open exceptions", exact: true }).waitFor();
    await primary.locator(".vendor-exception-row").first().waitFor();
  } else if (scenario.vendorRelationshipView === "activity") {
    await primary.getByRole("heading", { name: "Response history", exact: true }).waitFor();
    await primary.getByText(/^\d+ submitted revisions loaded$/).waitFor();
    await primary.getByText("Loading submitted response revisions…", { exact: true }).waitFor({ state: "hidden" });
  }
}

async function installFixtureObservation(page, check) {
  await page.evaluate((check) => {
    const original = window.fetch;
    const log = { reads: [], preparationReads: [], commands: [], check, phase: "PREPARING" };
    window.vendorRelationshipEvidence = log;
    const json = (body, status = 200) => new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
    window.fetch = async (input, init) => {
      const url = new URL(typeof input === "string" ? input : input.url, location.href);
      const method = String(init?.method ?? (typeof input === "object" ? input.method : null) ?? "GET").toUpperCase();
      const relevant = url.pathname.startsWith("/api/v1/vendors") || url.pathname.startsWith("/api/v1/vendor-") || url.pathname.startsWith("/api/v1/forms/documents") || url.pathname === "/api/v1/forms/responses";
      const entry = { path: url.pathname, query: url.search, method, phase: log.phase, initiatedAt: performance.now() };
      const reads = log.phase === "SELECTED_RELATIONSHIP" ? log.reads : log.preparationReads;
      // Authority.Resolve is a read query with a JSON body, despite its POST
      // transport. Keep it observed and scoped; every other API write remains
      // a command candidate (see src/api.ts and httpapi/handlers.go).
      const authorityQuery = method === "POST" && url.pathname === "/api/v1/authority/resolve";
      if (authorityQuery) {
        const body = JSON.parse(init?.body ?? "{}");
        entry.queryInput = { object_type: body.object_type, object_id: body.object_id, responsibility: body.responsibility, decision_type: body.decision_type };
        entry.requestKind = "AUTHORITY_READ_QUERY";
        reads.push(entry);
      } else if (method === "GET" && relevant) reads.push(entry);
      else if (!["GET", "HEAD", "OPTIONS"].includes(method) && url.pathname.startsWith("/api/v1/")) log.commands.push(entry);
      const summaryRead = url.pathname === "/api/v1/vendors/form-summaries";
      const relationshipFormsRead = /^\/api\/v1\/vendors\/[^/]+\/forms$/.test(url.pathname);
      const formRead = summaryRead || relationshipFormsRead;
      const assessmentRead = /^\/api\/v1\/vendors\/[^/]+\/assessments\/current$/.test(url.pathname);
      if (method === "GET" && check === "loading" && (formRead || assessmentRead)) { entry.status = "PENDING"; return new Promise(() => {}); }
      if (method === "GET" && (check === "unavailable" && formRead || check === "partial" && summaryRead || check === "assessment-unavailable" && assessmentRead)) {
        entry.status = 503; return json({ code: "sample_unavailable", message: "Sample vendor review information is unavailable." }, 503);
      }
      if (method === "GET" && check === "activation-denied" && /^\/api\/v1\/vendors\/[^/]+\/activation$/.test(url.pathname)) {
        entry.status = 403; return json({ code: "permission_denied", message: "Sample actor cannot read activation authorization." }, 403);
      }
      const result = await original(input, init);
      entry.status = result.status;
      if (method === "GET" && ["empty", "partial"].includes(check) && formRead && result.ok) {
        const body = await result.clone().json();
        if (summaryRead && check === "empty") {
          const summary = body.items.find((item) => item.relationship_id === "vendor-relationship-payments");
          if (!summary) throw new Error("Empty Vendor fixture requires the selected relationship summary");
          for (const field of ["outstanding_forms", "overdue_forms", "submitted_forms", "awaiting_review", "unassessed_forms", "assessed_forms", "partially_replaced_forms", "outdated_forms", "freshness_unknown_forms"]) summary[field] = 0;
          delete summary.highest_concern;
          entry.responseSummary = { relationship_id: summary.relationship_id, outstanding_forms: summary.outstanding_forms,
            submitted_forms: summary.submitted_forms, awaiting_review: summary.awaiting_review, assessed_forms: summary.assessed_forms };
        } else if (relationshipFormsRead) {
          if (check === "empty") { body.items = []; delete body.next_cursor; }
          else {
            if (body.items.length < 2) throw new Error("Partial Vendor fixture requires at least two existing form rows");
            body.items = body.items.slice(0, 1);
            body.next_cursor = "sample-vendor-forms-next";
          }
          entry.responseItems = body.items.length;
          entry.nextCursor = Boolean(body.next_cursor);
        }
        entry.fixtureState = check;
        return json(body);
      }
      if (method === "GET" && check === "long-labels" && (url.pathname === "/api/v1/vendors" || /^\/api\/v1\/vendors\/[^/]+$/.test(url.pathname) && url.pathname !== "/api/v1/vendors/form-summaries")) {
        const body = await result.clone().json();
        const record = body.items?.[0] ?? body;
        record.vendor.legal_name = "Sample · International Settlement and Merchant Assurance Services Limited";
        record.relationship.service_name = "Cross-border card transaction processing, delegated settlement and assurance reporting";
        record.vendor.jurisdiction = "Federal Republic of Nigeria";
        return json(body);
      }
      return result;
    };
  }, check);
}

export async function assertVendorRelationshipLayout(page, scenario) {
  const reading = await page.evaluate((view) => {
    const root = document.querySelector(".vendor-relationship-workspace");
    const visible = (node) => Boolean(node?.getClientRects().length && getComputedStyle(node).display !== "none" && getComputedStyle(node).visibility !== "hidden");
    const rect = (node) => { if (!visible(node)) return null; const b = node.getBoundingClientRect(); return { x: b.x, y: b.y, width: b.width, height: b.height, right: b.right, bottom: b.bottom }; };
    const header = root.querySelector(".vendor-relationship-header");
    const nav = root.querySelector(".vendor-relationship-navigation > .cs-tabs");
    const tabs = [...nav.querySelector(':scope > [role="tablist"]').querySelectorAll('[role="tab"]')];
    const selected = tabs.find((node) => node.getAttribute("aria-selected") === "true");
    const selectedPanelID = selected.getAttribute("aria-controls");
    const selectedPanel = document.getElementById(selectedPanelID);
    const outerPanels = [...nav.querySelectorAll(':scope > [role="tabpanel"]')];
    const visiblePanels = outerPanels.filter(visible);
    const firstVisible = (selector) => [...(selectedPanel?.querySelectorAll(selector) ?? [])].find(visible);
    const compact = nav.querySelector(":scope > .cs-tabs__compact .cs-select-field__trigger");
    const tablist = visible(tabs[0]) ? nav.querySelector(':scope > [role="tablist"]') : nav.querySelector(":scope > .cs-tabs__compact");
    const selectedBounds = rect(visible(selected) ? selected : compact);
    const menuBounds = rect(tablist);
    const main = firstVisible(".vendor-relationship-primary");
    const aside = firstVisible(".vendor-relationship-context");
    const mainBounds = rect(main); const asideBounds = rect(aside);
    const content = view === "overview" ? firstVisible(".vendor-relationship-status")
      : view === "evidence" ? firstVisible(".document-browser-heading, .vendor-forms-panel > header")
        : view === "issues" ? firstVisible(".vendor-overview-summary")
          : view === "activity" ? firstVisible('[aria-label="Submitted response history"] li, [aria-label="Submitted response history"] .cs-empty-state')
            : firstVisible(".vdd-workspace");
    const contentBounds = rect(content);
    const targets = [...new Set([...root.querySelectorAll('button, [role="tab"], input:not([type="hidden"]), select, textarea, summary')]
      .map((node) => node.closest(".cs-checkbox-field") ?? node))].filter((node) => visible(node) && !node.closest('[inert], [aria-hidden="true"]')).map((node) => ({ text: node.getAttribute("aria-label") ?? node.textContent?.trim().slice(0, 80), ...rect(node) }));
    const facts = aside ? [...aside.querySelectorAll("dt")].map((node) => node.textContent.trim()) : [];
    const headerBounds = rect(header);
    const h1 = header.querySelector("h1");
    const titleBounds = rect(h1);
    const status = firstVisible(".vendor-relationship-status");
    const requestsSummary = firstVisible(".vendor-request-details > summary")?.textContent.trim() ?? null;
    const responseHistory = firstVisible('[aria-label="Submitted response history"]');
    const documents = firstVisible(".document-browser");
    const pending = [...(main?.querySelectorAll('[aria-busy="true"], [role="status"], button') ?? [])].filter((node) => visible(node) && /^(Loading|Checking)/.test(node.textContent.trim()));
    const route = location.hash;
    const relationshipID = route.startsWith("#vendors/register/") ? decodeURIComponent(route.slice("#vendors/register/".length)) : null;
    return { kind: view, heading: headerBounds, headingText: h1.textContent.trim(), title: titleBounds,
      selectedRelationshipID: relationshipID, selectedRoute: route,
      localTabs: tabs.map((node) => node.textContent.trim()), selectedTab: selected.textContent.trim(), tablist: menuBounds,
      selectedPanelID, visiblePanelIDs: visiblePanels.map((node) => node.id), selectedPanelControlsMatch: visiblePanels.length === 1 && visiblePanels[0] === selectedPanel,
      selectedTabBounds: selectedBounds, menuMode: visible(tabs[0]) ? "tabs" : "select",
      compactValue: compact?.querySelector(".cs-select-field__value")?.textContent.trim() ?? null,
      selectedTabVisible: Boolean(selectedBounds && menuBounds && selectedBounds.x >= menuBounds.x - 1 && selectedBounds.right <= menuBounds.right + 1),
      main: mainBounds, aside: asideBounds,
      placement: mainBounds && asideBounds && asideBounds.x >= mainBounds.right - 1 ? "beside" : "stacked",
      firstUsefulContent: contentBounds,
      firstUsefulContentVisible: Boolean(contentBounds && contentBounds.y >= 0 && Math.min(contentBounds.bottom, innerHeight) - contentBounds.y >= 44),
      parentHeadingVisible: visible(document.querySelector(".vendors-topbar")), parentMenuVisible: visible(document.querySelector(".vendor-subnav")),
      registerVisible: visible(document.querySelector(".vendor-register")),
      mainHeading: main?.querySelector("h2,h3")?.textContent.trim() ?? null, relationshipFacts: facts,
      statusFields: status ? [...status.querySelectorAll("dt")].map((node) => ({ label: node.textContent.trim(), value: node.nextElementSibling?.textContent.trim(), bounds: rect(node.parentElement) })) : [],
      settledContent: { pendingVisible: pending.length, requestsSummary,
        complianceStatus: firstVisible(".vendor-compliance__heading .cs-status-badge")?.textContent.trim() ?? null,
        complianceRows: main?.querySelectorAll(".vendor-compliance__forms > li").length ?? 0,
        formsEmptyState: [...(main?.querySelectorAll(".vendor-compliance p") ?? [])].some((node) => visible(node) && node.textContent.trim() === "No forms requested"),
        formsShownScope: [...(main?.querySelectorAll(".vendor-compliance__scope") ?? [])].find((node) => /\d+ forms shown/.test(node.textContent))?.textContent.trim() ?? null,
        summaryCountsVisible: Boolean(firstVisible(".vendor-compliance__counts")),
        exceptionRows: main?.querySelectorAll(".vendor-exception-row").length ?? 0,
        responseRevisionSummary: responseHistory ? [...responseHistory.querySelectorAll("p")].map((node) => node.textContent.trim()).find((text) => /^\d+ submitted revisions loaded$/.test(text)) ?? null : null,
        documentState: documents ? documents.querySelector(".document-list-footer") ? "rows" : documents.querySelector(".cs-empty-state") ? "empty" : /Documents could not be loaded/.test(documents.textContent) ? "error" : "loading" : null },
      titleContained: Boolean(titleBounds && headerBounds && titleBounds.x >= headerBounds.x - 1 && titleBounds.right <= headerBounds.right + 1),
      documentWidth: document.documentElement.clientWidth, scrollWidth: document.documentElement.scrollWidth,
      minTargetWidth: Math.min(...targets.map((target) => target.width)), minTargetHeight: Math.min(...targets.map((target) => target.height)),
      undersizedTargets: targets.filter((target) => target.width < 44 || target.height < 44),
      activityScope: view === "activity" ? main?.querySelector('[aria-label="Submitted response history"]')?.textContent.trim().slice(0, 200) ?? null : null,
      apiReads: window.vendorRelationshipEvidence?.reads ?? [], commands: window.vendorRelationshipEvidence?.commands ?? [],
      preparationReads: window.vendorRelationshipEvidence?.preparationReads ?? [],
      selectionBoundary: window.vendorRelationshipEvidence?.selectionBoundary ?? null,
      evidenceKind: "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION",
    };
  }, scenario.vendorRelationshipView);
  const { main, aside } = reading;
  const expectedPlacement = scenario.viewport.width > 840 ? "beside" : "stacked";
  if (reading.localTabs.join("|") !== labels.join("|") || reading.selectedTab !== viewLabel(scenario.vendorRelationshipView)
    || reading.selectedRelationshipID !== fixtureRelationshipID || !reading.selectedPanelControlsMatch
    || reading.menuMode === "select" && reading.compactValue !== reading.selectedTab
    || !reading.selectedTabVisible || !reading.titleContained || reading.parentHeadingVisible || reading.parentMenuVisible || reading.registerVisible
    || reading.scrollWidth > reading.documentWidth + 1 || reading.heading.bottom > reading.tablist.y + 1
    || scenario.vendorRelationshipMatrix && !reading.firstUsefulContentVisible) {
    throw new Error(`${scenario.name}: relationship page heading, menu or first content differs from its source boundary: ${JSON.stringify(reading)}`);
  }
  if (!main || !aside || reading.placement !== expectedPlacement
    || expectedPlacement === "beside" && (Math.abs(aside.width - 316) > 2 || Math.abs(main.y - aside.y) > 2 || main.right > aside.x + 1)
    || expectedPlacement === "stacked" && (main.bottom > aside.y + 1 || Math.abs(main.x - aside.x) > 2 || Math.abs(main.width - aside.width) > 2)) {
    throw new Error(`${scenario.name}: relationship primary/context replacement geometry is invalid: ${JSON.stringify({ main, aside, expectedPlacement })}`);
  }
  if (scenario.touch && reading.undersizedTargets.length) throw new Error(`${scenario.name}: relationship controls below 44px: ${JSON.stringify(reading.undersizedTargets)}`);
  if (reading.commands.length) throw new Error(`${scenario.name}: viewing the relationship sent a material command: ${JSON.stringify(reading.commands)}`);
  assertRelationshipReads(reading.apiReads, scenario);
  if (scenario.vendorRelationshipView === "activity" && !responseHistoryOwner(reading.apiReads)) throw new Error(`${scenario.name}: Activity did not read submitted revisions for the selected relationship`);
  if (scenario.vendorRelationshipMatrix && (reading.settledContent.pendingVisible || scenario.vendorRelationshipView === "evidence" && !reading.settledContent.requestsSummary
    || scenario.vendorRelationshipView === "activity" && !reading.settledContent.responseRevisionSummary
    || scenario.vendorRelationshipView === "issues" && reading.settledContent.exceptionRows !== 5)) throw new Error(`${scenario.name}: settled scoped content is missing`);
  if (scenario.vendorRelationshipView === "activity" && !reading.activityScope?.includes("Response history")) throw new Error(`${scenario.name}: Activity must retain the submitted response history scope`);
  return reading;
}

export async function verifyVendorRelationshipScenario(page, scenario) {
  const check = scenario.vendorRelationshipCheck;
  if (scenario.verifyVendorRelationshipNavigation) {
    const selectedRoute = new URL(page.url()).hash;
    const nextActionNavigation = scenario.viewport.width <= 760 ? await verifyNextActionNavigation(page, scenario, selectedRoute) : undefined;
    const keyboard = await verifyKeyboardNavigation(page, scenario);
    const visited = [];
    for (const view of vendorRelationshipViews) {
      await selectVendorRelationshipView(page, view.id);
      await waitForVendorView(page, { ...scenario, vendorRelationshipView: view.id });
      visited.push(view.label);
      if (new URL(page.url()).hash !== selectedRoute) throw new Error(`${scenario.name}: section selection changed relationship identity`);
    }
    const historyReads = await page.evaluate(() => window.vendorRelationshipEvidence?.reads ?? []);
    const responseHistoryReachable = responseHistoryOwner(historyReads);
    if (!responseHistoryReachable) throw new Error(`${scenario.name}: Activity did not read submitted revisions for the selected relationship`);
    await selectVendorRelationshipView(page, "evidence");
    const docs = workspace(page).getByRole("tab", { name: "Documents", exact: true });
    if (await docs.isVisible()) await docs.click();
    else await selectEvidenceView(page, "Documents");
    await workspace(page).locator(".document-browser").waitFor();
    await workspace(page).getByText("Loading documents…", { exact: true }).waitFor({ state: "hidden" });
    const documentOwner = await observedDocumentOwner(page);
    if (!documentOwner) throw new Error(`${scenario.name}: Documents did not read the selected relationship`);
    await selectEvidenceView(page, "Forms");
    await workspace(page).locator(".vendor-forms-panel").waitFor();
    await workspace(page).getByRole("button", { name: "Back to vendor register", exact: true }).click();
    await page.getByRole("heading", { name: "Vendor register", exact: true }).waitFor();
    await page.waitForFunction(() => document.getElementById("vendor-portfolio-register") === document.activeElement).catch(async () => {
      const focus = await page.evaluate(() => ({ tag: document.activeElement?.tagName, id: document.activeElement?.id,
        role: document.activeElement?.getAttribute("role"), label: document.activeElement?.getAttribute("aria-label"), route: location.hash }));
      throw new Error(`${scenario.name}: returning to the register lost focus: ${JSON.stringify(focus)}`);
    });
    const backRestoredFocus = await page.locator("#vendor-portfolio-register").evaluate((node) => node === document.activeElement);
    if (!backRestoredFocus) throw new Error(`${scenario.name}: returning to the register lost focus`);
    await page.locator(".vendor-row").first().click();
    await workspace(page).getByRole("heading", { level: 1 }).waitFor();
    await waitForVendorView(page, { ...scenario, vendorRelationshipView: "overview" });
    await resetScroll(page);
    return { visited, keyboard, nextActionNavigation, selectedRoute, relationshipPreserved: true, evidenceDocumentsReachable: documentOwner, evidenceFormsReachable: true, responseHistoryReachable, backRestoredFocus, baselineRestored: true };
  }
  if (check === "documents") {
    await selectEvidenceView(page, "Documents");
    const documentBrowser = workspace(page).locator(".document-browser");
    await documentBrowser.waitFor();
    await documentBrowser.getByText("Loading documents…", { exact: true }).waitFor({ state: "hidden" });
    await resetScroll(page);
    const protectedDocumentOwner = await observedDocumentOwner(page);
    if (!protectedDocumentOwner) throw new Error(`${scenario.name}: Documents did not read the selected relationship`);
    return { documentsReachable: await documentBrowser.isVisible(), protectedDocumentOwner };
  }
  if (check === "unavailable") {
    const before = await page.evaluate(() => window.vendorRelationshipEvidence.reads.filter((read) => read.status === 503).length);
    await workspace(page).getByRole("button", { name: "Retry", exact: true }).click();
    await page.waitForFunction((before) => window.vendorRelationshipEvidence.reads.filter((read) => read.status === 503).length > before, before);
    await workspace(page).getByText("Compliance status unavailable", { exact: true }).waitFor();
    await resetScroll(page);
    return { unavailable: true, retryIssued: true, failureRetained: true };
  }
  if (check === "activation-denied") {
    const panel = workspace(page).locator(".vendor-activation-panel");
    await panel.getByRole("button", { name: "Reload activation checks", exact: true }).waitFor();
    if (await panel.getByRole("button", { name: "Activate vendor relationship", exact: true }).count()) throw new Error(`${scenario.name}: denied activation exposed a decision action`);
    await panel.scrollIntoViewIfNeeded();
    return { denied: true, deniedRead: "/activation", activationCommandAbsent: true, stateCapture: await captureBounds(panel, scenario) };
  }
  if (check === "assessment-unavailable") {
    const unavailable = workspace(page).getByText("Due diligence is unavailable", { exact: true });
    await unavailable.scrollIntoViewIfNeeded();
    return { assessmentUnavailable: true, stateCapture: await captureBounds(unavailable.locator(".."), scenario) };
  }
  if (check === "empty") {
    const compliance = workspace(page).locator(".vendor-compliance");
    const empty = compliance.getByText("No forms requested", { exact: true });
    const requestForm = compliance.getByRole("button", { name: "Request form", exact: true });
    return { empty: await empty.isVisible(), requestFormReachable: await requestForm.isVisible(),
      status: await compliance.locator(".vendor-compliance__heading .cs-status-badge").textContent(),
      forms: await compliance.locator(".vendor-compliance__forms > li").count() };
  }
  if (check === "partial") {
    const compliance = workspace(page).locator(".vendor-compliance");
    await compliance.evaluate((node) => node.scrollIntoView({ block: "start" }));
    const unavailable = compliance.getByText("Compliance status unavailable", { exact: true });
    const partial = compliance.getByText("1 forms shown", { exact: true });
    return { partial: true, status: await compliance.locator(".vendor-compliance__heading .cs-status-badge").textContent(),
      forms: await compliance.locator(".vendor-compliance__forms > li").count(),
      moreFormsReachable: await compliance.getByRole("button", { name: "Load more forms", exact: true }).isVisible(),
      unavailableCapture: await captureBounds(unavailable, scenario), partialCapture: await captureBounds(partial, scenario) };
  }
  if (check === "loading") return { loading: true, pendingReads: await page.evaluate(() => window.vendorRelationshipEvidence.reads.filter((read) => read.status === "PENDING").length) };
  if (check === "long-labels") return { longLabels: true, title: await workspace(page).getByRole("heading", { level: 1 }).textContent() };
  return undefined;
}

function assertRelationshipReads(reads, scenario) {
  const currentPath = `/api/v1/vendors/${fixtureRelationshipID}/assessments/current`;
  // This existing form-assessment fixture has submitted forms but no current
  // due-diligence assessment. The runtime intentionally maps its 404 to the
  // known "Not started" state; other failures must remain unavailable.
  const expectedStatus = scenario.vendorRelationshipCheck === "loading" ? "PENDING" : scenario.vendorRelationshipCheck === "assessment-unavailable" ? 503 : 404;
  if (!reads.some((read) => read.path === currentPath && read.status === expectedStatus)) throw new Error(`${scenario.name}: selected relationship assessment read is missing: ${JSON.stringify(reads)}`);
  for (const read of reads) {
    const record = read.path === "/api/v1/vendors/form-summaries" ? null : /^\/api\/v1\/vendors\/([^/]+)(?:\/.*)?$/.exec(read.path);
    const query = new URLSearchParams(read.query);
    if (record && decodeURIComponent(record[1]) !== fixtureRelationshipID
      || read.path === "/api/v1/authority/resolve" && (read.method !== "POST" || read.requestKind !== "AUTHORITY_READ_QUERY" || read.queryInput?.object_type !== "THIRD_PARTY_RELATIONSHIP" || read.queryInput?.object_id !== fixtureRelationshipID)
      || read.path === "/api/v1/vendor-work" && query.get("relationship_id") !== fixtureRelationshipID
      || read.path === "/api/v1/forms/documents" && query.get("relationship_id") !== fixtureRelationshipID
      || read.path === "/api/v1/forms/responses" && (query.get("subject_type") !== "VENDOR_RELATIONSHIP" || query.get("subject_id") !== fixtureRelationshipID)) {
      throw new Error(`${scenario.name}: a scoped reader escaped the selected relationship: ${read.path}${read.query}`);
    }
  }
}

async function observedDocumentOwner(page) {
  return page.evaluate((id) => {
    const reads = (window.vendorRelationshipEvidence?.reads ?? []).filter((read) => read.path === "/api/v1/forms/documents");
    return reads.length > 0 && reads.every((read) => { const query = new URLSearchParams(read.query); return read.status === 200 && query.get("relationship_id") === id && query.get("limit") === "25" && query.get("current_only") === "true"; });
  }, fixtureRelationshipID);
}

function responseHistoryOwner(reads) {
  const history = reads.filter((read) => read.path === "/api/v1/forms/responses");
  return history.length > 0 && history.every((read) => {
    const query = new URLSearchParams(read.query);
    return read.method === "GET" && read.status === 200 && query.get("subject_type") === "VENDOR_RELATIONSHIP"
      && query.get("subject_id") === fixtureRelationshipID && query.get("current_only") === "false"
      && query.get("sort") === "COMPLETED_DESC" && query.get("limit") === "25";
  });
}

async function verifyNextActionNavigation(page, scenario, selectedRoute) {
  const next = workspace(page).locator(".vendor-relationship-context:visible").getByRole("button", { name: "Open due diligence", exact: true });
  await next.scrollIntoViewIfNeeded();
  const fromScrollY = await page.evaluate(() => scrollY);
  await next.click();
  await waitForVendorView(page, { ...scenario, vendorRelationshipView: "due-diligence" });
  await page.waitForFunction(() => {
    const target = document.querySelector(".vendor-relationship-primary .vdd-workspace");
    const b = target?.getBoundingClientRect();
    return document.activeElement === target && b?.y >= 0 && Math.min(b.bottom, innerHeight) - b.y >= 44;
  });
  const target = workspace(page).locator(".vendor-relationship-primary:visible .vdd-workspace");
  const state = await captureBounds(target, scenario);
  const focused = await target.evaluate((node) => node === document.activeElement);
  const relationshipPreserved = new URL(page.url()).hash === selectedRoute;
  if (!fromScrollY || !focused || !state.visible || !relationshipPreserved) throw new Error(`${scenario.name}: the lower context action did not reveal its focused due-diligence owner`);
  await selectVendorRelationshipView(page, "overview");
  await waitForVendorView(page, { ...scenario, vendorRelationshipView: "overview" });
  await resetScroll(page);
  return { fromScrollY, focused, visible: state.visible, bounds: state.bounds, relationshipPreserved };
}

async function verifyKeyboardNavigation(page, scenario) {
  await selectVendorRelationshipView(page, "overview");
  const first = navigation(page).getByRole("tablist", { name: "Vendor sections", exact: true }).getByRole("tab", { name: "Overview", exact: true });
  const sequence = [];
  if (await first.isVisible()) {
    await first.focus();
    for (const view of [...vendorRelationshipViews.slice(1), vendorRelationshipViews[0]]) {
      await page.keyboard.press("ArrowRight");
      await navigation(page).getByRole("tab", { name: view.label, exact: true }).and(page.locator('[aria-selected="true"]')).waitFor();
      await waitForVendorView(page, { ...scenario, vendorRelationshipView: view.id });
      sequence.push(view.label);
    }
    return { mode: "tabs", sequence, complete: true };
  }
  const trigger = navigation(page).locator(":scope > .cs-tabs__compact .cs-select-field__trigger");
  await trigger.focus();
  await page.keyboard.press("Space");
  const options = await page.getByRole("option").allTextContents();
  if (options.map((label) => label.trim()).join("|") !== labels.join("|")) throw new Error(`${scenario.name}: compact navigation omitted a relationship section`);
  await page.keyboard.press("ArrowDown");
  await page.keyboard.press("Enter");
  await waitForVendorView(page, { ...scenario, vendorRelationshipView: "due-diligence" });
  sequence.push("Due diligence");
  await selectVendorRelationshipView(page, "overview");
  return { mode: "select", sequence, options: labels, complete: true };
}

async function selectEvidenceView(page, label) {
  const tab = workspace(page).getByRole("tab", { name: label, exact: true });
  if (await tab.isVisible()) await tab.click();
  else {
    const trigger = workspace(page).locator(".vendor-relationship-primary:visible .cs-tabs__compact .cs-select-field__trigger").first();
    await trigger.click();
    await page.getByRole("option", { name: label, exact: true }).click();
  }
}

async function captureBounds(target, scenario) {
  const bounds = await target.boundingBox();
  return { visible: Boolean(bounds && bounds.y >= 0 && bounds.y < scenario.viewport.height && bounds.x >= 0 && bounds.x + bounds.width <= scenario.viewport.width + 1), bounds };
}

export async function measureVendorRelationshipSourceLayout(page, view) {
  return page.evaluate((view) => {
    const root = document.querySelector("#vendor-detail");
    const rect = (node) => { const b = node.getBoundingClientRect(); return { x: b.x, y: b.y, width: b.width, height: b.height, right: b.right, bottom: b.bottom }; };
    const tabs = [...root.querySelectorAll('.detail-tabs [role="tab"]')];
    const selected = tabs.find((node) => node.getAttribute("aria-selected") === "true");
    const main = rect(root.querySelector(".detail-left")); const aside = rect(root.querySelector(".detail-right"));
    const first = rect(root.querySelector(".detail-status-card"));
    return { kind: view, heading: rect(root.querySelector(".detail-title-row")), localTabs: tabs.map((node) => node.textContent.trim()), selectedTab: selected.textContent.trim(),
      tablist: rect(root.querySelector(".detail-tabs")), selectedTabBounds: rect(selected), main, aside,
      placement: aside.x >= main.right - 1 ? "beside" : "stacked", firstUsefulContent: first,
      firstUsefulContentVisible: first.y >= 0 && Math.min(first.bottom, innerHeight) - first.y >= 44,
      parentHeadingVisible: Boolean(document.querySelector(".vendor-heading").getClientRects().length),
      parentMenuVisible: Boolean(document.querySelector(".vendor-tabs").getClientRects().length),
      statusFields: [...root.querySelectorAll(".detail-step")].map((node) => ({ label: node.querySelector("strong").textContent.trim(), bounds: rect(node) })),
      documentWidth: document.documentElement.clientWidth, scrollWidth: document.documentElement.scrollWidth };
  }, view);
}
