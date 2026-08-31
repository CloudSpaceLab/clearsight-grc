import { useCallback, useRef } from "react";
import type { FormLibraryItem } from "../../../formsTypes";
import type { LifecycleStatus } from "../../../monitoringTypes";
import { FocusedSheet } from "../../FocusedSheet";
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
  const onCloseRef = useRef(onClose);
  onCloseRef.current = onClose;
  const close = useCallback(() => onCloseRef.current(), []);

  if (!requestedID) return null;

  return <FocusedSheet
    label="Selected form template"
    closeLabel="Close form detail"
    panelClassName="forms-detail-drawer"
    onClose={close}
  >
    <div className="forms-detail-drawer-bar">
      <span>Form detail</span>
    </div>
    {item ? <TemplateDetail item={item} busy={busy} onEdit={onEdit} onTransition={onTransition}/>
      : <div className="forms-detail-drawer-body">
        <span className="forms-detail-kicker">Selected template</span>
        <h2>Template isn’t in this view</h2>
        <p>Clear the active filters to bring the selected template back into the current result set.</p>
        <button type="button" onClick={onClearFilters}>Clear filters</button>
      </div>}
  </FocusedSheet>;
}

function TemplateDetail({ item, busy, onEdit, onTransition }: { item: FormLibraryItem; busy: string | null; onEdit: () => void; onTransition: (to: LifecycleStatus) => void }) {
  const form = item.template;
  const approvalReady = form.status === "DRAFT" && isTemplateApprovalReady(form);
  const owner = form.responsible_team || (form.owner_principal_id ? "Assigned owner" : "Not assigned");
  const revise = item.operations?.find((operation) => operation.command === "forms.template.revise");
  const transition = item.operations?.find((operation) => operation.command === "forms.template.transition");
  const authorityReady = item.authority_available === true;
  const canRevise = Boolean(authorityReady && revise?.can_act);
  const canTransition = (to: LifecycleStatus) => Boolean(authorityReady && transition?.can_act && transition.allowed_targets?.includes(to));
  const unavailable = !authorityReady;
  const unavailableReason = unavailable
    ? "Current responsibilities could not be checked. Form changes are unavailable until the authority route is restored."
    : revise?.reason || transition?.reason;

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
        {canRevise && <button type="button" onClick={onEdit}>Edit draft</button>}
        {canTransition("PENDING_APPROVAL") && <button className="forms-primary" type="button" disabled={busy !== null || !approvalReady} onClick={() => onTransition("PENDING_APPROVAL")}>Send for approval</button>}
      </>}
      {form.status === "DRAFT" && canTransition("PENDING_APPROVAL") && !approvalReady && <small className="forms-muted">Open the editor to resolve approval-quality checks before submission.</small>}
      {form.status === "PENDING_APPROVAL" && <>
        {canTransition("ACTIVE") && <button className="forms-primary" type="button" disabled={busy !== null} onClick={() => onTransition("ACTIVE")}>Approve and activate</button>}
        {canTransition("REJECTED") && <button type="button" disabled={busy !== null} onClick={() => onTransition("REJECTED")}>Reject</button>}
      </>}
      {form.status === "ACTIVE" && <>
        {canTransition("PAUSED") && <button type="button" disabled={busy !== null} onClick={() => onTransition("PAUSED")}>Pause revision</button>}
        {canTransition("RETIRED") && <button type="button" disabled={busy !== null} onClick={() => onTransition("RETIRED")}>Retire revision</button>}
      </>}
      {form.status === "PAUSED" && <>
        {canTransition("ACTIVE") && <button className="forms-primary" type="button" disabled={busy !== null} onClick={() => onTransition("ACTIVE")}>Resume revision</button>}
        {canTransition("RETIRED") && <button type="button" disabled={busy !== null} onClick={() => onTransition("RETIRED")}>Retire revision</button>}
      </>}
      {!canRevise && !canTransition("PENDING_APPROVAL") && !canTransition("ACTIVE") && !canTransition("REJECTED") && !canTransition("PAUSED") && !canTransition("RETIRED") && unavailableReason && <small className="forms-muted">{unavailableReason}</small>}
    </div>
  </div>;
}
