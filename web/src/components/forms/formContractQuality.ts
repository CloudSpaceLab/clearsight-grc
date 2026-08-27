import { approvedFormats, fileType, isSelectionType, numericType, textType, type AuthoringField, type AuthoringSection, type FormQualityIssue } from "./formAuthoring";

type Block = (id: string, message: string, extra?: Partial<FormQualityIssue>) => void;

const approvedFormatValues = new Set<string>(approvedFormats.map((format) => format.value));
const maxFileBytes = 100 * 1024 * 1024;
const maxTotalFileBytes = 500 * 1024 * 1024;

export function validateSectionContractBounds(section: AuthoringSection, block: Block) {
  if (section.id.length > 80) block(`section-id:${section.id}`, "Section keys may contain at most 80 characters.", { sectionID: section.id });
  const weight = section.weight;
  if (weight !== undefined && (!Number.isInteger(weight) || weight < 0 || weight > 100)) {
    block(`section-weight:${section.id}`, `${section.title || "Section"} requires a whole-number weight from 0–100%.`, { sectionID: section.id });
  }
}

export function validateFieldContractBounds(field: AuthoringField, index: number, block: Block) {
  const label = displayField(field, index);
  if (field.id.length > 80) block(`field-id:${field.id}`, `${label} has a question key longer than 80 characters.`, { fieldID: field.id });

  if (field.record_target) {
    if (!validRecordTargetIdentifier(field.record_target.key, 200, true) || !validRecordTargetIdentifier(field.record_target.required_subject_type, 80, false)) {
      block(`record-target-format:${field.id}`, `${label} requires a valid bounded record target.`, { fieldID: field.id });
    }
  }

  if (field.attestation && (field.type !== "attestation" || field.attestation.length > 1000)) {
    block(`attestation-shape:${field.id}`, `${label} has invalid attestation text.`, { fieldID: field.id });
  }

  const options = field.type === "yes_no" ? ["Yes", "No"] : field.options ?? [];
  if (!isSelectionType(field.type) && options.length) block(`field-options-type:${field.id}`, `${label} cannot define choices for this response type.`, { fieldID: field.id });

  if (fileType(field.type)) {
    validateFileFormats(field, label, block);
  } else if (field.accepted_formats?.length) {
    block(`file-formats-type:${field.id}`, `${label} cannot define accepted file types.`, { fieldID: field.id });
  }

  validateConstraints(field, label, options.length, block);
  validateScoring(field, label, options, block);
}

function validateFileFormats(field: AuthoringField, label: string, block: Block) {
  for (const raw of field.accepted_formats ?? []) {
    const format = raw.split(";", 1)[0]?.trim().toLowerCase() ?? "";
    if (!approvedFormatValues.has(format)) {
      block(`file-format:${field.id}:${format}`, `${label} contains an unsupported file type.`, { fieldID: field.id });
      continue;
    }
    if (field.type === "photo" && format !== "image/png" && format !== "image/jpeg") {
      block(`photo-format:${field.id}:${format}`, `${label} accepts a non-image photo format.`, { fieldID: field.id });
    }
    if (field.type === "signature" && format !== "image/png") {
      block(`signature-format:${field.id}:${format}`, `${label} signatures must use PNG.`, { fieldID: field.id });
    }
  }
}

