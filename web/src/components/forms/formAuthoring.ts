import type { FormScoringMode, ReusableFormTemplateRef } from "../../formsTypes";
import type { CreateFormTemplateInput, FormFieldType, FormTemplate as MonitoringFormTemplate, FormTemplateField, FormTemplateSection } from "../../monitoringTypes";
import type { CaptureFieldConstraints, CapturePresentationMode, CaptureVisibilityCondition } from "../../types";

export type AuthoringSection = FormTemplateSection;
export type AuthoringField = Omit<FormTemplateField, "type"> & { type: FormFieldType };
export type FormQualityIssue = { id: string; message: string; blocking: boolean; sectionID?: string; fieldID?: string };
export type FormDraft = {
  code: string;
  name: string;
  purpose: string;
  scoringMode: FormScoringMode;
  presentation: CapturePresentationMode;
  allowModeSwitch: boolean;
  sections: AuthoringSection[];
  fields: AuthoringField[];
};

export const fieldTypes: Array<{ value: FormFieldType; label: string }> = [
  { value: "short_text", label: "Short answer" }, { value: "long_text", label: "Long answer" },
  { value: "email", label: "Email address" }, { value: "telephone", label: "Telephone number" },
  { value: "url", label: "Web address" }, { value: "integer", label: "Whole number" },
  { value: "decimal", label: "Decimal number" }, { value: "percentage", label: "Percentage" },
  { value: "currency", label: "Currency amount" }, { value: "date", label: "Date" },
  { value: "yes_no", label: "Yes or No" }, { value: "single_select", label: "Select one" },
  { value: "multi_select", label: "Select several" }, { value: "checkbox", label: "Checkbox" },
  { value: "attestation", label: "Attestation" }, { value: "file", label: "File" },
  { value: "photo", label: "Photo" }, { value: "signature", label: "Signature" },
  { value: "vendor_document", label: "Vendor document" },
];

