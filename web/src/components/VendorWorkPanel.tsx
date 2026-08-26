import { useEffect, useId, useMemo, useRef, useState } from "react";
import type { FormEvent } from "react";
import { ApiError, apiErrorKind } from "../http";
import { loadFormTemplates } from "../monitoringApi";
import type { FormTemplate } from "../monitoringTypes";
import type { CapturePresentationMode } from "../types";
import { loadVendorRelationship } from "../vendorApi";
import { loadVendorRelationshipLinks } from "../vendorLinkApi";
import type { VendorRelationshipLink, VendorLinkTargetType } from "../vendorLinkTypes";
import type { VendorRelationshipAggregate } from "../vendorTypes";
import {
  acceptVendorWork, cancelVendorWork, loadVendorWork, loadVendorWorkResponse, prepareVendorWork, requestVendorWorkChanges,
  retryVendorWorkDelivery, sendVendorWork, startVendorWorkReview, vendorWorkDocumentURL,
} from "../vendorWorkApi";
import type { VendorWorkRequest, VendorWorkResponseView, VendorWorkSendOutcome } from "../vendorWorkTypes";
import "../vendor-work.css";

type Props = ({ targetType: VendorLinkTargetType; targetID: string; relationshipID?: never } | { relationshipID: string; targetType?: never; targetID?: never }) & { onOpenRequest?: (requestID: string) => void };
type LinkedRelationship = { link: VendorRelationshipLink; relationship: VendorRelationshipAggregate | null };
type LoadState = "loading" | "ready" | "failed";
type ActionMode = "accept" | "changes" | "cancel" | null;

const invitationTTLMinutes = 7 * 24 * 60;

