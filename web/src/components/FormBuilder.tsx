import { useMemo, useState } from "react";
import type { FormEvent } from "react";
import { createFormTemplate } from "../monitoringApi";
import type { FormFieldType, FormTemplate, FormTemplateField } from "../monitoringTypes";
import type { CaptureAnswers, CaptureFieldConstraints, CaptureFormContract, CapturePresentationMode, CaptureSection, CaptureVisibilityCondition } from "../types";
import { CaptureForm } from "./capture/CaptureForm";

type Props = { onSaved: (form: FormTemplate) => void; onCancel: () => void };
type SectionDraft = CaptureSection;
type FieldDraft = Omit<FormTemplateField, "type"> & {
  type: FormFieldType;
  scored: boolean;
  weight: number;
  riskWhenNo: number;
  criticalNo: boolean;
};

const fieldTypes: Array<{ value: FormFieldType; label: string }> = [
  { value: "short_text", label: "Short answer" },
  { value: "long_text", label: "Long answer" },
  { value: "email", label: "Email address" },
  { value: "telephone", label: "Telephone number" },
  { value: "url", label: "Web address" },
  { value: "integer", label: "Whole number" },
  { value: "decimal", label: "Decimal number" },
  { value: "percentage", label: "Percentage" },
  { value: "currency", label: "Currency amount" },
  { value: "date", label: "Date" },
  { value: "yes_no", label: "Yes or No" },
  { value: "single_select", label: "Select one" },
  { value: "multi_select", label: "Select several" },
  { value: "checkbox", label: "Checkbox" },
  { value: "attestation", label: "Attestation" },
  { value: "file", label: "File" },
  { value: "photo", label: "Photo" },
  { value: "signature", label: "Signature" },
  { value: "vendor_document", label: "Vendor document" },
];

