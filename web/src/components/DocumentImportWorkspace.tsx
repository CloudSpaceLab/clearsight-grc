import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import { applyDocumentCoverageSuggestion, importDocument, loadDocumentCoverage, loadDocumentImport, loadDocumentImports, recompareDocumentCoverage, reviewDocumentCoverage, reviewDocumentProposal } from "../documentApi";
import type { CoverageCandidate, CoverageDecision, CoverageSuggestion, DocumentCoverage, DocumentImport, DocumentImportSummary, DocumentProposal, ProposalStatus } from "../documentTypes";
import { apiErrorKind } from "../http";
import { DocumentProposalHandoff } from "./DocumentProposalHandoff";
import { EmptyState } from "./EmptyState";
import { FileDropzone } from "./FileDropzone";

const documentAccept = ".txt,.md,.csv,.docx,.xlsx,.pdf,text/plain,text/markdown,text/csv,application/pdf,application/vnd.openxmlformats-officedocument.wordprocessingml.document,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet";
const maximumDocumentBytes = 20 * 1024 * 1024;

type ImportFocus = { documentID?: string; proposalID?: string };

export function DocumentImportWorkspace() {
  const [documents, setDocuments] = useState<DocumentImportSummary[]>([]);
  const [selected, setSelected] = useState<DocumentImport | null>(null);
  const [coverage, setCoverage] = useState<DocumentCoverage | null>(null);
  const [state, setState] = useState<"loading" | "live" | "unavailable">("loading");
  const [uploading, setUploading] = useState(false);
  const [reviewingProposalID, setReviewingProposalID] = useState<string | null>(null);
  const [coverageActionID, setCoverageActionID] = useState<string | null>(null);
  const [coverageNotice, setCoverageNotice] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [pendingFile, setPendingFile] = useState<File | null>(null);
  const [purpose, setPurpose] = useState("");
  const [sourceType, setSourceType] = useState("DOCUMENT");
  const [intakeOpen, setIntakeOpen] = useState(false);
  const [focus, setFocus] = useState<ImportFocus>(focusedImportTarget);

  async function refresh(selectID?: string) {
    setState("loading");
    try {
      const values = await loadDocumentImports();
      setDocuments(values);
      if (!values.length) setIntakeOpen(true);
      const target = selectID ?? selected?.id ?? values[0]?.id;
      if (target) {
        const [detail, comparison] = await Promise.all([loadDocumentImport(target), loadDocumentCoverage(target)]);
        setSelected(detail);
        setCoverage(comparison);
      } else {
        setSelected(null);
        setCoverage(null);
      }
      setState("live");
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Imported documents could not be loaded.");
      setState("unavailable");
    }
  }

  useEffect(() => {
    const syncFocus = () => setFocus(focusedImportTarget());
    window.addEventListener("hashchange", syncFocus);
    return () => window.removeEventListener("hashchange", syncFocus);
  }, []);

  useEffect(() => { void refresh(focus.documentID); }, [focus.documentID]);

  useEffect(() => {
    if (!selected || !focus.proposalID) return;
    const element = window.document.getElementById(`document-proposal-${focus.proposalID}`);
    if (!element) return;
    const disclosure = element.closest("details");
    if (disclosure instanceof HTMLDetailsElement) disclosure.open = true;
    window.requestAnimationFrame(() => element.scrollIntoView({ block: "start" }));
  }, [selected?.id, selected?.version, focus.proposalID]);

  useEffect(() => {
    if (!selected || (!isProcessing(selected) && coverage?.status !== "PENDING" && coverage?.status !== "COMPARING")) return;
    let active = true;
    let inFlight = false;
    const poll = async () => {
      if (!active || inFlight) return;
      inFlight = true;
      try {
        const [detail, summaries, comparison] = await Promise.all([loadDocumentImport(selected.id), loadDocumentImports(), loadDocumentCoverage(selected.id)]);
        if (!active) return;
        setSelected(detail);
        setDocuments(summaries);
        setCoverage(comparison);
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
  }, [selected?.id, selected?.extraction_status, selected?.analysis_status, coverage?.status]);

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
      setIntakeOpen(false);
      await refresh(created.id);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "The document could not be imported.");
    } finally {
      setUploading(false);
    }
  }

  async function choose(id: string) {
    if (reviewingProposalID || coverageActionID) return;
    setError(null);
    try {
      const [detail, comparison] = await Promise.all([loadDocumentImport(id), loadDocumentCoverage(id)]);
      setSelected(detail);
      setCoverage(comparison);
      setCoverageNotice(null);
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

  async function reviewCoverage(candidate: CoverageCandidate, decision: CoverageDecision, reason = "") {
    if (!selected || !coverage || coverageActionID) return;
    const matchID = decision === "ACCEPT_MATCH" ? candidate.matches[0]?.id : undefined;
    setCoverageActionID(candidate.id);
    setCoverageNotice(null);
    setError(null);
    try {
      setCoverage(await reviewDocumentCoverage(selected.id, coverage.version, [{ candidate_id: candidate.id, decision, match_id: matchID, reason: reason || undefined }]));
    } catch (cause) {
      const conflict = apiErrorKind(cause) === "conflict";
      setError(conflict ? "Programs or this assessment changed. The latest comparison has been loaded." : cause instanceof Error ? cause.message : "The coverage review could not be recorded.");
      if (conflict) await refresh(selected.id);
    } finally {
      setCoverageActionID(null);
    }
  }

  async function applySuggestion(suggestion: CoverageSuggestion) {
    if (!selected || !coverage || coverageActionID) return;
    setCoverageActionID(suggestion.id);
    setCoverageNotice(null);
    setError(null);
    try {
      const result = await applyDocumentCoverageSuggestion(selected.id, suggestion.id, coverage.version);
      setCoverage({ ...result.assessment, matters: result.assessment.matters ?? coverage.matters });
      setCoverageNotice(actionReceipt(result.object_type));
    } catch (cause) {
      const conflict = apiErrorKind(cause) === "conflict";
      setError(conflict ? "Programs or this assessment changed. The latest comparison has been loaded." : cause instanceof Error ? cause.message : "The recommended update could not be applied.");
      if (conflict) await refresh(selected.id);
    } finally {
      setCoverageActionID(null);
    }
  }

  async function recompare() {
    if (!selected || coverageActionID) return;
    setCoverageActionID("recompare");
    setError(null);
    try {
      await recompareDocumentCoverage(selected.id);
      setCoverage((current) => current ? { ...current, status: "COMPARING" } : current);
      setCoverageNotice("A fresh comparison has been queued.");
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "The comparison could not be queued.");
    } finally {
      setCoverageActionID(null);
    }
  }

  async function loadMoreCoverage() {
    if (!selected || !coverage?.next_cursor || coverageActionID) return;
    setCoverageActionID("load-more");
    try {
      const next = await loadDocumentCoverage(selected.id, coverage.next_cursor);
      setCoverage({ ...next, candidates: [...coverage.candidates, ...next.candidates], suggestions: [...coverage.suggestions, ...next.suggestions], matters: [...coverage.matters, ...next.matters] });
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "More obligations could not be loaded.");
    } finally {
      setCoverageActionID(null);
    }
  }

  return <section className="document-import-workspace" aria-labelledby="document-import-title">
    <div className={`document-import-header${intakeOpen ? " intake-open" : ""}`}>
      <div>
        <span className="eyebrow">Document import</span>
        <h2 id="document-import-title">Imported documents</h2>
        <p>Review imported documents and extracted obligations.</p>
        {!intakeOpen && <button className="primary-button document-import-open" type="button" onClick={() => setIntakeOpen(true)}>Import document</button>}
      </div>
      {intakeOpen && <form className="document-import-form" onSubmit={submit} noValidate>
        <div className="document-import-form-heading"><div><strong>Import a document</strong><span>TXT, Markdown, CSV, DOCX, XLSX or PDF · maximum 20 MB</span></div>{documents.length > 0 && <button type="button" aria-label="Close import form" onClick={() => setIntakeOpen(false)}>Close</button>}</div>
        <FileDropzone
          label="Document"
          description="Drop a supported file here or choose one from this device. Searchable PDFs are extracted automatically with page references. Scanned PDFs remain stored and clearly report when OCR is required."
          accept={documentAccept}
          disabled={uploading}
          busy={uploading}
          actionLabel="Choose document"
          fileName={pendingFile?.name}
          fileSize={pendingFile?.size}
          onSelect={(file) => {
            if (file.size > maximumDocumentBytes) {
              setPendingFile(null);
              setError("Choose a document no larger than 20 MB.");
              return;
            }
            setPendingFile(file);
            setError(null);
          }}
        />
        <label><span>What should reviewers look for?</span><input value={purpose} onChange={(event) => setPurpose(event.target.value)} placeholder="e.g. Changes affecting card operations" aria-required="true" disabled={uploading}/></label>
        <label><span>Document type</span><select value={sourceType} onChange={(event) => setSourceType(event.target.value)} disabled={uploading}><option value="DOCUMENT">Internal document</option><option value="REGULATORY">Regulatory source</option><option value="POLICY">Policy or standard</option><option value="FINDING">Finding or report</option></select></label>
        <button className="primary-button" type="submit" disabled={uploading}>{uploading ? "Storing…" : "Import document"}</button>
      </form>}
    </div>
    {error && <p className="error-text" role="alert">{error}</p>}
    {state === "loading"
      ? <div className="workspace-loading" aria-live="polite">Loading imported documents…</div>
      : state === "unavailable"
        ? <EmptyState kind="unavailable" label="Document imports" title="Imported documents could not be loaded" description="Try again before relying on this list." action="Try again" onAction={() => void refresh()}/>
        : !documents.length
          ? <EmptyState label="Document imports" title="No documents imported" description="Import a TXT, Markdown, CSV, DOCX, XLSX or PDF document. Searchable PDFs are extracted automatically; scanned PDFs remain stored and report when OCR is required."/>
          : <div className="document-import-layout">
              <div className="document-import-list" aria-label="Imported documents">{documents.map((document) => <button key={document.id} type="button" disabled={Boolean(reviewingProposalID)} className={selected?.id === document.id ? "document-import-row active" : "document-import-row"} onClick={() => void choose(document.id)}><strong>{document.file_name}</strong><span>{summaryLabel(document)}</span><small>{new Date(document.created_at).toLocaleString()}</small></button>)}</div>
              {selected && <DocumentInspector document={selected} coverage={coverage} coverageActionID={coverageActionID} coverageNotice={coverageNotice} reviewingProposalID={reviewingProposalID} onReview={review} onCoverageReview={reviewCoverage} onApplySuggestion={applySuggestion} onRecompare={recompare} onLoadMore={loadMoreCoverage}/>}
            </div>}
  </section>;
}

