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
import { validateAdvancedScoreProfile, validateFieldContractBounds, validateSectionContractBounds } from "./formContractQuality";
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
    validateSectionContractBounds(section, block);
  }

  const fieldIDs = new Set<string>();
  const fieldIndex = new Map<string, number>();
  draft.fields.forEach((field, index) => fieldIndex.set(field.id, index));
  for (const [index, field] of draft.fields.entries()) {
    if (!field.label.trim()) block(`field-label:${field.id}`, `Question ${index + 1} requires text.`, { fieldID: field.id });
    if (fieldIDs.has(field.id)) block(`field-duplicate:${field.id}`, `Question key ${field.id} is duplicated.`, { fieldID: field.id });
    fieldIDs.add(field.id);
    if (!sectionIDs.has(field.section_id ?? "")) block(`field-section:${field.id}`, `${displayField(field, index)} references a missing section.`, { fieldID: field.id });

    validateFieldContractBounds(field, index, block);

    if (isSelectionType(field.type)) {
      const options = field.type === "yes_no" ? ["Yes", "No"] : normalizeOptionText((field.options ?? []).join("\n"));
      if (options.length < 2 || options.length > 50) block(`field-options:${field.id}`, `${displayField(field, index)} requires 2–50 unique choices.`, { fieldID: field.id });
      if (field.scoring) {
        for (const option of options) {
          if (field.scoring.answer_scores[option] === undefined) block(`field-score:${field.id}:${option}`, `${displayField(field, index)} needs a score from 0–100 for “${option}”.`, { fieldID: field.id });
        }
      }
    } else if (field.scoring) {
      block(`field-scoring-type:${field.id}`, `${displayField(field, index)} cannot be scored with this response type.`, { fieldID: field.id });
    }

    if (field.type === "attestation" && !field.attestation?.trim()) block(`attestation:${field.id}`, `${displayField(field, index)} requires a statement to confirm.`, { fieldID: field.id });
    if (fileType(field.type) && !(field.accepted_formats?.length)) block(`file-formats:${field.id}`, `${displayField(field, index)} requires at least one approved file type.`, { fieldID: field.id });

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

  if (draft.scoringMode === "NONE" && (draft.fields.some((field) => field.scoring) || draft.scoreProfile)) block("scoring-none", "Remove scoring rules or choose a scoring mode.");
  if (draft.scoreProfile && draft.scoreProfile.mode !== draft.scoringMode) block("score-profile-mode", "The advanced score profile must use the selected scoring mode.");
  validateAdvancedScoreProfile(draft.scoreProfile, draft.fields, block);
  if (draft.scoringMode === "RISK" && draft.sections.some((section) => (section.weight ?? 0) !== 0)) block("risk-section-weight", "Section weights are only valid for compliance scoring.");
  if (draft.scoringMode === "COMPLIANCE") validateCompliance(draft, block);
  return issues;
}

// Mirrors formcontract.NormalizeDraft: COMPLIANCE drafts retain full structural,
// field, condition and per-answer validation while deferring only allocation
// completeness until approval. Reusing evaluateQuality keeps one client model.
export function evaluateDraftValidity(draft: FormDraft): FormQualityIssue[] {
  if (draft.scoringMode !== "COMPLIANCE") return evaluateQuality(draft);
  const issues = evaluateQuality({
    ...draft,
    scoringMode: "RISK",
    sections: draft.sections.map((section) => ({ ...section, weight: undefined })),
  });
  const block = (id: string, message: string, extra: Partial<FormQualityIssue> = {}) => issues.push({ id, message, blocking: true, ...extra });
  for (const section of draft.sections) validateSectionContractBounds(section, block);
  return issues;
}

export function isTemplateApprovalReady(template: FormTemplate) {
  return !evaluateQuality(draftFromTemplate(template)).some((issue) => issue.blocking);
}

function validateCompliance(draft: FormDraft, block: (id: string, message: string, extra?: Partial<FormQualityIssue>) => void) {
  const scoredSections = draft.sections.filter((section) => draft.fields.some((field) => field.section_id === section.id && field.scoring));
  if (draft.scoreProfile && !scoredSections.length) {
    for (const section of draft.sections) if ((section.weight ?? 0) !== 0) block(`compliance-profile-section:${section.id}`, `${section.title || "Section"} cannot carry a legacy section weight when advanced scoring is used.`, { sectionID: section.id });
    return;
  }
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
    if (weight === 0) block(`compliance-section:${section.id}`, `${section.title || "Scored section"} requires a section weight from 1–100%.`, { sectionID: section.id });
    sectionTotal += weight;
  }
  if (sectionTotal !== 100) {
    const delta = 100 - sectionTotal;
    block("compliance-section-total", delta > 0 ? `${delta}% remains to allocate across scored sections` : `${Math.abs(delta)}% is over-allocated across scored sections`);
  }
}

function displayField(field: AuthoringField, index: number) {
  return field.label.trim() || `Question ${index + 1}`;
}
