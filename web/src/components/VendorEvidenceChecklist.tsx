import { useEffect, useRef, useState } from "react";
import { apiErrorKind } from "../http";
import { loadVendorCollection, reconcileVendorEvidence, type VendorCollection, type VendorCollectionField } from "../vendorCollectionApi";
import { documentReviewAllowed, type DocumentOccurrence } from "../submittedDocumentApi";
import { DocumentDemoNotice } from "./documents/DocumentFile";
import type { VendorAssessmentDocument } from "../vendorAssessmentTypes";
import { Button, EmptyState, FocusedSheet, Notice, StatusBadge, TextArea, type StatusTone } from "./ui";
import { DocumentBrowser } from "./documents/DocumentBrowser";
import { DocumentPreview } from "./documents/DocumentPreview";
import "./vendor-evidence-checklist.css";

type Props = {
  assessmentID: string; assessmentVersion: number; relationshipID: string;
  onChanged?: () => void | Promise<void>;
  onLoaded?: (collection: VendorCollection | undefined) => void;
  documents?: VendorAssessmentDocument[];
  onReviewDocument?: (document: VendorAssessmentDocument, decision: "VALIDATE" | "REJECT") => void;
  onOpenDocument?: (document: VendorAssessmentDocument) => void;
};
type Filter = "ALL" | "VENDOR" | "BANK" | "ACCEPTED";

