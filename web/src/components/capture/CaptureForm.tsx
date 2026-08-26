import { useEffect, useState } from "react";
import type { CaptureAnswerValue, CaptureAnswers, CaptureField, CaptureFormContract, CapturePresentationMode } from "../../types";
import { CaptureFieldControl, type CaptureAttachment } from "./CaptureFieldControl";
import { answerText, effectivePresentationMode, normalizeFieldType, validateCaptureFields, visibleCaptureSections } from "./contract";
import { CaptureFieldSourceNotice } from "./sourceProvenance";

type Props = {
  contract: CaptureFormContract;
  answers: CaptureAnswers;
  attachments: Record<string, CaptureAttachment[]>;
  mode: CapturePresentationMode;
  external: boolean;
  uploadingField: string | null;
  onAnswer: (fieldID: string, value: CaptureAnswerValue) => void;
  onUpload: (field: CaptureField, files: File[], previewURL?: string) => void;
  onRemoveAttachment: (field: CaptureField, attachmentID: string) => void;
  onModeChange: (mode: "CLASSIC" | "WIZARD") => void;
  onBeforeSectionNavigation?: () => Promise<boolean> | boolean;
  onReview: () => void;
};

export function CaptureForm({ contract, answers, attachments, mode, external, uploadingField, onAnswer, onUpload, onRemoveAttachment, onModeChange, onBeforeSectionNavigation, onReview }: Props) {
  const sections = visibleCaptureSections(contract, answers);
  const effectiveMode = effectivePresentationMode(contract, answers, mode);
  const [sectionIndex, setSectionIndex] = useState(0);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [navigating, setNavigating] = useState(false);
  const unsupported = contract.fields.filter((field) => !normalizeFieldType(field.type));

  useEffect(() => {
    setSectionIndex((current) => Math.min(current, Math.max(sections.length - 1, 0)));
  }, [sections.length]);

  function update(fieldID: string, value: CaptureAnswerValue) {
    setErrors((current) => {
      if (!current[fieldID]) return current;
      const next = { ...current };
      delete next[fieldID];
      return next;
    });
    onAnswer(fieldID, value);
  }

  function validate(fields: CaptureField[], next: () => void) {
    const failures = validateCaptureFields(fields, answers);
    setErrors(Object.fromEntries(failures.map((failure) => [failure.fieldID, failure.message])));
    if (failures.length === 0) next();
  }

  function continueToNextSection() {
    if (!onBeforeSectionNavigation) {
      setSectionIndex((current) => current + 1);
      return;
    }
    setNavigating(true);
    void Promise.resolve(onBeforeSectionNavigation()).then((saved) => {
      if (saved) setSectionIndex((current) => current + 1);
    }).finally(() => setNavigating(false));
  }

  function renderFields(fields: CaptureField[]) {
    return fields.map((field) => <div className="capture-field-shell" id={`capture-field-${field.id}`} key={field.id}><CaptureFieldControl
      field={field}
      value={answers[field.id]}
	  attachments={attachments[field.id]}
      uploading={uploadingField === field.id}
      external={external}
      error={errors[field.id]}
      onChange={(value) => update(field.id, value)}
	  onUpload={(files, previewURL) => onUpload(field, files, previewURL)}
	  onRemove={(attachmentID) => onRemoveAttachment(field, attachmentID)}
    /><CaptureFieldSourceNotice field={field} value={answerText(answers[field.id])}/></div>);
  }

  const errorEntries = Object.entries(errors);
  const modeSwitch = contract.presentation.allow_mode_switch && <div className="capture-mode-switch" role="group" aria-label="Question layout"><button type="button" className="secondary-button" aria-pressed={effectiveMode === "CLASSIC"} onClick={() => onModeChange("CLASSIC")}>Show all questions</button><button type="button" className="secondary-button" aria-pressed={effectiveMode === "WIZARD"} onClick={() => onModeChange("WIZARD")}>Show one section at a time</button></div>;
  const errorSummary = errorEntries.length > 0 && <div className="capture-error-summary" role="alert" aria-labelledby="capture-error-summary-title"><strong id="capture-error-summary-title">Check the highlighted answers</strong><ul>{errorEntries.map(([fieldID, message]) => <li key={fieldID}><a href={`#capture-field-${fieldID}`}>{message}</a></li>)}</ul></div>;

  if (sections.length === 0) return <div className="capture-form"><p>No questions are required for the answers provided.</p><div className="wizard-actions"><button className="primary-button" type="button" onClick={onReview}>Review response</button></div></div>;

  if (effectiveMode === "CLASSIC") {
    const fields = sections.flatMap((section) => section.fields);
    return <div className="capture-form">{modeSwitch}{sections.length > 1 && <nav className="capture-section-index" aria-label="Request sections"><span>Sections</span><ol>{sections.map((section) => <li key={section.id}><a href={`#capture-section-${section.id}`}>{section.title}</a></li>)}</ol></nav>}{errorSummary}{unsupported.length > 0 && <UnsupportedFields fields={unsupported}/>} {sections.map((section) => <section className="capture-section" id={`capture-section-${section.id}`} aria-labelledby={`capture-section-title-${section.id}`} key={section.id}><h3 id={`capture-section-title-${section.id}`}>{section.title}</h3>{section.help && <p className="field-help">{section.help}</p>}{renderFields(section.fields)}</section>)}<div className="wizard-actions"><button className="primary-button" type="button" disabled={unsupported.length > 0 || Boolean(uploadingField)} onClick={() => validate(fields, onReview)}>{uploadingField ? "Uploading…" : "Review response"}</button></div></div>;
  }

  const section = sections[sectionIndex] ?? sections[0]!;
  const isLast = sectionIndex === sections.length - 1;
  return <div className="capture-form capture-wizard">{modeSwitch}<div className="capture-progress"><span aria-live="polite">Step {sectionIndex + 1} of {sections.length}</span><progress max={sections.length} value={sectionIndex + 1} aria-label={`Section ${sectionIndex + 1} of ${sections.length}`}/></div>{errorSummary}{unsupported.length > 0 && <UnsupportedFields fields={unsupported}/>}<section className="capture-section" aria-labelledby={`capture-section-title-${section.id}`}><h3 id={`capture-section-title-${section.id}`}>{section.title}</h3>{section.help && <p className="field-help">{section.help}</p>}{renderFields(section.fields)}</section><div className="wizard-actions">{sectionIndex > 0 && <button className="secondary-button" type="button" disabled={navigating} onClick={() => { setErrors({}); setSectionIndex((current) => current - 1); }}>Back</button>}<button className="primary-button" type="button" disabled={unsupported.length > 0 || Boolean(uploadingField) || navigating} onClick={() => validate(section.fields, () => isLast ? onReview() : void continueToNextSection())}>{uploadingField ? "Uploading…" : navigating ? "Saving…" : isLast ? "Review response" : "Continue"}</button></div></div>;
}

function UnsupportedFields({ fields }: { fields: CaptureField[] }) {
  return <div className="inline-error" role="alert"><strong>This request includes a field that cannot be collected here.</strong><p>{fields.map((field) => field.label).join(", ")}. Ask the sender to update the request.</p></div>;
}
