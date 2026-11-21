import { useEffect, useRef, useState, type FormEvent } from "react";
import { loadFormTemplatePage } from "../formsApi";
import type { ReusableFormTemplateRef } from "../formsTypes";
import { ApiError } from "../http";
import { completeVendorAssessment, loadCurrentVendorAssessment, loadVendorAssessment, reissueVendorAssessmentRequest, requestVendorAssessmentClarification, retryVendorAssessmentSetup, sendVendorAssessmentRequest, startVendorAssessment, startVendorAssessmentReview } from "../vendorAssessmentApi";
import type { VendorAssessment, VendorAssessmentConclusion, VendorAssessmentReviewAnswer, VendorAssessmentReviewView, VendorAssessmentSendState, VendorAssessmentSetupStatus } from "../vendorAssessmentTypes";
import type { VendorRelationshipAggregate } from "../vendorTypes";
import { checkVendorAuthority, type VendorAction } from "./vendorAuthority";
import { VendorDocuments } from "./VendorDocuments";
import { VendorActivation } from "./VendorActivation";
import { VendorFindings } from "./VendorFindings";
import { VendorCollection } from "./VendorCollection";
import "./vendor-review.css";

type LoadState = "loading" | "ready" | "error";

export function VendorReview({ relationship, assessmentId, onBack, onActivated, backLabel = "Vendors" }: { relationship: VendorRelationshipAggregate; assessmentId?: string; onBack: () => void; onActivated?: (value: VendorRelationshipAggregate["relationship"]) => void; backLabel?: string }) {
  const [assessment, setAssessment] = useState<VendorAssessment | null>(null);
  const [setup, setSetup] = useState<VendorAssessmentSetupStatus>();
  const [review, setReview] = useState<VendorAssessmentReviewView>();
  const [state, setState] = useState<LoadState>("loading");
  const [attempt, setAttempt] = useState(0);
  const [mode, setMode] = useState<"view" | "start" | "complete" | "send" | "sent" | "reissue" | "reissued" | "clarification" | "clarified">("view");
  const [sendState, setSendState] = useState<VendorAssessmentSendState>();
  const [setupQueued, setSetupQueued] = useState(false);
  const setupLock = useRef(false);
  const [uncertain, setUncertain] = useState(false);
  const [busy, setBusy] = useState(false);
  const [collectionPending, setCollectionPending] = useState(false);
  const [commandError, setCommandError] = useState("");
  const [stale, setStale] = useState(false);
  const [historical, setHistorical] = useState(false);
  const [authority, setAuthority] = useState<"checking" | "allowed" | "unavailable">("checking");
  const [clarificationAllowed, setClarificationAllowed] = useState(false);
  useEffect(() => {
    let active = true; setState("loading"); setReview(undefined); setAssessment(null); setSetup(undefined); setHistorical(false);
    loadCurrentVendorAssessment(relationship.relationship.id).catch((error: unknown) => {
      if (error instanceof ApiError && error.status === 404 && error.code === "vendor_assessment_not_found") return { assessment: null, setup: undefined };
      throw error;
    }).then(async (current) => {
      const selectedID = assessmentId || current.assessment?.id;
      const details = selectedID && (assessmentId || current.assessment && ["SUBMITTED", "UNDER_REVIEW", "COMPLETED"].includes(current.assessment.status)) ? await loadVendorAssessment(selectedID) : undefined;
      if (details && details.assessment.relationship_id !== relationship.relationship.id) throw new Error("Assessment relationship mismatch");
      if (active) { setAssessment(details?.assessment ?? current.assessment); setSetup(current.setup); setReview(details); setHistorical(Boolean(assessmentId && assessmentId !== current.assessment?.id)); setState("ready"); }
    }).catch((error) => {
      if (!active) return;
      void error; setState("error");
    });
    return () => { active = false; };
  }, [relationship.relationship.id, assessmentId, attempt]);

  const action: VendorAction | undefined = state !== "ready" || historical ? undefined : !assessment && relationship.relationship.status === "PROPOSED" ? "start" : assessment?.status === "SETUP_PENDING" && setup?.state === "FAILED" && setup.assessment_id === assessment.id ? "setup" : assessment?.status === "READY_TO_SEND" ? "send" : assessment?.status === "COLLECTING" && assessment.current_request_id ? "reissue" : assessment?.status === "SUBMITTED" ? "review" : assessment?.status === "UNDER_REVIEW" ? "complete" : undefined;
  const actionObjectID = action === "start" ? relationship.relationship.id : assessment?.id;
  useEffect(() => {
    let active = true; setAuthority("checking");
    if (action && actionObjectID) void checkVendorAuthority(action, actionObjectID).then((allowed) => { if (active) setAuthority(allowed ? "allowed" : "unavailable"); }).catch(() => { if (active) setAuthority("unavailable"); });
    return () => { active = false; };
  }, [action, actionObjectID, attempt]);
  useEffect(() => {
    let active = true; setClarificationAllowed(false);
    if (state === "ready" && !historical && assessment?.status === "UNDER_REVIEW" && assessment.current_request_id && assessment.submission_id && review?.answers.some((answer) => answer.visibility === "VISIBLE")) {
      void checkVendorAuthority("clarification", assessment.id).then((allowed) => { if (active) setClarificationAllowed(allowed); }).catch(() => { if (active) setClarificationAllowed(false); });
    }
    return () => { active = false; };
  }, [state, historical, assessment?.id, assessment?.version, assessment?.status, assessment?.current_request_id, assessment?.submission_id, review, attempt]);

  function reload() { setMode("view"); setUncertain(false); setStale(false); setCommandError(""); setSetupQueued(false); setAttempt((value) => value + 1); }
  function commandFailed(error: unknown) {
    if (error instanceof ApiError && error.status >= 400 && error.status < 500) { setCommandError(error.message); if (error.status === 409 && error.code !== "vendor_assessment_review_incomplete") setStale(true); }
    else setUncertain(true);
  }
  async function beginReview() {
    if (!assessment || busy || authority !== "allowed") return;
    setBusy(true);
    try { await startVendorAssessmentReview(assessment.id, { expected_version: assessment.version }); reload(); }
    catch (error) { commandFailed(error); } finally { setBusy(false); }
  }
  async function retrySetup() {
    if (!assessment || setupLock.current || authority !== "allowed" || action !== "setup") return;
    setupLock.current = true; setBusy(true);
    try {
      const result = await retryVendorAssessmentSetup(assessment.id, { expected_version: assessment.version });
      if (result.assessment.id !== assessment.id || result.assessment.relationship_id !== relationship.relationship.id || result.assessment.version <= assessment.version || result.assessment.status !== "SETUP_PENDING" || result.setup.assessment_id !== assessment.id || result.setup.state !== "READY") throw new Error("unconfirmed");
      setAssessment(result.assessment); setSetup(result.setup); setSetupQueued(true);
    } catch { setUncertain(true); } finally { setupLock.current = false; setBusy(false); }
  }

  return <section className="v-vendor-review"><fieldset className="v-vendor-review-content" disabled={collectionPending} onClickCapture={(event) => { if (collectionPending) { event.preventDefault(); event.stopPropagation(); } }} onSubmitCapture={(event) => { if (collectionPending) { event.preventDefault(); event.stopPropagation(); } }}><button className="v-back v-back-button" onClick={onBack}>← {backLabel}</button><header className="v-page-heading"><div><p className="v-eyebrow">Vendor relationship</p><h1>{relationship.vendor.legal_name}</h1><p>{relationship.relationship.service_name}</p></div><span className="v-status">{label(relationship.relationship.status)}</span></header>
    <nav className="v-local-nav" aria-label="Vendor sections"><span aria-current="page">Due diligence</span></nav>
    {state === "loading" && <p role="status">Loading assessment…</p>}
    {state === "error" && <div className="v-panel"><p role="alert">Assessment unavailable.</p><button onClick={reload}>Try again</button></div>}
    {uncertain && <div className="v-panel"><p role="alert">Review status could not be confirmed. Reload the assessment before trying again.</p><button onClick={reload}>Reload assessment</button></div>}
    {commandError && !uncertain && <p role="alert">{commandError}</p>}
    {stale && <button onClick={reload}>Reload assessment</button>}
    {state === "ready" && !uncertain && !stale && mode === "start" && <StartAssessment relationship={relationship} onCancel={() => { setCommandError(""); setMode("view"); }} onDone={reload} onFailed={commandFailed}/>}
    {state === "ready" && !uncertain && !stale && mode === "complete" && assessment && <CompleteAssessment assessment={assessment} onCancel={() => { setCommandError(""); setMode("view"); }} onDone={reload} onFailed={commandFailed}/>}
    {state === "ready" && !uncertain && !stale && mode === "send" && assessment && <SendAssessment assessment={assessment} onCancel={() => { setCommandError(""); setMode("view"); }} onDone={(result) => { setCommandError(""); setSendState(result); setMode("sent"); }} onFailed={commandFailed}/>}
    {state === "ready" && !uncertain && !stale && mode === "reissue" && assessment && <ReissueAssessment assessment={assessment} onCancel={() => setMode("view")} onDone={(result) => { setSendState(result); setMode("reissued"); }} onUncertain={() => setUncertain(true)}/>}
    {state === "ready" && !uncertain && !stale && mode === "clarification" && assessment && review && <Clarification assessment={assessment} answers={review.answers} onCancel={() => setMode("view")} onDone={(result) => { setSendState(result); setMode("clarified"); }} onUncertain={() => setUncertain(true)}/>}
    {state === "ready" && !uncertain && mode === "clarified" && <div className="v-panel"><h2>Clarification request</h2><p>{sendState === "DELIVERED" ? "Clarification sent." : sendState === "LINK_CREATED_EMAIL_NOT_SENT" ? "Clarification created; email not sent." : "Clarification access not issued."}</p><button onClick={reload}>View assessment</button></div>}
    {state === "ready" && !uncertain && mode === "reissued" && <div className="v-panel"><h2>Replacement invitation</h2><p>{sendState === "DELIVERED" ? "Replacement sent." : sendState === "LINK_CREATED_EMAIL_NOT_SENT" ? "Replacement created; email not sent." : "Replacement access not issued."}</p><button onClick={reload}>View assessment</button></div>}
    {state === "ready" && !uncertain && mode === "sent" && <div className="v-panel"><h2>Assessment request</h2><p>{sendState === "DELIVERED" ? "Request sent." : sendState === "LINK_CREATED_EMAIL_NOT_SENT" ? "Request created; email not sent." : "Request created; access not issued."}</p><button onClick={reload}>View assessment</button></div>}
    {state === "ready" && !uncertain && !stale && mode === "view" && <>
      {!assessment ? <div className="v-panel"><h2>Due diligence</h2><p>No assessment recorded for this relationship.</p>{action === "start" && authority === "allowed" && <button className="v-primary" onClick={() => setMode("start")}>Start onboarding assessment</button>}</div> : <>
        <div className="v-panel v-vendor-summary"><div><h2>Due diligence</h2>{historical && <p>Earlier assessment</p>}<p>{label(assessment.review_kind)} · {label(assessment.status)}</p>{assessment.review_due_at && <p>Review due {date(assessment.review_due_at)}</p>}{assessment.conclusion && <p>Conclusion: {label(assessment.conclusion)}</p>}</div>
          {authority === "allowed" && action === "review" && <button className="v-primary" disabled={busy} onClick={() => void beginReview()}>{busy ? "Starting…" : "Start review"}</button>}
          {authority === "allowed" && action === "complete" && <button className="v-primary" onClick={() => setMode("complete")}>Complete review</button>}
          {authority === "allowed" && action === "send" && <button className="v-primary" onClick={() => setMode("send")}>Send assessment request</button>}
          {authority === "allowed" && action === "setup" && <button className="v-primary" disabled={busy} onClick={() => void retrySetup()}>{busy ? "Retrying…" : "Retry assessment setup"}</button>}
          {authority === "allowed" && action === "reissue" && <button className="v-primary" onClick={() => setMode("reissue")}>Replace invitation</button>}
        </div>
        {assessment.status === "SETUP_PENDING" && setup?.state === "FAILED" && setup.assessment_id === assessment.id && <p role="alert">Assessment setup failed.</p>}
        {setupQueued && <p role="status">Assessment setup queued.</p>}
        {clarificationAllowed && <p><button type="button" onClick={() => setMode("clarification")}>Request clarification</button></p>}
        {review && <div className="v-panel v-vendor-answers"><h2>Submitted answers</h2>{review.response && <p>Submitted {date(review.response.submitted_at)}</p>}{review.answers.filter((answer) => answer.visibility === "VISIBLE").length === 0 ? <p>No visible answers recorded.</p> : <ul>{review.answers.filter((answer) => answer.visibility === "VISIBLE").map((answer) => <li key={answer.field_id}><strong>{answer.label}</strong>{answer.value ? <p>{answer.value.text || answer.value.values?.join(", ") || (answer.value.document || answer.value.artifact_ids?.length ? "Document submitted" : "No answer recorded")}</p> : !answer.collection_resolution && <p>No answer recorded</p>}{answer.collection_resolution && <><p>Bank document: {answer.collection_resolution.source.file_name}</p><p>Linked by {answer.collection_resolution.reconciled_by} · {date(answer.collection_resolution.reconciled_at)}</p><p>Link receipt: {answer.collection_resolution.id} · Version {answer.collection_resolution.version}</p><p>Reason: {answer.collection_resolution.rationale}</p></>}</li>)}</ul>}<VendorDocuments assessment={assessment} documents={review.documents} historical={historical} onChanged={(view) => { setReview(view); setAssessment(view.assessment); }} onFailed={commandFailed}/></div>}
        {review && <VendorFindings matters={review.matters} mayCreate={!historical && assessment.status === "UNDER_REVIEW"}/>}
      </>}
      {action && authority === "checking" && <p role="status">Checking assessment access…</p>}
      {action && authority === "unavailable" && <p role="alert">Assessment action unavailable. Check your role or reload the assessment. <button onClick={reload}>Reload assessment</button></p>}
    </>}
    {!assessmentId && <VendorActivation key={`${attempt}:${assessment?.version ?? "none"}`} relationshipId={relationship.relationship.id} relationshipScope={relationship.relationship} onActivated={(value) => onActivated?.(value)} onRefreshed={(value) => onActivated?.(value)}/>}
    </fieldset>
    {state === "ready" && !uncertain && !stale && mode === "view" && !historical && assessment?.current_request_id && ["COLLECTING", "SUBMITTED", "UNDER_REVIEW"].includes(assessment.status) && <VendorCollection key={`${assessment.id}:${assessment.version}`} assessmentId={assessment.id} relationshipId={relationship.relationship.id} onChanged={reload} onUncertain={() => setUncertain(true)} onPendingChange={setCollectionPending}/>}
  </section>;
}

