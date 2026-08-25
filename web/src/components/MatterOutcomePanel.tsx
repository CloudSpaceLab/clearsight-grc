import { useState } from "react";
import type { FormEvent } from "react";
import { apiErrorKind } from "../http";
import { defineMatterOutcomeCheck } from "../matterOperationsApi";
import type { MatterOperation } from "../matterOperationsApi";
import { recordVerificationResult, transitionMatter } from "../continuityCommands";
import type { MatterAggregate, VerificationContract } from "../types";

type Props = {
  aggregate: MatterAggregate;
  operations: MatterOperation[];
  onUpdated: (value: MatterAggregate) => void | Promise<void>;
  onReload: () => void;
};

type Active = { kind: "define" } | { kind: "result"; contractID: string } | { kind: "status" } | null;

function operationFor(operations: MatterOperation[], command: string, subresourceID?: string) {
  return operations.find((operation) => operation.command === command && (subresourceID === undefined || operation.subresource_id === subresourceID));
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

export function MatterOutcomePanel({ aggregate, operations, onUpdated, onReload }: Props) {
  const defineOperation = operationFor(operations, "matter.outcome.define");
  const transitionOperation = operationFor(operations, "matter.transition");
  const [active, setActive] = useState<Active>(null);
  const [expectedOutcome, setExpectedOutcome] = useState("");
  const [actionID, setActionID] = useState("");
  const [observationMinutes, setObservationMinutes] = useState("0");
  const [failureResponse, setFailureResponse] = useState("");
  const [result, setResult] = useState<"PASS" | "FAIL" | "INCONCLUSIVE">("PASS");
  const [observations, setObservations] = useState("");
  const [evidenceReferences, setEvidenceReferences] = useState("");
  const [rationale, setRationale] = useState("");
  const [target, setTarget] = useState("");
  const [saving, setSaving] = useState(false);
  const [notice, setNotice] = useState("");
  const [error, setError] = useState("");
  const [conflict, setConflict] = useState(false);

  function beginDefine() {
    setExpectedOutcome("");
    setActionID(aggregate.actions[0]?.id ?? "");
    setObservationMinutes("0");
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
          observationPeriodMinutes: Number(observationMinutes),
          failureResponse: failureResponse.trim(),
        });
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
      setNotice(completedKind === "define" ? "Outcome check defined." : completedKind === "result" ? "Outcome result recorded." : `Issue status changed to ${statusLabel(target)}.`);
    } catch (cause) {
      handleError(cause);
    } finally {
      setSaving(false);
    }
  }

  const activeContract = active?.kind === "result" ? aggregate.verification_contracts.find((contract) => contract.id === active.contractID) : undefined;
  const allowedStatusTargets = (transitionOperation?.allowed_targets ?? []).filter((value) => value !== "CLOSED" || aggregate.closure.ready);

  return <article className="matter-record-panel matter-outcome-panel" id="matter-operation-matter.outcome.define">
    <div className="matter-record-section-heading">
      <div><span className="eyebrow">Outcome checks</span><h2>Independent results</h2></div>
      {defineOperation?.can_act && !active && <button className="secondary-button" type="button" onClick={beginDefine}>Define outcome check</button>}
    </div>
    {aggregate.verification_contracts.length ? <div className="matter-outcome-list">{aggregate.verification_contracts.map((contract) => {
      const recorded = aggregate.verification_results.filter((item) => item.contract_id === contract.id).at(-1);
      const recordOperation = operationFor(operations, "matter.outcome.record", contract.id);
      const linkedAction = aggregate.actions.find((action) => action.id === contract.action_id);
      return <section className="matter-outcome-card" key={contract.id} aria-labelledby={`matter-outcome-${contract.id}`}>
        <div className="matter-action-heading"><div><h3 id={`matter-outcome-${contract.id}`}>{contract.expected_outcome}</h3>{linkedAction && <p>Checks the result of: {linkedAction.title}</p>}</div><span>{resultLabel(recorded?.result)}</span></div>
        <dl className="matter-outcome-meta">
          <div><dt>Reviewer</dt><dd>{recordOperation?.assigned_to?.display_name ?? defineOperation?.assigned_to?.display_name ?? contract.authority_principal_id ?? "Reviewer not resolved"}</dd></div>
          <div><dt>Observation period</dt><dd>{contract.observation_period_minutes} minutes</dd></div>
        </dl>
        {recorded?.rationale && <p className="matter-outcome-rationale"><strong>Recorded basis:</strong> {recorded.rationale}</p>}
        {contract.failure_response && <p className="matter-outcome-rationale"><strong>If not achieved:</strong> {failureLabel(contract.failure_response)}</p>}
        {!active && recordOperation?.can_act && <button className="secondary-button" type="button" aria-label={`Record result for ${contract.expected_outcome}`} onClick={() => beginResult(contract)}>Record outcome result</button>}
        {!recordOperation?.can_act && recordOperation?.reason && <p className="matter-operation-reason">{recordOperation.reason}</p>}
      </section>;
    })}</div> : <div className="matter-record-attention"><strong>No outcome check has been defined</strong><p>Define the result that must be independently confirmed before this issue can close.</p></div>}

    {aggregate.closure.ready ? <div className="matter-closure-state ready"><strong>Ready to close</strong><p>All stored actions and outcome checks satisfy the closure rules.</p></div> : aggregate.closure.reasons.length > 0 && <div className="matter-closure-state"><strong>Before this issue can close</strong><ul>{aggregate.closure.reasons.map((reason) => <li key={reason}>{reason}</li>)}</ul></div>}

    {!active && transitionOperation?.can_act && allowedStatusTargets.length > 0 && <button className="secondary-button matter-status-button" type="button" onClick={beginStatus}>{aggregate.closure.ready && allowedStatusTargets.includes("CLOSED") ? "Close issue" : "Change issue status"}</button>}
    {!transitionOperation?.can_act && transitionOperation?.reason && <p className="matter-operation-reason">{transitionOperation.reason}</p>}

    {active && <form className="matter-operation-form" onSubmit={submit}>
      {active.kind === "define" && <>
        <label className="wide"><span>Expected outcome</span><textarea rows={3} value={expectedOutcome} onChange={(event) => setExpectedOutcome(event.target.value)} required/></label>
        <label><span>Linked action</span><select value={actionID} onChange={(event) => setActionID(event.target.value)}><option value="">Issue-level outcome</option>{aggregate.actions.map((action) => <option key={action.id} value={action.id}>{action.title}</option>)}</select></label>
        <label><span>Observation period (minutes)</span><input type="number" min="0" value={observationMinutes} onChange={(event) => setObservationMinutes(event.target.value)} required/></label>
        <label className="wide"><span>If the outcome is not achieved</span><select value={failureResponse} onChange={(event) => setFailureResponse(event.target.value)} required><option value="">Select the required handling</option><option value="REOPEN">Reopen this issue for corrective work</option><option value="CREATE_MATTER">Create a follow-up issue</option><option value="ESCALATE">Escalate to the current escalation owner</option><option value="BLOCK_CLOSE">Keep this issue open</option></select></label>
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
      <div className="matter-form-actions wide"><button className="primary-button" type="submit" disabled={saving || (active.kind === "status" && !target)}>{saving ? "Saving…" : active.kind === "define" ? "Save outcome check" : active.kind === "result" ? "Record outcome result" : "Confirm issue status"}</button><button className="text-button" type="button" onClick={() => setActive(null)}>Cancel</button></div>
    </form>}
    {notice && <p className="inline-success" role="status">{notice}</p>}
  </article>;
}
