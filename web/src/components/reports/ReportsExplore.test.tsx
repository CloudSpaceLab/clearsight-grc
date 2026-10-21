import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import type { DomainMetricBundle, HomeMetricMemberPage, LossCurrencyFlow, LossPeriodBundle, OrganizationMetricBreakdown } from "../../metricApi";
import { ReportsExplore, type ReportsExploreProps } from "./ReportsExplore";

const period = { start_date: "2026-09-08", end_date: "2026-10-07" };
const domain: DomainMetricBundle = {
  generated_at: "2026-10-07T12:00:00Z", posture_as_of: "2026-10-07T12:00:00Z",
  scope_kind: "LEGAL_ENTITY", scope_id: "entity-1", source_id: "risk-source-1",
  source_revision: "risk-v1", definition_revision: "domain-v1", items: [{
    id: "risks_outside_appetite", label: "Outside appetite", value: 4, unit: "COUNT", condition: "ATTENTION",
    freshness: "CURRENT", completeness: "COMPLETE", population: 12, unknown: 0, excluded: 0,
    generated_at: "2026-10-07T12:00:00Z", source_revision: "risk-v1", definition_revision: "domain-v1",
    basis: "CURRENT_POSTURE", drill: { workspace: "risks", filter: "outside-appetite", consistency: "SOURCE_SNAPSHOT" },
  }],
};
const breakdown: OrganizationMetricBreakdown = {
  source_id: domain.source_id, metric_id: "risks_outside_appetite", definition_revision: "domain-v1", count: 4,
  items: [
    { key: "scope:operations", scope_id: "operations", label: "Operations", kind: "ORGANIZATION_SCOPE", value: 3 },
    { key: "unattributed", label: "Unassigned area", kind: "UNATTRIBUTED", value: 1 },
  ],
};
function flow(currency: string, amount: string, events = 1): LossCurrencyFlow {
  const money = (minor: string) => ({ minor_units: minor, currency });
  return { currency, gross: money(amount), recovery: money("0"), reversal: money("0"), net: money(amount),
    loss_event_count: events, recovery_event_count: 0, reversal_event_count: 0 };
}
const ngn = flow("NGN", "1040000000");
const loss: LossPeriodBundle = {
  generated_at: "2026-10-07T12:00:00Z", period_start: "2026-09-08T00:00:00Z", period_end: "2026-10-07T12:00:00Z",
  scope_kind: "LEGAL_ENTITY", scope_id: "entity-1", source_id: "loss-source-1", source_revision: "loss-v1", definition_revision: "loss-v1",
  event_count: 1, contributing_loss_count: 1, unattributed_event_count: 0, mixed_currencies: false,
  net_loss: ngn.net, currencies: [ngn], flow_resolution: "DAY", flow_points: [],
  organization_breakdown: [{ key: "operations", scope_id: "operations", label: "Operations", kind: "ORGANIZATION_SCOPE",
    loss_event_count: 1, contributing_loss_count: 1, mixed_currencies: false, net_loss: ngn.net, currencies: [ngn] }],
  comparison: { period_start: "2026-08-09T00:00:00Z", period_end: "2026-09-07T23:59:59Z", event_count: 0,
    contributing_loss_count: 0, mixed_currencies: false, currencies: [], event_delta: 1, direction: "UNKNOWN", comparison_quality: "MISSING" },
};
const members: HomeMetricMemberPage = {
  source_id: domain.source_id, metric_id: "risks_outside_appetite", definition_revision: "domain-v1", count: 4,
  items: [{ member_id: "member-1", target_type: "RISK", target_id: "risk-1", target_title: "Settlement risk", state: "OUTSIDE_APPETITE", accessible: true }],
};
function props() {
  return { period, periodState: "live" as const, scopeAuthorized: true, legalEntityID: "entity-1",
    loadDomain: vi.fn().mockResolvedValue(domain), loadLoss: vi.fn().mockResolvedValue(loss),
    loadBreakdown: vi.fn().mockResolvedValue(breakdown), loadMembers: vi.fn().mockResolvedValue(members),
    loadLossMembers: vi.fn().mockResolvedValue({ ...members, source_id: loss.source_id, definition_revision: loss.definition_revision,
      metric_id: "operational_loss_net", count: 1, items: [{ ...members.items[0], target_type: "LOSS", target_id: "loss-1", target_title: "Settlement loss" }] }),
    onOpenRisk: vi.fn(), onOpenLoss: vi.fn(), onOpenInsights: vi.fn() };
}
async function choose(field: string, label: string) {
  fireEvent.click(screen.getByRole("button", { name: new RegExp(`${field}$`) }));
  fireEvent.click(await screen.findByRole("option", { name: label }));
}
function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => { resolve = done; });
  return { promise, resolve };
}

