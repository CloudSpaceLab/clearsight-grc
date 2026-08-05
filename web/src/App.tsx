import { useEffect, useMemo, useState } from "react";
import { loadCaptureRequest, loadEvidenceRequests, loadEvidenceSources, loadIntegrity, loadMatters, loadOnboardingGuide, loadOnboardingState, loadPolicies, loadPrograms, loadReadiness, loadToday, loadWorkflowTasks, resolveAuthority, saveOnboardingState, submitCaptureRequest } from "./api";
import { EmptyState } from "./components/EmptyState";
import { EvidenceWorkspace } from "./components/EvidenceWorkspace";
import { IntroGuide } from "./components/IntroGuide";
import { MattersWorkspace } from "./components/MattersWorkspace";
import { PremiumIllustration } from "./components/PremiumIllustration";
import { ProgramsWorkspace } from "./components/ProgramsWorkspace";
import { ReadinessPanel } from "./components/ReadinessPanel";
import { WorkItemIcon } from "./components/WorkItemIcon";
import type { AttentionItem, AuthorityResolution, CaptureRequest, EvidenceRequest, EvidenceSource, IntegrityFinding, MatterAggregate, OnboardingGuide, OnboardingState, PolicySummary, ProgramAggregate, Readiness, WorkflowTask } from "./types";

type View = "today" | "programs" | "work" | "configure";
type LoadState = "loading" | "live" | "unavailable";
type ConnectionState = "loading" | "live" | "sample" | "unavailable";
const sampleMode = import.meta.env.VITE_ENABLE_SAMPLE_DATA === "true";
const fallbackItems: AttentionItem[] = [
  { id: "fallback-1", type: "REGULATORY_CHANGE", title: "Review proposed CBN digital-channel requirements", why_now: "Seven provisions may affect mobile banking and two payment vendors.", scope: "Digital Channels · Demonstration Bank Nigeria", state: "Applicability review", evidence: "Official source recorded", owner: "Regulatory Compliance", due_at: new Date(Date.now() + 3 * 86400000).toISOString(), primary_action: "Review the proposed requirements" },
  { id: "fallback-2", type: "MATTER", title: "Confirm four privileged-account owners", why_now: "1,246 accounts are resolved; four still need current business-need evidence.", scope: "Treasury Operations · July 2026", state: "Evidence requested", evidence: "1,246 of 1,250 resolved", owner: "Treasury Technology", due_at: new Date(Date.now() + 36 * 3600000).toISOString(), primary_action: "Confirm the account owners" },
];
const fallbackGuide: OnboardingGuide = { code: "control-assurance-first-run", role: "Control Assurance Lead", version: 1, title: "Review workspace setup", description: "A short guide to assigned work, approval routes, evidence requests and program status.", illustration: "guided-orbit", steps: [
  { id: "today", title: "Review assigned work", description: "Today lists open reviews, approvals and evidence requests assigned to your role.", action: "Open Today" },
  { id: "programs", title: "Check ongoing programs", description: "Programs show current requirements, safeguards, required evidence and open issues.", action: "View programs" },
  { id: "routing", title: "Check the approval route", description: "Each approval shows the active policy version, scope and selected authorizer.", action: "View route" },
  { id: "capture", title: "Provide missing evidence", description: "Existing information is shown first; requests contain only unresolved fields.", action: "Open request" },
] };

