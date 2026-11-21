import { useEffect, useState } from "react";
import { loadDocuments, type DocumentOccurrence, type FileKind } from "../../submittedDocumentApi";
import { Button, DataTable, EmptyState, Notice, SearchField, SelectField, StatusBadge, Tabs, type DataColumn } from "../ui";
import { assessmentReviewStatus, DocumentDemoNotice, DocumentFacts, FileIcon, fileKindLabel, fileSize, documentDate, fileStatus } from "./DocumentFile";
import { DocumentPreview } from "./DocumentPreview";
import "./documents.css";

const kinds: ReadonlyArray<{ id: FileKind | "ALL"; label: string }> = [
  { id: "ALL", label: "All files" }, { id: "PDF", label: "PDF files" }, { id: "IMAGE", label: "Images" },
  { id: "WORD", label: "Word documents" }, { id: "SPREADSHEET", label: "Spreadsheets" }, { id: "OTHER", label: "Other files" },
];
type DocumentView = "ALL" | "AWAITING_REVIEW" | "NEEDS_REFRESH" | "REVIEW_HISTORY";
const views: ReadonlyArray<{ id: DocumentView; label: string }> = [
  { id: "ALL", label: "All documents" },
  { id: "AWAITING_REVIEW", label: "Awaiting assessment review" },
  { id: "NEEDS_REFRESH", label: "Needs refresh" },
  { id: "REVIEW_HISTORY", label: "Assessment review history" },
];

