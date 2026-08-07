import { useEffect, useRef, useState } from "react";
import { submitCaptureRequest } from "../api";
import { uploadInternalCaptureArtifact, type CaptureArtifact, type CaptureReceipt } from "../captureApi";
import { apiErrorKind, type ApiErrorKind } from "../http";
import type { CaptureRequest } from "../types";
import { EmptyState } from "./EmptyState";
import { FileDropzone } from "./FileDropzone";
import { SignatureCapture } from "./SignatureCapture";

export type CaptureLoadState = "loading" | "live" | "unavailable" | "forbidden" | "not-found";

type CaptureField = CaptureRequest["fields"][number];
type SubmitResult = Pick<CaptureReceipt, "submitted_at"> & Partial<CaptureReceipt>;
type Attachment = Pick<CaptureArtifact, "file_name" | "media_type" | "size_bytes"> & { id?: string; preview_url?: string };

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
  const previewURLs = useRef<Record<string, string>>({});
  const mounted = useRef(true);
  const activeRequestKey = useRef("");
  const requestKey = request ? `${request.id}:${request.version}` : "";
  activeRequestKey.current = requestKey;

  useEffect(() => {
    revokeAllPreviews(previewURLs.current);
    previewURLs.current = {};
    setAnswers({});
    setAttachments({});
    setUploadingField(null);
    setReviewing(false);
    setReceipt(null);
    setError(null);
    setErrorKind(null);
    setSubmitting(false);
  }, [request?.id, request?.version]);

  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      revokeAllPreviews(previewURLs.current);
      previewURLs.current = {};
    };
  }, []);

  if (state === "loading") return <div className="panel-content"><span className="eyebrow">{external ? "Verification request" : "Evidence request"}</span><h2>Loading request</h2><p aria-live="polite" aria-busy="true">Getting the latest request…</p></div>;
  if (state === "forbidden") return <div className="panel-content"><EmptyState kind="forbidden" label="Request" title="You cannot open this request" description="Your current access does not allow you to view it."/></div>;
  if (state === "not-found") return <div className="panel-content"><EmptyState kind="not-found" label="Request" title="This request is no longer available" description="It may have been replaced, cancelled, or moved outside your access."/></div>;
  if (state === "unavailable") return <div className="panel-content"><EmptyState kind="unavailable" label="Request" title="The request could not be loaded" description="Try again. No response has been recorded." action={onReload ? "Try again" : undefined} onAction={onReload}/></div>;
  if (!request) return <div className="panel-content"><EmptyState label="Request" title="No request selected" description="Open a request from the evidence list."/></div>;

  const effectiveStatus = isPastDeadline(request) && ["READY", "IN_PROGRESS"].includes(request.status) ? "EXPIRED" : request.status;
  if (effectiveStatus !== "READY" && effectiveStatus !== "IN_PROGRESS") return <TerminalRequest request={request} status={effectiveStatus}/>;

  function updateAnswer(fieldID: string, value: string) {
    setAnswers((current) => ({ ...current, [fieldID]: value }));
  }

  async function upload(field: CaptureField, file: File, preferredPreviewURL?: string) {
    if (!request || uploadingField) return;
    const uploadRequestKey = requestKey;
    const previousAttachment = attachments[field.id];
    const previousObjectPreview = previewURLs.current[field.id];
    let previewURL = preferredPreviewURL;
    let createdObjectPreview = false;

    // For an initial image selection, preview immediately. During replacement, keep the
    // last valid attachment visible until the new upload succeeds so a transient failure
    // cannot destroy an already-valid answer.
    if (!previousAttachment && !previewURL && file.type.startsWith("image/") && typeof URL.createObjectURL === "function") {
      previewURL = URL.createObjectURL(file);
      createdObjectPreview = true;
      previewURLs.current[field.id] = previewURL;
    }
    if (!previousAttachment) {
      setAttachments((current) => ({ ...current, [field.id]: { file_name: file.name, media_type: file.type, size_bytes: file.size, preview_url: previewURL } }));
    }

    setUploadingField(field.id);
    setError(null);
    setErrorKind(null);
    try {
      const artifact = await (onUploadArtifact ?? uploadInternalCaptureArtifact)(request.id, file);
      if (!currentRequest(mounted.current, activeRequestKey.current, uploadRequestKey)) return;
      if (previousAttachment && !previewURL && file.type.startsWith("image/") && typeof URL.createObjectURL === "function") {
        previewURL = URL.createObjectURL(file);
        createdObjectPreview = true;
      }
      if (previousObjectPreview && previousObjectPreview !== previewURL) URL.revokeObjectURL(previousObjectPreview);
      if (createdObjectPreview && previewURL) previewURLs.current[field.id] = previewURL;
      else delete previewURLs.current[field.id];
      setAttachments((current) => ({ ...current, [field.id]: { id: artifact.id, file_name: artifact.file_name, media_type: artifact.media_type, size_bytes: artifact.size_bytes, preview_url: previewURL } }));
      updateAnswer(field.id, artifact.id);
    } catch (cause) {
      if (!currentRequest(mounted.current, activeRequestKey.current, uploadRequestKey)) return;
      if (createdObjectPreview && previewURL && previewURL !== previousObjectPreview) URL.revokeObjectURL(previewURL);
      if (previousObjectPreview) previewURLs.current[field.id] = previousObjectPreview;
      else delete previewURLs.current[field.id];
      if (!previousAttachment) {
        setAttachments((current) => {
          const next = { ...current };
          delete next[field.id];
          return next;
        });
      }
      const kind = apiErrorKind(cause);
      setErrorKind(kind);
      setError(kind === "validation" ? "That file cannot be used for this request." : previousAttachment ? "The replacement could not be uploaded. Your previous attachment is still selected." : "The file could not be uploaded. Your other answers are still here.");
    } finally {
      if (currentRequest(mounted.current, activeRequestKey.current, uploadRequestKey)) setUploadingField(null);
    }
  }

  async function submit() {
    if (!request || submitting) return;
    const submitRequestKey = requestKey;
    setError(null);
    setErrorKind(null);
    setSubmitting(true);
    try {
      const result = onSubmit ? await onSubmit(request, answers) : await submitCaptureRequest(request.id, request.version, answers);
      if (!currentRequest(mounted.current, activeRequestKey.current, submitRequestKey)) return;
      setReceipt(new Date(result.submitted_at).toLocaleString());
      setReviewing(false);
    } catch (cause) {
      if (!currentRequest(mounted.current, activeRequestKey.current, submitRequestKey)) return;
      const kind = apiErrorKind(cause);
      setErrorKind(kind);
      setError(errorMessage(kind, cause));
    } finally {
      if (currentRequest(mounted.current, activeRequestKey.current, submitRequestKey)) setSubmitting(false);
    }
  }

  if (receipt) return <div className="panel-content response-receipt"><span className="eyebrow">Receipt</span><div className="receipt-mark" aria-hidden="true">✓</div><h2>{external ? "Submitted" : "Response submitted"}</h2><p>{receipt}</p><p>{external ? "Your response was recorded." : "Recorded. Evidence quality is reviewed separately."}</p></div>;

  const fields = request.fields;
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
    <div className="capture-form">{fields.map((field) => <FieldControl key={field.id} field={field} value={answers[field.id] ?? ""} attachment={attachments[field.id]} uploading={uploadingField === field.id} external={external} onChange={(value) => updateAnswer(field.id, value)} onUpload={(file, previewURL) => void upload(field, file, previewURL)}/>)}</div>
    {error && <p className="error-text" role="alert">{error}</p>}
    <div className="wizard-actions"><button className="primary-button" type="button" onClick={() => setReviewing(true)} disabled={requiredMissing || unsupported.length > 0 || Boolean(uploadingField)}>{uploadingField ? "Uploading…" : "Review and submit"}</button></div>
  </div>;
}

