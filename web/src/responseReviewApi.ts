import { requestJSON } from "./http";

export type ResponseReviewDecision = "REVIEWED" | "CHANGES_REQUESTED";
export type ResponseReviewRecord = { id: string; version: number; decision: ResponseReviewDecision; comment: string; reviewer_id: string; reviewer_display_name?: string; reviewed_at: string };
export type ResponseReviewPage = { version: number; may_review: boolean; items: ResponseReviewRecord[]; next_before_version?: number };
export type RecordResponseReviewInput = { expected_version: number; decision: ResponseReviewDecision; comment: string };
const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";
const path = (id: string) => `/api/v1/forms/responses/${encodeURIComponent(id)}/review`;

export function loadResponseReviews(id: string, before?: number): Promise<ResponseReviewPage> {
  return requestJSON(apiBase, path(id) + (before ? `?before_version=${encodeURIComponent(before)}` : ""));
}
export function recordResponseReview(id: string, input: RecordResponseReviewInput): Promise<ResponseReviewRecord> {
  return requestJSON(apiBase, path(id), { method: "POST", body: JSON.stringify(input) });
}
