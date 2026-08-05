import { useEffect, useMemo, useState } from "react";
import { loadCaptureRequest, loadIntegrity, loadOnboardingGuide, loadOnboardingState, loadPolicies, loadReadiness, loadToday, loadWorkflowTasks, resolveAuthority, saveOnboardingState, submitCaptureRequest } from "./api";
import { EmptyState } from "./components/EmptyState";
import { IntroGuide } from "./components/IntroGuide";
import { PremiumIllustration } from "./components/PremiumIllustration";
import { ReadinessPanel } from "./components/ReadinessPanel";
import { WorkItemIcon } from "./components/WorkItemIcon";
import type { AttentionItem, AuthorityResolution, CaptureRequest, IntegrityFinding, OnboardingGuide, OnboardingState, PolicySummary, Readiness, WorkflowTask } from "./types";

const fallbackItems: AttentionItem[] = [
  { id: "fallback-1", type: "REGULATORY_CHANGE", title: "Review proposed CBN digital-channel obligations", why_now: "Seven source-linked provisions may affect mobile banking and two payment vendors.", scope: "Digital Channels · Bank NG", state: "Applicability review", evidence: "Official source verified", owner: "Regulatory Compliance", due_at: new Date(Date.now() + 3 * 86400000).toISOString(), primary_action: "Review obligations" },
  { id: "fallback-2", type: "MATTER", title: "Resolve four privileged-access exceptions", why_now: "IAM and HR evidence resolved 1,246 accounts; four still lack current business-need evidence.", scope: "Treasury Operations · July 2026", state: "Evidence requested", evidence: "1,246 of 1,250 resolved", owner: "Treasury Technology", due_at: new Date(Date.now() + 36 * 3600000).toISOString(), primary_action: "Confirm account owners" },
];
const fallbackGuide: OnboardingGuide = {
  code: "control-assurance-first-run",
  role: "Control Assurance Lead",
  version: 1,
  title: "Review workspace setup",
  description: "A short guide to assigned work, approval routes, evidence requests and readiness status.",
  illustration: "guided-orbit",
  steps: [
    { id: "today", title: "Review assigned work", description: "Today lists open reviews, approvals and evidence requests assigned to your role.", action: "Open Today" },
    { id: "routing", title: "Check the approval route", description: "Each approval shows the active policy version, scope and selected authorizer.", action: "View route" },
    { id: "capture", title: "Request additional evidence", description: "Existing evidence is shown first; requests contain only the unresolved fields.", action: "Open request" },
    { id: "readiness", title: "Review readiness status", description: "Readiness separates current, aging, at-risk, unknown and routing-blocked items.", action: "Finish setup" },
  ],
};

