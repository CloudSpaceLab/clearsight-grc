import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import { importDocument, loadDocumentImport, loadDocumentImports, reviewDocumentProposal } from "../documentApi";
import type { DocumentImport, DocumentImportSummary, DocumentProposal, ProposalStatus } from "../documentTypes";
import { apiErrorKind } from "../http";
import { EmptyState } from "./EmptyState";
import { FileDropzone } from "./FileDropzone";

const documentAccept = ".txt,.md,.csv,.docx,.xlsx,.pdf,text/plain,text/markdown,text/csv,application/pdf,application/vnd.openxmlformats-officedocument.wordprocessingml.document,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet";

export function DocumentImportWorkspace() {
  const [documents, setDocuments] = useState<DocumentImportSummary[]>([]);
  const [selected, setSelected] = useState<DocumentImport | null>(null);
  const [state, setState] = useState<"loading" | "live" | "unavailable">("loading");
  const [uploading, setUploading] = useState(false);
  const [reviewingProposalID, setReviewingProposalID] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [pendingFile, setPendingFile] = useState<File | null>(null);
  const [purpose, setPurpose] = useState("");
  const [sourceType, setSourceType] = useState("DOCUMENT");

  async function refresh(selectID?: string) {
    setState("loading");
    try {
      const values = await loadDocumentImports();
      setDocuments(values);
      const target = selectID ?? selected?.id ?? values[0]?.id;
      setSelected(target ? await loadDocumentImport(target) : null);
      setState("live");
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Imported documents could not be loaded.");
      setState("unavailable");
    }
  }

  useEffect(() => { void refresh(); }, []);

  useEffect(() => {
    if (!selected || !isProcessing(selected)) return;
    let active = true;
    let inFlight = false;
    const poll = async () => {
      if (!active || inFlight) return;
      inFlight = true;
      try {
        const [detail, summaries] = await Promise.all([loadDocumentImport(selected.id), loadDocumentImports()]);
        if (!active) return;
        setSelected(detail);
        setDocuments(summaries);
      } catch {
        // Preserve the last durable receipt. A later poll can recover.
      } finally {
        inFlight = false;
      }
    };
    const timer = window.setInterval(() => { void poll(); }, 1500);
    return () => {
      active = false;
      window.clearInterval(timer);
    };
  }, [selected?.id, selected?.extraction_status, selected?.analysis_status]);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!pendingFile) {
      setError("Choose a document to import.");
      return;
    }
    if (!purpose.trim()) {
      setError("Say what reviewers should look for in this document.");
      return;
    }
    setUploading(true);
    setError(null);
    try {
      const created = await importDocument(pendingFile, purpose.trim(), sourceType);
      setPendingFile(null);
      setPurpose("");
      setSourceType("DOCUMENT");
      await refresh(created.id);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "The document could not be imported.");
    } finally {
      setUploading(false);
    }
  }

  async function choose(id: string) {
    if (reviewingProposalID) return;
    setError(null);
    try {
      setSelected(await loadDocumentImport(id));
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "The imported document could not be loaded.");
    }
  }

  async function review(proposal: DocumentProposal, status: ProposalStatus) {
    if (!selected || reviewingProposalID) return;
    setReviewingProposalID(proposal.id);
    setError(null);
    try {
      const updated = await reviewDocumentProposal(selected.id, proposal.id, status, selected.version);
      setSelected(updated);
      setDocuments((current) => current.map((item) => item.id === updated.id ? updateSummary(item, updated) : item));
    } catch (cause) {
      const conflict = apiErrorKind(cause) === "conflict";
      setError(conflict ? "This import changed while you were reviewing it. The latest version has been loaded." : cause instanceof Error ? cause.message : "The review could not be recorded.");
      await refresh(selected.id);
    } finally {
      setReviewingProposalID(null);
    }
  }

  return <section className="document-import-workspace" aria-labelledby="document-import-title">
    <div className="document-import-header">
      <div>
        <span className="eyebrow">Document import</span>
        <h2 id="document-import-title">Import a document</h2>
        <p>Upload the original document. ClearSight will show processing status and any extracted proposals that need review.</p>
      </div>
      <form className="document-import-form" onSubmit={submit}>
        <FileDropzone
          label="Document"
          description="Drop a supported document here or choose one from this device."
          accept={documentAccept}
          disabled={uploading}
          busy={uploading}
          actionLabel="Choose document"
          fileName={pendingFile?.name}
          fileSize={pendingFile?.size}
          onSelect={setPendingFile}
        />
        <label><span>What should reviewers look for?</span><input value={purpose} onChange={(event) => setPurpose(event.target.value)} placeholder="e.g. Changes affecting card operations" required disabled={uploading}/></label>
        <label><span>Document type</span><select value={sourceType} onChange={(event) => setSourceType(event.target.value)} disabled={uploading}><option value="DOCUMENT">Internal document</option><option value="REGULATORY">Regulatory source</option><option value="POLICY">Policy or standard</option><option value="FINDING">Finding or report</option></select></label>
        <button className="primary-button" type="submit" disabled={uploading}>{uploading ? "Storing…" : "Import document"}</button>
      </form>
    </div>
    {error && <p className="error-text" role="alert">{error}</p>}
    {state === "loading"
      ? <div className="workspace-loading" aria-live="polite">Loading imported documents…</div>
      : state === "unavailable"
        ? <EmptyState kind="unavailable" label="Document imports" title="Imported documents could not be loaded" description="Try again before relying on this list." action="Try again" onAction={() => void refresh()}/>
        : !documents.length
          ? <EmptyState label="Document imports" title="No documents imported" description="Import a TXT, Markdown, CSV, DOCX, XLSX or PDF document. PDFs are stored even when text extraction is unavailable."/>
          : <div className="document-import-layout">
              <div className="document-import-list" aria-label="Imported documents">{documents.map((document) => <button key={document.id} type="button" disabled={Boolean(reviewingProposalID)} className={selected?.id === document.id ? "document-import-row active" : "document-import-row"} onClick={() => void choose(document.id)}><strong>{document.file_name}</strong><span>{summaryLabel(document)}</span><small>{new Date(document.created_at).toLocaleString()}</small></button>)}</div>
              {selected && <DocumentInspector document={selected} reviewingProposalID={reviewingProposalID} onReview={review}/>} 
            </div>}
  </section>;
}

