import { useState } from "react";
import type { AttentionItem, MatterAggregate, ProgramAggregate, RecordResponsibleParty } from "./types";
import type { ProgramOperations } from "./programOperationsApi";
import type { ProgramReviewDigest } from "./programReviewApi";
import { ProgramCurrentPosition } from "./components/ProgramCurrentPosition";
import { ProgramSetupWorkspace } from "./components/ProgramSetupWorkspace";
import { ProgramRequirementsPanel } from "./components/ProgramRequirementsPanel";
import { MatterOutcomePanel } from "./components/MatterOutcomePanel";
import { TodayInterventions } from "./components/TodayInterventions";
import "./program-record.css";
import "./matter-record.css";
import "./monitoring.css";

const now = "2026-09-09T09:00:00Z";
const owner = { id: "sample-logistics-owner", display_name: "Amara Cole", role: "Logistics owner", kind: "PERSON" };
const reviewer = { id: "sample-reviewer", display_name: "Jordan Ellis", role: "Independent reviewer", kind: "PERSON" };
const program: ProgramAggregate = {
  program: { id: "sample-delivery", tenant_id: "sample-logistics", legal_entity_id: "sample-entity", name: "Delivery record retention", code: "DELIVERY", type: "OPERATIONS", status: "ACTIVE", owning_function: "Logistics", owner_principal_id: owner.id, scope: { description: "Carrier delivery records" }, effective_from: now, created_at: now, updated_at: now, version: 3 },
  state_label: "Up to date", requirements: [], applicability: [], control_objectives: [], control_implementations: [], requirement_control_links: [], evidence_contracts: [], evidence_assessments: [], triggers: [],
};
const operations: ProgramOperations = { program_id: program.program.id, program_version: 3, authority_available: true, operations: [], responsible_parties: [{ scope: "RECORD", responsibility: "ACCOUNTABLE_OWNER", display_name: owner.display_name, kind: "PERSON" }], generated_at: now };
const digest: ProgramReviewDigest = { program_id: program.program.id, state: "NO_BASELINE", review_required: false, current_program_version: 3, current_projection_version: 0, current_overall: "UNKNOWN", open_matter_count: 0, changes: [], changes_total: 0, changes_omitted: 0, history_truncated: false, current_exceptions: [], current_exceptions_total: 0, new_exceptions: [], new_exceptions_total: 0, resolved_exceptions: [], resolved_exceptions_total: 0 };

// This fixture interceptor is imported only by the isolated evidence entry.
// Authoring changes remain in this page's memory and never call a deployed API.
export function installUITruthEvidence() {
  if (!(new URLSearchParams(window.location.search).get("fixture") ?? "").startsWith("ui-truth-")) return;
  let stored = structuredClone(program);
  const original = globalThis.fetch.bind(globalThis);
  globalThis.fetch = async (input, init) => {
    const url = new URL(typeof input === "string" ? input : input instanceof URL ? input.href : input.url, window.location.origin);
    const method = init?.method ?? "GET";
    const json = (value: unknown, status = 200) => new Response(JSON.stringify(value), { status, headers: { "Content-Type": "application/json" } });
    if (url.pathname === "/api/v1/context") return json({ tenant: { id: "sample-logistics", name: "Sample · Northstar Logistics" }, legal_entity: { id: "sample-entity", name: "Northstar Logistics" }, actor: { id: owner.id, name: owner.display_name }, mode: "demo" });
    if (url.pathname === "/api/v1/programs/setup-candidates") return json({ owner_candidates: [owner], approval_authority_candidates: [reviewer], has_more: false, generated_at: now });
    if (url.pathname === "/api/v1/programs" && method === "POST") {
      const body = JSON.parse(String(init?.body));
      stored = { ...stored, program: { ...stored.program, name: body.name, code: body.code, type: body.type, owning_function: body.owning_function, scope: body.scope, version: 1, status: "DRAFT" } };
      return json(stored);
    }
    if (url.pathname === `/api/v1/programs/${program.program.id}/requirements` && method === "POST") {
      const body = JSON.parse(String(init?.body));
      if (![body.actor, body.action, body.object].every((value) => typeof value === "string" && value.trim())) return json({ message: "Enter the required party, action and subject." }, 400);
      stored = { ...stored, program: { ...stored.program, version: stored.program.version + 1 }, requirements: [...stored.requirements, { ...body, id: `sample-requirement-${stored.requirements.length + 1}` }] };
      return json(stored);
    }
    if (url.pathname === `/api/v1/programs/${program.program.id}/operations`) return json({ ...operations, program_version: stored.program.version });
    if (url.pathname.startsWith(`/api/v1/programs/${program.program.id}/`) && method === "GET") return json({ items: [] });
    if (url.pathname === "/api/v1/evidence/sources") return json({ items: [] });
    return original(input, init);
  };
}

