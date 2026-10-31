import { useEffect, useState } from "react";
import { loadIntegrity, loadWorkflowTasks } from "../../api";
import { ApiError } from "../../http";
import type { IntegrityFinding, WorkflowTask } from "../../types";
import { EmptyState } from "../EmptyState";
import { GovernanceAdminPanel } from "../GovernanceAdminPanel";
import { WorkspaceErrorBoundary } from "../WorkspaceErrorBoundary";
import { ActionLink, Button, Notice } from "../ui";

type LoadState = "loading" | "live" | "restricted" | "unavailable";

export function AuthorityRoutingSection({ checksOnly = false, active = true }: { checksOnly?: boolean; active?: boolean }) {
  const [findings, setFindings] = useState<IntegrityFinding[]>([]);
  const [tasks, setTasks] = useState<WorkflowTask[]>([]);
  const [integrityState, setIntegrityState] = useState<LoadState>("loading");
  const [taskState, setTaskState] = useState<LoadState>("loading");
  const [reload, setReload] = useState(0);

  useEffect(() => {
    if (!active) return;
    let cancelled = false;
    setIntegrityState("loading");
    setTaskState("loading");
    void Promise.allSettled([loadIntegrity(), loadWorkflowTasks()]).then(([integrityResult, tasksResult]) => {
      if (cancelled) return;
      if (integrityResult.status === "fulfilled") { setFindings(integrityResult.value); setIntegrityState("live"); }
      else { setFindings([]); setIntegrityState(failureState(integrityResult.reason)); }
      if (tasksResult.status === "fulfilled") { setTasks(tasksResult.value); setTaskState("live"); }
      else { setTasks([]); setTaskState(failureState(tasksResult.reason)); }
    });
    return () => { cancelled = true; };
  }, [active, reload]);

  const degraded = integrityState === "unavailable" || taskState === "unavailable";
  return <section className="configure-domain" aria-labelledby="authority-routing-heading">
    <header className="configure-domain-header">
      <div><h2 id="authority-routing-heading">{checksOnly ? "Routing checks" : "Authority & routing"}</h2><p>Review unresolved routes and affected work.</p></div>
      {degraded && <Button onPress={() => setReload((value) => value + 1)}>Retry checks</Button>}
    </header>

    <section className="configure-status-strip" aria-label="Routing status">
      <div><span>Routing checks</span><strong>{integrityState === "loading" ? "Checking…" : integrityState === "restricted" ? "Access restricted" : integrityState === "unavailable" ? "Unavailable" : findings.length ? `${findings.length} findings returned` : "No findings returned"}</strong></div>
      <p>{integrityState === "loading" ? "Checking routing findings for the current scope." : integrityState === "restricted" ? "Your current access does not allow routing findings." : integrityState === "unavailable" ? "Current routing integrity could not be confirmed." : findings.length ? "Review the returned findings and their required actions." : "No routing findings were returned for the current scope."}</p>
    </section>

    <div className="configure-context-grid">
      <article className="configure-context-panel">
        <div className="configure-subheader"><div><h3>Integrity findings</h3><p>Missing owners, expired delegation or unresolved routes.</p></div></div>
        {integrityState === "loading" ? <div className="workspace-loading compact" aria-live="polite">Checking routing integrity…</div>
          : integrityState === "restricted" ? <Notice tone="warning">Routing findings are restricted with your current access.</Notice>
          : integrityState === "unavailable" ? <EmptyState kind="unavailable" label="Routing checks" title="Routing checks are unavailable" description="Retry to check routing findings for the current scope."/>
            : findings.length ? <div className="configure-compact-list">{findings.map((finding) => <div className={`finding-row severity-${finding.severity.toLowerCase()}`} key={`${finding.type}-${finding.summary}`}><strong>{finding.summary}</strong><span>{finding.required_action}</span></div>)}</div>
              : <div className="calm-empty"><div><strong>No routing findings returned</strong><p>Approval authority is checked when a decision is made.</p></div></div>}
      </article>

      <article className="configure-context-panel">
        <div className="configure-subheader"><div><h3>Workflow ownership</h3><p>Up to 20 tasks returned for the current scope.</p></div></div>
        {taskState === "loading" ? <div className="workspace-loading compact" aria-live="polite">Loading workflow ownership…</div>
          : taskState === "restricted" ? <Notice tone="warning">Workflow ownership is restricted with your current access.</Notice>
          : taskState === "unavailable" ? <EmptyState kind="unavailable" label="Workflow ownership" title="Workflow ownership is unavailable" description="Retry to check workflow tasks in the current scope."/>
            : tasks.length ? <><div className="configure-compact-list">{tasks.slice(0, 6).map((task) => <div className="task-row" key={task.id}><div><strong>{task.title}</strong><span>{humanize(task.responsibility)} · {humanize(task.step_key)}</span></div><mark>{humanize(task.status)}</mark></div>)}</div><p className="muted-copy">{Math.min(tasks.length, 6)} of {tasks.length} returned tasks</p><ActionLink href="#work">Open Work &amp; Approvals</ActionLink></>
              : <div className="calm-empty"><div><strong>No workflow tasks returned</strong><p>No tasks were returned for the current scope.</p></div></div>}
      </article>
    </div>

    {!checksOnly && <WorkspaceErrorBoundary label="Governance policies and delegations"><GovernanceAdminPanel/></WorkspaceErrorBoundary>}
  </section>;
}

function humanize(value: string) {
  return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}

function failureState(error: unknown): LoadState {
  return error instanceof ApiError && (error.status === 401 || error.status === 403) ? "restricted" : "unavailable";
}