function Clarification({ assessment, answers, onCancel, onDone, onUncertain }: { assessment: VendorAssessment; answers: VendorAssessmentReviewAnswer[]; onCancel: () => void; onDone: (value: VendorAssessmentSendState) => void; onUncertain: () => void }) {
  const fields = answers.filter((answer) => answer.visibility === "VISIBLE");
  const [chosen, setChosen] = useState<string[]>([]); const [message, setMessage] = useState(""); const [audience, setAudience] = useState(""); const [deadline, setDeadline] = useState(""); const [ttl, setTtl] = useState("10080"); const [error, setError] = useState(""); const [busy, setBusy] = useState(false); const lock = useRef(false);
  async function submit(event: FormEvent) {
    event.preventDefault(); const end = new Date(deadline); const reviewDue = new Date(assessment.review_due_at);
    if (lock.current) return;
    if (!chosen.length) { setError("Select at least one question."); return; }
    if (!audience.trim() || !message.trim()) { setError("Enter the recipient and request details."); return; }
    if (new TextEncoder().encode(message.trim()).length > 2000) { setError("Shorten the request details."); return; }
    if (Number.isNaN(end.getTime()) || Number.isNaN(reviewDue.getTime()) || end.getTime() <= Date.now() || end > reviewDue) { setError("Choose a future deadline no later than the review due date."); return; }
    lock.current = true; setBusy(true); setError("");
    try {
      const result = await requestVendorAssessmentClarification(assessment.id, { expected_version: assessment.version, request_fields: chosen, audience: audience.trim(), message: message.trim(), deadline: end.toISOString(), invitation_ttl_minutes: Number(ttl) });
      if (result.assessment?.id !== assessment.id || result.assessment.relationship_id !== assessment.relationship_id || result.assessment.version <= assessment.version || !["DELIVERED", "LINK_CREATED_EMAIL_NOT_SENT", "REQUEST_READY_INVITATION_NOT_ISSUED"].includes(result.state)) throw new Error("unconfirmed");
      onDone(result.state);
    } catch { onUncertain(); } finally { lock.current = false; setBusy(false); }
  }
  return <form className="v-panel v-intake v-vendor-clarification" onSubmit={(event) => void submit(event)}><h2>Request clarification</h2><fieldset><legend>Questions</legend>{fields.map((field) => <label key={field.field_id}><input type="checkbox" checked={chosen.includes(field.field_id)} onChange={(event) => setChosen((previous) => event.target.checked ? [...previous, field.field_id] : previous.filter((id) => id !== field.field_id))}/>{field.label}</label>)}</fieldset><label>Recipient email<input type="email" required value={audience} onChange={(event) => setAudience(event.target.value)}/></label><label>Request details<textarea required value={message} onChange={(event) => setMessage(event.target.value)}/></label><label>Response deadline<input type="datetime-local" required value={deadline} onChange={(event) => setDeadline(event.target.value)}/></label><label>Access expires<select value={ttl} onChange={(event) => setTtl(event.target.value)}><option value="1440">After 1 day</option><option value="10080">After 7 days</option><option value="43200">After 30 days</option></select></label><p>Access also expires at the response deadline.</p>{error && <p role="alert">{error}</p>}<div className="v-intake-actions"><button type="button" onClick={onCancel} disabled={busy}>Cancel</button><button className="v-primary" disabled={busy}>{busy ? "Sending…" : "Send clarification"}</button></div></form>;
}

