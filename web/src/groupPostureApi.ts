import { requestJSON } from "./http";
import type { MetricCompleteness } from "./metricApi";
import type { ReportingPeriodQuery } from "./reportingPeriod";
import { reportingPeriodPath } from "./reportingPeriod";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type GroupPostureCounts = {
  risks_outside_appetite: number;
  indicator_breaches: number;
  assurance_failures: number;
  loss_events: number;
};

export type GroupPostureCoverage = {
  authorized_children: number;
  included_children: number;
  missing_children: number;
  stale_children: number;
  partial_children: number;
  complete: boolean;
};

export type GroupPostureChild = {
  legal_entity_id: string;
  legal_entity_code: string;
  legal_entity_name: string;
  jurisdiction?: string;
  risk_state: "AVAILABLE" | "STALE" | "MISSING";
  completeness: MetricCompleteness;
  source_id?: string;
  source_generated_at?: string;
  source_revision?: string;
  definition_revision?: string;
  counts: GroupPostureCounts;
  unknown: number;
  excluded: number;
  freshness: "CURRENT" | "STALE";
};

export type GroupPostureBundle = {
  generated_at: string;
  period_start: string;
  period_end: string;
  definition_revision: string;
  risk_coverage: GroupPostureCoverage;
  loss_coverage: GroupPostureCoverage;
  counts: GroupPostureCounts;
  children: GroupPostureChild[];
};

export function loadGroupPosture(
  period: ReportingPeriodQuery,
  signal?: AbortSignal,
): Promise<GroupPostureBundle> {
  return requestJSON<GroupPostureBundle>(
    apiBase,
    reportingPeriodPath("/api/v1/metrics/group/posture", period),
    signal ? { signal } : undefined,
  );
}
