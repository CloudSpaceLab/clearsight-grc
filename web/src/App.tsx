import { lazy, Suspense, useEffect, useRef, useState } from "react";
import "./document-import.css";
import "./monitoring.css";
import "./program-record.css";
import {
  loadContext,
  loadEvidenceRequest,
  loadEvidenceRequests,
  loadEvidenceSources,
  loadReadiness,
  loadToday,
  resolveAuthority,
} from "./api";
import type { RuntimeContext } from "./api";
import { CapturePanel, ProgramsView, ReferenceJourneysView, RoutingPanel, TodayView, WorkView, type RoutingLoadState } from "./AppViews";
import { AdministrationMenu } from "./components/AdministrationMenu";
import { DemoEnvironmentMenu } from "./components/DemoEnvironmentMenu";
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
import type { AttentionItem, AuthorityResolution, CaptureRequest, EvidenceRequest, EvidenceSource, GuideStep, Readiness } from "./types";

const FormsWorkspace = lazy(() => import("./components/FormsWorkspace").then((module) => ({ default: module.FormsWorkspace })));
const VendorsWorkspace = lazy(() => import("./components/VendorsWorkspace").then((module) => ({ default: module.VendorsWorkspace })));
const ConfigureWorkspace = lazy(() => import("./components/configure/ConfigureWorkspace").then((module) => ({ default: module.ConfigureWorkspace })));
const OversightWorkspace = lazy(() => import("./components/oversight/OversightWorkspace").then((module) => ({ default: module.OversightWorkspace })));

type LoadState = "idle" | "loading" | "live" | "unavailable";
type ConnectionState = "loading" | "live" | "unavailable";
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
    oversight_read?: boolean;
  };
  actor: RuntimeContext["actor"] & { role_codes?: string[] };
};

