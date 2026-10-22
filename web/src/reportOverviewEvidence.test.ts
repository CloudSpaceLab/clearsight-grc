import { afterEach, expect, it, vi } from "vitest";
import { installReportingEvidence } from "./reportingEvidence";
import { reportOverviewEvidenceScopeID as groupID } from "./reportOverviewEvidence";
import type { DomainMetricBundle, HomeMetricBundle, HomeMetricMemberPage, LossPeriodBundle, OrganizationMetricBreakdown } from "./metricApi";

afterEach(() => {
  vi.unstubAllGlobals();
  window.history.replaceState(null, "", "/");
});

function install(fixture: string) {
  const previous = vi.fn().mockImplementation(async () => new Response(JSON.stringify({
    tenant: { id: "bank-demo" }, legal_entity: { id: "bank-ng" }, actor: { id: "existing-actor" },
    capabilities: { config_read: false }, delegated: true,
  }), { headers: { "Content-Type": "application/json" } }));
  vi.stubGlobal("fetch", previous);
  window.history.replaceState(null, "", `/?fixture=${fixture}`);
  installReportingEvidence();
  return previous;
}

async function read<T>(path: string): Promise<T> {
  const response = await fetch(path);
  expect(response.status).toBe(200);
  return response.json() as Promise<T>;
}

it.each([undefined, groupID])("reconciles loaded risk and loss source values for scope %s", async (scopeID) => {
  const previous = install("report-overview-loaded");
  const scopeQuery = scopeID ? `?organization_scope_id=${scopeID}` : "";
  const home = await read<HomeMetricBundle>(`/api/v1/metrics/home${scopeQuery}`);
  const domain = await read<DomainMetricBundle>(`/api/v1/metrics/domain${scopeQuery}`);
  expect(home.scope_id).toBe(scopeID ?? "bank-ng");
  expect(domain.scope_id).toBe(home.scope_id);
  expect(domain.scope_kind).toBe(scopeID ? "ORGANIZATION_SCOPE" : "LEGAL_ENTITY");
  expect(home.items.map((item) => item.id)).toEqual(["critical_high_open", "overdue_open", "routing_gaps", "outcome_failures"]);
  const risk = domain.items.find((item) => item.id === "risks_outside_appetite")!;
  expect(risk.value).toBe(scopeID ? 2 : 4);
  expect(risk.population).toBe(scopeID ? 5 : 12);
  const query = new URLSearchParams({ source_id: domain.source_id, definition_revision: risk.definition_revision });
  if (scopeID) query.set("organization_scope_id", scopeID);
  const breakdown = await read<OrganizationMetricBreakdown>(`/api/v1/metrics/domain/risks_outside_appetite/organization-breakdown?${query}`);
  expect(breakdown).toMatchObject({ source_id: domain.source_id, count: risk.value, definition_revision: risk.definition_revision });
  expect(breakdown.scope_id).toBe(scopeID);
  expect(breakdown.items.reduce((total, item) => total + item.value, 0)).toBe(risk.value);
  const periodQuery = new URLSearchParams({ start_date: home.reporting_period.start_date, end_date: home.reporting_period.end_date });
  if (scopeID) periodQuery.set("organization_scope_id", scopeID);
  const loss = await read<LossPeriodBundle>(`/api/v1/metrics/losses/period?${periodQuery}`);
  expect(loss.scope_id).toBe(home.scope_id);
  expect(loss.period_start.slice(0, 10)).toBe(home.reporting_period.start_date);
  expect(loss.period_end.slice(0, 10)).toBe(home.reporting_period.end_date);
  expect(loss.net_loss?.minor_units).toBe("1040000000");
  expect(loss.flow_points.reduce((total, point) => total + BigInt(point.net_loss!.minor_units), 0n)).toBe(BigInt(loss.net_loss!.minor_units));
  const currency = loss.currencies[0]!;
  expect(BigInt(currency.gross.minor_units) - BigInt(currency.recovery.minor_units) + BigInt(currency.reversal.minor_units)).toBe(BigInt(currency.net.minor_units));
  expect(loss.comparison.comparison_quality).toBe("MISSING");
  expect(previous).not.toHaveBeenCalled();
});

it("preserves the signed-in evidence actor and exposes only the known organization area", async () => {
  const previous = install("report-overview-loaded");
  const context = await read<{ actor: { id: string }; capabilities: { config_read: boolean }; scope_hierarchy: { organization_scopes: Array<{ id: string }> } }>("/api/v1/context");
  expect(context.actor.id).toBe("existing-actor");
  expect(context.capabilities.config_read).toBe(false);
  expect(context.scope_hierarchy.organization_scopes.map((item) => item.id)).toEqual([groupID]);
  expect(previous).toHaveBeenCalledOnce();
});

