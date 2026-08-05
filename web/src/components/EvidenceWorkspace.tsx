import type { EvidenceRequest, EvidenceSource } from "../types";
import { EmptyState } from "./EmptyState";
import { PremiumIllustration } from "./PremiumIllustration";

type Props = { sources: EvidenceSource[]; requests: EvidenceRequest[]; state: "loading" | "live" | "unavailable" };

function SourceIcon({ type }: { type: string }) {
  const common = { width: 20, height: 20, viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeWidth: 1.7, strokeLinecap: "round" as const, strokeLinejoin: "round" as const, "aria-hidden": true };
  if (type === "REGULATORY") return <svg {...common}><path d="M4 9h16M6 9V20M10 9V20M14 9V20M18 9V20M3 20h18M12 3 3 8h18z"/></svg>;
  if (type === "SYSTEM") return <svg {...common}><ellipse cx="12" cy="5" rx="7" ry="3"/><path d="M5 5v7c0 1.7 3.1 3 7 3s7-1.3 7-3V5M5 12v7c0 1.7 3.1 3 7 3s7-1.3 7-3v-7"/></svg>;
  return <svg {...common}><path d="M6 3h9l3 3v15H6z"/><path d="M15 3v4h4M9 12h6M9 16h6"/></svg>;
}

export function EvidenceWorkspace({ sources, requests, state }: Props) {
  const sourceIssues = sources.filter((source) => source.health !== "CURRENT").length;
  const openRequests = requests.filter((request) => request.status === "READY" || request.status === "IN_PROGRESS").length;
  return <>
    <section className="evidence-hero">
      <div><span className="eyebrow">Evidence operations</span><h2>{state === "unavailable" ? "Source and request data unavailable" : `${openRequests} open evidence request${openRequests === 1 ? "" : "s"}`}</h2><p>Source freshness, request status and submission deadlines for the current bank scope.</p></div>
      <PremiumIllustration variant="readiness"/>
    </section>
    <section className="evidence-summary" aria-label="Evidence summary">
      <div><span>Registered sources</span><strong>{state === "live" ? sources.length : "—"}</strong><small>Current scope</small></div>
      <div><span>Source issues</span><strong>{state === "live" ? sourceIssues : "—"}</strong><small>Stale, degraded or unavailable</small></div>
      <div><span>Open requests</span><strong>{state === "live" ? openRequests : "—"}</strong><small>Ready or in progress</small></div>
    </section>
    <section className="evidence-grid">
      <article className="config-card evidence-card">
        <div className="section-header"><div><h2>Sources</h2><p>Authority, freshness and latest successful observation.</p></div></div>
        {state === "unavailable" ? <EmptyState label="Sources" title="Source data unavailable" description="The source registry could not be loaded. Existing work is unchanged; retry when the API is available."/> : sources.length ? <div className="source-list">{sources.map((source) => <div className="source-row" key={source.id}><div className="source-icon"><SourceIcon type={source.type}/></div><div><strong>{source.name}</strong><span>{source.authority_class} · {source.type.replaceAll("_", " ")}</span></div><div className="source-freshness"><mark className={`health-${source.health.toLowerCase()}`}>{source.health}</mark><span>{source.last_success_at ? `Last success ${new Date(source.last_success_at).toLocaleString()}` : "No successful observation recorded"}</span></div></div>)}</div> : <EmptyState label="Sources" title="No sources registered" description="No authoritative or operational sources are registered in the current scope."/>}
      </article>
      <article className="config-card evidence-card">
        <div className="section-header"><div><h2>Evidence requests</h2><p>Recipient, status and response deadline.</p></div></div>
        {state === "unavailable" ? <EmptyState label="Requests" title="Request data unavailable" description="Evidence requests could not be loaded. Retry when the API is available."/> : requests.length ? <div className="request-list">{requests.map((request) => <details className="request-row" key={request.id}><summary><div><strong>{request.title}</strong><span>{request.audience_type} · {request.subject_type} · about {request.estimated_minutes} min</span></div><div><mark>{request.status.replaceAll("_", " ")}</mark><span>Due {new Date(request.deadline).toLocaleString()}</span></div><span className="request-disclosure">View</span></summary><div className="request-detail"><p>{request.purpose}</p><dl>{Object.entries(request.known_facts).map(([label, value]) => <div key={label}><dt>{label}</dt><dd>{value}</dd></div>)}</dl><p><strong>Recipient context:</strong> {request.why_you}</p></div></details>)}</div> : <EmptyState label="Requests" title="No evidence requests" description="There are no evidence requests in the current scope."/>}
      </article>
    </section>
    <section className="artifact-notice"><div className="artifact-notice-icon" aria-hidden="true">i</div><div><strong>Uploaded files remain unavailable until inspection is complete.</strong><p>The current foundation records file size, media type, storage key and SHA-256 integrity. Malware scanning and production object storage are not yet enabled.</p></div></section>
  </>;
}
