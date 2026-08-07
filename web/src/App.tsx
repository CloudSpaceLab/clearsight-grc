import { useEffect, useRef, useState } from "react";
import {
  loadAutomationPolicies,
  loadCaptureRequest,
  loadContext,
  loadEvidenceRequest,
  loadEvidenceRequests,
  loadEvidenceSources,
  loadIntegrity,
  loadPolicies,
  loadProjectionHealth,
  loadReadiness,
  loadToday,
  loadWorkflowTasks,
  reconcileProgramState,
  resolveAuthority,
} from "./api";
import type { RuntimeContext } from "./api";
import { CapturePanel, ConfigureView, ExploreView, ProgramsView, RoutingPanel, TodayView, WorkView } from "./AppViews";
import { DocumentImportWorkspace } from "./components/DocumentImportWorkspace";
import { NavigationIcon } from "./components/NavigationIcon";
import { RoleAwareOnboarding } from "./components/RoleAwareOnboarding";
import { parseRoute, routeHash } from "./appRouting";
import type { View, WorkspaceTarget, WorkTab } from "./appRouting";
import type { AttentionItem, AutomationPolicy, AuthorityResolution, CaptureRequest, EvidenceRequest, EvidenceSource, GuideStep, IntegrityFinding, PolicySummary, Readiness, WorkflowTask } from "./types";
import type { ProjectionHealth, ReconcileResult } from "./operationsTypes";

type LoadState = "idle" | "loading" | "live" | "unavailable";
type ConnectionState = "loading" | "live" | "sample" | "unavailable";
type ProductRuntime = RuntimeContext & {
  demo_mode?: boolean;
  capabilities?: { document_import?: boolean; reference_journeys?: boolean };
  actor: RuntimeContext["actor"] & { role_codes?: string[] };
};

const sampleMode = import.meta.env.VITE_ENABLE_SAMPLE_DATA === "true";
const fallbackItems: AttentionItem[] = [
  {
    id: "fallback-change",
    type: "REGULATORY_CHANGE",
    title: "Review proposed digital-channel requirements",
    why_now: "Seven provisions may affect mobile banking and two payment vendors.",
    scope: "Digital Channels · Reference data",
    state: "Applicability review",
    evidence: "Official source recorded",
    owner: "Regulatory Compliance",
    due_at: new Date(Date.now() + 3 * 86400000).toISOString(),
    primary_action: "Review the proposed requirements",
    intervention_class: "REVIEW",
    material_conclusion: "Seven source-linked provisions may change digital-channel obligations.",
    recommendation: {
      proposed_action: "Review the proposed requirements",
      rationale: "The source change may affect mobile banking and two payment vendors.",
    },
  },
];

