import { EmptyState } from "./components/EmptyState";
import { AutomationPolicies } from "./components/AutomationPolicies";
import { BankJourneysWorkspace } from "./components/BankJourneysWorkspace";
import { EvidenceWorkspace } from "./components/EvidenceWorkspace";
import { MattersWorkspace } from "./components/MattersWorkspace";
import { ProjectionHealthCard } from "./components/ProjectionHealthCard";
import { ProgramsWorkspace } from "./components/ProgramsWorkspace";
import { TodayInterventions } from "./components/TodayInterventions";
import { WorkspaceErrorBoundary } from "./components/WorkspaceErrorBoundary";
import type { AttentionItem, AutomationPolicy, AuthorityResolution, EvidenceRequest, EvidenceSource, IntegrityFinding, PolicySummary, Readiness, WorkflowTask } from "./types";
import type { ProjectionHealth, ReconcileResult } from "./operationsTypes";

export { CapturePanel } from "./components/CapturePanel";

type LoadState = "idle" | "loading" | "live" | "unavailable";
type SectionLoadState = Exclude<LoadState, "idle">;
type ConnectionState = "loading" | "live" | "sample" | "unavailable";
export type RoutingLoadState = "loading" | "live" | "unavailable" | "forbidden" | "not-found";

export function TodayView({ organizationName, items, connection, generatedAt, readiness, readinessState, onCapture, onOpenItem, onInspectAuthority }: { organizationName: string; items: AttentionItem[]; connection: ConnectionState; generatedAt?: string; readiness: Readiness | null; readinessState: SectionLoadState; onCapture?: () => void; onOpenItem: (item: AttentionItem) => void; onInspectAuthority: (item: AttentionItem) => void }) {
  const connectionLabel = connection === "live" ? generatedAt ? `Updated ${formatShortTime(generatedAt)}` : "Connected data" : connection === "sample" ? "Reference data" : connection === "unavailable" ? "Data unavailable" : "Connecting";
  return <>
    <header className="topbar today-topbar">
      <div><span className="eyebrow">{organizationName}</span><h1>Today</h1><p>Reviews, approvals, evidence responses and outcome checks that need you today.</p></div>
      <div className="topbar-actions"><span className={`connection ${connection}`}>{connectionLabel}</span>{onCapture && <><div className="today-desktop-actions"><button className="secondary-button" onClick={onCapture}>Respond to evidence request</button></div><details className="today-mobile-actions"><summary>More actions</summary><div><button className="secondary-button" onClick={onCapture}>Respond to evidence request</button></div></details></>}</div>
    </header>
    <TodayInterventions items={items} connection={connection} readiness={readiness} readinessState={readinessState} onOpenItem={onOpenItem} onInspectAuthority={onInspectAuthority}/>
  </>;
}

export function ProgramsView({ organizationName, actorPrincipalID, canConfigureSources, targetID, openFirst }: { organizationName: string; actorPrincipalID?: string; canConfigureSources?: boolean; targetID?: string; openFirst?: boolean }) {
  return <><header className="topbar"><div><span className="eyebrow">{organizationName}</span><h1>Programs</h1><p>Ongoing obligations, safeguards, evidence checks and open issues.</p></div></header><ProgramsWorkspace targetID={targetID} openFirst={openFirst} actorPrincipalID={actorPrincipalID} canConfigureSources={canConfigureSources}/></>;
}

export function ExploreView({ organizationName }: { organizationName: string }) {
  return <><header className="topbar"><div><span className="eyebrow">{organizationName}</span><h1>Explore</h1><p>Review sample compliance workflows from source requirement to confirmed outcome.</p></div></header><BankJourneysWorkspace/></>;
}

export function WorkView({ organizationName, tab, onTab, sources, requests, evidenceSourceState, evidenceRequestState, onEvidenceRetry, matterTargetID, openFirstMatter, evidenceTargetID, openFirstEvidence, onOpenEvidence }: { organizationName: string; tab: "matters" | "evidence"; onTab: (value: "matters" | "evidence") => void; sources: EvidenceSource[]; requests: EvidenceRequest[]; evidenceSourceState: SectionLoadState; evidenceRequestState: SectionLoadState; onEvidenceRetry: () => void; matterTargetID?: string; openFirstMatter?: boolean; evidenceTargetID?: string; openFirstEvidence?: boolean; onOpenEvidence: (id: string) => void }) {
  return <>
    <header className="topbar"><div><span className="eyebrow">{organizationName}</span><h1>Work</h1><p>Review decisions, exceptions and evidence that still need attention.</p></div></header>
    <nav className="workspace-tabs" aria-label="Work views"><button type="button" aria-current={tab === "matters" ? "page" : undefined} className={tab === "matters" ? "active" : ""} onClick={() => onTab("matters")}>Issues and changes</button><button type="button" aria-current={tab === "evidence" ? "page" : undefined} className={tab === "evidence" ? "active" : ""} onClick={() => onTab("evidence")}>Evidence review</button></nav>
    {tab === "matters" ? <WorkspaceErrorBoundary label="Issues and changes"><MattersWorkspace targetID={matterTargetID} openFirst={openFirstMatter}/></WorkspaceErrorBoundary> : <EvidenceWorkspace sources={sources} requests={requests} sourceState={evidenceSourceState} requestState={evidenceRequestState} targetID={evidenceTargetID} openFirst={openFirstEvidence} onOpenRequest={onOpenEvidence}/>} 
    {tab === "evidence" && (evidenceSourceState === "unavailable" || evidenceRequestState === "unavailable") && <div className="workspace-recovery-actions"><button className="secondary-button" type="button" onClick={onEvidenceRetry}>Retry unavailable evidence data</button></div>}
  </>;
}

