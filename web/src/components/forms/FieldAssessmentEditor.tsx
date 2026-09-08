import { useEffect, useState } from "react";
import { loadIdentityAccessOverview, type IdentityRole } from "../../identityAccessApi";
import type { FieldAssessment, FieldAssessmentMode, FormScoringMode } from "../../monitoringTypes";
import { Button, Notice, SelectField, TextField } from "../ui";
import type { AuthoringField } from "./formAuthoring";
import { assessmentConfigurationErrors, assessmentModes, needsBankReview } from "./fieldAssessment";

export function FieldAssessmentEditor({ field, onChange, scoringMode = "RISK" }: { field: AuthoringField; onChange: (change: Partial<AuthoringField>) => void; scoringMode?: FormScoringMode }) {
  const [roles, setRoles] = useState<IdentityRole[]>([]);
  const [rolesError, setRolesError] = useState(false);
  const bankReview = needsBankReview(field);
  useEffect(() => {
    if (!bankReview) return;
    let active = true;
    void loadIdentityAccessOverview().then((result) => { if (active) { setRoles(result.roles); setRolesError(false); } }).catch(() => { if (active) setRolesError(true); });
    return () => { active = false; };
  }, [bankReview]);
  const assessment = field.assessment;
  function update(patch: Partial<FieldAssessment>) {
    onChange({ assessment: { mode: "NONE", required: true, weight: 100, ...assessment, ...patch } });
  }
  function chooseMode(mode: FieldAssessmentMode) {
    const review = mode === "MANUAL" || mode === "AUTOMATIC_REVIEW";
    onChange({
      assessment: { weight: 100, ...assessment, mode, required: review ? assessment?.required ?? true : false, rubric: review ? assessment?.rubric ?? [] : undefined, reviewer_role: review ? assessment?.reviewer_role : undefined },
      ...(mode === "MANUAL" || mode === "NONE" ? { scoring: undefined } : {}),
    });
  }
  const roleOptions = roles.map((role) => ({ id: role.code, label: role.name }));
  if (assessment?.reviewer_role && !roleOptions.some((role) => role.id === assessment.reviewer_role)) roleOptions.push({ id: assessment.reviewer_role, label: `Saved responsibility: ${assessment.reviewer_role}` });
  return <fieldset className="builder-subpanel field-assessment-editor">
    <legend>Field assessment</legend>
    <SelectField label="How this field is assessed" value={assessment?.mode} placeholder={field.scoring ? "Existing automatic rules" : "Keep existing form rules"} allowsEmpty={false} options={assessmentModes} onChange={(mode) => { if (mode) chooseMode(mode); }}/>
    {assessment?.mode === "NONE" && <p className="field-note">This field collects an answer or evidence without adding assessment points. Required-response checks still apply.</p>}
    {(assessment?.mode === "AUTOMATIC" || assessment?.mode === "AUTOMATIC_REVIEW") && <p className="field-note">Configure answer points below or use the form’s scoring settings for conditions and rule testing.</p>}
    {assessment && assessment.mode !== "NONE" && <TextField label="Assessment weight" type="number" min={1} max={100} step={1} value={String(assessment.weight)} onChange={(value) => update({ weight: Number(value) })}/>}
    {bankReview && <>
      <label className="compact-control"><input type="checkbox" checked={assessment?.required ?? false} onChange={(event) => update({ required: event.target.checked })}/> Require bank review before assessment is complete</label>
      <SelectField label="Reviewer responsibility" value={assessment?.reviewer_role} placeholder="Choose a reviewer responsibility" options={roleOptions} onChange={(reviewer_role) => update({ reviewer_role })}/>
      {rolesError && <Notice tone="warning">Reviewer responsibilities could not be loaded. The saved review route is retained; reopen this field to retry.</Notice>}
      <p className="field-note">The current bank authority route determines who may record each judgement. Every judgement requires a rationale.</p>
      {(field.type === "file" || field.type === "vendor_document" || field.type === "photo") && <p className="field-note">A received document remains unassessed until a bank reviewer records the evidence outcome.</p>}
      <fieldset className="builder-subpanel"><legend>Bank rubric</legend>
        <p className="field-note">{scoringMode === "COMPLIANCE" ? "Higher points mean stronger compliance; lower points mean greater concern." : "Higher points mean greater risk; lower points mean less concern."}</p>
        {(assessment?.rubric ?? []).map((outcome, index) => <div key={outcome.id} className="field-assessment-editor__outcome">
          <TextField label={`Outcome ${index + 1}`} value={outcome.label} maxLength={200} onChange={(label) => update({ rubric: assessment!.rubric!.map((item, i) => i === index ? { ...item, label } : item) })}/>
          <TextField label={`Outcome ${index + 1} points`} type="number" min={0} max={100} step={1} value={String(outcome.points)} onChange={(value) => update({ rubric: assessment!.rubric!.map((item, i) => i === index ? { ...item, points: Number(value) } : item) })}/>
          <Button variant="quiet" onPress={() => update({ rubric: assessment!.rubric!.filter((_, i) => i !== index) })} aria-label={`Remove outcome ${index + 1}`}>Remove outcome</Button>
        </div>)}
        <Button variant="secondary" isDisabled={(assessment?.rubric?.length ?? 0) >= 50} onPress={() => {
          const rubric = assessment?.rubric ?? [];
          let next = rubric.length + 1;
          while (rubric.some((outcome) => outcome.id === `outcome_${next}`)) next++;
          update({ rubric: [...rubric, { id: `outcome_${next}`, label: "", points: 0 }] });
        }}>Add rubric outcome</Button>
        {(assessment?.rubric?.length ?? 0) >= 50 && <p className="field-note">This rubric has the maximum 50 outcomes.</p>}
      </fieldset>
    </>}
    {assessmentConfigurationErrors(field).map((error, index) => <p className="field-note" key={index}>{error}</p>)}
  </fieldset>;
}
