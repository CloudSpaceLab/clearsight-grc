import { useMemo, useState } from "react";
import type { FormEvent } from "react";
import "../form-authoring.css";
import type { ReusableFormTemplateRef } from "../formsTypes";
import { createFormTemplate } from "../monitoringApi";
import type { CreateFormTemplateInput, FormFieldType, FormScoringMode, FormTemplate } from "../monitoringTypes";
import type { CaptureFieldConstraints, CaptureFormContract } from "../types";
import { FocusedSheet } from "./FocusedSheet";
import { Notice } from "./ui";
import { FormPreview } from "./forms/FormPreview";
import { FormApprovalSheet } from "./forms/builder/FormApprovalSheet";
import { FormBuilderResponsiveNav } from "./forms/builder/FormBuilderResponsiveNav";
import { FormBuilderToolbar } from "./forms/builder/FormBuilderToolbar";
import { FormCanvas } from "./forms/builder/FormCanvas";
import { FormInspector } from "./forms/builder/FormInspector";
import { FormOutline } from "./forms/builder/FormOutline";
import { FormReviewDrawer, reviewIssueCount } from "./forms/builder/FormReviewDrawer";
import { ReusableSectionPicker } from "./forms/builder/ReusableSectionPicker";
import type { BuilderSelection } from "./forms/builder/builderSelection";
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
  type AuthoringField,
  type FormDraft,
  type FormQualityIssue,
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