export function ConfigureView({ policies, policyState, findings, integrityState, tasks, taskState, projectionHealth, projectionState, automationPolicies, automationPolicyState, state, onRetry, onReconcile }: { policies: PolicySummary[]; policyState: SectionLoadState; findings: IntegrityFinding[]; integrityState: SectionLoadState; tasks: WorkflowTask[]; taskState: SectionLoadState; projectionHealth: ProjectionHealth | null; projectionState: SectionLoadState; automationPolicies: AutomationPolicy[]; automationPolicyState: SectionLoadState; state: LoadState; onRetry: () => void; onReconcile: () => Promise<ReconcileResult> }) {
  if (state === "idle" || state === "loading") return <div id="configure-workspace"><header className="topbar"><div><span className="eyebrow">Governance configuration</span><h1>Routing and approvals</h1><p>Responsibility, approval limits, delegation and escalation rules.</p></div></header><section className="workspace-loading" aria-live="polite" aria-busy="true">Loading routing configuration…</section></div>;
  if (state === "unavailable") return <div id="configure-workspace"><header className="topbar"><div><span className="eyebrow">Governance configuration</span><h1>Routing and approvals</h1><p>Responsibility, approval limits, delegation and escalation rules.</p></div></header><EmptyState kind="unavailable" label="Routing and approvals" title="Routing configuration could not be loaded" description="Try again before reviewing or changing routing and approvals." action="Try again" onAction={onRetry}/></div>;
  const partial = [policyState, integrityState, taskState, projectionState, automationPolicyState].some((value) => value === "unavailable");
  return <div id="configure-workspace">
    <header className="topbar"><div><span className="eyebrow">Governance configuration</span><h1>Routing and approvals</h1><p>Responsibility, approval limits, delegation and escalation rules.</p></div></header>
    {partial && <div className="inline-notice" role="status"><strong>Some configuration sections could not be loaded.</strong> Review the available sections or <button className="text-button" type="button" onClick={onRetry}>try again</button>.</div>}
    <section className="workspace-brief configure-brief"><div><span className="eyebrow">Routing checks</span><h2>{integrityState === "unavailable" ? "Routing checks unavailable" : findings.length ? `${findings.length} configuration issue${findings.length === 1 ? "" : "s"}` : "No blocking configuration issues"}</h2><p>{integrityState === "unavailable" ? "Try again to check for missing owners, expired delegations and unresolved approval routes." : findings.length ? "Review each issue before changing routing or approvals." : "No missing owners, expired delegations or unresolved approval routes were found in the latest checks."}</p></div></section>
    <section className="config-grid">
      <article className="config-card"><div className="section-header"><div><h2>Active policies</h2><p>Approved versions and effective dates.</p></div></div>{policyState === "unavailable" ? <EmptyState kind="unavailable" label="Routing policies" title="Routing policies are unavailable" description="Policy details could not be loaded. Try again before changing routing."/> : policyState === "loading" ? <div className="workspace-loading compact" aria-live="polite">Loading policies…</div> : policies.length ? policies.map((policy) => <div className="policy-row" key={policy.id}><div><strong>{policy.name}</strong><span>{policy.code} · v{policy.version}{policy.effective_from ? ` · effective ${formatDate(policy.effective_from)}` : " · effective date unavailable"}</span></div><mark>{humanizeStatus(policy.status)}</mark></div>) : <EmptyState label="Routing policies" title="No active policies" description="There are no active routing policies in the current scope."/>}</article>
      <article className="config-card"><div className="section-header"><div><h2>Integrity findings</h2><p>Configuration checks.</p></div></div>{integrityState === "unavailable" ? <EmptyState kind="unavailable" label="Integrity findings" title="Routing checks are unavailable" description="Routing checks could not be completed. Try again before changing approval routes."/> : integrityState === "loading" ? <div className="workspace-loading compact" aria-live="polite">Loading routing checks…</div> : findings.length ? findings.map((finding) => <div className={`finding-row severity-${finding.severity.toLowerCase()}`} key={`${finding.type}-${finding.summary}`}><strong>{finding.summary}</strong><span>{finding.required_action}</span></div>) : <div className="calm-empty"><span>✓</span><div><strong>No blocking routing gaps</strong><p>All checked routes have at least one eligible approver.</p></div></div>}</article>
      <article className="config-card wide"><div className="section-header"><div><h2>Workflow ownership</h2><p>Open tasks and current assignees.</p></div></div>{taskState === "unavailable" ? <EmptyState kind="unavailable" label="Workflow ownership" title="Workflow ownership is unavailable" description="Assigned work could not be loaded. Try again before changing task ownership."/> : taskState === "loading" ? <div className="workspace-loading compact" aria-live="polite">Loading workflow ownership…</div> : <div className="task-table">{tasks.length ? tasks.map((task) => <div className="task-row" key={task.id}><div><strong>{task.title}</strong><span>{task.responsibility} · {task.step_key}</span></div><span>{task.context?.scope ?? task.context?.program ?? "Scope not provided"}</span><mark>{humanizeStatus(task.status)}</mark></div>) : <div className="calm-empty"><span>✓</span><div><strong>No unassigned workflow tasks</strong><p>Every open task in this scope has an assignee.</p></div></div>}</div>}</article>
      <ProjectionHealthCard health={projectionHealth} state={projectionState} onReconcile={onReconcile}/>
    </section>
    <AutomationPolicies policies={automationPolicies} state={automationPolicyState}/>
  </div>;
}

