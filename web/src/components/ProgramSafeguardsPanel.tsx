import { useState } from "react";
import type { FormEvent } from "react";
import {
  addProgramControlImplementation,
  addProgramControlObjective,
  assignProgramControlImplementation,
  linkProgramRequirementControl,
  retireProgramRequirementControlLink,
  reviseProgramControlImplementation,
  transitionProgramControlImplementation,
} from "../programOperationsApi";
import type { ProgramOperation } from "../programOperationsApi";
import type { ProgramAggregate, RecordResponsibleParty } from "../types";

type Props = {
  aggregate: ProgramAggregate;
  operations: ProgramOperation[];
  responsibleParties?: RecordResponsibleParty[];
  onUpdated: (value: ProgramAggregate) => void;
  onReload: () => void;
};
type Mode = "objective" | "safeguard" | "link" | null;
type ResourceAction = { kind: "edit" | "assign" | "transition"; implementationID: string } | null;

function today() { return new Date().toISOString().slice(0, 10); }
function isoDate(value: string) { return new Date(`${value}T00:00:00Z`).toISOString(); }
function statusLabel(value: string) {
  return value.replaceAll("_", " ").toLowerCase().replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}

export function ProgramSafeguardsPanel({ aggregate, operations, responsibleParties = [], onUpdated, onReload }: Props) {
  const operation = operations.find((value) => value.command === "program.safeguard.define");
  const [mode, setMode] = useState<Mode>(null);
  const [resourceAction, setResourceAction] = useState<ResourceAction>(null);
  const [retiringLinkID, setRetiringLinkID] = useState<string | null>(null);
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
  const [rationale, setRationale] = useState("");
  const [transitionTarget, setTransitionTarget] = useState("");

  const operationFor = (command: string, subresourceID: string) => operations.find((value) => value.command === command && value.subresource_id === subresourceID);

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

  function beginResourceAction(kind: "edit" | "assign" | "transition", implementationID: string) {
    const implementation = aggregate.control_implementations.find((value) => value.id === implementationID);
    if (!implementation) return;
    setMode(null); setResourceAction({ kind, implementationID }); setError(""); setRationale("");
    if (kind === "edit") {
      setSafeguardName(implementation.name); setDescription(implementation.description);
      setImplementationType(implementation.implementation_type); setScopeDescription(String(implementation.scope?.description ?? ""));
      setEffectiveFrom(implementation.effective_from?.slice(0, 10) ?? today());
    } else if (kind === "assign") {
      setOwnerID(operationFor("program.safeguard.assign", implementationID)?.candidates?.[0]?.id ?? "");
    } else {
      setTransitionTarget(operationFor("program.safeguard.transition", implementationID)?.allowed_targets?.[0] ?? "");
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
        status: "PLANNED",
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

  async function retireLink(event: FormEvent) {
    event.preventDefault();
    if (!retiringLinkID) return;
    setBusy(true); setError("");
    try {
      const value = await retireProgramRequirementControlLink(aggregate.program.id, retiringLinkID, aggregate.program.version, rationale);
      onUpdated(value); setRetiringLinkID(null); setRationale("");
    } catch (value) {
      setError(value instanceof Error ? value.message : "The coverage link could not be removed.");
    } finally { setBusy(false); }
  }

  async function saveResourceAction(event: FormEvent) {
    event.preventDefault();
    if (!resourceAction) return;
    const implementation = aggregate.control_implementations.find((value) => value.id === resourceAction.implementationID);
    if (!implementation?.version) return;
    setBusy(true); setError("");
    try {
      let value: ProgramAggregate;
      if (resourceAction.kind === "edit") {
        value = await reviseProgramControlImplementation(aggregate.program.id, implementation.id, aggregate.program.version, implementation.version, {
          name: safeguardName, description, implementationType, scope: { description: scopeDescription.trim() },
          effectiveFrom: isoDate(effectiveFrom), rationale,
        });
      } else if (resourceAction.kind === "assign") {
        value = await assignProgramControlImplementation(aggregate.program.id, implementation.id, aggregate.program.version, implementation.version, ownerID, rationale);
      } else {
        value = await transitionProgramControlImplementation(aggregate.program.id, implementation.id, aggregate.program.version, implementation.version, transitionTarget, rationale);
      }
      onUpdated(value); setResourceAction(null);
    } catch (value) {
      setError(value instanceof Error ? value.message : "The safeguard change could not be saved.");
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
            const implementationLinks = aggregate.requirement_control_links.filter((link) => link.implementation_id === implementation.id);
            const requirementCount = implementationLinks.length;
            const owner = responsibleParties.find((party) => party.scope === "SAFEGUARD" && party.subresource_id === implementation.id && party.responsibility === "PERFORMER")?.display_name ?? operation?.candidates?.find((candidate) => candidate.id === implementation.owner_principal_id)?.display_name ?? (implementation.owner_principal_id ? "Recorded safeguard owner unavailable" : "Safeguard owner not assigned");
            const updateOperation = operationFor("program.safeguard.update", implementation.id);
            const assignOperation = operationFor("program.safeguard.assign", implementation.id);
            const transitionOperation = operationFor("program.safeguard.transition", implementation.id);
            return <li key={implementation.id}><strong>{implementation.name}</strong><span>{owner} · {statusLabel(implementation.status)} · {requirementCount} linked requirement{requirementCount === 1 ? "" : "s"}</span><p>{implementation.description}</p>
              {implementationLinks.length > 0 && <ul>{implementationLinks.map((link) => {
                const requirement = aggregate.requirements.find((value) => value.id === link.requirement_id);
                const unlinkOperation = link.id ? operationFor("program.safeguard.unlink", link.id) : undefined;
                const requirementLabel = requirement?.title ?? "Recorded requirement";
                return <li key={link.id ?? `${link.requirement_id}:${link.implementation_id}`}>
                  <span>{requirementLabel} → {implementation.name}</span>
                  {unlinkOperation?.can_act && link.id && <button className="text-button" type="button" onClick={() => { setMode(null); setResourceAction(null); setRetiringLinkID(link.id!); setRationale(""); setError(""); }}>Remove {requirementLabel} coverage link</button>}
                </li>;
              })}</ul>}
              <div className="program-panel-actions">
              {updateOperation?.can_act && <button className="text-button" type="button" onClick={() => beginResourceAction("edit", implementation.id)}>Edit {implementation.name}</button>}
              {assignOperation?.can_act && <button className="text-button" type="button" onClick={() => beginResourceAction("assign", implementation.id)}>Change {implementation.name} owner</button>}
              {transitionOperation?.can_act && <button className="text-button" type="button" onClick={() => beginResourceAction("transition", implementation.id)}>Change {implementation.name} status</button>}
            </div>{!transitionOperation?.can_act && transitionOperation?.reason && <small>{transitionOperation.reason}</small>}</li>;
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

    {resourceAction && (() => {
      const implementation = aggregate.control_implementations.find((value) => value.id === resourceAction.implementationID);
      if (!implementation) return null;
      const assignOperation = operationFor("program.safeguard.assign", implementation.id);
      const transitionOperation = operationFor("program.safeguard.transition", implementation.id);
      return <form className="program-operation-form" onSubmit={(event) => void saveResourceAction(event)}>
        <div className="wide"><strong>{resourceAction.kind === "edit" ? `Edit ${implementation.name}` : resourceAction.kind === "assign" ? `Change ${implementation.name} owner` : `Change ${implementation.name} status`}</strong></div>
        {resourceAction.kind === "edit" && <>
          <label><span>Safeguard name</span><input required value={safeguardName} onChange={(event) => setSafeguardName(event.target.value)}/></label>
          <label><span>Safeguard type</span><select value={implementationType} onChange={(event) => setImplementationType(event.target.value)}><option value="CHECKLIST">Checklist</option><option value="REVIEW">Review</option><option value="RECONCILIATION">Reconciliation</option><option value="SYSTEM_CONTROL">System control</option><option value="MONITORING">Monitoring</option></select></label>
          <label className="wide"><span>How the safeguard works</span><textarea required value={description} onChange={(event) => setDescription(event.target.value)}/></label>
          <label className="wide"><span>Safeguard scope</span><textarea value={scopeDescription} onChange={(event) => setScopeDescription(event.target.value)}/></label>
          <label><span>Effective from</span><input required type="date" value={effectiveFrom} onChange={(event) => setEffectiveFrom(event.target.value)}/></label>
        </>}
        {resourceAction.kind === "assign" && <label><span>New safeguard owner</span><select required value={ownerID} onChange={(event) => setOwnerID(event.target.value)}><option value="" disabled>Select an eligible owner</option>{assignOperation?.candidates?.map((candidate) => <option key={candidate.id} value={candidate.id}>{candidate.display_name} · {candidate.role}</option>)}</select></label>}
        {resourceAction.kind === "transition" && <label><span>New safeguard status</span><select required value={transitionTarget} onChange={(event) => setTransitionTarget(event.target.value)}>{transitionOperation?.allowed_targets?.map((target) => <option key={target} value={target}>{statusLabel(target)}</option>)}</select></label>}
        <label className="wide"><span>Reason for this change</span><textarea required value={rationale} onChange={(event) => setRationale(event.target.value)}/></label>
        {resourceAction.kind === "edit" && implementation.status === "IMPLEMENTED" && <p className="program-operation-reason wide">Saving a material change returns this safeguard to In progress until its owner confirms the revised procedure is operating.</p>}
        {error && <p className="program-form-error wide" role="alert">{error} <button className="text-button" type="button" onClick={onReload}>Reload Program</button></p>}
        <div className="program-form-actions wide"><button className="primary-button" disabled={busy || !rationale || (resourceAction.kind === "assign" && !ownerID) || (resourceAction.kind === "transition" && !transitionTarget)} type="submit">{busy ? "Saving…" : resourceAction.kind === "edit" ? "Save safeguard changes" : resourceAction.kind === "assign" ? "Save safeguard owner" : "Save safeguard status"}</button><button className="text-button" type="button" onClick={() => setResourceAction(null)}>Cancel</button></div>
      </form>;
    })()}

    {mode === "link" && <form className="program-operation-form" onSubmit={(event) => void saveLink(event)}>
      <label><span>Requirement</span><select required value={requirementID} onChange={(event) => setRequirementID(event.target.value)}>{aggregate.requirements.filter((value) => value.status === "APPROVED").map((requirement) => <option key={requirement.id} value={requirement.id}>{requirement.code} · {requirement.title}</option>)}</select></label>
      <label><span>Safeguard</span><select required value={implementationID} onChange={(event) => setImplementationID(event.target.value)}>{aggregate.control_implementations.map((implementation) => <option disabled={covered.has(`${requirementID}:${implementation.id}`)} key={implementation.id} value={implementation.id}>{implementation.name}{covered.has(`${requirementID}:${implementation.id}`) ? " · already linked" : ""}</option>)}</select></label>
      {error && <p className="program-form-error wide" role="alert">{error} <button className="text-button" type="button" onClick={onReload}>Reload Program</button></p>}
      <div className="program-form-actions wide"><button className="primary-button" disabled={busy || !requirementID || !implementationID || covered.has(`${requirementID}:${implementationID}`)} type="submit">{busy ? "Saving…" : "Save coverage link"}</button><button className="text-button" type="button" onClick={() => setMode(null)}>Cancel</button></div>
    </form>}

    {retiringLinkID && (() => {
      const link = aggregate.requirement_control_links.find((value) => value.id === retiringLinkID);
      if (!link) return null;
      const requirement = aggregate.requirements.find((value) => value.id === link.requirement_id);
      const implementation = aggregate.control_implementations.find((value) => value.id === link.implementation_id);
      return <form className="program-operation-form" onSubmit={(event) => void retireLink(event)}>
        <div className="wide"><strong>Remove {requirement?.title ?? "requirement"} coverage link</strong><p>This stops {implementation?.name ?? "this safeguard"} from counting as current coverage. The former link remains available in Program history.</p></div>
        <label className="wide"><span>Reason for removing this coverage link</span><textarea required value={rationale} onChange={(event) => setRationale(event.target.value)}/></label>
        {error && <p className="program-form-error wide" role="alert">{error} <button className="text-button" type="button" onClick={onReload}>Reload Program</button></p>}
        <div className="program-form-actions wide"><button className="primary-button" disabled={busy || !rationale.trim()} type="submit">{busy ? "Removing…" : "Remove coverage link"}</button><button className="text-button" type="button" onClick={() => { setRetiringLinkID(null); setRationale(""); }}>Cancel</button></div>
      </form>;
    })()}

    {!operation?.can_act && operation?.reason && <p className="program-operation-reason">{operation.reason}</p>}
  </article>;
}