function DocumentInspector({ document, coverage, coverageActionID, coverageNotice, reviewingProposalID, onReview, onCoverageReview, onApplySuggestion, onRecompare, onLoadMore }: {
  document: DocumentImport;
  coverage: DocumentCoverage | null;
  coverageActionID: string | null;
  coverageNotice: string | null;
  reviewingProposalID: string | null;
  onReview: (proposal: DocumentProposal, status: ProposalStatus) => void;
  onCoverageReview: (candidate: CoverageCandidate, decision: CoverageDecision, reason?: string) => void;
  onApplySuggestion: (suggestion: CoverageSuggestion) => void;
  onRecompare: () => void;
  onLoadMore: () => void;
}) {
  const pending = document.proposals.filter((proposal) => proposal.status === "PENDING_REVIEW");
  const reviewed = document.proposals.filter((proposal) => proposal.status !== "PENDING_REVIEW");
  const handoffsPending = document.proposals.filter((proposal) => proposal.handoff?.status === "AWAITING_REVIEW" || proposal.handoff?.status === "AWAITING_AUTHORIZATION").length;
  const processing = isProcessing(document);
  const sectionsTotal = document.sections_total ?? document.sections.length;
  const sectionsOmitted = document.sections_omitted ?? 0;
  const contentTruncated = document.content_truncated ?? false;
  const storedOnly = document.extraction_status === "UNSUPPORTED";
  const stateLabel = storedOnly ? "Original stored" : human(document.extraction_status);
  const coveragePending = coverage?.candidates.filter((candidate) => candidate.eligible && !candidate.review).length ?? 0;
  const coverageTotal = coverage?.metrics.verified.denominator ?? 0;
  const terminalLabel = document.extraction_status === "FAILED" ? "Extraction failed" : storedOnly ? "Text review unavailable" : handoffsPending ? `${handoffsPending} governed approval${handoffsPending === 1 ? "" : "s"} pending` : coverage ? `${coverageTotal} eligible obligation${coverageTotal === 1 ? "" : "s"}` : pending.length ? `${pending.length} to review` : "No review pending";

  return <article className="document-import-inspector">
    <header><div><span className="eyebrow">{human(document.source_type)}</span><h2>{document.file_name}</h2><p>{document.purpose}</p></div><div className="document-state"><span>{stateLabel}</span><strong>{processing ? "Processing stored source" : terminalLabel}</strong></div></header>
    <div className="import-review-summary" aria-label="Import review summary">{coverage ? <><span><strong>{coverage.metrics.verified.denominator}</strong> eligible obligations</span><span><strong>{coveragePending}</strong> loaded for review</span><span><strong>{handoffsPending}</strong> governed handoffs pending</span></> : <><span><strong>{pending.length}</strong> intake review</span><span><strong>{handoffsPending}</strong> governed handoffs pending</span><span><strong>{document.sections.length}</strong> of {sectionsTotal} source sections retained</span></>}</div>
    {processing && <section className="workspace-loading" aria-live="polite" aria-busy="true"><strong>Original stored successfully.</strong><p>Extraction and analysis are running in the background. This page will update when processing completes.</p></section>}
    {!processing && coverage && <CoverageAssessment coverage={coverage} actionID={coverageActionID} notice={coverageNotice} onReview={onCoverageReview} onApply={onApplySuggestion} onRecompare={onRecompare} onLoadMore={onLoadMore}/>}
    {document.limitations.length > 0 && <section className="document-limitations"><h3>Important limitations</h3>{document.limitations.map((item) => <p key={item}>{item}</p>)}</section>}
    {!processing && <>
      {!coverage && <section><div className="section-header"><div><h3>Review required</h3><p>Accepting a proposal starts independent governed review. It does not create or activate a Requirement or Control.</p></div></div>{pending.length ? <div className="proposal-list">{pending.map((proposal) => <ProposalCard key={proposal.id} document={document} proposal={proposal} busy={reviewingProposalID === proposal.id} locked={Boolean(reviewingProposalID)} onReview={onReview}/>)}</div> : <div className="calm-empty compact"><span>✓</span><div><strong>Nothing waiting for intake review</strong><p>{document.analysis_status === "UNAVAILABLE" ? "No review proposal is available from this source." : "Accepted proposals continue through their governed handoff below."}</p></div></div>}</section>}
      {coverage && pending.length > 0 && <details className="import-secondary"><summary><span>Extraction proposals</span><strong>{pending.length} unreviewed</strong></summary><div className="proposal-list">{pending.map((proposal) => <ProposalCard key={proposal.id} document={document} proposal={proposal} busy={reviewingProposalID === proposal.id} locked={Boolean(reviewingProposalID)} onReview={onReview}/>)}</div></details>}
      {reviewed.length > 0 && <details className="import-secondary" open={handoffsPending > 0}><summary><span>Proposal outcomes & handoffs</span><strong>{reviewed.length}</strong></summary><div className="proposal-list">{reviewed.map((proposal) => <ProposalCard key={proposal.id} document={document} proposal={proposal} busy={false} locked={Boolean(reviewingProposalID)} onReview={onReview}/>)}</div></details>}
    </>}
    <details className="import-secondary"><summary><span>Original source details</span><strong>{document.sections.length} extracted</strong></summary><div><dl className="document-metadata"><div><dt>Original hash</dt><dd><code>{document.sha256}</code></dd></div><div><dt>File status</dt><dd>{human(document.artifact_status)}</dd></div><div><dt>Text extraction</dt><dd>{human(document.extraction_method)}</dd></div><div><dt>Completeness</dt><dd>{contentTruncated || sectionsOmitted ? `${document.sections.length} of ${sectionsTotal} sections extracted` : `All ${sectionsTotal} sections extracted`}</dd></div><div><dt>Version</dt><dd>{document.version}</dd></div></dl>{document.sections.length > 0 && <div className="document-sections">{document.sections.map((section) => <details key={section.id}><summary>{section.title}</summary><pre>{section.text}</pre></details>)}</div>}</div></details>
  </article>;
}

