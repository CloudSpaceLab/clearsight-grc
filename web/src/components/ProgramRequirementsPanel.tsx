import { useState } from "react";
import type { FormEvent } from "react";
import { addProgramRequirement, determineProgramApplicability, supersedeProgramRequirement } from "../programOperationsApi";
import type { ProgramOperation } from "../programOperationsApi";
import type { ProgramAggregate, Requirement } from "../types";

type Props = { aggregate: ProgramAggregate; operations: ProgramOperation[]; onUpdated: (value: ProgramAggregate) => void; onReload: () => void };
type Mode = "add" | "replace" | "applicability" | null;

function today() { return new Date().toISOString().slice(0, 10); }
function isoDate(value: string) { return new Date(`${value}T00:00:00Z`).toISOString(); }
function label(value?: string) { return (value || "Not recorded").replaceAll("_", " ").toLowerCase().replace(/(^|\s)\S/g, (letter) => letter.toUpperCase()); }

export function ProgramRequirementsPanel({ aggregate, operations, onUpdated, onReload }: Props) {
  const program = aggregate.program;
  const addOperation = operations.find((operation) => operation.command === "program.requirement.add");
  const applicabilityOperation = operations.find((operation) => operation.command === "program.applicability.decide");
  const [mode, setMode] = useState<Mode>(null);
  const [selected, setSelected] = useState<Requirement | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [code, setCode] = useState(""); const [title, setTitle] = useState(""); const [statement, setStatement] = useState("");
  const [sourceAnchor, setSourceAnchor] = useState(""); const [modality, setModality] = useState("MUST");
  const [actor, setActor] = useState("The bank"); const [action, setAction] = useState(""); const [object, setObject] = useState("");
  const [effectiveFrom, setEffectiveFrom] = useState(today()); const [rationale, setRationale] = useState("");
  const [requirementID, setRequirementID] = useState(aggregate.requirements.find((value) => value.status === "APPROVED")?.id ?? "");
  const [applicability, setApplicability] = useState("APPLICABLE"); const [scopeDescription, setScopeDescription] = useState("");

  function beginAdd() { setMode("add"); setSelected(null); setCode(""); setTitle(""); setStatement(""); setSourceAnchor(""); setModality("MUST"); setActor("The bank"); setAction(""); setObject(""); setEffectiveFrom(today()); setRationale(""); setError(""); }
  function beginReplace(requirement: Requirement) { setMode("replace"); setSelected(requirement); setCode(requirement.code); setTitle(requirement.title); setStatement(requirement.statement); setSourceAnchor(requirement.source_anchor ?? ""); setModality(requirement.modality ?? "MUST"); setActor(requirement.actor ?? "The bank"); setAction(requirement.action ?? ""); setObject(requirement.object ?? ""); setEffectiveFrom(today()); setRationale(""); setError(""); }
  function beginApplicability() { setMode("applicability"); setRequirementID(aggregate.requirements.find((value) => value.status === "APPROVED")?.id ?? ""); setApplicability("APPLICABLE"); setScopeDescription(""); setRationale(""); setEffectiveFrom(today()); setError(""); }

  async function saveRequirement(event: FormEvent) {
    event.preventDefault(); setBusy(true); setError("");
    const input = { code, title, statement, sourceAnchor, modality, actor, action, object, effectiveFrom: isoDate(effectiveFrom) };
    try {
      const value = mode === "replace" && selected
        ? await supersedeProgramRequirement(program.id, selected.id, program.version, { ...input, rationale })
        : await addProgramRequirement(program.id, program.version, input);
      onUpdated(value); setMode(null);
    } catch (value) { setError(value instanceof Error ? value.message : "The requirement could not be saved."); }
    finally { setBusy(false); }
  }

  async function saveApplicability(event: FormEvent) {
    event.preventDefault(); setBusy(true); setError("");
    try {
      const value = await determineProgramApplicability(program.id, program.version, { requirementID, status: applicability, scope: { description: scopeDescription.trim() }, rationale, effectiveFrom: isoDate(effectiveFrom) });
      onUpdated(value); setMode(null);
    } catch (value) { setError(value instanceof Error ? value.message : "The applicability decision could not be saved."); }
    finally { setBusy(false); }
  }

  return <article className="program-record-panel program-requirements-panel" id="program-requirements-panel">
    <div className="program-panel-heading"><div><span className="eyebrow">Requirements</span><h2>Obligations and applicability</h2></div><div className="program-panel-actions">{addOperation?.can_act && mode !== "add" && <button className="secondary-button" type="button" onClick={beginAdd}>Add requirement</button>}{applicabilityOperation?.can_act && aggregate.requirements.some((value) => value.status === "APPROVED") && mode !== "applicability" && <button className="secondary-button" type="button" onClick={beginApplicability}>Record applicability</button>}</div></div>
    <div className="program-requirement-list">{aggregate.requirements.length ? aggregate.requirements.map((requirement) => {
      const supersedeOperation = operations.find((operation) => operation.command === "program.requirement.supersede" && operation.subresource_id === requirement.id);
      const currentApplicability = [...aggregate.applicability].reverse().find((value) => value.requirement_id === requirement.id);
      return <section className="program-requirement-card" key={requirement.id}><div><span>{requirement.code} · {label(requirement.status)}</span><h3>{requirement.title}</h3><p>{requirement.statement}</p>{requirement.source_anchor && <small>Source: {requirement.source_anchor}</small>}</div><dl><div><dt>Does this apply?</dt><dd>{currentApplicability ? label(currentApplicability.status) : "Not decided"}</dd></div><div><dt>Effective from</dt><dd>{requirement.effective_from?.slice(0, 10) ?? "Not recorded"}</dd></div></dl>{supersedeOperation?.can_act && <button className="text-button" type="button" onClick={() => beginReplace(requirement)}>Replace requirement</button>}</section>;
    }) : <div className="program-empty-state"><strong>No requirements are recorded</strong><p>Add a source-anchored obligation before deciding whether it applies or defining its safeguards.</p></div>}</div>
    {(mode === "add" || mode === "replace") && <form className="program-operation-form" onSubmit={(event) => void saveRequirement(event)}>
      <label><span>Requirement code</span><input required value={code} onChange={(event) => setCode(event.target.value)}/></label>
      <label><span>Requirement title</span><input required value={title} onChange={(event) => setTitle(event.target.value)}/></label>
      <label className="wide"><span>{mode === "replace" ? "Replacement statement" : "Requirement statement"}</span><textarea required value={statement} onChange={(event) => setStatement(event.target.value)}/></label>
      <label className="wide"><span>{mode === "replace" ? "Replacement source and section" : "Official source and section"}</span><input required value={sourceAnchor} onChange={(event) => setSourceAnchor(event.target.value)}/></label>
      <label><span>Obligation strength</span><select value={modality} onChange={(event) => setModality(event.target.value)}><option value="MUST">Must</option><option value="MUST_NOT">Must not</option><option value="SHOULD">Should</option><option value="MAY">May</option><option value="EXPECTED">Expected</option></select></label>
      <label><span>Effective from</span><input required type="date" value={effectiveFrom} onChange={(event) => setEffectiveFrom(event.target.value)}/></label>
      <label><span>Who must act?</span><input value={actor} onChange={(event) => setActor(event.target.value)}/></label>
      <label><span>What must they do?</span><input value={action} onChange={(event) => setAction(event.target.value)}/></label>
      <label className="wide"><span>What does it apply to?</span><input value={object} onChange={(event) => setObject(event.target.value)}/></label>
      {mode === "replace" && <label className="wide"><span>Reason for replacing this requirement</span><textarea required value={rationale} onChange={(event) => setRationale(event.target.value)}/></label>}
      {error && <p className="program-form-error wide" role="alert">{error} <button className="text-button" type="button" onClick={onReload}>Reload Program</button></p>}
      <div className="program-form-actions wide"><button className="primary-button" disabled={busy} type="submit">{busy ? "Saving…" : mode === "replace" ? "Save replacement requirement" : "Save requirement"}</button><button className="text-button" type="button" onClick={() => setMode(null)}>Cancel</button></div>
    </form>}
    {mode === "applicability" && <form className="program-operation-form" onSubmit={(event) => void saveApplicability(event)}>
      <label className="wide"><span>Requirement</span><select required value={requirementID} onChange={(event) => setRequirementID(event.target.value)}>{aggregate.requirements.filter((value) => value.status === "APPROVED").map((requirement) => <option key={requirement.id} value={requirement.id}>{requirement.code} · {requirement.title}</option>)}</select></label>
      <label><span>Does this apply?</span><select value={applicability} onChange={(event) => setApplicability(event.target.value)}><option value="APPLICABLE">Yes</option><option value="PARTIALLY_APPLICABLE">Partly</option><option value="NOT_APPLICABLE">No</option><option value="POTENTIALLY_APPLICABLE">Needs assessment</option><option value="APPLIES_LATER">Applies later</option></select></label>
      <label><span>Effective from</span><input required type="date" value={effectiveFrom} onChange={(event) => setEffectiveFrom(event.target.value)}/></label>
      <label className="wide"><span>Scope of this decision</span><textarea value={scopeDescription} onChange={(event) => setScopeDescription(event.target.value)}/></label>
      <label className="wide"><span>Applicability rationale</span><textarea required value={rationale} onChange={(event) => setRationale(event.target.value)}/></label>
      {error && <p className="program-form-error wide" role="alert">{error}</p>}
      <div className="program-form-actions wide"><button className="primary-button" disabled={busy || !requirementID} type="submit">{busy ? "Saving…" : "Save applicability decision"}</button><button className="text-button" type="button" onClick={() => setMode(null)}>Cancel</button></div>
    </form>}
    {!addOperation?.can_act && addOperation?.reason && <p className="program-operation-reason">{addOperation.reason}</p>}
  </article>;
}
