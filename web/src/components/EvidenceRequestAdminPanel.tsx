import { useEffect, useState, type FormEvent } from "react";
import type { EvidenceInvitationCommand, IssuedEvidenceInvitation } from "../evidenceRequestAdminApi";

export type EvidenceRequestAdminLoadState = "loading" | "ready" | "degraded" | "unavailable";

export type EvidenceAdminRecipient = {
  label: string;
  audience: string;
};

export type EvidenceInvitationAdminItem = {
  id: string;
  audienceHint: string;
  purpose: string;
  expiresAt: string;
  maxRedemptions: number;
  redemptions: number;
  revokedAt?: string;
  issuedAt?: string;
};

export type EvidenceExternalSessionAdminItem = {
  id: string;
  audienceHint: string;
  expiresAt: string;
  startedAt: string;
};

export type EvidenceRequestAdminPanelProps = {
  requestTitle: string;
  recipients: EvidenceAdminRecipient[];
  invitations: EvidenceInvitationAdminItem[];
  activeSessions?: EvidenceExternalSessionAdminItem[];
  activeSessionsHasMore?: boolean;
  canManage: boolean;
  loadState: EvidenceRequestAdminLoadState;
  degradedReason?: string;
  sessionLoadState?: EvidenceRequestAdminLoadState;
  sessionDegradedReason?: string;
  issueInvitation: (input: EvidenceInvitationCommand) => Promise<IssuedEvidenceInvitation>;
  replaceInvitation: (invitationID: string, input: EvidenceInvitationCommand) => Promise<IssuedEvidenceInvitation>;
  revokeInvitation: (invitationID: string) => Promise<void>;
  revokeSession: (sessionID: string) => Promise<void>;
};

