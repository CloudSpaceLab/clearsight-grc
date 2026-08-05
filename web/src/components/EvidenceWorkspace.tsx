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

function sourceTypeLabel(value: string) {
  switch (value) {
    case "REGULATORY": return "Regulator or official publication";
    case "SYSTEM": return "Bank system";
    case "DOCUMENT": return "Document or file";
    case "HUMAN": return "Staff response";
    case "VENDOR": return "External provider";
    default: return value.replaceAll("_", " ").toLowerCase();
  }
}

function authorityLabel(value: string) {
  switch (value) {
    case "SYSTEM_OF_RECORD": return "Official bank record";
    case "AUTHORITATIVE": return "Authoritative source";
    case "PRIMARY": return "Primary source";
    case "SECONDARY": return "Supporting source";
    default: return value.replaceAll("_", " ").toLowerCase();
  }
}

function healthLabel(value: string) {
  switch (value) {
    case "CURRENT": return "Up to date";
    case "DEGRADED": return "Limited";
    case "STALE": return "Out of date";
    case "UNAVAILABLE": return "Unavailable";
    case "UNKNOWN": return "Not checked";
    default: return value.replaceAll("_", " ").toLowerCase();
  }
}

function requestStatusLabel(value: string) {
  switch (value) {
    case "READY": return "Ready to send";
    case "IN_PROGRESS": return "Response in progress";
    case "SUBMITTED": return "Response received";
    case "CANCELLED": return "Cancelled";
    case "EXPIRED": return "Past due";
    case "DRAFT": return "Draft";
    default: return value.replaceAll("_", " ").toLowerCase();
  }
}

export function EvidenceWorkspace({ sources, requests, state }: Props) {
  const sourceIssues = sources.filter((source) => source.health !== "CURRENT").length;
  const openRequests = requests.filter((request) => request.status === "READY" || request.status === "IN_PROGRESS").length;
  return <>
    <section className="evidence-hero">
      <div><span className="eyebrow">Sources and evidence</span><h2>{state === "unavailable" ? "Source and request data could not be loaded" : `${openRequests} open evidence request${openRequests === 1 ? "" : "s"}`}</h2><p>Check whether important sources are up to date and see which evidence responses are still outstanding.</p></div>
      <PremiumIllustration variant="readiness"/>
    </section>
    <section className="evidence-summary" aria-label="Evidence summary">
      <div><span>Sources</span><strong>{state === "live" ? sources.length : "—"}</strong><small>Registered in this bank scope</small></div>
      <div><span>Need review</span><strong>{state === "live" ? sourceIssues : "—"}</strong><small>Out of date, limited, unavailable or not checked</small></div>
      <div><span>Open requests</span><strong>{state === "live" ? openRequests : "—"}</strong><small>Ready to send or awaiting a response</small></div>
    </section>
    <section className="evidence-grid">
      <article className="config-card evidence-card">
        <div className="section-header"><div><h2>Sources</h2><p>What each source is used for and when it last worked.</p></div></div>
        {state === "unavailable" ? <EmptyState label="Sources" title="Sources could not be loaded" description="Try again when the service is available. Existing records have not been changed."/> : sources.length ? <div className="source-list">{sources.map((source) => <div className="source-row" key={source.id}><div className="source-icon"><SourceIcon type={source.type}/></div><div><strong>{source.name}</strong><span>{authorityLabel(source.authority_class)} · {sourceTypeLabel(source.type)}</span></div><div className="source-freshness"><mark className={`health-${source.health.toLowerCase()}`}>{healthLabel(source.health)}</mark><span>{source.last_success_at ? `Last worked ${new Date(source.last_success_at).toLocaleString()}` : "No successful check recorded"}</span></div></div>)}</div> : <EmptyState label="Sources" title="No sources in this scope" description="No official, system, document, staff or provider sources are registered in the current scope."/>}
      </article>
      <article className="config-card evidence-card">
        <div className="section-header"><div><h2>Evidence requests</h2><p>Who is responding, what is needed and when it is due.</p></div></div>
        {state === "unavailable" ? <EmptyState label="Requests" title="Evidence requests could not be loaded" description="Try again when the service is available. Existing requests have not been changed."/> : requests.length ? <div className="request-list">{requests.map((request) => <details className="request-row" key={request.id}><summary><div><strong>{request.title}</strong><span>{request.audience_type.replaceAll("_", " ").toLowerCase()} · about {request.estimated_minutes} min</span></div><div><mark>{requestStatusLabel(request.status)}</mark><span>Due {new Date(request.deadline).toLocaleString()}</span></div><span className="request-disclosure">View details</span></summary><div className="request-detail"><p>{request.purpose}</p><dl>{Object.entries(request.known_facts).map(([label, value]) => <div key={label}><dt>{label}</dt><dd>{value}</dd></div>)}</dl><p><strong>Why this person:</strong> {request.why_you}</p></div></details>)}</div> : <EmptyState label="Requests" title="No evidence requests in this scope" description="There are no draft, open or completed evidence requests in the current scope."/>}
      </article>
    </section>
    <section className="artifact-notice"><div className="artifact-notice-icon" aria-hidden="true">i</div><div><strong>New files cannot be used until their inspection status is known.</strong><p>The current build records file size, type and SHA-256 integrity. Malware scanning and production file storage are not yet enabled.</p></div></section>
  </>;
}
