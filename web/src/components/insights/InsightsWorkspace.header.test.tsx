import { render, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { InsightsWorkspace } from "./InsightsWorkspace";

vi.mock("../../indicatorApi", () => ({
  loadIndicatorPopulation: vi.fn().mockResolvedValue({ items: [], complete: true }),
}));

vi.mock("./RiskLossInsights", () => ({
  RiskLossInsights: () => <section aria-label="Risk and Loss insights">Scoped comparison</section>,
}));

const actions = {
  onKindChange: vi.fn(),
  onTarget: vi.fn(),
};

it("uses the shared Insights header for current authorized indicators", async () => {
  render(<InsightsWorkspace
    {...actions}
    kind="KRI"
    organizationName="Clear Bank"
    legalEntityName="Clear Bank Nigeria"
  />);

  const heading = screen.getByRole("heading", { name: "Insights", level: 1 });
  expect(heading.closest("header")?.className).toContain("cs-register-header");
  expect(screen.getByText("Current indicators for Clear Bank Nigeria.")).toBeTruthy();
  expect(await screen.findByText("No indicators in this scope")).toBeTruthy();
});

it("keeps the same shared header in a selected-period Risk and Loss view", () => {
  render(<InsightsWorkspace
    {...actions}
    kind="KRI"
    view="risk-loss"
    riskLossPeriod={{ start_date: "2026-09-01", end_date: "2026-10-01" }}
    organizationScopeID="scope-1"
    organizationScopeName="Payments"
    scopeAuthorized
  />);

  const heading = screen.getByRole("heading", { name: "Insights", level: 1 });
  expect(heading.closest("header")?.className).toContain("cs-register-header");
  expect(screen.getByText("Risk position and selected-period losses for Payments.")).toBeTruthy();
  expect(screen.getByRole("region", { name: "Risk and Loss insights" })).toBeTruthy();
});