function App({ presentation = "enterprise" }: { presentation?: RuntimePresentation }) {
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
  const [sources, setSources] = useState<EvidenceSource[]>([]);
  const [evidenceByID, setEvidenceByID] = useState<Record<string, EvidenceRequest>>({});
  const [evidenceWorkspaceIDs, setEvidenceWorkspaceIDs] = useState<string[]>([]);
  const [primaryEvidenceLoad, setPrimaryEvidenceLoad] = useState<PrimaryEvidenceLoad>({ state: "idle" });
  const [evidenceSourceState, setEvidenceSourceState] = useState<LoadState>("idle");
  const [evidenceRequestState, setEvidenceRequestState] = useState<LoadState>("idle");
  const [activePanel, setActivePanel] = useState<"none" | "routing" | "capture">("none");
  const captureLoadID = useRef(0);
  const routingLoadID = useRef(0);
  const evidenceTargetAttempts = useRef(new Set<string>());
  const evidenceWorkspaceLoadID = useRef(0);
  const primaryEvidenceLoadID = useRef(0);
  const evidenceScopeEpoch = useRef(0);
  const evidenceScopeKey = useRef<string | undefined>(undefined);
  const evidenceRouteTargetID = useRef<string | undefined>(target.evidenceID);
  evidenceRouteTargetID.current = target.evidenceID;

  const serverDemoMode = runtime?.demo_mode === true;
  const demoMode = serverDemoMode && presentation === "demo";
  const importsEnabled = runtime != null && runtime.capabilities?.document_import !== false;
  const configureEnabled = runtime != null && runtime.capabilities?.config_read !== false;
  const referenceJourneysEnabled = demoMode && runtime?.capabilities?.reference_journeys !== false;
  const oversightEnabled = runtime?.capabilities?.oversight_read === true;

  useEffect(() => {
    void Promise.allSettled([loadContext(), loadToday(), loadReadiness()]).then(([contextResult, todayResult, readinessResult]) => {
      const currentRuntime = contextResult.status === "fulfilled" ? contextResult.value as ProductRuntime : null;
      applyVerifiedRuntime(currentRuntime);
      if (todayResult.status === "fulfilled") {
        setItems(Array.isArray(todayResult.value.items) ? todayResult.value.items : []);
        setTodayGeneratedAt(todayResult.value.generated_at);
        setConnection("live");
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

  const primaryEvidenceTargetID = items.find((item) => item.action_target_type === "EVIDENCE_REQUEST" && item.action_target_id)?.action_target_id;
  const currentEvidenceScopeKey = evidenceRuntimeScopeKey(runtime);

  useEffect(() => {
    const loadID = ++primaryEvidenceLoadID.current;
    const scopeEpoch = evidenceScopeEpoch.current;
    if (!runtime?.actor.id || !primaryEvidenceTargetID) {
      setPrimaryEvidenceLoad({ state: "idle" });
      return;
    }
    setPrimaryEvidenceLoad({ targetID: primaryEvidenceTargetID, state: "loading" });
    void loadEvidenceRequest(primaryEvidenceTargetID, "eligibility_preload").then((request) => {
      if (scopeEpoch !== evidenceScopeEpoch.current || loadID !== primaryEvidenceLoadID.current) return;
      updateEvidenceEntity(request, scopeEpoch);
      setPrimaryEvidenceLoad({ targetID: primaryEvidenceTargetID, state: "live" });
    }).catch(() => {
      if (scopeEpoch !== evidenceScopeEpoch.current || loadID !== primaryEvidenceLoadID.current) return;
      setPrimaryEvidenceLoad({ targetID: primaryEvidenceTargetID, state: "unavailable" });
    });
    return () => { if (loadID === primaryEvidenceLoadID.current) primaryEvidenceLoadID.current++; };
  }, [currentEvidenceScopeKey, primaryEvidenceTargetID]);

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
    if ((!referenceJourneysEnabled && activeView === "explore") || (!importsEnabled && activeView === "imports") || (!configureEnabled && activeView === "configure") || (!oversightEnabled && activeView === "oversight")) navigate("today");
  }, [runtime, referenceJourneysEnabled, importsEnabled, configureEnabled, oversightEnabled, activeView]);

  async function loadEvidenceWorkspace(requestedID?: string) {
    const loadID = ++evidenceWorkspaceLoadID.current;
    const scopeEpoch = evidenceScopeEpoch.current;
    setEvidenceSourceState("loading");
    setEvidenceRequestState("loading");
    const [sourcesResult, requestsResult] = await Promise.allSettled([loadEvidenceSources(), loadEvidenceRequests()]);
    if (scopeEpoch !== evidenceScopeEpoch.current || loadID !== evidenceWorkspaceLoadID.current) return [];
    if (sourcesResult.status === "fulfilled") { setSources(sourcesResult.value); setEvidenceSourceState("live"); } else { setSources([]); setEvidenceSourceState("unavailable"); }
    if (requestsResult.status === "fulfilled") {
      let requests = requestsResult.value;
      if (requestedID && !requests.some((request) => request.id === requestedID)) {
        evidenceTargetAttempts.current.add(requestedID);
        const requested = await loadEvidenceRequest(requestedID).catch(() => null);
        if (scopeEpoch !== evidenceScopeEpoch.current || loadID !== evidenceWorkspaceLoadID.current) return [];
        if (requested) requests = [requested, ...requests];
      }
      mergeEvidenceEntities(requests, scopeEpoch);
      setEvidenceWorkspaceIDs(requests.map((request) => request.id));
      setEvidenceRequestState("live");
      return requests;
    }
    setEvidenceRequestState("unavailable");
    return [];
  }

  async function ensureEvidenceTarget(requestedID: string) {
    if (evidenceTargetAttempts.current.has(requestedID)) return;
    evidenceTargetAttempts.current.add(requestedID);
    const scopeEpoch = evidenceScopeEpoch.current;
    const requested = await loadEvidenceRequest(requestedID).catch(() => null);
    if (scopeEpoch !== evidenceScopeEpoch.current || evidenceRouteTargetID.current !== requestedID) return;
    if (requested) {
      updateEvidenceEntity(requested, scopeEpoch);
      setEvidenceWorkspaceIDs((current) => current.includes(requestedID) ? current : [requestedID, ...current]);
    }
  }

  function updateEvidenceEntity(updated: EvidenceRequest, scopeEpoch = evidenceScopeEpoch.current) {
    if (scopeEpoch !== evidenceScopeEpoch.current) return false;
    setEvidenceByID((current) => {
      const existing = current[updated.id];
      if (existing && existing.version > updated.version) return current;
      return { ...current, [updated.id]: updated };
    });
    return true;
  }

  function mergeEvidenceEntities(requests: EvidenceRequest[], scopeEpoch: number) {
    if (scopeEpoch !== evidenceScopeEpoch.current) return false;
    setEvidenceByID((current) => {
      let changed = false;
      const next = { ...current };
      for (const request of requests) {
        const existing = next[request.id];
        if (!existing || request.version >= existing.version) {
          next[request.id] = request;
          changed = true;
        }
      }
      return changed ? next : current;
    });
    return true;
  }

  function applyVerifiedRuntime(nextRuntime: ProductRuntime | null) {
    const nextScopeKey = evidenceRuntimeScopeKey(nextRuntime);
    if (evidenceScopeKey.current !== nextScopeKey) {
      evidenceScopeEpoch.current++;
      evidenceScopeKey.current = nextScopeKey;
      evidenceWorkspaceLoadID.current++;
      primaryEvidenceLoadID.current++;
      captureLoadID.current++;
      evidenceTargetAttempts.current.clear();
      setEvidenceByID({});
      setEvidenceWorkspaceIDs([]);
      setPrimaryEvidenceLoad({ state: "idle" });
      setSources([]);
      setEvidenceSourceState("idle");
      setEvidenceRequestState("idle");
      setActivePanel("none");
    }
    setRuntime(nextRuntime);
  }

  useEffect(() => {
    if (!runtime || activeView !== "work" || workTab !== "evidence") return;
    if (evidenceRequestState === "idle" || evidenceSourceState === "idle") { void loadEvidenceWorkspace(target.evidenceID); return; }
    if (evidenceRequestState === "live" && target.evidenceID && !evidenceWorkspaceIDs.includes(target.evidenceID)) void ensureEvidenceTarget(target.evidenceID);
  }, [runtime, activeView, workTab, evidenceRequestState, evidenceSourceState, target.evidenceID, evidenceWorkspaceIDs]);

  const evidenceRequests = evidenceWorkspaceIDs.flatMap((id) => evidenceByID[id] ? [evidenceByID[id]] : []);
  const organizationName = runtime?.tenant.name || runtime?.tenant.id || "Organization unavailable";
  const legalEntityName = runtime?.legal_entity.name || runtime?.legal_entity.id || "Legal entity unavailable";
  const actorName = runtime?.actor.name || runtime?.actor.id || "User unavailable";
  const roleName = humanRole(runtime?.actor.role_codes?.[0]) || "Role not provided";
  const operatingNavigation: Array<{ label: string; view: View }> = [
    { label: "Today", view: "today" },
    ...(oversightEnabled ? [{ label: "Oversight", view: "oversight" as View }] : []),
    { label: "Programs", view: "programs" },
    { label: "Work", view: "work" },
    { label: "Vendors", view: "vendors" },
    { label: "Forms", view: "forms" },
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
    if (!item.authority || !isAuthorityObjectType(item.action_target_type) || !item.action_target_id) return;
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
    <aside className="sidebar" aria-label="Primary navigation">
      <div className="brand-mark" aria-label="ClearSight">C</div>
      <nav>{operatingNavigation.map(({ label, view }) => <button className={view === activeView ? "nav-item active" : "nav-item"} key={view} aria-current={view === activeView ? "page" : undefined} onClick={() => navigate(view)}><NavigationIcon view={view}/><b>{label}</b></button>)}</nav>
      {configureEnabled && <div className="sidebar-secondary"><button className={activeView === "configure" ? "nav-item active" : "nav-item"} type="button" aria-current={activeView === "configure" ? "page" : undefined} onClick={() => navigate("configure")}><NavigationIcon view="configure"/><b>Configure</b></button></div>}
      <div className="avatar" aria-label={`Signed in as ${actorName}`}>{initials(actorName)}</div>
    </aside>
    <main>
      <div className="context-bar" aria-label="Active workspace context">
        <div><strong>{organizationName}</strong><span>{legalEntityName}</span></div>
        <div className="context-role">
          <DisplayPreferencesMenu/>
          <AdministrationMenu enabled={configureEnabled} onOpen={() => navigate("configure")}/>
          {serverDemoMode && <DemoEnvironmentMenu onOpenReferenceJourneys={referenceJourneysEnabled ? () => navigate("explore") : undefined}/>}
          <span>{roleName}</span>
          {serverDemoMode ? <mark>{demoMode ? "Stakeholder demo" : "Non-production data"}</mark> : null}
        </div>
      </div>
      {(activeView === "today" || activeView === "vendors") && <RoleAwareOnboarding runtime={runtime} surface={activeView === "vendors" ? "VENDORS" : "TODAY"} onStep={executeGuideStep}/>}
      {activeView === "today" && <TodayView organizationName={organizationName} items={items} connection={connection} generatedAt={todayGeneratedAt} readiness={readiness} readinessState={readinessState === "idle" ? "loading" : readinessState} onCapture={canOpenEvidence ? () => void openPrimaryEvidence() : undefined} onOpenItem={openAttention} onInspectAuthority={(item) => void inspectRouting(item)}/>} 
      {activeView === "oversight" && oversightEnabled && <Suspense fallback={<div className="workspace-loading" aria-live="polite" aria-busy="true">Loading oversight…</div>}><OversightWorkspace organizationName={organizationName} legalEntityName={legalEntityName} onOpenMatter={(id) => navigate("work", { matterID: id }, "matters")}/></Suspense>}
      {activeView === "programs" && <ProgramsView organizationName={organizationName} actorPrincipalID={runtime?.actor.id} canConfigureSources={runtime?.capabilities?.config_write === true} targetID={target.programID} openFirst={target.openFirstProgram} onOpenRequest={(id) => navigate("work", { evidenceID: id }, "evidence")} onAnalyzeDocument={importsEnabled ? () => navigate("imports") : undefined}/>}
      {activeView === "work" && <WorkView organizationName={organizationName} actorPrincipalID={runtime?.actor.id} evidenceScopeToken={evidenceScopeEpoch.current} tab={workTab} onTab={(tab) => navigate("work", {}, tab)} onBackMatter={() => navigate("work", {}, "matters")} sources={sources} requests={evidenceRequests} evidenceSourceState={evidenceSourceState === "idle" ? "loading" : evidenceSourceState} evidenceRequestState={evidenceRequestState === "idle" ? "loading" : evidenceRequestState} onEvidenceRetry={() => void loadEvidenceWorkspace(target.evidenceID)} onEvidenceRequestUpdated={updateEvidenceEntity} matterTargetID={target.matterID} openFirstMatter={target.openFirstMatter} evidenceTargetID={target.evidenceID} openFirstEvidence={target.openFirstEvidence} onOpenEvidence={(id) => void openCapture(id)} onAnalyzeDocument={importsEnabled ? () => navigate("imports") : undefined}/>}
      {activeView === "vendors" && <Suspense fallback={<div className="workspace-loading" aria-live="polite" aria-busy="true">Loading vendor relationships…</div>}><VendorsWorkspace organizationName={organizationName} legalEntityName={legalEntityName} targetID={target.vendorRelationshipID} guideIntent={vendorGuideIntent} onGuideIntentCompleted={completeVendorGuideIntent} onGuideIntentFailed={failVendorGuideIntent} onTarget={(id) => navigate("vendors", id ? { vendorRelationshipID: id } : {})} onOpenRequest={(id) => navigate("work", { evidenceID: id }, "evidence")} onOpenMatter={(id) => navigate("work", { matterID: id }, "matters")} onOpenForms={() => navigate("forms")}/></Suspense>}
      {activeView === "forms" && <Suspense fallback={<div className="workspace-loading" aria-live="polite" aria-busy="true">Loading Forms…</div>}><FormsWorkspace organizationName={organizationName} legalEntityName={legalEntityName} canConfigureCommunications={runtime?.capabilities?.config_write === true} targetID={target.formTemplateID} onTarget={(id) => navigate("forms", id ? { formTemplateID: id } : {})}/></Suspense>}
      {activeView === "imports" && importsEnabled && <><header className="topbar"><div><span className="eyebrow">{organizationName}</span><h1>Imports</h1><p>Import documents, compare their obligations with current Programs, controls and evidence, then review proposed updates.</p></div></header><DocumentImportWorkspace/></>}
      {activeView === "explore" && referenceJourneysEnabled && <ReferenceJourneysView organizationName={organizationName}/>}
      {activeView === "configure" && configureEnabled && <Suspense fallback={<div className="workspace-loading" aria-live="polite" aria-busy="true">Loading Configuration…</div>}><ConfigureWorkspace importsEnabled={importsEnabled} canReconcileProjection={runtime?.capabilities?.platform_operations_write === true} onOpenImports={() => navigate("imports")}/></Suspense>}
      <div id="cs-overlay-root" className="cs-overlay-root" aria-live="off"/>
    </main>
    <nav className="mobile-nav" aria-label="Mobile navigation">{operatingNavigation.map(({ label, view }) => <button key={view} type="button" aria-current={activeView === view ? "page" : undefined} onClick={() => navigate(view)}><NavigationIcon view={view}/><span>{label}</span></button>)}</nav>
    {activePanel !== "none" && <FocusedSheet label={activePanel === "routing" ? "Authority for selected work" : "Evidence request"} onClose={closePanel}>{activePanel === "routing" ? <RoutingPanel resolution={resolution} item={routingItem} legalEntityName={legalEntityName} state={routingState}/> : <CapturePanel request={capture} state={captureState} onReload={() => void reloadCapture()}/>}</FocusedSheet>}
  </div>;
}

function isAuthorityObjectType(value: AttentionItem["action_target_type"]): value is "PROGRAM" | "MATTER" | "EVIDENCE_REQUEST" {
  return value === "PROGRAM" || value === "MATTER" || value === "EVIDENCE_REQUEST";
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
