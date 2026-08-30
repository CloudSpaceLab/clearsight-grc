import { useEffect } from "react";
import type { FormLibraryItem } from "../../../formsTypes";
import type { LifecycleStatus } from "../../../monitoringTypes";
import { isTemplateApprovalReady } from "../formQuality";
import { formatDate, StatusPill } from "./TemplateLibraryTable";

type Props = {
  item?: FormLibraryItem;
  requestedID?: string;
  busy: string | null;
  onClose: () => void;
  onClearFilters: () => void;
  onEdit: () => void;
  onTransition: (to: LifecycleStatus) => void;
};

export function TemplateDetailDrawer({ item, requestedID, busy, onClose, onClearFilters, onEdit, onTransition }: Props) {
  useEffect(() => {
    if (!requestedID) return;
    const dismiss = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return;
      event.preventDefault();
      onClose();
    };
    window.addEventListener("keydown", dismiss);
    return () => window.removeEventListener("keydown", dismiss);
  }, [onClose, requestedID]);

  if (!requestedID) return null;

  return <aside className="forms-detail-drawer" aria-label="Selected form template">
    <div className="forms-detail-drawer-bar">
      <span>Form detail</span>
      <button type="button" className="forms-detail-close" aria-label="Close form detail" onClick={onClose}>×</button>
    </div>
    {item ? <TemplateDetail item={item} busy={busy} onEdit={onEdit} onTransition={onTransition}/>
      : <div className="forms-detail-drawer-body">
        <span className="forms-detail-kicker">Selected template</span>
        <h2>Template isn’t in this view</h2>
        <p>Clear the active filters to bring the selected template back into the current result set.</p>
        <button type="button" onClick={onClearFilters}>Clear filters</button>
      </div>}
  </aside>;
}

function TemplateDetail({ item, busy, onEdit, onTransition }: { item: FormLibraryItem; busy: string | null; onEdit: () => void; onTransition: (to: LifecycleStatus) => void }) {
  const form = item.template;
  const approvalReady = form.status === "DRAFT" && isTemplateApprovalReady(form);
  const owner = form.owner_principal_id || form.responsible_team || "Not assigned";

  return <div className="forms-detail-drawer-body">
    <header className="forms-detail-heading">
      <span className="forms-detail-kicker">{form.code}</span>
      <h2>{form.name}</h2>
      <p>{form.purpose}</p>
    </header>

    <div className="forms-detail-state">
      <div>
        <span>Latest stored</span>
        <strong><StatusPill status={form.status}/> v{form.version}</strong>
      </div>
      <div>
        <span>Reusable now</span>
        <strong>{item.active_version ? <><StatusPill status={item.active_status ?? "ACTIVE"}/> v{item.active_version}</> : "None"}</strong>
      </div>
    </div>

    <dl className="forms-detail-facts">
      <div><dt>Owner</dt><dd>{owner}</dd></div>
      <div><dt>Questions</dt><dd>{form.fields.length}</dd></div>
      <div><dt>Scoring</dt><dd>{form.scoring_mode || "NONE"}</dd></div>
      <div><dt>Updated</dt><dd>{formatDate(form.updated_at)}</dd></div>
      <div><dt>Next review</dt><dd>{form.next_review_at ? formatDate(form.next_review_at) : "Not scheduled"}</dd></div>
    </dl>

    {form.tags?.length ? <div className="forms-tags">{form.tags.map((tag) => <span key={tag}>{tag}</span>)}</div> : null}

    <div className="forms-detail-actions">
      {form.status === "DRAFT" && <>
        <button type="button" onClick={onEdit}>Edit draft</button>
        <button className="forms-primary" type="button" disabled={busy !== null || !approvalReady} onClick={() => onTransition("PENDING_APPROVAL")}>Send for approval</button>
      </>}
      {form.status === "DRAFT" && !approvalReady && <small className="forms-muted">Open the editor to resolve approval-quality checks before submission.</small>}
      {form.status === "PENDING_APPROVAL" && <>
        <button className="forms-primary" type="button" disabled={busy !== null} onClick={() => onTransition("ACTIVE")}>Approve and activate</button>
        <button type="button" disabled={busy !== null} onClick={() => onTransition("REJECTED")}>Reject</button>
      </>}
      {form.status === "ACTIVE" && <button type="button" disabled={busy !== null} onClick={() => onTransition("RETIRED")}>Retire revision</button>}
    </div>
  </div>;
}