export function VendorWorkPanel(props: Props) {
  const targetType = props.targetType;
  const targetID = props.targetID;
  const relationshipID = props.relationshipID;
  const onOpenRequest = props.onOpenRequest;
  const [work, setWork] = useState<VendorWorkRequest[]>([]);
  const [relationships, setRelationships] = useState<LinkedRelationship[]>([]);
  const [forms, setForms] = useState<FormTemplate[]>([]);
  const [state, setState] = useState<LoadState>("loading");
  const [setupAvailable, setSetupAvailable] = useState(true);
  const [creating, setCreating] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [selectedLinkID, setSelectedLinkID] = useState("");
  const [purpose, setPurpose] = useState("");
  const [instructions, setInstructions] = useState("");
  const [formKey, setFormKey] = useState("");
  const [presentation, setPresentation] = useState<CapturePresentationMode>("AUTOMATIC");
  const [audience, setAudience] = useState("");
  const [dueDate, setDueDate] = useState("");
  const [captureURLs, setCaptureURLs] = useState<Record<string, string>>({});
  const [nextCursor, setNextCursor] = useState("");
  const [loadingMore, setLoadingMore] = useState(false);
  const [loadMoreError, setLoadMoreError] = useState(false);
  const [linkNextCursor, setLinkNextCursor] = useState("");
  const [loadingMoreLinks, setLoadingMoreLinks] = useState(false);
  const [linkLoadError, setLinkLoadError] = useState(false);
  const loadSequence = useRef(0);
  const targetKey = relationshipID ? `RELATIONSHIP:${relationshipID}` : `${targetType}:${targetID}`;
  const activeTarget = useRef(targetKey);
  activeTarget.current = targetKey;
  const headingID = useId();
  const targetName = relationshipID ? "vendor relationship" : targetType === "PROGRAM" ? "Program" : "issue or change";
  const canCreate = Boolean(relationshipID || (targetType && targetID));

  useEffect(() => {
    resetDraft();
    setCreating(false);
    setNotice("");
    void loadAll();
  }, [targetKey]);

  async function loadAll() {
    const sequence = ++loadSequence.current;
    setState("loading");
    setLoadMoreError(false);
    setLinkLoadError(false);
    const [workResult, linksResult, formsResult] = await Promise.allSettled([
      relationshipID ? loadVendorWork({ relationship_id: relationshipID, limit: 20 }) : loadVendorWork({ target_type: targetType, target_id: targetID, limit: 20 }),
      relationshipID ? loadVendorRelationshipLinks({ relationship_id: relationshipID, limit: 50 }) : loadVendorRelationshipLinks({ target_type: targetType!, target_id: targetID!, limit: 50 }),
      loadFormTemplates(),
    ]);
    if (sequence !== loadSequence.current) return;
    if (workResult.status === "rejected") {
      setState("failed");
      return;
    }
    setWork(workResult.value.items);
    setNextCursor(workResult.value.next_cursor ?? "");
    setLinkNextCursor(linksResult.status === "fulfilled" ? linksResult.value.next_cursor ?? "" : "");
    if (relationshipID) {
      const value = await loadVendorRelationship(relationshipID).catch(() => null);
      if (sequence !== loadSequence.current) return;
      const links = linksResult.status === "fulfilled" ? linksResult.value.items.filter((item) => item.state === "ACTIVE") : [];
      setRelationships(value ? links.map((link) => ({ link, relationship: value })) : []);
    } else if (linksResult.status === "fulfilled") {
      const active = linksResult.value.items.filter((item) => item.state === "ACTIVE");
      const hydrated = await Promise.all(active.map(async (link) => ({ link, relationship: await loadVendorRelationship(link.relationship_id).catch(() => null) })));
      if (sequence !== loadSequence.current) return;
      setRelationships(hydrated);
    } else {
      setRelationships([]);
    }
    setForms(formsResult.status === "fulfilled" ? formsResult.value.filter((item) => item.status === "ACTIVE" && item.is_current) : []);
    setSetupAvailable(canCreate && linksResult.status === "fulfilled" && formsResult.status === "fulfilled");
    setState("ready");
  }

  async function loadMore() {
    if (!nextCursor || loadingMore) return;
    const requestedTarget = targetKey;
    setLoadingMore(true);
    setLoadMoreError(false);
    try {
      const page = await loadVendorWork(relationshipID
        ? { relationship_id: relationshipID, cursor: nextCursor, limit: 20 }
        : { target_type: targetType, target_id: targetID, cursor: nextCursor, limit: 20 });
      if (activeTarget.current !== requestedTarget) return;
      setWork((currentItems) => page.items.reduce(upsertWork, currentItems));
      setNextCursor(page.next_cursor ?? "");
    } catch {
      setLoadMoreError(true);
    } finally {
      setLoadingMore(false);
    }
  }

  async function loadMoreLinks() {
    if (!linkNextCursor || loadingMoreLinks) return;
    const requestedTarget = targetKey;
    setLoadingMoreLinks(true);
    setLinkLoadError(false);
    try {
      const page = await loadVendorRelationshipLinks(relationshipID
        ? { relationship_id: relationshipID, cursor: linkNextCursor, limit: 50 }
        : { target_type: targetType!, target_id: targetID!, cursor: linkNextCursor, limit: 50 });
      const active = page.items.filter((item) => item.state === "ACTIVE");
      let hydrated: LinkedRelationship[];
      if (relationshipID) {
        const value = relationships.find((item) => item.relationship)?.relationship ?? await loadVendorRelationship(relationshipID).catch(() => null);
        hydrated = value ? active.map((link) => ({ link, relationship: value })) : [];
      } else {
        hydrated = await Promise.all(active.map(async (link) => ({ link, relationship: await loadVendorRelationship(link.relationship_id).catch(() => null) })));
      }
      if (activeTarget.current !== requestedTarget) return;
      setRelationships((currentItems) => hydrated.reduce(upsertLinkedRelationship, currentItems));
      setLinkNextCursor(page.next_cursor ?? "");
    } catch {
      setLinkLoadError(true);
    } finally {
      setLoadingMoreLinks(false);
    }
  }

  function resetDraft() {
    setSelectedLinkID("");
    setPurpose("");
    setInstructions("");
    setFormKey("");
    setPresentation("AUTOMATIC");
    setAudience("");
    setDueDate("");
    setError("");
    setSaving(false);
  }

  const selectedRelationship = relationships.find((item) => item.link.id === selectedLinkID);
  const selectedForm = forms.find((item) => `${item.id}:${item.version}` === formKey);
  const canSubmit = Boolean(selectedRelationship?.relationship && selectedForm && purpose.trim() && instructions.trim() && validEmail(audience) && dueDate);
  const current = work.filter((item) => item.state !== "ACCEPTED" && item.state !== "CANCELLED");
  const history = work.filter((item) => item.state === "ACCEPTED" || item.state === "CANCELLED");

  async function prepareAndSend(event: FormEvent) {
    event.preventDefault();
    if (!canSubmit || !selectedRelationship?.relationship || !selectedForm) return;
    const dueAt = endOfDayUTC(dueDate);
    const requestedTarget = targetKey;
    setSaving(true);
    setError("");
    setNotice("");
    try {
      const prepared = await prepareVendorWork(selectedRelationship.relationship.relationship.id, {
        relationship_link_id: selectedRelationship.link.id,
        purpose: purpose.trim(), instructions: instructions.trim(), form_template_id: selectedForm.id,
        form_template_version: selectedForm.version, presentation, vendor_audience: audience.trim(), due_at: dueAt,
      });
      if (activeTarget.current !== requestedTarget) return;
      setWork((items) => upsertWork(items, prepared));
      try {
        const outcome = await sendVendorWork(prepared.relationship_id, prepared.id, { expected_version: prepared.version, vendor_audience: audience.trim(), invitation_ttl_minutes: invitationTTLMinutes });
        if (activeTarget.current !== requestedTarget) return;
        setWork((items) => upsertWork(items, outcome.work));
        if (outcome.capture_url) setCaptureURLs((current) => ({ ...current, [outcome.work.id]: outcome.capture_url! }));
        setNotice(deliveryNotice(outcome));
        setCreating(false);
        resetDraft();
      } catch {
        if (activeTarget.current !== requestedTarget) return;
        setError("The request is saved, but setup could not be completed. Use Complete setup on the request.");
      }
    } catch (caught) {
      if (activeTarget.current !== requestedTarget) return;
      setError(prepareError(caught));
    } finally {
      setSaving(false);
    }
  }

  if (state === "loading") return <section className="vendor-work-panel" aria-live="polite" aria-busy="true">Loading vendor requests for this {targetName}…</section>;
  if (state === "failed") return <section className="vendor-work-panel" role="alert"><h2>Vendor requests</h2><p>Vendor requests could not be loaded for this {targetName}. No request state is shown.</p><button type="button" className="secondary-button" onClick={() => void loadAll()}>Try again</button></section>;

  return <section className="vendor-work-panel" aria-labelledby={headingID}>
    <div className="section-heading-row"><div><h2 id={headingID}>Vendor requests</h2><p>Information, documents or confirmations requested from vendors for this {targetName}.</p></div>{canCreate && !creating && <button type="button" className="primary-button" disabled={!setupAvailable || (relationships.length === 0 && !linkNextCursor) || forms.length === 0} onClick={() => { setCreating(true); setError(""); setNotice(""); }}>Request vendor work</button>}</div>
    {canCreate && !setupAvailable && <p className="inline-notice">Linked vendors or approved forms are unavailable. Existing request history remains available.</p>}
    {canCreate && setupAvailable && relationships.length === 0 && !linkNextCursor && <p className="inline-notice">{relationshipID ? "Link this vendor relationship to a Program or issue before requesting vendor work." : `Link a vendor relationship to this ${targetName} before requesting vendor work.`}</p>}
    {canCreate && setupAvailable && forms.length === 0 && <p className="inline-notice">No current approved collection form is available. Activate a form before preparing this request.</p>}
    {notice && <p className="inline-success" role="status">{notice}</p>}
    {current.length === 0 && history.length === 0 && <p>No vendor requests have been recorded for this {targetName}.</p>}
    {current.length > 0 && <div className="vendor-work-group"><h3>Current requests</h3>{current.map((item) => <VendorWorkCard key={item.id} work={item} form={findForm(forms, item)} relationship={relationships.find((value) => value.link.id === item.relationship_link_id || value.relationship?.relationship.id === item.relationship_id)?.relationship ?? null} captureURL={captureURLs[item.id]} onOpenRequest={onOpenRequest} onChanged={(value, outcome) => { setWork((items) => upsertWork(items, value)); if (outcome) { setNotice(deliveryNotice(outcome)); if (outcome.capture_url) setCaptureURLs((currentURLs) => ({ ...currentURLs, [value.id]: outcome.capture_url! })); } }}/>)}</div>}
    {history.length > 0 && <details className="vendor-work-history" open={current.length === 0}><summary><span>Request history</span><strong>{history.length}</strong></summary><div>{history.map((item) => <VendorWorkCard key={item.id} work={item} form={findForm(forms, item)} relationship={relationships.find((value) => value.link.id === item.relationship_link_id || value.relationship?.relationship.id === item.relationship_id)?.relationship ?? null} onOpenRequest={onOpenRequest} onChanged={(value) => setWork((items) => upsertWork(items, value))}/>)}</div></details>}
    {nextCursor && <button type="button" className="secondary-button vendor-work-load-more" disabled={loadingMore} onClick={() => void loadMore()}>{loadingMore ? "Loading…" : "Load more vendor requests"}</button>}
    {loadMoreError && <p role="alert" className="inline-error">More vendor requests could not be loaded. The current list remains available.</p>}
    {creating && <form className="vendor-work-form" onSubmit={(event) => void prepareAndSend(event)}>
      <div><h3>Request vendor work</h3><p>Confirm the vendor, collection form and delivery details for this {targetName}.</p></div>
      <label>{relationshipID ? "Related Program or issue" : "Vendor relationship"}<select value={selectedLinkID} required onChange={(event) => setSelectedLinkID(event.target.value)}><option value="">{relationshipID ? "Choose related work" : "Choose a linked vendor"}</option>{relationships.map(({ link, relationship }) => <option key={link.id} value={link.id} disabled={!relationship}>{relationship ? relationshipID ? `${link.target_type === "PROGRAM" ? "Program" : "Issue or change"} · ${link.purpose_label}` : `${relationship.vendor.legal_name} — ${relationship.relationship.service_name}` : "Vendor details unavailable"}</option>)}</select></label>
      {linkNextCursor && <button type="button" className="secondary-button" disabled={loadingMoreLinks} onClick={() => void loadMoreLinks()}>{loadingMoreLinks ? "Loading…" : relationshipID ? "Load more related work" : "Load more linked vendors"}</button>}
      {linkLoadError && <p role="alert" className="inline-error">More related records could not be loaded. The current choices remain available.</p>}
      <label>Request purpose<input value={purpose} required maxLength={500} onChange={(event) => setPurpose(event.target.value)} placeholder="Confirm annual service controls"/></label>
      <label>Instructions for the vendor<textarea value={instructions} required maxLength={2000} rows={4} onChange={(event) => setInstructions(event.target.value)} placeholder="State what must be completed or provided."/></label>
      <div className="vendor-work-form-grid"><label>Collection form<select value={formKey} required onChange={(event) => setFormKey(event.target.value)}><option value="">Choose an approved form</option>{forms.map((item) => <option key={`${item.id}:${item.version}`} value={`${item.id}:${item.version}`}>{item.name} · version {item.version}</option>)}</select></label><label>Form layout<select value={presentation} onChange={(event) => setPresentation(event.target.value as CapturePresentationMode)}><option value="AUTOMATIC">Automatic</option><option value="CLASSIC">Classic</option><option value="WIZARD">Wizard</option></select></label></div>
      {selectedForm && <FormSummary form={selectedForm}/>}
      <div className="vendor-work-form-grid"><label>Vendor contact<input type="email" autoComplete="email" value={audience} required maxLength={320} onChange={(event) => setAudience(event.target.value)} placeholder="contact@vendor.example"/></label><label>Due date<input type="date" min={minimumDueDate()} value={dueDate} required onChange={(event) => setDueDate(event.target.value)}/></label></div>
      <div className="vendor-work-prefill"><strong>Prefill summary</strong><p>Known vendor and service details will be shown with this request.</p></div>
      {error && <p role="alert" className="inline-error">{error}</p>}
      <div className="form-actions"><button type="button" className="secondary-button" disabled={saving} onClick={() => { setCreating(false); resetDraft(); }}>Cancel</button><button type="submit" className="primary-button" disabled={saving || !canSubmit}>{saving ? "Preparing request…" : "Prepare and send request"}</button></div>
    </form>}
  </section>;
}

