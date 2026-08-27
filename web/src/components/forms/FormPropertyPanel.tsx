import { useMemo, useRef, useState } from "react";
import type { FormScoringMode, ReusableFormTemplateRef } from "../../formsTypes";
import type { FormFieldType, FormTemplate } from "../../monitoringTypes";
import type { CaptureFieldConstraints } from "../../types";
import { reusableRefLabel, type AuthoringField, type AuthoringSection } from "./formAuthoring";
import { FormFieldPropertyEditor } from "./FormFieldPropertyEditor";

type Props = {
  scoringMode: FormScoringMode;
  sections: AuthoringSection[];
  fields: AuthoringField[];
  reusableTemplates?: ReusableFormTemplateRef[];
  loadReusableTemplate?: (id: string, version: number) => Promise<FormTemplate>;
  onSectionsChange: (sections: AuthoringSection[]) => void;
  onFieldChange: (index: number, change: Partial<AuthoringField>) => void;
  onFieldTypeChange: (index: number, type: FormFieldType) => void;
  onFieldConstraint: (index: number, key: keyof CaptureFieldConstraints, value: number | string | undefined) => void;
  onFieldScoringToggle: (index: number, enabled: boolean) => void;
  onAddSection: () => void;
  onDuplicateSection: (sectionID: string) => void;
  onMoveSection: (index: number, offset: -1 | 1) => void;
  onRemoveSection: (sectionID: string) => void;
  onInsertReusableSection: (template: FormTemplate, sectionID: string) => void;
  onAddField: (type: FormFieldType) => void;
  onMoveField: (index: number, offset: -1 | 1) => void;
  onRemoveField: (index: number) => void;
};

