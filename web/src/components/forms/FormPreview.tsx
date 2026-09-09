import { useEffect, useState } from "react";
import type { CaptureAnswers, CaptureFormContract, CapturePresentationMode } from "../../types";
import { CaptureForm } from "../capture/CaptureForm";

type Props = { contract: CaptureFormContract; initialMode?: CapturePresentationMode };

export function FormPreview({ contract, initialMode }: Props) {
  const [mode, setMode] = useState<CapturePresentationMode>(initialMode ?? contract.presentation.default_mode);
  const [answers, setAnswers] = useState<CaptureAnswers>({});
  const [reviewing, setReviewing] = useState(false);

  useEffect(() => {
    setAnswers({});
    setReviewing(false);
    setMode(initialMode ?? contract.presentation.default_mode);
  }, [contract, initialMode]);

  return <section className="builder-preview form-preview-panel" aria-labelledby="form-preview-title">
    <div className="section-editor-heading"><div><h4 id="form-preview-title">Response preview</h4><p>Preview the questions and answer controls recipients will use.</p></div></div>
    <div className="form-document-canvas">{reviewing
      ? <section className="form-preview-review" aria-labelledby="form-preview-review-title"><h3 id="form-preview-review-title">Response review preview</h3><p>This is a template preview. No response will be submitted.</p><dl>{contract.fields.map((field) => <div key={field.id}><dt>{field.label}</dt><dd>{previewAnswer(answers[field.id])}</dd></div>)}</dl><button className="secondary-button" type="button" onClick={() => setReviewing(false)}>Return to questions</button></section>
      : <CaptureForm contract={contract} answers={answers} attachments={{}} mode={mode} external={false} uploadingField={null} onAnswer={(fieldID, value) => setAnswers((current) => ({ ...current, [fieldID]: value }))} onUpload={() => undefined} onRemoveAttachment={() => undefined} onModeChange={setMode} onReview={() => setReviewing(true)}/>}</div>
  </section>;
}

function previewAnswer(value: CaptureAnswers[string] | undefined) {
  if (!value) return "No answer entered";
  if (value.text?.trim()) return value.text.trim();
  if (value.values?.length) return value.values.join(", ");
  if (value.artifact_ids?.length || value.document?.artifact_id) return "File selected for preview";
  return "No answer entered";
}
