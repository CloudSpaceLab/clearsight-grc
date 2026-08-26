import { lazy, Suspense, useEffect, useRef, useState } from "react";
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
import { CapturePanel, ConfigureView, ExploreView, ProgramsView, RoutingPanel, TodayView, WorkView, type RoutingLoadState } from "./AppViews";
import { DisplayPreferencesMenu } from "./components/DisplayPreferences";
import { DocumentImportWorkspace } from "./components/DocumentImportWorkspace";
import { FocusedSheet } from "./components/FocusedSheet";
import { NavigationIcon } from "./components/NavigationIcon";
import { RoleAwareOnboarding } from "./components/RoleAwareOnboarding";
import type { CaptureLoadState } from "./components/CapturePanel";
import { apiErrorKind } from "./http";
import { parseRoute, routeHash } from "./appRouting";
import type { View, WorkspaceTarget, WorkTab } from "./appRouting";
import type { RuntimePresentation } from "./runtimePresentation";
import type { AttentionItem, AutomationPolicy, AuthorityResolution, CaptureRequest, EvidenceRequest, EvidenceSource, GuideStep, IntegrityFinding, PolicySummary, Readiness, WorkflowTask } from "./types";
import type { ProjectionHealth, ReconcileResult } from "./operationsTypes";

const VendorsWorkspace = lazy(() => import("./components/VendorsWorkspace").then((module) => ({ default: module.VendorsWorkspace })));

type LoadState = "idle" | "loading" | "live" | "unavailable";
type SectionLoadState = Exclude<LoadState, "idle">;
type ConnectionState = "loading" | "live" | "sample" | "unavailable";
type VendorGuideIntent = { id: number; type: "open-vendor-due-diligence" | "open-vendor-work" };
type ProductRuntime = RuntimeContext & {
  demo_mode?: boolean;
  capabilities?: {
    document_import?: boolean;
    reference_journeys?: boolean;
    config_read?: boolean;
    config_write?: boolean;
    platform_operations_read?: boolean;
    platform_operations_write?: boolean;
  };
  actor: RuntimeContext["actor"] & { role_codes?: string[] };
};

const sampleMode = import.meta.env.VITE_ENABLE_SAMPLE_DATA === "true";
const fallbackItems: AttentionItem[] = [{
  id: "fallback-change", type: "REGULATORY_CHANGE", title: "Review proposed digital-channel requirements", why_now: "Seven provisions may affect mobile banking and two payment vendors.", scope: "Digital Channels · Reference data", state: "Applicability review", evidence: "Official source recorded", owner: "Regulatory Compliance", due_at: new Date(Date.now() + 3 * 86400000).toISOString(), primary_action: "Review the proposed requirements", intervention_class: "REVIEW", material_conclusion: "Seven source-linked provisions may change digital-channel obligations.", recommendation: { proposed_action: "Review the proposed requirements", rationale: "The source change may affect mobile banking and two payment vendors." },
}];

