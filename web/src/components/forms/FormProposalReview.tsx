import { useEffect, useMemo, useState } from "react";
import type { DocumentExtractedElement, DocumentSourceAnchor } from "../../documentTypes";
import { acceptFormProposal, loadFormProposal, rejectFormProposal } from "../../formsApi";
import type { FormProposalFieldChange, FormTemplateProposal } from "../../formsTypes";
import { apiErrorKind } from "../../http";
import type { CaptureFormContract } from "../../types";
import { FormPreview } from "./FormPreview";

type Props = {
  proposal: FormTemplateProposal;
  sourceTitle?: string;
  sourceElements?: DocumentExtractedElement[];
  onProposalChange: (proposal: FormTemplateProposal) => void;
  onDraftCreated?: (templateID: string, version: number) => void;
};

export function FormProposalReview({ proposal, sourceTitle, sourceElements = [], onProposalChange, onDraftCreated }: Props) {
  const [selected, setSelected] = useState<Set<string>>(() => new Set(proposal.field_changes.map((change) => change.id)));
  const [busy, setBusy] = useState<"accept" | "reject" | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const available = new Set(proposal.field_changes.map((change) => change.id));
    setSelected((current) => new Set([...current].filter((id) => available.has(id))));
  }, [proposal.id, proposal.version, proposal.field_changes]);

  useEffect(() => {
    if (proposal.status !== "GENERATING") return;
    let active = true;
    const poll = async () => {
      try {
        const latest = await loadFormProposal(proposal.id);
        if (active) onProposalChange(latest);
      } catch {
        // Keep the durable generation receipt visible; the next poll may recover.
      }
    };
    const timer = window.setInterval(() => void poll(), 1500);
    return () => { active = false; window.clearInterval(timer); };
  }, [proposal.id, proposal.status, onProposalChange]);

  const selectedChanges = proposal.field_changes.filter((change) => selected.has(change.id));
  const preview = useMemo(() => previewContract(proposal, selected), [proposal, selected]);
  const allSelected = proposal.field_changes.length > 0 && selected.size === proposal.field_changes.length;

  function toggle(id: string) {
    setSelected((current) => {
      const next = new Set(current);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });
  }

  async function accept() {
    if (busy || selectedChanges.length === 0) return;
    setBusy("accept");
    setError(null);
    try {
      const accepted = await acceptFormProposal(proposal.id, proposal.version, selectedChanges.map((change) => change.id));
      onProposalChange(accepted);
      if (accepted.result_template_id && accepted.result_template_version) onDraftCreated?.(accepted.result_template_id, accepted.result_template_version);
    } catch (cause) {
      if (apiErrorKind(cause) === "conflict") {
        setError("This proposal changed while you were reviewing it. The latest version has been loaded; review the selected fields again.");
        try { onProposalChange(await loadFormProposal(proposal.id)); } catch { /* Preserve the conflict and current receipt. */ }
      } else {
        setError(cause instanceof Error ? cause.message : "The selected fields could not be turned into a draft.");
      }
    } finally {
      setBusy(null);
    }
  }

  async function reject() {
    if (busy) return;
    setBusy("reject");
    setError(null);
    try {
      onProposalChange(await rejectFormProposal(proposal.id, proposal.version));
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "The proposal could not be rejected.");
    } finally {
      setBusy(null);
    }
  }

  if (proposal.status === "GENERATING") return <section className="form-proposal-state workspace-loading" aria-live="polite" aria-busy="true"><strong>Building a source-backed field proposal…</strong><p>The imported file and its exact version remain unchanged while candidate fields are prepared.</p></section>;
  if (proposal.status === "FAILED") return <section className="form-proposal-state form-proposal-failed" role="alert"><strong>Field proposal needs attention</strong><p>{proposal.failure_message || "No reviewable field proposal was produced."}</p><small>The imported source is retained. You can use the manual form builder or retry after correcting the source.</small></section>;
  if (proposal.status === "ACCEPTED") return <section className="form-proposal-state form-proposal-complete" role="status"><strong>Draft form template created</strong><p>{proposal.accepted_change_ids?.length ?? 0} selected field changes were applied to draft revision {proposal.result_template_version}.</p>{proposal.result_template_id && <a href={`#/forms/${encodeURIComponent(proposal.result_template_id)}`}>Open draft template</a>}</section>;
  if (proposal.status === "REJECTED") return <section className="form-proposal-state"><strong>Proposal rejected</strong><p>No form template was created or changed.</p></section>;

  return <section className="form-proposal-review" aria-labelledby={`form-proposal-${proposal.id}`}>
    <header className="form-proposal-heading">
      <div><span className="eyebrow">{proposal.source_kind === "AI" ? "Governed AI proposal" : "Document field proposal"}</span><h3 id={`form-proposal-${proposal.id}`}>Review proposed form fields</h3><p>{sourceTitle ? `Compare proposed fields with ${sourceTitle} before creating a draft.` : "Choose the exact field changes to include before creating a draft."}</p></div>
      <div className="form-proposal-count"><strong>{selected.size}</strong><span>of {proposal.field_changes.length} selected</span></div>
    </header>
    {error && <p className="error-text" role="alert">{error}</p>}
    {(proposal.provenance.extraction_status === "PARTIAL" || proposal.provenance.extraction_status === "TRUNCATED") && <p className="form-proposal-notice" role="status">Only the retained portion of this source was analyzed. Review source gaps and unresolved items before using the draft.</p>}
    <div className="form-proposal-toolbar">
      <label><input type="checkbox" checked={allSelected} onChange={() => setSelected(allSelected ? new Set() : new Set(proposal.field_changes.map((change) => change.id)))}/> Select all proposed fields</label>
      <span>{proposal.unresolved_items.length} decision{proposal.unresolved_items.length === 1 ? "" : "s"} need author review</span>
    </div>
    <div className="form-proposal-layout">
      <div className="form-proposal-changes" aria-label="Proposed field changes">
        {proposal.field_changes.map((change) => <ProposalChange key={change.id} change={change} checked={selected.has(change.id)} elements={sourceElements} unresolved={proposal.unresolved_items.filter((item) => item.field_change_id === change.id)} onToggle={() => toggle(change.id)}/>)}
      </div>
      <aside className="form-proposal-preview">
        <FormPreview contract={preview} initialMode="CLASSIC" showModeControls={false}/>
        {proposal.proposed_contract.scoring_mode === "NONE" && <p className="form-proposal-scoring-note">Scoring weights were not inferred. Add compliance weights only after a reviewer confirms the scoring policy and the total equals 100.</p>}
      </aside>
    </div>
    <footer className="form-proposal-actions">
      <button className="secondary-button" type="button" disabled={Boolean(busy)} onClick={() => void reject()}>{busy === "reject" ? "Rejecting…" : "Reject proposal"}</button>
      <button className="primary-button" type="button" disabled={Boolean(busy) || selected.size === 0} onClick={() => void accept()}>{busy === "accept" ? "Creating draft…" : "Create draft from selected fields"}</button>
    </footer>
  </section>;
}

