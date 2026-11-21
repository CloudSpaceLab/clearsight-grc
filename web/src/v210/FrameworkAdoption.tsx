import { useEffect, useRef, useState, type FormEvent } from "react";
import { loadProgram, loadProgramSummaries } from "../api";
import { ApiError } from "../http";
import { decideProgramFrameworkAdoption, loadProgramOperations, proposeProgramFrameworkAdoption, type FrameworkEditionOption } from "../programOperationsApi";
import type { ProgramSummary } from "../summaryTypes";
import type { ProgramAggregate } from "../types";

export function FrameworkAdoption({ edition, onCancel, onPendingChange }: { edition: FrameworkEditionOption; onCancel: () => void; onPendingChange?: (value: boolean) => void }) {
  const [query, setQuery] = useState(""); const [search, setSearch] = useState("");
  const [programs, setPrograms] = useState<ProgramSummary[]>([]); const [cursor, setCursor] = useState<string>();
  const [listState, setListState] = useState<"loading" | "ready" | "error">("loading");
  const [moreBusy, setMoreBusy] = useState(false); const [moreError, setMoreError] = useState(false);
  const [programID, setProgramID] = useState(""); const [program, setProgram] = useState<ProgramAggregate>();
  const [state, setState] = useState<"idle" | "loading" | "ready" | "denied" | "changed" | "error" | "unknown" | "saved">("idle");
  const [reason, setReason] = useState(""); const [busy, setBusy] = useState(false); const [error, setError] = useState("");
  const [allowed, setAllowed] = useState({ propose: false, approve: false, reject: false });
  const [savedMessage, setSavedMessage] = useState("");
  const generation = useRef(0); const lock = useRef(false); const listGeneration = useRef(0);
  useEffect(() => {
    const token = ++listGeneration.current; setListState("loading"); setPrograms([]); setCursor(undefined); setMoreError(false);
    loadProgramSummaries({ q: search, limit: 20 }).then((page) => { if (token === listGeneration.current) { setPrograms(page.items); setCursor(page.next_cursor); setListState("ready"); } }).catch(() => { if (token === listGeneration.current) setListState("error"); });
    return () => { listGeneration.current++; };
  }, [search]);
  useEffect(() => {
    const token = ++generation.current; setProgram(undefined); setReason(""); setError("");
    if (!programID) { setState("idle"); return; }
    setState("loading");
    Promise.all([loadProgram(programID), loadProgramOperations(programID)]).then(([value, operations]) => {
      if (token !== generation.current) return;
      if (value.program.id !== programID || operations.program_id !== programID || value.program.version !== operations.program_version) { setState("changed"); return; }
      const can = (command: string) => operations.authority_available && operations.operations.some((item) => item.command === command && item.can_act);
      const permissions = { propose: can("program.framework.propose"), approve: can("program.framework.approve"), reject: can("program.framework.reject") };
      if (!permissions.propose && !permissions.approve && !permissions.reject) { setState("denied"); return; }
      setAllowed(permissions);
      setProgram(value); setState("ready");
    }).catch(() => { if (token === generation.current) setState("error"); });
    return () => { generation.current++; };
  }, [programID]);
  async function more() {
    if (!cursor || moreBusy) return; const token = listGeneration.current; setMoreBusy(true); setMoreError(false);
    try { const page = await loadProgramSummaries({ q: search, limit: 20, cursor }); if (token === listGeneration.current) { setPrograms((items) => [...items, ...page.items]); setCursor(page.next_cursor); } }
    catch { if (token === listGeneration.current) setMoreError(true); } finally { if (token === listGeneration.current) setMoreBusy(false); }
  }
  const linked = program?.framework_adoptions?.some((item) => item.edition_id === edition.id && (item.status === "PENDING" || item.status === "ACTIVE"));
  const pending = program?.framework_adoptions?.find((item) => item.edition_id === edition.id && item.status === "PENDING");
  const matchingEdition = pending && pending.publisher_code === edition.publisher_code && pending.framework_code === edition.framework_code && pending.edition_code === edition.edition_code;
  async function submit(event: FormEvent) {
    event.preventDefault();
    if (lock.current || state !== "ready" || !allowed.propose || !program || linked || !reason.trim() || new TextEncoder().encode(reason.trim()).length > 2000) return;
    lock.current = true; setBusy(true); onPendingChange?.(true); setError("");
    try {
      const result = await proposeProgramFrameworkAdoption(program.program.id, program.program.version, edition.id, reason.trim());
      if (result.program.id !== program.program.id || result.program.version <= program.program.version || !result.framework_adoptions?.some((item) => item.edition_id === edition.id && item.status === "PENDING")) throw new Error("unconfirmed");
      setSavedMessage("Adoption proposed. Approval is required."); setState("saved");
    } catch (cause) {
      if (cause instanceof ApiError && [400, 422].includes(cause.status)) { setError("Adoption not proposed. Check the reason and try again."); lock.current = false; }
      else if (cause instanceof ApiError && [403, 409].includes(cause.status)) setState("changed");
      else setState("unknown");
    } finally { setBusy(false); onPendingChange?.(false); }
  }
  async function decide(action: "approve" | "reject") {
    if (lock.current || state !== "ready" || !program || !pending || !matchingEdition || !allowed[action] || !reason.trim() || new TextEncoder().encode(reason.trim()).length > 2000) return;
    lock.current = true; setBusy(true); onPendingChange?.(true); setError("");
    try {
      const result = await decideProgramFrameworkAdoption(program.program.id, pending.id, program.program.version, action, reason.trim());
      const expected = action === "approve" ? "ACTIVE" : "REJECTED";
      if (result.program.id !== program.program.id || result.program.version <= program.program.version || !result.framework_adoptions?.some((item) => item.id === pending.id && item.edition_id === edition.id && item.status === expected)) throw new Error("unconfirmed");
      setSavedMessage(action === "approve" ? "Adoption approved." : "Adoption rejected."); setState("saved");
    } catch (cause) {
      if (cause instanceof ApiError && [400, 422].includes(cause.status)) { setError("Review not saved. Check the reason and try again."); lock.current = false; }
      else if (cause instanceof ApiError && [403, 409].includes(cause.status)) setState("changed");
      else setState("unknown");
    } finally { setBusy(false); onPendingChange?.(false); }
  }
  return <section className="v-panel v-framework-adoption"><h2>Framework adoption</h2><p>{edition.title} · {edition.edition_code}</p>
    {state === "saved" ? <><p role="status">{savedMessage}</p><a href={`#/programs/${encodeURIComponent(programID)}`}>Open program</a></> : state === "unknown" ? <><p role="alert">Adoption status unknown. Check the program before trying again.</p><a href={`#/programs/${encodeURIComponent(programID)}`} target="_blank" rel="noopener noreferrer">Open program in new tab</a></> : <>
      <fieldset disabled={busy} className="v-framework-adoption-fields"><form className="v-framework-program-search" onSubmit={(event) => { event.preventDefault(); setProgramID(""); setSearch(query.trim()); }}><label>Search programs<input value={query} onChange={(event) => setQuery(event.target.value)}/></label><button type="submit">Search</button></form>
      {listState === "loading" && <p role="status">Loading programs…</p>}{listState === "error" && <p role="alert">Programs unavailable.</p>}
      {listState === "ready" && <><label>Program<select value={programID} onChange={(event) => setProgramID(event.target.value)}><option value="">Select a program</option>{programs.map(({ program: item }) => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label>{!programs.length && <p>No programs match this search.</p>}{moreError && <p role="alert">More programs unavailable. Try again.</p>}{cursor && <button type="button" disabled={moreBusy} onClick={() => void more()}>{moreBusy ? "Loading…" : "Load more programs"}</button>}</>}
      {state === "loading" && <p role="status">Checking program access…</p>}{state === "denied" && <p>Adoption unavailable. Check your role.</p>}{state === "error" && <p role="alert">Program unavailable. Select it again.</p>}{state === "changed" && <p role="alert">Program changed. Reload before proposing adoption.</p>}
      {state === "ready" && (pending && (allowed.approve || allowed.reject) ? <section><h3>Review adoption</h3><p>Proposed {pending.proposed_at.slice(0, 10)}</p><p>{pending.rationale}</p>{!matchingEdition ? <p role="alert">Edition details do not match this proposal.</p> : <><label>Review reason<textarea required maxLength={2000} value={reason} onChange={(event) => setReason(event.target.value)}/></label>{error && <p role="alert">{error}</p>}{allowed.approve && <button type="button" className="v-primary" disabled={busy || !reason.trim() || new TextEncoder().encode(reason.trim()).length > 2000} onClick={() => void decide("approve")}>Approve adoption</button>}{allowed.reject && <button type="button" disabled={busy || !reason.trim() || new TextEncoder().encode(reason.trim()).length > 2000} onClick={() => void decide("reject")}>Reject adoption</button>}</>}</section> : linked ? <p>This edition is already active or awaiting approval.</p> : allowed.propose ? <form onSubmit={(event) => void submit(event)}><label>Reason for adoption<textarea required maxLength={2000} value={reason} onChange={(event) => setReason(event.target.value)}/></label>{error && <p role="alert">{error}</p>}<button className="v-primary" disabled={busy || !reason.trim() || new TextEncoder().encode(reason.trim()).length > 2000}>{busy ? "Proposing…" : "Propose adoption"}</button></form> : <p>Adoption unavailable. Check your role.</p>)}</fieldset>
    </>}
    <button type="button" disabled={busy} onClick={onCancel}>Close</button>
  </section>;
}
