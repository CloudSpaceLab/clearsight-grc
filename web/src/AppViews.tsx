import { EmptyState } from "./components/EmptyState";
import { BankJourneysWorkspace } from "./components/BankJourneysWorkspace";
import { EvidenceWorkspace } from "./components/EvidenceWorkspace";
import { MattersWorkspace } from "./components/MattersWorkspace";
import { ProjectionHealthCard } from "./components/ProjectionHealthCard";
import { ProgramsWorkspace } from "./components/ProgramsWorkspace";
import { TodayInterventions } from "./components/TodayInterventions";
import { WorkspaceErrorBoundary } from "./components/WorkspaceErrorBoundary";
import type { AttentionItem, AuthorityResolution, EvidenceRequest, EvidenceSource, IntegrityFinding, PolicySummary, Readiness, WorkflowTask } from "./types";
import type { ProjectionHealth, ReconcileResult } from "./operationsTypes";

export { CapturePanel } from "./components/CapturePanel";

type LoadState = "idle" | "loading" | "live" | "unavailable";
type ConnectionState = "loading" | "live" | "sample" | "unavailable";

export function TodayView({ organizationName, items, connection, readiness, readinessState, onRouting, onCapture, onOpenItem }: { organizationName: string; items: AttentionItem[]; connection: ConnectionState; readiness: Readiness | null; readinessState: Exclude<LoadState, "idle">; onRouting: () => void; onCapture?: () => void; onOpenItem: (item: AttentionItem) => void }) {
  const connectionLabel = connection === "live" ? "Live data" : connection === "sample" ? "Reference data" : connection === "unavailable" ? "Data unavailable" : "Connecting";
  return <>
    <header className="topbar today-topbar">
      <div><span className="eyebrow">{organizationName} · Operating brief</span><h1>Today</h1><p>Only work that requires your review, authority, evidence or outcome confirmation.</p></div>
      <div className="topbar-actions"><span className={`connection ${connection}`}>{connectionLabel}</span><button id="authority-action" className="secondary-button" onClick={onRouting}>View approval route</button>{onCapture && <button id="capture-action" className="secondary-button" onClick={onCapture}>Respond to evidence request</button>}</div>
    </header>
    <TodayInterventions items={items} connection={connection} readiness={readiness} readinessState={readinessState} onOpenItem={onOpenItem}/>
  </>;
}

export function ProgramsView({ organizationName, targetID, openFirst }: { organizationName: string; targetID?: string; openFirst?: boolean }) {
  return <><header className="topbar"><div><span className="eyebrow">{organizationName}</span><h1>Programs</h1><p>Ongoing obligations, safeguards, evidence checks and open issues.</p></div></header><ProgramsWorkspace targetID={targetID} openFirst={openFirst}/></>;
}

export function ExploreView({ organizationName }: { organizationName: string }) {
  return <><header className="topbar"><div><span className="eyebrow">{organizationName}</span><h1>Explore</h1><p>Connected reference workflows across obligations, evidence, decisions, responses and confirmed outcomes.</p></div></header><BankJourneysWorkspace/></>;
}

export function WorkView({ organizationName, tab, onTab, sources, requests, evidenceState, onEvidenceRetry, matterTargetID, openFirstMatter, evidenceTargetID, openFirstEvidence, onOpenEvidence }: { organizationName: string; tab: "matters" | "evidence"; onTab: (value: "matters" | "evidence") => void; sources: EvidenceSource[]; requests: EvidenceRequest[]; evidenceState: LoadState; onEvidenceRetry: () => void; matterTargetID?: string; openFirstMatter?: boolean; evidenceTargetID?: string; openFirstEvidence?: boolean; onOpenEvidence: (id: string) => void }) {
  const evidenceLoadState = evidenceState === "idle" ? "loading" : evidenceState;
  return <>
    <header className="topbar"><div><span className="eyebrow">{organizationName}</span><h1>Work</h1><p>Review decisions, exceptions and evidence that still require a person.</p></div></header>
    <div className="workspace-tabs" role="tablist" aria-label="Work views"><button type="button" role="tab" aria-selected={tab === "matters"} className={tab === "matters" ? "active" : ""} onClick={() => onTab("matters")}>Issues and changes</button><button type="button" role="tab" aria-selected={tab === "evidence"} className={tab === "evidence" ? "active" : ""} onClick={() => onTab("evidence")}>Evidence review</button></div>
    {tab === "matters" ? <WorkspaceErrorBoundary label="Issues and changes"><MattersWorkspace targetID={matterTargetID} openFirst={openFirstMatter}/></WorkspaceErrorBoundary> : evidenceLoadState === "unavailable" ? <div id="evidence-workspace"><EmptyState label="Evidence review" title="Evidence work could not be loaded" description="The service is unavailable. No source-health or request totals are shown." action="Try again" onAction={onEvidenceRetry}/></div> : <EvidenceWorkspace sources={sources} requests={requests} state={evidenceLoadState} targetID={evidenceTargetID} openFirst={openFirstEvidence} onOpenRequest={onOpenEvidence}/>} 
  </>;
}

