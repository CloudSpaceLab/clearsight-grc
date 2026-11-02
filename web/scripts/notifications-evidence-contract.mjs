import { notificationsEvidenceScenarios } from "./notifications-evidence-scenarios.mjs";

const same = (a, b) => JSON.stringify(a) === JSON.stringify(b);
const rectangle = (value) => value && ["x", "y", "width", "height", "right", "bottom"].every((key) => Number.isFinite(value[key]))
  && value.width > 0 && value.height > 0 && Math.abs(value.x + value.width - value.right) <= 1 && Math.abs(value.y + value.height - value.bottom) <= 1;
const framed = (value, viewport) => value?.visible === true && rectangle(value.bounds)
  && value.bounds.x >= 0 && value.bounds.right <= viewport.width && value.bounds.y >= 0 && value.bounds.bottom <= viewport.height;
const title = (view) => view[0].toUpperCase() + view.slice(1);
const inboxPath = "/api/v1/notifications";
const activityPath = "/api/v1/system-activity";
const firstTitle = "Review the Sample annual-return evidence change";
const expectedActivityEvents = [
  { id: "sample-activity-event-1", actor: "Sample Amina Yusuf", outcome: "Succeeded", record: "matter-gaid-change", source: "Workflow" },
  { id: "sample-activity-event-2", actor: "Sample Emeka Okoro", outcome: "Succeeded", record: "program-ndpa", source: "Programs" },
  { id: "sample-activity-event-3", actor: "Sample Ruth Mensah", outcome: "Denied", record: "sample-restricted-change", source: "Authority" },
];

