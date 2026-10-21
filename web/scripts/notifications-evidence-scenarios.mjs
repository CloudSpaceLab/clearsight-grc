// Same-screen source pairs use All inbox, Activity and Preferences. Fixture values
// are synthetic; successful commands here do not prove backend authorization.
export const notificationViewports = [
  { width: 1440, height: 900 }, { width: 1280, height: 800 }, { width: 1024, height: 768 },
  { width: 768, height: 900 }, { width: 390, height: 844 },
];
export const notificationRoute = (view) => view === "inbox" ? "#notifications" : `#notifications/${view}`;
const base = { title: "Notifications", density: "comfortable", fixture: "notifications-loaded" };
export const notificationsEvidenceScenarios = [
  ...["inbox", "activity", "preferences"].flatMap((view, index) => ["light", "dark"].flatMap((theme) => notificationViewports.map((viewport) => ({
    ...base, name: `${226 + index}-notifications-${view}-${theme}-${viewport.width}x${viewport.height}`,
    state: `notifications-${view}-${theme}-${viewport.width}`, route: notificationRoute(view), notificationView: view,
    theme, viewport, touch: viewport.width <= 1024, notificationMatrix: true,
    verifyNotificationInteraction: theme === "light" && [1440, 390].includes(viewport.width),
  })))),
  ...[
    { code: "229", view: "inbox", fixture: "notifications-empty", check: "empty", theme: "light", width: 1440 },
    { code: "229a", view: "inbox", fixture: "notifications-unavailable", check: "unavailable", theme: "dark", width: 390 },
    { code: "229b", view: "inbox", fixture: "notifications-read-failed", check: "read-failed", theme: "dark", width: 390 },
    { code: "229c", view: "inbox", fixture: "notifications-more-unavailable", check: "more-failed", theme: "light", width: 1440 },
    { code: "229d", view: "inbox", check: "selected", theme: "dark", width: 390 },
    { code: "229e", view: "inbox", check: "read", theme: "light", width: 1440 },
    { code: "229f", view: "preferences", fixture: "notifications-preferences-conflict", check: "preferences-conflict", theme: "dark", width: 390 },
    { code: "229g", view: "activity", fixture: "notifications-activity-denied", check: "denied", theme: "light", width: 1440 },
    { code: "229h", view: "activity", fixture: "notifications-activity-unavailable", check: "unavailable", theme: "dark", width: 390 },
    { code: "229i", view: "activity", fixture: "notifications-activity-foreign", check: "foreign", theme: "light", width: 1440 },
    { code: "229j", view: "activity", fixture: "notifications-empty", check: "empty", theme: "light", width: 390 },
    { code: "229k", view: "inbox", check: "unsafe", theme: "light", width: 390 },
  ].map(({ code, view, width, check, ...rest }) => ({ ...base, ...rest,
    name: `${code}-notifications-${view}-${check}-${rest.theme}-${width}`, state: `notifications-${view}-${check}`,
    route: notificationRoute(view), notificationView: view, notificationCheck: check,
    viewport: { width, height: width === 390 ? 844 : 900 }, touch: width === 390,
  })),
];

export const notificationsAccessibilityScenarios = ["inbox", "activity", "preferences"].flatMap((view) =>
  ["light", "dark"].flatMap((theme) => [1440, 390].map((width) => ({
    name: `notifications-${view}-${theme}-${width}`, heading: "Notifications", theme,
    path: `/?tour=off&fixture=notifications-loaded${notificationRoute(view)}`,
    viewport: { width, height: width === 390 ? 844 : 900 }, touch: width === 390,
    notificationView: view, scopeSelector: ".notifications-workspace",
  }))),
);
