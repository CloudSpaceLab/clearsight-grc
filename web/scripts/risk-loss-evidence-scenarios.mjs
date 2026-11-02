// Isolated synthetic UI evidence. These captures do not establish live authority
// execution or whole-screen operational acceptance.
export const riskLossViewports = [
  { width: 1440, height: 900 }, { width: 1280, height: 800 }, { width: 1024, height: 768 },
  { width: 768, height: 900 }, { width: 390, height: 844 },
];
export const riskLossViews = [
  { id: "risks", label: "Risks", heading: "Risks", baselineHeading: "Risks", fixture: "loss-s5", route: "#risks", source: "kri", sourceComparison: "TABLE_ANATOMY_ADAPTATION", population: "Risk register", firstRecord: "Sample network resilience", detailID: "sample-risk-network" },
  { id: "rcsa", label: "RCSA", heading: "Risk & control self-assessments", baselineHeading: "RCSA cycles", fixture: "rcsa-cycles", route: "#rcsa", source: "rcsa", sourceComparison: "SAME_SCREEN", population: "RCSA cycles", firstRecord: "Q3 Technology RCSA", detailID: "cycle-rcsa-1" },
  { id: "insights", label: "Key indicators", heading: "Key indicators", baselineHeading: "Insights", fixture: "indicator-insights", route: "#insights?kind=KRI", source: "kri", sourceComparison: "SAME_SCREEN", population: "Risk indicators", firstRecord: "Mobile transaction success rate", detailID: "sample-mobile-success" },
  { id: "losses", label: "Loss events", heading: "Loss ledger", baselineHeading: "Losses", fixture: "loss-s5", route: "#losses", source: "losses", sourceComparison: "SAME_SCREEN", population: "Operational loss register", firstRecord: "Duplicate merchant settlement", detailID: "loss-s5-1" },
];
const scenario = (view, check, theme, width, code) => ({
  ...view, name: `${code}-risk-loss-${view.id}-${check}-${theme}-${width}`, state: `risk-loss-${view.id}-${check}-${theme}-${width}`,
  riskLossView: view.id, riskLossCheck: check, theme, density: "comfortable", viewport: riskLossViewports.find((item) => item.width === width), touch: width <= 1024,
});
export const riskLossEvidenceScenarios = [
  ...riskLossViews.flatMap((view, index) => ["light", "dark"].flatMap((theme) => riskLossViewports.map((viewport) => ({
    ...scenario(view, "layout", theme, viewport.width, 250 + index), riskLossMatrix: true,
    verifyRiskLossNavigation: view.id === "risks" && (theme === "light" && viewport.width === 1440 || theme === "dark" && viewport.width === 390),
    name: `${250 + index}-risk-loss-${view.id}-${theme}-${viewport.width}x${viewport.height}`,
  })))),
  ...riskLossViews.flatMap((view, index) => [
    scenario(view, "detail", "light", 1440, `254${index}`), scenario(view, "detail", "dark", 390, `255${index}`),
    scenario(view, "history", index % 2 ? "dark" : "light", index % 2 ? 390 : 1440, `256${index}`),
    scenario(view, "empty", "light", 1440, `257${index}`), scenario(view, "denied", "dark", 390, `258${index}`),
    scenario(view, "partial", index % 2 ? "light" : "dark", index % 2 ? 1440 : 390, `259${index}`),
    scenario(view, "filters", "light", 390, `260${index}`),
  ]),
];
export const riskLossAccessibilityScenarios = riskLossViews.flatMap((view) => ["light", "dark"].flatMap((theme) => [1440, 390].map((width) => ({
  ...view, name: `risk-loss-${view.id}-${theme}-${width}`, heading: "Oversight", riskLossView: view.id,
  path: `/?tour=off&fixture=${view.fixture}#oversight`, theme, viewport: { width, height: width === 390 ? 844 : 900 },
  touch: width === 390, scopeSelector: ".risk-loss-workspace",
}))));
