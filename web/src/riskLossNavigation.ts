// Shared route order for the primary menu's risk disclosure and the local workspace menu.
export const riskLossDestinations = [
  { id: "risks", label: "Risks", view: "risks" },
  { id: "rcsa", label: "RCSA", view: "rcsa" },
  { id: "indicators", label: "Key indicators", view: "insights" },
  { id: "losses", label: "Loss events", view: "losses" },
] as const;

export type RiskLossSection = (typeof riskLossDestinations)[number]["view"];