function VendorWorkCard({ work, form, relationship, captureURL, onOpenRequest, onChanged }: { work: VendorWorkRequest; form?: FormTemplate; relationship: VendorRelationshipAggregate | null; captureURL?: string; onOpenRequest?: (requestID: string) => void; onChanged: (work: VendorWorkRequest, outcome?: VendorWorkSendOutcome) => void }) {
  const [mode, setMode] = useState<ActionMode>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [audience, setAudience] = useState("");
  const [rationale, setRationale] = useState("");
  const [message, setMessage] = useState("");
  const [fieldIDs, setFieldIDs] = useState<string[]>([]);
  const [dueDate, setDueDate] = useState("");
  const [copyNotice, setCopyNotice] = useState("");
  const [responseState, setResponseState] = useState<"closed" | "loading" | "ready" | "failed">("closed");
  const [responseView, setResponseView] = useState<VendorWorkResponseView>();
  const responseIdentity = `${work.current_request_id ?? ""}:${work.submission_id ?? ""}`;
  const activeResponseIdentity = useRef(responseIdentity);
  activeResponseIdentity.current = responseIdentity;
  const status = workStateLabel(work.state);
  const retry = work.delivery_state === "RETRY_REQUIRED" || work.delivery_state === "LINK_CREATED_EMAIL_NOT_SENT";
  const setupIncomplete = retry && work.state === "PREPARING" && !work.current_request_id;
  const canCancel = work.state !== "ACCEPTED" && work.state !== "CANCELLED";
  const changeFields = responseView?.answers.map((answer) => ({ id: answer.field_id, label: answer.label })) ?? form?.fields.map((field) => ({ id: field.id, label: field.label })) ?? [];
  const acceptanceBlocked = responseView?.documents.some((document) => document.artifact_status !== "AVAILABLE") ?? false;

  useEffect(() => {
    setResponseView(undefined);
    setResponseState("closed");
  }, [responseIdentity]);

  async function run(action: () => Promise<VendorWorkRequest | VendorWorkSendOutcome>) {
    setBusy(true);
    setError("");
    try {
      const result = await action();
      if ("work" in result) onChanged(result.work, result); else onChanged(result);
      setMode(null);
      setRationale("");
      setMessage("");
      setFieldIDs([]);
      setDueDate("");
    } catch (caught) {
      const kind = apiErrorKind(caught);
      const acceptanceConflict = caught instanceof ApiError && (caught.code === "vendor_work_acceptance_blocked" || caught.message === "A submitted document is pending inspection, quarantined or unavailable. Wait for inspection or request a replacement before accepting this response.");
      setError(acceptanceConflict ? "A submitted document cannot be used yet. Wait for inspection or request a replacement before accepting." : kind === "conflict" ? "This request changed. Reload vendor requests before trying again." : kind === "forbidden" || kind === "unauthorized" ? "Your current access does not allow this action." : "The request could not be updated. Your entries remain on this screen.");
    } finally {
      setBusy(false);
    }
  }

  async function openResponse() {
    const requestedIdentity = responseIdentity;
    setResponseState("loading");
    setError("");
    try {
      const view = await loadVendorWorkResponse(work.relationship_id, work.id);
      if (activeResponseIdentity.current !== requestedIdentity) return;
      if (view.work.id !== work.id || view.request.request_id !== view.response.request_id || view.request.request_id !== work.current_request_id) throw new Error("response scope mismatch");
      setResponseView(view);
      setResponseState("ready");
    } catch {
      if (activeResponseIdentity.current !== requestedIdentity) return;
      setResponseView(undefined);
      setResponseState("failed");
    }
  }

  async function beginReview() {
    if (!responseView || responseState !== "ready") return;
    setBusy(true);
    setError("");
    try {
      const updated = await startVendorWorkReview(work.relationship_id, work.id, { expected_version: work.version });
      onChanged(updated);
    } catch (caught) {
      const kind = apiErrorKind(caught);
      setError(kind === "conflict" ? "This request changed. Reload vendor requests before trying again." : "The response could not be opened for review. Try again.");
    } finally {
      setBusy(false);
    }
  }

  async function copySecureLink() {
    if (!captureURL) return;
    try {
      await navigator.clipboard.writeText(captureURL);
      setCopyNotice("Secure link copied.");
    } catch {
      setCopyNotice("The link could not be copied. Select and copy it from the field.");
    }
  }

  return <article className="vendor-work-card" data-testid={`vendor-work-${work.id}`}>
    <div className="vendor-work-card-heading"><div><span>{relationship ? `${relationship.vendor.legal_name} · ${relationship.relationship.service_name}` : "Vendor relationship"}</span><h4>{work.purpose}</h4></div><strong className={`vendor-work-state state-${work.state.toLowerCase().replaceAll("_", "-")}`}>{status}</strong></div>
    <p>{work.instructions}</p>
    <dl className="vendor-work-facts"><div><dt>Due</dt><dd>{formatDate(work.due_at)}</dd></div><div><dt>Collection</dt><dd>{form ? `${form.name} · version ${work.form_template_version}` : `Form version ${work.form_template_version}`}</dd></div><div><dt>Layout</dt><dd>{presentationLabel(work.presentation)}</dd></div></dl>
    {(setupIncomplete || work.recovery) && <p className="inline-notice">{setupIncomplete ? "The collection request was not created. Complete setup to create and send it." : work.recovery}</p>}
    {captureURL && <div className="vendor-work-secure-link"><label>Secure link<input readOnly value={captureURL} onFocus={(event) => event.currentTarget.select()}/></label><button type="button" className="secondary-button" onClick={() => void copySecureLink()}>Copy secure link</button>{copyNotice && <small role="status">{copyNotice}</small>}</div>}
    {retry && <div className="vendor-work-action"><label>Vendor contact<input type="email" autoComplete="email" value={audience} onChange={(event) => setAudience(event.target.value)}/></label><button type="button" className="primary-button" disabled={busy || !validEmail(audience)} onClick={() => void run(() => retryVendorWorkDelivery(work.relationship_id, work.id, { expected_version: work.version, vendor_audience: audience.trim(), invitation_ttl_minutes: invitationTTLMinutes }))}>{busy ? setupIncomplete ? "Completing…" : "Retrying…" : setupIncomplete ? "Complete setup" : "Retry delivery"}</button></div>}
    {!retry && work.state === "PREPARING" && <div className="vendor-work-action"><label>Vendor contact<input type="email" autoComplete="email" value={audience} onChange={(event) => setAudience(event.target.value)}/></label><button type="button" className="primary-button" disabled={busy || !validEmail(audience)} onClick={() => void run(() => sendVendorWork(work.relationship_id, work.id, { expected_version: work.version, vendor_audience: audience.trim(), invitation_ttl_minutes: invitationTTLMinutes }))}>{busy ? "Sending…" : "Send request"}</button></div>}
    {work.current_request_id && onOpenRequest && <button type="button" className="secondary-button" onClick={() => onOpenRequest(work.current_request_id!)}>Open collection request</button>}
    {(work.state === "RESPONSE_RECEIVED" || work.state === "UNDER_REVIEW") && responseState === "closed" && <button type="button" className="primary-button" disabled={!work.current_request_id} onClick={() => void openResponse()}>Review response</button>}
    {responseState === "loading" && <p aria-live="polite" aria-busy="true">Loading the submitted response…</p>}
    {responseState === "failed" && <div role="alert" className="inline-error"><p>The submitted response could not be loaded. Review has not started.</p><button type="button" className="secondary-button" onClick={() => void openResponse()}>Try again</button></div>}
    {responseView && responseState === "ready" && <VendorWorkResponseReview view={responseView}/>}
    {work.state === "RESPONSE_RECEIVED" && responseState === "ready" && <button type="button" className="primary-button" disabled={busy} onClick={() => void beginReview()}>{busy ? "Starting review…" : "Begin review"}</button>}
    {work.state === "UNDER_REVIEW" && responseState === "ready" && acceptanceBlocked && <p className="inline-notice">Acceptance is unavailable while a submitted document is pending inspection, quarantined or unavailable. Wait for inspection or request a replacement.</p>}
    {work.state === "UNDER_REVIEW" && responseState === "ready" && !mode && <div className="vendor-work-buttons"><button type="button" className="primary-button" disabled={acceptanceBlocked} onClick={() => setMode("accept")}>Accept response</button><button type="button" className="secondary-button" onClick={() => setMode("changes")}>Request changes</button></div>}
    {mode === "accept" && <div className="vendor-work-decision"><label>Acceptance basis<textarea rows={3} maxLength={2000} value={rationale} onChange={(event) => setRationale(event.target.value)}/></label><div className="form-actions"><button type="button" className="secondary-button" onClick={() => setMode(null)}>Back</button><button type="button" className="primary-button" disabled={busy || !rationale.trim()} onClick={() => void run(() => acceptVendorWork(work.relationship_id, work.id, { expected_version: work.version, rationale: rationale.trim() }))}>{busy ? "Accepting…" : "Confirm acceptance"}</button></div></div>}
    {mode === "changes" && <div className="vendor-work-decision"><label>What the vendor must change<textarea rows={3} maxLength={2000} value={message} onChange={(event) => setMessage(event.target.value)}/></label>{changeFields.length ? <fieldset><legend>Answers or documents to update</legend>{changeFields.map((field) => <label key={field.id}><input type="checkbox" checked={fieldIDs.includes(field.id)} onChange={(event) => setFieldIDs((current) => event.target.checked ? [...current, field.id] : current.filter((id) => id !== field.id))}/>{field.label}</label>)}</fieldset> : <p className="inline-notice">The form fields could not be loaded. Reload vendor requests before requesting changes.</p>}<div className="vendor-work-form-grid"><label>Vendor contact<input type="email" value={audience} onChange={(event) => setAudience(event.target.value)}/></label><label>Revised due date<input type="date" min={minimumDueDate()} value={dueDate} onChange={(event) => setDueDate(event.target.value)}/></label></div><div className="form-actions"><button type="button" className="secondary-button" onClick={() => setMode(null)}>Back</button><button type="button" className="primary-button" disabled={busy || !message.trim() || !fieldIDs.length || !validEmail(audience) || !dueDate} onClick={() => void run(() => requestVendorWorkChanges(work.relationship_id, work.id, { expected_version: work.version, message: message.trim(), field_ids: fieldIDs, vendor_audience: audience.trim(), due_at: endOfDayUTC(dueDate), invitation_ttl_minutes: invitationTTLMinutes }))}>{busy ? "Sending changes…" : "Send change request"}</button></div></div>}
    {mode === "cancel" && <div className="vendor-work-decision"><label>Cancellation reason<textarea rows={3} maxLength={1000} value={rationale} onChange={(event) => setRationale(event.target.value)}/></label><div className="form-actions"><button type="button" className="secondary-button" onClick={() => setMode(null)}>Back</button><button type="button" className="primary-button" disabled={busy || !rationale.trim()} onClick={() => void run(() => cancelVendorWork(work.relationship_id, work.id, { expected_version: work.version, reason: rationale.trim() }))}>{busy ? "Cancelling…" : "Cancel request"}</button></div></div>}
    {work.state === "ACCEPTED" && work.review_rationale && <p className="vendor-work-outcome"><strong>Acceptance basis</strong>{work.review_rationale}</p>}
    {work.state === "CANCELLED" && work.cancellation_reason && <p className="vendor-work-outcome"><strong>Cancellation reason</strong>{work.cancellation_reason}</p>}
    {canCancel && mode !== "cancel" && <button type="button" className="text-button vendor-work-cancel" disabled={busy} onClick={() => setMode("cancel")}>Cancel request</button>}
    {error && <p role="alert" className="inline-error">{error}</p>}
  </article>;
}