export function reviewNotificationsCaptures(captures) {
  const byName = new Map(captures.map((capture) => [capture.name, capture]));
  const failures = [];
  for (const scenario of notificationsEvidenceScenarios) {
    const capture = byName.get(scenario.name);
    if (!capture) { failures.push(`${scenario.name}: Notifications evidence is missing`); continue; }
    const layout = capture.notificationsLayout;
    const width = scenario.viewport.width;
    const view = scenario.notificationView;
    if (["route", "fixture", "state", "theme"].some((key) => capture[key] !== scenario[key])
      || !same(capture.viewport, scenario.viewport) || !layout || layout.kind !== view
      || layout.evidenceKind !== "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION"
      || capture.navigation?.activePage?.join("|") !== "Work & Approvals") {
      failures.push(`${scenario.name}: Notifications metadata, fixture identity or Work navigation owner is inconsistent`);
      if (!layout) continue;
    }
    if (layout.headingText !== "Notifications" || !rectangle(layout.heading) || !rectangle(layout.tablist)
      || !same(layout.localTabs, ["Inbox", "Activity", "Preferences"]) || layout.selectedTab !== title(view)
      || !rectangle(layout.selectedTabBounds) || layout.selectedTabVisible !== true
      || layout.selectedTabBounds.x < layout.tablist.x - 1 || layout.selectedTabBounds.right > layout.tablist.right + 1
      || layout.heading.bottom > layout.tablist.y + 1 || layout.unsupportedControls !== 0
      || layout.firstUsefulContentVisible !== true || !rectangle(layout.firstUsefulContent)
      || layout.firstUsefulContent.y < 0 || Math.min(layout.firstUsefulContent.bottom, scenario.viewport.height) - layout.firstUsefulContent.y < 44
      || layout.documentWidth !== width || layout.scrollWidth > width + 1
      || (scenario.touch && (layout.minTargetWidth < 44 || layout.minTargetHeight < 44 || layout.undersizedTargets?.length !== 0))) {
      failures.push(`${scenario.name}: Notifications heading, local menu, first useful content or touch/overflow evidence is missing`);
    }
    const denied = scenario.notificationCheck === "denied";
    const expectedPlacement = denied ? "restricted" : width > 900 ? "beside" : view === "activity" ? "context-above" : "stacked";
    const { main, aside } = layout;
    if (layout.placement !== expectedPlacement || !denied && (!rectangle(main) || !rectangle(aside)
      || expectedPlacement === "beside" && (main.width < aside.width || main.right > aside.x + 1 || Math.abs(main.y - aside.y) > 2)
      || expectedPlacement === "context-above" && (aside.bottom > main.y + 1 || Math.abs(main.x - aside.x) > 2)
      || expectedPlacement === "stacked" && (main.bottom > aside.y + 1 || Math.abs(main.x - aside.x) > 2))) {
      failures.push(`${scenario.name}: Notifications main/context replacement geometry is missing or reversed`);
    }
    const reads = layout.apiReads ?? [];
    const commands = layout.commands ?? [];
    const inboxReads = reads.filter((read) => read.path === inboxPath);
    const activityReads = reads.filter((read) => read.path === activityPath);
    if (!inboxReads.length || inboxReads.some((read) => {
      const query = new URLSearchParams(read.query);
      return query.get("limit") !== "25" || !["true", "false"].includes(query.get("unread_only"))
        || [...query.keys()].some((key) => !["limit", "unread_only", "cursor"].includes(key));
    }) || commands.some((command) => ![inboxPath + "/sample-notification-1/read", "/api/v1/preferences/notifications"].includes(command.path))) {
      failures.push(`${scenario.name}: Notifications reads or commands escaped their supported actor inbox contract`);
    }
    if (view === "activity" && (!denied && !activityReads.length || activityReads.some((read) => {
      const query = new URLSearchParams(read.query);
      const days = (Date.parse(query.get("to")) - Date.parse(query.get("from"))) / 86400000;
      return query.get("legal_entity_id") !== "bank-ng" || query.get("limit") !== "25" || ![7, 30, 90].includes(days);
    }) || denied && activityReads.length)) failures.push(`${scenario.name}: Activity permission, legal entity or period query is inconsistent`);
    if (view === "inbox" && !["notifications-empty", "notifications-unavailable"].includes(scenario.fixture)
      && (layout.recordCount !== 5 || layout.selectedCount !== 1 || layout.selectedRecord !== firstTitle || layout.detailTitle !== firstTitle
        || layout.unreadHeader !== "Notifications, 7 unread" || !same(layout.readFilter, [{ label: "All", selected: true }, { label: "Unread (7)", selected: false }])
        || layout.searchVisible !== (width > 600) || layout.filtersToggleVisible !== (width <= 600)
        || !Array.isArray(layout.recordHeights) || layout.recordHeights.length !== 5 || layout.recordHeights.some((height) => height < 90))) {
      failures.push(`${scenario.name}: Inbox must distinguish delivered rows, selected context and the server unread total`);
    }
    if (view === "activity" && scenario.fixture === "notifications-loaded" && (layout.recordCount !== 3 || layout.activityEvents?.length !== 3
      || !layout.activityVersionLimitation?.includes("Record versions are not recorded in this history.")
      || layout.activityEvents.some((event, index) => {
        const expected = expectedActivityEvents[index];
        return event.id !== expected.id || !event.action?.includes("Sample") || event.actor !== expected.actor || event.outcome !== expected.outcome
          || !event.text?.includes(expected.record) || !event.text?.includes(expected.source) || !same(event.fields, ["Record"]);
      }))) {
      failures.push(`${scenario.name}: Activity returned record, actor, outcome, source or context-only version limitation is absent`);
    }
    if (view === "preferences" && (!layout.preferences?.criticalRequired || layout.preferences.inAppHeading !== "In-app notifications" || layout.preferences.emailHeading !== "Email delivery")) {
      failures.push(`${scenario.name}: Preferences omitted channel structure or required critical email policy`);
    }
    const interaction = layout.interaction;
    if (scenario.verifyNotificationInteraction) {
      if (!same(interaction?.navigation?.arrowSequence, ["Activity", "Preferences", "Inbox"]) || interaction.navigation.routesMatched !== true
        || interaction.navigation.count !== 3 || interaction.baselineRestored !== true) failures.push(`${scenario.name}: local tabs lack keyboard and route preservation proof`);
      if (view === "inbox" && (interaction?.searchEmptyAndRecovery !== true || interaction.keyboardSelection !== true || interaction.selectedContextWithoutRead !== true
        || (width <= 900 && interaction.backRestoredFocus !== true) || interaction.serverUnreadFilter !== true || interaction.pagedRows !== 10
        || interaction.opaqueCursorPreserved !== true || interaction.refreshedRows !== 5 || interaction.serverUnreadCount !== 7
        || interaction.assignedWorkHandoff !== true || interaction.bellReturned !== true || commands.length)) {
        failures.push(`${scenario.name}: Inbox search, selection, server filter or paging proof is incomplete`);
      }
      if (view === "activity" && (!same(interaction?.periodDays, [30, 7, 30]) || interaction.entityID !== "bank-ng"
        || interaction.fixedPagingWindow !== true || interaction.opaqueCursorPreserved !== true || interaction.pagedEvents !== 5 || interaction.periodResetEvents !== 3)) {
        failures.push(`${scenario.name}: Activity requires a fixed paging window and selected-period reset proof`);
      }
      if (view === "preferences" && !validPreferences(interaction, commands, false)) failures.push(`${scenario.name}: preference edit/save proof is inconsistent with the real versioned command`);
    }
    const check = scenario.notificationCheck;
    if (["empty", "unavailable", "denied", "foreign"].includes(check) && !framed(interaction?.stateCapture, scenario.viewport)) {
      failures.push(`${scenario.name}: the asserted state must be visible inside the captured viewport`);
    }
    if (check === "empty" && (interaction?.empty !== true || interaction.rows !== 0)) failures.push(`${scenario.name}: empty population was not checked`);
    if (check === "unavailable" && (interaction?.unavailable !== true || interaction.retryIssued !== true || interaction.failureRetained !== true || interaction.rows !== 0)) failures.push(`${scenario.name}: unavailable retry must retain an honest failed state`);
    if (check === "denied" && (interaction?.restricted !== true || interaction.protectedRequests !== 0 || interaction.inboxAvailable !== true
      || interaction.permission !== "platform_operations_read")) failures.push(`${scenario.name}: restricted Activity must make no request while the actor inbox remains available`);
    if (check === "foreign" && (interaction?.foreignEntityRejected !== true || interaction.rows !== 0 || interaction.returnedStatus !== 200)) failures.push(`${scenario.name}: foreign-entity Activity rows were not rejected`);
    if (check === "more-failed" && (interaction?.retainedRows !== 5 || interaction.olderPageFailure !== true
      || interaction.cursor !== "sample-notifications/older?batch=2&recipient=sample" || interaction.failureVisible !== true
      || !rectangle(interaction.failureBounds) || interaction.failureBounds.y < 0 || interaction.failureBounds.bottom > scenario.viewport.height
      || interaction.failureBounds.x < 0 || interaction.failureBounds.right > width)) failures.push(`${scenario.name}: older-page failure did not retain delivered rows and cursor or show its failure notice`);
    if (["selected", "unsafe"].includes(check) && (interaction?.keyboardSelected !== true || interaction.detailFocused !== true || interaction.readCommands !== 0
      || commands.length || !interaction.detailBounds || (check === "unsafe" && (interaction.unsafeActionDisabled !== true
        || !framed(interaction.unsafeActionCapture, scenario.viewport))))) failures.push(`${scenario.name}: selected context must receive focus without marking read and show its disabled unsafe handoff`);
    if (["read", "read-failed"].includes(check)) {
      const failed = check === "read-failed";
      const command = interaction?.command;
      if (commands.length !== 1 || !same(commands[0], command) || command?.method !== "POST" || command.path !== `${inboxPath}/sample-notification-1/read`
        || !same(command.body, {}) || command.status !== (failed ? 503 : 200)
        || interaction.readResult !== (failed ? "FAILED" : "CONFIRMED") || interaction.unreadCount !== (failed ? 7 : 6)
        || interaction.retainedSelectedContext !== true || interaction.visibleRows !== (failed ? 3 : 2) || interaction.workCommands !== 0
        || !failed && (command.result?.id !== "sample-notification-1" || !command.result.read_at
          || !same(interaction.recordHandoff, { route: "#work/matters/matter-gaid-change", recordRendered: true, bellReturned: true, duplicateReadCommands: 0 }))) failures.push(`${scenario.name}: read receipt and work-state distinction is not proven`);
    }
    if (check === "preferences-conflict" && !validPreferences(interaction, commands, true)) failures.push(`${scenario.name}: preference conflict must retain the draft and reject the stale version`);
  }
  return { failures, check: { name: "Notifications source layout and supported workflows", status: failures.length ? "FAIL" : "PASS",
    detail: `${notificationsEvidenceScenarios.filter((scenario) => byName.has(scenario.name)).length}/${notificationsEvidenceScenarios.length} captures; three views at five widths in two themes plus independent read-state, paging, permission and preference states` } };
}

function validPreferences(interaction, commands, conflict) {
  const command = interaction?.command;
  const expectedBody = { daily_digest_enabled: true, digest_minute: 585, time_zone: "Africa/Lagos", quiet_hours_enabled: true,
    quiet_start_minute: 1330, quiet_end_minute: 380, expected_version: 3 };
  return interaction?.draftRetained === true && interaction.saved === !conflict && interaction.conflictRetainedDraft === conflict
    && interaction.criticalRequired === true && same(interaction.inputMinutes, [585, 1330, 380])
    && commands.length === 1 && same(commands[0], command) && command.method === "PUT" && command.path === "/api/v1/preferences/notifications"
    && same(command.body, expectedBody) && command.status === (conflict ? 409 : 200)
    && (conflict || command.result.version === 4 && command.result.critical_email_required === true);
}
