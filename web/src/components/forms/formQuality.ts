import type { FormTemplate } from "../../monitoringTypes";
import {
  draftFromTemplate,
  fileType,
  isSelectionType,
  normalizeOptionText,
  type AuthoringField,
  type FormDraft,
  type FormQualityIssue,
} from "./formAuthoring";
export type { FormQualityIssue } from "./formAuthoring";

export function evaluateQuality(draft: FormDraft): FormQualityIssue[] {
  const issues: FormQualityIssue[] = [];
  const block = (id: string, message: string, extra: Partial<FormQualityIssue> = {}) => issues.push({ id, message, blocking: true, ...extra });

  if (!draft.code.trim()) block("code", "Add a form code.");
  if (!draft.name.trim()) block("name", "Add a form name.");
  if (!draft.purpose.trim()) block("purpose", "Add a clear form purpose.");
  if (!draft.sections.length || draft.sections.length > 20) block("sections", "A form must contain 1–20 sections.");
  if (!draft.fields.length || draft.fields.length > 200) block("fields", "A form must contain 1–200 questions.");

  const sectionIDs = new Set<string>();
  for (const section of draft.sections) {
    if (!section.id.trim() || !section.title.trim()) block(`section:${section.id}`, "Every section requires a title.", { sectionID: section.id });
    if (sectionIDs.has(section.id)) block(`section-duplicate:${section.id}`, `Section key ${section.id} is duplicated.`, { sectionID: section.id });
    sectionIDs.add(section.id);
  }

  const fieldIDs = new Set<string>();
  const fieldIndex = new Map<string, number>();
  draft.fields.forEach((field, index) => fieldIndex.set(field.id, index));
  for (const [index, field] of draft.fields.entries()) {
    if (!field.label.trim()) block(`field-label:${field.id}`, `Question ${index + 1} requires text.`, { fieldID: field.id });
    if (fieldIDs.has(field.id)) block(`field-duplicate:${field.id}`, `Question key ${field.id} is duplicated.`, { fieldID: field.id });
    fieldIDs.add(field.id);
    if (!sectionIDs.has(field.section_id ?? "")) block(`field-section:${field.id}`, `${displayField(field, index)} references a missing section.`, { fieldID: field.id });

    if (isSelectionType(field.type)) {
      const options = field.type === "yes_no" ? ["Yes", "No"] : normalizeOptionText((field.options ?? []).join("\n"));
      if (options.length < 2 || options.length > 50) block(`field-options:${field.id}`, `${displayField(field, index)} requires 2–50 unique choices.`, { fieldID: field.id });
      if (field.scoring) {
        if (field.scoring.weight < 1 || field.scoring.weight > 100) block(`field-weight:${field.id}`, `${displayField(field, index)} requires a score weight from 1–100%.`, { fieldID: field.id });
        for (const option of options) {
          const score = field.scoring.answer_scores[option];
          if (score === undefined || score < 0 || score > 100) block(`field-score:${field.id}:${option}`, `${displayField(field, index)} needs a score from 0–100 for “${option}”.`, { fieldID: field.id });
        }
      }
    } else if (field.scoring) {
      block(`field-scoring-type:${field.id}`, `${displayField(field, index)} cannot be scored with this response type.`, { fieldID: field.id });
    }

    if (field.type === "attestation" && !field.attestation?.trim()) block(`attestation:${field.id}`, `${displayField(field, index)} requires a statement to confirm.`, { fieldID: field.id });
    if (fileType(field.type)) validateFile(field, index, block);
    validateLimits(field, index, block);

    if (field.condition) {
      const sourceIndex = fieldIndex.get(field.condition.field_id);
      if (sourceIndex === undefined || sourceIndex >= index) block(`condition-order:${field.id}`, `${displayField(field, index)} must depend on an earlier question.`, { fieldID: field.id });
      if (field.condition.operator !== "ANSWERED" && !(field.condition.values ?? []).some((value) => value.trim())) block(`condition-value:${field.id}`, `${displayField(field, index)} requires a display-rule value.`, { fieldID: field.id });
    }

    if ((field.collection_intent ?? "CAPTURE") !== "CAPTURE" && (!field.record_target?.key.trim() || !field.record_target?.required_subject_type.trim())) {
      block(`record-target:${field.id}`, `${displayField(field, index)} requires a record target for this collection purpose.`, { fieldID: field.id });
    }
    if (field.collection_intent === "REPLACE_HELD_DOCUMENT" && field.type !== "file" && field.type !== "vendor_document") {
      block(`replace-document:${field.id}`, `${displayField(field, index)} must use File or Vendor document when replacing a held document.`, { fieldID: field.id });
    }
  }

  if (draft.scoringMode === "NONE" && draft.fields.some((field) => field.scoring)) block("scoring-none", "Remove scored questions or choose a scoring mode.");
  if (draft.scoringMode === "RISK" && draft.sections.some((section) => (section.weight ?? 0) !== 0)) block("risk-section-weight", "Section weights are only valid for compliance scoring.");
  if (draft.scoringMode === "COMPLIANCE") validateCompliance(draft, block);
  return issues;
}

