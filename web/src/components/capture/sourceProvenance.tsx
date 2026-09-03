import type { CaptureAnswers, CaptureField, CaptureRequest } from "../../types";
import { normalizeFieldType } from "./contract";

export function initialSourceAnswers(request: CaptureRequest): CaptureAnswers {
  const answers: CaptureAnswers = {};
  for (const field of request.fields) {
    const previous = reusablePreviousResponse(request, field);
    if (previous) answers[field.id] = { text: previous.value };
    const prefill = sourceBinding(field, "PREFILL");
    const scalar = prefill?.resolution.state === "CURRENT" ? prefill.resolution.value : undefined;
    if (scalar?.kind !== "NULL" && scalar?.text) answers[field.id] = { text: scalar.text };
  }
  return answers;
}

export function CaptureFieldSourceNotice({ field, value }: { field: CaptureField; value: string }) {
  const prefill = sourceBinding(field, "PREFILL");
  const sourceValue = prefill?.resolution.state === "CURRENT" ? prefill.resolution.value?.text?.trim() ?? "" : "";
  if (prefill && sourceValue) {
    const corrected = value.trim() !== "" && value.trim() !== sourceValue;
    return <p className={corrected ? "source-origin corrected" : "source-origin"}><span aria-hidden="true">{corrected ? "↺" : "↳"}</span>{corrected ? `Corrected by you · source value was ${sourceValue}` : `Prefilled from ${prefill.resolution.binding_name}`}{prefill.resolution.receipt?.observed_at ? ` · observed ${new Date(prefill.resolution.receipt.observed_at).toLocaleString()}` : ""}</p>;
  }
  const options = sourceBinding(field, "OPTIONS");
  if (options?.resolution.state === "CURRENT") return <p className="source-origin"><span aria-hidden="true">↳</span>{`Choices from ${options.resolution.binding_name}`}{options.resolution.receipt?.observed_at ? ` · observed ${new Date(options.resolution.receipt.observed_at).toLocaleString()}` : ""}</p>;
  return null;
}

export function reviewSourceLabel(field: CaptureField, value?: string): string | null {
  const prefill = sourceBinding(field, "PREFILL");
  const sourceValue = prefill?.resolution.state === "CURRENT" ? prefill.resolution.value?.text?.trim() ?? "" : "";
  if (!prefill || !sourceValue) return null;
  return (value ?? "").trim() === sourceValue ? `Source-prefilled · ${prefill.resolution.binding_name}` : `Respondent correction · source value: ${sourceValue}`;
}

export function CaptureFieldPreviousResponseNotice({ request, field, value }: { request: CaptureRequest; field: CaptureField; value: string }) {
  if (currentSourceValue(field)) return null;
  const previous = reusablePreviousResponse(request, field);
  if (!previous) return null;
  const changed = value.trim() !== previous.value.trim();
  return <p className={changed ? "source-origin previous-response corrected" : "source-origin previous-response"}><span aria-hidden="true">{changed ? "↺" : "↳"}</span>{changed ? `Changed by you · previous response was ${previous.value}` : `From the response submitted on ${formatPreviousResponseDate(previous.previous_submitted_at)}`}</p>;
}

export function reviewProvenanceLabel(request: CaptureRequest, field: CaptureField, value?: string): string | null {
  const source = reviewSourceLabel(field, value);
  if (source) return source;
  const previous = reusablePreviousResponse(request, field);
  if (!previous) return null;
  return (value ?? "").trim() === previous.value.trim()
    ? `Previous response · submitted ${formatPreviousResponseDate(previous.previous_submitted_at)}`
    : `Respondent correction · previous response: ${previous.value}`;
}

function currentSourceValue(field: CaptureField) {
  const prefill = sourceBinding(field, "PREFILL");
  return prefill?.resolution.state === "CURRENT" && Boolean(prefill.resolution.value?.text?.trim());
}

function reusablePreviousResponse(request: CaptureRequest, field: CaptureField) {
  const type = normalizeFieldType(field.type);
  if (!type || !["short_text", "long_text", "email", "telephone", "url", "integer", "decimal", "percentage", "currency", "single_select", "date", "yes_no"].includes(type)) return null;
  const previous = request.previous_responses?.[field.id];
  if (!previous?.value?.trim() || !previous.previous_submitted_at) return null;
  if (type === "single_select" && field.options?.length && !field.options.includes(previous.value)) return null;
  return previous;
}

function formatPreviousResponseDate(value: string) {
  const parsed = Date.parse(value);
  return Number.isFinite(parsed)
    ? new Intl.DateTimeFormat("en-GB", { day: "numeric", month: "short", year: "numeric", timeZone: "UTC" }).format(new Date(parsed))
    : "an earlier date";
}

function sourceBinding(field: CaptureField, mode: string) {
  const reference = field.bindings?.find((candidate) => candidate.mode === mode);
  if (!reference) return null;
  const resolution = field.source_resolutions?.find((candidate) => candidate.mode === mode && candidate.binding_id === reference.binding_id && candidate.binding_version === reference.binding_version);
  return resolution ? { reference, resolution } : null;
}
