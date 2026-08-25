import { useCallback, useEffect, useRef, useState } from "react";
import { loadProgram } from "../api";
import { loadProgramOperations } from "../programOperationsApi";
import type { ProgramOperations } from "../programOperationsApi";
import { loadProgramReviewDigest } from "../programReviewApi";
import type { ProgramReviewDigest as ReviewDigest } from "../programReviewApi";
import type { ProgramAggregate } from "../types";
import { EmptyState } from "./EmptyState";
import { ProgramCurrentPosition } from "./ProgramCurrentPosition";
import { ProgramReviewDigest } from "./ProgramReviewDigest";

type Props = { programID: string; onBack: () => void };
type State = "loading" | "live" | "unavailable";

function statusLabel(value: string) {
  switch (value) {
    case "DRAFT": return "Setup in progress";
    case "ACTIVE": return "Active";
    case "PAUSED": return "Paused";
    case "RETIRED": return "Ended";
    default: return value.replaceAll("_", " ").toLowerCase();
  }
}

export function ProgramRecordWorkspace({ programID, onBack }: Props) {
  const [state, setState] = useState<State>("loading");
  const [aggregate, setAggregate] = useState<ProgramAggregate | null>(null);
  const [operations, setOperations] = useState<ProgramOperations | null>(null);
  const [digest, setDigest] = useState<ReviewDigest | null>(null);
  const loadID = useRef(0);

  const reload = useCallback(async () => {
    const current = ++loadID.current;
    setState("loading");
    try {
      const [nextAggregate, nextOperations, nextDigest] = await Promise.all([
        loadProgram(programID), loadProgramOperations(programID), loadProgramReviewDigest(programID),
      ]);
      if (current !== loadID.current) return;
      setAggregate(nextAggregate);
      setOperations(nextOperations);
      setDigest(nextDigest);
      setState("live");
    } catch {
      if (current !== loadID.current) return;
      setAggregate(null);
      setOperations(null);
      setDigest(null);
      setState("unavailable");
    }
  }, [programID]);

  useEffect(() => {
    void reload();
    return () => { loadID.current += 1; };
  }, [reload]);

  return <section className="program-record-workspace" aria-label="Program record">
    <button aria-label="Back to Programs" className="text-button program-record-back" type="button" onClick={onBack}>← Back to Programs</button>
    {state === "loading" && <div className="workspace-loading" aria-live="polite" aria-busy="true">Loading Program record and responsibilities…</div>}
    {state === "unavailable" && <EmptyState kind="unavailable" label="Program" title="Program responsibilities could not be loaded" description="The record is read-only until its current responsibilities and review position can be checked. Try again before making a change." action="Try again" onAction={() => void reload()}/>} 
    {state === "live" && aggregate && operations && digest && <>
      <header className="program-record-header">
        <div><span className="program-kicker">{aggregate.program.code} · {aggregate.program.owning_function}</span><h1>{aggregate.program.name}</h1><p>{aggregate.program.jurisdiction || "Jurisdiction not recorded"}</p></div>
        <dl><div><dt>Operating status</dt><dd>{statusLabel(aggregate.program.status)}</dd></div><div><dt>Calculated state</dt><dd>{aggregate.state_label}</dd></div><div><dt>Record version</dt><dd>{aggregate.program.version}</dd></div></dl>
      </header>
      {!operations.authority_available && <div className="inline-notice" role="status"><strong>Responsibilities are temporarily unavailable.</strong> Program values and stored owners remain visible, but changes are disabled until authority routing recovers.</div>}
      <ProgramCurrentPosition aggregate={aggregate} operations={operations} digest={digest}/>
      <section className="program-record-grid">
        <article className="program-record-panel" id="program-review-panel"><ProgramReviewDigest aggregate={aggregate} initialDigest={digest} onDigestUpdated={setDigest}/></article>
        <article className="program-record-panel" id="program-details-panel"><span className="eyebrow">Program details</span><h2>Scope and ownership</h2><p>Current Program identity, scope, effective dates and responsibility are shown here. Editing controls follow the signed-in owner route.</p></article>
        <article className="program-record-panel" id="program-requirements-panel"><span className="eyebrow">Requirements</span><h2>Obligations and applicability</h2><p>{aggregate.requirements.length} requirement{aggregate.requirements.length === 1 ? " is" : "s are"} recorded for this Program.</p></article>
        <article className="program-record-panel" id="program-safeguards-panel"><span className="eyebrow">Safeguards</span><h2>Control coverage</h2><p>{aggregate.control_implementations.length} safeguard implementation{aggregate.control_implementations.length === 1 ? " is" : "s are"} recorded.</p></article>
        <article className="program-record-panel" id="program-evidence-panel"><span className="eyebrow">Evidence checks</span><h2>Evidence and results</h2><p>{aggregate.evidence_contracts.length} evidence check{aggregate.evidence_contracts.length === 1 ? " is" : "s are"} defined.</p></article>
      </section>
    </>}
  </section>;
}
