import type { View } from "./appRouting";

export type RiskLossSection = Extract<View, "risks" | "rcsa" | "insights" | "losses">;

// The shell and workspace menu share the same supported destinations and order.
export const riskLossDestinations = [
  { id: "risks", label: "Risks", view: "risks" },
  { id: "rcsa", label: "RCSA", view: "rcsa" },
  { id: "indicators", label: "Key indicators", view: "insights" },
  { id: "losses", label: "Loss events", view: "losses" },
] as const satisfies readonly { id: string; label: string; view: RiskLossSection }[];

export const riskLossSections = riskLossDestinations.map(({ view, label }) => ({ id: view, label }));
