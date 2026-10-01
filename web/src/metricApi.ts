import { requestJSON } from "./http";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type MetricCompleteness = "COMPLETE" | "PARTIAL" | "UNKNOWN";
export type MetricCondition = "CLEAR" | "ATTENTION";

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
  drill: MetricDrillTarget;
};

export type HomeMetricBundle = {
  generated_at: string;
  period_start: string;
  period_end: string;
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

export function loadHomeMetrics(): Promise<HomeMetricBundle> {
  return requestJSON<HomeMetricBundle>(apiBase, "/api/v1/metrics/home");
}
