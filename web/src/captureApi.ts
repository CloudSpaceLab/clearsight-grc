import { loadContext } from "./api";
import { ApiError, parseJSON, requestJSON } from "./http";
import type {
  CaptureAnswerInputs,
  CaptureAnswerValue,
  CaptureAnswers,
  CapturePresentationMode,
  CaptureRequest,
  EvidenceRecipient,
} from "./types";

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
  presentation_mode: CapturePresentationMode;
  version: number;
  updated_at?: string;
};

export type SaveCaptureDraftInput = {
  answers: CaptureAnswerInputs;
  presentation_mode: CaptureDraft["presentation_mode"];
  expected_version: number;
};

export type FormAccessPolicy = "DIRECT_MAGIC_LINK" | "SHARED_LINK_EMAIL_OTP" | "DIRECT_LINK_EMAIL_OTP";
export type FormAccessAssurance = "LINK_POSSESSION" | "EMAIL_VERIFIED";

export type MaskedFormRecipient = {
  selector_id: string;
  hint: string;
  contact_label?: string;
};

export type FormAccessStart = {
  policy: FormAccessPolicy;
  recipients?: MaskedFormRecipient[];
  expires_at: string;
};

export type FormAccessOTPReceipt = {
  challenge_id: string;
  hint: string;
  expires_at: string;
};

export type RedeemedFormAccessSession = {
  session_id: string;
  session_token: string;
  distribution_id: string;
  request_id: string;
  audience_hint: string;
  assurance: FormAccessAssurance;
  expires_at: string;
};

export type FormAccessSession = {
  id: string;
  distribution_id: string;
  request_id: string;
  audience_hint: string;
  assurance: FormAccessAssurance;
  expires_at: string;
  revoked_at?: string;
  created_at: string;
};

export type FormResponseWorkspaceState = {
  id: string;
  distribution_id: string;
  status: "OPEN" | "LOCKED" | "COMPLETED" | "REVOKED";
  version: number;
  created_at: string;
  updated_at: string;
};

export type FormResponseWorkspace = {
  workspace: FormResponseWorkspaceState;
  answers: CaptureAnswers;
  presentation_mode: CapturePresentationMode;
  field_sequences: Record<string, number>;
  current_revision?: Record<string, unknown>;
};

export type FormResponseRecoveryContext = {
  legal_entity_id: string;
  distribution_id: string;
  schema_version: number;
  route_expires_at: string;
};

export type FormResponseWorkspacePayload = {
  session: FormAccessSession;
  request: CaptureRequest;
  workspace: FormResponseWorkspace;
  recovery_context?: FormResponseRecoveryContext;
};

export type FormWorkspaceEditInput = {
  field_id: string;
  value: CaptureAnswerValue;
  base_sequence: number;
};

export type SaveFormResponseWorkspaceInput = {
  expected_version: number;
  presentation_mode: CapturePresentationMode;
  edits: FormWorkspaceEditInput[];
};

export type SubmitFormResponseWorkspaceInput = {
  expected_version: number;
  attestation_field_ids?: string[];
};

export type FormWorkspaceSubmissionResult = {
  workspace: FormResponseWorkspaceState;
  revision: Record<string, unknown>;
  submission: CaptureReceipt;
};

export type FormWorkspaceFieldConflict = {
  field_id: string;
  server_value: CaptureAnswerValue;
  sequence: number;
};

export type FormWorkspaceConflict = {
  current_version: number;
  changed_fields: FormWorkspaceFieldConflict[];
};

export class FormWorkspaceConflictError extends ApiError {
  readonly conflict: FormWorkspaceConflict;

  constructor(conflict: FormWorkspaceConflict) {
    super(409, "The shared response changed while you were working.", "workspace_conflict");
    this.name = "FormWorkspaceConflictError";
    this.conflict = conflict;
  }
}

export function normalizeCaptureAnswer(value: CaptureAnswerValue | string): CaptureAnswerValue {
  return typeof value === "string" ? { text: value } : value;
}

