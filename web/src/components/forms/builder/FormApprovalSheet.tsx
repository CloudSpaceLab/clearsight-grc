import type { CreateFormTemplateInput } from "../../../monitoringTypes";
import { FocusedSheet } from "../../FocusedSheet";

type Props = {
  input: CreateFormTemplateInput;
  changedFromInitial: boolean;
  saving: boolean;
  onClose: () => void;
  onConfirm: () => void;
};

export function FormApprovalSheet({ input, changedFromInitial, saving, onClose, onConfirm }: Props) {
  return <FocusedSheet
    label="Send form for approval"
    closeLabel="Close approval confirmation"
    panelClassName="form-approval-sheet"
    onClose={onClose}
  >
    <div className="form-approval-content">
      <header>
        <span className="eyebrow">Governance</span>
        <h3>Ready for independent review</h3>
        <p>This does not activate the form. A different approver must review and approve the version shown here.</p>
      </header>

      <dl className="form-approval-facts">
        <div><dt>Form</dt><dd>{input.name.trim() || "Untitled form"}</dd></div>
        <div><dt>Code</dt><dd>{input.code}</dd></div>
        <div><dt>Sections</dt><dd>{input.sections.length}</dd></div>
        <div><dt>Questions</dt><dd>{input.fields.length}</dd></div>
        <div><dt>Scoring</dt><dd>{scoringLabel(input.scoring_mode)}</dd></div>
      </dl>

      <div className="form-approval-governance-note">
        <strong>{changedFromInitial ? "New draft version" : "Current draft version"}</strong>
        <p>{changedFromInitial
          ? "Your changes will be saved as a new draft version before it is submitted."
          : "No new revision is needed because the saved draft has not changed."}</p>
        <p>Activation still requires a separate approver.</p>
      </div>

      <footer>
        <button type="button" className="secondary-button" disabled={saving} onClick={onClose}>Keep editing</button>
        <button type="button" className="primary-button" disabled={saving} onClick={onConfirm}>{saving ? "Working…" : "Send for approval"}</button>
      </footer>
    </div>
  </FocusedSheet>;
}

function scoringLabel(mode: CreateFormTemplateInput["scoring_mode"]) {
  if (mode === "COMPLIANCE") return "Compliance";
  if (mode === "RISK") return "Risk";
  return "None";
}
