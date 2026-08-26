import type {
  CaptureAnswerValue,
  CaptureAnswers,
  CaptureField,
  CaptureFieldType,
  CaptureFormContract,
  CapturePresentationMode,
  CaptureRequest,
  CaptureSection,
  CaptureVisibilityCondition,
} from "../../types";

const supportedTypes = new Set<CaptureFieldType>([
  "short_text", "long_text", "email", "telephone", "url", "integer", "decimal", "percentage", "currency", "date",
  "yes_no", "single_select", "multi_select", "checkbox", "attestation", "file", "photo", "signature", "vendor_document",
]);

export function normalizeFieldType(value: string): CaptureFieldType | null {
  const type = value.trim().toLowerCase();
  if (type === "text") return "short_text";
  if (type === "number") return "decimal";
  return supportedTypes.has(type as CaptureFieldType) ? type as CaptureFieldType : null;
}

export function captureContract(request: CaptureRequest): CaptureFormContract {
  const sections = request.sections?.length
    ? request.sections
    : [{ id: "general", title: "Response" }];
  const sectionIDs = new Set(sections.map((section) => section.id));
  const fallbackSectionID = sections[0]?.id ?? "general";
  return {
    presentation: request.presentation ?? { default_mode: "AUTOMATIC", allow_mode_switch: false },
    sections,
    fields: request.fields.map((field) => ({
      ...field,
      section_id: field.section_id && sectionIDs.has(field.section_id) ? field.section_id : fallbackSectionID,
    })),
  };
}

export function answerText(value?: CaptureAnswerValue): string {
  return value?.text ?? "";
}

export function answerValues(value?: CaptureAnswerValue): string[] {
  return value?.values ?? [];
}

export function answerIsPresent(value?: CaptureAnswerValue): boolean {
  return Boolean(
    value?.text?.trim()
    || value?.values?.some((item) => item.trim())
    || value?.artifact_ids?.length
    || value?.document,
  );
}

export function conditionMatches(condition: CaptureVisibilityCondition | undefined, answers: CaptureAnswers): boolean {
  if (!condition) return true;
  const answer = answers[condition.field_id];
  if (condition.operator === "ANSWERED") return answerIsPresent(answer);
  const actual = answer?.text?.trim() ? [answer.text.trim()] : answerValues(answer).map((value) => value.trim()).filter(Boolean);
  const expected = condition.values ?? [];
  const matches = actual.some((value) => expected.includes(value));
  if (condition.operator === "EQUALS" || condition.operator === "IN") return answer !== undefined && matches;
  return answer === undefined || !matches;
}

export type VisibleCaptureSection = CaptureSection & { fields: CaptureField[] };

export function visibleCaptureSections(contract: CaptureFormContract, answers: CaptureAnswers): VisibleCaptureSection[] {
  return contract.sections.flatMap((section) => {
    if (!conditionMatches(section.condition, answers)) return [];
    const fields = contract.fields.filter((field) => field.section_id === section.id && conditionMatches(field.condition, answers));
    return fields.length ? [{ ...section, fields }] : [];
  });
}

export function visibleCaptureFields(contract: CaptureFormContract, answers: CaptureAnswers): CaptureField[] {
  return visibleCaptureSections(contract, answers).flatMap((section) => section.fields);
}

export function keepVisibleAnswers(contract: CaptureFormContract, answers: CaptureAnswers): CaptureAnswers {
  const visibleIDs = new Set(visibleCaptureFields(contract, answers).map((field) => field.id));
  return Object.fromEntries(Object.entries(answers).filter(([fieldID]) => visibleIDs.has(fieldID)));
}

export function effectivePresentationMode(contract: CaptureFormContract, answers: CaptureAnswers, requested: CapturePresentationMode): "CLASSIC" | "WIZARD" {
  if (requested !== "AUTOMATIC") return requested;
  const sections = visibleCaptureSections(contract, answers);
  const hasConditionalSection = contract.sections.some((section) => Boolean(section.condition));
  return sections.length > 2 || sections.reduce((count, section) => count + section.fields.length, 0) > 12 || hasConditionalSection ? "WIZARD" : "CLASSIC";
}

export type CaptureValidationError = { fieldID: string; message: string };