const formatOptions = [
  { value: "application/pdf", label: "PDF" },
  { value: "image/png", label: "PNG image" },
  { value: "image/jpeg", label: "JPEG image" },
  { value: "text/plain", label: "Text file" },
  { value: "text/csv", label: "CSV file" },
  { value: "application/vnd.openxmlformats-officedocument.wordprocessingml.document", label: "Word document" },
  { value: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", label: "Excel workbook" },
];

const passwordResetQuestions = [
  "Was the customer’s identity verified before the reset?",
  "Was the one-time code sent only to a registered channel?",
  "Were changes to recovery details separately authenticated?",
  "Were repeated failed reset attempts blocked or rate-limited?",
  "Were reset events logged and reviewed for unusual activity?",
];

function blankField(index: number, sectionID: string, type: FormFieldType = "short_text"): FieldDraft {
  return {
    id: `question_${index}`,
    section_id: sectionID,
    label: "",
    type,
    required: true,
    options: type === "yes_no" ? ["Yes", "No"] : selectionType(type) ? ["Option 1", "Option 2"] : undefined,
    accepted_formats: initialFormats(type),
    constraints: initialConstraints(type),
    scored: false,
    weight: 1,
    riskWhenNo: 100,
    criticalNo: false,
  };
}

export function FormBuilder({ onSaved, onCancel }: Props) {
  const [code, setCode] = useState("");
  const [name, setName] = useState("");
  const [purpose, setPurpose] = useState("");
  const [presentation, setPresentation] = useState<CapturePresentationMode>("AUTOMATIC");
  const [allowModeSwitch, setAllowModeSwitch] = useState(false);
  const [sections, setSections] = useState<SectionDraft[]>([{ id: "section_1", title: "Questions" }]);
  const [fields, setFields] = useState<FieldDraft[]>([blankField(1, "section_1")]);
  const [nextSection, setNextSection] = useState(2);
  const [nextField, setNextField] = useState(2);
  const [previewMode, setPreviewMode] = useState<"CLASSIC" | "WIZARD" | null>(null);
  const [previewAnswers, setPreviewAnswers] = useState<CaptureAnswers>({});
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  const previewContract = useMemo<CaptureFormContract>(() => ({
    presentation: { default_mode: presentation, allow_mode_switch: allowModeSwitch },
    sections: sections.map((section, index) => ({ ...section, title: section.title.trim() || `Section ${index + 1}` })),
    fields: fields.map((field, index) => previewField(field, index)),
  }), [allowModeSwitch, fields, presentation, sections]);

  function usePasswordResetReview() {
    setCode("PASSWORD-RESET-REVIEW");
    setName("Password reset security review");
    setPurpose("Confirm that password reset safeguards operated during the reporting period.");
    setSections([{ id: "section_1", title: "Security review" }]);
    setFields(passwordResetQuestions.map((label, index) => ({
      ...blankField(index + 1, "section_1", "yes_no"), label, scored: true, criticalNo: index < 3,
    })));
    setNextSection(2);
    setNextField(passwordResetQuestions.length + 1);
    setError("");
  }

  function updateField(index: number, change: Partial<FieldDraft>) {
    setFields((current) => current.map((field, fieldIndex) => fieldIndex === index ? { ...field, ...change } : field));
  }

  function changeFieldType(index: number, type: FormFieldType) {
    setFields((current) => current.map((field, fieldIndex) => {
      if (fieldIndex !== index) return field;
      return {
        ...field,
        type,
        options: type === "yes_no" ? ["Yes", "No"] : selectionType(type) ? field.options?.length && field.type !== "yes_no" ? field.options : ["Option 1", "Option 2"] : undefined,
        accepted_formats: initialFormats(type),
        attestation: type === "attestation" ? field.attestation ?? "" : undefined,
        constraints: initialConstraints(type),
        scored: type === "yes_no" ? field.scored : false,
      };
    }));
  }

  function updateConstraint(index: number, key: keyof CaptureFieldConstraints, value: number | string | undefined) {
    setFields((current) => current.map((field, fieldIndex) => {
      if (fieldIndex !== index) return field;
      const constraints = { ...field.constraints };
      if (value === undefined || value === "") delete constraints[key];
      else Object.assign(constraints, { [key]: value });
      return { ...field, constraints };
    }));
  }

  function addSection() {
    const id = `section_${nextSection}`;
    setSections((current) => [...current, { id, title: "" }]);
    setNextSection((current) => current + 1);
  }

  function moveSection(index: number, offset: -1 | 1) {
    setSections((current) => {
      const destination = index + offset;
      if (destination < 0 || destination >= current.length) return current;
      const reordered = [...current];
      const [section] = reordered.splice(index, 1);
      if (section) reordered.splice(destination, 0, section);
      return reordered;
    });
  }

  function removeSection(id: string) {
    setSections((current) => {
      if (current.length === 1) return current;
      const remaining = current.filter((section) => section.id !== id);
      const replacement = remaining[0]?.id ?? "section_1";
      setFields((currentFields) => currentFields.map((field) => field.section_id === id ? { ...field, section_id: replacement } : field));
      return remaining;
    });
  }

  function addField(type: FormFieldType) {
    setFields((current) => [...current, blankField(nextField, sections[0]?.id ?? "section_1", type)]);
    setNextField((current) => current + 1);
  }

  function removeField(index: number) {
    const removedID = fields[index]?.id;
    setFields((current) => current.filter((_, fieldIndex) => fieldIndex !== index).map((field) => field.condition?.field_id === removedID ? { ...field, condition: undefined } : field));
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    setError("");
    const validationError = validateDraft(code, name, purpose, sections, fields);
    if (validationError) {
      setError(validationError);
      return;
    }
    setSaving(true);
    try {
      const saved = await createFormTemplate({
        code: normalizedCode(code),
        name: name.trim(),
        purpose: purpose.trim(),
        presentation: { default_mode: presentation, allow_mode_switch: allowModeSwitch },
        sections: sections.map(cleanSection),
        fields: fields.map(cleanField),
      });
      onSaved(saved);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "The form draft could not be saved. Check the questions and try again.");
    } finally {
      setSaving(false);
    }
  }

  return <form className="monitoring-builder form-builder" noValidate onSubmit={submit}>
    <div className="monitoring-builder-heading">
      <div><span className="eyebrow">Collection form</span><h4>New collection form</h4><p>Define the information to collect and the response required for each question.</p></div>
      <button className="secondary-button" type="button" onClick={usePasswordResetReview}>Use password reset review</button>
    </div>

    <div className="monitoring-form-grid">
      <label><span>Form name</span><input value={name} maxLength={200} onChange={(event) => setName(event.target.value)} required/></label>
      <label><span>Code</span><input value={code} maxLength={80} onChange={(event) => setCode(event.target.value)} required/></label>
      <label className="full"><span>Purpose</span><textarea value={purpose} maxLength={1000} onChange={(event) => setPurpose(event.target.value)} rows={2} required/></label>
    </div>

    <fieldset className="builder-panel">
      <legend>Response layout</legend>
      <div className="builder-control-grid">
        <label><span>Default layout</span><select value={presentation} onChange={(event) => setPresentation(event.target.value as CapturePresentationMode)}><option value="AUTOMATIC">Choose by form length</option><option value="CLASSIC">Show all questions</option><option value="WIZARD">Show one section at a time</option></select></label>
        <label className="compact-control"><input type="checkbox" checked={allowModeSwitch} onChange={(event) => setAllowModeSwitch(event.target.checked)}/> Allow respondents to switch layouts</label>
      </div>
    </fieldset>

    <fieldset className="builder-panel builder-sections">
      <legend>Sections</legend>
      {sections.map((section, index) => <article className="section-editor" key={section.id}>
        <div className="section-editor-heading"><strong>{section.title.trim() || `Section ${index + 1}`}</strong><div className="builder-row-actions">
          <button className="text-button" type="button" disabled={index === 0} onClick={() => moveSection(index, -1)} aria-label={`Move ${section.title.trim() || `Section ${index + 1}`} up`}>Move up</button>
          <button className="text-button" type="button" disabled={index === sections.length - 1} onClick={() => moveSection(index, 1)} aria-label={`Move ${section.title.trim() || `Section ${index + 1}`} down`}>Move down</button>
          {sections.length > 1 && <button className="text-button danger-text" type="button" onClick={() => removeSection(section.id)}>Remove</button>}
        </div></div>
        <div className="builder-control-grid">
          <label><span>Section title</span><input value={section.title} maxLength={200} onChange={(event) => setSections((current) => current.map((item) => item.id === section.id ? { ...item, title: event.target.value } : item))} required/></label>
          <label><span>Section guidance</span><input value={section.help ?? ""} maxLength={1000} onChange={(event) => setSections((current) => current.map((item) => item.id === section.id ? { ...item, help: event.target.value } : item))}/></label>
        </div>
      </article>)}
      <button className="secondary-button builder-add-button" type="button" disabled={sections.length >= 20} onClick={addSection}>Add section</button>
    </fieldset>

    <fieldset className="question-list">
      <legend>Questions</legend>
      {fields.map((field, index) => <FieldEditor key={field.id} field={field} index={index} sections={sections} earlierFields={fields.slice(0, index)} onChange={(change) => updateField(index, change)} onTypeChange={(type) => changeFieldType(index, type)} onConstraint={(key, value) => updateConstraint(index, key, value)} onRemove={() => removeField(index)} removable={fields.length > 1}/>) }
      <div className="builder-add-actions"><button className="secondary-button" type="button" disabled={fields.length >= 200} onClick={() => addField("short_text")}>Add question</button><button className="secondary-button" type="button" disabled={fields.length >= 200} onClick={() => addField("yes_no")}>Add Yes/No question</button></div>
    </fieldset>

    <section className="builder-preview" aria-labelledby="form-preview-title">
      <div className="section-editor-heading"><div><h5 id="form-preview-title">Response preview</h5><p>Check the question layout before saving the draft.</p></div><div className="builder-row-actions" role="group" aria-label="Preview layout"><button className="secondary-button" type="button" aria-pressed={previewMode === "CLASSIC"} onClick={() => setPreviewMode("CLASSIC")}>Preview Classic</button><button className="secondary-button" type="button" aria-pressed={previewMode === "WIZARD"} onClick={() => setPreviewMode("WIZARD")}>Preview Wizard</button></div></div>
      {previewMode && <div className="form-builder-preview"><CaptureForm contract={previewContract} answers={previewAnswers} attachments={{}} mode={previewMode} external uploadingField={null} onAnswer={(fieldID, value) => setPreviewAnswers((current) => ({ ...current, [fieldID]: value }))} onUpload={() => undefined} onRemoveAttachment={() => undefined} onModeChange={setPreviewMode} onReview={() => undefined}/></div>}
    </section>

    {error && <p className="inline-form-error" role="alert">{error}</p>}
    <div className="monitoring-form-actions"><button className="text-button" type="button" onClick={onCancel}>Cancel</button><button className="primary-button" type="submit" disabled={saving}>{saving ? "Saving…" : "Save draft"}</button></div>
  </form>;
}