function validateConstraints(field: AuthoringField, label: string, optionCount: number, block: Block) {
  const c = field.constraints ?? {};
  if (textType(field.type)) {
    validateIntegerRange(c.min_length, c.max_length, 0, 5000, `text-bounds:${field.id}`, `${label} requires character limits from 0–5000.`, field.id, block);
  } else if (c.min_length !== undefined || c.max_length !== undefined) {
    block(`text-bounds-type:${field.id}`, `${label} cannot define character limits.`, { fieldID: field.id });
  }

  if (numericType(field.type)) {
    if (!finiteOrUndefined(c.minimum) || !finiteOrUndefined(c.maximum) || (c.minimum !== undefined && c.maximum !== undefined && c.minimum > c.maximum)) {
      block(`number-bounds:${field.id}`, `${label} has invalid numeric limits.`, { fieldID: field.id });
    }
    if (c.step !== undefined && (!Number.isFinite(c.step) || c.step <= 0)) block(`number-step:${field.id}`, `${label} requires a positive numeric step.`, { fieldID: field.id });
    if (c.decimal_precision !== undefined && (!Number.isInteger(c.decimal_precision) || c.decimal_precision < 0 || c.decimal_precision > 6)) {
      block(`decimal-precision:${field.id}`, `${label} requires 0–6 decimal places.`, { fieldID: field.id });
    }
  } else if (c.minimum !== undefined || c.maximum !== undefined || c.step !== undefined || c.decimal_precision !== undefined || c.currency) {
    block(`number-bounds-type:${field.id}`, `${label} cannot define numeric limits.`, { fieldID: field.id });
  }

  if (isSelectionType(field.type)) {
    const upper = field.type === "multi_select" ? optionCount : 1;
    validateIntegerRange(c.min_selections, c.max_selections, 0, upper, `selection-bounds:${field.id}`, `${label} has selection limits outside its available choices.`, field.id, block);
  } else if (c.min_selections !== undefined || c.max_selections !== undefined) {
    block(`selection-bounds-type:${field.id}`, `${label} cannot define selection limits.`, { fieldID: field.id });
  }

  if (fileType(field.type)) {
    validateIntegerRange(c.min_files, c.max_files, 0, 10, `file-count:${field.id}`, `${label} requires file counts from 0–10.`, field.id, block);
    if (c.max_files !== undefined && c.max_files < 1) block(`file-max-count:${field.id}`, `${label} must allow at least one file.`, { fieldID: field.id });
    if (c.max_file_bytes !== undefined && (!Number.isInteger(c.max_file_bytes) || c.max_file_bytes < 1 || c.max_file_bytes > maxFileBytes)) {
      block(`file-size:${field.id}`, `${label} must limit each file to at most 100 MB.`, { fieldID: field.id });
    }
    if (c.max_total_file_bytes !== undefined && (!Number.isInteger(c.max_total_file_bytes) || c.max_total_file_bytes < 1 || c.max_total_file_bytes > maxTotalFileBytes)) {
      block(`file-total:${field.id}`, `${label} must keep the combined upload limit at or below 500 MB.`, { fieldID: field.id });
    }
    if (c.max_file_bytes !== undefined && c.max_total_file_bytes !== undefined && c.max_total_file_bytes < c.max_file_bytes) {
      block(`file-total-small:${field.id}`, `${label} has a combined limit below its per-file limit.`, { fieldID: field.id });
    }
    if (["photo", "signature", "vendor_document"].includes(field.type)) {
      if (c.max_files !== undefined && c.max_files !== 1) block(`single-file:${field.id}`, `${label} accepts exactly one file.`, { fieldID: field.id });
      if (c.min_files !== undefined && c.min_files > 1) block(`single-file-min:${field.id}`, `${label} accepts exactly one file.`, { fieldID: field.id });
    }
  } else if (c.min_files !== undefined || c.max_files !== undefined || c.max_file_bytes !== undefined || c.max_total_file_bytes !== undefined) {
    block(`file-limits-type:${field.id}`, `${label} cannot define file limits.`, { fieldID: field.id });
  }
}

function validateScoring(field: AuthoringField, label: string, options: string[], block: Block) {
  if (!field.scoring) return;
  if (!Number.isInteger(field.scoring.weight) || field.scoring.weight < 1 || field.scoring.weight > 100) {
    block(`field-weight:${field.id}`, `${label} requires a whole-number score weight from 1–100%.`, { fieldID: field.id });
  }
  for (const [answer, score] of Object.entries(field.scoring.answer_scores)) {
    if (!options.includes(answer) || !Number.isInteger(score) || score < 0 || score > 100) {
      block(`field-score:${field.id}:${answer}`, `${label} contains an invalid answer score.`, { fieldID: field.id });
    }
  }
  for (const answer of field.scoring.critical_answers ?? []) {
    if (!options.includes(answer)) block(`field-critical:${field.id}:${answer}`, `${label} contains an invalid critical answer.`, { fieldID: field.id });
  }
}

function validateIntegerRange(minimum: number | undefined, maximum: number | undefined, lower: number, upper: number, id: string, message: string, fieldID: string, block: Block) {
  if ((minimum !== undefined && (!Number.isInteger(minimum) || minimum < lower || minimum > upper)) ||
      (maximum !== undefined && (!Number.isInteger(maximum) || maximum < lower || maximum > upper)) ||
      (minimum !== undefined && maximum !== undefined && minimum > maximum)) {
    block(id, message, { fieldID });
  }
}

function finiteOrUndefined(value: number | undefined) {
  return value === undefined || Number.isFinite(value);
}

function validRecordTargetIdentifier(raw: string, maximum: number, allowDot: boolean) {
  const value = raw.trim().toUpperCase();
  if (!value || value.length > maximum || value.startsWith(".") || value.endsWith(".") || value.includes("..")) return false;
  const pattern = allowDot ? /^[A-Z0-9_.]+$/ : /^[A-Z0-9_]+$/;
  return pattern.test(value);
}

function displayField(field: AuthoringField, index: number) {
  return field.label.trim() || `Question ${index + 1}`;
}
