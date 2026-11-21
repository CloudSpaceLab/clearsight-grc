import { useEffect, useRef, useState, type FormEvent } from "react";
import { apiErrorKind, requestBlob } from "../http";
import { documentContentURL, loadDocuments, type DocumentOccurrence, type DocumentQuery, type FileKind } from "../submittedDocumentApi";
import { withDocumentReturn } from "./documentReturn";
import { DocumentProvenance } from "./DocumentProvenance";
import "./documents.css";

const views = [
  { id: "ALL", label: "All documents" },
  { id: "AWAITING_REVIEW", label: "Awaiting assessment review" },
  { id: "NEEDS_REFRESH", label: "Needs refresh" },
  { id: "REVIEW_HISTORY", label: "Assessment review history" },
] as const;
type View = typeof views[number]["id"];
type DocumentRoute = { view: View; search: string; fileKind: FileKind | ""; layout: "list" | "grid"; formId: string };
const searchTooLong = (value: string) => new TextEncoder().encode(value.trim()).length > 200;
const searchLengthError = "Search is too long. Shorten the document name.";

function readDocumentRoute(): DocumentRoute {
  const params = new URLSearchParams(window.location.hash.split("?")[1] ?? "");
  const requestedView = params.get("view");
  const requestedKind = params.get("kind");
  return {
    view: views.some((item) => item.id === requestedView) ? requestedView as View : "ALL",
    search: params.get("q") ?? "",
    fileKind: ["PDF", "IMAGE", "WORD", "SPREADSHEET", "OTHER"].includes(requestedKind ?? "") ? requestedKind as FileKind : "",
    layout: params.get("layout") === "grid" ? "grid" : "list",
    formId: (params.get("form") ?? "").slice(0, 200),
  };
}

function rememberDocumentRoute(change: Partial<DocumentRoute>) {
  const route = { ...readDocumentRoute(), ...change };
  const params = new URLSearchParams();
  if (route.view !== "ALL") params.set("view", route.view);
  if (route.search) params.set("q", route.search);
  if (route.fileKind) params.set("kind", route.fileKind);
  if (route.layout !== "list") params.set("layout", route.layout);
  if (route.formId) params.set("form", route.formId);
  window.history.replaceState(window.history.state, "", `#/documents${params.size ? `?${params}` : ""}`);
}

