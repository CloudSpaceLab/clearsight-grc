import { useCallback, useEffect, useRef, useState } from "react";
import {
  issueEvidenceInvitation,
  listEvidenceInvitationMetadata,
  replaceEvidenceInvitation,
  revokeEvidenceInvitation,
  revokeEvidenceSession,
  type EvidenceInvitationMetadata,
} from "../evidenceRequestAdminApi";
import { EvidenceRequestAdminPanel, type EvidenceInvitationAdminItem, type EvidenceRequestAdminLoadState } from "./EvidenceRequestAdminPanel";

export function EvidenceRequestAdminContainer({ requestID, requestTitle }: { requestID: string; requestTitle: string }) {
  const [records, setRecords] = useState<EvidenceInvitationMetadata[]>([]);
  const [state, setState] = useState<EvidenceRequestAdminLoadState>("loading");
  const mounted = useRef(true);
  const loadGeneration = useRef(0);
  const currentRequestID = useRef(requestID);
  currentRequestID.current = requestID;

  const reload = useCallback(async () => {
    const generation = ++loadGeneration.current;
    setState("loading");
    try {
      const next = await listEvidenceInvitationMetadata(requestID);
      if (!mounted.current || generation !== loadGeneration.current) return;
      setRecords(next);
      setState("ready");
    } catch {
      if (!mounted.current || generation !== loadGeneration.current) return;
      setState("unavailable");
    }
  }, [requestID]);

  useEffect(() => {
    mounted.current = true;
    setRecords([]);
    setState("loading");
    void reload();
    return () => { mounted.current = false; };
  }, [reload]);

  const invitations: EvidenceInvitationAdminItem[] = records.map((record) => ({
    id: record.id,
    audienceHint: record.audience_hint,
    purpose: record.purpose,
    expiresAt: record.expires_at,
    maxRedemptions: record.max_redemptions,
    redemptions: record.redemptions,
    revokedAt: record.revoked_at,
    issuedAt: record.created_at,
  }));

  return <EvidenceRequestAdminPanel
    key={requestID}
    requestTitle={requestTitle}
    recipients={[]}
    invitations={invitations}
    canManage
    loadState={state}
    issueInvitation={async (input) => {
      const issued = await issueEvidenceInvitation(requestID, input);
      if (currentRequestID.current !== requestID) return issued;
      setRecords((current) => [{ id: issued.invitation_id, request_id: requestID, audience_hint: issued.audience_hint, purpose: input.purpose, expires_at: issued.expires_at, max_redemptions: 1, redemptions: 0, created_at: "" }, ...current]);
      void reload();
      return issued;
    }}
    replaceInvitation={async (invitationID, input) => {
      const issued = await replaceEvidenceInvitation(requestID, invitationID, input);
      if (currentRequestID.current !== requestID) return issued;
      const revokedAt = new Date().toISOString();
      setRecords((current) => [{ id: issued.invitation_id, request_id: requestID, audience_hint: issued.audience_hint, purpose: input.purpose, expires_at: issued.expires_at, max_redemptions: 1, redemptions: 0, created_at: "" }, ...current.map((record) => record.id === invitationID ? { ...record, revoked_at: revokedAt } : record)]);
      void reload();
      return issued;
    }}
    revokeInvitation={async (invitationID) => {
      await revokeEvidenceInvitation(requestID, invitationID);
      if (currentRequestID.current !== requestID) return;
      const revokedAt = new Date().toISOString();
      setRecords((current) => current.map((record) => record.id === invitationID ? { ...record, revoked_at: revokedAt } : record));
      void reload();
    }}
    revokeSession={(sessionID) => revokeEvidenceSession(requestID, sessionID)}
  />;
}