function App() {
  const initialRoute = parseRoute(window.location.hash);
  const [runtime, setRuntime] = useState<ProductRuntime | null>(null);
  const [activeView, setActiveView] = useState<View>(initialRoute.view);
  const [workTab, setWorkTab] = useState<WorkTab>(initialRoute.workTab ?? "matters");
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
  const [automationPolicies, setAutomationPolicies] = useState<AutomationPolicy[]>([]);
  const [automationPolicyState, setAutomationPolicyState] = useState<Exclude<LoadState, "idle">>("loading");
  const [projectionHealth, setProjectionHealth] = useState<ProjectionHealth | null>(null);
  const [sources, setSources] = useState<EvidenceSource[]>([]);
  const [evidenceRequests, setEvidenceRequests] = useState<EvidenceRequest[]>([]);
  const [evidenceState, setEvidenceState] = useState<LoadState>("idle");
  const [activePanel, setActivePanel] = useState<"none" | "routing" | "capture">("none");
  const captureLoadID = useRef(0);
  const evidenceTargetAttempts = useRef(new Set<string>());

  const demoMode = runtime?.demo_mode === true;
  const importsEnabled = runtime?.capabilities?.document_import !== false;

  useEffect(() => {
    void Promise.allSettled([loadContext(), loadToday(), loadReadiness()]).then(([contextResult, todayResult, readinessResult]) => {
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
    const syncRoute = () => {
      const route = parseRoute(window.location.hash);
      setActiveView(route.view);
      if (route.workTab) setWorkTab(route.workTab);
      setTarget(route.target);
    };
    window.addEventListener("hashchange", syncRoute);
    window.addEventListener("popstate", syncRoute);
    return () => {
      window.removeEventListener("hashchange", syncRoute);
      window.removeEventListener("popstate", syncRoute);
    };
  }, []);

  useEffect(() => {
    document.documentElement.dataset.clearsightDemo = demoMode ? "on" : "off";
    if (!demoMode && activeView === "explore") navigate("today");
  }, [demoMode, activeView]);

  async function loadEvidenceWorkspace(requestedID?: string) {
    setEvidenceState("loading");
    const [sourcesResult, requestsResult] = await Promise.allSettled([loadEvidenceSources(), loadEvidenceRequests()]);
    if (sourcesResult.status === "fulfilled" && requestsResult.status === "fulfilled") {
      let requests = requestsResult.value;
      if (requestedID && !requests.some((request) => request.id === requestedID)) {
        evidenceTargetAttempts.current.add(requestedID);
        const requested = await loadEvidenceRequest(requestedID).catch(() => null);
        if (requested) requests = [requested, ...requests];
      }
      setSources(sourcesResult.value);
      setEvidenceRequests(requests);
      setEvidenceState("live");
      return requests;
    }
    setEvidenceState("unavailable");
    return [];
  }

  async function ensureEvidenceTarget(requestedID: string) {
    if (evidenceTargetAttempts.current.has(requestedID)) return;
    evidenceTargetAttempts.current.add(requestedID);
    const requested = await loadEvidenceRequest(requestedID).catch(() => null);
    if (requested) setEvidenceRequests((current) => current.some((item) => item.id === requested.id) ? current : [requested, ...current]);
  }

  async function loadConfigureWorkspace() {
    setConfigureState("loading");
    setAutomationPolicyState("loading");
    const [policiesResult, integrityResult, tasksResult, projectionResult, automationResult] = await Promise.allSettled([
      loadPolicies(),
      loadIntegrity(),
      loadWorkflowTasks(),
      loadProjectionHealth(),
      loadAutomationPolicies(),
    ]);
    if (automationResult.status === "fulfilled") {
      setAutomationPolicies(automationResult.value);
      setAutomationPolicyState("live");
    } else {
      setAutomationPolicies([]);
      setAutomationPolicyState("unavailable");
    }
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
    if (activeView !== "work" || workTab !== "evidence") return;
    if (evidenceState === "idle") {
      void loadEvidenceWorkspace(target.evidenceID);
      return;
    }
    if (evidenceState === "live" && target.evidenceID && !evidenceRequests.some((request) => request.id === target.evidenceID)) {
      void ensureEvidenceTarget(target.evidenceID);
    }
  }, [activeView, workTab, evidenceState, target.evidenceID, evidenceRequests]);

  useEffect(() => {
    if (activeView === "configure" && configureState === "idle") void loadConfigureWorkspace();
  }, [activeView, configureState]);

  const organizationName = runtime?.tenant.name || "Connected organization";
  const legalEntityName = runtime?.legal_entity.name || "Connected legal entity";
  const actorName = runtime?.actor.name || runtime?.actor.id || "Signed-in user";
  const roleName = humanRole(runtime?.actor.role_codes?.[0]) || "Assigned user";
  const navigation: Array<{ label: string; view: View }> = [
    { label: "Today", view: "today" },
    { label: "Programs", view: "programs" },
    { label: "Work", view: "work" },
    ...(importsEnabled ? [{ label: "Imports", view: "imports" as View }] : []),
    ...(demoMode ? [{ label: "Explore", view: "explore" as View }] : []),
    { label: "Configure", view: "configure" },
  ];

  function navigate(view: View, nextTarget: WorkspaceTarget = {}, tab?: WorkTab) {
    const nextTab = tab ?? workTab;
    setActiveView(view);
    setTarget(nextTarget);
    if (tab) setWorkTab(tab);
    const hash = routeHash(view, nextTarget, nextTab);
    if (window.location.hash !== hash) window.history.pushState(null, "", hash);
  }

  function closePanel() {
    captureLoadID.current++;
    setActivePanel("none");
  }

  async function inspectRouting() {
    setActivePanel("routing");
    if (!resolution) setResolution(await resolveAuthority().catch(() => null));
  }

  async function openCapture(requestID?: string) {
    if (!requestID && !demoMode) return;
    const loadID = ++captureLoadID.current;
    setActivePanel("capture");

    if (requestID) {
      const existing = evidenceRequests.find((request) => request.id === requestID);
      if (existing) {
        setCapture(existing);
        return;
      }
      setCapture(null);
      const loaded = await loadEvidenceRequest(requestID).catch(() => null);
      if (loadID !== captureLoadID.current) return;
      setCapture(loaded);
      return;
    }

    if (capture) return;
    setCapture(null);
    const loaded = await loadCaptureRequest().catch(() => null);
    if (loadID !== captureLoadID.current) return;
    setCapture(loaded);
  }

  async function openPrimaryEvidence() {
    const item = items.find((candidate) => candidate.action_target_type === "EVIDENCE_REQUEST" && candidate.action_target_id);
    if (item?.action_target_id) {
      await openCapture(item.action_target_id);
      return;
    }
    if (demoMode) await openCapture();
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
      await openPrimaryEvidence();
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
      navigate("work", {
        evidenceID: step.intent === "open-first-evidence" ? requests[0]?.id : undefined,
        openFirstEvidence: step.intent === "open-first-evidence",
      }, "evidence");
      return;
    }
    if (step.view) navigate(step.view);
  }

  const canOpenEvidence = demoMode || items.some((item) => item.action_target_type === "EVIDENCE_REQUEST" && item.action_target_id);

  return <div className="app-shell">
    <aside className="sidebar" aria-label="Primary navigation">
      <div className="brand-mark" aria-label="ClearSight">C</div>
      <nav>{navigation.map(({ label, view }) => <button className={view === activeView ? "nav-item active" : "nav-item"} key={view} aria-current={view === activeView ? "page" : undefined} onClick={() => navigate(view)}><NavigationIcon view={view}/><b>{label}</b></button>)}</nav>
      <div className="avatar" aria-label={`Signed in as ${actorName}`}>{initials(actorName)}</div>
    </aside>
    <main>
      <div className="context-bar" aria-label="Active workspace context"><div><strong>{organizationName}</strong><span>{legalEntityName}</span></div><div className="context-role"><span>{roleName}</span>{demoMode && <mark>Stakeholder demo</mark>}</div></div>
      {activeView === "today" && <TodayView organizationName={organizationName} items={items} connection={connection} readiness={readiness} readinessState={readinessState === "idle" ? "loading" : readinessState} onRouting={inspectRouting} onCapture={canOpenEvidence ? () => void openPrimaryEvidence() : undefined} onOpenItem={openAttention}/>} 
      {activeView === "programs" && <ProgramsView organizationName={organizationName} targetID={target.programID} openFirst={target.openFirstProgram}/>} 
      {activeView === "work" && <WorkView organizationName={organizationName} tab={workTab} onTab={(tab) => navigate("work", {}, tab)} sources={sources} requests={evidenceRequests} evidenceState={evidenceState} onEvidenceRetry={() => void loadEvidenceWorkspace(target.evidenceID)} matterTargetID={target.matterID} openFirstMatter={target.openFirstMatter} evidenceTargetID={target.evidenceID} openFirstEvidence={target.openFirstEvidence} onOpenEvidence={(id) => void openCapture(id)}/>} 
      {activeView === "imports" && <><header className="topbar"><div><span className="eyebrow">{organizationName}</span><h1>Imports</h1><p>Bring controlled source material into ClearSight for traceable extraction and human review.</p></div></header><DocumentImportWorkspace/></>} 
      {activeView === "explore" && demoMode && <ExploreView organizationName={organizationName}/>} 
      {activeView === "configure" && <ConfigureView policies={policies} findings={integrity} tasks={tasks} projectionHealth={projectionHealth} automationPolicies={automationPolicies} automationPolicyState={automationPolicyState} state={configureState} onRetry={() => void loadConfigureWorkspace()} onReconcile={checkProgramStatusRecords}/>} 
    </main>
    <nav className="mobile-nav" aria-label="Mobile navigation">{navigation.map(({ label, view }) => <button key={view} type="button" aria-current={activeView === view ? "page" : undefined} onClick={() => navigate(view)}><NavigationIcon view={view}/><span>{label}</span></button>)}</nav>
    {activePanel !== "none" && <div className="panel-backdrop" onMouseDown={closePanel}><aside className="side-panel" onMouseDown={(event) => event.stopPropagation()} aria-label={activePanel === "routing" ? "Approval route" : "Evidence request"}><button className="panel-close" onClick={closePanel} aria-label="Close">×</button>{activePanel === "routing" ? <RoutingPanel resolution={resolution} legalEntityName={legalEntityName}/> : <CapturePanel request={capture}/>}</aside></div>}
    <RoleAwareOnboarding runtime={runtime} onStep={executeGuideStep}/>
  </div>;
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
