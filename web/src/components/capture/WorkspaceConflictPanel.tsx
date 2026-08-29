import type { CaptureWorkspaceConflict, CaptureWorkspaceConflictChoice } from "../../captureWorkspaceSync";
import type { CaptureAnswerValue, CaptureField } from "../../types";

export function WorkspaceConflictPanel({
  fields,
  conflicts,
  onResolve,
}: {
  fields: CaptureField[];
  conflicts: CaptureWorkspaceConflict[];
  onResolve: (fieldID: string, choice: CaptureWorkspaceConflictChoice) => void;
}) {
  if (conflicts.length === 0) return null;
  const labels = new Map(fields.map((field) => [field.id, field.label]));

  return <section className="inline-error" aria-labelledby="workspace-conflict-title">
    <strong id="workspace-conflict-title">Resolve changed answers</strong>
    <p>ClearSight received another update to the same response. Choose the answer to keep for each changed field.</p>
    {conflicts.map((conflict) => {
      const label = labels.get(conflict.fieldID) ?? "Changed answer";
      return <div className="capture-field" key={conflict.fieldID}>
        <strong>{label}</strong>
        <dl className="known-facts">
          <div><dt>ClearSight</dt><dd>{answerSummary(conflict.serverValue, "server")}</dd></div>
          <div><dt>Mine</dt><dd>{conflict.localOperation === "reselect" ? "Reselect file to upload" : answerSummary(conflict.localValue, "local")}</dd></div>
        </dl>
        <div className="wizard-actions">
          <button className="secondary-button" type="button" onClick={() => onResolve(conflict.fieldID, "server")}>Use ClearSight answer for {label}</button>
          <button className="primary-button" type="button" onClick={() => onResolve(conflict.fieldID, "local")}>Keep my answer for {label}</button>
        </div>
      </div>;
    })}
  </section>;
}

function answerSummary(value: CaptureAnswerValue, side: "server" | "local") {
  if (value.text) return value.text;
  if (value.values?.length) return value.values.join(", ");
  if (value.document) return side === "server" ? "Uploaded document" : "Selected document";
  if (value.artifact_ids?.length) return side === "server" ? "Uploaded file" : "Selected file";
  return "No answer";
}
