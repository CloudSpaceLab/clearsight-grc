import type { AttentionItem, AutomationPolicy, AuthorityResolution, CaptureRequest, EvidenceRequest, EvidenceSource, IntegrityFinding, MatterAggregate, OnboardingGuide, OnboardingState, PolicySummary, ProgramAggregate, Readiness, WorkflowTask } from "./types";
import type { MatterSummary, ProgramSummary, SummaryPage, SummaryQuery } from "./summaryTypes";
import type { ProjectionHealth, ReconcileResult } from "./operationsTypes";
import type { BankJourneysResponse } from "./verticalTypes";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";
const sampleCaptureRequestID = import.meta.env.VITE_SAMPLE_CAPTURE_REQUEST_ID as string | undefined;
const sampleAuthorityObjectID = import.meta.env.VITE_SAMPLE_AUTHORITY_OBJECT_ID as string | undefined;

export type RuntimeContext = {
  tenant: { id: string; name: string };
  legal_entity: { id: string; name: string };
  actor: { id: string; name: string; kind?: string; assurance_level?: string; authentication?: string; session_id?: string };
  mode: string;
};

let runtimeContext: Promise<RuntimeContext> | undefined;

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${apiBase}${path}`, { ...init, headers: { "Content-Type": "application/json", ...init?.headers } });
  if (!response.ok) {
    const body = (await response.json().catch(() => null)) as { message?: string; error?: { message?: string } } | null;
    throw new Error(body?.error?.message ?? body?.message ?? `Request failed with ${response.status}`);
  }
  return (await response.json()) as T;
}

export function loadContext(): Promise<RuntimeContext> {
  runtimeContext ??= request<RuntimeContext>("/api/v1/context").catch((error) => {
    runtimeContext = undefined;
    throw error;
  });
  return runtimeContext;
}

async function scopedPath(path: string, values: Record<string, string | number | undefined> = {}) {
  const context = await loadContext();
  const params = new URLSearchParams({ tenant_id: context.tenant.id });
  for (const [key, value] of Object.entries(values)) {
    if (value !== undefined && value !== "") params.set(key, String(value));
  }
  return `${path}?${params.toString()}`;
}

async function scopedRequest<T>(path: string, values: Record<string, string | number | undefined> = {}, init?: RequestInit): Promise<T> {
  return request<T>(await scopedPath(path, values), init);
}

function summaryValues(query: SummaryQuery) {
  return { q: query.q, status: query.status, cursor: query.cursor, limit: query.limit ?? 20 };
}

export function loadBankJourneys(): Promise<BankJourneysResponse> {
  return request<BankJourneysResponse>("/api/v1/bank-journeys");
}

export async function loadToday(): Promise<AttentionItem[]> {
  return (await request<{ items: AttentionItem[] }>("/api/v1/today")).items;
}

export async function resolveAuthority(): Promise<AuthorityResolution> {
  const context = await loadContext();
  const authorityObjectID = sampleAuthorityObjectID ?? await sampleMatterID();
  return request<AuthorityResolution>("/api/v1/authority/resolve", {
    method: "POST",
    body: JSON.stringify({
      tenant_id: context.tenant.id,
      legal_entity_id: context.legal_entity.id,
      object_type: authorityObjectID ? "MATTER" : "LEGAL_ENTITY",
      object_id: authorityObjectID ?? context.legal_entity.id,
      responsibility: "AUTHORIZER",
      materiality: 5,
    }),
  });
}

export async function loadCaptureRequest(): Promise<CaptureRequest> {
  if (sampleCaptureRequestID) {
    const value = await request<CaptureRequest>(`/api/v1/requests/${encodeURIComponent(sampleCaptureRequestID)}`);
    return { ...value, source: "legacy" };
  }
  const requestID = await sampleEvidenceRequestID();
  if (!requestID) throw new Error("No seeded evidence request is available for this role.");
  const value = await loadEvidenceRequest(requestID);
  return { ...value, source: "evidence" };
}

export async function submitCaptureRequest(id: string, version: number, answers: Record<string, string>, source: CaptureRequest["source"] = "legacy") {
  if (source === "evidence") {
    const context = await loadContext();
    return request<{ request_id: string; status: string; submitted_at: string }>(`/api/v1/evidence/requests/${encodeURIComponent(id)}/submissions?tenant_id=${encodeURIComponent(context.tenant.id)}`, { method: "POST", body: JSON.stringify({ tenant_id: context.tenant.id, expected_version: version, answers }) });
  }
  return request<{ request_id: string; status: string; submitted_at: string }>(`/api/v1/requests/${encodeURIComponent(id)}/submit`, { method: "POST", body: JSON.stringify({ version, answers }) });
}

export function loadReadiness(): Promise<Readiness> {
  return scopedRequest<Readiness>("/api/v1/compliance/readiness");
}

export async function loadAutomationPolicies(): Promise<AutomationPolicy[]> {
  return (await request<{ items: AutomationPolicy[] }>("/api/v1/compliance/automation-policies")).items;
}

export async function loadIntegrity(): Promise<IntegrityFinding[]> {
  return (await scopedRequest<{ findings: IntegrityFinding[] }>("/api/v1/authority/integrity")).findings;
}

export async function loadPolicies(): Promise<PolicySummary[]> {
  return (await scopedRequest<{ items: PolicySummary[] }>("/api/v1/authority/policies")).items;
}

export async function loadWorkflowTasks(): Promise<WorkflowTask[]> {
  return (await scopedRequest<{ items: WorkflowTask[] }>("/api/v1/workflow/tasks", { limit: 20 })).items;
}

export function loadOnboardingGuide(): Promise<OnboardingGuide> {
  return request<OnboardingGuide>("/api/v1/onboarding/guide?code=control-assurance-first-run");
}

export async function loadOnboardingState(): Promise<OnboardingState> {
  const context = await loadContext();
  return scopedRequest<OnboardingState>("/api/v1/onboarding/state", { principal_id: context.actor.id, guide_code: "control-assurance-first-run" });
}

export async function saveOnboardingState(value: Pick<OnboardingState, "current_step" | "completed" | "dismissed" | "version">): Promise<OnboardingState> {
  const context = await loadContext();
  return scopedRequest<OnboardingState>("/api/v1/onboarding/state", { principal_id: context.actor.id, guide_code: "control-assurance-first-run" }, {
    method: "PUT",
    body: JSON.stringify({ current_step: value.current_step, completed: value.completed, dismissed: value.dismissed, expected_version: value.version }),
  });
}

export async function loadEvidenceSources(): Promise<EvidenceSource[]> {
  return (await scopedRequest<{ items: EvidenceSource[] }>("/api/v1/evidence/sources", { limit: 50 })).items;
}

export async function loadEvidenceRequests(): Promise<EvidenceRequest[]> {
  return (await scopedRequest<{ items: EvidenceRequest[] }>("/api/v1/evidence/requests", { limit: 50 })).items;
}

export async function loadEvidenceRequest(id: string): Promise<EvidenceRequest> {
  return scopedRequest<EvidenceRequest>(`/api/v1/evidence/requests/${encodeURIComponent(id)}`);
}

export async function loadProjectionHealth(): Promise<ProjectionHealth[]> {
  return (await scopedRequest<{ items: ProjectionHealth[] }>("/api/v1/operations/projections")).items;
}

export async function reconcileProgramState(): Promise<ReconcileResult> {
  const context = await loadContext();
  return request<ReconcileResult>("/api/v1/operations/projections/reconcile", { method: "POST", body: JSON.stringify({ tenant_id: context.tenant.id, limit: 250 }) });
}

export function loadProgramSummaries(query: SummaryQuery = {}): Promise<SummaryPage<ProgramSummary>> {
  return scopedRequest<SummaryPage<ProgramSummary>>("/api/v1/program-summaries", summaryValues(query));
}

export function loadMatterSummaries(query: SummaryQuery = {}): Promise<SummaryPage<MatterSummary>> {
  return scopedRequest<SummaryPage<MatterSummary>>("/api/v1/matter-summaries", summaryValues(query));
}

export function loadProgram(id: string): Promise<ProgramAggregate> {
  return scopedRequest<ProgramAggregate>(`/api/v1/programs/${encodeURIComponent(id)}`);
}

export function loadMatter(id: string): Promise<MatterAggregate> {
  return scopedRequest<NullableMatterAggregate>(`/api/v1/matters/${encodeURIComponent(id)}`).then(normalizeMatterAggregate);
}

export async function loadPrograms(): Promise<ProgramAggregate[]> {
  return (await scopedRequest<{ items: ProgramAggregate[] }>("/api/v1/programs", { limit: 50 })).items;
}

export async function loadMatters(status = "OPEN"): Promise<MatterAggregate[]> {
  return (await scopedRequest<{ items: NullableMatterAggregate[] }>("/api/v1/matters", { limit: 50, status })).items.map(normalizeMatterAggregate);
}

async function sampleMatterID() {
  const journeys = await loadBankJourneys().catch(() => null);
  return journeys?.items.find((journey) => journey.action_target_type === "MATTER" && journey.action_target_id)?.action_target_id
    ?? journeys?.items.find((journey) => journey.matter_id)?.matter_id
    ?? "";
}

async function sampleEvidenceRequestID() {
  const journeys = await loadBankJourneys().catch(() => null);
  return journeys?.items.find((journey) => journey.action_target_type === "EVIDENCE_REQUEST" && journey.action_target_id)?.action_target_id
    ?? journeys?.items.find((journey) => journey.evidence_request_id)?.evidence_request_id
    ?? "";
}

type NullableMatterAggregate = Omit<MatterAggregate, "links" | "decisions" | "actions" | "verification_contracts" | "verification_results" | "response_packages" | "closure" | "matter"> & {
  matter: MatterAggregate["matter"] & { known_facts?: Record<string, unknown> | null; missing_facts?: unknown[] | null; contradictions?: unknown[] | null };
  links?: MatterAggregate["links"] | null;
  decisions?: MatterAggregate["decisions"] | null;
  actions?: MatterAggregate["actions"] | null;
  verification_contracts?: MatterAggregate["verification_contracts"] | null;
  verification_results?: MatterAggregate["verification_results"] | null;
  response_packages?: MatterAggregate["response_packages"] | null;
  closure?: MatterAggregate["closure"] | null;
};

function arrayOrEmpty<T>(value: T[] | null | undefined): T[] {
  return Array.isArray(value) ? value : [];
}

function normalizeMatterAggregate(value: NullableMatterAggregate): MatterAggregate {
  return {
    ...value,
    matter: {
      ...value.matter,
      known_facts: value.matter.known_facts ?? {},
      missing_facts: arrayOrEmpty(value.matter.missing_facts),
      contradictions: arrayOrEmpty(value.matter.contradictions),
    },
    links: arrayOrEmpty(value.links),
    decisions: arrayOrEmpty(value.decisions),
    actions: arrayOrEmpty(value.actions),
    verification_contracts: arrayOrEmpty(value.verification_contracts),
    verification_results: arrayOrEmpty(value.verification_results),
    response_packages: arrayOrEmpty(value.response_packages),
    closure: { ready: value.closure?.ready ?? false, reasons: arrayOrEmpty(value.closure?.reasons) },
  };
}