export function FormPropertyPanel(props: Props) {
  const [selectedTemplateKey, setSelectedTemplateKey] = useState("");
  const [sourceTemplate, setSourceTemplate] = useState<FormTemplate | null>(null);
  const [sourceSectionID, setSourceSectionID] = useState("");
  const [sourceState, setSourceState] = useState<"idle" | "loading" | "error">("idle");
  const loadSequence = useRef(0);

  const reusableByKey = useMemo(() => new Map((props.reusableTemplates ?? []).map((item) => [`${item.id}:${item.version}`, item])), [props.reusableTemplates]);

  async function chooseReusableTemplate(key: string) {
    const sequence = ++loadSequence.current;
    setSelectedTemplateKey(key);
    setSourceTemplate(null);
    setSourceSectionID("");
    setSourceState("idle");
    if (!key || !props.loadReusableTemplate) return;
    const selected = reusableByKey.get(key);
    if (!selected) return;
    setSourceState("loading");
    try {
      const template = await props.loadReusableTemplate(selected.id, selected.version);
      if (sequence !== loadSequence.current) return;
      if (template.id !== selected.id || template.version !== selected.version || template.status !== "ACTIVE") {
        setSourceState("error");
        return;
      }
      setSourceTemplate(template);
      setSourceSectionID(template.sections?.[0]?.id ?? "");
      setSourceState("idle");
    } catch {
      if (sequence === loadSequence.current) setSourceState("error");
    }
  }

  return <div className="form-property-panel">
    <fieldset className="builder-panel builder-sections">
      <legend>Sections</legend>
      {props.sections.map((section, index) => <article className="section-editor" key={section.id}>
        <div className="section-editor-heading">
          <strong>{section.title.trim() || `Section ${index + 1}`}</strong>
          <div className="builder-row-actions">
            <button className="text-button" type="button" disabled={index === 0} onClick={() => props.onMoveSection(index, -1)} aria-label={`Move ${section.title.trim() || `Section ${index + 1}`} up`}>Move up</button>
            <button className="text-button" type="button" disabled={index === props.sections.length - 1} onClick={() => props.onMoveSection(index, 1)} aria-label={`Move ${section.title.trim() || `Section ${index + 1}`} down`}>Move down</button>
            <button className="text-button" type="button" disabled={props.sections.length >= 20} onClick={() => props.onDuplicateSection(section.id)} aria-label={`Duplicate ${section.title.trim() || `Section ${index + 1}`}`}>Duplicate</button>
            {props.sections.length > 1 && <button className="text-button danger-text" type="button" onClick={() => props.onRemoveSection(section.id)}>Remove</button>}
          </div>
        </div>
        <div className="builder-control-grid">
          <label><span>Section title</span><input value={section.title} maxLength={200} onChange={(event) => props.onSectionsChange(props.sections.map((item) => item.id === section.id ? { ...item, title: event.target.value } : item))} required/></label>
          <label><span>Section guidance</span><input value={section.help ?? ""} maxLength={1000} onChange={(event) => props.onSectionsChange(props.sections.map((item) => item.id === section.id ? { ...item, help: event.target.value } : item))}/></label>
          {props.scoringMode === "COMPLIANCE" && <label><span>Compliance section weight (%)</span><input type="number" value={section.weight ?? ""} min={0} max={100} onChange={(event) => props.onSectionsChange(props.sections.map((item) => item.id === section.id ? { ...item, weight: event.target.value === "" ? 0 : Number(event.target.value) } : item))}/></label>}
        </div>
      </article>)}
      <div className="builder-add-actions"><button className="secondary-button builder-add-button" type="button" disabled={props.sections.length >= 20} onClick={props.onAddSection}>Add section</button></div>

      {(props.reusableTemplates?.length ?? 0) > 0 && props.loadReusableTemplate && <details className="builder-reuse-section">
        <summary>Insert section from active template</summary>
        <div className="builder-control-grid">
          <label><span>Active template revision</span><select value={selectedTemplateKey} onChange={(event) => void chooseReusableTemplate(event.target.value)}><option value="">Choose a template</option>{props.reusableTemplates!.map((ref) => <option key={`${ref.id}:${ref.version}`} value={`${ref.id}:${ref.version}`}>{reusableRefLabel(ref)}</option>)}</select></label>
          {sourceTemplate && <label><span>Section to insert</span><select value={sourceSectionID} onChange={(event) => setSourceSectionID(event.target.value)}>{(sourceTemplate.sections ?? []).map((section) => <option key={section.id} value={section.id}>{section.title}</option>)}</select></label>}
        </div>
        {sourceState === "loading" && <p className="field-note" role="status">Loading the exact active revision…</p>}
        {sourceState === "error" && <p className="inline-form-error" role="alert">The selected revision is no longer an active reusable template. No section was inserted.</p>}
        {sourceTemplate && <button className="secondary-button" type="button" disabled={!sourceSectionID || props.sections.length >= 20} onClick={() => props.onInsertReusableSection(sourceTemplate, sourceSectionID)}>Insert section</button>}
      </details>}
    </fieldset>

    <fieldset className="question-list">
      <legend>Questions</legend>
      {props.fields.map((field, index) => {
        const sectionPeers = props.fields.filter((candidate) => candidate.section_id === field.section_id);
        const sectionPosition = sectionPeers.findIndex((candidate) => candidate.id === field.id);
        return <FormFieldPropertyEditor
          key={field.id} field={field} index={index} scoringMode={props.scoringMode} sections={props.sections} earlierFields={props.fields.slice(0, index)}
          onChange={(change) => props.onFieldChange(index, change)} onTypeChange={(type) => props.onFieldTypeChange(index, type)}
          onConstraint={(key, value) => props.onFieldConstraint(index, key, value)} onScoringToggle={(enabled) => props.onFieldScoringToggle(index, enabled)}
          onMove={(offset) => props.onMoveField(index, offset)} onRemove={() => props.onRemoveField(index)} removable={props.fields.length > 1}
          first={sectionPosition <= 0} last={sectionPosition === sectionPeers.length - 1}
        />;
      })}
      <div className="builder-add-actions"><button className="secondary-button" type="button" disabled={props.fields.length >= 200} onClick={() => props.onAddField("short_text")}>Add question</button><button className="secondary-button" type="button" disabled={props.fields.length >= 200} onClick={() => props.onAddField("yes_no")}>Add Yes/No question</button></div>
    </fieldset>
  </div>;
}
