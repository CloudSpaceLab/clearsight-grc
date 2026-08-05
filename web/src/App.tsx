import { useEffect, useMemo, useState } from "react";
import { loadCaptureRequest, loadIntegrity, loadOnboardingGuide, loadOnboardingState, loadPolicies, loadReadiness, loadToday, loadWorkflowTasks, resolveAuthority, saveOnboardingState, submitCaptureRequest } from "./api";
import { EmptyState } from "./components/EmptyState";
import { IntroGuide } from "./components/IntroGuide";
import { PremiumIllustration } from "./components/PremiumIllustration";
import { ReadinessPanel } from "./components/ReadinessPanel";
import type { AttentionItem, AuthorityResolution, CaptureRequest, IntegrityFinding, OnboardingGuide, OnboardingState, PolicySummary, Readiness, WorkflowTask } from "./types";

const fallbackItems: AttentionItem[] = [
  { id: "fallback-1", type: "REGULATORY_CHANGE", title: "Review proposed CBN digital-channel obligations", why_now: "Seven source-linked provisions may affect mobile banking and two payment vendors.", scope: "Digital Channels · Bank NG", state: "Applicability review", evidence: "Official source verified", owner: "Regulatory Compliance", due_at: new Date(Date.now() + 3 * 86400000).toISOString(), primary_action: "Review seven proposed obligations" },
  { id: "fallback-2", type: "MATTER", title: "Resolve four privileged-access exceptions", why_now: "IAM and HR evidence resolved 1,246 accounts; four still lack current business-need evidence.", scope: "Treasury Operations · July 2026", state: "Waiting for focused response", evidence: "99.7% population resolved", owner: "Treasury Technology", due_at: new Date(Date.now() + 36 * 3600000).toISOString(), primary_action: "Confirm four account owners" }
];
const fallbackGuide: OnboardingGuide = { code: "control-assurance-first-run", role: "Control Assurance Lead", version: 1, title: "Your continuous assurance workspace", description: "Learn the three actions that keep Programs current without rebuilding registers.", illustration: "guided-orbit", steps: [
  { id: "today", title: "Start with what changed", description: "Today contains only material work requiring your judgment.", action: "Continue" },
  { id: "routing", title: "Inspect who is responsible", description: "Every review and approval shows the policy and authority that selected the actor.", action: "Continue" },
  { id: "capture", title: "Request only missing proof", description: "Focused capture uses existing evidence first and asks only unresolved facts.", action: "Continue" },
  { id: "readiness", title: "Watch drift, not dashboards", description: "Continuous readiness separates current, aging, blocked and human-judgment states.", action: "Enter workspace" }
] };

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
      })}</nav><button className="avatar" aria-label="Account menu">AO</button>
    </aside>
    <main>{activeView === "today" ? <TodayView items={items} dueSoon={dueSoon} connection={connection} readiness={readiness} onRouting={inspectRouting} onCapture={openCapture}/> : <ConfigureView policies={policies} findings={integrity} tasks={tasks}/>}</main>
    {activePanel !== "none" && <div className="panel-backdrop" onMouseDown={() => setActivePanel("none")}><aside className="side-panel" onMouseDown={(event) => event.stopPropagation()} aria-label={activePanel === "routing" ? "Authority explanation" : "Capture wizard"}><button className="panel-close" onClick={() => setActivePanel("none")} aria-label="Close">×</button>{activePanel === "routing" ? <RoutingPanel resolution={resolution}/> : <CapturePanel request={capture}/>}</aside></div>}
    {guide && guideState && !guideState.completed && !guideState.dismissed && <IntroGuide guide={guide} state={guideState} onAdvance={advanceGuide} onDismiss={dismissGuide}/>} 
  </div>;
}

