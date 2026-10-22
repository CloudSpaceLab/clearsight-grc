import { act, renderHook, waitFor } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import type { DomainMetricBundle, LossPeriodBundle } from "../../metricApi";
import { useReportsOverviewMetrics } from "./useReportsOverviewMetrics";

const period = { start_date: "2026-09-08", end_date: "2026-10-07" };
const domain: DomainMetricBundle = {
  generated_at: "2026-10-07T12:00:00Z", posture_as_of: "2026-10-07T12:00:00Z",
  scope_kind: "LEGAL_ENTITY", scope_id: "entity-1", source_id: "risk-1",
  source_revision: "risk-v1", definition_revision: "domain-v1", items: [],
};
const loss: LossPeriodBundle = {
  generated_at: "2026-10-07T12:00:00Z", period_start: "2026-09-08T00:00:00Z",
  period_end: "2026-10-07T12:00:00Z", scope_kind: "LEGAL_ENTITY", scope_id: "entity-1",
  source_id: "loss-1", source_revision: "loss-v1", definition_revision: "loss-v1",
  event_count: 0, contributing_loss_count: 0, unattributed_event_count: 0,
  mixed_currencies: false, currencies: [], organization_breakdown: [], flow_points: [], flow_resolution: "DAY",
  comparison: {
    period_start: "2026-08-09T00:00:00Z", period_end: "2026-09-07T23:59:59Z",
    event_count: 0, contributing_loss_count: 0, mixed_currencies: false, currencies: [],
    event_delta: 0, direction: "UNKNOWN", comparison_quality: "MISSING",
  },
};
const point: LossPeriodBundle["flow_points"][number] = {
  start: "2026-09-08T00:00:00Z", end: "2026-09-08T23:59:59Z",
  loss_event_count: 1, contributing_loss_count: 1, mixed_currencies: false, currencies: [],
};

function options() {
  return {
    metricPeriod: period, metricState: "live" as const, scopeAuthorized: true, legalEntityID: "entity-1",
    loadDomain: vi.fn().mockResolvedValue(domain), loadLoss: vi.fn().mockResolvedValue(loss),
  };
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((resolvePromise) => { resolve = resolvePromise; });
  return { promise, resolve };
}

it("loads the exact legal entity and reporting interval through the existing APIs", async () => {
  const input = options();
  const { result } = renderHook(() => useReportsOverviewMetrics(input));
  await waitFor(() => expect(result.current.lossState).toBe("live"));
  expect(result.current.domain).toEqual(domain);
  expect(result.current.loss).toEqual(loss);
  expect(input.loadDomain).toHaveBeenCalledWith(undefined, expect.any(AbortSignal));
  expect(input.loadLoss).toHaveBeenCalledWith(period, undefined, expect.any(AbortSignal));
});

it("requests and verifies the selected organization area for both risk and loss", async () => {
  const input = options();
  input.loadDomain.mockResolvedValue({ ...domain, scope_kind: "ORGANIZATION_SCOPE", scope_id: "group-1" });
  input.loadLoss.mockResolvedValue({ ...loss, scope_kind: "ORGANIZATION_SCOPE", scope_id: "group-1" });
  const { result } = renderHook(() => useReportsOverviewMetrics({ ...input, organizationScopeID: "group-1" }));
  await waitFor(() => expect(result.current.lossState).toBe("live"));
  expect(result.current.domainState).toBe("live");
  expect(input.loadDomain).toHaveBeenCalledWith("group-1", expect.any(AbortSignal));
  expect(input.loadLoss).toHaveBeenCalledWith(period, "group-1", expect.any(AbortSignal));
});

it.each([
  ["other entity", { ...domain, scope_id: "entity-other" }],
  ["wrong scope kind", { ...domain, scope_kind: "ORGANIZATION_SCOPE" }],
  ["missing response", null],
  ["missing metric items", { ...domain, items: undefined }],
])("rejects a risk response with %s without suppressing valid losses", async (_name, value) => {
  const input = options();
  input.loadDomain.mockResolvedValue(value);
  const { result } = renderHook(() => useReportsOverviewMetrics(input));
  await waitFor(() => expect(result.current.domainState).toBe("unavailable"));
  expect(result.current.domain).toBeNull();
  expect(result.current.lossState).toBe("live");
});

