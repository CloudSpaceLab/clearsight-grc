from pathlib import Path
import re


def replace_once(text: str, old: str, new: str, label: str) -> str:
    if old not in text:
        raise SystemExit(f"{label} anchor changed")
    return text.replace(old, new, 1)


# Register summary routes.
p = Path("internal/httpapi/server.go")
s = p.read_text()
program_route = '\tmux.HandleFunc("GET /api/v1/programs", api.listPrograms)'
matter_route = '\tmux.HandleFunc("GET /api/v1/matters", api.listMatters)'
if 'GET /api/v1/program-summaries' not in s:
    s = replace_once(s, program_route, '\tmux.HandleFunc("GET /api/v1/program-summaries", api.listProgramSummaries)\n' + program_route, "program route")
if 'GET /api/v1/matter-summaries' not in s:
    s = replace_once(s, matter_route, '\tmux.HandleFunc("GET /api/v1/matter-summaries", api.listMatterSummaries)\n' + matter_route, "matter route")
p.write_text(s)

# Use indexed generated search documents.
p = Path("internal/continuity/summaries_postgres.go")
s = p.read_text()
s = s.replace(
    "($3='' OR p.code ILIKE '%' || $3 || '%' OR p.name ILIKE '%' || $3 || '%' OR p.owning_function ILIKE '%' || $3 || '%' OR p.jurisdiction ILIKE '%' || $3 || '%')",
    "($3='' OR p.search_document @@ websearch_to_tsquery('simple'::regconfig,$3))",
)
s = s.replace(
    "($3='' OR m.reference ILIKE '%' || $3 || '%' OR m.title ILIKE '%' || $3 || '%' OR m.summary ILIKE '%' || $3 || '%' OR m.matter_type ILIKE '%' || $3 || '%')",
    "($3='' OR m.search_document @@ websearch_to_tsquery('simple'::regconfig,$3))",
)
p.write_text(s)

# Make non-visible workspaces load only when opened.
p = Path("web/src/App.tsx")
s = p.read_text()
s = s.replace("loadIntegrity, loadMatters, loadOnboardingGuide", "loadIntegrity, loadOnboardingGuide")
s = s.replace("loadPolicies, loadPrograms, loadReadiness", "loadPolicies, loadReadiness")
s = s.replace("IntegrityFinding, MatterAggregate, OnboardingGuide", "IntegrityFinding, OnboardingGuide")
s = s.replace("PolicySummary, ProgramAggregate, Readiness", "PolicySummary, Readiness")
s = s.replace('type LoadState = "loading" | "live" | "unavailable";', 'type LoadState = "idle" | "loading" | "live" | "unavailable";')
s = replace_once(
    s,
    '  const [tasks, setTasks] = useState<WorkflowTask[]>([]);',
    '  const [tasks, setTasks] = useState<WorkflowTask[]>([]);\n  const [configureState, setConfigureState] = useState<LoadState>("idle");',
    "configure state",
)
s = replace_once(
    s,
    '  const [evidenceState, setEvidenceState] = useState<LoadState>("loading");\n  const [programs, setPrograms] = useState<ProgramAggregate[]>([]);\n  const [programState, setProgramState] = useState<LoadState>("loading");\n  const [matters, setMatters] = useState<MatterAggregate[]>([]);\n  const [matterState, setMatterState] = useState<LoadState>("loading");',
    '  const [evidenceState, setEvidenceState] = useState<LoadState>("idle");',
    "eager workspace state",
)

first_effect = s.index("  useEffect(() => {")
due_marker = s.index("\n\n  const dueSoon", first_effect)
replacement = '''  useEffect(() => {
    Promise.allSettled([loadToday(), loadReadiness(), loadOnboardingGuide(), loadOnboardingState()]).then((results) => {
      const [todayResult, readinessResult, guideResult, stateResult] = results;
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
        tenant_id: "bank-demo",
        principal_id: "user-demo",
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
    const [policiesResult, integrityResult, tasksResult] = await Promise.allSettled([loadPolicies(), loadIntegrity(), loadWorkflowTasks()]);
    if (policiesResult.status === "fulfilled" && integrityResult.status === "fulfilled" && tasksResult.status === "fulfilled") {
      setPolicies(policiesResult.value);
      setIntegrity(integrityResult.value);
      setTasks(tasksResult.value);
      setConfigureState("live");
    } else {
      setConfigureState("unavailable");
    }
  }

  useEffect(() => {
    if (activeView === "work" && workTab === "evidence" && evidenceState === "idle") void loadEvidenceWorkspace();
  }, [activeView, workTab, evidenceState]);

  useEffect(() => {
    if (activeView === "configure" && configureState === "idle") void loadConfigureWorkspace();
  }, [activeView, configureState]);'''
