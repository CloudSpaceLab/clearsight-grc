import type { FormScoreProfile, FormTemplateField } from "./monitoringTypes";
import type { ResponseScore } from "./formsDistributionApi";
import type { CaptureAnswerValue } from "./types";
import { requestJSON } from "./http";

export type FieldAssessmentDecision = { id: string; field_id: string; outcome_id: string; points: number; rationale: string; reviewer_id: string; assessed_at: string; supersedes_id?: string };
export type ResponseFieldAssessment = { field: FormTemplateField; answer: CaptureAnswerValue; decision?: FieldAssessmentDecision; may_review?: boolean };
export type ResponseAssessmentDetail = {
  response_id: string; form_template_id: string; form_template_version: number; version: number; current: boolean;
  state: "NOT_REQUIRED" | "AWAITING_REVIEW" | "IN_REVIEW" | "ASSESSED";
  required_count: number; reviewed_required_count: number; reviewed_count: number;
  fields: ResponseFieldAssessment[]; automatic_score?: ResponseScore; assessed_score?: ResponseScore; may_review?: boolean;
  score_profile?: FormScoreProfile;
};
export type RecordResponseAssessmentInput = { expected_version: number; decisions: Array<{ field_id: string; outcome_id: string; rationale: string }> };
const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";
export function loadResponseAssessment(responseID: string): Promise<ResponseAssessmentDetail> {
  return requestJSON(apiBase, `/api/v1/forms/responses/${encodeURIComponent(responseID)}/assessment`);
}
export function recordResponseAssessment(responseID: string, input: RecordResponseAssessmentInput): Promise<ResponseAssessmentDetail> {
  return requestJSON(apiBase, `/api/v1/forms/responses/${encodeURIComponent(responseID)}/assessment`, {
    method: "POST", body: JSON.stringify({ expected_version: input.expected_version, decisions: input.decisions.map(({ field_id, outcome_id, rationale }) => ({ field_id, outcome_id, rationale })) }),
  });
}
