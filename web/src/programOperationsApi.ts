import { loadContext } from "./api";
import { continuityCommand } from "./continuityCommands";
import { requestJSON } from "./http";
import { normalizeProgramAggregate } from "./programAggregate";
import type { AuthorityPrincipal, ProgramAggregate, RecordResponsibleParty } from "./types";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type ProgramOperation = {
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

export type ProgramOperations = {
  program_id: string;
  program_version: number;
  authority_available: boolean;
  operations: ProgramOperation[];
  responsible_parties?: RecordResponsibleParty[];
  generated_at: string;
};

export type ProgramDetailsInput = {
  name: string;
  owningFunction: string;
  jurisdiction?: string;
  scope: Record<string, unknown>;
  effectiveFrom: string;
  effectiveUntil?: string;
  rationale: string;
};

export type ProgramRequirementInput = {
  sourceID?: string;
  code: string;
  title: string;
  statement: string;
  sourceAnchor: string;
  modality: string;
  actor?: string;
  action?: string;
  object?: string;
  effectiveFrom: string;
};

export type ProgramRequirementSupersessionInput = ProgramRequirementInput & { rationale: string };

export type ProgramApplicabilityInput = {
  requirementID: string;
  status: string;
  scope: Record<string, unknown>;
  rationale: string;
  effectiveFrom: string;
};

export type ProgramControlObjectiveInput = {
  code: string;
  name: string;
  outcome: string;
  status: string;
};

export type ProgramControlImplementationInput = {
  objectiveID: string;
  name: string;
  description: string;
  implementationType: string;
  ownerPrincipalID?: string;
  scope: Record<string, unknown>;
  status: string;
  effectiveFrom: string;
};

export type ProgramEvidenceContractInput = {
  requirementID?: string;
  controlImplementationID?: string;
  code: string;
  name: string;
  claim: string;
  acceptableSourceIDs: string[];
  populationScope: Record<string, unknown>;
  freshnessMinutes: number;
  minimumCoverage: number;
  independenceRequired: boolean;
  contradictionPolicy: string;
  failureAction: string;
  status: string;
};

export type ProgramEvidenceAssessmentInput = {
  contractID: string;
  conclusion: string;
  coverage: number;
  basis: Record<string, unknown>;
  validUntil?: string;
  assessedAt: string;
};

async function tenantPath(path: string) {
  const context = await loadContext();
  return `${path}?${new URLSearchParams({ tenant_id: context.tenant.id }).toString()}`;
}

async function programCommand(path: string, body: Record<string, unknown>): Promise<ProgramAggregate> {
  const value = await continuityCommand<Parameters<typeof normalizeProgramAggregate>[0]>(path, body);
  return normalizeProgramAggregate(value);
}

export async function loadProgramOperations(programID: string): Promise<ProgramOperations> {
  return requestJSON<ProgramOperations>(apiBase, await tenantPath(`/api/v1/programs/${encodeURIComponent(programID)}/operations`));
}

export function updateProgramDetails(programID: string, expectedVersion: number, input: ProgramDetailsInput): Promise<ProgramAggregate> {
  return programCommand(`/api/v1/programs/${encodeURIComponent(programID)}/details`, {
    expected_version: expectedVersion,
    name: input.name,
    owning_function: input.owningFunction,
    jurisdiction: input.jurisdiction,
    scope: input.scope,
    effective_from: input.effectiveFrom,
    effective_until: input.effectiveUntil,
    rationale: input.rationale,
  });
}

export function assignProgram(programID: string, expectedVersion: number, ownerPrincipalID: string, rationale: string): Promise<ProgramAggregate> {
  return programCommand(`/api/v1/programs/${encodeURIComponent(programID)}/assignment`, {
    expected_version: expectedVersion,
    owner_principal_id: ownerPrincipalID,
    rationale,
  });
}

export function assignProgramApprovalAuthority(programID: string, expectedVersion: number, candidateID: string, rationale: string): Promise<ProgramAggregate> {
  return programCommand(`/api/v1/programs/${encodeURIComponent(programID)}/approval-authority`, {
    expected_version: expectedVersion,
    candidate_id: candidateID,
    rationale,
  });
}

export function supersedeProgramRequirement(programID: string, requirementID: string, expectedVersion: number, input: ProgramRequirementSupersessionInput): Promise<ProgramAggregate> {
  return programCommand(`/api/v1/programs/${encodeURIComponent(programID)}/requirements/${encodeURIComponent(requirementID)}/supersede`, {
    expected_version: expectedVersion,
    source_id: input.sourceID,
    code: input.code,
    title: input.title,
    statement: input.statement,
    source_anchor: input.sourceAnchor,
    modality: input.modality,
    actor: input.actor,
    action: input.action,
    object: input.object,
    effective_from: input.effectiveFrom,
    rationale: input.rationale,
  });
}

export function addProgramRequirement(programID: string, expectedVersion: number, input: ProgramRequirementInput): Promise<ProgramAggregate> {
  return programCommand(`/api/v1/programs/${encodeURIComponent(programID)}/requirements`, {
    expected_version: expectedVersion,
    source_id: input.sourceID,
    code: input.code,
    title: input.title,
    statement: input.statement,
    source_anchor: input.sourceAnchor,
    modality: input.modality,
    actor: input.actor,
    action: input.action,
    object: input.object,
    status: "APPROVED",
    effective_from: input.effectiveFrom,
  });
}

export function determineProgramApplicability(programID: string, expectedVersion: number, input: ProgramApplicabilityInput): Promise<ProgramAggregate> {
  return programCommand(`/api/v1/programs/${encodeURIComponent(programID)}/applicability`, {
    expected_version: expectedVersion,
    requirement_id: input.requirementID,
    status: input.status,
    scope: input.scope,
    rationale: input.rationale,
    effective_from: input.effectiveFrom,
  });
}

export function addProgramControlObjective(programID: string, expectedVersion: number, input: ProgramControlObjectiveInput): Promise<ProgramAggregate> {
  return programCommand(`/api/v1/programs/${encodeURIComponent(programID)}/control-objectives`, {
    expected_version: expectedVersion,
    code: input.code,
    name: input.name,
    outcome: input.outcome,
    status: input.status,
  });
}

export function addProgramControlImplementation(programID: string, expectedVersion: number, input: ProgramControlImplementationInput): Promise<ProgramAggregate> {
  return programCommand(`/api/v1/programs/${encodeURIComponent(programID)}/control-implementations`, {
    expected_version: expectedVersion,
    objective_id: input.objectiveID,
    name: input.name,
    description: input.description,
    implementation_type: input.implementationType,
    owner_principal_id: input.ownerPrincipalID,
    scope: input.scope,
    status: input.status,
    effective_from: input.effectiveFrom,
  });
}

export function linkProgramRequirementControl(programID: string, expectedVersion: number, requirementID: string, implementationID: string): Promise<ProgramAggregate> {
  return programCommand(`/api/v1/programs/${encodeURIComponent(programID)}/control-links`, {
    expected_version: expectedVersion,
    requirement_id: requirementID,
    implementation_id: implementationID,
  });
}

export function addProgramEvidenceContract(programID: string, expectedVersion: number, input: ProgramEvidenceContractInput): Promise<ProgramAggregate> {
  return programCommand(`/api/v1/programs/${encodeURIComponent(programID)}/evidence-contracts`, {
    expected_version: expectedVersion,
    requirement_id: input.requirementID,
    control_implementation_id: input.controlImplementationID,
    code: input.code,
    name: input.name,
    claim: input.claim,
    acceptable_source_ids: input.acceptableSourceIDs,
    population_scope: input.populationScope,
    freshness_minutes: input.freshnessMinutes,
    minimum_coverage: input.minimumCoverage,
    independence_required: input.independenceRequired,
    contradiction_policy: input.contradictionPolicy,
    failure_action: input.failureAction,
    status: input.status,
  });
}

export function recordProgramEvidenceAssessment(programID: string, expectedVersion: number, input: ProgramEvidenceAssessmentInput): Promise<ProgramAggregate> {
  return programCommand(`/api/v1/programs/${encodeURIComponent(programID)}/evidence-assessments`, {
    expected_version: expectedVersion,
    contract_id: input.contractID,
    conclusion: input.conclusion,
    coverage: input.coverage,
    basis: input.basis,
    valid_until: input.validUntil,
    assessed_at: input.assessedAt,
  });
}

export function transitionProgram(programID: string, expectedVersion: number, to: string, rationale: string): Promise<ProgramAggregate> {
  return programCommand(`/api/v1/programs/${encodeURIComponent(programID)}/transition`, {
    expected_version: expectedVersion,
    to,
    rationale,
  });
}
