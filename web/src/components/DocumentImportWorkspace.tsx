import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import { importDocument, loadDocumentImport, loadDocumentImports, reviewDocumentProposal } from "../documentApi";
import type { DocumentImport, DocumentImportSummary, DocumentProposal, ProposalStatus } from "../documentTypes";
import { EmptyState } from "./EmptyState";

export function DocumentImportWorkspace() {
  const [documents, setDocuments] = useState<DocumentImportSummary[]>([]);
  const [selected, setSelected] = useState<DocumentImport | null>(null);
  const [state, setState] = useState<"loading" | "live" | "unavailable">("loading");
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [purpose, setPurpose] = useState("Review source material and identify statements requiring governed review");
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
        // Keep the last durable receipt visible. A later poll may recover.
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
    const form = event.currentTarget;
    const input = form.elements.namedItem("document") as HTMLInputElement | null;
    const file = input?.files?.[0];
    if (!file) {
      setError("Choose a document to import.");
      return;
    }
    setUploading(true);
    setError(null);
    try {
      const created = await importDocument(file, purpose, sourceType);
      form.reset();
      setPurpose("Review source material and identify statements requiring governed review");
      setSourceType("DOCUMENT");
      await refresh(created.id);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "The document could not be imported.");
    } finally {
      setUploading(false);
    }
  }

  async function choose(id: string) {
    setError(null);
    try {
      setSelected(await loadDocumentImport(id));
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "The imported document could not be loaded.");
    }
  }

  async function review(proposal: DocumentProposal, status: ProposalStatus) {
    if (!selected) return;
    setError(null);
    try {
      const updated = await reviewDocumentProposal(selected.id, proposal.id, status, selected.version);
      setSelected(updated);
      setDocuments((current) => current.map((item) => item.id === updated.id ? updateSummary(item, updated) : item));
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "The review could not be recorded.");
      await refresh(selected.id);
    }
  }

  return <section className="document-import-workspace" aria-labelledby="document-import-title">
    <div className="document-import-header">
      <div><span className="eyebrow">Governed source intake</span><h2 id="document-import-title">Import and analyse documents</h2><p>Store the original first, then process supported content within bounded extraction limits and review only source-anchored proposals that still require a person.</p></div>
      <form className="document-import-form" onSubmit={submit}>
        <label><span>Document</span><input name="document" type="file" accept=".txt,.md,.csv,.docx,.xlsx,.pdf,text/plain,text/markdown,text/csv,application/pdf" required/></label>
        <label><span>Purpose</span><input value={purpose} onChange={(event) => setPurpose(event.target.value)} required/></label>
        <label><span>Source type</span><select value={sourceType} onChange={(event) => setSourceType(event.target.value)}><option value="DOCUMENT">Internal document</option><option value="REGULATORY">Regulatory source</option><option value="POLICY">Policy or standard</option><option value="FINDING">Finding or report</option></select></label>
        <button className="primary-button" type="submit" disabled={uploading}>{uploading ? "Storing…" : "Import document"}</button>
      </form>
    </div>
    {error && <p className="error-text" role="alert">{error}</p>}
    {state === "loading" ? <div className="workspace-loading" aria-live="polite">Loading imported documents…</div> : state === "unavailable" ? <EmptyState label="Document imports" title="Imported documents could not be loaded" description="No import or analysis claims are shown while the service is unavailable." action="Try again" onAction={() => void refresh()}/> : !documents.length ? <EmptyState label="Document imports" title="No documents imported" description="Import a TXT, Markdown, CSV, DOCX or XLSX file to create a governed review record. PDFs are retained but require an approved extractor or OCR adapter before analysis."/> : <div className="document-import-layout">
      <div className="document-import-list" aria-label="Imported documents">{documents.map((document) => <button key={document.id} type="button" className={selected?.id === document.id ? "document-import-row active" : "document-import-row"} onClick={() => void choose(document.id)}><strong>{document.file_name}</strong><span>{summaryLabel(document)}</span><small>{new Date(document.created_at).toLocaleString()}</small></button>)}</div>
      {selected && <DocumentInspector document={selected} onReview={review}/>} 
    </div>}
  </section>;
}

