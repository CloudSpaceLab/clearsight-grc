from pathlib import Path


def replace_once(text: str, old: str, new: str, label: str) -> str:
    if old not in text:
        raise SystemExit(f"{label} anchor changed")
    return text.replace(old, new, 1)

p = Path("web/src/App.tsx")
s = p.read_text()
s = replace_once(
    s,
    'import { loadCaptureRequest, loadEvidenceRequests, loadEvidenceSources, loadIntegrity, loadOnboardingGuide, loadOnboardingState, loadPolicies, loadReadiness, loadToday, loadWorkflowTasks, resolveAuthority, saveOnboardingState, submitCaptureRequest } from "./api";',
    'import { loadCaptureRequest, loadEvidenceRequests, loadEvidenceSources, loadIntegrity, loadOnboardingGuide, loadOnboardingState, loadPolicies, loadProjectionHealth, loadReadiness, loadToday, loadWorkflowTasks, reconcileProgramState, resolveAuthority, saveOnboardingState, submitCaptureRequest } from "./api";',
    "API imports",
)
s = replace_once(
    s,
    'import { PremiumIllustration } from "./components/PremiumIllustration";',
    'import { PremiumIllustration } from "./components/PremiumIllustration";\nimport { ProjectionHealthCard } from "./components/ProjectionHealthCard";',
    "ProjectionHealthCard import",
)
s = replace_once(
    s,
    'import type { AttentionItem, AuthorityResolution, CaptureRequest, EvidenceRequest, EvidenceSource, IntegrityFinding, OnboardingGuide, OnboardingState, PolicySummary, Readiness, WorkflowTask } from "./types";',
    'import type { AttentionItem, AuthorityResolution, CaptureRequest, EvidenceRequest, EvidenceSource, IntegrityFinding, OnboardingGuide, OnboardingState, PolicySummary, Readiness, WorkflowTask } from "./types";\nimport type { ProjectionHealth, ReconcileResult } from "./operationsTypes";',
    "operations type imports",
)
s = replace_once(
    s,
    '  const [configureState, setConfigureState] = useState<LoadState>("idle");',
    '  const [configureState, setConfigureState] = useState<LoadState>("idle");\n  const [projectionHealth, setProjectionHealth] = useState<ProjectionHealth | null>(null);',
    "projection state",
)
old_loader = '''  async function loadConfigureWorkspace() {
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
  }'''
new_loader = '''  async function loadConfigureWorkspace() {
    setConfigureState("loading");
    const [policiesResult, integrityResult, tasksResult, projectionResult] = await Promise.allSettled([loadPolicies(), loadIntegrity(), loadWorkflowTasks(), loadProjectionHealth()]);
    if (policiesResult.status === "fulfilled" && integrityResult.status === "fulfilled" && tasksResult.status === "fulfilled" && projectionResult.status === "fulfilled") {
      setPolicies(policiesResult.value);
      setIntegrity(integrityResult.value);
      setTasks(tasksResult.value);
      setProjectionHealth(projectionResult.value[0] ?? null);
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
  }'''
s = replace_once(s, old_loader, new_loader, "Configure loader")
s = replace_once(
    s,
    '{activeView === "configure" && <ConfigureView policies={policies} findings={integrity} tasks={tasks} state={configureState} onRetry={() => void loadConfigureWorkspace()}/>} ',
    '{activeView === "configure" && <ConfigureView policies={policies} findings={integrity} tasks={tasks} projectionHealth={projectionHealth} state={configureState} onRetry={() => void loadConfigureWorkspace()} onReconcile={checkProgramStatusRecords}/>} ',
    "Configure render",
)
old_decl = 'function ConfigureView({ policies, findings, tasks, state, onRetry }: { policies: PolicySummary[]; findings: IntegrityFinding[]; tasks: WorkflowTask[]; state: LoadState; onRetry: () => void }) {'
new_decl = 'function ConfigureView({ policies, findings, tasks, projectionHealth, state, onRetry, onReconcile }: { policies: PolicySummary[]; findings: IntegrityFinding[]; tasks: WorkflowTask[]; projectionHealth: ProjectionHealth | null; state: LoadState; onRetry: () => void; onReconcile: () => Promise<ReconcileResult> }) {'
s = replace_once(s, old_decl, new_decl, "ConfigureView signature")
old_end = '</div></article></section></>;\n}'
new_end = '</div></article><ProjectionHealthCard health={projectionHealth} onReconcile={onReconcile}/></section></>;\n}'
# Restrict to the ConfigureView return by replacing the first matching terminal
# after the function declaration.
position = s.index(new_decl)
end_position = s.index(old_end, position)
s = s[:end_position] + s[end_position:].replace(old_end, new_end, 1)
p.write_text(s)

p = Path("web/src/continuity.css")
s = p.read_text()
if ".projection-health-grid" not in s:
    s += '''

.projection-health-card .section-header { align-items:flex-start; }
.projection-health-card mark { white-space:nowrap; }
.projection-health-grid { display:grid; grid-template-columns:repeat(4,minmax(0,1fr)); gap:12px; }
.projection-health-grid > div { display:grid; gap:4px; padding:14px; border:1px solid var(--border); border-radius:12px; background:var(--surface-2); }
.projection-health-grid span, .projection-health-grid small { color:var(--muted); }
.projection-health-grid strong { font-size:18px; }
.card-actions { display:flex; justify-content:flex-end; margin-top:14px; }
.success-text { margin:12px 0 0; color:var(--success); }
@media (max-width: 900px) { .projection-health-grid { grid-template-columns:repeat(2,minmax(0,1fr)); } }
@media (max-width: 560px) { .projection-health-grid { grid-template-columns:1fr; } .card-actions .secondary-button { width:100%; } }
'''
p.write_text(s)
