import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import { loadProgram } from "../api";
import { loadProgramOperations } from "../programOperationsApi";
import type { ProgramOperations } from "../programOperationsApi";
import { loadProgramReviewDigest } from "../programReviewApi";
import type { ProgramReviewDigest as ReviewDigest } from "../programReviewApi";
import type { ProgramAggregate } from "../types";
import { EmptyState } from "./EmptyState";
import { ProgramCurrentPosition } from "./ProgramCurrentPosition";
import { ProgramReviewDigest } from "./ProgramReviewDigest";
import { ProgramDetailsPanel } from "./ProgramDetailsPanel";
import { ProgramRequirementsPanel } from "./ProgramRequirementsPanel";
import { ProgramSafeguardsPanel } from "./ProgramSafeguardsPanel";
import { ProgramEvidencePanel } from "./ProgramEvidencePanel";
import { ProgramIssuesPanel } from "./ProgramIssuesPanel";
import { ProgramStatusPanel } from "./ProgramStatusPanel";

type Props = { programID: string; onBack: () => void; actorPrincipalID?: string; canConfigureSources?: boolean; onOpenMatter?: (matterID: string) => void };
type LoadState = "loading" | "live" | "unavailable";

function statusLabel(value: string) {
  switch (value) {
    case "DRAFT": return "Setup in progress";
    case "ACTIVE": return "Active";
    case "PAUSED": return "Paused";
    case "RETIRED": return "Ended";
    default: return value.replaceAll("_", " ").toLowerCase();
  }
}

