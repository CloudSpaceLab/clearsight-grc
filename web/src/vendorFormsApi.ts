import type { CreateDistributionInput, CreateDistributionRecipient, ResponseScore } from "./formsDistributionApi";
import { requestJSON } from "./http";
export type VendorFormsFilter = "AWAITING_VENDOR" | "AWAITING_REVIEW" | "WITH_RISKS" | "HIGH_RISK" | "OVERDUE" | "NOT_ASSESSED";
export type VendorFormsQuery = { filter?: VendorFormsFilter; form_template_id?: string; cursor?: string; limit?: number };
export type VendorFormRow = {
  request_id: string; relationship_id: string; distribution_id?: string; response_id?: string; form_template_id: string; form_template_version: number;
  title: string; purpose?: string; origin_type?: string; origin_id?: string; response_state: string; recipient_hint?: string; delivery_state?: string;
  deadline: string; updated_at: string; submitted_at?: string; required_count: number | null; answered_required: number | null;
  held_required?: number | null;
  missing_fields: Array<{ id: string; label: string }>; score?: ResponseScore; assessed_score?: ResponseScore; assessment_state?: string; required_reviews: number; completed_reviews: number; current: boolean; response_currency?: "CURRENT" | "PARTIALLY_REPLACED" | "HISTORICAL";
};
export type VendorFormsPage = { items: VendorFormRow[]; next_cursor?: string; observed_at: string };
export type VendorFormSummary = { relationship_id: string; outstanding_forms: number; overdue_forms: number; submitted_forms: number; awaiting_review: number; unassessed_forms: number; assessed_forms: number; highest_concern?: string; observed_at: string; partially_replaced_forms?: number };
export type VendorRequestSettings = Omit<CreateDistributionInput, "subject_type" | "subject_id" | "recipients">;
export type VendorRequestInput = VendorRequestSettings & { batch_id: string; targets: Array<{ relationship_id: string; recipient: CreateDistributionRecipient }> };
export type VendorRequestReceipt = { batch_id: string; items: Array<{ relationship_id: string; status: "CREATED" | "PREPARED" | "FAILED"; distribution_id?: string; distribution_state?: string; error?: string }> };
const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";
export function loadVendorForms(relationshipID: string, query: VendorFormsQuery = {}): Promise<VendorFormsPage> {
  const params = new URLSearchParams({ limit: String(query.limit ?? 25) });
  for (const key of ["filter", "form_template_id", "cursor"] as const) if (query[key]) params.set(key, query[key]);
  return requestJSON(apiBase, `/api/v1/vendors/${encodeURIComponent(relationshipID)}/forms?${params}`);
}
export function loadVendorFormSummaries(relationshipIDs: string[], formTemplateID?: string): Promise<{ items: VendorFormSummary[] }> {
  const params = new URLSearchParams({ relationship_ids: relationshipIDs.join(","), limit: "50" });
  if (formTemplateID) params.set("form_template_id", formTemplateID);
  return requestJSON(apiBase, `/api/v1/vendors/form-summaries?${params}`);
}
export function requestVendorForms(input: VendorRequestInput): Promise<VendorRequestReceipt> {
  return requestJSON(apiBase, "/api/v1/vendors/form-requests", { method: "POST", body: JSON.stringify(input) });
}
