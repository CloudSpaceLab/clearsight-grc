// Rendered Notifications anatomy and real API-shape UI interactions. These
// fixtures are deliberately separate from backend authorization evidence.
const labels = ["Inbox", "Activity", "Preferences"];
const title = (view) => view[0].toUpperCase() + view.slice(1);
const firstTitle = "Review the Sample annual-return evidence change";
const secondTitle = "Sample privacy review mentions you";
const notificationCursor = "sample-notifications/older?batch=2&recipient=sample";
const activityCursor = "sample-activity/older?batch=2&entity=bank-ng";
const root = (page) => page.locator(".notifications-workspace");
const panel = (page) => root(page).locator('[role="tabpanel"]:visible');
const rows = (page) => panel(page).locator(".notification-inbox__list .cs-selectable-record");

export async function prepareNotifications(page, scenario) {
  await root(page).waitFor({ state: "visible" });
  await root(page).getByRole("tab", { name: title(scenario.notificationView), exact: true }).waitFor({ state: "visible" });
  if (scenario.notificationView === "inbox") {
    if (scenario.fixture === "notifications-unavailable") await panel(page).getByRole("heading", { name: "Notifications unavailable", exact: true }).waitFor({ state: "visible" });
    else if (scenario.fixture === "notifications-empty") await panel(page).getByRole("heading", { name: "No notifications loaded", exact: true }).waitFor({ state: "visible" });
    else await rows(page).filter({ hasText: firstTitle }).waitFor({ state: "visible" });
  } else if (scenario.notificationView === "activity") {
    const failure = scenario.fixture === "notifications-activity-denied" ? "Activity access is restricted"
      : ["notifications-activity-unavailable", "notifications-activity-foreign"].includes(scenario.fixture) ? "Activity could not be loaded"
        : scenario.fixture === "notifications-empty" ? "No activity in this period" : null;
    if (failure) await panel(page).getByRole("heading", { name: failure, exact: true }).waitFor({ state: "visible" });
    else await panel(page).locator('[data-activity-event-id="sample-activity-event-1"]').waitFor({ state: "visible" });
  } else await panel(page).getByRole("button", { name: "Save email preferences", exact: true }).waitFor({ state: "visible" });
  await resetScroll(page);
}

