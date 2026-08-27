import { useEffect, useState } from "react";
import type { CaptureAnswers, CaptureFormContract } from "../../types";
import { CaptureForm } from "../capture/CaptureForm";

type Props = { contract: CaptureFormContract };

export function FormPreview({ contract }: Props) {
  const [mode, setMode] = useState<"CLASSIC" | "WIZARD" | null>(null);
  const [answers, setAnswers] = useState<CaptureAnswers>({});

  useEffect(() => {
    setAnswers({});
  }, [contract]);

  return <section className="builder-preview form-preview-panel" aria-labelledby="form-preview-title">
    <div className="section-editor-heading"><div><h5 id="form-preview-title">Response preview</h5><p>Preview the exact normalized contract used by the capture renderer.</p></div><div className="builder-row-actions" role="group" aria-label="Preview layout"><button className="secondary-button" type="button" aria-pressed={mode === "CLASSIC"} onClick={() => setMode("CLASSIC")}>Preview Classic</button><button className="secondary-button" type="button" aria-pressed={mode === "WIZARD"} onClick={() => setMode("WIZARD")}>Preview Wizard</button></div></div>
    {mode && <div className="form-document-canvas"><CaptureForm contract={contract} answers={answers} attachments={{}} mode={mode} external uploadingField={null} onAnswer={(fieldID, value) => setAnswers((current) => ({ ...current, [fieldID]: value }))} onUpload={() => undefined} onRemoveAttachment={() => undefined} onModeChange={setMode} onReview={() => undefined}/></div>}
  </section>;
}