function App() {
  const [activeView, setActiveView] = useState<"today" | "configure">("today");
  const [items, setItems] = useState<AttentionItem[]>(fallbackItems);
  const [connection, setConnection] = useState<"loading" | "live" | "fallback">("loading");
  const [resolution, setResolution] = useState<AuthorityResolution | null>(null);
  const [capture, setCapture] = useState<CaptureRequest | null>(null);
  const [readiness, setReadiness] = useState<Readiness | null>(null);
  const [policies, setPolicies] = useState<PolicySummary[]>([]);
  const [integrity, setIntegrity] = useState<IntegrityFinding[]>([]);
  const [tasks, setTasks] = useState<WorkflowTask[]>([]);
  const [guide, setGuide] = useState<OnboardingGuide | null>(null);
  const [guideState, setGuideState] = useState<OnboardingState | null>(null);
  const [activePanel, setActivePanel] = useState<"none" | "routing" | "capture">("none");

  useEffect(() => {
    Promise.allSettled([loadToday(), loadReadiness(), loadPolicies(), loadIntegrity(), loadWorkflowTasks(), loadOnboardingGuide(), loadOnboardingState()]).then((results) => {
      const [todayResult, readinessResult, policiesResult, integrityResult, tasksResult, guideResult, stateResult] = results;
      if (todayResult.status === "fulfilled") { setItems(todayResult.value); setConnection("live"); } else setConnection("fallback");
      if (readinessResult.status === "fulfilled") setReadiness(readinessResult.value);
      if (policiesResult.status === "fulfilled") setPolicies(policiesResult.value);
      if (integrityResult.status === "fulfilled") setIntegrity(integrityResult.value);
      if (tasksResult.status === "fulfilled") setTasks(tasksResult.value);
      setGuide(guideResult.status === "fulfilled" ? guideResult.value : fallbackGuide);
      setGuideState(stateResult.status === "fulfilled" ? stateResult.value : { tenant_id: "bank-demo", principal_id: "user-demo", guide_code: fallbackGuide.code, guide_version: 1, current_step: 0, completed: false, dismissed: sessionStorage.getItem("clearsight-guide-dismissed") === "1", version: 0 });
    });
  }, []);

  const dueSoon = useMemo(() => items.filter((item) => Date.parse(item.due_at) - Date.now() < 4 * 86400000).length, [items]);
  async function inspectRouting() { setActivePanel("routing"); if (!resolution) setResolution(await resolveAuthority().catch(() => null)); }
  async function openCapture() { setActivePanel("capture"); if (!capture) setCapture(await loadCaptureRequest().catch(() => null)); }
  async function advanceGuide(next: OnboardingState) { setGuideState(next); const saved = await saveOnboardingState(next).catch(() => null); if (saved) setGuideState(saved); }
  async function dismissGuide() { sessionStorage.setItem("clearsight-guide-dismissed", "1"); if (!guideState) return; const next = { ...guideState, dismissed: true }; setGuideState(next); const saved = await saveOnboardingState(next).catch(() => null); if (saved) setGuideState(saved); }

  return <div className="app-shell">
    <aside className="sidebar" aria-label="Primary navigation">
      <div className="brand-mark" aria-label="ClearSight">C</div>
      <nav>{["Today", "Programs", "Work", "Explore", "Configure"].map((label) => {
        const selectable = label === "Today" || label === "Configure";
        const active = (label === "Today" && activeView === "today") || (label === "Configure" && activeView === "configure");
        return <button className={active ? "nav-item active" : "nav-item"} key={label} aria-current={active ? "page" : undefined} onClick={() => selectable && setActiveView(label === "Today" ? "today" : "configure")}><span>{label.slice(0, 1)}</span><b>{label}</b></button>;
      })}</nav>
      <button className="avatar" aria-label="Account menu">AO</button>
    </aside>
    <main>{activeView === "today" ? <TodayView items={items} dueSoon={dueSoon} connection={connection} readiness={readiness} onRouting={inspectRouting} onCapture={openCapture}/> : <ConfigureView policies={policies} findings={integrity} tasks={tasks}/>}</main>
    {activePanel !== "none" && <div className="panel-backdrop" onMouseDown={() => setActivePanel("none")}><aside className="side-panel" onMouseDown={(event) => event.stopPropagation()} aria-label={activePanel === "routing" ? "Approval route" : "Evidence request"}><button className="panel-close" onClick={() => setActivePanel("none")} aria-label="Close">×</button>{activePanel === "routing" ? <RoutingPanel resolution={resolution}/> : <CapturePanel request={capture}/>}</aside></div>}
    {guide && guideState && !guideState.completed && !guideState.dismissed && <IntroGuide guide={guide} state={guideState} onAdvance={advanceGuide} onDismiss={dismissGuide}/>} 
  </div>;
}

