import { requestJSON } from "./http";
import { reportingPeriodPath, type ReportingPeriod, type ReportingPeriodQuery } from "./reportingPeriod";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type MetricCompleteness = "COMPLETE" | "PARTIAL" | "UNKNOWN";
export type MetricCondition = "CLEAR" | "ATTENTION";
export type MetricBasis = "CURRENT_POSTURE";

export type MetricDrillTarget = {
  workspace: string;
  filter: string;
  consistency: "CURRENT_STATE";
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
  scope_kind: "LEGAL_ENTITY";
  freshness: "CURRENT" | "STALE";
  completeness: MetricCompleteness;
  population: number;
  excluded?: number;
  unknown?: number;
  source_revision: string;
  definition_revision: string;
  items: HomeMetric[];
};

export function loadHomeMetrics(period?: ReportingPeriodQuery): Promise<HomeMetricBundle> {
  return requestJSON<HomeMetricBundle>(apiBase, reportingPeriodPath("/api/v1/metrics/home", period));
}
