import type { CaptureAnswers, CaptureField, CaptureRequest } from "../../types";

export function initialSourceAnswers(request: CaptureRequest): CaptureAnswers {
  const answers: CaptureAnswers = {};
  for (const field of request.fields) {
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

function sourceBinding(field: CaptureField, mode: string) {
  const reference = field.bindings?.find((candidate) => candidate.mode === mode);
  if (!reference) return null;
  const resolution = field.source_resolutions?.find((candidate) => candidate.mode === mode && candidate.binding_id === reference.binding_id && candidate.binding_version === reference.binding_version);
  return resolution ? { reference, resolution } : null;
}