function App({ presentation = "demo" }: { presentation?: RuntimePresentation }) {
  const initialRoute = parseRoute(window.location.hash);
  const [runtime, setRuntime] = useState<ProductRuntime | null>(null);
  const [activeView, setActiveView] = useState<View>(initialRoute.view);
  const [workTab, setWorkTab] = useState<WorkTab>(initialRoute.workTab ?? "matters");
  const [target, setTarget] = useState<WorkspaceTarget>(initialRoute.target);
  const [vendorGuideIntent, setVendorGuideIntent] = useState<VendorGuideIntent>();
  const vendorGuideIntentID = useRef(0);
  const vendorGuideAck = useRef<{ id: number; resolve: () => void; reject: (reason?: unknown) => void } | undefined>(undefined);
  const [items, setItems] = useState<AttentionItem[]>([]);
  const [todayGeneratedAt, setTodayGeneratedAt] = useState<string | undefined>();
  const [connection, setConnection] = useState<ConnectionState>("loading");
  const [readinessState, setReadinessState] = useState<LoadState>("loading");
  const [resolution, setResolution] = useState<AuthorityResolution | null>(null);
  const [routingItem, setRoutingItem] = useState<AttentionItem | null>(null);
  const [routingState, setRoutingState] = useState<RoutingLoadState>("loading");
  const [capture, setCapture] = useState<CaptureRequest | null>(null);
  const [captureState, setCaptureState] = useState<CaptureLoadState>("loading");
  const [captureRequestID, setCaptureRequestID] = useState<string | undefined>();
  const [readiness, setReadiness] = useState<Readiness | null>(null);
  const [policies, setPolicies] = useState<PolicySummary[]>([]);
  const [policyState, setPolicyState] = useState<SectionLoadState>("loading");
  const [integrity, setIntegrity] = useState<IntegrityFinding[]>([]);
  const [integrityState, setIntegrityState] = useState<SectionLoadState>("loading");
  const [tasks, setTasks] = useState<WorkflowTask[]>([]);
  const [taskState, setTaskState] = useState<SectionLoadState>("loading");
  const [configureState, setConfigureState] = useState<LoadState>("idle");
  const [automationPolicies, setAutomationPolicies] = useState<AutomationPolicy[]>([]);
  const [automationPolicyState, setAutomationPolicyState] = useState<SectionLoadState>("loading");
  const [projectionHealth, setProjectionHealth] = useState<ProjectionHealth | null>(null);
  const [projectionState, setProjectionState] = useState<SectionLoadState>("loading");
  const [sources, setSources] = useState<EvidenceSource[]>([]);
  const [evidenceRequests, setEvidenceRequests] = useState<EvidenceRequest[]>([]);
  const [evidenceSourceState, setEvidenceSourceState] = useState<LoadState>("idle");
  const [evidenceRequestState, setEvidenceRequestState] = useState<LoadState>("idle");
  const [activePanel, setActivePanel] = useState<"none" | "routing" | "capture">("none");
  const captureLoadID = useRef(0);
  const routingLoadID = useRef(0);
  const evidenceTargetAttempts = useRef(new Set<string>());

  const serverDemoMode = runtime?.demo_mode === true;
  const demoMode = serverDemoMode && presentation === "demo";
  const importsEnabled = runtime != null && runtime.capabilities?.document_import !== false;
  const configureEnabled = runtime != null && runtime.capabilities?.config_read !== false;

  useEffect(() => {
    void Promise.allSettled([loadContext(), loadToday(), loadReadiness()]).then(([contextResult, todayResult, readinessResult]) => {
      const currentRuntime = contextResult.status === "fulfilled" ? contextResult.value as ProductRuntime : null;
      setRuntime(currentRuntime);
      const allowFallback = (currentRuntime?.demo_mode === true && presentation === "demo") ||
        (currentRuntime == null && sampleMode && presentation === "demo");
      if (todayResult.status === "fulfilled") {
        setItems(todayResult.value.items);
        setTodayGeneratedAt(todayResult.value.generated_at);
        setConnection("live");
      } else if (allowFallback) {
        setItems(fallbackItems);
        setTodayGeneratedAt(undefined);
        setConnection("sample");
      } else {
        setItems([]);
        setTodayGeneratedAt(undefined);
        setConnection("unavailable");
      }
      if (readinessResult.status === "fulfilled") {
        setReadiness(readinessResult.value);
        setReadinessState("live");
      } else {
        setReadiness(null);
        setReadinessState("unavailable");
      }
    });
  }, [presentation]);

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
    if (!runtime) return;
    if ((!demoMode && activeView === "explore") || (!importsEnabled && activeView === "imports") || (!configureEnabled && activeView === "configure")) navigate("today");
  }, [runtime, demoMode, importsEnabled, configureEnabled, activeView]);

  async function loadEvidenceWorkspace(requestedID?: string) {
    setEvidenceSourceState("loading");
    setEvidenceRequestState("loading");
    const [sourcesResult, requestsResult] = await Promise.allSettled([loadEvidenceSources(), loadEvidenceRequests()]);
    if (sourcesResult.status === "fulfilled") { setSources(sourcesResult.value); setEvidenceSourceState("live"); } else { setSources([]); setEvidenceSourceState("unavailable"); }
    if (requestsResult.status === "fulfilled") {
      let requests = requestsResult.value;
      if (requestedID && !requests.some((request) => request.id === requestedID)) {
        evidenceTargetAttempts.current.add(requestedID);
        const requested = await loadEvidenceRequest(requestedID).catch(() => null);
        if (requested) requests = [requested, ...requests];
      }
      setEvidenceRequests(requests);
      setEvidenceRequestState("live");
      return requests;
    }
    setEvidenceRequests([]);
    setEvidenceRequestState("unavailable");
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
    setPolicyState("loading"); setIntegrityState("loading"); setTaskState("loading"); setProjectionState("loading"); setAutomationPolicyState("loading");
    const [policiesResult, integrityResult, tasksResult, projectionResult, automationResult] = await Promise.allSettled([loadPolicies(), loadIntegrity(), loadWorkflowTasks(), loadProjectionHealth(), loadAutomationPolicies()]);
    if (policiesResult.status === "fulfilled") { setPolicies(policiesResult.value); setPolicyState("live"); } else { setPolicies([]); setPolicyState("unavailable"); }
    if (integrityResult.status === "fulfilled") { setIntegrity(integrityResult.value); setIntegrityState("live"); } else { setIntegrity([]); setIntegrityState("unavailable"); }
    if (tasksResult.status === "fulfilled") { setTasks(tasksResult.value); setTaskState("live"); } else { setTasks([]); setTaskState("unavailable"); }
    if (projectionResult.status === "fulfilled") { setProjectionHealth(projectionResult.value[0] ?? null); setProjectionState("live"); } else { setProjectionHealth(null); setProjectionState("unavailable"); }
    if (automationResult.status === "fulfilled") { setAutomationPolicies(automationResult.value); setAutomationPolicyState("live"); } else { setAutomationPolicies([]); setAutomationPolicyState("unavailable"); }
    setConfigureState([policiesResult, integrityResult, tasksResult, projectionResult].some((result) => result.status === "fulfilled") ? "live" : "unavailable");
  }

  async function checkProgramStatusRecords(): Promise<ReconcileResult> {
    const result = await reconcileProgramState();
    const health = await loadProjectionHealth();
    setProjectionHealth(health[0] ?? null);
    setProjectionState("live");
    return result;
  }

  useEffect(() => {
    if (activeView !== "work" || workTab !== "evidence") return;
    if (evidenceRequestState === "idle" || evidenceSourceState === "idle") { void loadEvidenceWorkspace(target.evidenceID); return; }
    if (evidenceRequestState === "live" && target.evidenceID && !evidenceRequests.some((request) => request.id === target.evidenceID)) void ensureEvidenceTarget(target.evidenceID);
  }, [activeView, workTab, evidenceRequestState, evidenceSourceState, target.evidenceID, evidenceRequests]);

  useEffect(() => {
    if (activeView === "configure" && configureEnabled && configureState === "idle") void loadConfigureWorkspace();
  }, [activeView, configureEnabled, configureState]);

  const organizationName = runtime?.tenant.name || runtime?.tenant.id || "Organization unavailable";
  const legalEntityName = runtime?.legal_entity.name || runtime?.legal_entity.id || "Legal entity unavailable";
  const actorName = runtime?.actor.name || runtime?.actor.id || "User unavailable";
  const roleName = humanRole(runtime?.actor.role_codes?.[0]) || "Role not provided";
  const navigation: Array<{ label: string; view: View }> = [
    { label: "Today", view: "today" }, { label: "Programs", view: "programs" }, { label: "Vendors", view: "vendors" }, { label: "Work", view: "work" },
    ...(importsEnabled ? [{ label: "Imports", view: "imports" as View }] : []),
    ...(demoMode ? [{ label: "Explore", view: "explore" as View }] : []),
    ...(configureEnabled ? [{ label: "Configure", view: "configure" as View }] : []),
  ];

  function navigate(view: View, nextTarget: WorkspaceTarget = {}, tab?: WorkTab) {
    const nextTab = tab ?? workTab;
    setActiveView(view); setTarget(nextTarget); if (tab) setWorkTab(tab);
    const hash = routeHash(view, nextTarget, nextTab);
    if (window.location.hash !== hash) window.history.pushState(null, "", hash);
  }

  function closePanel() { captureLoadID.current++; routingLoadID.current++; setActivePanel("none"); }

  async function inspectRouting(item: AttentionItem) {
    if (!item.authority || !item.action_target_type || !item.action_target_id) return;
    const loadID = ++routingLoadID.current;
    setRoutingItem(item); setResolution(null); setRoutingState("loading"); setActivePanel("routing");
    try {
      const next = await resolveAuthority({ object_type: item.action_target_type, object_id: item.action_target_id, responsibility: item.authority.responsibility, decision_type: item.authority.decision_type, materiality: item.authority.materiality });
      if (loadID !== routingLoadID.current) return;
      setResolution(next); setRoutingState("live");
    } catch (error) {
      if (loadID !== routingLoadID.current) return;
      const kind = apiErrorKind(error);
      setRoutingState(kind === "forbidden" || kind === "unauthorized" ? "forbidden" : kind === "not_found" ? "not-found" : "unavailable");
    }
  }

  async function openCapture(requestID?: string) {
    if (!requestID && !demoMode) return;
    const loadID = ++captureLoadID.current;
    setCaptureRequestID(requestID); setCapture(null); setCaptureState("loading"); setActivePanel("capture");
    try {
      if (requestID) {
        const existing = evidenceRequests.find((request) => request.id === requestID);
        const loaded = existing ?? await loadEvidenceRequest(requestID);
        if (loadID !== captureLoadID.current) return;
        setCapture(loaded); setCaptureState("live"); return;
      }
      const loaded = await loadCaptureRequest();
      if (loadID !== captureLoadID.current) return;
      setCaptureRequestID(loaded.id); setCapture(loaded); setCaptureState("live");
    } catch (error) {
      if (loadID !== captureLoadID.current) return;
      const kind = apiErrorKind(error);
      setCaptureState(kind === "forbidden" || kind === "unauthorized" ? "forbidden" : kind === "not_found" ? "not-found" : "unavailable");
    }
  }

  async function reloadCapture() { if (captureRequestID) await openCapture(captureRequestID); else if (demoMode) await openCapture(); }
  async function openPrimaryEvidence() { const item = items.find((candidate) => candidate.action_target_type === "EVIDENCE_REQUEST" && candidate.action_target_id); if (item?.action_target_id) await openCapture(item.action_target_id); else if (demoMode) await openCapture(); }

  function openAttention(item: AttentionItem) {
    if (item.action_target_type === "PROGRAM") navigate("programs", { programID: item.action_target_id });
    if (item.action_target_type === "MATTER") navigate("work", { matterID: item.action_target_id }, "matters");
    if (item.action_target_type === "EVIDENCE_REQUEST") navigate("work", { evidenceID: item.action_target_id }, "evidence");
  }

  async function executeGuideStep(step: GuideStep) {
    if (step.intent === "open-routing") { navigate("today"); const authorityItem = items.find((item) => item.authority && item.action_target_type && item.action_target_id); if (authorityItem) await inspectRouting(authorityItem); return; }
    if (step.intent === "open-capture") { navigate("today"); await openPrimaryEvidence(); return; }
    if (step.intent === "open-first-attention" && items[0]) { openAttention(items[0]); return; }
    if (step.intent === "open-first-program") { navigate("programs", { openFirstProgram: true }); return; }
    if (step.intent === "open-first-matter") { navigate("work", { openFirstMatter: true }, "matters"); return; }
    const vendorIntent = step.intent;
    if (vendorIntent === "open-vendor-due-diligence" || vendorIntent === "open-vendor-work") {
      return new Promise<void>((resolve, reject) => {
        vendorGuideAck.current?.reject(new Error("Vendor guide action was replaced."));
        const id = ++vendorGuideIntentID.current;
        vendorGuideAck.current = { id, resolve, reject };
        setVendorGuideIntent({ id, type: vendorIntent });
        navigate("vendors", activeView === "vendors" ? target : {});
      });
    }
    if (step.intent === "switch-evidence" || step.intent === "open-first-evidence") {
      const requests = evidenceRequestState === "idle" ? await loadEvidenceWorkspace() : evidenceRequests;
      navigate("work", { evidenceID: step.intent === "open-first-evidence" ? requests[0]?.id : undefined, openFirstEvidence: step.intent === "open-first-evidence" }, "evidence"); return;
    }
    if (step.view) navigate(step.view);
  }

  function completeVendorGuideIntent(id: number) {
    if (vendorGuideAck.current?.id === id) {
      vendorGuideAck.current.resolve();
      vendorGuideAck.current = undefined;
    }
    setVendorGuideIntent((current) => current?.id === id ? undefined : current);
  }

  function failVendorGuideIntent(id: number) {
    if (vendorGuideAck.current?.id === id) {
      vendorGuideAck.current.reject(new Error("Vendor workspace could not be loaded."));
      vendorGuideAck.current = undefined;
    }
  }

  const canOpenEvidence = demoMode || items.some((item) => item.action_target_type === "EVIDENCE_REQUEST" && item.action_target_id);

  return <div className="app-shell">
    <aside className="sidebar" aria-label="Primary navigation"><div className="brand-mark" aria-label="ClearSight">C</div><nav>{navigation.map(({ label, view }) => <button className={view === activeView ? "nav-item active" : "nav-item"} key={view} aria-current={view === activeView ? "page" : undefined} onClick={() => navigate(view)}><NavigationIcon view={view}/><b>{label}</b></button>)}</nav><div className="avatar" aria-label={`Signed in as ${actorName}`}>{initials(actorName)}</div></aside>
    <main>
      <div className="context-bar" aria-label="Active workspace context"><div><strong>{organizationName}</strong><span>{legalEntityName}</span></div><div className="context-role"><DisplayPreferencesMenu/><span>{roleName}</span>{demoMode ? <mark>Stakeholder demo</mark> : serverDemoMode && presentation === "live-preview" ? <mark>Live preview · Non-production</mark> : null}</div></div>
      <RoleAwareOnboarding runtime={runtime} onStep={executeGuideStep}/>
      {activeView === "today" && <TodayView organizationName={organizationName} items={items} connection={connection} generatedAt={todayGeneratedAt} readiness={readiness} readinessState={readinessState === "idle" ? "loading" : readinessState} onCapture={canOpenEvidence ? () => void openPrimaryEvidence() : undefined} onOpenItem={openAttention} onInspectAuthority={(item) => void inspectRouting(item)}/>} 
      {activeView === "programs" && <ProgramsView organizationName={organizationName} actorPrincipalID={runtime?.actor.id} canConfigureSources={runtime?.capabilities?.config_write === true} targetID={target.programID} openFirst={target.openFirstProgram} onOpenRequest={(id) => navigate("work", { evidenceID: id }, "evidence")}/>}
      {activeView === "vendors" && <Suspense fallback={<div className="workspace-loading" aria-live="polite" aria-busy="true">Loading vendor relationships…</div>}><VendorsWorkspace organizationName={organizationName} legalEntityName={legalEntityName} targetID={target.vendorRelationshipID} guideIntent={vendorGuideIntent} onGuideIntentCompleted={completeVendorGuideIntent} onGuideIntentFailed={failVendorGuideIntent} onTarget={(id) => navigate("vendors", id ? { vendorRelationshipID: id } : {})} onOpenRequest={(id) => navigate("work", { evidenceID: id }, "evidence")} onOpenMatter={(id) => navigate("work", { matterID: id }, "matters")}/></Suspense>}
      {activeView === "work" && <WorkView organizationName={organizationName} tab={workTab} onTab={(tab) => navigate("work", {}, tab)} sources={sources} requests={evidenceRequests} evidenceSourceState={evidenceSourceState === "idle" ? "loading" : evidenceSourceState} evidenceRequestState={evidenceRequestState === "idle" ? "loading" : evidenceRequestState} onEvidenceRetry={() => void loadEvidenceWorkspace(target.evidenceID)} matterTargetID={target.matterID} openFirstMatter={target.openFirstMatter} evidenceTargetID={target.evidenceID} openFirstEvidence={target.openFirstEvidence} onOpenEvidence={(id) => void openCapture(id)}/>} 
      {activeView === "imports" && importsEnabled && <><header className="topbar"><div><span className="eyebrow">{organizationName}</span><h1>Imports</h1><p>Compare regulatory documents with current Programs, controls and evidence.</p></div></header><DocumentImportWorkspace/></>}
      {activeView === "explore" && demoMode && <ExploreView organizationName={organizationName}/>} 
      {activeView === "configure" && configureEnabled && <ConfigureView policies={policies} policyState={policyState} findings={integrity} integrityState={integrityState} tasks={tasks} taskState={taskState} projectionHealth={projectionHealth} projectionState={projectionState} automationPolicies={automationPolicies} automationPolicyState={automationPolicyState} state={configureState} onRetry={() => void loadConfigureWorkspace()} onReconcile={checkProgramStatusRecords}/>} 
    </main>
    <nav className="mobile-nav" aria-label="Mobile navigation">{navigation.map(({ label, view }) => <button key={view} type="button" aria-current={activeView === view ? "page" : undefined} onClick={() => navigate(view)}><NavigationIcon view={view}/><span>{label}</span></button>)}</nav>
    {activePanel !== "none" && <FocusedSheet label={activePanel === "routing" ? "Authority for selected work" : "Evidence request"} onClose={closePanel}>{activePanel === "routing" ? <RoutingPanel resolution={resolution} item={routingItem} legalEntityName={legalEntityName} state={routingState}/> : <CapturePanel request={capture} state={captureState} onReload={() => void reloadCapture()}/>}</FocusedSheet>}
  </div>;
}

function humanRole(value?: string) {
  if (!value) return "";
  if (/^[A-Z0-9]+$/.test(value)) return value;
  return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}
function initials(value: string) { const parts = value.trim().split(/\s+/).filter(Boolean); const first = parts.at(0)?.at(0) ?? value.at(0) ?? ""; const last = parts.length > 1 ? parts.at(-1)?.at(0) ?? "" : value.at(1) ?? ""; return `${first}${last}`.toUpperCase(); }

export default App;