export function UITruthEvidencePage({ state }: { state: string }) {
  const [editedProgram, setEditedProgram] = useState(program);
  const [setupOpen, setSetupOpen] = useState(true);
  const isSetup = state === "setup";
  const isRequirement = state === "requirement";
  const isOutcome = state.startsWith("reviewer-");
  const isSchedule = state.startsWith("schedule-");
  const snapshot: ProgramAggregate = state === "program-missing" ? program : {
    ...program,
    current_state: { id: "sample-state", program_version: state === "program-stale" ? 2 : 3, projection_version: 1, generated_at: now, overall_state: "CURRENT", dimensions: {}, reasons: [], open_matter_count: 0 },
  };
  const matter: MatterAggregate = {
    matter: { id: "sample-delivery-issue", tenant_id: "sample-logistics", legal_entity_id: "sample-entity", reference: "DEL-004", type: "CONTROL_GAP", status: "VERIFICATION", priority: 3, title: "Restore delivery record retention", summary: "Review the retained records after the storage repair.", scope: {}, known_facts: {}, missing_facts: [], contradictions: [], created_at: now, updated_at: now, version: 4 },
    type_label: "Control gap", status_label: "Outcome check", next_action: "Review outcome history", links: [], actions: [], decisions: [], response_packages: [],
    verification_contracts: [{ id: "sample-check", expected_outcome: "All delivery records remain available for the required retention period", scope: { description: "Carrier records received in August", measurement_method: "Sample the retention report and retrieve the selected records" }, baseline: { description: "Four records could not be retrieved" }, threshold: { success_condition: "Every selected record can be retrieved" }, observation_period_minutes: 1440, authority_principal_id: "former-reviewer", failure_response: "BLOCK_CLOSE", status: "ACTIVE" }],
    verification_results: [{ id: "sample-result", contract_id: "sample-check", result: "PASS", reviewer_principal_id: "former-reviewer", observed_at: "2026-09-08T10:00:00Z", rationale: "All 25 selected records were retrieved." }],
    closure: { ready: false, reasons: [] },
  } as MatterAggregate;
  const parties: RecordResponsibleParty[] = state === "reviewer-reassigned" ? [{ scope: "OUTCOME_RESULT", subresource_id: "sample-result", responsibility: "REVIEWER", display_name: "Morgan Reed", kind: "PERSON" }] : [];
  const attention: AttentionItem = { id: "sample-check-work", type: "MATTER_WORK", title: "Check delivery record retention", scope: "Northstar Logistics · August records", state: "Awaiting review", owner: reviewer.display_name, due_at: "2026-09-12T12:00:00Z", why_now: "The retention repair needs an independent outcome check.", evidence: "Retention report received 9 September", primary_action: "Review the retention report", intervention_class: "VERIFICATION", verification: { expected_outcome: "All selected records can be retrieved", method: "Independent sample review", next_check_at: state === "schedule-invalid" ? "invalid" : undefined } };
  return <main className="operating-evidence-page">
    <header className="topbar"><div><span className="eyebrow">Sample data · Northstar Logistics</span><h1>{isSetup || isRequirement ? "Delivery record requirements" : isOutcome ? "Delivery outcome review" : isSchedule ? "Assigned outcome check" : "Delivery record Program"}</h1><p>Sample records dated 9 September 2026.</p></div></header>
    {isSetup ? setupOpen ? <ProgramSetupWorkspace actorPrincipalID={owner.id} canConfigureSources={false} onCreated={setEditedProgram} onClose={() => setSetupOpen(false)}/> : <button type="button" onClick={() => setSetupOpen(true)}>Open Program setup</button>
      : isRequirement ? <ProgramRequirementsPanel aggregate={editedProgram} operations={[{ command: "program.requirement.add", label: "Add requirement", responsibility: "ACCOUNTABLE_OWNER", can_act: true, reason: "Assigned to Amara Cole." }]} onUpdated={setEditedProgram} onReload={() => window.location.reload()}/>
      : isOutcome ? <MatterOutcomePanel aggregate={matter} responsibleParties={parties} operations={[
        { command: "matter.outcome.record", subresource_id: "sample-check", label: "Record result", responsibility: "REVIEWER", can_act: false, reason: "Current review is assigned to Jordan Ellis.", assigned_to: reviewer },
        { command: "matter.outcome.define", label: "Define check", responsibility: "ACCOUNTABLE_OWNER", can_act: false, reason: "Assigned to Amara Cole.", assigned_to: owner },
      ]} onUpdated={() => window.location.reload()} onReload={() => window.location.reload()}/>
      : isSchedule ? <TodayInterventions items={[attention]} connection="live" readiness={null} readinessState="unavailable" onOpenItem={() => window.location.reload()}/>
      : <ProgramCurrentPosition aggregate={snapshot} operations={operations} digest={digest}/>}
  </main>;
}
