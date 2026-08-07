import { TodayInterventions } from "./TodayInterventions";
import type { AttentionItem, Readiness } from "../types";

const generatedAt = "2026-08-07T22:45:00Z";

const items: AttentionItem[] = [
  {
    id: "lifecycle-verification",
    type: "MATTER_WORK",
    title: "Confirm restored ATM availability",
    why_now: "The observation period has completed and this outcome check has no recorded result.",
    scope: "ATM availability · Issue ATM-2026-014",
    state: "Ready",
    evidence: "Outcome check ready",
    owner: "Independent reviewer",
    due_at: "2026-08-08T09:30:00Z",
    primary_action: "Record outcome check",
    action_target_type: "MATTER",
    action_target_id: "matter-verification",
    intervention_class: "VERIFICATION",
    material_conclusion: "The observation period is complete; the restored ATM still needs an independent outcome check.",
    authority: { responsibility: "REVIEWER", decision_type: "matter.outcome.record", materiality: 5 },
    verification: {
      state: "Outcome check ready",
      expected_outcome: "ATM remains available for one hour after restoration.",
      method: "Independent outcome review",
      next_check_at: "2026-08-08T09:30:00Z",
    },
  },
  {
    id: "lifecycle-acknowledgement",
    type: "MATTER_WORK",
    title: "NDPC incident response",
    why_now: "The response was transmitted and is waiting for acknowledgement to be recorded.",
    scope: "Authority response · NDPC",
    state: "Ready",
    evidence: "Transmission recorded",
    owner: "Regulatory Affairs",
    due_at: "2026-08-08T12:00:00Z",
    primary_action: "Record acknowledgement",
    action_target_type: "MATTER",
    action_target_id: "matter-authority-response",
    intervention_class: "EXTERNAL_REPRESENTATION",
    material_conclusion: "The package has been transmitted; acknowledgement is the only valid next transition.",
    authority: { responsibility: "ACKNOWLEDGEMENT_RECORDER", decision_type: "matter.response.transition", materiality: 4 },
  },
];

const readiness: Readiness = {
  tenant_id: "bank-demo",
  status: "AT_RISK",
  baseline_known: false,
  generated_at: generatedAt,
  dimensions: { current: 0, aging: 0, at_risk: 1, unknown: 1, blocked_routing: 0, pending_human: 2 },
  active_drifts: [],
  recommended_actions: [],
};

export function LifecycleTodayEvidencePage() {
  return <div className="app-shell" data-evidence-fixture="today-lifecycle">
    <main className="main-content">
      <header className="topbar"><div><span className="eyebrow">Meridian Trust Bank</span><h1>Today</h1><p>Work requiring your review, decision or confirmation.</p></div></header>
      <TodayInterventions items={items} connection="live" readiness={readiness} readinessState="live" onOpenItem={() => {}} onInspectAuthority={() => {}}/>
    </main>
  </div>;
}
