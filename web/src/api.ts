import type { AttentionItem, AuthorityResolution, CaptureRequest, EvidenceRequest, EvidenceSource, IntegrityFinding, MatterAggregate, OnboardingGuide, OnboardingState, PolicySummary, ProgramAggregate, Readiness, WorkflowTask } from "./types";
import type { MatterSummary, ProgramSummary, SummaryPage, SummaryQuery } from "./summaryTypes";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${apiBase}${path}`, { ...init, headers: { "Content-Type": "application/json", ...init?.headers } });
  if (!response.ok) {
    const body = (await response.json().catch(() => null)) as { message?: string } | null;
    throw new Error(body?.message ?? `Request failed with ${response.status}`);
  }
  return (await response.json()) as T;
}

function summaryQuery(path: string, query: SummaryQuery) {
  const params = new URLSearchParams({ tenant_id: "bank-demo", limit: String(query.limit ?? 20) });
  if (query.q) params.set("q", query.q);
  if (query.status) params.set("status", query.status);
  if (query.cursor) params.set("cursor", query.cursor);
  return `${path}?${params.toString()}`;
}

export async function loadToday(): Promise<AttentionItem[]> { return (await request<{ items: AttentionItem[] }>("/api/v1/today")).items; }
export function resolveAuthority(): Promise<AuthorityResolution> { return request<AuthorityResolution>("/api/v1/authority/resolve", { method: "POST", body: JSON.stringify({ tenant_id: "bank-demo", legal_entity_id: "bank-ng", object_type: "MATTER", object_id: "matter-demo", responsibility: "AUTHORIZER", materiality: 5 }) }); }
export function loadCaptureRequest(): Promise<CaptureRequest> { return request<CaptureRequest>("/api/v1/requests/req_branch_generator"); }
export function submitCaptureRequest(id: string, version: number, answers: Record<string, string>) { return request<{ request_id: string; status: string; submitted_at: string }>(`/api/v1/requests/${id}/submit`, { method: "POST", body: JSON.stringify({ version, answers }) }); }
export function loadReadiness(): Promise<Readiness> { return request<Readiness>("/api/v1/compliance/readiness?tenant_id=bank-demo"); }
export async function loadIntegrity(): Promise<IntegrityFinding[]> { return (await request<{ findings: IntegrityFinding[] }>("/api/v1/authority/integrity?tenant_id=bank-demo")).findings; }
export async function loadPolicies(): Promise<PolicySummary[]> { return (await request<{ items: PolicySummary[] }>("/api/v1/authority/policies?tenant_id=bank-demo")).items; }
export async function loadWorkflowTasks(): Promise<WorkflowTask[]> { return (await request<{ items: WorkflowTask[] }>("/api/v1/workflow/tasks?tenant_id=bank-demo&limit=20")).items; }
export function loadOnboardingGuide(): Promise<OnboardingGuide> { return request<OnboardingGuide>("/api/v1/onboarding/guide?code=control-assurance-first-run"); }
export function loadOnboardingState(): Promise<OnboardingState> { return request<OnboardingState>("/api/v1/onboarding/state?tenant_id=bank-demo&principal_id=user-demo&guide_code=control-assurance-first-run"); }
export function saveOnboardingState(value: Pick<OnboardingState, "current_step" | "completed" | "dismissed" | "version">): Promise<OnboardingState> { return request<OnboardingState>("/api/v1/onboarding/state?tenant_id=bank-demo&principal_id=user-demo&guide_code=control-assurance-first-run", { method: "PUT", body: JSON.stringify({ current_step: value.current_step, completed: value.completed, dismissed: value.dismissed, expected_version: value.version }) }); }
export async function loadEvidenceSources(): Promise<EvidenceSource[]> { return (await request<{ items: EvidenceSource[] }>("/api/v1/evidence/sources?tenant_id=bank-demo&limit=50")).items; }
export async function loadEvidenceRequests(): Promise<EvidenceRequest[]> { return (await request<{ items: EvidenceRequest[] }>("/api/v1/evidence/requests?tenant_id=bank-demo&limit=50")).items; }

export function loadProgramSummaries(query: SummaryQuery = {}): Promise<SummaryPage<ProgramSummary>> {
  return request<SummaryPage<ProgramSummary>>(summaryQuery("/api/v1/program-summaries", query));
}

export function loadMatterSummaries(query: SummaryQuery = {}): Promise<SummaryPage<MatterSummary>> {
  return request<SummaryPage<MatterSummary>>(summaryQuery("/api/v1/matter-summaries", query));
}

export function loadProgram(id: string): Promise<ProgramAggregate> {
  return request<ProgramAggregate>(`/api/v1/programs/${encodeURIComponent(id)}?tenant_id=bank-demo`);
}

export function loadMatter(id: string): Promise<MatterAggregate> {
  return request<MatterAggregate>(`/api/v1/matters/${encodeURIComponent(id)}?tenant_id=bank-demo`);
}

// Full aggregate list helpers remain for compatibility and diagnostics. Primary
// workspaces use summary endpoints and fetch an aggregate only when opened.
export async function loadPrograms(): Promise<ProgramAggregate[]> { return (await request<{ items: ProgramAggregate[] }>("/api/v1/programs?tenant_id=bank-demo&limit=50")).items; }
export async function loadMatters(status = "OPEN"): Promise<MatterAggregate[]> { const suffix = status ? `&status=${encodeURIComponent(status)}` : ""; return (await request<{ items: MatterAggregate[] }>(`/api/v1/matters?tenant_id=bank-demo&limit=50${suffix}`)).items; }
