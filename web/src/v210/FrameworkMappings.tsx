import { useEffect, useRef, useState, type FormEvent } from "react";
import { ApiError } from "../http";
import { decideProgramFrameworkMapping, loadFrameworkClauseCandidates, loadProgramFrameworkMappings, proposeProgramFrameworkMapping, type FrameworkClauseCandidatePage, type FrameworkMappingKind, type ProgramFrameworkMappingPage } from "../programFrameworkCandidatesApi";
import type { ProgramOperation } from "../programOperationsApi";
import type { ProgramAggregate } from "../types";

export function FrameworkMappings({ aggregate, operations, onUpdated, disabled = false, onPendingChange }: { aggregate: ProgramAggregate; operations: ProgramOperation[]; onUpdated: (value: ProgramAggregate) => void; disabled?: boolean; onPendingChange?: (value: boolean) => void }) {
  const active = (aggregate.framework_adoptions ?? []).filter((item) => item.status === "ACTIVE");
  const [adoptionID, setAdoptionID] = useState(active[0]?.id ?? "");
  const [candidates, setCandidates] = useState<FrameworkClauseCandidatePage>(); const [mappings, setMappings] = useState<ProgramFrameworkMappingPage>();
  const [state, setState] = useState<"loading" | "ready" | "error" | "changed" | "unknown">("loading");
  const [attempt, setAttempt] = useState(0); const [busy, setBusy] = useState(false); const [pageBusy, setPageBusy] = useState(false); const [pageError, setPageError] = useState("");
  const [selected, setSelected] = useState<{ clauseID: string; requirementID: string }>(); const [reviewID, setReviewID] = useState("");
  const [kind, setKind] = useState<FrameworkMappingKind>("EXACT"); const [reason, setReason] = useState(""); const [message, setMessage] = useState("");
  const lock = useRef(false); const generation = useRef(0);
  const adoption = active.find((item) => item.id === adoptionID);
  useEffect(() => { if (!adoption && active.length) setAdoptionID(active[0]!.id); }, [aggregate.program.id, aggregate.program.version, adoptionID]);
  useEffect(() => {
    const token = ++generation.current; setSelected(undefined); setReviewID(""); setReason(""); setCandidates(undefined); setMappings(undefined); setPageError("");
    if (!adoption) { setState("error"); return; }
    setState("loading");
    Promise.all([loadFrameworkClauseCandidates(aggregate.program.id, adoption.id), loadProgramFrameworkMappings(aggregate.program.id, adoption.id)]).then(([clauses, links]) => {
      if (token !== generation.current) return;
      if (!Array.isArray(clauses.candidates) || clauses.candidates.some((item) => !Array.isArray(item.candidates)) || !Array.isArray(links.items)) { setState("error"); return; }
      if (clauses.program_id !== aggregate.program.id || clauses.adoption_id !== adoption.id || clauses.edition_id !== adoption.edition_id || links.adoption_id !== adoption.id || clauses.program_version !== aggregate.program.version || links.program_version !== aggregate.program.version) { setState("changed"); return; }
      setCandidates(clauses); setMappings(links); setState("ready");
    }).catch(() => { if (token === generation.current) setState("error"); });
    return () => { generation.current++; };
  }, [aggregate.program.id, aggregate.program.version, adoptionID, adoption?.edition_id, attempt]);
  const can = (command: string) => operations.some((item) => item.command === command && item.can_act);
  async function more(which: "clauses" | "links") {
    if (!adoption || pageBusy || disabled) return;
    const token = generation.current; setPageBusy(true); setPageError("");
    try {
      if (which === "clauses" && candidates?.next_after_code) {
        const page = await loadFrameworkClauseCandidates(aggregate.program.id, adoption.id, candidates.next_after_code);
        if (token !== generation.current) return;
        if (!Array.isArray(page.candidates) || page.candidates.some((item) => !Array.isArray(item.candidates))) { setPageError("Additional records unavailable. Try again."); return; }
        if (page.program_id !== aggregate.program.id || page.adoption_id !== adoption.id || page.edition_id !== adoption.edition_id || page.program_version !== aggregate.program.version) { setState("changed"); return; }
        setCandidates({ ...page, candidates: [...candidates.candidates, ...page.candidates] });
      } else if (which === "links" && mappings?.next_after_id) {
        const page = await loadProgramFrameworkMappings(aggregate.program.id, adoption.id, mappings.next_after_id);
        if (token !== generation.current) return;
        if (!Array.isArray(page.items)) { setPageError("Additional records unavailable. Try again."); return; }
        if (page.adoption_id !== adoption.id || page.program_version !== aggregate.program.version) { setState("changed"); return; }
        setMappings({ ...page, items: [...mappings.items, ...page.items] });
      }
    } catch { if (token === generation.current) setPageError("Additional records unavailable. Try again."); }
    finally { if (token === generation.current) setPageBusy(false); }
  }
  async function propose(event: FormEvent) {
    event.preventDefault();
    if (disabled || lock.current || state !== "ready" || !adoption || !selected || !mappings || mappings.next_after_id || !can("program.framework.mapping.propose") || !reason.trim() || mappings.items.some((item) => item.clause_id === selected.clauseID && item.requirement_id === selected.requirementID)) return;
    lock.current = true; setBusy(true); onPendingChange?.(true); setMessage("");
    try {
      const result = await proposeProgramFrameworkMapping(aggregate.program.id, adoption.id, aggregate.program.version, selected.clauseID, selected.requirementID, kind, reason.trim());
      if (result.program.program.id !== aggregate.program.id || result.program.program.version <= aggregate.program.version || result.mapping.adoption_id !== adoption.id || result.mapping.clause_id !== selected.clauseID || result.mapping.requirement_id !== selected.requirementID || result.mapping.status !== "PENDING") throw new Error("unconfirmed");
      setSelected(undefined); setMessage("Link proposed. Approval is required."); lock.current = false; onUpdated(result.program);
    } catch (error) { if (error instanceof ApiError && [400, 422].includes(error.status)) { setMessage("Link not proposed. Check the reason."); lock.current = false; } else if (error instanceof ApiError && [403, 409].includes(error.status)) setState("changed"); else setState("unknown"); }
    finally { setBusy(false); onPendingChange?.(false); }
  }
  async function decide(action: "approve" | "reject") {
    const mapping = mappings?.items.find((item) => item.id === reviewID && item.status === "PENDING");
    if (disabled || lock.current || state !== "ready" || !adoption || !mapping || !can(`program.framework.mapping.${action}`) || !reason.trim()) return;
    lock.current = true; setBusy(true); onPendingChange?.(true); setMessage("");
    try {
      const result = await decideProgramFrameworkMapping(aggregate.program.id, adoption.id, mapping.id, aggregate.program.version, action, reason.trim());
      if (result.program.program.id !== aggregate.program.id || result.program.program.version <= aggregate.program.version || result.mapping.id !== mapping.id || result.mapping.adoption_id !== adoption.id || result.mapping.status !== (action === "approve" ? "APPROVED" : "REJECTED")) throw new Error("unconfirmed");
      setReviewID(""); setMessage(action === "approve" ? "Link approved." : "Link rejected."); lock.current = false; onUpdated(result.program);
    } catch (error) { if (error instanceof ApiError && [400, 422].includes(error.status)) { setMessage("Review not saved. Check the reason."); lock.current = false; } else if (error instanceof ApiError && [403, 409].includes(error.status)) setState("changed"); else setState("unknown"); }
    finally { setBusy(false); onPendingChange?.(false); }
  }
  if (!active.length) return null;
  return <section className="v-framework-mappings"><h3>Framework requirements</h3><label>Framework<select disabled={busy || disabled} value={adoptionID} onChange={(event) => setAdoptionID(event.target.value)}>{active.map((item) => <option key={item.id} value={item.id}>{item.framework_code} · {item.edition_code}</option>)}</select></label>
    {state === "loading" && <p role="status">Loading requirements…</p>}{state === "error" && <p role="alert">Framework requirements unavailable. <button onClick={() => setAttempt((value) => value + 1)}>Try again</button></p>}{state === "changed" && <p role="alert">Program changed. Reload before linking requirements. <button type="button" onClick={() => window.location.reload()}>Reload program</button></p>}{state === "unknown" && <p role="alert">Link status unknown. Reload the program before trying again. <button type="button" onClick={() => window.location.reload()}>Reload program</button></p>}
    {message && <p role="status">{message}</p>}
    {state === "ready" && candidates && mappings && <><ul>{candidates.candidates.map((clause) => <li key={clause.clause_id}><strong>{clause.clause_code}</strong><p>{clause.source_anchor}</p>{clause.candidates.map((candidate) => <div key={candidate.requirement_id}><span>{candidate.requirement_code} · {candidate.title}</span>{can("program.framework.mapping.propose") && !mappings.next_after_id && !mappings.items.some((item) => item.clause_id === clause.clause_id && item.requirement_id === candidate.requirement_id) && aggregate.requirements.some((item) => item.id === candidate.requirement_id) && <button disabled={busy || disabled} type="button" onClick={() => { setSelected({ clauseID: clause.clause_id, requirementID: candidate.requirement_id }); setReviewID(""); setReason(""); }}>Link {clause.clause_code} to {candidate.title}</button>}</div>)}</li>)}</ul>
      {candidates.next_after_code && <button disabled={pageBusy || disabled} type="button" onClick={() => void more("clauses")}>Load more requirements</button>}{mappings.next_after_id && <><p>Load remaining links before proposing another.</p><button disabled={pageBusy || disabled} type="button" onClick={() => void more("links")}>Load more links</button></>}{pageError && <p role="alert">{pageError}</p>}
      {mappings.items.filter((item) => item.status === "PENDING").map((item) => <div key={item.id}><p>Link awaiting review · {item.clause_code ?? "Clause"} · {aggregate.requirements.find((requirement) => requirement.id === item.requirement_id)?.title ?? "Requirement unavailable"}</p>{(can("program.framework.mapping.approve") || can("program.framework.mapping.reject")) && <button disabled={busy || disabled} type="button" onClick={() => { setReviewID(item.id); setSelected(undefined); setReason(""); }}>Review link for {item.clause_code ?? "requirement"}</button>}</div>)}
      {selected && <form onSubmit={(event) => void propose(event)}><label>Link type<select value={kind} onChange={(event) => setKind(event.target.value as FrameworkMappingKind)}><option value="EXACT">Exact</option><option value="PARTIAL">Partial</option><option value="SUPPORTING">Supporting</option></select></label><label>Reason for link<textarea required value={reason} onChange={(event) => setReason(event.target.value)}/></label><button className="v-primary" disabled={busy || disabled || !reason.trim()}>Propose link</button><button type="button" disabled={busy} onClick={() => setSelected(undefined)}>Cancel</button></form>}
      {reviewID && <section><label>Review reason<textarea value={reason} onChange={(event) => setReason(event.target.value)}/></label>{can("program.framework.mapping.approve") && <button type="button" disabled={busy || disabled || !reason.trim()} onClick={() => void decide("approve")}>Approve link</button>}{can("program.framework.mapping.reject") && <button type="button" disabled={busy || disabled || !reason.trim()} onClick={() => void decide("reject")}>Reject link</button>}<button type="button" disabled={busy} onClick={() => setReviewID("")}>Cancel</button></section>}
    </>}
  </section>;
}
