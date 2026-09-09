import { requestJSON } from "./http";
import type { DocumentOccurrence } from "./submittedDocumentApi";
import type { VendorAssessment, VendorAssessmentRequestSummary } from "./vendorAssessmentTypes";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type VendorCollectionResolution = {
  id: string; version: number; source: DocumentOccurrence;
  reconciled_by: string; reconciled_at: string; rationale: string;
};
export type VendorCollectionField = {
  field_id: string; label: string; type: string; required: boolean;
  collection_state: "MISSING" | "RECEIVED" | "REUSED" | "NOT_REQUIRED" | "CONDITION_UNKNOWN";
  vendor_action_required: boolean;
  bank_review_state: "PENDING" | "VALIDATED" | "REJECTED" | "NOT_REQUIRED";
  resolution?: VendorCollectionResolution;
};
export type VendorCollection = {
  assessment_id: string; assessment_version: number; request_id?: string; request_version?: number;
  prepared: boolean; can_reconcile: boolean; observed_at: string; fields: VendorCollectionField[];
  vendor_pending_count: number; bank_pending_count: number; accepted_document_count?: number;
  deadline?: string; audience_hint?: string; can_start_review?: boolean;
};
export type ReconcileVendorEvidenceInput = {
  expected_version: number; request_id: string; expected_request_version: number;
  source_submission_id: string; source_field_id: string; source_artifact_id: string;
  source_response_revision_id?: string; rationale: string;
};
export type VendorReconciliationResult = { collection: VendorCollection; receipt: VendorCollectionResolution };
export type PrepareVendorCollectionInput = { expected_version: number; audience: string; deadline: string };
export type PrepareVendorCollectionResult = { assessment: VendorAssessment; request: VendorAssessmentRequestSummary; state: "PREPARED" };

export function loadVendorCollection(assessmentID: string, signal?: AbortSignal): Promise<VendorCollection> {
  return requestJSON(apiBase, `/api/v1/vendor-assessments/${encodeURIComponent(assessmentID)}/collection`, { signal });
}
export function reconcileVendorEvidence(assessmentID: string, fieldID: string, input: ReconcileVendorEvidenceInput): Promise<VendorReconciliationResult> {
  return requestJSON(apiBase, `/api/v1/vendor-assessments/${encodeURIComponent(assessmentID)}/collection/${encodeURIComponent(fieldID)}/reconcile`, { method: "POST", body: JSON.stringify(input) });
}
export function prepareVendorCollection(assessmentID: string, input: PrepareVendorCollectionInput): Promise<PrepareVendorCollectionResult> {
  return requestJSON(apiBase, `/api/v1/vendor-assessments/${encodeURIComponent(assessmentID)}/prepare-request`, { method: "POST", body: JSON.stringify(input) });
}
