import type { FormFieldType, FormScoringMode } from "../../../monitoringTypes";
import type { CaptureFieldConstraints } from "../../../types";
import type { AuthoringField, AuthoringSection, FormDraft } from "../formAuthoring";
import { FormFieldPropertyEditor } from "../FormFieldPropertyEditor";
import type { BuilderSelection } from "./builderSelection";

type Props = {
  draft: FormDraft;
  selection: BuilderSelection;
  onPatch: (patch: Partial<FormDraft>) => void;
  onScoringMode: (mode: FormScoringMode) => void;
  onSectionsChange: (sections: AuthoringSection[]) => void;
  onFieldChange: (index: number, change: Partial<AuthoringField>) => void;
  onFieldTypeChange: (index: number, type: FormFieldType) => void;
  onFieldConstraint: (index: number, key: keyof CaptureFieldConstraints, value: number | string | undefined) => void;
  onFieldScoringToggle: (index: number, enabled: boolean) => void;
  onMoveSection: (index: number, offset: -1 | 1) => void;
  onDuplicateSection: (sectionID: string) => void;
  onRemoveSection: (sectionID: string) => void;
  onMoveField: (index: number, offset: -1 | 1) => void;
  onRemoveField: (index: number) => void;
};

export function FormInspector(props: Props) {
  if (props.selection.kind === "overview") return <OverviewInspector {...props}/>;
  if (props.selection.kind === "section") return <SectionInspector {...props} sectionID={props.selection.sectionID}/>;
  return <QuestionInspector {...props} fieldID={props.selection.fieldID}/>;
}

function PaneHeading({ eyebrow, title, detail }: { eyebrow: string; title: string; detail?: string }) {
  return <div className="form-builder-pane-heading form-inspector-heading">
    <div><span>{eyebrow}</span><strong>{title}</strong></div>
    {detail && <small>{detail}</small>}
  </div>;
}

function OverviewInspector({ draft, onPatch, onScoringMode }: Props) {
  return <aside className="form-builder-inspector" aria-label="Form settings">
    <PaneHeading eyebrow="Inspector" title="Form settings"/>
    <div className="form-inspector-content">
      <label><span>Code</span><input aria-label="Code" value={draft.code} maxLength={80} placeholder="VENDOR-SECURITY-REVIEW" onChange={(event) => onPatch({ code: event.target.value })} required/></label>
      <details open>
        <summary>Response experience</summary>
        <div className="form-inspector-group">
          <label><span>Default layout</span><select value={draft.presentation} onChange={(event) => onPatch({ presentation: event.target.value as FormDraft["presentation"] })}><option value="AUTOMATIC">Choose by form length</option><option value="CLASSIC">Show all questions</option><option value="WIZARD">Show one section at a time</option></select></label>
          <label className="compact-control"><input type="checkbox" aria-label="Allow respondents to switch layouts" checked={draft.allowModeSwitch} onChange={(event) => onPatch({ allowModeSwitch: event.target.checked })}/> Allow respondents to switch layouts</label>
        </div>
      </details>
      <details>
        <summary>Scoring</summary>
        <div className="form-inspector-group">
          <label><span>Scoring mode</span><select value={draft.scoringMode} onChange={(event) => onScoringMode(event.target.value as FormScoringMode)}><option value="NONE">No score</option><option value="RISK">Risk score</option><option value="COMPLIANCE">Compliance score</option></select></label>
          <p>Scoring is optional. It remains hidden from ordinary question editing until enabled.</p>
        </div>
      </details>
    </div>
  </aside>;
}

function SectionInspector({ draft, sectionID, onSectionsChange, onMoveSection, onDuplicateSection, onRemoveSection }: Props & { sectionID: string }) {
  const index = draft.sections.findIndex((section) => section.id === sectionID);
  const section = draft.sections[index];
  if (!section) return <aside className="form-builder-inspector" aria-label="Section settings"><PaneHeading eyebrow="Inspector" title="Section unavailable"/></aside>;

  function change(change: Partial<AuthoringSection>) {
    onSectionsChange(draft.sections.map((candidate) => candidate.id === section.id ? { ...candidate, ...change } : candidate));
  }

  return <aside className="form-builder-inspector" aria-label="Section settings">
    <PaneHeading eyebrow="Inspector" title={section.title.trim() || `Section ${index + 1}`} detail="Section settings"/>
    <div className="form-inspector-content">
      <label><span>Section title</span><input value={section.title} maxLength={200} onChange={(event) => change({ title: event.target.value })}/></label>
      <label><span>Section guidance</span><textarea value={section.help ?? ""} maxLength={1000} rows={3} onChange={(event) => change({ help: event.target.value })}/></label>
      {draft.scoringMode === "COMPLIANCE" && <label><span>Compliance weight (%)</span><input type="number" min={0} max={100} value={section.weight ?? ""} onChange={(event) => change({ weight: event.target.value === "" ? 0 : Number(event.target.value) })}/></label>}
      <details>
        <summary>Section actions</summary>
        <div className="form-inspector-actions">
          <button type="button" disabled={index === 0} onClick={() => onMoveSection(index, -1)}>Move up</button>
          <button type="button" disabled={index === draft.sections.length - 1} onClick={() => onMoveSection(index, 1)}>Move down</button>
          <button type="button" disabled={draft.sections.length >= 20} aria-label={`Duplicate ${section.title.trim() || `Section ${index + 1}`}`} onClick={() => onDuplicateSection(section.id)}>Duplicate section</button>
          {draft.sections.length > 1 && <button type="button" className="danger-text" onClick={() => onRemoveSection(section.id)}>Delete section</button>}
        </div>
      </details>
    </div>
  </aside>;
}

function QuestionInspector(props: Props & { fieldID: string }) {
  const index = props.draft.fields.findIndex((field) => field.id === props.fieldID);
  const field = props.draft.fields[index];
  if (!field) return <aside className="form-builder-inspector" aria-label="Question settings"><PaneHeading eyebrow="Inspector" title="Question unavailable"/></aside>;
  const peers = props.draft.fields.filter((candidate) => candidate.section_id === field.section_id);
  const peerIndex = peers.findIndex((candidate) => candidate.id === field.id);

  return <aside className="form-builder-inspector" aria-label="Question settings">
    <PaneHeading eyebrow="Inspector" title={field.label.trim() || `Question ${index + 1}`} detail="Question settings"/>
    <div className="form-inspector-content question-inspector-content">
      <FormFieldPropertyEditor
        field={field}
        index={index}
        scoringMode={props.draft.scoringMode}
        sections={props.draft.sections}
        earlierFields={props.draft.fields.slice(0, index)}
        onChange={(change) => props.onFieldChange(index, change)}
        onTypeChange={(type) => props.onFieldTypeChange(index, type)}
        onConstraint={(key, value) => props.onFieldConstraint(index, key, value)}
        onScoringToggle={(enabled) => props.onFieldScoringToggle(index, enabled)}
        onMove={(offset) => props.onMoveField(index, offset)}
        onRemove={() => props.onRemoveField(index)}
        removable={props.draft.fields.length > 1}
        first={peerIndex <= 0}
        last={peerIndex === peers.length - 1}
        inspector
      />
    </div>
  </aside>;
}
