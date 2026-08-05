import type { AttentionItem, AuthorityResolution, CaptureRequest, EvidenceRequest, EvidenceSource, IntegrityFinding, MatterAggregate, OnboardingGuide, OnboardingState, PolicySummary, ProgramAggregate, Readiness, WorkflowTask } from "./types";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";
async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${apiBase}${path}`, { ...init, headers: { "Content-Type": "application/json", ...init?.headers } });
  if (!response.ok) {
    const body = (await response.json().catch(() => null)) as { message?: string } | null;
    throw new Error(body?.message ?? `Request failed with ${response.status}`);
  }
  return (await response.json()) as T;
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
export async function loadPrograms(): Promise<ProgramAggregate[]> { return (await request<{ items: ProgramAggregate[] }>("/api/v1/programs?tenant_id=bank-demo&limit=50")).items; }
export async function loadMatters(status = "OPEN"): Promise<MatterAggregate[]> { const suffix = status ? `&status=${encodeURIComponent(status)}` : ""; return (await request<{ items: MatterAggregate[] }>(`/api/v1/matters?tenant_id=bank-demo&limit=50${suffix}`)).items; }