it("renders one source-bound risk chart and matching exact rows with the source Data rail", async () => {
  const input = props();
  render(<ReportsExplore {...input}/>);
  const chart = await screen.findByRole("list", { name: "Risks outside appetite by organization area" });
  const table = screen.getByRole("table", { name: "Explore measure breakdown" });
  expect(within(chart).getByRole("group", { name: "Operations: 3" })).toBeTruthy();
  expect(within(table).getByRole("row", { name: "Operations, 3" })).toBeTruthy();
  expect(within(table).getByRole("cell", { name: "Context: No organization area recorded" })).toBeTruthy();
  expect(input.loadBreakdown).toHaveBeenCalledWith("risks_outside_appetite", domain.source_id, "domain-v1", undefined, expect.any(AbortSignal));
  expect(screen.getByRole("button", { name: /Breakdown$/ })).toHaveProperty("disabled", true);
  expect(screen.getByRole("button", { name: /Comparison$/ })).toHaveProperty("disabled", true);
  expect(screen.getByText(/reporting period does not change it/)).toBeTruthy();
  expect(screen.getByText(/12 risks in scope · 0 unknown · 0 excluded/)).toBeTruthy();
  expect(screen.getByText(/Source revision risk-v1 · Definition domain-v1/)).toBeTruthy();
  expect(screen.queryByRole("dialog")).toBeNull();
  expect(screen.queryByRole("button", { name: /Save view|Add to report/ })).toBeNull();
  fireEvent.click(screen.getByRole("button", { name: "Open Insights workspace" }));
  expect(input.onOpenInsights).toHaveBeenCalledOnce();
});

it("changes the selected measure to exact loss amounts and then loss events", async () => {
  render(<ReportsExplore {...props()}/>);
  await screen.findByRole("list", { name: "Risks outside appetite by organization area" });
  await choose("Measure", "Net operational loss");
  const amounts = await screen.findByRole("table", { name: "Explore measure breakdown" });
  expect(within(amounts).getByRole("cell", { name: /^Net operational loss:/ }).textContent?.replace(/\D/g, "")).toBe("1040000000");
  expect(screen.getByRole("list", { name: "Net operational loss by organization area" })).toBeTruthy();
  expect(screen.queryByRole("list", { name: "Risks outside appetite by organization area" })).toBeNull();
  await choose("Measure", "Loss events");
  expect(within(screen.getByRole("table", { name: "Explore measure breakdown" })).getByRole("cell", { name: "Loss events: 1" })).toBeTruthy();
  expect(screen.getByText(/2026-09-08 to 2026-10-07/)).toBeTruthy();
});

it("associates fixed-field explanations without dimming them with the disabled controls", () => {
  render(<ReportsExplore {...props()} scopeAuthorized={false}/>);
  for (const [name, description] of [
    ["Breakdown", "Measures are available by organization area."],
    ["Comparison", "Previous-period values by organization area are unavailable."],
  ] as const) {
    const group = screen.getByRole("group", { name, description });
    expect(within(group).getByRole("button")).toHaveProperty("disabled", true);
    const explanation = within(group).getByText(description);
    expect(explanation.closest(".cs-select-field[data-disabled]")).toBeNull();
  }
});