function App() {
  const [activeView, setActiveView] = useState<View>("today");
  const [workTab, setWorkTab] = useState<"matters" | "evidence">("matters");
  const [items, setItems] = useState<AttentionItem[]>([]);
  const [connection, setConnection] = useState<ConnectionState>("loading");
  const [readinessState, setReadinessState] = useState<LoadState>("loading");
  const [resolution, setResolution] = useState<AuthorityResolution | null>(null);
  const [capture, setCapture] = useState<CaptureRequest | null>(null);
  const [readiness, setReadiness] = useState<Readiness | null>(null);
  const [policies, setPolicies] = useState<PolicySummary[]>([]);
  const [integrity, setIntegrity] = useState<IntegrityFinding[]>([]);
  const [tasks, setTasks] = useState<WorkflowTask[]>([]);
  const [sources, setSources] = useState<EvidenceSource[]>([]);
  const [evidenceRequests, setEvidenceRequests] = useState<EvidenceRequest[]>([]);
  const [evidenceState, setEvidenceState] = useState<LoadState>("loading");
  const [programs, setPrograms] = useState<ProgramAggregate[]>([]);
  const [programState, setProgramState] = useState<LoadState>("loading");
  const [matters, setMatters] = useState<MatterAggregate[]>([]);
  const [matterState, setMatterState] = useState<LoadState>("loading");
  const [guide, setGuide] = useState<OnboardingGuide | null>(null);
  const [guideState, setGuideState] = useState<OnboardingState | null>(null);
  const [activePanel, setActivePanel] = useState<"none" | "routing" | "capture">("none");

  useEffect(() => {
    Promise.allSettled([loadToday(), loadReadiness(), loadPolicies(), loadIntegrity(), loadWorkflowTasks(), loadEvidenceSources(), loadEvidenceRequests(), loadPrograms(), loadMatters(), loadOnboardingGuide(), loadOnboardingState()]).then((results) => {
      const [todayResult, readinessResult, policiesResult, integrityResult, tasksResult, sourcesResult, requestsResult, programsResult, mattersResult, guideResult, stateResult] = results;
      if (todayResult.status === "fulfilled") {
      setItems(todayResult.value);
      setConnection("live");
    } else if (sampleMode) {
      setItems(fallbackItems);
      setConnection("sample");
    } else {
      setItems([]);
      setConnection("unavailable");
    }
    if (readinessResult.status === "fulfilled") { setReadiness(readinessResult.value); setReadinessState("live"); } else setReadinessState("unavailable");
      if (policiesResult.status === "fulfilled") setPolicies(policiesResult.value);
      if (integrityResult.status === "fulfilled") setIntegrity(integrityResult.value);
      if (tasksResult.status === "fulfilled") setTasks(tasksResult.value);
      if (sourcesResult.status === "fulfilled" && requestsResult.status === "fulfilled") { setSources(sourcesResult.value); setEvidenceRequests(requestsResult.value); setEvidenceState("live"); } else setEvidenceState("unavailable");
      if (programsResult.status === "fulfilled") { setPrograms(programsResult.value); setProgramState("live"); } else setProgramState("unavailable");
      if (mattersResult.status === "fulfilled") { setMatters(mattersResult.value); setMatterState("live"); } else setMatterState("unavailable");
      setGuide(guideResult.status === "fulfilled" ? guideResult.value : fallbackGuide);
      setGuideState(stateResult.status === "fulfilled" ? stateResult.value : { tenant_id: "bank-demo", principal_id: "user-demo", guide_code: fallbackGuide.code, guide_version: 1, current_step: 0, completed: false, dismissed: sessionStorage.getItem("clearsight-guide-dismissed") === "1", version: 0 });
    });
  }, []);

  const dueSoon = useMemo(() => {
    const now = Date.now();
    return items.filter((item) => {
      const due = Date.parse(item.due_at);
      const remaining = due - now;
      return Number.isFinite(due) && remaining >= 0 && remaining <= 4 * 86400000;
    }).length;
  }, [items]);
  async function inspectRouting() { setActivePanel("routing"); if (!resolution) setResolution(await resolveAuthority().catch(() => null)); }
  async function openCapture() { setActivePanel("capture"); if (!capture) setCapture(await loadCaptureRequest().catch(() => null)); }
  async function advanceGuide(next: OnboardingState) { setGuideState(next); const saved = await saveOnboardingState(next).catch(() => null); if (saved) setGuideState(saved); }
  async function dismissGuide() { sessionStorage.setItem("clearsight-guide-dismissed", "1"); if (!guideState) return; const next = { ...guideState, dismissed: true }; setGuideState(next); const saved = await saveOnboardingState(next).catch(() => null); if (saved) setGuideState(saved); }

  return <div className="app-shell">
    <aside className="sidebar" aria-label="Primary navigation">
      <div className="brand-mark" aria-label="ClearSight">C</div>
      <nav>{["Today", "Programs", "Work", "Explore", "Configure"].map((label) => {
        const view: View | null = label === "Today" ? "today" : label === "Programs" ? "programs" : label === "Work" ? "work" : label === "Configure" ? "configure" : null;
        const active = view === activeView;
        return <button className={active ? "nav-item active" : "nav-item"} key={label} aria-current={active ? "page" : undefined} disabled={!view} aria-disabled={!view} title={!view ? `${label} is not available in this build` : undefined} onClick={() => view && setActiveView(view)}><span>{label.slice(0, 1)}</span><b>{label}</b></button>;
      })}</nav>
      <div className="avatar" aria-label="Signed in as Amaka Okafor">AO</div>
    </aside>
    <main>
      {activeView === "today" && <TodayView items={items} dueSoon={dueSoon} connection={connection} readiness={readiness} readinessState={readinessState} onRouting={inspectRouting} onCapture={openCapture}/>}
      {activeView === "programs" && <ProgramsView programs={programs} state={programState}/>} 
      {activeView === "work" && <WorkView tab={workTab} onTab={setWorkTab} matters={matters} matterState={matterState} sources={sources} requests={evidenceRequests} evidenceState={evidenceState}/>} 
      {activeView === "configure" && <ConfigureView policies={policies} findings={integrity} tasks={tasks}/>} 
    </main>
    {activePanel !== "none" && <div className="panel-backdrop" onMouseDown={() => setActivePanel("none")}><aside className="side-panel" onMouseDown={(event) => event.stopPropagation()} aria-label={activePanel === "routing" ? "Approval route" : "Evidence request"}><button className="panel-close" onClick={() => setActivePanel("none")} aria-label="Close">×</button>{activePanel === "routing" ? <RoutingPanel resolution={resolution}/> : <CapturePanel request={capture}/>}</aside></div>}
    {guide && guideState && !guideState.completed && !guideState.dismissed && <IntroGuide guide={guide} state={guideState} onAdvance={advanceGuide} onDismiss={dismissGuide}/>} 
  </div>;
}

