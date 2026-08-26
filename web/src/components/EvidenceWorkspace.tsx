import { useEffect, useMemo, useRef, useState } from "react";
import { declareWrongCaptureRecipient, reassignCaptureRecipient } from "../captureApi";
import { canRespondToEvidenceRequest } from "../evidenceAuthorization";
import type { EvidenceRequest, EvidenceSource } from "../types";
import { EmptyState } from "./EmptyState";

type LoadState = "loading" | "live" | "unavailable";
type Props = { sources: EvidenceSource[]; requests: EvidenceRequest[]; sourceState: LoadState; requestState: LoadState; actorPrincipalID?: string; evidenceScopeToken: number; targetID?: string; openFirst?: boolean; onOpenRequest: (id: string) => void; onRequestUpdated: (request: EvidenceRequest, scopeToken: number) => boolean };
type EditState = { reason: string; recipient: string; busy: boolean; error: string };

function SourceIcon({ type }: { type: string }) {
  const common = { width: 20, height: 20, viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeWidth: 1.7, strokeLinecap: "round" as const, strokeLinejoin: "round" as const, "aria-hidden": true };
  if (type === "REGULATORY") return <svg {...common}><path d="M4 9h16M6 9V20M10 9V20M14 9V20M18 9V20M3 20h18M12 3 3 8h18z"/></svg>;
  if (type === "SYSTEM") return <svg {...common}><ellipse cx="12" cy="5" rx="7" ry="3"/><path d="M5 5v7c0 1.7 3.1 3 7 3s7-1.3 7-3V5M5 12v7c0 1.7 3.1 3 7 3s7-1.3 7-3v-7"/></svg>;
  return <svg {...common}><path d="M6 3h9l3 3v15H6z"/><path d="M15 3v4h4M9 12h6M9 16h6"/></svg>;
}

function label(value: string) {
  const labels: Record<string, string> = {
    REGULATORY: "Regulator or official publication", SYSTEM: "Bank system", DOCUMENT: "Document or file", HUMAN: "Staff response", VENDOR: "External provider",
    SYSTEM_OF_RECORD: "Official bank record", AUTHORITATIVE: "Authoritative source", PRIMARY: "Primary source", SECONDARY: "Supporting source",
    CURRENT: "Up to date", DEGRADED: "Limited", STALE: "Out of date", UNAVAILABLE: "Unavailable", UNKNOWN: "Not checked",
    READY: "Response required", IN_PROGRESS: "Response in progress", SUBMITTED: "Response received", CANCELLED: "Cancelled", EXPIRED: "Past due", DRAFT: "Draft",
    ASSIGNED: "Assigned", REASSIGNMENT_REQUIRED: "Needs reassignment", LEGACY_UNASSIGNED: "Unassigned historical request",
  };
  return labels[value] ?? value.replaceAll("_", " ").toLowerCase().replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}

const operationalDateTime = new Intl.DateTimeFormat("en-NG", { dateStyle: "medium", timeStyle: "short" });

function formatOperationalDateTime(value: string) {
  return operationalDateTime.format(new Date(value));
}

const emptyEdit = (): EditState => ({ reason: "", recipient: "", busy: false, error: "" });

