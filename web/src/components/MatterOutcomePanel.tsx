import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import { loadEvidenceSources } from "../api";
import { apiErrorKind } from "../http";
import { defineMatterOutcomeCheck, retireMatterOutcomeCheck, supersedeMatterOutcomeCheck } from "../matterOperationsApi";
import type { MatterOperation } from "../matterOperationsApi";
import { recordVerificationResult, transitionMatter } from "../continuityCommands";
import type { EvidenceSource, MatterAggregate, RecordResponsibleParty, VerificationContract } from "../types";

type Props = {
  aggregate: MatterAggregate;
  operations: MatterOperation[];
  responsibleParties?: RecordResponsibleParty[];
  onUpdated: (value: MatterAggregate) => void | Promise<void>;
  onReload: () => void;
};

type Active = { kind: "define" } | { kind: "supersede"; contractID: string } | { kind: "retire"; contractID: string } | { kind: "result"; contractID: string } | { kind: "status" } | null;

function operationFor(operations: MatterOperation[], command: string, subresourceID?: string) {
  const matching = operations.filter((operation) => operation.command === command && (subresourceID === undefined || operation.subresource_id === subresourceID));
  return matching.find((operation) => operation.can_act) ?? matching[0];
}

function lines(value: string) {
  return value.split("\n").map((item) => item.trim()).filter(Boolean);
}

function resultLabel(result?: string) {
  switch (result) {
    case "PASS": return "Outcome confirmed";
    case "FAIL": return "Outcome not achieved";
    case "INCONCLUSIVE": return "More evidence needed";
    default: return "Not checked yet";
  }
}

function statusLabel(value: string) {
  switch (value) {
    case "TRIAGE": return "Initial review";
    case "DECISION_REQUIRED": return "Decision needed";
    case "ACTION_IN_PROGRESS": return "Work in progress";
    case "VERIFICATION": return "Outcome check";
    case "CLOSED": return "Closed";
    case "CANCELLED": return "Cancelled";
    default: return value.replaceAll("_", " ").toLowerCase().replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
  }
}

function failureLabel(value: string) {
  switch (value) {
    case "REOPEN": return "Reopen this issue for corrective work";
    case "CREATE_MATTER": return "Create a follow-up issue";
    case "ESCALATE": return "Escalate to the current escalation owner";
    case "BLOCK_CLOSE": return "Keep this issue open";
    default: return value;
  }
}

function contractText(value: Record<string, unknown> | undefined, keys: string[], includeLegacyScalars = true) {
  for (const key of keys) {
    const entry = value?.[key];
    if (typeof entry === "string" && entry.trim()) return entry.trim();
  }
  if (includeLegacyScalars) {
    const uuid = /^[0-9a-f]{8}-[0-9a-f-]{27,}$/i;
    const entries = Object.entries(value ?? {}).filter(([key, entry]) => {
      if (/(^|_)(id|ids|tenant|principal|actor)($|_)/i.test(key)) return false;
      return typeof entry === "number" || typeof entry === "boolean" || (typeof entry === "string" && entry.trim() !== "" && !uuid.test(entry.trim()));
    }).slice(0, 4);
    if (entries.length) return entries.map(([key, entry]) => {
      const label = key.replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
      const displayed = typeof entry === "boolean" ? (entry ? "Yes" : "No") : String(entry).slice(0, 160);
      return `${label}: ${displayed}`;
    }).join(" · ");
  }
  return "Not recorded";
}

function contractField(value: Record<string, unknown> | undefined, keys: string[]) {
  for (const key of keys) {
    const entry = value?.[key];
    if (typeof entry === "string" && entry.trim()) return entry.trim();
  }
  return "";
}

function observationPeriodLabel(minutes: number) {
  if (minutes === 0) return "Check immediately";
  if (minutes % 1440 === 0) return `${minutes / 1440} ${minutes === 1440 ? "day" : "days"}`;
  if (minutes % 60 === 0) return `${minutes / 60} ${minutes === 60 ? "hour" : "hours"}`;
  return `${minutes} minutes`;
}

