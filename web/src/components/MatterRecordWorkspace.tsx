import { useCallback, useEffect, useRef, useState } from "react";
import { loadMatter } from "../api";
import { loadMatterOperations } from "../matterOperationsApi";
import type { MatterOperations } from "../matterOperationsApi";
import type { MatterAggregate } from "../types";
import { EmptyState } from "./EmptyState";
import { MatterCurrentHandoff } from "./MatterCurrentHandoff";

type Props = {
  matterID: string;
  onBack: () => void;
};

type State = "loading" | "live" | "unavailable";

function priorityLabel(value: number) {
  if (value >= 5) return "Critical";
  if (value === 4) return "High";
  if (value === 3) return "Medium";
  if (value === 2) return "Normal";
  return "Low";
}

function humanize(value: string) {
  return value.replaceAll("_", " ").toLowerCase().replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}

function formatFact(value: unknown) {
  if (value === null || value === undefined || value === "") return "Not recorded";
  return typeof value === "object" ? JSON.stringify(value) : String(value);
}

export function MatterRecordWorkspace({ matterID, onBack }: Props) {
  const [state, setState] = useState<State>("loading");
  const [aggregate, setAggregate] = useState<MatterAggregate | null>(null);
  const [operations, setOperations] = useState<MatterOperations | null>(null);
  const loadID = useRef(0);

  const reload = useCallback(async () => {
    const current = ++loadID.current;
    setState("loading");
    try {
      const [nextAggregate, nextOperations] = await Promise.all([loadMatter(matterID), loadMatterOperations(matterID)]);
      if (current !== loadID.current) return;
      setAggregate(nextAggregate);
      setOperations(nextOperations);
      setState("live");
    } catch {
      if (current !== loadID.current) return;
      setAggregate(null);
      setOperations(null);
      setState("unavailable");
    }
  }, [matterID]);

  useEffect(() => {
    void reload();
    return () => { loadID.current += 1; };
  }, [reload]);

  return <section className="matter-record-workspace" aria-label="Issue or change record">
    <button aria-label="Back to issues and changes" className="text-button matter-record-back" type="button" onClick={onBack}>← Back to issues and changes</button>
    {state === "loading" && <div className="workspace-loading" aria-live="polite" aria-busy="true">Loading issue record and responsibilities…</div>}
    {state === "unavailable" && <EmptyState kind="unavailable" label="Issue or change" title="Issue responsibilities could not be loaded" description="The record is read-only until its current responsibilities can be checked. Try again before changing information or assigned work." action="Try again" onAction={() => void reload()}/>} 
    {state === "live" && aggregate && operations && <>
      <header className="matter-record-header">
        <div>
          <span className="matter-kicker">{aggregate.type_label} · {aggregate.matter.reference}</span>
          <h1>{aggregate.matter.title}</h1>
          <p>{aggregate.matter.summary}</p>
        </div>
        <dl>
          <div><dt>Priority</dt><dd>{priorityLabel(aggregate.matter.priority)}</dd></div>
          <div><dt>Status</dt><dd>{aggregate.status_label}</dd></div>
          <div><dt>Record version</dt><dd>{aggregate.matter.version}</dd></div>
        </dl>
      </header>
      {!operations.authority_available && <div className="inline-notice" role="status"><strong>Responsibilities are temporarily unavailable.</strong> Values and stored owners remain visible, but changes are disabled until authority routing recovers.</div>}
      <MatterCurrentHandoff aggregate={aggregate} operations={operations.operations}/>
      <section className="matter-record-grid">
        <article className="matter-record-panel">
          <div className="matter-record-section-heading"><div><span className="eyebrow">Evidence and facts</span><h2>Recorded information</h2></div><span>{Object.keys(aggregate.matter.known_facts).length} facts</span></div>
          <dl className="matter-record-facts">{Object.entries(aggregate.matter.known_facts).map(([key, value]) => <div key={key}><dt>{humanize(key)}</dt><dd>{formatFact(value)}</dd></div>)}</dl>
          {aggregate.matter.missing_facts.length > 0 && <div className="matter-record-attention"><strong>Information still needed</strong><ul>{aggregate.matter.missing_facts.map((item, index) => <li key={`${index}-${formatFact(item)}`}>{formatFact(item)}</li>)}</ul></div>}
        </article>
        <article className="matter-record-panel">
          <div className="matter-record-section-heading"><div><span className="eyebrow">Assigned work</span><h2>Actions and outcome checks</h2></div><span>{aggregate.actions.length} actions</span></div>
          {aggregate.actions.length ? aggregate.actions.map((action) => {
            const actionOperation = operations.operations.find((operation) => operation.command === "matter.action.transition" && operation.subresource_id === action.id);
            return <div className="matter-record-row" id={`matter-operation-matter.action.transition-${action.id}`} key={action.id}><div><strong>{action.title}</strong><p>{action.description}</p><small>Assigned to {actionOperation?.assigned_to?.display_name ?? "Owner not resolved"}</small></div><span>{humanize(action.status)}</span></div>;
          }) : <p>No actions have been recorded for this issue.</p>}
          {aggregate.verification_contracts.length === 0 && <div className="matter-record-attention"><strong>No outcome check has been defined</strong><p>Define the result that must be independently confirmed before this issue can close.</p></div>}
        </article>
      </section>
    </>}
  </section>;
}
