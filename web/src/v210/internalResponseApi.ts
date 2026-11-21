import { requestJSON } from "../http";
import type { CaptureArtifact, FormResponseWorkspace, FormWorkspaceSubmissionResult, SaveFormResponseWorkspaceInput, SubmitFormResponseWorkspaceInput } from "../captureApi";
import type { CaptureRequest } from "../types";

export type InternalWorkspacePayload = { request: CaptureRequest; workspace: FormResponseWorkspace };
const base = import.meta.env.VITE_API_BASE_URL ?? "";
const path = (requestId: string) => `/api/v1/forms/requests/${encodeURIComponent(requestId)}/workspace`;
export function loadInternalWorkspace(requestId: string) { return requestJSON<InternalWorkspacePayload>(base, path(requestId)); }
export function saveInternalWorkspace(requestId: string, input: SaveFormResponseWorkspaceInput) { return requestJSON<FormResponseWorkspace>(base, path(requestId), { method: "PATCH", body: JSON.stringify(input) }); }
export function submitInternalWorkspace(requestId: string, input: SubmitFormResponseWorkspaceInput) { return requestJSON<FormWorkspaceSubmissionResult>(base, `${path(requestId)}/submissions`, { method: "POST", body: JSON.stringify(input) }); }
export function uploadInternalDocument(requestId: string, file: File, fieldId: string) {
  const body = new FormData(); body.append("field_id", fieldId); body.append("file", file, file.name);
  return requestJSON<CaptureArtifact>(base, `${path(requestId)}/documents`, { method: "POST", body });
}
