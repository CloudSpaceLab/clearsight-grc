import type { OversightSnapshot } from "../../oversightApi";
import { OversightWorkspace } from "./OversightWorkspace";

const generatedAt = new Date().toISOString();
const snapshot: OversightSnapshot = {
  generated_at: generatedAt,
  period_start: "2026-06-03T08:00:00Z",
  period_end: "2026-09-01T08:00:00Z",
  projection_version: "oversight-v2",
  freshness: "CURRENT",
  source_high_water: { matters: generatedAt, actions: generatedAt, workflow_tasks: generatedAt, verification_results: generatedAt, continuity_events: generatedAt },
  coverage: { population: 42, excluded: 1, unknown: 2 },
  counts: { critical_high: 7, overdue: 4, due_soon: 3, routing_failures: 1, unassigned: 2, outcome_failures: 1 },
  interventions: [
    { target_type: "MATTER", target_id: "vendor-address", title: "Verify Northstar Payments registered address", category: "VENDOR_DEFICIENCY", state: "VERIFICATION", priority: 5, owner_name: "Ada Okafor", due_at: "2026-08-31T08:00:00Z", reason: "The issue is overdue and remains open.", next_action: "Review recovery plan" },
    { target_type: "MATTER", target_id: "access-review", title: "Resolve privileged-access review exceptions", category: "CONTROL_GAP", state: "ACTION_IN_PROGRESS", priority: 4, owner_name: "Chidi Eze", due_at: "2026-09-04T08:00:00Z", reason: "The issue is high priority and remains open.", next_action: "Review current facts" },
    { target_type: "MATTER", target_id: "incident", title: "Assign the payment incident follow-up", category: "INCIDENT", state: "TRIAGE", priority: 4, reason: "The issue has no accountable owner.", next_action: "Assign an eligible owner" },
  ],
  pressure: [
    { category: "CONTROL_GAP", critical: 1, high: 4, other: 3, overdue: 2 },
    { category: "VENDOR_DEFICIENCY", critical: 2, high: 2, other: 1, overdue: 1 },
    { category: "AUDIT_FINDING", critical: 0, high: 2, other: 4, overdue: 1 },
  ],
  aging: [{ label: "0–7 days", count: 5 }, { label: "8–30 days", count: 9 }, { label: "31–90 days", count: 4 }, { label: "Over 90 days", count: 2 }],
  performance: [
    { owner_id: "ada", owner_name: "Ada Okafor", current_load: 5, completed: 8, median_hours: 30, p75_hours: 52, sla_attainment: .875, reassigned: 2, returned: 1, blocked: 1, blocked_hours: 6, reopened: 1, measurement_samples: 8 },
    { owner_id: "chidi", owner_name: "Chidi Eze", current_load: 7, completed: 11, median_hours: 42, p75_hours: 68, sla_attainment: .818, reassigned: 0, returned: 0, blocked: 2, blocked_hours: 14, reopened: 0, measurement_samples: 11 },
  ],
  estimates: [{ category: "VENDOR_DEFICIENCY", sample_size: 12, median_hours: 48, lower_hours: 30, upper_hours: 72, confidence: "MEDIUM", estimated_by: "Closed issues of the same type in this legal entity during the selected period" }],
  history_quality: { completed_population: 14, complete_lifecycle: 12, missing_created_event: 1, missing_terminal_event: 1, excluded_from_durations: 2, reassigned_owner_excluded: 3 },
};

export function OversightEvidencePage() {
  return <OversightWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" onOpenMatter={() => {}} loadSnapshot={async () => snapshot}/>;
}
