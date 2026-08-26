import type { CoverageDecision, DocumentCoverage, DocumentImport, ProposalStatus } from "./documentTypes";

export const staticDemoEnabled = import.meta.env.VITE_STATIC_DEMO === "true";

export class StaticDemoHTTPError extends Error {
  constructor(readonly status: number, readonly code: string, message: string) {
    super(message);
    this.name = "StaticDemoHTTPError";
  }
}

const now = "2026-08-06T15:30:00Z";
const future = new Date(Date.now() + 72 * 60 * 60 * 1000).toISOString();
const programID = "program-ndpa";
const matterID = "matter-gaid-change";
const evidenceID = "evidence-annual-return";

let program: Record<string, any>;
let programSummary: Record<string, any>;
let programDetail: Record<string, any>;
let programOperations: Record<string, any>;
const monitoringCheck = { id: "monitor-return", tenant_id: "bank-demo", program_id: programID, code: "RETURN-READINESS", name: "Annual return readiness", claim: "Every return section has an owner and approved evidence.", input_kind: "SOURCE", binding_id: "binding-return", binding_version: 2, status: "ACTIVE", is_current: true, submitted_by: "role-dpo", freshness_minutes: 1440, minimum_coverage: 1, failure_action: "RECOMMEND_MATTER", version: 3, created_at: now, updated_at: now };
const monitoringResult = { id: "monitor-result-1", monitoring_check_id: monitoringCheck.id, monitoring_check_version: 3, evaluated_at: now, evaluation: { score: 20, band: "MODERATE", coverage: .8, rule_results: [{ rule_id: "return-owner", field_id: "section_owner", outcome: "FAIL", reason: "Two sections do not have an approved owner." }] } };
const demoForms: any[] = [];
const demoChecks: any[] = [monitoringCheck];

let programReviewAcknowledged = false;
function programReviewDigest() {
  const checkpoint = {
    id: "review-ndpa-1", tenant_id: "bank-demo", program_id: programID, principal_id: "role-cro",
    program_version: programReviewAcknowledged ? 12 : 10, projection_version: programReviewAcknowledged ? 3 : 2,
    accepted_at: programReviewAcknowledged ? now : "2026-08-04T09:00:00Z",
  };
  if (programReviewAcknowledged) return {
    program_id: programID, state: "CURRENT", review_required: false, checkpoint,
    current_program_version: 12, current_projection_version: 3, current_overall: "EVIDENCE_INSUFFICIENT", baseline_overall: "EVIDENCE_INSUFFICIENT",
    open_matter_count: 2, open_matter_delta: 0, changes: [], changes_total: 0, changes_omitted: 0,
    current_exceptions: programSummary.reasons, current_exceptions_total: 1, new_exceptions: [], new_exceptions_total: 0, resolved_exceptions: [], resolved_exceptions_total: 0,
  };
  return {
    program_id: programID, state: "CHANGED", review_required: true, checkpoint,
    current_program_version: 12, current_projection_version: 3, current_overall: "EVIDENCE_INSUFFICIENT", baseline_overall: "CURRENT",
    open_matter_count: 2, open_matter_delta: 2,
    changes: [
      { kind: "STATE", summary: "Overall status changed from current to evidence insufficient." },
      { kind: "EVIDENCE", summary: "Evidence for Annual return evidence package was assessed as partially supported.", object_type: "EVIDENCE_CONTRACT", object_id: "contract-return" },
      { kind: "CHANGE", summary: "Observed change: requirement changed.", object_type: "REQUIREMENT", object_id: "req-2" },
    ],
    changes_total: 3, changes_omitted: 0,
    current_exceptions: programSummary.reasons, current_exceptions_total: 1,
    new_exceptions: programSummary.reasons, new_exceptions_total: 1,
    resolved_exceptions: [], resolved_exceptions_total: 0,
  };
}

