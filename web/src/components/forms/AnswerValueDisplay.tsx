import type { ReactNode } from "react";
import type { CaptureAnswerValue } from "../../types";
import { normalizeFieldType } from "../capture/contract";
import { Button, StatusBadge } from "../ui";

type Props = {
  type: string;
  answer?: CaptureAnswerValue;
  attachments?: { file_name: string }[];
  className?: string;
  emptyLabel?: string;
  evidenceEmptyLabel?: string;
  onOpenDocuments?: () => void;
};

const evidenceTypes = new Set(["file", "photo", "signature", "vendor_document"]);

export function AnswerValueDisplay({ type, answer, attachments = [], className, emptyLabel = "Not provided", evidenceEmptyLabel = "No evidence submitted for this field.", onOpenDocuments }: Props) {
  const fieldType = normalizeFieldType(type);
  const content = renderAnswer(fieldType, answer, attachments);
  if (content === null) {
    const label = evidenceTypes.has(fieldType ?? "") ? evidenceEmptyLabel : emptyLabel;
    return <span className={className}>{label}</span>;
  }
  const count = evidenceCount(answer);
  return <span className={className}>
    {content}
    {count > 0 && onOpenDocuments && <span className="answer-value-display__launcher"><Button variant="quiet" onPress={onOpenDocuments}>Open submitted documents</Button></span>}
  </span>;
}

function renderAnswer(fieldType: ReturnType<typeof normalizeFieldType>, answer?: CaptureAnswerValue, attachments: { file_name: string }[] = []): ReactNode | null {
  if (!answer) return null;
  const count = evidenceCount(answer);
  const metadata = evidenceMetadata(answer, attachments, fieldType === "vendor_document");
  switch (fieldType) {
    case "long_text":
      return <span style={{ whiteSpace: "pre-wrap" }}>{answer.text || null}</span>;
    case "short_text":
    case "email":
    case "url":
    case "telephone":
      return answer.text || valueText(answer) || null;
    case "integer":
    case "decimal":
    case "percentage":
      return formatNumber(answer.text);
    case "currency":
      return formatCurrency(answer.text);
    case "date":
      return answer.text ? formatDate(answer.text) : null;
    case "yes_no":
      return yesNo(answer) ? <StatusBadge tone="neutral">{yesNo(answer)}</StatusBadge> : null;
    case "checkbox":
    case "attestation":
      return answer.text === "true" ? "Confirmed" : "Not confirmed";
    case "single_select":
      return answer.text || answer.values?.[0] || null;
    case "multi_select":
      return answer.values?.length ? chipList(answer.values) : null;
    case "file":
      return count > 0 ? <>{`${count} submitted ${count === 1 ? "document" : "documents"}`}{metadata ? ` · ${metadata}` : ""}</> : null;
    case "photo":
      return count > 0 ? <>{`Photo attached${metadata ? ` · ${metadata}` : ""}`}</> : null;
    case "signature":
      return count > 0 ? "Signed" : null;
    case "vendor_document":
      return count > 0 ? <>{metadata || `${count} submitted ${count === 1 ? "document" : "documents"}`}</> : null;
    default:
      return answer.text || valueText(answer) || null;
  }
}

function valueText(answer: CaptureAnswerValue) { return answer.values?.length ? answer.values.join(", ") : answer.artifact_ids?.length ? `${answer.artifact_ids.length} submitted ${answer.artifact_ids.length === 1 ? "document" : "documents"}` : answer.document?.artifact_id ? "Submitted document" : ""; }

function evidenceCount(answer?: CaptureAnswerValue) { return answer?.artifact_ids?.length ?? (answer?.document?.artifact_id ? 1 : 0); }

function evidenceMetadata(answer: CaptureAnswerValue, attachments: { file_name: string }[], preferDocument = false) {
  if (answer?.document && (preferDocument || attachments.length === 0)) return [
    answer.document.document_type,
    answer.document.reference,
    answer.document.issued_by,
    answer.document.expires_on ? `expires ${formatDate(answer.document.expires_on)}` : "",
  ].filter(Boolean).join(" · ");
  const names = attachments.filter((attachment) => attachment.file_name).map((attachment) => attachment.file_name);
  if (names.length) return names.join(", ");
  return undefined;
}

function yesNo(answer: CaptureAnswerValue) {
  const raw = answer.text ?? answer.values?.[0];
  if (!raw) return undefined;
  const normalized = raw.trim().toLowerCase();
  if (["yes", "true", "1"].includes(normalized)) return "Yes";
  if (["no", "false", "0"].includes(normalized)) return "No";
  return raw;
}

function chipList(values: string[]): ReactNode {
  return <span className="answer-value-display__chips">{values.map((value) => <StatusBadge key={value} tone="neutral">{value}</StatusBadge>)}</span>;
}

function formatNumber(value?: string) {
  if (!value) return null;
  const parsed = Number(value);
  if (!Number.isFinite(parsed)) return value;
  return new Intl.NumberFormat(undefined, { maximumFractionDigits: 1 }).format(parsed);
}

function formatCurrency(value?: string) {
  if (!value) return null;
  const parsed = Number(value);
  if (!Number.isFinite(parsed)) return value;
  return new Intl.NumberFormat(undefined, { style: "currency", currency: "NGN" }).format(parsed);
}

export function formatDate(value: string) {
  const parsed = Date.parse(`${value}T00:00:00`);
  return Number.isFinite(parsed) ? new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(new Date(parsed)) : value;
}