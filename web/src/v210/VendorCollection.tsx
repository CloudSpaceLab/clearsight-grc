import { useEffect, useRef, useState, type FormEvent } from "react";
import { loadVendorCollection, reconcileVendorEvidence, type VendorCollection as Collection, type VendorCollectionField } from "../vendorCollectionApi";
import { loadDocuments, type DocumentOccurrence } from "../submittedDocumentApi";
import "./vendor-collection.css";

export function VendorCollection({ assessmentId, relationshipId, onChanged, onUncertain, onPendingChange, reviewableFields = [], onReview, reviewVersion, reviewRequestID }: { assessmentId: string; relationshipId: string; onChanged: () => void; onUncertain: () => void; onPendingChange?: (pending: boolean) => void; reviewableFields?: string[]; onReview?: (fieldID: string) => void; reviewVersion?: number; reviewRequestID?: string }) {
  const [collection, setCollection] = useState<Collection>(); const [state, setState] = useState<"loading" | "ready" | "error">("loading");
  const [attempt, setAttempt] = useState(0); const [field, setField] = useState<VendorCollectionField>();
  const [files, setFiles] = useState<DocumentOccurrence[]>([]); const [fileState, setFileState] = useState<"loading" | "ready" | "error">("loading");
  const [cursor, setCursor] = useState<string>(); const [moreBusy, setMoreBusy] = useState(false); const [moreError, setMoreError] = useState(false);
  const [selected, setSelected] = useState(""); const [reason, setReason] = useState(""); const [busy, setBusy] = useState(false);
  const [outcome, setOutcome] = useState<"unknown" | "linked">(); const lock = useRef(false); const generation = useRef(0); const moreLock = useRef<number | null>(null);
  useEffect(() => {
    const controller = new AbortController(); setState("loading"); setCollection(undefined);
    loadVendorCollection(assessmentId, controller.signal).then((value) => { if (!controller.signal.aborted) { if (value.assessment_id !== assessmentId || !Array.isArray(value.fields)) { setState("error"); return; } setCollection(value); setState("ready"); } }).catch(() => { if (!controller.signal.aborted) setState("error"); });
    return () => controller.abort();
  }, [assessmentId, attempt]);
  useEffect(() => {
    if (!field) return;
    const controller = new AbortController(); const token = ++generation.current; moreLock.current = null; setMoreBusy(false); setFiles([]); setCursor(undefined); setFileState("loading"); setMoreError(false);
    loadDocuments({ relationship_id: relationshipId, current_only: true, limit: 25 }, controller.signal).then((page) => { if (token === generation.current) { setFiles(page.items.filter((file) => eligible(file, relationshipId))); setCursor(page.next_cursor); setFileState("ready"); } }).catch(() => { if (token === generation.current && !controller.signal.aborted) setFileState("error"); });
    return () => { controller.abort(); generation.current++; };
  }, [field?.field_id, relationshipId]);
  async function more() {
    if (!cursor || moreBusy || moreLock.current !== null || lock.current) return; const token = generation.current; moreLock.current = token; setMoreBusy(true); setMoreError(false);
    try { const page = await loadDocuments({ relationship_id: relationshipId, current_only: true, limit: 25, cursor }); if (token === generation.current) { setFiles((items) => [...items, ...page.items.filter((file) => eligible(file, relationshipId))]); setCursor(page.next_cursor); } }
    catch { if (token === generation.current) setMoreError(true); } finally { if (moreLock.current === token) moreLock.current = null; if (token === generation.current) setMoreBusy(false); }
  }
  async function submit(event: FormEvent) {
    event.preventDefault(); const file = files.find((value) => value.id === selected);
    if (lock.current || moreLock.current !== null || !collection?.can_reconcile || !collection.request_id || !collection.request_version || !field || !file || !reason.trim() || new TextEncoder().encode(reason.trim()).length > 2000) return;
    lock.current = true; setBusy(true); onPendingChange?.(true);
    try {
      const value = await reconcileVendorEvidence(assessmentId, field.field_id, { expected_version: collection.assessment_version, request_id: collection.request_id, expected_request_version: collection.request_version, source_submission_id: file.submission_id, source_field_id: file.field_id, source_artifact_id: file.artifact_id, ...(file.response_revision_id ? { source_response_revision_id: file.response_revision_id } : {}), rationale: reason.trim() });
      if (value.collection.assessment_id !== assessmentId || value.collection.request_id !== collection.request_id || value.collection.assessment_version <= collection.assessment_version || !value.receipt?.id || value.receipt.source?.artifact_id !== file.artifact_id || value.receipt.source?.submission_id !== file.submission_id || value.receipt.source?.field_id !== file.field_id || (file.response_revision_id && value.receipt.source?.response_revision_id !== file.response_revision_id)) throw new Error("unconfirmed");
      setOutcome("linked"); setField(undefined); onChanged();
    } catch { setOutcome("unknown"); onUncertain(); }
    finally { setBusy(false); onPendingChange?.(false); }
  }
  if (state === "loading") return <p role="status">Loading requested documents…</p>;
  if (state === "error" || !collection) return <section className="v-panel v-vendor-collection"><h2>Requested documents</h2><p role="alert">Requested documents unavailable.</p><button onClick={() => setAttempt((value) => value + 1)}>Try again</button></section>;
  const fields = collection.fields.filter((value) => value.type === "vendor_document");
  if (!fields.length) return null;
  return <section className="v-panel v-vendor-collection"><h2>Requested documents</h2><ul>{fields.map((item) => <li key={item.field_id}><div><strong>{item.label}</strong><small>{collectionLabel(item.collection_state)} · {reviewLabel(item.bank_review_state)}</small></div>{!field && !outcome && item.bank_review_state === "PENDING" && collection.assessment_version === reviewVersion && Boolean(reviewRequestID) && collection.request_id === reviewRequestID && reviewableFields.includes(item.field_id) && onReview && <button type="button" onClick={() => onReview(item.field_id)}>Review {item.label}</button>}{!field && !outcome && collection.can_reconcile && collection.request_id && collection.request_version && item.collection_state === "MISSING" && <button type="button" onClick={() => { setSelected(""); setReason(""); setField(item); }}>Select document for {item.label}</button>}</li>)}</ul>
    {field && !outcome && <form className="v-intake" onSubmit={(event) => void submit(event)}><h3>{field.label}</h3><p>Choose a current vendor-submitted document.</p>{fileState === "loading" && <p role="status">Loading documents…</p>}{fileState === "error" && <p role="alert">Documents unavailable. Close and try again.</p>}{fileState === "ready" && <>{files.length === 0 && <p>No eligible documents in the loaded results.</p>}{files.length > 0 && <fieldset><legend>Document</legend>{files.map((file) => <label className="v-vendor-collection-choice" key={file.id}><input type="radio" name="document" disabled={busy} checked={selected === file.id} onChange={() => setSelected(file.id)}/><span>{file.file_name}<small>Submitted {formatDate(file.submitted_at)}{file.expires_on ? ` · Expires ${formatDate(file.expires_on)}` : ""}{file.artifact_status === "STORED_UNSCANNED" ? " · Sample file; no antivirus scan performed" : ""}</small></span></label>)}</fieldset>}{moreError && <p role="alert">More documents unavailable.</p>}{cursor && <button type="button" disabled={moreBusy || busy} onClick={() => void more()}>{moreBusy ? "Loading…" : "Load more documents"}</button>}</>}<label>Reason<textarea required disabled={busy} maxLength={2000} value={reason} onChange={(event) => setReason(event.target.value)}/></label>{new TextEncoder().encode(reason.trim()).length > 2000 && <p role="alert">Reason must be 2,000 bytes or less.</p>}<p>Review is still required after linking.</p><div className="v-intake-actions"><button type="button" disabled={busy} onClick={() => setField(undefined)}>Cancel</button><button className="v-primary" disabled={busy || moreBusy || !selected || !reason.trim() || new TextEncoder().encode(reason.trim()).length > 2000}>{busy ? "Linking…" : "Link document"}</button></div></form>}
    {outcome === "linked" && <><p role="status">Document linked. Review is still required.</p><button type="button" onClick={onChanged}>Reload assessment</button></>}{outcome === "unknown" && <><p role="alert">Document link status unknown. Reload the assessment before trying again.</p><button type="button" onClick={onChanged}>Reload assessment</button></>}
  </section>;
}

function eligible(file: DocumentOccurrence, relationshipId: string) { return file.relationship_id === relationshipId && file.current && file.submission_channel === "MAGIC_LINK" && Boolean(file.submission_id && file.field_id && file.artifact_id) && (!file.review || !["REJECTED", "EXPIRED"].includes(file.review.status)) && (!file.expires_on || (/^\d{4}-\d{2}-\d{2}$/.test(file.expires_on) && file.expires_on >= new Date().toISOString().slice(0, 10))) && (file.artifact_status === "AVAILABLE" || (file.artifact_status === "STORED_UNSCANNED" && file.demo_unscanned_allowed === true)); }
function collectionLabel(value: string) { return ({ MISSING: "Missing", RECEIVED: "Received", REUSED: "Linked", NOT_REQUIRED: "Not required", CONDITION_UNKNOWN: "Applicability unknown" } as Record<string, string>)[value] ?? "Status unknown"; }
function reviewLabel(value: string) { return ({ PENDING: "Review pending", VALIDATED: "Accepted", REJECTED: "Rejected", NOT_REQUIRED: "Review not required" } as Record<string, string>)[value] ?? "Review status unknown"; }
function formatDate(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? "date unavailable" : new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeZone: "UTC" }).format(date); }
