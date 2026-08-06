import { useState } from "react";
import { submitCaptureRequest } from "./api";
import { EmptyState } from "./components/EmptyState";
import { BankJourneysWorkspace } from "./components/BankJourneysWorkspace";
import { EvidenceWorkspace } from "./components/EvidenceWorkspace";
import { MattersWorkspace } from "./components/MattersWorkspace";
import { PremiumIllustration } from "./components/PremiumIllustration";
import { ProjectionHealthCard } from "./components/ProjectionHealthCard";
import { ProgramsWorkspace } from "./components/ProgramsWorkspace";
import { ReadinessPanel } from "./components/ReadinessPanel";
import { WorkspaceErrorBoundary } from "./components/WorkspaceErrorBoundary";
import { WorkItemIcon } from "./components/WorkItemIcon";
import type { AttentionItem, AuthorityResolution, CaptureRequest, EvidenceRequest, EvidenceSource, IntegrityFinding, PolicySummary, Readiness, WorkflowTask } from "./types";
import type { ProjectionHealth, ReconcileResult } from "./operationsTypes";

type LoadState = "idle" | "loading" | "live" | "unavailable";
type ConnectionState = "loading" | "live" | "sample" | "unavailable";

export function TodayView({ organizationName, items, dueSoon, connection, readiness, readinessState, onRouting, onCapture, onOpenItem }: { organizationName: string; items: AttentionItem[]; dueSoon: number; connection: ConnectionState; readiness: Readiness | null; readinessState: Exclude<LoadState, "idle">; onRouting: () => void; onCapture: () => void; onOpenItem: (item: AttentionItem) => void }) {
  const current = readiness?.baseline_known ? readiness.dimensions.current : "—";
  const openCount = connection === "unavailable" ? "—" : items.length;
  const dueCount = connection === "unavailable" ? "—" : dueSoon;
  const connectionLabel = connection === "live" ? "Live data" : connection === "sample" ? "Sample data" : connection === "unavailable" ? "Data unavailable" : "Connecting";
  return <>
    <header className="topbar"><div><span className="eyebrow">{organizationName} · Control Assurance</span><h1>Today</h1><p>Reviews, approvals and evidence requests assigned to you.</p></div><div className="topbar-actions"><span className={`connection ${connection}`}>{connectionLabel}</span><button id="authority-action" className="secondary-button" onClick={onRouting}>View approval route</button><button id="capture-action" className="primary-button" onClick={onCapture}>Open sample evidence request</button></div></header>
    <section className="brief-grid" id="today-brief"><div className="brief-stat"><span>Open items</span><strong>{openCount}</strong><small>Assigned to you</small></div><div className="brief-stat"><span>Due within 4 days</span><strong>{dueCount}</strong><small>Excludes overdue items</small></div><div className="brief-stat verified"><span>Items up to date</span><strong>{current}</strong><small>{readiness?.baseline_known ? "No action due" : "Population not connected"}</small></div></section>
    <ReadinessPanel readiness={readiness} state={readinessState}/>
    <section className="section-header"><div><h2>Assigned to you</h2><p>Reviews, approvals and evidence requests.</p></div></section>
    {connection === "unavailable" ? <EmptyState label="Work queue" title="Assigned work could not be loaded" description="The service is unavailable. No current work count is shown."/> : items.length ? <section className="attention-list">{items.map((item) => <AttentionCard item={item} key={item.id} onOpen={onOpenItem}/>)}</section> : <EmptyState label="Work queue" title="No assigned items" description="There are no open reviews, approvals or evidence requests assigned to you in the connected scope."/>}
    {readiness?.baseline_known && <section className="quiet-section"><div><span className="verified-dot"/> Readiness was updated {new Date(readiness.generated_at).toLocaleString()}</div><p>{readiness.dimensions.current} item{readiness.dimensions.current === 1 ? " is" : "s are"} currently recorded with no action due.</p></section>}
  </>;
}

export function ProgramsView({ organizationName }: { organizationName: string }) {
  return <><header className="topbar"><div><span className="eyebrow">{organizationName}</span><h1>Programs</h1><p>Ongoing obligations, safeguards, evidence checks and open issues.</p></div></header><ProgramsWorkspace/></>;
}

export function ExploreView({ organizationName }: { organizationName: string }) {
  return <><header className="topbar"><div><span className="eyebrow">{organizationName}</span><h1>Explore</h1><p>Connected workflows across obligations, evidence, decisions, responses and confirmed outcomes.</p></div></header><BankJourneysWorkspace/></>;
}