export async function assertNotificationsLayout(page, scenario) {
  const reading = await page.evaluate((view) => {
    const workspace = document.querySelector(".notifications-workspace");
    const active = [...workspace.querySelectorAll('[role="tabpanel"]')].find((node) => node.getClientRects().length && getComputedStyle(node).display !== "none");
    const rect = (node) => { if (!node?.getClientRects().length) return null; const b = node.getBoundingClientRect(); return { x: b.x, y: b.y, width: b.width, height: b.height, right: b.right, bottom: b.bottom }; };
    const get = (selector) => active?.querySelector(selector);
    const tablist = workspace.querySelector('[role="tablist"]');
    const tabs = [...tablist.querySelectorAll('[role="tab"]')];
    const selectedTab = tabs.find((node) => node.getAttribute("aria-selected") === "true");
    const heading = workspace.querySelector(".notifications-workspace__header");
    const main = get(view === "inbox" ? ".notification-inbox__list-panel" : view === "activity" ? ".notification-activity__panel" : ".notification-settings__in-app");
    const aside = get(view === "inbox" ? ".notification-detail" : view === "activity" ? ".notification-activity__aside" : ".notification-preferences");
    const recordNodes = [...active.querySelectorAll(view === "inbox" ? ".notification-inbox__list .cs-selectable-record" : "[data-activity-event-id]")];
    const recordBoxes = recordNodes.map(rect);
    const useful = view === "inbox" ? recordNodes[0] ?? get(".cs-empty-state")
      : view === "activity" ? innerWidth <= 900 ? aside?.querySelector(".notification-activity__proof > div") ?? get(".cs-empty-state") : recordNodes[0] ?? get(".cs-empty-state")
        : main?.querySelector("dl > div");
    const usefulBox = rect(useful);
    const tabBox = rect(tablist); const selectedBox = rect(selectedTab);
    const mainBox = rect(main); const asideBox = rect(aside);
    const placement = !mainBox || !asideBox ? "restricted" : innerWidth > 900 ? "beside" : view === "activity" ? "context-above" : "stacked";
    const search = get('[type="search"]');
    const readFilter = get('[aria-label="Inbox read filter"]');
    const selectedRecord = recordNodes.find((node) => node.getAttribute("aria-pressed") === "true");
    // A React Aria checkbox's label is its visible activation target; its native
    // input is deliberately clipped for accessibility, not the touch surface.
    const touchTargets = [...new Set([...workspace.querySelectorAll('button, input:not([type="hidden"]), [role="tab"]')].map((node) => node.closest(".cs-checkbox-field") ?? node))].filter((node) => node.getClientRects().length && !node.closest('[inert]')).map((node) => ({ text: node.getAttribute("aria-label") ?? node.textContent.trim().slice(0, 80), ...rect(node) }));
    return {
      kind: view, heading: rect(heading), headingText: heading.querySelector("h1").textContent.trim(),
      localTabs: tabs.map((node) => node.textContent.trim()), selectedTab: selectedTab?.textContent.trim(), tablist: tabBox, selectedTabBounds: selectedBox,
      selectedTabVisible: Boolean(tabBox && selectedBox && selectedBox.x >= tabBox.x - 1 && selectedBox.right <= tabBox.right + 1),
      main: mainBox, aside: asideBox, placement, toolbar: rect(get(".notification-inbox__toolbar")),
      firstUsefulContent: usefulBox, firstUsefulContentText: useful?.textContent.trim().slice(0, 200) ?? null,
      firstUsefulContentVisible: Boolean(usefulBox && usefulBox.y >= 0 && Math.min(usefulBox.bottom, innerHeight) - usefulBox.y >= 44),
      documentWidth: document.documentElement.clientWidth, scrollWidth: document.documentElement.scrollWidth,
      recordCount: recordNodes.length, recordHeights: recordBoxes.map((box) => box.height),
      firstRecord: recordBoxes[0] ?? null, firstRecordVisible: Boolean(recordBoxes[0] && recordBoxes[0].y < innerHeight && recordBoxes[0].bottom > 0),
      selectedCount: recordNodes.filter((node) => node.getAttribute("aria-pressed") === "true").length,
      selectedRecord: selectedRecord?.querySelector(".cs-selectable-record__title")?.textContent.trim() ?? null,
      detailTitle: get(".notification-detail__header h2")?.textContent.trim() ?? null,
      readFilter: readFilter ? [...readFilter.querySelectorAll("button")].map((node) => ({ label: node.textContent.trim(), selected: node.getAttribute("aria-pressed") === "true" })) : null,
      unreadHeader: document.querySelector(".notification-center__launcher button")?.getAttribute("aria-label") ?? null,
      searchVisible: Boolean(search?.getClientRects().length), filtersToggleVisible: Boolean(get(".notification-inbox__filter-toggle")?.getClientRects().length),
      emptyHeadings: [...active.querySelectorAll(".cs-empty-state h2, .cs-empty-state h3")].map((node) => node.textContent.trim()),
      activityEvents: view === "activity" ? recordNodes.map((node) => ({ id: node.getAttribute("data-activity-event-id"), action: node.querySelector("h4")?.textContent.trim(),
        actor: node.querySelector(".notification-activity__actor strong")?.textContent.trim(),
        outcome: node.querySelector(".notification-activity__record-heading .cs-status-badge")?.textContent.trim(),
        fields: [...node.querySelectorAll(".notification-activity__facts dt")].map((field) => field.textContent.trim()), text: node.textContent.trim() })) : [],
      activityVersionLimitation: view === "activity" ? get(".notification-activity__hint")?.textContent.trim() : null,
      preferences: view === "preferences" ? { daily: get('input[type="checkbox"]')?.checked,
        times: [...active.querySelectorAll('input[type="time"]')].map((node) => node.value), criticalRequired: /Critical required/.test(active.textContent),
        inAppHeading: main?.querySelector("h3")?.textContent.trim(), emailHeading: aside?.querySelector("#notification-email-heading")?.textContent.trim() } : null,
      unsupportedControls: [...active.querySelectorAll("button")].filter((node) => /^(Needs action|Mark visible as read|Snooze|Mark as unread)$/.test(node.textContent.trim())).length,
      minTargetWidth: Math.min(...touchTargets.map((node) => node.width)), minTargetHeight: Math.min(...touchTargets.map((node) => node.height)),
      undersizedTargets: touchTargets.filter((node) => node.width < 44 || node.height < 44),
    };
  }, scenario.notificationView);
  if (reading.headingText !== "Notifications" || reading.localTabs.join("|") !== labels.join("|") || reading.selectedTab !== title(scenario.notificationView)
    || !reading.selectedTabVisible || !reading.firstUsefulContentVisible || reading.scrollWidth > reading.documentWidth + 1 || reading.unsupportedControls) {
    throw new Error(`${scenario.name}: Notifications heading, local menu or first useful content differs from the source contract: ${JSON.stringify(reading)}`);
  }
  const { main, aside, heading, tablist, placement } = reading;
  if (heading.bottom > tablist.y + 1) throw new Error(`${scenario.name}: heading overlaps the Notification views menu`);
  if (placement === "beside" && (Math.abs(main.y - aside.y) > 2 || main.right > aside.x + 1 || main.width < aside.width)
    || placement === "context-above" && (aside.bottom > main.y + 1 || Math.abs(main.x - aside.x) > 2)
    || placement === "stacked" && (main.bottom > aside.y + 1 || Math.abs(main.x - aside.x) > 2)) {
    throw new Error(`${scenario.name}: Notifications replacement layout is inconsistent: ${JSON.stringify({ main, aside, placement })}`);
  }
  if (scenario.touch && reading.undersizedTargets.length) throw new Error(`${scenario.name}: Notifications controls below44px: ${JSON.stringify(reading.undersizedTargets)}`);
  if (scenario.notificationView === "inbox" && !["notifications-empty", "notifications-unavailable"].includes(scenario.fixture)) {
    if (reading.recordCount !== 5 || reading.selectedCount !== 1 || reading.selectedRecord !== firstTitle || reading.detailTitle !== firstTitle
      || reading.unreadHeader !== "Notifications, 7 unread" || reading.readFilter[0]?.label !== "All" || !reading.readFilter[0]?.selected
      || reading.readFilter[1]?.label !== "Unread (7)" || reading.searchVisible !== (scenario.viewport.width > 600)
      || reading.filtersToggleVisible !== (scenario.viewport.width <= 600)) throw new Error(`${scenario.name}: Inbox delivery, default selection or compact filter replacement is incorrect: ${JSON.stringify(reading)}`);
  }
  if (scenario.notificationView === "activity" && scenario.fixture === "notifications-loaded"
    && (reading.recordCount !== 3 || !reading.activityVersionLimitation?.includes("Record versions are not recorded in this history.")
      || reading.activityEvents.some((item) => !item.action?.includes("Sample") || !item.actor?.includes("Sample") || !item.outcome
        || !item.text.includes(item.id) || item.fields.some((field) => ["Scope", "Version"].includes(field))))) {
    throw new Error(`${scenario.name}: Activity must expose returned events and disclose absent versions in its context`);
  }
  if (scenario.notificationView === "preferences" && (!reading.preferences?.criticalRequired || reading.preferences.inAppHeading !== "In-app notifications" || !reading.preferences.emailHeading)) {
    throw new Error(`${scenario.name}: Preferences must retain the in-app card and required critical email policy`);
  }
  return { ...reading, apiReads: await reads(page), commands: await commands(page), evidenceKind: "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION" };
}

