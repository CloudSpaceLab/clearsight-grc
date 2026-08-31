import type { FormScoringMode } from "../../formsTypes";
import type { FormFieldType } from "../../monitoringTypes";
import type { CaptureFieldConstraints, CaptureVisibilityCondition } from "../../types";
import {
  approvedFormats,
  fieldTypes,
  fileType,
  isSelectionType,
  normalizeOptionText,
  numericType,
  textType,
  type AuthoringField,
  type AuthoringSection,
} from "./formAuthoring";
import { SelectField, type SelectOption } from "../ui";

type Props = {
  field: AuthoringField;
  index: number;
  scoringMode: FormScoringMode;
  sections: AuthoringSection[];
  earlierFields: AuthoringField[];
  onChange: (change: Partial<AuthoringField>) => void;
  onTypeChange: (type: FormFieldType) => void;
  onConstraint: (key: keyof CaptureFieldConstraints, value: number | string | undefined) => void;
  onScoringToggle: (enabled: boolean) => void;
  onMove: (offset: -1 | 1) => void;
  onRemove: () => void;
  removable: boolean;
  first: boolean;
  last: boolean;
  inspector?: boolean;
};

export function FormFieldPropertyEditor(props: Props) {
  if (props.inspector) return <InspectorEditor {...props}/>;
  return <LegacyEditor {...props}/>;
}

function InspectorEditor({ field, index, scoringMode, sections, earlierFields, onChange, onTypeChange, onConstraint, onScoringToggle, onMove, onRemove, removable, first, last }: Props) {
  return <div className="question-inspector-editor">
    <div className="form-inspector-primary-settings">
      <label className="compact-control form-inspector-required"><input type="checkbox" checked={field.required} onChange={(event) => onChange({ required: event.target.checked })}/> Required response</label>
      <EditorSelect label="Section" value={field.section_id ?? sections[0]?.id ?? ""} options={sections.map((section, sectionIndex) => ({ id: section.id, label: section.title.trim() || `Section ${sectionIndex + 1}` }))} onChange={(value) => onChange({ section_id: value })}/>
      <EditorSelect label="Inspector response type" value={field.type} options={fieldTypes.map((type) => ({ id: type.value, label: type.label }))} onChange={(value) => onTypeChange(value as FormFieldType)}/>
    </div>

    <details open className="form-inspector-disclosure">
      <summary>Validation & response options</summary>
      <div className="form-inspector-disclosure-body"><TypeSettings field={field} scoringMode={scoringMode} onChange={onChange} onConstraint={onConstraint} onScoringToggle={onScoringToggle}/></div>
    </details>

    <details className="form-inspector-disclosure">
      <summary>Logic{field.condition ? <span>Configured</span> : null}</summary>
      <div className="form-inspector-disclosure-body"><ConditionSettings field={field} earlierFields={earlierFields} onChange={onChange}/></div>
    </details>

    <details className="form-inspector-disclosure">
      <summary>Data handling{(field.collection_intent ?? "CAPTURE") !== "CAPTURE" || field.browser_cache_policy === "NO_BROWSER_CACHE" ? <span>Custom</span> : null}</summary>
      <div className="form-inspector-disclosure-body"><CollectionSettings field={field} onChange={onChange}/></div>
    </details>

    <details className="form-inspector-disclosure">
      <summary>Question actions</summary>
      <div className="form-inspector-actions">
        <button type="button" disabled={first} onClick={() => onMove(-1)} aria-label={`Move ${field.label.trim() || `Question ${index + 1}`} up`}>Move up</button>
        <button type="button" disabled={last} onClick={() => onMove(1)} aria-label={`Move ${field.label.trim() || `Question ${index + 1}`} down`}>Move down</button>
        {removable && <button type="button" className="danger-text" onClick={onRemove}>Delete question</button>}
      </div>
    </details>
  </div>;
}

