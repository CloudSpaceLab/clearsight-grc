export const privacySourcePath = "clearsight-privacy-data-designs/privacy-workspace-preview.html";
export const privacySourceSHA256 = "4baf53940862d9a3af1fbc1be32da7b27595a22efb0646759cb91685a0cb5e8d";
export const privacyActivityID = "ropa-activity-customer-account-opening";
export const privacyActivityName = "Customer account opening";
export const privacyViewports = [
  { width: 1440, height: 900 }, { width: 1280, height: 800 }, { width: 1024, height: 768 },
  { width: 768, height: 900 }, { width: 390, height: 844 },
];
export const privacyViews = [
  { id: "overview", label: "Overview", route: "#ropa/overview", source: "overview" },
  { id: "register", label: "ROPA", route: "#ropa", source: "ropa" },
];
const make = (view, check, theme, width, fixture = "current") => ({
  name: `270-privacy-${view}-${check}-${theme}-${width}`, state: `privacy-${view}-${check}-${theme}-${width}`,
  privacyView: view, privacyCheck: check, route: privacyViews.find((item) => item.id === view).route,
  theme, density: "comfortable", viewport: privacyViewports.find((item) => item.width === width), touch: width <= 1024,
  fixture: `privacy-v2-${fixture}`,
});
export const privacyEvidenceScenarios = [
  ...privacyViews.flatMap((view) => ["light", "dark"].flatMap((theme) => privacyViewports.map(({ width }) => ({ ...make(view.id, "layout", theme, width), privacyMatrix: true })))),
  ...["light", "dark"].flatMap((theme) => privacyViewports.map(({ width }) => make("register", "detail", theme, width))),
  ...[1440, 390].flatMap((width) => [
    make("register", "history", width === 390 ? "dark" : "light", width),
    make("register", "filters", width === 390 ? "dark" : "light", width, "paged"),
  ]),
  ...privacyViews.flatMap((view) => [
    make(view.id, "empty", "light", 390, "empty"),
    make(view.id, "denied", "dark", 390, "denied"),
  ]),
  make("overview", "stale", "dark", 1440, "stale"),
  make("overview", "partial", "light", 390, "partial"),
  make("overview", "unknown", "dark", 390, "unknown"),
  make("register", "recovery", "light", 1440, "recovery"),
  make("register", "page-empty", "light", 1440, "paged-empty"),
  make("register", "page-error", "dark", 390, "paged-error"),
  make("register", "not-found", "dark", 390),
];
export const privacyAccessibilityScenarios = privacyViews.flatMap((view) => ["light", "dark"].flatMap((theme) => [1440, 390].map((width) => ({
  ...make(view.id, "layout", theme, width), name: `privacy-${view.id}-${theme}-${width}`,
  path: `/?tour=off&fixture=privacy-v2-current${view.route}`, heading: "Privacy & data", scopeSelector: ".privacy-workspace",
}))));
export const privacySupplementalScreenshotNames = privacyEvidenceScenarios
  .filter((scenario) => ["page-empty", "page-error"].includes(scenario.privacyCheck)).map((scenario) => `${scenario.name}-page.png`);