export async function verifyNotificationScenario(page, scenario) {
  const view = scenario.notificationView;
  const check = scenario.notificationCheck;
  if (scenario.verifyNotificationInteraction) {
    const navigation = await verifyTabs(page, view);
    const interaction = view === "inbox" ? await verifyInbox(page, scenario) : view === "activity" ? await verifyActivity(page) : await verifyPreferences(page);
    await resetScroll(page);
    return { navigation, ...interaction, baselineRestored: true };
  }
  if (!check) return undefined;
  if (check === "empty") {
    const expected = view === "inbox" ? "No notifications loaded" : "No activity in this period";
    const heading = panel(page).getByRole("heading", { name: expected, exact: true });
    await heading.waitFor({ state: "visible" });
    const stateCapture = await frameCaptureTarget(heading.locator(".."), scenario);
    return { empty: true, population: view === "inbox" ? "DELIVERED_TO_ACTOR" : "SELECTED_ENTITY_PERIOD", stateCapture,
      rows: await panel(page).locator(view === "inbox" ? ".cs-selectable-record" : "[data-activity-event-id]").count() };
  }
  if (check === "unavailable") {
    const path = view === "inbox" ? "/api/v1/notifications" : "/api/v1/system-activity";
    const before = (await reads(page)).filter((read) => read.path === path).length;
    await panel(page).getByRole("button", { name: view === "inbox" ? "Retry notifications" : "Retry activity", exact: true }).click();
    await page.waitForFunction(({ path, before }) => (window.notificationEvidenceReads ?? []).filter((read) => read.path === path).length > before, { path, before });
    await prepareNotifications(page, scenario);
    const heading = panel(page).getByRole("heading", { name: view === "inbox" ? "Notifications unavailable" : "Activity could not be loaded", exact: true });
    const stateCapture = await frameCaptureTarget(heading.locator(".."), scenario);
    return { unavailable: true, retryIssued: true, failureRetained: true, stateCapture,
      rows: await panel(page).locator(view === "inbox" ? ".cs-selectable-record" : "[data-activity-event-id]").count() };
  }
  if (check === "denied") {
    if ((await reads(page)).some((read) => read.path === "/api/v1/system-activity")) throw new Error(`${scenario.name}: restricted Activity made a protected request`);
    const stateCapture = await frameCaptureTarget(panel(page).getByRole("heading", { name: "Activity access is restricted", exact: true }).locator(".."), scenario);
    return { permission: "platform_operations_read", restricted: true, protectedRequests: 0, stateCapture,
      inboxAvailable: (await reads(page)).some((read) => read.path === "/api/v1/notifications" && read.status === 200) };
  }
  if (check === "foreign") {
    if (await panel(page).locator("[data-activity-event-id]").count()) throw new Error(`${scenario.name}: a mixed-entity Activity response leaked rows`);
    const stateCapture = await frameCaptureTarget(panel(page).getByRole("heading", { name: "Activity could not be loaded", exact: true }).locator(".."), scenario);
    return { foreignEntityRejected: true, rows: 0, stateCapture, returnedStatus: (await reads(page)).find((read) => read.path === "/api/v1/system-activity")?.status };
  }
  if (check === "preferences-conflict") return verifyPreferences(page, true);
  if (check === "more-failed") {
    await panel(page).getByRole("button", { name: "Load more", exact: true }).click();
    const failure = panel(page).getByText("More notifications could not be loaded. The current list remains available.", { exact: true });
    await failure.waitFor({ state: "visible" });
    if (await rows(page).count() !== 5) throw new Error(`${scenario.name}: older-page failure cleared delivered notifications`);
    // This state capture must show its error. The matrix already records the
    // page-top composition; scrolling back there would conceal this result.
    await failure.scrollIntoViewIfNeeded();
    const failureBounds = await failure.evaluate((node) => {
      const box = node.getBoundingClientRect();
      return { x: box.x, y: box.y, width: box.width, height: box.height, right: box.right, bottom: box.bottom };
    });
    const failureVisible = failureBounds.y >= 0 && failureBounds.bottom <= scenario.viewport.height
      && failureBounds.x >= 0 && failureBounds.right <= scenario.viewport.width;
    if (!failureVisible) throw new Error(`${scenario.name}: older-page failure notice is outside the captured viewport`);
    return { retainedRows: 5, olderPageFailure: true, failureVisible, failureBounds,
      cursor: lastQuery(await reads(page), "/api/v1/notifications").get("cursor") };
  }
  if (["selected", "unsafe"].includes(check)) {
    const selectedTitle = check === "unsafe" ? "Sample external reference unavailable" : secondTitle;
    await rows(page).filter({ hasText: selectedTitle }).focus();
    await page.keyboard.press("Enter");
    const detail = panel(page).getByRole("complementary", { name: "Selected notification", exact: true });
    await detail.getByRole("heading", { name: selectedTitle, exact: true }).waitFor({ state: "visible" });
    await page.waitForFunction(() => document.activeElement?.matches(".notification-detail"));
    const open = detail.getByRole("button", { name: "Open record", exact: true });
    if (check === "unsafe" && !await open.isDisabled()) throw new Error(`${scenario.name}: unsupported external action path is enabled`);
    if ((await commands(page)).length) throw new Error(`${scenario.name}: selecting a notification unexpectedly changed its read state`);
    const unsafeActionCapture = check === "unsafe" ? await frameCaptureTarget(open, scenario) : undefined;
    return { keyboardSelected: true, selectedTitle, detailFocused: true, readCommands: 0, unsafeActionDisabled: check === "unsafe" ? true : undefined,
      ...(unsafeActionCapture ? { unsafeActionCapture } : {}), detailBounds: await detail.boundingBox() };
  }
  if (check === "read" || check === "read-failed") {
    await panel(page).getByRole("button", { name: /^Unread \(7\)$/ }).click();
    await waitRowCount(page, 3);
    const detail = panel(page).getByRole("complementary", { name: "Selected notification", exact: true });
    await detail.getByRole("button", { name: "Mark as read", exact: true }).click();
    await page.waitForFunction(() => (window.notificationEvidenceCommands ?? []).some((command) => command.path.endsWith("/read")));
    const readCommands = (await commands(page)).filter((command) => command.path.endsWith("/read"));
    const command = readCommands[0];
    if (readCommands.length !== 1 || command.path !== "/api/v1/notifications/sample-notification-1/read" || Object.keys(command.body).length) throw new Error(`${scenario.name}: read receipt command includes an unexpected record or identity`);
    const failed = check === "read-failed";
    if (failed) {
      await detail.getByText("Could not mark the notification as read. Try again.", { exact: true }).waitFor({ state: "visible" });
      await waitRowCount(page, 3);
    } else {
      await detail.getByRole("button", { name: "Marked as read", exact: true }).waitFor({ state: "visible" });
      await waitRowCount(page, 2);
    }
    const expected = failed ? 7 : 6;
    await page.getByRole("button", { name: `Notifications, ${expected} unread`, exact: true }).waitFor({ state: "visible" });
    const header = await page.locator(".notification-center__launcher button").getAttribute("aria-label");
    if (header !== `Notifications, ${expected} unread` || command.status !== (failed ? 503 : 200)
      || await detail.getByRole("heading", { name: firstTitle, exact: true }).count() !== 1) throw new Error(`${scenario.name}: read result diverges from selected context and authoritative unread count`);
    const visibleRows = await rows(page).count();
    let recordHandoff;
    if (!failed) {
      await detail.getByRole("button", { name: "Open record", exact: true }).click();
      await page.waitForFunction(() => window.location.hash === "#work/matters/matter-gaid-change");
      await page.getByRole("region", { name: "Issue or change record", exact: true }).waitFor({ state: "visible" });
      await page.getByRole("button", { name: "Notifications, 6 unread", exact: true }).click();
      await root(page).waitFor({ state: "visible" });
      await panel(page).getByRole("button", { name: "All", exact: true }).click();
      await waitRowCount(page, 5);
      await panel(page).getByRole("complementary", { name: "Selected notification", exact: true }).getByRole("button", { name: "Marked as read", exact: true }).waitFor({ state: "visible" });
      if ((await commands(page)).length !== 1) throw new Error("Opening the already-read record issued an extra command");
      recordHandoff = { route: "#work/matters/matter-gaid-change", recordRendered: true, bellReturned: true, duplicateReadCommands: 0 };
    }
    if (scenario.viewport.width > 900) await resetScroll(page);
    return { readResult: failed ? "FAILED" : "CONFIRMED", command, unreadCount: expected, retainedSelectedContext: true,
      visibleRows, workCommands: 0, ...(recordHandoff ? { recordHandoff } : {}) };
  }
  throw new Error(`Unknown Notifications evidence check: ${check}`);
}

