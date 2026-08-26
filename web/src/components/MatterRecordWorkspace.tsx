import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import { loadMatter, loadMatterAt } from "../api";
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
import { RecordSnapshotControl } from "./RecordSnapshotControl";

type Props = {
  matterID: string;
  onBack: () => void;
};

type LoadState = "loading" | "live" | "unavailable";

function priorityLabel(value: number) {
  if (value >= 5) return "Critical";
  if (value === 4) return "High";
  if (value === 3) return "Medium";
  if (value === 2) return "Normal";
  return "Low";
}

export function MatterRecordWorkspace({ matterID, onBack }: Props) {
  const [aggregateState, setAggregateState] = useState<LoadState>("loading");
  const [operationsState, setOperationsState] = useState<LoadState>("loading");
  const [aggregate, setAggregate] = useState<MatterAggregate | null>(null);
  const [operations, setOperations] = useState<MatterOperations | null>(null);
  const loadIDs = useRef({ aggregate: 0, operations: 0 });
  const activeTarget = useRef({ id: matterID, generation: 0 });
  const startedTargetID = useRef<string | null>(null);
  const mounted = useRef(false);

  useLayoutEffect(() => {
    if (activeTarget.current.id === matterID) return;
    activeTarget.current = { id: matterID, generation: activeTarget.current.generation + 1 };
    setAggregate(null);
    setOperations(null);
    setAggregateState("loading");
    setOperationsState("loading");
  }, [matterID]);

  useEffect(() => {
    mounted.current = true;
    return () => { mounted.current = false; };
  }, []);

  const renderTarget = activeTarget.current;

  const loadAggregate = useCallback(async (target = activeTarget.current) => {
    const current = ++loadIDs.current.aggregate;
    setAggregateState("loading");
    try {
      const nextAggregate = await loadMatter(target.id);
      if (!mounted.current || current !== loadIDs.current.aggregate || activeTarget.current !== target) return;
      setAggregate(nextAggregate);
      setAggregateState("live");
    } catch {
      if (!mounted.current || current !== loadIDs.current.aggregate || activeTarget.current !== target) return;
      setAggregateState("unavailable");
    }
  }, []);

  const loadOperations = useCallback(async (target = activeTarget.current) => {
    const current = ++loadIDs.current.operations;
    setOperations(null);
    setOperationsState("loading");
    try {
      const nextOperations = await loadMatterOperations(target.id);
      if (!mounted.current || current !== loadIDs.current.operations || activeTarget.current !== target) return;
      setOperations(nextOperations);
      setOperationsState("live");
    } catch {
      if (!mounted.current || current !== loadIDs.current.operations || activeTarget.current !== target) return;
      setOperations(null);
      setOperationsState("unavailable");
    }
  }, []);

  const reloadRecord = useCallback(async () => {
    await Promise.all([loadAggregate(), loadOperations()]);
  }, [loadAggregate, loadOperations]);

  useEffect(() => {
    if (startedTargetID.current === matterID) return;
    startedTargetID.current = matterID;
    const target = activeTarget.current;
    setAggregate(null);
    setOperations(null);
    setAggregateState("loading");
    setOperationsState("loading");
    void loadAggregate(target);
    void loadOperations(target);
  }, [matterID, loadAggregate, loadOperations]);

  async function applyUpdated(value: MatterAggregate) {
    if (activeTarget.current !== renderTarget || value.matter.id !== renderTarget.id) return;
    loadIDs.current.aggregate += 1;
    setAggregate(value);
    setAggregateState("live");
    await loadOperations();
  }

  const operationVersionMatches = Boolean(aggregate && operations && operations.matter_version === aggregate.matter.version);
  const operationsOutdated = operationsState === "live" && Boolean(aggregate && operations) && !operationVersionMatches;
  const currentOperations = operationsState === "live" && operations?.authority_available && operationVersionMatches
    ? operations.operations
    : operations?.operations.map((operation) => ({ ...operation, can_act: false })) ?? [];
  const responsibleParties = operationsState === "live" && operationVersionMatches ? operations?.responsible_parties ?? [] : [];

  return <section className="matter-record-workspace" aria-label="Issue or change record">
    <button aria-label="Back to issues and changes" className="text-button matter-record-back" type="button" onClick={onBack}>← Back to issues and changes</button>
    {aggregateState === "loading" && !aggregate && <div className="workspace-loading" aria-live="polite" aria-busy="true">Loading issue record…</div>}
    {aggregateState === "unavailable" && !aggregate && <EmptyState kind="unavailable" label="Issue or change" title="Issue record could not be loaded" description="The current issue information is unavailable. Retry the issue record before relying on its status or assigned work." action="Retry issue record" onAction={() => void loadAggregate()}/>}
    {aggregate && <>
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
      <RecordSnapshotControl recordLabel="issue" loadSnapshot={async (at) => { const value = await loadMatterAt(aggregate.matter.id, at); return { version: value.matter.version, status: value.matter.status, updatedAt: value.matter.updated_at }; }}/>
      {operationsState === "loading" && <div className="inline-notice" role="status"><strong>Checking current responsibilities.</strong> Issue values remain visible while the current responsibility route is loaded.</div>}
      {operationsState === "unavailable" && <div className="inline-notice" role="status"><strong>Issue responsibilities could not be checked.</strong> Issue values remain visible, but changes are disabled until the current responsibility route is available. <button className="text-button" type="button" onClick={() => void loadOperations()}>Retry responsibilities</button></div>}
      {operations && !operations.authority_available && <div className="inline-notice" role="status"><strong>Responsibilities are temporarily unavailable.</strong> Values and stored owners remain visible, but changes are disabled until authority routing recovers. <button className="text-button" type="button" onClick={() => void loadOperations()}>Retry responsibilities</button></div>}
      {operationsState === "live" && operations?.responsibility_labels_complete === false && <div className="inline-notice" role="status"><strong>Some assignee names could not be loaded.</strong> Recorded responsibilities remain visible, and available actions still use the current authority route. <button className="text-button" type="button" onClick={() => void loadOperations()}>Reload assignee names</button></div>}
      {operationsOutdated && <div className="inline-notice" role="status"><strong>Issue responsibilities are out of date.</strong> Issue values remain visible, but changes are disabled until responsibilities match issue version {aggregate.matter.version}. <button className="text-button" type="button" onClick={() => void reloadRecord()}>Reload issue data</button></div>}
      <MatterCurrentHandoff aggregate={aggregate} operations={currentOperations} responsibleParties={responsibleParties}/>
      <section className="matter-record-grid">
        <MatterDetailsPanel aggregate={aggregate} operations={currentOperations} responsibleParties={responsibleParties} onUpdated={applyUpdated} onReload={() => void reloadRecord()}/>
        <MatterInformationPanel aggregate={aggregate} operations={currentOperations} onUpdated={applyUpdated} onReload={() => void reloadRecord()}/>
        <MatterActionsPanel aggregate={aggregate} operations={currentOperations} responsibleParties={responsibleParties} onUpdated={applyUpdated} onReload={() => void reloadRecord()}/>
        <MatterDecisionResponsePanel aggregate={aggregate} operations={currentOperations} onUpdated={applyUpdated} onReload={() => void reloadRecord()}/>
        <MatterOutcomePanel aggregate={aggregate} operations={currentOperations} responsibleParties={responsibleParties} onUpdated={applyUpdated} onReload={() => void reloadRecord()}/>
      </section>
    </>}
  </section>;
}
