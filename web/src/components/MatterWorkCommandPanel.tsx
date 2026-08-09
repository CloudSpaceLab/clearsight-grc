import { useMemo, useState } from "react";
import type { FormEvent } from "react";
import { apiErrorKind } from "../http";
import { recordMatterDecision, recordVerificationResult, transitionResponsePackage } from "../continuityCommands";
import type { MatterAggregate, WorkflowTask } from "../types";

type Props = {
  aggregate: MatterAggregate;
  task: WorkflowTask;
  onUpdated: (value: MatterAggregate) => void;
  onCompleted: (taskID: string) => void;
};

type SubmitState = "idle" | "saving" | "saved" | "error";

function humanize(value: string) {
  return value.replaceAll("_", " ").toLowerCase().replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}

function messageFor(error: unknown) {
  const kind = apiErrorKind(error);
  if (kind === "forbidden" || kind === "unauthorized") return "Your current authority no longer permits this action. The work item has not been changed.";
  if (kind === "conflict") return "This record changed since it was loaded. Reload the issue before acting on it.";
  if (kind === "not_found") return "This work item is no longer available.";
  return error instanceof Error && error.message ? error.message : "The governed action could not be recorded.";
}

function allowedTargets(task: WorkflowTask) {
  return (task.context?.allowed_targets ?? "").split(",").map((value) => value.trim()).filter(Boolean);
}

export function MatterWorkCommandPanel({ aggregate, task, onUpdated, onCompleted }: Props) {
  const command = task.context?.command_name ?? "";
  const targets = useMemo(() => allowedTargets(task), [task]);
  const [target, setTarget] = useState(task.context?.target_status || targets[0] || "");
  const [rationale, setRationale] = useState("");
  const [selectedOption, setSelectedOption] = useState("");
  const [result, setResult] = useState<"PASS" | "FAIL" | "INCONCLUSIVE">("PASS");
  const [state, setState] = useState<SubmitState>("idle");
  const [error, setError] = useState("");

  const subresourceID = task.context?.subresource_id ?? task.context?.verification_contract_id ?? "";
  const currentDecision = aggregate.decisions.find((decision) => decision.id === subresourceID);
  const currentResponse = aggregate.response_packages.find((response) => response.id === subresourceID);
  const currentContract = aggregate.verification_contracts.find((contract) => contract.id === subresourceID);

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (state === "saving" || state === "saved") return;
    setState("saving");
    setError("");
    try {
      let updated: MatterAggregate;
      if (command === "matter.response.transition" && currentResponse && target) {
        updated = await transitionResponsePackage(aggregate.matter.id, currentResponse.id, aggregate.matter.version, target, rationale.trim());
      } else if (command === "matter.outcome.record" && currentContract) {
        updated = await recordVerificationResult(aggregate.matter.id, aggregate.matter.version, {
          contractID: currentContract.id,
          result,
          rationale: rationale.trim(),
        });
      } else if (command === "matter.decision.record" && currentDecision && target) {
        updated = await recordMatterDecision(aggregate.matter.id, aggregate.matter.version, {
          type: currentDecision.type,
          status: target,
          selectedOption: selectedOption.trim() || currentDecision.selected_option,
          rationale: rationale.trim(),
        });
      } else {
        throw new Error("This work packet does not map to an executable command.");
      }
      onUpdated(updated);
      onCompleted(task.id);
      setState("saved");
    } catch (value) {
      setError(messageFor(value));
      setState("error");
    }
  }

  const supported = (command === "matter.response.transition" && Boolean(currentResponse))
    || (command === "matter.outcome.record" && Boolean(currentContract))
    || (command === "matter.decision.record" && Boolean(currentDecision));

  if (!supported) return <section className="handoff-summary"><h3>Current governed work</h3><strong>{task.title}</strong><p>The current work packet cannot be executed from this screen. Its canonical record remains unchanged.</p></section>;

  return <section className="handoff-summary" aria-labelledby={`work-command-${task.id}`}>
    <span className="eyebrow">Your governed action</span>
    <h3 id={`work-command-${task.id}`}>{task.context?.primary_action || task.title}</h3>
    <p>{task.context?.why_now || "This action is assigned to you by the current workflow and authority policy."}</p>
    <form className="governed-command-form" onSubmit={submit}>
      {(command === "matter.response.transition" || command === "matter.decision.record") && <label><span>Outcome</span><select value={target} onChange={(event) => setTarget(event.target.value)} required>{targets.map((value) => <option key={value} value={value}>{humanize(value)}</option>)}</select></label>}
      {command === "matter.decision.record" && ["APPROVED", "CONDITIONALLY_APPROVED", "REJECTED"].includes(target) && <label><span>Selected option</span><input value={selectedOption} onChange={(event) => setSelectedOption(event.target.value)} placeholder={currentDecision?.selected_option || "Decision option"} required={!currentDecision?.selected_option}/></label>}
      {command === "matter.outcome.record" && <label><span>Outcome check</span><select value={result} onChange={(event) => setResult(event.target.value as typeof result)}><option value="PASS">Outcome confirmed</option><option value="FAIL">Outcome not achieved</option><option value="INCONCLUSIVE">More evidence needed</option></select></label>}
      <label><span>Rationale</span><textarea value={rationale} onChange={(event) => setRationale(event.target.value)} placeholder="Record the concise basis for this action" required rows={3}/></label>
      {error && <p className="inline-error" role="alert">{error}</p>}
      {state === "saved" ? <div className="inline-notice" role="status">Recorded. The issue detail now reflects the authoritative server result; routed work will converge on the next projection cycle.</div> : <button className="primary-button" type="submit" disabled={state === "saving" || !target && command !== "matter.outcome.record"}>{state === "saving" ? "Recording…" : task.context?.primary_action || "Record action"}</button>}
    </form>
  </section>;
}
