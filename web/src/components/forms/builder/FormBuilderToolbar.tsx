type Props = {
  title: string;
  saving: boolean;
  approvalReady: boolean;
  reviewCount: number;
  canSendForApproval: boolean;
  onCancel: () => void;
  onPreview: () => void;
  onReview: () => void;
  onSave: () => void;
  onSendForApproval?: () => void;
};

export function FormBuilderToolbar({ title, saving, approvalReady, reviewCount, canSendForApproval, onCancel, onPreview, onReview, onSave, onSendForApproval }: Props) {
  return <header className="form-builder-toolbar">
    <div className="form-builder-toolbar-leading">
      <button type="button" className="text-button form-builder-back" onClick={onCancel} aria-label="Back to Forms">‹ Forms</button>
      <span className="form-builder-toolbar-divider" aria-hidden="true"/>
      <div className="form-builder-toolbar-title">
        <strong>{title.trim() || "Untitled form"}</strong>
        <small>{saving ? "Saving…" : "Draft"}</small>
      </div>
    </div>
    <div className="form-builder-toolbar-actions">
      <button type="button" className="text-button" onClick={onPreview}>Preview</button>
      <button type="button" className={reviewCount ? "form-builder-review-button needs-review" : "form-builder-review-button"} onClick={onReview}>Review{reviewCount ? ` · ${reviewCount}` : ""}</button>
      <button type="button" className={canSendForApproval ? "secondary-button" : "primary-button"} disabled={saving} onClick={onSave}>{saving ? "Saving…" : "Save draft"}</button>
      {canSendForApproval && onSendForApproval && <button type="button" className="primary-button" disabled={saving || !approvalReady} onClick={onSendForApproval}>{saving ? "Working…" : "Send for approval"}</button>}
    </div>
  </header>;
}
