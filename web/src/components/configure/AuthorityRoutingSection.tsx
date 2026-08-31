import { useCallback, useEffect, useState } from "react";
import { loadIntegrity, loadWorkflowTasks } from "../../api";
import type { IntegrityFinding, WorkflowTask } from "../../types";
import { EmptyState } from "../EmptyState";
import { GovernanceAdminPanel } from "../GovernanceAdminPanel";
import { WorkspaceErrorBoundary } from "../WorkspaceErrorBoundary";

type LoadState = "loading" | "live" | "unavailable";

export function AuthorityRoutingSection() {
  const [findings, setFindings] = useState<IntegrityFinding[]>([]);
  const [tasks, setTasks] = useState<WorkflowTask[]>([]);
  const [integrityState, setIntegrityState] = useState<LoadState>("loading");
  const [taskState, setTaskState] = useState<LoadState>("loading");

  const load = useCallback(async () => {
    setIntegrityState("loading");
    setTaskState("loading");
    const [integrityResult, tasksResult] = await Promise.allSettled([loadIntegrity(), loadWorkflowTasks()]);
    if (integrityResult.status === "fulfilled") { setFindings(integrityResult.value); setIntegrityState("live"); }
    else { setFindings([]); setIntegrityState("unavailable"); }
    if (tasksResult.status === "fulfilled") { setTasks(tasksResult.value); setTaskState("live"); }
    else { setTasks([]); setTaskState("unavailable"); }
  }, []);

  useEffect(() => { void load(); }, [load]);

  const degraded = integrityState === "unavailable" || taskState === "unavailable";
  return <section className="configure-domain" aria-labelledby="authority-routing-heading">
    <header className="configure-domain-header">
      <div><span className="eyebrow">Configuration · authority</span><h2 id="authority-routing-heading">Authority & routing</h2><p>Review routing integrity, governed responsibility policies, delegations and the work affected by current ownership.</p></div>
      {degraded && <button className="secondary-button" type="button" onClick={() => void load()}>Retry checks</button>}
    </header>

    <section className="configure-status-strip" aria-label="Routing status">
      <div><span>Routing checks</span><strong>{integrityState === "loading" ? "Checking…" : integrityState === "unavailable" ? "Unavailable" : findings.length ? `${findings.length} need review` : "No blocking gaps"}</strong></div>
      <p>{integrityState === "unavailable" ? "Current routing integrity could not be confirmed." : findings.length ? "Resolve the material routing gaps before relying on affected approval routes." : "All checked routes currently have an eligible path."}</p>
    </section>

    <div className="configure-context-grid">
      <article className="configure-context-panel">
        <div className="configure-subheader"><div><h3>Integrity findings</h3><p>Missing owners, expired delegation or unresolved routes.</p></div></div>
        {integrityState === "loading" ? <div className="workspace-loading compact" aria-live="polite">Checking routing integrity…</div>
          : integrityState === "unavailable" ? <EmptyState kind="unavailable" label="Routing checks" title="Routing checks are unavailable" description="The governance inventory below remains independently available where its own data can be confirmed."/>
            : findings.length ? <div className="configure-compact-list">{findings.map((finding) => <div className={`finding-row severity-${finding.severity.toLowerCase()}`} key={`${finding.type}-${finding.summary}`}><strong>{finding.summary}</strong><span>{finding.required_action}</span></div>)}</div>
              : <div className="calm-empty"><span>✓</span><div><strong>No blocking routing gaps</strong><p>No missing owners, expired delegations or unresolved approval routes were found.</p></div></div>}
      </article>

      <article className="configure-context-panel">
        <div className="configure-subheader"><div><h3>Affected workflow ownership</h3><p>Supporting context only; assigned work remains canonical in Today and Work.</p></div></div>
        {taskState === "loading" ? <div className="workspace-loading compact" aria-live="polite">Loading workflow ownership…</div>
          : taskState === "unavailable" ? <EmptyState kind="unavailable" label="Workflow ownership" title="Workflow ownership is unavailable" description="Routing configuration can still be inspected, but affected work could not be confirmed."/>
            : tasks.length ? <div className="configure-compact-list">{tasks.slice(0, 6).map((task) => <div className="task-row" key={task.id}><div><strong>{task.title}</strong><span>{task.responsibility} · {task.step_key}</span></div><mark>{humanize(task.status)}</mark></div>)}</div>
              : <div className="calm-empty"><span>✓</span><div><strong>No unassigned workflow tasks</strong><p>Every open task returned in this scope has an assignee.</p></div></div>}
      </article>
    </div>

    <WorkspaceErrorBoundary label="Governance policies and delegations"><GovernanceAdminPanel/></WorkspaceErrorBoundary>
  </section>;
}

function humanize(value: string) {
  return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}
