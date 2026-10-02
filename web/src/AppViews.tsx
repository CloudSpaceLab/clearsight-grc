import { EmptyState } from "./components/EmptyState";
import { AssignedWorkQueue, type AssignedWorkState } from "./components/AssignedWorkQueue";
import { BankJourneysWorkspace } from "./components/BankJourneysWorkspace";
import { EvidenceWorkspace } from "./components/EvidenceWorkspace";
import { MattersWorkspace } from "./components/MattersWorkspace";
import { ProgramsWorkspace } from "./components/ProgramsWorkspace";
import { TodayInterventions } from "./components/TodayInterventions";
import { Tabs } from "./components/ui";
import { WorkspaceErrorBoundary } from "./components/WorkspaceErrorBoundary";
import type { AttentionItem, AuthorityResolution, EvidenceRequest, EvidenceSource, Readiness } from "./types";
import { initials } from "./components/Monogram";
import type { ProgramItemTarget, ProgramSection, WorkTab } from "./appRouting";

export { CapturePanel } from "./components/CapturePanel";

type LoadState = "loading" | "live" | "unavailable";
type ConnectionState = "loading" | "live" | "unavailable";
export type RoutingLoadState = "loading" | "live" | "unavailable" | "forbidden" | "not-found";

export function TodayView({ organizationName, items, connection, generatedAt, readiness, readinessState, onCapture, onOpenItem, onInspectAuthority }: { organizationName: string; items: AttentionItem[]; connection: ConnectionState; generatedAt?: string; readiness: Readiness | null; readinessState: LoadState; onCapture?: () => void; onOpenItem: (item: AttentionItem) => void; onInspectAuthority: (item: AttentionItem) => void }) {
  const connectionLabel = connection === "live" ? generatedAt ? `Updated ${formatShortTime(generatedAt)}` : "Connected data" : connection === "unavailable" ? "Data unavailable" : "Connecting";
  return <>
    <header className="topbar today-topbar">
      <div><span className="eyebrow">{organizationName}</span><h1>Oversight</h1><p>Assigned decisions, evidence, outcome checks and operational exceptions that need your action.</p></div>
      <div className="topbar-actions"><span className={`connection ${connection}`}>{connectionLabel}</span>{onCapture && <><div className="today-desktop-actions"><button className="secondary-button" onClick={onCapture}>Respond to evidence request</button></div><details className="today-mobile-actions"><summary>More actions</summary><div><button className="secondary-button" onClick={onCapture}>Respond to evidence request</button></div></details></>}</div>
    </header>
    <TodayInterventions items={items} connection={connection} readiness={readiness} readinessState={readinessState} onOpenItem={onOpenItem} onInspectAuthority={onInspectAuthority}/>
  </>;
}

export function ProgramsView({ organizationName, actorPrincipalID, canConfigureSources, targetID, targetSection, programItem, onSectionChange, openFirst, onOpenRequest, onOpenForm, onAnalyzeDocument }: { organizationName: string; actorPrincipalID?: string; canConfigureSources?: boolean; targetID?: string; targetSection?: ProgramSection; programItem?: ProgramItemTarget; onSectionChange?: (programID: string, section: ProgramSection) => void; openFirst?: boolean; onOpenRequest?: (requestID: string) => void; onOpenForm?: (formID: string) => void; onAnalyzeDocument?: () => void }) {
  return <><header className="topbar"><div><span className="eyebrow">{organizationName}</span><h1>Programs</h1><p>Ongoing obligations, safeguards, evidence checks and open issues.</p></div>{onAnalyzeDocument && <div className="topbar-actions"><button className="secondary-button" type="button" aria-label="Analyze document to create or update Programs" onClick={onAnalyzeDocument}>Analyze document</button></div>}</header><ProgramsWorkspace targetID={targetID} targetSection={targetSection} programItem={programItem} onSectionChange={onSectionChange} openFirst={openFirst} actorPrincipalID={actorPrincipalID} canConfigureSources={canConfigureSources} onOpenRequest={onOpenRequest} onOpenForm={onOpenForm}/></>;
}

export function ReferenceJourneysView({ organizationName }: { organizationName: string }) {
  return <><header className="topbar"><div><span className="eyebrow">Demo environment · {organizationName}</span><h1>Reference journeys</h1><p>Walk through sample, non-production compliance scenarios using the same ClearSight operating records.</p></div></header><BankJourneysWorkspace/></>;
}