function ReissueAssessment({ assessment, onCancel, onDone, onUncertain }: { assessment: VendorAssessment; onCancel: () => void; onDone: (value: VendorAssessmentSendState) => void; onUncertain: () => void }) {
  const [audience, setAudience] = useState(""); const [ttl, setTtl] = useState("10080"); const [busy, setBusy] = useState(false); const lock = useRef(false);
  async function submit(event: FormEvent) {
    event.preventDefault(); if (!audience.trim() || lock.current || !assessment.current_request_id) return;
    lock.current = true; setBusy(true);
    try {
      const result = await reissueVendorAssessmentRequest(assessment.id, { expected_version: assessment.version, audience: audience.trim(), invitation_ttl_minutes: Number(ttl) });
      if (result.assessment?.id !== assessment.id || result.assessment.relationship_id !== assessment.relationship_id || result.assessment.version <= assessment.version || result.request?.id !== assessment.current_request_id || !["DELIVERED", "LINK_CREATED_EMAIL_NOT_SENT", "REQUEST_READY_INVITATION_NOT_ISSUED"].includes(result.state)) throw new Error("unconfirmed");
      onDone(result.state);
    } catch { onUncertain(); } finally { lock.current = false; setBusy(false); }
  }
  return <form className="v-panel v-intake" onSubmit={(event) => void submit(event)}><h2>Replace invitation</h2><p>The previous link may stop working.</p><label>Recipient email<input required type="email" value={audience} onChange={(event) => setAudience(event.target.value)}/></label><p>Use the original recipient email.</p><label>Access expires<select value={ttl} onChange={(event) => setTtl(event.target.value)}><option value="1440">After 1 day</option><option value="10080">After 7 days</option><option value="43200">After 30 days</option></select></label><p>Access also expires at the response deadline.</p><div className="v-intake-actions"><button type="button" onClick={onCancel} disabled={busy}>Cancel</button><button className="v-primary" disabled={busy}>{busy ? "Sending…" : "Send replacement"}</button></div></form>;
}

