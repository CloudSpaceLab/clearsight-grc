import { requestJSON } from "./http";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type GroupChildState = "AVAILABLE" | "STALE" | "MISSING";

export type GroupCounts = {
  critical_high: number;
  overdue: number;
  due_soon: number;
  routing_failures: number;
  unassigned: number;
  outcome_failures: number;
};

export type GroupRecordCoverage = {
  population: number;
  excluded?: number;
  unknown?: number;
};

export type GroupCoverage = {
  authorized_children: number;
  included_children: number;
  missing_children: number;
  stale_children: number;
  complete: boolean;
};

export type GroupDomainPosture = {
  risks_outside_appetite: number;
  indicator_breaches: number;
  assurance_failures: number;
};

export type GroupChild = {
  legal_entity_id: string;
  legal_entity_code: string;
  legal_entity_name: string;
  jurisdiction?: string;
  state: GroupChildState;
  child_snapshot_id?: string;
  child_generated_at?: string;
  child_projection_version?: string;
  coverage: GroupRecordCoverage;
  counts: GroupCounts;
  source_high_water?: Record<string, string>;
  domain_state: GroupChildState;
  domain_source_id?: string;
  domain_generated_at?: string;
  domain_definition_revision?: string;
  domain_posture: GroupDomainPosture;
  domain_source_high_water?: Record<string, string>;
};

export type GroupOversightSnapshot = {
  revision_id: string;
  generated_at: string;
  projection_version: string;
  freshness: "CURRENT" | "STALE";
  posture_freshness: "CURRENT" | "STALE";
  coverage: GroupCoverage;
  posture_coverage: GroupCoverage;
  posture: GroupDomainPosture;
  record_coverage: GroupRecordCoverage;
  counts: GroupCounts;
  children: GroupChild[];
};

export function loadGroupOversight(signal?: AbortSignal): Promise<GroupOversightSnapshot> {
  return requestJSON<GroupOversightSnapshot>(
    apiBase,
    "/api/v1/oversight/group",
    signal ? { signal } : undefined,
  );
}