function LegacyEditor({ field, index, scoringMode, sections, earlierFields, onChange, onTypeChange, onConstraint, onScoringToggle, onMove, onRemove, removable, first, last }: Props) {
  return <article className="question-editor typed-question-editor">
    <div className="question-editor-heading"><div className="question-number">{index + 1}</div><strong>{field.label.trim() || `Question ${index + 1}`}</strong><div className="builder-row-actions"><button className="text-button" type="button" disabled={first} onClick={() => onMove(-1)} aria-label={`Move ${field.label.trim() || `Question ${index + 1}`} up`}>Up</button><button className="text-button" type="button" disabled={last} onClick={() => onMove(1)} aria-label={`Move ${field.label.trim() || `Question ${index + 1}`} down`}>Down</button>{removable && <button className="text-button danger-text" type="button" onClick={onRemove}>Remove</button>}</div></div>
    <div className="builder-control-grid question-core-fields">
      <label className="full"><span>Question</span><input aria-label="Question" value={field.label} maxLength={200} onChange={(event) => onChange({ label: event.target.value })} required/></label>
      <EditorSelect label="Response type" value={field.type} options={fieldTypes.map((type) => ({ id: type.value, label: type.label }))} onChange={(value) => onTypeChange(value as FormFieldType)}/>
      <EditorSelect label="Section" value={field.section_id ?? sections[0]?.id ?? ""} options={sections.map((section, sectionIndex) => ({ id: section.id, label: section.title.trim() || `Section ${sectionIndex + 1}` }))} onChange={(value) => onChange({ section_id: value })}/>
      <label className="full"><span>Response guidance</span><input value={field.description ?? ""} maxLength={1000} onChange={(event) => onChange({ description: event.target.value })}/></label>
      <label className="compact-control"><input type="checkbox" checked={field.required} onChange={(event) => onChange({ required: event.target.checked })}/> Required response</label>
    </div>
    <TypeSettings field={field} scoringMode={scoringMode} onChange={onChange} onConstraint={onConstraint} onScoringToggle={onScoringToggle}/>
    <fieldset className="builder-subpanel"><legend>Data handling</legend><CollectionSettings field={field} onChange={onChange}/></fieldset>
    <fieldset className="builder-subpanel condition-editor"><legend>Logic</legend><ConditionSettings field={field} earlierFields={earlierFields} onChange={onChange}/></fieldset>
  </article>;
}

function CollectionSettings({ field, onChange }: { field: AuthoringField; onChange: (change: Partial<AuthoringField>) => void }) {
  return <div className="builder-control-grid">
    <EditorSelect label="Collection purpose" value={field.collection_intent ?? "CAPTURE"} options={[{ id: "CAPTURE", label: "Capture a response" }, { id: "CONFIRM_OR_CORRECT", label: "Confirm or correct a held value" }, { id: "REPLACE_HELD_DOCUMENT", label: "Replace a held document" }]} onChange={(value) => onChange({ collection_intent: value as AuthoringField["collection_intent"], ...(value === "CAPTURE" ? { record_target: undefined } : {}) })}/>
    <details className="form-inspector-technical"><summary>Technical recovery</summary><EditorSelect label="Browser recovery" value={field.browser_cache_policy ?? "ALLOWED"} options={[{ id: "ALLOWED", label: "Allow encrypted browser recovery" }, { id: "NO_BROWSER_CACHE", label: "Do not cache in browser" }]} onChange={(value) => onChange({ browser_cache_policy: value as AuthoringField["browser_cache_policy"] })}/></details>
    {(field.collection_intent ?? "CAPTURE") !== "CAPTURE" && <><label><span>Record target key</span><input value={field.record_target?.key ?? ""} placeholder="VENDOR.REGISTRATION_NUMBER" onChange={(event) => onChange({ record_target: { key: event.target.value, required_subject_type: field.record_target?.required_subject_type ?? "VENDOR_RELATIONSHIP" } })}/></label><label><span>Required subject type</span><input value={field.record_target?.required_subject_type ?? "VENDOR_RELATIONSHIP"} onChange={(event) => onChange({ record_target: { key: field.record_target?.key ?? "", required_subject_type: event.target.value } })}/></label></>}
  </div>;
}

function ConditionSettings({ field, earlierFields, onChange }: { field: AuthoringField; earlierFields: AuthoringField[]; onChange: (change: Partial<AuthoringField>) => void }) {
  const condition = field.condition;
  return <div className="builder-control-grid">
    <SelectField label="Show this question when" value={condition?.field_id || undefined} placeholder="Always shown" allowsEmpty options={earlierFields.map((candidate, candidateIndex) => ({ id: candidate.id, label: candidate.label.trim() || `Question ${candidateIndex + 1}` }))} onChange={(value) => onChange({ condition: value ? { field_id: value, operator: "EQUALS", values: [""] } : undefined })}/>
    {condition && <EditorSelect
      label="Condition"
      value={condition.operator}
      options={[{ id: "EQUALS", label: "Answer is" }, { id: "NOT_EQUALS", label: "Answer is not" }, { id: "IN", label: "Answer is one of" }, { id: "NOT_IN", label: "Answer is not one of" }, { id: "ANSWERED", label: "Has an answer" }]}
      onChange={(value) => onChange({ condition: { ...condition, operator: value as CaptureVisibilityCondition["operator"], values: value === "ANSWERED" ? undefined : condition.values?.length ? condition.values : [""] } })}
    />}
    {condition && condition.operator !== "ANSWERED" && <label className="full"><span>{condition.operator === "IN" || condition.operator === "NOT_IN" ? "Condition values" : "Condition value"}</span><input aria-label={condition.operator === "IN" || condition.operator === "NOT_IN" ? "Condition values" : "Condition value"} value={(condition.values ?? []).join(", ")} onChange={(event) => onChange({ condition: { ...condition, values: event.target.value.split(",").map((value) => value.trim()) } })}/></label>}
    {!condition && <p className="field-note">This question is always shown. Add logic only when the response should depend on an earlier answer.</p>}
  </div>;
}