function TodayView({ items, dueSoon, connection, readiness, onRouting, onCapture }: { items: AttentionItem[]; dueSoon: number; connection: string; readiness: Readiness | null; onRouting: () => void; onCapture: () => void }) {
  const current = readiness?.baseline_known ? readiness.dimensions.current : "—";
  const currentNote = readiness?.baseline_known ? "No action due" : "Population not connected";
  return <>
    <header className="topbar"><div><span className="eyebrow">Demonstration Bank Nigeria · Control Assurance</span><h1>Today</h1><p>Open reviews, approvals and evidence requests assigned to you.</p></div><div className="topbar-actions"><span className={`connection ${connection}`}>{connection === "live" ? "Connected" : connection === "fallback" ? "Sample data" : "Connecting"}</span><button id="authority-action" className="secondary-button" onClick={onRouting}>View approval route</button><button id="capture-action" className="primary-button" onClick={onCapture}>Request evidence</button></div></header>
    <section className="brief-grid" id="today-brief"><div className="brief-stat"><span>Open items</span><strong>{items.length}</strong><small>Assigned to you</small></div><div className="brief-stat"><span>Due in 4 days</span><strong>{dueSoon}</strong><small>Open items</small></div><div className="brief-stat verified"><span>Current items</span><strong>{current}</strong><small>{currentNote}</small></div></section>
    <ReadinessPanel readiness={readiness}/>
    <section className="section-header"><div><h2>Assigned to you</h2><p>Reviews, approvals and evidence requests.</p></div><button className="text-button">Detailed view</button></section>
    {items.length ? <section className="attention-list">{items.map((item) => <AttentionCard item={item} key={item.id}/>)}</section> : <EmptyState label="Work queue" title="No assigned items" description="There are no open reviews, approvals or evidence requests assigned to you." action="View programs"/>}
    <section className="quiet-section"><div><span className="verified-dot"/> No material changes detected in 6 programs</div><p>Evidence and source-check results are available in each program.</p></section>
  </>;
}

function ConfigureView({ policies, findings, tasks }: { policies: PolicySummary[]; findings: IntegrityFinding[]; tasks: WorkflowTask[] }) {
  return <>
    <header className="topbar"><div><span className="eyebrow">Governance configuration</span><h1>Routing and approvals</h1><p>Configure responsibility, approval limits, delegation and escalation rules.</p></div><button className="primary-button">Create policy draft</button></header>
    <section className="configure-hero"><div><span className="eyebrow">Routing checks</span><h2>{findings.length ? `${findings.length} configuration issue${findings.length === 1 ? "" : "s"}` : "No blocking configuration issues"}</h2><p>Checks cover missing owners, unresolved selectors, duplicate priorities, expired delegations and missing authorizers.</p></div><PremiumIllustration variant="routing"/></section>
    <section className="config-grid">
      <article className="config-card"><div className="section-header"><div><h2>Active policies</h2><p>Approved versions and effective dates.</p></div><button className="text-button">Run simulation</button></div>{policies.length ? policies.map((policy) => <div className="policy-row" key={policy.id}><div><strong>{policy.name}</strong><span>{policy.code} · v{policy.version}</span></div><mark>{policy.status}</mark></div>) : <EmptyState label="Routing policies" title="No active policies" description="Create a draft, run representative simulations and submit it for independent approval." action="Create policy draft"/>}</article>
      <article className="config-card"><div className="section-header"><div><h2>Integrity findings</h2><p>Configuration checks.</p></div></div>{findings.length ? findings.map((finding) => <div className={`finding-row severity-${finding.severity.toLowerCase()}`} key={`${finding.type}-${finding.summary}`}><strong>{finding.summary}</strong><span>{finding.required_action}</span></div>) : <div className="calm-empty"><span>✓</span><div><strong>No blocking routing gaps</strong><p>All evaluated routes resolved to an active principal.</p></div></div>}</article>
      <article className="config-card wide"><div className="section-header"><div><h2>Workflow ownership</h2><p>Open tasks and current assignees.</p></div></div><div className="task-table">{tasks.map((task) => <div className="task-row" key={task.id}><div><strong>{task.title}</strong><span>{task.responsibility} · {task.step_key}</span></div><span>{task.context?.scope ?? task.context?.program ?? "Bank NG"}</span><mark>{task.status.replaceAll("_", " ")}</mark></div>)}</div></article>
    </section>
  </>;
}