async function verifyTabs(page, originalView) {
  const list = root(page).getByRole("tablist", { name: "Notification views" });
  await list.getByRole("tab", { name: "Inbox", exact: true }).click();
  await list.getByRole("tab", { name: "Inbox", exact: true }).focus();
  const sequence = [];
  for (const selected of ["Activity", "Preferences", "Inbox"]) {
    await page.keyboard.press("ArrowRight");
    await page.waitForFunction((selected) => [...document.querySelectorAll('.notifications-workspace [role="tab"]')].some((node) => node.textContent.trim() === selected && node.getAttribute("aria-selected") === "true"), selected);
    sequence.push(selected);
    const expectedHash = selected === "Inbox" ? "#notifications" : `#notifications/${selected.toLowerCase()}`;
    if (new URL(page.url()).hash !== expectedHash) throw new Error(`Notification views did not update route for ${selected}`);
  }
  await list.getByRole("tab", { name: title(originalView), exact: true }).click();
  return { arrowSequence: sequence, routesMatched: true, count: 3 };
}

async function verifyInbox(page, scenario) {
  await waitRowCount(page, 5);
  if (scenario.viewport.width <= 600) await panel(page).getByRole("button", { name: "Show notification filters", exact: true }).click();
  const search = panel(page).getByRole("searchbox", { name: "Find in loaded notifications", exact: true });
  await search.fill("sample-notification-no-match-000");
  await panel(page).getByRole("heading", { name: "No loaded notifications match", exact: true }).waitFor({ state: "visible" });
  await panel(page).getByRole("button", { name: "Clear search", exact: true }).click();
  await waitRowCount(page, 5);
  await rows(page).filter({ hasText: secondTitle }).focus();
  await page.keyboard.press("Enter");
  const detail = panel(page).getByRole("complementary", { name: "Selected notification", exact: true });
  await detail.getByRole("heading", { name: secondTitle, exact: true }).waitFor({ state: "visible" });
  if (scenario.viewport.width <= 900) {
    await page.waitForFunction(() => document.activeElement?.matches(".notification-detail"));
    await detail.getByRole("button", { name: "Back to notifications", exact: true }).click();
    if (!await rows(page).filter({ hasText: secondTitle }).evaluate((node) => node === document.activeElement)) throw new Error("Back to notifications did not restore selected record focus");
  }
  if ((await commands(page)).length) throw new Error("Selection and filtering must not mark a notification as read");
  await panel(page).getByRole("button", { name: /^Unread \(7\)$/ }).click();
  await waitRowCount(page, 3);
  if (lastQuery(await reads(page), "/api/v1/notifications").get("unread_only") !== "true") throw new Error("Unread did not issue the supported server query");
  await panel(page).getByRole("button", { name: "All", exact: true }).click();
  await waitRowCount(page, 5);
  await panel(page).getByRole("button", { name: "Load more", exact: true }).click();
  await waitRowCount(page, 10);
  if (lastQuery(await reads(page), "/api/v1/notifications").get("cursor") !== notificationCursor) throw new Error("Notification pagination changed the opaque cursor");
  await panel(page).getByRole("button", { name: "Refresh", exact: true }).click();
  await waitRowCount(page, 5);
  if (scenario.viewport.width <= 600) await panel(page).getByRole("button", { name: "Hide notification filters", exact: true }).click();
  await panel(page).getByRole("button", { name: "Open assigned work", exact: true }).click();
  await page.waitForFunction(() => window.location.hash === "#work");
  await page.getByRole("heading", { name: "Work & approvals", exact: true }).waitFor({ state: "visible" });
  await page.getByRole("button", { name: "Notifications, 7 unread", exact: true }).click();
  await root(page).waitFor({ state: "visible" });
  await waitRowCount(page, 5);
  return { searchEmptyAndRecovery: true, keyboardSelection: true, selectedContextWithoutRead: true,
    backRestoredFocus: scenario.viewport.width <= 900 ? true : null, serverUnreadFilter: true, pagedRows: 10,
    opaqueCursorPreserved: true, refreshedRows: 5, serverUnreadCount: 7, assignedWorkHandoff: true, bellReturned: true };
}

