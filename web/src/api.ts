import type { AttentionItem, AuthorityResolution, CaptureRequest, EvidenceRequest, EvidenceSource, IntegrityFinding, MatterAggregate, OnboardingGuide, OnboardingState, PolicySummary, ProgramAggregate, Readiness, WorkflowTask } from "./types";
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
  return request<AuthorityResolution>("/api/v1/authority/resolve", {
    method: "POST",
    body: JSON.stringify({
      tenant_id: context.tenant.id,
      legal_entity_id: context.legal_entity.id,
      object_type: sampleAuthorityObjectID ? "MATTER" : "LEGAL_ENTITY",
      object_id: sampleAuthorityObjectID ?? context.legal_entity.id,
      responsibility: "AUTHORIZER",
      materiality: 5,
    }),
  });
}

export function loadCaptureRequest(): Promise<CaptureRequest> {
  if (!sampleCaptureRequestID) return Promise.reject(new Error("No sample capture request is configured for this build."));
  return request<CaptureRequest>(`/api/v1/requests/${encodeURIComponent(sampleCaptureRequestID)}`);
}

export function submitCaptureRequest(id: string, version: number, answers: Record<string, string>) {
  return request<{ request_id: string; status: string; submitted_at: string }>(`/api/v1/requests/${encodeURIComponent(id)}/submit`, { method: "POST", body: JSON.stringify({ version, answers }) });
}

export function loadReadiness(): Promise<Readiness> {
  return scopedRequest<Readiness>("/api/v1/compliance/readiness");
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
  return scopedRequest<MatterAggregate>(`/api/v1/matters/${encodeURIComponent(id)}`);
}

export async function loadPrograms(): Promise<ProgramAggregate[]> {
  return (await scopedRequest<{ items: ProgramAggregate[] }>("/api/v1/programs", { limit: 50 })).items;
}

export async function loadMatters(status = "OPEN"): Promise<MatterAggregate[]> {
  return (await scopedRequest<{ items: MatterAggregate[] }>("/api/v1/matters", { limit: 50, status })).items;
}
