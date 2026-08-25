import { loadContext, resolveAuthority } from "./api";
import { requestJSON } from "./http";
import type { MatterAggregate, ProgramAggregate, WorkflowTask } from "./types";
import { normalizeProgramAggregate } from "./programAggregate";

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

type CreateProgramInput = {
  code: string;
  name: string;
  type: string;
  owningFunction: string;
  jurisdiction?: string;
  scopeDescription?: string;
};

type AddRequirementInput = {
  code: string;
  title: string;
  statement: string;
  sourceAnchor?: string;
};

export type CreateMatterInput = {
  type: string;
  priority: number;
  title: string;
  summary: string;
  affectedArea: string;
  knownInformation?: string;
  missingInformation?: string[];
  dueAt?: string;
  programID?: string;
};

export async function continuityCommand<T>(path: string, body: Record<string, unknown>): Promise<T> {
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

export async function createProgram(input: CreateProgramInput): Promise<ProgramAggregate> {
  const context = await loadContext();
  const value = await continuityCommand<Parameters<typeof normalizeProgramAggregate>[0]>("/api/v1/programs", {
    legal_entity_id: context.legal_entity.id,
    code: input.code,
    name: input.name,
    type: input.type,
    owning_function: input.owningFunction,
    owner_principal_id: context.actor.id,
    authority_principal_id: context.actor.id,
    jurisdiction: input.jurisdiction,
    scope: { description: input.scopeDescription ?? "" },
    effective_from: new Date().toISOString(),
  });
  return normalizeProgramAggregate(value);
}

export async function createMatter(input: CreateMatterInput): Promise<MatterAggregate> {
  const context = await loadContext();
  return continuityCommand<MatterAggregate>("/api/v1/matters", {
    type: input.type,
    priority: input.priority,
    title: input.title,
    summary: input.summary,
    scope: { access: "INTERNAL", area: input.affectedArea },
    known_facts: input.knownInformation ? { notes: input.knownInformation } : {},
    missing_facts: input.missingInformation ?? [],
    contradictions: [],
    owner_principal_id: context.actor.id,
    due_at: input.dueAt,
    program_id: input.programID,
  });
}

export async function addProgramRequirement(programID: string, expectedVersion: number, input: AddRequirementInput): Promise<ProgramAggregate> {
  const value = await continuityCommand<Parameters<typeof normalizeProgramAggregate>[0]>(`/api/v1/programs/${encodeURIComponent(programID)}/requirements`, {
    expected_version: expectedVersion,
    code: input.code,
    title: input.title,
    statement: input.statement,
    source_anchor: input.sourceAnchor,
    modality: "MUST",
    actor: "The bank",
    action: "maintain the stated safeguard",
    object: "the monitored channel",
    status: "APPROVED",
    effective_from: new Date().toISOString(),
  });
  return normalizeProgramAggregate(value);
}

export async function transitionProgram(programID: string, expectedVersion: number, to: string, rationale: string): Promise<ProgramAggregate> {
  const value = await continuityCommand<Parameters<typeof normalizeProgramAggregate>[0]>(`/api/v1/programs/${encodeURIComponent(programID)}/transition`, {
    expected_version: expectedVersion,
    to,
    rationale,
  });
  return normalizeProgramAggregate(value);
}

export function transitionMatter(matterID: string, expectedVersion: number, to: string, rationale: string): Promise<MatterAggregate> {
  return continuityCommand<MatterAggregate>(`/api/v1/matters/${encodeURIComponent(matterID)}/transition`, {
    expected_version: expectedVersion,
    to,
    rationale,
  });
}

export function addMatterAction(matterID: string, expectedVersion: number, input: MatterActionInput): Promise<MatterAggregate> {
  return continuityCommand<MatterAggregate>(`/api/v1/matters/${encodeURIComponent(matterID)}/actions`, {
    expected_version: expectedVersion,
    title: input.title,
    description: input.description,
    owner_principal_id: input.ownerPrincipalID,
    due_at: input.dueAt,
  });
}

export function transitionMatterAction(matterID: string, actionID: string, expectedVersion: number, to: string, rationale: string): Promise<MatterAggregate> {
  return continuityCommand<MatterAggregate>(`/api/v1/matters/${encodeURIComponent(matterID)}/actions/${encodeURIComponent(actionID)}/transition`, {
    expected_version: expectedVersion,
    to,
    rationale,
  });
}

export function recordMatterDecision(matterID: string, expectedVersion: number, input: MatterDecisionInput): Promise<MatterAggregate> {
  return continuityCommand<MatterAggregate>(`/api/v1/matters/${encodeURIComponent(matterID)}/decisions`, {
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
  return continuityCommand<MatterAggregate>(`/api/v1/matters/${encodeURIComponent(matterID)}/verification-results`, {
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
  return continuityCommand<MatterAggregate>(`/api/v1/matters/${encodeURIComponent(matterID)}/responses`, {
    expected_version: expectedVersion,
    purpose: input.purpose,
    audience: input.audience,
    manifest: input.manifest ?? {},
  });
}

export function transitionResponsePackage(matterID: string, responseID: string, expectedVersion: number, to: string, rationale: string): Promise<MatterAggregate> {
  return continuityCommand<MatterAggregate>(`/api/v1/matters/${encodeURIComponent(matterID)}/responses/${encodeURIComponent(responseID)}/transition`, {
    expected_version: expectedVersion,
    to,
    rationale,
  });
}