function TypeSettings({ field, scoringMode, onChange, onConstraint, onScoringToggle }: { field: AuthoringField; scoringMode: FormScoringMode; onChange: (change: Partial<AuthoringField>) => void; onConstraint: (key: keyof CaptureFieldConstraints, value: number | string | undefined) => void; onScoringToggle: (enabled: boolean) => void }) {
  const type = field.type;
  return <>
    {textType(type) && <fieldset className="builder-subpanel"><legend>Response limits</legend><div className="builder-control-grid"><NumberInput label="Minimum characters" value={field.constraints?.min_length} min={0} max={5000} onChange={(value) => onConstraint("min_length", value)}/><NumberInput label="Maximum characters" value={field.constraints?.max_length} min={0} max={5000} onChange={(value) => onConstraint("max_length", value)}/></div></fieldset>}
    {numericType(type) && <fieldset className="builder-subpanel"><legend>Response limits</legend><div className="builder-control-grid"><NumberInput label="Minimum value" value={field.constraints?.minimum} onChange={(value) => onConstraint("minimum", value)}/><NumberInput label="Maximum value" value={field.constraints?.maximum} onChange={(value) => onConstraint("maximum", value)}/><NumberInput label="Step" value={field.constraints?.step} min={0.000001} onChange={(value) => onConstraint("step", value)}/>{type !== "integer" && <NumberInput label="Decimal places" value={field.constraints?.decimal_precision} min={0} max={6} onChange={(value) => onConstraint("decimal_precision", value)}/>} {type === "currency" && <EditorSelect label="Currency" value={field.constraints?.currency ?? "NGN"} options={["NGN", "USD", "EUR", "GBP"].map((id) => ({ id, label: id }))} onChange={(value) => onConstraint("currency", value)}/>}</div></fieldset>}
    {type === "date" && <fieldset className="builder-subpanel"><legend>Response limits</legend><div className="builder-control-grid"><label><span>Earliest date</span><input type="date" value={field.constraints?.min_date ?? ""} onChange={(event) => onConstraint("min_date", event.target.value)}/></label><label><span>Latest date</span><input type="date" value={field.constraints?.max_date ?? ""} onChange={(event) => onConstraint("max_date", event.target.value)}/></label></div></fieldset>}
    {isSelectionType(type) && <fieldset className="builder-subpanel"><legend>Choices</legend>{type === "yes_no" ? <p className="field-note">Respondents choose Yes or No.</p> : <label><span>Choices</span><textarea aria-label="Choices" rows={4} value={(field.options ?? []).join("\n")} onChange={(event) => onChange({ options: normalizeOptionText(event.target.value) })} onPaste={(event) => { const pasted = event.clipboardData.getData("text"); if (/[\r\n\t]/.test(pasted)) { event.preventDefault(); onChange({ options: normalizeOptionText(`${(field.options ?? []).join("\n")}\n${pasted}`) }); } }}/></label>}{type === "multi_select" && <div className="builder-control-grid"><NumberInput label="Minimum selections" value={field.constraints?.min_selections} min={0} max={field.options?.length ?? 50} onChange={(value) => onConstraint("min_selections", value)}/><NumberInput label="Maximum selections" value={field.constraints?.max_selections} min={0} max={field.options?.length ?? 50} onChange={(value) => onConstraint("max_selections", value)}/></div>}</fieldset>}
    {type === "attestation" && <fieldset className="builder-subpanel"><legend>Attestation</legend><label><span>Statement to confirm</span><textarea value={field.attestation ?? ""} maxLength={1000} rows={3} onChange={(event) => onChange({ attestation: event.target.value })} required/></label></fieldset>}
    {fileType(type) && <FileSettings field={field} onChange={onChange} onConstraint={onConstraint}/>} 
    {isSelectionType(type) && scoringMode !== "NONE" && <ScoringSettings field={field} scoringMode={scoringMode} onChange={onChange} onToggle={onScoringToggle}/>} 
    {!textType(type) && !numericType(type) && type !== "date" && !isSelectionType(type) && type !== "attestation" && !fileType(type) && <p className="field-note">This response type has no additional validation options.</p>}
  </>;
}

