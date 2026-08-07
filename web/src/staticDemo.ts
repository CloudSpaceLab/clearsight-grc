import type { DocumentImport, ProposalStatus } from "./documentTypes";

export const staticDemoEnabled = import.meta.env.VITE_STATIC_DEMO === "true";

const now = "2026-08-06T15:30:00Z";
const future = "2026-08-09T15:30:00Z";
const programID = "program-ndpa";
const matterID = "matter-gaid-change";
const evidenceID = "evidence-annual-return";

const program = {
  id: programID, tenant_id: "bank-demo", legal_entity_id: "bank-ng", code: "NDPA", name: "Nigeria Data Protection Programme", type: "PRIVACY", status: "ACTIVE", owning_function: "Data Protection Office", owner_principal_id: "role-dpo", authority_principal_id: "role-cro", jurisdiction: "Nigeria", scope: { legal_entity: "Bank Nigeria", business_lines: ["Retail", "Corporate", "Digital"] }, effective_from: "2025-01-01T00:00:00Z", created_at: "2026-07-01T09:00:00Z", updated_at: now, version: 12,
};
const programSummary = {
  program, state_label: "Evidence incomplete", overall_state: "EVIDENCE_INSUFFICIENT", reasons: [{ code: "EVIDENCE_COVERAGE", summary: "Two annual-return evidence sections still need an accountable owner.", object_type: "EVIDENCE_CONTRACT", object_id: "contract-return" }], open_matter_count: 2, requirement_count: 5, safeguard_count: 5, evidence_check_count: 5, state_generated_at: now,
};
const programDetail = {
  state_label: "Evidence incomplete", program,
  requirements: [
    { id: "req-1", code: "NDPA-RET-01", title: "Maintain accountable data-processing governance", statement: "The bank must maintain approved responsibilities, records and oversight for personal-data processing.", status: "APPROVED", source_anchor: "Nigeria Data Protection Act 2023 · governance duties" },
    { id: "req-2", code: "GAID-RETURN-01", title: "Submit the annual compliance return", statement: "The bank must complete the annual return with source-linked evidence and internal approval before filing.", status: "APPROVED", source_anchor: "GAID 2025 · annual return" },
  ],
  applicability: [{ id: "app-1", requirement_id: "req-1", status: "APPLICABLE", rationale: "The bank processes customer and employee personal data in Nigeria." }],
  control_objectives: [{ id: "obj-1", code: "PRIV-GOV", name: "Accountable privacy governance", outcome: "Responsibilities and decisions remain current and evidenced.", status: "ACTIVE" }],
  control_implementations: [{ id: "control-1", name: "Annual privacy compliance review", description: "DPO-led review with accountable evidence owners and executive sign-off.", implementation_type: "PROCESS", status: "IMPLEMENTED" }],
  requirement_control_links: [{ requirement_id: "req-1", implementation_id: "control-1" }],
  evidence_contracts: [
    { id: "contract-return", code: "GAID-RETURN", name: "Annual return evidence package", claim: "Every required return section has an owner, authoritative source, review status and approval date.", status: "ACTIVE", freshness_minutes: 525600, minimum_coverage: 1 },
    { id: "contract-training", code: "PRIV-TRAIN", name: "Privacy role training", claim: "Assigned privacy responsibilities have current training evidence.", status: "ACTIVE", freshness_minutes: 525600, minimum_coverage: .95 },
  ],
  evidence_assessments: [{ id: "assessment-1", contract_id: "contract-return", conclusion: "PARTIALLY_SUPPORTED", coverage: .8, assessed_at: now }],
  current_state: { id: "state-1", overall_state: "EVIDENCE_INSUFFICIENT", dimensions: { requirements: "CURRENT", controls: "CURRENT", evidence: "EVIDENCE_INSUFFICIENT" }, reasons: programSummary.reasons, open_matter_count: 2, generated_at: now, program_version: 12 },
  triggers: [{ id: "trigger-1", type: "REQUIREMENT_CHANGED", observed_at: "2026-08-04T10:00:00Z", source: "NDPC publication monitor" }],
};

