import { loadContext } from "./api";
import { requestJSON } from "./http";
import type { CaptureAnswerInputs, CaptureAnswers, CaptureRequest, EvidenceRecipient } from "./types";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type CaptureArtifact = {
  id: string;
  request_id: string;
  file_name: string;
  media_type: string;
  size_bytes: number;
  sha256: string;
  status: string;
};

export type CaptureSession = {
  id: string;
  request_id: string;
  audience_hint: string;
  expires_at: string;
};

export type RedeemedCaptureSession = {
  session_id: string;
  session_token: string;
  request_id: string;
  audience_hint: string;
  expires_at: string;
};

export type CaptureSessionPayload = {
  session: CaptureSession;
  request: CaptureRequest;
};

export type CaptureReceipt = {
  request_id: string;
  submission_id?: string;
  status: string;
  submitted_at: string;
  version?: number;
};

export type CaptureDraft = {
  answers: CaptureAnswers;
  presentation_mode: "CLASSIC" | "WIZARD" | "AUTOMATIC";
  version: number;
  updated_at?: string;
};

export type SaveCaptureDraftInput = {
  answers: CaptureAnswerInputs;
  presentation_mode: CaptureDraft["presentation_mode"];
  expected_version: number;
};

export function normalizeCaptureAnswers(answers: CaptureAnswerInputs): CaptureAnswers {
  return Object.fromEntries(Object.entries(answers).map(([fieldID, value]) => [
    fieldID,
    typeof value === "string" ? { text: value } : value,
  ]));
}

function request<T>(path: string, init?: RequestInit): Promise<T> {
  return requestJSON<T>(apiBase, path, init);
}

export async function uploadInternalCaptureArtifact(requestID: string, file: File, fieldID?: string): Promise<CaptureArtifact> {
  const context = await loadContext();
  const body = new FormData();
  body.append("tenant_id", context.tenant.id);
  body.append("request_id", requestID);
	if (fieldID) body.append("field_id", fieldID);
  body.append("file", file, file.name);
  return request<CaptureArtifact>("/api/v1/evidence/artifacts", { method: "POST", body });
}

export async function declareWrongCaptureRecipient(requestID: string, expectedVersion: number, reason: string): Promise<CaptureRequest> {
  const context = await loadContext();
  return request<CaptureRequest>(`/api/v1/evidence/requests/${encodeURIComponent(requestID)}/wrong-recipient?tenant_id=${encodeURIComponent(context.tenant.id)}`, {
    method: "POST",
    body: JSON.stringify({ expected_version: expectedVersion, reason }),
  });
}

export async function reassignCaptureRecipient(
  requestID: string,
  expectedVersion: number,
  recipient: Pick<EvidenceRecipient, "type" | "principal_id"> & { audience?: string },
  reason: string,
): Promise<CaptureRequest> {
  const context = await loadContext();
  return request<CaptureRequest>(`/api/v1/evidence/requests/${encodeURIComponent(requestID)}/recipient?tenant_id=${encodeURIComponent(context.tenant.id)}`, {
    method: "PUT",
    body: JSON.stringify({ expected_version: expectedVersion, recipient, reason }),
  });
}

export function redeemCaptureInvitation(token: string, audience: string): Promise<RedeemedCaptureSession> {
  return request<RedeemedCaptureSession>("/api/v1/evidence/invitations/redeem", {
    method: "POST",
    body: JSON.stringify({ token, audience }),
  });
}

export function loadCaptureSession(sessionToken: string): Promise<CaptureSessionPayload> {
  return request<CaptureSessionPayload>("/api/v1/evidence/session", {
    headers: { Authorization: `Bearer ${sessionToken}` },
  });
}

export function loadCaptureDraft(sessionToken: string): Promise<CaptureDraft> {
  return request<CaptureDraft>("/api/v1/evidence/session/draft", {
    headers: { Authorization: `Bearer ${sessionToken}` },
  });
}

export function saveCaptureDraft(sessionToken: string, input: SaveCaptureDraftInput): Promise<CaptureDraft> {
  return request<CaptureDraft>("/api/v1/evidence/session/draft", {
    method: "PUT",
    headers: { Authorization: `Bearer ${sessionToken}` },
    body: JSON.stringify({ ...input, answers: normalizeCaptureAnswers(input.answers) }),
  });
}

export async function submitInternalCaptureRequest(requestID: string, version: number, answers: CaptureAnswerInputs): Promise<CaptureReceipt> {
  const context = await loadContext();
  return request<CaptureReceipt>(`/api/v1/evidence/requests/${encodeURIComponent(requestID)}/submissions?tenant_id=${encodeURIComponent(context.tenant.id)}`, {
    method: "POST",
    body: JSON.stringify({ tenant_id: context.tenant.id, expected_version: version, answers: normalizeCaptureAnswers(answers) }),
  });
}

export function submitCaptureSession(sessionToken: string, version: number, answers: CaptureAnswerInputs): Promise<CaptureReceipt> {
  return request<CaptureReceipt>("/api/v1/evidence/session/submissions", {
    method: "POST",
    headers: { Authorization: `Bearer ${sessionToken}` },
    body: JSON.stringify({ expected_version: version, answers: normalizeCaptureAnswers(answers) }),
  });
}

export async function uploadCaptureSessionArtifact(sessionToken: string, file: File, fieldID?: string): Promise<CaptureArtifact> {
  const body = new FormData();
	if (fieldID) body.append("field_id", fieldID);
  body.append("file", file, file.name);
  return request<CaptureArtifact>("/api/v1/evidence/artifacts", {
    method: "POST",
    headers: { Authorization: `Bearer ${sessionToken}` },
    body,
  });
}
