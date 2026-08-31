import type { AuthoringField, AuthoringSection, FormQualityIssue } from "../formAuthoring";
import type { BuilderSelection } from "./builderSelection";
import { selectionKey } from "./builderSelection";

type Props = {
  sections: AuthoringSection[];
  fields: AuthoringField[];
  issues: FormQualityIssue[];
  selection: BuilderSelection;
  onSelect: (selection: BuilderSelection) => void;
  onAddSection: () => void;
  onAddField: () => void;
};

export function FormOutline({ sections, fields, issues, selection, onSelect, onAddSection, onAddField }: Props) {
  const selectedKey = selectionKey(selection);
  const blockingSections = new Set(issues.filter((issue) => issue.blocking && issue.sectionID).map((issue) => issue.sectionID));
  const blockingFields = new Set(issues.filter((issue) => issue.blocking && issue.fieldID).map((issue) => issue.fieldID));

  return <nav className="form-builder-outline" aria-label="Form outline">
    <div className="form-builder-pane-heading">
      <span>Outline</span>
      <small>{sections.length} {sections.length === 1 ? "section" : "sections"}</small>
    </div>

    <button
      type="button"
      className={selectedKey === "overview" ? "form-outline-item active" : "form-outline-item"}
      aria-current={selectedKey === "overview" ? "true" : undefined}
      onClick={() => onSelect({ kind: "overview" })}
    >
      <span className="form-outline-icon" aria-hidden="true">⌂</span>
      <span>Overview</span>
    </button>

    <ol className="form-outline-sections">
      {sections.map((section, sectionIndex) => {
        const sectionFields = fields.filter((field) => field.section_id === section.id);
        const sectionSelected = selectedKey === `section:${section.id}`;
        return <li key={section.id}>
          <button
            type="button"
            className={sectionSelected ? "form-outline-item section active" : "form-outline-item section"}
            aria-current={sectionSelected ? "true" : undefined}
            onClick={() => onSelect({ kind: "section", sectionID: section.id })}
          >
            <span className="form-outline-index">{sectionIndex + 1}</span>
            <span className="form-outline-label">{section.title.trim() || `Section ${sectionIndex + 1}`}</span>
            {blockingSections.has(section.id) && <span className="form-outline-issue" aria-label="Has a blocking review issue"/>}
          </button>
          <ol className="form-outline-fields">
            {sectionFields.map((field, fieldIndex) => {
              const active = selectedKey === `field:${field.id}`;
              return <li key={field.id}>
                <button
                  type="button"
                  className={active ? "form-outline-item field active" : "form-outline-item field"}
                  aria-current={active ? "true" : undefined}
                  onClick={() => onSelect({ kind: "field", fieldID: field.id })}
                >
                  <span className="form-outline-dot" aria-hidden="true"/>
                  <span className="form-outline-label">{field.label.trim() || `Question ${fieldIndex + 1}`}</span>
                  {field.condition && <span className="form-outline-conditional" aria-label="Conditional question">◇</span>}
                  {blockingFields.has(field.id) && <span className="form-outline-issue" aria-label="Has a blocking review issue"/>}
                </button>
              </li>;
            })}
          </ol>
        </li>;
      })}
    </ol>

    <div className="form-outline-actions">
      <button type="button" className="text-button" disabled={sections.length >= 20} onClick={onAddSection}>+ Section</button>
      <button type="button" className="text-button" disabled={fields.length >= 200} onClick={onAddField}>+ Question</button>
    </div>
  </nav>;
}