const matter = {
  id: matterID, tenant_id: "bank-demo", reference: "CHG-2026-0042", type: "REGULATORY_CHANGE", status: "ACTION_IN_PROGRESS", priority: 4, title: "Implement GAID annual-return evidence requirements", summary: "Update the annual return process, evidence ownership and internal approval date.", scope: { journey_code: "REGULATORY_CHANGE", filing_year: 2027 }, known_facts: { official_source: "GAID 2025", filing_deadline: "31 March 2027", affected_process: "Annual privacy compliance return", complete_sections: 8, required_sections: 10 }, missing_facts: ["Owner for the processor register section", "Final DPCO review date"], contradictions: [], owner_principal_id: "role-dpo", required_authority: "AUTHORIZER", due_at: future, created_at: "2026-08-04T10:00:00Z", updated_at: now, version: 8,
};
const matterSummary = { matter, type_label: "Regulatory change", status_label: "Action in progress", next_action: "Complete the remaining evidence ownership updates", program_count: 1, open_action_count: 1, outcome_check_count: 1 };
const matterDetail = {
  type_label: "Regulatory change", status_label: "Action in progress", next_action: "Complete the remaining evidence ownership updates", matter,
  links: [{ id: "link-1", program_id: programID, relationship: "AFFECTS" }],
  decisions: [{ id: "decision-1", type: "IMPLEMENTATION_APPROACH", status: "APPROVED", selected_option: "UPDATE_CURRENT_PROCESS", rationale: "Use the existing annual return process with source-linked owners and an earlier internal approval date.", decided_at: "2026-08-05T11:00:00Z" }],
  actions: [{ id: "action-1", title: "Complete the annual return evidence checklist", description: "Assign the two remaining sections and record the DPCO review date.", status: "IN_PROGRESS", due_at: future }],
  verification_contracts: [{ id: "verify-1", expected_outcome: "All ten return sections have an owner, authoritative source and approved review status.", status: "ACTIVE", observation_period_minutes: 0 }],
  verification_results: [], response_packages: [], closure: { ready: false, reasons: ["One action remains in progress.", "The independent outcome check has not passed."] },
};

const evidenceRequest = {
  id: evidenceID, tenant_id: "bank-demo", subject_type: "MATTER", subject_id: matterID, title: "Confirm the remaining annual-return evidence owners", purpose: "Complete the evidence ownership record before the DPCO review.", why_you: "You own the affected privacy operations records.", sensitivity: "INTERNAL", audience_type: "INTERNAL", estimated_minutes: 6, deadline: future, known_facts: { filing_year: "2027", completed_sections: "8 of 10", internal_approval_date: "1 March 2027" }, fields: [{ id: "processor_register_owner", label: "Processor register owner", type: "text", required: true, description: "Name the accountable role or position." }, { id: "dpco_review_date", label: "DPCO review date", type: "text", required: true, description: "Enter the approved review date." }], status: "READY", version: 1, created_at: now, updated_at: now,
};

const todayItems = [
  { id: "today-change", type: "REGULATORY_CHANGE", title: matter.title, why_now: "The source change is approved and two evidence sections still need owners before the internal review.", scope: "Nigeria Data Protection · Regulatory change", state: "Action in progress", evidence: "8 of 10 sections complete", owner: "Data Protection Office", due_at: future, primary_action: "Complete the evidence ownership update", action_target_type: "MATTER", action_target_id: matterID },
  { id: "today-evidence", type: "EVIDENCE_REQUEST", title: evidenceRequest.title, why_now: evidenceRequest.why_you, scope: "Annual privacy return · Evidence", state: "Response required", evidence: "Known facts prefilled", owner: "Privacy Operations", due_at: future, primary_action: "Provide the two missing details", action_target_type: "EVIDENCE_REQUEST", action_target_id: evidenceID },
  { id: "today-program", type: "PROGRAM", title: "Review the Nigeria Data Protection Programme", why_now: "The latest status is evidence incomplete, not current.", scope: "Nigeria · Privacy", state: "Evidence incomplete", evidence: "5 evidence checks", owner: "Data Protection Officer", due_at: future, primary_action: "Review status reasons", action_target_type: "PROGRAM", action_target_id: programID },
];

let document: DocumentImport = {
  id: "document-gaid", tenant_id: "bank-demo", legal_entity_id: "bank-ng", file_name: "gaid-annual-return-notice.docx", media_type: "application/vnd.openxmlformats-officedocument.wordprocessingml.document", purpose: "Assess updated annual return requirements", source_type: "REGULATORY", size_bytes: 186420, sha256: "c6c882f6f0a23c3b7aa213a7ddab2b62d2fa86289d5378a430da356228b551d8", storage_key: "document-imports/bank-demo/document-gaid/gaid-annual-return-notice.docx", artifact_status: "STORED_UNSCANNED", extraction_status: "EXTRACTED", extraction_method: "DOCX_XML_V1", analysis_status: "REVIEW_REQUIRED", analysis_method: "DETERMINISTIC_RULES_V1", limitations: ["This stakeholder deployment uses retained reference data; no original customer or bank document is present.", "Analysis proposals require human review and do not create a legal interpretation or compliance conclusion."], sections: [{ id: "section-return", sequence: 1, title: "Annual return requirements", text: "The institution must maintain an accountable owner and authoritative source for every required annual return section. The completed return shall be reviewed before filing." }], proposals: [{ id: "proposal-owner", kind: "REQUIREMENT_CANDIDATE", title: "Possible requirement", statement: "The institution must maintain an accountable owner and authoritative source for every required annual return section.", confidence: .86, anchor: { section_id: "section-return", quote: "The institution must maintain an accountable owner and authoritative source for every required annual return section." }, status: "PENDING_REVIEW" }], created_by: "role-cro", created_at: now, updated_at: now, version: 1,
};

