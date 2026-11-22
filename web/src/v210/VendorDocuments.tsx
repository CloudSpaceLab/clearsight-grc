import { useEffect, useRef, useState, type FormEvent } from "react";
import { requestBlob } from "../http";
import { reviewVendorAssessmentDocument, vendorAssessmentDocumentURL } from "../vendorAssessmentApi";
import type { VendorAssessment, VendorAssessmentDocument, VendorAssessmentReviewView } from "../vendorAssessmentTypes";
import { checkVendorAuthority } from "./vendorAuthority";

export function VendorDocuments({ assessment, documents, historical, onChanged, onFailed, onPendingChange, onReviewAvailability, reviewTarget }: { assessment: VendorAssessment; documents: VendorAssessmentDocument[]; historical: boolean; onChanged: (view: VendorAssessmentReviewView) => void; onFailed: (error: unknown) => void; onPendingChange?: (pending: boolean) => void; onReviewAvailability?: (allowed: boolean) => void; reviewTarget?: { fieldID: string; sequence: number } }) {
  const [allowed, setAllowed] = useState(false); const [checked, setChecked] = useState(false);
  const [selected, setSelected] = useState<VendorAssessmentDocument>(); const [decision, setDecision] = useState<"" | "VALIDATE" | "REJECT">("");
  const [source, setSource] = useState<"" | "VENDOR_SUPPLIED" | "BANK_VALIDATED" | "OFFICIAL_SOURCE">(""); const [documentType, setDocumentType] = useState("");
  const [validUntil, setValidUntil] = useState("");
  const [busy, setBusy] = useState(false); const [openError, setOpenError] = useState("");
  const lock = useRef(false); const generation = useRef(0); const section = useRef<HTMLElement>(null);
  useEffect(() => { generation.current++; return () => { generation.current++; }; }, [assessment.id, assessment.version, assessment.tenant_id, assessment.legal_entity_id, assessment.relationship_id]);
  useEffect(() => {
    let active = true; setAllowed(false); setChecked(false); onReviewAvailability?.(false);
    if (historical || assessment.status !== "UNDER_REVIEW" || documents.length === 0) return;
    checkVendorAuthority("document", assessment.id).then((value) => { if (active) { setAllowed(value); setChecked(true); onReviewAvailability?.(value); } }).catch(() => { if (active) setChecked(true); });
    return () => { active = false; };
  }, [assessment.id, assessment.version, assessment.status, historical, documents.length]);
  useEffect(() => {
    if (!reviewTarget || !allowed || historical || assessment.status !== "UNDER_REVIEW") return;
    const matches = documents.filter((file) => file.field_id === reviewTarget.fieldID && file.status === "SUBMITTED" && available(file));
    if (matches.length !== 1) return;
    const file = matches[0]!; setSelected(file); setDocumentType(file.document_type || ""); setDecision(""); setSource(""); setValidUntil(file.expires_on || "");
    section.current?.focus({ preventScroll: true }); section.current?.scrollIntoView?.({ block: "start" });
  }, [reviewTarget, allowed, historical, assessment.id, assessment.version, assessment.status, documents]);
  async function open(file: VendorAssessmentDocument) {
    if (!file.request_id || busy) return;
    setBusy(true); setOpenError("");
    try { const result = await requestBlob("", vendorAssessmentDocumentURL(assessment.id, file.request_id, file.artifact_id)); const url = URL.createObjectURL(result.blob); const link = document.createElement("a"); link.href = url; link.download = result.filename || file.file_name; link.click(); window.setTimeout(() => URL.revokeObjectURL(url), 60_000); }
    catch { setOpenError(`Could not download ${file.file_name}. Try again.`); } finally { setBusy(false); }
  }
  async function submit(event: FormEvent) {
    event.preventDefault(); if (!allowed || !selected || selected.status !== "SUBMITTED" || !available(selected) || !decision || !source || !documentType.trim() || lock.current || !assessment.tenant_id || !assessment.legal_entity_id || !assessment.relationship_id) return;
    const token = generation.current; lock.current = true; setBusy(true); onPendingChange?.(true);
    let view: VendorAssessmentReviewView;
    try {
      view = await reviewVendorAssessmentDocument(assessment.id, selected.artifact_id, { expected_version: assessment.version, field_id: selected.field_id, decision, document_type: documentType.trim(), evidence_class: source, ...(validUntil ? { valid_until: validUntil } : {}) });
      if (token !== generation.current) return;
      const matches = Array.isArray(view.documents) ? view.documents.filter((item) => item.field_id === selected.field_id && item.artifact_id === selected.artifact_id && (!selected.request_id || item.request_id === selected.request_id)) : [];
      if (view.assessment?.id !== assessment.id || view.assessment.tenant_id !== assessment.tenant_id || view.assessment.legal_entity_id !== assessment.legal_entity_id || view.assessment.relationship_id !== assessment.relationship_id || view.assessment.status !== "UNDER_REVIEW" || !Number.isSafeInteger(view.assessment.version) || view.assessment.version <= assessment.version || matches.length !== 1 || matches[0]?.status !== (decision === "VALIDATE" ? "VALIDATED" : "REJECTED") || matches[0]?.document_type !== documentType.trim() || matches[0]?.evidence_class !== source || (matches[0]?.expires_on || "") !== (validUntil || selected.expires_on || "")) throw new Error("unconfirmed");
    } catch (error) { if (token === generation.current) onFailed(error); return; }
    finally { lock.current = false; if (token === generation.current) { setBusy(false); onPendingChange?.(false); } }
    if (token === generation.current) { setSelected(undefined); onChanged(view); }
  }
  return <section ref={section} tabIndex={-1}><h2>Documents</h2>{openError && <p role="alert">{openError}</p>}{documents.length === 0 ? <p>No documents submitted.</p> : <ul>{documents.map((file) => <li key={`${file.field_id}:${file.artifact_id}`}><strong>{file.file_name}</strong><p>{file.status ? label(file.status) : "Status unknown"}</p>{file.reference && <p>Reference: {file.reference}</p>}{file.issued_by && <p>Issuer: {file.issued_by}</p>}{file.issued_on && <p>Issued: {file.issued_on}</p>}{file.expires_on && <p>Expires: {file.expires_on}</p>}{file.artifact_status === "STORED_UNSCANNED" && file.demo_unscanned_allowed && <p>Demo file. No antivirus scan was performed.</p>}{file.status === "SUBMITTED" && available(file) && file.request_id && <button type="button" disabled={busy} onClick={() => void open(file)}>Download {file.file_name}</button>} {allowed && file.status === "SUBMITTED" && available(file) && <button type="button" onClick={() => { setSelected(file); setDocumentType(file.document_type || ""); setDecision(""); setSource(""); setValidUntil(file.expires_on || ""); }}>Review {file.file_name}</button>}</li>)}</ul>}{checked && !allowed && !historical && assessment.status === "UNDER_REVIEW" && <p>Document review unavailable. Check your role.</p>}
    {selected && <form className="v-panel v-intake" onSubmit={(event) => void submit(event)}><h3>Review {selected.file_name}</h3><label>Decision<select required value={decision} onChange={(event) => setDecision(event.target.value as typeof decision)}><option value="">Select decision</option><option value="VALIDATE">Accept</option><option value="REJECT">Reject</option></select></label><label>Document type<input required value={documentType} onChange={(event) => setDocumentType(event.target.value)}/></label><label>Document source<select required value={source} onChange={(event) => setSource(event.target.value as typeof source)}><option value="">Select source</option><option value="VENDOR_SUPPLIED">Vendor supplied</option><option value="BANK_VALIDATED">Bank validated</option><option value="OFFICIAL_SOURCE">Official source</option></select></label><label>Valid until<input type="date" value={validUntil} onChange={(event) => setValidUntil(event.target.value)}/></label><div className="v-intake-actions"><button type="button" disabled={busy} onClick={() => setSelected(undefined)}>Cancel</button><button className="v-primary" disabled={busy}>{busy ? "Saving…" : "Save document review"}</button></div></form>}
  </section>;
}

function available(file: VendorAssessmentDocument) { return file.artifact_status === "AVAILABLE" || (file.artifact_status === "STORED_UNSCANNED" && file.demo_unscanned_allowed === true); }
function label(value: string) { const text = value.toLowerCase().replaceAll("_", " "); return text.charAt(0).toUpperCase() + text.slice(1); }