function humanizeStatus(value: string) { return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase()); }

export function RoutingPanel({ resolution, item, legalEntityName, state }: { resolution: AuthorityResolution | null; item: AttentionItem | null; legalEntityName: string; state: RoutingLoadState }) {
  if (state === "loading") return <div className="panel-content"><span className="eyebrow">Authority</span><h2>Checking approval authority</h2><p aria-live="polite" aria-busy="true">Checking responsibility and the current approval policy…</p></div>;
  if (state === "forbidden") return <div className="panel-content"><EmptyState kind="forbidden" label="Authority" title="Authority details are restricted" description="Your role cannot view approvers or policy details for this item."/></div>;
  if (state === "not-found") return <div className="panel-content"><EmptyState kind="not-found" label="Authority" title="Authority could not be resolved for this item" description="The linked record or routing context is no longer available."/></div>;
  if (state === "unavailable" || !resolution || !item) return <div className="panel-content"><EmptyState kind="unavailable" label="Authority" title="Authority resolution is unavailable" description="Try again before approving or assigning this item."/></div>;
  const candidates = resolution.candidate_principals ?? [];
  const hasResolvedPrincipal = Boolean(resolution.principal?.id);
  return <div className="panel-content"><span className="eyebrow">Authority · {item.authority?.responsibility ?? "Current responsibility"}</span><h2>Authority for this item</h2><p><strong>{item.title}</strong></p><h3>Approval details</h3>{hasResolvedPrincipal && <div className="resolution-card"><div className="principal-avatar">{initials(resolution.principal.display_name)}</div><div><strong>{resolution.principal.display_name}</strong><span>{resolution.principal.role} · {resolution.principal.kind}</span></div><mark>Resolved</mark></div>}{candidates.length > 1 && <section className="authority-candidates"><h3>Eligible approvers</h3><p>{resolution.strategy ? `Selection method: ${humanizeStatus(resolution.strategy)}` : "More than one person is eligible under the current policy."}</p><ul>{candidates.map((candidate) => <li key={candidate.id}><strong>{candidate.display_name}</strong><span>{candidate.role} · {candidate.kind}</span></li>)}</ul></section>}<dl className="explanation-list"><div><dt>Legal entity</dt><dd>{legalEntityName}</dd></div><div><dt>Responsibility</dt><dd>{item.authority?.responsibility ?? "Unavailable"}</dd></div>{item.authority?.decision_type && <div><dt>Decision type</dt><dd>{humanizeStatus(item.authority.decision_type)}</dd></div>}<div><dt>Materiality</dt><dd>{item.authority ? String(item.authority.materiality) : "Unavailable"}</dd></div><div><dt>Policy version</dt><dd>{resolution.policy_version || "Unavailable"}</dd></div><div><dt>Why this route</dt><dd>{resolution.explanation || "No additional explanation was provided."}</dd></div></dl></div>;
}

function initials(value: string) { const parts = value.trim().split(/\s+/).filter(Boolean); const first = parts.at(0)?.at(0) ?? value.at(0) ?? ""; const last = parts.length > 1 ? parts.at(-1)?.at(0) ?? "" : value.at(1) ?? ""; return `${first}${last}`.toUpperCase(); }
function formatShortTime(value: string) { const parsed = Date.parse(value); if (!Number.isFinite(parsed)) return "recently"; return new Intl.DateTimeFormat(undefined, { hour: "numeric", minute: "2-digit" }).format(new Date(parsed)); }
function formatDate(value: string) { const parsed = Date.parse(value); return Number.isFinite(parsed) ? new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(new Date(parsed)) : "unavailable"; }
