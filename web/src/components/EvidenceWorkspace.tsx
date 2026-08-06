import { useEffect, useMemo, useState } from "react";
import type { EvidenceRequest, EvidenceSource } from "../types";
import { EmptyState } from "./EmptyState";

type Props = { sources: EvidenceSource[]; requests: EvidenceRequest[]; state: "loading" | "live" | "unavailable"; targetID?: string; openFirst?: boolean; onOpenRequest: (id: string) => void };

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
    READY: "Ready to send", IN_PROGRESS: "Response in progress", SUBMITTED: "Response received", CANCELLED: "Cancelled", EXPIRED: "Past due", DRAFT: "Draft",
  };
  return labels[value] ?? value.replaceAll("_", " ").toLowerCase().replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}

export function EvidenceWorkspace({ sources, requests, state, targetID, openFirst = false, onOpenRequest }: Props) {
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

  useEffect(() => {
    const id = targetID ?? (openFirst ? requests[0]?.id : undefined);
    if (!id) return;
    setOpenID(id);
    window.setTimeout(() => document.getElementById(`evidence-request-${id}`)?.scrollIntoView({ behavior: "smooth", block: "center" }), 80);
  }, [targetID, openFirst, requests]);

  return <div id="evidence-workspace">
    <section className="workspace-brief">
      <div><span className="eyebrow">Evidence review</span><h2>{state === "unavailable" ? "Evidence work could not be loaded" : `${openRequests} open evidence request${openRequests === 1 ? "" : "s"}`}</h2><p>Human requests and review work stay primary; the complete source inventory is available only when you need deeper context.</p></div>
      <div className="workspace-brief-facts" aria-label="Evidence work summary"><span><strong>{state === "live" ? openRequests : "—"}</strong> open requests</span><span><strong>{state === "live" ? sourceIssues : "—"}</strong> source exceptions</span></div>
    </section>
    <section className="evidence-workbench">
      <div className="section-header"><div><h2>Evidence requests</h2><p>Who is responding, what is unresolved and when the response is due.</p></div></div>
      <div className="evidence-toolbar"><label><span>Search requests</span><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Title, purpose, audience or reason"/></label><label><span>Status</span><select value={requestStatus} onChange={(event) => setRequestStatus(event.target.value)}><option value="">All statuses</option><option value="READY">Ready</option><option value="IN_PROGRESS">In progress</option><option value="SUBMITTED">Submitted</option><option value="EXPIRED">Past due</option></select></label></div>
      {state === "unavailable" ? <EmptyState label="Requests" title="Evidence requests could not be loaded" description="Try again when the service is available. Existing requests have not been changed."/> : filteredRequests.length ? <div className="request-list">{filteredRequests.map((request) => <details className={targetID === request.id ? "request-row targeted" : "request-row"} id={`evidence-request-${request.id}`} key={request.id} open={openID === request.id} onToggle={(event) => { if ((event.currentTarget as HTMLDetailsElement).open) setOpenID(request.id); else if (openID === request.id) setOpenID(undefined); }}><summary><div><strong>{request.title}</strong><span>{label(request.audience_type)} · about {request.estimated_minutes} min</span></div><div><mark>{label(request.status)}</mark><span>Due {new Date(request.deadline).toLocaleString()}</span></div><span className="request-disclosure">View details</span></summary><div className="request-detail"><p>{request.purpose}</p><dl>{Object.entries(request.known_facts).map(([factLabel, value]) => <div key={factLabel}><dt>{label(factLabel)}</dt><dd>{value}</dd></div>)}</dl><p><strong>Why this person:</strong> {request.why_you}</p><div className="request-actions">{["READY", "IN_PROGRESS"].includes(request.status) && <button className="primary-button" type="button" onClick={() => onOpenRequest(request.id)}>Respond to request</button>}<small>Submitting a response does not by itself confirm evidence sufficiency.</small></div></div></details>)}</div> : <EmptyState label="Requests" title={requests.length ? "No requests match these filters" : "No evidence requests in this scope"} description={requests.length ? "Change the search or status filter to see other evidence work." : "There are no draft, open or completed evidence requests in the current scope."} action={query || requestStatus ? "Clear filters" : undefined} onAction={() => { setQuery(""); setRequestStatus(""); }}/>} 
    </section>
    <details className="source-inventory">
      <summary><div><span className="eyebrow">Source inventory</span><strong>{sourceIssues ? `${sourceIssues} source${sourceIssues === 1 ? " needs" : "s need"} review` : "No source exception in the loaded scope"}</strong></div><span>{sources.length} registered source{sources.length === 1 ? "" : "s"}</span></summary>
      <div className="source-list">{state === "unavailable" ? <EmptyState label="Sources" title="Sources could not be loaded" description="Try again when the service is available. Existing records have not been changed."/> : sources.length ? sources.map((source) => <div className="source-row" key={source.id}><div className="source-icon"><SourceIcon type={source.type}/></div><div><strong>{source.name}</strong><span>{label(source.authority_class)} · {label(source.type)}</span></div><div className="source-freshness"><mark className={`health-${source.health.toLowerCase()}`}>{label(source.health)}</mark><span>{source.last_success_at ? `Last worked ${new Date(source.last_success_at).toLocaleString()}` : "No successful check recorded"}</span></div></div>) : <EmptyState label="Sources" title="No sources in this scope" description="No official, system, document, staff or provider sources are registered in the current scope."/>}</div>
    </details>
    <section className="artifact-notice"><div className="artifact-notice-icon" aria-hidden="true">i</div><div><strong>New files cannot be used until their inspection status is known.</strong><p>The current build records file size, type and SHA-256 integrity. Malware scanning and production file storage are not yet enabled.</p></div></section>
  </div>;
}
