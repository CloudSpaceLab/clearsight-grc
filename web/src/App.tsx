import { useEffect, useMemo, useState } from "react";
import { loadCaptureRequest, loadContext, loadEvidenceRequests, loadEvidenceSources, loadIntegrity, loadOnboardingGuide, loadOnboardingState, loadPolicies, loadProjectionHealth, loadReadiness, loadToday, loadWorkflowTasks, reconcileProgramState, resolveAuthority, saveOnboardingState } from "./api";
import type { RuntimeContext } from "./api";
import { IntroGuide } from "./components/IntroGuide";
import { CapturePanel, ConfigureView, ExploreView, ProgramsView, RoutingPanel, TodayView, WorkView } from "./AppViews";
import type { AttentionItem, AuthorityResolution, CaptureRequest, EvidenceRequest, EvidenceSource, IntegrityFinding, OnboardingGuide, OnboardingState, PolicySummary, Readiness, WorkflowTask } from "./types";
import type { ProjectionHealth, ReconcileResult } from "./operationsTypes";

type View = "today" | "programs" | "work" | "explore" | "configure";
type LoadState = "idle" | "loading" | "live" | "unavailable";
type ConnectionState = "loading" | "live" | "sample" | "unavailable";
const sampleMode = import.meta.env.VITE_ENABLE_SAMPLE_DATA === "true";
const fallbackItems: AttentionItem[] = [
  { id: "fallback-1", type: "REGULATORY_CHANGE", title: "Review proposed digital-channel requirements", why_now: "Seven provisions may affect mobile banking and two payment vendors.", scope: "Digital Channels · Reference data", state: "Applicability review", evidence: "Official source recorded", owner: "Regulatory Compliance", due_at: new Date(Date.now() + 3 * 86400000).toISOString(), primary_action: "Review the proposed requirements" },
  { id: "fallback-2", type: "MATTER", title: "Confirm four privileged-account owners", why_now: "1,246 accounts are resolved; four still need current business-need evidence.", scope: "Treasury Operations · Reference data", state: "Evidence requested", evidence: "1,246 of 1,250 resolved", owner: "Treasury Technology", due_at: new Date(Date.now() + 36 * 3600000).toISOString(), primary_action: "Confirm the account owners" },
];
const fallbackGuide: OnboardingGuide = { code: "control-assurance-first-run", role: "Control Assurance Lead", version: 1, title: "Review workspace setup", description: "A short guide to assigned work, approval routes, evidence requests and program status.", illustration: "guided-orbit", steps: [
  { id: "today", title: "Review assigned work", description: "Today lists open reviews, approvals and evidence requests assigned to your role.", action: "Open Today" },
  { id: "programs", title: "Check ongoing programs", description: "Programs show current requirements, safeguards, required evidence and open issues.", action: "View programs" },
  { id: "routing", title: "Check the approval route", description: "Each approval shows the active policy version, scope and selected authorizer.", action: "View route" },
  { id: "capture", title: "Provide missing evidence", description: "Existing information is shown first; requests contain only unresolved fields.", action: "Open request" },
] };

