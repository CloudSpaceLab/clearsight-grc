import { useMemo, useRef, useState } from "react";
import type { ReusableFormTemplateRef } from "../../../formsTypes";
import type { FormTemplate } from "../../../monitoringTypes";
import { reusableRefLabel } from "../formAuthoring";

type Props = {
  reusableTemplates?: ReusableFormTemplateRef[];
  loadReusableTemplate?: (id: string, version: number) => Promise<FormTemplate>;
  sectionLimitReached: boolean;
  onInsert: (template: FormTemplate, sectionID: string) => void;
};

export function ReusableSectionPicker({ reusableTemplates, loadReusableTemplate, sectionLimitReached, onInsert }: Props) {
  const [selectedTemplateKey, setSelectedTemplateKey] = useState("");
  const [sourceTemplate, setSourceTemplate] = useState<FormTemplate | null>(null);
  const [sourceSectionID, setSourceSectionID] = useState("");
  const [sourceState, setSourceState] = useState<"idle" | "loading" | "error">("idle");
  const loadSequence = useRef(0);
  const reusableByKey = useMemo(() => new Map((reusableTemplates ?? []).map((item) => [`${item.id}:${item.version}`, item])), [reusableTemplates]);

  if (!reusableTemplates?.length || !loadReusableTemplate) return null;

  async function chooseTemplate(key: string) {
    const sequence = ++loadSequence.current;
    setSelectedTemplateKey(key);
    setSourceTemplate(null);
    setSourceSectionID("");
    setSourceState("idle");
    if (!key || !loadReusableTemplate) return;
    const selected = reusableByKey.get(key);
    if (!selected) return;
    setSourceState("loading");
    try {
      const template = await loadReusableTemplate(selected.id, selected.version);
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

  return <details className="form-outline-reuse">
    <summary>Reuse approved section</summary>
    <div className="form-outline-reuse-body">
      <label><span>Active template revision</span><select value={selectedTemplateKey} onChange={(event) => void chooseTemplate(event.target.value)}><option value="">Choose a template</option>{reusableTemplates.map((ref) => <option key={`${ref.id}:${ref.version}`} value={`${ref.id}:${ref.version}`}>{reusableRefLabel(ref)}</option>)}</select></label>
      {sourceTemplate && <label><span>Section to insert</span><select value={sourceSectionID} onChange={(event) => setSourceSectionID(event.target.value)}>{(sourceTemplate.sections ?? []).map((section) => <option key={section.id} value={section.id}>{section.title}</option>)}</select></label>}
      {sourceState === "loading" && <p role="status">Loading the approved revision…</p>}
      {sourceState === "error" && <p className="inline-form-error" role="alert">That revision is no longer an active reusable template. Nothing was inserted.</p>}
      {sourceTemplate && <button type="button" className="secondary-button" disabled={!sourceSectionID || sectionLimitReached} onClick={() => onInsert(sourceTemplate, sourceSectionID)}>Insert section</button>}
    </div>
  </details>;
}
