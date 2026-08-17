import { useEffect, useMemo, useState } from "react";
import type { FormEvent } from "react";
import { apiErrorKind } from "../http";
import { loadActorMatterWork, recordMatterDecision, recordVerificationResult, transitionMatterAction, transitionResponsePackage } from "../continuityCommands";
import type { MatterAggregate, WorkflowTask } from "../types";

type Props = {
  aggregate: MatterAggregate;
  onUpdated: (value: MatterAggregate) => void;
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
  return error instanceof Error && error.message ? error.message : "The action could not be recorded.";
}

function allowedTargets(task: WorkflowTask) {
  return (task.context?.allowed_targets ?? "").split(",").map((value) => value.trim()).filter(Boolean);
}

export function MatterWorkCommand({ aggregate, task, onUpdated, onCompleted }: Props & { task: WorkflowTask; onCompleted: (taskID: string) => void }) {
  const command = task.context?.command_name ?? "";
  const targets = useMemo(() => allowedTargets(task), [task]);
  const [target, setTarget] = useState(task.context?.target_status || targets[0] || "");
  const [rationale, setRationale] = useState("");
  const [selectedOption, setSelectedOption] = useState("");
  const [result, setResult] = useState<"PASS" | "FAIL" | "INCONCLUSIVE">("PASS");
  const [state, setState] = useState<SubmitState>("idle");
  const [error, setError] = useState("");

  const subresourceID = task.context?.subresource_id ?? task.context?.verification_contract_id ?? task.context?.action_id ?? "";
  const currentDecision = aggregate.decisions.find((decision) => decision.id === subresourceID);
  const currentResponse = aggregate.response_packages.find((response) => response.id === subresourceID);
  const currentContract = aggregate.verification_contracts.find((contract) => contract.id === subresourceID);
  const currentAction = aggregate.actions.find((action) => action.id === subresourceID);
  const actionCommand = command === "matter.action.transition";
  const rationaleRequired = !actionCommand || target === "BLOCKED" || target === "CANCELLED";

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (state === "saving" || state === "saved") return;
    setState("saving");
    setError("");
    try {
      let updated: MatterAggregate;
      if (actionCommand && currentAction && target) {
        updated = await transitionMatterAction(aggregate.matter.id, currentAction.id, aggregate.matter.version, target, rationale.trim());
      } else if (command === "matter.response.transition" && currentResponse && target) {
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

  const supported = (actionCommand && Boolean(currentAction) && targets.length > 0)
    || (command === "matter.response.transition" && Boolean(currentResponse) && targets.length > 0)
    || (command === "matter.outcome.record" && Boolean(currentContract))
    || (command === "matter.decision.record" && Boolean(currentDecision) && targets.length > 0);

  if (!supported) return null;

  return <section className="handoff-summary" aria-labelledby={`work-command-${task.id}`}>
    <span className="eyebrow">Assigned action</span>
    <h3 id={`work-command-${task.id}`}>{task.context?.primary_action || task.title}</h3>
    <p>{task.context?.why_now || "This action is assigned to you by the current workflow and authority policy."}</p>
    <form className="governed-command-form" onSubmit={submit}>
      {(actionCommand || command === "matter.response.transition" || command === "matter.decision.record") && <label><span>{actionCommand ? "Next state" : "Outcome"}</span><select value={target} onChange={(event) => setTarget(event.target.value)} required>{targets.map((value) => <option key={value} value={value}>{humanize(value)}</option>)}</select></label>}
      {command === "matter.decision.record" && ["APPROVED", "CONDITIONALLY_APPROVED", "REJECTED"].includes(target) && <label><span>Selected option</span><input value={selectedOption} onChange={(event) => setSelectedOption(event.target.value)} placeholder={currentDecision?.selected_option || "Decision option"} required={!currentDecision?.selected_option}/></label>}
      {command === "matter.outcome.record" && <label><span>Outcome check</span><select value={result} onChange={(event) => setResult(event.target.value as typeof result)}><option value="PASS">Outcome confirmed</option><option value="FAIL">Outcome not achieved</option><option value="INCONCLUSIVE">More evidence needed</option></select></label>}
      <label><span>Rationale{rationaleRequired ? "" : " (optional)"}</span><textarea value={rationale} onChange={(event) => setRationale(event.target.value)} placeholder={rationaleRequired ? "Record the concise basis for this action" : "Add context only if it helps the next reviewer"} required={rationaleRequired} rows={3}/></label>
      {error && <p className="inline-error" role="alert">{error}</p>}
      {state === "saved" ? <div className="inline-notice" role="status">Action recorded. The issue and assigned work have been updated.</div> : <button className="primary-button" type="submit" disabled={state === "saving" || (!target && command !== "matter.outcome.record")}>{state === "saving" ? "Recording…" : task.context?.primary_action || "Record action"}</button>}
    </form>
  </section>;
}

export function MatterWorkCommandPanel({ aggregate, onUpdated }: Props) {
  const [tasks, setTasks] = useState<WorkflowTask[]>([]);
  const [unavailable, setUnavailable] = useState(false);
  const [completionNotice, setCompletionNotice] = useState("");

  useEffect(() => {
    let active = true;
    void loadActorMatterWork().then((values) => {
      if (!active) return;
      setUnavailable(false);
      setTasks(values.filter((task) => task.context?.matter_id === aggregate.matter.id && Boolean(task.context?.command_name) && ["READY", "IN_PROGRESS", "BLOCKED"].includes(task.status)));
    }).catch(() => {
      if (active) setUnavailable(true);
    });
    return () => { active = false; };
  }, [aggregate.matter.id]);

  if (unavailable) return <div className="inline-notice">Assigned actions could not be loaded. The issue remains read-only until current workflow routing is available.</div>;
  if (completionNotice) return <div className="inline-notice" role="status">{completionNotice}</div>;
  if (!tasks.length) return null;

  return <div className="governed-command-stack" aria-label="Assigned actions">
    {tasks.map((task) => <MatterWorkCommand key={task.id} aggregate={aggregate} task={task} onUpdated={onUpdated} onCompleted={(taskID) => { setTasks((current) => current.filter((item) => item.id !== taskID)); setCompletionNotice("Action recorded. The issue and assigned work have been updated."); }}/>) }
  </div>;
}
