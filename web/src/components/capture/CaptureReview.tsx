import type { ApiErrorKind } from "../../http";
import type { CaptureAnswerValue, CaptureAnswers, CaptureField, CaptureRequest } from "../../types";
import type { CaptureAttachment } from "./CaptureFieldControl";
import { answerIsPresent, answerText, documentAlreadyReceived } from "./contract";
import { reviewProvenanceLabel } from "./sourceProvenance";
import { AnswerValueDisplay } from "../forms/AnswerValueDisplay";
import { ResponseAnswerSheet, type AnswerSheetField, type AnswerSheetItem } from "../forms/ResponseAnswerSheet";

type Props = { request: CaptureRequest; fields: CaptureField[]; answers: CaptureAnswers; attachments: Record<string, CaptureAttachment[]>; external?: boolean; submitting: boolean; error: string | null; errorKind: ApiErrorKind | null; onEdit: () => void; onReload?: () => void; onSubmit: () => void };

export function CaptureReview({ request, fields, answers, attachments, external = false, submitting, error, errorKind, onEdit, onReload, onSubmit }: Props) {
  const items: AnswerSheetItem[] = fields.map((field) => ({
    field: {
      id: field.id,
      label: field.label,
      type: field.type as AnswerSheetField["type"],
      required: field.required,
      section_id: field.section_id,
      description: field.description,
    },
    answer: answers[field.id],
  }));
  return <div className="panel-content response-review">
    <span className="eyebrow">Review response</span><h2>Check your response</h2><p>{request.title}</p>
    <ResponseAnswerSheet
      fields={items}
      sections={request.sections}
      emptyLabel="Not provided"
      evidenceEmptyLabel="No evidence submitted for this field."
      headingLevel="h3"
      excludeFromAttention={(item) => { const field = request.fields.find((candidate) => candidate.id === item.field.id); return field ? documentAlreadyReceived(field) : false; }}
      renderValue={(item) => { const field = request.fields.find((candidate) => candidate.id === item.field.id) ?? item.field; const sourceLabel = reviewProvenanceLabel(request, field, answerText(item.answer)); return <>{reviewValueDisplay(field, item.answer, attachments[item.field.id])}{sourceLabel && <small className="source-origin-review">{sourceLabel}</small>}</>; }}
    />
    <details className="capture-context"><summary>Request details</summary><p>{request.purpose}</p><dl className="known-facts">{Object.entries(request.known_facts).map(([key, value]) => <div key={key}><dt>{humanize(key)}</dt><dd>{value}</dd></div>)}</dl><p>Due {new Date(request.deadline).toLocaleString()} · {humanize(request.sensitivity)}</p></details>
    {error && <p className="error-text" role="alert">{error}</p>}
    <div className="wizard-actions"><button className="secondary-button" type="button" onClick={onEdit} disabled={submitting}>Edit response</button>{errorKind === "conflict" && onReload && <button className="secondary-button" type="button" onClick={onReload} disabled={submitting}>Reload request</button>}<button className="primary-button" type="button" onClick={onSubmit} disabled={submitting}>{submitting ? "Submitting…" : external ? "Submit evidence" : "Submit response"}</button></div>
  </div>;
}

function reviewValueDisplay(field: CaptureField, answer?: CaptureAnswerValue, attachments: CaptureAttachment[] = []) {
  if (documentAlreadyReceived(field) && !answerIsPresent(answer)) return "No upload needed.";
  return <AnswerValueDisplay type={field.type} answer={answer} attachments={attachments} emptyLabel="Not provided" evidenceEmptyLabel="No evidence submitted for this field."/>;
}

function humanize(value: string) { return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase()); }