export function ConfigureView({ policies, findings, tasks, projectionHealth, state, onRetry, onReconcile }: { policies: PolicySummary[]; findings: IntegrityFinding[]; tasks: WorkflowTask[]; projectionHealth: ProjectionHealth | null; state: LoadState; onRetry: () => void; onReconcile: () => Promise<ReconcileResult> }) {
  if (state === "idle" || state === "loading") return <div id="configure-workspace"><header className="topbar"><div><span className="eyebrow">Governance configuration</span><h1>Routing and approvals</h1><p>Responsibility, approval limits, delegation and escalation rules.</p></div></header><section className="workspace-loading" aria-live="polite" aria-busy="true">Loading routing configuration…</section></div>;
  if (state === "unavailable") return <div id="configure-workspace"><header className="topbar"><div><span className="eyebrow">Governance configuration</span><h1>Routing and approvals</h1><p>Responsibility, approval limits, delegation and escalation rules.</p></div></header><EmptyState label="Routing and approvals" title="Routing configuration could not be loaded" description="The service is unavailable. No policy or integrity claims are shown." action="Try again" onAction={onRetry}/></div>;
  return <div id="configure-workspace">
    <header className="topbar"><div><span className="eyebrow">Governance configuration</span><h1>Routing and approvals</h1><p>Responsibility, approval limits, delegation and escalation rules.</p></div></header>
    <section className="workspace-brief configure-brief"><div><span className="eyebrow">Routing checks</span><h2>{findings.length ? `${findings.length} configuration issue${findings.length === 1 ? "" : "s"}` : "No blocking configuration issues"}</h2><p>Missing owners, unresolved selectors, duplicate priorities, expired delegations and missing authorizers are surfaced here.</p></div></section>
    <section className="config-grid"><article className="config-card"><div className="section-header"><div><h2>Active policies</h2><p>Approved versions and effective dates.</p></div></div>{policies.length ? policies.map((policy) => <div className="policy-row" key={policy.id}><div><strong>{policy.name}</strong><span>{policy.code} · v{policy.version}</span></div><mark>{humanizeStatus(policy.status)}</mark></div>) : <EmptyState label="Routing policies" title="No active policies" description="There are no active routing policies in the current scope."/>}</article><article className="config-card"><div className="section-header"><div><h2>Integrity findings</h2><p>Configuration checks.</p></div></div>{findings.length ? findings.map((finding) => <div className={`finding-row severity-${finding.severity.toLowerCase()}`} key={`${finding.type}-${finding.summary}`}><strong>{finding.summary}</strong><span>{finding.required_action}</span></div>) : <div className="calm-empty"><span>✓</span><div><strong>No blocking routing gaps</strong><p>All evaluated routes resolved to an active principal.</p></div></div>}</article><article className="config-card wide"><div className="section-header"><div><h2>Workflow ownership</h2><p>Open tasks and current assignees.</p></div></div><div className="task-table">{tasks.length ? tasks.map((task) => <div className="task-row" key={task.id}><div><strong>{task.title}</strong><span>{task.responsibility} · {task.step_key}</span></div><span>{task.context?.scope ?? task.context?.program ?? "Connected scope"}</span><mark>{humanizeStatus(task.status)}</mark></div>) : <div className="calm-empty"><span>✓</span><div><strong>No unassigned workflow tasks</strong><p>The current configuration queue contains no open ownership work.</p></div></div>}</div></article><ProjectionHealthCard health={projectionHealth} onReconcile={onReconcile}/></section>
  </div>;
}

function humanizeStatus(value: string) {
  return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}

export function RoutingPanel({ resolution, legalEntityName }: { resolution: AuthorityResolution | null; legalEntityName: string }) {
  return <div className="panel-content"><span className="eyebrow">Approval route</span><h2>Current authorizer</h2><p>The route uses the legal entity, issue type, importance, delegation and active policy version.</p>{resolution ? <div className="resolution-card"><div className="principal-avatar">CR</div><div><strong>{resolution.principal.display_name}</strong><span>{resolution.principal.role} · {resolution.principal.kind}</span></div><mark>Selected</mark></div> : <div className="skeleton">The approval route is unavailable. Confirm that an authority object is configured for this build.</div>}<dl className="explanation-list"><div><dt>Responsibility</dt><dd>Authorizer</dd></div><div><dt>Legal entity</dt><dd>{legalEntityName}</dd></div><div><dt>Importance</dt><dd>Critical · Executive approval</dd></div><div><dt>Policy</dt><dd>{resolution?.policy_version ?? "Unavailable"}</dd></div></dl><div className="sequence"><span>Control owner</span><i>→</i><span>Control Assurance</span><i>→</i><b>CRO</b></div></div>;
}