async function verifyActivity(page) {
  await panel(page).locator('[data-activity-event-id="sample-activity-event-1"]').waitFor({ state: "visible" });
  const initial = lastQuery(await reads(page), "/api/v1/system-activity");
  await panel(page).getByRole("button", { name: "Load older activity", exact: true }).click();
  await page.waitForFunction(() => document.querySelectorAll('.notification-activity [data-activity-event-id]').length === 5);
  const older = lastQuery(await reads(page), "/api/v1/system-activity");
  if (older.get("cursor") !== activityCursor || older.get("from") !== initial.get("from") || older.get("to") !== initial.get("to")) throw new Error("Activity pagination changed its period or opaque cursor");
  await choosePeriod(page, "Last 7 days");
  await page.waitForFunction(() => document.querySelector('.notification-activity__summary')?.textContent.includes("Last 7 days") && document.querySelectorAll('.notification-activity [data-activity-event-id]').length === 3);
  const seven = lastQuery(await reads(page), "/api/v1/system-activity");
  if ((Date.parse(seven.get("to")) - Date.parse(seven.get("from"))) / 86400000 !== 7 || seven.has("cursor") || seven.get("legal_entity_id") !== "bank-ng") throw new Error("Activity period did not reset the scoped query");
  await choosePeriod(page, "Last 30 days");
  await panel(page).locator('[data-activity-event-id="sample-activity-event-1"]').waitFor({ state: "visible" });
  return { periodDays: [30, 7, 30], entityID: "bank-ng", fixedPagingWindow: true, opaqueCursorPreserved: true, pagedEvents: 5, periodResetEvents: 3 };
}