let matter: Record<string, any>;
let matterSummary: Record<string, any>;
let matterDetail: Record<string, any>;
let evidenceRequest: Record<string, any>;
let todayItems: Array<Record<string, any>>;
type StaticDemoFixtures = {
  program: Record<string, any>;
  programSummary: Record<string, any>;
  programDetail: Record<string, any>;
  programOperations: Record<string, any>;
  matter: Record<string, any>;
  matterSummary: Record<string, any>;
  matterDetail: Record<string, any>;
  evidenceRequest: Record<string, any>;
  todayItems: Array<Record<string, any>>;
  guide: Record<string, any>;
  document: DocumentImport;
  documentCoverage: DocumentCoverage;
};

let document: DocumentImport;
let documentCoverage: DocumentCoverage;

export async function loadStaticDemoFixtures(fetcher: typeof fetch = globalThis.fetch) {
  const response = await fetcher("/static-demo-fixtures.json");
  if (!response.ok) throw new Error(`Static demo fixtures are unavailable (HTTP ${response.status}).`);
  const fixtures = await response.json() as StaticDemoFixtures;
  program = clone(fixtures.program);
  programSummary = clone(fixtures.programSummary);
  programSummary.program = program;
  programDetail = clone(fixtures.programDetail);
  programDetail.program = program;
  programOperations = clone(fixtures.programOperations);
  matter = clone(fixtures.matter);
  matter.due_at = future;
  matterSummary = clone(fixtures.matterSummary);
  matterSummary.matter = matter;
  matterDetail = clone(fixtures.matterDetail);
  matterDetail.matter = matter;
  for (const action of matterDetail.actions ?? []) action.due_at = future;
  evidenceRequest = clone(fixtures.evidenceRequest);
  evidenceRequest.deadline = future;
  todayItems = clone(fixtures.todayItems);
  for (const item of todayItems) item.due_at = future;
  guide = clone(fixtures.guide);
  document = clone(fixtures.document);
  documentCoverage = clone(fixtures.documentCoverage);
}
let guide: Record<string, any>;
export async function staticDemoRequest<T>(path: string, init?: RequestInit): Promise<T> {
  if (!staticDemoEnabled) throw new Error("Static demo transport is disabled");
  const url = new URL(path, "https://clearsight.demo");
  const pathname = url.pathname;
  const method = (init?.method ?? "GET").toUpperCase();
  const fixture = activeFixture();

  if (fixture === "today-loading" && pathname === "/api/v1/today") await delay(1800);
  if (fixture === "today-unavailable" && pathname === "/api/v1/today") throw new StaticDemoHTTPError(503, "today_unavailable", "Today's work is unavailable.");
  if (fixture === "evidence-requests-unavailable" && pathname === "/api/v1/evidence/requests") throw new StaticDemoHTTPError(503, "evidence_unavailable", "Evidence requests are temporarily unavailable.");
  if (fixture === "authority-forbidden" && pathname === "/api/v1/authority/resolve") throw new StaticDemoHTTPError(403, "permission_denied", "Authority inspection is restricted.");
  if (fixture === "capture-conflict" && pathname.includes(`/api/v1/evidence/requests/${evidenceID}/submissions`) && method === "POST") throw new StaticDemoHTTPError(409, "version_conflict", "The request changed while you were working.");

  if (pathname === "/api/v1/context") {
    const productionUnavailable = fixture === "today-unavailable";
    const noConfig = fixture === "no-config-access";
    return clone({ tenant: { id: "bank-demo", name: "Meridian Trust Bank" }, legal_entity: { id: "bank-ng", name: "Meridian Trust Bank Nigeria" }, actor: { id: "role-cro", name: "Chief Risk Officer", kind: "PERSON", role_codes: ["CRO", "EXECUTIVE"], assurance_level: "MFA", authentication: "STATIC_DEMO", session_id: "pages-demo" }, mode: "static-stakeholder-demo", demo_mode: !productionUnavailable, capabilities: { document_import: true, reference_journeys: !productionUnavailable, config_read: !noConfig, config_write: !noConfig, platform_operations_read: !noConfig, platform_operations_write: !noConfig } }) as T;
  }
  if (pathname === "/api/v1/today") return clone({ items: fixture === "today-empty" ? [] : todayItems, generated_at: now }) as T;
  if (pathname === "/api/v1/compliance/readiness") return clone({ tenant_id: "bank-demo", status: "AT_RISK", baseline_known: false, generated_at: now, dimensions: { current: 0, aging: 1, at_risk: 1, unknown: 1, blocked_routing: 0, pending_human: 1 }, active_drifts: [{ id: "drift-1", subject_type: "PROGRAM", subject_id: programID, dimension: "EVIDENCE", severity: 4, summary: "Two annual-return evidence sections are incomplete.", required_action: "Assign owners and complete DPCO review.", detected_at: now }], recommended_actions: ["Complete the two missing evidence ownership records.", "Confirm the final DPCO review date."] }) as T;
  if (pathname === "/api/v1/programs" && method === "POST") {
    const input = parseBody(init) as Record<string, any>;
    const createdProgram = { ...program, id: "program-created", code: input.code, name: input.name, type: input.type, status: "DRAFT", owning_function: input.owning_function, owner_principal_id: input.owner_candidate_id, authority_principal_id: input.approval_authority_candidate_id, jurisdiction: input.jurisdiction, scope: input.scope ?? {}, effective_from: input.effective_from, version: 1 };
    return clone({ state_label: "Setup in progress", program: createdProgram, requirements: [], applicability: [], control_objectives: [], control_implementations: [], requirement_control_links: [], evidence_contracts: [], evidence_assessments: [], triggers: [] }) as T;
  }
  if (pathname === "/api/v1/programs/setup-candidates" && method === "GET") return clone({ owner_candidates: [{ id: "role-dpo", display_name: "Data Protection Officer", kind: "POSITION", role: "DPO" }], approval_authority_candidates: [{ id: "role-cro", display_name: "Chief Risk Officer", kind: "POSITION", role: "CRO" }, { id: "role-deputy-cro", display_name: "Deputy Chief Risk Officer", kind: "POSITION", role: "Deputy CRO" }], has_more: false, generated_at: now }) as T;
  if (pathname === "/api/v1/program-summaries") return clone({ items: matches(url, programSummary.program.name, programSummary.program.code) ? [programSummary] : [], generated_at: now }) as T;
  if (pathname === `/api/v1/programs/${programID}`) return clone(programDetail) as T;
  if (pathname === `/api/v1/programs/${programID}/operations` && method === "GET") return clone({ ...programOperations, program_version: program.version }) as T;
  if (pathname === `/api/v1/programs/${programID}/details` && method === "POST") {
    const input = requireProgramVersion(init);
    program.name = String(input.name ?? program.name); program.owning_function = String(input.owning_function ?? program.owning_function); program.jurisdiction = String(input.jurisdiction ?? program.jurisdiction); program.scope = input.scope ?? program.scope; program.effective_from = String(input.effective_from ?? program.effective_from); (program as any).effective_until = input.effective_until;
    finishProgramMutation(); return clone(programDetail) as T;
  }
  if (pathname === `/api/v1/programs/${programID}/assignment` && method === "POST") {
    const input = requireProgramVersion(init); program.owner_principal_id = String(input.owner_principal_id ?? program.owner_principal_id); finishProgramMutation(); return clone(programDetail) as T;
  }
  if (pathname === `/api/v1/programs/${programID}/approval-authority` && method === "POST") {
    const input = requireProgramVersion(init); program.authority_principal_id = String(input.candidate_id ?? program.authority_principal_id); finishProgramMutation(); return clone(programDetail) as T;
  }
  if (pathname === `/api/v1/programs/${programID}/requirements` && method === "POST") {
    const input = requireProgramVersion(init); const detail = programDetail as any; detail.requirements.push({ id: `req-${detail.requirements.length + 1}`, ...input, source_anchor: input.source_anchor, effective_from: input.effective_from, status: "APPROVED" }); finishProgramMutation(); return clone(detail) as T;
  }
  if (/\/api\/v1\/programs\/[^/]+\/requirements\/[^/]+\/supersede$/.test(pathname) && method === "POST") {
    const input = requireProgramVersion(init); const detail = programDetail as any; const priorID = pathname.split("/").at(-2); const prior = detail.requirements.find((value: any) => value.id === priorID); if (prior) prior.status = "SUPERSEDED"; detail.requirements.push({ id: `req-${detail.requirements.length + 1}`, ...input, source_anchor: input.source_anchor, effective_from: input.effective_from, status: "APPROVED" }); finishProgramMutation(); return clone(detail) as T;
  }
  if (pathname === `/api/v1/programs/${programID}/applicability` && method === "POST") {
    const input = requireProgramVersion(init); const detail = programDetail as any; detail.applicability.push({ id: `app-${detail.applicability.length + 1}`, ...input, requirement_id: input.requirement_id, effective_from: input.effective_from }); finishProgramMutation(); return clone(detail) as T;
  }
  if (pathname === `/api/v1/programs/${programID}/control-objectives` && method === "POST") {
    const input = requireProgramVersion(init); const detail = programDetail as any; detail.control_objectives.push({ id: `obj-${detail.control_objectives.length + 1}`, ...input }); finishProgramMutation(); return clone(detail) as T;
  }
  if (pathname === `/api/v1/programs/${programID}/control-implementations` && method === "POST") {
    const input = requireProgramVersion(init); const detail = programDetail as any; detail.control_implementations.push({ id: `control-${detail.control_implementations.length + 1}`, ...input, objective_id: input.objective_id, implementation_type: input.implementation_type, owner_principal_id: input.owner_principal_id, effective_from: input.effective_from }); finishProgramMutation(); return clone(detail) as T;
  }
  if (pathname === `/api/v1/programs/${programID}/control-links` && method === "POST") {
    const input = requireProgramVersion(init); const detail = programDetail as any; detail.requirement_control_links.push({ id: `coverage-link-${detail.requirement_control_links.length + 1}`, requirement_id: input.requirement_id, implementation_id: input.implementation_id }); finishProgramMutation(); return clone(detail) as T;
  }
  if (/\/api\/v1\/programs\/[^/]+\/control-links\/[^/]+\/retirement$/.test(pathname) && method === "POST") {
    requireProgramVersion(init); const detail = programDetail as any; const linkID = decodeURIComponent(pathname.split("/").at(-2) ?? ""); detail.requirement_control_links = detail.requirement_control_links.filter((value: any) => value.id !== linkID); finishProgramMutation(); return clone(detail) as T;
  }
  if (pathname === `/api/v1/programs/${programID}/evidence-contracts` && method === "POST") {
    const input = requireProgramVersion(init); const detail = programDetail as any; detail.evidence_contracts.push({ id: `contract-${detail.evidence_contracts.length + 1}`, ...input, requirement_id: input.requirement_id, control_implementation_id: input.control_implementation_id, acceptable_source_ids: input.acceptable_source_ids, freshness_minutes: input.freshness_minutes, minimum_coverage: input.minimum_coverage, independence_required: input.independence_required, contradiction_policy: input.contradiction_policy, failure_action: input.failure_action }); finishProgramMutation(); return clone(detail) as T;
  }
  if (/\/api\/v1\/programs\/[^/]+\/evidence-contracts\/[^/]+\/assessments$/.test(pathname) && method === "POST") {
    const input = requireProgramVersion(init); const detail = programDetail as any; const contractID = decodeURIComponent(pathname.split("/").at(-2) ?? ""); if (input.contract_id !== contractID) throw new StaticDemoHTTPError(400, "contract_mismatch", "The evidence check in the request does not match the selected check."); detail.evidence_assessments.push({ id: `assessment-${detail.evidence_assessments.length + 1}`, ...input, contract_id: contractID, assessed_at: input.assessed_at, valid_until: input.valid_until, assessed_by: "role-cro" }); finishProgramMutation(); return clone(detail) as T;
  }
  if (pathname === `/api/v1/programs/${programID}/transition` && method === "POST") {
    const input = parseBody(init) as { expected_version?: number; to?: string; rationale?: string };
    if (input.expected_version !== program.version) throw new StaticDemoHTTPError(409, "version_conflict", "The Program changed before the status update was recorded.");
    const target = input.to ?? "";
    const transitions: Record<string, string[]> = { DRAFT: ["ACTIVE", "RETIRED"], ACTIVE: ["PAUSED", "RETIRED"], PAUSED: ["ACTIVE", "RETIRED"] };
    if (!(transitions[program.status] ?? []).includes(target)) throw new StaticDemoHTTPError(409, "transition_invalid", `The Program cannot move from ${program.status} to ${target || "an empty status"}.`);
    if (!input.rationale?.trim()) throw new StaticDemoHTTPError(400, "rationale_required", "A rationale is required for the Program status change.");
    program.status = target;
    program.version += 1;
    program.updated_at = now;
    programSummary.program_version = program.version;
    programSummary.projection_stale = true;
    return clone({ ...programDetail, program }) as T;
  }
  if (pathname === `/api/v1/programs/${programID}/review-digest` && method === "GET") return clone(programReviewDigest()) as T;
  if (pathname === `/api/v1/programs/${programID}/reviews` && method === "POST") {
    const input = parseBody(init) as { expected_program_version?: number; expected_projection_version?: number };
    if (input.expected_program_version !== 12 || input.expected_projection_version !== 3) throw new StaticDemoHTTPError(409, "version_conflict", "The Program changed while it was being reviewed.");
    programReviewAcknowledged = true;
    return clone(programReviewDigest()) as T;
  }
  if (pathname === "/api/v1/matter-summaries") return clone({ items: matches(url, matter.title, matter.reference) ? [matterSummary] : [], generated_at: now }) as T;
  if (pathname === "/api/v1/matters" && method === "POST") {
    const input = parseBody(init) as Record<string, any>;
    const createdMatter = { ...matter, id: "matter-created", reference: "MAT-DEMO-NEW", type: input.type ?? "CONTROL_GAP", status: "DRAFT", priority: input.priority ?? 3, title: input.title ?? "New issue", summary: input.summary ?? "New Program issue", scope: input.scope ?? {}, known_facts: input.known_facts ?? {}, missing_facts: input.missing_facts ?? [], contradictions: input.contradictions ?? [], version: 1 };
    return clone({ type_label: "Control gap", status_label: "Draft", next_action: "Start initial review", matter: createdMatter, links: input.program_id ? [{ id: "link-created", program_id: input.program_id, relationship: "AFFECTS" }] : [], decisions: [], actions: [], verification_contracts: [], verification_results: [], response_packages: [], closure: { ready: false, reasons: [] } }) as T;
  }
  if (pathname === `/api/v1/matters/${matterID}`) return clone(matterDetail) as T;
  if (/\/api\/v1\/matters\/[^/]+\/links\/[^/]+\/retirement$/.test(pathname) && method === "POST") {
    const input = parseBody(init) as { expected_version?: number }; if (input.expected_version !== matter.version) throw new StaticDemoHTTPError(409, "version_conflict", "The issue changed before the Program link was removed."); const linkID = decodeURIComponent(pathname.split("/").at(-2) ?? ""); matterDetail.links = matterDetail.links.filter((value: { id: string }) => value.id !== linkID); matter.version += 1; matter.updated_at = now; return clone(matterDetail) as T;
  }
  if (pathname === "/api/v1/evidence/sources") return clone({ items: [{ id: "source-ndpc", tenant_id: "bank-demo", legal_entity_id: "bank-ng", code: "NDPC-PUBLICATIONS", name: "NDPC official publications", type: "REGULATORY", authority_class: "AUTHORITATIVE", expected_freshness_minutes: 1440, last_observed_at: now, last_success_at: now, health: "CURRENT", status: "ACTIVE", version: 3 }, { id: "source-iam", tenant_id: "bank-demo", legal_entity_id: "bank-ng", code: "IAM-ENTITLEMENTS", name: "Identity and access records", type: "SYSTEM", authority_class: "SYSTEM_OF_RECORD", expected_freshness_minutes: 60, last_observed_at: now, last_success_at: "2026-08-06T14:30:00Z", health: "DEGRADED", status: "ACTIVE", version: 8 }] }) as T;
  if (pathname === "/api/v1/form-templates" && method === "GET") return clone({ items: demoForms }) as T;
  if (pathname === "/api/v1/form-templates" && method === "POST") {
    const input = parseBody(init) as Record<string, any>; const form = { id: `form-${demoForms.length + 1}`, tenant_id: "bank-demo", ...input, status: "DRAFT", version: 1, created_at: now, updated_at: now }; demoForms.push(form); return clone(form) as T;
  }
  if (/\/api\/v1\/form-templates\/[^/]+\/transition$/.test(pathname) && method === "POST") {
    const input = parseBody(init) as Record<string, any>; const id = pathname.split("/").at(-2); const prior = demoForms.find((value) => value.id === id); if (!prior || prior.version !== input.expected_version) throw new StaticDemoHTTPError(409, "version_conflict", "The form changed before its status was updated."); const updated = { ...prior, status: input.to, version: prior.version + 1, updated_at: now }; demoForms.splice(demoForms.indexOf(prior), 1, updated); return clone(updated) as T;
  }
  if (/\/api\/v1\/form-templates\/[^/]+\/collections$/.test(pathname) && method === "POST") return clone({ ...evidenceRequest, id: "evidence-monitoring-collection", title: "Complete the Program monitoring collection", status: "OPEN", version: 1 }) as T;
  if (pathname === `/api/v1/programs/${programID}/monitoring-checks` && method === "GET") return clone({ items: demoChecks }) as T;
  if (pathname === `/api/v1/programs/${programID}/monitoring-checks` && method === "POST") {
    const input = parseBody(init) as Record<string, any>; const check = { id: `monitor-${demoChecks.length + 1}`, tenant_id: "bank-demo", program_id: programID, ...input, status: "DRAFT", is_current: true, submitted_by: "role-cro", version: 1, created_at: now, updated_at: now }; demoChecks.push(check); return clone(check) as T;
  }
  if (/\/api\/v1\/monitoring-checks\/[^/]+\/transition$/.test(pathname) && method === "POST") {
    const input = parseBody(init) as Record<string, any>; const id = pathname.split("/").at(-2); const prior = demoChecks.find((value) => value.id === id); if (!prior || prior.version !== input.expected_version) throw new StaticDemoHTTPError(409, "version_conflict", "The monitoring check changed before its status was updated."); const updated = { ...prior, status: input.to, version: prior.version + 1, updated_at: now }; demoChecks.splice(demoChecks.indexOf(prior), 1, updated); return clone(updated) as T;
  }
  if (/\/api\/v1\/monitoring-checks\/[^/]+\/results$/.test(pathname) && method === "GET") return clone({ items: pathname.includes(monitoringCheck.id) ? [monitoringResult] : [] }) as T;
  if (/\/api\/v1\/monitoring-checks\/[^/]+\/evaluate-source$/.test(pathname) && method === "POST") return clone(monitoringResult) as T;
  if (pathname === "/api/v1/evidence/requests") return clone({ items: [fixture === "capture-terminal" ? { ...evidenceRequest, status: "EXPIRED" } : fixture === "long-content" ? { ...evidenceRequest, title: "Confirm the accountable owner for the processor register covering the Nigeria annual-return process across retail, corporate, digital and delegated processing operations", purpose: "Confirm the smallest unresolved ownership fact while preserving the full legal-entity, filing-year, source and review context needed by the DPCO without requiring the respondent to reconstruct the wider compliance programme." } : evidenceRequest] }) as T;
  if (pathname === `/api/v1/evidence/requests/${evidenceID}`) {
    const eligibilityPreload = url.searchParams.get("request_intent") === "eligibility_preload";
    if (fixture === "capture-not-found" && !eligibilityPreload) throw new StaticDemoHTTPError(404, "request_not_found", "The request is no longer available.");
    return clone(fixture === "capture-terminal" && !eligibilityPreload ? { ...evidenceRequest, status: "EXPIRED" } : evidenceRequest) as T;
  }
  if (pathname.includes(`/api/v1/evidence/requests/${evidenceID}/submissions`) && method === "POST") return clone({ request_id: evidenceID, status: "SUBMITTED", submitted_at: now }) as T;
  if (pathname === "/api/v1/authority/resolve") {
    const input = parseBody(init) as { responsibility?: string };
    const reviewer = input.responsibility === "REVIEWER";
    const principal = reviewer ? { id: "role-dpco", display_name: "Data Protection Compliance Officer", kind: "POSITION", role: "DPCO reviewer" } : { id: "role-cro", display_name: "Chief Risk Officer", kind: "POSITION", role: "CRO" };
    const candidates = reviewer ? [principal, { id: "role-deputy-dpco", display_name: "Deputy Data Protection Compliance Officer", kind: "POSITION", role: "Deputy DPCO reviewer" }] : [principal];
    return clone({ principal, candidate_principals: candidates, strategy: candidates.length > 1 ? "ANY_OF" : "SINGLE", rule_id: reviewer ? "rule-privacy-review" : "rule-critical-risk", policy_version: "risk-authority-v4", explanation: reviewer ? "The current privacy review policy permits either active independent DPCO reviewer for this Program." : "The current material authority policy resolves the CRO for this scoped decision." }) as T;
  }
  if (pathname === "/api/v1/authority/integrity") return clone({ findings: [] }) as T;
  if (pathname === "/api/v1/authority/policies") {
    if (fixture === "configure-partial") throw new StaticDemoHTTPError(503, "policy_unavailable", "Routing policies are temporarily unavailable.");
    return clone({ items: [{ id: "policy-risk", code: "RISK-AUTH", name: "Risk and compliance decision authority", status: "ACTIVE", version: 4, effective_from: "2026-07-01T00:00:00Z" }] }) as T;
  }
  if (pathname === "/api/v1/workflow/tasks") return clone({ items: [{ id: "task-dpco", tenant_id: "bank-demo", workflow_id: "workflow-return", step_key: "DPCO_REVIEW", responsibility: "REVIEWER", principal_id: "role-dpco", title: "Confirm the final DPCO review date", status: "READY", due_at: future, context: { program: "Nigeria Data Protection Programme" }, version: 1 }] }) as T;
  if (pathname === "/api/v1/operations/projections") return clone({ items: [{ tenant_id: "bank-demo", projection: "program_state", display_name: "Program status", state: "CURRENT", pending: 0, failed: 0, lag_seconds: 3, last_completed: now, updated_at: now }] }) as T;
  if (pathname === "/api/v1/operations/projections/reconcile") return clone({ tenant_id: "bank-demo", checked: 1, queued: 0, already_queued: 0, current: 1 }) as T;
  if (pathname === "/api/v1/compliance/automation-policies") return clone({ items: [] }) as T;
  if (pathname === "/api/v1/bank-journeys") return clone({ items: [], sample: true }) as T;
  if (pathname === "/api/v1/document-imports") return method === "POST" ? clone(document) as T : clone({ items: [document] }) as T;
  if (pathname === `/api/v1/document-imports/${document.id}/coverage` && method === "GET") return clone(documentCoverage) as T;
  if (pathname === `/api/v1/document-imports/${document.id}/coverage/review` && method === "POST") {
    const input = parseBody(init) as { expected_version?: number; decisions?: Array<{ candidate_id?: string; decision?: CoverageDecision; match_id?: string; reason?: string }> };
    if (input.expected_version !== documentCoverage.version) throw new StaticDemoHTTPError(409, "version_conflict", "The coverage assessment changed while it was being reviewed.");
    const decisions = input.decisions ?? [];
    documentCoverage = {
      ...documentCoverage, version: documentCoverage.version + 1, updated_at: now,
      candidates: documentCoverage.candidates.map((candidate) => {
        const decision = decisions.find((item) => item.candidate_id === candidate.id);
        return decision?.decision ? { ...candidate, classification: decision.decision === "NOT_APPLICABLE" ? "NOT_APPLICABLE" : candidate.classification, review: { decision: decision.decision, match_id: decision.match_id, reason: decision.reason, reviewer_id: "role-cro", reviewed_at: now } } : candidate;
      }),
    };
    return clone(documentCoverage) as T;
  }
  if (pathname === `/api/v1/document-imports/${document.id}/coverage/recompare` && method === "POST") return clone({ status: "QUEUED" }) as T;
  if (pathname.includes(`/api/v1/document-imports/${document.id}/coverage/suggestions/`) && pathname.endsWith("/apply") && method === "POST") {
    const input = parseBody(init) as { expected_version?: number };
    if (input.expected_version !== documentCoverage.version) throw new StaticDemoHTTPError(409, "version_conflict", "The coverage assessment changed while the recommendation was being applied.");
    documentCoverage = { ...documentCoverage, version: documentCoverage.version + 1, updated_at: now, suggestions: documentCoverage.suggestions.map((suggestion) => ({ ...suggestion, status: "APPLIED", applied_type: "REQUIREMENT", applied_id: "req-gaid-review-draft" })) };
    return clone({ assessment: documentCoverage, object_type: "REQUIREMENT", object_id: "req-gaid-review-draft" }) as T;
  }
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
  throw new StaticDemoHTTPError(501, "fixture_not_implemented", `Static stakeholder demo does not implement ${method} ${pathname}`);
}

function activeFixture() { return typeof window === "undefined" ? "" : new URLSearchParams(window.location.search).get("fixture") ?? ""; }
function requireProgramVersion(init?: RequestInit) {
  const input = parseBody(init) as Record<string, any>;
  if (input.expected_version !== program.version) throw new StaticDemoHTTPError(409, "version_conflict", "The Program changed before this update was recorded.");
  return input;
}
function finishProgramMutation() {
  program.version += 1; program.updated_at = now; programSummary.program_version = program.version; programSummary.projection_stale = true;
}
function delay(ms: number) { return new Promise((resolve) => window.setTimeout(resolve, ms)); }
function matches(url: URL, ...values: string[]) { const query = (url.searchParams.get("q") ?? "").trim().toLowerCase(); const status = url.searchParams.get("status") ?? ""; if (status && ![program.status, matter.status, "OPEN"].includes(status)) return false; return !query || values.some((value) => value.toLowerCase().includes(query)); }
function parseBody(init?: RequestInit) { if (typeof init?.body !== "string") return {}; try { return JSON.parse(init.body) as unknown; } catch { return {}; } }
function clone<T>(value: T): T { return typeof structuredClone === "function" ? structuredClone(value) : JSON.parse(JSON.stringify(value)) as T; }
