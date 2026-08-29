import type { ApiErrorKind } from "../../http";
import type { CaptureAnswerValue, CaptureAnswers, CaptureField, CaptureRequest } from "../../types";
import type { CaptureAttachment } from "./CaptureFieldControl";
import { answerText, normalizeFieldType } from "./contract";
import { reviewSourceLabel } from "./sourceProvenance";

type Props = { request: CaptureRequest; fields: CaptureField[]; answers: CaptureAnswers; attachments: Record<string, CaptureAttachment[]>; external?: boolean; submitting: boolean; error: string | null; errorKind: ApiErrorKind | null; onEdit: () => void; onReload?: () => void; onSubmit: () => void };

export function CaptureReview({ request, fields, answers, attachments, external = false, submitting, error, errorKind, onEdit, onReload, onSubmit }: Props) {
  const groups = groupReviewFields(fields, answers);
  return <div className="panel-content response-review">
    <span className="eyebrow">Review response</span><h2>Check your response</h2><p>{request.title}</p>
    <div className="capture-review-groups">{groups.map((group) => <section key={group.label} aria-labelledby={`capture-review-${group.key}`}><h3 id={`capture-review-${group.key}`}>{group.label}</h3><dl className="capture-review-list">{group.fields.map((field) => { const sourceLabel = reviewSourceLabel(field, answerText(answers[field.id])); return <div key={field.id}><dt>{field.label}</dt><dd>{reviewValue(field, answers[field.id], attachments[field.id])}{sourceLabel && <small className="source-origin-review">{sourceLabel}</small>}</dd></div>; })}</dl></section>)}</div>
    <details className="capture-context"><summary>Request details</summary><p>{request.purpose}</p><dl className="known-facts">{Object.entries(request.known_facts).map(([key, value]) => <div key={key}><dt>{humanize(key)}</dt><dd>{value}</dd></div>)}</dl><p>Due {new Date(request.deadline).toLocaleString()} · {humanize(request.sensitivity)}</p></details>
    {error && <p className="error-text" role="alert">{error}</p>}
    <div className="wizard-actions"><button className="secondary-button" type="button" onClick={onEdit} disabled={submitting}>Edit response</button>{errorKind === "conflict" && onReload && <button className="secondary-button" type="button" onClick={onReload} disabled={submitting}>Reload request</button>}<button className="primary-button" type="button" onClick={onSubmit} disabled={submitting}>{submitting ? "Submitting…" : external ? "Submit evidence" : "Submit response"}</button></div>
  </div>;
}

function groupReviewFields(fields: CaptureField[], answers: CaptureAnswers) {
  const definitions = [
    { key: "confirmed", label: "Confirmed held information", fields: [] as CaptureField[] },
    { key: "updates", label: "Proposed updates", fields: [] as CaptureField[] },
    { key: "replacements", label: "Replacement documents", fields: [] as CaptureField[] },
    { key: "new-files", label: "New files and documents", fields: [] as CaptureField[] },
    { key: "other", label: "Other responses", fields: [] as CaptureField[] },
  ];
  for (const field of fields) {
    const answer = answers[field.id];
    if (field.collection_intent === "REPLACE_HELD_DOCUMENT") definitions[2]!.fields.push(field);
    else if (field.record_baseline && answer?.text === field.record_baseline.display_value) definitions[0]!.fields.push(field);
    else if (field.record_baseline && answer) definitions[1]!.fields.push(field);
    else if (["file", "photo", "vendor_document"].includes(normalizeFieldType(field.type) ?? "")) definitions[3]!.fields.push(field);
    else definitions[4]!.fields.push(field);
  }
  return definitions.filter((group) => group.fields.length > 0);
}

function reviewValue(field: CaptureField, answer?: CaptureAnswerValue, attachments: CaptureAttachment[] = []) {
  const type = normalizeFieldType(field.type);
  if (!answer) return "Not provided";
  if (type === "signature") return answer.artifact_ids?.length ? "Signed" : "Not provided";
	if (type === "photo") return answer.artifact_ids?.length ? attachments[0] ? `Photo attached · ${attachments[0].file_name}` : "Photo attached" : "Not provided";
	if (type === "file") return answer.artifact_ids?.length ? attachments.length ? `${attachments.length} file${attachments.length === 1 ? "" : "s"} attached · ${attachments.map((attachment) => attachment.file_name).join(", ")}` : `${answer.artifact_ids.length} file${answer.artifact_ids.length === 1 ? "" : "s"} attached` : "Not provided";
  if (type === "vendor_document") return answer.document ? [answer.document.document_type, answer.document.reference, answer.document.expires_on ? `expires ${formatDate(answer.document.expires_on)}` : ""].filter(Boolean).join(" · ") : "Not provided";
  if (type === "multi_select") return answer.values?.length ? answer.values.join(", ") : "Not provided";
  if (type === "checkbox" || type === "attestation") return answer.text === "true" ? "Confirmed" : "Not confirmed";
  if (!answer.text?.trim()) return "Not provided";
  return type === "date" ? formatDate(answer.text) : answer.text;
}

function formatDate(value: string) { const parsed = Date.parse(`${value}T00:00:00`); return Number.isFinite(parsed) ? new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(new Date(parsed)) : value; }
function humanize(value: string) { return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase()); }
