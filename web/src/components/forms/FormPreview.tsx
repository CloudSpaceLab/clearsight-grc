import { useEffect, useState } from "react";
import type { CaptureAnswers, CaptureFormContract } from "../../types";
import { CaptureForm } from "../capture/CaptureForm";

type Props = { contract: CaptureFormContract; initialMode?: "CLASSIC" | "WIZARD"; showModeControls?: boolean };

export function FormPreview({ contract, initialMode, showModeControls = true }: Props) {
  const [mode, setMode] = useState<"CLASSIC" | "WIZARD" | null>(initialMode ?? null);
  const [answers, setAnswers] = useState<CaptureAnswers>({});
  const [reviewing, setReviewing] = useState(false);

  useEffect(() => {
    setAnswers({});
    setReviewing(false);
  }, [contract]);

  return <section className="builder-preview form-preview-panel" aria-labelledby="form-preview-title">
    <div className="section-editor-heading"><div><h5 id="form-preview-title">Response preview</h5><p>Preview the questions and answer controls recipients will use.</p></div>{showModeControls && <div className="builder-row-actions" role="group" aria-label="Preview layout"><button className="secondary-button" type="button" aria-pressed={mode === "CLASSIC"} onClick={() => setMode("CLASSIC")}>Preview Classic</button><button className="secondary-button" type="button" aria-pressed={mode === "WIZARD"} onClick={() => setMode("WIZARD")}>Preview Wizard</button></div>}</div>
    {mode && <div className="form-document-canvas">{reviewing
      ? <section className="form-preview-review" aria-labelledby="form-preview-review-title"><h3 id="form-preview-review-title">Response review preview</h3><p>This is a template preview. No response will be submitted.</p><dl>{contract.fields.map((field) => <div key={field.id}><dt>{field.label}</dt><dd>{previewAnswer(answers[field.id])}</dd></div>)}</dl><button className="secondary-button" type="button" onClick={() => setReviewing(false)}>Return to questions</button></section>
      : <CaptureForm contract={contract} answers={answers} attachments={{}} mode={mode} external={false} uploadingField={null} onAnswer={(fieldID, value) => setAnswers((current) => ({ ...current, [fieldID]: value }))} onUpload={() => undefined} onRemoveAttachment={() => undefined} onModeChange={setMode} onReview={() => setReviewing(true)}/>}</div>}
  </section>;
}

function previewAnswer(value: CaptureAnswers[string] | undefined) {
  if (!value) return "No answer entered";
  if (value.text?.trim()) return value.text.trim();
  if (value.values?.length) return value.values.join(", ");
  if (value.artifact_ids?.length || value.document?.artifact_id) return "File selected for preview";
  return "No answer entered";
}