function DocumentInspector({ document, reviewingProposalID, onReview }: { document: DocumentImport; reviewingProposalID: string | null; onReview: (proposal: DocumentProposal, status: ProposalStatus) => void }) {
  const pending = document.proposals.filter((proposal) => proposal.status === "PENDING_REVIEW");
  const reviewed = document.proposals.filter((proposal) => proposal.status !== "PENDING_REVIEW");
  const processing = isProcessing(document);
  const sectionsTotal = document.sections_total ?? document.sections.length;
  const sectionsOmitted = document.sections_omitted ?? 0;
  const contentTruncated = document.content_truncated ?? false;
  const terminalLabel = document.extraction_status === "FAILED" ? "Extraction failed" : document.extraction_status === "UNSUPPORTED" ? "Stored only" : pending.length ? `${pending.length} to review` : "No review pending";

  return <article className="document-import-inspector">
    <header><div><span className="eyebrow">{human(document.source_type)}</span><h2>{document.file_name}</h2><p>{document.purpose}</p></div><div className="document-state"><span>{human(document.extraction_status)}</span><strong>{processing ? "Processing stored source" : terminalLabel}</strong></div></header>
    <div className="import-review-summary" aria-label="Import review summary"><span><strong>{pending.length}</strong> to review</span><span><strong>{reviewed.length}</strong> reviewed</span><span><strong>{document.sections.length}</strong> of {sectionsTotal} source sections retained</span></div>
    {processing && <section className="workspace-loading" aria-live="polite" aria-busy="true"><strong>Original stored successfully.</strong><p>Extraction and analysis are running in the background. This page will update when processing completes.</p></section>}
    {document.limitations.length > 0 && <section className="document-limitations"><h3>Important limitations</h3>{document.limitations.map((item) => <p key={item}>{item}</p>)}</section>}
    {!processing && <>
      <section><div className="section-header"><div><h3>Review required</h3><p>Accepting a proposal records your review. It does not by itself create or approve a requirement, control or conclusion.</p></div></div>{pending.length ? <div className="proposal-list">{pending.map((proposal) => <ProposalCard key={proposal.id} proposal={proposal} busy={reviewingProposalID === proposal.id} locked={Boolean(reviewingProposalID)} onReview={onReview}/>)}</div> : <div className="calm-empty compact"><span>✓</span><div><strong>Nothing waiting for review</strong><p>{document.analysis_status === "UNAVAILABLE" ? "No review proposal is available from this source." : "Reviewed proposals remain available below."}</p></div></div>}</section>
      {reviewed.length > 0 && <details className="import-secondary"><summary><span>Reviewed proposals</span><strong>{reviewed.length}</strong></summary><div className="proposal-list">{reviewed.map((proposal) => <ProposalCard key={proposal.id} proposal={proposal} busy={false} locked={Boolean(reviewingProposalID)} onReview={onReview}/>)}</div></details>}
    </>}
    <details className="import-secondary"><summary><span>Original source details</span><strong>{document.sections.length} retained</strong></summary><div><dl className="document-metadata"><div><dt>Original hash</dt><dd><code>{document.sha256}</code></dd></div><div><dt>File status</dt><dd>{human(document.artifact_status)}</dd></div><div><dt>Text extraction</dt><dd>{human(document.extraction_method)}</dd></div><div><dt>Completeness</dt><dd>{contentTruncated || sectionsOmitted ? `${sectionsOmitted} sections omitted; content bounded` : "Complete within configured extraction limits"}</dd></div><div><dt>Version</dt><dd>{document.version}</dd></div></dl>{document.sections.length > 0 && <div className="document-sections">{document.sections.map((section) => <details key={section.id}><summary>{section.title}</summary><pre>{section.text}</pre></details>)}</div>}</div></details>
  </article>;
}

