import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import type { DomainMetricBundle, LossPeriodBundle } from "../../metricApi";
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
const domain: DomainMetricBundle = {
  source_id: "risk-1", source_revision: "risk-v1", definition_revision: "domain-v1",
  generated_at: "2026-10-07T12:00:00Z", posture_as_of: "2026-10-07T12:00:00Z",
  scope_id: "entity-1", scope_kind: "LEGAL_ENTITY", items: [],
};
const loss: LossPeriodBundle = {
  source_id: "loss-1", source_revision: "loss-v1", definition_revision: "loss-v1",
  generated_at: "2026-10-07T12:00:00Z", period_start: "2026-09-08T00:00:00Z", period_end: "2026-10-07T12:00:00Z",
  scope_kind: "ORGANIZATION_SCOPE", scope_id: "scope-1", event_count: 0, contributing_loss_count: 0,
  unattributed_event_count: 0, mixed_currencies: false, currencies: [], organization_breakdown: [],
  flow_resolution: "DAY", flow_points: [], comparison: {
    period_start: "2026-08-09T00:00:00Z", period_end: "2026-09-07T23:59:59Z", event_count: 0,
    contributing_loss_count: 0, mixed_currencies: false, currencies: [], event_delta: 0,
    direction: "UNKNOWN", comparison_quality: "MISSING",
  },
};

beforeEach(() => {
  api.loadDomainMetrics.mockReset().mockResolvedValue(domain);
  api.loadLossPeriodMetrics.mockReset().mockResolvedValue(loss);
});

it("loads legal-entity Risk posture and selected-scope Loss flow for the exact interval", async () => {
  render(<RiskLossInsights period={period} organizationScopeID="scope-1" scopeAuthorized/>);
  await waitFor(() => expect(api.loadDomainMetrics).toHaveBeenCalledWith(undefined, expect.any(AbortSignal)));
  expect(api.loadLossPeriodMetrics).toHaveBeenCalledWith(period, "scope-1", expect.any(AbortSignal));
  expect(await screen.findByText(/risk-1 · loss-1/)).toBeTruthy();
  expect(screen.getByText(/Risk posture is current as of its governed source revision/)).toBeTruthy();
  expect(screen.getByText(/Risk posture is for the legal entity/)).toBeTruthy();
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
  api.loadDomainMetrics.mockReturnValueOnce(pending).mockResolvedValue({ ...domain, source_id: "risk-new" });
  api.loadLossPeriodMetrics.mockResolvedValue({ ...loss, scope_id: "scope-new", source_id: "loss-new" });
  const { rerender } = render(<RiskLossInsights period={period} organizationScopeID="scope-old" scopeAuthorized/>);
  rerender(<RiskLossInsights period={period} organizationScopeID="scope-new" scopeAuthorized/>);
  expect(await screen.findByText(/risk-new · loss-new/)).toBeTruthy();
  resolveFirst({ ...domain, source_id: "risk-old" });
  await waitFor(() => expect(screen.getByText(/risk-new · loss-new/)).toBeTruthy());
  expect(screen.queryByText(/risk-old/)).toBeNull();
});
