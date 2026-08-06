import { useEffect, useMemo, useState } from "react";
import { loadCaptureRequest, loadContext, loadEvidenceRequests, loadEvidenceSources, loadIntegrity, loadPolicies, loadProjectionHealth, loadReadiness, loadToday, loadWorkflowTasks, reconcileProgramState, resolveAuthority } from "./api";
import type { RuntimeContext } from "./api";
import { CapturePanel, ConfigureView, ExploreView, ProgramsView, RoutingPanel, TodayView, WorkView } from "./AppViews";
import { DocumentImportWorkspace } from "./components/DocumentImportWorkspace";
import { RoleAwareOnboarding } from "./components/RoleAwareOnboarding";
import type { AttentionItem, AuthorityResolution, CaptureRequest, EvidenceRequest, EvidenceSource, GuideStep, IntegrityFinding, PolicySummary, Readiness, WorkflowTask } from "./types";
import type { ProjectionHealth, ReconcileResult } from "./operationsTypes";

type View = "today" | "programs" | "work" | "imports" | "explore" | "configure";
type LoadState = "idle" | "loading" | "live" | "unavailable";
type ConnectionState = "loading" | "live" | "sample" | "unavailable";
type ProductRuntime = RuntimeContext & { demo_mode?: boolean; capabilities?: { document_import?: boolean; reference_journeys?: boolean }; actor: RuntimeContext["actor"] & { role_codes?: string[] } };
type WorkspaceTarget = { programID?: string; matterID?: string; evidenceID?: string; openFirstProgram?: boolean; openFirstMatter?: boolean; openFirstEvidence?: boolean };

const sampleMode = import.meta.env.VITE_ENABLE_SAMPLE_DATA === "true";
const fallbackItems: AttentionItem[] = [
  { id: "fallback-1", type: "REGULATORY_CHANGE", title: "Review proposed digital-channel requirements", why_now: "Seven provisions may affect mobile banking and two payment vendors.", scope: "Digital Channels · Reference data", state: "Applicability review", evidence: "Official source recorded", owner: "Regulatory Compliance", due_at: new Date(Date.now() + 3 * 86400000).toISOString(), primary_action: "Review the proposed requirements" },
  { id: "fallback-2", type: "MATTER", title: "Confirm four privileged-account owners", why_now: "1,246 accounts are resolved; four still need current business-need evidence.", scope: "Treasury Operations · Reference data", state: "Evidence requested", evidence: "1,246 of 1,250 resolved", owner: "Treasury Technology", due_at: new Date(Date.now() + 36 * 3600000).toISOString(), primary_action: "Confirm the account owners" },
];