function FieldEditor({ field, index, sections, earlierFields, onChange, onTypeChange, onConstraint, onRemove, removable }: {
  field: FieldDraft;
  index: number;
  sections: SectionDraft[];
  earlierFields: FieldDraft[];
  onChange: (change: Partial<FieldDraft>) => void;
  onTypeChange: (type: FormFieldType) => void;
  onConstraint: (key: keyof CaptureFieldConstraints, value: number | string | undefined) => void;
  onRemove: () => void;
  removable: boolean;
}) {
  const condition = field.condition;
  return <article className="question-editor typed-question-editor">
    <div className="question-editor-heading"><div className="question-number">{index + 1}</div><strong>{field.label.trim() || `Question ${index + 1}`}</strong>{removable && <button className="text-button danger-text" type="button" onClick={onRemove}>Remove</button>}</div>
    <div className="builder-control-grid question-core-fields">
      <label className="full"><span>Question</span><input aria-label="Question" value={field.label} maxLength={200} onChange={(event) => onChange({ label: event.target.value })} required/></label>
      <label><span>Response type</span><select value={field.type} onChange={(event) => onTypeChange(event.target.value as FormFieldType)}>{fieldTypes.map((type) => <option value={type.value} key={type.value}>{type.label}</option>)}</select></label>
      <label><span>Section</span><select value={field.section_id} onChange={(event) => onChange({ section_id: event.target.value })}>{sections.map((section, sectionIndex) => <option value={section.id} key={section.id}>{section.title.trim() || `Section ${sectionIndex + 1}`}</option>)}</select></label>
      <label className="full"><span>Response guidance</span><input value={field.description ?? ""} maxLength={1000} onChange={(event) => onChange({ description: event.target.value })}/></label>
      <label className="compact-control"><input type="checkbox" checked={field.required} onChange={(event) => onChange({ required: event.target.checked })}/> Required response</label>
    </div>

    <TypeSettings field={field} onChange={onChange} onConstraint={onConstraint}/>

    <fieldset className="builder-subpanel condition-editor">
      <legend>Display rule</legend>
      <div className="builder-control-grid">
        <label><span>Show this question when</span><select value={condition?.field_id ?? ""} onChange={(event) => onChange({ condition: event.target.value ? { field_id: event.target.value, operator: "EQUALS", values: [""] } : undefined })}><option value="">Always shown</option>{earlierFields.map((candidate, candidateIndex) => <option value={candidate.id} key={candidate.id}>{candidate.label.trim() || `Question ${candidateIndex + 1}`}</option>)}</select></label>
        {condition && <label><span>Condition</span><select value={condition.operator} onChange={(event) => onChange({ condition: { ...condition, operator: event.target.value as CaptureVisibilityCondition["operator"], values: event.target.value === "ANSWERED" ? undefined : condition.values?.length ? condition.values : [""] } })}><option value="EQUALS">Answer is</option><option value="NOT_EQUALS">Answer is not</option><option value="IN">Answer is one of</option><option value="NOT_IN">Answer is not one of</option><option value="ANSWERED">Has an answer</option></select></label>}
        {condition && condition.operator !== "ANSWERED" && <label className="full"><span>{condition.operator === "IN" || condition.operator === "NOT_IN" ? "Condition values" : "Condition value"}</span><input aria-label={condition.operator === "IN" || condition.operator === "NOT_IN" ? "Condition values" : "Condition value"} value={(condition.values ?? []).join(", ")} onChange={(event) => onChange({ condition: { ...condition, values: splitValues(event.target.value) } })}/></label>}
      </div>
    </fieldset>
  </article>;
}

