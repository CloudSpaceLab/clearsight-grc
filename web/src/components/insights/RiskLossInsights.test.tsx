import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { RiskLossInsights } from "./RiskLossInsights";

const api = vi.hoisted(() => ({ loadDomainMetrics: vi.fn(), loadLossPeriodMetrics: vi.fn() }));
vi.mock("../../metricApi", () => api);
vi.mock("../oversight/DomainPostureSummary", () => ({
  DomainPostureSummary: ({ bundle, lossBundle }: { bundle?: { source_id: string }; lossBundle?: { source_id: string } }) =>
    <div aria-label="Governed metrics">{bundle?.source_id ?? "Risk loading"} · {lossBundle?.source_id ?? "Loss loading"}</div>,
}));
vi.mock("../oversight/OrganizationRiskSummary", () => ({ OrganizationRiskSummary: () => null }));
vi.mock("../oversight/RiskMovement", () => ({ RiskMovement: () => null }));
vi.mock("../oversight/OrganizationLossSummary", () => ({ OrganizationLossSummary: () => null }));
vi.mock("../oversight/LossMovement", () => ({ LossMovement: () => null }));

const period = { start_date: "2026-09-08", end_date: "2026-10-07" };
const loss = { source_id: "loss-1", period_start: "2026-09-08T00:00:00Z", period_end: "2026-10-07T12:00:00Z", scope_kind: "ORGANIZATION_SCOPE", scope_id: "scope-1" };

beforeEach(() => {
  api.loadDomainMetrics.mockReset().mockResolvedValue({ source_id: "risk-1", items: [] });
  api.loadLossPeriodMetrics.mockReset().mockResolvedValue(loss);
});

it("loads authoritative Risk and Loss reads with the same selected scope and exact Loss interval", async () => {
  render(<RiskLossInsights period={period} organizationScopeID="scope-1" scopeAuthorized/>);
  await waitFor(() => expect(api.loadDomainMetrics).toHaveBeenCalledWith("scope-1", expect.any(AbortSignal)));
  expect(api.loadLossPeriodMetrics).toHaveBeenCalledWith(period, "scope-1", expect.any(AbortSignal));
  expect(await screen.findByText(/risk-1 · loss-1/)).toBeTruthy();
  expect(screen.getByText(/Risk posture is current as of its governed source revision/)).toBeTruthy();
});

it("does not query data when the bookmarked organization scope is not authorized", async () => {
  render(<RiskLossInsights period={period} organizationScopeID="scope-restricted" scopeAuthorized={false}/>);
  expect(screen.getByText(/not active or authorized/)).toBeTruthy();
  expect(api.loadDomainMetrics).not.toHaveBeenCalled();
  expect(api.loadLossPeriodMetrics).not.toHaveBeenCalled();
});

it("rejects Loss data from a different reporting period instead of showing mismatched amounts", async () => {
  api.loadLossPeriodMetrics.mockResolvedValueOnce({ ...loss, period_start: "2026-07-01T00:00:00Z" });
  render(<RiskLossInsights period={period} organizationScopeID="scope-1" scopeAuthorized/>);
  expect(await screen.findByText(/risk-1 · Loss loading/)).toBeTruthy();
  expect(screen.queryByText(/risk-1 · loss-1/)).toBeNull();
});

it("does not render a prior scoped response after the selected department changes", async () => {
  let resolveFirst!: (value: unknown) => void;
  const pending = new Promise((resolve) => { resolveFirst = resolve; });
  api.loadDomainMetrics.mockReturnValueOnce(pending).mockResolvedValue({ source_id: "risk-new", items: [] });
  api.loadLossPeriodMetrics.mockResolvedValue({ ...loss, scope_id: "scope-new", source_id: "loss-new" });
  const { rerender } = render(<RiskLossInsights period={period} organizationScopeID="scope-old" scopeAuthorized/>);
  rerender(<RiskLossInsights period={period} organizationScopeID="scope-new" scopeAuthorized/>);
  expect(await screen.findByText(/risk-new · loss-new/)).toBeTruthy();
  resolveFirst({ source_id: "risk-old", items: [] });
  await waitFor(() => expect(screen.getByText(/risk-new · loss-new/)).toBeTruthy());
  expect(screen.queryByText(/risk-old/)).toBeNull();
});