type CoverageFilter = "ALL" | "REVIEW" | "GAPS" | "COVERED";

function CoverageAssessment({ coverage, actionID, notice, onReview, onApply, onRecompare, onLoadMore }: {
  coverage: DocumentCoverage;
  actionID: string | null;
  notice: string | null;
  onReview: (candidate: CoverageCandidate, decision: CoverageDecision, reason?: string) => void;
  onApply: (suggestion: CoverageSuggestion) => void;
  onRecompare: () => void;
  onLoadMore: () => void;
}) {
  const [filter, setFilter] = useState<CoverageFilter>("ALL");
  const denominator = coverage.metrics.verified.denominator;
  const verified = coverage.metrics.verified.numerator;
  const estimated = coverage.metrics.estimated_verified.numerator;
  const candidates = coverage.candidates.filter((candidate) => candidate.eligible).filter((candidate) => {
    if (filter === "REVIEW") return !candidate.review;
    if (filter === "GAPS") return candidate.classification === "GAP" || candidate.classification === "MAPPED_CONTROL_GAP" || candidate.classification === "MAPPED_NO_CURRENT_EVIDENCE";
    if (filter === "COVERED") return candidate.classification === "VERIFIED_COVERAGE";
    return true;
  });
  if (coverage.status === "PENDING" || coverage.status === "COMPARING") {
    return <section className="coverage-assessment workspace-loading" aria-live="polite" aria-busy="true"><strong>Comparing extracted obligations with current Programs…</strong><p>Checking requirement matches, controls, evidence and related issues.</p></section>;
  }
  if (coverage.status === "FAILED") {
    return <section className="coverage-assessment coverage-callout danger"><div><strong>Coverage comparison needs attention</strong><p>{coverage.failure_message || "The source was retained, but the comparison did not complete."}</p></div><button className="secondary-button" type="button" onClick={onRecompare}>Try comparison again</button></section>;
  }
  return <section className="coverage-assessment" aria-labelledby="coverage-assessment-title">
    <div className="coverage-heading">
      <div><span className="eyebrow">Program comparison</span><h3 id="coverage-assessment-title">Coverage assessment</h3><p>Verified coverage requires an accepted requirement match, an implemented control, and current supporting evidence.</p></div>
      {coverage.status === "STALE" && <button className="primary-button" type="button" disabled={Boolean(actionID)} onClick={onRecompare}>Compare again</button>}
    </div>
    {coverage.status === "STALE" && <div className="coverage-callout warning"><strong>Programs changed after this comparison</strong><p>Compare again before reviewing matches or applying updates.</p></div>}
    {notice && <p className="coverage-notice" role="status">{notice}</p>}
    <div className="coverage-scoreboard" aria-label="Document coverage summary">
      <div className="coverage-primary"><span>Verified document coverage</span><strong className="coverage-primary-value">{percentage(verified, denominator)}</strong><small>{verified} of {denominator} obligations verified after review</small></div>
      <div><span>Estimated</span><strong>{percentage(estimated, denominator)}</strong><small>{estimated} of {denominator} estimated before review</small></div>
      <div><span>Mapped to requirements</span><strong>{percentage(coverage.metrics.requirement_mapped.numerator, denominator)}</strong><small>{coverage.metrics.requirement_mapped.numerator} of {denominator}</small></div>
      <div><span>Controls implemented</span><strong>{percentage(coverage.metrics.control_implemented.numerator, denominator)}</strong><small>{coverage.metrics.control_implemented.numerator} of {denominator}</small></div>
      <div><span>Evidence current</span><strong>{percentage(coverage.metrics.evidence_supported.numerator, denominator)}</strong><small>{coverage.metrics.evidence_supported.numerator} of {denominator}</small></div>
    </div>
    <div className="coverage-toolbar">
      <div><h4>Extracted obligations</h4><span>{denominator} count toward this assessment</span></div>
      <div className="coverage-filters" aria-label="Filter obligations">{(["ALL", "REVIEW", "GAPS", "COVERED"] as CoverageFilter[]).map((value) => <button key={value} type="button" aria-pressed={filter === value} onClick={() => setFilter(value)}>{filterLabel(value)}</button>)}</div>
    </div>
    {candidates.length ? <div className="coverage-candidates">{candidates.map((candidate) => <CoverageCandidateCard key={candidate.id} candidate={candidate} suggestion={coverage.suggestions.find((item) => item.candidate_id === candidate.id)} matters={coverage.matters.filter((item) => item.candidate_id === candidate.id)} busy={actionID === candidate.id || actionID === coverage.suggestions.find((item) => item.candidate_id === candidate.id)?.id} locked={Boolean(actionID) || coverage.status === "STALE"} onReview={onReview} onApply={onApply}/>)}</div> : <div className="calm-empty compact"><span>✓</span><div><strong>No obligations in this view</strong><p>Choose another filter to see the remaining extracted obligations.</p></div></div>}
    {coverage.next_cursor && <button className="secondary-button coverage-load-more" type="button" disabled={Boolean(actionID)} onClick={onLoadMore}>{actionID === "load-more" ? "Loading…" : "Load more obligations"}</button>}
    <p className="coverage-disclaimer">Percentages describe coverage against the extracted eligible obligations in this document. They are not a legal opinion or a claim of overall regulatory compliance.</p>
  </section>;
}

