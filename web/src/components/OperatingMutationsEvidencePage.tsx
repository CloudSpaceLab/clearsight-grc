import { useState } from "react";
import "../program-record.css";
import type { MatterAggregate, ProgramAggregate, WorkflowTask } from "../types";
import { MatterWorkCommand } from "./MatterWorkCommandPanel";
import { ProgramLifecycleControls } from "./ProgramLifecycleControls";

const matter: MatterAggregate = {
  type_label: "Regulatory change",
  status_label: "Work in progress",
  next_action: "Complete the assigned remediation",
  matter: {
    id: "matter-operating-evidence", tenant_id: "bank-demo", reference: "CHG-2026-0042", type: "REGULATORY_CHANGE", status: "ACTION_IN_PROGRESS", priority: 4,
    title: "Implement annual-return evidence requirements", summary: "Complete the remaining owner assignment and record the current implementation state.",
    scope: {}, known_facts: {}, missing_facts: [], contradictions: [], created_at: "2026-08-09T08:00:00Z", updated_at: "2026-08-09T10:00:00Z", version: 8,
  },
  links: [], decisions: [],
  actions: [{
    id: "action-operating-evidence", title: "Complete the annual return evidence checklist",
    description: "Assign the remaining section owners and record the approved review date.", status: "IN_PROGRESS",
  }],
  verification_contracts: [], verification_results: [], response_packages: [], closure: { ready: false, reasons: ["One action remains open."] },
};

const actionTask: WorkflowTask = {
  id: "task-operating-evidence", tenant_id: "bank-demo", workflow_id: "workflow-operating-evidence", step_key: "matter-action", responsibility: "ACCOUNTABLE_OWNER",
  principal_id: "role-cro", title: "Complete the annual return evidence checklist", status: "IN_PROGRESS", version: 2,
  context: {
    type: "MATTER_ACTION", matter_id: matter.matter.id, action_id: "action-operating-evidence", command_name: "matter.action.transition",
    subresource_type: "ACTION", subresource_id: "action-operating-evidence", allowed_targets: "IMPLEMENTED,BLOCKED,CANCELLED", target_status: "",
    primary_action: "Update action", why_now: "This remediation is assigned to you and is still in progress.",
  },
};

const ownerReviewMatter: MatterAggregate = {
  ...matter,
  type_label: "Control gap",
  status_label: "Initial review",
  next_action: "Confirm scope and owner",
  matter: {
    ...matter.matter,
    id: "matter-owner-review-evidence",
    reference: "GAP-2026-0048",
    type: "CONTROL_GAP",
    status: "TRIAGE",
    title: "Restore an unavailable source",
    summary: "Confirm the affected Program scope and accountable owner before impact review begins.",
    owner_principal_id: "role-program-owner",
    version: 3,
  },
  actions: [],
};

const ownerReviewTask: WorkflowTask = {
  id: "task-owner-review-evidence", tenant_id: "bank-demo", workflow_id: "workflow-owner-review-evidence",
  step_key: "matter:matter-owner-review-evidence:ASSESSMENT", responsibility: "ACCOUNTABLE_OWNER",
  principal_id: "role-program-owner", title: ownerReviewMatter.matter.title, status: "READY", version: 1,
  context: {
    type: "MATTER_WORK", matter_id: ownerReviewMatter.matter.id, command_name: "matter.transition",
    allowed_targets: "ASSESSMENT", target_status: "ASSESSMENT", primary_action: "Confirm scope and owner",
    why_now: "This open issue is assigned to you and its current review step is ready.",
  },
};

const program: ProgramAggregate = {
  state_label: "Evidence incomplete",
  program: {
    id: "program-ndpa", tenant_id: "bank-demo", code: "NDPA", name: "Nigeria Data Protection Programme", type: "PRIVACY", status: "ACTIVE", owning_function: "Data Protection Office",
    owner_principal_id: "role-dpo", authority_principal_id: "role-cro", jurisdiction: "Nigeria", scope: {}, effective_from: "2025-01-01T00:00:00Z",
    created_at: "2026-07-01T09:00:00Z", updated_at: "2026-08-09T10:00:00Z", version: 12,
  },
  requirements: [], applicability: [], control_objectives: [], control_implementations: [], requirement_control_links: [], evidence_contracts: [], evidence_assessments: [], triggers: [],
};

export function OperatingMutationsEvidencePage() {
  const [currentProgram, setCurrentProgram] = useState<ProgramAggregate>(program);
  return <main className="operating-evidence-page">
    <header className="topbar"><div><span className="eyebrow">Review workspace</span><h1>Operating actions</h1><p>Review assigned issue actions and permitted Program status changes.</p></div></header>
    <div className="operating-evidence-grid">
      <section className="progressive-detail" aria-labelledby="owner-review-evidence-title">
        <span className="eyebrow">Work · issue owner</span>
        <h2 id="owner-review-evidence-title">{ownerReviewMatter.matter.title}</h2>
        <p>{ownerReviewMatter.matter.summary}</p>
        <MatterWorkCommand aggregate={ownerReviewMatter} task={ownerReviewTask} onUpdated={() => undefined} onCompleted={() => undefined}/>
      </section>
      <section className="progressive-detail" aria-labelledby="matter-evidence-title">
        <span className="eyebrow">Work · assigned action</span>
        <h2 id="matter-evidence-title">{matter.matter.title}</h2>
        <p>{matter.matter.summary}</p>
        <MatterWorkCommand aggregate={matter} task={actionTask} onUpdated={() => undefined} onCompleted={() => undefined}/>
      </section>
      <section className="progressive-detail" aria-labelledby="program-evidence-title">
        <span className="eyebrow">Program status</span>
        <h2 id="program-evidence-title">{program.program.name}</h2>
        <p>Available status changes depend on the current Program status and your approval authority.</p>
        <ProgramLifecycleControls aggregate={currentProgram} onUpdated={setCurrentProgram}/>
      </section>
    </div>
  </main>;
}
