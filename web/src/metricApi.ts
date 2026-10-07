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
