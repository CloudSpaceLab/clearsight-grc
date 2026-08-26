import { loadContext } from "./api";
import { continuityCommand } from "./continuityCommands";
import { requestJSON } from "./http";
import type { AuthorityPrincipal, MatterAggregate, RecordResponsibleParty } from "./types";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type MatterOperation = {
  command: string;
  subresource_id?: string;
  label: string;
  responsibility: string;
  can_act: boolean;
  assigned_to?: AuthorityPrincipal;
  candidates?: AuthorityPrincipal[];
  reason: string;
  allowed_targets?: string[];
};

export type MatterOperations = {
  matter_id: string;
  matter_version: number;
  authority_available: boolean;
  operations: MatterOperation[];
  responsible_parties?: RecordResponsibleParty[];
  generated_at: string;
};

export type MatterContextChangeKind =
  | "ADD_FACT"
  | "CORRECT_FACT"
  | "RETIRE_FACT"
  | "ADD_MISSING"
  | "RESOLVE_MISSING"
  | "ADD_CONTRADICTION"
  | "RESOLVE_CONTRADICTION";

type MatterDetailsInput = {
  title: string;
  summary: string;
  priority: number;
  dueAt?: string;
  scope: Record<string, unknown>;
  rationale: string;
};

type MatterContextChangeInput = {
  kind: MatterContextChangeKind;
  key?: string;
  label: string;
  value?: string | number | boolean | null;
  rationale: string;
  evidenceReferences?: unknown[];
};

type MatterActionUpdateInput = {
  title: string;
  description: string;
  dueAt?: string;
  rationale: string;
};

type MatterOutcomeCheckInput = {
  actionID?: string;
  expectedOutcome: string;
  observationPeriodMinutes: number;
  failureResponse: string;
  baseline?: Record<string, unknown>;
  scope?: Record<string, unknown>;
  threshold?: Record<string, unknown>;
  measurementSourceID?: string;
  reviewerCandidateID: string;
};

type MatterOutcomeCheckRevisionInput = MatterOutcomeCheckInput & {
  rationale: string;
};

type MatterLinkInput = {
  programID: string;
  requirementID?: string;
  controlID?: string;
  relationship: string;
};

async function tenantPath(path: string) {
  const context = await loadContext();
  return { context, path: `${path}?${new URLSearchParams({ tenant_id: context.tenant.id }).toString()}` };
}

export async function loadMatterOperations(matterID: string): Promise<MatterOperations> {
  const scoped = await tenantPath(`/api/v1/matters/${encodeURIComponent(matterID)}/operations`);
  return requestJSON<MatterOperations>(apiBase, scoped.path);
}

export function updateMatterDetails(matterID: string, expectedVersion: number, input: MatterDetailsInput): Promise<MatterAggregate> {
  return continuityCommand<MatterAggregate>(`/api/v1/matters/${encodeURIComponent(matterID)}/details`, {
    expected_version: expectedVersion,
    title: input.title,
    summary: input.summary,
    priority: input.priority,
    due_at: input.dueAt,
    scope: input.scope,
    rationale: input.rationale,
  });
}

export function changeMatterContext(matterID: string, expectedVersion: number, input: MatterContextChangeInput): Promise<MatterAggregate> {
  return continuityCommand<MatterAggregate>(`/api/v1/matters/${encodeURIComponent(matterID)}/context-changes`, {
    expected_version: expectedVersion,
    kind: input.kind,
    key: input.key,
    label: input.label,
    value: input.value,
    evidence_references: input.evidenceReferences ?? [],
    rationale: input.rationale,
  });
}

export function assignMatter(matterID: string, expectedVersion: number, ownerPrincipalID: string, rationale: string): Promise<MatterAggregate> {
  return continuityCommand<MatterAggregate>(`/api/v1/matters/${encodeURIComponent(matterID)}/assignment`, {
    expected_version: expectedVersion,
    owner_principal_id: ownerPrincipalID,
    rationale,
  });
}

export function updateMatterAction(matterID: string, actionID: string, expectedVersion: number, input: MatterActionUpdateInput): Promise<MatterAggregate> {
  return continuityCommand<MatterAggregate>(`/api/v1/matters/${encodeURIComponent(matterID)}/actions/${encodeURIComponent(actionID)}`, {
    expected_version: expectedVersion,
    title: input.title,
    description: input.description,
    due_at: input.dueAt,
    rationale: input.rationale,
  });
}

export function assignMatterAction(matterID: string, actionID: string, expectedVersion: number, ownerPrincipalID: string, rationale: string): Promise<MatterAggregate> {
  return continuityCommand<MatterAggregate>(`/api/v1/matters/${encodeURIComponent(matterID)}/actions/${encodeURIComponent(actionID)}/assignment`, {
    expected_version: expectedVersion,
    owner_principal_id: ownerPrincipalID,
    rationale,
  });
}

export function defineMatterOutcomeCheck(matterID: string, expectedVersion: number, input: MatterOutcomeCheckInput): Promise<MatterAggregate> {
  return continuityCommand<MatterAggregate>(`/api/v1/matters/${encodeURIComponent(matterID)}/verification-contracts`, {
    expected_version: expectedVersion,
    action_id: input.actionID,
    expected_outcome: input.expectedOutcome,
    baseline: input.baseline ?? {},
    scope: input.scope ?? {},
    measurement_source_id: input.measurementSourceID,
    threshold: input.threshold ?? {},
    observation_period_minutes: input.observationPeriodMinutes,
    reviewer_candidate_id: input.reviewerCandidateID,
    failure_response: input.failureResponse,
  });
}

export function supersedeMatterOutcomeCheck(matterID: string, contractID: string, expectedVersion: number, input: MatterOutcomeCheckRevisionInput): Promise<MatterAggregate> {
  return continuityCommand<MatterAggregate>(`/api/v1/matters/${encodeURIComponent(matterID)}/verification-contracts/${encodeURIComponent(contractID)}/supersede`, {
    expected_version: expectedVersion,
    action_id: input.actionID,
    expected_outcome: input.expectedOutcome,
    baseline: input.baseline ?? {},
    scope: input.scope ?? {},
    measurement_source_id: input.measurementSourceID,
    threshold: input.threshold ?? {},
    observation_period_minutes: input.observationPeriodMinutes,
    reviewer_candidate_id: input.reviewerCandidateID,
    failure_response: input.failureResponse,
    rationale: input.rationale,
  });
}

export function retireMatterOutcomeCheck(matterID: string, contractID: string, expectedVersion: number, rationale: string): Promise<MatterAggregate> {
  return continuityCommand<MatterAggregate>(`/api/v1/matters/${encodeURIComponent(matterID)}/verification-contracts/${encodeURIComponent(contractID)}/retire`, {
    expected_version: expectedVersion,
    rationale,
  });
}

export function addMatterLink(matterID: string, expectedVersion: number, input: MatterLinkInput): Promise<MatterAggregate> {
  return continuityCommand<MatterAggregate>(`/api/v1/matters/${encodeURIComponent(matterID)}/links`, {
    expected_version: expectedVersion,
    program_id: input.programID,
    requirement_id: input.requirementID,
    control_id: input.controlID,
    relationship: input.relationship,
  });
}
