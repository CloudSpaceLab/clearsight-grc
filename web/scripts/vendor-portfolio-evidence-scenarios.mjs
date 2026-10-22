// This is composition evidence using synthetic, isolated UI fixture reads.
// It does not establish live backend authorization or screen acceptance.
export const vendorPortfolioViewports = [
  { width: 1440, height: 900 }, { width: 1280, height: 800 }, { width: 1024, height: 768 },
  { width: 768, height: 900 }, { width: 390, height: 844 },
];
export const vendorPortfolioViews = [
  { id: "overview", label: "Overview" }, { id: "register", label: "Register" },
  { id: "onboarding", label: "Onboarding" }, { id: "assessments", label: "Assessments" },
  { id: "documents", label: "Documents" },
];
const base = { title: "Vendors", density: "comfortable", fixture: "vendor-form-assessment" };
export const vendorPortfolioEvidenceScenarios = [
  ...vendorPortfolioViews.flatMap((view, index) => ["light", "dark"].flatMap((theme) => vendorPortfolioViewports.map((viewport) => ({
    ...base, name: `${240 + index}-vendor-portfolio-${view.id}-${theme}-${viewport.width}x${viewport.height}`,
    state: `vendor-portfolio-${view.id}-${theme}-${viewport.width}`, route: `#vendors/${view.id}`,
    vendorPortfolioView: view.id, vendorPortfolioMatrix: true, theme, viewport, touch: viewport.width <= 1024,
    verifyVendorPortfolioNavigation: view.id === "overview" && theme === "light" && [1440, 390].includes(viewport.width),
    verifyVendorPortfolioFilters: view.id === "register" && theme === "light" && viewport.width === 390,
  })))),
  ...[
    { code: "245", view: "overview", check: "loading", theme: "dark", width: 390 },
    { code: "245a", view: "overview", check: "unavailable", theme: "light", width: 1440 },
    { code: "245b", view: "register", check: "empty", theme: "light", width: 1440 },
    { code: "245c", view: "register", check: "partial", theme: "dark", width: 390 },
    { code: "245d", view: "assessments", check: "summaries-unavailable", theme: "dark", width: 390 },
    { code: "245e", view: "register", check: "search-paging", theme: "light", width: 1440 },
    { code: "245f", view: "documents", check: "long-labels", theme: "light", width: 390 },
    { code: "245g", view: "documents", check: "documents-drill", theme: "dark", width: 390 },
    { code: "245h", view: "onboarding", check: "onboarding-drill", theme: "light", width: 1440 },
    { code: "245i", view: "assessments", check: "assessments-drill", theme: "dark", width: 390 },
    { code: "245j", view: "overview", check: "denied", theme: "light", width: 390 },
    { code: "245k", view: "overview", check: "exceptions-unavailable", theme: "dark", width: 1440 },
    { code: "245l", view: "assessments", check: "summary-missing", theme: "light", width: 1440 },
  ].map(({ code, view, check, width, ...rest }) => ({ ...base, ...rest,
    name: `${code}-vendor-portfolio-${check}-${rest.theme}-${width}`, state: `vendor-portfolio-${check}`,
    route: `#vendors/${view}`, vendorPortfolioView: view, vendorPortfolioCheck: check,
    viewport: { width, height: width === 390 ? 844 : 900 }, touch: width === 390,
  })),
];
export const vendorPortfolioAccessibilityScenarios = vendorPortfolioViews.flatMap((view) =>
  ["light", "dark"].flatMap((theme) => [1440, 390].map((width) => ({
    name: `vendor-portfolio-${view.id}-${theme}-${width}`, heading: "Vendors", theme,
    path: `/?tour=off&fixture=vendor-form-assessment#vendors/${view.id}`, fixture: "vendor-form-assessment",
    viewport: { width, height: width === 390 ? 844 : 900 }, touch: width === 390,
    vendorPortfolioView: view.id, scopeSelector: ".vendor-portfolio-workspace",
  }))),
);