export function WorkView({ organizationName, tab, onTab, sources, requests, evidenceState, onEvidenceRetry }: { organizationName: string; tab: "matters" | "evidence"; onTab: (value: "matters" | "evidence") => void; sources: EvidenceSource[]; requests: EvidenceRequest[]; evidenceState: LoadState; onEvidenceRetry: () => void }) {
  const evidenceLoadState = evidenceState === "idle" ? "loading" : evidenceState;
  return <><header className="topbar"><div><span className="eyebrow">{organizationName}</span><h1>Work</h1><p>Issues, changes, evidence requests and the sources they rely on.</p></div></header><div className="workspace-tabs" role="tablist" aria-label="Work views"><button type="button" role="tab" aria-selected={tab === "matters"} className={tab === "matters" ? "active" : ""} onClick={() => onTab("matters")}>Issues and changes</button><button type="button" role="tab" aria-selected={tab === "evidence"} className={tab === "evidence" ? "active" : ""} onClick={() => onTab("evidence")}>Sources and evidence</button></div>{tab === "matters" ? <WorkspaceErrorBoundary label="Issues and changes"><MattersWorkspace/></WorkspaceErrorBoundary> : evidenceLoadState === "unavailable" ? <EmptyState label="Sources and evidence" title="Sources and evidence could not be loaded" description="The service is unavailable. No source-health or request totals are shown." action="Try again" onAction={onEvidenceRetry}/> : <EvidenceWorkspace sources={sources} requests={requests} state={evidenceLoadState}/>}</>;
}

export function ConfigureView({ policies, findings, tasks, projectionHealth, state, onRetry, onReconcile }: { policies: PolicySummary[]; findings: IntegrityFinding[]; tasks: WorkflowTask[]; projectionHealth: ProjectionHealth | null; state: LoadState; onRetry: () => void; onReconcile: () => Promise<ReconcileResult> }) {
  if (state === "idle" || state === "loading") return <><header className="topbar"><div><span className="eyebrow">Governance configuration</span><h1>Routing and approvals</h1><p>Responsibility, approval limits, delegation and escalation rules.</p></div></header><section className="workspace-loading" aria-live="polite" aria-busy="true">Loading routing configuration…</section></>;
  if (state === "unavailable") return <><header className="topbar"><div><span className="eyebrow">Governance configuration</span><h1>Routing and approvals</h1><p>Responsibility, approval limits, delegation and escalation rules.</p></div></header><EmptyState label="Routing and approvals" title="Routing configuration could not be loaded" description="The service is unavailable. No policy or integrity claims are shown." action="Try again" onAction={onRetry}/></>;
  return <><header className="topbar"><div><span className="eyebrow">Governance configuration</span><h1>Routing and approvals</h1><p>Responsibility, approval limits, delegation and escalation rules.</p></div></header><section className="configure-hero"><div><span className="eyebrow">Routing checks</span><h2>{findings.length ? `${findings.length} configuration issue${findings.length === 1 ? "" : "s"}` : "No blocking configuration issues"}</h2><p>Checks cover missing owners, unresolved selectors, duplicate priorities, expired delegations and missing authorizers.</p></div><PremiumIllustration variant="routing"/></section><section className="config-grid"><article className="config-card"><div className="section-header"><div><h2>Active policies</h2><p>Approved versions and effective dates.</p></div></div>{policies.length ? policies.map((policy) => <div className="policy-row" key={policy.id}><div><strong>{policy.name}</strong><span>{policy.code} · v{policy.version}</span></div><mark>{humanizeStatus(policy.status)}</mark></div>) : <EmptyState label="Routing policies" title="No active policies" description="There are no active routing policies in the current scope."/>}</article><article className="config-card"><div className="section-header"><div><h2>Integrity findings</h2><p>Configuration checks.</p></div></div>{findings.length ? findings.map((finding) => <div className={`finding-row severity-${finding.severity.toLowerCase()}`} key={`${finding.type}-${finding.summary}`}><strong>{finding.summary}</strong><span>{finding.required_action}</span></div>) : <div className="calm-empty"><span>✓</span><div><strong>No blocking routing gaps</strong><p>All evaluated routes resolved to an active principal.</p></div></div>}</article><article className="config-card wide"><div className="section-header"><div><h2>Workflow ownership</h2><p>Open tasks and current assignees.</p></div></div><div className="task-table">{tasks.map((task) => <div className="task-row" key={task.id}><div><strong>{task.title}</strong><span>{task.responsibility} · {task.step_key}</span></div><span>{task.context?.scope ?? task.context?.program ?? "Connected scope"}</span><mark>{humanizeStatus(task.status)}</mark></div>)}</div></article><ProjectionHealthCard health={projectionHealth} onReconcile={onReconcile}/></section></>;
}

