import { useCallback, useEffect, useState } from "react";
import { loadProjectionHealth, reconcileProgramState } from "../../api";
import type { ProjectionHealth, ReconcileResult } from "../../operationsTypes";
import { ProjectionHealthCard } from "../ProjectionHealthCard";

type LoadState = "loading" | "live" | "unavailable";

export function SystemOperationsSection({ canReconcile }: { canReconcile: boolean }) {
  const [health, setHealth] = useState<ProjectionHealth | null>(null);
  const [state, setState] = useState<LoadState>("loading");

  const load = useCallback(async () => {
    setState("loading");
    try {
      const items = await loadProjectionHealth();
      setHealth(items[0] ?? null);
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

  return <section className="configure-domain" aria-labelledby="system-operations-heading">
    <header className="configure-domain-header">
      <div><span className="eyebrow">Configuration · operations</span><h2 id="system-operations-heading">System operations</h2><p>Inspect background processing that keeps calculated Program status aligned with recorded changes.</p></div>
      {state === "unavailable" && <button className="secondary-button" type="button" onClick={() => void load()}>Retry</button>}
    </header>
    <ProjectionHealthCard health={health} state={state} canReconcile={canReconcile} onReconcile={reconcile}/>
  </section>;
}
