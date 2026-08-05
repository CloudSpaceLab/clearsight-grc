import { useState } from "react";
import type { ProjectionHealth, ReconcileResult } from "../operationsTypes";

function stateLabel(value: ProjectionHealth["state"]) {
  switch (value) {
    case "CURRENT": return "Current";
    case "UPDATE_PENDING": return "Updates pending";
    case "DELAYED": return "Delayed";
    case "NEEDS_ATTENTION": return "Needs attention";
    default: return "Not configured";
  }
}

function stateClass(value: ProjectionHealth["state"]) {
  if (value === "CURRENT") return "status-good";
  if (value === "UPDATE_PENDING") return "status-warning";
  if (value === "DELAYED" || value === "NEEDS_ATTENTION") return "status-critical";
  return "status-neutral";
}

function formatTime(value?: string) {
  if (!value) return "Not recorded";
  const parsed = Date.parse(value);
  return Number.isFinite(parsed) ? new Date(parsed).toLocaleString() : "Not recorded";
}

export function ProjectionHealthCard({ health, onReconcile }: { health: ProjectionHealth | null; onReconcile: () => Promise<ReconcileResult> }) {
  const [running, setRunning] = useState(false);
  const [result, setResult] = useState<ReconcileResult | null>(null);
  const [error, setError] = useState("");

  async function checkRecords() {
    setRunning(true);
    setError("");
    try {
      setResult(await onReconcile());
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Status records could not be checked.");
    } finally {
      setRunning(false);
    }
  }

  return <article className="config-card wide projection-health-card">
    <div className="section-header">
      <div><h2>Program status updates</h2><p>Checks whether calculated Program status has caught up with recorded changes.</p></div>
      <mark className={stateClass(health?.state ?? "NOT_CONFIGURED")}>{stateLabel(health?.state ?? "NOT_CONFIGURED")}</mark>
    </div>
    {!health ? <p>Program status update health is unavailable.</p> : <div className="projection-health-grid">
      <div><span>Waiting</span><strong>{health.pending}</strong><small>Status updates not yet completed</small></div>
      <div><span>Failed</span><strong>{health.failed}</strong><small>Updates requiring operator review</small></div>
      <div><span>Oldest waiting update</span><strong>{health.oldest_pending ? `${Math.max(0, Math.round(health.lag_seconds / 60))} min` : "—"}</strong><small>{health.oldest_pending ? formatTime(health.oldest_pending) : "No updates waiting"}</small></div>
      <div><span>Last completed</span><strong>{health.last_completed ? formatTime(health.last_completed) : "Not recorded"}</strong><small>Latest successful Program status update</small></div>
    </div>}
    {health?.last_error && <p className="error-text">Latest error: {health.last_error}</p>}
    {result && <p className="success-text">Checked {result.checked} Programs. {result.queued} status update{result.queued === 1 ? " was" : "s were"} queued.</p>}
    {error && <p className="error-text">{error}</p>}
    <div className="card-actions"><button type="button" className="secondary-button" disabled={running} onClick={() => void checkRecords()}>{running ? "Checking…" : "Check status records"}</button></div>
  </article>;
}
