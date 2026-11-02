import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import { addProgramRequirement, determineProgramApplicability, supersedeProgramRequirement } from "../programOperationsApi";
import type { ProgramOperation } from "../programOperationsApi";
import type { ProgramAggregate, Requirement } from "../types";
import { useFrameworkClauseCandidates } from "../hooks/useFrameworkClauseCandidates";
import { decideProgramFrameworkMapping, proposeProgramFrameworkMapping } from "../programFrameworkCandidatesApi";
import type { FrameworkMappingKind } from "../programFrameworkCandidatesApi";
import { TextField } from "./ui";

type Props = { aggregate: ProgramAggregate; operations: ProgramOperation[]; reviewAdoptionID?: string; onUpdated: (value: ProgramAggregate) => void; onReload: () => void };
type Mode = "add" | "replace" | "applicability" | null;

function today() { return new Date().toISOString().slice(0, 10); }
function isoDate(value: string) { return new Date(`${value}T00:00:00Z`).toISOString(); }
function label(value?: string) { return (value || "Not recorded").replaceAll("_", " ").toLowerCase().replace(/(^|\s)\S/g, (letter) => letter.toUpperCase()); }

export function ProgramRequirementsPanel({ aggregate, operations, reviewAdoptionID, onUpdated, onReload }: Props) {
  const program = aggregate.program;
  const addOperation = operations.find((operation) => operation.command === "program.requirement.add");
  const applicabilityOperation = operations.find((operation) => operation.command === "program.applicability.decide");
  const activeStandards = (aggregate.framework_adoptions ?? []).filter((value) => value.status === "ACTIVE");
  // Only an active adoption from the current authorized Program may open
  // clause review; route intent alone is never a catalog identity or decision.
  const requestedAdoptionID = activeStandards.some((value) => value.id === reviewAdoptionID) ? reviewAdoptionID : undefined;
  const standardReview = useFrameworkClauseCandidates(program.id, program.version);
  const [showStandards, setShowStandards] = useState(false);
  const mappingProposalOperation = operations.find((operation) => operation.command === "program.framework.mapping.propose");
  const mappingApproveOperation = operations.find((operation) => operation.command === "program.framework.mapping.approve");
  const mappingRejectOperation = operations.find((operation) => operation.command === "program.framework.mapping.reject");
  const [selectedMapping, setSelectedMapping] = useState<{ clauseID: string; requirementID: string } | null>(null);
  const [mappingKind, setMappingKind] = useState<FrameworkMappingKind>("EXACT");
  const [mappingRationale, setMappingRationale] = useState("");
  const [mappingReviewRationale, setMappingReviewRationale] = useState("");
  const [reviewingMappingID, setReviewingMappingID] = useState<string | null>(null);
  const [mappingBusy, setMappingBusy] = useState(false);
  const [mappingError, setMappingError] = useState("");

  useEffect(() => {
    setShowStandards(Boolean(requestedAdoptionID));
    setSelectedMapping(null);
    setReviewingMappingID(null);
    setMappingReviewRationale("");
    setMappingError("");
  }, [program.id, program.version, requestedAdoptionID]);

  useEffect(() => {
    if (requestedAdoptionID) void standardReview.load(requestedAdoptionID);
  }, [requestedAdoptionID, program.id, program.version, standardReview.load]);
  const [mode, setMode] = useState<Mode>(null);
  const [selected, setSelected] = useState<Requirement | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [code, setCode] = useState(""); const [title, setTitle] = useState(""); const [statement, setStatement] = useState("");
  const [sourceAnchor, setSourceAnchor] = useState(""); const [modality, setModality] = useState("");
  const [actor, setActor] = useState(""); const [action, setAction] = useState(""); const [object, setObject] = useState("");
  const [effectiveFrom, setEffectiveFrom] = useState(today()); const [rationale, setRationale] = useState("");
  const [requirementID, setRequirementID] = useState(aggregate.requirements.find((value) => value.status === "APPROVED")?.id ?? "");
  const [applicability, setApplicability] = useState("APPLICABLE"); const [scopeDescription, setScopeDescription] = useState("");

  async function submitMapping(event: FormEvent) {
    event.preventDefault();
    if (!selectedMapping || !mappingProposalOperation?.can_act || !standardReview.adoptionID ||
        !mappingRationale.trim() || standardReview.mappingStatus !== "loaded" ||
        !standardReview.mappingPage || standardReview.mappingPage.next_after_id ||
        standardReview.mappingPage.items.some((mapping) =>
          mapping.clause_id === selectedMapping.clauseID && mapping.requirement_id === selectedMapping.requirementID)) return;
    setMappingBusy(true); setMappingError("");
    try {
      const result = await proposeProgramFrameworkMapping(
        program.id, standardReview.adoptionID, program.version,
        selectedMapping.clauseID, selectedMapping.requirementID,
        mappingKind, mappingRationale.trim(),
      );
      onUpdated(result.program);
      setSelectedMapping(null);
      setMappingRationale("");
    } catch (value) {
      setMappingError(value instanceof Error ? value.message : "Requirement link could not be submitted.");
    } finally { setMappingBusy(false); }
  }

  async function reviewMapping(mappingID: string, action: "approve" | "reject") {
    const allowed = action === "approve" ? mappingApproveOperation?.can_act : mappingRejectOperation?.can_act;
    if (!allowed || reviewingMappingID !== mappingID || !mappingReviewRationale.trim() || !standardReview.adoptionID) return;
    setMappingBusy(true); setMappingError("");
    try {
      const result = await decideProgramFrameworkMapping(
        program.id, standardReview.adoptionID, mappingID,
        program.version, action, mappingReviewRationale.trim(),
      );
      onUpdated(result.program);
      setMappingReviewRationale("");
      setReviewingMappingID(null);
    } catch (value) {
      setMappingError(value instanceof Error ? value.message : "Requirement link review could not be saved.");
    } finally { setMappingBusy(false); }
  }

  function beginAdd() { setMode("add"); setSelected(null); setCode(""); setTitle(""); setStatement(""); setSourceAnchor(""); setModality(""); setActor(""); setAction(""); setObject(""); setEffectiveFrom(today()); setRationale(""); setError(""); }
  function beginReplace(requirement: Requirement) { setMode("replace"); setSelected(requirement); setCode(requirement.code); setTitle(requirement.title); setStatement(requirement.statement); setSourceAnchor(requirement.source_anchor ?? ""); setModality(requirement.modality ?? ""); setActor(requirement.actor ?? ""); setAction(requirement.action ?? ""); setObject(requirement.object ?? ""); setEffectiveFrom(today()); setRationale(""); setError(""); }
  function beginApplicability() { setMode("applicability"); setRequirementID(aggregate.requirements.find((value) => value.status === "APPROVED")?.id ?? ""); setApplicability("APPLICABLE"); setScopeDescription(""); setRationale(""); setEffectiveFrom(today()); setError(""); }

  async function saveRequirement(event: FormEvent) {
    event.preventDefault();
    if (![actor, action, object].every((value) => value.trim())) { setError("Enter who must act, what they must do and what the requirement applies to."); return; }
    if (!modality) { setError("Select the obligation strength stated in the source."); return; }
    setBusy(true); setError("");
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
    {activeStandards.length > 0 && <section className="program-record-subsection" aria-label="Standard clause review">
      <div className="program-panel-heading">
        <div><h3>Standard clauses</h3><p className="program-operation-reason">Suggested requirement matches require review.</p></div>
        <button className="secondary-button" type="button" onClick={() => {
          setShowStandards((previous) => !previous);
          if (!showStandards && activeStandards[0]) void standardReview.load(activeStandards[0].id);
        }}>{showStandards ? "Hide clauses" : "Review clauses"}</button>
      </div>
      {showStandards && <>
        <label><span>Standard</span><select aria-label="Review standard" value={standardReview.adoptionID} onChange={(event) => { setSelectedMapping(null); setMappingRationale(""); setReviewingMappingID(null); setMappingReviewRationale(""); setMappingError(""); void standardReview.load(event.target.value); }}>
          {activeStandards.map((standard) => <option key={standard.id} value={standard.id}>{standard.publisher_code} · {standard.framework_code} {standard.edition_code}</option>)}
        </select></label>
        {standardReview.status === "loading" && <p role="status">Loading clauses…</p>}
        {standardReview.status === "error" && <p role="alert">Clause matches could not be loaded. <button className="text-button" type="button" onClick={() => void standardReview.load(standardReview.adoptionID)}>Retry</button></p>}
        {standardReview.status === "loaded" && standardReview.page?.candidates.length === 0 && <p role="status">No clauses are published for this edition.</p>}
        {standardReview.mappingStatus === "unavailable" && <p role="alert">Requirement link decisions are unavailable. <button type="button" className="text-button" onClick={() => void standardReview.load(standardReview.adoptionID)}>Retry</button></p>}
        {standardReview.mappingPage?.items.some((item) => item.status === "PENDING") && <div className="program-operation-form" aria-label="Pending standard links">
          <h4>Links awaiting review</h4>
          {standardReview.mappingPage.items.filter((item) => item.status === "PENDING").map((item) =>
            <section className="program-requirement-card" key={item.id} aria-label={`Pending link ${item.clause_code ?? item.clause_id}`}>
              <div><strong>{item.clause_code ?? item.clause_id}</strong><p>{aggregate.requirements.find((requirement) => requirement.id === item.requirement_id)?.title ?? item.requirement_id} · {label(item.mapping_kind)}</p></div>
              {reviewingMappingID === item.id ? <div className="program-operation-form">
                <label className="wide"><span>Decision rationale</span><textarea aria-label={`Decision rationale for ${item.clause_code ?? item.clause_id}`} required value={mappingReviewRationale} onChange={(event) => setMappingReviewRationale(event.target.value)}/></label>
                <div className="program-form-actions wide">
                  {mappingApproveOperation?.can_act && <button className="primary-button" type="button" disabled={mappingBusy || !mappingReviewRationale.trim()} onClick={() => void reviewMapping(item.id, "approve")}>Approve link</button>}
                  {mappingRejectOperation?.can_act && <button className="secondary-button" type="button" disabled={mappingBusy || !mappingReviewRationale.trim()} onClick={() => void reviewMapping(item.id, "reject")}>Reject link</button>}
                  <button type="button" className="text-button" disabled={mappingBusy} onClick={() => { setReviewingMappingID(null); setMappingReviewRationale(""); }}>Cancel</button>
                </div>
              </div> : (mappingApproveOperation?.can_act || mappingRejectOperation?.can_act) &&
                <button className="secondary-button" type="button" disabled={mappingBusy} onClick={() => { setReviewingMappingID(item.id); setMappingReviewRationale(""); setMappingError(""); }}>Review link</button>}
            </section>
          )}
        </div>}
        {standardReview.mappingPage?.next_after_id && <>
          <p role="status" className="program-operation-reason">More links are available. Load the remaining decisions before proposing another link.</p>
          <button type="button" className="secondary-button" disabled={standardReview.mappingStatus === "loading"} onClick={() => void standardReview.loadMoreMappings()}>More linked requirements</button>
        </>}
        {mappingError && <p role="alert" className="program-form-error">{mappingError} <button type="button" className="text-button" onClick={onReload}>Reload Program</button></p>}
        {standardReview.page && <div className="program-requirement-list">
          {standardReview.page.candidates.map((clause) => <section className="program-requirement-card" key={clause.clause_code} aria-label={`Clause ${clause.clause_code}`}>
            <div><strong>{clause.clause_code}</strong><p>Source: {clause.source_anchor}</p>
              <small>{clause.status === "UNMAPPED" ? "No matching requirement" : clause.status === "AMBIGUOUS" ? "Multiple possible matches" : "Review suggested match"}</small>
            </div>
            {clause.candidates.length > 0 && <ul>{clause.candidates.map((candidate) =>
              <li key={candidate.requirement_id}>{candidate.requirement_code} · {candidate.title}{candidate.applicability ? ` · ${label(candidate.applicability)}` : ""}
                {standardReview.mappingPage?.items.some((mapping) => mapping.clause_id === clause.clause_id && mapping.requirement_id === candidate.requirement_id && mapping.status === "APPROVED") && <small>Linked</small>}
                {standardReview.mappingPage?.items.some((mapping) => mapping.clause_id === clause.clause_id && mapping.requirement_id === candidate.requirement_id && mapping.status === "PENDING") && <small>Awaiting review</small>}
                {mappingProposalOperation?.can_act && clause.clause_id && standardReview.mappingStatus === "loaded" &&
                  standardReview.mappingPage && !standardReview.mappingPage.next_after_id &&
                  !standardReview.mappingPage.items.some((mapping) => mapping.clause_id === clause.clause_id && mapping.requirement_id === candidate.requirement_id) &&
                  <button className="text-button" type="button" onClick={() => {
                    setSelectedMapping({ clauseID: clause.clause_id, requirementID: candidate.requirement_id });
                    setMappingKind("EXACT"); setMappingRationale(""); setMappingError("");
                  }}>Propose link</button>}
              </li>
            )}</ul>}
            {selectedMapping?.clauseID === clause.clause_id && <form className="program-operation-form" onSubmit={(event) => void submitMapping(event)}>
              <label><span>Mapping</span><select value={mappingKind} onChange={(event) => setMappingKind(event.target.value as FrameworkMappingKind)}>
                <option value="EXACT">Exact</option><option value="PARTIAL">Partial</option><option value="SUPPORTING">Supporting</option>
              </select></label>
              <label className="wide"><span>Link rationale</span><textarea required value={mappingRationale} onChange={(event) => setMappingRationale(event.target.value)}/></label>
              <div className="program-form-actions wide">
                <button className="primary-button" type="submit" disabled={mappingBusy || !mappingRationale.trim()}>{mappingBusy ? "Submitting…" : "Submit link for approval"}</button>
                <button className="text-button" type="button" onClick={() => setSelectedMapping(null)}>Cancel</button>
              </div>
            </form>}
            {clause.candidates_truncated && <p className="program-operation-reason">{clause.total_candidates} candidates · showing first {clause.candidates.length}</p>}
          </section>)}
        </div>}
        {standardReview.page?.next_after_code && <button className="secondary-button" type="button" onClick={() =>
          void standardReview.load(standardReview.adoptionID, standardReview.page!.next_after_code)
        }>Next clauses</button>}
      </>}
    </section>}
    <div className="program-requirement-list">{aggregate.requirements.length ? aggregate.requirements.map((requirement) => {
      const supersedeOperation = operations.find((operation) => operation.command === "program.requirement.supersede" && operation.subresource_id === requirement.id);
      const currentApplicability = [...aggregate.applicability].reverse().find((value) => value.requirement_id === requirement.id);
      return <section className="program-requirement-card" key={requirement.id} id={`program-requirement-${encodeURIComponent(requirement.id)}`} tabIndex={-1} aria-label={`Requirement: ${requirement.title}`}><div><span>{requirement.code} · {label(requirement.status)}</span><h3>{requirement.title}</h3><p>{requirement.statement}</p>{requirement.source_anchor && <small>Source: {requirement.source_anchor}</small>}</div><dl><div><dt>Does this apply?</dt><dd>{currentApplicability ? label(currentApplicability.status) : "Not decided"}</dd></div><div><dt>Effective from</dt><dd>{requirement.effective_from?.slice(0, 10) ?? "Not recorded"}</dd></div></dl>{supersedeOperation?.can_act && <button className="text-button" type="button" onClick={() => beginReplace(requirement)}>Replace requirement</button>}</section>;
    }) : <div className="program-empty-state"><strong>No requirements are recorded</strong><p>Add a source-anchored obligation before deciding whether it applies or defining its safeguards.</p></div>}</div>
    {(mode === "add" || mode === "replace") && <form className="program-operation-form" onSubmit={(event) => void saveRequirement(event)}>
      <label><span>Requirement code</span><input required value={code} onChange={(event) => setCode(event.target.value)}/></label>
      <label><span>Requirement title</span><input required value={title} onChange={(event) => setTitle(event.target.value)}/></label>
      <label className="wide"><span>{mode === "replace" ? "Replacement statement" : "Requirement statement"}</span><textarea required value={statement} onChange={(event) => setStatement(event.target.value)}/></label>
      <label className="wide"><span>{mode === "replace" ? "Replacement source and section" : "Official source and section"}</span><input required value={sourceAnchor} onChange={(event) => setSourceAnchor(event.target.value)}/></label>
      <label><span>Obligation strength</span><select required value={modality} onChange={(event) => setModality(event.target.value)}><option value="">Select the source's wording</option><option value="MUST">Must</option><option value="MUST_NOT">Must not</option><option value="SHOULD">Should</option><option value="MAY">May</option><option value="EXPECTED">Expected</option></select></label>
      <label><span>Effective from</span><input required type="date" value={effectiveFrom} onChange={(event) => setEffectiveFrom(event.target.value)}/></label>
      <TextField label="Who must act?" value={actor} onChange={setActor} isRequired/>
      <TextField label="What must they do?" value={action} onChange={setAction} isRequired/>
      <div className="wide"><TextField label="What does it apply to?" value={object} onChange={setObject} isRequired/></div>
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
