import { useMemo, useState } from "react";
import type { FormEvent } from "react";
import "../form-authoring.css";
import type { ReusableFormTemplateRef } from "../formsTypes";
import { createFormTemplate } from "../monitoringApi";
import type { CreateFormTemplateInput, FormFieldType, FormScoringMode, FormTemplate } from "../monitoringTypes";
import type { CaptureFieldConstraints, CaptureFormContract } from "../types";
import { FormPreview } from "./forms/FormPreview";
import { FormPropertyPanel } from "./forms/FormPropertyPanel";
import { FormQualityPanel } from "./forms/FormQualityPanel";
import {
  addRequiredSignOff,
  blankField,
  buildCreateInput,
  changeFieldType,
  copySectionFromTemplate,
  draftFromTemplate,
  duplicateSection,
  enableScoring,
  maxGeneratedNumber,
  reconcileAuthoringOrder,
  updateConstraint,
  type FormDraft,
} from "./forms/formAuthoring";
import { evaluateDraftValidity, evaluateQuality } from "./forms/formQuality";

type Props = {
  programID?: string;
  initialValue?: FormTemplate;
  onSaved: (form: FormTemplate) => void;
  onCancel: () => void;
  saveDraft?: (input: CreateFormTemplateInput) => Promise<FormTemplate>;
  onSendForApproval?: (form: FormTemplate) => Promise<FormTemplate | void>;
  reusableTemplates?: ReusableFormTemplateRef[];
  loadReusableTemplate?: (id: string, version: number) => Promise<FormTemplate>;
  allowIncompleteComplianceDraft?: boolean;
};

const passwordResetQuestions = [
  "Was the customer’s identity verified before the reset?",
  "Was the one-time code sent only to a registered channel?",
  "Were changes to recovery details separately authenticated?",
  "Were repeated failed reset attempts blocked or rate-limited?",
  "Were reset events logged and reviewed for unusual activity?",
];

