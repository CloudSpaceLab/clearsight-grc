import type { EvidenceRequest } from "./types";

export function isEvidenceRequestAssignedToActor(request: EvidenceRequest, actorPrincipalID?: string) {
  return Boolean(actorPrincipalID?.trim()) &&
    request.audience_type === "INTERNAL" &&
    request.recipient?.state === "ASSIGNED" &&
    request.recipient.type === "INTERNAL_PRINCIPAL" &&
    request.recipient.principal_id === actorPrincipalID;
}

export function canRespondToEvidenceRequest(request: EvidenceRequest, actorPrincipalID?: string) {
  return ["READY", "IN_PROGRESS"].includes(request.status) &&
    isEvidenceRequestAssignedToActor(request, actorPrincipalID);
}