function VendorWorkResponseReview({ view }: { view: VendorWorkResponseView }) {
  return <section className="vendor-work-response" aria-label="Vendor response">
    <div className="vendor-work-response-heading"><div><h5>Vendor response</h5><p>Submitted {formatDateTime(view.response.submitted_at)} · Form version {view.request.form_template_version}</p></div><span>{view.answers.filter((answer) => answer.visibility === "VISIBLE" && answer.value).length} of {view.answers.filter((answer) => answer.visibility === "VISIBLE").length} visible fields answered</span></div>
    <div className="vendor-work-response-group"><h6>Submitted answers</h6>{view.answers.length ? <dl className="vendor-work-answer-list">{view.answers.map((answer) => <VendorWorkAnswer key={answer.field_id} answer={answer}/>)}</dl> : <p>No answer fields were included in this submitted response.</p>}</div>
    <div className="vendor-work-response-group"><h6>Supporting documents</h6>{view.documents.length ? <div className="vendor-work-documents">{view.documents.map((document) => <VendorWorkDocument key={document.artifact_id} view={view} document={document}/>)}</div> : <p>No supporting documents were submitted with this response.</p>}</div>
  </section>;
}

function VendorWorkAnswer({ answer }: { answer: VendorWorkResponseView["answers"][number] }) {
  const conditional = answer.visibility === "CONDITIONALLY_OMITTED";
  const missing = answer.required && !answer.value;
  const sourceValue = answer.provenance?.source_value?.text;
  return <div role="group" aria-label={`Response: ${answer.label}`}>
    <dt>{answer.label}{answer.required ? " · Required" : ""}</dt>
    <dd className={conditional || missing ? "vendor-work-answer-limitation" : undefined}>{conditional ? "Not requested because its condition was not met" : missing ? "Required response missing" : `Vendor response: ${formatResponseAnswer(answer.value)}`}</dd>
    {sourceValue && <dd>Source value: {sourceValue}</dd>}
    {answer.provenance && <dd className="vendor-work-provenance">{responseProvenanceLabel(answer.provenance)}</dd>}
    {answer.provenance?.validations?.map((validation, index) => <dd className="vendor-work-validation" key={`${answer.field_id}-validation-${index}`}>{responseValidationLabel(validation)}</dd>)}
  </div>;
}