export function DocumentBrowser({ scopeLabel, relationshipID, responseRevisionID, onChoose, selectionUnavailableReason, headingLevel = "h2" }: { scopeLabel: string; relationshipID?: string; responseRevisionID?: string; onChoose?: (file: DocumentOccurrence) => void; selectionUnavailableReason?: (file: DocumentOccurrence) => string; headingLevel?: "h1" | "h2" }) {
  const Heading = headingLevel;
  const [kind, setKind] = useState<FileKind | "ALL">("ALL");
  const [view, setView] = useState<DocumentView>("ALL");
  const [query, setQuery] = useState("");
  const [includeHistory, setIncludeHistory] = useState(false);
  const [filtersOpen, setFiltersOpen] = useState(false);
  const [pages, setPages] = useState<string[]>([]);
  const [nextCursor, setNextCursor] = useState<string>();
  const [items, setItems] = useState<DocumentOccurrence[]>([]);
  const [selectedID, setSelectedID] = useState<string>();
  const [previewID, setPreviewID] = useState<string>();
  const [state, setState] = useState<"loading" | "ready" | "error">("loading");
  const [reload, setReload] = useState(0);
  const cursor = pages.at(-1);
  useEffect(() => {
    const controller = new AbortController();
    setState("loading"); setItems([]); setSelectedID(undefined); setPreviewID(undefined); setNextCursor(undefined);
    const timer = window.setTimeout(() => {
      void loadDocuments({ file_kind: kind === "ALL" ? undefined : kind, query: query.trim(), relationship_id: relationshipID,
        response_revision_id: responseRevisionID, view: view === "ALL" ? undefined : view,
        current_only: responseRevisionID || view === "REVIEW_HISTORY" ? false : !includeHistory, cursor, limit: 25 }, controller.signal)
        .then((page) => { if (!controller.signal.aborted) { setItems(page.items); setNextCursor(page.next_cursor); setState("ready"); } })
        .catch(() => { if (!controller.signal.aborted) setState("error"); });
    }, 150);
    return () => { window.clearTimeout(timer); controller.abort(); };
  }, [kind, query, relationshipID, responseRevisionID, includeHistory, view, cursor, reload]);
  const selected = items.find((item) => item.id === selectedID);
  const selectionReason = selected && selectionUnavailableReason ? selectionUnavailableReason(selected) : "";
  const preview = items.find((item) => item.id === previewID);
  function open(file: DocumentOccurrence) { setSelectedID(file.id); setPreviewID(file.id); }
  function selectKind(value: FileKind | "ALL" | undefined) {
    if (value === undefined || value === kind) return;
    setKind(value); setPages([]);
  }
  const columns: readonly DataColumn<DocumentOccurrence>[] = [
    { id: "name", header: "Name", mobileLayout: "full-width", render: (file) => <span className="document-file-name"><FileIcon kind={file.file_kind}/><strong title={file.file_name}>{file.file_name}</strong></span>, accessibleText: (file) => file.file_name },
    { id: "kind", header: "Kind", render: (file) => fileKindLabel(file.file_kind), accessibleText: (file) => fileKindLabel(file.file_kind) },
    { id: "submitted", header: "Submitted", render: (file) => documentDate(file.submitted_at), accessibleText: (file) => documentDate(file.submitted_at) },
    { id: "status", header: "File status", kind: "status", render: (file) => <StatusBadge tone={file.artifact_status === "AVAILABLE" ? "neutral" : "warning"}>{fileStatus(file)}</StatusBadge>, accessibleText: fileStatus },
    { id: "review", header: "Assessment review", kind: "status", render: (file) => <StatusBadge tone={file.review?.status === "REJECTED" || file.review?.status === "EXPIRED" ? "warning" : "neutral"}>{assessmentReviewStatus(file)}</StatusBadge>, accessibleText: assessmentReviewStatus },
  ];
  const body = <div className="document-browser-body">
      <nav className="document-kinds" aria-label="File types">{kinds.map((item) => <Button key={item.id} variant={kind === item.id ? "secondary" : "quiet"} aria-pressed={kind === item.id} onPress={() => selectKind(item.id)}><span className="document-kind-label"><FileIcon kind={item.id}/><span>{item.label}</span><span aria-hidden="true">{kind === item.id ? "✓" : ""}</span></span></Button>)}</nav>
      <div className="document-browser-content">
        <div className="document-kind-selector"><SelectField label="File type" value={kind} placeholder="File type" options={kinds} allowsEmpty={false} onChange={selectKind}/></div>
        <div className="document-toolbar"><SearchField label="Search file names" placeholder="Search file names" value={query} onChange={(value) => { setQuery(value); setPages([]); }}/>{!responseRevisionID && view === "ALL" && <Button aria-expanded={filtersOpen} onPress={() => setFiltersOpen(!filtersOpen)}>Filters{includeHistory ? " · History included" : ""}</Button>}</div>
        {(view === "AWAITING_REVIEW" || view === "REVIEW_HISTORY") && <p className="document-review-limitation">Only recorded vendor assessment reviews are shown.</p>}
        {filtersOpen && !responseRevisionID && <div className="document-filters"><SelectField label="Submitted versions" placeholder="Current versions" value={includeHistory ? "ALL" : "CURRENT"} allowsEmpty={false} options={[{ id: "CURRENT", label: "Current versions" }, { id: "ALL", label: "Include previous versions" }]} onChange={(value) => { setIncludeHistory(value === "ALL"); setPages([]); }}/></div>}
        {selected && onChoose && <div className="document-toolbar" aria-label="Document selection"><div><strong>{selected.file_name}</strong>{selectionReason && <p>{selectionReason}</p>}</div><Button variant="primary" isDisabled={Boolean(selectionReason)} onPress={() => onChoose(selected)}>Choose this document</Button></div>}
        {state === "loading" && <p role="status">Loading documents…</p>}
        {state === "error" && <Notice tone="error"><strong>Documents could not be loaded.</strong> Check your connection and access, then try again. <Button onPress={() => setReload((value) => value + 1)}>Reload documents</Button></Notice>}
        {state === "ready" && (items.length ? <>
          <DataTable ariaLabel={scopeLabel} responsiveTo="container" rows={items} rowKey={(file) => file.id} rowName={(file) => `${file.file_name}, ${fileKindLabel(file.file_kind)}, ${fileStatus(file)}, ${assessmentReviewStatus(file)}`}
            columns={columns} selectedKey={selectedID} onSelectionChange={(file) => setSelectedID(file.id)} onRowAction={open} rowActionLabel="Preview"
            pagination={{ label: "Document pages", onPrevious: pages.length ? () => setPages((value) => value.slice(0, -1)) : undefined, onNext: nextCursor ? () => setPages((value) => [...value, nextCursor]) : undefined }}/>
          <footer className="document-list-footer"><span>{items.length} {items.length === 1 ? "file" : "files"}{nextCursor ? " · More available" : pages.length ? " · Previous results available" : ""}</span></footer>
        </> : <EmptyState population={`${scopeLabel} · ${kind === "ALL" ? "All file types" : fileKindLabel(kind)}${query ? ` · “${query}”` : ""}`} title="No matching documents" description="No submitted documents match. Change the file type or search."/>)}
      </div>
      {selected && <aside className="document-inspector" aria-label="Selected file details"><FileIcon kind={selected.file_kind}/><h3>{selected.file_name}</h3><p>{fileKindLabel(selected.file_kind)} · {fileSize(selected.size_bytes)}</p><DocumentDemoNotice file={selected}/><Button onPress={() => open(selected)}>Preview file</Button><DocumentFacts file={selected}/></aside>}
    </div>;
  return <section className={`document-browser${onChoose ? " document-browser--selecting" : ""}`} data-standalone={headingLevel === "h1" || undefined} aria-label={scopeLabel}>
    <header className="document-browser-heading"><div><Heading>Documents</Heading><p>{scopeLabel}</p></div><Button variant="quiet" onPress={() => setReload((value) => value + 1)}>Refresh files</Button></header>
    {headingLevel === "h1" ? <Tabs ariaLabel="Document views" compactLabel="Document view" items={views} selectedKey={view} onSelectionChange={(next) => { setView(next); setPages([]); setFiltersOpen(false); setIncludeHistory(false); }}>{() => body}</Tabs> : body}
    {preview && <DocumentPreview key={preview.id} file={preview} onClose={() => setPreviewID(undefined)}/>}
  </section>;
}
