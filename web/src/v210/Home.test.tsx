import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { Home } from "./Home";
const work = vi.hoisted(() => ({ counts: vi.fn() }));
vi.mock("../api", () => ({ loadActorWorkDueCounts: work.counts }));
const sources = vi.hoisted(() => ({ metrics: vi.fn(), oversight: vi.fn(), indicators: vi.fn() }));
vi.mock("../metricApi", () => ({ loadHomeMetrics: sources.metrics }));
vi.mock("../oversightApi", () => ({ loadOversight: sources.oversight }));
vi.mock("../indicatorApi", () => ({ loadIndicatorPopulation: sources.indicators }));
const context = { tenant: { id: "bank", name: "Bank" }, legal_entity: { id: "entity", name: "Bank" }, actor: { id: "person", name: "Ada" }, mode: "live" };
function payload(start = "2026-10-22") {
  const reporting_period = { start_date: start, end_date: "2026-11-20", mode: "CURRENT_WINDOW", max_days: 365, historical_end_supported: false };
  return { metrics: { scope_kind: "LEGAL_ENTITY", scope_id: "entity", reporting_period, source_revision: "r1", posture_as_of: "2026-11-20T00:00:00Z", items: [] }, oversight: { reporting_period, projection_version: "r1", interventions: [] } };
}
beforeEach(() => { vi.resetAllMocks(); work.counts.mockRejectedValue(new Error("unavailable")); const data = payload(); sources.metrics.mockResolvedValue(data.metrics); sources.oversight.mockResolvedValue(data.oversight); sources.indicators.mockResolvedValue({ items: [], complete: true }); });
it("applies the selected period to both authorized sources", async () => {
  render(<Home context={context}/>);
  const start = await screen.findByLabelText("From");
  fireEvent.change(start, { target: { value: "2026-10-01" } });
  const data = payload("2026-10-01"); sources.metrics.mockResolvedValue(data.metrics); sources.oversight.mockResolvedValue(data.oversight);
  fireEvent.click(screen.getByRole("button", { name: "Apply dates" }));
  await waitFor(() => expect(sources.metrics).toHaveBeenLastCalledWith({ start_date: "2026-10-01", end_date: "2026-11-20" }));
  expect(sources.oversight).toHaveBeenLastCalledWith({ start_date: "2026-10-01", end_date: "2026-11-20" });
  expect(await screen.findByText("Current posture is unaffected by the date range.")).toBeTruthy();
});
it("does not display metrics when source revisions differ", async () => {
  const data = payload(); sources.oversight.mockResolvedValue({ ...data.oversight, projection_version: "different" });
  render(<Home context={context}/>);
  expect(await screen.findByText("Home data unavailable.")).toBeTruthy();
  expect(screen.queryByLabelText("From")).toBeNull();
});
it("treats missing period limits as unavailable", async () => {
  const data = payload(); sources.metrics.mockResolvedValue({ ...data.metrics, reporting_period: { ...data.metrics.reporting_period, max_days: undefined } });
  render(<Home context={context}/>);
  expect(await screen.findByText("Home data unavailable.")).toBeTruthy();
});
it("can reset the date range after a failed period query", async () => {
  render(<Home context={context}/>);
  fireEvent.change(await screen.findByLabelText("From"), { target: { value: "2026-10-01" } });
  sources.metrics.mockRejectedValueOnce(new Error("unavailable"));
  fireEvent.click(screen.getByRole("button", { name: "Apply dates" }));
  fireEvent.click(await screen.findByRole("button", { name: "Reset dates" }));
  await waitFor(() => expect(sources.metrics).toHaveBeenLastCalledWith(undefined));
  expect(await screen.findByLabelText("From")).toBeTruthy();
});
it("shows current indicators with recorded state without hiding Home when they are unavailable", async () => {
  sources.indicators.mockResolvedValue({ items: [{ indicator: { check_id: "check-1", check_name: "Service uptime", state: "BREACH", reason: "Below target" } }], complete: false, next_cursor: "more" });
  render(<Home context={context}/>);
  expect(await screen.findByText("Service uptime")).toBeTruthy();
  expect(screen.getByText("Breach")).toBeTruthy();
  expect(screen.getByText("Below target")).toBeTruthy();
});
it("keeps the overview available when indicator access fails", async () => {
  sources.indicators.mockRejectedValue(new Error("denied"));
  render(<Home context={context}/>);
  expect(await screen.findByText("Indicators unavailable.")).toBeTruthy();
  expect(screen.getByRole("heading", { name: "Needs attention" })).toBeTruthy();
});

