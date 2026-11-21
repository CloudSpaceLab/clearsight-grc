import { useEffect, useRef, useState } from "react";
import { loadDocumentImport, loadDocumentImports } from "../documentApi";
import type { DocumentImport, DocumentImportSummary } from "../documentTypes";
import "./document-imports.css";

const extractionLabel = (value: string) => ({ PENDING: "Processing", EXTRACTED: "Extracted", PARTIAL: "Partially extracted", TRUNCATED: "Extraction truncated", UNSUPPORTED: "Format unsupported", FAILED: "Extraction failed" } as Record<string, string>)[value] ?? "Status unavailable";

/** Not routed: the import API has tenant/entity scoping but no per-person read authorization. */
export function DocumentImports() {
  const [items, setItems] = useState<DocumentImportSummary[]>([]);
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");
  const [attempt, setAttempt] = useState(0);
  const [selected, setSelected] = useState<DocumentImport>();
  const [detailState, setDetailState] = useState<"idle" | "loading" | "ready" | "error">("idle");
  const generation = useRef(0);
  const heading = useRef<HTMLHeadingElement>(null);
  const trigger = useRef<HTMLButtonElement>(null);
  useEffect(() => {
    let active = true; setStatus("loading"); setItems([]);
    loadDocumentImports().then((result) => { if (active) { setItems(result); setStatus("ready"); } }).catch(() => { if (active) setStatus("error"); });
    return () => { active = false; generation.current++; };
  }, [attempt]);
  useEffect(() => { if (detailState === "ready") heading.current?.focus(); }, [detailState, selected]);
  async function open(id: string, button: HTMLButtonElement) {
    const token = ++generation.current; trigger.current = button; setSelected(undefined); setDetailState("loading");
    try { const result = await loadDocumentImport(id); if (generation.current === token) { if (result.id !== id) throw new Error("mismatched document"); setSelected(result); setDetailState("ready"); } }
    catch { if (generation.current === token) setDetailState("error"); }
  }
  function close() { generation.current++; setSelected(undefined); setDetailState("idle"); if (trigger.current?.isConnected) trigger.current.focus(); }
  return <div className="v-imported-documents">
    {status === "loading" && <p role="status">Loading imported documents…</p>}
    {status === "error" && <div className="v-panel"><p role="alert">Imported documents unavailable.</p><button type="button" onClick={() => setAttempt((value) => value + 1)}>Try again</button></div>}
    {status === "ready" && <div className={`v-imported-document-layout${selected ? " has-selection" : ""}`}><section className="v-panel" aria-label="Imported documents"><h2>Imported documents</h2>
      {items.length === 0 ? <p>No imported documents in the current scope.</p> : <ul className="v-attention">{items.map((item) => <li key={item.id}><div><strong>{item.file_name}</strong><small>{extractionLabel(item.extraction_status)}</small></div><button type="button" aria-label={`Open ${item.file_name}`} onClick={(event) => void open(item.id, event.currentTarget)}>Details</button></li>)}</ul>}
      {items.length >= 50 && <p>Showing the first 50 imported documents.</p>}
    </section>
    {detailState === "loading" && <p role="status">Loading document…</p>}
    {detailState === "error" && <div className="v-panel"><p role="alert">Document unavailable. Check access and try again.</p><button type="button" onClick={close}>Close</button></div>}
    {selected && <aside className="v-panel v-imported-document-detail" aria-label="Imported document details"><div className="v-imported-document-heading"><h2 ref={heading} tabIndex={-1}>{selected.file_name}</h2><button type="button" aria-label="Close document details" onClick={close}>×</button></div><p>{extractionLabel(selected.extraction_status)}</p>
      {selected.limitations.map((limitation, index) => <p key={index}>{limitation}</p>)}
      {selected.sections_omitted ? <p>{selected.sections_omitted} {selected.sections_omitted === 1 ? "section omitted." : "sections omitted."}</p> : null}
      {selected.sections.length ? <section aria-label="Extracted text"><h3>Extracted text</h3>{selected.sections.map((section) => <article key={section.id}><h4>{section.title}</h4><p>{section.text}</p></article>)}</section> : <p>No extracted text available.</p>}
    </aside>}
    </div>}
  </div>;
}
