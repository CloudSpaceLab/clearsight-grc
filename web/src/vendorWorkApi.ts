import { requestJSON } from "./http";
import type {
  PrepareVendorWorkInput, RequestVendorWorkChangesInput, SendVendorWorkInput, VendorWorkPage,
  VendorWorkQuery, VendorWorkRequest, VendorWorkResponseView, VendorWorkSendOutcome,
} from "./vendorWorkTypes";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export function loadVendorWork(query: VendorWorkQuery): Promise<VendorWorkPage> {
  const params = new URLSearchParams();
  if (query.relationship_id) params.set("relationship_id", query.relationship_id);
  if (query.target_type) params.set("target_type", query.target_type);
  if (query.target_id) params.set("target_id", query.target_id);
  if (query.cursor) params.set("cursor", query.cursor);
  params.set("limit", String(query.limit ?? 20));
  return requestJSON<VendorWorkPage>(apiBase, `/api/v1/vendor-work?${params.toString()}`);
}

export function prepareVendorWork(relationshipID: string, input: PrepareVendorWorkInput): Promise<VendorWorkRequest> {
  return requestJSON<VendorWorkRequest>(apiBase, `/api/v1/vendors/${encodeURIComponent(relationshipID)}/work/prepare`, { method: "POST", body: JSON.stringify(input) });
}

export function loadVendorWorkResponse(relationshipID: string, workID: string): Promise<VendorWorkResponseView> {
  return requestJSON<VendorWorkResponseView>(apiBase, `/api/v1/vendors/${encodeURIComponent(relationshipID)}/work/${encodeURIComponent(workID)}/response`);
}

export function vendorWorkDocumentURL(relationshipID: string, workID: string, requestID: string, artifactID: string): string {
  return `${apiBase}/api/v1/vendors/${encodeURIComponent(relationshipID)}/work/${encodeURIComponent(workID)}/requests/${encodeURIComponent(requestID)}/documents/${encodeURIComponent(artifactID)}/open`;
}

export function sendVendorWork(relationshipID: string, workID: string, input: SendVendorWorkInput): Promise<VendorWorkSendOutcome> {
  return command<VendorWorkSendOutcome>(relationshipID, workID, "send", input);
}

export function startVendorWorkReview(relationshipID: string, workID: string, input: { expected_version: number }): Promise<VendorWorkRequest> {
  return command<VendorWorkRequest>(relationshipID, workID, "review/start", input);
}

export function requestVendorWorkChanges(relationshipID: string, workID: string, input: RequestVendorWorkChangesInput): Promise<VendorWorkSendOutcome> {
  return command<VendorWorkSendOutcome>(relationshipID, workID, "changes", input);
}

export function acceptVendorWork(relationshipID: string, workID: string, input: { expected_version: number; rationale: string }): Promise<VendorWorkRequest> {
  return command<VendorWorkRequest>(relationshipID, workID, "accept", input);
}

export function cancelVendorWork(relationshipID: string, workID: string, input: { expected_version: number; reason: string }): Promise<VendorWorkRequest> {
  return command<VendorWorkRequest>(relationshipID, workID, "cancel", input);
}

export function retryVendorWorkDelivery(relationshipID: string, workID: string, input: SendVendorWorkInput): Promise<VendorWorkSendOutcome> {
  return command<VendorWorkSendOutcome>(relationshipID, workID, "retry", input);
}

function command<T>(relationshipID: string, workID: string, suffix: string, input: unknown): Promise<T> {
  return requestJSON<T>(apiBase, `/api/v1/vendors/${encodeURIComponent(relationshipID)}/work/${encodeURIComponent(workID)}/${suffix}`, { method: "POST", body: JSON.stringify(input) });
}
