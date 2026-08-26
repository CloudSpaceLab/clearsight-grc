import { useState } from "react";
import type { FormEvent } from "react";
import { apiErrorKind } from "../http";
import { assignMatterAction, updateMatterAction } from "../matterOperationsApi";
import type { MatterOperation } from "../matterOperationsApi";
import { addMatterAction, transitionMatterAction } from "../continuityCommands";
import type { MatterAction, MatterAggregate, RecordResponsibleParty } from "../types";
import { selectedDateEndOfLocalDay, storedDeadlineLocalDate } from "../dueDate";

type Props = {
  aggregate: MatterAggregate;
  operations: MatterOperation[];
  responsibleParties?: RecordResponsibleParty[];
  onUpdated: (value: MatterAggregate) => void | Promise<void>;
  onReload: () => void;
};

type Active = { kind: "add" } | { kind: "edit" | "assign" | "status"; actionID: string } | null;

function operationFor(operations: MatterOperation[], command: string, actionID?: string) {
  return operations.find((operation) => operation.command === command && (actionID === undefined || operation.subresource_id === actionID));
}

function formatDate(value?: string) {
  if (!value || !Number.isFinite(Date.parse(value))) return "No action deadline";
  return `Action due ${new Intl.DateTimeFormat("en-GB", { day: "numeric", month: "short", year: "numeric" }).format(new Date(value))}`;
}

function statusLabel(value: string) {
  switch (value) {
    case "PLANNED": return "Not started";
    case "IN_PROGRESS": return "In progress";
    case "BLOCKED": return "Blocked";
    case "IMPLEMENTED": return "Work completed; outcome not confirmed";
    case "CANCELLED": return "Cancelled";
    default: return value.replaceAll("_", " ").toLowerCase().replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
  }
}