export const approvedFormats = [
  { value: "application/pdf", label: "PDF" }, { value: "image/png", label: "PNG image" },
  { value: "image/jpeg", label: "JPEG image" }, { value: "text/plain", label: "Text file" },
  { value: "text/csv", label: "CSV file" },
  { value: "application/vnd.openxmlformats-officedocument.wordprocessingml.document", label: "Word document" },
  { value: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", label: "Excel workbook" },
] as const;

export function newDraft(): FormDraft {
  return {
    code: "", name: "", purpose: "", scoringMode: "NONE", presentation: "AUTOMATIC", allowModeSwitch: false,
    sections: [{ id: "section_1", title: "Questions" }], fields: [blankField(1, "section_1")],
  };
}

export function draftFromTemplate(template?: MonitoringFormTemplate): FormDraft {
  if (!template) return newDraft();
  return {
    code: template.code, name: template.name, purpose: template.purpose,
    scoringMode: template.scoring_mode ?? inferScoringMode(template.fields),
    presentation: template.presentation?.default_mode ?? "AUTOMATIC",
    allowModeSwitch: template.presentation?.allow_mode_switch ?? false,
    sections: (template.sections?.length ? template.sections : [{ id: "section_1", title: "Questions" }]).map(cloneSection),
    fields: template.fields.map((field) => cloneField(field as AuthoringField)),
  };
}

export function blankField(index: number, sectionID: string, type: FormFieldType = "short_text"): AuthoringField {
  return {
    id: `question_${index}`, section_id: sectionID, label: "", type, required: true,
    options: type === "yes_no" ? ["Yes", "No"] : selectionType(type) ? ["Option 1", "Option 2"] : undefined,
    accepted_formats: initialFormats(type), constraints: initialConstraints(type), collection_intent: "CAPTURE",
    browser_cache_policy: "ALLOWED",
  };
}

export function normalizeOptionText(value: string): string[] {
  const seen = new Set<string>();
  const values: string[] = [];
  for (const raw of value.split(/[\r\n\t]+/)) {
    const option = raw.trim();
    if (!option) continue;
    const key = option.toLocaleLowerCase();
    if (seen.has(key)) continue;
    seen.add(key);
    values.push(option);
    if (values.length === 50) break;
  }
  return values;
}

export function changeFieldType(field: AuthoringField, type: FormFieldType, scoringMode: FormScoringMode): AuthoringField {
  const options = type === "yes_no" ? ["Yes", "No"] : selectionType(type)
    ? field.options?.length && field.type !== "yes_no" ? uniqueOptions(field.options) : ["Option 1", "Option 2"]
    : undefined;
  const next: AuthoringField = {
    ...field, type, options, accepted_formats: initialFormats(type),
    attestation: type === "attestation" ? field.attestation ?? "" : undefined,
    constraints: initialConstraints(type), scoring: selectionType(type) && scoringMode !== "NONE" && field.scoring
      ? scoringForOptions(field.scoring.weight, options ?? [], scoringMode, field.scoring)
      : undefined,
  };
  if (next.collection_intent === "REPLACE_HELD_DOCUMENT" && type !== "file" && type !== "vendor_document") {
    next.collection_intent = "CAPTURE";
    next.record_target = undefined;
  }
  return next;
}

export function enableScoring(field: AuthoringField, scoringMode: FormScoringMode): AuthoringField {
  if (scoringMode === "NONE" || !selectionType(field.type)) return { ...field, scoring: undefined };
  const options = field.type === "yes_no" ? ["Yes", "No"] : uniqueOptions(field.options ?? []);
  return { ...field, scoring: scoringForOptions(field.scoring?.weight ?? 100, options, scoringMode, field.scoring) };
}

export function scoringForOptions(weight: number, options: string[], mode: FormScoringMode, previous?: FormTemplateField["scoring"]): NonNullable<FormTemplateField["scoring"]> {
  const answer_scores: Record<string, number> = {};
  options.forEach((option, index) => {
    const existing = previous?.answer_scores?.[option];
    answer_scores[option] = existing ?? (mode === "COMPLIANCE" ? (index === 0 ? 100 : 0) : (option === "No" ? 100 : 0));
  });
  return { weight: clamp(weight, 1, 100), answer_scores, critical_answers: (previous?.critical_answers ?? []).filter((answer) => options.includes(answer)) };
}

export function updateConstraint(field: AuthoringField, key: keyof CaptureFieldConstraints, value: number | string | undefined): AuthoringField {
  const constraints = { ...field.constraints };
  if (value === undefined || value === "") delete constraints[key];
  else Object.assign(constraints, { [key]: value });
  return { ...field, constraints };
}

export function duplicateSection(
  sectionID: string,
  sections: AuthoringSection[],
  fields: AuthoringField[],
  nextSection: number,
  nextField: number,
): { section: AuthoringSection; fields: AuthoringField[]; nextSection: number; nextField: number } | undefined {
  const source = sections.find((section) => section.id === sectionID);
  if (!source) return undefined;
  return cloneSectionWithFields(source, fields.filter((field) => field.section_id === sectionID), nextSection, nextField, true);
}

export function copySectionFromTemplate(
  template: MonitoringFormTemplate,
  sectionID: string,
  nextSection: number,
  nextField: number,
): { section: AuthoringSection; fields: AuthoringField[]; nextSection: number; nextField: number } | undefined {
  const source = template.sections?.find((section) => section.id === sectionID);
  if (!source) return undefined;
  return cloneSectionWithFields(source, template.fields.filter((field) => field.section_id === sectionID) as AuthoringField[], nextSection, nextField, false);
}

function cloneSectionWithFields(source: AuthoringSection, sourceFields: AuthoringField[], nextSection: number, nextField: number, markCopy: boolean) {
  const newSectionID = `section_${nextSection}`;
  const fieldIDs = new Map<string, string>();
  sourceFields.forEach((field, index) => fieldIDs.set(field.id, `question_${nextField + index}`));
  const rewriteCondition = (condition?: CaptureVisibilityCondition): CaptureVisibilityCondition | undefined => {
    if (!condition) return undefined;
    const field_id = fieldIDs.get(condition.field_id);
    return field_id ? { ...condition, field_id, values: condition.values ? [...condition.values] : undefined } : undefined;
  };
  return {
    section: { ...cloneSection(source), id: newSectionID, title: markCopy ? `${source.title} copy` : source.title, condition: rewriteCondition(source.condition) },
    fields: sourceFields.map((field) => {
      const id = fieldIDs.get(field.id)!;
      const scoring = field.scoring ? { ...field.scoring, id, answer_scores: { ...field.scoring.answer_scores }, critical_answers: [...(field.scoring.critical_answers ?? [])] } : undefined;
      return { ...cloneField(field), id, section_id: newSectionID, condition: rewriteCondition(field.condition), scoring };
    }),
    nextSection: nextSection + 1,
    nextField: nextField + sourceFields.length,
  };
}

export function buildCreateInput(draft: FormDraft): CreateFormTemplateInput {
  return {
    code: normalizedCode(draft.code), name: draft.name.trim(), purpose: draft.purpose.trim(), scoring_mode: draft.scoringMode,
    presentation: { default_mode: draft.presentation, allow_mode_switch: draft.allowModeSwitch },
    sections: draft.sections.map((section) => ({
      id: section.id.trim(), title: section.title.trim(), ...(section.help?.trim() ? { help: section.help.trim() } : {}),
      ...(section.weight ? { weight: section.weight } : {}), ...(section.condition ? { condition: cleanCondition(section.condition) } : {}),
    })),
    fields: draft.fields.map(cleanField),
  };
}

export function hasRequiredSignOff(fields: AuthoringField[]) {
  return fields.some((field) => field.required && (field.type === "attestation" || field.type === "signature"));
}

export function addRequiredSignOff(fields: AuthoringField[], sectionID: string, nextField: number) {
  const field = blankField(nextField, sectionID, "attestation");
  field.label = "Required sign-off";
  field.attestation = "I confirm that the information provided is complete and accurate to the best of my knowledge.";
  field.required = true;
  return { field, nextField: nextField + 1 };
}

export function reusableRefLabel(ref: ReusableFormTemplateRef) {
  return `${ref.name} · active v${ref.version}`;
}

export function maxGeneratedNumber(values: string[], prefix: string) {
  return values.reduce((max, value) => {
    const match = new RegExp(`^${prefix}_(\\d+)$`).exec(value);
    return Math.max(max, match ? Number(match[1]) : 0);
  }, 0) + 1;
}

function cleanField(field: AuthoringField): FormTemplateField {
  const options = selectionType(field.type) ? (field.type === "yes_no" ? ["Yes", "No"] : uniqueOptions(field.options ?? [])) : undefined;
  const scoring = field.scoring && options ? scoringForOptions(field.scoring.weight, options, "RISK", field.scoring) : undefined;
  return {
    id: field.id.trim(), section_id: field.section_id?.trim(), label: field.label.trim(), type: field.type, required: field.required,
    ...(field.description?.trim() ? { description: field.description.trim() } : {}), ...(options ? { options } : {}),
    ...(fileType(field.type) && field.accepted_formats?.length ? { accepted_formats: uniqueLower(field.accepted_formats) } : {}),
    ...(field.type === "attestation" && field.attestation?.trim() ? { attestation: field.attestation.trim() } : {}),
    ...(compactConstraints(field.constraints) ? { constraints: compactConstraints(field.constraints) } : {}),
    ...(field.condition ? { condition: cleanCondition(field.condition) } : {}), ...(scoring ? { scoring } : {}),
    ...(field.collection_intent && field.collection_intent !== "CAPTURE" ? { collection_intent: field.collection_intent } : {}),
    ...(field.record_target ? { record_target: { key: field.record_target.key.trim().toUpperCase(), required_subject_type: field.record_target.required_subject_type.trim().toUpperCase() } } : {}),
    ...(field.browser_cache_policy && field.browser_cache_policy !== "ALLOWED" ? { browser_cache_policy: field.browser_cache_policy } : {}),
  };
}

function cleanCondition(condition: CaptureVisibilityCondition): CaptureVisibilityCondition {
  return { field_id: condition.field_id, operator: condition.operator, ...(condition.operator === "ANSWERED" ? {} : { values: (condition.values ?? []).map((value) => value.trim()).filter(Boolean) }) };
}

function compactConstraints(value?: CaptureFieldConstraints) {
  if (!value) return undefined;
  const entries = Object.entries(value).filter(([, item]) => item !== undefined && item !== "");
  return entries.length ? Object.fromEntries(entries) as CaptureFieldConstraints : undefined;
}

function cloneSection(section: AuthoringSection): AuthoringSection {
  return { ...section, condition: section.condition ? { ...section.condition, values: section.condition.values ? [...section.condition.values] : undefined } : undefined };
}
function cloneField(field: AuthoringField): AuthoringField {
  return { ...field, options: field.options ? [...field.options] : undefined, accepted_formats: field.accepted_formats ? [...field.accepted_formats] : undefined, constraints: field.constraints ? { ...field.constraints } : undefined, condition: field.condition ? { ...field.condition, values: field.condition.values ? [...field.condition.values] : undefined } : undefined, scoring: field.scoring ? { ...field.scoring, answer_scores: { ...field.scoring.answer_scores }, critical_answers: [...(field.scoring.critical_answers ?? [])] } : undefined, record_target: field.record_target ? { ...field.record_target } : undefined };
}
function inferScoringMode(fields: FormTemplateField[]): FormScoringMode { return fields.some((field) => field.scoring) ? "RISK" : "NONE"; }
function normalizedCode(value: string) { return value.trim().toUpperCase().replace(/[^A-Z0-9]+/g, "-").replace(/^-|-$/g, ""); }
function uniqueOptions(values: string[]) { const seen = new Set<string>(); return values.map((value) => value.trim()).filter((value) => { const key = value.toLocaleLowerCase(); if (!value || seen.has(key)) return false; seen.add(key); return true; }).slice(0, 50); }
function uniqueLower(values: string[]) { return [...new Set(values.map((value) => value.trim().toLowerCase()).filter(Boolean))]; }
function selectionType(type: FormFieldType) { return ["yes_no", "single_select", "multi_select"].includes(type); }
export function fileType(type: FormFieldType) { return ["file", "photo", "signature", "vendor_document"].includes(type); }
export function textType(type: FormFieldType) { return ["short_text", "long_text", "email", "telephone", "url"].includes(type); }
export function numericType(type: FormFieldType) { return ["integer", "decimal", "percentage", "currency"].includes(type); }
export function isSelectionType(type: FormFieldType) { return selectionType(type); }
function initialFormats(type: FormFieldType) { if (type === "photo") return ["image/jpeg", "image/png"]; if (type === "signature") return ["image/png"]; if (type === "file" || type === "vendor_document") return ["application/pdf"]; return undefined; }
function initialConstraints(type: FormFieldType): CaptureFieldConstraints | undefined { if (type === "currency") return { currency: "NGN" }; if (type === "percentage") return { minimum: 0, maximum: 100 }; if (["photo", "signature", "vendor_document", "file"].includes(type)) return { max_files: 1 }; return undefined; }
function clamp(value: number, min: number, max: number) { return Math.min(max, Math.max(min, Number.isFinite(value) ? value : min)); }
