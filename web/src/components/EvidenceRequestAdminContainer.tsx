import { useCallback, useEffect, useRef, useState } from "react";
import {
  issueEvidenceInvitation,
  listEvidenceActiveSessions,
  listEvidenceInvitationMetadata,
  replaceEvidenceInvitation,
  revokeEvidenceInvitation,
  revokeEvidenceSession,
  type EvidenceInvitationMetadata,
  type EvidenceActiveSessionMetadata,
} from "../evidenceRequestAdminApi";
import { EvidenceRequestAdminPanel, type EvidenceInvitationAdminItem, type EvidenceRequestAdminLoadState } from "./EvidenceRequestAdminPanel";

export function EvidenceRequestAdminContainer({ requestID, requestTitle }: { requestID: string; requestTitle: string }) {
  const [records, setRecords] = useState<EvidenceInvitationMetadata[]>([]);
  const [sessionRecords, setSessionRecords] = useState<EvidenceActiveSessionMetadata[]>([]);
  const [sessionsHaveMore, setSessionsHaveMore] = useState(false);
  const [state, setState] = useState<EvidenceRequestAdminLoadState>("loading");
  const [sessionState, setSessionState] = useState<EvidenceRequestAdminLoadState>("loading");
  const mounted = useRef(true);
  const invitationLoadGeneration = useRef(0);
  const sessionLoadGeneration = useRef(0);
  const currentRequestID = useRef(requestID);
  currentRequestID.current = requestID;

  const reload = useCallback(async () => {
    const generation = ++invitationLoadGeneration.current;
    setState("loading");
    try {
      const next = await listEvidenceInvitationMetadata(requestID);
      if (!mounted.current || generation !== invitationLoadGeneration.current) return;
      setRecords(next);
      setState("ready");
    } catch {
      if (!mounted.current || generation !== invitationLoadGeneration.current) return;
      setState("unavailable");
    }
  }, [requestID]);

  const reloadSessions = useCallback(async () => {
    const generation = ++sessionLoadGeneration.current;
    setSessionState("loading");
    try {
      const next = await listEvidenceActiveSessions(requestID);
      if (!mounted.current || generation !== sessionLoadGeneration.current) return;
      setSessionRecords(next.items);
      setSessionsHaveMore(next.has_more);
      setSessionState("ready");
    } catch {
      if (!mounted.current || generation !== sessionLoadGeneration.current) return;
      setSessionState("unavailable");
    }
  }, [requestID]);

  useEffect(() => {
    mounted.current = true;
    setRecords([]);
    setSessionRecords([]);
    setSessionsHaveMore(false);
    setState("loading");
    setSessionState("loading");
    void reload();
    void reloadSessions();
    return () => { mounted.current = false; };
  }, [reload, reloadSessions]);

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
  const activeSessions = sessionRecords.map((record) => ({
    id: record.id,
    audienceHint: record.audience_hint,
    expiresAt: record.expires_at,
    startedAt: record.created_at,
  }));

  return <EvidenceRequestAdminPanel
    key={requestID}
    requestTitle={requestTitle}
    recipients={[]}
    invitations={invitations}
    activeSessions={activeSessions}
    activeSessionsHasMore={sessionsHaveMore}
    canManage
    loadState={state}
    sessionLoadState={sessionState}
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
      setSessionRecords([]);
      void reload();
      void reloadSessions();
      return issued;
    }}
    revokeInvitation={async (invitationID) => {
      await revokeEvidenceInvitation(requestID, invitationID);
      if (currentRequestID.current !== requestID) return;
      const revokedAt = new Date().toISOString();
      setRecords((current) => current.map((record) => record.id === invitationID ? { ...record, revoked_at: revokedAt } : record));
      setSessionRecords([]);
      void reload();
      void reloadSessions();
    }}
    revokeSession={async (sessionID) => {
      await revokeEvidenceSession(requestID, sessionID);
      if (currentRequestID.current !== requestID) return;
      setSessionRecords((current) => current.filter((record) => record.id !== sessionID));
      void reloadSessions();
    }}
  />;
}