export function MatterActionsPanel({ aggregate, operations, responsibleParties = [], onUpdated, onReload }: Props) {
  const addOperation = operationFor(operations, "matter.action.add");
  const [active, setActive] = useState<Active>(null);
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [owner, setOwner] = useState("");
  const [date, setDate] = useState("");
  const [target, setTarget] = useState("");
  const [rationale, setRationale] = useState("");
  const [saving, setSaving] = useState(false);
  const [notice, setNotice] = useState("");
  const [error, setError] = useState("");
  const [conflict, setConflict] = useState(false);

  function actionFor(actionID: string) { return aggregate.actions.find((action) => action.id === actionID); }

  function startAdd() {
    setActive({ kind: "add" }); setTitle(""); setDescription(""); setDate(""); setRationale(""); setError(""); setNotice("");
    setOwner(addOperation?.candidates?.[0]?.id ?? aggregate.matter.owner_principal_id ?? "");
  }

  function startAction(kind: "edit" | "assign" | "status", action: MatterAction) {
    setActive({ kind, actionID: action.id }); setTitle(action.title); setDescription(action.description); setDate(storedDeadlineLocalDate(action.due_at)); setRationale(""); setError(""); setNotice("");
    const operation = operationFor(operations, kind === "assign" ? "matter.action.assign" : kind === "status" ? "matter.action.transition" : "matter.action.update", action.id);
    setOwner(operation?.candidates?.[0]?.id ?? action.owner_principal_id ?? "");
    setTarget(operation?.allowed_targets?.[0] ?? "");
  }

  function handleError(cause: unknown) {
    const isConflict = apiErrorKind(cause) === "conflict";
    setConflict(isConflict);
    setError(isConflict ? "This issue changed since you opened it. Your entries have been kept." : cause instanceof Error && cause.message ? cause.message : "The action could not be changed.");
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!active) return;
    setSaving(true); setError(""); setConflict(false); setNotice("");
    try {
      let updated: MatterAggregate;
      if (active.kind === "add") {
        updated = await addMatterAction(aggregate.matter.id, aggregate.matter.version, { title: title.trim(), description: description.trim(), ownerPrincipalID: owner || undefined, dueAt: selectedDateEndOfLocalDay(date) });
      } else if (active.kind === "edit") {
        updated = await updateMatterAction(aggregate.matter.id, active.actionID, aggregate.matter.version, { title: title.trim(), description: description.trim(), dueAt: selectedDateEndOfLocalDay(date), rationale: rationale.trim() });
      } else if (active.kind === "assign") {
        updated = await assignMatterAction(aggregate.matter.id, active.actionID, aggregate.matter.version, owner, rationale.trim());
      } else {
        updated = await transitionMatterAction(aggregate.matter.id, active.actionID, aggregate.matter.version, target, rationale.trim());
      }
      const completedKind = active.kind;
      await onUpdated(updated);
      setActive(null);
      setNotice(completedKind === "add" ? "Action added." : completedKind === "assign" ? "Action owner updated." : completedKind === "status" ? "Action status updated." : "Action updated.");
    } catch (cause) { handleError(cause); } finally { setSaving(false); }
  }

  const activeAction = active && active.kind !== "add" ? actionFor(active.actionID) : undefined;
  const activeOperation = active && active.kind !== "add" ? operationFor(operations, active.kind === "edit" ? "matter.action.update" : active.kind === "assign" ? "matter.action.assign" : "matter.action.transition", active.actionID) : addOperation;
  const candidates = activeOperation?.candidates ?? addOperation?.candidates ?? [];
  const rationaleRequired = active?.kind === "edit" || active?.kind === "assign" || (active?.kind === "status" && ["BLOCKED", "CANCELLED"].includes(target));

  return <article className="matter-record-panel matter-actions-panel" id="matter-operation-matter.action.add">
    <div className="matter-record-section-heading"><div><span className="eyebrow">Assigned work</span><h2>Actions</h2></div>{addOperation?.can_act && !active && <button className="secondary-button" type="button" onClick={startAdd}>Add action</button>}</div>
    {aggregate.actions.length ? <div className="matter-action-list">{aggregate.actions.map((action) => {
      const editOperation = operationFor(operations, "matter.action.update", action.id);
      const assignOperation = operationFor(operations, "matter.action.assign", action.id);
      const statusOperation = operationFor(operations, "matter.action.transition", action.id);
      const terminal = ["IMPLEMENTED", "CANCELLED"].includes(action.status);
      const actionResponsibility = action.required_responsibility || "PERFORMER";
      const storedOwner = responsibleParties.find((party) => party.scope === "ACTION" && party.subresource_id === action.id && party.responsibility === actionResponsibility)?.display_name;
      const ownerName = statusOperation?.assigned_to?.display_name ?? storedOwner ?? (action.owner_principal_id ? "Recorded action owner unavailable" : "Action owner not assigned");
      return <section className="matter-action-card" id={`matter-operation-matter.action.transition-${action.id}`} key={action.id} aria-labelledby={`matter-action-${action.id}`}>
        <div className="matter-action-heading"><div><h3 id={`matter-action-${action.id}`}>{action.title}</h3><p>{action.description}</p></div><span>{statusLabel(action.status)}</span></div>
        <div className="matter-action-meta"><span>{actionResponsibility === "ESCALATION_OWNER" ? "Escalation owner" : "Action owner"}: <strong>{ownerName}</strong></span><span>{formatDate(action.due_at)}</span></div>
        {!active && <div className="matter-action-controls">
          {editOperation?.can_act && !terminal && <button className="secondary-button" type="button" aria-label={`Edit ${action.title}`} onClick={() => startAction("edit", action)}>Edit action</button>}
          {assignOperation?.can_act && !terminal && <button className="secondary-button" type="button" aria-label={`Change owner for ${action.title}`} onClick={() => startAction("assign", action)}>Change owner</button>}
          {statusOperation?.can_act && !terminal && <button className="secondary-button" type="button" aria-label={`Update status for ${action.title}`} onClick={() => startAction("status", action)}>Update status</button>}
        </div>}
        {!statusOperation?.can_act && statusOperation?.reason && <p className="matter-operation-reason">{statusOperation.reason}</p>}
      </section>;
    })}</div> : <p>No actions have been recorded for this issue.</p>}
    {active && <form className="matter-operation-form" onSubmit={submit}>
      {(active.kind === "add" || active.kind === "edit") && <><label><span>Action title</span><input value={title} onChange={(event) => setTitle(event.target.value)} required/></label><label className="wide"><span>Action description</span><textarea value={description} onChange={(event) => setDescription(event.target.value)} rows={3} required/></label><label><span>Action due date</span><input type="date" value={date} onChange={(event) => setDate(event.target.value)}/></label></>}
      {active.kind === "add" && <label><span>Action owner</span><select value={owner} onChange={(event) => setOwner(event.target.value)} required><option value="">Select an eligible performer</option>{candidates.map((candidate) => <option key={candidate.id} value={candidate.id}>{candidate.display_name} · {candidate.role}</option>)}</select></label>}
      {active.kind === "assign" && <label><span>New action owner</span><select value={owner} onChange={(event) => setOwner(event.target.value)} required><option value="">Select an eligible performer</option>{candidates.map((candidate) => <option key={candidate.id} value={candidate.id}>{candidate.display_name} · {candidate.role}</option>)}</select></label>}
      {active.kind === "status" && <label><span>Next action status</span><select value={target} onChange={(event) => setTarget(event.target.value)} required>{activeOperation?.allowed_targets?.map((value) => <option key={value} value={value}>{statusLabel(value)}</option>)}</select></label>}
      {active.kind !== "add" && <label className="wide"><span>{active.kind === "edit" ? "Reason for changing this action" : active.kind === "assign" ? "Reason for action reassignment" : `Status rationale${rationaleRequired ? "" : " (optional)"}`}</span><textarea value={rationale} onChange={(event) => setRationale(event.target.value)} rows={2} required={rationaleRequired}/></label>}
      {error && <div className="matter-form-error wide" role="alert"><span>{error}</span>{conflict && <button className="secondary-button" type="button" onClick={onReload}>Reload current issue</button>}</div>}
      <div className="matter-form-actions wide"><button className="primary-button" type="submit" disabled={saving || (active.kind === "status" && !target)}>{saving ? "Saving…" : active.kind === "add" ? "Create assigned action" : active.kind === "edit" ? "Save action" : active.kind === "assign" ? "Assign action owner" : "Update action status"}</button><button className="text-button" type="button" onClick={() => setActive(null)}>Cancel</button></div>
      {activeAction && <p className="matter-form-context wide">Changing: {activeAction.title}</p>}
    </form>}
    {notice && <p className="inline-success" role="status">{notice}</p>}
  </article>;
}