function App() {
  const [runtime, setRuntime] = useState<RuntimeContext | null>(null);
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
  const [configureState, setConfigureState] = useState<LoadState>("idle");
  const [projectionHealth, setProjectionHealth] = useState<ProjectionHealth | null>(null);
  const [sources, setSources] = useState<EvidenceSource[]>([]);
  const [evidenceRequests, setEvidenceRequests] = useState<EvidenceRequest[]>([]);
  const [evidenceState, setEvidenceState] = useState<LoadState>("idle");
  const [guide, setGuide] = useState<OnboardingGuide | null>(null);
  const [guideState, setGuideState] = useState<OnboardingState | null>(null);
  const [activePanel, setActivePanel] = useState<"none" | "routing" | "capture">("none");

  useEffect(() => {
    Promise.allSettled([loadContext(), loadToday(), loadReadiness(), loadOnboardingGuide(), loadOnboardingState()]).then((results) => {
      const [contextResult, todayResult, readinessResult, guideResult, stateResult] = results;
      const currentRuntime = contextResult.status === "fulfilled" ? contextResult.value : null;
      setRuntime(currentRuntime);
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
      if (readinessResult.status === "fulfilled") {
        setReadiness(readinessResult.value);
        setReadinessState("live");
      } else {
        setReadinessState("unavailable");
      }
      setGuide(guideResult.status === "fulfilled" ? guideResult.value : fallbackGuide);
      setGuideState(stateResult.status === "fulfilled" ? stateResult.value : {
        tenant_id: currentRuntime?.tenant.id ?? "",
        principal_id: currentRuntime?.actor.id ?? "",
        guide_code: fallbackGuide.code,
        guide_version: 1,
        current_step: 0,
        completed: false,
        dismissed: sessionStorage.getItem("clearsight-guide-dismissed") === "1",
        version: 0,
      });
    });
  }, []);

  async function loadEvidenceWorkspace() {
    setEvidenceState("loading");
    const [sourcesResult, requestsResult] = await Promise.allSettled([loadEvidenceSources(), loadEvidenceRequests()]);
    if (sourcesResult.status === "fulfilled" && requestsResult.status === "fulfilled") {
      setSources(sourcesResult.value);
      setEvidenceRequests(requestsResult.value);
      setEvidenceState("live");
    } else {
      setEvidenceState("unavailable");
    }
  }

  async function loadConfigureWorkspace() {
    setConfigureState("loading");
    const [policiesResult, integrityResult, tasksResult, projectionResult] = await Promise.allSettled([loadPolicies(), loadIntegrity(), loadWorkflowTasks(), loadProjectionHealth()]);
    if (policiesResult.status === "fulfilled" && integrityResult.status === "fulfilled" && tasksResult.status === "fulfilled") {
      setPolicies(policiesResult.value);
      setIntegrity(integrityResult.value);
      setTasks(tasksResult.value);
      setProjectionHealth(projectionResult.status === "fulfilled" ? projectionResult.value[0] ?? null : null);
      setConfigureState("live");
    } else {
      setConfigureState("unavailable");
    }
  }

  async function checkProgramStatusRecords(): Promise<ReconcileResult> {
    const result = await reconcileProgramState();
    const health = await loadProjectionHealth();
    setProjectionHealth(health[0] ?? null);
    return result;
  }

  useEffect(() => {
    if (activeView === "work" && workTab === "evidence" && evidenceState === "idle") void loadEvidenceWorkspace();
  }, [activeView, workTab, evidenceState]);

  useEffect(() => {
    if (activeView === "configure" && configureState === "idle") void loadConfigureWorkspace();
  }, [activeView, configureState]);

  const dueSoon = useMemo(() => {
    const now = Date.now();
    return items.filter((item) => {
      const due = Date.parse(item.due_at);
      const remaining = due - now;
      return Number.isFinite(due) && remaining >= 0 && remaining <= 4 * 86400000;
    }).length;
  }, [items]);

  const organizationName = runtime?.tenant.name || "Connected organization";
  const legalEntityName = runtime?.legal_entity.name || "Connected legal entity";
  const actorName = runtime?.actor.name || runtime?.actor.id || "Signed-in user";
  const avatarText = initials(actorName);

  async function inspectRouting() { setActivePanel("routing"); if (!resolution) setResolution(await resolveAuthority().catch(() => null)); }
  async function openCapture() { setActivePanel("capture"); if (!capture) setCapture(await loadCaptureRequest().catch(() => null)); }
  async function advanceGuide(next: OnboardingState) { setGuideState(next); const saved = await saveOnboardingState(next).catch(() => null); if (saved) setGuideState(saved); }
  async function dismissGuide() { sessionStorage.setItem("clearsight-guide-dismissed", "1"); if (!guideState) return; const next = { ...guideState, dismissed: true }; setGuideState(next); const saved = await saveOnboardingState(next).catch(() => null); if (saved) setGuideState(saved); }
  function openAttention(item: AttentionItem) {
    if (item.action_target_type === "PROGRAM") setActiveView("programs");
    if (item.action_target_type === "MATTER") { setWorkTab("matters"); setActiveView("work"); }
    if (item.action_target_type === "EVIDENCE_REQUEST") { setWorkTab("evidence"); setActiveView("work"); }
  }

  return <div className="app-shell">
    <aside className="sidebar" aria-label="Primary navigation">
      <div className="brand-mark" aria-label="ClearSight">C</div>
      <nav>{["Today", "Programs", "Work", "Explore", "Configure"].map((label) => {
        const view: View | null = label === "Today" ? "today" : label === "Programs" ? "programs" : label === "Work" ? "work" : label === "Explore" ? "explore" : label === "Configure" ? "configure" : null;
        const active = view === activeView;
        return <button className={active ? "nav-item active" : "nav-item"} key={label} aria-current={active ? "page" : undefined} disabled={!view} aria-disabled={!view} title={!view ? `${label} is not available in this build` : undefined} onClick={() => view && setActiveView(view)}><span>{label.slice(0, 1)}</span><b>{label}</b></button>;
      })}</nav>
      <div className="avatar" aria-label={`Signed in as ${actorName}`}>{avatarText}</div>
    </aside>
    <main>
      {activeView === "today" && <TodayView organizationName={organizationName} items={items} dueSoon={dueSoon} connection={connection} readiness={readiness} readinessState={readinessState === "idle" ? "loading" : readinessState} onRouting={inspectRouting} onCapture={openCapture} onOpenItem={openAttention}/>}
      {activeView === "programs" && <ProgramsView organizationName={organizationName}/>} 
      {activeView === "work" && <WorkView organizationName={organizationName} tab={workTab} onTab={setWorkTab} sources={sources} requests={evidenceRequests} evidenceState={evidenceState} onEvidenceRetry={() => void loadEvidenceWorkspace()}/>} 
      {activeView === "explore" && <ExploreView organizationName={organizationName}/>} 
      {activeView === "configure" && <ConfigureView policies={policies} findings={integrity} tasks={tasks} projectionHealth={projectionHealth} state={configureState} onRetry={() => void loadConfigureWorkspace()} onReconcile={checkProgramStatusRecords}/>} 
    </main>
    {activePanel !== "none" && <div className="panel-backdrop" onMouseDown={() => setActivePanel("none")}><aside className="side-panel" onMouseDown={(event) => event.stopPropagation()} aria-label={activePanel === "routing" ? "Approval route" : "Evidence request"}><button className="panel-close" onClick={() => setActivePanel("none")} aria-label="Close">×</button>{activePanel === "routing" ? <RoutingPanel resolution={resolution} legalEntityName={legalEntityName}/> : <CapturePanel request={capture}/>}</aside></div>}
    {guide && guideState && !guideState.completed && !guideState.dismissed && <IntroGuide guide={guide} state={guideState} onAdvance={advanceGuide} onDismiss={dismissGuide}/>} 
  </div>;
}

function initials(value: string) {
  const parts = value.trim().split(/\s+/).filter(Boolean);
  const first = parts.at(0)?.at(0) ?? value.at(0) ?? "";
  const last = parts.length > 1 ? parts.at(-1)?.at(0) ?? "" : value.at(1) ?? "";
  return `${first}${last}`.toUpperCase();
}

export default App;
