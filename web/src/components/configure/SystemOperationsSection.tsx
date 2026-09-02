import { useCallback, useEffect, useState } from "react";
import { loadBackgroundJobs, loadProjectionHealth, reconcileProgramState, retryBackgroundJob } from "../../api";
import type { BackgroundJobSnapshot, ProjectionHealth, ReconcileResult } from "../../operationsTypes";
import { ProjectionHealthCard } from "../ProjectionHealthCard";

type LoadState = "loading" | "live" | "unavailable";

export function SystemOperationsSection({ canReconcile }: { canReconcile: boolean }) {
  const [health, setHealth] = useState<ProjectionHealth | null>(null);
  const [state, setState] = useState<LoadState>("loading");
  const [jobs, setJobs] = useState<BackgroundJobSnapshot | null>(null);
  const [recovery, setRecovery] = useState<{ jobID: string; rationale: string } | null>(null);
  const [recoveryMessage, setRecoveryMessage] = useState("");

  const load = useCallback(async () => {
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

  useEffect(() => { void load(); }, [load]);

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
    await load();
  }

  const terminalJobs = jobs?.jobs.filter((job) => job.state === "FAILED" || job.state === "DEAD_LETTERED") ?? [];

  return <section className="configure-domain" aria-labelledby="system-operations-heading">
    <header className="configure-domain-header">
      <div><span className="eyebrow">Configuration · operations</span><h2 id="system-operations-heading">System operations</h2><p>Inspect background processing that keeps calculated Program status aligned with recorded changes.</p></div>
      {state === "unavailable" && <button className="secondary-button" type="button" onClick={() => void load()}>Retry</button>}
    </header>
    <ProjectionHealthCard health={health} state={state} canReconcile={canReconcile} onReconcile={reconcile}/>
    <section className="configure-card" aria-labelledby="terminal-jobs-heading">
      <div><h3 id="terminal-jobs-heading">Failed background work</h3><p>{terminalJobs.length ? `${terminalJobs.length} stored jobs require a recovery decision.` : "No terminal background job is recorded in the current tenant."}</p></div>
      {recoveryMessage && <p role="status">{recoveryMessage}</p>}
      {terminalJobs.length > 0 && <div className="configure-record-list">{terminalJobs.map((job) => <article key={`${job.queue}-${job.id}`}>
        <div><strong>{job.kind}</strong><span>{job.queue} · {job.failure_code ?? "FAILURE_DETAIL_UNAVAILABLE"} · {job.attempts} attempts</span></div>
        {canReconcile && <button className="secondary-button" type="button" onClick={() => { setRecovery({ jobID: job.id, rationale: "" }); setRecoveryMessage(""); }}>Review retry</button>}
      </article>)}</div>}
      {recovery && <form onSubmit={(event) => { event.preventDefault(); void retryJob(); }}><label><span>Why is this retry safe now?</span><textarea value={recovery.rationale} onChange={(event) => setRecovery({ ...recovery, rationale: event.target.value })} required minLength={20}/></label><div className="form-actions"><button className="secondary-button" type="button" onClick={() => setRecovery(null)}>Cancel</button><button className="primary-button" type="submit" disabled={recovery.rationale.trim().length < 20}>Schedule retry</button></div></form>}
    </section>
  </section>;
}