it("rejects unsupported scope, wrong source and a reporting interval outside the supplied basis", async () => {
  install("report-overview-loaded");
  expect((await fetch("/api/v1/metrics/home?organization_scope_id=other-area")).status).toBe(403);
  expect((await fetch("/api/v1/metrics/domain?legal_entity_id=other-entity")).status).toBe(403);
  expect((await fetch("/api/v1/metrics/domain/risks_outside_appetite/organization-breakdown?source_id=old&definition_revision=enterprise-domain-v1")).status).toBe(409);
  expect((await fetch("/api/v1/metrics/losses/period?start_date=2026-01-01&end_date=2026-10-07")).status).toBe(400);
});

it("keeps partial risk data independent from unavailable loss data", async () => {
  install("report-overview-partial");
  const domain = await read<DomainMetricBundle>("/api/v1/metrics/domain");
  expect(domain.items[0]).toMatchObject({ value: 4, completeness: "PARTIAL", unknown: 2 });
  expect((await fetch("/api/v1/metrics/losses/period?start_date=2026-09-08&end_date=2026-10-07")).status).toBe(503);
});

it("returns a metric scope denial without replacing the existing report library fixture", async () => {
  install("report-overview-denied");
  expect((await fetch("/api/v1/metrics/home")).status).toBe(403);
  expect((await fetch("/api/v1/metrics/domain")).status).toBe(403);
  const reports = await read<{ items: unknown[] }>("/api/v1/reports/runs?tenant_id=bank-demo");
  expect(reports.items).toHaveLength(2);
});

it("leaves report-definitions metrics delegated for the unavailable baseline", async () => {
  const previous = install("report-definitions");
  const result = await read<{ delegated: boolean }>("/api/v1/metrics/home");
  expect(result.delegated).toBe(true);
  expect(previous).toHaveBeenCalledOnce();
});

it.each([undefined, groupID])("provides assessed zero risk and zero loss activity for the named empty Explore fixture in scope %s", async (scopeID) => {
  const previous = install("report-explore-empty");
  const groupQuery = scopeID ? `?organization_scope_id=${scopeID}` : "";
  const domain = await read<DomainMetricBundle>(`/api/v1/metrics/domain${groupQuery}`);
  expect(domain.items[0]).toMatchObject({ value: 0, population: scopeID ? 5 : 12, completeness: "COMPLETE", freshness: "CURRENT", unknown: 0, excluded: 0 });
  const query = new URLSearchParams({ source_id: domain.source_id, definition_revision: domain.definition_revision });
  if (scopeID) query.set("organization_scope_id", scopeID);
  const breakdown = await read<OrganizationMetricBreakdown>(`/api/v1/metrics/domain/risks_outside_appetite/organization-breakdown?${query}`);
  expect(breakdown.count).toBe(0);
  expect(breakdown.items).toEqual([]);
  const lossQuery = new URLSearchParams({ start_date: "2026-09-08", end_date: "2026-10-07" });
  if (scopeID) lossQuery.set("organization_scope_id", scopeID);
  const loss = await read<LossPeriodBundle>(`/api/v1/metrics/losses/period?${lossQuery}`);
  expect(loss).toMatchObject({ scope_id: scopeID ?? "bank-ng", event_count: 0, contributing_loss_count: 0, currencies: [], organization_breakdown: [] });
  expect(loss.net_loss).toBeUndefined();
  expect(previous).not.toHaveBeenCalled();
});

it.each([undefined, groupID])("keeps the mixed-currency fixture's signed exact amounts separate in scope %s", async (scopeID) => {
  install("report-explore-mixed-currency");
  const query = new URLSearchParams({ start_date: "2026-09-08", end_date: "2026-10-07" });
  if (scopeID) query.set("organization_scope_id", scopeID);
  const loss = await read<LossPeriodBundle>(`/api/v1/metrics/losses/period?${query}`);
  expect(loss).toMatchObject({ scope_id: scopeID ?? "bank-ng", mixed_currencies: true, event_count: 1, contributing_loss_count: 2 });
  expect(loss.net_loss).toBeUndefined();
  expect(loss.currencies.map((item) => [item.currency, item.net.minor_units])).toEqual([["NGN", "900719925474099301"], ["USD", "-450"]]);
  for (const flow of loss.currencies) {
    expect(BigInt(flow.gross.minor_units) - BigInt(flow.recovery.minor_units) + BigInt(flow.reversal.minor_units)).toBe(BigInt(flow.net.minor_units));
    expect(loss.organization_breakdown.reduce((total, item) => total + BigInt(item.currencies.find((currency) => currency.currency === flow.currency)!.net.minor_units), 0n)).toBe(BigInt(flow.net.minor_units));
    expect(loss.flow_points.reduce((total, item) => total + BigInt(item.currencies.find((currency) => currency.currency === flow.currency)!.net.minor_units), 0n)).toBe(BigInt(flow.net.minor_units));
  }
});

