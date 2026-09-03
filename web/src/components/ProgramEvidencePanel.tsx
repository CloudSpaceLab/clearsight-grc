import { useEffect, useMemo, useState } from "react";
import type { FormEvent } from "react";
import { loadEvidenceSources } from "../api";
import { addProgramEvidenceContract, recordProgramEvidenceAssessment, reviseProgramEvidenceContract, transitionProgramEvidenceContract } from "../programOperationsApi";
import type { ProgramOperation } from "../programOperationsApi";
import type { EvidenceSource, ProgramAggregate, RecordResponsibleParty } from "../types";

type Props = {
  aggregate: ProgramAggregate;
  operations: ProgramOperation[];
	responsibleParties?: RecordResponsibleParty[];
  actorPrincipalID: string;
  canConfigureSources: boolean;
  canOperate?: boolean;
  onUpdated: (value: ProgramAggregate) => void;
  onReload: () => void;
  onOpenMatter?: (matterID: string) => void;
};
type Mode = "define" | "assess" | null;
type ResourceAction = { kind: "edit" | "transition"; contractID: string } | null;

function futureDate(days: number) { return new Date(Date.now() + days * 86400000).toISOString().slice(0, 10); }
function statusLabel(value: string) { return value.replaceAll("_", " ").toLowerCase().replace(/(^|\s)\S/g, (letter) => letter.toUpperCase()); }
function lines(value: string) { return value.split(/\r?\n/).map((line) => line.trim()).filter(Boolean); }