function DocumentInspector({ document, onReview }: { document: DocumentImport; onReview: (proposal: DocumentProposal, status: ProposalStatus) => void }) {
  const pending = document.proposals.filter((proposal) => proposal.status === "PENDING_REVIEW");
  const reviewed = document.proposals.filter((proposal) => proposal.status !== "PENDING_REVIEW");
  const processing = isProcessing(document);
  const sectionsTotal = document.sections_total ?? document.sections.length;
  const sectionsOmitted = document.sections_omitted ?? 0;
  const contentTruncated = document.content_truncated ?? false;
  const terminalLabel = document.extraction_status === "FAILED" ? "Extraction failed" : document.extraction_status === "UNSUPPORTED" ? "Stored only" : pending.length ? `${pending.length} to review` : "No review pending";
  return <article className="document-import-inspector">
    <header><div><span className="eyebrow">{human(document.source_type)}</span><h2>{document.file_name}</h2><p>{document.purpose}</p></div><div className="document-state"><span>{human(document.extraction_status)}</span><strong>{processing ? "Processing stored source" : terminalLabel}</strong></div></header>
    <div className="import-review-summary" aria-label="Prepared import review"><span><strong>{pending.length}</strong> require review</span><span><strong>{reviewed.length}</strong> reviewed</span><span><strong>{document.sections.length}</strong> of {sectionsTotal} source sections retained</span></div>
    {processing && <section className="workspace-loading" aria-live="polite" aria-busy="true"><strong>Original stored successfully.</strong><p>Extraction and analysis are running as recoverable background work. This page will update when processing completes.</p></section>}
    {document.limitations.length > 0 && <section className="document-limitations"><h3>Important limitations</h3>{document.limitations.map((item) => <p key={item}>{item}</p>)}</section>}
    {!processing && <>
      <section><div className="section-header"><div><h3>Review required</h3><p>Accepting a proposal records review only; it does not create or approve an obligation, control or conclusion.</p></div></div>{pending.length ? <div className="proposal-list">{pending.map((proposal) => <ProposalCard key={proposal.id} proposal={proposal} onReview={onReview}/>)}</div> : <div className="calm-empty compact"><span>✓</span><div><strong>No proposal awaiting review</strong><p>{document.analysis_status === "UNAVAILABLE" ? "No review proposal is available from this source." : "Reviewed proposals remain available below for reconstruction."}</p></div></div>}</section>
      {reviewed.length > 0 && <details className="import-secondary"><summary><span>Reviewed proposals</span><strong>{reviewed.length}</strong></summary><div className="proposal-list">{reviewed.map((proposal) => <ProposalCard key={proposal.id} proposal={proposal} onReview={onReview}/>)}</div></details>}
    </>}
    <details className="import-secondary"><summary><span>Source reconstruction</span><strong>{document.sections.length} retained</strong></summary><div><dl className="document-metadata"><div><dt>Original hash</dt><dd><code>{document.sha256}</code></dd></div><div><dt>Artifact state</dt><dd>{human(document.artifact_status)}</dd></div><div><dt>Extraction</dt><dd>{human(document.extraction_method)}</dd></div><div><dt>Completeness</dt><dd>{contentTruncated || sectionsOmitted ? `${sectionsOmitted} sections omitted; content bounded` : "Complete within configured extraction budgets"}</dd></div><div><dt>Version</dt><dd>{document.version}</dd></div></dl>{document.sections.length > 0 && <div className="document-sections">{document.sections.map((section) => <details key={section.id}><summary>{section.title}</summary><pre>{section.text}</pre></details>)}</div>}</div></details>
  </article>;
}

function ProposalCard({ proposal, onReview }: { proposal: DocumentProposal; onReview: (proposal: DocumentProposal, status: ProposalStatus) => void }) {
  return <article className="proposal-card"><div className="proposal-title"><div><span>{human(proposal.kind)}</span><h4>{proposal.title}</h4></div><mark>{human(proposal.status)}</mark></div><p>{proposal.statement}</p><blockquote>{proposal.anchor.quote}</blockquote><small>Source: {proposal.anchor.sheet ? `${proposal.anchor.sheet}, row ${proposal.anchor.row_start}` : proposal.anchor.page ? `page ${proposal.anchor.page}` : proposal.anchor.section_id}</small>{proposal.status === "PENDING_REVIEW" && <div className="proposal-actions"><button className="secondary-button" type="button" onClick={() => onReview(proposal, "REJECTED")}>Reject proposal</button><button className="primary-button" type="button" onClick={() => onReview(proposal, "ACCEPTED")}>Accept for governed follow-up</button></div>}</article>;
}

function isProcessing(document: Pick<DocumentImport, "extraction_status" | "analysis_status">) {
  return document.extraction_status === "PENDING" || document.analysis_status === "PENDING";
}

function summaryLabel(document: DocumentImportSummary) {
  if (document.extraction_status === "PENDING" || document.analysis_status === "PENDING") return "Stored · processing";
  if (document.extraction_status === "FAILED") return "Extraction failed · original retained";
  if (document.extraction_status === "UNSUPPORTED") return "Stored · extraction unavailable";
  if (document.pending_proposal_count) return `${document.pending_proposal_count} proposal${document.pending_proposal_count === 1 ? "" : "s"} require review`;
  return "No proposal awaiting review";
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