function TodayView({ items, dueSoon, connection, readiness, readinessState, onRouting, onCapture }: { items: AttentionItem[]; dueSoon: number; connection: ConnectionState; readiness: Readiness | null; readinessState: LoadState; onRouting: () => void; onCapture: () => void }) {
  const current = readiness?.baseline_known ? readiness.dimensions.current : "—";
  const openCount = connection === "unavailable" ? "—" : items.length;
  const dueCount = connection === "unavailable" ? "—" : dueSoon;
  const connectionLabel = connection === "live" ? "Live data" : connection === "sample" ? "Sample data" : connection === "unavailable" ? "Data unavailable" : "Connecting";
  return <>
    <header className="topbar"><div><span className="eyebrow">Demonstration Bank Nigeria · Control Assurance</span><h1>Today</h1><p>Reviews, approvals and evidence requests assigned to you.</p></div><div className="topbar-actions"><span className={`connection ${connection}`}>{connectionLabel}</span><button id="authority-action" className="secondary-button" onClick={onRouting}>View approval route</button><button id="capture-action" className="primary-button" onClick={onCapture}>Open evidence request</button></div></header>
    <section className="brief-grid" id="today-brief"><div className="brief-stat"><span>Open items</span><strong>{openCount}</strong><small>Assigned to you</small></div><div className="brief-stat"><span>Due within 4 days</span><strong>{dueCount}</strong><small>Excludes overdue items</small></div><div className="brief-stat verified"><span>Items up to date</span><strong>{current}</strong><small>{readiness?.baseline_known ? "No action due" : "Population not connected"}</small></div></section>
    <ReadinessPanel readiness={readiness} state={readinessState}/>
    <section className="section-header"><div><h2>Assigned to you</h2><p>Reviews, approvals and evidence requests.</p></div></section>
    {connection === "unavailable" ? <EmptyState label="Work queue" title="Assigned work could not be loaded" description="The service is unavailable. No current work count is shown."/> : items.length ? <section className="attention-list">{items.map((item) => <AttentionCard item={item} key={item.id}/>)}</section> : <EmptyState label="Work queue" title="No assigned items" description="There are no open reviews, approvals or evidence requests assigned to you in the connected scope."/>}
    {readiness?.baseline_known && <section className="quiet-section"><div><span className="verified-dot"/> Readiness was updated {new Date(readiness.generated_at).toLocaleString()}</div><p>{readiness.dimensions.current} item{readiness.dimensions.current === 1 ? " is" : "s are"} currently recorded with no action due.</p></section>}
  </>;
}

