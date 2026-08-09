import { requestJSON } from "./http";
import type { ProgramState, StateReason } from "./types";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type ProgramReviewCheckpoint = {
  id: string;
  tenant_id: string;
  program_id: string;
  principal_id: string;
  program_version: number;
  projection_version: number;
  accepted_at: string;
};

export type ProgramReviewChange = {
  kind: string;
  summary: string;
  object_type?: string;
  object_id?: string;
};

export type ProgramReviewDigest = {
  program_id: string;
  state: "NO_BASELINE" | "CHANGED" | "CURRENT" | string;
  review_required: boolean;
  checkpoint?: ProgramReviewCheckpoint;
  current_program_version: number;
  current_projection_version: number;
  current_overall: ProgramState;
  baseline_overall?: ProgramState;
  open_matter_count: number;
  open_matter_delta?: number;
  changes: ProgramReviewChange[];
  changes_total: number;
  changes_omitted: number;
  history_truncated: boolean;
  current_exceptions: StateReason[];
  current_exceptions_total: number;
  new_exceptions: StateReason[];
  new_exceptions_total: number;
  resolved_exceptions: StateReason[];
  resolved_exceptions_total: number;
};

export function loadProgramReviewDigest(programID: string): Promise<ProgramReviewDigest> {
  return requestJSON<ProgramReviewDigest>(apiBase, `/api/v1/programs/${encodeURIComponent(programID)}/review-digest`);
}

export function acceptProgramReview(programID: string, expectedProgramVersion: number, expectedProjectionVersion: number): Promise<ProgramReviewDigest> {
  return requestJSON<ProgramReviewDigest>(apiBase, `/api/v1/programs/${encodeURIComponent(programID)}/reviews`, {
    method: "POST",
    body: JSON.stringify({
      expected_program_version: expectedProgramVersion,
      expected_projection_version: expectedProjectionVersion,
    }),
  });
}
