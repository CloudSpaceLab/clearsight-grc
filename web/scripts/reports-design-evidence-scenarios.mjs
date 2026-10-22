// Source-matched Reports views. These descriptors identify evidence states;
// geometry and data assertions remain in the browser and manifest reviewers.
export const reportsDesignViewports = [
  { width: 1440, height: 900 },
  { width: 1280, height: 800 },
  { width: 1024, height: 768 },
  { width: 768, height: 900 },
  { width: 390, height: 844 },
];

const base = { route: "#reports", title: "Reports & insights", density: "comfortable" };

export const reportsDesignEvidenceScenarios = [
  ...["explore", "builder"].flatMap((view) => ["light", "dark"].flatMap((theme) =>
    reportsDesignViewports.map((viewport) => ({
      ...base,
      name: `${view === "explore" ? "222" : "223"}-report-${view}-${theme}-${viewport.width}x${viewport.height}`,
      state: `report-${view}-${theme}-${viewport.width}`,
      theme, viewport, touch: viewport.width <= 1024,
      fixture: view === "explore" ? "report-overview-loaded" : "report-definitions",
      reportDesignMatrix: true,
      ...(view === "explore" ? { openReportExplore: true } : { openReportBuilder: true }),
      verifyReportNavigation: theme === "light" && [1440, 390].includes(viewport.width),
    })))),
  ...[
    { name: "224-report-explore-net-loss-light-1440x900", state: "report-explore-net-loss", theme: "light", viewport: { width: 1440, height: 900 }, reportExploreMeasure: "Net operational loss" },
    { name: "224a-report-explore-loss-events-dark-mobile-390x844", state: "report-explore-loss-events", theme: "dark", viewport: { width: 390, height: 844 }, reportExploreMeasure: "Loss events" },
    { name: "224b-report-explore-selected-area-light-1440x900", state: "report-explore-selected-area", theme: "light", viewport: { width: 1440, height: 900 }, reportOrganizationScope: "sample-scope-risk-ops" },
    { name: "224c-report-explore-partial-dark-1440x900", state: "report-explore-partial", fixture: "report-overview-partial", theme: "dark", viewport: { width: 1440, height: 900 } },
    { name: "224d-report-explore-unavailable-light-1440x900", state: "report-explore-unavailable", fixture: "report-definitions", theme: "light", viewport: { width: 1440, height: 900 } },
    { name: "224e-report-explore-denied-dark-mobile-390x844", state: "report-explore-denied", fixture: "report-overview-denied", theme: "dark", viewport: { width: 390, height: 844 } },
    { name: "224f-report-explore-loss-unavailable-dark-mobile-390x844", state: "report-explore-loss-unavailable", fixture: "report-overview-partial", theme: "dark", viewport: { width: 390, height: 844 }, reportExploreMeasure: "Net operational loss" },
    { name: "224g-report-explore-empty-light-1440x900", state: "report-explore-empty", fixture: "report-explore-empty", theme: "light", viewport: { width: 1440, height: 900 } },
    { name: "224h-report-explore-empty-loss-events-dark-390x844", state: "report-explore-empty-loss-events", fixture: "report-explore-empty", theme: "dark", viewport: { width: 390, height: 844 }, reportExploreMeasure: "Loss events" },
    { name: "224i-report-explore-empty-net-loss-light-390x844", state: "report-explore-empty-net-loss", fixture: "report-explore-empty", theme: "light", viewport: { width: 390, height: 844 }, reportExploreMeasure: "Net operational loss" },
    { name: "224j-report-explore-mixed-ngn-light-1440x900", state: "report-explore-mixed-ngn", fixture: "report-explore-mixed-currency", theme: "light", viewport: { width: 1440, height: 900 }, reportExploreMeasure: "Net operational loss", reportExploreCurrency: "NGN", verifyReportCurrency: true },
    { name: "224k-report-explore-mixed-usd-dark-390x844", state: "report-explore-mixed-usd", fixture: "report-explore-mixed-currency", theme: "dark", viewport: { width: 390, height: 844 }, reportExploreMeasure: "Net operational loss", reportExploreCurrency: "USD" },
    { name: "224l-report-explore-members-light-1440x900", state: "report-explore-members", theme: "light", viewport: { width: 1440, height: 900 }, openReportExploreMembers: true },
    { name: "224m-report-explore-mixed-ngn-dark-390x844", state: "report-explore-mixed-ngn-mobile", fixture: "report-explore-mixed-currency", theme: "dark", viewport: { width: 390, height: 844 }, reportExploreMeasure: "Net operational loss", reportExploreCurrency: "NGN", verifyReportCurrency: true },
  ].map((scenario) => ({ ...base, fixture: "report-overview-loaded", openReportExplore: true,
    touch: scenario.viewport.width <= 1024, ...scenario })),
  { ...base, name: "225-report-builder-flow-light-1440x900", state: "report-builder-flow", fixture: "report-builder-flow",
    theme: "light", viewport: { width: 1440, height: 900 }, openReportBuilder: true, reportBuilderFlow: true },
  { ...base, name: "225a-report-builder-blocked-dark-390x844", state: "report-builder-blocked", fixture: "report-builder-blocked",
    theme: "dark", viewport: { width: 390, height: 844 }, touch: true, openReportBuilder: true, reportBuilderFlow: true },
];

export const reportsDesignAccessibilityScenarios = ["explore", "builder", "explore-members"].flatMap((view) =>
  ["light", "dark"].flatMap((theme) => [1440, 390].map((width) => ({
    name: `reports-${view}-${theme}-${width}`, heading: "Reports & insights", theme,
    path: `/?tour=off&fixture=${view === "builder" ? "report-definitions" : "report-overview-loaded"}#reports`,
    viewport: { width, height: width === 390 ? 844 : 900 }, touch: width === 390,
    reportView: view, scopeSelector: view === "explore-members" ? '[role="dialog"]' : ".reports-workspace",
  }))),
);