export function validateCaptureFields(fields: CaptureField[], answers: CaptureAnswers): CaptureValidationError[] {
  const errors: CaptureValidationError[] = [];
  for (const field of fields) {
    const value = answers[field.id];
    const type = normalizeFieldType(field.type);
    if (field.required && !answerIsPresent(value)) {
      errors.push({ fieldID: field.id, message: `${field.label} is required` });
      continue;
    }
    if (!answerIsPresent(value) || !type) continue;
    const text = answerText(value).trim();
    const constraints = field.constraints ?? {};
    if (text && constraints.min_length !== undefined && text.length < constraints.min_length) errors.push({ fieldID: field.id, message: `${field.label} must contain at least ${constraints.min_length} characters` });
    if (text && constraints.max_length !== undefined && text.length > constraints.max_length) errors.push({ fieldID: field.id, message: `${field.label} must contain no more than ${constraints.max_length} characters` });
    if (type === "email" && text && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(text)) errors.push({ fieldID: field.id, message: `${field.label} must be a valid email address` });
    if (type === "telephone" && text && !/^[+()0-9\s.-]{7,30}$/.test(text)) errors.push({ fieldID: field.id, message: `${field.label} must be a valid telephone number` });
    if (type === "url" && text && !validHTTPURL(text)) errors.push({ fieldID: field.id, message: `${field.label} must be a valid web address` });
    if (["integer", "decimal", "percentage", "currency"].includes(type) && text) {
      const number = Number(text);
      if (!Number.isFinite(number)) errors.push({ fieldID: field.id, message: `${field.label} must be a number` });
      else if (type === "integer" && !Number.isInteger(number)) errors.push({ fieldID: field.id, message: `${field.label} must be a whole number` });
      else if (constraints.minimum !== undefined && number < constraints.minimum) errors.push({ fieldID: field.id, message: `${field.label} must be at least ${constraints.minimum}` });
      else if (constraints.maximum !== undefined && number > constraints.maximum) errors.push({ fieldID: field.id, message: `${field.label} must be no more than ${constraints.maximum}` });
      else if (type === "percentage" && number < (constraints.minimum ?? 0)) errors.push({ fieldID: field.id, message: `${field.label} must be at least ${constraints.minimum ?? 0}` });
      else if (type === "percentage" && number > (constraints.maximum ?? 100)) errors.push({ fieldID: field.id, message: `${field.label} must be no more than ${constraints.maximum ?? 100}` });
      else if (constraints.decimal_precision !== undefined && decimalPlaces(text) > constraints.decimal_precision) errors.push({ fieldID: field.id, message: `${field.label} allows ${constraints.decimal_precision} decimal places` });
    }
    if (type === "date" && text && constraints.min_date && text < constraints.min_date) errors.push({ fieldID: field.id, message: `${field.label} must be on or after ${constraints.min_date}` });
    if (type === "date" && text && constraints.max_date && text > constraints.max_date) errors.push({ fieldID: field.id, message: `${field.label} must be on or before ${constraints.max_date}` });
    const selections = answerValues(value).length;
    if (constraints.min_selections !== undefined && selections < constraints.min_selections) errors.push({ fieldID: field.id, message: `${field.label} requires at least ${constraints.min_selections} selections` });
    if (constraints.max_selections !== undefined && selections > constraints.max_selections) errors.push({ fieldID: field.id, message: `${field.label} allows no more than ${constraints.max_selections} selections` });
	const fileCount = type === "vendor_document" ? (value?.document?.artifact_id ? 1 : 0) : value?.artifact_ids?.length ?? 0;
	if (constraints.min_files !== undefined && (field.required || fileCount > 0) && fileCount < constraints.min_files) errors.push({ fieldID: field.id, message: `${field.label} requires at least ${constraints.min_files} files` });
	if (constraints.max_files !== undefined && fileCount > constraints.max_files) errors.push({ fieldID: field.id, message: `${field.label} allows no more than ${constraints.max_files} files` });
    if (type === "vendor_document" && value?.document && (!value.document.artifact_id || !value.document.document_type.trim())) errors.push({ fieldID: field.id, message: `${field.label} requires a file and document type` });
  }
  return errors;
}

function validHTTPURL(value: string) {
  try {
    const parsed = new URL(value);
    return parsed.protocol === "https:" || parsed.protocol === "http:";
  } catch {
    return false;
  }
}

function decimalPlaces(value: string) {
  return value.includes(".") ? value.split(".", 2)[1]?.length ?? 0 : 0;
}
