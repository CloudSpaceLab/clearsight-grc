import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import { loadPrograms } from "../api";
import { apiErrorKind } from "../http";
import { addMatterLink, assignMatter, retireMatterLink, updateMatterDetails } from "../matterOperationsApi";
import type { MatterOperation } from "../matterOperationsApi";
import type { MatterAggregate, ProgramAggregate, RecordResponsibleParty } from "../types";
import { selectedDateEndOfLocalDay, storedDeadlineLocalDate } from "../dueDate";
import { Button, FocusedSheet, Notice, SelectField, TextArea } from "./ui";
import { matterOperationControlID } from "./matterHandoff";

type Props = {
  aggregate: MatterAggregate;
  operations: MatterOperation[];
  responsibleParties?: RecordResponsibleParty[];
  assignmentIntent?: number;
  suppressAssignmentAction?: boolean;
  onUpdated: (value: MatterAggregate) => void | Promise<void>;
  onReload: () => void;
};

function operationFor(operations: MatterOperation[], command: string) {
  return operations.find((operation) => operation.command === command);
}

export function MatterDetailsPanel({ aggregate, operations, responsibleParties = [], assignmentIntent = 0, suppressAssignmentAction = false, onUpdated, onReload }: Props) {
  const detailsOperation = operationFor(operations, "matter.details.update");
  const assignmentOperation = operationFor(operations, "matter.assign");
  const linkOperation = operationFor(operations, "matter.link");
  const [editing, setEditing] = useState(false);
  const [assigning, setAssigning] = useState(false);
  const [linking, setLinking] = useState(false);
  const [retiringLinkID, setRetiringLinkID] = useState("");
  const [retirementReason, setRetirementReason] = useState("");
  const [title, setTitle] = useState(aggregate.matter.title);
  const [summary, setSummary] = useState(aggregate.matter.summary);
  const [priority, setPriority] = useState(String(aggregate.matter.priority));
  const [date, setDate] = useState(storedDeadlineLocalDate(aggregate.matter.due_at));
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

  function beginAssignment() {
    setNewOwner(assignmentOperation?.candidates?.find((candidate) => candidate.id !== aggregate.matter.owner_principal_id)?.id ?? assignmentOperation?.candidates?.[0]?.id ?? "");
    setAssignmentReason("");
    setAssigning(true);
    setEditing(false);
    setLinking(false);
    setNotice("");
    setError("");
    setConflict(false);
  }

  useEffect(() => {
    if (assignmentIntent > 0) beginAssignment();
  }, [assignmentIntent]);

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
        title: title.trim(), summary: summary.trim(), priority: Number(priority), dueAt: selectedDateEndOfLocalDay(date),
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

  async function removeLink(event: FormEvent) {
    event.preventDefault();
    setSaving(true); setError(""); setConflict(false); setNotice("");
    try {
      const updated = await retireMatterLink(aggregate.matter.id, retiringLinkID, aggregate.matter.version, retirementReason.trim());
      await onUpdated(updated);
      setRetiringLinkID(""); setRetirementReason(""); setNotice("Program link removed. The former link remains in issue history.");
    } catch (value) { handleError(value); } finally { setSaving(false); }
  }

  const owner = assignmentOperation?.assigned_to ?? detailsOperation?.assigned_to;
  const storedOwner = responsibleParties.find((party) => party.scope === "RECORD" && party.responsibility === "ACCOUNTABLE_OWNER")?.display_name;
  return <article className="matter-record-panel matter-details-panel" id="matter-operation-matter.details.update">
    <div className="matter-record-section-heading">
      <div><span className="eyebrow">Issue details</span><h2>Scope, timing and owner</h2></div>
      <div className="matter-panel-actions">
        {detailsOperation?.can_act && !editing && <button id={matterOperationControlID(detailsOperation)} className="secondary-button" type="button" onClick={() => { setEditing(true); setAssigning(false); setLinking(false); setNotice(""); }}>Edit issue details</button>}
        {assignmentOperation?.can_act && !assigning && !suppressAssignmentAction && <Button id={matterOperationControlID(assignmentOperation)} variant="secondary" onPress={beginAssignment}>Change issue owner</Button>}
        {linkOperation?.can_act && !linking && <button className="secondary-button" type="button" onClick={() => void beginLink()}>Link to Program</button>}
      </div>
    </div>
    <dl className="matter-record-facts">
      <div><dt>Affected area</dt><dd>{affectedArea || "Not recorded"}</dd></div>
      <div><dt>Accountable owner</dt><dd>{owner?.display_name ?? storedOwner ?? (aggregate.matter.owner_principal_id ? "Recorded issue owner unavailable" : "Issue owner not assigned")}</dd></div>
      <div><dt>Due date</dt><dd>{date || "Not recorded"}</dd></div>
    </dl>
    <section className="matter-program-links" aria-labelledby="matter-program-links-title"><strong id="matter-program-links-title">Linked Programs</strong>{aggregate.links.length ? <ul>{aggregate.links.map((link) => { const program = programs.find((value) => value.program.id === link.program_id); const unlink = operations.find((operation) => operation.command === "matter.unlink" && operation.subresource_id === link.id); return <li key={link.id}><span>{program?.program.name ?? "Linked Program name unavailable"}</span><small>{link.relationship.replaceAll("_", " ").toLowerCase()}</small>{unlink?.can_act && <button className="text-button" type="button" onClick={() => { setRetiringLinkID(link.id); setRetirementReason(""); setEditing(false); setAssigning(false); setLinking(false); setNotice(""); setError(""); }}>Remove Program link</button>}</li>; })}</ul> : <p>This issue is not linked to a Program.</p>}</section>
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
    {assigning && <FocusedSheet label="Change issue owner" closeLabel="Close owner reassignment" onClose={() => setAssigning(false)}>
      <div className="cs-sheet-heading"><span className="eyebrow">Accountable ownership</span><h2>Change issue owner</h2><p>Choose a person returned by the current authority route and record why responsibility is changing.</p></div>
      <form className="cs-sheet-form" onSubmit={saveAssignment}>
        <dl className="cs-sheet-facts"><div><dt>Issue</dt><dd>{aggregate.matter.title}</dd></div><div><dt>Current accountable owner</dt><dd>{owner?.display_name ?? storedOwner ?? "Issue owner not assigned"}</dd></div></dl>
        <SelectField label="New issue owner" value={newOwner || undefined} placeholder="Select an eligible owner" allowsEmpty={false} isRequired options={(assignmentOperation?.candidates ?? []).map((candidate) => ({ id: candidate.id, label: candidate.role ? `${candidate.display_name} · ${candidate.role}` : candidate.display_name }))} onChange={(value) => setNewOwner(value ?? "")}/>
        <Notice tone="info">After the assignment is recorded, ClearSight will attempt delivery of an assignment email to the staff mailbox held in the active directory. If no usable mailbox is available, the assignment still takes effect and email delivery is recorded as unavailable.</Notice>
        <TextArea label="Reason for reassignment" value={assignmentReason} onChange={setAssignmentReason} rows={3} isRequired description="This reason remains with the issue ownership history."/>
        {error && <Notice tone="error"><span>{error}</span>{conflict && <Button variant="secondary" onPress={onReload}>Reload current issue</Button>}</Notice>}
        <div className="cs-sheet-actions"><Button type="button" variant="quiet" isDisabled={saving} onPress={() => setAssigning(false)}>Cancel</Button><Button type="submit" variant="primary" isDisabled={!newOwner || !assignmentReason.trim()} isLoading={saving}>Assign issue owner</Button></div>
      </form>
    </FocusedSheet>}
    {linking && <form className="matter-operation-form" onSubmit={saveLink}>
      <label className="wide"><span>Program</span><select value={programID} onChange={(event) => setProgramID(event.target.value)} required disabled={programsLoading}><option value="">{programsLoading ? "Loading visible Programs…" : "Select a visible Program"}</option>{programs.map((program) => <option key={program.program.id} value={program.program.id}>{program.program.name} · {program.program.code}</option>)}</select></label>
      <label><span>Relationship</span><select value={relationship} onChange={(event) => setRelationship(event.target.value)}><option value="AFFECTS">Affects this Program</option><option value="AROSE_FROM">Arose from this Program</option><option value="REMEDIATES">Remediates a gap in this Program</option><option value="EXCEPTION_TO">Records an exception to this Program</option></select></label>
      {!programsLoading && programs.length === 0 && !error && <div className="matter-record-attention wide"><strong>No visible Programs are available</strong><p>Create the Program first or ask its owner to grant access, then retry this link.</p></div>}
      {error && <div className="matter-form-error wide" role="alert"><span>{error}</span>{conflict && <button className="secondary-button" type="button" onClick={onReload}>Reload current issue</button>}</div>}
      <div className="matter-form-actions wide"><button className="primary-button" type="submit" disabled={saving || programsLoading || !programID}>{saving ? "Saving…" : "Save Program link"}</button><button className="text-button" type="button" onClick={() => setLinking(false)}>Cancel</button></div>
    </form>}
    {retiringLinkID && <form className="matter-operation-form" onSubmit={removeLink}>
      <div className="matter-record-attention wide"><strong>Remove this Program link?</strong><p>The issue and Program remain available. The former relationship stays in history for audit and reconstruction.</p></div>
      <label className="wide"><span>Reason for removing this Program link</span><textarea value={retirementReason} onChange={(event) => setRetirementReason(event.target.value)} rows={2} required/></label>
      {error && <div className="matter-form-error wide" role="alert"><span>{error}</span>{conflict && <button className="secondary-button" type="button" onClick={onReload}>Reload current issue</button>}</div>}
      <div className="matter-form-actions wide"><button className="primary-button" type="submit" disabled={saving || !retirementReason.trim()}>{saving ? "Removing…" : "Remove link"}</button><button className="text-button" type="button" onClick={() => { setRetiringLinkID(""); setRetirementReason(""); }}>Cancel</button></div>
    </form>}
    {notice && <p className="inline-success" role="status">{notice}</p>}
    {!detailsOperation?.can_act && detailsOperation?.reason && <p className="matter-operation-reason">{detailsOperation.reason}</p>}
  </article>;
}