export function ProgramEvidencePanel({ aggregate, operations, responsibleParties = [], canOperate = true, onUpdated, onReload }: Props) {
  const defineOperation = operations.find((value) => value.command === "program.evidence.define");
  const [sources, setSources] = useState<EvidenceSource[]>([]);
  const [sourcesState, setSourcesState] = useState<"loading" | "live" | "unavailable">("loading");
	const activeSources = sources.filter((source) => source.status === "ACTIVE");
  const activeContracts = aggregate.evidence_contracts.filter((contract) => contract.status === "ACTIVE");
  const assessableContracts = activeContracts.filter((contract) => operations.some((operation) => operation.command === "program.evidence.assess" && operation.subresource_id === contract.id && operation.can_act));
  const [mode, setMode] = useState<Mode>(null);
  const [resourceAction, setResourceAction] = useState<ResourceAction>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const targetOptions = useMemo(() => [
    ...aggregate.requirements.filter((value) => value.status === "APPROVED").map((value) => ({ value: `requirement:${value.id}`, label: `Requirement · ${value.code} · ${value.title}` })),
    ...aggregate.control_implementations.map((value) => ({ value: `safeguard:${value.id}`, label: `Safeguard · ${value.name}` })),
  ], [aggregate.requirements, aggregate.control_implementations]);
  const [target, setTarget] = useState(targetOptions[0]?.value ?? "");
  const [code, setCode] = useState(""); const [name, setName] = useState(""); const [claim, setClaim] = useState("");
  const [sourceIDs, setSourceIDs] = useState<string[]>([]); const [population, setPopulation] = useState("");
  const [freshnessDays, setFreshnessDays] = useState(30); const [coveragePercent, setCoveragePercent] = useState(100);
  const [independenceRequired, setIndependenceRequired] = useState(false); const [contradictionPolicy, setContradictionPolicy] = useState("REVIEW"); const [failureAction, setFailureAction] = useState("MATTER");
  const [contractID, setContractID] = useState(assessableContracts[0]?.id ?? "");
  const [conclusion, setConclusion] = useState("SUPPORTED"); const [assessmentCoverage, setAssessmentCoverage] = useState(100);
  const [basis, setBasis] = useState(""); const [references, setReferences] = useState(""); const [validUntil, setValidUntil] = useState(futureDate(30));
  const [rationale, setRationale] = useState(""); const [transitionTarget, setTransitionTarget] = useState("");

  const operationFor = (command: string, subresourceID: string) => operations.find((value) => value.command === command && value.subresource_id === subresourceID);

  useEffect(() => {
    let active = true; setSourcesState("loading");
    void loadEvidenceSources(aggregate.program.legal_entity_id).then((values) => { if (active) { setSources(values); setSourcesState("live"); } })
      .catch(() => { if (active) setSourcesState("unavailable"); });
    return () => { active = false; };
  }, [aggregate.program.id, aggregate.program.legal_entity_id]);

  useEffect(() => {
    if (!canOperate) setMode(null);
  }, [canOperate]);

  function beginDefine() {
    if (!canOperate) return;
    setMode("define"); setError(""); setTarget(targetOptions[0]?.value ?? ""); setCode(""); setName(""); setClaim(""); setSourceIDs([]); setPopulation(""); setFreshnessDays(30); setCoveragePercent(100); setIndependenceRequired(false); setContradictionPolicy("REVIEW"); setFailureAction("MATTER");
  }
  function beginAssess(targetContractID = assessableContracts[0]?.id ?? "") {
    if (!canOperate) return;
    setMode("assess"); setError(""); setContractID(targetContractID); setConclusion("SUPPORTED"); setAssessmentCoverage(100); setBasis(""); setReferences(""); setValidUntil(futureDate(30));
  }
  function beginResourceAction(kind: "edit" | "transition", contractID: string) {
    if (!canOperate) return;
    const contract = aggregate.evidence_contracts.find((value) => value.id === contractID);
    if (!contract) return;
    setMode(null); setResourceAction({ kind, contractID }); setError(""); setRationale("");
    if (kind === "edit") {
      setName(contract.name); setClaim(contract.claim); setSourceIDs(contract.acceptable_source_ids ?? []);
      setPopulation(String(contract.population_scope?.description ?? "")); setFreshnessDays(Math.max(1, Math.round(contract.freshness_minutes / 1440)));
      setCoveragePercent(contract.minimum_coverage * 100); setIndependenceRequired(Boolean(contract.independence_required));
      setContradictionPolicy(contract.contradiction_policy ?? "REVIEW"); setFailureAction(contract.failure_action ?? "MATTER");
    } else {
      setTransitionTarget(operationFor("program.evidence.transition", contractID)?.allowed_targets?.[0] ?? "");
    }
  }
  function toggleSource(id: string) { setSourceIDs((current) => current.includes(id) ? current.filter((value) => value !== id) : [...current, id]); }

  async function saveDefinition(event: FormEvent) {
    event.preventDefault();
    if (!canOperate) return;
    setBusy(true); setError("");
    const [kind, targetID] = target.split(":", 2);
    try {
      const value = await addProgramEvidenceContract(aggregate.program.id, aggregate.program.version, {
        requirementID: kind === "requirement" ? targetID : undefined,
        controlImplementationID: kind === "safeguard" ? targetID : undefined,
        code, name, claim, acceptableSourceIDs: sourceIDs,
        populationScope: { description: population.trim() },
        freshnessMinutes: Math.round(freshnessDays * 1440), minimumCoverage: coveragePercent / 100,
        independenceRequired, contradictionPolicy, failureAction, status: "DRAFT",
      });
      onUpdated(value); setMode(null);
    } catch (value) { setError(value instanceof Error ? value.message : "The evidence check could not be saved."); }
    finally { setBusy(false); }
  }

  async function saveAssessment(event: FormEvent) {
    event.preventDefault();
    if (!canOperate) return;
    setBusy(true); setError("");
    try {
      const value = await recordProgramEvidenceAssessment(aggregate.program.id, aggregate.program.version, {
        contractID, conclusion, coverage: assessmentCoverage / 100,
        basis: { summary: basis.trim(), evidence_references: lines(references) },
        validUntil: validUntil ? new Date(`${validUntil}T23:59:59.999Z`).toISOString() : undefined,
        assessedAt: new Date().toISOString(),
      });
      onUpdated(value); setMode(null);
    } catch (value) { setError(value instanceof Error ? value.message : "The evidence result could not be saved."); }
    finally { setBusy(false); }
  }

  async function saveResourceAction(event: FormEvent) {
    event.preventDefault();
    if (!canOperate || !resourceAction) return;
    const contract = aggregate.evidence_contracts.find((value) => value.id === resourceAction.contractID);
    if (!contract?.version) return;
    setBusy(true); setError("");
    try {
      const value = resourceAction.kind === "edit"
        ? await reviseProgramEvidenceContract(aggregate.program.id, contract.id, aggregate.program.version, contract.version, {
          name, claim, acceptableSourceIDs: sourceIDs, populationScope: { description: population.trim() },
          freshnessMinutes: Math.round(freshnessDays * 1440), minimumCoverage: coveragePercent / 100,
          independenceRequired, contradictionPolicy, failureAction, rationale,
        })
        : await transitionProgramEvidenceContract(aggregate.program.id, contract.id, aggregate.program.version, contract.version, transitionTarget, rationale);
      onUpdated(value); setResourceAction(null);
    } catch (value) { setError(value instanceof Error ? value.message : "The evidence check change could not be saved."); }
    finally { setBusy(false); }
  }

  return <article className="program-record-panel program-evidence-panel" id="program-evidence-panel">
    <div className="program-panel-heading"><div><span className="eyebrow">Evidence</span><h2>Evidence checks and results</h2></div><div className="program-panel-actions">
      {canOperate && defineOperation?.can_act && targetOptions.length > 0 && <button className="secondary-button" type="button" onClick={beginDefine}>Define evidence check</button>}
      {canOperate && assessableContracts.length > 0 && <button className="secondary-button" type="button" onClick={() => beginAssess()}>Record evidence result</button>}
    </div></div>

    {!canOperate && <p className="program-operation-reason">Evidence changes are disabled until current Program responsibilities are available. Existing evidence checks and results remain visible.</p>}
    {canOperate && aggregate.evidence_contracts.length > 0 && activeContracts.length === 0 && <p className="program-operation-reason">No evidence result can be recorded yet. A current reviewer must activate a Draft evidence check first.</p>}
    {canOperate && activeContracts.length > 0 && assessableContracts.length === 0 && <p className="program-operation-reason">No Active evidence check is assigned to you for assessment. Each check shows its current reviewer below.</p>}

    {aggregate.evidence_contracts.length ? <div className="program-evidence-list">{aggregate.evidence_contracts.map((contract) => {
      const assessments = aggregate.evidence_assessments.filter((value) => value.contract_id === contract.id).sort((left, right) => right.assessed_at.localeCompare(left.assessed_at));
      const latest = assessments[0]; const current = Boolean(latest?.valid_until && new Date(latest.valid_until).getTime() >= Date.now());
      const sourceNames = (contract.acceptable_source_ids ?? []).map((id) => sources.find((source) => source.id === id)?.name ?? "Source name unavailable");
      const reviewerLabel = (assessmentID: string, reviewerID?: string, fallback = "Reviewer name unavailable") => responsibleParties.find((party) => party.scope === "EVIDENCE_ASSESSMENT" && party.subresource_id === assessmentID && party.responsibility === "REVIEWER")?.display_name ?? operations.flatMap((operation) => operation.candidates ?? []).find((candidate) => candidate.id === reviewerID)?.display_name ?? fallback;
      const reviseOperation = operationFor("program.evidence.revise", contract.id);
      const transitionOperation = operationFor("program.evidence.transition", contract.id);
      const assessmentOperation = operationFor("program.evidence.assess", contract.id);
      return <section className="program-evidence-card" key={contract.id}>
        <div><span>{contract.code} · {statusLabel(contract.status)}</span><h3>{contract.name}</h3><p>{contract.claim}</p><small>Required coverage {Math.round(contract.minimum_coverage * 100)}% · maximum age {Math.round(contract.freshness_minutes / 1440)} days</small>{sourceNames.length > 0 && <small>Accepted sources: {sourceNames.join(", ")}</small>}<div className="program-panel-actions">
          {canOperate && reviseOperation?.can_act && <button className="text-button" type="button" onClick={() => beginResourceAction("edit", contract.id)}>Edit {contract.name}</button>}
          {canOperate && transitionOperation?.can_act && <button className="text-button" type="button" onClick={() => beginResourceAction("transition", contract.id)}>Review {contract.name} status</button>}
          {canOperate && assessmentOperation?.can_act && <button className="text-button" type="button" onClick={() => beginAssess(contract.id)}>Record result for {contract.name}</button>}
        </div>{!transitionOperation?.can_act && transitionOperation?.reason && <small>{transitionOperation.reason}</small>}{contract.status === "ACTIVE" && !assessmentOperation?.can_act && assessmentOperation?.reason && <small>{assessmentOperation.reason}</small>}</div>
        <div className={latest ? "program-assessment-result" : "program-assessment-result empty"}>{latest ? <><strong>{statusLabel(latest.conclusion)}</strong><span>{Math.round(latest.coverage * 100)}% coverage · {current ? `current until ${latest.valid_until!.slice(0, 10)}` : latest.valid_until ? `expired ${latest.valid_until.slice(0, 10)}` : "validity not recorded"}</span><small>Assessed {latest.assessed_at.slice(0, 10)} by {reviewerLabel(latest.id, latest.assessed_by)}</small></> : <><strong>No evidence result recorded</strong><span>A reviewer must assess the evidence before this check can support the Program outcome.</span></>}</div>
        {assessments.length > 0 && <details><summary>View evidence result history ({assessments.length})</summary><p>Showing {Math.min(assessments.length, 20)} of {assessments.length} stored results for Program version {aggregate.program.version}.</p><ol>{assessments.slice(0, 20).map((assessment) => <li key={assessment.id}><strong>{statusLabel(assessment.conclusion)}</strong><span>{Math.round(assessment.coverage * 100)}% coverage · assessed {assessment.assessed_at.slice(0, 10)} by {reviewerLabel(assessment.id, assessment.assessed_by, "Recorded reviewer unavailable")}</span><small>Sources: {sourceNames.length ? sourceNames.join(", ") : "No accepted source label is recorded"}</small>{typeof assessment.basis?.summary === "string" && assessment.basis.summary && <p>{assessment.basis.summary}</p>}</li>)}</ol>{assessments.length > 20 && <p>Older evidence results are not shown. The Program record contains {assessments.length - 20} additional results.</p>}</details>}
      </section>;
    })}</div> : <div className="program-empty-state"><strong>No evidence checks are defined</strong><p>Define what must be proved, acceptable sources, freshness and required coverage for an applicable requirement or safeguard.</p></div>}

    {mode === "define" && <form className="program-operation-form" onSubmit={(event) => void saveDefinition(event)}>
      <label className="wide"><span>Evidence applies to</span><select required value={target} onChange={(event) => setTarget(event.target.value)}>{targetOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}</select></label>
      <label><span>Evidence code</span><input required value={code} onChange={(event) => setCode(event.target.value)}/></label>
      <label><span>Evidence check name</span><input required value={name} onChange={(event) => setName(event.target.value)}/></label>
      <label className="wide"><span>What must the evidence prove?</span><textarea required value={claim} onChange={(event) => setClaim(event.target.value)}/></label>
      <fieldset className="program-source-options wide"><legend>Acceptable sources</legend>{sourcesState === "loading" && <span>Loading current evidence sources…</span>}{sourcesState === "unavailable" && <span>Connected sources could not be loaded. You can save a manual check without a connected source, or reload before selecting one.</span>}{activeSources.map((source) => <label key={source.id}><input aria-label={source.name} type="checkbox" checked={sourceIDs.includes(source.id)} onChange={() => toggleSource(source.id)}/><span>{source.name}</span><small>{source.code} · {statusLabel(source.health)}</small></label>)}</fieldset>
      <label className="wide"><span>Population covered</span><textarea value={population} onChange={(event) => setPopulation(event.target.value)}/></label>
      <label><span>Maximum evidence age (days)</span><input required min="1" type="number" value={freshnessDays} onChange={(event) => setFreshnessDays(Number(event.target.value))}/></label>
      <label><span>Required population coverage (%)</span><input required min="0" max="100" step="0.1" type="number" value={coveragePercent} onChange={(event) => setCoveragePercent(Number(event.target.value))}/></label>
      <label className="program-check-label"><input type="checkbox" checked={independenceRequired} onChange={(event) => setIndependenceRequired(event.target.checked)}/><span>Independent review required</span></label>
      <label><span>Contradiction handling</span><select value={contradictionPolicy} onChange={(event) => setContradictionPolicy(event.target.value)}><option value="HOLD">Hold the check</option><option value="REVIEW">Require review</option><option value="FAIL">Fail the check</option></select></label>
      <label><span>If evidence fails</span><select value={failureAction} onChange={(event) => setFailureAction(event.target.value)}><option value="MATTER">Create a linked issue</option></select></label>
      <p>Create a linked issue when a result does not support the claim.</p>
      {error && <p className="program-form-error wide" role="alert">{error} <button className="text-button" type="button" onClick={onReload}>Reload Program</button></p>}
      <div className="program-form-actions wide"><button className="primary-button" disabled={busy || !target} type="submit">{busy ? "Saving…" : "Save evidence check"}</button><button className="text-button" type="button" onClick={() => setMode(null)}>Cancel</button></div>
    </form>}

    {mode === "assess" && <form className="program-operation-form" onSubmit={(event) => void saveAssessment(event)}>
      <label className="wide"><span>Evidence check</span><select required value={contractID} onChange={(event) => setContractID(event.target.value)}>{assessableContracts.map((contract) => <option key={contract.id} value={contract.id}>{contract.code} · {contract.name}</option>)}</select></label>
      <label><span>Conclusion</span><select value={conclusion} onChange={(event) => setConclusion(event.target.value)}><option value="SUPPORTED">Supported</option><option value="PARTIALLY_SUPPORTED">Partly supported</option><option value="UNSUPPORTED">Unsupported</option><option value="CONTRADICTED">Contradicted</option><option value="INDETERMINATE">Indeterminate</option><option value="EXPIRED">Expired</option></select></label>
      <label><span>Population coverage (%)</span><input required min="0" max="100" step="0.1" type="number" value={assessmentCoverage} onChange={(event) => setAssessmentCoverage(Number(event.target.value))}/></label>
      <label className="wide"><span>Assessment basis</span><textarea required value={basis} onChange={(event) => setBasis(event.target.value)}/></label>
      <label className="wide"><span>Evidence references</span><textarea value={references} onChange={(event) => setReferences(event.target.value)} placeholder="Add one source, artifact or report reference per line."/></label>
      <label><span>Result valid until</span><input type="date" value={validUntil} onChange={(event) => setValidUntil(event.target.value)}/></label>
      {error && <p className="program-form-error wide" role="alert">{error} <button className="text-button" type="button" onClick={onReload}>Reload Program</button></p>}
      <div className="program-form-actions wide"><button className="primary-button" disabled={busy || !contractID} type="submit">{busy ? "Saving…" : "Save evidence result"}</button><button className="text-button" type="button" onClick={() => setMode(null)}>Cancel</button></div>
    </form>}

    {resourceAction && (() => {
      const contract = aggregate.evidence_contracts.find((value) => value.id === resourceAction.contractID);
      if (!contract) return null;
      const transitionOperation = operationFor("program.evidence.transition", contract.id);
      return <form className="program-operation-form" onSubmit={(event) => void saveResourceAction(event)}>
        <div className="wide"><strong>{resourceAction.kind === "edit" ? `Edit ${contract.name}` : `Review ${contract.name} status`}</strong></div>
        {resourceAction.kind === "edit" && <>
          <label><span>Evidence check name</span><input required value={name} onChange={(event) => setName(event.target.value)}/></label>
          <label className="wide"><span>What must the evidence prove?</span><textarea required value={claim} onChange={(event) => setClaim(event.target.value)}/></label>
          <fieldset className="program-source-options wide"><legend>Acceptable sources</legend>{sourcesState === "loading" && <span>Loading current evidence sources…</span>}{sourcesState === "unavailable" && <span>Connected sources could not be loaded. Existing source references are preserved; you can save a manual check or reload before changing them.</span>}{activeSources.map((source) => <label key={source.id}><input aria-label={source.name} type="checkbox" checked={sourceIDs.includes(source.id)} onChange={() => toggleSource(source.id)}/><span>{source.name}</span><small>{source.code} · {statusLabel(source.health)}</small></label>)}</fieldset>
          <label className="wide"><span>Population covered</span><textarea value={population} onChange={(event) => setPopulation(event.target.value)}/></label>
          <label><span>Maximum evidence age (days)</span><input required min="1" type="number" value={freshnessDays} onChange={(event) => setFreshnessDays(Number(event.target.value))}/></label>
          <label><span>Required population coverage (%)</span><input required min="0" max="100" step="0.1" type="number" value={coveragePercent} onChange={(event) => setCoveragePercent(Number(event.target.value))}/></label>
          <label className="program-check-label"><input type="checkbox" checked={independenceRequired} onChange={(event) => setIndependenceRequired(event.target.checked)}/><span>Independent review required</span></label>
          <label><span>Contradiction handling</span><select value={contradictionPolicy} onChange={(event) => setContradictionPolicy(event.target.value)}><option value="HOLD">Hold the check</option><option value="REVIEW">Require review</option><option value="FAIL">Fail the check</option></select></label>
          <label><span>If evidence fails</span><select value={failureAction} onChange={(event) => setFailureAction(event.target.value)}><option value="MATTER">Create a linked issue</option></select></label>
        </>}
        {resourceAction.kind === "transition" && <label><span>Evidence check status</span><select required value={transitionTarget} onChange={(event) => setTransitionTarget(event.target.value)}>{transitionOperation?.allowed_targets?.map((target) => <option key={target} value={target}>{statusLabel(target)}</option>)}</select></label>}
        {resourceAction.kind === "edit" && contract.status === "ACTIVE" && <p className="program-operation-reason wide">Saving changed source, freshness, coverage or contradiction rules returns this evidence check to Draft. A current reviewer must activate it again before it supports the Program.</p>}
        <label className="wide"><span>Reason for this change</span><textarea required value={rationale} onChange={(event) => setRationale(event.target.value)}/></label>
        {error && <p className="program-form-error wide" role="alert">{error} <button className="text-button" type="button" onClick={onReload}>Reload Program</button></p>}
        <div className="program-form-actions wide"><button className="primary-button" disabled={busy || !rationale || (resourceAction.kind === "transition" && !transitionTarget)} type="submit">{busy ? "Saving…" : resourceAction.kind === "edit" ? "Save evidence check changes" : "Save evidence check status"}</button><button className="text-button" type="button" onClick={() => setResourceAction(null)}>Cancel</button></div>
      </form>;
    })()}

    {!defineOperation?.can_act && defineOperation?.reason && <p className="program-operation-reason">{defineOperation.reason}</p>}
  </article>;
}