function SendAssessment({ assessment, onCancel, onDone, onFailed }: { assessment: VendorAssessment; onCancel: () => void; onDone: (value: VendorAssessmentSendState) => void; onFailed: (error: unknown) => void }) {
  const [audience, setAudience] = useState(""); const [deadline, setDeadline] = useState(""); const [ttl, setTtl] = useState("10080"); const [busy, setBusy] = useState(false);
  async function submit(event: FormEvent) {
    event.preventDefault(); const when = new Date(deadline); if (!audience.trim() || Number.isNaN(when.getTime()) || busy) return; setBusy(true);
    try { const outcome = await sendVendorAssessmentRequest(assessment.id, { expected_version: assessment.version, audience: audience.trim(), deadline: when.toISOString(), invitation_ttl_minutes: Number(ttl) }); onDone(outcome.state); }
    catch (error) { onFailed(error); } finally { setBusy(false); }
  }
  return <form className="v-panel v-intake" onSubmit={(event) => void submit(event)}><h2>Send assessment request</h2><label>Recipient email<input required type="email" value={audience} onChange={(event) => setAudience(event.target.value)}/></label><label>Response deadline<input required type="datetime-local" value={deadline} onChange={(event) => setDeadline(event.target.value)}/></label><label>Access expires<select value={ttl} onChange={(event) => setTtl(event.target.value)}><option value="1440">After 1 day</option><option value="10080">After 7 days</option><option value="43200">After 30 days</option></select></label><p>Access also expires at the response deadline.</p><div className="v-intake-actions"><button type="button" onClick={onCancel} disabled={busy}>Cancel</button><button className="v-primary" disabled={busy}>{busy ? "Sending…" : "Send request"}</button></div></form>;
}