function CoverageCandidateCard({ candidate, suggestion, matters, busy, locked, onReview, onApply }: {
  candidate: CoverageCandidate;
  suggestion?: CoverageSuggestion;
  matters: DocumentCoverage["matters"];
  busy: boolean;
  locked: boolean;
  onReview: (candidate: CoverageCandidate, decision: CoverageDecision, reason?: string) => void;
  onApply: (suggestion: CoverageSuggestion) => void;
}) {
  const [notApplicableOpen, setNotApplicableOpen] = useState(false);
  const [reason, setReason] = useState("");
  const match = candidate.matches[0];
  return <article className={`coverage-candidate classification-${candidate.classification.toLowerCase().replaceAll("_", "-")}`} aria-busy={busy || undefined}>
    <div className="coverage-candidate-heading"><div><span>{sourceLabel(candidate)}</span><h5>{candidate.statement}</h5></div><mark>{classificationLabel(candidate.classification)}</mark></div>
    <blockquote>{candidate.anchor.quote}</blockquote>
    {match ? <div className="coverage-match">
      <div className="coverage-match-heading"><div><span>Best existing match · {Math.round(match.score * 100)}%</span><strong>{match.program_name}</strong><small>{match.program_code} / {match.requirement_code} · {match.requirement_title}</small></div><CoverageChain match={match}/></div>
      <p>{match.rationale}</p>
      <details><summary>Why this match</summary><div>{match.components.map((component) => <p key={component.name}><strong>{human(component.name)}</strong><span>{Math.round(component.score * 100)}%</span><small>{component.reason}</small></p>)}</div></details>
    </div> : <div className="coverage-no-match"><strong>No reliable requirement match</strong><p>Link this obligation to an existing requirement or create a draft Program.</p></div>}
    {matters.map((matter) => <div className="coverage-matter" key={matter.matter_id}><span>Related issue</span><strong>{matter.reference} · {matter.title}</strong><small>{human(matter.status)}</small></div>)}
    {candidate.review ? <p className="coverage-reviewed">Reviewed: {human(candidate.review.decision)}{candidate.review.reason ? ` · ${candidate.review.reason}` : ""}</p> : <div className="coverage-actions">
      {match && <button className="primary-button" type="button" disabled={locked} onClick={() => onReview(candidate, "ACCEPT_MATCH")}>{busy ? "Recording…" : "Confirm match"}</button>}
      {match && <button className="secondary-button" type="button" disabled={locked} onClick={() => onReview(candidate, "REJECT_MATCH")}>No valid match</button>}
      <button className="text-button" type="button" disabled={locked} onClick={() => setNotApplicableOpen((value) => !value)}>Not applicable</button>
    </div>}
    {notApplicableOpen && !candidate.review && <div className="coverage-not-applicable"><label><span>Why is this obligation out of scope?</span><textarea value={reason} onChange={(event) => setReason(event.target.value)} rows={2}/></label><button className="secondary-button" type="button" disabled={locked || !reason.trim()} onClick={() => onReview(candidate, "NOT_APPLICABLE", reason.trim())}>Record as not applicable</button></div>}
    {suggestion && <div className="coverage-recommendation"><div><span>Recommended next step</span><strong>{suggestion.title}</strong><p>{suggestion.rationale}</p></div>{suggestion.status === "PROPOSED" ? <button className="secondary-button" type="button" disabled={locked} onClick={() => onApply(suggestion)}>{busy ? "Applying…" : suggestionButtonLabel(suggestion.type)}</button> : <small>{suggestion.status === "APPLIED" ? `${human(suggestion.applied_type || "Update")} created` : human(suggestion.status)}</small>}</div>}
  </article>;
}