function ProposalChange({ change, checked, elements, unresolved, onToggle }: { change: FormProposalFieldChange; checked: boolean; elements: DocumentExtractedElement[]; unresolved: FormTemplateProposal["unresolved_items"]; onToggle: () => void }) {
  const excerpt = sourceExcerpt(change.anchor, elements);
  return <article className={`form-proposal-change${checked ? " selected" : ""}`}>
    <div className="form-proposal-change-heading">
      <label><input type="checkbox" checked={checked} onChange={onToggle} aria-label={`Include ${change.field.label}`}/><span><small>{changeLabel(change.kind)}</small><strong>{change.field.label}</strong></span></label>
      <mark>{Math.round(change.confidence * 100)}% confidence</mark>
    </div>
    <dl><div><dt>Field type</dt><dd>{human(change.field.type)}</dd></div><div><dt>Required</dt><dd>{change.field.required ? "Yes" : "Not set"}</dd></div></dl>
    <div className="form-proposal-source"><span>{anchorLabel(change.anchor)}</span>{excerpt && <blockquote>{excerpt}</blockquote>}</div>
    {unresolved.length > 0 && <ul className="form-proposal-unresolved">{unresolved.map((item, index) => <li key={`${item.code}-${index}`}>{item.message}</li>)}</ul>}
  </article>;
}

function previewContract(proposal: FormTemplateProposal, selected: Set<string>): CaptureFormContract {
  const changes = new Map(proposal.field_changes.map((change) => [change.field.id, change]));
  const fields = proposal.proposed_contract.fields.filter((field) => {
    const change = changes.get(field.id);
    return !change || (selected.has(change.id) && change.kind !== "REMOVE_FIELD");
  });
  const sectionIDs = new Set(fields.map((field) => field.section_id).filter(Boolean));
  return { presentation: proposal.proposed_contract.presentation, sections: proposal.proposed_contract.sections.filter((section) => sectionIDs.has(section.id)), fields };
}

function sourceExcerpt(anchor: DocumentSourceAnchor, elements: DocumentExtractedElement[]) {
  if (!anchor.page && !anchor.sheet && !anchor.paragraph && !anchor.table && !anchor.cell) return undefined;
  return elements.find((element) => element.anchor.page === anchor.page && element.anchor.sheet === anchor.sheet && (element.anchor.paragraph === anchor.paragraph || element.anchor.cell === anchor.cell))?.text;
}

function anchorLabel(anchor: DocumentSourceAnchor) {
  const parts: string[] = [];
  if (anchor.page) parts.push(`Page ${anchor.page}`);
  if (anchor.sheet) parts.push(anchor.sheet);
  if (anchor.row_start) parts.push(anchor.row_end && anchor.row_end !== anchor.row_start ? `rows ${anchor.row_start}–${anchor.row_end}` : `row ${anchor.row_start}`);
  if (anchor.paragraph) parts.push(`paragraph ${anchor.paragraph}`);
  if (anchor.table) parts.push(`table ${anchor.table}`);
  if (anchor.cell) parts.push(`cell ${anchor.cell}`);
  return parts.length ? parts.join(" · ") : "No source location retained";
}

function changeLabel(kind: FormProposalFieldChange["kind"]) {
  if (kind === "UPDATE_FIELD") return "Update field";
  if (kind === "REMOVE_FIELD") return "Remove field";
  return "Add field";
}

function human(value: string) { return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase()); }