function humanizeStatus(value: string) {
  return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}

function AttentionCard({ item, onOpen }: { item: AttentionItem; onOpen: (item: AttentionItem) => void }) {
  const parsed = Date.parse(item.due_at);
  const due = Number.isFinite(parsed) ? new Intl.DateTimeFormat(undefined, { month: "short", day: "numeric", hour: "numeric", minute: "2-digit" }).format(new Date(parsed)) : "Due date unavailable";
  const canOpen = Boolean(item.action_target_type && item.action_target_id);
  return <article className="attention-card"><div className="attention-icon"><WorkItemIcon type={item.type}/></div><div className="attention-content"><div className="card-kicker"><span>{item.state}</span><time>{due}</time></div><h3>{item.title}</h3><p>{item.why_now}</p><div className="card-meta"><span>{item.scope}</span><span>{item.evidence}</span><span>{item.owner}</span></div></div><div className="card-next"><span>Required action</span><strong>{item.primary_action}</strong>{canOpen && <button className="text-button" type="button" onClick={() => onOpen(item)}>Open workspace</button>}</div></article>;
}

export function RoutingPanel({ resolution, legalEntityName }: { resolution: AuthorityResolution | null; legalEntityName: string }) { return <div className="panel-content"><span className="eyebrow">Approval route</span><h2>Current authorizer</h2><p>The route uses the legal entity, issue type, importance, delegation and active policy version.</p>{resolution ? <div className="resolution-card"><div className="principal-avatar">CR</div><div><strong>{resolution.principal.display_name}</strong><span>{resolution.principal.role} · {resolution.principal.kind}</span></div><mark>Selected</mark></div> : <div className="skeleton">The approval route is unavailable. Confirm that an authority object is configured for this build.</div>}<dl className="explanation-list"><div><dt>Responsibility</dt><dd>Authorizer</dd></div><div><dt>Legal entity</dt><dd>{legalEntityName}</dd></div><div><dt>Importance</dt><dd>Critical · Executive approval</dd></div><div><dt>Policy</dt><dd>{resolution?.policy_version ?? "Unavailable"}</dd></div></dl><div className="sequence"><span>Control owner</span><i>→</i><span>Control Assurance</span><i>→</i><b>CRO</b></div></div>; }

export function CapturePanel({ request }: { request: CaptureRequest | null }) {
  const [answers, setAnswers] = useState<Record<string, string>>({}); const [receipt, setReceipt] = useState<string | null>(null); const [error, setError] = useState<string | null>(null);
  async function submit() { if (!request) return; setError(null); try { const result = await submitCaptureRequest(request.id, request.version, answers, request.source); setReceipt(`Submitted ${new Date(result.submitted_at).toLocaleString()}`); } catch (cause) { setError(cause instanceof Error ? cause.message : "Submission failed"); } }
  if (!request) return <div className="panel-content"><span className="eyebrow">Evidence request</span><h2>No sample request configured</h2><p>Set <code>VITE_SAMPLE_CAPTURE_REQUEST_ID</code> for a development build, or open a live request from Explore or Work.</p></div>;
  if (receipt) return <div className="panel-content"><span className="eyebrow">Submission receipt</span><h2>Response submitted</h2><PremiumIllustration variant="empty"/><p>{receipt}</p><p>The response has been recorded. It will still be checked against the evidence requirements.</p></div>;
  return <div className="panel-content"><span className="eyebrow">Evidence request · about {request.estimated_minutes} minutes</span><h2>{request.title}</h2><p>{request.purpose}</p><div className="why-you"><strong>Why you received this</strong><span>{request.why_you}</span></div><h3>Information already available</h3><dl className="known-facts">{Object.entries(request.known_facts).map(([key, value]) => <div key={key}><dt>{humanizeStatus(key)}</dt><dd>{value}</dd></div>)}</dl>{request.fields.map((field) => <label className="field" key={field.id}><span>{field.label}{field.required ? " *" : ""}</span>{field.type === "single_select" ? <select value={answers[field.id] ?? ""} onChange={(event) => setAnswers({ ...answers, [field.id]: event.target.value })}><option value="">Select one</option>{field.options?.map((option) => <option key={option}>{option}</option>)}</select> : <textarea value={answers[field.id] ?? ""} onChange={(event) => setAnswers({ ...answers, [field.id]: event.target.value })} placeholder={field.description}/>}</label>)}{error && <p className="error-text">{error}</p>}<div className="wizard-actions"><button className="primary-button" onClick={submit}>Review and submit</button></div></div>;
}
