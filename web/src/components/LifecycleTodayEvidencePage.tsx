import { NavigationIcon } from "./NavigationIcon";
import { TodayInterventions } from "./TodayInterventions";
import type { AttentionItem, Readiness } from "../types";

const generatedAt = "2026-08-07T22:45:00Z";

const items: AttentionItem[] = [
  {
    id: "lifecycle-owner-review",
    type: "MATTER_WORK",
    title: "Restore an unavailable source",
    why_now: "This open issue is assigned to you and its current review step is ready.",
    scope: "Data protection Program · Control gap",
    state: "Ready",
    evidence: "Current issue ownership and source-health record",
    owner: "Program owner",
    due_at: "2026-08-08T08:30:00Z",
    primary_action: "Confirm scope and owner",
    action_target_type: "MATTER",
    action_target_id: "matter-source-recovery",
    intervention_class: "REVIEW",
    authority: { responsibility: "ACCOUNTABLE_OWNER", decision_type: "matter.transition", materiality: 4 },
  },
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
    <aside className="sidebar" aria-label="Primary navigation">
      <div className="brand-mark" aria-label="ClearSight">C</div>
      <nav><button className="nav-item active" type="button" aria-current="page"><NavigationIcon view="today"/><b>Today</b></button></nav>
      <div className="avatar" aria-label="Signed in as Ada Nwosu">AN</div>
    </aside>
    <main>
      <div className="context-bar" aria-label="Active workspace context"><div><strong>Meridian Trust Bank</strong><span>Meridian Trust Bank Nigeria</span></div><div className="context-role"><span>Independent reviewer</span></div></div>
      <header className="topbar today-topbar"><div><span className="eyebrow">Meridian Trust Bank</span><h1>Today</h1><p>Work requiring your review, decision or confirmation.</p></div></header>
      <TodayInterventions items={items} connection="live" readiness={readiness} readinessState="live" onOpenItem={() => {}} onInspectAuthority={() => {}}/>
    </main>
  </div>;
}
