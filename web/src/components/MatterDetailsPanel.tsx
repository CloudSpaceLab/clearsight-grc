import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import { loadPrograms } from "../api";
import { apiErrorKind } from "../http";
import { addMatterLink, assignMatter, updateMatterDetails } from "../matterOperationsApi";
import type { MatterOperation } from "../matterOperationsApi";
import type { MatterAggregate, ProgramAggregate } from "../types";

type Props = {
  aggregate: MatterAggregate;
  operations: MatterOperation[];
  onUpdated: (value: MatterAggregate) => void | Promise<void>;
  onReload: () => void;
};

function dateInput(value?: string) {
  return value && Number.isFinite(Date.parse(value)) ? new Date(value).toISOString().slice(0, 10) : "";
}

function dueAt(value: string) {
  return value ? new Date(`${value}T00:00:00.000Z`).toISOString() : undefined;
}

function operationFor(operations: MatterOperation[], command: string) {
  return operations.find((operation) => operation.command === command);
}

export function MatterDetailsPanel({ aggregate, operations, onUpdated, onReload }: Props) {
  const detailsOperation = operationFor(operations, "matter.details.update");
  const assignmentOperation = operationFor(operations, "matter.assign");
  const linkOperation = operationFor(operations, "matter.link");
  const [editing, setEditing] = useState(false);
  const [assigning, setAssigning] = useState(false);
  const [linking, setLinking] = useState(false);
  const [title, setTitle] = useState(aggregate.matter.title);
  const [summary, setSummary] = useState(aggregate.matter.summary);
  const [priority, setPriority] = useState(String(aggregate.matter.priority));
  const [date, setDate] = useState(dateInput(aggregate.matter.due_at));
  const [affectedArea, setAffectedArea] = useState(String(aggregate.matter.scope.affected_area ?? aggregate.matter.scope.area ?? ""));
  const [rationale, setRationale] = useState("");
  const [newOwner, setNewOwner] = useState(assignmentOperation?.candidates?.[0]?.id ?? "");
  const [assignmentReason, setAssignmentReason] = useState("");
  const [programs, setPrograms] = useState<ProgramAggregate[]>([]);
  const [programID, setProgramID] = useState("");
  const [relationship, setRelationship] = useState("AFFECTS");
  const [programsLoading, setProgramsLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [notice, setNotice] = useState("");
  const [error, setError] = useState("");
  const [conflict, setConflict] = useState(false);

  useEffect(() => {
    if (!aggregate.links.length) return;
    let active = true;
    void loadPrograms().then((values) => { if (active) setPrograms(values); }).catch(() => undefined);
    return () => { active = false; };
  }, [aggregate.links.length]);

  function handleError(value: unknown) {
    const isConflict = apiErrorKind(value) === "conflict";
    setConflict(isConflict);
    setError(isConflict
      ? "This issue changed since you opened it. Your entries have been kept."
      : value instanceof Error && value.message ? value.message : "The issue could not be changed.");
  }

  async function saveDetails(event: FormEvent) {
    event.preventDefault();
    setSaving(true); setError(""); setConflict(false); setNotice("");
    try {
      const updated = await updateMatterDetails(aggregate.matter.id, aggregate.matter.version, {
        title: title.trim(), summary: summary.trim(), priority: Number(priority), dueAt: dueAt(date),
        scope: { ...aggregate.matter.scope, affected_area: affectedArea.trim() }, rationale: rationale.trim(),
      });
      await onUpdated(updated);
      setEditing(false); setRationale(""); setNotice("Issue details updated.");
    } catch (value) { handleError(value); } finally { setSaving(false); }
  }

  async function saveAssignment(event: FormEvent) {
    event.preventDefault();
    setSaving(true); setError(""); setConflict(false); setNotice("");
    try {
      const updated = await assignMatter(aggregate.matter.id, aggregate.matter.version, newOwner, assignmentReason.trim());
      await onUpdated(updated);
      setAssigning(false); setAssignmentReason(""); setNotice("Issue owner updated.");
    } catch (value) { handleError(value); } finally { setSaving(false); }
  }

  async function beginLink() {
    setLinking(true); setEditing(false); setAssigning(false); setNotice(""); setError(""); setProgramsLoading(true);
    try {
      const values = await loadPrograms();
      setPrograms(values);
      setProgramID(values.find((program) => !aggregate.links.some((link) => link.program_id === program.program.id))?.program.id ?? values[0]?.program.id ?? "");
    } catch {
      setError("Programs could not be loaded. Try again before linking this issue.");
    } finally { setProgramsLoading(false); }
  }

  async function saveLink(event: FormEvent) {
    event.preventDefault();
    setSaving(true); setError(""); setConflict(false); setNotice("");
    try {
      const updated = await addMatterLink(aggregate.matter.id, aggregate.matter.version, { programID, relationship });
      await onUpdated(updated);
      setLinking(false); setNotice("Program link saved.");
    } catch (value) { handleError(value); } finally { setSaving(false); }
  }

  const owner = assignmentOperation?.assigned_to ?? detailsOperation?.assigned_to;
  return <article className="matter-record-panel matter-details-panel" id="matter-operation-matter.details.update">
    <div className="matter-record-section-heading">
      <div><span className="eyebrow">Issue details</span><h2>Scope, timing and owner</h2></div>
      <div className="matter-panel-actions">
        {detailsOperation?.can_act && !editing && <button className="secondary-button" type="button" onClick={() => { setEditing(true); setAssigning(false); setLinking(false); setNotice(""); }}>Edit issue details</button>}
        {assignmentOperation?.can_act && !assigning && <button className="secondary-button" type="button" onClick={() => { setAssigning(true); setEditing(false); setLinking(false); setNotice(""); }}>Change issue owner</button>}
        {linkOperation?.can_act && !linking && <button className="secondary-button" type="button" onClick={() => void beginLink()}>Link to Program</button>}
      </div>
    </div>
    <dl className="matter-record-facts">
      <div><dt>Affected area</dt><dd>{affectedArea || "Not recorded"}</dd></div>
      <div><dt>Accountable owner</dt><dd>{owner?.display_name ?? aggregate.matter.owner_principal_id ?? "Owner not resolved"}</dd></div>
      <div><dt>Due date</dt><dd>{date || "Not recorded"}</dd></div>
    </dl>
    <section className="matter-program-links" aria-labelledby="matter-program-links-title"><strong id="matter-program-links-title">Linked Programs</strong>{aggregate.links.length ? <ul>{aggregate.links.map((link) => { const program = programs.find((value) => value.program.id === link.program_id); return <li key={link.id}><span>{program?.program.name ?? "Linked Program name unavailable"}</span><small>{link.relationship.replaceAll("_", " ").toLowerCase()}</small></li>; })}</ul> : <p>This issue is not linked to a Program.</p>}</section>
    {editing && <form className="matter-operation-form" onSubmit={saveDetails}>
      <label><span>Title</span><input value={title} onChange={(event) => setTitle(event.target.value)} required/></label>
      <label className="wide"><span>Summary</span><textarea value={summary} onChange={(event) => setSummary(event.target.value)} rows={3} required/></label>
      <label><span>Priority</span><select value={priority} onChange={(event) => setPriority(event.target.value)}><option value="1">Low</option><option value="2">Normal</option><option value="3">Medium</option><option value="4">High</option><option value="5">Critical</option></select></label>
      <label><span>Due date</span><input type="date" value={date} onChange={(event) => setDate(event.target.value)}/></label>
      <label><span>Affected area</span><input value={affectedArea} onChange={(event) => setAffectedArea(event.target.value)}/></label>
      <label className="wide"><span>Reason for this change</span><textarea value={rationale} onChange={(event) => setRationale(event.target.value)} rows={2} required/></label>
      {error && <div className="matter-form-error wide" role="alert"><span>{error}</span>{conflict && <button className="secondary-button" type="button" onClick={onReload}>Reload current issue</button>}</div>}
      <div className="matter-form-actions wide"><button className="primary-button" type="submit" disabled={saving}>{saving ? "Saving…" : "Save issue details"}</button><button className="text-button" type="button" onClick={() => setEditing(false)}>Cancel</button></div>
    </form>}
    {assigning && <form className="matter-operation-form" onSubmit={saveAssignment}>
      <label><span>New issue owner</span><select value={newOwner} onChange={(event) => setNewOwner(event.target.value)} required><option value="">Select an eligible owner</option>{assignmentOperation?.candidates?.map((candidate) => <option key={candidate.id} value={candidate.id}>{candidate.display_name} · {candidate.role}</option>)}</select></label>
      <label className="wide"><span>Reason for reassignment</span><textarea value={assignmentReason} onChange={(event) => setAssignmentReason(event.target.value)} rows={2} required/></label>
      {error && <div className="matter-form-error wide" role="alert"><span>{error}</span>{conflict && <button className="secondary-button" type="button" onClick={onReload}>Reload current issue</button>}</div>}
      <div className="matter-form-actions wide"><button className="primary-button" type="submit" disabled={saving || !newOwner}>{saving ? "Assigning…" : "Assign issue owner"}</button><button className="text-button" type="button" onClick={() => setAssigning(false)}>Cancel</button></div>
    </form>}
    {linking && <form className="matter-operation-form" onSubmit={saveLink}>
      <label className="wide"><span>Program</span><select value={programID} onChange={(event) => setProgramID(event.target.value)} required disabled={programsLoading}><option value="">{programsLoading ? "Loading visible Programs…" : "Select a visible Program"}</option>{programs.map((program) => <option key={program.program.id} value={program.program.id}>{program.program.name} · {program.program.code}</option>)}</select></label>
      <label><span>Relationship</span><select value={relationship} onChange={(event) => setRelationship(event.target.value)}><option value="AFFECTS">Affects this Program</option><option value="AROSE_FROM">Arose from this Program</option><option value="REMEDIATES">Remediates a gap in this Program</option><option value="EXCEPTION_TO">Records an exception to this Program</option></select></label>
      {!programsLoading && programs.length === 0 && !error && <div className="matter-record-attention wide"><strong>No visible Programs are available</strong><p>Create the Program first or ask its owner to grant access, then retry this link.</p></div>}
      {error && <div className="matter-form-error wide" role="alert"><span>{error}</span>{conflict && <button className="secondary-button" type="button" onClick={onReload}>Reload current issue</button>}</div>}
      <div className="matter-form-actions wide"><button className="primary-button" type="submit" disabled={saving || programsLoading || !programID}>{saving ? "Saving…" : "Save Program link"}</button><button className="text-button" type="button" onClick={() => setLinking(false)}>Cancel</button></div>
    </form>}
    {notice && <p className="inline-success" role="status">{notice}</p>}
    {!detailsOperation?.can_act && detailsOperation?.reason && <p className="matter-operation-reason">{detailsOperation.reason}</p>}
  </article>;
}
