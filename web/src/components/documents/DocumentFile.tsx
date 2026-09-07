import { documentContentURL, type DocumentOccurrence, type FileKind } from "../../submittedDocumentApi";
import { ActionLink } from "../ui";

export function FileIcon({ kind }: { kind: FileKind | "ALL" }) {
  return <svg className="document-file-icon" data-kind={kind} viewBox="0 0 32 38" aria-hidden="true"><path d="M5 1h14l8 8v27H5z"/><path d="M19 1v9h8"/>{kind === "IMAGE" ? <><circle cx="12" cy="17" r="2"/><path d="m8 29 6-7 4 4 3-3 4 6"/></> : kind === "SPREADSHEET" ? <><path d="M9 16h14v14H9zM9 23h14M16 16v14"/></> : <><path d="M10 17h12M10 22h12M10 27h8"/></>}</svg>;
}
export function fileKindLabel(kind: FileKind) { return ({ PDF: "PDF", IMAGE: "Image", WORD: "Word document", SPREADSHEET: "Spreadsheet", OTHER: "Other file" })[kind] ?? "Other file"; }
export function fileSize(bytes: number) { return Number.isFinite(bytes) && bytes >= 0 ? bytes < 1024 ? `${bytes} B` : bytes < 1048576 ? `${(bytes / 1024).toFixed(1)} KB` : `${(bytes / 1048576).toFixed(1)} MB` : "Size not recorded"; }
export function documentDate(value?: string) { return value && Number.isFinite(Date.parse(value)) ? new Date(value).toLocaleDateString(undefined, { year: "numeric", month: "short", day: "numeric" }) : "Not recorded"; }
export function fileStatus(file: DocumentOccurrence) { return ({ AVAILABLE: "Ready to view", STORED_UNSCANNED: "Safety check pending", QUARANTINED: "Quarantined", DELETED: "Removed" })[file.artifact_status] ?? "File unavailable"; }

export function DocumentFacts({ file }: { file: DocumentOccurrence }) {
  return <dl className="document-facts">
    <div><dt>Form</dt><dd>{file.form_title}</dd></div><div><dt>Question</dt><dd>{file.field_label}</dd></div>
    <div><dt>Last submitted</dt><dd><DocumentTime value={file.submitted_at}/></dd></div><div><dt>Submitted by</dt><dd>{file.submitted_by || "Not recorded"}</dd></div>
    <div><dt>Uploaded</dt><dd><DocumentTime value={file.uploaded_at}/></dd></div><div><dt>Uploaded by</dt><dd>{file.uploaded_by || "Not recorded"}</dd></div>
    <div><dt>Expiry date</dt><dd>{documentDate(file.expires_on)}</dd></div><div><dt>Submitted version</dt><dd>{file.current ? "Current" : "Previous"}</dd></div>
    <div><dt>Document review</dt><dd>{file.review ? ({ VALIDATED: "Validated by reviewer", REJECTED: "Rejected", EXPIRED: "Expired", SUBMITTED: "Awaiting review", SUPERSEDED: "Replaced" })[file.review.status] ?? "Review status unavailable" : "No review recorded"}</dd></div>
    {file.review && <><div><dt>Reviewed by</dt><dd>{file.review.reviewed_by || "Not recorded"}</dd></div><div><dt>Reviewed</dt><dd><DocumentTime value={file.review.reviewed_at}/></dd></div></>}
  </dl>;
}

function DocumentTime({ value }: { value?: string }) {
  return value && Number.isFinite(Date.parse(value)) ? <time dateTime={value}>{new Date(value).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" })}</time> : <>Not recorded</>;
}

export function DocumentDownload({ file }: { file: DocumentOccurrence }) {
  return file.artifact_status === "AVAILABLE" ? <ActionLink href={documentContentURL(file, true)} download={file.file_name}>Download file</ActionLink> : null;
}