function StartAssessment({ relationship, onCancel, onDone, onFailed }: { relationship: VendorRelationshipAggregate; onCancel: () => void; onDone: () => void; onFailed: (error: unknown) => void }) {
  const [forms, setForms] = useState<ReusableFormTemplateRef[]>([]); const [state, setState] = useState<LoadState>("loading");
  const [formID, setFormID] = useState(""); const [due, setDue] = useState(""); const [busy, setBusy] = useState(false);
  const [cursor, setCursor] = useState<string>(); const [searchDraft, setSearchDraft] = useState(""); const [search, setSearch] = useState(""); const [formAttempt, setFormAttempt] = useState(0); const [moreBusy, setMoreBusy] = useState(false); const [moreError, setMoreError] = useState(false); const generation = useRef(0);
  useEffect(() => {
    const token = ++generation.current; setState("loading"); setForms([]); setCursor(undefined); setFormID(""); setMoreBusy(false); setMoreError(false);
    loadFormTemplatePage({ search: search || undefined, limit: 50 }).then((page) => { if (generation.current === token) { setForms(activeForms(page.items)); setCursor(page.next_cursor); setState("ready"); } }).catch(() => { if (generation.current === token) setState("error"); });
    return () => { generation.current++; };
  }, [search, formAttempt]);
  async function moreForms() {
    if (!cursor || moreBusy) return;
    const token = generation.current; setMoreBusy(true); setMoreError(false);
    try { const page = await loadFormTemplatePage({ search: search || undefined, cursor, limit: 50 }); if (generation.current === token) { setForms((previous) => [...previous, ...activeForms(page.items)]); setCursor(page.next_cursor); } }
    catch { if (generation.current === token) setMoreError(true); } finally { if (generation.current === token) setMoreBusy(false); }
  }
  async function submit(event: FormEvent) {
    event.preventDefault(); const form = forms.find((item) => item.id === formID); const when = new Date(due);
    if (!form || Number.isNaN(when.getTime()) || busy) return;
    setBusy(true);
    try { await startVendorAssessment(relationship.relationship.id, { relationship_version: relationship.relationship.version, review_kind: "ONBOARDING", form_template_id: form.id, form_template_version: form.version, review_due_at: when.toISOString() }); onDone(); }
    catch (error) { onFailed(error); } finally { setBusy(false); }
  }
  return <form className="v-panel v-intake" onSubmit={(event) => void submit(event)}><h2>Onboarding assessment</h2><label>Search forms<input value={searchDraft} onChange={(event) => setSearchDraft(event.target.value)}/></label><button type="button" onClick={() => { setSearch(searchDraft.trim()); setFormAttempt((value) => value + 1); }}>Find forms</button>{state === "loading" && <p role="status">Loading forms…</p>}{state === "error" && <p role="alert">Assessment forms unavailable.</p>}{state === "ready" && <><label>Assessment form<select value={formID} required onChange={(event) => setFormID(event.target.value)}><option value="">Select a form</option>{forms.map((form) => <option key={form.id} value={form.id}>{form.name} · v{form.version}</option>)}</select></label>{forms.length === 0 && <p>No active forms in the loaded results.</p>}{moreError && <p role="alert">More forms unavailable.</p>}{cursor && <button type="button" disabled={moreBusy} onClick={() => void moreForms()}>{moreBusy ? "Loading…" : "Load more forms"}</button>}<label>Review due<input type="datetime-local" value={due} required onChange={(event) => setDue(event.target.value)}/></label><div className="v-intake-actions"><button type="button" disabled={busy} onClick={onCancel}>Cancel</button><button className="v-primary" disabled={busy || forms.length === 0}>{busy ? "Starting…" : "Start assessment"}</button></div></>}{state !== "ready" && <button type="button" onClick={onCancel}>Cancel</button>}</form>;
}

