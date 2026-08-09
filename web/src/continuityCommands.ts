import { loadContext, resolveAuthority } from "./api";
import { requestJSON } from "./http";
import type { MatterAggregate, ProgramAggregate, WorkflowTask } from "./types";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

type JSONValue = string | number | boolean | null | JSONValue[] | { [key: string]: JSONValue };
type JSONObject = { [key: string]: JSONValue };

type MatterActionInput = {
  title: string;
  description: string;
  ownerPrincipalID?: string;
  dueAt?: string;
};

type MatterDecisionInput = {
  type: string;
  status: string;
  options?: JSONValue[];
  selectedOption?: string;
  rationale: string;
  conditions?: JSONValue[];
  expiresAt?: string;
};

type VerificationResultInput = {
  contractID: string;
  result: "PASS" | "FAIL" | "INCONCLUSIVE";
  observations?: JSONObject;
  evidenceReferences?: JSONValue[];
  rationale: string;
  observedAt?: string;
};

type ResponsePackageInput = {
  purpose: string;
  audience: string;
  manifest?: JSONObject;
};

async function command<T>(path: string, body: Record<string, unknown>): Promise<T> {
  const context = await loadContext();
  const params = new URLSearchParams({ tenant_id: context.tenant.id });
  return requestJSON<T>(apiBase, `${path}?${params.toString()}`, {
    method: "POST",
    body: JSON.stringify({ tenant_id: context.tenant.id, ...body }),
  });
}

export async function canCurrentActorTransitionProgram(programID: string): Promise<boolean> {
  const context = await loadContext();
  const resolution = await resolveAuthority({
    object_type: "PROGRAM",
    object_id: programID,
    responsibility: "AUTHORIZER",
    decision_type: "program.transition",
    materiality: 3,
  });
  return [resolution.principal, ...(resolution.candidate_principals ?? [])]
    .some((candidate) => candidate?.id === context.actor.id);
}

export async function loadActorMatterWork(limit = 100): Promise<WorkflowTask[]> {
  const context = await loadContext();
  const params = new URLSearchParams({ tenant_id: context.tenant.id, limit: String(limit) });
  return (await requestJSON<{ items: WorkflowTask[] }>(apiBase, `/api/v1/workflow/tasks?${params.toString()}`)).items;
}

export function transitionProgram(programID: string, expectedVersion: number, to: string, rationale: string): Promise<ProgramAggregate> {
  return command<ProgramAggregate>(`/api/v1/programs/${encodeURIComponent(programID)}/transition`, {
    expected_version: expectedVersion,
    to,
    rationale,
  });
}

export function transitionMatter(matterID: string, expectedVersion: number, to: string, rationale: string): Promise<MatterAggregate> {
  return command<MatterAggregate>(`/api/v1/matters/${encodeURIComponent(matterID)}/transition`, {
    expected_version: expectedVersion,
    to,
    rationale,
  });
}

export function addMatterAction(matterID: string, expectedVersion: number, input: MatterActionInput): Promise<MatterAggregate> {
  return command<MatterAggregate>(`/api/v1/matters/${encodeURIComponent(matterID)}/actions`, {
    expected_version: expectedVersion,
    title: input.title,
    description: input.description,
    owner_principal_id: input.ownerPrincipalID,
    due_at: input.dueAt,
  });
}

export function transitionMatterAction(matterID: string, actionID: string, expectedVersion: number, to: string, rationale: string): Promise<MatterAggregate> {
  return command<MatterAggregate>(`/api/v1/matters/${encodeURIComponent(matterID)}/actions/${encodeURIComponent(actionID)}/transition`, {
    expected_version: expectedVersion,
    to,
    rationale,
  });
}

export function recordMatterDecision(matterID: string, expectedVersion: number, input: MatterDecisionInput): Promise<MatterAggregate> {
  return command<MatterAggregate>(`/api/v1/matters/${encodeURIComponent(matterID)}/decisions`, {
    expected_version: expectedVersion,
    type: input.type,
    status: input.status,
    options: input.options ?? [],
    selected_option: input.selectedOption,
    rationale: input.rationale,
    conditions: input.conditions ?? [],
    expires_at: input.expiresAt,
  });
}

export function recordVerificationResult(matterID: string, expectedVersion: number, input: VerificationResultInput): Promise<MatterAggregate> {
  return command<MatterAggregate>(`/api/v1/matters/${encodeURIComponent(matterID)}/verification-results`, {
    expected_version: expectedVersion,
    contract_id: input.contractID,
    result: input.result,
    observations: input.observations ?? {},
    evidence_references: input.evidenceReferences ?? [],
    rationale: input.rationale,
    observed_at: input.observedAt ?? new Date().toISOString(),
  });
}

export function addResponsePackage(matterID: string, expectedVersion: number, input: ResponsePackageInput): Promise<MatterAggregate> {
  return command<MatterAggregate>(`/api/v1/matters/${encodeURIComponent(matterID)}/responses`, {
    expected_version: expectedVersion,
    purpose: input.purpose,
    audience: input.audience,
    manifest: input.manifest ?? {},
  });
}

export function transitionResponsePackage(matterID: string, responseID: string, expectedVersion: number, to: string, rationale: string): Promise<MatterAggregate> {
  return command<MatterAggregate>(`/api/v1/matters/${encodeURIComponent(matterID)}/responses/${encodeURIComponent(responseID)}/transition`, {
    expected_version: expectedVersion,
    to,
    rationale,
  });
}