function VendorWorkDocument({ view, document }: { view: VendorWorkResponseView; document: VendorWorkResponseView["documents"][number] }) {
  const available = document.artifact_status === "AVAILABLE";
  const openURL = available ? vendorWorkDocumentURL(view.work.relationship_id, view.work.id, view.request.request_id, document.artifact_id) : "";
  return <article className="vendor-work-document" aria-label={document.file_name}>
    <div><strong>{document.file_name}</strong><span>{documentTypeLabel(document.document_type)} · {formatBytes(document.size_bytes)}</span><span>{artifactStateLabel(document.artifact_status)} · {evidenceClassLabel(document.evidence_class)}</span>{!available && <small>{artifactRecovery(document.artifact_status)}</small>}</div>
    {openURL && <button type="button" className="secondary-button" onClick={() => window.open(openURL, "_blank", "noopener,noreferrer")}>Open document</button>}
  </article>;
}

function FormSummary({ form }: { form: FormTemplate }) {
  const required = form.fields.filter((field) => field.required).length;
  const uploads = form.fields.filter((field) => ["file", "photo", "signature", "vendor_document"].includes(field.type)).length;
  return <div className="vendor-work-form-summary"><strong>{form.name}</strong><span>{form.fields.length} {form.fields.length === 1 ? "field" : "fields"} · {required} required · {uploads} {uploads === 1 ? "document upload" : "document uploads"}</span><small>{form.purpose}</small></div>;
}