export function MatterOutcomePanel({ aggregate, operations, responsibleParties = [], onUpdated, onReload }: Props) {
  const defineOperation = operationFor(operations, "matter.outcome.define");
  const transitionOperation = operationFor(operations, "matter.transition");
  const [active, setActive] = useState<Active>(null);
  const [expectedOutcome, setExpectedOutcome] = useState("");
  const [actionID, setActionID] = useState("");
  const [scopeCovered, setScopeCovered] = useState("");
  const [measurementMethod, setMeasurementMethod] = useState("");
  const [currentBaseline, setCurrentBaseline] = useState("");
  const [successThreshold, setSuccessThreshold] = useState("");
  const [measurementSourceID, setMeasurementSourceID] = useState("");
  const [observationDays, setObservationDays] = useState("1");
  const [reviewerCandidateID, setReviewerCandidateID] = useState("");
  const [failureResponse, setFailureResponse] = useState("");
  const [sources, setSources] = useState<EvidenceSource[]>([]);
  const [sourcesState, setSourcesState] = useState<"loading" | "live" | "unavailable">("loading");
	const activeSources = sources.filter((source) => source.status === "ACTIVE");
  const [result, setResult] = useState<"PASS" | "FAIL" | "INCONCLUSIVE">("PASS");
  const [observations, setObservations] = useState("");
  const [evidenceReferences, setEvidenceReferences] = useState("");
  const [rationale, setRationale] = useState("");
  const [target, setTarget] = useState("");
  const [saving, setSaving] = useState(false);
  const [notice, setNotice] = useState("");
  const [error, setError] = useState("");
  const [conflict, setConflict] = useState(false);

  useEffect(() => {
    let current = true;
    setSourcesState("loading");
    void loadEvidenceSources(aggregate.matter.legal_entity_id)
      .then((values) => {
        if (!current) return;
        setSources(values.filter((value) => !value.legal_entity_id || value.legal_entity_id === aggregate.matter.legal_entity_id));
        setSourcesState("live");
      })
      .catch(() => { if (current) setSourcesState("unavailable"); });
    return () => { current = false; };
  }, [aggregate.matter.id, aggregate.matter.legal_entity_id]);

  function beginDefine() {
    setExpectedOutcome("");
    setActionID(aggregate.actions[0]?.id ?? "");
    setScopeCovered("");
    setMeasurementMethod("");
    setCurrentBaseline("");
    setSuccessThreshold("");
    setMeasurementSourceID("");
    setObservationDays("1");
    setReviewerCandidateID(defineOperation?.candidates?.[0]?.id ?? defineOperation?.assigned_to?.id ?? "");
    setFailureResponse("");
    setRationale("");
    setError("");
    setNotice("");
    setActive({ kind: "define" });
  }

  function beginResult(contract: VerificationContract) {
    setResult("PASS");
    setObservations("");
    setEvidenceReferences("");
    setRationale("");
    setError("");
    setNotice("");
    setActive({ kind: "result", contractID: contract.id });
  }

  function beginSupersede(contract: VerificationContract) {
    const operation = operationFor(operations, "matter.outcome.supersede", contract.id);
    setExpectedOutcome(contract.expected_outcome);
    setActionID(contract.action_id ?? "");
    setScopeCovered(contractField(contract.scope, ["description", "population"]));
    setMeasurementMethod(contractField(contract.scope, ["measurement_method"]));
    setCurrentBaseline(contractField(contract.baseline, ["description", "current_state", "summary"]));
    setSuccessThreshold(contractField(contract.threshold, ["success_condition", "description"]));
    setMeasurementSourceID(activeSources.some((source) => source.id === contract.measurement_source_id) ? contract.measurement_source_id ?? "" : "");
    setObservationDays(String(contract.observation_period_minutes / 1440));
    setReviewerCandidateID((operation?.candidates ?? []).some((candidate) => candidate.id === contract.authority_principal_id) ? contract.authority_principal_id ?? "" : operation?.candidates?.[0]?.id ?? "");
    setFailureResponse(contract.failure_response ?? "");
    setRationale("");
    setError("");
    setNotice("");
    setActive({ kind: "supersede", contractID: contract.id });
  }

  function beginRetire(contract: VerificationContract) {
    setRationale("");
    setError("");
    setNotice("");
    setActive({ kind: "retire", contractID: contract.id });
  }

  function beginStatus() {
    const allowed = transitionOperation?.allowed_targets ?? [];
    const preferred = aggregate.closure.ready && allowed.includes("CLOSED") ? "CLOSED" : allowed.find((value) => value !== "CLOSED") ?? "";
    setTarget(preferred);
    setRationale("");
    setError("");
    setNotice("");
    setActive({ kind: "status" });
  }

  function handleError(cause: unknown) {
    const isConflict = apiErrorKind(cause) === "conflict";
    setConflict(isConflict);
    setError(isConflict ? "This issue changed since you opened it. Your entries have been kept." : cause instanceof Error && cause.message ? cause.message : "The outcome workflow could not be updated.");
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!active) return;
    setSaving(true);
    setError("");
    setConflict(false);
    setNotice("");
    try {
      let updated: MatterAggregate;
      if (active.kind === "define") {
        updated = await defineMatterOutcomeCheck(aggregate.matter.id, aggregate.matter.version, {
          actionID: actionID || undefined,
          expectedOutcome: expectedOutcome.trim(),
          baseline: { description: currentBaseline.trim() },
          scope: { description: scopeCovered.trim(), measurement_method: measurementMethod.trim() },
          threshold: { success_condition: successThreshold.trim() },
          measurementSourceID: measurementSourceID || undefined,
          observationPeriodMinutes: Math.round(Number(observationDays) * 1440),
          reviewerCandidateID,
          failureResponse: failureResponse.trim(),
        });
      } else if (active.kind === "supersede") {
        updated = await supersedeMatterOutcomeCheck(aggregate.matter.id, active.contractID, aggregate.matter.version, {
          actionID: actionID || undefined,
          expectedOutcome: expectedOutcome.trim(),
          baseline: { description: currentBaseline.trim() },
          scope: { description: scopeCovered.trim(), measurement_method: measurementMethod.trim() },
          threshold: { success_condition: successThreshold.trim() },
          measurementSourceID: measurementSourceID || undefined,
          observationPeriodMinutes: Math.round(Number(observationDays) * 1440),
          reviewerCandidateID,
          failureResponse: failureResponse.trim(),
          rationale: rationale.trim(),
        });
      } else if (active.kind === "retire") {
        updated = await retireMatterOutcomeCheck(aggregate.matter.id, active.contractID, aggregate.matter.version, rationale.trim());
      } else if (active.kind === "result") {
        updated = await recordVerificationResult(aggregate.matter.id, aggregate.matter.version, {
          contractID: active.contractID,
          result,
          observations: observations.trim() ? { note: observations.trim() } : {},
          evidenceReferences: lines(evidenceReferences),
          rationale: rationale.trim(),
        });
      } else {
        updated = await transitionMatter(aggregate.matter.id, aggregate.matter.version, target, rationale.trim());
      }
      const completedKind = active.kind;
      await onUpdated(updated);
      setActive(null);
      setNotice(completedKind === "define" ? "Outcome check defined." : completedKind === "supersede" ? "Outcome check replaced. The previous version remains in history." : completedKind === "retire" ? "Outcome check ended. Its history remains available." : completedKind === "result" ? "Outcome result recorded." : `Issue status changed to ${statusLabel(target)}.`);
    } catch (cause) {
      handleError(cause);
    } finally {
      setSaving(false);
    }
  }

  const activeContract = active && "contractID" in active ? aggregate.verification_contracts.find((contract) => contract.id === active.contractID) : undefined;
  const activeDefinitionOperation = active?.kind === "supersede" ? operationFor(operations, "matter.outcome.supersede", active.contractID) : defineOperation;
  const allowedStatusTargets = (transitionOperation?.allowed_targets ?? []).filter((value) => value !== "CLOSED" || aggregate.closure.ready);

  return <article className="matter-record-panel matter-outcome-panel" id="matter-operation-matter.outcome.define">
    <div className="matter-record-section-heading">
      <div><span className="eyebrow">Outcome checks</span><h2>Independent results</h2></div>
      {defineOperation?.can_act && !active && <button className="secondary-button" type="button" onClick={beginDefine}>Define outcome check</button>}
    </div>
    {aggregate.verification_contracts.length ? <div className="matter-outcome-list">{aggregate.verification_contracts.map((contract) => {
      const results = aggregate.verification_results.filter((item) => item.contract_id === contract.id).sort((left, right) => right.observed_at.localeCompare(left.observed_at));
      const recorded = results[0];
      const recordOperation = operationFor(operations, "matter.outcome.record", contract.id);
      const supersedeOperation = operationFor(operations, "matter.outcome.supersede", contract.id);
      const retireOperation = operationFor(operations, "matter.outcome.retire", contract.id);
      const linkedAction = aggregate.actions.find((action) => action.id === contract.action_id);
      const sourceLabel = sources.find((source) => source.id === contract.measurement_source_id)?.name ?? (contract.measurement_source_id ? "Source name unavailable" : "Not recorded");
      const assignedReviewer = responsibleParties.find((party) => party.scope === "OUTCOME_CHECK" && party.subresource_id === contract.id && party.responsibility === "REVIEWER")?.display_name ?? operations.flatMap((operation) => operation.candidates ?? []).find((candidate) => candidate.id === contract.authority_principal_id)?.display_name;
      const reviewerLabel = (resultID: string, reviewerID?: string) => responsibleParties.find((party) => party.scope === "OUTCOME_RESULT" && party.subresource_id === resultID && party.responsibility === "REVIEWER")?.display_name ?? operations.flatMap((operation) => operation.candidates ?? []).find((candidate) => candidate.id === reviewerID)?.display_name ?? recordOperation?.assigned_to?.display_name ?? "Recorded reviewer unavailable";
      return <section className="matter-outcome-card" key={contract.id} aria-labelledby={`matter-outcome-${contract.id}`}>
        <div className="matter-action-heading"><div><h3 id={`matter-outcome-${contract.id}`}>{contract.expected_outcome}</h3>{linkedAction && <p>Checks the result of: {linkedAction.title}</p>}{contract.supersedes_contract_id && <p>This replacement is the current outcome check; the earlier version remains in this history.</p>}</div><span>{contract.status === "RETIRED" ? "Ended" : resultLabel(recorded?.result)}</span></div>
        <dl className="matter-outcome-meta">
          <div><dt>Scope covered</dt><dd>{contractText(contract.scope, ["description", "population"])}</dd></div>
          <div><dt>How the outcome is measured</dt><dd>{contractText(contract.scope, ["measurement_method"], false)}</dd></div>
          <div><dt>Current baseline</dt><dd>{contractText(contract.baseline, ["description", "current_state", "summary"])}</dd></div>
          <div><dt>Success threshold</dt><dd>{contractText(contract.threshold, ["success_condition", "description"])}</dd></div>
          <div><dt>Measurement source</dt><dd>{sourceLabel}</dd></div>
          <div><dt>Observation period</dt><dd>{observationPeriodLabel(contract.observation_period_minutes)}</dd></div>
          <div><dt>Independent reviewer</dt><dd>{assignedReviewer ?? recordOperation?.assigned_to?.display_name ?? defineOperation?.assigned_to?.display_name ?? (contract.authority_principal_id ? "Recorded reviewer unavailable" : "Reviewer not assigned")}</dd></div>
        </dl>
        {recorded?.rationale && <p className="matter-outcome-rationale"><strong>Recorded basis:</strong> {recorded.rationale}</p>}
        {results.length > 0 && <details><summary>View outcome result history ({results.length})</summary><p>Showing {Math.min(results.length, 20)} of {results.length} stored results for issue version {aggregate.matter.version}.</p><ol>{results.slice(0, 20).map((item) => <li key={item.id}><strong>{resultLabel(item.result)}</strong><span>Recorded {item.observed_at.slice(0, 10)} by {reviewerLabel(item.id, item.reviewer_principal_id)}</span>{item.rationale && <p>{item.rationale}</p>}</li>)}</ol>{results.length > 20 && <p>Older results are not shown. The issue record contains {results.length - 20} additional results.</p>}</details>}
        {contract.failure_response && <p className="matter-outcome-rationale"><strong>If not achieved:</strong> {failureLabel(contract.failure_response)}</p>}
        {!active && contract.status === "ACTIVE" && <div className="matter-form-actions">
          {recordOperation?.can_act && <button className="secondary-button" type="button" aria-label={`Record result for ${contract.expected_outcome}`} onClick={() => beginResult(contract)}>Record outcome result</button>}
          {supersedeOperation?.can_act && <button className="secondary-button" type="button" aria-label={`Replace outcome check for ${contract.expected_outcome}`} onClick={() => beginSupersede(contract)}>Replace outcome check</button>}
          {retireOperation?.can_act && <button className="text-button" type="button" aria-label={`End outcome check for ${contract.expected_outcome}`} onClick={() => beginRetire(contract)}>End outcome check</button>}
        </div>}
        {contract.status === "ACTIVE" && !recordOperation?.can_act && recordOperation?.reason && <p className="matter-operation-reason">{recordOperation.reason}</p>}
        {contract.status === "ACTIVE" && !supersedeOperation?.can_act && supersedeOperation?.reason && <p className="matter-operation-reason">{supersedeOperation.reason}</p>}
      </section>;
    })}</div> : <div className="matter-record-attention"><strong>No outcome check has been defined</strong><p>Define the result that must be independently confirmed before this issue can close.</p></div>}

    {aggregate.closure.ready ? <div className="matter-closure-state ready"><strong>Ready to close</strong><p>All stored actions and outcome checks satisfy the closure rules.</p></div> : aggregate.closure.reasons.length > 0 && <div className="matter-closure-state"><strong>Before this issue can close</strong><ul>{aggregate.closure.reasons.map((reason) => <li key={reason}>{reason}</li>)}</ul></div>}

    {!active && transitionOperation?.can_act && allowedStatusTargets.length > 0 && <button className="secondary-button matter-status-button" type="button" onClick={beginStatus}>{aggregate.closure.ready && allowedStatusTargets.includes("CLOSED") ? "Close issue" : "Change issue status"}</button>}
    {!transitionOperation?.can_act && transitionOperation?.reason && <p className="matter-operation-reason">{transitionOperation.reason}</p>}

    {active && <form className="matter-operation-form" onSubmit={submit}>
      {(active.kind === "define" || active.kind === "supersede") && <>
        <label className="wide"><span>Expected outcome</span><textarea rows={3} value={expectedOutcome} onChange={(event) => setExpectedOutcome(event.target.value)} required/></label>
        <label><span>Linked action</span><select value={actionID} onChange={(event) => setActionID(event.target.value)}><option value="">Issue-level outcome</option>{aggregate.actions.map((action) => <option key={action.id} value={action.id}>{action.title}</option>)}</select></label>
        <label className="wide"><span>Scope covered</span><textarea rows={2} value={scopeCovered} onChange={(event) => setScopeCovered(event.target.value)} required placeholder="Business process, population, service or locations this check covers"/></label>
        <label className="wide"><span>How the outcome will be measured</span><textarea rows={2} value={measurementMethod} onChange={(event) => setMeasurementMethod(event.target.value)} required placeholder="Report, sample or manual review used to measure this result"/></label>
        <label className="wide"><span>Current baseline</span><textarea rows={2} value={currentBaseline} onChange={(event) => setCurrentBaseline(event.target.value)} required placeholder="Current measured state before the work is completed"/></label>
        <label className="wide"><span>Success threshold</span><textarea rows={2} value={successThreshold} onChange={(event) => setSuccessThreshold(event.target.value)} required placeholder="The measurable condition that confirms the outcome"/></label>
        <label><span>Registered measurement source (optional)</span><select value={measurementSourceID} onChange={(event) => setMeasurementSourceID(event.target.value)} disabled={sourcesState !== "live"}><option value="">Manual review / no registered source</option>{activeSources.map((source) => <option key={source.id} value={source.id}>{source.name}</option>)}</select>{sourcesState === "loading" && <small>Loading registered evidence sources…</small>}{sourcesState === "unavailable" && <small>Registered evidence sources could not be loaded. You can still save the manual measurement method above.</small>}</label>
        <label><span>Observation period (days)</span><input type="number" min="0" max="365" step="any" value={observationDays} onChange={(event) => setObservationDays(event.target.value)} required/></label>
        <label><span>Independent reviewer</span><select value={reviewerCandidateID} onChange={(event) => setReviewerCandidateID(event.target.value)} required><option value="">Select the person responsible for the outcome result</option>{(activeDefinitionOperation?.candidates ?? []).map((candidate) => <option key={candidate.id} value={candidate.id}>{candidate.display_name}{candidate.role ? ` · ${candidate.role}` : ""}</option>)}</select>{!(activeDefinitionOperation?.candidates?.length) && <small>No current reviewer candidate is available. Ask a GRC administrator to restore the reviewer route.</small>}</label>
        <label className="wide"><span>If the outcome is not achieved</span><select value={failureResponse} onChange={(event) => setFailureResponse(event.target.value)} required><option value="">Select the required handling</option><option value="REOPEN">Reopen this issue for corrective work</option><option value="CREATE_MATTER">Create a follow-up issue</option><option value="ESCALATE">Escalate to the current escalation owner</option><option value="BLOCK_CLOSE">Keep this issue open</option></select></label>
        {active.kind === "supersede" && <label className="wide"><span>Reason for replacing this outcome check</span><textarea rows={2} value={rationale} onChange={(event) => setRationale(event.target.value)} required/></label>}
      </>}
      {active.kind === "retire" && <>
        <p className="matter-form-context wide">Ending: {activeContract?.expected_outcome}</p>
        <p className="wide">This stops new results for this outcome check. Its contract and result history remain on the issue record.</p>
        <label className="wide"><span>Reason for ending this outcome check</span><textarea rows={2} value={rationale} onChange={(event) => setRationale(event.target.value)} required/></label>
      </>}
      {active.kind === "result" && <>
        <p className="matter-form-context wide">Checking: {activeContract?.expected_outcome}</p>
        <label><span>Check result</span><select value={result} onChange={(event) => setResult(event.target.value as typeof result)}><option value="PASS">Outcome achieved</option><option value="FAIL">Outcome not achieved</option><option value="INCONCLUSIVE">More evidence needed</option></select></label>
        <label className="wide"><span>Observations</span><textarea rows={3} value={observations} onChange={(event) => setObservations(event.target.value)} required/></label>
        <label className="wide"><span>Evidence references (optional)</span><textarea rows={2} value={evidenceReferences} onChange={(event) => setEvidenceReferences(event.target.value)} placeholder="One artifact or source reference per line"/></label>
        <label className="wide"><span>Result rationale</span><textarea rows={2} value={rationale} onChange={(event) => setRationale(event.target.value)} required/></label>
      </>}
      {active.kind === "status" && <>
        {allowedStatusTargets.length > 1 && <label><span>Next issue status</span><select value={target} onChange={(event) => setTarget(event.target.value)}>{allowedStatusTargets.map((value) => <option key={value} value={value}>{statusLabel(value)}</option>)}</select></label>}
        <label className="wide"><span>Reason for status change</span><textarea rows={2} value={rationale} onChange={(event) => setRationale(event.target.value)} required/></label>
      </>}
      {error && <div className="matter-form-error wide" role="alert"><span>{error}</span>{conflict && <button className="secondary-button" type="button" onClick={onReload}>Reload current issue</button>}</div>}
      <div className="matter-form-actions wide"><button className="primary-button" type="submit" disabled={saving || (active.kind === "status" && !target) || ((active.kind === "define" || active.kind === "supersede") && !reviewerCandidateID) || ((active.kind === "supersede" || active.kind === "retire") && !rationale.trim())}>{saving ? "Saving…" : active.kind === "define" ? "Save outcome check" : active.kind === "supersede" ? "Replace outcome check" : active.kind === "retire" ? "End outcome check" : active.kind === "result" ? "Record outcome result" : "Confirm issue status"}</button><button className="text-button" type="button" onClick={() => setActive(null)}>Cancel</button></div>
    </form>}
    {notice && <p className="inline-success" role="status">{notice}</p>}
  </article>;
}
