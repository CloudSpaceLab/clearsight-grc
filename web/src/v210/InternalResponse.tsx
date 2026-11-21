import { useEffect, useRef, useState } from "react";
import { apiErrorKind } from "../http";
import { captureContract, normalizeFieldType, validateCaptureFields, visibleCaptureFields, visibleCaptureSections } from "../captureContract";
import type { CaptureAnswers, CaptureAnswerValue, CaptureField } from "../types";
import { loadInternalWorkspace, saveInternalWorkspace, submitInternalWorkspace, uploadInternalDocument, type InternalWorkspacePayload } from "./internalResponseApi";
import "./capture.css";

const supported = new Set(["short_text", "long_text", "email", "telephone", "url", "integer", "decimal", "percentage", "currency", "date", "yes_no", "single_select", "multi_select", "checkbox", "attestation", "file", "photo"]);
function unsupported(field: CaptureField) { const type = normalizeFieldType(field.type); return !type || !supported.has(type) || Boolean(field.bindings?.length || (field.collection_intent && field.collection_intent !== "CAPTURE") || field.record_target); }

/** Not routed until authenticated workspace endpoints pass backend acceptance. */
export function InternalResponse({ requestId }: { requestId: string }) {
  const [payload, setPayload] = useState<InternalWorkspacePayload>(); const [answers, setAnswers] = useState<CaptureAnswers>({}); const [dirty, setDirty] = useState<Set<string>>(new Set());
  const [phase, setPhase] = useState<"loading" | "ready" | "unavailable" | "submitted" | "unknown">("loading"); const [attempt, setAttempt] = useState(0);
  const [busy, setBusy] = useState(false); const [saved, setSaved] = useState(false); const [error, setError] = useState(""); const [conflict, setConflict] = useState(false); const [uncertain, setUncertain] = useState(false);
  const [latest, setLatest] = useState<InternalWorkspacePayload>(); const [choices, setChoices] = useState<Record<string, "local" | "server">>({}); const lock = useRef(false);
  const [fileDetails, setFileDetails] = useState<Record<string, { name: string; status: string }>>({});
  function valid(value: InternalWorkspacePayload) { return value.request.id === requestId && ["READY", "IN_PROGRESS"].includes(value.request.status) && value.workspace.workspace.status === "OPEN" && !!value.workspace.workspace.id && !!value.workspace.workspace.distribution_id; }
  useEffect(() => { let active = true; setPhase("loading"); loadInternalWorkspace(requestId).then((value) => { if (!active) return; if (!valid(value)) { setPhase("unavailable"); return; } setPayload(value); setAnswers(value.workspace.answers); setDirty(new Set()); setPhase("ready"); }).catch(() => { if (active) setPhase("unavailable"); }); return () => { active = false; }; }, [requestId, attempt]);
  useEffect(() => { if (phase !== "ready" || (!dirty.size && !busy)) return; const warn = (event: BeforeUnloadEvent) => { event.preventDefault(); event.returnValue = ""; }; window.addEventListener("beforeunload", warn); return () => window.removeEventListener("beforeunload", warn); }, [phase, dirty, busy]);
  function change(field: CaptureField, value: CaptureAnswerValue) { setAnswers((items) => ({ ...items, [field.id]: value })); setDirty((items) => new Set(items).add(field.id)); setSaved(false); setError(""); }
  async function upload(field: CaptureField, file: File) {
    if (!payload || lock.current || uncertain || conflict) return;
    const ids = answers[field.id]?.artifact_ids ?? [];
    if (field.constraints?.max_files !== undefined && ids.length >= field.constraints.max_files) { setError(`${field.label}: maximum ${field.constraints.max_files} documents.`); return; }
    if (field.constraints?.max_file_bytes !== undefined && file.size > field.constraints.max_file_bytes) { setError(`${field.label}: document exceeds the size limit.`); return; }
    lock.current = true; setBusy(true); setError("");
    try {
      const artifact = await uploadInternalDocument(requestId, file, field.id);
      if (!artifact.id || artifact.request_id !== requestId) throw new Error("unconfirmed");
      if (["REJECTED", "INFECTED", "QUARANTINED"].includes(artifact.status)) { setError("Document unavailable. Select another file."); return; }
      setFileDetails((items) => ({ ...items, [artifact.id]: { name: artifact.file_name || file.name, status: artifact.status } }));
      change(field, { artifact_ids: [...ids, artifact.id] });
    } catch { setUncertain(true); setError("Document upload could not be confirmed. Contact the requester."); }
    finally { lock.current = false; setBusy(false); }
  }
  async function save() {
    if (!payload || lock.current || uncertain || conflict || !dirty.size) return;
    const visible = new Set(visibleCaptureFields(captureContract(payload.request), answers).map((field) => field.id));
    if ([...dirty].some((id) => !visible.has(id))) { setError("A changed answer is no longer visible. Review the response before saving."); return; }
    const edits = [...dirty].map((id) => ({ field_id: id, base_sequence: payload.workspace.field_sequences[id] ?? 0, value: answers[id] ?? {} }));
    lock.current = true; setBusy(true); setError("");
    try { const result = await saveInternalWorkspace(requestId, { expected_version: payload.workspace.workspace.version, presentation_mode: payload.workspace.presentation_mode, edits }); if (result.workspace.id !== payload.workspace.workspace.id || result.workspace.status !== "OPEN" || result.workspace.version < payload.workspace.workspace.version) throw new Error("unconfirmed"); setPayload({ ...payload, workspace: result }); setAnswers(result.answers); setDirty(new Set()); setSaved(true); }
    catch (failure) { if (apiErrorKind(failure) === "conflict") { setConflict(true); setError("The response changed. Compare answers before saving."); } else if (apiErrorKind(failure) === "validation") setError("Response not saved. Check your answers and try again."); else { setUncertain(true); setError("Save status unknown. Contact the requester before continuing."); } }
    finally { lock.current = false; setBusy(false); }
  }
  async function compare() {
    if (!payload || lock.current) return; lock.current = true; setBusy(true);
    try { const value = await loadInternalWorkspace(requestId); if (!valid(value) || value.workspace.workspace.id !== payload.workspace.workspace.id || value.workspace.workspace.distribution_id !== payload.workspace.workspace.distribution_id) throw new Error("unavailable"); setLatest(value); setChoices({}); setError(""); }
    catch { setError("Latest answers unavailable. Try again."); } finally { lock.current = false; setBusy(false); }
  }
  function applyChoices() {
    if (!payload || !latest) return;
    const changed = [...dirty].filter((id) => (payload.workspace.field_sequences[id] ?? 0) !== (latest.workspace.field_sequences[id] ?? 0));
    if (changed.some((id) => (answers[id]?.artifact_ids?.length ?? 0) > 0 || (latest.workspace.answers[id]?.artifact_ids?.length ?? 0) > 0 || ["file", "photo"].includes(normalizeFieldType(payload.request.fields.find((field) => field.id === id)?.type ?? "") ?? ""))) { setError("A document changed. Contact the requester before continuing."); return; }
    if (changed.some((id) => !choices[id])) { setError("Choose an answer for each changed question."); return; }
    const next = { ...latest.workspace.answers }; const remaining = new Set<string>();
    for (const id of dirty) if (!changed.includes(id) || choices[id] === "local") { next[id] = answers[id] ?? {}; remaining.add(id); }
    const visible = new Set(visibleCaptureFields(captureContract(latest.request), next).map((field) => field.id));
    if ([...remaining].some((id) => !visible.has(id))) { setError("A saved answer would hide another changed answer. Adjust your choices before continuing."); return; }
    setPayload(latest); setAnswers(next); setDirty(remaining); setLatest(undefined); setChoices({}); setConflict(false); setError(""); setSaved(false);
  }
  async function submit() {
    if (!payload || lock.current || uncertain || conflict || dirty.size) return;
    const fields = visibleCaptureFields(captureContract(payload.request), answers); if (fields.some(unsupported)) return;
    const errors = validateCaptureFields(fields, answers); if (errors.length) { setError(errors[0]?.message ?? "Complete required answers."); return; }
    lock.current = true; setBusy(true); setError("");
    try { const result = await submitInternalWorkspace(requestId, { expected_version: payload.workspace.workspace.version, attestation_field_ids: fields.filter((field) => normalizeFieldType(field.type) === "attestation" && answers[field.id]?.text === "true").map((field) => field.id) }); const revision = result.revision; if (result.submission.request_id !== requestId || !result.submission.submission_id || result.workspace.id !== payload.workspace.workspace.id || result.workspace.version <= payload.workspace.workspace.version || !revision.id || revision.submission_id !== result.submission.submission_id || revision.current !== true) throw new Error("unconfirmed"); setPhase("submitted"); }
    catch (failure) { if (apiErrorKind(failure) === "conflict") { setConflict(true); setError("The response changed. Compare answers before saving."); } else if (apiErrorKind(failure) === "validation") setError("Response not submitted. Check your answers and try again."); else setPhase("unknown"); } finally { lock.current = false; setBusy(false); }
  }
  if (phase === "loading") return <p role="status">Loading form…</p>;
  if (phase === "unavailable") return <section><h1>Form unavailable</h1><p>Contact the requester.</p><button onClick={() => setAttempt((value) => value + 1)}>Try again</button></section>;
  if (phase === "submitted") return <section><h1>Response submitted</h1><p>The requester will review your response.</p></section>;
  if (phase === "unknown") return <section><h1>Submission status unknown</h1><p>Contact the requester to confirm whether the response was received.</p></section>;
  if (!payload) return null;
  return <section className="v-internal-response"><header className="v-page-heading"><div><p className="v-eyebrow">Form request</p><h1>{payload.request.title}</h1></div></header>{payload.request.purpose && <p>{payload.request.purpose}</p>}
    {visibleCaptureSections(captureContract(payload.request), answers).map((section) => <section className="v-capture-panel" key={section.id}><h2>{section.title}</h2>{section.fields.map((field) => <InternalField key={field.id} field={field} value={answers[field.id]} fileDetails={fileDetails} disabled={busy || uncertain || conflict} onChange={(value) => change(field, value)} onUpload={(file) => void upload(field, file)}/>)}</section>)}
    {error && <p role="alert">{error}</p>}{saved && <p role="status">Response saved.</p>}
    {conflict ? <section className="v-capture-panel"><h2>Compare answers</h2>{!latest ? <button disabled={busy} onClick={() => void compare()}>Compare answers</button> : <>{[...dirty].filter((id) => (payload.workspace.field_sequences[id] ?? 0) !== (latest.workspace.field_sequences[id] ?? 0)).map((id) => <fieldset key={id}><legend>{payload.request.fields.find((field) => field.id === id)?.label ?? "Changed question"}</legend><label><input type="radio" name={`conflict-${id}`} checked={choices[id] === "local"} onChange={() => setChoices((value) => ({ ...value, [id]: "local" }))}/>Keep your answer: {answerLabel(answers[id])}</label><label><input type="radio" name={`conflict-${id}`} checked={choices[id] === "server"} onChange={() => setChoices((value) => ({ ...value, [id]: "server" }))}/>Use latest answer: {answerLabel(latest.workspace.answers[id])}</label></fieldset>)}{dirty.size === 0 && <p>Review the latest answers before submitting.</p>}<button onClick={applyChoices}>Apply choices</button></>}</section> : !uncertain && <div className="v-capture-actions">{dirty.size ? <button className="v-primary" disabled={busy} onClick={() => void save()}>{busy ? "Saving…" : "Save response"}</button> : !visibleCaptureFields(captureContract(payload.request), answers).some(unsupported) && <button className="v-primary" disabled={busy} onClick={() => void submit()}>{busy ? "Submitting…" : "Submit response"}</button>}</div>}
  </section>;
}

