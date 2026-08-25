import { useState } from "react";
import type { FormEvent } from "react";
import { transitionProgram } from "../programOperationsApi";
import type { ProgramOperation } from "../programOperationsApi";
import type { ProgramAggregate } from "../types";

type Props = { aggregate: ProgramAggregate; operations: ProgramOperation[]; onUpdated: (value: ProgramAggregate) => void; onReload: () => void };

function label(value: string) {
  switch (value) { case "ACTIVE": return "Active"; case "PAUSED": return "Paused"; case "RETIRED": return "Ended"; default: return value.replaceAll("_", " ").toLowerCase(); }
}
function action(value: string) {
  switch (value) { case "ACTIVE": return "Activate Program"; case "PAUSED": return "Pause Program"; case "RETIRED": return "End Program"; default: return "Change Program status"; }
}

export function ProgramStatusPanel({ aggregate, operations, onUpdated, onReload }: Props) {
  const operation = operations.find((value) => value.command === "program.transition");
  const targets = operation?.allowed_targets ?? [];
  const [target, setTarget] = useState(targets[0] ?? "");
  const [rationale, setRationale] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit(event: FormEvent) {
    event.preventDefault(); setBusy(true); setError("");
    try {
      const value = await transitionProgram(aggregate.program.id, aggregate.program.version, target, rationale.trim());
      onUpdated(value); setRationale("");
    } catch (value) { setError(value instanceof Error ? value.message : "The Program status could not be changed."); }
    finally { setBusy(false); }
  }

  return <article className="program-record-panel" id="program-status-panel">
    <div className="program-panel-heading"><div><span className="eyebrow">Program lifecycle</span><h2>Operating status</h2></div></div>
    <dl className="program-record-facts"><div><dt>Current status</dt><dd>{label(aggregate.program.status)}</dd></div><div><dt>Current authorizer</dt><dd>{operation?.assigned_to?.display_name ?? "Not assigned"}</dd></div></dl>
    {operation?.can_act && targets.length > 0 ? <form className="program-operation-form" onSubmit={(event) => void submit(event)}>
      <label><span>New operating status</span><select value={targets.includes(target) ? target : targets[0]} onChange={(event) => setTarget(event.target.value)}>{targets.map((value) => <option value={value} key={value}>{label(value)}</option>)}</select></label>
      <label className="wide"><span>Reason for status change</span><textarea required value={rationale} onChange={(event) => setRationale(event.target.value)}/></label>
      {error && <p className="program-form-error wide" role="alert">{error} <button className="text-button" type="button" onClick={onReload}>Reload Program</button></p>}
      <div className="program-form-actions wide"><button className="primary-button" disabled={busy || !target || !rationale.trim()} type="submit">{busy ? "Saving…" : action(target)}</button></div>
    </form> : operation?.reason ? <p className="program-operation-reason">{operation.reason}</p> : <p className="program-operation-reason">No status change is available from the current operating status.</p>}
  </article>;
}
