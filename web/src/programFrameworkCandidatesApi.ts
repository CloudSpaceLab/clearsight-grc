import { loadContext } from "./api";
import { continuityCommand } from "./continuityCommands";
import { normalizeProgramAggregate } from "./programAggregate";
import type { ProgramAggregate } from "./types";
import { requestJSON } from "./http";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type FrameworkClauseMatch = {
  requirement_id: string;
  requirement_code: string;
  title: string;
  applicability?: string;
};

export type FrameworkClauseCandidate = {
  clause_id: string;
  clause_code: string;
  source_anchor: string;
  status: "UNMAPPED" | "REVIEW_CANDIDATE" | "AMBIGUOUS";
  match_basis?: "EXACT_SOURCE_ANCHOR" | "EXACT_CODE";
  total_candidates: number;
  candidates_truncated?: boolean;
  candidates: FrameworkClauseMatch[];
};

export type FrameworkClauseCandidatePage = {
  program_id: string;
  program_version: number;
  adoption_id: string;
  edition_id: string;
  next_after_code: string;
  candidates: FrameworkClauseCandidate[];
};

export async function loadFrameworkClauseCandidates(
  programID: string,
  adoptionID: string,
  afterCode = "",
): Promise<FrameworkClauseCandidatePage> {
  const context = await loadContext();
  const path = `/api/v1/programs/${encodeURIComponent(programID)}/standards/${encodeURIComponent(adoptionID)}/candidates`;
  const query = new URLSearchParams({ tenant_id: context.tenant.id, limit: "100" });
  if (afterCode) query.set("after_code", afterCode);
  return requestJSON<FrameworkClauseCandidatePage>(apiBase, `${path}?${query.toString()}`);
}

export type FrameworkMappingKind = "EXACT" | "PARTIAL" | "SUPPORTING";

export type ProgramFrameworkMapping = {
  id: string;
  clause_id: string;
  clause_code?: string;
  requirement_id: string;
  adoption_id: string;
  mapping_kind: FrameworkMappingKind;
  rationale: string;
  mapped_by: string;
  reviewed_by?: string;
  status: "PENDING" | "APPROVED" | "REJECTED";
  version: number;
};

export type ProgramFrameworkMappingPage = {
  items: ProgramFrameworkMapping[];
  next_after_id: string;
  adoption_id: string;
  program_version: number;
};

async function frameworkMutation(
  url: string,
  body: Record<string, unknown>,
): Promise<{ mapping: ProgramFrameworkMapping; program: ProgramAggregate }> {
  const result = await continuityCommand<{ mapping: ProgramFrameworkMapping; program: Parameters<typeof normalizeProgramAggregate>[0] }>(url, body);
  return { mapping: result.mapping, program: normalizeProgramAggregate(result.program) };
}

export async function loadProgramFrameworkMappings(
  programID: string, adoptionID: string, afterID = "",
): Promise<ProgramFrameworkMappingPage> {
  const context = await loadContext();
  const path = `/api/v1/programs/${encodeURIComponent(programID)}/standards/${encodeURIComponent(adoptionID)}/mappings`;
  const query = new URLSearchParams({ tenant_id: context.tenant.id, limit: "100" });
  if (afterID) query.set("after_id", afterID);
  return requestJSON<ProgramFrameworkMappingPage>(apiBase, `${path}?${query.toString()}`);
}

export function proposeProgramFrameworkMapping(
  programID: string, adoptionID: string, expectedVersion: number,
  clauseID: string, requirementID: string, mappingKind: FrameworkMappingKind, rationale: string,
) {
  const path = `/api/v1/programs/${encodeURIComponent(programID)}/standards/${encodeURIComponent(adoptionID)}/mappings/propose`;
  return frameworkMutation(path, {
    expected_version: expectedVersion, clause_id: clauseID,
    requirement_id: requirementID, mapping_kind: mappingKind, rationale,
  });
}

export function decideProgramFrameworkMapping(
  programID: string, adoptionID: string, mappingID: string, expectedVersion: number,
  action: "approve" | "reject", rationale: string,
) {
  const path = `/api/v1/programs/${encodeURIComponent(programID)}/standards/${encodeURIComponent(adoptionID)}/mappings/${encodeURIComponent(mappingID)}/${action}`;
  return frameworkMutation(path, { expected_version: expectedVersion, rationale });
}
