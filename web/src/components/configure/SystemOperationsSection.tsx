import { useCallback, useEffect, useState } from "react";
import { loadBackgroundJobs, loadProjectionHealth, reconcileProgramState, retryBackgroundJob } from "../../api";
import type { BackgroundJobSnapshot, ProjectionHealth, ReconcileResult } from "../../operationsTypes";
import { ProjectionHealthCard } from "../ProjectionHealthCard";
import { Button, Notice, Tabs, TextArea } from "../ui";
import { SystemActivityPanel } from "./SystemActivityPanel";

type LoadState = "loading" | "live" | "unavailable";
type OperationsView = "health" | "activity" | "audit";

const operationViews = [
  { id: "health", label: "Health" },
  { id: "activity", label: "Activity" },
  { id: "audit", label: "Audit log" },
] satisfies ReadonlyArray<{ id: OperationsView; label: string }>;

export function SystemOperationsSection({ canReconcile }: { canReconcile: boolean }) {
  const [view, setView] = useState<OperationsView>("health");
  const [health, setHealth] = useState<ProjectionHealth | null>(null);
  const [state, setState] = useState<LoadState>("loading");
  const [jobs, setJobs] = useState<BackgroundJobSnapshot | null>(null);
  const [recovery, setRecovery] = useState<{ jobID: string; rationale: string } | null>(null);
  const [recoveryMessage, setRecoveryMessage] = useState("");

  const loadHealth = useCallback(async () => {
    setState("loading");
    try {
      const [items, background] = await Promise.all([loadProjectionHealth(), loadBackgroundJobs()]);
      setHealth(items[0] ?? null);
      setJobs(background);
      setState("live");
    } catch {
      setHealth(null);
      setState("unavailable");
    }
  }, []);

  useEffect(() => {
    if (view === "health") void loadHealth();
  }, [loadHealth, view]);

  async function reconcile(): Promise<ReconcileResult> {
    const result = await reconcileProgramState();
    const items = await loadProjectionHealth();
    setHealth(items[0] ?? null);
    setState("live");
    return result;
  }

  async function retryJob() {
    const job = jobs?.jobs.find((item) => item.id === recovery?.jobID);
    if (!job || !recovery || recovery.rationale.trim().length < 20) return;
    const receipt = await retryBackgroundJob(job.id, job.queue, job.attempts, recovery.rationale);
    setRecoveryMessage(`${job.kind} recovery was scheduled at ${new Date(receipt.retried_at).toLocaleString()}.`);
    setRecovery(null);
    await loadHealth();
  }

  const terminalJobs = jobs?.jobs.filter((job) => job.state === "FAILED" || job.state === "DEAD_LETTERED") ?? [];

  function healthView() {
    return <div className="system-operations-health">
      <ProjectionHealthCard health={health} state={state} canReconcile={canReconcile} onReconcile={reconcile}/>
      <section className="configure-card" aria-labelledby="terminal-jobs-heading">
        <div><h3 id="terminal-jobs-heading">Failed background work</h3><p>{terminalJobs.length ? `${terminalJobs.length} stored jobs require a recovery decision.` : "No terminal background job is recorded in the current tenant."}</p></div>
        {recoveryMessage && <Notice tone="success">{recoveryMessage}</Notice>}
        {terminalJobs.length > 0 && <div className="configure-record-list">{terminalJobs.map((job) => <article key={`${job.queue}-${job.id}`}>
          <div><strong>{job.kind}</strong><span>{job.queue} · {job.failure_code ?? "FAILURE_DETAIL_UNAVAILABLE"} · {job.attempts} attempts</span></div>
          {canReconcile && <Button variant="secondary" size="compact" onPress={() => { setRecovery({ jobID: job.id, rationale: "" }); setRecoveryMessage(""); }}>Review retry</Button>}
        </article>)}</div>}
        {recovery && <form className="system-operation-recovery" onSubmit={(event) => { event.preventDefault(); void retryJob(); }}>
          <TextArea label="Why is this retry safe now?" value={recovery.rationale} onChange={(rationale) => setRecovery({ ...recovery, rationale })} isRequired rows={4}/>
          <div className="form-actions"><Button variant="secondary" onPress={() => setRecovery(null)}>Cancel</Button><Button type="submit" variant="primary" isDisabled={recovery.rationale.trim().length < 20}>Schedule retry</Button></div>
        </form>}
      </section>
    </div>;
  }

  return <section className="configure-domain" aria-labelledby="system-operations-heading">
    <header className="configure-domain-header">
      <div><span className="eyebrow">Configuration · operations</span><h2 id="system-operations-heading">System operations</h2><p>Inspect platform health, recent system activity and the recorded audit trail without mixing operational recovery with governed business work.</p></div>
      {view === "health" && state === "unavailable" && <Button variant="secondary" onPress={() => void loadHealth()}>Retry</Button>}
    </header>

    <Tabs ariaLabel="System operations views" items={operationViews} selectedKey={view} onSelectionChange={setView}>
      {(selected) => selected === "health" ? healthView() : <SystemActivityPanel mode={selected}/>} 
    </Tabs>
  </section>;
}