it("shows verified assignments for the current person independently of bank posture", async () => {
  work.counts.mockResolvedValue({ scope: { tenant_id: "bank", legal_entity_id: "entity", principal_id: "person" }, coverage: "WORKFLOW_ASSIGNMENTS_ONLY", count_basis: "SOURCE_SNAPSHOT", as_of: "2026-11-20T10:00:00Z", total: 5, overdue: 1, due_soon: 2, undated: 1 });
  render(<Home context={context}/>);
  expect(await screen.findByRole("heading", { name: "My assignments" })).toBeTruthy();
  expect(screen.getByText("5 open")).toBeTruthy();
  expect(screen.getByText("1 overdue")).toBeTruthy();
  expect(screen.getByText("2 due within 7 days")).toBeTruthy();
  expect(screen.getByRole("link", { name: "View my work" }).getAttribute("href")).toBe("#/work");
});

it("does not show assignment counts for another person", async () => {
  work.counts.mockResolvedValue({ scope: { tenant_id: "bank", legal_entity_id: "entity", principal_id: "other" }, coverage: "WORKFLOW_ASSIGNMENTS_ONLY", count_basis: "SOURCE_SNAPSHOT", as_of: "2026-11-20T10:00:00Z", total: 5, overdue: 1, due_soon: 2, undated: 1 });
  render(<Home context={context}/>);
  expect(await screen.findByText("Assignments unavailable.")).toBeTruthy();
  expect(screen.queryByText("5 open")).toBeNull();
});


it("keeps the person's verified assignments visible when bank overview data is unavailable", async () => {
  work.counts.mockResolvedValue({ scope: { tenant_id: "bank", legal_entity_id: "entity", principal_id: "person" }, coverage: "WORKFLOW_ASSIGNMENTS_ONLY", count_basis: "SOURCE_SNAPSHOT", as_of: "2026-11-20T10:00:00Z", total: 1, overdue: 0, due_soon: 1, undated: 0 });
  sources.metrics.mockRejectedValue(new Error("unavailable"));
  render(<Home context={context}/>);
  expect(await screen.findByText("Home data unavailable.")).toBeTruthy();
  expect(await screen.findByText("1 open")).toBeTruthy();
});

it("never exposes a previous person's counts after the actor changes", async () => {
  let resolveOld!: (value: unknown) => void;
  work.counts.mockReturnValueOnce(new Promise((resolve) => { resolveOld = resolve; })).mockResolvedValueOnce({ scope: { tenant_id: "bank", legal_entity_id: "entity", principal_id: "other" }, coverage: "WORKFLOW_ASSIGNMENTS_ONLY", count_basis: "SOURCE_SNAPSHOT", as_of: "2026-11-20T10:00:00Z", total: 1, overdue: 0, due_soon: 1, undated: 0 });
  const { rerender } = render(<Home context={context}/>);
  rerender(<Home context={{ ...context, actor: { id: "other", name: "Tola" } }}/>);
  expect(await screen.findByText("1 open")).toBeTruthy();
  resolveOld({ scope: { tenant_id: "bank", legal_entity_id: "entity", principal_id: "person" }, coverage: "WORKFLOW_ASSIGNMENTS_ONLY", count_basis: "SOURCE_SNAPSHOT", as_of: "2026-11-20T10:00:00Z", total: 9, overdue: 2, due_soon: 1, undated: 0 });
  await waitFor(() => expect(screen.queryByText("9 open")).toBeNull());
});

