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
  target_type: "MATTER" | "PROGRAM";
  target_id: string;
  target_title: string;
  state: string;
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
  cursor?: string,
  limit = 50,
  signal?: AbortSignal,
): Promise<HomeMetricMemberPage> {
  const query = new URLSearchParams({
    source_id: sourceID,
    definition_revision: definitionRevision,
    limit: String(limit),
  });
  if (cursor) query.set("cursor", cursor);
  return requestJSON<HomeMetricMemberPage>(
    apiBase,
    `/api/v1/metrics/home/${encodeURIComponent(metricID)}/members?${query.toString()}`,
    signal ? { signal } : undefined,
  );
}