function TypeSettings({ field, onChange, onConstraint }: { field: FieldDraft; onChange: (change: Partial<FieldDraft>) => void; onConstraint: (key: keyof CaptureFieldConstraints, value: number | string | undefined) => void }) {
  const type = field.type;
  if (textType(type)) return <fieldset className="builder-subpanel"><legend>Response limits</legend><div className="builder-control-grid"><NumberInput label="Minimum characters" value={field.constraints?.min_length} min={0} max={5000} onChange={(value) => onConstraint("min_length", value)}/><NumberInput label="Maximum characters" value={field.constraints?.max_length} min={0} max={5000} onChange={(value) => onConstraint("max_length", value)}/></div></fieldset>;
  if (numericType(type)) return <fieldset className="builder-subpanel"><legend>Response limits</legend><div className="builder-control-grid"><NumberInput label="Minimum value" value={field.constraints?.minimum} onChange={(value) => onConstraint("minimum", value)}/><NumberInput label="Maximum value" value={field.constraints?.maximum} onChange={(value) => onConstraint("maximum", value)}/><NumberInput label="Step" value={field.constraints?.step} min={0} onChange={(value) => onConstraint("step", value)}/>{type !== "integer" && <NumberInput label="Decimal places" value={field.constraints?.decimal_precision} min={0} max={6} onChange={(value) => onConstraint("decimal_precision", value)}/>} {type === "currency" && <label><span>Currency</span><select value={field.constraints?.currency ?? "NGN"} onChange={(event) => onConstraint("currency", event.target.value)}><option>NGN</option><option>USD</option><option>EUR</option><option>GBP</option></select></label>}</div></fieldset>;
  if (type === "date") return <fieldset className="builder-subpanel"><legend>Response limits</legend><div className="builder-control-grid"><label><span>Earliest date</span><input type="date" value={field.constraints?.min_date ?? ""} onChange={(event) => onConstraint("min_date", event.target.value)}/></label><label><span>Latest date</span><input type="date" value={field.constraints?.max_date ?? ""} onChange={(event) => onConstraint("max_date", event.target.value)}/></label></div></fieldset>;
  if (selectionType(type)) return <><fieldset className="builder-subpanel"><legend>Choices</legend>{type === "yes_no" ? <p className="field-note">Respondents choose Yes or No.</p> : <label><span>Choices</span><textarea aria-label="Choices" rows={4} value={(field.options ?? []).join("\n")} onChange={(event) => onChange({ options: splitLines(event.target.value).slice(0, 50) })}/></label>}{type === "multi_select" && <div className="builder-control-grid"><NumberInput label="Minimum selections" value={field.constraints?.min_selections} min={0} max={field.options?.length ?? 50} onChange={(value) => onConstraint("min_selections", value)}/><NumberInput label="Maximum selections" value={field.constraints?.max_selections} min={0} max={field.options?.length ?? 50} onChange={(value) => onConstraint("max_selections", value)}/></div>}</fieldset>{type === "yes_no" && <fieldset className="builder-subpanel"><legend>Risk scoring</legend><label className="compact-control"><input type="checkbox" checked={field.scored} onChange={(event) => onChange({ scored: event.target.checked })}/> Include response in the risk score</label>{field.scored && <div className="builder-control-grid"><NumberInput label="Weight" value={field.weight} min={1} max={100} onChange={(value) => onChange({ weight: value ?? 1 })}/><NumberInput label="Risk when No" value={field.riskWhenNo} min={0} max={100} onChange={(value) => onChange({ riskWhenNo: value ?? 0 })}/><label className="compact-control"><input type="checkbox" checked={field.criticalNo} onChange={(event) => onChange({ criticalNo: event.target.checked })}/> Treat a No response as critical</label></div>}</fieldset>}</>;
  if (type === "attestation") return <fieldset className="builder-subpanel"><legend>Attestation</legend><label><span>Statement to confirm</span><textarea value={field.attestation ?? ""} maxLength={1000} rows={3} onChange={(event) => onChange({ attestation: event.target.value })} required/></label></fieldset>;
  if (fileType(type)) {
    const available = type === "photo" ? formatOptions.filter((format) => format.value.startsWith("image/")) : type === "signature" ? formatOptions.filter((format) => format.value === "image/png") : formatOptions;
    return <><fieldset className="builder-subpanel"><legend>Accepted files</legend><div className="format-options">{available.map((format) => <label className="compact-control" key={format.value}><input type="checkbox" checked={field.accepted_formats?.includes(format.value) ?? false} disabled={type === "signature"} onChange={(event) => onChange({ accepted_formats: event.target.checked ? [...(field.accepted_formats ?? []), format.value] : (field.accepted_formats ?? []).filter((value) => value !== format.value) })}/>{format.label}</label>)}</div></fieldset><fieldset className="builder-subpanel"><legend>Response limits</legend><div className="builder-control-grid"><NumberInput label="Minimum files" value={field.constraints?.min_files} min={0} max={10} disabled={type === "photo" || type === "signature" || type === "vendor_document"} onChange={(value) => onConstraint("min_files", value)}/><NumberInput label="Maximum files" value={field.constraints?.max_files} min={1} max={10} disabled={type === "photo" || type === "signature" || type === "vendor_document"} onChange={(value) => onConstraint("max_files", value)}/><NumberInput label="Maximum file size (MB)" value={field.constraints?.max_file_bytes ? field.constraints.max_file_bytes / (1024 * 1024) : undefined} min={1} max={100} onChange={(value) => onConstraint("max_file_bytes", value ? value * 1024 * 1024 : undefined)}/><NumberInput label="Combined file limit (MB)" value={field.constraints?.max_total_file_bytes ? field.constraints.max_total_file_bytes / (1024 * 1024) : undefined} min={1} max={500} onChange={(value) => onConstraint("max_total_file_bytes", value ? value * 1024 * 1024 : undefined)}/></div></fieldset></>;
  }
  return null;
}