export function VendorEvidenceChecklist({ assessmentID, assessmentVersion, relationshipID, onChanged, onLoaded, documents = [], onReviewDocument, onOpenDocument }: Props) {
  const [collection, setCollection] = useState<VendorCollection>();
  const [state, setState] = useState<"loading" | "ready" | "error">("loading");
  const [reload, setReload] = useState(0);
  const [filter, setFilter] = useState<Filter>("ALL");
  const [target, setTarget] = useState<VendorCollectionField>();
  const [source, setSource] = useState<DocumentOccurrence>();
  const [preview, setPreview] = useState<DocumentOccurrence>();
  const [rationale, setRationale] = useState("");
  const [error, setError] = useState("");
  const [conflict, setConflict] = useState(false);
  const [notice, setNotice] = useState("");
  const [refreshWarning, setRefreshWarning] = useState(false);
  const [saving, setSaving] = useState(false);
  const savingRef = useRef(false);
  const mounted = useRef(true);
  const loadedRef = useRef(onLoaded);
  loadedRef.current = onLoaded;
  useEffect(() => { mounted.current = true; return () => { mounted.current = false; }; }, []);
  useEffect(() => {
    const controller = new AbortController();
    setState("loading"); loadedRef.current?.(undefined);
    void loadVendorCollection(assessmentID, controller.signal).then((value) => {
      if (controller.signal.aborted) return;
      setCollection(value); setState("ready"); loadedRef.current?.(value);
    }).catch(() => { if (!controller.signal.aborted) { setCollection(undefined); setState("error"); } });
    return () => controller.abort();
  }, [assessmentID, assessmentVersion, reload]);

  function choose(field: VendorCollectionField) {
    setTarget(field); setSource(undefined); setRationale(""); setError(""); setConflict(false);
  }
  function close() { if (!savingRef.current) { setTarget(undefined); setSource(undefined); setError(""); } }
  async function save() {
    if (savingRef.current || conflict || !collection?.can_reconcile || !collection.request_id || !collection.request_version || !target || !source || !rationale.trim()) return;
    savingRef.current = true; setSaving(true); setError("");
    try {
      const result = await reconcileVendorEvidence(assessmentID, target.field_id, {
        expected_version: collection.assessment_version, request_id: collection.request_id, expected_request_version: collection.request_version,
        source_submission_id: source.submission_id, source_field_id: source.field_id, source_artifact_id: source.artifact_id,
        source_response_revision_id: source.response_revision_id, rationale: rationale.trim(),
      });
      if (!mounted.current) return;
      setCollection(result.collection); loadedRef.current?.(result.collection); setTarget(undefined); setSource(undefined);
      setNotice("Document linked.");
      setRefreshWarning(false);
      try { await onChanged?.(); } catch { if (mounted.current) setRefreshWarning(true); }
    } catch (cause) {
      if (!mounted.current) return;
      const stale = apiErrorKind(cause) === "conflict";
      setConflict(stale);
      setError(stale ? "The request or evidence changed. Reload the checklist and choose the document again." : "The document could not be linked. Check its availability and your review access, then try again. Your reason is saved in this form.");
    } finally { savingRef.current = false; if (mounted.current) setSaving(false); }
  }

  if (state === "loading") return <section className="vendor-checklist" aria-label="Request checklist" aria-busy="true"><p role="status">Checking requested items and evidence already held…</p></section>;
  if (state === "error" || !collection) return <section className="vendor-checklist" aria-label="Request checklist"><Notice tone="error"><strong>Checklist unavailable.</strong> <Button onPress={() => setReload((value) => value + 1)}>Reload checklist</Button></Notice></section>;

  const accepted = collection.fields.filter((field) => field.bank_review_state === "VALIDATED" && !field.vendor_action_required).length;
  const visible = collection.fields.filter((field) => filter === "VENDOR" ? isMissing(field) : filter === "BANK" ? field.bank_review_state === "PENDING" : filter === "ACCEPTED" ? field.bank_review_state === "VALIDATED" && !field.vendor_action_required : true)
    .toSorted((a, b) => priority(a) - priority(b));
  const summaries: { id: Filter; label: string; count: number }[] = [
    { id: "ALL", label: "All items", count: collection.fields.length },
    { id: "VENDOR", label: "Missing", count: collection.fields.filter(isMissing).length },
    { id: "BANK", label: "Awaiting review", count: collection.bank_pending_count },
    { id: "ACCEPTED", label: "Accepted", count: accepted },
  ];

  return <section className="vendor-checklist" aria-label="Request checklist">
    <header className="vendor-checklist__heading"><div><h3>Checklist</h3><p>Current request · Checked {date(collection.observed_at)}</p></div><Button variant="quiet" onPress={() => setReload((value) => value + 1)}>Refresh checklist</Button></header>
    {notice && <Notice tone="success">{notice}</Notice>}
    {refreshWarning && <Notice tone="warning">The document link is saved. Reload the vendor review to refresh its other decisions.</Notice>}
    <div className="vendor-checklist__summary" role="group" aria-label="Filter requested items">
      {summaries.map((item) => <div key={item.id} className={`vendor-checklist__stat vendor-checklist__stat--${item.id.toLowerCase()}`}><strong aria-hidden="true">{item.count}</strong><Button variant={filter === item.id ? "secondary" : "quiet"} aria-pressed={filter === item.id} aria-label={`${item.label} ${item.count}`} onPress={() => setFilter(item.id)}>{item.label}</Button></div>)}
    </div>
    {!collection.prepared && <Notice tone="info">Prepare the request to link existing documents.</Notice>}
    {collection.prepared && collection.fields.length > 0 && collection.vendor_pending_count === 0 && !collection.fields.some((field) => field.collection_state === "CONDITION_UNKNOWN") && <p className="vendor-checklist__collection-complete"><strong>No missing items.</strong></p>}
    <div className="vendor-checklist__items">
      {visible.map((field) => {
        const status = fieldStatus(field);
        const document = documents.find((item) => item.field_id === field.field_id);
        const reviewReason = document ? reviewUnavailableReason(document) : "";
        const canChoose = collection.prepared && collection.can_reconcile && field.type.toLowerCase() === "vendor_document" && !["NOT_REQUIRED", "CONDITION_UNKNOWN"].includes(field.collection_state) && field.bank_review_state !== "VALIDATED";
        return <article key={field.field_id} className="vendor-checklist__item" aria-label={field.label}>
          <div className="vendor-checklist__item-main"><div className="vendor-checklist__item-title"><h4>{field.label}</h4><StatusBadge tone={status.tone}>{status.label}</StatusBadge></div>{status.detail && <p>{status.detail}</p>}
            {field.resolution && <div className="vendor-checklist__evidence"><strong>{field.resolution.source.file_name}</strong><span>Submitted {date(field.resolution.source.submitted_at)} · Linked {date(field.resolution.reconciled_at)}</span><p>{field.resolution.rationale}</p><DocumentDemoNotice file={field.resolution.source}/><Button variant="quiet" size="compact" onPress={() => setPreview(field.resolution!.source)}>View existing document</Button></div>}
            {!field.resolution && document && <div className="vendor-checklist__evidence"><strong>{document.file_name}</strong>{document.expires_on && <span>Expires {date(document.expires_on)}</span>}<DocumentDemoNotice file={document}/>{onOpenDocument && <Button variant="quiet" size="compact" onPress={() => onOpenDocument(document)}>View submitted document</Button>}</div>}
          </div>
          <div className="vendor-checklist__item-actions">
            {document && field.bank_review_state === "PENDING" && onReviewDocument && <><Button isDisabled={Boolean(reviewReason)} onPress={() => onReviewDocument(document, "VALIDATE")}>Review document</Button>{reviewReason && <p>{reviewReason}</p>}</>}
            {canChoose && <Button variant="quiet" onPress={() => choose(field)}>{field.collection_state === "REUSED" || field.collection_state === "RECEIVED" ? "Replace document" : "Use existing document"}</Button>}
          </div>
        </article>;
      })}
      {visible.length === 0 && <EmptyState population="Requested items matching this filter" title="No matching items" description="Select All items to view the checklist."/>}
    </div>
    {target && <FocusedSheet label={`Use existing document for ${target.label}`} size="wide" isDismissable={!saving} onClose={close}>
      <div className="vendor-checklist__selection"><h2>Use existing document</h2><p>Requested item: <strong>{target.label}</strong></p>
        {error && <Notice tone="error">{error}{conflict && <Button onPress={() => { close(); setReload((value) => value + 1); }}>Reload checklist</Button>}</Notice>}
        {!source ? <DocumentBrowser scopeLabel="Documents already submitted for this vendor service" relationshipID={relationshipID} onChoose={setSource} selectionUnavailableReason={unavailableReason}/> : <>
          <div className="vendor-checklist__evidence"><strong>{source.file_name}</strong><span>Submitted {date(source.submitted_at)} · {source.form_title}</span><span>Original request item: {source.field_label}</span><DocumentDemoNotice file={source}/><Button variant="quiet" onPress={() => setPreview(source)}>Preview selected document</Button></div>
          <TextArea label="Reason for reuse" description="Confirm the document covers this service and requirement." value={rationale} onChange={setRationale} maxLength={2000} rows={3} isDisabled={saving}/>
          <div className="vendor-checklist__selection-actions"><Button isDisabled={saving} onPress={() => setSource(undefined)}>Choose another document</Button><Button variant="primary" isLoading={saving} isDisabled={!rationale.trim() || conflict} onPress={() => void save()}>Use this document</Button></div>
        </>}
      </div>
    </FocusedSheet>}
    {preview && <DocumentPreview file={preview} onClose={() => setPreview(undefined)}/>}
  </section>;
}

