// Isolated browser fixtures, imported only through reportingEvidence/evidenceMain.
import type { DomainMetricBundle, HomeMetric, HomeMetricBundle, HomeMetricMember, HomeMetricMemberPage, LossCurrencyFlow, LossPeriodBundle, OrganizationMetricBreakdown } from "./metricApi";

export const reportOverviewEvidenceScopeID = "sample-scope-risk-ops";
const entityID = "bank-ng";
const at = "2026-10-07T12:00:00Z";
const start = "2026-09-08";
const end = "2026-10-07";
const fixtures = new Set(["report-overview-loaded", "report-overview-partial", "report-overview-denied", "report-explore-empty", "report-explore-mixed-currency"]);
const domainRevision = "enterprise-domain-v1";
type ExploreVariant = "empty" | "mixed" | undefined;

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
  const variant: ExploreVariant = fixture === "report-explore-empty" ? "empty" : fixture === "report-explore-mixed-currency" ? "mixed" : undefined;
  const scoped = Boolean(group);
  if (path === "/api/v1/metrics/home") return json(home(scoped));
  if (path === "/api/v1/metrics/domain") return json(domain(scoped, partial, variant));
  if (path === "/api/v1/metrics/domain/risks_outside_appetite/organization-breakdown") {
    const bundle = domain(scoped, partial, variant);
    if (url.searchParams.get("source_id") !== bundle.source_id
      || url.searchParams.get("definition_revision") !== domainRevision) {
      return failure(409, "metric_source_changed", "Reload risk concentration.");
    }
    const result: OrganizationMetricBreakdown = {
      source_id: bundle.source_id, metric_id: "risks_outside_appetite", definition_revision: domainRevision,
      count: bundle.items[0]!.value, ...(scoped ? { scope_id: reportOverviewEvidenceScopeID } : {}),
      items: variant === "empty" ? [] : scoped
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
    return json(loss(scoped, variant));
  }
  const memberPath = /^\/api\/v1\/metrics\/(domain|losses)\/([^/]+)\/members$/.exec(path);
  if (memberPath) {
    const [, kind, metricID] = memberPath;
    if (kind === "domain" && metricID !== "risks_outside_appetite"
      || kind === "losses" && metricID !== "operational_loss_net" && metricID !== "operational_loss_events") {
      return failure(404, "metric_not_found", "This measure has no retained member fixture.");
    }
    if (kind === "losses" && partial) return failure(503, "loss_metrics_unavailable", "Loss data unavailable.");
    const source = kind === "domain" ? domain(scoped, partial, variant) : loss(scoped, variant);
    if (url.searchParams.get("source_id") !== source.source_id
      || url.searchParams.get("definition_revision") !== source.definition_revision) {
      return failure(409, "metric_source_changed", "Reload the measure before reviewing its records.");
    }
    const items = kind === "domain" ? riskMembers(scoped, variant) : lossMembers(metricID!, variant);
    return memberPage(url, source.source_id, metricID!, source.definition_revision, items);
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

function domain(scoped: boolean, partial: boolean, variant?: ExploreVariant): DomainMetricBundle {
  const risk = metric("risks_outside_appetite", "Outside appetite", variant === "empty" ? 0 : scoped ? 2 : 4, scoped, domainRevision, "outside-appetite");
  return {
    ...scope(scoped), generated_at: at, posture_as_of: at,
    source_id: `${variant === "empty" ? "8f720000" : variant === "mixed" ? "8f730000" : "8f710000"}-0000-4000-8000-00000000000${scoped ? 2 : 1}`,
    source_revision: domainRevision, definition_revision: domainRevision,
    items: [{ ...risk, completeness: partial ? "PARTIAL" : "COMPLETE", unknown: partial ? 2 : 0 }],
  };
}

// Matches the existing loss-s5 fixture: one settlement loss, two recoveries and
// one reversal. Period net = 1,840,000,000 - 920,000,000 + 120,000,000 minor units.
function loss(scoped: boolean, variant?: ExploreVariant): LossPeriodBundle {
  const money = (value: string) => ({ minor_units: value, currency: "NGN" });
  const currency: LossCurrencyFlow = {
    currency: "NGN", gross: money("1840000000"), recovery: money("920000000"),
    reversal: money("120000000"), net: money("1040000000"),
    loss_event_count: 1, recovery_event_count: 2, reversal_event_count: 1,
  };
  const bundle: LossPeriodBundle = {
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
  if (!variant) return bundle;
  if (variant === "empty") return {
    ...bundle, source_id: `8f7a0000-0000-4000-8000-00000000000${scoped ? 2 : 1}`,
    event_count: 0, contributing_loss_count: 0, net_loss: undefined, currencies: [],
    organization_breakdown: [], flow_points: [], comparison: { ...bundle.comparison, event_delta: 0 },
  };
  const large: LossCurrencyFlow = {
    currency: "NGN", gross: money("900719925474099301"), recovery: money("0"), reversal: money("0"),
    net: money("900719925474099301"), loss_event_count: 1, recovery_event_count: 0, reversal_event_count: 0,
  };
  const usd = (minor_units: string) => ({ minor_units, currency: "USD" });
  const recovery: LossCurrencyFlow = { currency: "USD", gross: usd("0"), recovery: usd("450"), reversal: usd("0"),
    net: usd("-450"), loss_event_count: 0, recovery_event_count: 1, reversal_event_count: 0 };
  const currencies = [large, recovery];
  return {
    ...bundle, source_id: `8f7b0000-0000-4000-8000-00000000000${scoped ? 2 : 1}`,
    contributing_loss_count: 2, mixed_currencies: true, net_loss: undefined, currencies,
    organization_breakdown: [{ ...bundle.organization_breakdown[0]!, contributing_loss_count: 2,
      mixed_currencies: true, net_loss: undefined, currencies }],
    flow_points: [{ start: "2026-10-03T00:00:00Z", end: "2026-10-03T23:59:59Z",
      loss_event_count: 1, contributing_loss_count: 2, mixed_currencies: true, currencies }],
  };
}

function riskMembers(scoped: boolean, variant: ExploreVariant): HomeMetricMember[] {
  if (variant === "empty") return [];
  const titles = ["Sample settlement processing risk", "Sample supplier continuity risk", "Sample treasury limit risk", "Sample access review risk"];
  return titles.slice(0, scoped ? 2 : 4).map((title, index) => ({
    member_id: `report-risk-member-${index + 1}`, target_type: "RISK", target_id: `sample-report-risk-${index + 1}`,
    target_title: title, state: "OUTSIDE_APPETITE", accessible: true,
  }));
}

function lossMembers(metricID: string, variant: ExploreVariant): HomeMetricMember[] {
  if (variant === "empty") return [];
  const items: HomeMetricMember[] = [{ member_id: "report-loss-member-1", target_type: "LOSS",
    target_id: variant === "mixed" ? "sample-report-loss-ngn" : "loss-s5-1",
    target_title: variant === "mixed" ? "Sample high-value settlement loss" : "Sample duplicate merchant settlement",
    state: "RECORDED", accessible: true }];
  if (variant === "mixed" && metricID === "operational_loss_net") items.push({ member_id: "report-loss-member-2", target_type: "LOSS",
    target_id: "sample-report-loss-usd", target_title: "Sample prior-period recovery", state: "RECORDED", accessible: true });
  return items;
}

function memberPage(url: URL, sourceID: string, metricID: string, revision: string, items: HomeMetricMember[]) {
  const limit = Number(url.searchParams.get("limit") ?? "50");
  const cursor = url.searchParams.get("cursor");
  const cursorPrefix = `report-members:${sourceID}:`;
  const offset = cursor?.startsWith(cursorPrefix) ? Number(cursor.slice(cursorPrefix.length)) : cursor ? NaN : 0;
  if (!Number.isSafeInteger(limit) || limit < 1 || limit > 50 || !Number.isSafeInteger(offset) || offset < 0 || offset > items.length) {
    return failure(400, "metric_members_invalid", "Select a valid retained record page.");
  }
  const page: HomeMetricMemberPage = { source_id: sourceID, metric_id: metricID, definition_revision: revision,
    count: items.length, items: items.slice(offset, offset + limit),
    ...(offset + limit < items.length ? { next_cursor: `${cursorPrefix}${offset + limit}` } : {}),
  };
  return json(page);
}

function json(value: unknown, status = 200) {
  return new Response(JSON.stringify(value), { status, headers: { "Content-Type": "application/json" } });
}

function failure(status: number, code: string, message: string) {
  return json({ error: { code, message } }, status);
}
