import type { ApiErrorKind } from "../../http";
import type { CaptureAnswerValue, CaptureAnswers, CaptureField, CaptureRequest } from "../../types";
import type { CaptureAttachment } from "./CaptureFieldControl";
import { answerText, normalizeFieldType } from "./contract";
import { reviewSourceLabel } from "./sourceProvenance";

type Props = { request: CaptureRequest; fields: CaptureField[]; answers: CaptureAnswers; attachments: Record<string, CaptureAttachment>; submitting: boolean; error: string | null; errorKind: ApiErrorKind | null; onEdit: () => void; onReload?: () => void; onSubmit: () => void };

export function CaptureReview({ request, fields, answers, attachments, submitting, error, errorKind, onEdit, onReload, onSubmit }: Props) {
  return <div className="panel-content response-review">
    <span className="eyebrow">Review response</span><h2>Check your response</h2><p>{request.title}</p>
    <dl className="capture-review-list">{fields.map((field) => { const sourceLabel = reviewSourceLabel(field, answerText(answers[field.id])); return <div key={field.id}><dt>{field.label}</dt><dd>{reviewValue(field, answers[field.id], attachments[field.id])}{sourceLabel && <small className="source-origin-review">{sourceLabel}</small>}</dd></div>; })}</dl>
    <details className="capture-context"><summary>Request details</summary><p>{request.purpose}</p><dl className="known-facts">{Object.entries(request.known_facts).map(([key, value]) => <div key={key}><dt>{humanize(key)}</dt><dd>{value}</dd></div>)}</dl><p>Due {new Date(request.deadline).toLocaleString()} · {humanize(request.sensitivity)}</p></details>
    {error && <p className="error-text" role="alert">{error}</p>}
    <div className="wizard-actions"><button className="secondary-button" type="button" onClick={onEdit} disabled={submitting}>Edit response</button>{errorKind === "conflict" && onReload && <button className="secondary-button" type="button" onClick={onReload} disabled={submitting}>Reload request</button>}<button className="primary-button" type="button" onClick={onSubmit} disabled={submitting}>{submitting ? "Submitting…" : "Submit response"}</button></div>
  </div>;
}

function reviewValue(field: CaptureField, answer?: CaptureAnswerValue, attachment?: CaptureAttachment) {
  const type = normalizeFieldType(field.type);
  if (!answer) return "Not provided";
  if (type === "signature") return answer.artifact_ids?.length ? "Signed" : "Not provided";
  if (type === "photo") return answer.artifact_ids?.length ? attachment ? `Photo attached · ${attachment.file_name}` : "Photo attached" : "Not provided";
  if (type === "file") return answer.artifact_ids?.length ? attachment ? `File attached · ${attachment.file_name}` : "File attached" : "Not provided";
  if (type === "vendor_document") return answer.document ? [answer.document.document_type, answer.document.reference, answer.document.expires_on ? `expires ${formatDate(answer.document.expires_on)}` : ""].filter(Boolean).join(" · ") : "Not provided";
  if (type === "multi_select") return answer.values?.length ? answer.values.join(", ") : "Not provided";
  if (type === "checkbox" || type === "attestation") return answer.text === "true" ? "Confirmed" : "Not confirmed";
  if (!answer.text?.trim()) return "Not provided";
  return type === "date" ? formatDate(answer.text) : answer.text;
}

function formatDate(value: string) { const parsed = Date.parse(`${value}T00:00:00`); return Number.isFinite(parsed) ? new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(new Date(parsed)) : value; }
function humanize(value: string) { return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase()); }