function AttentionCard({ item }: { item: AttentionItem }) {
  const due = new Intl.DateTimeFormat(undefined, { month: "short", day: "numeric", hour: "numeric", minute: "2-digit" }).format(new Date(item.due_at));
  return <article className="attention-card"><div className="attention-icon"><WorkItemIcon type={item.type}/></div><div className="attention-content"><div className="card-kicker"><span>{item.state}</span><time>{due}</time></div><h3>{item.title}</h3><p>{item.why_now}</p><div className="card-meta"><span>{item.scope}</span><span>{item.evidence}</span><span>{item.owner}</span></div></div><button className="card-action">{item.primary_action}<span>→</span></button></article>;
}

function RoutingPanel({ resolution }: { resolution: AuthorityResolution | null }) {
  return <div className="panel-content"><span className="eyebrow">Approval route</span><h2>Current authorizer</h2><p>The route is calculated from legal entity, matter type, materiality, delegation and the active policy version.</p>{resolution ? <div className="resolution-card"><div className="principal-avatar">CR</div><div><strong>{resolution.principal.display_name}</strong><span>{resolution.principal.role} · {resolution.principal.kind}</span></div><mark>Selected</mark></div> : <div className="skeleton">The approval route is unavailable while the API is offline.</div>}<dl className="explanation-list"><div><dt>Responsibility</dt><dd>Authorizer</dd></div><div><dt>Legal entity</dt><dd>Demonstration Bank Nigeria</dd></div><div><dt>Materiality</dt><dd>5 · Executive approval</dd></div><div><dt>Policy</dt><dd>{resolution?.policy_version ?? "Sample policy"}</dd></div></dl><div className="sequence"><span>Control owner</span><i>→</i><span>Control Assurance</span><i>→</i><b>CRO</b></div><button className="primary-button full">View policy</button></div>;
}

function CapturePanel({ request }: { request: CaptureRequest | null }) {
  const [answers, setAnswers] = useState<Record<string, string>>({});
  const [receipt, setReceipt] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  async function submit() {
    if (!request) return;
    setError(null);
    try {
      const result = await submitCaptureRequest(request.id, request.version, answers);
      setReceipt(`Submitted ${new Date(result.submitted_at).toLocaleString()}`);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Submission failed");
    }
  }
  if (!request) return <div className="panel-content"><span className="eyebrow">Evidence request</span><h2>Request unavailable</h2><p>Connect to the API to load this request.</p></div>;
  if (receipt) return <div className="panel-content"><span className="eyebrow">Submission receipt</span><h2>Response submitted</h2><PremiumIllustration variant="empty"/><p>{receipt}</p><p>The submission has been recorded. Evidence sufficiency is assessed separately.</p></div>;
  return <div className="panel-content"><span className="eyebrow">Evidence request · about {request.estimated_minutes} minutes</span><h2>{request.title}</h2><p>{request.purpose}</p><div className="why-you"><strong>Why you received this</strong><span>{request.why_you}</span></div><h3>Information already available</h3><dl className="known-facts">{Object.entries(request.known_facts).map(([key, value]) => <div key={key}><dt>{key}</dt><dd>{value}</dd></div>)}</dl>{request.fields.map((field) => <label className="field" key={field.id}><span>{field.label}{field.required ? " *" : ""}</span>{field.type === "single_select" ? <select value={answers[field.id] ?? ""} onChange={(event) => setAnswers({ ...answers, [field.id]: event.target.value })}><option value="">Select one</option>{field.options?.map((option) => <option key={option}>{option}</option>)}</select> : <textarea value={answers[field.id] ?? ""} onChange={(event) => setAnswers({ ...answers, [field.id]: event.target.value })} placeholder={field.description}/>}</label>)}{error && <p className="error-text">{error}</p>}<div className="wizard-actions"><button className="secondary-button">Wrong recipient</button><button className="primary-button" onClick={submit}>Review and submit</button></div></div>;
}

export default App;