function CoverageChain({ match }: { match: NonNullable<CoverageCandidate["matches"][number]> }) {
  return <div className="coverage-chain" aria-label="Coverage chain"><span className="active">Requirement</span><span className={match.coverage.control_implemented ? "active" : ""}>Control</span><span className={match.coverage.evidence_supported ? "active" : ""}>Evidence</span></div>;
}

function ProposalCard({ document, proposal, busy, locked, onReview }: { document: DocumentImport; proposal: DocumentProposal; busy: boolean; locked: boolean; onReview: (proposal: DocumentProposal, status: ProposalStatus) => void }) {
  const state = proposal.handoff?.status ? human(proposal.handoff.status) : human(proposal.status);
  return <article id={`document-proposal-${proposal.id}`} className="proposal-card" aria-busy={busy || undefined}><div className="proposal-title"><div><span>{human(proposal.kind)}</span><h4>{proposal.title}</h4></div><mark>{state}</mark></div><p>{proposal.statement}</p><blockquote>{proposal.anchor.quote}</blockquote><small>Source: {proposal.anchor.sheet ? `${proposal.anchor.sheet}, row ${proposal.anchor.row_start}` : proposal.anchor.page ? `page ${proposal.anchor.page}` : proposal.anchor.section_id}</small>{proposal.status === "PENDING_REVIEW" && <div className="proposal-actions"><button className="secondary-button" type="button" disabled={locked} onClick={() => onReview(proposal, "REJECTED")}>{busy ? "Recording…" : "Reject"}</button><button className="primary-button" type="button" disabled={locked} onClick={() => onReview(proposal, "ACCEPTED")}>{busy ? "Recording…" : "Accept for governed review"}</button></div>}{proposal.status === "ACCEPTED" && <DocumentProposalHandoff documentID={document.id} documentVersion={document.version} legalEntityID={document.legal_entity_id} proposal={proposal} locked={locked}/>}</article>;
}

