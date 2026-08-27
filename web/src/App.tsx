import { lazy, Suspense, useEffect, useRef, useState } from "react";
import {
  loadAIGovernancePolicies,
  loadAIGovernanceWorkloads,
  loadAutomationPolicies,
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
import { initials } from "./components/Monogram";
import { RoleAwareOnboarding } from "./components/RoleAwareOnboarding";
import type { CaptureLoadState } from "./components/CapturePanel";
import { apiErrorKind } from "./http";
import { canRespondToEvidenceRequest, isEvidenceRequestAssignedToActor } from "./evidenceAuthorization";
import { parseRoute, routeHash } from "./appRouting";
import type { View, WorkspaceTarget, WorkTab } from "./appRouting";
import type { RuntimePresentation } from "./runtimePresentation";
import type { AIGovernancePolicy, AIGovernanceWorkload, AttentionItem, AutomationPolicy, AuthorityResolution, CaptureRequest, EvidenceRequest, EvidenceSource, GuideStep, IntegrityFinding, PolicySummary, Readiness, WorkflowTask } from "./types";
import type { ProjectionHealth, ReconcileResult } from "./operationsTypes";

const FormsWorkspace = lazy(() => import("./components/FormsWorkspace").then((module) => ({ default: module.FormsWorkspace })));
const VendorsWorkspace = lazy(() => import("./components/VendorsWorkspace").then((module) => ({ default: module.VendorsWorkspace })));

type LoadState = "idle" | "loading" | "live" | "unavailable";
type SectionLoadState = Exclude<LoadState, "idle">;
type ConnectionState = "loading" | "live" | "sample" | "unavailable";
type PrimaryEvidenceLoad = { targetID?: string; state: "idle" | "loading" | "live" | "unavailable" };
type VendorGuideIntent = { id: number; type: "open-vendor-due-diligence" | "open-vendor-work" | "open-vendor-next-action" };
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
  const captureLoadID = useRef(0);
  const [activePanel, setActivePanel] = useState<"none" | "routing" | "capture">("none");
  const [evidenceByID, setEvidenceByID] = useState<Record<string, EvidenceRequest>>({});
  const [evidenceWorkspaceIDs, setEvidenceWorkspaceIDs] = useState<string[]>([]);
  const [evidenceRequestState, setEvidenceRequestState] = useState<LoadState>("idle");
  const [evidenceSources, setEvidenceSources] = useState<EvidenceSource[]>([]);
  const [evidenceSourceState, setEvidenceSourceState] = useState<LoadState>("idle");
  const evidenceScopeEpoch = useRef(0);
  const [primaryEvidenceLoad, setPrimaryEvidenceLoad] = useState<PrimaryEvidenceLoad>({ state: "idle" });
  const [readiness, setReadiness] = useState<Readiness | null>(null);
  const [integrity, setIntegrity] = useState<IntegrityFinding[]>([]);
  const [integrityState, setIntegrityState] = useState<SectionLoadState>("loading");
  const [policies, setPolicies] = useState<PolicySummary[]>([]);
  const [policyState, setPolicyState] = useState<SectionLoadState>("loading");
  const [tasks, setTasks] = useState<WorkflowTask[]>([]);
  const [taskState, setTaskState] = useState<SectionLoadState>("loading");
  const [projectionHealth, setProjectionHealth] = useState<ProjectionHealth | null>(null);
  const [projectionState, setProjectionState] = useState<SectionLoadState>("loading");
  const [automationPolicies, setAutomationPolicies] = useState<AutomationPolicy[]>([]);
  const [automationPolicyState, setAutomationPolicyState] = useState<SectionLoadState>("loading");
  const [aiGovernancePolicies, setAIGovernancePolicies] = useState<AIGovernancePolicy[]>([]);
  const [aiGovernancePolicyState, setAIGovernancePolicyState] = useState<SectionLoadState>("loading");
  const [aiGovernanceWorkloads, setAIGovernanceWorkloads] = useState<AIGovernanceWorkload[]>([]);
  const [aiGovernanceWorkloadState, setAIGovernanceWorkloadState] = useState<SectionLoadState>("loading");
  const [configureState, setConfigureState] = useState<LoadState>("idle");

  const demoMode = runtime?.demo_mode === true || sampleMode;
  const serverDemoMode = runtime?.demo_mode === true;
  const importsEnabled = runtime?.capabilities?.document_import === true;
  const configureEnabled = runtime?.capabilities?.config_read === true;
  const primaryEvidenceTargetID = evidenceWorkspaceIDs[0];

  useEffect(() => {
    let active = true;
    void loadContext().then((context) => { if (active) setRuntime(context as ProductRuntime); }).catch(() => { if (active) setRuntime(null); });
    return () => { active = false; };
  }, []);

  useEffect(() => {
    const sync = () => {
      const route = parseRoute(window.location.hash);
      setActiveView(route.view); setTarget(route.target); if (route.workTab) setWorkTab(route.workTab);
    };
    window.addEventListener("popstate", sync); window.addEventListener("hashchange", sync);
    return () => { window.removeEventListener("popstate", sync); window.removeEventListener("hashchange", sync); };
  }, []);

  useEffect(() => {
    if (!runtime) return;
    let active = true;
    void Promise.all([loadToday(), loadReadiness()]).then(([today, nextReadiness]) => {
      if (!active) return;
      setItems(today.items); setTodayGeneratedAt(today.generated_at); setConnection("live"); setReadiness(nextReadiness); setReadinessState("live");
    }).catch(() => {
      if (!active) return;
      if (sampleMode) { setItems(fallbackItems); setConnection("sample"); } else { setItems([]); setConnection("unavailable"); }
      setReadiness(null); setReadinessState("unavailable");
    });
    return () => { active = false; };
  }, [runtime]);

  useEffect(() => {
    evidenceScopeEpoch.current += 1;
    setEvidenceByID({});
    setEvidenceWorkspaceIDs([]);
    setEvidenceRequestState("idle");
    setEvidenceSources([]);
    setEvidenceSourceState("idle");
    setPrimaryEvidenceLoad({ state: "idle" });
  }, [runtime?.tenant.id, runtime?.legal_entity.id]);

  useEffect(() => {
    if (!runtime || primaryEvidenceLoad.state !== "idle") return;
    let active = true;
    const scopeEpoch = evidenceScopeEpoch.current;
    setPrimaryEvidenceLoad({ targetID: primaryEvidenceTargetID, state: "loading" });
    void loadEvidenceRequests(1).then((result) => {
      if (!active || scopeEpoch !== evidenceScopeEpoch.current) return;
      const request = result.items[0];
      if (request) updateEvidenceEntity(request, scopeEpoch);
      setPrimaryEvidenceLoad({ targetID: request?.id, state: request ? "live" : "unavailable" });
    }).catch(() => { if (active && scopeEpoch === evidenceScopeEpoch.current) setPrimaryEvidenceLoad({ state: "unavailable" }); });
    return () => { active = false; };
  }, [runtime, primaryEvidenceLoad.state, primaryEvidenceTargetID]);

  function updateEvidenceEntity(request: EvidenceRequest, scopeEpoch = evidenceScopeEpoch.current) {
    if (scopeEpoch !== evidenceScopeEpoch.current) return;
    setEvidenceByID((current) => ({ ...current, [request.id]: request }));
    setEvidenceWorkspaceIDs((current) => current.includes(request.id) ? current : [...current, request.id]);
  }

  async function loadEvidenceWorkspace(preferredID?: string) {
    if (!runtime) return [] as EvidenceRequest[];
    const scopeEpoch = evidenceScopeEpoch.current;
    setEvidenceRequestState("loading");
    setEvidenceSourceState("loading");
    const [requestResult, sourceResult] = await Promise.allSettled([loadEvidenceRequests(50), loadEvidenceSources(50)]);
    if (scopeEpoch !== evidenceScopeEpoch.current) return [] as EvidenceRequest[];
    let requests: EvidenceRequest[] = [];
    if (requestResult.status === "fulfilled") {
      requests = requestResult.value.items;
      setEvidenceByID((current) => ({ ...current, ...Object.fromEntries(requests.map((request) => [request.id, request])) }));
      setEvidenceWorkspaceIDs((current) => [...new Set([...current, ...requests.map((request) => request.id)])]);
      setEvidenceRequestState("live");
    } else {
      setEvidenceRequestState("unavailable");
    }
    if (sourceResult.status === "fulfilled") { setEvidenceSources(sourceResult.value); setEvidenceSourceState("live"); } else { setEvidenceSources([]); setEvidenceSourceState("unavailable"); }
    if (preferredID && !requests.some((request) => request.id === preferredID)) {
      const exact = await ensureEvidenceTarget(preferredID, scopeEpoch);
      if (exact) requests = [...requests, exact];
    }
    return requests;
  }

  async function ensureEvidenceTarget(id: string, scopeEpoch = evidenceScopeEpoch.current) {
    if (evidenceByID[id]) return evidenceByID[id];
    try {
      const request = await loadEvidenceRequest(id, "workspace_target");
      if (scopeEpoch !== evidenceScopeEpoch.current) return undefined;
      updateEvidenceEntity(request, scopeEpoch);
      return request;
    } catch {
      return undefined;
    }
  }

  async function loadConfigureWorkspace() {
    setConfigureState("loading");
    const [policyResult, integrityResult, taskResult, projectionResult, automationResult, aiPolicyResult, aiWorkloadResult] = await Promise.allSettled([
      loadPolicies(), loadIntegrity(), loadWorkflowTasks(), loadProjectionHealth(), loadAutomationPolicies(), loadAIGovernancePolicies(), loadAIGovernanceWorkloads(),
    ]);
    setPolicies(policyResult.status === "fulfilled" ? policyResult.value : []);
    setPolicyState(policyResult.status === "fulfilled" ? "live" : "unavailable");
    setIntegrity(integrityResult.status === "fulfilled" ? integrityResult.value : []);
    setIntegrityState(integrityResult.status === "fulfilled" ? "live" : "unavailable");
    setTasks(taskResult.status === "fulfilled" ? taskResult.value : []);
    setTaskState(taskResult.status === "fulfilled" ? "live" : "unavailable");
    setProjectionHealth(projectionResult.status === "fulfilled" ? projectionResult.value : null);
    setProjectionState(projectionResult.status === "fulfilled" ? "live" : "unavailable");
    setAutomationPolicies(automationResult.status === "fulfilled" ? automationResult.value : []);
    setAutomationPolicyState(automationResult.status === "fulfilled" ? "live" : "unavailable");
    setAIGovernancePolicies(aiPolicyResult.status === "fulfilled" ? aiPolicyResult.value : []);
    setAIGovernancePolicyState(aiPolicyResult.status === "fulfilled" ? "live" : "unavailable");
    setAIGovernanceWorkloads(aiWorkloadResult.status === "fulfilled" ? aiWorkloadResult.value : []);
    setAIGovernanceWorkloadState(aiWorkloadResult.status === "fulfilled" ? "live" : "unavailable");
    setConfigureState("live");
  }

  async function checkProgramStatusRecords() {
    setProjectionState("loading");
    try {
      const result: ReconcileResult = await reconcileProgramState();
      setProjectionHealth(result.health); setProjectionState("live");
    } catch {
      setProjectionState("unavailable");
    }
  }

  useEffect(() => {
    if (!runtime || activeView !== "work") return;
    if (evidenceRequestState === "idle" || evidenceSourceState === "idle") void loadEvidenceWorkspace(target.evidenceID);
    if (evidenceRequestState === "live" && target.evidenceID && !evidenceWorkspaceIDs.includes(target.evidenceID)) void ensureEvidenceTarget(target.evidenceID);
  }, [runtime, activeView, workTab, evidenceRequestState, evidenceSourceState, target.evidenceID, evidenceWorkspaceIDs]);

  useEffect(() => {
    if (activeView === "configure" && configureEnabled && configureState === "idle") void loadConfigureWorkspace();
  }, [activeView, configureEnabled, configureState]);

  const evidenceRequests = evidenceWorkspaceIDs.flatMap((id) => evidenceByID[id] ? [evidenceByID[id]] : []);
  const organizationName = runtime?.tenant.name || runtime?.tenant.id || "Organization unavailable";
  const legalEntityName = runtime?.legal_entity.name || runtime?.legal_entity.id || "Legal entity unavailable";
  const actorName = runtime?.actor.name || runtime?.actor.id || "User unavailable";
  const roleName = humanRole(runtime?.actor.role_codes?.[0]) || "Role not provided";
  const navigation: Array<{ label: string; view: View }> = [
    { label: "Today", view: "today" }, { label: "Programs", view: "programs" }, { label: "Forms", view: "forms" }, { label: "Vendors", view: "vendors" }, { label: "Work", view: "work" },
    ...(importsEnabled ? [{ label: "Imports", view: "imports" as View }] : []),
    ...(demoMode ? [{ label: "Explore", view: "explore" as View }] : []),
    ...(configureEnabled ? [{ label: "Configure", view: "configure" as View }] : []),
  ];

  function cancelVendorGuideIntent(reason: string) {
    const pending = vendorGuideAck.current;
    vendorGuideAck.current = undefined;
    setVendorGuideIntent(undefined);
    pending?.reject(new Error(reason));
  }

  useEffect(() => () => {
    const pending = vendorGuideAck.current;
    vendorGuideAck.current = undefined;
    pending?.reject(new Error("Vendor guide action was cancelled."));
  }, []);

  function navigate(view: View, nextTarget: WorkspaceTarget = {}, tab?: WorkTab) {
    if (view !== "vendors") cancelVendorGuideIntent("Vendor guide action was cancelled.");
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

  async function openCapture(requestID: string) {
    const loadID = ++captureLoadID.current;
    const scopeEpoch = evidenceScopeEpoch.current;
    setCaptureRequestID(requestID); setCapture(null); setCaptureState("loading"); setActivePanel("capture");
    try {
      const loaded = await loadEvidenceRequest(requestID, "capture_revalidation");
      if (scopeEpoch !== evidenceScopeEpoch.current || loadID !== captureLoadID.current) return;
      updateEvidenceEntity(loaded, scopeEpoch);
      if (!canRespondToEvidenceRequest(loaded, runtime?.actor.id)) {
        if (isEvidenceRequestAssignedToActor(loaded, runtime?.actor.id)) {
          setCapture(loaded); setCaptureState("live"); return;
        }
        setCapture(null); setCaptureState("forbidden"); return;
      }
      setCapture(loaded); setCaptureState("live");
    } catch (error) {
      if (scopeEpoch !== evidenceScopeEpoch.current || loadID !== captureLoadID.current) return;
      const kind = apiErrorKind(error);
      setCaptureState(kind === "forbidden" || kind === "unauthorized" ? "forbidden" : kind === "not_found" ? "not-found" : "unavailable");
    }
  }

  async function reloadCapture() { if (captureRequestID) await openCapture(captureRequestID); }
  async function openPrimaryEvidence() {
    if (!primaryEvidenceRequest) return;
    await openCapture(primaryEvidenceRequest.id);
  }

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
    if (vendorIntent === "open-vendor-due-diligence" || vendorIntent === "open-vendor-work" || vendorIntent === "open-vendor-next-action") {
      return new Promise<void>((resolve, reject) => {
        const previous = vendorGuideAck.current;
        vendorGuideAck.current = undefined;
        previous?.reject(new Error("Vendor guide action was replaced."));
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

  const primaryEvidenceEntity = primaryEvidenceLoad.state === "live" && primaryEvidenceLoad.targetID === primaryEvidenceTargetID && primaryEvidenceTargetID ? evidenceByID[primaryEvidenceTargetID] : undefined;
  const primaryEvidenceRequest = primaryEvidenceEntity && canRespondToEvidenceRequest(primaryEvidenceEntity, runtime?.actor.id) ? primaryEvidenceEntity : undefined;
  const canOpenEvidence = Boolean(primaryEvidenceRequest);

  function completeVendorGuideIntent(id: number) {
    if (vendorGuideAck.current?.id === id) {
      vendorGuideAck.current.resolve();
      vendorGuideAck.current = undefined;
    }
    setVendorGuideIntent((current) => current?.id === id ? undefined : current);
  }

  function failVendorGuideIntent(id: number) {
    if (vendorGuideAck.current?.id === id) {
      const pending = vendorGuideAck.current;
      vendorGuideAck.current = undefined;
      pending.reject(new Error("Vendor workspace could not be loaded."));
    }
    setVendorGuideIntent((current) => current?.id === id ? undefined : current);
  }

  return <div className="app-shell">
    <aside className="sidebar" aria-label="Primary navigation"><div className="brand-mark" aria-label="ClearSight">C</div><nav>{navigation.map(({ label, view }) => <button className={view === activeView ? "nav-item active" : "nav-item"} key={view} aria-current={view === activeView ? "page" : undefined} onClick={() => navigate(view)}><NavigationIcon view={view}/><b>{label}</b></button>)}</nav><div className="avatar" aria-label={`Signed in as ${actorName}`}>{initials(actorName)}</div></aside>
    <main>
      <div className="context-bar" aria-label="Active workspace context"><div><strong>{organizationName}</strong><span>{legalEntityName}</span></div><div className="context-role"><DisplayPreferencesMenu/><span>{roleName}</span>{demoMode ? <mark>Stakeholder demo</mark> : serverDemoMode && presentation === "live-preview" ? <mark>Live preview · Non-production</mark> : null}</div></div>
      {(activeView === "today" || activeView === "vendors") && <RoleAwareOnboarding runtime={runtime} surface={activeView === "vendors" ? "VENDORS" : "TODAY"} onStep={executeGuideStep}/>}
      {activeView === "today" && <TodayView organizationName={organizationName} items={items} connection={connection} generatedAt={todayGeneratedAt} readiness={readiness} readinessState={readinessState === "idle" ? "loading" : readinessState} onCapture={canOpenEvidence ? () => void openPrimaryEvidence() : undefined} onOpenItem={openAttention} onInspectAuthority={(item) => void inspectRouting(item)}/>} 
      {activeView === "programs" && <ProgramsView organizationName={organizationName} actorPrincipalID={runtime?.actor.id} canConfigureSources={runtime?.capabilities?.config_write === true} targetID={target.programID} openFirst={target.openFirstProgram} onOpenRequest={(id) => navigate("work", { evidenceID: id }, "evidence")}/>}
      {activeView === "forms" && <Suspense fallback={<div className="workspace-loading" aria-live="polite" aria-busy="true">Loading Forms…</div>}><FormsWorkspace organizationName={organizationName} legalEntityName={legalEntityName} appearanceScope={runtime?.legal_entity.id} targetID={target.formTemplateID} onTarget={(id) => navigate("forms", id ? { formTemplateID: id } : {})}/></Suspense>}
      {activeView === "vendors" && <Suspense fallback={<div className="workspace-loading" aria-live="polite" aria-busy="true">Loading vendor relationships…</div>}><VendorsWorkspace organizationName={organizationName} legalEntityName={legalEntityName} targetID={target.vendorRelationshipID} guideIntent={vendorGuideIntent} onGuideIntentCompleted={completeVendorGuideIntent} onGuideIntentFailed={failVendorGuideIntent} onTarget={(id) => navigate("vendors", id ? { vendorRelationshipID: id } : {})} onOpenRequest={(id) => navigate("work", { evidenceID: id }, "evidence")} onOpenMatter={(id) => navigate("work", { matterID: id }, "matters")}/></Suspense>}
      {activeView === "work" && <WorkView organizationName={organizationName} actorPrincipalID={runtime?.actor.id} evidenceScopeToken={evidenceScopeEpoch.current} tab={workTab} onTab={(tab) => navigate("work", {}, tab)} onBackMatter={() => navigate("work", {}, "matters")} sources={sources} requests={evidenceRequests} evidenceSourceState={evidenceSourceState === "idle" ? "loading" : evidenceSourceState} evidenceRequestState={evidenceRequestState === "idle" ? "loading" : evidenceRequestState} onEvidenceRetry={() => void loadEvidenceWorkspace(target.evidenceID)} onEvidenceRequestUpdated={updateEvidenceEntity} matterTargetID={target.matterID} openFirstMatter={target.openFirstMatter} evidenceTargetID={target.evidenceID} openFirstEvidence={target.openFirstEvidence} onOpenEvidence={(id) => void openCapture(id)}/>}
      {activeView === "imports" && importsEnabled && <><header className="topbar"><div><span className="eyebrow">{organizationName}</span><h1>Imports</h1><p>Compare regulatory documents with current Programs, controls and evidence.</p></div></header><DocumentImportWorkspace/></>}
      {activeView === "explore" && demoMode && <ExploreView organizationName={organizationName}/>}
      {activeView === "configure" && configureEnabled && <ConfigureView policies={policies} policyState={policyState} findings={integrity} integrityState={integrityState} tasks={tasks} taskState={taskState} projectionHealth={projectionHealth} projectionState={projectionState} canReconcileProjection={runtime?.capabilities?.platform_operations_write === true} automationPolicies={automationPolicies} automationPolicyState={automationPolicyState} aiGovernancePolicies={aiGovernancePolicies} aiGovernancePolicyState={aiGovernancePolicyState} aiGovernanceWorkloads={aiGovernanceWorkloads} aiGovernanceWorkloadState={aiGovernanceWorkloadState} state={configureState} onRetry={() => void loadConfigureWorkspace()} onReconcile={checkProgramStatusRecords}/>}
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
function evidenceRuntimeScopeKey(runtime: ProductRuntime | null) {
  if (!runtime) return undefined;
  return `${runtime.tenant.id}\u0000${runtime.legal_entity.id}\u0000${runtime.actor.id}`;
}

export default App;