export function normalizeCaptureAnswers(answers: CaptureAnswerInputs): CaptureAnswers {
  return Object.fromEntries(Object.entries(answers).map(([fieldID, value]) => [
    fieldID,
    normalizeCaptureAnswer(value),
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

/** @deprecated Distribution-backed external forms use the policy-driven access ceremony below. */
export function redeemCaptureInvitation(token: string, audience: string): Promise<RedeemedCaptureSession> {
  return request<RedeemedCaptureSession>("/api/v1/evidence/invitations/redeem", {
    method: "POST",
    body: JSON.stringify({ token, audience }),
  });
}

/** @deprecated Distribution-backed external forms use loadFormResponseWorkspace. */
export function loadCaptureSession(sessionToken: string): Promise<CaptureSessionPayload> {
  return request<CaptureSessionPayload>("/api/v1/evidence/session", {
    headers: { Authorization: `Bearer ${sessionToken}` },
  });
}

/** @deprecated Distribution-backed external forms use the shared response workspace. */
export function loadCaptureDraft(sessionToken: string): Promise<CaptureDraft> {
  return request<CaptureDraft>("/api/v1/evidence/session/draft", {
    headers: { Authorization: `Bearer ${sessionToken}` },
  });
}

/** @deprecated Distribution-backed external forms use saveFormResponseWorkspace. */
export function saveCaptureDraft(sessionToken: string, input: SaveCaptureDraftInput): Promise<CaptureDraft> {
  return request<CaptureDraft>("/api/v1/evidence/session/draft", {
    method: "PUT",
    headers: { Authorization: `Bearer ${sessionToken}` },
    body: JSON.stringify({ ...input, answers: normalizeCaptureAnswers(input.answers) }),
  });
}

export function startFormAccess(routeSelector: string): Promise<FormAccessStart> {
  return request<FormAccessStart>("/api/v1/evidence/access/start", {
    method: "POST",
    body: JSON.stringify({ route_selector: routeSelector }),
  });
}

export function sendFormAccessOTP(routeSelector: string, recipientSelector: string): Promise<FormAccessOTPReceipt> {
  return request<FormAccessOTPReceipt>("/api/v1/evidence/access/otp/send", {
    method: "POST",
    body: JSON.stringify({ route_selector: routeSelector, recipient_selector: recipientSelector }),
  });
}

export function verifyFormAccessOTP(routeSelector: string, challengeID: string, code: string): Promise<RedeemedFormAccessSession> {
  return request<RedeemedFormAccessSession>("/api/v1/evidence/access/otp/verify", {
    method: "POST",
    body: JSON.stringify({ route_selector: routeSelector, challenge_id: challengeID, code }),
  });
}

export function redeemFormAccess(routeSelector: string): Promise<RedeemedFormAccessSession> {
  return request<RedeemedFormAccessSession>("/api/v1/evidence/access/redeem", {
    method: "POST",
    body: JSON.stringify({ route_selector: routeSelector }),
  });
}

export function loadFormResponseWorkspace(sessionToken: string): Promise<FormResponseWorkspacePayload> {
  return request<FormResponseWorkspacePayload>("/api/v1/evidence/session/workspace", {
    headers: { Authorization: `Bearer ${sessionToken}` },
  });
}

export async function saveFormResponseWorkspace(sessionToken: string, input: SaveFormResponseWorkspaceInput): Promise<FormResponseWorkspace> {
  const response = await fetch(`${apiBase}/api/v1/evidence/session/workspace`, {
    method: "PATCH",
    credentials: "include",
    headers: {
      Authorization: `Bearer ${sessionToken}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(input),
  });
  if (response.status === 409) {
    const body = await response.json().catch(() => null) as unknown;
    if (isWorkspaceConflict(body)) throw new FormWorkspaceConflictError(body);
    throw new ApiError(409, "The shared response changed while you were working.", "workspace_conflict");
  }
  return parseJSON<FormResponseWorkspace>(response);
}

export function submitFormResponseWorkspace(sessionToken: string, input: SubmitFormResponseWorkspaceInput): Promise<FormWorkspaceSubmissionResult> {
  return request<FormWorkspaceSubmissionResult>("/api/v1/evidence/session/workspace/submissions", {
    method: "POST",
    headers: { Authorization: `Bearer ${sessionToken}` },
    body: JSON.stringify(input),
  });
}

export async function submitInternalCaptureRequest(requestID: string, version: number, answers: CaptureAnswerInputs): Promise<CaptureReceipt> {
  const context = await loadContext();
  return request<CaptureReceipt>(`/api/v1/evidence/requests/${encodeURIComponent(requestID)}/submissions?tenant_id=${encodeURIComponent(context.tenant.id)}`, {
    method: "POST",
    body: JSON.stringify({ tenant_id: context.tenant.id, expected_version: version, answers: normalizeCaptureAnswers(answers) }),
  });
}

/** @deprecated Distribution-backed external forms submit through submitFormResponseWorkspace. */
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

function isWorkspaceConflict(value: unknown): value is FormWorkspaceConflict {
  if (!value || typeof value !== "object") return false;
  const conflict = value as Partial<FormWorkspaceConflict>;
  return Number.isInteger(conflict.current_version)
    && Array.isArray(conflict.changed_fields)
    && conflict.changed_fields.every((field) => Boolean(field)
      && typeof field === "object"
      && typeof field.field_id === "string"
      && Number.isInteger(field.sequence)
      && Boolean(field.server_value)
      && typeof field.server_value === "object");
}
