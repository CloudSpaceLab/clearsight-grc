import { useCallback, useEffect, useRef, useState } from "react";
import { loadMatter } from "../api";
import { loadMatterOperations } from "../matterOperationsApi";
import type { MatterOperations } from "../matterOperationsApi";
import type { MatterAggregate } from "../types";
import { EmptyState } from "./EmptyState";
import { MatterCurrentHandoff } from "./MatterCurrentHandoff";
import { MatterDetailsPanel } from "./MatterDetailsPanel";
import { MatterInformationPanel } from "./MatterInformationPanel";
import { MatterActionsPanel } from "./MatterActionsPanel";
import { MatterOutcomePanel } from "./MatterOutcomePanel";
import { MatterDecisionResponsePanel } from "./MatterDecisionResponsePanel";

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

  async function applyUpdated(value: MatterAggregate) {
    setAggregate(value);
    try {
      setOperations(await loadMatterOperations(matterID));
    } catch {
      setOperations((current) => current ? {
        ...current,
        matter_version: value.matter.version,
        authority_available: false,
        operations: current.operations.map((operation) => ({ ...operation, can_act: false, reason: "Responsibilities could not be refreshed after this change." })),
      } : null);
    }
  }

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
        <MatterDetailsPanel aggregate={aggregate} operations={operations.operations} onUpdated={applyUpdated} onReload={() => void reload()}/>
        <MatterInformationPanel aggregate={aggregate} operations={operations.operations} onUpdated={applyUpdated} onReload={() => void reload()}/>
        <MatterActionsPanel aggregate={aggregate} operations={operations.operations} onUpdated={applyUpdated} onReload={() => void reload()}/>
        <MatterDecisionResponsePanel aggregate={aggregate} operations={operations.operations} onUpdated={applyUpdated} onReload={() => void reload()}/>
        <MatterOutcomePanel aggregate={aggregate} operations={operations.operations} onUpdated={applyUpdated} onReload={() => void reload()}/>
      </section>
    </>}
  </section>;
}
