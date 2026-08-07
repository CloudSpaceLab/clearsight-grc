import { useEffect, useState } from "react";
import { submitCaptureRequest } from "../api";
import { uploadInternalCaptureArtifact, type CaptureArtifact, type CaptureReceipt } from "../captureApi";
import { apiErrorKind, type ApiErrorKind } from "../http";
import type { CaptureRequest } from "../types";
import { EmptyState } from "./EmptyState";
import { SignatureCapture } from "./SignatureCapture";

export type CaptureLoadState = "loading" | "live" | "unavailable" | "forbidden" | "not-found";

type CaptureField = CaptureRequest["fields"][number] & { accepted_formats?: string[] };
type SubmitResult = Pick<CaptureReceipt, "submitted_at"> & Partial<CaptureReceipt>;
type Attachment = Pick<CaptureArtifact, "id" | "file_name" | "media_type">;

type Props = {
  request: CaptureRequest | null;
  state?: CaptureLoadState;
  onReload?: () => void;
  external?: boolean;
  onSubmit?: (request: CaptureRequest, answers: Record<string, string>) => Promise<SubmitResult>;
  onUploadArtifact?: (requestID: string, file: File) => Promise<CaptureArtifact>;
};

export function CapturePanel({ request, state = "live", onReload, external = false, onSubmit, onUploadArtifact }: Props) {
  const [answers, setAnswers] = useState<Record<string, string>>({});
  const [attachments, setAttachments] = useState<Record<string, Attachment>>({});
  const [uploadingField, setUploadingField] = useState<string | null>(null);
  const [reviewing, setReviewing] = useState(false);
  const [receipt, setReceipt] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [errorKind, setErrorKind] = useState<ApiErrorKind | null>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    setAnswers({});
    setAttachments({});
    setUploadingField(null);
    setReviewing(false);
    setReceipt(null);
    setError(null);
    setErrorKind(null);
    setSubmitting(false);
  }, [request?.id, request?.version]);

  if (state === "loading") return <div className="panel-content"><span className="eyebrow">{external ? "Verification request" : "Evidence request"}</span><h2>Loading request</h2><p aria-live="polite" aria-busy="true">Getting the latest request…</p></div>;
  if (state === "forbidden") return <div className="panel-content"><EmptyState kind="forbidden" label="Request" title="You cannot open this request" description="Your current access does not allow you to view it."/></div>;
  if (state === "not-found") return <div className="panel-content"><EmptyState kind="not-found" label="Request" title="This request is no longer available" description="It may have been replaced, cancelled, or moved outside your access."/></div>;
  if (state === "unavailable") return <div className="panel-content"><EmptyState kind="unavailable" label="Request" title="The request could not be loaded" description="Try again. No response has been recorded." action={onReload ? "Try again" : undefined} onAction={onReload}/></div>;
  if (!request) return <div className="panel-content"><EmptyState label="Request" title="No request selected" description="Open a request from your work list."/></div>;

  const effectiveStatus = isPastDeadline(request) && ["READY", "IN_PROGRESS"].includes(request.status) ? "EXPIRED" : request.status;
  if (effectiveStatus !== "READY" && effectiveStatus !== "IN_PROGRESS") return <TerminalRequest request={request} status={effectiveStatus}/>;

  function updateAnswer(fieldID: string, value: string) {
    setAnswers((current) => ({ ...current, [fieldID]: value }));
  }

  async function upload(field: CaptureField, file: File) {
    if (!request || uploadingField) return;
    setUploadingField(field.id);
    setError(null);
    setErrorKind(null);
    try {
      const artifact = await (onUploadArtifact ?? uploadInternalCaptureArtifact)(request.id, file);
      setAttachments((current) => ({ ...current, [field.id]: artifact }));
      updateAnswer(field.id, artifact.id);
    } catch (cause) {
      const kind = apiErrorKind(cause);
      setErrorKind(kind);
      setError(kind === "validation" ? "That file cannot be used for this request." : "The file could not be uploaded. Your other answers are still here.");
    } finally {
      setUploadingField(null);
    }
  }

  async function submit() {
    if (!request || submitting) return;
    setError(null);
    setErrorKind(null);
    setSubmitting(true);
    try {
      const result = onSubmit
        ? await onSubmit(request, answers)
        : await submitCaptureRequest(request.id, request.version, answers);
      setReceipt(new Date(result.submitted_at).toLocaleString());
      setReviewing(false);
    } catch (cause) {
      const kind = apiErrorKind(cause);
      setErrorKind(kind);
      setError(errorMessage(kind, cause));
    } finally {
      setSubmitting(false);
    }
  }

  if (receipt) return <div className="panel-content response-receipt"><span className="eyebrow">Receipt</span><div className="receipt-mark" aria-hidden="true">✓</div><h2>{external ? "Submitted" : "Response submitted"}</h2><p>{receipt}</p><p>{external ? "Your verification was recorded." : "Recorded. Evidence quality is reviewed separately."}</p></div>;

  const fields = request.fields as CaptureField[];
  const unsupported = fields.filter((field) => !supportedFieldType(field.type));
  const requiredMissing = fields.some((field) => field.required && !(answers[field.id] ?? "").trim());

  if (reviewing) return <div className="panel-content response-review">
    <span className="eyebrow">Review</span><h2>Check your response</h2><p>{request.title}</p>
    <dl className="capture-review-list">{fields.map((field) => <div key={field.id}><dt>{field.label}</dt><dd>{reviewValue(field, answers[field.id], attachments[field.id])}</dd></div>)}</dl>
    <details className="capture-context"><summary>Request details</summary><p>{request.purpose}</p><dl className="known-facts">{Object.entries(request.known_facts).map(([key, value]) => <div key={key}><dt>{humanize(key)}</dt><dd>{value}</dd></div>)}</dl><p>Due {new Date(request.deadline).toLocaleString()} · {humanize(request.sensitivity)}</p></details>
    {error && <p className="error-text" role="alert">{error}</p>}
    <div className="wizard-actions"><button className="secondary-button" type="button" onClick={() => setReviewing(false)} disabled={submitting}>Edit</button>{errorKind === "conflict" && onReload && <button className="secondary-button" type="button" onClick={onReload} disabled={submitting}>Reload request</button>}<button className="primary-button" type="button" onClick={() => void submit()} disabled={submitting}>{submitting ? "Submitting…" : external ? "Submit verification" : "Submit response"}</button></div>
  </div>;

  return <div className="panel-content">
    <span className="eyebrow">{external ? "Verification request" : "Evidence request"} · about {request.estimated_minutes} min</span><h2>{request.title}</h2><p>{request.purpose}</p>
    <div className="why-you"><strong>Why this was sent to you</strong><span>{request.why_you}</span></div>
    {Object.keys(request.known_facts).length > 0 && <><h3>Already filled in</h3><dl className="known-facts">{Object.entries(request.known_facts).map(([key, value]) => <div key={key}><dt>{humanize(key)}</dt><dd>{value}</dd></div>)}</dl></>}
    {unsupported.length > 0 && <div className="inline-error" role="alert"><strong>This request includes a field this version cannot safely collect.</strong><p>{unsupported.map((field) => field.label).join(", ")}. Ask the sender to update the request.</p></div>}
    <div className="capture-form">{fields.map((field) => <FieldControl key={field.id} field={field} value={answers[field.id] ?? ""} attachment={attachments[field.id]} uploading={uploadingField === field.id} onChange={(value) => updateAnswer(field.id, value)} onUpload={(file) => void upload(field, file)}/>)}</div>
    {error && <p className="error-text" role="alert">{error}</p>}
    <div className="wizard-actions"><button className="primary-button" type="button" onClick={() => setReviewing(true)} disabled={requiredMissing || unsupported.length > 0 || Boolean(uploadingField)}>{uploadingField ? "Uploading…" : "Review and submit"}</button></div>
  </div>;
}