it("keeps mixed currencies separate and exact signed money beyond Number precision", async () => {
  const input = props();
  const large = flow("NGN", "900719925474099301");
  const recovery = flow("USD", "-450", 0);
  const mixed = { ...loss, mixed_currencies: true, net_loss: undefined, currencies: [large, recovery],
    organization_breakdown: [{ ...loss.organization_breakdown[0]!, mixed_currencies: true, net_loss: undefined, currencies: [large, recovery] }] };
  input.loadLoss.mockResolvedValue(mixed);
  const { rerender } = render(<ReportsExplore {...input}/>);
  await screen.findByRole("list", { name: "Risks outside appetite by organization area" });
  await choose("Measure", "Net operational loss");
  let cell = within(screen.getByRole("table", { name: "Explore measure breakdown" })).getByRole("cell", { name: /^Net operational loss:/ });
  expect(cell.textContent?.replace(/\D/g, "")).toBe("900719925474099301");
  await choose("Currency", "USD");
  cell = within(screen.getByRole("table", { name: "Explore measure breakdown" })).getByRole("cell", { name: /^Net operational loss:/ });
  expect(cell.textContent).toContain("-");
  expect(cell.textContent?.replace(/\D/g, "")).toBe("450");
  expect(screen.getByText(/Showing USD only/)).toBeTruthy();
  expect(screen.getByText(/Negative amounts represent net recoveries/)).toBeTruthy();
  rerender(<ReportsExplore {...input} periodState="loading"/>);
  expect(screen.queryByRole("table", { name: "Explore measure breakdown" })).toBeNull();
  expect(screen.getByRole("button", { name: /Measure$/ }).textContent).toContain("Net operational loss");
  rerender(<ReportsExplore {...input} periodState="unavailable"/>);
  expect(screen.queryByText(/Showing USD only/)).toBeNull();
  rerender(<ReportsExplore {...input} refreshKey={1}/>);
  await screen.findByText(/Showing USD only/);
  expect(screen.getByRole("button", { name: /Currency$/ }).textContent).toContain("USD");
});

it.each([
  ["source", { ...breakdown, source_id: "other-source" }],
  ["definition", { ...breakdown, definition_revision: "other-definition" }],
  ["scope", { ...breakdown, scope_id: "other-scope" }],
  ["total", { ...breakdown, items: [breakdown.items[0]!] }],
  ["malformed row", { ...breakdown, items: [null] }],
])("rejects a risk breakdown with mismatched %s", async (_reason, value) => {
  const input = props();
  input.loadBreakdown.mockResolvedValue(value);
  render(<ReportsExplore {...input}/>);
  expect(await screen.findByText(/measure is unavailable for this scope/)).toBeTruthy();
  expect(screen.queryByRole("list", { name: "Risks outside appetite by organization area" })).toBeNull();
  expect(screen.queryByRole("table", { name: "Explore measure breakdown" })).toBeNull();
  await choose("Measure", "Loss events");
  expect(await screen.findByRole("list", { name: "Loss events by organization area" })).toBeTruthy();
});

it("ignores an aborted area's slow breakdown after scope changes", async () => {
  const input = props();
  const old = deferred<OrganizationMetricBreakdown>();
  input.loadDomain.mockImplementation((scope?: string) => Promise.resolve({ ...domain, scope_kind: "ORGANIZATION_SCOPE", scope_id: scope, source_id: `risk-${scope}` }));
  input.loadLoss.mockImplementation((_period, scope?: string) => Promise.resolve({ ...loss, scope_kind: "ORGANIZATION_SCOPE", scope_id: scope }));
  input.loadBreakdown.mockReturnValueOnce(old.promise).mockResolvedValue({ ...breakdown, source_id: "risk-new", scope_id: "new", items: [{ ...breakdown.items[0]!, label: "New area", value: 4 }] });
  const { rerender } = render(<ReportsExplore {...input} organizationScopeID="old"/>);
  await waitFor(() => expect(input.loadBreakdown).toHaveBeenCalledOnce());
  const signal = input.loadBreakdown.mock.calls[0]![4] as AbortSignal;
  rerender(<ReportsExplore {...input} organizationScopeID="new"/>);
  expect(signal.aborted).toBe(true);
  expect(await screen.findByRole("row", { name: "New area, 4" })).toBeTruthy();
  await act(async () => old.resolve({ ...breakdown, source_id: "risk-old", scope_id: "old" }));
  expect(screen.queryByRole("row", { name: "Operations, 3" })).toBeNull();
  expect(input.loadDomain).toHaveBeenLastCalledWith("new", expect.any(AbortSignal));
});

it("removes loaded protected values immediately when authorization is withdrawn", async () => {
  const input = props();
  const { rerender } = render(<ReportsExplore {...input}/>);
  await screen.findByRole("row", { name: "Operations, 3" });
  fireEvent.click(screen.getByRole("button", { name: "Review underlying records" }));
  await screen.findByText("Settlement risk");
  rerender(<ReportsExplore {...input} scopeAuthorized={false}/>);
  expect(screen.queryByRole("table", { name: "Explore measure breakdown" })).toBeNull();
  expect(screen.queryByText("Settlement risk")).toBeNull();
  expect(screen.queryByRole("dialog")).toBeNull();
  expect(screen.getByText(/scope is not authorized/)).toBeTruthy();
  expect(input.loadDomain).toHaveBeenCalledOnce();
  expect(input.loadLoss).toHaveBeenCalledOnce();
});

