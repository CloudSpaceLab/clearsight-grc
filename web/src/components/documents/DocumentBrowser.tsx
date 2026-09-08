import { useEffect, useState } from "react";
import { loadDocuments, type DocumentOccurrence, type FileKind } from "../../submittedDocumentApi";
import { Button, DataTable, EmptyState, Notice, SearchField, SelectField, StatusBadge, type DataColumn } from "../ui";
import { DocumentFacts, FileIcon, fileKindLabel, fileSize, documentDate, fileStatus } from "./DocumentFile";
import { DocumentPreview } from "./DocumentPreview";
import "./documents.css";

const kinds: ReadonlyArray<{ id: FileKind | "ALL"; label: string }> = [
  { id: "ALL", label: "All files" }, { id: "PDF", label: "PDF files" }, { id: "IMAGE", label: "Images" },
  { id: "WORD", label: "Word documents" }, { id: "SPREADSHEET", label: "Spreadsheets" }, { id: "OTHER", label: "Other files" },
];

export function DocumentBrowser({ scopeLabel, relationshipID, responseRevisionID }: { scopeLabel: string; relationshipID?: string; responseRevisionID?: string }) {
  const [kind, setKind] = useState<FileKind | "ALL">("ALL");
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
        response_revision_id: responseRevisionID, current_only: responseRevisionID ? false : !includeHistory, cursor, limit: 25 }, controller.signal)
        .then((page) => { if (!controller.signal.aborted) { setItems(page.items); setNextCursor(page.next_cursor); setState("ready"); } })
        .catch(() => { if (!controller.signal.aborted) setState("error"); });
    }, 150);
    return () => { window.clearTimeout(timer); controller.abort(); };
  }, [kind, query, relationshipID, responseRevisionID, includeHistory, cursor, reload]);
  const selected = items.find((item) => item.id === selectedID);
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
    { id: "preview", header: "Preview", kind: "action", render: (file) => <Button size="compact" variant="quiet" aria-label={`Preview ${file.file_name}`} onPress={() => open(file)}>Preview</Button>, accessibleText: (file) => `Preview ${file.file_name}` },
  ];
  return <section className="document-browser" aria-label={scopeLabel}>
    <header className="document-browser-heading"><div><h2>Documents</h2><p>{scopeLabel}</p></div><Button variant="quiet" onPress={() => setReload((value) => value + 1)}>Refresh files</Button></header>
    <div className="document-browser-body">
      <nav className="document-kinds" aria-label="File types">{kinds.map((item) => <Button key={item.id} variant={kind === item.id ? "secondary" : "quiet"} aria-pressed={kind === item.id} onPress={() => selectKind(item.id)}><span className="document-kind-label"><FileIcon kind={item.id}/>{item.label}</span></Button>)}</nav>
      <div className="document-browser-content">
        <div className="document-kind-selector"><SelectField label="File type" value={kind} placeholder="File type" options={kinds} allowsEmpty={false} onChange={selectKind}/></div>
        <div className="document-toolbar"><SearchField label="Search file names" placeholder="Search file names" value={query} onChange={(value) => { setQuery(value); setPages([]); }}/>{!responseRevisionID && <Button aria-expanded={filtersOpen} onPress={() => setFiltersOpen(!filtersOpen)}>Filters{includeHistory ? " · History included" : ""}</Button>}</div>
        {filtersOpen && !responseRevisionID && <div className="document-filters"><SelectField label="Submitted versions" placeholder="Current versions" value={includeHistory ? "ALL" : "CURRENT"} allowsEmpty={false} options={[{ id: "CURRENT", label: "Current versions" }, { id: "ALL", label: "Include previous versions" }]} onChange={(value) => { setIncludeHistory(value === "ALL"); setPages([]); }}/></div>}
        {state === "loading" && <p role="status">Loading documents…</p>}
        {state === "error" && <Notice tone="error"><strong>Documents could not be loaded.</strong> Check your connection and access, then try again. <Button onPress={() => setReload((value) => value + 1)}>Reload documents</Button></Notice>}
        {state === "ready" && (items.length ? <>
          <DataTable ariaLabel={scopeLabel} rows={items} rowKey={(file) => file.id} rowName={(file) => `${file.file_name}, ${fileKindLabel(file.file_kind)}, ${fileStatus(file)}`}
            columns={columns} selectedKey={selectedID} onSelectionChange={(file) => setSelectedID(file.id)} onRowAction={open}
            pagination={{ label: "Document pages", onPrevious: pages.length ? () => setPages((value) => value.slice(0, -1)) : undefined, onNext: nextCursor ? () => setPages((value) => [...value, nextCursor]) : undefined }}/>
          <footer className="document-list-footer"><span>{items.length} {items.length === 1 ? "file" : "files"} on this page{nextCursor ? " · More available" : ""}</span><span>Select a file · Press Space to preview</span></footer>
        </> : <EmptyState population={`${scopeLabel} · ${kind === "ALL" ? "All file types" : fileKindLabel(kind)}${query ? ` · “${query}”` : ""}`} title="No matching documents" description="No submitted documents were found for this view. Change the file type or search to check another view."/>)}
      </div>
      {selected && <aside className="document-inspector" aria-label="Selected file details"><FileIcon kind={selected.file_kind}/><h3>{selected.file_name}</h3><p>{fileKindLabel(selected.file_kind)} · {fileSize(selected.size_bytes)}</p><Button onPress={() => open(selected)}>Preview file</Button><DocumentFacts file={selected}/></aside>}
    </div>
    {preview && <DocumentPreview key={preview.id} file={preview} onClose={() => setPreviewID(undefined)}/>}
  </section>;
}