it.each([
  ["report-overview-loaded", undefined], ["report-overview-loaded", groupID],
  ["report-explore-empty", undefined], ["report-explore-empty", groupID],
  ["report-explore-mixed-currency", undefined], ["report-explore-mixed-currency", groupID],
])("returns retained members matching %s source, definition and scope %s", async (fixture, scopeID) => {
  const previous = install(fixture!);
  const groupQuery = scopeID ? `?organization_scope_id=${scopeID}` : "";
  const domain = await read<DomainMetricBundle>(`/api/v1/metrics/domain${groupQuery}`);
  const query = new URLSearchParams({ source_id: domain.source_id, definition_revision: domain.definition_revision, limit: "50" });
  if (scopeID) query.set("organization_scope_id", scopeID);
  const risk = await read<HomeMetricMemberPage>(`/api/v1/metrics/domain/risks_outside_appetite/members?${query}`);
  expect(risk).toMatchObject({ source_id: domain.source_id, definition_revision: domain.definition_revision, metric_id: "risks_outside_appetite", count: domain.items[0]!.value });
  expect(risk.items).toHaveLength(risk.count);
  expect(risk.items.every((item) => item.target_type === "RISK" && item.accessible && item.target_id)).toBe(true);
  const periodQuery = new URLSearchParams({ start_date: "2026-09-08", end_date: "2026-10-07" });
  if (scopeID) periodQuery.set("organization_scope_id", scopeID);
  const loss = await read<LossPeriodBundle>(`/api/v1/metrics/losses/period?${periodQuery}`);
  query.set("source_id", loss.source_id);
  query.set("definition_revision", loss.definition_revision);
  for (const [metric, count] of [["operational_loss_net", loss.contributing_loss_count], ["operational_loss_events", loss.event_count]] as const) {
    const page = await read<HomeMetricMemberPage>(`/api/v1/metrics/losses/${metric}/members?${query}`);
    expect(page).toMatchObject({ source_id: loss.source_id, definition_revision: loss.definition_revision, metric_id: metric, count });
    expect(page.items).toHaveLength(count);
    expect(page.items.every((item) => item.target_type === "LOSS" && item.accessible && item.target_id)).toBe(true);
  }
  expect(previous).not.toHaveBeenCalled();
});

it("rejects member reads for a different scope, source, definition or fixture instead of returning unrelated members", async () => {
  install("report-overview-loaded");
  const domain = await read<DomainMetricBundle>("/api/v1/metrics/domain");
  const query = new URLSearchParams({ source_id: domain.source_id, definition_revision: domain.definition_revision });
  const path = "/api/v1/metrics/domain/risks_outside_appetite/members";
  expect((await fetch(`${path}?${query}&organization_scope_id=${groupID}`)).status).toBe(409);
  expect((await fetch(`${path}?${query}&organization_scope_id=other`)).status).toBe(403);
  query.set("definition_revision", "other-definition");
  expect((await fetch(`${path}?${query}`)).status).toBe(409);
  query.set("definition_revision", domain.definition_revision);
  query.set("source_id", "8f730000-0000-4000-8000-000000000001");
  expect((await fetch(`${path}?${query}`)).status).toBe(409);
  expect((await fetch(`/api/v1/metrics/domain/unknown_metric/members?${query}`)).status).toBe(404);
});

it("keeps member counts stable across source-bound pages and rejects a foreign cursor", async () => {
  install("report-overview-loaded");
  const domain = await read<DomainMetricBundle>("/api/v1/metrics/domain");
  const query = new URLSearchParams({ source_id: domain.source_id, definition_revision: domain.definition_revision, limit: "2" });
  const path = "/api/v1/metrics/domain/risks_outside_appetite/members";
  const first = await read<HomeMetricMemberPage>(`${path}?${query}`);
  expect(first.count).toBe(4);
  expect(first.items).toHaveLength(2);
  expect(first.next_cursor).toContain(domain.source_id);
  query.set("cursor", first.next_cursor!);
  const second = await read<HomeMetricMemberPage>(`${path}?${query}`);
  expect(second.count).toBe(first.count);
  expect(second.items).toHaveLength(2);
  expect(new Set([...first.items, ...second.items].map((item) => item.member_id)).size).toBe(4);
  expect(second.next_cursor).toBeUndefined();
  query.set("cursor", "report-members:another-source:2");
  expect((await fetch(`${path}?${query}`)).status).toBe(400);
});

it("keeps member interception read-only and inside the named fixture boundary", async () => {
  const previous = install("report-explore-mixed-currency");
  await fetch("/api/v1/metrics/domain/risks_outside_appetite/members", { method: "POST", body: "{}" });
  expect(previous).toHaveBeenCalledOnce();
});