type ResponsivePane = "outline" | "settings";

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
  const [selection, setSelection] = useState<BuilderSelection>(() => draft.fields[0] ? { kind: "field", fieldID: draft.fields[0].id } : { kind: "overview" });
  const [previewOpen, setPreviewOpen] = useState(false);
  const [reviewOpen, setReviewOpen] = useState(false);
  const [approvalOpen, setApprovalOpen] = useState(false);
  const [responsivePane, setResponsivePane] = useState<ResponsivePane | null>(null);
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
  const reviewCount = reviewIssueCount(issues, draft.fields);
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
      scoreProfile: current.scoreProfile && current.scoreProfile.mode === scoringMode ? current.scoreProfile : undefined,
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
    setSelection({ kind: "section", sectionID: id });
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
    setSelection({ kind: "section", sectionID: copied.section.id });
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
    setSelection({ kind: "section", sectionID: copied.section.id });
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
    if (selection.kind === "section" && selection.sectionID === sectionID) setSelection({ kind: "overview" });
  }

  function addField(type: FormFieldType = "short_text", requestedSectionID?: string) {
    if (draft.fields.length >= 200) return;
    const sectionID = requestedSectionID && draft.sections.some((section) => section.id === requestedSectionID)
      ? requestedSectionID
      : draft.sections[0]?.id ?? "section_1";
    const field = blankField(nextField, sectionID, type);
    setDraft((current) => ({ ...current, fields: [...current.fields, field] }));
    setNextField((current) => current + 1);
    setSelection({ kind: "field", fieldID: field.id });
  }

  function moveField(index: number, offset: -1 | 1) {
    const source = draft.fields[index];
    if (!source) return;
    const siblingIndices = draft.fields.flatMap((field, fieldIndex) => field.section_id === source.section_id ? [fieldIndex] : []);
    const siblingPosition = siblingIndices.indexOf(index);
    const destination = siblingIndices[siblingPosition + offset];
    if (siblingPosition < 0 || destination === undefined) return;
    const fields = [...draft.fields];
    [fields[index], fields[destination]] = [fields[destination]!, fields[index]!];
    commitFieldOrder(fields);
  }

  function reorderField(fromIndex: number, toIndex: number) {
    const source = draft.fields[fromIndex];
    const destination = draft.fields[toIndex];
    if (!source || !destination || source.section_id !== destination.section_id || fromIndex === toIndex) return;
    const fields = [...draft.fields];
    const [moved] = fields.splice(fromIndex, 1);
    if (!moved) return;
    fields.splice(toIndex, 0, moved);
    commitFieldOrder(fields);
  }

  function commitFieldOrder(fields: AuthoringField[]) {
    const positions = new Map(fields.map((field, index) => [field.id, index]));
    const invalid = fields.find((field, index) => {
      if (!field.condition) return false;
      const sourceIndex = positions.get(field.condition.field_id);
      return sourceIndex === undefined || sourceIndex >= index;
    });
    if (invalid) {
      setError(`${invalid.label.trim() || "This question"} must remain after the question it depends on.`);
      return;
    }
    setDraft((current) => ({ ...current, fields }));
    setError("");
  }

  function duplicateField(index: number) {
    if (draft.fields.length >= 200) return;
    const source = draft.fields[index];
    if (!source) return;
    const id = `question_${nextField}`;
    const duplicate: AuthoringField = {
      ...source,
      id,
      label: source.label.trim() ? `${source.label} copy` : "",
      options: source.options ? [...source.options] : undefined,
      accepted_formats: source.accepted_formats ? [...source.accepted_formats] : undefined,
      constraints: source.constraints ? { ...source.constraints } : undefined,
      condition: source.condition ? { ...source.condition, values: source.condition.values ? [...source.condition.values] : undefined } : undefined,
      scoring: source.scoring ? {
        ...source.scoring,
        answer_scores: { ...source.scoring.answer_scores },
        critical_answers: source.scoring.critical_answers ? [...source.scoring.critical_answers] : undefined,
      } : undefined,
      record_target: source.record_target ? { ...source.record_target } : undefined,
    };
    const fields = [...draft.fields];
    fields.splice(index + 1, 0, duplicate);
    setDraft((current) => ({ ...current, fields }));
    setNextField((current) => current + 1);
    setSelection({ kind: "field", fieldID: id });
    setError("");
  }

  function removeField(index: number) {
    const removing = draft.fields[index];
    setDraft((current) => {
      if (current.fields.length <= 1) return current;
      const fields = current.fields.filter((_, fieldIndex) => fieldIndex !== index);
      return { ...current, ...reconcileAuthoringOrder(current.sections, fields) };
    });
    if (removing && selection.kind === "field" && selection.fieldID === removing.id) {
      setSelection({ kind: "section", sectionID: removing.section_id ?? draft.sections[0]?.id ?? "section_1" });
    }
  }

  function addSignOff() {
    if (draft.fields.length >= 200) return;
    const sectionID = draft.sections[draft.sections.length - 1]?.id ?? draft.sections[0]?.id ?? "section_1";
    const result = addRequiredSignOff(draft.fields, sectionID, nextField);
    setDraft((current) => ({ ...current, fields: [...current.fields, result.field] }));
    setNextField(result.nextField);
    setSelection({ kind: "field", fieldID: result.field.id });
  }

  function fixReviewIssue(issue: FormQualityIssue) {
    if (issue.fieldID) setSelection({ kind: "field", fieldID: issue.fieldID });
    else if (issue.sectionID) setSelection({ kind: "section", sectionID: issue.sectionID });
    else setSelection({ kind: "overview" });
    if (issue.id === "code") setResponsivePane("settings");
    setReviewOpen(false);
    window.setTimeout(() => {
      let target: HTMLElement | null = null;
      if (issue.fieldID) {
        const card = Array.from(document.querySelectorAll<HTMLElement>("[data-builder-field-id]"))
          .find((element) => element.dataset.builderFieldId === issue.fieldID);
        target = card?.querySelector<HTMLElement>(".form-question-prompt") ?? null;
      } else if (issue.sectionID) {
        const section = Array.from(document.querySelectorAll<HTMLElement>("[data-builder-section-id]"))
          .find((element) => element.dataset.builderSectionId === issue.sectionID);
        target = section?.querySelector<HTMLElement>("input") ?? null;
      } else if (issue.id === "code") {
        target = document.querySelector<HTMLElement>("[aria-label='Code']");
      } else if (issue.id === "purpose") {
        target = document.querySelector<HTMLElement>("[aria-label='Purpose']");
      } else {
        target = document.querySelector<HTMLElement>("[aria-label='Form name']");
      }
      target?.scrollIntoView?.({ block: "center" });
      target?.focus();
    }, 0);
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
      scoreProfile: undefined,
      presentation: "AUTOMATIC",
      allowModeSwitch: false,
      sections: [{ id: "section_1", title: "Security review" }],
      fields,
    });
    setNextSection(2);
    setNextField(fields.length + 1);
    setSelection({ kind: "field", fieldID: fields[0]!.id });
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

  async function saveCurrentDraft() {
    if (saving) return;
    setError("");
    if (!draftValid) {
      setError(qualityError(draftIssues, "Resolve the draft validation checks before saving."));
      setReviewOpen(true);
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

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    void saveCurrentDraft();
  }

  async function sendForApproval() {
    if (!onSendForApproval || saving) return;
    setError("");
    if (!approvalReady) {
      setError(qualityError());
      setReviewOpen(true);
      return;
    }
    const needsPersist = !initialValue || changedFromInitial;
    let candidate: FormTemplate | undefined;
    setSaving(true);
    try {
      candidate = needsPersist ? await persist(createInput) : initialValue;
      if (!candidate) throw new Error("The current form draft could not be loaded.");
      const transitioned = await onSendForApproval(candidate);
      onSaved(transitioned ?? candidate);
    } catch (caught) {
      if (needsPersist && candidate) {
        onSaved(candidate);
        return;
      }
      setError(caught instanceof Error ? caught.message : "The form could not be sent for approval. Reload the current revision and try again.");
    } finally {
      setSaving(false);
    }
  }

  function renderOutlinePane(closeAfterSelection = false) {
    const closeResponsivePane = () => {
      if (closeAfterSelection) setResponsivePane(null);
    };
    return <div className="form-builder-outline-shell">
      <FormOutline
        sections={draft.sections}
        fields={draft.fields}
        issues={issues}
        selection={selection}
        onSelect={(next) => { setSelection(next); closeResponsivePane(); }}
        onAddSection={() => { addSection(); closeResponsivePane(); }}
        onAddField={() => { addField("short_text"); closeResponsivePane(); }}
      />
      <ReusableSectionPicker
        reusableTemplates={reusableTemplates}
        loadReusableTemplate={loadReusableTemplate}
        sectionLimitReached={draft.sections.length >= 20}
        onInsert={(template, sectionID) => { insertReusableSection(template, sectionID); closeResponsivePane(); }}
      />
      {!initialValue && <button className="text-button form-outline-example" type="button" onClick={() => { usePasswordResetReview(); closeResponsivePane(); }}>Use password reset example</button>}
    </div>;
  }

  function renderInspectorPane() {
    return <FormInspector
      draft={draft}
      templateRevision={initialValue ? { id: initialValue.id, version: initialValue.version } : undefined}
      selection={selection}
      onPatch={patch}
      onScoringMode={setScoringMode}
      onSectionsChange={(sections) => patch({ sections })}
      onFieldChange={updateField}
      onFieldTypeChange={updateFieldType}
      onFieldConstraint={updateFieldConstraint}
      onFieldScoringToggle={toggleFieldScoring}
      onMoveSection={moveSection}
      onDuplicateSection={duplicateCurrentSection}
      onRemoveSection={removeSection}
      onMoveField={moveField}
      onRemoveField={removeField}
    />;
  }

  return <form className="monitoring-builder form-builder forms-authoring-shell form-builder-workspace" noValidate onSubmit={submit}>
    <FormBuilderToolbar
      title={draft.name}
      saving={saving}
      approvalReady={approvalReady}
      reviewCount={reviewCount}
      canSendForApproval={Boolean(onSendForApproval)}
      onCancel={onCancel}
      onPreview={() => setPreviewOpen(true)}
      onReview={() => setReviewOpen(true)}
      onSave={() => void saveCurrentDraft()}
      onSendForApproval={onSendForApproval ? () => setApprovalOpen(true) : undefined}
    />

    <FormBuilderResponsiveNav
      onOutline={() => setResponsivePane("outline")}
      onSettings={() => setResponsivePane("settings")}
    />

    <div className="form-builder-grid">
      {renderOutlinePane()}

      <FormCanvas
        draft={draft}
        selection={selection}
        onPatch={patch}
        onSectionsChange={(sections) => patch({ sections })}
        onFieldChange={updateField}
        onFieldTypeChange={updateFieldType}
        onSelect={setSelection}
        onAddField={(sectionID) => addField("short_text", sectionID)}
        onAddSection={addSection}
        onMoveField={moveField}
        onReorderField={reorderField}
        onDuplicateField={duplicateField}
        onRemoveField={removeField}
      />

      {renderInspectorPane()}
    </div>

    {error && <Notice tone="error">{error}</Notice>}

    {previewOpen && <FocusedSheet label="Form preview" closeLabel="Close form preview" panelClassName="form-preview-sheet" onClose={() => setPreviewOpen(false)}>
      <div className="form-preview-sheet-content"><FormPreview contract={previewContract}/></div>
    </FocusedSheet>}

    {reviewOpen && <FormReviewDrawer
      issues={issues}
      fields={draft.fields}
      initialValue={initialValue}
      onFix={fixReviewIssue}
      onAddRequiredSignOff={() => { addSignOff(); setReviewOpen(false); }}
      onClose={() => setReviewOpen(false)}
    />}

    {approvalOpen && <FormApprovalSheet
      input={createInput}
      changedFromInitial={changedFromInitial}
      saving={saving}
      onClose={() => setApprovalOpen(false)}
      onConfirm={() => { setApprovalOpen(false); void sendForApproval(); }}
    />}

    {responsivePane === "outline" && <FocusedSheet
      label="Form outline"
      closeLabel="Close form outline"
      panelClassName="form-builder-responsive-sheet form-builder-outline-sheet"
      onClose={() => setResponsivePane(null)}
    >
      {renderOutlinePane(true)}
    </FocusedSheet>}

    {responsivePane === "settings" && <FocusedSheet
      label="Form settings"
      closeLabel="Close form settings"
      panelClassName="form-builder-responsive-sheet form-builder-settings-sheet"
      onClose={() => setResponsivePane(null)}
    >
      {renderInspectorPane()}
    </FocusedSheet>}
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