function TodayView({ items, dueSoon, connection, readiness, onRouting, onCapture }: { items: AttentionItem[]; dueSoon: number; connection: string; readiness: Readiness | null; onRouting: () => void; onCapture: () => void }) {
  return <><header className="topbar"><div><span className="eyebrow">Demonstration Bank Nigeria · Control Assurance</span><h1>Today</h1><p>Only the work that needs your judgment or action.</p></div><div className="topbar-actions"><span className={`connection ${connection}`}>{connection === "live" ? "API live" : connection === "fallback" ? "Preview data" : "Connecting"}</span><button id="authority-action" className="secondary-button" onClick={onRouting}>Inspect authority</button><button id="capture-action" className="primary-button" onClick={onCapture}>Open capture wizard</button></div></header>
    <section className="brief-grid" id="today-brief"><div className="brief-stat"><span>Needs attention</span><strong>{items.length}</strong><small>Material items only</small></div><div className="brief-stat"><span>Due soon</span><strong>{dueSoon}</strong><small>Within four days</small></div><div className="brief-stat verified"><span>Automatically maintained</span><strong>{readiness?.dimensions.current ?? 18}</strong><small>No intervention required</small></div></section>
    <ReadinessPanel readiness={readiness}/>
    <section className="section-header"><div><h2>Your attention</h2><p>Grouped by the outcome required from you.</p></div><button className="text-button">Analyst view</button></section>
    {items.length ? <section className="attention-list">{items.map((item) => <AttentionCard item={item} key={item.id}/>)}</section> : <EmptyState title="Everything material is currently handled" description="ClearSight is still monitoring evidence, sources, obligations and routing integrity." action="Review maintained Programs"/>}
    <section className="quiet-section"><div><span className="verified-dot"/> No material change in 6 Programs</div><p>Evidence and source checks completed recently.</p></section></>;
}

function ConfigureView({ policies, findings, tasks }: { policies: PolicySummary[]; findings: IntegrityFinding[]; tasks: WorkflowTask[] }) {
  return <><header className="topbar"><div><span className="eyebrow">Governance configuration</span><h1>Route work safely</h1><p>Define responsibility and authority once, simulate it, then monitor integrity continuously.</p></div><button className="primary-button">Create policy draft</button></header>
    <section className="configure-hero"><div><span className="eyebrow">Routing integrity</span><h2>{findings.length ? `${findings.length} issue${findings.length === 1 ? "" : "s"} need attention` : "All critical routes are covered"}</h2><p>ClearSight checks missing owners, unresolved selectors, duplicate priorities, expired delegations and absent authorizers.</p></div><PremiumIllustration variant="routing"/></section>
    <section className="config-grid"><article className="config-card"><div className="section-header"><div><h2>Active policies</h2><p>Versioned and effective-dated.</p></div><button className="text-button">Simulate</button></div>{policies.length ? policies.map((policy) => <div className="policy-row" key={policy.id}><div><strong>{policy.name}</strong><span>{policy.code} · v{policy.version}</span></div><mark>{policy.status}</mark></div>) : <EmptyState title="No routing policies are active" description="Create a draft, test representative scenarios and publish through maker-checker approval." action="Create policy draft"/>}</article>
      <article className="config-card"><div className="section-header"><div><h2>Integrity findings</h2><p>Autonomous configuration checks.</p></div></div>{findings.length ? findings.map((finding) => <div className={`finding-row severity-${finding.severity.toLowerCase()}`} key={`${finding.type}-${finding.summary}`}><strong>{finding.summary}</strong><span>{finding.required_action}</span></div>) : <div className="calm-empty"><span>✓</span><div><strong>No blocking routing gaps</strong><p>Policy resolution is currently healthy.</p></div></div>}</article>
      <article className="config-card wide"><div className="section-header"><div><h2>Current workflow ownership</h2><p>Active work remains explainable and re-routable.</p></div></div><div className="task-table">{tasks.map((task) => <div className="task-row" key={task.id}><div><strong>{task.title}</strong><span>{task.responsibility} · {task.step_key}</span></div><span>{task.context?.scope ?? task.context?.program ?? "Bank NG"}</span><mark>{task.status.replaceAll("_", " ")}</mark></div>)}</div></article></section></>;
}

