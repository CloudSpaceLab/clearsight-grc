import { useEffect, useRef, useState, type FormEvent } from "react";
import { ApiError } from "../http";
import { addProgramRequirement, determineProgramApplicability, loadProgramOperations, supersedeProgramRequirement, type ProgramOperation } from "../programOperationsApi";
import type { ProgramAggregate, Requirement } from "../types";
import { FrameworkMappings } from "./FrameworkMappings";
import "./program-requirements.css";

type Mode = { kind: "add" } | { kind: "edit" | "applicability"; requirement: Requirement };
export function ProgramRequirements({ aggregate, onUpdated }: { aggregate: ProgramAggregate; onUpdated: (value: ProgramAggregate) => void }) {
  const [operations, setOperations] = useState<ProgramOperation[]>([]);
  const [authority, setAuthority] = useState<"loading" | "ready" | "changed" | "error">("loading");
  const [attempt, setAttempt] = useState(0); const [mode, setMode] = useState<Mode>();
  const [title, setTitle] = useState(""); const [statement, setStatement] = useState(""); const [answer, setAnswer] = useState("");
  const [code, setCode] = useState(""); const [sourceAnchor, setSourceAnchor] = useState(""); const [modality, setModality] = useState("");
  const [actor, setActor] = useState(""); const [action, setAction] = useState(""); const [object, setObject] = useState("");
  const [reason, setReason] = useState(""); const [scope, setScope] = useState(""); const [effective, setEffective] = useState(""); const [busy, setBusy] = useState(false);
  const [status, setStatus] = useState<"" | "updated" | "draft-saved" | "unknown" | "changed">(""); const [error, setError] = useState("");
  const [mappingPending, setMappingPending] = useState(false);
  const lock = useRef(false); const generation = useRef(0); const inputRef = useRef<HTMLInputElement | HTMLSelectElement>(null);
  const restoreFocus = useRef<{ id: string; kind: Mode["kind"] } | null>(null);
  useEffect(() => {
    let active = true; generation.current++; lock.current = false; setBusy(false); setAuthority("loading"); setOperations([]); setMode(undefined);
    loadProgramOperations(aggregate.program.id).then((value) => {
      if (!active) return;
      if (value.program_id !== aggregate.program.id || value.program_version !== aggregate.program.version) { setAuthority("changed"); return; }
      if (!value.authority_available) { setAuthority("error"); return; }
      setOperations(value.operations); setAuthority("ready");
    }).catch(() => { if (active) setAuthority("error"); });
    return () => { active = false; generation.current++; };
  }, [aggregate.program.id, aggregate.program.version, attempt]);
  useEffect(() => { if (mode) inputRef.current?.focus(); else if (restoreFocus.current) { const target = restoreFocus.current; restoreFocus.current = null; if (target.kind === "add") document.getElementById("v-add-requirement")?.focus(); else document.getElementById(`v-requirement-${encodeURIComponent(target.id)}`)?.querySelector<HTMLButtonElement>(`button[data-action="${target.kind}"]`)?.focus(); } }, [mode]);
  const applicability = Array.isArray(aggregate.applicability) ? aggregate.applicability : undefined;
  const can = (command: string, requirement?: Requirement) => authority === "ready" && operations.some((item) => item.command === command && item.can_act && (!requirement || item.subresource_id === requirement.id));
  function open(kind: "edit" | "applicability", requirement: Requirement) {
    if (lock.current || mappingPending || status === "unknown" || status === "changed") return;
    restoreFocus.current = { id: requirement.id, kind };
    setMode({ kind, requirement }); setTitle(requirement.title); setStatement(requirement.statement); setAnswer(""); setReason(""); setScope(""); setError(""); setStatus("");
    const today = new Date().toISOString().slice(0, 10);
    const prior = requirement.effective_from?.slice(0, 10) ?? "";
    setEffective(kind === "edit" && prior >= today ? new Date(Date.parse(`${prior}T00:00:00Z`) + 86400000).toISOString().slice(0, 10) : today);
  }
  function beginAdd() {
    if (lock.current || mappingPending || status === "unknown" || status === "changed" || !can("program.requirement.add")) return;
    restoreFocus.current = { id: "", kind: "add" }; setMode({ kind: "add" }); setCode(""); setTitle(""); setStatement(""); setSourceAnchor(""); setModality(""); setActor(""); setAction(""); setObject(""); setEffective(new Date().toISOString().slice(0, 10)); setError(""); setStatus("");
  }
  async function submit(event: FormEvent) {
    event.preventDefault();
    if (lock.current || mappingPending || !mode || authority !== "ready" || !effective) return;
    const requirement = mode.kind === "add" ? undefined : mode.requirement;
    if (mode.kind === "add" && (!can("program.requirement.add") || ![code, title, statement, sourceAnchor, modality, actor, action, object].every((value) => value.trim()))) return;
    if (mode.kind === "edit" && (!can("program.requirement.supersede", mode.requirement) || !title.trim() || !statement.trim() || !reason.trim() || !mode.requirement.source_anchor || (mode.requirement.effective_from && Date.parse(`${effective}T00:00:00Z`) <= Date.parse(mode.requirement.effective_from)))) return;
    if (mode.kind === "applicability" && (!applicability || !can("program.applicability.decide") || !answer || (answer !== "APPLICABLE" && !reason.trim()))) return;
    const date = new Date(`${effective}T00:00:00Z`); if (Number.isNaN(date.getTime())) return;
    lock.current = true; const request = generation.current; setBusy(true); setError(""); let confirmedResult: ProgramAggregate | undefined;
    try {
      const result = mode.kind === "add" ? await addProgramRequirement(aggregate.program.id, aggregate.program.version, { code: code.trim(), title: title.trim(), statement: statement.trim(), sourceAnchor: sourceAnchor.trim(), modality, actor: actor.trim(), action: action.trim(), object: object.trim(), effectiveFrom: date.toISOString() }) : mode.kind === "edit" ? await supersedeProgramRequirement(aggregate.program.id, requirement!.id, aggregate.program.version, {
        sourceID: requirement!.source_id, code: requirement!.code, title: title.trim(), statement: statement.trim(), sourceAnchor: requirement!.source_anchor ?? "", modality: requirement!.modality ?? "", actor: requirement!.actor ?? "", action: requirement!.action ?? "", object: requirement!.object ?? "", effectiveFrom: date.toISOString(), rationale: reason.trim(),
      }) : await determineProgramApplicability(aggregate.program.id, aggregate.program.version, { requirementID: requirement!.id, status: answer, scope: scope.trim() ? { description: scope.trim() } : {}, rationale: reason.trim(), effectiveFrom: date.toISOString() });
      const confirmed = result.program.id === aggregate.program.id && result.program.version > aggregate.program.version && (mode.kind === "add"
        ? result.requirements.some((item) => Boolean(item.id) && !aggregate.requirements.some((old) => old.id === item.id) && item.code === code.trim().toUpperCase() && item.title === title.trim() && item.statement === statement.trim() && item.source_anchor === sourceAnchor.trim() && item.modality === modality && item.actor === actor.trim() && item.action === action.trim() && item.object === object.trim() && Date.parse(item.effective_from ?? "") === date.getTime() && item.status === "DRAFT")
        : mode.kind === "edit" ? result.requirements.some((item) => item.id !== requirement!.id && item.title === title.trim() && item.statement === statement.trim() && item.status === "APPROVED")
        : result.applicability.some((item) => item.requirement_id === requirement!.id && item.status === answer && !applicability?.some((old) => old.id === item.id)));
      if (generation.current !== request) return;
      if (!confirmed) throw new Error("unconfirmed");
      setMode(undefined); setStatus(mode.kind === "add" ? "draft-saved" : "updated"); lock.current = false; confirmedResult = result;
    } catch (cause) {
      if (generation.current !== request) return;
      if (cause instanceof ApiError && [400, 422].includes(cause.status)) { setError("Requirement not saved. Check the details and try again."); lock.current = false; }
      else if (cause instanceof ApiError && [403, 409].includes(cause.status)) { setMode(undefined); setStatus("changed"); }
      else { setMode(undefined); setStatus("unknown"); }
    } finally { if (generation.current === request) setBusy(false); }
    if (confirmedResult && generation.current === request) onUpdated(confirmedResult);
  }
  return <section className="v-panel v-program-requirements"><h2>Requirements</h2>
    {!mode && status !== "unknown" && status !== "changed" && can("program.requirement.add") && <div className="v-program-requirement-actions"><button id="v-add-requirement" type="button" disabled={mappingPending} onClick={beginAdd}>Add requirement</button></div>}
    {!applicability && <p role="alert">Applicability decisions unavailable.</p>}
    {authority === "loading" && <p role="status">Checking requirement access…</p>}
    {authority === "error" && <p role="alert">Requirement actions unavailable. <button type="button" onClick={() => setAttempt((value) => value + 1)}>Try again</button></p>}
    {authority === "changed" && <p role="alert">Program changed. Reload before editing requirements. <button type="button" onClick={() => window.location.reload()}>Reload program</button></p>}
    {status === "updated" && <p role="status">Requirement updated.</p>}
    {status === "draft-saved" && <p role="status">Requirement draft saved. Review required.</p>}
    {status === "unknown" && <p role="alert">Requirement status unknown. Reload the program before trying again. <button type="button" onClick={() => window.location.reload()}>Reload program</button></p>}
    {status === "changed" && <p role="alert">Program changed. Reload before editing requirements. <button type="button" onClick={() => window.location.reload()}>Reload program</button></p>}
    {aggregate.requirements.length === 0 ? <p>No requirements recorded.</p> : <ul className="v-program-requirement-list">{aggregate.requirements.map((requirement) => <li id={`v-requirement-${encodeURIComponent(requirement.id)}`} key={requirement.id}><strong>{requirement.code} · {requirement.title}</strong><p>{requirement.statement}</p>{requirement.status === "DRAFT" && <p>Draft · Review required</p>}{(applicability ?? []).filter((item) => item.requirement_id === requirement.id).length > 0 && <details><summary>Applicability decisions</summary><ul>{(applicability ?? []).filter((item) => item.requirement_id === requirement.id).map((item) => <li key={item.id}><strong>{applicabilityLabel(item.status)}</strong>{item.effective_from && <span> · Effective {item.effective_from.slice(0, 10)}</span>}<p>{typeof item.scope?.description === "string" && item.scope.description.trim() ? item.scope.description : "Scope not recorded"}</p>{item.rationale && <p>{item.rationale}</p>}</li>)}</ul></details>}{requirement.status === "APPROVED" && !requirement.effective_until && !mode && status !== "unknown" && status !== "changed" && <div className="v-program-requirement-actions">{can("program.requirement.supersede", requirement) && requirement.source_anchor && <button type="button" disabled={mappingPending} data-action="edit" onClick={() => open("edit", requirement)}>Edit {requirement.title}</button>}{applicability && can("program.applicability.decide") && <button type="button" disabled={mappingPending} data-action="applicability" onClick={() => open("applicability", requirement)}>Set applicability for {requirement.title}</button>}</div>}</li>)}</ul>}
    {mode && <form onSubmit={(event) => void submit(event)} className="v-program-requirement-form"><h3>{mode.kind === "add" ? "Add requirement" : mode.kind === "edit" ? "Edit requirement" : "Applicability"}</h3>{mode.kind !== "add" ? <p>{mode.requirement.title}</p> : <p>All fields required.</p>}<fieldset disabled={busy}>
      {mode.kind === "add" ? <><label>Requirement code<input required value={code} onChange={(event) => setCode(event.target.value)}/></label><label>Requirement title<input ref={inputRef as React.RefObject<HTMLInputElement>} required value={title} onChange={(event) => setTitle(event.target.value)}/></label><label>Requirement text<textarea required value={statement} onChange={(event) => setStatement(event.target.value)}/></label><label>Source and section<input required value={sourceAnchor} onChange={(event) => setSourceAnchor(event.target.value)}/></label><label>Obligation strength<select required value={modality} onChange={(event) => setModality(event.target.value)}><option value="">Select an answer</option><option value="MUST">Must</option><option value="MUST_NOT">Must not</option><option value="SHOULD">Should</option><option value="MAY">May</option><option value="EXPECTED">Expected</option></select></label><label>Who must act?<input required value={actor} onChange={(event) => setActor(event.target.value)}/></label><label>What must they do?<input required value={action} onChange={(event) => setAction(event.target.value)}/></label><label>What does it apply to?<input required value={object} onChange={(event) => setObject(event.target.value)}/></label></> : mode.kind === "edit" ? <><label>Requirement title<input ref={inputRef as React.RefObject<HTMLInputElement>} required value={title} onChange={(event) => setTitle(event.target.value)}/></label><label>Requirement text<textarea required value={statement} onChange={(event) => setStatement(event.target.value)}/></label><label>Reason for change<textarea required value={reason} onChange={(event) => setReason(event.target.value)}/></label></> : <div className="v-program-applicability"><label>Does this apply?<select ref={inputRef as React.RefObject<HTMLSelectElement>} required value={answer} onChange={(event) => { setAnswer(event.target.value); setReason(""); }}><option value="">Select an answer</option><option value="APPLICABLE">Yes</option><option value="PARTIALLY_APPLICABLE">Partly</option><option value="NOT_APPLICABLE">Not applicable</option><option value="POTENTIALLY_APPLICABLE">Needs assessment</option><option value="APPLIES_LATER">Applies later</option></select></label>{answer && answer !== "APPLICABLE" && <label>{answer === "NOT_APPLICABLE" ? "Why is this not applicable?" : "Reason for decision"}<textarea required value={reason} onChange={(event) => setReason(event.target.value)}/></label>}</div>}
      {mode.kind === "applicability" && <label>Scope<textarea value={scope} onChange={(event) => setScope(event.target.value)}/></label>}<label>Effective date<input required type="date" value={effective} onChange={(event) => setEffective(event.target.value)}/></label>{error && <p role="alert">{error}</p>}
      <div className="v-program-requirement-actions"><button className="v-primary" disabled={busy || !effective || (mode.kind === "add" ? ![code, title, statement, sourceAnchor, modality, actor, action, object].every((value) => value.trim()) : mode.kind === "edit" ? !title.trim() || !statement.trim() || !reason.trim() : !answer || (answer !== "APPLICABLE" && !reason.trim()))}>{busy ? "Saving…" : mode.kind === "applicability" ? "Save applicability" : "Save requirement"}</button><button type="button" onClick={() => setMode(undefined)}>Cancel</button></div>
    </fieldset></form>}
    <FrameworkMappings aggregate={aggregate} operations={operations} onUpdated={onUpdated} disabled={authority !== "ready" || Boolean(mode) || busy || status === "unknown" || status === "changed"} onPendingChange={setMappingPending}/>
  </section>;
}

function applicabilityLabel(status: string) { return ({ APPLICABLE: "Applicable", NOT_APPLICABLE: "Not applicable", PARTIALLY_APPLICABLE: "Partly applicable", POTENTIALLY_APPLICABLE: "Needs assessment", APPLIES_LATER: "Applies later" } as Record<string, string>)[status] ?? "Decision unavailable"; }
