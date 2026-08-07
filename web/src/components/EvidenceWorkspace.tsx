import { useEffect, useMemo, useState } from "react";
import type { EvidenceRequest, EvidenceSource } from "../types";
import { EmptyState } from "./EmptyState";

type LoadState = "loading" | "live" | "unavailable";
type Props = { sources: EvidenceSource[]; requests: EvidenceRequest[]; sourceState: LoadState; requestState: LoadState; targetID?: string; openFirst?: boolean; onOpenRequest: (id: string) => void };

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
  };
  return labels[value] ?? value.replaceAll("_", " ").toLowerCase().replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}

export function EvidenceWorkspace({ sources, requests, sourceState, requestState, targetID, openFirst = false, onOpenRequest }: Props) {
  const [query, setQuery] = useState("");
  const [requestStatus, setRequestStatus] = useState("");
  const [openID, setOpenID] = useState<string | undefined>(targetID);
  const sourceIssues = sources.filter((source) => source.health !== "CURRENT").length;
  const openRequests = requests.filter((request) => request.status === "READY" || request.status === "IN_PROGRESS").length;
  const filteredRequests = useMemo(() => requests.filter((request) => {
    const matchesStatus = !requestStatus || request.status === requestStatus;
    const text = `${request.title} ${request.purpose} ${request.why_you} ${request.audience_type}`.toLowerCase();
    return matchesStatus && text.includes(query.trim().toLowerCase());
  }), [requests, query, requestStatus]);
  const targetMissing = Boolean(requestState === "live" && targetID && !requests.some((request) => request.id === targetID));

  useEffect(() => {
    const id = targetID ?? (openFirst ? requests[0]?.id : undefined);
    if (!id) return;
    setOpenID(id);
    window.setTimeout(() => document.getElementById(`evidence-request-${id}`)?.scrollIntoView({ behavior: "smooth", block: "center" }), 80);
  }, [targetID, openFirst, requests]);

  if (requestState === "loading" && sourceState === "loading" && !requests.length && !sources.length) return <section id="evidence-workspace" className="workspace-loading" aria-live="polite" aria-busy="true">Loading evidence work…</section>;

  const headline = requestState === "unavailable" ? "Evidence requests are unavailable" : requestState === "loading" ? "Refreshing evidence requests" : `${openRequests} open evidence request${openRequests === 1 ? "" : "s"}`;
  return <div id="evidence-workspace">
    <section className="workspace-brief">
      <div><span className="eyebrow">Evidence</span><h2>{headline}</h2><p>Open requests first. Source status and history are available below when you need them.</p></div>
      <div className="workspace-brief-facts" aria-label="Evidence work summary"><span><strong>{requestState === "live" ? openRequests : "—"}</strong> open requests</span><span><strong>{sourceState === "live" ? sourceIssues : "—"}</strong> source issues</span></div>
    </section>
    {targetMissing && <EmptyState kind="not-found" label="Evidence request" title="This request could not be loaded" description="It may be outside your access or no longer available."/>}
    <section className="evidence-workbench">
      <div className="section-header"><div><h2>Evidence requests</h2><p>See what is needed, who should respond and when it is due.</p></div></div>
      <div className="evidence-toolbar"><label><span>Search requests</span><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search by title or purpose" disabled={requestState !== "live"}/></label><label><span>Status</span><select value={requestStatus} onChange={(event) => setRequestStatus(event.target.value)} disabled={requestState !== "live"}><option value="">All statuses</option><option value="READY">Response required</option><option value="IN_PROGRESS">In progress</option><option value="SUBMITTED">Submitted</option><option value="EXPIRED">Past due</option></select></label></div>
      {requestState === "unavailable" ? <EmptyState kind="unavailable" label="Requests" title="Evidence requests could not be loaded" description="Source status remains available below. Existing requests have not been changed."/> : requestState === "loading" ? <div className="workspace-loading compact" aria-live="polite" aria-busy="true">Refreshing evidence requests…</div> : filteredRequests.length ? <div className="request-list">{filteredRequests.map((request) => <details className={targetID === request.id ? "request-row targeted" : "request-row"} id={`evidence-request-${request.id}`} key={request.id} open={openID === request.id} onToggle={(event) => { if (event.currentTarget.open) setOpenID(request.id); else if (openID === request.id) setOpenID(undefined); }}><summary><div><strong>{request.title}</strong><span>{label(request.audience_type)} · about {request.estimated_minutes} min</span></div><div><mark>{label(request.status)}</mark><span>Due {new Date(request.deadline).toLocaleString()}</span></div><span className="request-disclosure">View details</span></summary><div className="request-detail"><p>{request.purpose}</p><dl>{Object.entries(request.known_facts).map(([factLabel, value]) => <div key={factLabel}><dt>{label(factLabel)}</dt><dd>{value}</dd></div>)}</dl><p><strong>Why this person:</strong> {request.why_you}</p><div className="request-actions">{["READY", "IN_PROGRESS"].includes(request.status) && <button className="primary-button" type="button" onClick={() => onOpenRequest(request.id)}>Open request</button>}<small>A response is recorded first. Evidence quality is assessed separately.</small></div></div></details>)}</div> : <EmptyState kind={requests.length ? "no-results" : "empty"} label="Requests" title={requests.length ? "No requests match these filters" : "No evidence requests in this scope"} description={requests.length ? "Change the search or status filter to see other requests." : "There are no evidence requests in the current scope."} action={query || requestStatus ? "Clear filters" : undefined} onAction={() => { setQuery(""); setRequestStatus(""); }}/>} 
    </section>
    <details className="source-inventory">
      <summary><div><span className="eyebrow">Sources</span><strong>{sourceState === "unavailable" ? "Source status is unavailable" : sourceIssues ? `${sourceIssues} source${sourceIssues === 1 ? " needs" : "s need"} review` : "No source issues in the loaded scope"}</strong></div><span>{sourceState === "live" ? `${sources.length} registered source${sources.length === 1 ? "" : "s"}` : "Coverage unavailable"}</span></summary>
      <div className="source-list">{sourceState === "unavailable" ? <EmptyState kind="unavailable" label="Sources" title="Source status could not be loaded" description="Evidence requests above remain available. No source conclusion is inferred."/> : sourceState === "loading" ? <div className="workspace-loading compact" aria-live="polite" aria-busy="true">Refreshing source status…</div> : sources.length ? sources.map((source) => <div className="source-row" key={source.id}><div className="source-icon"><SourceIcon type={source.type}/></div><div><strong>{source.name}</strong><span>{label(source.authority_class)} · {label(source.type)}</span></div><div className="source-freshness"><mark className={`health-${source.health.toLowerCase()}`}>{label(source.health)}</mark><span>{source.last_success_at ? `Last successful check ${new Date(source.last_success_at).toLocaleString()}` : "No successful check recorded"}</span></div></div>) : <EmptyState label="Sources" title="No sources in this scope" description="No official, system, document, staff or provider sources are registered in the current scope."/>}</div>
    </details>
    <section className="artifact-notice"><div className="artifact-notice-icon" aria-hidden="true">i</div><div><strong>Uploaded files are reviewed before they count as usable evidence.</strong><p>A file can be attached to a request immediately. It is not treated as usable evidence until the required inspection succeeds.</p></div></section>
  </div>;
}