function priority(field: VendorCollectionField) { return field.vendor_action_required ? 0 : field.bank_review_state === "PENDING" ? 1 : field.collection_state === "CONDITION_UNKNOWN" ? 2 : field.bank_review_state === "VALIDATED" ? 4 : 3; }
function isMissing(field: VendorCollectionField) { return field.vendor_action_required && field.collection_state !== "CONDITION_UNKNOWN"; }
function fieldStatus(field: VendorCollectionField): { label: string; tone: StatusTone; detail?: string } {
  if (field.collection_state === "CONDITION_UNKNOWN") return { label: "Applicability pending", tone: "unknown", detail: "Depends on unanswered questions." };
  if (field.vendor_action_required) return { label: "Missing", tone: "warning", detail: field.resolution && !field.resolution.source.current ? "Document replaced. Link the current version." : field.resolution && !documentReviewAllowed(field.resolution.source) ? "Linked document unavailable. Review its source." : field.bank_review_state === "REJECTED" || field.resolution?.source.review?.status === "REJECTED" ? "Rejected evidence. Vendor replacement required." : field.resolution ? "Linked evidence needs replacement." : "Vendor" };
  if (field.collection_state === "NOT_REQUIRED") return { label: "Not required", tone: "neutral" };
  if (field.bank_review_state === "VALIDATED") return { label: "Accepted", tone: "success" };
  if (field.bank_review_state === "REJECTED") return { label: "Rejected", tone: "error", detail: "Review the finding before proceeding." };
  if (field.bank_review_state === "PENDING") return { label: "Awaiting review", tone: "info", detail: "No upload needed." };
  if (field.collection_state === "REUSED") return { label: "Linked", tone: "info", detail: "No upload needed." };
  if (field.collection_state === "RECEIVED") return { label: "Received", tone: "neutral" };
  return { label: "Not supplied", tone: "neutral", detail: "Optional" };
}
function date(value: string) { const parsed = new Date(value); return Number.isNaN(parsed.getTime()) ? "Date not recorded" : parsed.toLocaleDateString("en-GB", { day: "numeric", month: "short", year: "numeric" }); }
function unavailableReason(file: DocumentOccurrence) {
  if (file.submission_channel !== "MAGIC_LINK") return "Choose a document submitted by the vendor.";
  if (!documentReviewAllowed(file)) return "This file is not available for use. Choose an available document.";
  if (!file.current) return "This document has been replaced. Choose its current version.";
  if (file.review?.status === "REJECTED") return "Document rejected. Choose different evidence.";
  if (file.expires_on && (!Number.isFinite(Date.parse(file.expires_on)) || file.expires_on.slice(0, 10) < new Date().toISOString().slice(0, 10))) return "This document is expired or its validity is unclear. Choose current evidence.";
  return "";
}
function reviewUnavailableReason(file: VendorAssessmentDocument) {
  if (file.status === "REJECTED") return "This document was rejected. Request a replacement before reviewing it again.";
  if (file.status === "EXPIRED") return "This document has expired. Request current evidence.";
  return documentReviewAllowed(file) ? "" : "This file is unavailable for review. Wait for its safety check or choose another document.";
}
