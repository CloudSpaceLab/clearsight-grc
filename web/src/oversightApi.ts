import { requestJSON } from "./http";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type OversightSnapshot = {
  generated_at: string;
  period_start: string;
  period_end: string;
  projection_version: string;
  freshness: "CURRENT" | "STALE";
  source_high_water: Record<string, string>;
  coverage: { population: number; excluded?: number; unknown?: number };
  counts: { critical_high: number; overdue: number; due_soon: number; routing_failures: number; unassigned: number; outcome_failures: number };
  interventions: Array<{ target_type: string; target_id: string; title: string; category: string; state: string; priority: number; owner_id?: string; owner_name?: string; due_at?: string; reason: string; next_action: string }>;
  pressure: Array<{ category: string; critical: number; high: number; other: number; overdue: number }>;
  aging: Array<{ label: string; count: number }>;
  performance: Array<{ owner_id: string; owner_name: string; current_load: number; completed: number; median_hours?: number; p75_hours?: number; sla_attainment?: number; reassigned?: number; returned?: number; blocked: number; reopened: number; measurement_samples: number }>;
  estimates: Array<{ category: string; sample_size: number; median_hours: number; lower_hours: number; upper_hours: number; confidence: string; estimated_by: string }>;
};

export function loadOversight() {
  return requestJSON<OversightSnapshot>(apiBase, "/api/v1/oversight");
}