function NumberInput({ label, value, min, max, disabled, onChange }: { label: string; value?: number; min?: number; max?: number; disabled?: boolean; onChange: (value: number | undefined) => void }) {
  return <label><span>{label}</span><input type="number" value={value ?? ""} min={min} max={max} disabled={disabled} onChange={(event) => onChange(event.target.value === "" ? undefined : Number(event.target.value))}/></label>;
}

function validateDraft(code: string, name: string, purpose: string, sections: SectionDraft[], fields: FieldDraft[]) {
  if (!code.trim() || !name.trim() || !purpose.trim()) return "Enter a code, name and purpose before saving the form draft.";
  if (sections.some((section) => !section.title.trim())) return "Enter a title for every section before saving the form draft.";
  if (fields.some((field) => !field.label.trim())) return "Enter every question before saving the form draft.";
  for (const field of fields) {
    if ((field.type === "single_select" || field.type === "multi_select") && (field.options?.length ?? 0) < 2) return `${field.label} requires at least two choices.`;
    if (field.type === "attestation" && !field.attestation?.trim()) return `${field.label} requires an attestation statement.`;
    if (fileType(field.type) && (field.accepted_formats?.length ?? 0) === 0) return `${field.label} requires at least one accepted file type.`;
	if (fileType(field.type) && field.constraints?.min_files !== undefined && field.constraints?.max_files !== undefined && field.constraints.min_files > field.constraints.max_files) return `${field.label} has a minimum file count above its maximum.`;
	if (fileType(field.type) && field.constraints?.max_file_bytes !== undefined && field.constraints?.max_total_file_bytes !== undefined && field.constraints.max_total_file_bytes < field.constraints.max_file_bytes) return `${field.label} has a combined limit below its per-file limit.`;
    if (field.condition?.operator !== "ANSWERED" && field.condition && !(field.condition.values ?? []).some((value) => value.trim())) return `${field.label} requires a condition value.`;
  }
  return "";
}