async function verifyPreferences(page, conflict = false) {
  const settings = panel(page).locator(".notification-preferences");
  await settings.getByLabel("Digest time", { exact: true }).fill("09:45");
  await settings.getByLabel("Starts", { exact: true }).fill("22:10");
  await settings.getByLabel("Ends", { exact: true }).fill("06:20");
  await root(page).getByRole("tab", { name: "Inbox", exact: true }).click();
  await root(page).getByRole("tab", { name: "Preferences", exact: true }).click();
  const actual = await Promise.all(["Digest time", "Starts", "Ends"].map((label) => settings.getByLabel(label, { exact: true }).inputValue()));
  if (actual.join("|") !== "09:45|22:10|06:20") throw new Error("Notification preference drafts were lost across local navigation");
  await settings.getByRole("button", { name: "Save email preferences", exact: true }).click();
  await page.waitForFunction(() => (window.notificationEvidenceCommands ?? []).some((command) => command.method === "PUT"));
  const command = (await commands(page)).find((item) => item.method === "PUT");
  const expectedBody = { daily_digest_enabled: true, digest_minute: 585, time_zone: "Africa/Lagos", quiet_hours_enabled: true, quiet_start_minute: 1330, quiet_end_minute: 380, expected_version: 3 };
  if (JSON.stringify(command?.body) !== JSON.stringify(expectedBody) || command.status !== (conflict ? 409 : 200)) throw new Error(`Notification preferences did not send real minute values and expected version: ${JSON.stringify(command)}`);
  if (conflict) {
    await settings.getByText("Email preferences changed. Reload them before saving again.", { exact: true }).waitFor({ state: "visible" });
    await settings.getByRole("button", { name: "Reload preferences", exact: true }).waitFor({ state: "visible" });
    if (await settings.getByLabel("Digest time", { exact: true }).inputValue() !== "09:45") throw new Error("Conflict discarded the edited notification preferences");
  } else {
    await settings.getByText("Email preferences saved.", { exact: true }).waitFor({ state: "visible" });
    if (command.result.version !== 4 || command.result.critical_email_required !== true) throw new Error("Saved preferences lost their version or required critical email policy");
  }
  return { draftRetained: true, command, saved: !conflict, conflictRetainedDraft: conflict, criticalRequired: true, inputMinutes: [585, 1330, 380] };
}