export function WorkView({ organizationName, actorPrincipalID, evidenceScopeToken, tab, onTab, onBackMatter, assignedItems, assignedState, onOpenAssignedItem, onInspectAuthority, sources, requests, evidenceSourceState, evidenceRequestState, onEvidenceRetry, onEvidenceRequestUpdated, matterTargetID, openFirstMatter, evidenceTargetID, openFirstEvidence, onOpenEvidence, onAnalyzeDocument }: { organizationName: string; actorPrincipalID?: string; evidenceScopeToken: number; tab: WorkTab; onTab: (value: WorkTab) => void; onBackMatter: () => void; assignedItems: AttentionItem[]; assignedState: AssignedWorkState; onOpenAssignedItem: (item: AttentionItem) => void; onInspectAuthority?: (item: AttentionItem) => void; sources: EvidenceSource[]; requests: EvidenceRequest[]; evidenceSourceState: LoadState; evidenceRequestState: LoadState; onEvidenceRetry: () => void; onEvidenceRequestUpdated: (request: EvidenceRequest, scopeToken: number) => boolean; matterTargetID?: string; openFirstMatter?: boolean; evidenceTargetID?: string; openFirstEvidence?: boolean; onOpenEvidence: (id: string) => void; onAnalyzeDocument?: () => void }) {
  return <>
    <header className="topbar"><div><span className="eyebrow">{organizationName}</span><h1>Work</h1><p>Act on assigned decisions, exceptions and evidence, or inspect the underlying work records.</p></div>{tab === "matters" && onAnalyzeDocument && <div className="topbar-actions"><button className="secondary-button" type="button" aria-label="Analyze document to create an issue or change" onClick={onAnalyzeDocument}>Analyze document</button></div>}</header>
    <Tabs ariaLabel="Work views" compactLabel="Work view" items={workViews} selectedKey={tab} onSelectionChange={onTab}>{(selected) => <>
      {selected === "assigned" && <AssignedWorkQueue items={assignedItems} state={assignedState} onOpenItem={onOpenAssignedItem} onInspectAuthority={onInspectAuthority}/>}
      {selected === "matters" && <WorkspaceErrorBoundary label="Issues and changes"><MattersWorkspace targetID={matterTargetID} openFirst={openFirstMatter} onBack={onBackMatter} onOpenRequest={onOpenEvidence}/></WorkspaceErrorBoundary>}
      {selected === "evidence" && <><EvidenceWorkspace sources={sources} requests={requests} sourceState={evidenceSourceState} requestState={evidenceRequestState} actorPrincipalID={actorPrincipalID} evidenceScopeToken={evidenceScopeToken} targetID={evidenceTargetID} openFirst={openFirstEvidence} onOpenRequest={onOpenEvidence} onRequestUpdated={onEvidenceRequestUpdated}/>{(evidenceSourceState === "unavailable" || evidenceRequestState === "unavailable") && <div className="workspace-recovery-actions"><button className="secondary-button" type="button" onClick={onEvidenceRetry}>Retry unavailable evidence data</button></div>}</>}
    </>}</Tabs>
  </>;
}

const workViews = [
  { id: "assigned", label: "Assigned" },
  { id: "matters", label: "Issues & changes" },
  { id: "evidence", label: "Evidence" },
] as const;

export function RoutingPanel({ resolution, item, legalEntityName, state }: { resolution: AuthorityResolution | null; item: AttentionItem | null; legalEntityName: string; state: RoutingLoadState }) {
  if (state === "loading") return <div className="panel-content"><span className="eyebrow">Authority</span><h2>Checking approval authority</h2><p aria-live="polite" aria-busy="true">Checking responsibility and the current approval policy…</p></div>;
  if (state === "forbidden") return <div className="panel-content"><EmptyState kind="forbidden" label="Authority" title="Authority details are restricted" description="Your role cannot view approvers or policy details for this item."/></div>;
  if (state === "not-found") return <div className="panel-content"><EmptyState kind="not-found" label="Authority" title="Authority could not be resolved for this item" description="The linked record or routing context is no longer available."/></div>;
  if (state === "unavailable" || !resolution || !item) return <div className="panel-content"><EmptyState kind="unavailable" label="Authority" title="Authority resolution is unavailable" description="Try again before approving or assigning this item."/></div>;
  const candidates = resolution.candidate_principals ?? [];
  const hasResolvedPrincipal = Boolean(resolution.principal?.id);
  return <div className="panel-content"><span className="eyebrow">Authority · {item.authority?.responsibility ?? "Current responsibility"}</span><h2>Authority for this item</h2><p><strong>{item.title}</strong></p><h3>Approval details</h3>{hasResolvedPrincipal && <div className="resolution-card"><div className="principal-avatar">{initials(resolution.principal.display_name)}</div><div><strong>{resolution.principal.display_name}</strong><span>{resolution.principal.role} · {resolution.principal.kind}</span></div><mark>Resolved</mark></div>}{candidates.length > 1 && <section className="authority-candidates"><h3>Eligible approvers</h3><p>{resolution.strategy ? `Selection method: ${humanizeStatus(resolution.strategy)}` : "More than one person is eligible under the current policy."}</p><ul>{candidates.map((candidate) => <li key={candidate.id}><strong>{candidate.display_name}</strong><span>{candidate.role} · {candidate.kind}</span></li>)}</ul></section>}<dl className="explanation-list"><div><dt>Legal entity</dt><dd>{legalEntityName}</dd></div><div><dt>Responsibility</dt><dd>{item.authority?.responsibility ?? "Unavailable"}</dd></div>{item.authority?.decision_type && <div><dt>Decision type</dt><dd>{humanizeStatus(item.authority.decision_type)}</dd></div>}<div><dt>Materiality</dt><dd>{item.authority ? String(item.authority.materiality) : "Unavailable"}</dd></div><div><dt>Policy version</dt><dd>{resolution.policy_version || "Unavailable"}</dd></div><div><dt>Why this route</dt><dd>{resolution.explanation || "No additional explanation was provided."}</dd></div></dl></div>;
}

function humanizeStatus(value: string) { return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase()); }
function formatShortTime(value: string) { const parsed = Date.parse(value); if (!Number.isFinite(parsed)) return "recently"; return new Intl.DateTimeFormat(undefined, { hour: "numeric", minute: "2-digit" }).format(new Date(parsed)); }