function cleanSection(section: SectionDraft): SectionDraft {
  const help = section.help?.trim();
  return { id: section.id, title: section.title.trim(), ...(help ? { help } : {}) };
}

function cleanField(field: FieldDraft): FormTemplateField {
  const constraints = compactConstraints(field.constraints);
  const description = field.description?.trim();
  const condition = field.condition ? {
    field_id: field.condition.field_id,
    operator: field.condition.operator,
    ...(field.condition.operator === "ANSWERED" ? {} : { values: (field.condition.values ?? []).map((value) => value.trim()).filter(Boolean) }),
  } : undefined;
  return {
    id: field.id,
    section_id: field.section_id,
    label: field.label.trim(),
    type: field.type,
    required: field.required,
    ...(description ? { description } : {}),
    ...(selectionType(field.type) ? { options: field.type === "yes_no" ? ["Yes", "No"] : field.options?.map((option) => option.trim()).filter(Boolean) } : {}),
    ...(fileType(field.type) && field.accepted_formats?.length ? { accepted_formats: field.accepted_formats } : {}),
    ...(field.type === "attestation" ? { attestation: field.attestation?.trim() } : {}),
    ...(constraints ? { constraints } : {}),
    ...(condition ? { condition } : {}),
    ...(field.type === "yes_no" && field.scored ? { scoring: { weight: field.weight, answer_scores: { Yes: 0, No: field.riskWhenNo }, critical_answers: field.criticalNo ? ["No"] : [] } } : {}),
  };
}