it.each([
  { completeness: "PARTIAL", freshness: "CURRENT", unknown: 2, population: 12 },
  { completeness: "UNKNOWN", freshness: "STALE", unknown: undefined, population: 0 },
])("never turns zero with incomplete assessment coverage into an appetite conclusion: %j", async (quality) => {
  const input = props();
  input.loadDomain.mockResolvedValue({ ...domain, items: [{ ...domain.items[0]!, ...quality, value: 0 }] });
  input.loadBreakdown.mockResolvedValue({ ...breakdown, count: 0, items: [] });
  render(<ReportsExplore {...input}/>);
  await screen.findByText(/Current, complete assessment coverage is not confirmed/);
  expect(screen.queryByRole("heading", { name: "No risks outside appetite" })).toBeNull();
  if (quality.population === 0) {
    expect(screen.getByRole("heading", { name: "No assessed risk population" })).toBeTruthy();
    expect(screen.getByText("Unassessed")).toBeTruthy();
    expect(screen.getByText("Source stale")).toBeTruthy();
    expect(screen.getByText(/Unknown count unavailable/)).toBeTruthy();
    expect(input.loadBreakdown).not.toHaveBeenCalled();
  } else {
    expect(screen.getByText(/12 risks in scope · 2 unknown · 0 excluded/)).toBeTruthy();
    expect(screen.getByText(/Risk coverage is partial/)).toBeTruthy();
    expect(screen.queryByText(/\b12 assessed\b/)).toBeNull();
  }
});

it("keeps valid zero activity distinct from unavailable monetary data", async () => {
  const input = props();
  input.loadLoss.mockResolvedValue({ ...loss, event_count: 0, contributing_loss_count: 0, net_loss: undefined,
    currencies: [], organization_breakdown: [] });
  render(<ReportsExplore {...input}/>);
  await screen.findByRole("list", { name: "Risks outside appetite by organization area" });
  await choose("Measure", "Loss events");
  expect(screen.getByRole("heading", { name: "No new loss events" })).toBeTruthy();
  await choose("Measure", "Net operational loss");
  expect(screen.getByRole("heading", { name: "No monetary loss flow" })).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Review underlying records" })).toBeNull();
});

it.each([
  { ...loss, organization_breakdown: [null] },
  { ...loss, organization_breakdown: [{ ...loss.organization_breakdown[0]!, currencies: [null] }] },
  { ...loss, organization_breakdown: [{ ...loss.organization_breakdown[0]!, currencies: [flow("NGN", "103")] }] },
  { ...loss, currencies: [], organization_breakdown: [{ ...loss.organization_breakdown[0]!, currencies: [] }] },
  { ...loss, net_loss: { minor_units: "0", currency: "NGN" }, currencies: [flow("NGN", "0")],
    organization_breakdown: [{ ...loss.organization_breakdown[0]!, net_loss: undefined, currencies: [] }] },
])("does not render malformed or unreconciled monetary rows", async (value) => {
  const input = props();
  input.loadLoss.mockResolvedValue(value);
  render(<ReportsExplore {...input}/>);
  await screen.findByRole("list", { name: "Risks outside appetite by organization area" });
  await choose("Measure", "Net operational loss");
  expect(await screen.findByText(/measure is unavailable for this scope/)).toBeTruthy();
  expect(screen.queryByRole("table", { name: "Explore measure breakdown" })).toBeNull();
});

it("shows a verified zero net amount while retaining its contributing loss records", async () => {
  const input = props();
  const zero = flow("NGN", "0");
  input.loadLoss.mockResolvedValue({ ...loss, net_loss: zero.net, currencies: [zero],
    organization_breakdown: [{ ...loss.organization_breakdown[0]!, net_loss: zero.net, currencies: [zero] }] });
  render(<ReportsExplore {...input}/>);
  await screen.findByRole("list", { name: "Risks outside appetite by organization area" });
  await choose("Measure", "Net operational loss");
  const cell = within(screen.getByRole("table", { name: "Explore measure breakdown" })).getByRole("cell", { name: /^Net operational loss:/ });
  expect(cell.textContent?.replace(/\D/g, "")).toBe("000");
  expect(screen.queryByRole("heading", { name: "No monetary loss flow" })).toBeNull();
  expect(screen.getByRole("button", { name: "Review underlying records" })).toHaveProperty("disabled", false);
});

