// Isolated browser fixtures, imported only through reportingEvidence/evidenceMain.
import type { DomainMetricBundle, HomeMetric, HomeMetricBundle, LossCurrencyFlow, LossPeriodBundle, OrganizationMetricBreakdown } from "./metricApi";

export const reportOverviewEvidenceScopeID = "sample-scope-risk-ops";
const entityID = "bank-ng";
const at = "2026-10-07T12:00:00Z";
const start = "2026-09-08";
const end = "2026-10-07";
const fixtures = new Set(["report-overview-loaded", "report-overview-partial", "report-overview-denied"]);
const domainRevision = "enterprise-domain-v1";

export async function reportOverviewEvidence(
  fixture: string | null, url: URL, readContext: () => Promise<Response>,
): Promise<Response | undefined> {
  if (!fixture || !fixtures.has(fixture)) return undefined;
  const path = url.pathname;
  if (path === "/api/v1/context") {
    const response = await readContext();
    if (!response.ok) return response;
    const context = await response.json();
    return json({ ...context, scope_hierarchy: {
      state: "COMPLETE",
      root: { id: "bank-demo", name: "Meridian Trust Bank", kind: "ORGANIZATION", filterable: false },
      current: { id: entityID, name: "Meridian Trust Bank Nigeria", kind: "LEGAL_ENTITY", current: true, filterable: false },
      legal_entities: [{ id: entityID, name: "Meridian Trust Bank Nigeria", kind: "LEGAL_ENTITY", current: true, filterable: false }],
      organization_scopes: [{ id: reportOverviewEvidenceScopeID, code: "RISK-OPS", name: "Risk Operations",
        kind: "DEPARTMENT", department_path: ["BANK", "RISK", "OPERATIONS"], filterable: true }],
      organization_scopes_truncated: false,
    } });
  }
  if (!path.startsWith("/api/v1/metrics/")) return undefined;
  const group = url.searchParams.get("organization_scope_id");
  if (fixture === "report-overview-denied" || (group && group !== reportOverviewEvidenceScopeID)
    || (url.searchParams.has("legal_entity_id") && url.searchParams.get("legal_entity_id") !== entityID)
    || (url.searchParams.has("tenant_id") && url.searchParams.get("tenant_id") !== "bank-demo")) {
    return failure(403, "metric_scope_denied", "Reporting metrics are unavailable for this scope.");
  }
  const partial = fixture === "report-overview-partial";
  const scoped = Boolean(group);
  if (path === "/api/v1/metrics/home") return json(home(scoped));
  if (path === "/api/v1/metrics/domain") return json(domain(scoped, partial));
  if (path === "/api/v1/metrics/domain/risks_outside_appetite/organization-breakdown") {
    const bundle = domain(scoped, partial);
    if (url.searchParams.get("source_id") !== bundle.source_id
      || url.searchParams.get("definition_revision") !== domainRevision) {
      return failure(409, "metric_source_changed", "Reload risk concentration.");
    }
    const result: OrganizationMetricBreakdown = {
      source_id: bundle.source_id, metric_id: "risks_outside_appetite", definition_revision: domainRevision,
      count: scoped ? 2 : 4, ...(scoped ? { scope_id: reportOverviewEvidenceScopeID } : {}),
      items: scoped
        ? [{ key: "direct", label: "Risk Operations", kind: "DIRECT", value: 2 }]
        : [{ key: `scope:${reportOverviewEvidenceScopeID}`, scope_id: reportOverviewEvidenceScopeID,
          label: "Risk Operations", kind: "ORGANIZATION_SCOPE", value: 2 },
        { key: "direct", label: "Legal entity", kind: "DIRECT", value: 2 }],
    };
    return json(result);
  }
  if (path === "/api/v1/metrics/losses/period") {
    if (url.searchParams.get("start_date") !== start || url.searchParams.get("end_date") !== end) {
      return failure(400, "reporting_period_invalid", "Select the reporting period supplied by Home.");
    }
    if (partial) return failure(503, "loss_metrics_unavailable", "Loss data unavailable.");
    return json(loss(scoped));
  }
  return undefined;
}

function scope(scoped: boolean) {
  return { scope_id: scoped ? reportOverviewEvidenceScopeID : entityID,
    scope_kind: scoped ? "ORGANIZATION_SCOPE" as const : "LEGAL_ENTITY" as const };
}

function metric(id: string, label: string, value: number, scoped: boolean, revision: string, filter: string): HomeMetric {
  return {
    id, label, value, unit: "COUNT", condition: value ? "ATTENTION" : "CLEAR", freshness: "CURRENT",
    completeness: "COMPLETE", population: scoped ? 5 : 12, excluded: 0, unknown: 0,
    generated_at: at, source_revision: revision === domainRevision ? revision : "oversight-v5",
    definition_revision: revision, basis: "CURRENT_POSTURE",
    drill: { workspace: revision === domainRevision ? "risks" : "oversight", filter, consistency: "SOURCE_SNAPSHOT" },
  };
}

