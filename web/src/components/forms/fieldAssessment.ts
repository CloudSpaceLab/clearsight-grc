import type { FieldAssessmentMode, FormTemplateField } from "../../monitoringTypes";

export const assessmentModes = [
  { id: "NONE", label: "Not scored" },
  { id: "MANUAL", label: "Review" },
  { id: "AUTOMATIC", label: "Automatic rules" },
  { id: "AUTOMATIC_REVIEW", label: "Automatic rules, then review" },
] satisfies Array<{ id: FieldAssessmentMode; label: string }>;

export function needsBankReview(field: FormTemplateField) {
  return field.assessment?.mode === "MANUAL" || field.assessment?.mode === "AUTOMATIC_REVIEW";
}

export function fieldAssessmentLabel(field: FormTemplateField) {
  if (!field.assessment) return field.scoring ? "Automatic rules" : "";
  return assessmentModes.find((mode) => mode.id === field.assessment?.mode)?.label ?? "Assessment unavailable";
}

export function assessmentConfigurationErrors(field: FormTemplateField): string[] {
  const assessment = field.assessment;
  if (!assessment) return [];
  const errors: string[] = [];
  if (!assessmentModes.some((mode) => mode.id === assessment.mode)) errors.push("Choose how this field is assessed.");
  if (assessment.mode !== "NONE" && (!Number.isInteger(assessment.weight) || assessment.weight < 1 || assessment.weight > 100)) errors.push("Set an assessment weight from 1–100.");
  if ((assessment.mode === "NONE" || assessment.mode === "MANUAL") && field.scoring) errors.push("Remove automatic answer points for this assessment mode.");
  if (!needsBankReview(field) && (assessment.required || assessment.rubric?.length)) errors.push("Choose a review mode to require a review or rubric.");
  if (needsBankReview(field)) {
    if (!assessment.reviewer_role?.trim() || assessment.reviewer_role.length > 128) errors.push("Choose a reviewer responsibility.");
    const rubric = assessment.rubric ?? [];
    if (!rubric.length || rubric.length > 50) errors.push("Add 1–50 rubric outcomes.");
    const ids = new Set<string>();
    for (const outcome of rubric) {
      if (!outcome.id.trim() || outcome.id.length > 80 || ids.has(outcome.id.trim()) || !outcome.label.trim() || outcome.label.length > 200) errors.push("Give each rubric outcome a unique reference and a name of at most 200 characters.");
      ids.add(outcome.id.trim());
      if (!Number.isInteger(outcome.points) || outcome.points < 0 || outcome.points > 100) errors.push("Set rubric outcome points from 0–100.");
    }
  }
  return errors;
}
