import { requestJSON } from "./http";
import { reportingPeriodPath, type ReportingPeriod, type ReportingPeriodQuery } from "./reportingPeriod";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type MetricCompleteness = "COMPLETE" | "PARTIAL" | "UNKNOWN";
export type MetricCondition = "CLEAR" | "ATTENTION";
export type MetricBasis = "CURRENT_POSTURE";

export type MetricDrillTarget = {
  workspace: string;
  filter: string;
  consistency: "CURRENT_STATE" | "SOURCE_SNAPSHOT";
};

export type HomeMetric = {
  id: string;
  label: string;
  value: number;
  unit: "COUNT";
  condition: MetricCondition;
  freshness: "CURRENT" | "STALE";
  completeness: MetricCompleteness;
  population: number;
  excluded?: number;
  unknown?: number;
  generated_at: string;
  source_revision: string;
  definition_revision: string;
  basis: MetricBasis;
  drill: MetricDrillTarget;
};

export type HomeMetricBundle = {
  generated_at: string;
  period_start: string;
  period_end: string;
  reporting_period: ReportingPeriod;
  posture_as_of: string;
  scope_id: string;
  scope_kind: "LEGAL_ENTITY" | "ORGANIZATION_SCOPE";
  freshness: "CURRENT" | "STALE";
  completeness: MetricCompleteness;
  population: number;
  excluded?: number;
  unknown?: number;
  source_id?: string;
  source_revision: string;
  definition_revision: string;
  items: HomeMetric[];
};

export function loadHomeMetrics(period?: ReportingPeriodQuery, organizationScopeID?: string): Promise<HomeMetricBundle> {
  return requestJSON<HomeMetricBundle>(apiBase, reportingPeriodPath("/api/v1/metrics/home", period, organizationScopeID));
}


export type HomeMetricMember = {
  member_id: string;
  target_type: "MATTER" | "PROGRAM" | "RISK" | "LOSS";
  target_id?: string;
  target_title: string;
  state: string;
  accessible: boolean;
};

export type HomeMetricMemberPage = {
  source_id: string;
  metric_id: string;
  definition_revision: string;
  count: number;
  items: HomeMetricMember[];
  next_cursor?: string;
};

export function loadHomeMetricMembers(
  metricID: string,
  sourceID: string,
  definitionRevision: string,
  organizationScopeID?: string,
  cursor?: string,
  limit = 50,
  signal?: AbortSignal,
): Promise<HomeMetricMemberPage> {
  const query = new URLSearchParams({
    source_id: sourceID,
    definition_revision: definitionRevision,
    limit: String(limit),
  });
  if (organizationScopeID) query.set("organization_scope_id", organizationScopeID);
  if (cursor) query.set("cursor", cursor);
  return requestJSON<HomeMetricMemberPage>(
    apiBase,
    `/api/v1/metrics/home/${encodeURIComponent(metricID)}/members?${query.toString()}`,
    signal ? { signal } : undefined,
  );
}


export type DomainMetricBundle = {
  generated_at: string;
  posture_as_of: string;
  scope_id: string;
  scope_kind: "LEGAL_ENTITY" | "ORGANIZATION_SCOPE";
  source_id: string;
  source_revision: string;
  definition_revision: string;
  items: HomeMetric[];
};

export function loadDomainMetrics(organizationScopeID?: string, signal?: AbortSignal): Promise<DomainMetricBundle> {
  const query = new URLSearchParams();
  if (organizationScopeID) query.set("organization_scope_id", organizationScopeID);
  const suffix = query.size ? `?${query.toString()}` : "";
  return requestJSON<DomainMetricBundle>(
    apiBase,
    `/api/v1/metrics/domain${suffix}`,
    signal ? { signal } : undefined,
  );
}

export function loadDomainMetricMembers(
  metricID: string,
  sourceID: string,
  definitionRevision: string,
  organizationScopeID?: string,
  cursor?: string,
  limit = 50,
  signal?: AbortSignal,
): Promise<HomeMetricMemberPage> {
  const query = new URLSearchParams({
    source_id: sourceID,
    definition_revision: definitionRevision,
    limit: String(limit),
  });
  if (organizationScopeID) query.set("organization_scope_id", organizationScopeID);
  if (cursor) query.set("cursor", cursor);
  return requestJSON<HomeMetricMemberPage>(
    apiBase,
    `/api/v1/metrics/domain/${encodeURIComponent(metricID)}/members?${query.toString()}`,
    signal ? { signal } : undefined,
  );
}


export type OrganizationMetricBucket = {
  key: string;
  scope_id?: string;
  label: string;
  kind: "ORGANIZATION_SCOPE" | "DIRECT" | "UNATTRIBUTED" | "UNAVAILABLE";
  value: number;
};

export type OrganizationMetricBreakdown = {
  source_id: string;
  metric_id: string;
  definition_revision: string;
  count: number;
  scope_id?: string;
  items: OrganizationMetricBucket[];
};

export function loadDomainMetricOrganizationBreakdown(
  metricID: string,
  sourceID: string,
  definitionRevision: string,
  organizationScopeID?: string,
  signal?: AbortSignal,
): Promise<OrganizationMetricBreakdown> {
  const query = new URLSearchParams({
    source_id: sourceID,
    definition_revision: definitionRevision,
  });
  if (organizationScopeID) query.set("organization_scope_id", organizationScopeID);
  return requestJSON<OrganizationMetricBreakdown>(
    apiBase,
    `/api/v1/metrics/domain/${encodeURIComponent(metricID)}/organization-breakdown?${query.toString()}`,
    signal ? { signal } : undefined,
  );
}