it("removes prior overview and indicators immediately when the actor changes", async () => {
  const first = payload();
  sources.oversight.mockResolvedValueOnce({ ...first.oversight, interventions: [{ target_type: "MATTER", target_id: "one", title: "Private issue", owner_name: "Ada", state: "OPEN", priority: 4 }] }).mockReturnValueOnce(new Promise(() => {}));
  sources.indicators.mockResolvedValueOnce({ items: [{ indicator: { check_id: "one", check_name: "Private indicator", state: "WATCH", reason: "Review" } }], complete: true }).mockReturnValueOnce(new Promise(() => {}));
  const { rerender } = render(<Home context={context}/>);
  expect(await screen.findByText("Private issue")).toBeTruthy();
  expect(await screen.findByText("Private indicator")).toBeTruthy();
  rerender(<Home context={{ ...context, actor: { id: "other", name: "Tola" } }}/>);
  expect(screen.queryByText("Private issue")).toBeNull();
  expect(screen.queryByText("Private indicator")).toBeNull();
});

it("rejects impossible assignment snapshot dates", async () => {
  work.counts.mockResolvedValue({ scope: { tenant_id: "bank", legal_entity_id: "entity", principal_id: "person" }, coverage: "WORKFLOW_ASSIGNMENTS_ONLY", count_basis: "SOURCE_SNAPSHOT", as_of: "2026-02-31T10:00:00Z", total: 5, overdue: 1, due_soon: 2, undated: 1 });
  render(<Home context={context}/>);
  expect(await screen.findByText("Assignments unavailable.")).toBeTruthy();
  expect(screen.queryByText("5 open")).toBeNull();
});