it.each([
  ["other entity", { ...loss, scope_id: "entity-other" }],
  ["wrong scope kind", { ...loss, scope_kind: "ORGANIZATION_SCOPE" }],
  ["other reporting start", { ...loss, period_start: "2026-08-01T00:00:00Z" }],
  ["other reporting end", { ...loss, period_end: "2026-10-06T12:00:00Z" }],
  ["malformed reporting period", { ...loss, period_start: "2026-09-08-invalid" }],
  ["missing response", undefined],
  ["missing flow points", { ...loss, flow_points: undefined }],
  ["missing comparison", { ...loss, comparison: undefined }],
  ["null flow point", { ...loss, flow_points: [null] }],
  ["invalid flow date", { ...loss, flow_points: [{ ...point, start: "invalid" }] }],
  ["missing flow count", { ...loss, flow_points: [{ ...point, loss_event_count: undefined }] }],
  ["null currency", { ...loss, mixed_currencies: true, currencies: [null] }],
  ["missing currency label", { ...loss, mixed_currencies: true, currencies: [{}] }],
  ["empty comparison", { ...loss, comparison: {} }],
  ["invalid comparison quality", { ...loss, comparison: { ...loss.comparison, comparison_quality: "OTHER" } }],
  ["invalid comparison direction", { ...loss, comparison: { ...loss.comparison, direction: "OTHER" } }],
  ["missing comparison delta", { ...loss, comparison: { ...loss.comparison, event_delta: undefined } }],
  ["malformed net amount", { ...loss, net_loss: { minor_units: "not-money", currency: "NGN" } }],
  ["missing net currency", { ...loss, net_loss: { minor_units: "100" } }],
  ["missing money flow", { ...loss, net_loss: { minor_units: "100", currency: "NGN" }, flow_points: [point] }],
  ["mismatched money flow", { ...loss, net_loss: { minor_units: "100", currency: "NGN" },
    flow_points: [{ ...point, net_loss: { minor_units: "100", currency: "USD" } }] }],
  ["malformed comparison amount", { ...loss, comparison: { ...loss.comparison, net_delta: { minor_units: "100" } } }],
])("rejects a loss response with %s without suppressing valid risks", async (_name, value) => {
  const input = options();
  input.loadLoss.mockResolvedValue(value);
  const { result } = renderHook(() => useReportsOverviewMetrics(input));
  await waitFor(() => expect(result.current.lossState).toBe("unavailable"));
  expect(result.current.loss).toBeNull();
  expect(result.current.domainState).toBe("live");
});

it("preserves exact signed amounts and explicit zero-flow buckets in a valid money series", async () => {
  const input = options();
  const money = { minor_units: "-900719925474099300", currency: "NGN" };
  const value = { ...loss, net_loss: money, flow_points: [
    { ...point, net_loss: money },
    { ...point, start: "2026-09-09T00:00:00Z", loss_event_count: 0, net_loss: { ...money, minor_units: "0" } },
  ], comparison: { ...loss.comparison, comparison_quality: "COMPLETE", direction: "IMPROVED", net_delta: money } };
  input.loadLoss.mockResolvedValue(value);
  const { result } = renderHook(() => useReportsOverviewMetrics(input));

  await waitFor(() => expect(result.current.lossState).toBe("live"));
  expect(result.current.loss).toEqual(value);
});

it("rejects a different group and an entity-wide fallback under a selected group", async () => {
  const input = options();
  input.loadDomain.mockResolvedValue({ ...domain, scope_kind: "ORGANIZATION_SCOPE", scope_id: "group-other" });
  const { result } = renderHook(() => useReportsOverviewMetrics({ ...input, organizationScopeID: "group-1" }));
  await waitFor(() => expect(result.current.domainState).toBe("unavailable"));
  expect(result.current.lossState).toBe("unavailable");
  expect(result.current.domain).toBeNull();
  expect(result.current.loss).toBeNull();
});

it.each([
  { scopeAuthorized: false },
  { legalEntityID: undefined },
  { metricState: "unavailable" as const },
  { metricPeriod: undefined },
  { metricPeriod: { start_date: "2026-02-31", end_date: "2026-10-07" } },
  { metricPeriod: { start_date: "2026-10-08", end_date: "2026-10-07" } },
])("does not query data without an authorized scope and valid reporting basis: %j", (change) => {
  const input = options();
  const { result } = renderHook(() => useReportsOverviewMetrics({ ...input, ...change }));
  expect(input.loadDomain).not.toHaveBeenCalled();
  expect(input.loadLoss).not.toHaveBeenCalled();
  expect(result.current.domainState).toBe("unavailable");
  expect(result.current.lossState).toBe("unavailable");
});

