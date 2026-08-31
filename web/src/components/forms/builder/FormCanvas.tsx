import { useRef } from "react";
import type { FormFieldType } from "../../../monitoringTypes";
import type { AuthoringField, AuthoringSection, FormDraft } from "../formAuthoring";
import { fieldTypes } from "../formAuthoring";
import type { BuilderSelection } from "./builderSelection";

type Props = {
  draft: FormDraft;
  selection: BuilderSelection;
  onPatch: (patch: Partial<FormDraft>) => void;
  onSectionsChange: (sections: AuthoringSection[]) => void;
  onFieldChange: (index: number, change: Partial<AuthoringField>) => void;
  onFieldTypeChange: (index: number, type: FormFieldType) => void;
  onSelect: (selection: BuilderSelection) => void;
  onAddField: (sectionID?: string) => void;
  onAddSection: () => void;
  onMoveField: (index: number, offset: -1 | 1) => void;
  onReorderField: (fromIndex: number, toIndex: number) => void;
  onDuplicateField: (index: number) => void;
  onRemoveField: (index: number) => void;
};

export function FormCanvas({ draft, selection, onPatch, onSectionsChange, onFieldChange, onFieldTypeChange, onSelect, onAddField, onAddSection, onMoveField, onReorderField, onDuplicateField, onRemoveField }: Props) {
  const pointerDrag = useRef<{ fromIndex: number; startX: number; startY: number; moved: boolean } | null>(null);

  return <main className="form-builder-canvas" aria-label="Form canvas">
    <div className="form-canvas-document">
      <header data-builder-target="overview" className={selection.kind === "overview" ? "form-canvas-header selected" : "form-canvas-header"} onClick={() => onSelect({ kind: "overview" })}>
        <input
          className="form-canvas-title"
          aria-label="Form name"
          value={draft.name}
          maxLength={200}
          placeholder="Untitled form"
          onFocus={() => onSelect({ kind: "overview" })}
          onChange={(event) => onPatch({ name: event.target.value })}
          required
        />
        <textarea
          className="form-canvas-purpose"
          aria-label="Purpose"
          value={draft.purpose}
          maxLength={1000}
          rows={2}
          placeholder="Describe why respondents are completing this form…"
          onFocus={() => onSelect({ kind: "overview" })}
          onChange={(event) => onPatch({ purpose: event.target.value })}
          required
        />
        {!draft.purpose.trim() && <small className="form-canvas-example">Example: Collect annual evidence and control attestations from critical providers.</small>}
      </header>

      <div className="form-canvas-sections">
        {draft.sections.map((section, sectionIndex) => {
          const sectionFields = draft.fields.filter((field) => field.section_id === section.id);
          const sectionSelected = selection.kind === "section" && selection.sectionID === section.id;
          return <section data-builder-section-id={section.id} className={sectionSelected ? "form-canvas-section selected" : "form-canvas-section"} key={section.id} aria-label={section.title.trim() || `Section ${sectionIndex + 1}`}>
            <div className="form-canvas-section-heading" onClick={() => onSelect({ kind: "section", sectionID: section.id })}>
              <span className="form-canvas-section-number">{String(sectionIndex + 1).padStart(2, "0")}</span>
              <div>
                <input
                  aria-label={`Section ${sectionIndex + 1} title`}
                  value={section.title}
                  maxLength={200}
                  placeholder={`Section ${sectionIndex + 1}`}
                  onFocus={() => onSelect({ kind: "section", sectionID: section.id })}
                  onChange={(event) => onSectionsChange(draft.sections.map((candidate) => candidate.id === section.id ? { ...candidate, title: event.target.value } : candidate))}
                />
                <input
                  aria-label={`Section ${sectionIndex + 1} guidance`}
                  value={section.help ?? ""}
                  maxLength={1000}
                  placeholder="Add optional guidance for this section…"
                  onFocus={() => onSelect({ kind: "section", sectionID: section.id })}
                  onChange={(event) => onSectionsChange(draft.sections.map((candidate) => candidate.id === section.id ? { ...candidate, help: event.target.value } : candidate))}
                />
              </div>
            </div>

            <div className="form-canvas-questions">
              {sectionFields.map((field) => {
                const index = draft.fields.findIndex((candidate) => candidate.id === field.id);
                const siblingPosition = sectionFields.findIndex((candidate) => candidate.id === field.id);
                const selected = selection.kind === "field" && selection.fieldID === field.id;
                return <article
                  className={selected ? "form-canvas-question selected" : "form-canvas-question"}
                  data-builder-field-id={field.id}
                  key={field.id}
                  onClick={() => onSelect({ kind: "field", fieldID: field.id })}
                >
                  <div className="form-question-topline">
                    <span
                      className="form-question-drag"
                      aria-hidden="true"
                      title={`Drag question ${index + 1} to reorder`}
                      onPointerDown={(event) => {
                        if (event.button !== 0) return;
                        pointerDrag.current = { fromIndex: index, startX: event.clientX, startY: event.clientY, moved: false };
                        event.currentTarget.setPointerCapture?.(event.pointerId);
                        onSelect({ kind: "field", fieldID: field.id });
                      }}
                      onPointerMove={(event) => {
                        const drag = pointerDrag.current;
                        if (!drag || drag.moved) return;
                        drag.moved = Math.hypot(event.clientX - drag.startX, event.clientY - drag.startY) >= 6;
                      }}
                      onPointerUp={(event) => {
                        const drag = pointerDrag.current;
                        pointerDrag.current = null;
                        if (!drag?.moved) return;
                        const target = document.elementFromPoint(event.clientX, event.clientY)?.closest<HTMLElement>("[data-builder-field-id]");
                        const toIndex = target ? draft.fields.findIndex((candidate) => candidate.id === target.dataset.builderFieldId) : -1;
                        if (toIndex >= 0 && toIndex !== drag.fromIndex) onReorderField(drag.fromIndex, toIndex);
                      }}
                      onPointerCancel={() => { pointerDrag.current = null; }}
                    >⠿</span>
                    {field.condition && <span className="form-question-conditional">◇ Conditional</span>}
                    <details className="form-question-menu" onClick={(event) => event.stopPropagation()}>
                      <summary aria-label={`Question ${index + 1} actions`}>•••</summary>
                      <div>
                        <button type="button" disabled={siblingPosition === 0} onClick={() => onMoveField(index, -1)}>Move up</button>
                        <button type="button" disabled={siblingPosition === sectionFields.length - 1} onClick={() => onMoveField(index, 1)}>Move down</button>
                        <button type="button" disabled={draft.fields.length >= 200} onClick={() => onDuplicateField(index)}>Duplicate question</button>
                        {draft.fields.length > 1 && <button type="button" className="danger-text" onClick={() => onRemoveField(index)}>Delete</button>}
                      </div>
                    </details>
                  </div>

                  <input
                    className="form-question-prompt"
                    aria-label="Question"
                    value={field.label}
                    maxLength={200}
                    placeholder="Ask a clear question…"
                    onFocus={() => onSelect({ kind: "field", fieldID: field.id })}
                    onChange={(event) => onFieldChange(index, { label: event.target.value })}
                    required
                  />
                  <input
                    className="form-question-guidance"
                    aria-label={`Question ${index + 1} response guidance`}
                    value={field.description ?? ""}
                    maxLength={1000}
                    placeholder="Add guidance for the respondent…"
                    onFocus={() => onSelect({ kind: "field", fieldID: field.id })}
                    onChange={(event) => onFieldChange(index, { description: event.target.value })}
                  />

                  <div className="form-question-controls">
                    <label>
                      <span className="sr-only">Response type</span>
                      <select aria-label="Response type" value={field.type} onFocus={() => onSelect({ kind: "field", fieldID: field.id })} onChange={(event) => onFieldTypeChange(index, event.target.value as FormFieldType)}>
                        {fieldTypes.map((type) => <option value={type.value} key={type.value}>{type.label}</option>)}
                      </select>
                    </label>
                    <label className="form-question-required"><span>Required</span><input type="checkbox" checked={field.required} onChange={(event) => onFieldChange(index, { required: event.target.checked })}/></label>
                  </div>
                </article>;
              })}
            </div>

            <button className="form-canvas-add" type="button" disabled={draft.fields.length >= 200} onClick={() => onAddField(section.id)}>+ Add question</button>
          </section>;
        })}
      </div>

      <button className="form-canvas-add-section" type="button" disabled={draft.sections.length >= 20} onClick={onAddSection}>+ Add section</button>
    </div>
  </main>;
}