function AttentionCard({ item }: { item: AttentionItem }) { const due = new Intl.DateTimeFormat(undefined, { month: "short", day: "numeric", hour: "numeric", minute: "2-digit" }).format(new Date(item.due_at)); return <article className="attention-card"><div className="attention-icon">{item.type === "REGULATORY_CHANGE" ? "§" : "!"}</div><div className="attention-content"><div className="card-kicker"><span>{item.state}</span><time>{due}</time></div><h3>{item.title}</h3><p>{item.why_now}</p><div className="card-meta"><span>{item.scope}</span><span>{item.evidence}</span><span>{item.owner}</span></div></div><button className="card-action">{item.primary_action}<span>→</span></button></article>; }
function RoutingPanel({ resolution }: { resolution: AuthorityResolution | null }) { return <div className="panel-content"><span className="eyebrow">Policy simulation</span><h2>Who authorizes this material Matter?</h2><p>ClearSight resolves responsibility from scope, policy, materiality and current authority—not a hard-coded assignee.</p>{resolution ? <div className="resolution-card"><div className="principal-avatar">CR</div><div><strong>{resolution.principal.display_name}</strong><span>{resolution.principal.role} · {resolution.principal.kind}</span></div><mark>Eligible</mark></div> : <div className="skeleton">API unavailable — start the Go service to resolve the live route.</div>}<dl className="explanation-list"><div><dt>Responsibility</dt><dd>Authorizer</dd></div><div><dt>Legal entity</dt><dd>Demonstration Bank Nigeria</dd></div><div><dt>Materiality</dt><dd>5 · Executive authority</dd></div><div><dt>Policy</dt><dd>{resolution?.policy_version ?? "demo-2026-08-05"}</dd></div></dl><div className="sequence"><span>Control owner</span><i>→</i><span>Control Assurance</span><i>→</i><b>CRO</b></div><button className="primary-button full">Open routing policy</button></div>; }
function CapturePanel({ request }: { request: CaptureRequest | null }) { const [answers, setAnswers] = useState<Record<string, string>>({}); const [receipt, setReceipt] = useState<string | null>(null); const [error, setError] = useState<string | null>(null); async function submit(){if(!request)return;setError(null);try{const result=await submitCaptureRequest(request.id,request.version,answers);setReceipt(`Submitted ${new Date(result.submitted_at).toLocaleString()}`)}catch(cause){setError(cause instanceof Error?cause.message:"Submission failed")}} if(!request)return <div className="panel-content"><span className="eyebrow">Focused capture</span><h2>Request unavailable</h2><p>Start the API to load the live request-scoped wizard.</p></div>; if(receipt)return <div className="panel-content"><span className="eyebrow">Receipt</span><h2>Response recorded</h2><PremiumIllustration variant="empty"/><p>{receipt}</p><p>Your response is now an Observation. Evidence sufficiency will be evaluated separately.</p></div>; return <div className="panel-content"><span className="eyebrow">Focused capture · {request.estimated_minutes} minutes</span><h2>{request.title}</h2><p>{request.purpose}</p><div className="why-you"><strong>Why you</strong><span>{request.why_you}</span></div><h3>Already known</h3><dl className="known-facts">{Object.entries(request.known_facts).map(([key,value])=><div key={key}><dt>{key}</dt><dd>{value}</dd></div>)}</dl>{request.fields.map((field)=><label className="field" key={field.id}><span>{field.label}{field.required?" *":""}</span>{field.type==="single_select"?<select value={answers[field.id]??""} onChange={(event)=>setAnswers({...answers,[field.id]:event.target.value})}><option value="">Select one</option>{field.options?.map((option)=><option key={option}>{option}</option>)}</select>:<textarea value={answers[field.id]??""} onChange={(event)=>setAnswers({...answers,[field.id]:event.target.value})} placeholder={field.description}/>}</label>)}{error&&<p className="error-text">{error}</p>}<div className="wizard-actions"><button className="secondary-button">Wrong recipient</button><button className="primary-button" onClick={submit}>Review and submit</button></div></div>; }
export default App;