export function Documents() {
  const [initialRoute] = useState(readDocumentRoute);
  const [view, setView] = useState<View>(initialRoute.view);
  const [layout, setLayout] = useState<"list" | "grid">(initialRoute.layout);
  const [draft, setDraft] = useState(initialRoute.search);
  const [search, setSearch] = useState(initialRoute.search);
  const [searchError, setSearchError] = useState(searchTooLong(initialRoute.search) ? searchLengthError : "");
  const [fileKind, setFileKind] = useState<FileKind | "">(initialRoute.fileKind);
  const [formScope, setFormScope] = useState<{ id: string; name: string } | undefined>(initialRoute.formId ? { id: initialRoute.formId, name: "Selected form" } : undefined);
  const [selected, setSelected] = useState<DocumentOccurrence>();
  const [selectionEpoch, setSelectionEpoch] = useState(0);
  const selectionTrigger = useRef<HTMLButtonElement | null>(null);
  const scopeButton = useRef<HTMLButtonElement | null>(null);
  const focusScope = useRef(false);
  const [items, setItems] = useState<DocumentOccurrence[]>([]);
  const [cursor, setCursor] = useState<string>();
  const [state, setState] = useState<"loading" | "ready" | "error" | "invalid">("loading");
  const [moreBusy, setMoreBusy] = useState(false);
  const [moreError, setMoreError] = useState(false);
  const [downloadBusy, setDownloadBusy] = useState("");
  const [downloadError, setDownloadError] = useState("");
  const [attempt, setAttempt] = useState(0);
  const generation = useRef(0);
  const moreLock = useRef<number | null>(null);
  const query: DocumentQuery = { query: search || undefined, file_kind: fileKind || undefined, form_template_id: formScope?.id, view: view === "ALL" ? undefined : view, current_only: view === "ALL" || view === "REVIEW_HISTORY" ? false : undefined, limit: 25 };

  useEffect(() => { if (focusScope.current && formScope) { focusScope.current = false; scopeButton.current?.focus(); } }, [formScope]);

  useEffect(() => {
    const restore = () => {
      if (!window.location.hash.match(/^#\/?documents(?:\?|$)/)) return;
      const route = readDocumentRoute();
      setView(route.view); setDraft(route.search); setSearch(route.search); setFileKind(route.fileKind); setLayout(route.layout); setFormScope(route.formId ? { id: route.formId, name: "Selected form" } : undefined);
      setSearchError(searchTooLong(route.search) ? searchLengthError : "");
    };
    window.addEventListener("popstate", restore);
    window.addEventListener("hashchange", restore);
    return () => { window.removeEventListener("popstate", restore); window.removeEventListener("hashchange", restore); };
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    const token = ++generation.current;
    moreLock.current = null;
    if (searchTooLong(search)) { setState("invalid"); setItems([]); setSelected(undefined); setCursor(undefined); return () => { controller.abort(); generation.current++; }; }
    setState("loading"); setItems([]); setSelected(undefined); setCursor(undefined); setMoreError(false); setMoreBusy(false);
    loadDocuments(query, controller.signal).then((page) => {
      if (generation.current === token) { setItems(page.items); setCursor(page.next_cursor); setState("ready"); setFormScope((current) => { const match = current && page.items.find((item) => item.form_template_id === current.id && item.form_title); return match ? { ...current!, name: match.form_title } : current; }); }
    }).catch(() => { if (generation.current === token && !controller.signal.aborted) setState("error"); });
    return () => { controller.abort(); generation.current++; };
  }, [view, search, fileKind, formScope?.id, attempt]);

  async function more() {
    if (!cursor || moreBusy || moreLock.current !== null) return;
    const token = generation.current;
    moreLock.current = token; setMoreBusy(true); setMoreError(false);
    try { const page = await loadDocuments({ ...query, cursor }); if (generation.current === token) { if (page.next_cursor === cursor) { setMoreError(true); return; } setItems((previous) => [...previous, ...page.items]); setCursor(page.next_cursor); } }
    catch (error) {
      if (generation.current === token) {
        if (["unauthorized", "forbidden"].includes(apiErrorKind(error))) {
          setItems([]); setSelected(undefined); setCursor(undefined); setState("error");
        } else setMoreError(true);
      }
    }
    finally { if (moreLock.current === token) moreLock.current = null; if (generation.current === token) setMoreBusy(false); }
  }

  async function download(file: DocumentOccurrence) {
    if (downloadBusy) return;
    setDownloadBusy(file.id); setDownloadError("");
    try {
      const result = await requestBlob("", documentContentURL(file, true));
      const url = URL.createObjectURL(result.blob);
      const link = document.createElement("a"); link.href = url; link.download = result.filename || file.file_name; link.click();
      window.setTimeout(() => URL.revokeObjectURL(url), 60_000);
    } catch { setDownloadError(`Could not download ${file.file_name}. Try again.`); }
    finally { setDownloadBusy(""); }
  }

  function submit(event: FormEvent) { event.preventDefault(); const value = draft.trim(); if (searchTooLong(value)) { setSearchError(searchLengthError); return; } setSearchError(""); setSearch(value); rememberDocumentRoute({ search: value }); }
  function openDocument(file: DocumentOccurrence, trigger: HTMLButtonElement) {
    selectionTrigger.current = trigger;
    setSelected(file);
    setSelectionEpoch((value) => value + 1);
  }
  function closeDocument() {
    setSelected(undefined);
    if (selectionTrigger.current?.isConnected) selectionTrigger.current.focus();
    else document.getElementById("v-document-search")?.focus();
  }
  function viewForm(file: DocumentOccurrence) {
    if (!file.form_template_id) return;
    focusScope.current = true;
    setDraft(""); setSearch(""); setFileKind(""); setView("ALL"); rememberDocumentRoute({ search: "", fileKind: "", view: "ALL", formId: file.form_template_id });
    setFormScope({ id: file.form_template_id, name: file.form_title || "Name unavailable" }); setSelected(undefined);
  }
  return <section className="v-documents"><header className="v-page-heading"><div><p className="v-eyebrow">Documents</p><h1>Documents</h1></div></header>
    <nav className="v-local-nav" aria-label="Document views">{views.map((option) => <button type="button" key={option.id} aria-current={view === option.id ? "page" : undefined} onClick={() => { setView(option.id); rememberDocumentRoute({ view: option.id }); }}>{option.label}</button>)}</nav>
    <div className="v-documents-filters"><form className="v-program-search" onSubmit={submit}><label htmlFor="v-document-search">Find a document</label><div><input id="v-document-search" value={draft} maxLength={200} aria-invalid={Boolean(searchError)} aria-describedby={searchError ? "v-document-search-error" : undefined} onChange={(event) => { setDraft(event.target.value); setSearchError(""); }} placeholder="Document name"/><button>Search</button></div></form><label className="v-document-kind">File type<select value={fileKind} onChange={(event) => { const value = event.target.value as FileKind | ""; setFileKind(value); rememberDocumentRoute({ fileKind: value }); }}><option value="">All types</option><option value="PDF">PDF</option><option value="IMAGE">Images</option><option value="WORD">Word documents</option><option value="SPREADSHEET">Spreadsheets</option><option value="OTHER">Other</option></select></label></div>
    {searchError && <p id="v-document-search-error" role="alert">{searchError}</p>}
    {formScope && <div className="v-document-scope"><span>Form: {formScope.name}</span><button ref={scopeButton} type="button" onClick={() => { setFormScope(undefined); rememberDocumentRoute({ formId: "" }); }}>Clear form filter</button></div>}
    {state === "loading" && <p role="status">Loading documents…</p>}
    {state === "error" && <div className="v-panel"><p role="alert">Documents unavailable.</p><button onClick={() => setAttempt(attempt + 1)}>Try again</button></div>}
    {downloadError && <p role="alert">{downloadError}</p>}
    {state === "ready" && <div className={`v-documents-content${selected ? " has-selection" : ""}`}><section className="v-panel v-document-list" aria-label={views.find((item) => item.id === view)?.label}>
      <div className="v-document-toolbar"><h2>{views.find((item) => item.id === view)?.label}</h2><div className="v-document-layout" role="group" aria-label="Document layout"><button type="button" aria-pressed={layout === "list"} onClick={() => { setLayout("list"); rememberDocumentRoute({ layout: "list" }); }}>List view</button><button type="button" aria-pressed={layout === "grid"} onClick={() => { setLayout("grid"); rememberDocumentRoute({ layout: "grid" }); }}>Grid view</button></div></div>
      {items.length === 0 ? <p>No documents match this view.</p> : <ul className={`v-attention v-document-items is-${layout}`}>{items.map((file) => <li key={file.id} className={selected?.id === file.id ? "is-selected" : undefined}><div><span className="v-document-file-kind" aria-hidden="true">{file.file_kind === "PDF" ? "PDF" : file.file_kind === "IMAGE" ? "IMG" : file.file_kind === "WORD" ? "DOC" : file.file_kind === "SPREADSHEET" ? "XLS" : "FILE"}</span><button type="button" className="v-document-name" aria-label={`Open ${file.file_name}`} aria-pressed={selected?.id === file.id} onClick={(event) => openDocument(file, event.currentTarget)}>{file.file_name}</button><small>{file.form_title} · {file.current ? "Latest submission" : "Earlier submission"}</small><small>Submitted {formatDate(file.submitted_at)}{file.expires_on ? ` · Expires ${formatDate(file.expires_on)}` : ""}</small>{file.artifact_status === "STORED_UNSCANNED" && file.demo_unscanned_allowed && <small>Demo file. No antivirus scan was performed.</small>}</div><div className="v-document-row-actions"><button type="button" aria-label={`View details ${file.file_name}`} onClick={(event) => openDocument(file, event.currentTarget)}>Details</button>{canDownload(file) ? <button type="button" disabled={Boolean(downloadBusy)} onClick={() => void download(file)}>{downloadBusy === file.id ? "Downloading…" : "Download"}</button> : <span className="v-status">File unavailable</span>}</div></li>)}</ul>}
      {moreError && <p role="alert">More documents unavailable.</p>}{cursor && <button disabled={moreBusy} onClick={() => void more()}>{moreBusy ? "Loading…" : "Load more"}</button>}
    </section>{selected && <DocumentDetail key={selected.id} file={selected} focusEpoch={selectionEpoch} onClose={closeDocument} onViewForm={() => viewForm(selected)} onDownload={() => void download(selected)} downloadBusy={Boolean(downloadBusy)}/>}</div>}
  </section>;
}

function DocumentDetail({ file, focusEpoch, onClose, onViewForm, onDownload, downloadBusy }: { file: DocumentOccurrence; focusEpoch: number; onClose: () => void; onViewForm: () => void; onDownload: () => void; downloadBusy: boolean }) {
  const heading = useRef<HTMLHeadingElement>(null);
  const [preview, setPreview] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(false);
  const controller = useRef<AbortController | null>(null);
  const previewURL = useRef("");
  useEffect(() => () => { controller.current?.abort(); if (previewURL.current) URL.revokeObjectURL(previewURL.current); }, []);
  const mediaType = file.media_type?.split(";", 1)[0]?.trim().toLowerCase();
  const pdf = mediaType === "application/pdf";
  const image = ["image/png", "image/jpeg", "image/gif", "image/webp"].includes(mediaType ?? "");
  const eligible = canDownload(file) || (file.artifact_status === "STORED_UNSCANNED" && file.demo_preview_available === true);
  const pdfUnsupported = pdf && typeof navigator !== "undefined" && navigator.pdfViewerEnabled === false;
  const previewAllowed = (pdf || image) && eligible && !pdfUnsupported;
  async function loadPreview() {
    if (loading || preview) return;
    const pending = new AbortController(); controller.current = pending; setLoading(true); setError(false);
    try {
      const result = await requestBlob("", documentContentURL(file), { signal: pending.signal, cache: "no-store" });
      const actualType = result.blob.type.split(";", 1)[0]?.trim().toLowerCase();
      const knownSize = Number.isSafeInteger(file.size_bytes) && file.size_bytes >= 0;
      if (actualType !== mediaType || (file.size_bytes != null && !knownSize) || (knownSize && result.blob.size !== file.size_bytes)) throw new Error("Document content does not match its metadata");
      if (!pending.signal.aborted) { previewURL.current = URL.createObjectURL(result.blob); setPreview(previewURL.current); }
    }
    catch { if (!pending.signal.aborted) setError(true); }
    finally { if (!pending.signal.aborted) setLoading(false); }
  }
  useEffect(() => {
    heading.current?.focus();
    if (window.innerWidth <= 850) heading.current?.scrollIntoView?.({ block: "start" });
    if (previewAllowed) void loadPreview();
  }, [file.id, focusEpoch]);
  function imageFailed() { if (previewURL.current) URL.revokeObjectURL(previewURL.current); previewURL.current = ""; setPreview(""); setError(true); }
  return <aside className="v-panel v-document-detail" aria-label="Document details"><div className="v-document-detail-heading"><div><p className="v-document-detail-label">Document details</p><h2 ref={heading} tabIndex={-1}>{file.file_name}</h2></div><button type="button" onClick={onClose} aria-label="Close document details">×</button></div>{file.artifact_status === "STORED_UNSCANNED" && (file.demo_preview_available || file.demo_unscanned_allowed) && <p>Demo file. No antivirus scan was performed.</p>}{pdfUnsupported && eligible && <p>{canDownload(file) ? "PDF preview unavailable in this browser. Download the document." : "PDF preview unavailable in this browser."}</p>}{eligible && !pdf && !image && <p>{canDownload(file) ? "Preview unavailable for this file type. Download the document." : "Preview unavailable for this file type."}</p>}<div className="v-document-actions">{pdfUnsupported && canDownload(file) && <button type="button" aria-label={`Download ${file.file_name}`} disabled={downloadBusy} onClick={onDownload}>{downloadBusy ? "Downloading…" : "Download"}</button>}{previewAllowed && !preview && <button type="button" disabled={loading} onClick={() => void loadPreview()}>{loading ? "Loading…" : "Preview document"}</button>}{file.form_template_id && <button type="button" onClick={onViewForm}>View form documents</button>}{file.response_revision_id && <a className="v-primary" href={withDocumentReturn(`#/forms/responses/${encodeURIComponent(file.response_revision_id)}`)}>Open response</a>}{file.relationship_id && file.assessment_id && <a className="v-document-link" href={withDocumentReturn(`#/vendors/${encodeURIComponent(file.relationship_id)}/assessments/${encodeURIComponent(file.assessment_id)}`)}>Open vendor assessment</a>}</div>{error && <p role="alert">Preview unavailable. Try again.</p>}{preview && <div className="v-document-preview">{pdf ? <iframe title={`Preview: ${file.file_name}`} src={preview} sandbox=""/> : <img src={preview} alt={`Preview: ${file.file_name}`} onError={imageFailed}/>}</div>}<dl><div><dt>Form</dt><dd>{file.form_title || "Not recorded"}</dd></div><div><dt>Question</dt><dd>{file.field_label || "Not recorded"}</dd></div><div><dt>Form version</dt><dd>{file.form_template_version ?? "Not recorded"}</dd></div><DocumentProvenance file={file}/><div><dt>Submission</dt><dd>{file.current ? "Latest submission" : "Earlier submission"}</dd></div>{file.expires_on && <div><dt>Expires</dt><dd>{formatDate(file.expires_on)}</dd></div>}</dl></aside>;
}

function canDownload(file: DocumentOccurrence) { return file.artifact_status === "AVAILABLE" || (file.artifact_status === "STORED_UNSCANNED" && file.demo_unscanned_allowed === true); }
function formatDate(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? "date unavailable" : new Intl.DateTimeFormat(undefined, { day: "numeric", month: "short", year: "numeric", timeZone: "UTC" }).format(date); }
