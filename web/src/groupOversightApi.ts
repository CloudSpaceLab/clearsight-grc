import { requestJSON } from "./http";
import type { HomeMetric, MetricCompleteness } from "./metricApi";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type GroupChildState = "CURRENT" | "STALE" | "MISSING";

export type GroupCounts = {
  critical_high: number;
  overdue: number;
  due_soon: number;
  routing_failures: number;
  unassigned: number;
  outcome_failures: number;
};

export type GroupCoverage = {
  authorized_children: number;
  contributing_children: number;
  missing_children: number;
  stale_children: number;
  population: number;
  excluded?: number;
  unknown?: number;
};

export type GroupChild = {
  legal_entity_id: string;
  code: string;
  name: string;
  jurisdiction?: string;
  state: GroupChildState;
  snapshot_id?: string;
  generated_at?: string;
  posture_as_of?: string;
  coverage?: { population: number; excluded?: number; unknown?: number };
  counts?: GroupCounts;
};

export type GroupContributor = {
  legal_entity_id: string;
  snapshot_id: string;
  generated_at: string;
  posture_as_of: string;
  projection_version: string;
};

export type GroupOversightSnapshot = {
  scope_id: string;
  scope_name: string;
  scope_kind: "ORGANIZATION";
  generated_at: string;
  posture_as_of: string;
  freshness: "CURRENT" | "STALE";
  projection_version: string;
  contributor_revision: string;
  coverage: GroupCoverage;
  counts: GroupCounts;
  children: GroupChild[];
  contributors: GroupContributor[];
};

export type GroupMetricBundle = {
  generated_at: string;
  posture_as_of: string;
  scope_id: string;
  scope_kind: "ORGANIZATION";
  freshness: "CURRENT" | "STALE";
  completeness: MetricCompleteness;
  population: number;
  excluded?: number;
  unknown?: number;
  source_revision: string;
  definition_revision: string;
  items: HomeMetric[];
};

export type GroupOversightResponse = {
  snapshot: GroupOversightSnapshot;
  metrics: GroupMetricBundle;
};

export function loadGroupOversight(signal?: AbortSignal): Promise<GroupOversightResponse> {
  return requestJSON<GroupOversightResponse>(
    apiBase,
    "/api/v1/oversight/group",
    signal ? { signal } : undefined,
  );
}