function ScoringSettings({ field, scoringMode, onChange, onToggle }: { field: AuthoringField; scoringMode: FormScoringMode; onChange: (change: Partial<AuthoringField>) => void; onToggle: (enabled: boolean) => void }) {
  const options = field.type === "yes_no" ? ["Yes", "No"] : field.options ?? [];
  return <fieldset className="builder-subpanel"><legend>{scoringMode === "COMPLIANCE" ? "Compliance scoring" : "Risk scoring"}</legend><label className="compact-control"><input type="checkbox" checked={Boolean(field.scoring)} onChange={(event) => onToggle(event.target.checked)}/> Include response in the {scoringMode === "COMPLIANCE" ? "compliance" : "risk"} score</label>{field.scoring && <div className="builder-score-grid"><NumberInput label="Question weight (%)" value={field.scoring.weight} min={1} max={100} onChange={(value) => onChange({ scoring: { ...field.scoring!, weight: value ?? 1 } })}/>{options.map((option) => <NumberInput key={option} label={`${option} score`} value={field.scoring!.answer_scores[option] ?? 0} min={0} max={100} onChange={(value) => onChange({ scoring: { ...field.scoring!, answer_scores: { ...field.scoring!.answer_scores, [option]: value ?? 0 } } })}/>) }<div className="builder-critical-options"><span>Critical answers</span>{options.map((option) => <label className="compact-control" key={option}><input type="checkbox" checked={field.scoring!.critical_answers?.includes(option) ?? false} onChange={(event) => onChange({ scoring: { ...field.scoring!, critical_answers: event.target.checked ? [...(field.scoring!.critical_answers ?? []), option] : (field.scoring!.critical_answers ?? []).filter((answer) => answer !== option) } })}/>{option}</label>)}</div></div>}</fieldset>;
}

function FileSettings({ field, onChange, onConstraint }: { field: AuthoringField; onChange: (change: Partial<AuthoringField>) => void; onConstraint: (key: keyof CaptureFieldConstraints, value: number | string | undefined) => void }) {
  const available = field.type === "photo" ? approvedFormats.filter((format) => format.value.startsWith("image/")) : field.type === "signature" ? approvedFormats.filter((format) => format.value === "image/png") : approvedFormats;
  const singleFile = field.type === "photo" || field.type === "signature" || field.type === "vendor_document";
  return <><fieldset className="builder-subpanel"><legend>Accepted files</legend><div className="format-options">{available.map((format) => <label className="compact-control" key={format.value}><input type="checkbox" checked={field.accepted_formats?.includes(format.value) ?? false} disabled={field.type === "signature"} onChange={(event) => onChange({ accepted_formats: event.target.checked ? [...(field.accepted_formats ?? []), format.value] : (field.accepted_formats ?? []).filter((value) => value !== format.value) })}/>{format.label}</label>)}</div></fieldset><fieldset className="builder-subpanel"><legend>Response limits</legend><div className="builder-control-grid"><NumberInput label="Minimum files" value={field.constraints?.min_files} min={0} max={10} disabled={singleFile} onChange={(value) => onConstraint("min_files", value)}/><NumberInput label="Maximum files" value={field.constraints?.max_files} min={1} max={10} disabled={singleFile} onChange={(value) => onConstraint("max_files", value)}/><NumberInput label="Maximum file size (MB)" value={field.constraints?.max_file_bytes ? field.constraints.max_file_bytes / (1024 * 1024) : undefined} min={1} max={100} onChange={(value) => onConstraint("max_file_bytes", value ? value * 1024 * 1024 : undefined)}/><NumberInput label="Combined file limit (MB)" value={field.constraints?.max_total_file_bytes ? field.constraints.max_total_file_bytes / (1024 * 1024) : undefined} min={1} max={500} onChange={(value) => onConstraint("max_total_file_bytes", value ? value * 1024 * 1024 : undefined)}/></div></fieldset></>;
}

function NumberInput({ label, value, min, max, disabled, onChange }: { label: string; value?: number; min?: number; max?: number; disabled?: boolean; onChange: (value: number | undefined) => void }) {
  return <label><span>{label}</span><input type="number" value={value ?? ""} min={min} max={max} disabled={disabled} onChange={(event) => onChange(event.target.value === "" ? undefined : Number(event.target.value))}/></label>;
}

function EditorSelect<T extends string>({ label, value, options, onChange }: { label: string; value: T; options: SelectOption<T>[]; onChange: (value: T) => void }) {
  return <SelectField label={label} value={value} placeholder={`Choose ${label.toLowerCase()}`} options={options} allowsEmpty={false} onChange={(next) => { if (next) onChange(next); }}/>;
}