function findForm(forms: FormTemplate[], work: VendorWorkRequest) { return forms.find((form) => form.id === work.form_template_id && form.version === work.form_template_version); }
function upsertWork(items: VendorWorkRequest[], next: VendorWorkRequest) { return [next, ...items.filter((item) => item.id !== next.id)].sort((left, right) => Date.parse(right.updated_at) - Date.parse(left.updated_at)); }
function upsertLinkedRelationship(items: LinkedRelationship[], next: LinkedRelationship) { return [...items.filter((item) => item.link.id !== next.link.id), next]; }
function validEmail(value: string) { return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value.trim()); }
function endOfDayUTC(value: string) { return new Date(`${value}T23:59:59.000Z`).toISOString(); }
function minimumDueDate() { const value = new Date(); value.setUTCDate(value.getUTCDate() + 1); return value.toISOString().slice(0, 10); }
function formatDate(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? "Not recorded" : date.toLocaleDateString(); }
function formatDateTime(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? "at an unavailable time" : date.toLocaleString(); }
function formatBytes(value: number) { if (!Number.isFinite(value) || value < 0) return "Size unavailable"; if (value < 1024) return `${value} bytes`; if (value < 1024 * 1024) return `${Math.round(value / 1024)} KB`; return `${(value / (1024 * 1024)).toFixed(1)} MB`; }
function formatResponseAnswer(value?: VendorWorkResponseView["answers"][number]["value"]) { if (!value) return "No answer submitted"; if (value.text?.trim()) return value.text; if (value.values?.length) return value.values.join(", "); if (value.document) return value.document.reference || documentTypeLabel(value.document.document_type); if (value.artifact_ids?.length) return `${value.artifact_ids.length} ${value.artifact_ids.length === 1 ? "file" : "files"} submitted`; return "No answer submitted"; }
function responseProvenanceLabel(provenance: NonNullable<VendorWorkResponseView["answers"][number]["provenance"]>) { const observed = provenance.source_receipt?.observed_at ? formatDate(provenance.source_receipt.observed_at) : ""; if (provenance.origin === "SOURCE_PREFILLED") return `Prefilled from ${provenance.source_receipt?.source_id || provenance.source || "a connected source"}${observed ? ` · Observed ${observed}` : ""}`; if (provenance.origin === "RESPONDENT_CORRECTED") return `Updated by the vendor${observed ? ` · Source value observed ${observed}` : ""}`; if (provenance.origin === "RESPONDENT_ENTERED") return "Entered by the vendor"; return observed ? `Answer source recorded · Observed ${observed}` : "Answer source recorded"; }
function responseValidationLabel(validation: NonNullable<NonNullable<VendorWorkResponseView["answers"][number]["provenance"]>["validations"]>[number]) { const state = validation.state ?? "UNKNOWN"; const label = state === "CURRENT" ? "Validation current" : state === "STALE" ? "Validation out of date" : state === "UNAVAILABLE" ? "Validation source unavailable" : state === "NOT_FOUND" ? "Source did not confirm this value" : `Validation result: ${documentTypeLabel(state)}`; return [label, validation.binding_name || validation.source_id, validation.receipt?.observed_at ? `Checked ${formatDate(validation.receipt.observed_at)}` : ""].filter(Boolean).join(" · "); }
function documentTypeLabel(value: string) { return value.replaceAll("_", " ").toLowerCase().replace(/(^|\s)\S/g, (letter) => letter.toUpperCase()); }
function artifactStateLabel(value: string) { if (value === "AVAILABLE") return "Available"; if (value === "QUARANTINED") return "Quarantined"; if (value === "STORED_UNSCANNED") return "Security scan pending"; if (value === "DELETED") return "Unavailable"; return documentTypeLabel(value); }
function evidenceClassLabel(value: string) { if (value === "VENDOR_SUPPLIED") return "Vendor supplied"; if (value === "BANK_VALIDATED") return "Bank validated"; if (value === "OFFICIAL_SOURCE") return "Official source"; return documentTypeLabel(value); }
function artifactRecovery(value: string) { if (value === "QUARANTINED") return "This document is quarantined. Request a clean replacement before review."; if (value === "STORED_UNSCANNED") return "The security scan is pending. Open will be available after the document passes inspection."; if (value === "DELETED") return "This document is unavailable. Request a replacement before continuing."; return "This document is not available to open. Reload the response or request a replacement."; }
function presentationLabel(value: CapturePresentationMode) { if (value === "WIZARD") return "Wizard"; if (value === "CLASSIC") return "Classic"; return "Automatic"; }
function workStateLabel(value: VendorWorkRequest["state"]) { return ({ PREPARING: "Ready to send", AWAITING_VENDOR: "Waiting for vendor", RESPONSE_RECEIVED: "Response received", UNDER_REVIEW: "Under review", CHANGES_REQUESTED: "Changes requested", ACCEPTED: "Accepted", CANCELLED: "Cancelled" } as const)[value]; }
function deliveryNotice(outcome: VendorWorkSendOutcome) { return outcome.state === "DELIVERED" ? "Vendor request sent." : outcome.state === "LINK_CREATED_EMAIL_NOT_SENT" ? "The secure link is ready, but email delivery was not confirmed. Copy the link or retry delivery." : outcome.recovery || "The request is saved. Retry delivery when the service is available."; }
function prepareError(error: unknown) { const kind = apiErrorKind(error); if (kind === "conflict") return "This vendor relationship or form changed. Refresh this section before trying again."; if (kind === "forbidden" || kind === "unauthorized") return "Your current access does not allow this vendor request to be prepared."; if (kind === "validation") return "Check the vendor, form, contact and due date before trying again."; return "The vendor request could not be prepared. Your entries remain on this screen."; }