it("reviews the retained source members, pages safely, and opens an accessible risk", async () => {
  const input = props();
  input.loadMembers.mockResolvedValueOnce({ ...members, next_cursor: "page-2" }).mockResolvedValue({ ...members, items: [
    { ...members.items[0]!, member_id: "member-2", target_id: "risk-2", target_title: "Liquidity risk" },
  ] });
  render(<ReportsExplore {...input}/>);
  await screen.findByRole("list", { name: "Risks outside appetite by organization area" });
  fireEvent.click(screen.getByRole("button", { name: "Review underlying records" }));
  const sheet = await screen.findByRole("dialog", { name: "Risks outside appetite records" });
  await within(sheet).findByText("Settlement risk");
  expect(input.loadMembers).toHaveBeenCalledWith("risks_outside_appetite", domain.source_id, "domain-v1", undefined, undefined, 50, expect.any(AbortSignal));
  fireEvent.click(within(sheet).getByRole("button", { name: "Next page" }));
  expect(within(sheet).queryByText("Settlement risk")).toBeNull();
  await within(sheet).findByText("Liquidity risk");
  expect(input.loadMembers).toHaveBeenLastCalledWith("risks_outside_appetite", domain.source_id, "domain-v1", undefined, "page-2", 50, expect.any(AbortSignal));
  fireEvent.click(within(sheet).getByRole("button", { name: /Open record for Liquidity risk/ }));
  expect(input.onOpenRisk).toHaveBeenCalledWith("risk-2");
});

it("rejects a different retained member source and retries the same snapshot", async () => {
  const input = props();
  input.loadMembers.mockResolvedValueOnce({ ...members, source_id: "different-source" }).mockResolvedValue(members);
  render(<ReportsExplore {...input}/>);
  await screen.findByRole("list", { name: "Risks outside appetite by organization area" });
  fireEvent.click(screen.getByRole("button", { name: "Review underlying records" }));
  const sheet = await screen.findByRole("dialog", { name: "Risks outside appetite records" });
  expect(await within(sheet).findByText("Records for this metric snapshot are unavailable.")).toBeTruthy();
  expect(within(sheet).queryByText("Settlement risk")).toBeNull();
  fireEvent.click(within(sheet).getByRole("button", { name: "Try again" }));
  await within(sheet).findByText("Settlement risk");
  expect(input.loadMembers).toHaveBeenCalledTimes(2);
});

it("uses retained loss contributors for the net-loss drill and opens their Loss record", async () => {
  const input = props();
  render(<ReportsExplore {...input}/>);
  await screen.findByRole("list", { name: "Risks outside appetite by organization area" });
  await choose("Measure", "Net operational loss");
  fireEvent.click(screen.getByRole("button", { name: "Review underlying records" }));
  const sheet = await screen.findByRole("dialog", { name: "Net operational loss records" });
  await within(sheet).findByText("Settlement loss");
  expect(input.loadLossMembers).toHaveBeenCalledWith("operational_loss_net", loss.source_id, "loss-v1", undefined, undefined, 50, expect.any(AbortSignal));
  fireEvent.click(within(sheet).getByRole("button", { name: /Open record for Settlement loss/ }));
  expect(input.onOpenLoss).toHaveBeenCalledWith("loss-1");
});

it("does not request metrics for an unverified scope or unavailable reporting period", () => {
  const input = props();
  const { rerender } = render(<ReportsExplore {...input} scopeAuthorized={false}/>);
  expect(input.loadDomain).not.toHaveBeenCalled();
  expect(input.loadLoss).not.toHaveBeenCalled();
  rerender(<ReportsExplore {...input} periodState="unavailable"/>);
  expect(input.loadDomain).not.toHaveBeenCalled();
  expect(input.loadLoss).not.toHaveBeenCalled();
});

it("retains the last measure when a parent hides and reveals the mounted analysis", async () => {
  const input: ReportsExploreProps = props();
  const { rerender } = render(<div hidden={false}><ReportsExplore {...input}/></div>);
  await screen.findByRole("list", { name: "Risks outside appetite by organization area" });
  await choose("Measure", "Loss events");
  rerender(<div hidden><ReportsExplore {...input}/></div>);
  rerender(<div hidden={false}><ReportsExplore {...input}/></div>);
  expect(screen.getByRole("button", { name: /Measure$/ }).textContent).toContain("Loss events");
  expect(screen.getByRole("list", { name: "Loss events by organization area" })).toBeTruthy();
});