it("clears both sources before rendering a new selection and ignores late aborted responses", async () => {
  const input = options();
  const oldDomain = deferred<DomainMetricBundle>();
  const oldLoss = deferred<LossPeriodBundle>();
  input.loadDomain.mockReturnValueOnce(oldDomain.promise).mockResolvedValue({ ...domain, scope_id: "entity-2", source_id: "risk-new" });
  input.loadLoss.mockReturnValueOnce(oldLoss.promise).mockResolvedValue({ ...loss, scope_id: "entity-2", source_id: "loss-new" });
  const renders: Array<{ entity: string; risk: string | undefined; loss: string | undefined }> = [];
  const { result, rerender } = renderHook(({ entity }) => {
    const value = useReportsOverviewMetrics({ ...input, legalEntityID: entity });
    renders.push({ entity, risk: value.domain?.source_id, loss: value.loss?.source_id });
    return value;
  }, { initialProps: { entity: "entity-1" } });
  const signal = input.loadDomain.mock.calls[0]?.[1] as AbortSignal;
  rerender({ entity: "entity-2" });
  expect(signal.aborted).toBe(true);
  await waitFor(() => expect(result.current.loss?.source_id).toBe("loss-new"));
  await act(async () => { oldDomain.resolve(domain); oldLoss.resolve(loss); });
  expect(result.current.domain?.source_id).toBe("risk-new");
  expect(result.current.loss?.source_id).toBe("loss-new");
  expect(renders.filter((item) => item.entity === "entity-2").every((item) => item.risk !== "risk-1" && item.loss !== "loss-1")).toBe(true);
});

it("refreshes both sources and immediately removes previously loaded results", async () => {
  const input = options();
  const nextDomain = deferred<DomainMetricBundle>();
  const nextLoss = deferred<LossPeriodBundle>();
  input.loadDomain.mockResolvedValueOnce(domain).mockReturnValue(nextDomain.promise);
  input.loadLoss.mockResolvedValueOnce(loss).mockReturnValue(nextLoss.promise);
  const { result, rerender } = renderHook(({ refreshKey }) => useReportsOverviewMetrics({ ...input, refreshKey }), {
    initialProps: { refreshKey: 0 },
  });
  await waitFor(() => expect(result.current.lossState).toBe("live"));
  rerender({ refreshKey: 1 });
  expect(input.loadDomain).toHaveBeenCalledTimes(2);
  expect(input.loadLoss).toHaveBeenCalledTimes(2);
  expect(result.current.domain).toBeNull();
  expect(result.current.loss).toBeNull();
  expect(result.current.domainState).toBe("loading");
  await act(async () => {
    nextDomain.resolve({ ...domain, source_id: "risk-refreshed" });
    nextLoss.resolve({ ...loss, source_id: "loss-refreshed" });
  });
  expect(result.current.domain?.source_id).toBe("risk-refreshed");
  expect(result.current.loss?.source_id).toBe("loss-refreshed");
});

it("removes previously loaded data when scope authorization is withdrawn", async () => {
  const input = options();
  const { result, rerender } = renderHook(({ authorized }) => useReportsOverviewMetrics({ ...input, scopeAuthorized: authorized }), {
    initialProps: { authorized: true },
  });
  await waitFor(() => expect(result.current.lossState).toBe("live"));
  rerender({ authorized: false });
  expect(result.current.domain).toBeNull();
  expect(result.current.loss).toBeNull();
  expect(result.current.domainState).toBe("unavailable");
  expect(result.current.lossState).toBe("unavailable");
  expect(input.loadDomain).toHaveBeenCalledTimes(1);
  expect(input.loadLoss).toHaveBeenCalledTimes(1);
});

it("keeps the successful source available when the other request fails", async () => {
  const input = options();
  input.loadDomain.mockRejectedValue(new Error("Risk service unavailable"));
  const { result } = renderHook(() => useReportsOverviewMetrics(input));
  await waitFor(() => expect(result.current.lossState).toBe("live"));
  expect(result.current.domainState).toBe("unavailable");
  expect(result.current.loss).toEqual(loss);
});