it("prioritizes oversight on the administrative dashboard", async () => {
  work.counts.mockResolvedValue({ scope: { tenant_id: "bank", legal_entity_id: "entity", principal_id: "person" }, coverage: "WORKFLOW_ASSIGNMENTS_ONLY", count_basis: "SOURCE_SNAPSHOT", as_of: "2026-11-20T10:00:00Z", total: 5, overdue: 1, due_soon: 2, undated: 1 });
  const data = payload(); sources.metrics.mockResolvedValue({ ...data.metrics, items: [{ id: "overdue", label: "Bank overdue", value: 3, freshness: "CURRENT", completeness: "COMPLETE" }] });
  render(<Home context={{ ...context, actor: { ...context.actor, role_codes: ["SYSTEM_ADMIN"] } }}/>);
  expect(await screen.findByRole("heading", { name: "Dashboard" })).toBeTruthy();
  const metric = await screen.findByText("Bank overdue");
  const personal = await screen.findByRole("heading", { name: "My assignments" });
  expect(metric.compareDocumentPosition(personal) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
});

it("shows the oversight population and its recorded exclusions without inventing coverage", async () => {
  const data = payload();
  sources.oversight.mockResolvedValue({ ...data.oversight, coverage: { population: 10, excluded: 1, unknown: 2 }, freshness: "CURRENT", generated_at: "2026-11-20T10:00:00Z" });
  render(<Home context={{ ...context, actor: { ...context.actor, role_codes: ["SYSTEM_ADMIN"] } }}/>);
  expect(await screen.findByText("7 of 10 issues included")).toBeTruthy();
  expect(screen.getByText("1 excluded · 2 unknown")).toBeTruthy();
  expect(screen.getByText(/As of .*UTC/)).toBeTruthy();
});

it("identifies stale or unknown oversight coverage freshness", async () => {
  const data = payload();
  sources.oversight.mockResolvedValue({ ...data.oversight, coverage: { population: 2, excluded: 0, unknown: 1 }, freshness: "STALE", generated_at: "2026-11-20T10:00:00Z" });
  render(<Home context={{ ...context, actor: { ...context.actor, role_codes: ["SYSTEM_ADMIN"] } }}/>);
  expect(await screen.findByText(/Outdated/)).toBeTruthy();
});

it("withholds coverage when its population or exclusions are inconsistent", async () => {
  const data = payload();
  sources.oversight.mockResolvedValue({ ...data.oversight, coverage: { population: 2, excluded: 2, unknown: 1 }, freshness: "CURRENT" });
  render(<Home context={{ ...context, actor: { ...context.actor, role_codes: ["SYSTEM_ADMIN"] } }}/>);
  expect(await screen.findByText("Coverage unavailable.")).toBeTruthy();
  expect(screen.queryByText(/of 2 issues included/)).toBeNull();
});

it("does not invent a percentage for an empty oversight population", async () => {
  const data = payload();
  sources.oversight.mockResolvedValue({ ...data.oversight, coverage: { population: 0, excluded: 0, unknown: 0 }, freshness: "CURRENT" });
  render(<Home context={{ ...context, actor: { ...context.actor, role_codes: ["SYSTEM_ADMIN"] } }}/>);
  expect(await screen.findByText("0 of 0 issues included")).toBeTruthy();
  expect(screen.queryByText("0%" )).toBeNull();
});

it("keeps the personal overview for non-administrative roles", async () => {
  render(<Home context={{ ...context, actor: { ...context.actor, role_codes: ["RISK_MANAGER"] }, capabilities: { config_read: true } }}/>);
  expect(await screen.findByRole("heading", { name: "Overview" })).toBeTruthy();
  expect(screen.queryByRole("heading", { name: "Dashboard" })).toBeNull();
});

it("opens a priority issue by its exact identifier", async () => {
  const data = payload();
  sources.oversight.mockResolvedValue({ ...data.oversight, interventions: [{ target_type: "MATTER", target_id: "issue/one", title: "Review access exceptions", owner_name: "Ada", state: "OPEN", priority: 4 }] });
  render(<Home context={context}/>);
  expect((await screen.findByRole("link", { name: "Open Review access exceptions" })).getAttribute("href")).toBe("#/work/issues/issue%2Fone");
});

it("does not offer an issue link for unknown target types or invalid identifiers", async () => {
  const data = payload();
  sources.oversight.mockResolvedValue({ ...data.oversight, interventions: [
    { target_type: "PROGRAM", target_id: "program-1", title: "Program item", state: "OPEN", priority: 4 },
    { target_type: "MATTER", target_id: "bad\nidentifier", title: "Invalid issue", state: "OPEN", priority: 4 },
  ] });
  render(<Home context={context}/>);
  expect(await screen.findByText("Program item")).toBeTruthy();
  expect(screen.queryByRole("link", { name: "Open Program item" })).toBeNull();
  expect(screen.queryByRole("link", { name: "Open Invalid issue" })).toBeNull();
});

it.each(["SYSTEM_ADMIN", "SUPER_ADMIN", "GRC_ADMIN", "GRC_ADMINISTRATOR"])("uses the administrative presentation for %s", async (role) => {
  render(<Home context={{ ...context, actor: { ...context.actor, role_codes: [role] } }}/>);
  expect(await screen.findByRole("heading", { name: "Dashboard" })).toBeTruthy();
});

it("keeps assignments available on an administrative dashboard when overview data fails", async () => {
  work.counts.mockResolvedValue({ scope: { tenant_id: "bank", legal_entity_id: "entity", principal_id: "person" }, coverage: "WORKFLOW_ASSIGNMENTS_ONLY", count_basis: "SOURCE_SNAPSHOT", as_of: "2026-11-20T10:00:00Z", total: 1, overdue: 0, due_soon: 1, undated: 0 });
  sources.metrics.mockRejectedValue(new Error("unavailable"));
  render(<Home context={{ ...context, actor: { ...context.actor, role_codes: ["SYSTEM_ADMIN"] } }}/>);
  expect(await screen.findByText("Dashboard data unavailable.")).toBeTruthy();
  expect(await screen.findByText("1 open")).toBeTruthy();
});