function ProgramsView({ programs, state }: { programs: ProgramAggregate[]; state: LoadState }) {
  return <><header className="topbar"><div><span className="eyebrow">Demonstration Bank Nigeria</span><h1>Programs</h1><p>Ongoing obligations, controls, evidence checks and open issues.</p></div></header><ProgramsWorkspace programs={programs} state={state}/></>;
}

function WorkView({ tab, onTab, matters, matterState, sources, requests, evidenceState }: { tab: "matters" | "evidence"; onTab: (value: "matters" | "evidence") => void; matters: MatterAggregate[]; matterState: LoadState; sources: EvidenceSource[]; requests: EvidenceRequest[]; evidenceState: LoadState }) {
  return <><header className="topbar"><div><span className="eyebrow">Demonstration Bank Nigeria</span><h1>Work</h1><p>Issues, changes, evidence requests and the sources they rely on.</p></div></header><div className="workspace-tabs" role="tablist" aria-label="Work views"><button type="button" role="tab" aria-selected={tab === "matters"} className={tab === "matters" ? "active" : ""} onClick={() => onTab("matters")}>Issues and changes</button><button type="button" role="tab" aria-selected={tab === "evidence"} className={tab === "evidence" ? "active" : ""} onClick={() => onTab("evidence")}>Sources and evidence</button></div>{tab === "matters" ? <MattersWorkspace matters={matters} state={matterState}/> : <EvidenceWorkspace sources={sources} requests={requests} state={evidenceState}/>}</>;
}

function ConfigureView({ policies, findings, tasks }: { policies: PolicySummary[]; findings: IntegrityFinding[]; tasks: WorkflowTask[] }) {
  return <><header className="topbar"><div><span className="eyebrow">Governance configuration</span><h1>Routing and approvals</h1><p>Responsibility, approval limits, delegation and escalation rules.</p></div></header><section className="configure-hero"><div><span className="eyebrow">Routing checks</span><h2>{findings.length ? `${findings.length} configuration issue${findings.length === 1 ? "" : "s"}` : "No blocking configuration issues"}</h2><p>Checks cover missing owners, unresolved selectors, duplicate priorities, expired delegations and missing authorizers.</p></div><PremiumIllustration variant="routing"/></section><section className="config-grid"><article className="config-card"><div className="section-header"><div><h2>Active policies</h2><p>Approved versions and effective dates.</p></div></div>{policies.length ? policies.map((policy) => <div className="policy-row" key={policy.id}><div><strong>{policy.name}</strong><span>{policy.code} · v{policy.version}</span></div><mark>{humanizeStatus(policy.status)}</mark></div>) : <EmptyState label="Routing policies" title="No active policies" description="There are no active routing policies in the current scope."/>}</article><article className="config-card"><div className="section-header"><div><h2>Integrity findings</h2><p>Configuration checks.</p></div></div>{findings.length ? findings.map((finding) => <div className={`finding-row severity-${finding.severity.toLowerCase()}`} key={`${finding.type}-${finding.summary}`}><strong>{finding.summary}</strong><span>{finding.required_action}</span></div>) : <div className="calm-empty"><span>✓</span><div><strong>No blocking routing gaps</strong><p>All evaluated routes resolved to an active principal.</p></div></div>}</article><article className="config-card wide"><div className="section-header"><div><h2>Workflow ownership</h2><p>Open tasks and current assignees.</p></div></div><div className="task-table">{tasks.map((task) => <div className="task-row" key={task.id}><div><strong>{task.title}</strong><span>{task.responsibility} · {task.step_key}</span></div><span>{task.context?.scope ?? task.context?.program ?? "Bank NG"}</span><mark>{humanizeStatus(task.status)}</mark></div>)}</div></article></section></>;
}