function previewField(field: FieldDraft, index: number): FormTemplateField {
  return { ...cleanField({ ...field, label: field.label.trim() || `Question ${index + 1}` }), options: selectionType(field.type) ? field.type === "yes_no" ? ["Yes", "No"] : field.options?.filter(Boolean).length ? field.options.filter(Boolean) : ["Option 1", "Option 2"] : undefined };
}

function compactConstraints(value?: CaptureFieldConstraints) {
  if (!value) return undefined;
  const entries = Object.entries(value).filter(([, item]) => item !== undefined && item !== "");
  return entries.length ? Object.fromEntries(entries) as CaptureFieldConstraints : undefined;
}

function normalizedCode(value: string) {
  return value.trim().toUpperCase().replace(/[^A-Z0-9]+/g, "-").replace(/^-|-$/g, "");
}

function splitLines(value: string) {
  return value.split(/\r?\n/).map((item) => item.trim()).filter(Boolean);
}

function splitValues(value: string) {
  return value.split(",").map((item) => item.trim());
}

function textType(type: FormFieldType) { return ["short_text", "long_text", "email", "telephone", "url"].includes(type); }
function numericType(type: FormFieldType) { return ["integer", "decimal", "percentage", "currency"].includes(type); }
function selectionType(type: FormFieldType) { return ["yes_no", "single_select", "multi_select"].includes(type); }
function fileType(type: FormFieldType) { return ["file", "photo", "signature", "vendor_document"].includes(type); }

function initialFormats(type: FormFieldType) {
  if (type === "photo") return ["image/jpeg", "image/png"];
  if (type === "signature") return ["image/png"];
  if (type === "file" || type === "vendor_document") return ["application/pdf"];
  return undefined;
}

function initialConstraints(type: FormFieldType): CaptureFieldConstraints | undefined {
  if (type === "currency") return { currency: "NGN" };
  if (type === "percentage") return { minimum: 0, maximum: 100 };
	if (type === "photo" || type === "signature" || type === "vendor_document") return { max_files: 1 };
	if (type === "file") return { max_files: 1 };
  return undefined;
}
