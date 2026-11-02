// Relationship detail is a bounded source page. All records remain explicitly
// synthetic UI fixtures; these captures do not prove backend authorization.
export const vendorRelationshipViewports = [
  { width: 1440, height: 900 }, { width: 1280, height: 800 }, { width: 1024, height: 768 },
  { width: 768, height: 900 }, { width: 390, height: 844 },
];
export const vendorRelationshipViews = [
  { id: "overview", label: "Overview", source: "summary" },
  { id: "due-diligence", label: "Due diligence", source: "diligence" },
  { id: "evidence", label: "Evidence", source: "evidence" },
  { id: "issues", label: "Issues & actions", source: "actions" },
  { id: "activity", label: "Activity", source: "history" },
];
const base = { route: "#vendors/register", title: "Vendors", density: "comfortable", fixture: "vendor-form-assessment" };
export const vendorRelationshipEvidenceScenarios = [
  ...vendorRelationshipViews.flatMap((view, index) => ["light", "dark"].flatMap((theme) => vendorRelationshipViewports.map((viewport) => ({
    ...base, name: `${230 + index}-vendor-relationship-${view.id}-${theme}-${viewport.width}x${viewport.height}`,
    state: `vendor-relationship-${view.id}-${theme}-${viewport.width}`, vendorRelationshipView: view.id,
    theme, viewport, touch: viewport.width <= 1024, vendorRelationshipMatrix: true,
    verifyVendorRelationshipNavigation: view.id === "overview" && theme === "light" && [1440, 390].includes(viewport.width),
  })))),
  ...[
    { code: "235", view: "overview", check: "loading", theme: "dark", width: 390 },
    { code: "235a", view: "overview", check: "unavailable", theme: "light", width: 1440 },
    { code: "235b", view: "overview", check: "long-labels", theme: "light", width: 390 },
    { code: "235c", view: "evidence", check: "documents", theme: "dark", width: 390 },
    { code: "235d", view: "due-diligence", check: "activation-denied", theme: "light", width: 1440 },
    { code: "235e", view: "due-diligence", check: "assessment-unavailable", theme: "dark", width: 390 },
    { code: "235f", view: "overview", check: "empty", theme: "light", width: 1440 },
    { code: "235g", view: "overview", check: "partial", theme: "dark", width: 390 },
  ].map(({ code, view, check, width, ...rest }) => ({ ...base, ...rest,
    name: `${code}-vendor-relationship-${check}-${rest.theme}-${width}`, state: `vendor-relationship-${check}`,
    vendorRelationshipView: view, vendorRelationshipCheck: check,
    viewport: { width, height: width === 390 ? 844 : 900 }, touch: width === 390,
  })),
];

export const vendorRelationshipAccessibilityScenarios = vendorRelationshipViews.flatMap((view) =>
  ["light", "dark"].flatMap((theme) => [1440, 390].map((width) => ({
    name: `vendor-relationship-${view.id}-${theme}-${width}`, heading: "Vendors", theme,
    path: "/?tour=off&fixture=vendor-form-assessment#vendors/register", fixture: "vendor-form-assessment",
    viewport: { width, height: width === 390 ? 844 : 900 }, touch: width === 390,
    vendorRelationshipView: view.id, scopeSelector: ".vendor-relationship-workspace",
  }))),
);