function ProposalCard({ proposal, busy, locked, onReview }: { proposal: DocumentProposal; busy: boolean; locked: boolean; onReview: (proposal: DocumentProposal, status: ProposalStatus) => void }) {
  return <article className="proposal-card" aria-busy={busy || undefined}><div className="proposal-title"><div><span>{human(proposal.kind)}</span><h4>{proposal.title}</h4></div><mark>{human(proposal.status)}</mark></div><p>{proposal.statement}</p><blockquote>{proposal.anchor.quote}</blockquote><small>Source: {proposal.anchor.sheet ? `${proposal.anchor.sheet}, row ${proposal.anchor.row_start}` : proposal.anchor.page ? `page ${proposal.anchor.page}` : proposal.anchor.section_id}</small>{proposal.status === "PENDING_REVIEW" && <div className="proposal-actions"><button className="secondary-button" type="button" disabled={locked} onClick={() => onReview(proposal, "REJECTED")}>{busy ? "Recording…" : "Reject"}</button><button className="primary-button" type="button" disabled={locked} onClick={() => onReview(proposal, "ACCEPTED")}>{busy ? "Recording…" : "Accept proposal"}</button></div>}</article>;
}

function isProcessing(document: Pick<DocumentImport, "extraction_status" | "analysis_status">) {
  return document.extraction_status === "PENDING" || document.analysis_status === "PENDING";
}

function summaryLabel(document: DocumentImportSummary) {
  if (document.extraction_status === "PENDING" || document.analysis_status === "PENDING") return "Stored · processing";
  if (document.extraction_status === "FAILED") return "Extraction failed · original retained";
  if (document.extraction_status === "UNSUPPORTED") return "Stored · extraction unavailable";
  if (document.pending_proposal_count) return `${document.pending_proposal_count} proposal${document.pending_proposal_count === 1 ? "" : "s"} to review`;
  return "No proposals waiting for review";
}

function updateSummary(current: DocumentImportSummary, detail: DocumentImport): DocumentImportSummary {
  const pending = detail.proposals.filter((proposal) => proposal.status === "PENDING_REVIEW").length;
  return {
    ...current,
    extraction_status: detail.extraction_status,
    analysis_status: detail.analysis_status,
    sections_total: detail.sections_total ?? detail.sections.length,
    sections_omitted: detail.sections_omitted ?? 0,
    proposals_total: detail.proposals_total ?? detail.proposals.length,
    proposals_omitted: detail.proposals_omitted ?? 0,
    pending_proposal_count: pending,
    reviewed_proposal_count: detail.proposals.length - pending,
    content_truncated: detail.content_truncated ?? false,
    processed_at: detail.processed_at,
    updated_at: detail.updated_at,
    version: detail.version,
  };
}

function human(value: string) {
  return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}