async function choosePeriod(page, label) {
  const field = panel(page).locator(".cs-select-field").filter({ has: page.locator(".cs-select-field__label", { hasText: /^Activity period$/ }) });
  await field.locator(".cs-select-field__trigger").click();
  await page.getByRole("option", { name: label, exact: true }).click();
}
async function waitRowCount(page, count) { await page.waitForFunction((count) => [...document.querySelectorAll('.notifications-workspace [role="tabpanel"]')].find((node) => node.getClientRects().length)?.querySelectorAll('.notification-inbox__list .cs-selectable-record').length === count, count); }
async function frameCaptureTarget(target, scenario) {
  await target.evaluate((node) => node.scrollIntoView({ block: "center", inline: "nearest", behavior: "instant" }));
  const bounds = await target.evaluate((node) => {
    const box = node.getBoundingClientRect();
    return { x: box.x, y: box.y, width: box.width, height: box.height, right: box.right, bottom: box.bottom };
  });
  const visible = bounds.width > 0 && bounds.height > 0 && bounds.x >= 0 && bounds.right <= scenario.viewport.width
    && bounds.y >= 0 && bounds.bottom <= scenario.viewport.height;
  if (!visible) throw new Error(`${scenario.name}: the asserted state is outside the captured viewport: ${JSON.stringify(bounds)}`);
  return { visible, bounds };
}
async function resetScroll(page) { await page.evaluate(async () => { window.scrollTo({ top: 0, left: 0, behavior: "instant" }); await new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))); }); }
async function reads(page) { return page.evaluate(() => window.notificationEvidenceReads ?? []); }
async function commands(page) { return page.evaluate(() => window.notificationEvidenceCommands ?? []); }
function lastQuery(readings, path) { return new URLSearchParams(readings.filter((read) => read.path === path).at(-1)?.query); }