function focusedImportTarget(): ImportFocus {
  const parts = window.location.hash.replace(/^#\/?/, "").split("/").filter(Boolean);
  if (parts[0] !== "imports") return {};
  return { documentID: decodeRoutePart(parts[1]), proposalID: decodeRoutePart(parts[2]) };
}

function decodeRoutePart(value?: string) {
  if (!value) return undefined;
  try {
    return decodeURIComponent(value);
  } catch {
    return undefined;
  }
}

function isProcessing(document: Pick<DocumentImport, "extraction_status" | "analysis_status">) {
  return document.extraction_status === "PENDING" || document.analysis_status === "PENDING";
}

function summaryLabel(document: DocumentImportSummary) {
  if (document.extraction_status === "PENDING" || document.analysis_status === "PENDING") return "Stored · processing";
  if (document.extraction_status === "FAILED") return "Extraction failed · original retained";
  if (document.extraction_status === "UNSUPPORTED") return "Stored · extraction unavailable";
  if (document.pending_proposal_count) return `${document.pending_proposal_count} proposal${document.pending_proposal_count === 1 ? "" : "s"} to review`;
  return "No intake proposals waiting";
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

function percentage(numerator: number, denominator: number) {
  return denominator > 0 ? `${Math.round((numerator / denominator) * 100)}%` : "—";
}

function filterLabel(value: CoverageFilter) {
  if (value === "REVIEW") return "Needs review";
  if (value === "GAPS") return "Gaps";
  if (value === "COVERED") return "Covered";
  return "All";
}

function sourceLabel(candidate: CoverageCandidate) {
  if (candidate.anchor.sheet) return `${candidate.anchor.sheet} · row ${candidate.anchor.row_start ?? "—"}`;
  if (candidate.anchor.page) return `Source · page ${candidate.anchor.page}`;
  return `Source · ${candidate.anchor.section_id}`;
}

function classificationLabel(value: CoverageCandidate["classification"]) {
  const labels: Record<CoverageCandidate["classification"], string> = {
    VERIFIED_COVERAGE: "Verified coverage",
    MAPPED_NO_CURRENT_EVIDENCE: "Evidence gap",
    MAPPED_CONTROL_GAP: "Control gap",
    PARTIAL_MATCH: "Possible match",
    GAP: "Coverage gap",
    NEEDS_REVIEW: "Needs review",
    NOT_APPLICABLE: "Not applicable",
  };
  return labels[value];
}

function suggestionButtonLabel(value: CoverageSuggestion["type"]) {
  if (value === "CREATE_PROGRAM") return "Create draft Program";
  if (value === "ADD_REQUIREMENT") return "Add draft requirement";
  if (value === "CREATE_MATTER") return "Create issue";
  return "Link requirement";
}

function actionReceipt(objectType?: string) {
  if (objectType === "PROGRAM") return "Draft Program created. It must follow the normal approval lifecycle.";
  if (objectType === "REQUIREMENT") return "Draft requirement added. It must be reviewed and approved before becoming effective.";
  if (objectType === "MATTER") return "Issue created and linked to this source-backed obligation.";
  return "The recommendation was applied.";
}