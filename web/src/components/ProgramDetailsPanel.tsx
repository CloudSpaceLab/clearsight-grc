import { useState } from "react";
import type { FormEvent } from "react";
import { assignProgram, assignProgramApprovalAuthority, updateProgramDetails } from "../programOperationsApi";
import type { ProgramOperation } from "../programOperationsApi";
import type { ProgramAggregate, RecordResponsibleParty } from "../types";

type Props = {
  aggregate: ProgramAggregate;
  operations: ProgramOperation[];
  responsibleParties?: RecordResponsibleParty[];
  onUpdated: (value: ProgramAggregate) => void;
  onReload: () => void;
};

type Mode = "details" | "owner" | "approval" | null;

function dateValue(value?: string) { return value?.slice(0, 10) ?? ""; }
function isoDate(value: string) { return value ? new Date(`${value}T00:00:00Z`).toISOString() : undefined; }
function scopeLines(scope: Record<string, unknown>) {
  const value = scope.business_lines;
  return Array.isArray(value) ? value.map(String).join(", ") : "";
}

export function ProgramDetailsPanel({ aggregate, operations, responsibleParties = [], onUpdated, onReload }: Props) {
  const program = aggregate.program;
  const detailsOperation = operations.find((operation) => operation.command === "program.details.update");
  const assignOperation = operations.find((operation) => operation.command === "program.assign");
  const approvalOperation = operations.find((operation) => operation.command === "program.approval-authority.assign");
  const owner = assignOperation?.assigned_to ?? detailsOperation?.assigned_to;
  const storedOwner = responsibleParties.find((party) => party.scope === "RECORD" && party.responsibility === "ACCOUNTABLE_OWNER")?.display_name;
  const approvalAuthority = approvalOperation?.assigned_to;
  const storedApprovalAuthority = responsibleParties.find((party) => party.scope === "RECORD" && party.responsibility === "AUTHORIZER")?.display_name;
  const [mode, setMode] = useState<Mode>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [name, setName] = useState(program.name);
  const [owningFunction, setOwningFunction] = useState(program.owning_function);
  const [jurisdiction, setJurisdiction] = useState(program.jurisdiction ?? "");
  const [businessLines, setBusinessLines] = useState(scopeLines(program.scope));
  const [scopeNotes, setScopeNotes] = useState(String(program.scope.description ?? ""));
  const [effectiveFrom, setEffectiveFrom] = useState(dateValue(program.effective_from));
  const [effectiveUntil, setEffectiveUntil] = useState(dateValue(program.effective_until));
  const [rationale, setRationale] = useState("");
  const [newOwner, setNewOwner] = useState("");
  const [ownerRationale, setOwnerRationale] = useState("");
  const [newApprovalAuthority, setNewApprovalAuthority] = useState("");
  const [approvalRationale, setApprovalRationale] = useState("");

  function beginDetails() {
    setName(program.name); setOwningFunction(program.owning_function); setJurisdiction(program.jurisdiction ?? "");
    setBusinessLines(scopeLines(program.scope)); setScopeNotes(String(program.scope.description ?? ""));
    setEffectiveFrom(dateValue(program.effective_from)); setEffectiveUntil(dateValue(program.effective_until));
    setRationale(""); setError(""); setMode("details");
  }

  function beginOwner() {
	const candidate = assignOperation?.candidates?.find((value) => value.id !== program.owner_principal_id && value.id !== program.authority_principal_id);
    setNewOwner(candidate?.id ?? ""); setOwnerRationale(""); setError(""); setMode("owner");
  }

  function beginApprovalAuthority() {
    const candidate = approvalOperation?.candidates?.find((value) => value.id !== program.authority_principal_id && value.id !== program.owner_principal_id);
    setNewApprovalAuthority(candidate?.id ?? ""); setApprovalRationale(""); setError(""); setMode("approval");
  }

  async function saveDetails(event: FormEvent) {
    event.preventDefault();
    setBusy(true); setError("");
    try {
      const value = await updateProgramDetails(program.id, program.version, {
        name, owningFunction, jurisdiction,
        scope: {
          ...program.scope,
          business_lines: businessLines.split(",").map((item) => item.trim()).filter(Boolean),
          description: scopeNotes.trim(),
        },
        effectiveFrom: isoDate(effectiveFrom)!, effectiveUntil: isoDate(effectiveUntil), rationale,
      });
      onUpdated(value); setMode(null);
    } catch (value) {
      setError(value instanceof Error ? value.message : "Program details could not be saved.");
    } finally { setBusy(false); }
  }

  async function saveOwner(event: FormEvent) {
    event.preventDefault();
    setBusy(true); setError("");
    try {
      const value = await assignProgram(program.id, program.version, newOwner, ownerRationale);
      onUpdated(value); setMode(null);
    } catch (value) {
      setError(value instanceof Error ? value.message : "The Program owner could not be changed.");
    } finally { setBusy(false); }
  }

  async function saveApprovalAuthority(event: FormEvent) {
    event.preventDefault(); setBusy(true); setError("");
    try {
      const value = await assignProgramApprovalAuthority(program.id, program.version, newApprovalAuthority, approvalRationale);
      onUpdated(value); setMode(null);
    } catch (value) {
      setError(value instanceof Error ? value.message : "The Program approval authority could not be changed.");
    } finally { setBusy(false); }
  }

  return <article className="program-record-panel" id="program-details-panel">
    <div className="program-panel-heading"><div><span className="eyebrow">Program details</span><h2>Scope and ownership</h2></div><div className="program-panel-actions">{detailsOperation?.can_act && mode !== "details" && <button className="secondary-button" type="button" onClick={beginDetails}>Edit Program details</button>}{assignOperation?.can_act && mode !== "owner" && <button className="secondary-button" type="button" onClick={beginOwner}>Change Program owner</button>}{approvalOperation?.can_act && mode !== "approval" && <button className="secondary-button" type="button" onClick={beginApprovalAuthority}>Change approval authority</button>}</div></div>
    <dl className="program-record-facts"><div><dt>Code</dt><dd>{program.code}</dd></div><div><dt>Owning function</dt><dd>{program.owning_function}</dd></div><div><dt>Jurisdiction</dt><dd>{program.jurisdiction || "Not recorded"}</dd></div><div><dt>Owner</dt><dd>{owner?.display_name ?? storedOwner ?? (program.owner_principal_id ? "Recorded Program owner unavailable" : "Program owner not assigned")}</dd></div><div><dt>Approval authority</dt><dd>{approvalAuthority?.display_name ?? storedApprovalAuthority ?? (program.authority_principal_id ? "Recorded approval authority unavailable" : "Approval authority not assigned")}</dd></div><div><dt>Effective from</dt><dd>{dateValue(program.effective_from) || "Not recorded"}</dd></div><div><dt>Effective until</dt><dd>{dateValue(program.effective_until) || "No end date"}</dd></div><div><dt>Business lines</dt><dd>{scopeLines(program.scope) || "Not recorded"}</dd></div></dl>
    {mode === "details" && <form className="program-operation-form" onSubmit={(event) => void saveDetails(event)}>
      <label><span>Program name</span><input required value={name} onChange={(event) => setName(event.target.value)}/></label>
      <label><span>Owning function</span><input required value={owningFunction} onChange={(event) => setOwningFunction(event.target.value)}/></label>
      <label><span>Jurisdiction</span><input value={jurisdiction} onChange={(event) => setJurisdiction(event.target.value)}/></label>
      <label><span>Business lines</span><input value={businessLines} onChange={(event) => setBusinessLines(event.target.value)} placeholder="Retail, Corporate"/></label>
      <label className="wide"><span>Scope notes</span><textarea value={scopeNotes} onChange={(event) => setScopeNotes(event.target.value)}/></label>
      <label><span>Effective from</span><input required type="date" value={effectiveFrom} onChange={(event) => setEffectiveFrom(event.target.value)}/></label>
      <label><span>Effective until</span><input type="date" value={effectiveUntil} onChange={(event) => setEffectiveUntil(event.target.value)}/></label>
      <label className="wide"><span>Reason for this change</span><textarea required value={rationale} onChange={(event) => setRationale(event.target.value)}/></label>
      {error && <p className="program-form-error wide" role="alert">{error} <button className="text-button" type="button" onClick={onReload}>Reload Program</button></p>}
      <div className="program-form-actions wide"><button className="primary-button" disabled={busy} type="submit">{busy ? "Saving…" : "Save Program details"}</button><button className="text-button" type="button" onClick={() => setMode(null)}>Cancel</button></div>
    </form>}
    {mode === "owner" && <form className="program-operation-form" onSubmit={(event) => void saveOwner(event)}>
      <label className="wide"><span>New Program owner</span><select required value={newOwner} onChange={(event) => setNewOwner(event.target.value)}><option value="">Select an eligible owner</option>{assignOperation?.candidates?.filter((candidate) => candidate.id !== program.owner_principal_id && candidate.id !== program.authority_principal_id).map((candidate) => <option key={candidate.id} value={candidate.id}>{candidate.display_name}</option>)}</select></label>
      <label className="wide"><span>Reason for changing owner</span><textarea required value={ownerRationale} onChange={(event) => setOwnerRationale(event.target.value)}/></label>
      {error && <p className="program-form-error wide" role="alert">{error}</p>}
      <div className="program-form-actions wide"><button className="primary-button" disabled={busy || !newOwner} type="submit">{busy ? "Saving…" : "Save Program owner"}</button><button className="text-button" type="button" onClick={() => setMode(null)}>Cancel</button></div>
    </form>}
    {mode === "approval" && <form className="program-operation-form" onSubmit={(event) => void saveApprovalAuthority(event)}>
      <label className="wide"><span>New approval authority</span><select required value={newApprovalAuthority} onChange={(event) => setNewApprovalAuthority(event.target.value)}><option value="">Select an eligible approver</option>{approvalOperation?.candidates?.filter((candidate) => candidate.id !== program.authority_principal_id && candidate.id !== program.owner_principal_id).map((candidate) => <option key={candidate.id} value={candidate.id}>{candidate.display_name}</option>)}</select></label>
      <label className="wide"><span>Reason for changing approval authority</span><textarea required value={approvalRationale} onChange={(event) => setApprovalRationale(event.target.value)}/></label>
      {error && <p className="program-form-error wide" role="alert">{error}</p>}
      <div className="program-form-actions wide"><button className="primary-button" disabled={busy || !newApprovalAuthority} type="submit">{busy ? "Saving…" : "Save approval authority"}</button><button className="text-button" type="button" onClick={() => setMode(null)}>Cancel</button></div>
    </form>}
    {!detailsOperation?.can_act && detailsOperation?.reason && <p className="program-operation-reason">{detailsOperation.reason}</p>}
  </article>;
}