export function FormBuilder({
  programID,
  initialValue,
  onSaved,
  onCancel,
  saveDraft,
  onSendForApproval,
  reusableTemplates,
  loadReusableTemplate,
  allowIncompleteComplianceDraft = false,
}: Props) {
  const [draft, setDraft] = useState<FormDraft>(() => draftFromTemplate(initialValue));
  const [nextSection, setNextSection] = useState(() => maxGeneratedNumber(draft.sections.map((section) => section.id), "section"));
  const [nextField, setNextField] = useState(() => maxGeneratedNumber(draft.fields.map((field) => field.id), "question"));
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  const createInput = useMemo(() => buildCreateInput(draft), [draft]);
  const previewContract = useMemo<CaptureFormContract>(() => ({
    presentation: createInput.presentation,
    sections: createInput.sections,
    fields: createInput.fields,
  }), [createInput]);
  const issues = useMemo(() => evaluateQuality(draft), [draft]);
  const draftIssues = useMemo(
    () => allowIncompleteComplianceDraft ? evaluateDraftValidity(draft) : evaluateQuality(draft),
    [allowIncompleteComplianceDraft, draft],
  );
  const approvalReady = !issues.some((issue) => issue.blocking);
  const draftValid = !draftIssues.some((issue) => issue.blocking);
  const initialInput = useMemo(() => initialValue ? buildCreateInput(draftFromTemplate(initialValue)) : undefined, [initialValue]);
  const changedFromInitial = !initialInput || JSON.stringify(initialInput) !== JSON.stringify(createInput);

  function patch(patchValue: Partial<FormDraft>) {
    setDraft((current) => ({ ...current, ...patchValue }));
    setError("");
  }

  function setScoringMode(scoringMode: FormScoringMode) {
    setDraft((current) => ({
      ...current,
      scoringMode,
      sections: scoringMode === "COMPLIANCE" ? current.sections : current.sections.map((section) => ({ ...section, weight: undefined })),
      fields: scoringMode === "NONE" ? current.fields.map((field) => ({ ...field, scoring: undefined })) : current.fields,
    }));
    setError("");
  }

  function updateField(index: number, change: Partial<FormDraft["fields"][number]>) {
    setDraft((current) => {
      const fields = current.fields.map((field, fieldIndex) => {
        if (fieldIndex !== index) return field;
        let next = { ...field, ...change };
        if (change.options && next.scoring) next = enableScoring(next, current.scoringMode);
        return next;
      });
      return { ...current, ...reconcileAuthoringOrder(current.sections, fields) };
    });
    setError("");
  }

  function updateFieldType(index: number, type: FormFieldType) {
    setDraft((current) => ({
      ...current,
      fields: current.fields.map((field, fieldIndex) => fieldIndex === index ? changeFieldType(field, type, current.scoringMode) : field),
    }));
    setError("");
  }

  function updateFieldConstraint(index: number, key: keyof CaptureFieldConstraints, value: number | string | undefined) {
    setDraft((current) => ({
      ...current,
      fields: current.fields.map((field, fieldIndex) => fieldIndex === index ? updateConstraint(field, key, value) : field),
    }));
    setError("");
  }

  function toggleFieldScoring(index: number, enabled: boolean) {
    setDraft((current) => ({
      ...current,
      fields: current.fields.map((field, fieldIndex) => fieldIndex === index
        ? enabled ? enableScoring(field, current.scoringMode) : { ...field, scoring: undefined }
        : field),
    }));
    setError("");
  }

  function addSection() {
    if (draft.sections.length >= 20) return;
    const id = `section_${nextSection}`;
    setDraft((current) => ({ ...current, sections: [...current.sections, { id, title: "" }] }));
    setNextSection((current) => current + 1);
  }

  function duplicateCurrentSection(sectionID: string) {
    if (draft.sections.length >= 20) return;
    const copied = duplicateSection(sectionID, draft.sections, draft.fields, nextSection, nextField);
    if (!copied) return;
    setDraft((current) => {
      const sections = [...current.sections, copied.section];
      const fields = [...current.fields, ...copied.fields];
      return { ...current, ...reconcileAuthoringOrder(sections, fields) };
    });
    setNextSection(copied.nextSection);
    setNextField(copied.nextField);
  }

  function insertReusableSection(template: FormTemplate, sectionID: string) {
    if (draft.sections.length >= 20) return;
    const copied = copySectionFromTemplate(template, sectionID, nextSection, nextField);
    if (!copied) return;
    setDraft((current) => {
      const sections = [...current.sections, copied.section];
      const fields = [...current.fields, ...copied.fields];
      return { ...current, ...reconcileAuthoringOrder(sections, fields) };
    });
    setNextSection(copied.nextSection);
    setNextField(copied.nextField);
  }

  function moveSection(index: number, offset: -1 | 1) {
    setDraft((current) => {
      const sections = move(current.sections, index, offset);
      return { ...current, ...reconcileAuthoringOrder(sections, current.fields) };
    });
  }

  function removeSection(sectionID: string) {
    setDraft((current) => {
      if (current.sections.length <= 1) return current;
      const sections = current.sections.filter((section) => section.id !== sectionID);
      const replacement = sections[0]?.id;
      if (!replacement) return current;
      const fields = current.fields.map((field) => field.section_id === sectionID ? { ...field, section_id: replacement } : field);
      return { ...current, ...reconcileAuthoringOrder(sections, fields) };
    });
  }

  function addField(type: FormFieldType) {
    if (draft.fields.length >= 200) return;
    const sectionID = draft.sections[0]?.id ?? "section_1";
    setDraft((current) => ({ ...current, fields: [...current.fields, blankField(nextField, sectionID, type)] }));
    setNextField((current) => current + 1);
  }

  function moveField(index: number, offset: -1 | 1) {
    setDraft((current) => {
      const source = current.fields[index];
      if (!source) return current;
      const siblingIndices = current.fields.flatMap((field, fieldIndex) => field.section_id === source.section_id ? [fieldIndex] : []);
      const siblingPosition = siblingIndices.indexOf(index);
      const destination = siblingIndices[siblingPosition + offset];
      if (siblingPosition < 0 || destination === undefined) return current;
      const fields = [...current.fields];
      [fields[index], fields[destination]] = [fields[destination]!, fields[index]!];
      return { ...current, ...reconcileAuthoringOrder(current.sections, fields) };
    });
  }

  function removeField(index: number) {
    setDraft((current) => {
      if (current.fields.length <= 1) return current;
      const fields = current.fields.filter((_, fieldIndex) => fieldIndex !== index);
      return { ...current, ...reconcileAuthoringOrder(current.sections, fields) };
    });
  }

  function addSignOff() {
    if (draft.fields.length >= 200) return;
    const sectionID = draft.sections[draft.sections.length - 1]?.id ?? draft.sections[0]?.id ?? "section_1";
    const result = addRequiredSignOff(draft.fields, sectionID, nextField);
    setDraft((current) => ({ ...current, fields: [...current.fields, result.field] }));
    setNextField(result.nextField);
  }

  function usePasswordResetReview() {
    const fields = passwordResetQuestions.map((label, index) => {
      const field = blankField(index + 1, "section_1", "yes_no");
      field.label = label;
      field.scoring = {
        weight: 1,
        answer_scores: { Yes: 0, No: 100 },
        critical_answers: index < 3 ? ["No"] : [],
      };
      return field;
    });
    setDraft({
      code: "PASSWORD-RESET-REVIEW",
      name: "Password reset security review",
      purpose: "Confirm that password reset safeguards operated during the reporting period.",
      scoringMode: "RISK",
      presentation: "AUTOMATIC",
      allowModeSwitch: false,
      sections: [{ id: "section_1", title: "Security review" }],
      fields,
    });
    setNextSection(2);
    setNextField(fields.length + 1);
    setError("");
  }

  async function persist(input: CreateFormTemplateInput) {
    if (saveDraft) return saveDraft(input);
    if (programID) return createFormTemplate(programID, input);
    throw new Error("This form editor is not connected to a governed save command.");
  }

  function qualityError(sourceIssues = issues, fallback = "Resolve the approval-quality checks before continuing.") {
    const first = sourceIssues.find((issue) => issue.blocking);
    return first?.message ?? fallback;
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    if (!draftValid) {
      setError(qualityError(draftIssues, "Resolve the draft validation checks before saving."));
      return;
    }
    setSaving(true);
    try {
      const saved = await persist(createInput);
      onSaved(saved);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "The form draft could not be saved. Check the current form and try again.");
    } finally {
      setSaving(false);
    }
  }

  async function sendForApproval() {
    if (!onSendForApproval || saving) return;
    setError("");
    if (!approvalReady) {
      setError(qualityError());
      return;
    }
    const needsPersist = !initialValue || changedFromInitial;
    let candidate: FormTemplate | undefined;
    setSaving(true);
    try {
      candidate = needsPersist ? await persist(createInput) : initialValue;
      if (!candidate) throw new Error("The current governed draft could not be resolved.");
      const transitioned = await onSendForApproval(candidate);
      onSaved(transitioned ?? candidate);
    } catch (caught) {
      if (needsPersist && candidate) {
        // Persistence already created a new immutable revision. Return that exact
        // draft to the workspace instead of retrying against its stale parent.
        onSaved(candidate);
        return;
      }
      setError(caught instanceof Error ? caught.message : "The form could not be sent for approval. Reload the current revision and try again.");
    } finally {
      setSaving(false);
    }
  }

  return <form className="monitoring-builder form-builder forms-authoring-shell" noValidate onSubmit={submit}>
    <div className="monitoring-builder-heading">
      <div><span className="eyebrow">Collection form</span><h4>{initialValue ? `Edit ${initialValue.name}` : "New collection form"}</h4><p>Build the exact governed response contract, validate it, then save a draft or send the current revision for independent approval.</p></div>
      {!initialValue && <button className="secondary-button" type="button" onClick={usePasswordResetReview}>Use password reset review</button>}
    </div>

    <div className="monitoring-form-grid">
      <label><span>Form name</span><input value={draft.name} maxLength={200} onChange={(event) => patch({ name: event.target.value })} required/></label>
      <label><span>Code</span><input value={draft.code} maxLength={80} onChange={(event) => patch({ code: event.target.value })} required/></label>
      <label className="full"><span>Purpose</span><textarea value={draft.purpose} maxLength={1000} onChange={(event) => patch({ purpose: event.target.value })} rows={2} required/></label>
    </div>

    <fieldset className="builder-panel">
      <legend>Response and scoring</legend>
      <div className="builder-control-grid">
        <label><span>Default layout</span><select value={draft.presentation} onChange={(event) => patch({ presentation: event.target.value as FormDraft["presentation"] })}><option value="AUTOMATIC">Choose by form length</option><option value="CLASSIC">Show all questions</option><option value="WIZARD">Show one section at a time</option></select></label>
        <label><span>Scoring mode</span><select value={draft.scoringMode} onChange={(event) => setScoringMode(event.target.value as FormScoringMode)}><option value="NONE">No score</option><option value="RISK">Risk score</option><option value="COMPLIANCE">Compliance score</option></select></label>
        <label className="compact-control"><input type="checkbox" checked={draft.allowModeSwitch} onChange={(event) => patch({ allowModeSwitch: event.target.checked })}/> Allow respondents to switch layouts</label>
      </div>
    </fieldset>

    <div className="form-authoring-layout">
      <FormPropertyPanel
        scoringMode={draft.scoringMode}
        sections={draft.sections}
        fields={draft.fields}
        reusableTemplates={reusableTemplates}
        loadReusableTemplate={loadReusableTemplate}
        onSectionsChange={(sections) => patch({ sections })}
        onFieldChange={updateField}
        onFieldTypeChange={updateFieldType}
        onFieldConstraint={updateFieldConstraint}
        onFieldScoringToggle={toggleFieldScoring}
        onAddSection={addSection}
        onDuplicateSection={duplicateCurrentSection}
        onMoveSection={moveSection}
        onRemoveSection={removeSection}
        onInsertReusableSection={insertReusableSection}
        onAddField={addField}
        onMoveField={moveField}
        onRemoveField={removeField}
      />
      <FormQualityPanel scoringMode={draft.scoringMode} sections={draft.sections} fields={draft.fields} issues={issues} onAddRequiredSignOff={addSignOff}/>
    </div>

    <FormPreview contract={previewContract}/>

    {error && <p className="inline-form-error" role="alert">{error}</p>}
    <div className="monitoring-form-actions">
      <button className="text-button" type="button" onClick={onCancel}>Cancel</button>
      <button className={onSendForApproval ? "secondary-button" : "primary-button"} type="submit" disabled={saving}>{saving ? "Saving…" : "Save draft"}</button>
      {onSendForApproval && <button className="primary-button" type="button" disabled={saving || !approvalReady} onClick={() => void sendForApproval()}>{saving ? "Working…" : "Send for approval"}</button>}
    </div>
  </form>;
}

function move<T>(items: T[], index: number, offset: -1 | 1) {
  const destination = index + offset;
  if (destination < 0 || destination >= items.length) return items;
  const result = [...items];
  const [item] = result.splice(index, 1);
  if (item !== undefined) result.splice(destination, 0, item);
  return result;
}
