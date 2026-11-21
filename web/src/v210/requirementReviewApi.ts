import { continuityCommand } from '../continuityCommands';
import { requestJSON } from '../http';
import type { Program } from '../types';

const apiBase = import.meta.env.VITE_API_BASE_URL ?? '';
export type RequirementReview = { id: string; tenant_id: string; program_id: string; requirement_id: string; proposed_by: string; proposal_rationale: string; status: string; version: number; reviewed_by?: string; review_rationale?: string };
export type RequirementReviewPage = { program_id: string; program_version: number; items: RequirementReview[]; next_after_id: string };
export type RequirementReviewReceipt = { review: RequirementReview; program: Program };
export function loadRequirementReviews(tenant: string, program: string): Promise<RequirementReviewPage> {
  const query = new URLSearchParams({ tenant_id: tenant });
  return requestJSON(apiBase, `/api/v1/programs/${encodeURIComponent(program)}/requirement-reviews?${query}`);
}
export function proposeRequirementReview(program: string, requirement: string, version: number, rationale: string): Promise<RequirementReviewReceipt> {
  return continuityCommand(`/api/v1/programs/${encodeURIComponent(program)}/requirements/${encodeURIComponent(requirement)}/review`, { expected_version: version, rationale });
}
export function decideRequirementReview(program: string, review: string, version: number, action: 'approve' | 'reject', rationale: string): Promise<RequirementReviewReceipt> {
  return continuityCommand(`/api/v1/programs/${encodeURIComponent(program)}/requirement-reviews/${encodeURIComponent(review)}/${action}`, { expected_version: version, rationale });
}
