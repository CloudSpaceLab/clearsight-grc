import { useState } from "react";
import type { FormEvent } from "react";
import {
  addProgramControlImplementation,
  addProgramControlObjective,
  linkProgramRequirementControl,
} from "../programOperationsApi";
import type { ProgramOperation } from "../programOperationsApi";
import type { ProgramAggregate } from "../types";

type Props = {
  aggregate: ProgramAggregate;
  operations: ProgramOperation[];
  onUpdated: (value: ProgramAggregate) => void;
  onReload: () => void;
};
type Mode = "objective" | "safeguard" | "link" | null;

function today() { return new Date().toISOString().slice(0, 10); }
function isoDate(value: string) { return new Date(`${value}T00:00:00Z`).toISOString(); }
function statusLabel(value: string) {
  return value.replaceAll("_", " ").toLowerCase().replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}

export function ProgramSafeguardsPanel({ aggregate, operations, onUpdated, onReload }: Props) {
  const operation = operations.find((value) => value.command === "program.safeguard.define");
  const [mode, setMode] = useState<Mode>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [objectiveCode, setObjectiveCode] = useState("");
  const [objectiveName, setObjectiveName] = useState("");
  const [outcome, setOutcome] = useState("");
  const [objectiveID, setObjectiveID] = useState(aggregate.control_objectives[0]?.id ?? "");
  const [safeguardName, setSafeguardName] = useState("");
  const [description, setDescription] = useState("");
  const [implementationType, setImplementationType] = useState("CHECKLIST");
  const [ownerID, setOwnerID] = useState(operation?.candidates?.[0]?.id ?? "");
  const [scopeDescription, setScopeDescription] = useState("");
  const [effectiveFrom, setEffectiveFrom] = useState(today());
  const [requirementID, setRequirementID] = useState(aggregate.requirements.find((value) => value.status === "APPROVED")?.id ?? "");
  const [implementationID, setImplementationID] = useState(aggregate.control_implementations[0]?.id ?? "");

  function begin(nextMode: Exclude<Mode, null>) {
    setMode(nextMode);
    setError("");
    if (nextMode === "objective") {
      setObjectiveCode(""); setObjectiveName(""); setOutcome("");
    } else if (nextMode === "safeguard") {
      setObjectiveID(aggregate.control_objectives[0]?.id ?? "");
      setSafeguardName(""); setDescription(""); setImplementationType("CHECKLIST");
      setOwnerID(operation?.candidates?.[0]?.id ?? ""); setScopeDescription(""); setEffectiveFrom(today());
    } else {
      setRequirementID(aggregate.requirements.find((value) => value.status === "APPROVED")?.id ?? "");
      setImplementationID(aggregate.control_implementations[0]?.id ?? "");
    }
  }

  async function saveObjective(event: FormEvent) {
    event.preventDefault(); setBusy(true); setError("");
    try {
      const value = await addProgramControlObjective(aggregate.program.id, aggregate.program.version, {
        code: objectiveCode, name: objectiveName, outcome, status: "ACTIVE",
      });
      onUpdated(value); setMode(null);
    } catch (value) {
      setError(value instanceof Error ? value.message : "The control objective could not be saved.");
    } finally { setBusy(false); }
  }

  async function saveSafeguard(event: FormEvent) {
    event.preventDefault(); setBusy(true); setError("");
    try {
      const value = await addProgramControlImplementation(aggregate.program.id, aggregate.program.version, {
        objectiveID,
        name: safeguardName,
        description,
        implementationType,
        ownerPrincipalID: ownerID,
        scope: { description: scopeDescription.trim() },
        status: "IMPLEMENTED",
        effectiveFrom: isoDate(effectiveFrom),
      });
      onUpdated(value); setMode(null);
    } catch (value) {
      setError(value instanceof Error ? value.message : "The safeguard could not be saved.");
    } finally { setBusy(false); }
  }

  async function saveLink(event: FormEvent) {
    event.preventDefault(); setBusy(true); setError("");
    try {
      const value = await linkProgramRequirementControl(aggregate.program.id, aggregate.program.version, requirementID, implementationID);
      onUpdated(value); setMode(null);
    } catch (value) {
      setError(value instanceof Error ? value.message : "The requirement coverage link could not be saved.");
    } finally { setBusy(false); }
  }

  const covered = new Set(aggregate.requirement_control_links.map((link) => `${link.requirement_id}:${link.implementation_id}`));

  return <article className="program-record-panel program-safeguards-panel" id="program-safeguards-panel">
    <div className="program-panel-heading">
      <div><span className="eyebrow">Safeguards</span><h2>Safeguards and coverage</h2></div>
      {operation?.can_act && <div className="program-panel-actions">
        <button className="secondary-button" type="button" onClick={() => begin("objective")}>Add control objective</button>
        {aggregate.control_objectives.length > 0 && <button className="secondary-button" type="button" onClick={() => begin("safeguard")}>Add safeguard</button>}
        {aggregate.requirements.some((value) => value.status === "APPROVED") && aggregate.control_implementations.length > 0 && <button className="secondary-button" type="button" onClick={() => begin("link")}>Link requirement to safeguard</button>}
      </div>}
    </div>

    {aggregate.control_objectives.length === 0 ? <div className="program-empty-state"><strong>No control objectives are recorded</strong><p>Add the outcome the bank must maintain, then record the safeguard and accountable performer.</p></div> : <div className="program-safeguard-list">
      {aggregate.control_objectives.map((objective) => {
        const implementations = aggregate.control_implementations.filter((value) => value.objective_id === objective.id);
        return <section className="program-safeguard-card" key={objective.id}>
          <div><span>{objective.code} · {statusLabel(objective.status)}</span><h3>{objective.name}</h3><p>{objective.outcome}</p></div>
          {implementations.length ? <ul>{implementations.map((implementation) => {
            const requirementCount = aggregate.requirement_control_links.filter((link) => link.implementation_id === implementation.id).length;
            const owner = operation?.candidates?.find((candidate) => candidate.id === implementation.owner_principal_id)?.display_name ?? implementation.owner_principal_id ?? "Owner not assigned";
            return <li key={implementation.id}><strong>{implementation.name}</strong><span>{owner} · {statusLabel(implementation.status)} · {requirementCount} linked requirement{requirementCount === 1 ? "" : "s"}</span><p>{implementation.description}</p></li>;
          })}</ul> : <p>No safeguards implement this objective yet.</p>}
        </section>;
      })}
    </div>}

    {mode === "objective" && <form className="program-operation-form" onSubmit={(event) => void saveObjective(event)}>
      <label><span>Objective code</span><input required value={objectiveCode} onChange={(event) => setObjectiveCode(event.target.value)}/></label>
      <label><span>Objective name</span><input required value={objectiveName} onChange={(event) => setObjectiveName(event.target.value)}/></label>
      <label className="wide"><span>Intended outcome</span><textarea required value={outcome} onChange={(event) => setOutcome(event.target.value)}/></label>
      {error && <p className="program-form-error wide" role="alert">{error} <button className="text-button" type="button" onClick={onReload}>Reload Program</button></p>}
      <div className="program-form-actions wide"><button className="primary-button" disabled={busy} type="submit">{busy ? "Saving…" : "Save control objective"}</button><button className="text-button" type="button" onClick={() => setMode(null)}>Cancel</button></div>
    </form>}

    {mode === "safeguard" && <form className="program-operation-form" onSubmit={(event) => void saveSafeguard(event)}>
      <label><span>Control objective</span><select required value={objectiveID} onChange={(event) => setObjectiveID(event.target.value)}>{aggregate.control_objectives.map((objective) => <option key={objective.id} value={objective.id}>{objective.code} · {objective.name}</option>)}</select></label>
      <label><span>Safeguard owner</span><select required value={ownerID} onChange={(event) => setOwnerID(event.target.value)}><option value="" disabled>Select an eligible owner</option>{operation?.candidates?.map((candidate) => <option key={candidate.id} value={candidate.id}>{candidate.display_name} · {candidate.role}</option>)}</select></label>
      <label><span>Safeguard name</span><input required value={safeguardName} onChange={(event) => setSafeguardName(event.target.value)}/></label>
      <label><span>Safeguard type</span><select value={implementationType} onChange={(event) => setImplementationType(event.target.value)}><option value="CHECKLIST">Checklist</option><option value="REVIEW">Review</option><option value="RECONCILIATION">Reconciliation</option><option value="SYSTEM_CONTROL">System control</option><option value="MONITORING">Monitoring</option></select></label>
      <label className="wide"><span>How the safeguard works</span><textarea required value={description} onChange={(event) => setDescription(event.target.value)}/></label>
      <label className="wide"><span>Safeguard scope</span><textarea value={scopeDescription} onChange={(event) => setScopeDescription(event.target.value)}/></label>
      <label><span>Effective from</span><input required type="date" value={effectiveFrom} onChange={(event) => setEffectiveFrom(event.target.value)}/></label>
      {error && <p className="program-form-error wide" role="alert">{error} <button className="text-button" type="button" onClick={onReload}>Reload Program</button></p>}
      <div className="program-form-actions wide"><button className="primary-button" disabled={busy || !ownerID || !objectiveID} type="submit">{busy ? "Saving…" : "Save safeguard"}</button><button className="text-button" type="button" onClick={() => setMode(null)}>Cancel</button></div>
    </form>}

    {mode === "link" && <form className="program-operation-form" onSubmit={(event) => void saveLink(event)}>
      <label><span>Requirement</span><select required value={requirementID} onChange={(event) => setRequirementID(event.target.value)}>{aggregate.requirements.filter((value) => value.status === "APPROVED").map((requirement) => <option key={requirement.id} value={requirement.id}>{requirement.code} · {requirement.title}</option>)}</select></label>
      <label><span>Safeguard</span><select required value={implementationID} onChange={(event) => setImplementationID(event.target.value)}>{aggregate.control_implementations.map((implementation) => <option disabled={covered.has(`${requirementID}:${implementation.id}`)} key={implementation.id} value={implementation.id}>{implementation.name}{covered.has(`${requirementID}:${implementation.id}`) ? " · already linked" : ""}</option>)}</select></label>
      {error && <p className="program-form-error wide" role="alert">{error} <button className="text-button" type="button" onClick={onReload}>Reload Program</button></p>}
      <div className="program-form-actions wide"><button className="primary-button" disabled={busy || !requirementID || !implementationID || covered.has(`${requirementID}:${implementationID}`)} type="submit">{busy ? "Saving…" : "Save coverage link"}</button><button className="text-button" type="button" onClick={() => setMode(null)}>Cancel</button></div>
    </form>}

    {!operation?.can_act && operation?.reason && <p className="program-operation-reason">{operation.reason}</p>}
  </article>;
}
