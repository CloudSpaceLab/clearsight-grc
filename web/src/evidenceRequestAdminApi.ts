import { requestJSON, requestVoid } from "./http";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type EvidenceInvitationMetadata = {
  id: string;
  request_id: string;
  audience_hint: string;
  purpose: string;
  expires_at: string;
  max_redemptions: number;
  redemptions: number;
  revoked_at?: string;
  created_at: string;
};

export type EvidenceInvitationCommand = {
  audience: string;
  purpose: string;
  ttlMinutes: number;
};

export type IssuedEvidenceInvitation = {
  invitation_id: string;
  token: string;
  audience_hint: string;
  expires_at: string;
};

export type EvidenceRecipientCandidate = { principal_id: string; display_name: string; context_label?: string };
export type EvidenceRecipientCandidatePage = { items: EvidenceRecipientCandidate[]; has_more: boolean };

export type EvidenceActiveSessionMetadata = {
  id: string;
  audience_hint: string;
  expires_at: string;
  created_at: string;
};

export type EvidenceActiveSessionPage = { items: EvidenceActiveSessionMetadata[]; has_more: boolean };

type InvitationMetadataResponse = { items?: EvidenceInvitationMetadata[] | null };

export async function listEvidenceInvitationMetadata(requestID: string): Promise<EvidenceInvitationMetadata[]> {
  const response = await requestJSON<InvitationMetadataResponse>(apiBase, invitationCollectionPath(requestID), { method: "GET" });
  return Array.isArray(response.items) ? response.items : [];
}

export async function listEvidenceRecipientCandidates(requestID: string, query = ""): Promise<EvidenceRecipientCandidatePage> {
  const params = new URLSearchParams({ limit: "50" });
  if (query.trim()) params.set("q", query.trim());
  const response = await requestJSON<{ items?: EvidenceRecipientCandidate[] | null; has_more?: boolean }>(apiBase, `/api/v1/evidence/requests/${encodeURIComponent(requestID)}/recipient-candidates?${params.toString()}`, { method: "GET" });
  return { items: Array.isArray(response.items) ? response.items : [], has_more: response.has_more === true };
}

export async function listEvidenceActiveSessions(requestID: string): Promise<EvidenceActiveSessionPage> {
  const response = await requestJSON<{ items?: EvidenceActiveSessionMetadata[] | null; has_more?: boolean }>(
    apiBase,
    `/api/v1/evidence/requests/${encodeURIComponent(requestID)}/sessions?limit=50`,
    { method: "GET" },
  );
  return { items: Array.isArray(response.items) ? response.items : [], has_more: response.has_more === true };
}

export function issueEvidenceInvitation(requestID: string, input: EvidenceInvitationCommand): Promise<IssuedEvidenceInvitation> {
  return requestJSON<IssuedEvidenceInvitation>(apiBase, invitationCollectionPath(requestID), commandRequest(input));
}

export function replaceEvidenceInvitation(requestID: string, invitationID: string, input: EvidenceInvitationCommand): Promise<IssuedEvidenceInvitation> {
  return requestJSON<IssuedEvidenceInvitation>(
    apiBase,
    `${invitationCollectionPath(requestID)}/${encodeURIComponent(invitationID)}/replace`,
    commandRequest(input),
  );
}

export function revokeEvidenceInvitation(requestID: string, invitationID: string): Promise<void> {
  return requestVoid(apiBase, `${invitationCollectionPath(requestID)}/${encodeURIComponent(invitationID)}/revoke`, { method: "POST" });
}

export function revokeEvidenceSession(requestID: string, sessionID: string): Promise<void> {
  return requestVoid(
    apiBase,
    `/api/v1/evidence/requests/${encodeURIComponent(requestID)}/sessions/${encodeURIComponent(sessionID)}/revoke`,
    { method: "POST" },
  );
}

function invitationCollectionPath(requestID: string) {
  return `/api/v1/evidence/requests/${encodeURIComponent(requestID)}/invitations`;
}

function commandRequest(input: EvidenceInvitationCommand): RequestInit {
  return {
    method: "POST",
    body: JSON.stringify({ audience: input.audience, purpose: input.purpose, ttl_minutes: input.ttlMinutes }),
  };
}