export function ProgramRecordWorkspace({ programID, onBack, actorPrincipalID = "", canConfigureSources = false, onOpenMatter = (matterID) => { window.location.hash = `#work/matters/${encodeURIComponent(matterID)}`; } }: Props) {
  const [aggregateState, setAggregateState] = useState<LoadState>("loading");
  const [operationsState, setOperationsState] = useState<LoadState>("loading");
  const [reviewState, setReviewState] = useState<LoadState>("loading");
  const [aggregate, setAggregate] = useState<ProgramAggregate | null>(null);
  const [operations, setOperations] = useState<ProgramOperations | null>(null);
  const [digest, setDigest] = useState<ReviewDigest | null>(null);
  const loadIDs = useRef({ aggregate: 0, operations: 0, review: 0 });
  const activeTarget = useRef({ id: programID, generation: 0 });
  const startedTargetID = useRef<string | null>(null);
  const mounted = useRef(false);

  useLayoutEffect(() => {
    if (activeTarget.current.id === programID) return;
    activeTarget.current = { id: programID, generation: activeTarget.current.generation + 1 };
    setAggregate(null);
    setOperations(null);
    setDigest(null);
    setAggregateState("loading");
    setOperationsState("loading");
    setReviewState("loading");
  }, [programID]);

  useEffect(() => {
    mounted.current = true;
    return () => { mounted.current = false; };
  }, []);

  const renderTarget = activeTarget.current;

  const loadAggregate = useCallback(async (target = activeTarget.current) => {
    const current = ++loadIDs.current.aggregate;
    setAggregateState("loading");
    try {
      const nextAggregate = await loadProgram(target.id);
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
      const nextOperations = await loadProgramOperations(target.id);
      if (!mounted.current || current !== loadIDs.current.operations || activeTarget.current !== target) return;
      setOperations(nextOperations);
      setOperationsState("live");
    } catch {
      if (!mounted.current || current !== loadIDs.current.operations || activeTarget.current !== target) return;
      setOperations(null);
      setOperationsState("unavailable");
    }
  }, []);

  const loadReview = useCallback(async (target = activeTarget.current) => {
    const current = ++loadIDs.current.review;
    setDigest(null);
    setReviewState("loading");
    try {
      const nextDigest = await loadProgramReviewDigest(target.id);
      if (!mounted.current || current !== loadIDs.current.review || activeTarget.current !== target) return;
      setDigest(nextDigest);
      setReviewState("live");
    } catch {
      if (!mounted.current || current !== loadIDs.current.review || activeTarget.current !== target) return;
      setDigest(null);
      setReviewState("unavailable");
    }
  }, []);

  const reloadRecord = useCallback(async () => {
    await Promise.all([loadAggregate(), loadOperations(), loadReview()]);
  }, [loadAggregate, loadOperations, loadReview]);

  useEffect(() => {
    if (startedTargetID.current === programID) return;
    startedTargetID.current = programID;
    const target = activeTarget.current;
    setAggregate(null);
    setOperations(null);
    setDigest(null);
    setAggregateState("loading");
    setOperationsState("loading");
    setReviewState("loading");
    void loadAggregate(target);
    void loadOperations(target);
    void loadReview(target);
  }, [programID, loadAggregate, loadOperations, loadReview]);

  async function applyUpdated(value: ProgramAggregate) {
    if (activeTarget.current !== renderTarget || value.program.id !== renderTarget.id) return;
    loadIDs.current.aggregate += 1;
    setAggregate(value);
    setAggregateState("live");
    await Promise.all([loadOperations(), loadReview()]);
  }

  function applyDigestUpdated(value: ReviewDigest) {
    if (activeTarget.current !== renderTarget || value.program_id !== renderTarget.id) return;
    setDigest(value);
  }

  const operationVersionMatches = Boolean(aggregate && operations && operations.program_version === aggregate.program.version);
  const operationsOutdated = operationsState === "live" && Boolean(aggregate && operations) && !operationVersionMatches;
  const expectedProjectionVersion = aggregate?.current_state?.projection_version ?? 0;
  const digestVersionMatches = Boolean(aggregate && digest && digest.current_program_version === aggregate.program.version && digest.current_projection_version === expectedProjectionVersion);
  const reviewOutdated = reviewState === "live" && Boolean(aggregate && digest) && !digestVersionMatches;
  const mutationsReady = operationsState === "live" && Boolean(operations?.authority_available) && operationVersionMatches && reviewState === "live" && Boolean(digest) && digestVersionMatches;
  const canAcknowledgeReview = mutationsReady && Boolean(operations?.operations.some((operation) => operation.command === "program.review.accept" && operation.can_act));
  const displayedOperations: ProgramOperations = operations
    ? {
        ...operations,
        operations: mutationsReady ? operations.operations : operations.operations.map((operation) => ({
          ...operation,
          can_act: false,
          reason: reviewState === "unavailable"
            ? "Program review status must be available before this change can be made."
            : "Current responsibilities must be available before this change can be made.",
        })),
      }
    : {
        program_id: programID,
        program_version: aggregate?.program.version ?? 0,
        authority_available: false,
        generated_at: "",
        operations: [],
      };

  return <section className="program-record-workspace" aria-label="Program record">
    <button aria-label="Back to Programs" className="text-button program-record-back" type="button" onClick={onBack}>← Back to Programs</button>
    {aggregateState === "loading" && !aggregate && <div className="workspace-loading" aria-live="polite" aria-busy="true">Loading Program record…</div>}
    {aggregateState === "unavailable" && !aggregate && <EmptyState kind="unavailable" label="Program" title="Program record could not be loaded" description="The current Program information is unavailable. Retry the Program record before relying on its status or evidence." action="Retry Program record" onAction={() => void loadAggregate()}/>}
    {aggregate && <>
      <header className="program-record-header">
        <div><span className="program-kicker">{aggregate.program.code} · {aggregate.program.owning_function}</span><h1>{aggregate.program.name}</h1><p>{aggregate.program.jurisdiction || "Jurisdiction not recorded"}</p></div>
        <dl><div><dt>Operating status</dt><dd>{statusLabel(aggregate.program.status)}</dd></div><div><dt>Calculated state</dt><dd>{aggregate.state_label}</dd></div><div><dt>Record version</dt><dd>{aggregate.program.version}</dd></div></dl>
      </header>
      {operationsState === "loading" && <div className="inline-notice" role="status"><strong>Checking current responsibilities.</strong> Program values remain visible while the current responsibility route is loaded.</div>}
      {operationsState === "unavailable" && <div className="inline-notice" role="status"><strong>Program responsibilities could not be checked.</strong> Program values and stored owners remain visible, but changes are disabled until the current responsibility route is available. <button className="text-button" type="button" onClick={() => void loadOperations()}>Retry responsibilities</button></div>}
      {operations && !operations.authority_available && <div className="inline-notice" role="status"><strong>Responsibilities are temporarily unavailable.</strong> Program values and stored owners remain visible, but changes are disabled until authority routing recovers. <button className="text-button" type="button" onClick={() => void loadOperations()}>Retry responsibilities</button></div>}
      {operationsOutdated && <div className="inline-notice" role="status"><strong>Program responsibilities are out of date.</strong> Program values remain visible, but changes are disabled until responsibilities match Program version {aggregate.program.version}. <button className="text-button" type="button" onClick={() => void reloadRecord()}>Reload Program data</button></div>}
      {reviewState === "loading" && <div className="inline-notice" role="status"><strong>Checking the current review status.</strong> Program values remain visible while the latest review position is loaded.</div>}
      {reviewState === "unavailable" && <div className="inline-notice" role="status"><strong>Program review status could not be checked.</strong> Program values and calculated status remain visible, but changes are disabled until the current review position is available. <button className="text-button" type="button" onClick={() => void loadReview()}>Retry review status</button></div>}
      {reviewOutdated && <div className="inline-notice" role="status"><strong>Program review status is out of date.</strong> Program values and review history remain visible, but changes are disabled until review status matches Program version {aggregate.program.version} and calculated-status version {expectedProjectionVersion}. <button className="text-button" type="button" onClick={() => void loadReview()}>Reload review status</button></div>}
      {digest
        ? <ProgramCurrentPosition aggregate={aggregate} operations={displayedOperations} digest={digest}/>
        : <section className="program-current-position" aria-labelledby="program-current-position-heading"><div><span className="eyebrow">Current position</span><h2 id="program-current-position-heading">{aggregate.state_label}</h2><div className="program-position-reasons"><h3>Why this status</h3><ul>{(aggregate.current_state?.reasons ?? []).map((reason) => <li key={`${reason.code}-${reason.object_id ?? ""}`}>{reason.summary}</li>)}</ul></div></div><div className="program-readonly-next"><strong>Changes are disabled</strong><span>Retry the Program review status before making a change.</span></div></section>}
      <section className="program-record-grid">
        <article className="program-record-panel" id="program-review-panel">{digest ? <ProgramReviewDigest aggregate={aggregate} initialDigest={digest} canAcknowledge={canAcknowledgeReview} onDigestUpdated={applyDigestUpdated}/> : <div className="inline-notice">Review actions are disabled until the current review status is available.</div>}</article>
		<ProgramStatusPanel aggregate={aggregate} operations={displayedOperations.operations} onUpdated={(value) => void applyUpdated(value)} onReload={() => void reloadRecord()}/>
		<ProgramDetailsPanel aggregate={aggregate} operations={displayedOperations.operations} onUpdated={(value) => void applyUpdated(value)} onReload={() => void reloadRecord()}/>
		<ProgramRequirementsPanel aggregate={aggregate} operations={displayedOperations.operations} onUpdated={(value) => void applyUpdated(value)} onReload={() => void reloadRecord()}/>
		<ProgramSafeguardsPanel aggregate={aggregate} operations={displayedOperations.operations} onUpdated={(value) => void applyUpdated(value)} onReload={() => void reloadRecord()}/>
		<ProgramEvidencePanel aggregate={aggregate} operations={displayedOperations.operations} actorPrincipalID={actorPrincipalID} canConfigureSources={canConfigureSources && mutationsReady} canOperate={mutationsReady} onUpdated={(value) => void applyUpdated(value)} onReload={() => void reloadRecord()}/>
		<ProgramIssuesPanel aggregate={aggregate} canCreateIssue={mutationsReady} onOpenMatter={onOpenMatter}/>
      </section>
    </>}
  </section>;
}