export type MetricTrendPoint = {
  at: string;
  value: number;
  freshness: "CURRENT" | "STALE";
  completeness: MetricCompleteness;
  population: number;
  excluded?: number;
  unknown?: number;
  source_revision: string;
};

export type MetricTrendSeries = {
  metric_id: string;
  definition_revision: string;
  start: string;
  end: string;
  resolution: "HOUR" | "DAY";
  points: MetricTrendPoint[];
  current?: MetricTrendPoint;
  baseline?: MetricTrendPoint;
  delta?: number;
  direction: "IMPROVED" | "WORSENED" | "UNCHANGED" | "UNKNOWN";
  comparison_quality: "COMPLETE" | "LIMITED" | "MISSING";
};

export type OrganizationTrendPoint = {
  date: string;
  at: string;
  value: number;
  source_revision: string;
  source_complete: boolean;
};

export type OrganizationTrendSeries = {
  metric_id: string;
  definition_revision: string;
  organization_scope_id: string;
  start: string;
  end: string;
  resolution: "DAY";
  points: OrganizationTrendPoint[];
};

export function loadDomainMetricTrend(
  metricID: string,
  startDate: string,
  endDate: string,
  signal?: AbortSignal,
): Promise<MetricTrendSeries> {
  const query = new URLSearchParams({ start_date: startDate, end_date: endDate });
  return requestJSON<MetricTrendSeries>(
    apiBase,
    `/api/v1/metrics/domain/${encodeURIComponent(metricID)}/trend?${query.toString()}`,
    signal ? { signal } : undefined,
  );
}

export function loadDomainMetricOrganizationTrend(
  metricID: string,
  organizationScopeID: string,
  startDate: string,
  endDate: string,
  signal?: AbortSignal,
): Promise<OrganizationTrendSeries> {
  const query = new URLSearchParams({
    organization_scope_id: organizationScopeID,
    start_date: startDate,
    end_date: endDate,
  });
  return requestJSON<OrganizationTrendSeries>(
    apiBase,
    `/api/v1/metrics/domain/${encodeURIComponent(metricID)}/organization-trend?${query.toString()}`,
    signal ? { signal } : undefined,
  );
}


export type MoneyValue = {
  minor_units: string;
  currency: string;
};

export type LossCurrencyFlow = {
  currency: string;
  gross: MoneyValue;
  recovery: MoneyValue;
  reversal: MoneyValue;
  net: MoneyValue;
  loss_event_count: number;
  recovery_event_count: number;
  reversal_event_count: number;
};

export type LossOrganizationFlow = {
  key: string;
  scope_id?: string;
  label: string;
  kind: "ORGANIZATION_SCOPE" | "DIRECT" | "UNATTRIBUTED" | "UNAVAILABLE";
  loss_event_count: number;
  contributing_loss_count: number;
  mixed_currencies: boolean;
  net_loss?: MoneyValue;
  currencies: LossCurrencyFlow[];
};

export type LossFlowPoint = {
  start: string;
  end: string;
  loss_event_count: number;
  contributing_loss_count: number;
  mixed_currencies: boolean;
  net_loss?: MoneyValue;
  currencies: LossCurrencyFlow[];
};

export type LossPeriodComparison = {
  period_start: string;
  period_end: string;
  event_count: number;
  contributing_loss_count: number;
  mixed_currencies: boolean;
  net_loss?: MoneyValue;
  currencies: LossCurrencyFlow[];
  event_delta: number;
  net_delta?: MoneyValue;
  direction: "IMPROVED" | "WORSENED" | "UNCHANGED" | "UNKNOWN";
  comparison_quality: "COMPLETE" | "LIMITED" | "MISSING";
};

export type LossPeriodBundle = {
  generated_at: string;
  period_start: string;
  period_end: string;
  scope_id: string;
  scope_kind: "LEGAL_ENTITY" | "ORGANIZATION_SCOPE";
  source_id: string;
  source_revision: string;
  definition_revision: string;
  event_count: number;
  contributing_loss_count: number;
  unattributed_event_count: number;
  mixed_currencies: boolean;
  net_loss?: MoneyValue;
  currencies: LossCurrencyFlow[];
  organization_breakdown: LossOrganizationFlow[];
  flow_resolution: "DAY" | "WEEK";
  flow_points: LossFlowPoint[];
  comparison: LossPeriodComparison;
};

export function loadLossPeriodMetrics(
  period: ReportingPeriodQuery,
  organizationScopeID?: string,
  signal?: AbortSignal,
): Promise<LossPeriodBundle> {
  const query = new URLSearchParams({
    start_date: period.start_date,
    end_date: period.end_date,
  });
  if (organizationScopeID) query.set("organization_scope_id", organizationScopeID);
  return requestJSON<LossPeriodBundle>(
    apiBase,
    `/api/v1/metrics/losses/period?${query.toString()}`,
    signal ? { signal } : undefined,
  );
}

export function loadLossPeriodMetricMembers(
  metricID: "operational_loss_events" | "operational_loss_net",
  sourceID: string,
  definitionRevision: string,
  organizationScopeID?: string,
  cursor?: string,
  limit = 50,
  signal?: AbortSignal,
): Promise<HomeMetricMemberPage> {
  const query = new URLSearchParams({
    source_id: sourceID,
    definition_revision: definitionRevision,
    limit: String(limit),
  });
  if (organizationScopeID) query.set("organization_scope_id", organizationScopeID);
  if (cursor) query.set("cursor", cursor);
  return requestJSON<HomeMetricMemberPage>(
    apiBase,
    `/api/v1/metrics/losses/${encodeURIComponent(metricID)}/members?${query.toString()}`,
    signal ? { signal } : undefined,
  );
}