const guide = { code: "executive-first-run", profile: "executive", role: "Executive risk or compliance leader", version: 1, title: "Read the operating brief", description: "Distinguish current status from unknown, stale, at-risk and overdue work without opening several dashboards.", illustration: "guided-orbit", steps: [
  { id: "brief", title: "Start with what needs attention", description: "Today separates assigned work, due items and readiness so an empty count is never mistaken for a complete population.", action: "Review Today", view: "today", target: "today-brief" },
  { id: "attention", title: "Open one material record", description: "Move from the brief to the exact Program, issue or evidence request instead of a generic dashboard.", action: "Open first item", view: "today", target: "attention-list", intent: "open-first-attention" },
  { id: "programs", title: "Understand ongoing exposure", description: "Programs explain current status from requirements, safeguards, evidence and open issues.", action: "Open Programs", view: "programs", target: "programs-workspace" },
  { id: "finish", title: "Use the reason, not only the colour", description: "Every material status should expose the source, reason, owner and next valid action.", action: "Finish introduction", view: "programs", target: "programs-workspace" },
] };

export async function staticDemoRequest<T>(path: string, init?: RequestInit): Promise<T> {
  if (!staticDemoEnabled) throw new Error("Static demo transport is disabled");
  const url = new URL(path, "https://clearsight.demo");
  const pathname = url.pathname;
  const method = (init?.method ?? "GET").toUpperCase();

  if (pathname === "/api/v1/context") return clone({ tenant: { id: "bank-demo", name: "Meridian Trust Bank" }, legal_entity: { id: "bank-ng", name: "Meridian Trust Bank Nigeria" }, actor: { id: "role-cro", name: "Chief Risk Officer", kind: "PERSON", role_codes: ["CRO", "EXECUTIVE"], assurance_level: "MFA", authentication: "STATIC_DEMO", session_id: "pages-demo" }, mode: "static-stakeholder-demo", demo_mode: true, capabilities: { document_import: true, reference_journeys: true } }) as T;
  if (pathname === "/api/v1/today") return clone({ items: todayItems, generated_at: now }) as T;
  if (pathname === "/api/v1/compliance/readiness") return clone({ tenant_id: "bank-demo", status: "AT_RISK", baseline_known: true, generated_at: now, dimensions: { current: 3, aging: 1, at_risk: 2, unknown: 0, blocked_routing: 0, pending_human: 2 }, active_drifts: [{ id: "drift-1", subject_type: "PROGRAM", subject_id: programID, dimension: "EVIDENCE", severity: 4, summary: "Two annual-return evidence sections are incomplete.", required_action: "Assign owners and complete DPCO review.", detected_at: now }], recommended_actions: ["Complete the two missing evidence ownership records.", "Confirm the final DPCO review date."] }) as T;
  if (pathname === "/api/v1/program-summaries") return clone({ items: matches(url, programSummary.program.name, programSummary.program.code) ? [programSummary] : [], generated_at: now }) as T;
  if (pathname === `/api/v1/programs/${programID}`) return clone(programDetail) as T;
  if (pathname === "/api/v1/matter-summaries") return clone({ items: matches(url, matter.title, matter.reference) ? [matterSummary] : [], generated_at: now }) as T;
  if (pathname === `/api/v1/matters/${matterID}`) return clone(matterDetail) as T;
  if (pathname === "/api/v1/evidence/sources") return clone({ items: [{ id: "source-ndpc", tenant_id: "bank-demo", legal_entity_id: "bank-ng", code: "NDPC-PUBLICATIONS", name: "NDPC official publications", type: "REGULATORY", authority_class: "AUTHORITATIVE", expected_freshness_minutes: 1440, last_observed_at: now, last_success_at: now, health: "CURRENT", status: "ACTIVE", version: 3 }, { id: "source-iam", tenant_id: "bank-demo", legal_entity_id: "bank-ng", code: "IAM-ENTITLEMENTS", name: "Identity and access records", type: "SYSTEM", authority_class: "SYSTEM_OF_RECORD", expected_freshness_minutes: 60, last_observed_at: now, last_success_at: "2026-08-06T14:30:00Z", health: "DEGRADED", status: "ACTIVE", version: 8 }] }) as T;
  if (pathname === "/api/v1/evidence/requests") return clone({ items: [evidenceRequest] }) as T;
  if (pathname === `/api/v1/evidence/requests/${evidenceID}`) return clone(evidenceRequest) as T;
  if (pathname.includes(`/api/v1/evidence/requests/${evidenceID}/submissions`) && method === "POST") return clone({ request_id: evidenceID, status: "SUBMITTED", submitted_at: now }) as T;
  if (pathname === "/api/v1/authority/resolve") return clone({ principal: { id: "role-cro", display_name: "Chief Risk Officer", kind: "POSITION", role: "CRO" }, rule_id: "rule-critical-risk", policy_version: "risk-authority-v4", explanation: "Critical Nigeria regulatory changes require CRO authorization after independent compliance review." }) as T;
  if (pathname === "/api/v1/authority/integrity") return clone({ findings: [] }) as T;
  if (pathname === "/api/v1/authority/policies") return clone({ items: [{ id: "policy-risk", code: "RISK-AUTH", name: "Risk and compliance decision authority", status: "ACTIVE", version: 4, effective_from: "2026-07-01T00:00:00Z" }] }) as T;
  if (pathname === "/api/v1/workflow/tasks") return clone({ items: [{ id: "task-dpco", tenant_id: "bank-demo", workflow_id: "workflow-return", step_key: "DPCO_REVIEW", responsibility: "REVIEWER", principal_id: "role-dpco", title: "Confirm the final DPCO review date", status: "READY", due_at: future, context: { program: "Nigeria Data Protection Programme" }, version: 1 }] }) as T;
  if (pathname === "/api/v1/operations/projections") return clone({ items: [{ name: "program_state", status: "CURRENT", last_reconciled_at: now, lag_seconds: 3, record_count: 1 }] }) as T;
  if (pathname === "/api/v1/operations/projections/reconcile") return clone({ inspected: 1, repaired: 0, failed: 0, completed_at: now }) as T;
  if (pathname === "/api/v1/bank-journeys") return clone({ items: [] }) as T;
  if (pathname === "/api/v1/document-imports") return method === "POST" ? clone(document) as T : clone({ items: [document] }) as T;
  if (pathname === `/api/v1/document-imports/${document.id}`) return clone(document) as T;
  if (pathname.includes(`/api/v1/document-imports/${document.id}/proposals/`) && method === "POST") {
    const body = parseBody(init) as { status?: ProposalStatus; note?: string };
    document = { ...document, version: document.version + 1, updated_at: now, proposals: document.proposals.map((item) => ({ ...item, status: body.status ?? item.status, reviewed_by: "role-cro", reviewed_at: now, review_note: body.note })) };
    return clone(document) as T;
  }
  if (pathname === "/api/v1/onboarding/guide") return clone(guide) as T;
  if (pathname === "/api/v1/onboarding/state") {
    const code = url.searchParams.get("guide_code") ?? "executive-first-run";
    const key = `clearsight-static-guide:${code}`;
    const fallback = { tenant_id: "bank-demo", principal_id: "role-cro", guide_code: code, guide_version: 1, current_step: 0, completed: false, dismissed: false, version: 0 };
    if (method === "PUT") {
      const input = parseBody(init) as Record<string, unknown>;
      const current = JSON.parse(localStorage.getItem(key) ?? JSON.stringify(fallback)) as Record<string, unknown>;
      const value = { ...current, current_step: input.current_step, completed: input.completed, dismissed: input.dismissed, version: Number(current.version ?? 0) + 1, updated_at: now };
      localStorage.setItem(key, JSON.stringify(value));
      return clone(value) as T;
    }
    return clone(JSON.parse(localStorage.getItem(key) ?? JSON.stringify(fallback))) as T;
  }
  throw new Error(`Static stakeholder demo does not implement ${method} ${pathname}`);
}

function matches(url: URL, ...values: string[]) {
  const query = (url.searchParams.get("q") ?? "").trim().toLowerCase();
  const status = url.searchParams.get("status") ?? "";
  if (status && ![program.status, matter.status, "OPEN"].includes(status)) return false;
  return !query || values.some((value) => value.toLowerCase().includes(query));
}

function parseBody(init?: RequestInit) {
  if (typeof init?.body !== "string") return {};
  try { return JSON.parse(init.body) as unknown; } catch { return {}; }
}

function clone<T>(value: T): T {
  return typeof structuredClone === "function" ? structuredClone(value) : JSON.parse(JSON.stringify(value)) as T;
}
