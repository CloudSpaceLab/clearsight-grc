export const peopleAccessSourcePath = "clearsight-people-access-designs/people-access-preview.html";
export const peopleAccessSourceSHA256 = "3300853e7959e06f76e52b484e77c71c6f0c2af876351861921e0dead0aaa7f4";
export const peopleAccessViewports = [{ width: 1440, height: 900 }, { width: 1280, height: 800 }, { width: 1024, height: 768 }, { width: 768, height: 900 }, { width: 390, height: 844 }];
export const peopleAccessViews = [
  { id: "people", label: "People", route: "#configure/access", source: "people" },
  { id: "roles", label: "Roles", route: "#configure/access/roles", source: "roles" },
  { id: "routes", label: "Approval routes", route: "#configure/authority", source: "routes" },
  { id: "delegations", label: "Delegations", route: "#configure/authority/delegations", source: "delegations" },
];
const make = (view, check, theme, width, fixture = "current") => ({ name: `280-people-access-${view}-${check}-${theme}-${width}`, state: `people-access-${view}-${check}-${theme}-${width}`,
  peopleAccessView: view, peopleAccessCheck: check, route: peopleAccessViews.find((item) => item.id === view).route, theme, density: "comfortable",
  viewport: peopleAccessViewports.find((item) => item.width === width), touch: width <= 1024, fixture: `people-v2-${fixture}` });
export const peopleAccessEvidenceScenarios = [
  ...peopleAccessViews.flatMap((view) => ["light", "dark"].flatMap((theme) => peopleAccessViewports.map(({ width }) => ({ ...make(view.id, "layout", theme, width), peopleAccessMatrix: true })))),
  ...peopleAccessViews.flatMap((view) => ["light", "dark"].flatMap((theme) => [1024, 768, 390].map((width) => make(view.id, "context", theme, width)))),
  ...[1440, 390].flatMap((width) => [make("people", "history", width === 390 ? "dark" : "light", width), make("people", "filters", width === 390 ? "dark" : "light", width)]),
  ...peopleAccessViews.flatMap((view) => [make(view.id, "empty", "light", 390, "empty"), make(view.id, "denied", "dark", 390, "denied")]),
  make("people", "limited", "light", 390, "limited"), make("people", "recovery", "light", 1440, "recovery"),
  make("people", "readonly", "dark", 390, "readonly"), make("routes", "readonly", "light", 1440, "readonly"),
  make("routes", "partial", "light", 390, "partial"), make("delegations", "partial", "dark", 390, "partial"),
  ...peopleAccessViews.map((view, index) => make(view.id, "missing", index % 2 ? "light" : "dark", 390)),
  ...["routes", "delegations"].flatMap((view) => [make(view, "composer", "light", 1440), make(view, "composer", "dark", 390)]),
  make("routes", "definition", "light", 1440, "policy-review"), make("routes", "definition", "dark", 390, "policy-review"),
];
export const peopleAccessAccessibilityScenarios = [...peopleAccessViews.flatMap((view) => ["light", "dark"].flatMap((theme) => [1440, 390].map((width) => ({
  ...make(view.id, "layout", theme, width), name: `people-access-${view.id}-${theme}-${width}`, path: `/?tour=off&fixture=people-v2-current${view.route}`, heading: "People & access", scopeSelector: ".people-access-workspace",
})))), ...peopleAccessEvidenceScenarios.filter((item) => ["composer", "definition"].includes(item.peopleAccessCheck)).map((scenario) => ({ ...scenario, name: `people-access-${scenario.peopleAccessView}-${scenario.peopleAccessCheck}-${scenario.theme}-${scenario.viewport.width}`, path: `/?tour=off&fixture=${scenario.fixture}${scenario.route}`, heading: "People & access", scopeSelector: scenario.peopleAccessCheck === "composer" ? '.governance-composer-sheet [role="dialog"]' : ".people-access-workspace" }))];
export const peopleAccessSupplementalScreenshotNames = peopleAccessEvidenceScenarios.filter((scenario) => scenario.viewport.width === 390 && ["composer", "definition"].includes(scenario.peopleAccessCheck)).map((scenario) => `${scenario.name}-continuation.png`);
