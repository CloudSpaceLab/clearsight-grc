import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import { addProgramRequirement, createProgram, loadProgramSetupCandidates } from "../continuityCommands";
import type { ProgramSetupCandidates } from "../continuityCommands";
import type { ProgramAggregate } from "../types";
import { MonitoringSetup } from "./MonitoringSetup";

type Props = { actorPrincipalID: string; canConfigureSources: boolean; onCreated: (aggregate: ProgramAggregate) => void; onClose: () => void };

export function ProgramSetupWorkspace({ actorPrincipalID, canConfigureSources, onCreated, onClose }: Props) {
  const [aggregate, setAggregate] = useState<ProgramAggregate | null>(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [candidates, setCandidates] = useState<ProgramSetupCandidates | null>(null);
  const [candidateState, setCandidateState] = useState<"loading" | "live" | "unavailable">("loading");
  const [ownerCandidateID, setOwnerCandidateID] = useState("");
  const [approvalAuthorityCandidateID, setApprovalAuthorityCandidateID] = useState("");

  async function loadCandidates() {
    setCandidateState("loading");
    try {
      const value = await loadProgramSetupCandidates();
      setCandidates(value);
      setOwnerCandidateID(value.owner_candidates[0]?.id ?? "");
      setApprovalAuthorityCandidateID(value.approval_authority_candidates.find((candidate) => candidate.id !== value.owner_candidates[0]?.id)?.id ?? "");
      setCandidateState("live");
    } catch {
      setCandidates(null); setCandidateState("unavailable");
    }
  }

  useEffect(() => { void loadCandidates(); }, []);

  async function saveProgram(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    setSaving(true); setError("");
    try {
      const created = await createProgram({
        name: String(data.get("name") ?? "").trim(), code: String(data.get("code") ?? "").trim(),
        type: String(data.get("type") ?? "CHANNEL"), owningFunction: String(data.get("owning_function") ?? "").trim(),
        jurisdiction: String(data.get("jurisdiction") ?? "").trim(), scopeDescription: String(data.get("scope") ?? "").trim(),
        ownerCandidateID, approvalAuthorityCandidateID,
      });
      setAggregate(created); onCreated(created); setNotice("Program created. Add its requirements and monitoring checks.");
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "The Program could not be created.");
    } finally { setSaving(false); }
  }

  async function saveRequirement(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!aggregate) return;
    const form = event.currentTarget;
    const data = new FormData(form);
    setSaving(true); setError("");
    try {
      const updated = await addProgramRequirement(aggregate.program.id, aggregate.program.version, {
        code: String(data.get("code") ?? "").trim(), title: String(data.get("title") ?? "").trim(),
        statement: String(data.get("statement") ?? "").trim(), sourceAnchor: String(data.get("source_anchor") ?? "").trim(),
      });
      setAggregate(updated); onCreated(updated); form.reset(); setNotice("Requirement added.");
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "The requirement could not be added.");
    } finally { setSaving(false); }
  }

  return <section className="program-setup-workspace" aria-labelledby="program-setup-title">
    <div className="setup-heading"><div><span className="eyebrow">Program setup</span><h2 id="program-setup-title">{aggregate ? aggregate.program.name : "New Program"}</h2><p>{aggregate ? "Add requirements and the checks that will monitor them." : "Define the activity, channel or obligation that needs ongoing oversight."}</p></div><button className="text-button" type="button" onClick={onClose}>Close</button></div>
    <ol className="setup-progress" aria-label="Program setup progress"><li className="current"><span>1</span>Program</li><li className={aggregate ? "current" : ""}><span>2</span>Requirements</li><li className={aggregate ? "current" : ""}><span>3</span>Monitoring</li></ol>
    {error && <p className="inline-form-error" role="alert">{error}</p>}
    {notice && <p className="inline-success" role="status">{notice}</p>}
    {!aggregate && candidateState === "loading" && <p className="inline-notice" role="status">Checking current Program ownership and approval responsibilities.</p>}
    {!aggregate && candidateState === "unavailable" && <p className="inline-form-error" role="alert">Current Program responsibilities could not be confirmed. Program creation is disabled. <button className="text-button" type="button" onClick={() => void loadCandidates()}>Retry responsibilities</button></p>}
    {!aggregate ? <form className="setup-form" onSubmit={saveProgram}>
      <div className="monitoring-form-grid">
        <label><span>Program name</span><input name="name" required placeholder="Mobile banking"/></label>
        <label><span>Code</span><input name="code" required placeholder="MOBILE"/></label>
        <label><span>Program type</span><select name="type" defaultValue="CHANNEL"><option value="CHANNEL">Channel</option><option value="REGULATORY">Regulatory obligation</option><option value="CYBERSECURITY">Cybersecurity</option><option value="OPERATIONS">Operations</option><option value="THIRD_PARTY">Third party</option><option value="PRIVACY">Privacy</option></select></label>
        <label><span>Owning function</span><input name="owning_function" required placeholder="Digital Banking"/></label>
        <label><span>Accountable owner</span><select required disabled={candidateState !== "live"} value={ownerCandidateID} onChange={(event) => { const value = event.target.value; setOwnerCandidateID(value); if (approvalAuthorityCandidateID === value) setApprovalAuthorityCandidateID(""); }}><option value="">Select an eligible owner</option>{candidates?.owner_candidates.map((candidate) => <option key={candidate.id} value={candidate.id}>{candidate.display_name}{candidate.role ? ` · ${candidate.role}` : ""}</option>)}</select></label>
        <label><span>Approval authority</span><select required disabled={candidateState !== "live"} value={approvalAuthorityCandidateID} onChange={(event) => setApprovalAuthorityCandidateID(event.target.value)}><option value="">Select an eligible approver</option>{candidates?.approval_authority_candidates.filter((candidate) => candidate.id !== ownerCandidateID).map((candidate) => <option key={candidate.id} value={candidate.id}>{candidate.display_name}{candidate.role ? ` · ${candidate.role}` : ""}</option>)}</select></label>
        <label><span>Jurisdiction</span><input name="jurisdiction" placeholder="Nigeria"/></label>
        <label className="full"><span>Scope</span><textarea name="scope" rows={3} required placeholder="Retail mobile banking channel, including customer authentication and password reset"/></label>
      </div>
      <div className="monitoring-form-actions"><button className="text-button" type="button" onClick={onClose}>Cancel</button><button className="primary-button" disabled={saving || candidateState !== "live" || !ownerCandidateID || !approvalAuthorityCandidateID || ownerCandidateID === approvalAuthorityCandidateID} type="submit">{saving ? "Creating…" : "Create Program"}</button></div>
    </form> : <div className="setup-sections">
      <section className="setup-section"><div className="setup-section-title"><div><h3>Requirements</h3><p>Record what must be true for this Program.</p></div><span>{aggregate.requirements.length} added</span></div>
        {aggregate.requirements.length > 0 && <ul className="setup-requirement-list">{aggregate.requirements.map((requirement) => <li key={requirement.id}><strong>{requirement.title}</strong><span>{requirement.statement}</span></li>)}</ul>}
        <form className="requirement-form" onSubmit={saveRequirement}>
          <label><span>Title</span><input name="title" required placeholder="Live face verification is available"/></label>
          <label><span>Code</span><input name="code" required placeholder="FACE-VERIFY"/></label>
          <label className="full"><span>Requirement</span><textarea name="statement" required rows={3} placeholder="The mobile banking channel must complete live face verification during onboarding and high-risk account recovery."/></label>
          <label className="full"><span>Source reference (optional)</span><input name="source_anchor" placeholder="Policy, regulation or internal standard"/></label>
          <button className="secondary-button" type="submit" disabled={saving}>{saving ? "Adding…" : "Add requirement"}</button>
        </form>
      </section>
      <MonitoringSetup aggregate={aggregate} actorPrincipalID={actorPrincipalID} canConfigureSources={canConfigureSources}/>
    </div>}
  </section>;
}