export function isTemplateApprovalReady(template: FormTemplate) {
  return !evaluateQuality(draftFromTemplate(template)).some((issue) => issue.blocking);
}

function validateCompliance(draft: FormDraft, block: (id: string, message: string, extra?: Partial<FormQualityIssue>) => void) {
  const scoredSections = draft.sections.filter((section) => draft.fields.some((field) => field.section_id === section.id && field.scoring));
  if (!scoredSections.length) {
    block("compliance-scored-fields", "Compliance scoring requires at least one scored question.");
    return;
  }
  let sectionTotal = 0;
  for (const section of draft.sections) {
    const scoredFields = draft.fields.filter((field) => field.section_id === section.id && field.scoring);
    if (!scoredFields.length) {
      if ((section.weight ?? 0) !== 0) block(`compliance-unscored-section:${section.id}`, `${section.title || "Unscored section"} cannot carry compliance weight without scored questions.`, { sectionID: section.id });
      continue;
    }
    const fieldTotal = scoredFields.reduce((sum, field) => sum + (field.scoring?.weight ?? 0), 0);
    if (fieldTotal !== 100) {
      const delta = 100 - fieldTotal;
      block(
        `compliance-fields:${section.id}`,
        delta > 0 ? `${delta}% remains to allocate in ${section.title || "this section"}` : `${Math.abs(delta)}% is over-allocated in ${section.title || "this section"}`,
        { sectionID: section.id },
      );
    }
    const weight = section.weight ?? 0;
    if (weight < 1 || weight > 100) block(`compliance-section:${section.id}`, `${section.title || "Scored section"} requires a section weight from 1–100%.`, { sectionID: section.id });
    sectionTotal += weight;
  }
  if (sectionTotal !== 100) {
    const delta = 100 - sectionTotal;
    block("compliance-section-total", delta > 0 ? `${delta}% remains to allocate across scored sections` : `${Math.abs(delta)}% is over-allocated across scored sections`);
  }
}

function validateFile(field: AuthoringField, index: number, block: (id: string, message: string, extra?: Partial<FormQualityIssue>) => void) {
  if (!(field.accepted_formats?.length)) block(`file-formats:${field.id}`, `${displayField(field, index)} requires at least one approved file type.`, { fieldID: field.id });
  const { min_files, max_files, max_file_bytes, max_total_file_bytes } = field.constraints ?? {};
  if (min_files !== undefined && max_files !== undefined && min_files > max_files) block(`file-count:${field.id}`, `${displayField(field, index)} has a minimum file count above its maximum.`, { fieldID: field.id });
  if (max_file_bytes !== undefined && (max_file_bytes < 1 || max_file_bytes > 100 * 1024 * 1024)) block(`file-size:${field.id}`, `${displayField(field, index)} must limit each file to at most 100 MB.`, { fieldID: field.id });
  if (max_total_file_bytes !== undefined && (max_total_file_bytes < 1 || max_total_file_bytes > 500 * 1024 * 1024)) block(`file-total:${field.id}`, `${displayField(field, index)} must keep the combined upload limit at or below 500 MB.`, { fieldID: field.id });
  if (max_file_bytes !== undefined && max_total_file_bytes !== undefined && max_total_file_bytes < max_file_bytes) block(`file-total-small:${field.id}`, `${displayField(field, index)} has a combined limit below its per-file limit.`, { fieldID: field.id });
  if (["photo", "signature", "vendor_document"].includes(field.type) && max_files !== undefined && max_files !== 1) block(`single-file:${field.id}`, `${displayField(field, index)} accepts exactly one file.`, { fieldID: field.id });
}

function validateLimits(field: AuthoringField, index: number, block: (id: string, message: string, extra?: Partial<FormQualityIssue>) => void) {
  const constraints = field.constraints ?? {};
  if (constraints.min_length !== undefined && constraints.max_length !== undefined && constraints.min_length > constraints.max_length) block(`text-range:${field.id}`, `${displayField(field, index)} has a minimum length above its maximum.`, { fieldID: field.id });
  if (constraints.minimum !== undefined && constraints.maximum !== undefined && constraints.minimum > constraints.maximum) block(`number-range:${field.id}`, `${displayField(field, index)} has a minimum value above its maximum.`, { fieldID: field.id });
  if (constraints.min_date && constraints.max_date && constraints.min_date > constraints.max_date) block(`date-range:${field.id}`, `${displayField(field, index)} has an earliest date after its latest date.`, { fieldID: field.id });
  if (constraints.min_selections !== undefined && constraints.max_selections !== undefined && constraints.min_selections > constraints.max_selections) block(`selection-range:${field.id}`, `${displayField(field, index)} has a minimum selection count above its maximum.`, { fieldID: field.id });
}

function displayField(field: AuthoringField, index: number) {
  return field.label.trim() || `Question ${index + 1}`;
}
