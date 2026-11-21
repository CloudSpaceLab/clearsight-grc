import { useEffect, useRef, useState, type FormEvent } from "react";
import { loadContext, loadProgramSummaries, type RuntimeContext } from "../api";
import { loadFormTemplateItem } from "../formsApi";
import type { FormLibraryItem } from "../formsTypes";
import { createDistribution, loadDistributionPage, loadRecipientCandidates, type Distribution, type RecipientCandidate } from "../formsDistributionApi";
import { requestJSON } from "../http";
import type { ProgramSummary } from "../summaryTypes";
import "./form-assignment.css";

export function FormAssignment({ formId }: { formId: string }) {
  const [context, setContext] = useState<RuntimeContext>(); const [form, setForm] = useState<FormLibraryItem>();
  const [programs, setPrograms] = useState<ProgramSummary[]>([]); const [programID, setProgramID] = useState("");
  const [loading, setLoading] = useState(true); const [error, setError] = useState("");
  const [authority, setAuthority] = useState<"none" | "checking" | "allowed" | "denied">("none");
  const [programQuery, setProgramQuery] = useState(""); const [peopleQuery, setPeopleQuery] = useState("");
  const [candidates, setCandidates] = useState<RecipientCandidate[]>([]); const [person, setPerson] = useState<RecipientCandidate>();
  const [purpose, setPurpose] = useState(""); const [deadline, setDeadline] = useState(""); const [minutes, setMinutes] = useState("10");
  const [searching, setSearching] = useState<"programs" | "people" | "">("");
  const [busy, setBusy] = useState(false); const [result, setResult] = useState<"created" | "unknown">();
  const [existing, setExisting] = useState<Distribution[]>(); const [checkError, setCheckError] = useState(false);
  const lock = useRef(false); const createAttempted = useRef(false); const searchGeneration = useRef(0); const formGeneration = useRef(0); const renderedForm = useRef(formId); renderedForm.current = formId;
  useEffect(() => {
    let active = true; formGeneration.current++; searchGeneration.current++; lock.current = false; createAttempted.current = false;
    setLoading(true); setError(""); setContext(undefined); setForm(undefined); setPrograms([]); setProgramID(""); setAuthority("none"); setCandidates([]); setPerson(undefined); setSearching(""); setBusy(false); setResult(undefined); setExisting(undefined); setCheckError(false); setProgramQuery(""); setPeopleQuery(""); setPurpose(""); setDeadline(""); setMinutes("10");
    Promise.all([loadContext(), loadFormTemplateItem(formId), loadProgramSummaries({ limit: 20 })]).then(([who, item, page]) => {
      if (!active) return;
      if (item.template.id !== formId || !item.active_version || item.active_status !== "ACTIVE" || !who.actor.id || who.actor.kind?.toUpperCase() === "SERVICE" || !who.tenant.id || !who.legal_entity.id || who.legal_entity.id === "*") { setError("Assignment unavailable. Check the form and your role."); return; }
      setContext(who); setForm(item); setPrograms(page.items);
    }).catch(() => { if (active) setError("Assignment unavailable. Check the form and your role."); }).finally(() => { if (active) setLoading(false); });
    return () => { active = false; formGeneration.current++; searchGeneration.current++; };
  }, [formId]);
  useEffect(() => {
    if (!context || !programID) { setAuthority("none"); return; }
    let active = true; setAuthority("checking");
    const input = { tenant_id: context.tenant.id, legal_entity_id: context.legal_entity.id, object_type: "LEGAL_ENTITY", object_id: context.legal_entity.id, responsibility: "ACCOUNTABLE_OWNER", decision_type: "forms.distribution.create", materiality: 3 };
    requestJSON<{ principal: { id: string }; candidate_principals?: { id: string }[] }>(import.meta.env.VITE_API_BASE_URL ?? "", "/api/v1/authority/resolve", { method: "POST", body: JSON.stringify(input) }).then((route) => {
      if (active) setAuthority(route.principal?.id === context.actor.id || route.candidate_principals?.some((candidate) => candidate.id === context.actor.id) ? "allowed" : "denied");
    }).catch(() => { if (active) setAuthority("denied"); });
    return () => { active = false; };
  }, [context, programID]);

  async function searchPrograms(event: FormEvent) {
    event.preventDefault(); if (busy || searching) return; const token = ++searchGeneration.current;
    setSearching("programs"); setError(""); setAuthority("none"); setProgramID(""); setPrograms([]); setCandidates([]); setPerson(undefined);
    try { const page = await loadProgramSummaries({ q: programQuery.trim(), limit: 20 }); if (searchGeneration.current === token && renderedForm.current === formId) setPrograms(page.items); }
    catch { if (searchGeneration.current === token && renderedForm.current === formId) setError("Programs unavailable. Try the search again."); }
    finally { if (searchGeneration.current === token && renderedForm.current === formId) setSearching(""); }
  }
  async function searchPeople() {
    if (busy || authority !== "allowed" || searching || peopleQuery.trim().length < 2) return; const token = ++searchGeneration.current;
    setSearching("people"); setError(""); setCandidates([]); setPerson(undefined);
    try { const page = await loadRecipientCandidates(peopleQuery.trim(), 12); if (searchGeneration.current === token && renderedForm.current === formId) setCandidates(page.items); }
    catch { if (searchGeneration.current === token && renderedForm.current === formId) setError("People unavailable. Try the search again."); }
    finally { if (searchGeneration.current === token && renderedForm.current === formId) setSearching(""); }
  }
  async function assign(event: FormEvent) {
    event.preventDefault(); const when = new Date(deadline); const duration = Number(minutes);
    if (lock.current || createAttempted.current || authority !== "allowed" || !form?.active_version || !programs.some((item) => item.program.id === programID) || !person || !purpose.trim() || Number.isNaN(when.getTime()) || when.getTime() <= Date.now() || !Number.isInteger(duration) || duration < 1 || duration > 60) { setError("Choose a respondent, purpose and future deadline."); return; }
    const token = formGeneration.current; createAttempted.current = true; lock.current = true; setBusy(true); setError("");
    try {
      const result = await createDistribution({ form_template_id: formId, form_template_version: form.active_version, subject_type: "PROGRAM", subject_id: programID, title: form.template.name, purpose: purpose.trim(), access_policy: "DIRECT_LINK_EMAIL_OTP", estimated_minutes: duration, deadline: when.toISOString(), route_expires_at: when.toISOString(), recipients: [{ role: "TO", type: "INTERNAL_PRINCIPAL", principal_id: person.principal_id }] });
      if (formGeneration.current !== token || renderedForm.current !== formId) return;
      if (!result.distribution.id || result.distribution.form_template_id !== formId || result.distribution.form_template_version !== form.active_version || result.distribution.subject_type !== "PROGRAM" || result.distribution.subject_id !== programID || result.distribution.status !== "OPEN" || !result.recipients.some((item) => item.role === "TO" && item.type === "INTERNAL_PRINCIPAL" && item.principal_id === person.principal_id)) throw new Error("unconfirmed");
      setResult("created");
    } catch { if (formGeneration.current === token && renderedForm.current === formId) setResult("unknown"); }
    finally { if (formGeneration.current === token && renderedForm.current === formId) { lock.current = false; setBusy(false); } }
  }
  async function checkAssignments() {
    if (lock.current || !programID) return; const token = formGeneration.current; lock.current = true; setBusy(true); setCheckError(false);
    try { const page = await loadDistributionPage({ subject_type: "PROGRAM", subject_id: programID, limit: 20 }); if (formGeneration.current === token && renderedForm.current === formId) setExisting(page.items); }
    catch { if (formGeneration.current === token && renderedForm.current === formId) setCheckError(true); }
    finally { if (formGeneration.current === token && renderedForm.current === formId) { lock.current = false; setBusy(false); } }
  }
  if (loading || (form && form.template.id !== formId)) return <p role="status">Loading assignment…</p>;
  if (!context || !form) return <section><h1>Assignment unavailable</h1><p role="alert">{error || "Check the form and your role."}</p><a className="v-back" href="#/forms">Forms</a></section>;
  if (result === "created") return <section><h1>Form request created</h1><p>{form.template.name} · {person?.display_name}</p><a className="v-back" href="#/forms">Forms</a></section>;
  if (result === "unknown") return <section><h1>Assignment status unknown</h1><p>Check assignments before trying again.</p><button type="button" disabled={busy} onClick={() => void checkAssignments()}>{busy ? "Loading…" : "Check assignments"}</button>{checkError && <p role="alert">Assignments unavailable. Contact an administrator before retrying.</p>}{existing && <div className="v-panel v-assignment-existing"><h2>Recent assignments</h2>{existing.length ? <ul>{existing.map((item) => <li key={item.id}>{item.title} · {status(item.status)} · Due {formatDate(item.deadline)}</li>)}</ul> : <p>No assignments found in the first 20 records for this program.</p>}<p>Check earlier assignments before creating another request.</p></div>}<a className="v-back" href="#/forms">Forms</a></section>;
  return <section className="v-assignment"><a className="v-back" href={`#/forms/${encodeURIComponent(formId)}`}>← Form</a><header className="v-page-heading"><div><p className="v-eyebrow">Form request</p><h1>Assign {form.template.name}</h1><p>Version {form.active_version}</p></div></header>
    <form className="v-program-search" onSubmit={(event) => void searchPrograms(event)}><label htmlFor="v-assignment-program-search">Find a program</label><div><input id="v-assignment-program-search" disabled={busy} value={programQuery} onChange={(event) => { setProgramQuery(event.target.value); if (searching === "programs") { searchGeneration.current++; setSearching(""); } }}/><button disabled={busy || Boolean(searching)}>{searching === "programs" ? "Searching…" : "Search programs"}</button></div></form>
    <form className="v-panel v-assignment-form" onSubmit={(event) => void assign(event)}><label>Program<select required disabled={busy} value={programID} onChange={(event) => { searchGeneration.current++; setSearching(""); setCandidates([]); setPerson(undefined); setAuthority(event.target.value ? "checking" : "none"); setProgramID(event.target.value); }}><option value="">Select a program</option>{programs.map((item) => <option key={item.program.id} value={item.program.id}>{item.program.name}</option>)}</select></label>
      {authority === "none" && <p>Choose a program before searching for a respondent.</p>}{authority === "checking" && <p role="status">Checking assignment access…</p>}{authority === "denied" && <p role="alert">Assignment unavailable. Check your role or contact an administrator.</p>}
      <label>Find respondent<input disabled={busy} value={peopleQuery} minLength={2} onChange={(event) => { setPeopleQuery(event.target.value); setPerson(undefined); setCandidates([]); if (searching === "people") { searchGeneration.current++; setSearching(""); } }}/></label>{authority === "allowed" && peopleQuery.trim().length < 2 && <small>Enter at least 2 characters.</small>}<button type="button" disabled={busy || authority !== "allowed" || Boolean(searching) || peopleQuery.trim().length < 2} onClick={() => void searchPeople()}>{searching === "people" ? "Searching…" : "Search people"}</button>
      {candidates.length > 0 && <div className="v-assignment-candidates">{candidates.map((candidate) => <button key={candidate.principal_id} type="button" disabled={busy} onClick={() => { setPerson(candidate); setCandidates([]); }}>{candidate.display_name}{candidate.context_label ? ` · ${candidate.context_label}` : ""}</button>)}</div>}{person && <p>Respondent: <strong>{person.display_name}</strong></p>}
      <label>Purpose<textarea required disabled={busy} maxLength={1600} value={purpose} onChange={(event) => setPurpose(event.target.value)}/></label><label>Response deadline<input required disabled={busy} type="datetime-local" value={deadline} onChange={(event) => setDeadline(event.target.value)}/></label><label>Estimated minutes<input required disabled={busy} type="number" min="1" max="60" value={minutes} onChange={(event) => setMinutes(event.target.value)}/></label>{error && <p role="alert">{error}</p>}{authority === "allowed" && <button className="v-primary" disabled={busy}>{busy ? "Creating…" : "Create assignment"}</button>}
    </form>
  </section>;
}

function status(value: string) { const text = value.toLowerCase().replaceAll("_", " "); return text.charAt(0).toUpperCase() + text.slice(1); }
function formatDate(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? "date unavailable" : new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(date); }
