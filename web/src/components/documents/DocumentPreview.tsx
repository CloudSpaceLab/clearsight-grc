import { useEffect, useState } from "react";
import { documentContentURL, previewKind, type DocumentOccurrence } from "../../submittedDocumentApi";
import { Button, FocusedDialog, Notice } from "../ui";
import { DocumentDownload, DocumentFacts, FileIcon, fileKindLabel, fileSize } from "./DocumentFile";

export function DocumentPreview({ file, onClose }: { file: DocumentOccurrence; onClose: () => void }) {
  const kind = previewKind(file);
  const pdfUnavailable = kind === "pdf" && navigator.pdfViewerEnabled === false;
  const [url, setURL] = useState<string>();
  const [failed, setFailed] = useState(false);
  const [reload, setReload] = useState(0);
  useEffect(() => {
    if (!kind || pdfUnavailable) return;
    const controller = new AbortController();
    let objectURL: string | undefined;
    setURL(undefined); setFailed(false);
    void fetch(documentContentURL(file), { credentials: "include", cache: "no-store", signal: controller.signal })
      .then(async (response) => {
        if (!response.ok) throw new Error("Document unavailable");
        const blob = await response.blob();
        if (blob.size !== file.size_bytes || blob.type !== file.media_type || !previewKind({ media_type: blob.type, artifact_status: "AVAILABLE" })) throw new Error("Unsupported preview content");
        if (controller.signal.aborted) return;
        objectURL = URL.createObjectURL(blob); setURL(objectURL);
      }).catch(() => { if (!controller.signal.aborted) setFailed(true); });
    return () => { controller.abort(); if (objectURL) URL.revokeObjectURL(objectURL); };
  }, [file, kind, pdfUnavailable, reload]);
  return <FocusedDialog label={`Preview ${file.file_name}`} closeLabel="Close preview" size="wide" onClose={onClose} panelClassName="document-quick-look">
    <header className="document-preview-heading"><FileIcon kind={file.file_kind}/><div><h2>{file.file_name}</h2><p>{fileKindLabel(file.file_kind)} · {fileSize(file.size_bytes)}</p></div></header>
    <div className="document-preview-actions"><DocumentDownload file={file}/></div>
    <div className="document-preview-body"><div className="document-preview-canvas">
      {file.artifact_status !== "AVAILABLE" ? <Notice tone="warning">{file.artifact_status === "QUARANTINED" ? "This file is quarantined. It cannot be previewed or downloaded." : file.artifact_status === "STORED_UNSCANNED" ? "The file safety check has not completed. Preview and download are unavailable until it passes." : "This file is unavailable. Its submission details remain visible below."}</Notice>
        : pdfUnavailable ? <div className="document-preview-fallback"><FileIcon kind={file.file_kind}/><p>This browser cannot preview PDFs. Download the file to view it in a PDF application.</p></div>
        : !kind ? <div className="document-preview-fallback"><FileIcon kind={file.file_kind}/><p>Inline preview is not available for this file type. Download the file to view it in its application.</p></div>
        : failed ? <Notice tone="error">The document preview could not be loaded. Your access or the file may have changed. <Button onPress={() => setReload((value) => value + 1)}>Retry preview</Button></Notice>
        : !url ? <p role="status">Loading document preview…</p>
        : kind === "image" ? <img src={url} alt={`Submitted document: ${file.file_name}`} onError={() => setFailed(true)}/>
        : <iframe src={url} title={`Document preview: ${file.file_name}`} referrerPolicy="no-referrer"/>}
    </div><aside className="document-preview-details"><h3>File details</h3><DocumentFacts file={file}/><p>File safety checks do not establish whether the document is genuine or suitable for its purpose.</p></aside></div>
  </FocusedDialog>;
}