function answerLabel(value?: CaptureAnswerValue) { return value?.text || value?.values?.join(", ") || (value?.artifact_ids?.length ? `${value.artifact_ids.length} document(s)` : "No answer"); }
function InternalField({ field, value, fileDetails, disabled, onChange, onUpload }: { field: CaptureField; value?: CaptureAnswerValue; fileDetails: Record<string, { name: string; status: string }>; disabled: boolean; onChange: (value: CaptureAnswerValue) => void; onUpload: (file: File) => void }) {
  const type = normalizeFieldType(field.type); const text = value?.text ?? "";
  if (unsupported(field)) return <p role="alert">{field.label} is unavailable. Contact the requester.</p>;
  if (type === "file" || type === "photo") return <div className="v-capture-field"><label>{field.label}{field.required && " *"}<input aria-label={field.label} type="file" disabled={disabled} accept={field.accepted_formats?.join(",") || (type === "photo" ? "image/*" : undefined)} onChange={(event) => { const file = event.target.files?.[0]; if (file) onUpload(file); event.target.value = ""; }}/></label>{value?.artifact_ids?.length ? <><ul>{value.artifact_ids.map((id) => { const detail = fileDetails[id]; return <li key={id}>{detail ? `${detail.name} · ${detail.status === "AVAILABLE" ? "Scanned at upload" : detail.status === "STORED_UNSCANNED" ? "Not scanned at upload" : "Scan status unavailable"}` : "Document · Scan status unavailable"}</li>; })}</ul><button type="button" disabled={disabled} onClick={() => onChange({ artifact_ids: (value.artifact_ids ?? []).slice(0, -1) })}>Remove last document</button></> : null}</div>;
  if (type === "checkbox" || type === "attestation") return <label className="v-capture-check"><input type="checkbox" disabled={disabled} checked={text === "true"} onChange={(event) => onChange({ text: event.target.checked ? "true" : "" })}/>{type === "attestation" ? field.attestation || field.label : field.label}{field.required && " *"}</label>;
  if (type === "multi_select") return <fieldset className="v-capture-field"><legend>{field.label}{field.required && " *"}</legend>{field.options?.map((option) => <label key={option}><input type="checkbox" disabled={disabled} checked={value?.values?.includes(option) ?? false} onChange={(event) => onChange({ values: event.target.checked ? [...(value?.values ?? []), option] : (value?.values ?? []).filter((item) => item !== option) })}/>{option}</label>)}</fieldset>;
  if (type === "yes_no" || type === "single_select") return <label className="v-capture-field">{field.label}{field.required && " *"}<select aria-label={field.label} disabled={disabled} value={text} onChange={(event) => onChange({ text: event.target.value })}><option value="">Select an answer</option>{(type === "yes_no" ? ["Yes", "No"] : field.options ?? []).map((option) => <option key={option}>{option}</option>)}</select></label>;
  const inputType = type === "email" ? "email" : type === "telephone" ? "tel" : type === "url" ? "url" : type === "date" ? "date" : ["integer", "decimal", "percentage", "currency"].includes(type ?? "") ? "number" : "text";
  return <label className="v-capture-field">{field.label}{field.required && " *"}{type === "long_text" ? <textarea aria-label={field.label} disabled={disabled} value={text} onChange={(event) => onChange({ text: event.target.value })}/> : <input aria-label={field.label} type={inputType} disabled={disabled} value={text} onChange={(event) => onChange({ text: event.target.value })}/>}</label>;
}