s = s[:first_effect] + replacement + s[due_marker:]

s = replace_once(
    s,
    '{activeView === "today" && <TodayView items={items} dueSoon={dueSoon} connection={connection} readiness={readiness} readinessState={readinessState} onRouting={inspectRouting} onCapture={openCapture}/>} ',
    '{activeView === "today" && <TodayView items={items} dueSoon={dueSoon} connection={connection} readiness={readiness} readinessState={readinessState === "idle" ? "loading" : readinessState} onRouting={inspectRouting} onCapture={openCapture}/>} ',
    "Today view render",
)
s = replace_once(
    s,
    '{activeView === "programs" && <ProgramsView programs={programs} state={programState}/>} ',
    '{activeView === "programs" && <ProgramsView/>} ',
    "Programs view render",
)
s = replace_once(
    s,
    '{activeView === "work" && <WorkView tab={workTab} onTab={setWorkTab} matters={matters} matterState={matterState} sources={sources} requests={evidenceRequests} evidenceState={evidenceState}/>} ',
    '{activeView === "work" && <WorkView tab={workTab} onTab={setWorkTab} sources={sources} requests={evidenceRequests} evidenceState={evidenceState} onEvidenceRetry={() => void loadEvidenceWorkspace()}/>} ',
    "Work view render",
)
s = replace_once(
    s,
    '{activeView === "configure" && <ConfigureView policies={policies} findings={integrity} tasks={tasks}/>} ',
    '{activeView === "configure" && <ConfigureView policies={policies} findings={integrity} tasks={tasks} state={configureState} onRetry={() => void loadConfigureWorkspace()}/>} ',
    "Configure view render",
)

s, programs_count = re.subn(
    r'function ProgramsView\(\{ programs, state \}: \{ programs: ProgramAggregate\[\]; state: LoadState \}\) \{.*?\n\}',
    '''function ProgramsView() {
  return <><header className="topbar"><div><span className="eyebrow">Demonstration Bank Nigeria</span><h1>Programs</h1><p>Ongoing obligations, safeguards, evidence checks and open issues.</p></div></header><ProgramsWorkspace/></>;
}''',
    s,
    count=1,
    flags=re.S,
)
if programs_count != 1:
    raise SystemExit("ProgramsView declaration changed")

s, work_count = re.subn(
    r'function WorkView\(\{ tab, onTab, matters, matterState, sources, requests, evidenceState \}: \{.*?\n\}',
    '''function WorkView({ tab, onTab, sources, requests, evidenceState, onEvidenceRetry }: { tab: "matters" | "evidence"; onTab: (value: "matters" | "evidence") => void; sources: EvidenceSource[]; requests: EvidenceRequest[]; evidenceState: LoadState; onEvidenceRetry: () => void }) {
  const evidenceLoadState = evidenceState === "idle" ? "loading" : evidenceState;
  return <><header className="topbar"><div><span className="eyebrow">Demonstration Bank Nigeria</span><h1>Work</h1><p>Issues, changes, evidence requests and the sources they rely on.</p></div></header><div className="workspace-tabs" role="tablist" aria-label="Work views"><button type="button" role="tab" aria-selected={tab === "matters"} className={tab === "matters" ? "active" : ""} onClick={() => onTab("matters")}>Issues and changes</button><button type="button" role="tab" aria-selected={tab === "evidence"} className={tab === "evidence" ? "active" : ""} onClick={() => onTab("evidence")}>Sources and evidence</button></div>{tab === "matters" ? <MattersWorkspace/> : evidenceLoadState === "unavailable" ? <EmptyState label="Sources and evidence" title="Sources and evidence could not be loaded" description="The service is unavailable. No source-health or request totals are shown." action="Try again" onAction={onEvidenceRetry}/> : <EvidenceWorkspace sources={sources} requests={requests} state={evidenceLoadState}/>}</>;
}''',
    s,
    count=1,
    flags=re.S,
)
if work_count != 1:
    raise SystemExit("WorkView declaration changed")