function home(scoped: boolean): HomeMetricBundle {
  return {
    ...scope(scoped), generated_at: at, period_start: `${start}T00:00:00Z`, period_end: at,
    reporting_period: { start_date: start, end_date: end, mode: "CURRENT_WINDOW", max_days: 365, historical_end_supported: false },
    posture_as_of: at, freshness: "CURRENT", completeness: scoped ? "UNKNOWN" : "COMPLETE", population: scoped ? 5 : 12,
    source_id: `8f700000-0000-4000-8000-00000000000${scoped ? 2 : 1}`,
    excluded: 0, unknown: 0, source_revision: "oversight-v5", definition_revision: "home-oversight-v3",
    items: [
      metric("critical_high_open", "Critical and high", scoped ? 1 : 3, scoped, "home-oversight-v3", "critical-high"),
      metric("overdue_open", "Overdue", scoped ? 1 : 2, scoped, "home-oversight-v3", "overdue"),
      metric("routing_gaps", "Routing gaps", 0, scoped, "home-oversight-v3", "routing-gaps"),
      metric("outcome_failures", "Outcome failures", scoped ? 0 : 1, scoped, "home-oversight-v3", "outcome-failures"),
    ].map((item) => ({ ...item, completeness: scoped ? "UNKNOWN" : "COMPLETE" })),
  };
}

function domain(scoped: boolean, partial: boolean): DomainMetricBundle {
  const risk = metric("risks_outside_appetite", "Outside appetite", scoped ? 2 : 4, scoped, domainRevision, "outside-appetite");
  return {
    ...scope(scoped), generated_at: at, posture_as_of: at,
    source_id: `8f710000-0000-4000-8000-00000000000${scoped ? 2 : 1}`,
    source_revision: domainRevision, definition_revision: domainRevision,
    items: [{ ...risk, completeness: partial ? "PARTIAL" : "COMPLETE", unknown: partial ? 2 : 0 }],
  };
}

// Matches the existing loss-s5 fixture: one settlement loss, two recoveries and
// one reversal. Period net = 1,840,000,000 - 920,000,000 + 120,000,000 minor units.
function loss(scoped: boolean): LossPeriodBundle {
  const money = (value: string) => ({ minor_units: value, currency: "NGN" });
  const currency: LossCurrencyFlow = {
    currency: "NGN", gross: money("1840000000"), recovery: money("920000000"),
    reversal: money("120000000"), net: money("1040000000"),
    loss_event_count: 1, recovery_event_count: 2, reversal_event_count: 1,
  };
  return {
    ...scope(scoped), generated_at: at, period_start: `${start}T00:00:00Z`, period_end: at,
    source_id: `8f790000-0000-4000-8000-00000000000${scoped ? 2 : 1}`,
    source_revision: "operational-loss-ledger-v1", definition_revision: "operational-loss-period-v1",
    event_count: 1, contributing_loss_count: 1, unattributed_event_count: 0,
    mixed_currencies: false, net_loss: currency.net, currencies: [currency], flow_resolution: "DAY",
    organization_breakdown: [{ key: scoped ? "direct" : `scope:${reportOverviewEvidenceScopeID}`,
      ...(scoped ? {} : { scope_id: reportOverviewEvidenceScopeID }), label: "Risk Operations",
      kind: scoped ? "DIRECT" : "ORGANIZATION_SCOPE", loss_event_count: 1, contributing_loss_count: 1,
      mixed_currencies: false, net_loss: currency.net, currencies: [currency] }],
    flow_points: [
      ["2026-10-03", "1840000000", 1], ["2026-10-04", "-620000000", 0], ["2026-10-05", "-180000000", 0],
    ].map(([date, net, events]) => ({
      start: `${date}T00:00:00Z`, end: `${date}T23:59:59Z`, loss_event_count: Number(events),
      contributing_loss_count: 1, mixed_currencies: false, net_loss: money(String(net)), currencies: [],
    })),
    comparison: { period_start: "2026-08-09T00:00:00Z", period_end: "2026-09-07T23:59:59Z",
      event_count: 0, contributing_loss_count: 0, mixed_currencies: false, currencies: [],
      event_delta: 1, direction: "UNKNOWN", comparison_quality: "MISSING" },
  };
}

function json(value: unknown, status = 200) {
  return new Response(JSON.stringify(value), { status, headers: { "Content-Type": "application/json" } });
}

function failure(status: number, code: string, message: string) {
  return json({ error: { code, message } }, status);
}