export function EvidenceWorkspace({ sources, requests, sourceState, requestState, actorPrincipalID, evidenceScopeToken, targetID, openFirst = false, onOpenRequest, onRequestUpdated }: Props) {
  const [query, setQuery] = useState("");
  const [requestStatus, setRequestStatus] = useState("");
  const [openID, setOpenID] = useState<string | undefined>(targetID);
  const [overrides, setOverrides] = useState<Record<string, EvidenceRequest>>({});
  const [edits, setEdits] = useState<Record<string, EditState>>({});
  const currentScopeToken = useRef(evidenceScopeToken);
  currentScopeToken.current = evidenceScopeToken;
  const currentRequests = useMemo(() => requests.map((request) => {
    const override = overrides[request.id];
    return override && override.version >= request.version ? override : request;
  }), [requests, overrides]);
  const sourceIssues = sources.filter((source) => source.health !== "CURRENT").length;
  const openRequests = currentRequests.filter((request) => request.status === "READY" || request.status === "IN_PROGRESS").length;
  const filteredRequests = useMemo(() => currentRequests.filter((request) => {
    const matchesStatus = !requestStatus || request.status === requestStatus;
    const text = `${request.title} ${request.purpose} ${request.why_you} ${request.audience_type}`.toLowerCase();
    return matchesStatus && text.includes(query.trim().toLowerCase());
  }), [currentRequests, query, requestStatus]);
  const targetMissing = Boolean(requestState === "live" && targetID && !currentRequests.some((request) => request.id === targetID));

  useEffect(() => {
    setOverrides({});
    setEdits({});
    setOpenID(targetID);
  }, [evidenceScopeToken]);
  useEffect(() => {
    const id = targetID ?? (openFirst ? currentRequests[0]?.id : undefined);
    if (!id) return;
    setOpenID(id);
    window.setTimeout(() => document.getElementById(`evidence-request-${id}`)?.scrollIntoView({ behavior: "smooth", block: "center" }), 80);
  }, [targetID, openFirst, currentRequests]);

  function updateEdit(id: string, patch: Partial<EditState>) {
    setEdits((current) => ({ ...current, [id]: { ...(current[id] ?? emptyEdit()), ...patch } }));
  }

  async function declareWrong(request: EvidenceRequest) {
    const edit = edits[request.id] ?? emptyEdit();
    const reason = edit.reason.trim();
    if (!reason || edit.busy) return;
    updateEdit(request.id, { busy: true, error: "" });
    try {
      const updated = await declareWrongCaptureRecipient(request.id, request.version, reason) as EvidenceRequest;
      if (currentScopeToken.current !== evidenceScopeToken) return;
      if (!onRequestUpdated(updated, evidenceScopeToken)) {
        return;
      }
      setOverrides((current) => ({ ...current, [request.id]: updated }));
      updateEdit(request.id, { busy: false, reason: "" });
    } catch {
      if (currentScopeToken.current !== evidenceScopeToken) return;
      updateEdit(request.id, { busy: false, error: "The request could not be returned. Reload it before trying again." });
    }
  }

  async function reassign(request: EvidenceRequest) {
    const edit = edits[request.id] ?? emptyEdit();
    const recipient = edit.recipient.trim();
    const reason = edit.reason.trim();
    if (!recipient || !reason || edit.busy) return;
    updateEdit(request.id, { busy: true, error: "" });
    const internal = request.audience_type === "INTERNAL";
    try {
      const updated = await reassignCaptureRecipient(
        request.id,
        request.version,
        internal ? { type: "INTERNAL_PRINCIPAL", principal_id: recipient } : { type: "EXTERNAL_AUDIENCE", audience: recipient },
        reason,
      ) as EvidenceRequest;
      if (currentScopeToken.current !== evidenceScopeToken) return;
      if (!onRequestUpdated(updated, evidenceScopeToken)) {
        return;
      }
      setOverrides((current) => ({ ...current, [request.id]: updated }));
      updateEdit(request.id, { busy: false, reason: "", recipient: "" });
    } catch {
      if (currentScopeToken.current !== evidenceScopeToken) return;
      updateEdit(request.id, { busy: false, error: "The recipient could not be changed. Check the current request and recipient, then try again." });
    }
  }

  if (requestState === "loading" && sourceState === "loading" && !currentRequests.length && !sources.length) return <section id="evidence-workspace" className="workspace-loading" aria-live="polite" aria-busy="true">Loading evidence work…</section>;

  const headline = requestState === "unavailable" ? "Evidence requests are unavailable" : requestState === "loading" ? "Refreshing evidence requests" : `${openRequests} open evidence request${openRequests === 1 ? "" : "s"}`;
  return <div id="evidence-workspace">
    <section className="workspace-brief">
      <div><span className="eyebrow">Evidence</span><h2>{headline}</h2><p>Open requests first. Source status and history are available below when you need them.</p></div>
      <div className="workspace-brief-facts" aria-label="Evidence work summary"><span><strong>{requestState === "live" ? openRequests : "—"}</strong> open request{requestState === "live" && openRequests === 1 ? "" : "s"}</span><span><strong>{sourceState === "live" ? sourceIssues : "—"}</strong> source issue{sourceState === "live" && sourceIssues === 1 ? "" : "s"}</span></div>
    </section>
    {targetMissing && <EmptyState kind="not-found" label="Evidence request" title="This request could not be loaded" description="It may be outside your access or no longer available."/>}
    <section className="evidence-workbench">
      <div className="section-header"><div><h2>Evidence requests</h2><p>See what is needed, who should respond and when it is due.</p></div></div>
      <div className="evidence-toolbar"><label><span>Search requests</span><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search by title or purpose" disabled={requestState !== "live"}/></label><label><span>Status</span><select value={requestStatus} onChange={(event) => setRequestStatus(event.target.value)} disabled={requestState !== "live"}><option value="">All statuses</option><option value="READY">Response required</option><option value="IN_PROGRESS">In progress</option><option value="SUBMITTED">Submitted</option><option value="EXPIRED">Past due</option></select></label></div>
      {requestState === "unavailable" ? <EmptyState kind="unavailable" label="Requests" title="Evidence requests could not be loaded" description="Source status remains available below. Existing requests have not been changed."/> : requestState === "loading" ? <div className="workspace-loading compact" aria-live="polite" aria-busy="true">Refreshing evidence requests…</div> : filteredRequests.length ? <div className="request-list">{filteredRequests.map((request) => {
        const edit = edits[request.id] ?? emptyEdit();
        const recipientState = request.recipient?.state ?? (request.recipient ? "ASSIGNED" : undefined);
        const isOpen = ["READY", "IN_PROGRESS"].includes(request.status);
        const isAssignedActor = canRespondToEvidenceRequest(request, actorPrincipalID);
        const canManage = isOpen && Boolean(actorPrincipalID) && request.created_by === actorPrincipalID && recipientState !== "LEGACY_UNASSIGNED";
        const recipientLabel = request.audience_type === "EXTERNAL" ? "External recipient" : request.recipient?.type === "INTERNAL_PRINCIPAL" ? "Assigned person" : undefined;
        const responseUnavailable = isOpen && recipientState === "ASSIGNED" && !isAssignedActor;
        return <details className={targetID === request.id ? "request-row targeted" : "request-row"} id={`evidence-request-${request.id}`} key={request.id} open={openID === request.id} onToggle={(event) => { if (event.currentTarget.open) setOpenID(request.id); else if (openID === request.id) setOpenID(undefined); }}><summary><div><strong>{request.title}</strong><span>{label(request.audience_type)} · about {request.estimated_minutes} min</span></div><div><mark>{label(recipientState === "REASSIGNMENT_REQUIRED" ? recipientState : request.status)}</mark><span>Due {formatOperationalDateTime(request.deadline)}</span></div><span className="request-disclosure">View details</span></summary><div className="request-detail"><p>{request.purpose}</p><dl>{Object.entries(request.known_facts).map(([factLabel, value]) => <div key={factLabel}><dt>{label(factLabel)}</dt><dd>{value}</dd></div>)}</dl><p><strong>Why this person:</strong> {request.why_you}</p>{recipientLabel && <p><strong>Current recipient:</strong> {recipientLabel}</p>}{recipientState === "REASSIGNMENT_REQUIRED" && <div className="inline-notice" role="status"><strong>Recipient correction required.</strong> {request.recipient?.issue_reason || "The assigned person reported that this request belongs elsewhere."}</div>}<div className="request-actions">{isAssignedActor && <button className="primary-button" type="button" onClick={() => onOpenRequest(request.id)}>Open request</button>}{responseUnavailable && <div className="inline-notice">{request.audience_type === "EXTERNAL" ? <strong>The external recipient must respond through the active invitation.</strong> : <strong>Only the assigned person can respond to this request.</strong>} {canManage ? <span>You created this request. Change the recipient if the assignment is wrong.</span> : <span>Ask the request creator to change the recipient if the assignment is wrong.</span>}</div>}<small>A response is recorded first. Evidence quality is assessed separately.</small></div>{isAssignedActor && <details className="recipient-lifecycle-action"><summary>This request isn&apos;t mine</summary><label className="capture-field"><span>Why should it be reassigned?</span><textarea value={edit.reason} maxLength={500} onChange={(event) => updateEdit(request.id, { reason: event.target.value, error: "" })}/></label>{edit.error && <p className="error-text" role="alert">{edit.error}</p>}<button className="secondary-button" type="button" disabled={!edit.reason.trim() || edit.busy} onClick={() => void declareWrong(request)}>{edit.busy ? "Returning…" : "Return to requester"}</button></details>}{canManage && <details className="recipient-lifecycle-action" open={recipientState === "REASSIGNMENT_REQUIRED"}><summary>{recipientState === "REASSIGNMENT_REQUIRED" ? "Reassign this request" : "Change recipient"}</summary><label className="capture-field"><span>{request.audience_type === "INTERNAL" ? "New person ID" : "New recipient address"}</span><input value={edit.recipient} onChange={(event) => updateEdit(request.id, { recipient: event.target.value, error: "" })} placeholder={request.audience_type === "INTERNAL" ? "Exact active principal ID" : "name@example.com"}/></label><label className="capture-field"><span>Reason for change</span><textarea value={edit.reason} maxLength={500} onChange={(event) => updateEdit(request.id, { reason: event.target.value, error: "" })}/></label>{request.audience_type !== "INTERNAL" && <p className="field-help">Changing the external recipient revokes existing invitation and session access.</p>}{edit.error && <p className="error-text" role="alert">{edit.error}</p>}<button className="secondary-button" type="button" disabled={!edit.recipient.trim() || !edit.reason.trim() || edit.busy} onClick={() => void reassign(request)}>{edit.busy ? "Reassigning…" : "Save recipient"}</button></details>}</div></details>;
      })}</div> : <EmptyState kind={currentRequests.length ? "no-results" : "empty"} label="Requests" title={currentRequests.length ? "No requests match these filters" : "No evidence requests in this scope"} description={currentRequests.length ? "Change the search or status filter to see other requests." : "There are no evidence requests in the current scope."} action={query || requestStatus ? "Clear filters" : undefined} onAction={() => { setQuery(""); setRequestStatus(""); }}/>} 
    </section>
    <details className="source-inventory">
      <summary><div><span className="eyebrow">Sources</span><strong>{sourceState === "unavailable" ? "Source status is unavailable" : sourceIssues ? `${sourceIssues} source${sourceIssues === 1 ? " needs" : "s need"} review` : "No source issues in the loaded scope"}</strong></div><span>{sourceState === "live" ? `${sources.length} registered source${sources.length === 1 ? "" : "s"}` : "Coverage unavailable"}</span></summary>
      <div className="source-list">{sourceState === "unavailable" ? <EmptyState kind="unavailable" label="Sources" title="Source status could not be loaded" description="Evidence requests remain available. Try again before assessing source freshness."/> : sourceState === "loading" ? <div className="workspace-loading compact" aria-live="polite">Refreshing source status…</div> : sources.length ? sources.map((source) => <div className="source-row" key={source.id}><div className="source-icon"><SourceIcon type={source.type}/></div><div><strong>{source.name}</strong><span>{label(source.authority_class)} · {label(source.type)}</span></div><div className="source-freshness"><mark className={`health-${source.health.toLowerCase()}`}>{label(source.health)}</mark><span>{source.last_success_at ? `Last successful check ${formatOperationalDateTime(source.last_success_at)}` : "No successful check recorded"}</span></div></div>) : <EmptyState label="Sources" title="No sources in this scope" description="No official, system, document, staff or provider sources are registered in the current scope."/>}</div>
    </details>
    <section className="artifact-notice"><div className="artifact-notice-icon" aria-hidden="true">i</div><div><strong>Uploaded files are reviewed before they count as usable evidence.</strong><p>A file can be attached to a request immediately. It is not treated as usable evidence until the required inspection succeeds.</p></div></section>
  </div>;
}