configure_decl = 'function ConfigureView({ policies, findings, tasks }: { policies: PolicySummary[]; findings: IntegrityFinding[]; tasks: WorkflowTask[] }) {'
configure_replacement = '''function ConfigureView({ policies, findings, tasks, state, onRetry }: { policies: PolicySummary[]; findings: IntegrityFinding[]; tasks: WorkflowTask[]; state: LoadState; onRetry: () => void }) {
  if (state === "idle" || state === "loading") return <><header className="topbar"><div><span className="eyebrow">Governance configuration</span><h1>Routing and approvals</h1><p>Responsibility, approval limits, delegation and escalation rules.</p></div></header><section className="workspace-loading">Loading routing configuration…</section></>;
  if (state === "unavailable") return <><header className="topbar"><div><span className="eyebrow">Governance configuration</span><h1>Routing and approvals</h1><p>Responsibility, approval limits, delegation and escalation rules.</p></div></header><EmptyState label="Routing and approvals" title="Routing configuration could not be loaded" description="The service is unavailable. No policy or integrity claims are shown." action="Try again" onAction={onRetry}/></>;'''
s = replace_once(s, configure_decl, configure_replacement, "ConfigureView declaration")
p.write_text(s)

# Styling for search, retry, and pagination states.
p = Path("web/src/continuity.css")
s = p.read_text()
if ".workspace-toolbar" not in s:
    s += '''

.workspace-toolbar { display:grid; grid-template-columns:minmax(240px,1fr) 190px auto; gap:12px; align-items:end; margin:18px 0; padding:16px; border:1px solid var(--border); border-radius:14px; background:var(--surface); }
.workspace-toolbar label { display:grid; gap:6px; color:var(--muted); font-size:12px; }
.workspace-toolbar input, .workspace-toolbar select { min-height:42px; border:1px solid var(--border); border-radius:10px; background:var(--surface-2); color:var(--text); padding:0 12px; font:inherit; }
.workspace-toolbar input:focus, .workspace-toolbar select:focus { outline:2px solid var(--cyan); outline-offset:2px; }
.load-more { display:flex; justify-content:center; padding:20px 0 8px; }
.inline-error { display:flex; align-items:center; justify-content:space-between; gap:16px; padding:12px; border:1px solid color-mix(in srgb, var(--coral) 45%, var(--border)); border-radius:10px; }
.workspace-loading { min-height:220px; display:grid; place-items:center; color:var(--muted); border:1px solid var(--border); border-radius:16px; background:var(--surface); }
@media (max-width: 800px) { .workspace-toolbar { grid-template-columns:1fr; } .workspace-toolbar .secondary-button { width:100%; } }
'''
p.write_text(s)

# Formal API contract.
p = Path("api/openapi.yaml")
s = p.read_text().replace("  version: 0.5.0", "  version: 0.6.0")
anchor = "  /api/v1/programs:\n"
if "/api/v1/program-summaries:" not in s:
    block = '''  /api/v1/program-summaries:
    get:
      summary: List bounded Program summaries
      parameters:
        - { $ref: '#/components/parameters/Tenant' }
        - { name: q, in: query, schema: { type: string } }
        - { name: status, in: query, schema: { type: string } }
        - { name: cursor, in: query, schema: { type: string } }
        - { name: limit, in: query, schema: { type: integer, minimum: 1, maximum: 100, default: 20 } }
      responses:
        '200': { description: Keyset-paginated Program summaries }
        '400': { description: Invalid cursor }
  /api/v1/matter-summaries:
    get:
      summary: List bounded issue and change summaries
      parameters:
        - { $ref: '#/components/parameters/Tenant' }
        - { name: q, in: query, schema: { type: string } }
        - { name: status, in: query, schema: { type: string, default: OPEN } }
        - { name: cursor, in: query, schema: { type: string } }
        - { name: limit, in: query, schema: { type: integer, minimum: 1, maximum: 100, default: 20 } }
      responses:
        '200': { description: Keyset-paginated issue and change summaries }
        '400': { description: Invalid cursor }

'''
    s = replace_once(s, anchor, block + anchor, "OpenAPI Program path")
p.write_text(s)

# Documentation status.
p = Path("docs/implementation-plan.md")
s = p.read_text()
s = s.replace(
    "- [ ] projection-first high-cardinality list/read model and performance baselines;",
    "- [x] projection-first Program and Matter summaries, indexed search, keyset pagination and lazy detail loading;\n- [ ] representative 100,000-row p95/p99 release evidence and query-plan retention;",
)
if "projection-first Program and Matter summaries" not in s:
    s += "\n- [x] Projection-first Program and Matter summaries, indexed search, keyset pagination and lazy detail loading.\n- [ ] Representative 100,000-row p95/p99 release evidence and retained query plans.\n"
p.write_text(s)

p = Path("docs/README.md")
s = p.read_text()
if "architecture/operational-read-models.md" not in s:
    s = s.replace(
        "8. [`product/respond-and-capture.md`]",
        "8. [`architecture/operational-read-models.md`](architecture/operational-read-models.md) — bounded list projections, search, pagination and lazy detail.\n9. [`product/respond-and-capture.md`]",
    )
p.write_text(s)
