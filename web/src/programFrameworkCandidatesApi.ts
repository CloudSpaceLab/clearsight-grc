import { loadContext } from "./api";
import { requestJSON } from "./http";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type FrameworkClauseMatch = {
  requirement_id: string;
  requirement_code: string;
  title: string;
  applicability?: string;
};

export type FrameworkClauseCandidate = {
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
