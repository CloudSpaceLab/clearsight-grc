import { useEffect, useRef, useState, type FormEvent } from "react";
import { loadFormResponseWorkspace, redeemFormAccess, saveFormResponseWorkspace, sendFormAccessOTP, startFormAccess, submitFormResponseWorkspace, uploadCaptureSessionArtifact, verifyFormAccessOTP, type FormAccessOTPReceipt, type FormAccessStart, type FormResponseWorkspacePayload, type RedeemedFormAccessSession } from "../captureApi";
import { captureContract, normalizeFieldType, validateCaptureFields, visibleCaptureFields, visibleCaptureSections } from "../captureContract";
import type { CaptureAnswers, CaptureAnswerValue, CaptureField } from "../types";
import { apiErrorKind } from "../http";
import "./capture.css";

type Phase = "open" | "loading" | "recipient" | "code" | "response" | "unavailable" | "submitted" | "submission-unknown";
const supported = new Set(["short_text", "long_text", "email", "telephone", "url", "integer", "decimal", "percentage", "currency", "date", "yes_no", "single_select", "multi_select", "checkbox", "attestation", "file", "photo"]);

export function Capture({ invitationToken }: { invitationToken: string }) {
  const [phase, setPhase] = useState<Phase>(invitationToken ? "open" : "unavailable");
  const [access, setAccess] = useState<FormAccessStart>(); const [recipient, setRecipient] = useState("");
  const [challenge, setChallenge] = useState<FormAccessOTPReceipt>(); const [code, setCode] = useState("");
  const [session, setSession] = useState(""); const [payload, setPayload] = useState<FormResponseWorkspacePayload>();
  const [answers, setAnswers] = useState<CaptureAnswers>({}); const [dirty, setDirty] = useState<Set<string>>(new Set());
  const [busy, setBusy] = useState(false); const [error, setError] = useState("");
  const [uncertain, setUncertain] = useState(false); const [saved, setSaved] = useState(false);
  const [conflict, setConflict] = useState<"" | "save" | "submit">("");
  const [latest, setLatest] = useState<FormResponseWorkspacePayload>();
  const [choices, setChoices] = useState<Record<string, "local" | "server">>({});
  const [fileDetails, setFileDetails] = useState<Record<string, { name: string; status: string }>>({});
  const lock = useRef(false);
  useEffect(() => {
    if (phase !== "response" || (!dirty.size && !busy)) return;
    const warn = (event: BeforeUnloadEvent) => { event.preventDefault(); event.returnValue = ""; };
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [phase, dirty, busy]);

  async function openSession(value: RedeemedFormAccessSession) {
    const result = await loadFormResponseWorkspace(value.session_token);
    if (result.session.request_id !== value.request_id || result.session.distribution_id !== value.distribution_id || result.workspace.workspace.status !== "OPEN" || result.request.id !== value.request_id || !["READY", "INVITED", "VIEWED", "IN_PROGRESS"].includes(result.request.status)) throw new Error("unavailable");
    setSession(value.session_token); setPayload(result); setAnswers(result.workspace.answers); setDirty(new Set()); setUncertain(false); setSaved(false); setPhase("response");
  }

  async function open() {
    if (lock.current) return; lock.current = true; setBusy(true); setPhase("loading"); setError("");
    try {
      const result = await startFormAccess(invitationToken); setAccess(result);
      if (result.policy === "DIRECT_MAGIC_LINK") await openSession(await redeemFormAccess(invitationToken));
      else if ((result.policy === "SHARED_LINK_EMAIL_OTP" || result.policy === "DIRECT_LINK_EMAIL_OTP") && result.recipients?.length) {
        if (result.policy === "DIRECT_LINK_EMAIL_OTP") setRecipient(result.recipients[0]!.selector_id);
        setPhase("recipient");
      } else setPhase("unavailable");
    } catch { setPhase("unavailable"); }
    finally { lock.current = false; setBusy(false); }
  }

  async function issueCode(event: FormEvent) {
    event.preventDefault(); if (!recipient || lock.current) return; lock.current = true; setBusy(true); setError("");
    try { setChallenge(await sendFormAccessOTP(invitationToken, recipient)); setCode(""); setPhase("code"); }
    catch { setError("Code could not be sent. Try again or contact the requester."); }
    finally { lock.current = false; setBusy(false); }
  }
  async function verifyCode(event: FormEvent) {
    event.preventDefault(); if (!challenge || !/^\d{6}$/.test(code) || lock.current) return; lock.current = true; setBusy(true); setError("");
    try { await openSession(await verifyFormAccessOTP(invitationToken, challenge.challenge_id, code)); }
    catch { setError("Code could not be verified. Check the code or contact the requester."); }
    finally { lock.current = false; setBusy(false); }
  }
  function change(field: CaptureField, value: CaptureAnswerValue) { setAnswers((previous) => ({ ...previous, [field.id]: value })); setDirty((previous) => new Set(previous).add(field.id)); setSaved(false); setError(""); }
  async function upload(field: CaptureField, file: File) {
    if (!payload || lock.current || uncertain) return;
    const ids = answers[field.id]?.artifact_ids ?? [];
    if (field.constraints?.max_files !== undefined && ids.length >= field.constraints.max_files) { setError(`${field.label}: maximum ${field.constraints.max_files} documents.`); return; }
    if (field.constraints?.max_file_bytes !== undefined && file.size > field.constraints.max_file_bytes) { setError(`${field.label}: document exceeds the size limit.`); return; }
    lock.current = true; setBusy(true); setError("");
    try {
      const artifact = await uploadCaptureSessionArtifact(session, file, field.id);
      if (!artifact.id || artifact.request_id !== payload.request.id) throw new Error("unconfirmed");
      if (["REJECTED", "INFECTED", "QUARANTINED"].includes(artifact.status)) { setError("Document unavailable. Select another file."); return; }
      setFileDetails((value) => ({ ...value, [artifact.id]: { name: artifact.file_name || file.name, status: artifact.status } }));
      change(field, { artifact_ids: [...ids, artifact.id] });
    } catch { setError("Document upload could not be confirmed. Try again."); }
    finally { lock.current = false; setBusy(false); }
  }
  async function save() {
    if (!payload || lock.current || uncertain || dirty.size === 0) return;
    const visible = new Set(visibleCaptureFields(captureContract(payload.request), answers).map((field) => field.id));
    const edits = [...dirty].filter((id) => visible.has(id)).map((id) => ({ field_id: id, value: answers[id] ?? {}, base_sequence: payload.workspace.field_sequences[id] ?? 0 }));
    if (edits.length !== dirty.size) { setError("A changed answer is no longer visible. Reload the request before saving."); setUncertain(true); return; }
    lock.current = true; setBusy(true); setError("");
    try {
      const result = await saveFormResponseWorkspace(session, { expected_version: payload.workspace.workspace.version, presentation_mode: payload.workspace.presentation_mode, edits });
      if (result.workspace.id !== payload.workspace.workspace.id || result.workspace.status !== "OPEN" || result.workspace.version <= payload.workspace.workspace.version) throw new Error("unconfirmed");
      setPayload({ ...payload, workspace: result }); setAnswers(result.answers); setDirty(new Set()); setSaved(true);
    } catch (failure) { if (apiErrorKind(failure) === "conflict") { setConflict("save"); setLatest(undefined); setChoices({}); setError("The response changed. Compare answers before saving."); } else if (apiErrorKind(failure) === "validation") setError("Response not saved. Check your answers and try again."); else { setError("Response could not be confirmed. Unsaved answers remain below. Reload before saving again."); setUncertain(true); } }
    finally { lock.current = false; setBusy(false); }
  }
  async function reload() {
    if (!payload || lock.current) return; lock.current = true; setBusy(true); setError("");
    try {
      const result = await loadFormResponseWorkspace(session);
      if (result.workspace.workspace.id !== payload.workspace.workspace.id || result.workspace.workspace.status !== "OPEN" || result.session.request_id !== payload.session.request_id || result.session.distribution_id !== payload.session.distribution_id) throw new Error("unavailable");
      setPayload(result); setAnswers(result.workspace.answers); setDirty(new Set()); setUncertain(false); setSaved(false);
    } catch { setError("Request unavailable. Contact the requester."); }
    finally { lock.current = false; setBusy(false); }
  }
  async function compare() {
    if (!payload || lock.current) return; lock.current = true; setBusy(true);
    try {
      const result = await loadFormResponseWorkspace(session);
      if (result.workspace.workspace.id !== payload.workspace.workspace.id || result.workspace.workspace.status !== "OPEN" || result.session.request_id !== payload.session.request_id || result.session.distribution_id !== payload.session.distribution_id) throw new Error("unavailable");
      setLatest(result); setChoices({}); setError("");
    } catch { setError("Latest answers unavailable. Try again."); }
    finally { lock.current = false; setBusy(false); }
  }
  function applyChoices() {
    if (!payload || !latest) return;
    const conflicting = [...dirty].filter((id) => (payload.workspace.field_sequences[id] ?? 0) !== (latest.workspace.field_sequences[id] ?? 0));
    if (conflicting.some((id) => isDocumentConflict(payload.request.fields.find((field) => field.id === id), answers[id], latest.workspace.answers[id]))) { setError("Document answers changed. Contact the requester before continuing."); return; }
    if (conflicting.some((id) => !choices[id])) { setError("Choose an answer for each changed question."); return; }
    const nextAnswers = { ...latest.workspace.answers }; const nextDirty = new Set<string>();
    for (const id of dirty) { if (!conflicting.includes(id) || choices[id] === "local") { nextAnswers[id] = answers[id] ?? {}; nextDirty.add(id); } }
    const visible = new Set(visibleCaptureFields(captureContract(latest.request), nextAnswers).map((field) => field.id));
    if ([...nextDirty].some((id) => !visible.has(id))) { setError("A saved answer would hide another changed answer. Adjust your choices before continuing."); return; }
    setPayload(latest); setAnswers(nextAnswers); setDirty(nextDirty); setLatest(undefined); setConflict(""); setChoices({}); setError(""); setSaved(false);
  }
  async function submit() {
    if (!payload || lock.current || uncertain || dirty.size) return;
    const fields = visibleCaptureFields(captureContract(payload.request), answers);
    if (fields.some(unsupported)) return;
    const errors = validateCaptureFields(fields, answers);
    if (errors.length) { setError(errors[0]?.message ?? "Complete required answers."); return; }
    lock.current = true; setBusy(true); setError("");
    try {
      const attestation_field_ids = fields.filter((field) => normalizeFieldType(field.type) === "attestation" && answers[field.id]?.text === "true").map((field) => field.id);
      const result = await submitFormResponseWorkspace(session, { expected_version: payload.workspace.workspace.version, attestation_field_ids });
      const revision = result.revision;
      if (!result.submission.submission_id || result.submission.request_id !== payload.request.id || result.workspace.id !== payload.workspace.workspace.id || result.workspace.version <= payload.workspace.workspace.version || !(revision.id || revision.ID) || (revision.submission_id ?? revision.SubmissionID) !== result.submission.submission_id || (revision.current ?? revision.Current) !== true) throw new Error("unconfirmed");
      setPhase("submitted");
    } catch (failure) { if (apiErrorKind(failure) === "conflict") { setConflict("submit"); setLatest(undefined); setChoices({}); setError("The response changed. Compare answers before submitting."); } else if (apiErrorKind(failure) === "validation") setError("Response not submitted. Check your answers and try again."); else setPhase("submission-unknown"); }
    finally { lock.current = false; setBusy(false); }
  }

  return <main className="v-capture"><header className="v-capture-brand">ClearSight</header><div className="v-capture-content">
    {phase === "open" && <section className="v-capture-panel"><p className="v-capture-eyebrow">FORM REQUEST</p><h1>Open form request</h1><button className="v-capture-primary" onClick={() => void open()}>Open request</button></section>}
    {phase === "loading" && <p role="status">Opening request…</p>}
    {phase === "unavailable" && <section className="v-capture-panel"><h1>Request unavailable</h1><p>Contact the requester for a new invitation.</p></section>}
    {phase === "recipient" && <section className="v-capture-panel"><h1>Verify your email</h1><form onSubmit={(event) => void issueCode(event)}><fieldset><legend>Invitation recipient</legend>{access?.recipients?.map((item) => <label className="v-capture-choice" key={item.selector_id}><input type="radio" name="recipient" value={item.selector_id} checked={recipient === item.selector_id} onChange={() => setRecipient(item.selector_id)}/>{item.hint}</label>)}</fieldset>{error && <p role="alert">{error}</p>}<button className="v-capture-primary" disabled={!recipient || busy}>{busy ? "Sending…" : "Send code"}</button></form></section>}
    {phase === "code" && <section className="v-capture-panel"><h1>Verify your email</h1><p>Code sent to {challenge?.hint}.</p><form onSubmit={(event) => void verifyCode(event)}><label>Verification code<input autoComplete="one-time-code" inputMode="numeric" maxLength={6} value={code} onChange={(event) => setCode(event.target.value.replace(/\D/g, "").slice(0, 6))}/></label>{error && <p role="alert">{error}</p>}<button className="v-capture-primary" disabled={code.length !== 6 || busy}>{busy ? "Verifying…" : "Verify code"}</button><button type="button" disabled={busy} onClick={() => { setError(""); setPhase("recipient"); }}>Request another code</button></form></section>}
    {phase === "submitted" && <section className="v-capture-panel"><h1>Response submitted</h1><p>The requester will review your response.</p></section>}
    {phase === "submission-unknown" && <section className="v-capture-panel"><h1>Submission status unknown</h1><p>Contact the requester to confirm whether the response was received.</p></section>}
    {phase === "response" && payload && <section><p className="v-capture-eyebrow">FORM REQUEST</p><h1>{payload.request.title}</h1>{payload.request.purpose && <p>{payload.request.purpose}</p>}{payload.request.deadline && <p>Due {formatDate(payload.request.deadline)}</p>}
      {visibleCaptureSections(captureContract(payload.request), answers).map((section) => <section className="v-capture-panel" key={section.id}><h2>{section.title}</h2>{section.help && <p>{section.help}</p>}{section.fields.map((field) => <CaptureField key={field.id} field={field} value={answers[field.id]} fileDetails={fileDetails} disabled={busy || uncertain || Boolean(conflict)} onChange={(value) => change(field, value)} onUpload={(file) => void upload(field, file)}/>)}</section>)}
      {error && <p role="alert">{error}</p>}{saved && <p role="status">Response saved.</p>}
      {conflict ? <section className="v-capture-panel"><h2>Compare answers</h2>{!latest ? <button type="button" disabled={busy} onClick={() => void compare()}>Compare answers</button> : <>{[...dirty].filter((id) => (payload.workspace.field_sequences[id] ?? 0) !== (latest.workspace.field_sequences[id] ?? 0)).map((id) => isDocumentConflict(payload.request.fields.find((field) => field.id === id), answers[id], latest.workspace.answers[id]) ? <p key={id} role="alert">Document answers changed. Contact the requester before continuing.</p> : <fieldset key={id}><legend>{payload.request.fields.find((field) => field.id === id)?.label ?? "Changed question"}</legend><label><input type="radio" name={`conflict-${id}`} checked={choices[id] === "local"} onChange={() => setChoices((value) => ({ ...value, [id]: "local" }))}/>Keep your answer: {conflictLabel(answers[id])}</label><label><input type="radio" name={`conflict-${id}`} checked={choices[id] === "server"} onChange={() => setChoices((value) => ({ ...value, [id]: "server" }))}/>Use latest answer: {conflictLabel(latest.workspace.answers[id])}</label></fieldset>)}{dirty.size === 0 && <p>Review the latest answers before submitting.</p>}{![...dirty].some((id) => (payload.workspace.field_sequences[id] ?? 0) !== (latest.workspace.field_sequences[id] ?? 0) && isDocumentConflict(payload.request.fields.find((field) => field.id === id), answers[id], latest.workspace.answers[id])) && <button type="button" onClick={applyChoices}>Apply choices</button>}</>}</section> : uncertain ? <button type="button" disabled={busy} onClick={() => void reload()}>Discard unsaved changes and reload</button> : <div className="v-capture-actions">{dirty.size > 0 && <button className="v-capture-primary" disabled={busy} onClick={() => void save()}>{busy ? "Saving…" : "Save response"}</button>}{dirty.size === 0 && !visibleCaptureFields(captureContract(payload.request), answers).some(unsupported) && <button className="v-capture-primary" disabled={busy} onClick={() => void submit()}>{busy ? "Submitting…" : "Submit response"}</button>}</div>}
    </section>}
  </div></main>;
}

function unsupported(field: CaptureField) { const type = normalizeFieldType(field.type); return !type || !supported.has(type) || Boolean(field.bindings?.length || (field.collection_intent && field.collection_intent !== "CAPTURE") || field.record_target); }
function isDocumentConflict(field?: CaptureField, local?: CaptureAnswerValue, server?: CaptureAnswerValue) { return ["file", "photo", "vendor_document", "signature"].includes(field ? normalizeFieldType(field.type) ?? "" : "") || Boolean(local?.artifact_ids?.length || server?.artifact_ids?.length || local?.document || server?.document); }
function conflictLabel(value?: CaptureAnswerValue) { return value?.text || value?.values?.join(", ") || (value?.artifact_ids?.length ? `${value.artifact_ids.length} documents` : "No answer"); }

function CaptureField({ field, value, fileDetails, disabled, onChange, onUpload }: { field: CaptureField; value?: CaptureAnswerValue; fileDetails: Record<string, { name: string; status: string }>; disabled: boolean; onChange: (value: CaptureAnswerValue) => void; onUpload: (file: File) => void }) {
  const type = normalizeFieldType(field.type); const label = field.label; const text = value?.text ?? "";
  if (unsupported(field)) return <div className="v-capture-field"><p role="alert">{label} is unavailable. Contact the requester.</p></div>;
  if (type === "file" || type === "photo") return <div className="v-capture-field"><label>{label}{field.required && " *"}{field.description && <small>{field.description}</small>}<input aria-label={label} type="file" disabled={disabled} accept={field.accepted_formats?.join(",") || (type === "photo" ? "image/*" : undefined)} onChange={(event) => { const file = event.target.files?.[0]; if (file) onUpload(file); event.target.value = ""; }}/></label>{value?.artifact_ids?.length ? <><p>{value.artifact_ids.length} {value.artifact_ids.length === 1 ? "document attached" : "documents attached"}</p><ul>{value.artifact_ids.map((id) => { const detail = fileDetails[id]; return <li key={id}>{detail ? `${detail.name} · ${detail.status === "AVAILABLE" ? "Scanned at upload" : detail.status === "STORED_UNSCANNED" ? "Not scanned at upload" : "Scan status unavailable"}` : "Document · Scan status unavailable"}</li>; })}</ul><button type="button" disabled={disabled} onClick={() => onChange({ artifact_ids: (value.artifact_ids ?? []).slice(0, -1) })}>Remove last document</button></> : null}</div>;
  if (type === "checkbox" || type === "attestation") return <label className="v-capture-check"><input type="checkbox" disabled={disabled} checked={text === "true"} onChange={(event) => onChange({ text: event.target.checked ? "true" : "" })}/>{type === "attestation" ? field.attestation || label : label}{field.required && " *"}</label>;
  if (type === "multi_select") return <fieldset className="v-capture-field"><legend>{label}{field.required && " *"}</legend>{field.description && <p>{field.description}</p>}{field.options?.map((option) => <label className="v-capture-choice" key={option}><input type="checkbox" disabled={disabled} checked={value?.values?.includes(option) ?? false} onChange={(event) => onChange({ values: event.target.checked ? [...(value?.values ?? []), option] : (value?.values ?? []).filter((item) => item !== option) })}/>{option}</label>)}</fieldset>;
  if (type === "yes_no" || type === "single_select") return <label className="v-capture-field">{label}{field.required && " *"}{field.description && <small>{field.description}</small>}<select aria-label={label} disabled={disabled} value={text} onChange={(event) => onChange({ text: event.target.value })}><option value="">Select an answer</option>{(type === "yes_no" ? ["Yes", "No"] : field.options ?? []).map((option) => <option key={option}>{option}</option>)}</select></label>;
  const inputType = type === "email" ? "email" : type === "telephone" ? "tel" : type === "url" ? "url" : type === "date" ? "date" : ["integer", "decimal", "percentage", "currency"].includes(type ?? "") ? "number" : "text";
  return <label className="v-capture-field">{label}{field.required && " *"}{field.description && <small>{field.description}</small>}{type === "long_text" ? <textarea aria-label={label} disabled={disabled} value={text} onChange={(event) => onChange({ text: event.target.value })}/> : <input aria-label={label} type={inputType} disabled={disabled} value={text} onChange={(event) => onChange({ text: event.target.value })}/>}</label>;
}
function formatDate(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? "date unavailable" : new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeZone: "UTC" }).format(date); }