function FieldControl({ field, value, attachment, uploading, onChange, onUpload }: { field: CaptureField; value: string; attachment?: Attachment; uploading: boolean; onChange: (value: string) => void; onUpload: (file: File) => void }) {
  if (!supportedFieldType(field.type)) return null;
  const type = field.type.toLowerCase();
  if (type === "signature") return <SignatureCapture value={value} label={`${field.label}${field.required ? " *" : ""}`} attestation={field.description} onChange={onChange}/>;
  if (type === "photo" || type === "file") {
    const accept = field.accepted_formats?.join(",") || (type === "photo" ? "image/*" : undefined);
    return <fieldset className="capture-field"><legend>{field.label}{field.required ? " *" : ""}</legend>{field.description && <p className="field-help">{field.description}</p>}<label className="upload-control"><input type="file" accept={accept} capture={type === "photo" ? "environment" : undefined} disabled={uploading} onChange={(event) => { const file = event.target.files?.[0]; if (file) onUpload(file); }}/><span className="upload-trigger">{uploading ? "Uploading…" : attachment ? type === "photo" ? "Replace photo" : "Replace file" : type === "photo" ? "Take or choose photo" : "Choose file"}</span></label>{attachment && <div className="upload-status"><strong>{attachment.file_name}</strong><span>Attached</span></div>}</fieldset>;
  }
  if (type === "single_select" && (field.options?.length ?? 0) <= 4) return <fieldset className="capture-field"><legend>{field.label}{field.required ? " *" : ""}</legend>{field.description && <p className="field-help">{field.description}</p>}<div className="choice-grid">{field.options?.map((option) => <label className={value === option ? "choice-option selected" : "choice-option"} key={option}><input type="radio" name={field.id} value={option} checked={value === option} onChange={() => onChange(option)}/><span>{option}</span></label>)}</div></fieldset>;
  if (type === "single_select") return <label className="capture-field"><span>{field.label}{field.required ? " *" : ""}</span>{field.description && <small className="field-help">{field.description}</small>}<select value={value} onChange={(event) => onChange(event.target.value)}><option value="">Choose one</option>{field.options?.map((option) => <option key={option}>{option}</option>)}</select></label>;
  if (type === "long_text") return <label className="capture-field"><span>{field.label}{field.required ? " *" : ""}</span>{field.description && <small className="field-help">{field.description}</small>}<textarea value={value} onChange={(event) => onChange(event.target.value)}/></label>;
  if (type === "date") return <label className="capture-field"><span>{field.label}{field.required ? " *" : ""}</span>{field.description && <small className="field-help">{field.description}</small>}<input type="date" value={value} onChange={(event) => onChange(event.target.value)}/></label>;
  if (type === "number") return <label className="capture-field"><span>{field.label}{field.required ? " *" : ""}</span>{field.description && <small className="field-help">{field.description}</small>}<input type="number" inputMode="decimal" value={value} onChange={(event) => onChange(event.target.value)}/></label>;
  return <label className="capture-field"><span>{field.label}{field.required ? " *" : ""}</span>{field.description && <small className="field-help">{field.description}</small>}<input type="text" value={value} onChange={(event) => onChange(event.target.value)}/></label>;
}