function activeForms(items: Awaited<ReturnType<typeof loadFormTemplatePage>>["items"]): ReusableFormTemplateRef[] { return items.flatMap((item) => item.active_version && item.active_status === "ACTIVE" ? [{ id: item.template.id, name: item.template.name, code: item.template.code, version: item.active_version }] : []); }

function CompleteAssessment({ assessment, onCancel, onDone, onFailed }: { assessment: VendorAssessment; onCancel: () => void; onDone: () => void; onFailed: (error: unknown) => void }) {
  const [conclusion, setConclusion] = useState<"" | VendorAssessmentConclusion>(""); const [rationale, setRationale] = useState(""); const [uncertainty, setUncertainty] = useState(""); const [busy, setBusy] = useState(false);
  async function submit(event: FormEvent) {
    event.preventDefault(); if (!conclusion || !rationale.trim() || busy) return; setBusy(true);
    try { await completeVendorAssessment(assessment.id, { expected_version: assessment.version, conclusion, rationale: rationale.trim(), ...(uncertainty.trim() ? { uncertainty: uncertainty.trim() } : {}) }); onDone(); }
    catch (error) { onFailed(error); } finally { setBusy(false); }
  }
  return <form className="v-panel v-intake" onSubmit={(event) => void submit(event)}><h2>Complete review</h2><label>Conclusion<select required value={conclusion} onChange={(event) => setConclusion(event.target.value as typeof conclusion)}><option value="">Select conclusion</option><option value="SATISFACTORY">Satisfactory</option><option value="SATISFACTORY_WITH_CONDITIONS">Satisfactory with conditions</option><option value="UNSATISFACTORY">Unsatisfactory</option><option value="INCONCLUSIVE">Inconclusive</option></select></label><label>Rationale<textarea required maxLength={4000} value={rationale} onChange={(event) => setRationale(event.target.value)}/></label><label>Uncertainty<textarea maxLength={2000} value={uncertainty} onChange={(event) => setUncertainty(event.target.value)}/></label><div className="v-intake-actions"><button type="button" onClick={onCancel} disabled={busy}>Cancel</button><button className="v-primary" disabled={busy}>{busy ? "Completing…" : "Complete review"}</button></div></form>;
}

function label(value: string) { const text = value.toLowerCase().replaceAll("_", " "); return text.charAt(0).toUpperCase() + text.slice(1); }
function date(value: string) { const parsed = new Date(value); return Number.isNaN(parsed.getTime()) ? "unknown" : new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(parsed); }