function humanizeStatus(value: string) {
  return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}

function AttentionCard({ item }: { item: AttentionItem }) {
  const parsed = Date.parse(item.due_at);
  const due = Number.isFinite(parsed) ? new Intl.DateTimeFormat(undefined, { month: "short", day: "numeric", hour: "numeric", minute: "2-digit" }).format(new Date(parsed)) : "Due date unavailable";
  return <article className="attention-card"><div className="attention-icon"><WorkItemIcon type={item.type}/></div><div className="attention-content"><div className="card-kicker"><span>{item.state}</span><time>{due}</time></div><h3>{item.title}</h3><p>{item.why_now}</p><div className="card-meta"><span>{item.scope}</span><span>{item.evidence}</span><span>{item.owner}</span></div></div><div className="card-next"><span>Required action</span><strong>{item.primary_action}</strong></div></article>;
}
function RoutingPanel({ resolution }: { resolution: AuthorityResolution | null }) { return <div className="panel-content"><span className="eyebrow">Approval route</span><h2>Current authorizer</h2><p>The route uses the legal entity, issue type, importance, delegation and active policy version.</p>{resolution ? <div className="resolution-card"><div className="principal-avatar">CR</div><div><strong>{resolution.principal.display_name}</strong><span>{resolution.principal.role} · {resolution.principal.kind}</span></div><mark>Selected</mark></div> : <div className="skeleton">The approval route is unavailable while the API is offline.</div>}<dl className="explanation-list"><div><dt>Responsibility</dt><dd>Authorizer</dd></div><div><dt>Legal entity</dt><dd>Demonstration Bank Nigeria</dd></div><div><dt>Importance</dt><dd>Critical · Executive approval</dd></div><div><dt>Policy</dt><dd>{resolution?.policy_version ?? "Unavailable"}</dd></div></dl><div className="sequence"><span>Control owner</span><i>→</i><span>Control Assurance</span><i>→</i><b>CRO</b></div></div>; }
function CapturePanel({ request }: { request: CaptureRequest | null }) {
  const [answers, setAnswers] = useState<Record<string, string>>({}); const [receipt, setReceipt] = useState<string | null>(null); const [error, setError] = useState<string | null>(null);
  async function submit() { if (!request) return; setError(null); try { const result = await submitCaptureRequest(request.id, request.version, answers); setReceipt(`Submitted ${new Date(result.submitted_at).toLocaleString()}`); } catch (cause) { setError(cause instanceof Error ? cause.message : "Submission failed"); } }
  if (!request) return <div className="panel-content"><span className="eyebrow">Evidence request</span><h2>Request unavailable</h2><p>Connect to the API to load this request.</p></div>;
  if (receipt) return <div className="panel-content"><span className="eyebrow">Submission receipt</span><h2>Response submitted</h2><PremiumIllustration variant="empty"/><p>{receipt}</p><p>The response has been recorded. It will still be checked against the evidence requirements.</p></div>;
  return <div className="panel-content"><span className="eyebrow">Evidence request · about {request.estimated_minutes} minutes</span><h2>{request.title}</h2><p>{request.purpose}</p><div className="why-you"><strong>Why you received this</strong><span>{request.why_you}</span></div><h3>Information already available</h3><dl className="known-facts">{Object.entries(request.known_facts).map(([key, value]) => <div key={key}><dt>{key}</dt><dd>{value}</dd></div>)}</dl>{request.fields.map((field) => <label className="field" key={field.id}><span>{field.label}{field.required ? " *" : ""}</span>{field.type === "single_select" ? <select value={answers[field.id] ?? ""} onChange={(event) => setAnswers({ ...answers, [field.id]: event.target.value })}><option value="">Select one</option>{field.options?.map((option) => <option key={option}>{option}</option>)}</select> : <textarea value={answers[field.id] ?? ""} onChange={(event) => setAnswers({ ...answers, [field.id]: event.target.value })} placeholder={field.description}/>}</label>)}{error && <p className="error-text">{error}</p>}<div className="wizard-actions"><button className="primary-button" onClick={submit}>Review and submit</button></div></div>;
}
export default App;