function TerminalRequest({ request, status }: { request: CaptureRequest; status: string }) {
  const copy: Record<string, [string, string]> = {
    SUBMITTED: ["Response already submitted", "This request already has a response."],
    EXPIRED: ["This request has expired", "The deadline has passed. Ask the sender to extend or replace the request."],
    CANCELLED: ["This request was cancelled", "No further response can be submitted."],
    DRAFT: ["This request is not ready", "The sender has not released it for response yet."],
  };
  const [title, description] = copy[status] ?? ["This request is read-only", `The request is currently ${humanize(status)} and cannot accept a response.`];
  return <div className="panel-content"><EmptyState kind="not-found" label="Request" title={title} description={description}/><p className="request-terminal-context">{request.title}</p></div>;
}

function supportedFieldType(type: string) {
  return ["text", "short_text", "long_text", "single_select", "date", "number", "photo", "file", "signature"].includes(type.toLowerCase());
}

function reviewValue(field: CaptureField, answer?: string, attachment?: Attachment) {
  if (!answer?.trim()) return "Not provided";
  const type = field.type.toLowerCase();
  if (type === "signature") return "Signed";
  if (type === "photo") return attachment ? `Photo attached · ${attachment.file_name}` : "Photo attached";
  if (type === "file") return attachment ? `File attached · ${attachment.file_name}` : "File attached";
  if (type === "date") {
    const parsed = Date.parse(`${answer}T00:00:00`);
    return Number.isFinite(parsed) ? new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(new Date(parsed)) : answer;
  }
  return answer;
}

function isPastDeadline(request: CaptureRequest) {
  const deadline = Date.parse(request.deadline);
  return Number.isFinite(deadline) && deadline <= Date.now();
}

function errorMessage(kind: ApiErrorKind, cause: unknown) {
  if (kind === "conflict") return "This request changed while you were working. Reload it before submitting. Your current entries remain on this screen.";
  if (kind === "forbidden" || kind === "unauthorized") return "Your access to this request has ended. Ask the sender to confirm your access or send a new link.";
  if (kind === "not_found") return "This request is no longer available.";
  if (kind === "unavailable") return "The response could not be submitted right now. Your entries remain on this screen.";
  return cause instanceof Error ? cause.message : "The response could not be submitted.";
}

function humanize(value: string) {
  return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}
