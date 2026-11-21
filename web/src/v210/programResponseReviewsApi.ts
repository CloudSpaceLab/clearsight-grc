import { requestJSON } from "../http";
import type { CompletedResponseSummary } from "../formsDistributionApi";

export type ProgramResponseReviewPage = {
  scope: { tenant_id: string; legal_entity_id: string; principal_id: string };
  coverage: string;
  items: CompletedResponseSummary[];
  next_cursor?: string;
};

export function loadProgramResponseReviews(cursor?: string, signal?: AbortSignal): Promise<ProgramResponseReviewPage> {
  const params = new URLSearchParams({ limit: "20" });
  if (cursor) params.set("cursor", cursor);
  return requestJSON(import.meta.env.VITE_API_BASE_URL ?? "", `/api/v1/forms/program-reviews?${params}`, signal ? { signal } : undefined);
}