function App() {
  const initialRoute = parseRoute(window.location.hash);
  const [runtime, setRuntime] = useState<ProductRuntime | null>(null);
  const [activeView, setActiveView] = useState<View>(initialRoute.view);
  const [workTab, setWorkTab] = useState<"matters" | "evidence">(initialRoute.workTab ?? "matters");
  const [target, setTarget] = useState<WorkspaceTarget>(initialRoute.target);
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
  const [activePanel, setActivePanel] = useState<"none" | "routing" | "capture">("none");

  const demoMode = runtime?.demo_mode === true;
  const importsEnabled = runtime?.capabilities?.document_import !== false;

  useEffect(() => {
    Promise.allSettled([loadContext(), loadToday(), loadReadiness()]).then((results) => {
      const [contextResult, todayResult, readinessResult] = results;
      const currentRuntime = contextResult.status === "fulfilled" ? contextResult.value as ProductRuntime : null;
      setRuntime(currentRuntime);
      const allowFallback = currentRuntime?.demo_mode === true || (currentRuntime == null && sampleMode);
      if (todayResult.status === "fulfilled") {
        setItems(todayResult.value);
        setConnection("live");
      } else if (allowFallback) {
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
    });
  }, []);

  useEffect(() => {
    function routeFromHash() {
      const route = parseRoute(window.location.hash);
      setActiveView(route.view);
      if (route.workTab) setWorkTab(route.workTab);
      setTarget(route.target);
    }
    window.addEventListener("hashchange", routeFromHash);
    return () => window.removeEventListener("hashchange", routeFromHash);
  }, []);

  useEffect(() => {
    document.documentElement.dataset.clearsightDemo = demoMode ? "on" : "off";
    if (!demoMode && activeView === "explore") navigate("today");
  }, [demoMode, activeView]);

  async function loadEvidenceWorkspace() {
    setEvidenceState("loading");
    const [sourcesResult, requestsResult] = await Promise.allSettled([loadEvidenceSources(), loadEvidenceRequests()]);
    if (sourcesResult.status === "fulfilled" && requestsResult.status === "fulfilled") {
      setSources(sourcesResult.value);
      setEvidenceRequests(requestsResult.value);
      setEvidenceState("live");
      return requestsResult.value;
    }
    setEvidenceState("unavailable");
    return [];
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
  const roleName = humanRole(runtime?.actor.role_codes?.[0]) || "Assigned user";
  const avatarText = initials(actorName);
  const navigation: Array<{ label: string; view: View }> = [
    { label: "Today", view: "today" }, { label: "Programs", view: "programs" }, { label: "Work", view: "work" },
    ...(importsEnabled ? [{ label: "Imports", view: "imports" as View }] : []),
    ...(demoMode ? [{ label: "Explore", view: "explore" as View }] : []),
    { label: "Configure", view: "configure" },
  ];

  function navigate(view: View, nextTarget: WorkspaceTarget = {}, tab?: "matters" | "evidence") {
    setActiveView(view);
    setTarget(nextTarget);
    if (tab) setWorkTab(tab);
    const hash = routeHash(view, nextTarget, tab ?? workTab);
    if (window.location.hash !== hash) window.history.pushState(null, "", hash);
  }

  async function inspectRouting() {
    setActivePanel("routing");
    if (!resolution) setResolution(await resolveAuthority().catch(() => null));
  }

  async function openCapture(requestID?: string) {
    if (!demoMode && !requestID) return;
    setActivePanel("capture");
    if (requestID) {
      const found = evidenceRequests.find((request) => request.id === requestID);
      if (found) setCapture({ ...found, source: "evidence" });
      return;
    }
    if (!capture) setCapture(await loadCaptureRequest().catch(() => null));
  }

  function openAttention(item: AttentionItem) {
    if (item.action_target_type === "PROGRAM") navigate("programs", { programID: item.action_target_id });
    if (item.action_target_type === "MATTER") navigate("work", { matterID: item.action_target_id }, "matters");
    if (item.action_target_type === "EVIDENCE_REQUEST") navigate("work", { evidenceID: item.action_target_id }, "evidence");
  }

  async function executeGuideStep(step: GuideStep) {
    if (step.intent === "open-routing") {
      navigate("today");
      await inspectRouting();
      return;
    }
    if (step.intent === "open-capture") {
      navigate("today");
      await openCapture();
      return;
    }
    if (step.intent === "open-first-attention" && items[0]) {
      openAttention(items[0]);
      return;
    }
    if (step.intent === "open-first-program") {
      navigate("programs", { openFirstProgram: true });
      return;
    }
    if (step.intent === "open-first-matter") {
      navigate("work", { openFirstMatter: true }, "matters");
      return;
    }
    if (step.intent === "switch-evidence" || step.intent === "open-first-evidence") {
      const requests = evidenceState === "idle" ? await loadEvidenceWorkspace() : evidenceRequests;
      navigate("work", { evidenceID: step.intent === "open-first-evidence" ? requests[0]?.id : undefined, openFirstEvidence: step.intent === "open-first-evidence" }, "evidence");
      return;
    }
    if (step.view) navigate(step.view);
  }

  return <div className="app-shell">
    <aside className="sidebar" aria-label="Primary navigation">
      <div className="brand-mark" aria-label="ClearSight">C</div>
      <nav>{navigation.map(({ label, view }) => {
        const active = view === activeView;
        return <button className={active ? "nav-item active" : "nav-item"} key={label} aria-current={active ? "page" : undefined} onClick={() => navigate(view)}><NavigationIcon view={view}/><b>{label}</b></button>;
      })}</nav>
      <div className="avatar" aria-label={`Signed in as ${actorName}`}>{avatarText}</div>
    </aside>
    <main>
      <div className="context-bar" aria-label="Active workspace context">
        <div><strong>{organizationName}</strong><span>{legalEntityName}</span></div>
        <div className="context-role"><span>{roleName}</span>{demoMode && <mark>Stakeholder demo</mark>}</div>
      </div>
      {activeView === "today" && <TodayView organizationName={organizationName} items={items} dueSoon={dueSoon} connection={connection} readiness={readiness} readinessState={readinessState === "idle" ? "loading" : readinessState} onRouting={inspectRouting} onCapture={() => void openCapture()} onOpenItem={openAttention}/>}
      {activeView === "programs" && <ProgramsView organizationName={organizationName} targetID={target.programID} openFirst={target.openFirstProgram}/>} 
      {activeView === "work" && <WorkView organizationName={organizationName} tab={workTab} onTab={(tab) => navigate("work", {}, tab)} sources={sources} requests={evidenceRequests} evidenceState={evidenceState} onEvidenceRetry={() => void loadEvidenceWorkspace()} matterTargetID={target.matterID} openFirstMatter={target.openFirstMatter} evidenceTargetID={target.evidenceID} openFirstEvidence={target.openFirstEvidence} onOpenEvidence={(id) => void openCapture(id)}/>} 
      {activeView === "imports" && <><header className="topbar"><div><span className="eyebrow">{organizationName}</span><h1>Imports</h1><p>Bring controlled source material into ClearSight for traceable extraction and human review.</p></div></header><DocumentImportWorkspace/></>} 
      {activeView === "explore" && demoMode && <ExploreView organizationName={organizationName}/>} 
      {activeView === "configure" && <ConfigureView policies={policies} findings={integrity} tasks={tasks} projectionHealth={projectionHealth} state={configureState} onRetry={() => void loadConfigureWorkspace()} onReconcile={checkProgramStatusRecords}/>} 
    </main>
    <nav className="mobile-nav" aria-label="Mobile navigation">{navigation.map(({ label, view }) => <button key={view} type="button" aria-current={activeView === view ? "page" : undefined} onClick={() => navigate(view)}><NavigationIcon view={view}/><span>{label}</span></button>)}</nav>
    {activePanel !== "none" && <div className="panel-backdrop" onMouseDown={() => setActivePanel("none")}><aside className="side-panel" onMouseDown={(event) => event.stopPropagation()} aria-label={activePanel === "routing" ? "Approval route" : "Evidence request"}><button className="panel-close" onClick={() => setActivePanel("none")} aria-label="Close">×</button>{activePanel === "routing" ? <RoutingPanel resolution={resolution} legalEntityName={legalEntityName}/> : <CapturePanel request={capture}/>}</aside></div>}
    <RoleAwareOnboarding runtime={runtime} onStep={executeGuideStep}/>
  </div>;
}

function NavigationIcon({ view }: { view: View }) {
  const common = { width: 20, height: 20, viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeWidth: 1.7, strokeLinecap: "round" as const, strokeLinejoin: "round" as const, "aria-hidden": true };
  if (view === "today") return <svg {...common}><path d="M4 5h16v14H4z"/><path d="M8 3v4M16 3v4M7 11h4M7 15h7"/></svg>;
  if (view === "programs") return <svg {...common}><path d="M5 4h14v16H5z"/><path d="M8 8h8M8 12h8M8 16h5"/></svg>;
  if (view === "work") return <svg {...common}><path d="M9 5h6l1 3h4v11H4V8h4z"/><path d="M8 13h8"/></svg>;
  if (view === "imports") return <svg {...common}><path d="M12 3v12M8 7l4-4 4 4"/><path d="M5 14v6h14v-6"/></svg>;
  if (view === "explore") return <svg {...common}><circle cx="12" cy="12" r="9"/><path d="m15 9-2 5-5 2 2-5z"/></svg>;
  return <svg {...common}><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.9l.1.1-2.8 2.8-.1-.1a1.7 1.7 0 0 0-1.9-.3 1.7 1.7 0 0 0-1 1.6V21h-4v-.1a1.7 1.7 0 0 0-1-1.6 1.7 1.7 0 0 0-1.9.3l-.1.1L4.2 17l.1-.1a1.7 1.7 0 0 0 .3-1.9A1.7 1.7 0 0 0 3 14H3v-4h.1a1.7 1.7 0 0 0 1.6-1 1.7 1.7 0 0 0-.3-1.9L4.2 7 7 4.2l.1.1A1.7 1.7 0 0 0 9 4.6a1.7 1.7 0 0 0 1-1.6V3h4v.1a1.7 1.7 0 0 0 1 1.6 1.7 1.7 0 0 0 1.9-.3l.1-.1L19.8 7l-.1.1a1.7 1.7 0 0 0-.3 1.9 1.7 1.7 0 0 0 1.6 1h.1v4H21a1.7 1.7 0 0 0-1.6 1z"/></svg>;
}

function parseRoute(hash: string): { view: View; workTab?: "matters" | "evidence"; target: WorkspaceTarget } {
  const parts = hash.replace(/^#\/?/, "").split("/").filter(Boolean);
  const view = (["today", "programs", "work", "imports", "explore", "configure"].includes(parts[0] ?? "") ? parts[0] : "today") as View;
  if (view === "programs") return { view, target: { programID: parts[1] } };
  if (view === "work") {
    const tab = parts[1] === "evidence" ? "evidence" : "matters";
    return { view, workTab: tab, target: tab === "evidence" ? { evidenceID: parts[2] } : { matterID: parts[2] } };
  }
  return { view, target: {} };
}

function routeHash(view: View, target: WorkspaceTarget, tab: "matters" | "evidence") {
  if (view === "programs" && target.programID) return `#programs/${encodeURIComponent(target.programID)}`;
  if (view === "work") {
    const id = tab === "evidence" ? target.evidenceID : target.matterID;
    return `#work/${tab}${id ? `/${encodeURIComponent(id)}` : ""}`;
  }
  return `#${view}`;
}

function humanRole(value?: string) {
  return value?.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase()) ?? "";
}

function initials(value: string) {
  const parts = value.trim().split(/\s+/).filter(Boolean);
  const first = parts.at(0)?.at(0) ?? value.at(0) ?? "";
  const last = parts.length > 1 ? parts.at(-1)?.at(0) ?? "" : value.at(1) ?? "";
  return `${first}${last}`.toUpperCase();
}

export default App;
