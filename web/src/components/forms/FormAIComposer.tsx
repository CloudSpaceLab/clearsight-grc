import { useState } from "react";
import type { DocumentImport } from "../../documentTypes";
import { createAIFormProposal, createAIFormRevisionProposal } from "../../formsApi";
import type { FormTemplateProposal, RequestAIFormProposalInput } from "../../formsTypes";
import { apiErrorKind } from "../../http";

type BaseTemplate = { id: string; name: string; version: number };
type Props = { baseTemplate?: BaseTemplate; sourceDocument?: DocumentImport; onProposal: (proposal: FormTemplateProposal) => void; onCancel?: () => void };

export function FormAIComposer({ baseTemplate, sourceDocument, onProposal, onCancel }: Props) {
  const [objective, setObjective] = useState("");
  const [selectedRefs, setSelectedRefs] = useState<Set<string>>(new Set());
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const elements = (sourceDocument?.elements ?? []).filter((element) => element.ref && (element.text || element.control?.label)).slice(0, 100);

  async function submit() {
    const value = objective.trim();
    if (!value) { setError("Describe the form to create or the specific changes reviewers need."); return; }
    setBusy(true);
    setError(null);
    const input: RequestAIFormProposalInput = { objective: value };
    if (sourceDocument && selectedRefs.size > 0) {
      input.source_document_id = sourceDocument.id;
      input.expected_source_document_version = sourceDocument.version;
      input.source_element_refs = [...selectedRefs];
    }
    try {
      const proposal = baseTemplate
        ? await createAIFormRevisionProposal(baseTemplate.id, baseTemplate.version, input)
        : await createAIFormProposal(input);
      onProposal(proposal);
    } catch (cause) {
      setError(apiErrorKind(cause) === "unavailable"
        ? "Governed AI authoring is unavailable. The manual form builder remains available and no draft was changed."
        : cause instanceof Error ? cause.message : "A reviewable field proposal could not be generated.");
    } finally {
      setBusy(false);
    }
  }

  return <section className="form-ai-composer" aria-labelledby="form-ai-composer-title">
    <header><div><span className="eyebrow">Optional drafting support</span><h3 id="form-ai-composer-title">{baseTemplate ? `Propose changes to ${baseTemplate.name}` : "Draft form fields with governed AI"}</h3><p>AI can propose fields, but it cannot publish a template or infer compliance weights. You choose every change before a draft is created.</p></div>{onCancel && <button type="button" className="text-button" onClick={onCancel}>Close</button>}</header>
    {error && <p className="error-text" role="alert">{error}</p>}
    <label className="form-ai-objective"><span>What should this form collect or change?</span><textarea rows={3} value={objective} onChange={(event) => setObjective(event.target.value)} placeholder="e.g. Collect current vendor ownership, operating certificate and expiry details."/></label>
    {elements.length > 0 && <details className="form-ai-sources"><summary>Choose source passages <span>{selectedRefs.size ? `${selectedRefs.size} selected` : "optional"}</span></summary><div>{elements.map((element) => <label key={element.ref}><input type="checkbox" checked={selectedRefs.has(element.ref!)} onChange={() => setSelectedRefs((current) => { const next = new Set(current); if (next.has(element.ref!)) next.delete(element.ref!); else next.add(element.ref!); return next; })}/><span><strong>{element.control?.label || element.text}</strong><small>{sourceLocation(element.anchor)}</small></span></label>)}</div></details>}
    <div className="form-ai-actions">{onCancel && <button className="secondary-button" type="button" disabled={busy} onClick={onCancel}>Use manual builder</button>}<button className="primary-button" type="button" disabled={busy} onClick={() => void submit()}>{busy ? "Preparing proposal…" : baseTemplate ? `Propose changes to revision ${baseTemplate.version}` : "Generate field proposal"}</button></div>
  </section>;
}

function sourceLocation(anchor: NonNullable<DocumentImport["elements"]>[number]["anchor"]) {
  if (anchor.sheet) return `${anchor.sheet}${anchor.row_start ? ` · row ${anchor.row_start}` : ""}`;
  if (anchor.page) return `Page ${anchor.page}`;
  return anchor.paragraph || anchor.cell || "Stored source passage";
}