function FieldControl({ field, value, attachment, uploading, external, onChange, onUpload }: { field: CaptureField; value: string; attachment?: Attachment; uploading: boolean; external: boolean; onChange: (value: string) => void; onUpload: (file: File, previewURL?: string) => void }) {
  if (!supportedFieldType(field.type)) return null;
  const type = normalizedFieldType(field.type);
  if (type === "signature") return <SignatureCapture value={attachment?.preview_url} label={`${field.label}${field.required ? " *" : ""}`} attestation={field.description} busy={uploading} onCapture={onUpload}/>;
  if (type === "photo" || type === "file") {
    const photo = type === "photo";
    const accept = normalizedAcceptedFormats(field.accepted_formats) ?? (photo ? "image/*" : undefined);
    return <div className="capture-field"><FileDropzone
      label={`${field.label}${field.required ? " *" : ""}`}
      description={field.description}
      accept={accept}
      capture={photo ? "environment" : undefined}
      compact={!photo}
      busy={uploading}
      actionLabel={photo ? "Take or add photo" : "Choose file"}
      replaceLabel={photo ? "Replace photo" : "Replace file"}
      fileName={attachment?.file_name}
      fileSize={attachment?.size_bytes}
      previewUrl={attachment?.preview_url}
      onSelect={(file) => onUpload(file)}
    /></div>;
  }
  if (type === "single_select" && (field.options?.length ?? 0) <= 4) return <fieldset className="capture-field"><legend>{field.label}{field.required ? " *" : ""}</legend>{field.description && <p className="field-help">{field.description}</p>}<div className="choice-grid">{field.options?.map((option) => <label className={value === option ? "choice-option selected" : "choice-option"} key={option}><input type="radio" name={field.id} value={option} checked={value === option} onChange={() => onChange(option)}/><span>{option}</span></label>)}</div></fieldset>;
  if (type === "single_select") return <label className="capture-field"><span>{field.label}{field.required ? " *" : ""}</span>{field.description && <small className="field-help">{field.description}</small>}<select value={value} onChange={(event) => onChange(event.target.value)}><option value="">Choose one</option>{field.options?.map((option) => <option key={option}>{option}</option>)}</select></label>;
  if (type === "long_text" && external && !field.required) return <details className="capture-optional-field"><summary>{field.label}</summary><label className="capture-field">{field.description && <small className="field-help">{field.description}</small>}<textarea aria-label={field.label} value={value} onChange={(event) => onChange(event.target.value)}/></label></details>;
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

function currentRequest(isMounted: boolean, activeKey: string, operationKey: string) {
  return isMounted && activeKey === operationKey;
}

function supportedFieldType(type: string) {
  return ["text", "short_text", "long_text", "single_select", "date", "number", "photo", "file", "signature"].includes(normalizedFieldType(type));
}

function normalizedFieldType(type: string) {
  return type.trim().toLowerCase();
}

function normalizedAcceptedFormats(values?: string[]) {
  const formats = (values ?? []).map((value) => value.split(";", 1)[0]?.trim().toLowerCase()).filter((value): value is string => Boolean(value));
  return formats.length ? formats.join(",") : undefined;
}

function reviewValue(field: CaptureField, answer?: string, attachment?: Attachment) {
  if (!answer?.trim()) return "Not provided";
  const type = normalizedFieldType(field.type);
  if (type === "signature") return "Signed";
  if (type === "photo") return attachment ? `Photo attached · ${attachment.file_name}` : "Photo attached";
  if (type === "file") return attachment ? `File attached · ${attachment.file_name}` : "File attached";
  if (type === "date") {
    const parsed = Date.parse(`${answer}T00:00:00`);
    return Number.isFinite(parsed) ? new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(new Date(parsed)) : answer;
  }
  return answer;
}

function revokeAllPreviews(values: Record<string, string>) {
  if (typeof URL.revokeObjectURL !== "function") return;
  for (const value of Object.values(values)) URL.revokeObjectURL(value);
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