export async function measureNotificationsSourceLayout(page, view) {
  const reading = await page.evaluate((view) => {
    const pane = document.querySelector(`#${view}-pane`);
    const rect = (node) => { if (!node?.getClientRects().length) return null; const b = node.getBoundingClientRect(); return { x: b.x, y: b.y, width: b.width, height: b.height, right: b.right, bottom: b.bottom }; };
    const tabs = [...document.querySelectorAll('.notification-nav [role="tab"]')];
    const label = (node) => node.dataset.view[0].toUpperCase() + node.dataset.view.slice(1);
    const main = pane.querySelector(view === "inbox" ? ".notice-list-panel" : view === "activity" ? ".activity-panel" : ".pref-card:not(.schedule-card)");
    const aside = pane.querySelector(view === "inbox" ? ".notice-detail" : view === "activity" ? ".activity-aside" : ".schedule-card");
    const recordNodes = [...pane.querySelectorAll(view === "inbox" ? ".notice-item" : ".activity-record")];
    const useful = view === "inbox" ? recordNodes[0] : view === "activity" ? innerWidth <= 900 ? aside.querySelector(".activity-proof > div") : recordNodes[0] : main.querySelector(".pref-option");
    const usefulBox = rect(useful);
    return { kind: view, heading: rect(document.querySelector(".notice-heading")), headingText: document.querySelector(".notice-heading h1").textContent.trim(),
      localTabs: tabs.map(label), selectedTab: label(tabs.find((node) => node.getAttribute("aria-selected") === "true")),
      main: rect(main), aside: rect(aside), toolbar: rect(pane.querySelector(".inbox-toolbar")),
      placement: innerWidth > 900 ? "beside" : view === "activity" ? "context-above" : "stacked",
      firstUsefulContent: usefulBox, firstUsefulContentText: useful?.textContent.trim().slice(0, 200) ?? null,
      firstUsefulContentVisible: Boolean(usefulBox && usefulBox.y >= 0 && Math.min(usefulBox.bottom, innerHeight) - usefulBox.y >= 44),
      recordCount: recordNodes.length, recordHeights: recordNodes.map((node) => rect(node).height),
      firstRecord: rect(recordNodes[0]), firstRecordVisible: Boolean(recordNodes[0] && rect(recordNodes[0]).y < innerHeight && rect(recordNodes[0]).bottom > 0),
      inboxFilter: view === "inbox" ? pane.querySelector('[data-inbox-filter].selected')?.dataset.inboxFilter : null,
      searchVisible: view === "inbox" ? Boolean(pane.querySelector('#notification-search')?.getClientRects().length) : null,
      documentWidth: document.documentElement.clientWidth, scrollWidth: document.documentElement.scrollWidth,
    };
  }, view);
  if (reading.headingText !== "Notifications" || reading.localTabs.join("|") !== labels.join("|") || reading.selectedTab !== title(view)
    || !reading.main || !reading.aside || !reading.firstUsefulContentVisible || reading.scrollWidth > reading.documentWidth + 1
    || (view === "inbox" && reading.inboxFilter !== "all")) throw new Error(`Original Notifications ${view} anatomy did not render as required: ${JSON.stringify(reading)}`);
  return reading;
}