export function EvidenceRequestAdminPanel({
  requestTitle,
  recipients,
  invitations,
  activeSessions,
  activeSessionsHasMore = false,
  canManage,
  loadState,
  degradedReason,
  sessionLoadState = loadState,
  sessionDegradedReason,
  issueInvitation,
  replaceInvitation,
  revokeInvitation,
  revokeSession,
}: EvidenceRequestAdminPanelProps) {
  const activeInvitation = invitations.find((item) => invitationState(item) === "Active");
  const [audience, setAudience] = useState(recipients[0]?.audience ?? "");
  const [purpose, setPurpose] = useState(activeInvitation?.purpose ?? "");
  const [ttlMinutes, setTTLMinutes] = useState(1440);
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");
  const [issued, setIssued] = useState<IssuedEvidenceInvitation | null>(null);
  const [copyNotice, setCopyNotice] = useState("");
  const mutationsEnabled = canManage && loadState === "ready";
  const sessionMutationsEnabled = canManage && sessionLoadState === "ready";
  const visibleInvitations = invitations.slice(0, INVENTORY_LIMIT);
  const visibleSessions = activeSessions?.slice(0, INVENTORY_LIMIT) ?? [];

  useEffect(() => {
    if (activeInvitation?.purpose) setPurpose((current) => current || activeInvitation.purpose);
  }, [activeInvitation?.id, activeInvitation?.purpose]);

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!mutationsEnabled || !audience || !purpose.trim()) return;
    const input = { audience, purpose: purpose.trim(), ttlMinutes };
    const action = activeInvitation ? "replace" : "issue";
    setBusy(action);
    setError("");
    setCopyNotice("");
    setIssued(null);
    try {
      const result = activeInvitation
        ? await replaceInvitation(activeInvitation.id, input)
        : await issueInvitation(input);
      setIssued(result);
    } catch {
      setError(activeInvitation
        ? "The active invitation could not be replaced. Check the current recipient and expiry, then reload the evidence request before trying again."
        : "The invitation could not be created. Check the current recipient and expiry, then reload the evidence request before trying again.");
    } finally {
      setBusy("");
    }
  }

  async function revokeInvitationRecord(item: EvidenceInvitationAdminItem) {
    setBusy(`invitation:${item.id}`);
    setError("");
    try {
      await revokeInvitation(item.id);
      if (issued?.invitation_id === item.id) setIssued(null);
    } catch {
      setError("The invitation could not be revoked. Reload the evidence request to confirm its current state before trying again.");
    } finally {
      setBusy("");
    }
  }

  async function revokeExternalSession(item: EvidenceExternalSessionAdminItem) {
    setBusy(`session:${item.id}`);
    setError("");
    try {
      await revokeSession(item.id);
    } catch {
      setError("The external session could not be ended. Reload the evidence request to confirm its current state before trying again.");
    } finally {
      setBusy("");
    }
  }

  async function copyIssuedLink() {
    if (!issued) return;
    setError("");
    setCopyNotice("");
    if (!navigator.clipboard) {
      setError("Clipboard access is unavailable. Select the one-time invitation link and copy it manually before hiding it.");
      return;
    }
    try {
      await navigator.clipboard.writeText(invitationLink(issued.token));
      setCopyNotice("Invitation link copied. Deliver it through the approved channel before hiding it.");
    } catch {
      setError("The invitation link could not be copied. Select the link and copy it manually before hiding it.");
    }
  }

  function hideIssuedLink() {
    setIssued(null);
    setError("");
    setCopyNotice("");
  }

  return <section aria-labelledby="evidence-request-admin-title">
    <header>
      <span className="eyebrow">Evidence request access</span>
      <h2 id="evidence-request-admin-title">{requestTitle}</h2>
      <p>Review who can respond, then create or replace the invitation link and deliver it through the approved channel. Submitted evidence still requires the recorded review and verification steps.</p>
    </header>

    {loadState === "loading" && <p role="status">Loading invitation records. Changes remain unavailable until requester authority is confirmed.</p>}
    {loadState === "degraded" && <p role="alert">{degradedReason || "Current requester authority could not be confirmed."} The loaded invitation records remain available, but changes are disabled until authority can be confirmed.</p>}
    {loadState === "unavailable" && invitations.length > 0 && <p role="alert">{degradedReason || "The latest invitation records could not be loaded."} Previously loaded records remain available, but changes are disabled.</p>}
    {loadState === "unavailable" && invitations.length === 0 && <p role="alert">The current invitation history could not be loaded. Refresh the evidence request to try again; invitation changes are unavailable.</p>}
    {loadState === "ready" && !canManage && <p>You can review invitation records, but your current responsibility does not allow access changes for this evidence request.</p>}
    {error && <p role="alert">{error}</p>}
    {copyNotice && <p role="status">{copyNotice}</p>}

    {issued && <div role="status">
      <strong>Invitation link — shown once</strong>
      <p>Copy this link and deliver it through the approved channel now. It is not saved in this workspace and cannot be recovered after you hide it.</p>
      <label>One-time invitation link<input readOnly value={invitationLink(issued.token)}/></label>
      <button className="secondary-button" type="button" onClick={() => void copyIssuedLink()}>Copy invitation link</button>
      <button className="text-button" type="button" onClick={hideIssuedLink}>Hide invitation link</button>
    </div>}

    <section aria-labelledby="evidence-invitation-action">
      <h3 id="evidence-invitation-action">{activeInvitation ? "Replace active invitation" : "Create an invitation"}</h3>
      <p>{activeInvitation
        ? "Replacing the invitation ends the current invitation and its external sessions before a new one is issued."
        : "Create one purpose-bound invitation, then deliver its one-time link through the approved channel."}</p>
      <form onSubmit={(event) => void submit(event)}>
        <label>Recipient email or approved audience
          {recipients.length ? <select required value={audience} disabled={!mutationsEnabled || busy !== ""} onChange={(event) => setAudience(event.target.value)}>
            <option value="">Choose approved recipient</option>
            {recipients.map((recipient) => <option key={recipient.audience} value={recipient.audience}>{recipient.label} · {recipient.audience}</option>)}
          </select> : <input required type="email" value={audience} disabled={!mutationsEnabled || busy !== ""} onChange={(event) => setAudience(event.target.value)} placeholder="name@example.com"/>}
        </label>
        <label>Invitation purpose<textarea required value={purpose} disabled={!mutationsEnabled || busy !== ""} onChange={(event) => setPurpose(event.target.value)}/></label>
        <label>Invitation expiry
          <select value={ttlMinutes} disabled={!mutationsEnabled || busy !== ""} onChange={(event) => setTTLMinutes(Number(event.target.value))}>
            {EXPIRY_OPTIONS.map((option) => <option key={option.minutes} value={option.minutes}>{option.label}</option>)}
          </select>
        </label>
        <button className="primary-button" type="submit" disabled={!mutationsEnabled || busy !== "" || !audience || !purpose.trim()}>
          {busy === "issue" ? "Creating invitation…" : busy === "replace" ? "Replacing invitation…" : activeInvitation ? "Replace invitation" : "Create invitation"}
        </button>
      </form>
    </section>

    <section aria-labelledby="evidence-invitation-inventory">
      <h3 id="evidence-invitation-inventory">Invitation history</h3>
      {invitations.length > INVENTORY_LIMIT && <p>Showing the first {INVENTORY_LIMIT} invitations. More records are available.</p>}
      {loadState === "ready" && invitations.length === 0 && <p>No invitations have been issued for this evidence request.</p>}
      {visibleInvitations.map((item) => <article key={item.id}>
        <header><div><span className="eyebrow">Recipient</span><h4>{item.audienceHint}</h4></div><strong>{invitationState(item)}</strong></header>
        <dl>
          <div><dt>Purpose</dt><dd>{item.purpose}</dd></div>
          <div><dt>Issued</dt><dd>{item.issuedAt ? formatDateTime(item.issuedAt) : "Issue time awaiting invitation history refresh"}</dd></div>
          <div><dt>Expires</dt><dd>{formatDateTime(item.expiresAt)}</dd></div>
          <div><dt>Uses</dt><dd>{item.redemptions} of {item.maxRedemptions}</dd></div>
        </dl>
        {invitationState(item) === "Active" && <button className="secondary-button" type="button" disabled={!mutationsEnabled || busy !== ""} onClick={() => void revokeInvitationRecord(item)}>{busy === `invitation:${item.id}` ? "Revoking invitation…" : `Revoke invitation for ${item.audienceHint}`}</button>}
      </article>)}
    </section>

    {activeSessions && <section aria-labelledby="evidence-session-inventory">
      <h3 id="evidence-session-inventory">Active external sessions</h3>
      {activeSessionsHasMore && <p>Showing the first {INVENTORY_LIMIT} active external sessions. More records are available.</p>}
      {sessionLoadState === "loading" && <p role="status">Loading active external sessions. Session changes remain unavailable until requester authority is confirmed.</p>}
      {sessionLoadState === "degraded" && <p role="alert">{sessionDegradedReason || "Current requester authority for external sessions could not be confirmed."} Previously loaded session records remain available, but session changes are disabled.</p>}
      {sessionLoadState === "unavailable" && activeSessions.length > 0 && <p role="alert">{sessionDegradedReason || "The latest active external sessions could not be loaded."} Previously loaded session records remain available, but session changes are disabled.</p>}
      {sessionLoadState === "unavailable" && activeSessions.length === 0 && <p role="alert">Active external sessions could not be loaded. Refresh the evidence request to try again; session changes are unavailable.</p>}
      {sessionLoadState === "ready" && activeSessions.length === 0 && <p>No active external sessions were returned for this evidence request.</p>}
      {visibleSessions.map((item) => <article key={item.id}>
        <h4>{item.audienceHint}</h4>
        <p>Started {formatDateTime(item.startedAt)} · expires {formatDateTime(item.expiresAt)}</p>
        <button className="secondary-button" type="button" disabled={!sessionMutationsEnabled || busy !== ""} onClick={() => void revokeExternalSession(item)}>{busy === `session:${item.id}` ? "Ending session…" : `End session for ${item.audienceHint}`}</button>
      </article>)}
    </section>}
  </section>;
}

function invitationState(item: EvidenceInvitationAdminItem) {
  if (item.revokedAt) return "Revoked";
  if (item.redemptions >= item.maxRedemptions) return "Used";
  if (new Date(item.expiresAt).valueOf() <= Date.now()) return "Expired";
  return "Active";
}

function formatDateTime(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.valueOf()) ? "Date unavailable" : new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(date);
}

const INVENTORY_LIMIT = 50;
const EXPIRY_OPTIONS = [
  { minutes: 60, label: "1 hour" },
  { minutes: 1440, label: "1 day" },
  { minutes: 10080, label: "7 days" },
  { minutes: 43200, label: "30 days" },
];

function invitationLink(token: string) {
  const url = new URL(window.location.href);
  url.search = "";
  url.hash = new URLSearchParams({ form_access: token }).toString();
  return url.toString();
}
